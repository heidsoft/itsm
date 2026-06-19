package marketplace

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/ent/tenantinstallation"
	"go.uber.org/zap"
)

// Service 市场服务
type Service struct {
	db     *ent.Client
	logger *zap.SugaredLogger
}

// NewService 创建市场服务
func NewService(db *ent.Client, logger *zap.SugaredLogger) *Service {
	return &Service{
		db:     db,
		logger: logger,
	}
}

// ListItems 查询市场商品列表
func (s *Service) ListItems(ctx context.Context, itemType, category, search string, isOfficial *bool, page, pageSize int) (*ent.MarketplaceItemList, int, error) {
	query := s.db.MarketplaceItem.Query().
		Where(marketplaceitem.StatusEQ("published"))

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
				marketplaceitem.TitleContains(search),
				marketplaceitem.DescriptionContains(search),
				marketplaceitem.TagsContains(search),
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

	return &ent.MarketplaceItemList{
		Items: items,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, total, nil
}

// GetItem 获取商品详情
func (s *Service) GetItem(ctx context.Context, itemID int) (*ent.MarketplaceItem, error) {
	item, err := s.db.MarketplaceItem.Query().
		Where(marketplaceitem.ID(itemID)).
		Where(marketplaceitem.StatusEQ("published")).
		WithVersions().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("item not found")
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}
	return item, nil
}

// InstallItem 租户安装商品
func (s *Service) InstallItem(ctx context.Context, tenantID, itemID int, installedBy string) (*ent.TenantInstallation, error) {
	// 检查商品是否存在
	item, err := s.db.MarketplaceItem.Get(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}

	// 检查是否已经安装
	exists, err := s.db.TenantInstallation.Query().
		Where(
			tenantinstallation.TenantID(tenantID),
			tenantinstallation.ItemID(itemID),
			tenantinstallation.StatusNEQ("uninstalled"),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check installation: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("item already installed")
	}

	// 创建安装记录
	installation, err := s.db.TenantInstallation.Create().
		SetTenantID(tenantID).
		SetItemID(itemID).
		SetInstalledVersion(item.LatestVersion).
		SetStatus("installing").
		SetInstalledBy(installedBy).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create installation: %w", err)
	}

	// 增加安装计数
	_, err = s.db.MarketplaceItem.UpdateOneID(itemID).
		AddInstallCount(1).
		Save(ctx)
	if err != nil {
		s.logger.Warnw("Failed to increment install count", "item_id", itemID, "error", err)
	}

	// TODO: 实际安装逻辑：加载连接器/技能/插件，注册到系统中

	// 更新安装状态为active
	installation, err = s.db.TenantInstallation.UpdateOne(installation).
		SetStatus("active").
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
			tenantinstallation.StatusNEQ("uninstalled"),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("installation not found")
		}
		return fmt.Errorf("failed to find installation: %w", err)
	}

	// TODO: 实际卸载逻辑：从系统中注销连接器/技能/插件，清理资源

	// 更新状态为uninstalled
	_, err = s.db.TenantInstallation.UpdateOne(installation).
		SetStatus("uninstalled").
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
			tenantinstallation.StatusNEQ("uninstalled"),
		).
		WithItem().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("installation not found")
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
		query = query.Where(tenantinstallation.StatusEQ(status))
	} else {
		query = query.Where(tenantinstallation.StatusNEQ("uninstalled"))
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

	// TODO: 验证配置是否符合Schema

	// 更新配置
	updated, err := s.db.TenantInstallation.UpdateOne(installation).
		SetConfig(config).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	// TODO: 通知组件配置更新

	return updated, nil
}
