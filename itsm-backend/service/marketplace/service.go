package marketplace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"itsm-backend/connector"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/ent/predicate"
	"itsm-backend/ent/tenantinstallation"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"go.uber.org/zap"
)

// Service 市场服务
type Service struct {
	db               *ent.Client
	logger           *zap.SugaredLogger
	connectorManager *connector.Manager
}

var (
	ErrMarketplaceItemNotFound        = errors.New("marketplace item not found")
	ErrMarketplaceItemUnavailable     = errors.New("marketplace item unavailable")
	ErrMarketplaceInstallationAbsent  = errors.New("marketplace installation not found")
	ErrMarketplaceInstalledByMissing  = errors.New("marketplace installed_by is required")
	ErrMarketplaceConfigSchemaInvalid = errors.New("config does not match item schema")

	// ErrMarketplaceDependencyUnsatisfied 待装版本声明的依赖在当前租户内未满足。
	// 这是业务冲突（409）而非参数错误：商品本身可安装，只是前置组件没装或版本不符。
	// 具体缺口见 DependencyUnsatisfiedError.Problems（可逐项展示给管理员）。
	ErrMarketplaceDependencyUnsatisfied = errors.New("marketplace dependency unsatisfied")

	// ErrMarketplaceItemRequiredByDependents 目标组件仍被同租户内其他已安装组件依赖。
	// 与安装期闸门对称：能装的链路，也要能拦住把中间环节拆掉的卸载。
	ErrMarketplaceItemRequiredByDependents = errors.New("marketplace item is required by installed dependents")

	// ErrMarketplaceConnectorNotReady 连接器商品的运行态同步失败。
	//
	// 这里的「失败」指的是 marketplace 的安装记录（DB）与 connector.Manager 的
	// 运行实例（内存）不一致：记录已存在，但连接器没起来（缺凭据/端点不可达/
	// provider 未注册）。此前该错误被 logger.Warnw 吞掉后照样把状态置 active，
	// 于是市场页显示「已安装」，而 send/回调/通知投递全部失败——故障现场与
	// 管理界面相反，排障方向被误导。现在改为显式错误 + 安装记录 failed。
	ErrMarketplaceConnectorNotReady = errors.New("connector runtime is not ready")

	// ErrMarketplaceConnectorStillRunning 卸载时运行态未真正注销。
	//
	// Revoke 以 (tenant, name, provider) 为键删除内存实例；历史上 provider 可能
	// 被写成与安装时不符的值，导致键不匹配、实例残留，而安装记录已被标成
	// uninstalled——「以为卸了，实际还在发消息」。卸载必须校验实例确实消失。
	ErrMarketplaceConnectorStillRunning = errors.New("connector instance is still running after revocation")
)

// NewService 创建市场服务
func NewService(db *ent.Client, logger *zap.SugaredLogger) *Service {
	return &Service{
		db:     db,
		logger: logger,
	}
}

// SetConnectorManager 注入连接器管理器，安装/卸载时自动 provision/revoke 连接器
func (s *Service) SetConnectorManager(mgr *connector.Manager) {
	s.connectorManager = mgr
}

// ListItems 查询市场商品列表
// 返回 camelCase DTO，不得直接序列化 ent 模型。
func (s *Service) ListItems(ctx context.Context, itemType, category, search string, isOfficial *bool, page, pageSize int) ([]*dto.MarketplaceItemResponse, int, error) {
	query := s.db.MarketplaceItem.Query().
		Where(marketplaceitem.StatusEQ(marketplaceitem.StatusPublished))

	if itemType != "" {
		query = query.Where(marketplaceitem.TypeEQ(marketplaceitem.Type(itemType)))
	}
	if category != "" {
		query = query.Where(marketplaceitem.CategoryEQ(category))
	}
	if isOfficial != nil {
		query = query.Where(marketplaceitem.IsOfficialEQ(*isOfficial))
	}
	if search != "" {
		query = query.Where(
			marketplaceitem.Or(
				marketplaceitem.NameContains(search),
				marketplaceitem.TitleContains(search),
				marketplaceitem.ProviderContains(search),
				marketplaceitem.DescriptionContains(search),
				predicate.MarketplaceItem(func(selector *sql.Selector) {
					selector.Where(sqljson.ValueContains(marketplaceitem.FieldTags, search))
				}),
			),
		)
	}

	// 查询总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count items: %w", err)
	}

	// 分页查询
	items, err := query.
		WithVersions().
		Order(ent.Desc(marketplaceitem.FieldInstallCount), ent.Desc(marketplaceitem.FieldRating)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list items: %w", err)
	}

	return dto.ToMarketplaceItemResponseList(items), total, nil
}

// GetItem 获取商品详情
// 返回 camelCase DTO，不得直接序列化 ent 模型。
func (s *Service) GetItem(ctx context.Context, itemID int) (*dto.MarketplaceItemResponse, error) {
	item, err := s.db.MarketplaceItem.Query().
		Where(marketplaceitem.ID(itemID)).
		Where(marketplaceitem.StatusEQ(marketplaceitem.StatusPublished)).
		WithVersions().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrMarketplaceItemNotFound
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}
	return dto.ToMarketplaceItemResponse(item), nil
}

// reactivateUninstalledInstallation 复用并重置历史卸载记录
// 用于处理 UNIQUE(tenant_id, item_id) 约束冲突场景：
// 同一租户对同一商品在 uninstall 后再次 install 时，保留历史但重新激活。
func (s *Service) reactivateUninstalledInstallation(
	ctx context.Context,
	tenantID, itemID int,
	item *ent.MarketplaceItem,
	installedBy string,
) (*ent.TenantInstallation, error) {
	history, err := s.db.TenantInstallation.Query().
		Where(
			tenantinstallation.TenantID(tenantID),
			tenantinstallation.ItemID(itemID),
			tenantinstallation.StatusEQ(tenantinstallation.StatusUninstalled),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// 历史记录在并发场景下不存在，说明另一个并发请求已重新激活
			return s.GetInstallation(ctx, tenantID, itemID)
		}
		return nil, fmt.Errorf("failed to locate uninstalled history: %w", err)
	}

	// 复用历史记录：先回到 installing，再走统一的「provision → 状态」路径。
	//
	// 旧实现直接 SetStatus(active) 且完全不 provision——卸载后再装，安装记录
	// 显示已安装，连接器实例却根本不存在，是同一类「记录与运行态分裂」。
	reactivated, err := s.db.TenantInstallation.UpdateOneID(history.ID).
		SetInstalledVersion(item.LatestVersion).
		SetStatus(tenantinstallation.StatusInstalling).
		SetInstalledBy(installedBy).
		ClearErrorMessage().
		SetLastUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to reactivate installation: %w", err)
	}

	_, err = s.db.MarketplaceItem.UpdateOneID(itemID).
		AddInstallCount(1).
		Save(ctx)
	if err != nil {
		s.logger.Warnw("Failed to increment install count during reactivation", "item_id", itemID, "error", err)
	}

	s.logger.Infow(
		"Reactivated uninstalled installation",
		"tenant_id", tenantID,
		"item_id", itemID,
		"history_id", history.ID,
	)
	return s.activateInstallation(ctx, item, reactivated, installedBy, reactivated.Config)
}

// InstallItem 租户安装商品
func (s *Service) InstallItem(ctx context.Context, tenantID, itemID int, installedBy string) (*ent.TenantInstallation, error) {
	// 业务保护四道闸
	// 1) 商品必须存在
	item, err := s.db.MarketplaceItem.Get(ctx, itemID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrMarketplaceItemNotFound
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}
	// 2) 仅允许安装已发布的商品，禁止安装草稿/已下架
	if item.Status != marketplaceitem.StatusPublished {
		return nil, fmt.Errorf("%w (status=%s)", ErrMarketplaceItemUnavailable, item.Status)
	}
	// 3) 安装人记录不可为空
	if installedBy == "" {
		return nil, ErrMarketplaceInstalledByMissing
	}
	// 4) 已存在安装记录时的两种语义：
	//    - active / disabled：幂等返回当前安装，避免重复安装报错、也不重复 provision
	//    - installing / failed：上次没装成（典型是连接器还没填凭据），本次视为重试
	existingInstallation, err := s.db.TenantInstallation.Query().
		Where(
			tenantinstallation.TenantID(tenantID),
			tenantinstallation.ItemID(itemID),
			tenantinstallation.StatusNEQ(tenantinstallation.StatusUninstalled),
		).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, fmt.Errorf("failed to check installation: %w", err)
		}
	} else if existingInstallation.Status == tenantinstallation.StatusActive ||
		existingInstallation.Status == tenantinstallation.StatusDisabled {
		s.logger.Infow("Item already installed, returning existing installation", "tenant_id", tenantID, "item_id", itemID, "installation_id", existingInstallation.ID)
		return existingInstallation, nil
	} else {
		// 上次停在 installing/failed：把「安装」按钮当成重试入口，而不是无脑返回
		// 那条永远起不来的记录。重试前同样要过依赖闸门，否则会把一条本不该
		// 成立的安装记录推向 active。
		if err := s.validateDependencies(ctx, tenantID, item.ID, item.Name, item.LatestVersion); err != nil {
			return nil, err
		}
		s.logger.Infow("Retrying unfinished marketplace installation", "tenant_id", tenantID, "item_id", itemID, "installation_id", existingInstallation.ID, "status", existingInstallation.Status)
		return s.activateInstallation(ctx, item, existingInstallation, installedBy, existingInstallation.Config)
	}

	// 依赖闸门：版本声明的依赖必须已在当前租户安装且版本满足。
	// 放在落安装记录之前——先写记录再校验失败会留下一条永远起不来的 installing 行，
	// 让「没装上」和「装了一半」在界面上不可区分。
	if err := s.validateDependencies(ctx, tenantID, item.ID, item.Name, item.LatestVersion); err != nil {
		return nil, err
	}

	// 5) 尝试创建安装记录（处理 UNIQUE(tenant_id, item_id) 冲突场景：存在历史卸载记录）
	installation, err := s.db.TenantInstallation.Create().
		SetTenantID(tenantID).
		SetItemID(itemID).
		SetInstalledVersion(item.LatestVersion).
		SetStatus(tenantinstallation.StatusInstalling).
		SetInstalledBy(installedBy).
		Save(ctx)
	if err != nil {
		// 防御：UNIQUE(tenant_id, item_id) 约束冲突时，说明存在历史 uninstalled 记录
		// 我们复用历史记录，将其重新激活（保留审计历史，避免数据丢失）
		if ent.IsConstraintError(err) {
			return s.reactivateUninstalledInstallation(ctx, tenantID, itemID, item, installedBy)
		}
		return nil, fmt.Errorf("failed to create installation: %w", err)
	}

	// 增加安装计数
	_, err = s.db.MarketplaceItem.UpdateOneID(itemID).
		AddInstallCount(1).
		Save(ctx)
	if err != nil {
		s.logger.Warnw("Failed to increment install count", "item_id", itemID, "error", err)
	}

	// 执行实际安装：连接器类商品必须在运行态真正 provision 成功才算装完。
	// 失败时 install 记录保留为 failed + error_message（见 activateInstallation）。
	return s.activateInstallation(ctx, item, installation, installedBy, nil)
}

// activateInstallation 把「安装记录」与「连接器运行态」对齐后再落最终状态。
//
// 顺序是关键：先 provision，再改状态。反过来的话，provision 失败时记录已经是
// active，市场页与运行态就永久不一致（旧实现正是如此：Warn 之后照样 SetStatus(active)）。
// provision 失败时记录留为 failed + error_message：既不丢安装痕迹，也给出可重试入口
// ——用户补全凭据后可以重新点安装，或直接 PUT config（两者都会重新 provision）。
func (s *Service) activateInstallation(
	ctx context.Context,
	item *ent.MarketplaceItem,
	installation *ent.TenantInstallation,
	installedBy string,
	config map[string]interface{},
) (*ent.TenantInstallation, error) {
	if err := s.provisionConnector(ctx, item, installation.TenantID, installedBy, config); err != nil {
		s.logger.Warnw("Connector provisioning failed during install",
			"item_id", item.ID, "tenant_id", installation.TenantID, "installation_id", installation.ID, "error", err)
		failed, updateErr := s.db.TenantInstallation.UpdateOne(installation).
			SetStatus(tenantinstallation.StatusFailed).
			SetErrorMessage(err.Error()).
			SetLastUpdatedAt(time.Now()).
			Save(ctx)
		if updateErr != nil {
			return nil, fmt.Errorf("failed to record installation failure: %w", updateErr)
		}
		return failed, fmt.Errorf("%w: %v", ErrMarketplaceConnectorNotReady, err)
	}

	active, err := s.db.TenantInstallation.UpdateOne(installation).
		SetStatus(tenantinstallation.StatusActive).
		ClearErrorMessage().
		SetLastUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to activate installation: %w", err)
	}

	s.logger.Infow("Item installed successfully",
		"tenant_id", installation.TenantID, "item_id", item.ID, "item_name", item.Name)
	return active, nil
}

// UninstallItem 租户卸载商品
func (s *Service) UninstallItem(ctx context.Context, tenantID, itemID int) error {
	// 查找安装记录
	installation, err := s.db.TenantInstallation.Query().
		Where(
			tenantinstallation.TenantID(tenantID),
			tenantinstallation.ItemID(itemID),
			tenantinstallation.StatusNEQ(tenantinstallation.StatusUninstalled),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrMarketplaceInstallationAbsent
		}
		return fmt.Errorf("failed to find installation: %w", err)
	}

	// 反向依赖闸门：先确认没有已安装组件依赖它，再动运行态。
	// 顺序不能反——先 revoke 连接器再判定不允许卸载，会留下
	// 「安装记录还是 active、连接器已经没了」的分裂状态。
	item, err := s.db.MarketplaceItem.Get(ctx, itemID)
	if err != nil && !ent.IsNotFound(err) {
		return fmt.Errorf("failed to load item: %w", err)
	}
	if err == nil {
		dependents, depErr := s.findDependents(ctx, tenantID, itemID, item.Name)
		if depErr != nil {
			return depErr
		}
		if len(dependents) > 0 {
			return &DependentsBlockUninstallError{Dependents: dependents}
		}
	}

	// 执行实际卸载：连接器必须在运行态真正注销。
	// 失败时不落 uninstalled——否则市场页显示「已卸载」，而连接器仍在收发消息、
	// 仍能接收回调，是比安装失败更危险的分裂状态（凭据仍活）。
	if err := s.revokeConnector(ctx, installation, tenantID); err != nil {
		s.logger.Warnw("Connector revocation failed during uninstall",
			"item_id", itemID, "tenant_id", tenantID, "error", err)
		return fmt.Errorf("%w: %v", ErrMarketplaceConnectorStillRunning, err)
	}

	// 更新状态为uninstalled
	_, err = s.db.TenantInstallation.UpdateOne(installation).
		SetStatus(tenantinstallation.StatusUninstalled).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to uninstall item: %w", err)
	}

	// 减少安装计数
	_, err = s.db.MarketplaceItem.UpdateOneID(itemID).
		AddInstallCount(-1).
		Save(ctx)
	if err != nil {
		s.logger.Warnw("Failed to decrement install count", "item_id", itemID, "error", err)
	}

	s.logger.Infow("Item uninstalled successfully", "tenant_id", tenantID, "item_id", itemID)
	return nil
}

// GetInstallation 获取租户的安装信息
func (s *Service) GetInstallation(ctx context.Context, tenantID, itemID int) (*ent.TenantInstallation, error) {
	installation, err := s.db.TenantInstallation.Query().
		Where(
			tenantinstallation.TenantID(tenantID),
			tenantinstallation.ItemID(itemID),
			tenantinstallation.StatusNEQ(tenantinstallation.StatusUninstalled),
		).
		WithItem().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrMarketplaceInstallationAbsent
		}
		return nil, fmt.Errorf("failed to get installation: %w", err)
	}
	return installation, nil
}

// ListInstallations 列出租户的所有已安装组件
func (s *Service) ListInstallations(ctx context.Context, tenantID int, status string) ([]*ent.TenantInstallation, error) {
	query := s.db.TenantInstallation.Query().
		Where(tenantinstallation.TenantID(tenantID)).
		WithItem()

	if status != "" {
		query = query.Where(tenantinstallation.StatusEQ(tenantinstallation.Status(status)))
	} else {
		query = query.Where(tenantinstallation.StatusNEQ(tenantinstallation.StatusUninstalled))
	}

	installations, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list installations: %w", err)
	}
	return installations, nil
}

// UpdateInstallationConfig 更新组件配置
func (s *Service) UpdateInstallationConfig(ctx context.Context, tenantID, itemID int, config map[string]interface{}) (*ent.TenantInstallation, error) {
	installation, err := s.GetInstallation(ctx, tenantID, itemID)
	if err != nil {
		return nil, err
	}

	// 出站配置带掩码，前端可能原样回写；掩码叶子还原为库中现值，避免真实密钥被 "***" 覆盖。
	config = RestoreMaskedConfig(config, installation.Config)

	// 获取 item 用于 schema 验证和类型判断
	item, err := s.db.MarketplaceItem.Get(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get marketplace item: %w", err)
	}

	// 验证配置是否符合商品的 ConfigSchema
	if err := s.validateConfigSchema(item, config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMarketplaceConfigSchemaInvalid, err)
	}

	// 先让运行态用新配置起来，成功后才落库。
	//
	// 反序（旧实现：先 SetConfig 再 provision）在 provision 失败时留下
	// 「DB 是新凭据、内存是旧凭据」的分裂状态——用户以为换了 token 已生效，
	// 实际仍在用旧的发消息，而唯一的痕迹只有一条 Warn 日志。
	if err := s.provisionConnector(ctx, item, tenantID, installation.InstalledBy, config); err != nil {
		s.logger.Warnw("Connector re-provisioning failed during config update",
			"item_id", itemID, "tenant_id", tenantID, "error", err)
		// 记录失败原因，但不动 config：运行态与 DB 都还是上一份可用配置。
		if _, logErr := s.db.TenantInstallation.UpdateOne(installation).
			SetErrorMessage(err.Error()).
			SetLastUpdatedAt(time.Now()).
			Save(ctx); logErr != nil {
			s.logger.Warnw("Failed to record connector config error", "item_id", itemID, "error", logErr)
		}
		return nil, fmt.Errorf("%w: %v", ErrMarketplaceConnectorNotReady, err)
	}

	update := s.db.TenantInstallation.UpdateOne(installation).
		SetConfig(config).
		SetLastUpdatedAt(time.Now()).
		ClearErrorMessage()
	// 之前停在 installing/failed 的连接器安装，凭据补齐即视为装好——
	// 这是「安装失败 → 补配置 → 变可用」闭环的最后一步。
	if item.Type == marketplaceitem.TypeConnector &&
		(installation.Status == tenantinstallation.StatusInstalling ||
			installation.Status == tenantinstallation.StatusFailed) {
		update = update.SetStatus(tenantinstallation.StatusActive)
	}
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	return updated, nil
}

// GetConnectorInstallation returns the active marketplace installation for a built-in connector.
// Built-in connector runtime names are short (for example "feishu") while marketplace item names
// may use a display slug (for example "feishu-connector"), so both are accepted.
func (s *Service) GetConnectorInstallation(ctx context.Context, tenantID int, connectorName string) (*ent.TenantInstallation, error) {
	installation, err := s.db.TenantInstallation.Query().
		Where(
			tenantinstallation.TenantID(tenantID),
			tenantinstallation.StatusNEQ(tenantinstallation.StatusUninstalled),
			tenantinstallation.HasItemWith(
				marketplaceitem.TypeEQ(marketplaceitem.TypeConnector),
				marketplaceitem.Or(
					marketplaceitem.NameEQ(connectorName),
					marketplaceitem.NameEQ(connectorName+"-connector"),
				),
			),
		).
		WithItem().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("connector installation not found")
		}
		return nil, fmt.Errorf("failed to get connector installation: %w", err)
	}
	return installation, nil
}

// MergeConnectorInstallationConfig merges a partial connector config into the tenant installation.
func (s *Service) MergeConnectorInstallationConfig(ctx context.Context, tenantID int, connectorName string, patch map[string]interface{}) (*ent.TenantInstallation, error) {
	installation, err := s.GetConnectorInstallation(ctx, tenantID, connectorName)
	if err != nil {
		return nil, err
	}
	config := make(map[string]interface{}, len(installation.Config)+len(patch))
	for k, v := range installation.Config {
		config[k] = v
	}
	for k, v := range patch {
		config[k] = v
	}
	updated, err := s.db.TenantInstallation.UpdateOne(installation).
		SetConfig(config).
		SetLastUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to merge connector installation config: %w", err)
	}
	return updated, nil
}

// provisionConnector 将 marketplace 安装的连接器注册到 connector.Manager。
// 仅对 type=connector 的商品生效，其他类型安全跳过。
// config 参数为可选的自定义配置；为 nil 时使用 item 默认值创建空配置。
func (s *Service) provisionConnector(ctx context.Context, item *ent.MarketplaceItem, tenantID int, installedBy string, config map[string]interface{}) error {
	if s.connectorManager == nil || item == nil {
		return nil
	}
	if item.Type != marketplaceitem.TypeConnector {
		return nil
	}
	connectorName := item.Name
	if len(connectorName) > 10 && connectorName[len(connectorName)-10:] == "-connector" {
		connectorName = connectorName[:len(connectorName)-10]
	}

	// 从 config 提取 credentials 和 settings
	var credentials map[string]string
	var settings map[string]interface{}
	if config != nil {
		if cred, ok := config["credentials"]; ok {
			credentials = toStringMap(cred)
		}
		if sett, ok := config["settings"]; ok {
			settings = toInterfaceMap(sett)
		}
	}

	cfg := connector.Config{
		TenantID:    tenantID,
		Name:        connectorName,
		Provider:    connectorName,
		Enabled:     true,
		Credentials: credentials,
		Settings:    settings,
		Labels: map[string]string{
			"marketplace_item_id": fmt.Sprintf("%d", item.ID),
			"marketplace_name":    item.Name,
			"marketplace_title":   item.Title,
		},
	}
	return s.connectorManager.Provision(ctx, cfg)
}

// revokeConnector 从 connector.Manager 中注销连接器实例。
// 仅对 type=connector 的商品生效。
func (s *Service) revokeConnector(ctx context.Context, installation *ent.TenantInstallation, tenantID int) error {
	if s.connectorManager == nil || installation == nil {
		return nil
	}
	// 获取 item 以确认类型和名称
	item, err := s.db.MarketplaceItem.Get(ctx, installation.ItemID)
	if err != nil {
		return fmt.Errorf("failed to get item for revoke: %w", err)
	}
	if item.Type != marketplaceitem.TypeConnector {
		return nil
	}
	connectorName := item.Name
	if len(connectorName) > 10 && connectorName[len(connectorName)-10:] == "-connector" {
		connectorName = connectorName[:len(connectorName)-10]
	}
	cfg := connector.Config{
		TenantID: tenantID,
		Name:     connectorName,
		Provider: connectorName,
	}
	s.connectorManager.Revoke(cfg)

	// 校验运行态确实注销了。Revoke 以 (tenant, name, provider) 为键删实例，
	// 历史上 provider 若被写成与安装时不同的值，键不匹配 → 实例残留而安装记录
	// 已被标成 uninstalled。这里按 name 复查，把「假卸载」变成显式错误。
	for _, running := range s.connectorManager.ListByTenant(tenantID) {
		if running.Name == connectorName {
			return fmt.Errorf("%w: %s", ErrMarketplaceConnectorStillRunning, connectorName)
		}
	}
	return nil
}

// validateConfigSchema 轻量级 JSON Schema 验证。
// 检查 config_schema 中声明的 required 字段是否存在，以及基本类型匹配。
// 如果 item 没有 config_schema，直接通过。
func (s *Service) validateConfigSchema(item *ent.MarketplaceItem, config map[string]interface{}) error {
	if item == nil || item.ConfigSchema == nil {
		return nil
	}
	schema := item.ConfigSchema

	// 检查 required 字段
	if required, ok := schema["required"].([]interface{}); ok {
		for _, req := range required {
			if field, ok := req.(string); ok {
				if _, exists := config[field]; !exists {
					return fmt.Errorf("missing required field: %s", field)
				}
			}
		}
	}

	// 检查 properties 的类型
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for field, propDef := range properties {
			val, exists := config[field]
			if !exists {
				continue // 非必填字段缺失时跳过
			}
			if propMap, ok := propDef.(map[string]interface{}); ok {
				if expectedType, ok := propMap["type"].(string); ok {
					if err := validateFieldType(field, val, expectedType); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// validateFieldType 验证单个字段的类型
func validateFieldType(field string, val interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := val.(string); !ok {
			return fmt.Errorf("field %s must be a string", field)
		}
	case "number", "integer":
		switch val.(type) {
		case float64, int, int64:
			// OK
		default:
			return fmt.Errorf("field %s must be a number", field)
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("field %s must be a boolean", field)
		}
	case "object":
		if _, ok := val.(map[string]interface{}); !ok {
			return fmt.Errorf("field %s must be an object", field)
		}
	case "array":
		if _, ok := val.([]interface{}); !ok {
			return fmt.Errorf("field %s must be an array", field)
		}
	}
	return nil
}

// toStringMap 将 interface{} 转为 map[string]string
func toStringMap(val interface{}) map[string]string {
	out := make(map[string]string)
	if m, ok := val.(map[string]interface{}); ok {
		for k, v := range m {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}

// toInterfaceMap 将 interface{} 转为 map[string]interface{}
func toInterfaceMap(val interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	if m, ok := val.(map[string]interface{}); ok {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}
