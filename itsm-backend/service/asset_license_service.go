package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/assetlicense"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

// AssetLicenseService 许可证管理服务

// 许可证域稳定错误语义：
//   - ErrLicenseNotFound：按租户谓词查不到许可证（不存在或跨租户），handler 映射 404/4004。
//   - ErrLicenseInsufficient：席位不足，业务冲突而非后端故障，handler 映射 409/4090。
//   - ErrLicenseSeatRace：并发扣减冲突（条件 UPDATE 未命中），可重试，handler 映射 409/4090。
var (
	ErrLicenseNotFound     = errors.New("license not found")
	ErrLicenseInsufficient = errors.New("insufficient license seats")
	ErrLicenseSeatRace     = errors.New("license seat concurrent conflict")
)

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
	// 无条件分页：若只在 `page > 0 && pageSize > 0` 时才加 Offset/Limit，调用方漏写分页参数
	// 会让查询不带 LIMIT，一次把该租户整表读出来。HTTP 入口由 common.GetPaginationFromQuery
	// 归一，这里的 ValidatePagination 兜住非 HTTP 调用方。
	page, pageSize = common.ValidatePagination(page, pageSize)
	query := s.client.AssetLicense.Query().Where(assetlicense.TenantIDEQ(tenantID))

	if licenseType != "" {
		query = query.Where(assetlicense.LicenseTypeEQ(licenseType))
	}
	if status != "" {
		query = query.Where(assetlicense.StatusEQ(status))
	}

	// 统计总数
	total, err := query.Clone().Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count licenses", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to count licenses: %w", err)
	}

	offset := (page - 1) * pageSize
	// created_at 非唯一，缺 ID 并列键时翻页归属由数据库返回顺序决定，会重或漏。
	licenseEntities, err := query.
		Offset(offset).
		Limit(pageSize).
		Order(ent.Desc(assetlicense.FieldCreatedAt), ent.Asc(assetlicense.FieldID)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list licenses", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to list licenses: %w", err)
	}

	pagination := common.NewPaginationResponse(page, pageSize, int64(total))
	return &dto.LicenseListResponse{
		Items:      dto.ToLicenseResponseList(licenseEntities),
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: pagination.TotalPages,
	}, nil
}

// UpdateLicense 更新许可证。
//
// used_quantity 是分配操作维护的派生值：允许直接写会让「已用 > 总数」或
// 「已用 != len(users)」绕过 AssignUsers 的席位校验。这里拒绝直接改已用数，
// 替换授权用户时按去重后的用户数重算席位，并用 used_quantity 条件更新做并发保护。
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

	if req.UsedQuantity != nil {
		return nil, common.NewBusinessError(common.BadRequestCode, "已使用数量由许可证分配操作维护，不能直接修改", "")
	}

	update := s.client.AssetLicense.Update().
		Where(assetlicense.IDEQ(id), assetlicense.TenantIDEQ(tenantID))

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
	if len(req.Tags) > 0 {
		update.SetTags(req.Tags)
	}

	totalQuantity := licenseEntity.TotalQuantity
	if req.TotalQuantity != nil {
		if *req.TotalQuantity < 0 {
			return nil, common.NewBusinessError(common.BadRequestCode, "总数量不能为负数", "")
		}
		totalQuantity = *req.TotalQuantity
		update.SetTotalQuantity(totalQuantity)
	}

	usedQuantity := licenseEntity.UsedQuantity
	seatsTouched := req.TotalQuantity != nil
	if len(req.Users) > 0 {
		// 全量替换授权列表：与 AssignUsers 的「静默跳过无效 id」不同，
		// 这里不能丢弃调用方明确提交的 id——静默丢弃会连带移除同名用户。
		valid, invalid, err := s.normalizeLicenseUsers(ctx, tenantID, req.Users)
		if err != nil {
			s.logger.Errorw("Failed to validate license assignees", "error", err, "license_id", id)
			return nil, fmt.Errorf("failed to validate license assignees: %w", err)
		}
		if len(invalid) > 0 {
			return nil, common.NewBusinessError(common.BadRequestCode, "授权用户列表包含无效或跨租户用户", "")
		}
		usedQuantity = len(valid)
		update.SetUsers(valid).SetUsedQuantity(usedQuantity)
		seatsTouched = true
	}

	if usedQuantity > totalQuantity {
		return nil, common.NewBusinessError(common.ConflictCode,
			fmt.Sprintf("总数量 %d 小于已分配用户数 %d", totalQuantity, usedQuantity), "")
	}

	if usedQuantity >= totalQuantity {
		update.SetStatus(string(dto.LicenseStatusDepleted))
	} else if licenseEntity.Status == string(dto.LicenseStatusDepleted) {
		update.SetStatus(string(dto.LicenseStatusActive))
	}

	if seatsTouched {
		// 读到的 used_quantity 必须仍是写入时的值：并发分配先提交时本次放弃，
		// 不得把对方刚占用的席位覆盖掉。
		update.Where(assetlicense.UsedQuantityEQ(licenseEntity.UsedQuantity))
	}

	affected, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update license", "error", err, "license_id", id)
		return nil, fmt.Errorf("failed to update license: %w", err)
	}
	if affected == 0 {
		if seatsTouched {
			return nil, common.NewBusinessError(common.ConflictCode, "席位已被其他请求占用，请重试", "")
		}
		return nil, ErrLicenseNotFound
	}

	updated, err := s.client.AssetLicense.Query().
		Where(assetlicense.IDEQ(id), assetlicense.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to reload license", "error", err, "license_id", id)
		return nil, fmt.Errorf("failed to reload license: %w", err)
	}

	return dto.ToLicenseResponse(updated), nil
}

// DeleteLicense 删除许可证
func (s *AssetLicenseService) DeleteLicense(ctx context.Context, id, tenantID int) error {
	deleted, err := s.client.AssetLicense.Delete().
		Where(assetlicense.IDEQ(id), assetlicense.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete license", "error", err, "license_id", id)
		return fmt.Errorf("failed to delete license: %w", err)
	}
	// 条件删除影响行数为 0 视为 not-found，不得静默假装成功。
	if deleted == 0 {
		return ErrLicenseNotFound
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

// normalizeLicenseUsers 对授权用户 id 去重，并按本租户归属拆分。
// valid 是本租户真实存在的用户（保持首次出现顺序），invalid 是其余 id。
func (s *AssetLicenseService) normalizeLicenseUsers(ctx context.Context, tenantID int, userIDs []int) (valid, invalid []int, err error) {
	seen := make(map[int]struct{}, len(userIDs))
	unique := make([]int, 0, len(userIDs))
	for _, uid := range userIDs {
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		unique = append(unique, uid)
	}
	if len(unique) == 0 {
		return nil, nil, nil
	}

	existing, err := s.client.User.Query().
		Where(user.IDIn(unique...), user.TenantIDEQ(tenantID)).
		IDs(ctx)
	if err != nil {
		return nil, nil, err
	}
	existingSet := make(map[int]struct{}, len(existing))
	for _, uid := range existing {
		existingSet[uid] = struct{}{}
	}

	valid = make([]int, 0, len(unique))
	for _, uid := range unique {
		if _, ok := existingSet[uid]; ok {
			valid = append(valid, uid)
		} else {
			invalid = append(invalid, uid)
		}
	}
	return valid, invalid, nil
}

// AssignUsers 分配许可证给用户
// read-modify-write 用事务 + 原子 UPDATE 条件检查，避免并发请求下席位超发。
// 同时对传入的 userIDs 去重，避免重复分配的用户计入新增席位。
func (s *AssetLicenseService) AssignUsers(ctx context.Context, id, tenantID int, userIDs []int) (*dto.LicenseResponse, error) {
	// 1. 去重并校验租户归属：跨租户或无效 id 静默跳过（与站内通知收件人语义一致，
	// B5 锚点），防止他租用户 id 被写进 license.Users（跨租户引用注入面）。
	deduped, invalid, err := s.normalizeLicenseUsers(ctx, tenantID, userIDs)
	if err != nil {
		s.logger.Errorw("Failed to validate license assignees", "error", err, "license_id", id)
		return nil, fmt.Errorf("failed to validate license assignees: %w", err)
	}
	if len(invalid) > 0 {
		s.logger.Infow("Skipped license assignees outside tenant", "license_id", id, "invalid_user_ids", invalid)
	}
	if len(deduped) == 0 {
		// 无有效新增用户，直接返回原记录
		licenseEntity, err := s.client.AssetLicense.Query().
			Where(assetlicense.IDEQ(id), assetlicense.TenantIDEQ(tenantID)).
			First(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, ErrLicenseNotFound
			}
			return nil, fmt.Errorf("failed to get license: %w", err)
		}
		return dto.ToLicenseResponse(licenseEntity), nil
	}
	// 2. 事务：在同一事务内 FOR UPDATE 读 + 条件 UPDATE，防止并发超发
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	licenseEntity, err := tx.AssetLicense.Query().
		Where(assetlicense.IDEQ(id), assetlicense.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrLicenseNotFound
		}
		s.logger.Errorw("Failed to get license", "error", err, "license_id", id)
		return nil, fmt.Errorf("failed to get license: %w", err)
	}

	// 3. 计算真正需要新增的用户（剔除已在列表中的）
	existing := make(map[int]struct{}, len(licenseEntity.Users))
	for _, u := range licenseEntity.Users {
		existing[u] = struct{}{}
	}
	newUsers := make([]int, 0, len(licenseEntity.Users)+len(deduped))
	newUsers = append(newUsers, licenseEntity.Users...)
	needed := 0
	for _, uid := range deduped {
		if _, ok := existing[uid]; !ok {
			existing[uid] = struct{}{}
			newUsers = append(newUsers, uid)
			needed++
		}
	}

	// 没有真正需要新增的用户，直接返回
	if needed == 0 {
		_ = tx.Rollback()
		return dto.ToLicenseResponse(licenseEntity), nil
	}

	// 4. 检查可用数量
	availableQty := licenseEntity.TotalQuantity - licenseEntity.UsedQuantity
	if needed > availableQty {
		_ = tx.Rollback()
		// 席位不足是业务冲突：不得伪装成 500。handler 映射 409/4090。
		return nil, fmt.Errorf("%w: available %d, requested %d", ErrLicenseInsufficient, availableQty, needed)
	}

	// 5. 原子 UPDATE：WHERE 条件包含 used_quantity + needed <= total_quantity + tenant 过滤
	// 这保证了即使在 SERIALIZABLE 之外的隔离级别下也不会超发
	newUsed := licenseEntity.UsedQuantity + needed
	result, err := tx.AssetLicense.Update().
		Where(
			assetlicense.IDEQ(id),
			assetlicense.TenantIDEQ(tenantID),
			// 乐观条件：当前 used_quantity 必须等于我们刚读出的值（隐式 CAS + 总量保护）
			assetlicense.UsedQuantityEQ(licenseEntity.UsedQuantity),
			assetlicense.TotalQuantityGTE(newUsed),
		).
		SetUsers(newUsers).
		SetUsedQuantity(newUsed).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to assign users to license", "error", err, "license_id", id)
		return nil, fmt.Errorf("failed to assign users: %w", err)
	}
	if result == 0 {
		_ = tx.Rollback()
		// 并发扣减未命中：条件 UPDATE 原子保护（version 语义），映射 409 可重试。
		return nil, ErrLicenseSeatRace
	}

	// 6. 检查是否已耗尽（在同一事务内 UPDATE，避免读已提交时的不一致）
	if newUsed >= licenseEntity.TotalQuantity {
		_, err = tx.AssetLicense.Update().
			Where(assetlicense.IDEQ(id), assetlicense.TenantIDEQ(tenantID)).
			SetStatus(string(dto.LicenseStatusDepleted)).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("更新 depleted 状态失败: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	// 返回最终记录
	finalEntity, err := s.client.AssetLicense.Query().
		Where(assetlicense.IDEQ(id), assetlicense.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get final license: %w", err)
	}
	s.logger.Infow("Users assigned to license", "license_id", id, "new_user_count", needed)
	return dto.ToLicenseResponse(finalEntity), nil
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
