package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/asset"

	"go.uber.org/zap"
)

// AssetService 资产管理服务
type AssetService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewAssetService 创建资产管理服务
func NewAssetService(client *ent.Client, logger *zap.SugaredLogger) *AssetService {
	return &AssetService{
		client: client,
		logger: logger,
	}
}

// CreateAsset 创建资产
func (s *AssetService) CreateAsset(ctx context.Context, req *dto.CreateAssetRequest, tenantID int) (*dto.AssetResponse, error) {
	assetEntity, err := s.client.Asset.Create().
		SetAssetNumber(req.AssetNumber).
		SetName(req.Name).
		SetDescription(req.Description).
		SetType(req.Type).
		SetStatus(string(dto.AssetStatusAvailable)).
		SetCategory(req.Category).
		SetSubcategory(req.Subcategory).
		SetTenantID(tenantID).
		SetNillableCiID(req.CIID).
		SetNillableAssignedTo(req.AssignedTo).
		SetNillableLocationID(req.LocationID).
		SetSerialNumber(req.SerialNumber).
		SetModel(req.Model).
		SetManufacturer(req.Manufacturer).
		SetVendor(req.Vendor).
		SetPurchaseDate(req.PurchaseDate).
		SetNillablePurchasePrice(req.PurchasePrice).
		SetWarrantyExpiry(req.WarrantyExpiry).
		SetSupportExpiry(req.SupportExpiry).
		SetLocation(req.Location).
		SetDepartment(req.Department).
		SetNillableParentAssetID(req.ParentAssetID).
		SetSpecifications(req.Specifications).
		SetCustomFields(req.CustomFields).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create asset", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to create asset: %w", err)
	}

	// 设置标签
	if len(req.Tags) > 0 {
		assetEntity.Update().SetTags(req.Tags).Exec(ctx)
	}

	response := dto.ToAssetResponse(assetEntity)

	s.logger.Infow("Asset created successfully", "asset_id", assetEntity.ID, "tenant_id", tenantID)
	return response, nil
}

// GetAssetByID 根据ID获取资产
func (s *AssetService) GetAssetByID(ctx context.Context, id, tenantID int) (*dto.AssetResponse, error) {
	assetEntity, err := s.client.Asset.Query().
		Where(asset.IDEQ(id), asset.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get asset", "error", err, "asset_id", id)
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	return dto.ToAssetResponse(assetEntity), nil
}

// ListAssets 获取资产列表
func (s *AssetService) ListAssets(ctx context.Context, tenantID int, page, pageSize int, assetType, status, category string) (*dto.AssetListResponse, error) {
	query := s.client.Asset.Query().Where(asset.TenantIDEQ(tenantID))

	if assetType != "" {
		query = query.Where(asset.TypeEQ(assetType))
	}
	if status != "" {
		query = query.Where(asset.StatusEQ(status))
	}
	if category != "" {
		query = query.Where(asset.CategoryEQ(category))
	}

	// 统计总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count assets", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to count assets: %w", err)
	}

	// 分页查询
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}

	assetEntities, err := query.Order(ent.Desc(asset.FieldCreatedAt)).All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list assets", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}

	assets := dto.ToAssetResponseList(assetEntities)

	return &dto.AssetListResponse{
		Total:  total,
		Assets: assets,
	}, nil
}

// UpdateAsset 更新资产
func (s *AssetService) UpdateAsset(ctx context.Context, id, tenantID int, req *dto.UpdateAssetRequest) (*dto.AssetResponse, error) {
	assetEntity, err := s.client.Asset.Query().
		Where(asset.IDEQ(id), asset.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get asset", "error", err, "asset_id", id)
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	update := assetEntity.Update()

	if req.Name != nil {
		update.SetName(*req.Name)
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
	}
	if req.Type != nil {
		update.SetType(*req.Type)
	}
	if req.Category != nil {
		update.SetCategory(*req.Category)
	}
	if req.Subcategory != nil {
		update.SetSubcategory(*req.Subcategory)
	}
	if req.CIID != nil {
		update.SetCiID(*req.CIID)
	}
	if req.AssignedTo != nil {
		update.SetAssignedTo(*req.AssignedTo)
	}
	if req.LocationID != nil {
		update.SetLocationID(*req.LocationID)
	}
	if req.SerialNumber != nil {
		update.SetSerialNumber(*req.SerialNumber)
	}
	if req.Model != nil {
		update.SetModel(*req.Model)
	}
	if req.Manufacturer != nil {
		update.SetManufacturer(*req.Manufacturer)
	}
	if req.Vendor != nil {
		update.SetVendor(*req.Vendor)
	}
	if req.PurchaseDate != nil {
		update.SetPurchaseDate(*req.PurchaseDate)
	}
	if req.PurchasePrice != nil {
		update.SetPurchasePrice(*req.PurchasePrice)
	}
	if req.WarrantyExpiry != nil {
		update.SetWarrantyExpiry(*req.WarrantyExpiry)
	}
	if req.SupportExpiry != nil {
		update.SetSupportExpiry(*req.SupportExpiry)
	}
	if req.Location != nil {
		update.SetLocation(*req.Location)
	}
	if req.Department != nil {
		update.SetDepartment(*req.Department)
	}
	if req.ParentAssetID != nil {
		update.SetParentAssetID(*req.ParentAssetID)
	}
	if req.Specifications != nil {
		update.SetSpecifications(req.Specifications)
	}
	if req.CustomFields != nil {
		update.SetCustomFields(req.CustomFields)
	}
	if len(req.Tags) > 0 {
		update.SetTags(req.Tags)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update asset", "error", err, "asset_id", id)
		return nil, fmt.Errorf("failed to update asset: %w", err)
	}

	return dto.ToAssetResponse(updated), nil
}

// UpdateAssetStatus 更新资产状态
func (s *AssetService) UpdateAssetStatus(ctx context.Context, id, tenantID int, status string, assignedTo *int) (*dto.AssetResponse, error) {
	assetEntity, err := s.client.Asset.Query().
		Where(asset.IDEQ(id), asset.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get asset", "error", err, "asset_id", id)
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	update := assetEntity.Update().SetStatus(status)

	if assignedTo != nil {
		update.SetAssignedTo(*assignedTo)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update asset status", "error", err, "asset_id", id, "status", status)
		return nil, fmt.Errorf("failed to update asset status: %w", err)
	}

	s.logger.Infow("Asset status updated", "asset_id", id, "status", status)
	return dto.ToAssetResponse(updated), nil
}

// DeleteAsset 删除资产
func (s *AssetService) DeleteAsset(ctx context.Context, id, tenantID int) error {
	_, err := s.client.Asset.Delete().
		Where(asset.IDEQ(id), asset.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete asset", "error", err, "asset_id", id)
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	s.logger.Infow("Asset deleted", "asset_id", id)
	return nil
}

// GetAssetStats 获取资产统计
func (s *AssetService) GetAssetStats(ctx context.Context, tenantID int) (*dto.AssetStatsResponse, error) {
	stats := &dto.AssetStatsResponse{}

	total, err := s.client.Asset.Query().Where(asset.TenantIDEQ(tenantID)).Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count assets", "error", err)
		return nil, err
	}
	stats.Total = total

	// 统计各状态数量
	available, _ := s.client.Asset.Query().Where(asset.TenantIDEQ(tenantID), asset.StatusEQ(string(dto.AssetStatusAvailable))).Count(ctx)
	inUse, _ := s.client.Asset.Query().Where(asset.TenantIDEQ(tenantID), asset.StatusEQ(string(dto.AssetStatusInUse))).Count(ctx)
	maintenance, _ := s.client.Asset.Query().Where(asset.TenantIDEQ(tenantID), asset.StatusEQ(string(dto.AssetStatusMaintenance))).Count(ctx)
	retired, _ := s.client.Asset.Query().Where(asset.TenantIDEQ(tenantID), asset.StatusEQ(string(dto.AssetStatusRetired))).Count(ctx)
	disposed, _ := s.client.Asset.Query().Where(asset.TenantIDEQ(tenantID), asset.StatusEQ(string(dto.AssetStatusDisposed))).Count(ctx)

	stats.Available = available
	stats.InUse = inUse
	stats.Maintenance = maintenance
	stats.Retired = retired
	stats.Disposed = disposed

	return stats, nil
}

// AssignAsset 分配资产
func (s *AssetService) AssignAsset(ctx context.Context, id, tenantID, userID int) (*dto.AssetResponse, error) {
	assetEntity, err := s.client.Asset.Query().
		Where(asset.IDEQ(id), asset.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get asset", "error", err, "asset_id", id)
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	updated, err := assetEntity.Update().
		SetAssignedTo(userID).
		SetStatus(string(dto.AssetStatusInUse)).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to assign asset", "error", err, "asset_id", id, "user_id", userID)
		return nil, fmt.Errorf("failed to assign asset: %w", err)
	}

	s.logger.Infow("Asset assigned", "asset_id", id, "user_id", userID)
	return dto.ToAssetResponse(updated), nil
}

// RetireAsset 退役资产
func (s *AssetService) RetireAsset(ctx context.Context, id, tenantID int, reason string) (*dto.AssetResponse, error) {
	assetEntity, err := s.client.Asset.Query().
		Where(asset.IDEQ(id), asset.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get asset", "error", err, "asset_id", id)
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	updated, err := assetEntity.Update().
		SetStatus(string(dto.AssetStatusRetired)).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to retire asset", "error", err, "asset_id", id)
		return nil, fmt.Errorf("failed to retire asset: %w", err)
	}

	s.logger.Infow("Asset retired", "asset_id", id, "reason", reason)
	return dto.ToAssetResponse(updated), nil
}
