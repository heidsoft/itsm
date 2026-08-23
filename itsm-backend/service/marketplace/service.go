package marketplace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"itsm-backend/connector"
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
	ErrMarketplaceItemNotFound       = errors.New("marketplace item not found")
	ErrMarketplaceItemUnavailable    = errors.New("marketplace item unavailable")
	ErrMarketplaceInstallationAbsent = errors.New("marketplace installation not found")
	ErrMarketplaceInstalledByMissing = errors.New("marketplace installed_by is required")
	ErrMarketplaceConfigSchemaInvalid = errors.New("config does not match item schema")
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
func (s *Service) ListItems(ctx context.Context, itemType, category, search string, isOfficial *bool, page, pageSize int) ([]*ent.MarketplaceItem, int, error) {
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

	return items, total, nil
}

// GetItem 获取商品详情
func (s *Service) GetItem(ctx context.Context, itemID int) (*ent.MarketplaceItem, error) {
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
	return item, nil
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

	reactivated, err := s.db.TenantInstallation.UpdateOneID(history.ID).
		SetInstalledVersion(item.LatestVersion).
		SetStatus(tenantinstallation.StatusActive).
		SetInstalledBy(installedBy).
		SetErrorMessage("").
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
	return reactivated, nil
}

// InstallItem 租户安装商品
func (s *Service) InstallItem(ctx context.Context, tenantID, itemID int, installedBy string) (*ent.TenantInstallation, error) {
	// P2-04 修复：业务保护四道闸
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
	// 4) 已存在有效安装则幂等返回当前安装，避免重复安装报错
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
	} else {
		s.logger.Infow("Item already installed, returning existing installation", "tenant_id", tenantID, "item_id", itemID, "installation_id", existingInstallation.ID)
		return existingInstallation, nil
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

	// 执行实际安装：如果是连接器类型，provision 到 connector.Manager
	if err := s.provisionConnector(ctx, item, tenantID, installedBy, nil); err != nil {
		s.logger.Warnw("Connector provisioning failed during install", "item_id", itemID, "error", err)
		// 不阻塞安装流程，连接器可在后续配置更新时重新 provision
	}

	// 更新安装状态为active
	installation, err = s.db.TenantInstallation.UpdateOne(installation).
		SetStatus(tenantinstallation.StatusActive).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to activate installation: %w", err)
	}

	s.logger.Infow("Item installed successfully", "tenant_id", tenantID, "item_id", itemID, "item_name", item.Name)
	return installation, nil
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

	// 执行实际卸载：如果是连接器类型，从 connector.Manager 中注销
	if err := s.revokeConnector(ctx, installation, tenantID); err != nil {
		s.logger.Warnw("Connector revocation failed during uninstall", "item_id", itemID, "error", err)
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

	// 获取 item 用于 schema 验证和类型判断
	item, err := s.db.MarketplaceItem.Get(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get marketplace item: %w", err)
	}

	// 验证配置是否符合商品的 ConfigSchema
	if err := s.validateConfigSchema(item, config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMarketplaceConfigSchemaInvalid, err)
	}

	// 更新配置
	updated, err := s.db.TenantInstallation.UpdateOne(installation).
		SetConfig(config).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	// 通知组件配置更新：如果是连接器类型，用新配置重新 provision
	if err := s.provisionConnector(ctx, item, tenantID, installation.InstalledBy, config); err != nil {
		s.logger.Warnw("Connector re-provisioning failed during config update", "item_id", itemID, "error", err)
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
