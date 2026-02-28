package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/assetlicense"

	"go.uber.org/zap"
)

// AssetLicenseService 许可证管理服务
type AssetLicenseService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewAssetLicenseService 创建许可证管理服务
func NewAssetLicenseService(client *ent.Client, logger *zap.SugaredLogger) *AssetLicenseService {
	return &AssetLicenseService{
		client: client,
		logger: logger,
	}
}

// CreateLicense 创建许可证
func (s *AssetLicenseService) CreateLicense(ctx context.Context, req *dto.CreateLicenseRequest, tenantID int) (*dto.LicenseResponse, error) {
	licenseEntity, err := s.client.AssetLicense.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetVendor(req.Vendor).
		SetLicenseType(req.LicenseType).
		SetTotalQuantity(req.TotalQuantity).
		SetUsedQuantity(0).
		SetTenantID(tenantID).
		SetNillableAssetID(req.AssetID).
		SetPurchaseDate(req.PurchaseDate).
		SetNillablePurchasePrice(req.PurchasePrice).
		SetExpiryDate(req.ExpiryDate).
		SetSupportVendor(req.SupportVendor).
		SetSupportContact(req.SupportContact).
		SetRenewalCost(req.RenewalCost).
		SetStatus(string(dto.LicenseStatusActive)).
		SetNotes(req.Notes).
		SetUsers(req.Users).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create license", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to create license: %w", err)
	}

	// 设置标签
	if len(req.Tags) > 0 {
		licenseEntity.Update().SetTags(req.Tags).Exec(ctx)
	}

	response := dto.ToLicenseResponse(licenseEntity)

	s.logger.Infow("License created successfully", "license_id", licenseEntity.ID, "tenant_id", tenantID)
	return response, nil
}

// GetLicenseByID 根据ID获取许可证
func (s *AssetLicenseService) GetLicenseByID(ctx context.Context, id, tenantID int) (*dto.LicenseResponse, error) {
	licenseEntity, err := s.client.AssetLicense.Query().
		Where(assetlicense.IDEQ(id), assetlicense.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get license", "error", err, "license_id", id)
		return nil, fmt.Errorf("failed to get license: %w", err)
	}

	return dto.ToLicenseResponse(licenseEntity), nil
}

// ListLicenses 获取许可证列表
func (s *AssetLicenseService) ListLicenses(ctx context.Context, tenantID int, page, pageSize int, licenseType, status string) (*dto.LicenseListResponse, error) {
	query := s.client.AssetLicense.Query().Where(assetlicense.TenantIDEQ(tenantID))

	if licenseType != "" {
		query = query.Where(assetlicense.LicenseTypeEQ(licenseType))
	}
	if status != "" {
		query = query.Where(assetlicense.StatusEQ(status))
	}

	// 统计总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count licenses", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to count licenses: %w", err)
	}

	// 分页查询
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}

	licenseEntities, err := query.Order(ent.Desc(assetlicense.FieldCreatedAt)).All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list licenses", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to list licenses: %w", err)
	}

	licenses := dto.ToLicenseResponseList(licenseEntities)

	return &dto.LicenseListResponse{
		Total:    total,
		Licenses: licenses,
	}, nil
}

// UpdateLicense 更新许可证
func (s *AssetLicenseService) UpdateLicense(ctx context.Context, id, tenantID int, req *dto.UpdateLicenseRequest) (*dto.LicenseResponse, error) {
	licenseEntity, err := s.client.AssetLicense.Query().
		Where(assetlicense.IDEQ(id), assetlicense.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get license", "error", err, "license_id", id)
		return nil, fmt.Errorf("failed to get license: %w", err)
	}

	update := licenseEntity.Update()

	if req.Name != nil {
		update.SetName(*req.Name)
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
	}
	if req.Vendor != nil {
		update.SetVendor(*req.Vendor)
	}
	if req.LicenseType != nil {
		update.SetLicenseType(*req.LicenseType)
	}
	if req.TotalQuantity != nil {
		update.SetTotalQuantity(*req.TotalQuantity)
	}
	if req.UsedQuantity != nil {
		update.SetUsedQuantity(*req.UsedQuantity)
	}
	if req.AssetID != nil {
		update.SetAssetID(*req.AssetID)
	}
	if req.PurchaseDate != nil {
		update.SetPurchaseDate(*req.PurchaseDate)
	}
	if req.PurchasePrice != nil {
		update.SetPurchasePrice(*req.PurchasePrice)
	}
	if req.ExpiryDate != nil {
		update.SetExpiryDate(*req.ExpiryDate)
	}
	if req.SupportVendor != nil {
		update.SetSupportVendor(*req.SupportVendor)
	}
	if req.SupportContact != nil {
		update.SetSupportContact(*req.SupportContact)
	}
	if req.RenewalCost != nil {
		update.SetRenewalCost(*req.RenewalCost)
	}
	if req.Notes != nil {
		update.SetNotes(*req.Notes)
	}
	if len(req.Users) > 0 {
		update.SetUsers(req.Users)
	}
	if len(req.Tags) > 0 {
		update.SetTags(req.Tags)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update license", "error", err, "license_id", id)
		return nil, fmt.Errorf("failed to update license: %w", err)
	}

	return dto.ToLicenseResponse(updated), nil
}

// DeleteLicense 删除许可证
func (s *AssetLicenseService) DeleteLicense(ctx context.Context, id, tenantID int) error {
	_, err := s.client.AssetLicense.Delete().
		Where(assetlicense.IDEQ(id), assetlicense.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete license", "error", err, "license_id", id)
		return fmt.Errorf("failed to delete license: %w", err)
	}

	s.logger.Infow("License deleted", "license_id", id)
	return nil
}

// GetLicenseStats 获取许可证统计
func (s *AssetLicenseService) GetLicenseStats(ctx context.Context, tenantID int) (*dto.LicenseStatsResponse, error) {
	stats := &dto.LicenseStatsResponse{}

	total, err := s.client.AssetLicense.Query().Where(assetlicense.TenantIDEQ(tenantID)).Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count licenses", "error", err)
		return nil, err
	}
	stats.Total = total

	// 统计各状态数量
	active, _ := s.client.AssetLicense.Query().Where(assetlicense.TenantIDEQ(tenantID), assetlicense.StatusEQ(string(dto.LicenseStatusActive))).Count(ctx)
	expired, _ := s.client.AssetLicense.Query().Where(assetlicense.TenantIDEQ(tenantID), assetlicense.StatusEQ(string(dto.LicenseStatusExpired))).Count(ctx)
	expiringSoon, _ := s.client.AssetLicense.Query().Where(assetlicense.TenantIDEQ(tenantID), assetlicense.StatusEQ(string(dto.LicenseStatusExpiringSoon))).Count(ctx)
	depleted, _ := s.client.AssetLicense.Query().Where(assetlicense.TenantIDEQ(tenantID), assetlicense.StatusEQ(string(dto.LicenseStatusDepleted))).Count(ctx)

	stats.Active = active
	stats.Expired = expired
	stats.ExpiringSoon = expiringSoon
	stats.Depleted = depleted

	// 计算合规率 (有效的/总数)
	if total > 0 {
		stats.ComplianceRate = float64(active) / float64(total) * 100
	}

	return stats, nil
}

// AssignUsers 分配许可证给用户
func (s *AssetLicenseService) AssignUsers(ctx context.Context, id, tenantID int, userIDs []int) (*dto.LicenseResponse, error) {
	licenseEntity, err := s.client.AssetLicense.Query().
		Where(assetlicense.IDEQ(id), assetlicense.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get license", "error", err, "license_id", id)
		return nil, fmt.Errorf("failed to get license: %w", err)
	}

	// 检查是否超过可用数量
	availableQty := licenseEntity.TotalQuantity - licenseEntity.UsedQuantity
	if len(userIDs) > availableQty {
		return nil, fmt.Errorf("insufficient license quantity: available %d, requested %d", availableQty, len(userIDs))
	}

	// 合并用户列表
	newUsers := append(licenseEntity.Users, userIDs...)
	updated, err := licenseEntity.Update().
		SetUsers(newUsers).
		SetUsedQuantity(len(newUsers)).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to assign users to license", "error", err, "license_id", id)
		return nil, fmt.Errorf("failed to assign users: %w", err)
	}

	// 检查是否已耗尽
	if updated.UsedQuantity >= updated.TotalQuantity {
		updated.Update().SetStatus(string(dto.LicenseStatusDepleted)).Exec(ctx)
	}

	s.logger.Infow("Users assigned to license", "license_id", id, "user_count", len(userIDs))
	return dto.ToLicenseResponse(updated), nil
}

// CheckAndUpdateLicenseStatus 检查并更新许可证状态
func (s *AssetLicenseService) CheckAndUpdateLicenseStatus(ctx context.Context) error {
	licenses, err := s.client.AssetLicense.Query().All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to query licenses", "error", err)
		return err
	}

	now := time.Now()
	for _, license := range licenses {
		// 检查是否过期
		if license.ExpiryDate != "" {
			expiryTime, err := time.Parse("2006-01-02", license.ExpiryDate)
			if err == nil {
				if expiryTime.Before(now) {
					license.Update().SetStatus(string(dto.LicenseStatusExpired)).Exec(ctx)
					continue
				} else if expiryTime.Before(now.AddDate(0, 1, 0)) { // 1个月内过期
					license.Update().SetStatus(string(dto.LicenseStatusExpiringSoon)).Exec(ctx)
				}
			}
		}

		// 检查是否已耗尽
		if license.UsedQuantity >= license.TotalQuantity && license.Status != string(dto.LicenseStatusDepleted) {
			license.Update().SetStatus(string(dto.LicenseStatusDepleted)).Exec(ctx)
		}
	}

	s.logger.Infow("License status check completed", "total_licenses", len(licenses))
	return nil
}
