package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/ent/tenantinstallation"
)

// 版本升级执行路径（2026-11-03）。
//
// 此前的状态：TenantInstallation.auto_upgrade 默认 true、installed_version 只在
// 安装时写过一次，目录侧 PublishItemVersion 会把 latest_version 推进到新版本，
// 但没有任何代码把「目录已有新版本」变成「租户真的用上了新版本」。结果是：
// 目录管理员发布 1.1.0 后，租户界面显示已安装 1.1.0（因为新安装才取 latest_version），
// 存量租户永远停在 1.0.x，而 auto_upgrade 开关点了也没用——它没有消费者。
//
// 本文件的三条不变量：
//  1. 升级目标必须是已发布的版本行。latest_version 是「目录当前指向」，
//     版本行的 status 才是「这个发布物存在且可用」；只看前者能升级到草稿甚至已撤回版本。
//  2. 依赖闸门在升级时重新执行。安装期校验过的那次针对的是旧版本的依赖声明，
//     新版本可能新增前置组件；不做这一步，升级就是把悬空依赖装进租户。
//  3. 运行态先起来，版本号才换。provision 失败时保持 installed_version 与 status 不变，
//     只把原因写进 error_message——租户跑的仍是旧版本，界面必须这么说。
//     installed_version 与审计行在同一个事务里写，避免出现「升了级但查不到是谁升的」。

var (
	// ErrMarketplaceUpgradeTargetInvalid 升级目标不可用：既不是目录里已发布的版本，
	// 或当前没有可升级的目标。属于冲突/参数问题，不是内部错误。
	ErrMarketplaceUpgradeTargetInvalid = errors.New("marketplace upgrade target is not available")

	// ErrMarketplaceUpgradeBlocked 升级被运行态或依赖闸门挡下，安装记录保持原版本。
	ErrMarketplaceUpgradeBlocked = errors.New("marketplace upgrade blocked")
)

// UpgradeResult 一次升级调用的结果。
//
// Upgraded=false 且 Installation 非空表示「已在目标版本，无需升级」：这是幂等成功，
// 不是错误——自动升级 pass 每分钟都会碰到大量已最新记录，用错误表达会让失败率虚高。
type UpgradeResult struct {
	Installation *ent.TenantInstallation
	FromVersion  string
	ToVersion    string
	Upgraded     bool
	// RolledBack 区分「升级」与「回滚」。降级同样走 UpgradeInstallation，
	// 响应文案不能一律说「已从 X 升级到 Y」。
	RolledBack bool
}

// normalizeUpgradeVersion 去掉版本号前的 v 别名并去空白。
//
// 只归一形态、不校验格式：格式是否合法由「能不能查到同名的已发布版本行」决定，
// 查不到就落到 upgrade-target-invalid，没必要在这里再维护一套版本号规则。
func normalizeUpgradeVersion(raw string) string {
	v := strings.TrimSpace(raw)
	return strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
}

// compareSemverStrings 按数字段比较版本号字符串，用于判断目标版本比当前更旧。
// 只用于展示文案，不参与准入判定（准入由「是否查得到已发布版本行」决定）；
// 非数字段按 0 处理，因此带后缀的预发布版本不会被误判为升级。
func compareSemverStrings(a, b string) int {
	pa := strings.Split(normalizeUpgradeVersion(a), ".")
	pb := strings.Split(normalizeUpgradeVersion(b), ".")
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		x, y := 0, 0
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}

// UpgradeInstallation 把租户的安装记录升到 targetVersion（空则升到目录的 latest_version）。
//
// targetVersion 允许写成 1.1.0 或 v1.1.0：目录入库时统一去掉了 v 前缀，
// 但界面与脚本习惯保留前缀，这里做归一化而不是要求调用方猜存储形态。
// 回滚同样走这里——目标版本比当前更旧也允许，只要它是已发布版本行。
// 这就是「出问题先退一版」的唯一入口；若按版本号比大小拒绝降级，回滚只能改数据库。
func (s *Service) UpgradeInstallation(ctx context.Context, tenantID, itemID int, targetVersion, actor string) (*UpgradeResult, error) {
	item, err := s.db.MarketplaceItem.Get(ctx, itemID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrMarketplaceItemNotFound
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}
	if !IsDeliverableType(item.Type) {
		return nil, fmt.Errorf("%w: type=%s", ErrMarketplaceDeliveryUnsupported, item.Type)
	}
	installation, err := s.GetInstallation(ctx, tenantID, itemID)
	if err != nil {
		return nil, err
	}

	target := normalizeUpgradeVersion(targetVersion)
	if target == "" {
		target = strings.TrimSpace(item.LatestVersion)
	}
	if target == "" {
		return nil, fmt.Errorf("%w: Product %s has no version to upgrade to", ErrMarketplaceUpgradeTargetInvalid, item.Name)
	}
	if target == normalizeUpgradeVersion(installation.InstalledVersion) {
		return &UpgradeResult{Installation: installation, FromVersion: target, ToVersion: target}, nil
	}
	if _, err := s.publishedVersionRow(ctx, itemID, target); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMarketplaceUpgradeTargetInvalid, err)
	}

	// 依赖闸门针对「目标版本的声明」重新校验：旧版本装得上不代表新版本装得上。
	if err := s.validateDependencies(ctx, tenantID, itemID, item.Name, target); err != nil {
		return nil, err
	}

	// 运行态先行。这里沿用当前安装记录的配置：连接器换版本通常不换凭据，
	// 而技能/连接器的 provision 都是幂等的（按 (tenant, name, provider) 覆盖实例）。
	from := installation.InstalledVersion
	if err := s.provisionRuntime(ctx, item, tenantID, installation.InstalledBy, target, installation.Config); err != nil {
		s.logger.Warnw("Marketplace upgrade blocked by runtime provisioning",
			"tenant_id", tenantID, "item_id", itemID, "item_type", string(item.Type),
			"from_version", from, "to_version", target, "error", err)
		// 版本没换、状态没动：旧发布物仍在运行，只记录失败原因供界面展示。
		if _, logErr := s.db.TenantInstallation.UpdateOne(installation).
			SetErrorMessage(fmt.Sprintf("升级到 %s 失败：%v", target, err)).
			SetLastUpdatedAt(time.Now()).
			Save(ctx); logErr != nil {
			s.logger.Warnw("Failed to record marketplace upgrade error", "tenant_id", tenantID, "item_id", itemID, "error", logErr)
		}
		return nil, fmt.Errorf("%w: %v", ErrMarketplaceUpgradeBlocked, err)
	}

	upgraded, err := s.recordUpgrade(ctx, installation, item, from, target, actor)
	if err != nil {
		return nil, err
	}
	rolledBack := compareSemverStrings(target, from) < 0
	s.logger.Infow("Marketplace installation version changed",
		"tenant_id", tenantID, "item_id", itemID, "item_name", item.Name,
		"from_version", from, "to_version", target, "rolled_back", rolledBack, "actor", actor)
	return &UpgradeResult{Installation: upgraded, FromVersion: from, ToVersion: target, Upgraded: true, RolledBack: rolledBack}, nil
}

// recordUpgrade 在同一事务里写版本号与审计行。
//
// 之前停在 installing/failed 的记录升到新版本即视为装好（与 config 补齐的路径同口径）。
func (s *Service) recordUpgrade(
	ctx context.Context,
	installation *ent.TenantInstallation,
	item *ent.MarketplaceItem,
	from, to, actor string,
) (*ent.TenantInstallation, error) {
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	update := tx.TenantInstallation.UpdateOne(installation).
		SetInstalledVersion(to).
		SetLastUpdatedAt(time.Now()).
		ClearErrorMessage()
	if installation.Status == tenantinstallation.StatusInstalling ||
		installation.Status == tenantinstallation.StatusFailed {
		update = update.SetStatus(tenantinstallation.StatusActive)
	}
	updated, err := update.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to record upgraded version: %w", err)
	}
	if err := writeMarketplaceAudit(ctx, tx.Client(), installation.TenantID, actor, item.ID,
		fmt.Sprintf("service://marketplace/items/%d/upgrade", item.ID),
		map[string]interface{}{
			"itemId":         item.ID,
			"itemType":       string(item.Type),
			"fromVersion":    from,
			"toVersion":      to,
			"actor":          actor,
			"installationId": installation.ID,
		}); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit upgrade: %w", err)
	}
	return s.withItemEdge(ctx, updated), nil
}

// SetAutoUpgrade 开关某条安装记录的自动升级意愿。
//
// 这个字段此前只能被安装写入、之后无人读取也没人改，是个假开关。
// 现在它是 RunAutoUpgradePass 的准入条件之一，因此改动必须留审计：
// 「谁同意了这个租户被静默升级」在故障回溯时是必需的一条。
func (s *Service) SetAutoUpgrade(ctx context.Context, tenantID, itemID int, enabled bool, actor string) (*ent.TenantInstallation, error) {
	installation, err := s.GetInstallation(ctx, tenantID, itemID)
	if err != nil {
		return nil, err
	}
	if installation.AutoUpgrade == enabled {
		return installation, nil
	}

	tx, err := s.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	updated, err := tx.TenantInstallation.UpdateOne(installation).
		SetAutoUpgrade(enabled).
		SetLastUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to set auto upgrade: %w", err)
	}
	if err := writeMarketplaceAudit(ctx, tx.Client(), tenantID, actor, itemID,
		fmt.Sprintf("service://marketplace/items/%d/auto-upgrade", itemID),
		map[string]interface{}{
			"itemId":         itemID,
			"autoUpgrade":    enabled,
			"actor":          actor,
			"installationId": installation.ID,
		}); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit auto upgrade change: %w", err)
	}
	s.logger.Infow("Marketplace auto upgrade preference updated",
		"tenant_id", tenantID, "item_id", itemID, "enabled", enabled, "actor", actor)
	return s.withItemEdge(ctx, updated), nil
}

// AutoUpgradeSummary 一轮自动升级的统计。
type AutoUpgradeSummary struct {
	Scanned  int
	Upgraded int
	Skipped  int
	Failed   int
	// Failures 逐条「哪台没升成」，供日志与运维界面定位；不是错误本身。
	Failures []string
}

// RunAutoUpgradePass 扫描本进程内所有开启自动升级的生效安装并升到目录最新版本。
//
// 调用方必须传系统上下文（跨租户后台任务），并且只在 worker/all 进程里跑：
// api 进程执行会让每个副本都抢着改同一批安装记录。
// 单条失败不影响其余租户——一个商品的坏版本行不该冻结整个市场的升级。
func (s *Service) RunAutoUpgradePass(ctx context.Context) (AutoUpgradeSummary, error) {
	installations, err := s.db.TenantInstallation.Query().
		Where(
			tenantinstallation.AutoUpgrade(true),
			tenantinstallation.StatusEQ(tenantinstallation.StatusActive),
			tenantinstallation.HasItemWith(
				marketplaceitem.TypeIn(deliverableTypes()...),
				marketplaceitem.StatusEQ(marketplaceitem.StatusPublished),
			),
		).
		WithItem().
		All(ctx)
	if err != nil {
		return AutoUpgradeSummary{}, fmt.Errorf("failed to list auto-upgrade installations: %w", err)
	}

	summary := AutoUpgradeSummary{Scanned: len(installations), Failures: []string{}}
	for _, installation := range installations {
		item, err := installation.Edges.ItemOrErr()
		if err != nil {
			summary.Skipped++
			summary.Failures = append(summary.Failures, fmt.Sprintf("installation=%d 没有对应商品：%v", installation.ID, err))
			continue
		}
		if strings.TrimSpace(item.LatestVersion) == "" || item.LatestVersion == installation.InstalledVersion {
			summary.Skipped++
			continue
		}
		result, err := s.UpgradeInstallation(ctx, installation.TenantID, item.ID, "", "auto-upgrade")
		if err != nil {
			summary.Failed++
			summary.Failures = append(summary.Failures, fmt.Sprintf("tenant=%d item=%s -> %s：%v",
				installation.TenantID, item.Name, item.LatestVersion, err))
			s.logger.Warnw("Auto upgrade failed",
				"tenant_id", installation.TenantID, "item_id", item.ID, "item_name", item.Name,
				"target_version", item.LatestVersion, "error", err)
			continue
		}
		if result.Upgraded {
			summary.Upgraded++
		} else {
			summary.Skipped++
		}
	}
	return summary, nil
}

// writeMarketplaceAudit 记录一条领域级审计。
//
// action 用 update：中间件的 AuditableAction 词表没有 upgrade，自造词会让审计
// 过滤器漏掉这类事件。request_body 只放 ID、版本号与操作人，不含配置内容——
// 连接器凭据可能就在 config 里，绝不能进审计。
func writeMarketplaceAudit(ctx context.Context, client *ent.Client, tenantID int, actor string, itemID int, path string, details map[string]interface{}) error {
	body, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal marketplace audit details: %w", err)
	}
	create := client.AuditLog.Create().
		SetTenantID(tenantID).
		SetResource(fmt.Sprintf("marketplace_item:%d", itemID)).
		SetAction("update").
		SetPath(path).
		SetMethod("MARKETPLACE").
		SetStatusCode(0).
		SetRequestBody(string(body))
	// actor 不是数字时（系统任务写 "auto-upgrade"）不写 user_id，
	// 而不是填 0——0 会在审计列表里显示成一个不存在的用户。
	if userID, convErr := strconv.Atoi(strings.TrimSpace(actor)); convErr == nil {
		create = create.SetUserID(userID)
	}
	if _, err := create.Save(ctx); err != nil {
		return fmt.Errorf("failed to write marketplace audit: %w", err)
	}
	return nil
}
