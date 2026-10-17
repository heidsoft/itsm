package service_catalog

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"itsm-backend/common"
)

// Service defines the business logic
type Service struct {
	repo   Repository
	logger *zap.SugaredLogger
}

// nameConflict 给管理员返回可直接定位的中文冲突提示，重名的具体值只进 Detail 供日志使用。
func nameConflict(name string) error {
	return common.NewAppError(common.ErrCodeConflict, "服务目录名称已存在，请换一个名称", http.StatusConflict, nil).WithDetail(name)
}

// NewService creates a new Service
func NewService(repo Repository, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// validateReferencesFor 把仓储层的引用校验错误映射成页面能直接展示的中文提示。
// 这里禁止把 err.Error() 当成用户消息：它会带上驱动原文和英文错误串。
func validateReferencesFor(ctx context.Context, repo Repository, tenantID, ciTypeID, cloudServiceID int) error {
	err := repo.ValidateReferences(ctx, tenantID, ciTypeID, cloudServiceID)
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrCITypeUnavailable):
		return common.NewBadRequestError("关联的 CI 类型不存在或未启用", err)
	case errors.Is(err, ErrCloudServiceUnavailable):
		return common.NewBadRequestError("关联的云服务不存在或未启用", err)
	default:
		return common.NewInternalError("关联资源校验失败", err)
	}
}

func (s *Service) Create(ctx context.Context, catalog *ServiceCatalog) (*ServiceCatalog, error) {
	catalog.Name = strings.TrimSpace(catalog.Name)
	catalog.Category = strings.TrimSpace(catalog.Category)
	if catalog.Name == "" || catalog.Category == "" {
		return nil, common.NewBadRequestError("服务名称与服务分类不能为空", nil)
	}
	if catalog.DeliveryTime == 0 {
		catalog.DeliveryTime = 1
	}
	if catalog.DeliveryTime < 1 || catalog.DeliveryTime > 3650 {
		return nil, common.NewBadRequestError("交付天数必须在 1 到 3650 之间", nil)
	}
	if catalog.Status == "" {
		catalog.Status = "enabled"
	}
	if !isValidCatalogStatus(catalog.Status) {
		return nil, common.NewBadRequestError(msgInvalidStatus, nil)
	}
	if catalog.CloudServiceID > 0 && catalog.CITypeID == 0 {
		return nil, common.NewBadRequestError("关联云服务时必须选择 CI 类型", nil)
	}
	exists, err := s.repo.NameExists(ctx, catalog.TenantID, catalog.Name, 0)
	if err != nil {
		return nil, common.NewInternalError("Failed to validate service catalog name", err)
	}
	if exists {
		return nil, nameConflict(catalog.Name)
	}
	if err := validateReferencesFor(ctx, s.repo, catalog.TenantID, catalog.CITypeID, catalog.CloudServiceID); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, catalog)
}

func (s *Service) Get(ctx context.Context, tenantID int, id int) (*ServiceCatalog, error) {
	return s.repo.Get(ctx, tenantID, id)
}

func (s *Service) List(ctx context.Context, tenantID int, filters ListFilters) ([]*ServiceCatalog, int, error) {
	// 页长不在这里夹紧：HTTP 入口用 common.GetPaginationFromQuery 做唯一所有者，
	// 非 HTTP 调用方（tests/scenarios 直接打 repo）由 EntRepository.List 兜底。
	// handler、Service.List、Service.Search 各写一遍夹紧，就是同一件事的三个所有者。
	return s.repo.List(ctx, tenantID, filters)
}

func (s *Service) Update(ctx context.Context, tenantID int, id int, patch *CatalogPatch) (*ServiceCatalog, error) {
	current, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if patch == nil {
		return current, nil
	}

	if patch.Name != nil {
		name := strings.TrimSpace(*patch.Name)
		if name == "" {
			return nil, common.NewBadRequestError("服务名称不能为空", nil)
		}
		current.Name = name
	}
	if patch.Category != nil {
		category := strings.TrimSpace(*patch.Category)
		if category == "" {
			return nil, common.NewBadRequestError("服务分类不能为空", nil)
		}
		current.Category = category
	}
	if patch.Description != nil {
		current.Description = *patch.Description
	}
	if patch.Icon != nil {
		current.Icon = *patch.Icon
	}
	if patch.ServiceType != nil {
		serviceType := strings.TrimSpace(*patch.ServiceType)
		if serviceType != "" && !isValidServiceType(serviceType) {
			return nil, common.NewBadRequestError("服务类型只能是 vm/rds/oss/network/storage/security/custom", nil)
		}
		current.ServiceType = serviceType
	}
	if patch.Price != nil {
		if *patch.Price < 0 {
			return nil, common.NewBadRequestError("价格不能为负数", nil)
		}
		current.Price = *patch.Price
	}
	if patch.Unit != nil {
		unit := strings.TrimSpace(*patch.Unit)
		if unit != "" && !isValidUnit(unit) {
			return nil, common.NewBadRequestError("计价单位只能是 月/次/用户", nil)
		}
		current.Unit = unit
	}
	if patch.DeliveryTime != nil {
		if *patch.DeliveryTime < 1 || *patch.DeliveryTime > 3650 {
			return nil, common.NewBadRequestError("交付天数必须在 1 到 3650 之间", nil)
		}
		current.DeliveryTime = *patch.DeliveryTime
	}
	if patch.RequiresApproval != nil {
		current.RequiresApproval = *patch.RequiresApproval
	}
	if patch.ApprovalLevel != nil {
		if *patch.ApprovalLevel < 0 || *patch.ApprovalLevel > 3 {
			return nil, common.NewBadRequestError("审批级别必须在 0 到 3 之间（0 表示未配置）", nil)
		}
		current.ApprovalLevel = *patch.ApprovalLevel
	}
	if patch.SLAResponseTime != nil {
		if *patch.SLAResponseTime < 0 {
			return nil, common.NewBadRequestError("SLA 响应时限不能为负数", nil)
		}
		current.SLAResponseTime = *patch.SLAResponseTime
	}
	if patch.SLAResolutionTime != nil {
		if *patch.SLAResolutionTime < 0 {
			return nil, common.NewBadRequestError("SLA 解决时限不能为负数", nil)
		}
		current.SLAResolutionTime = *patch.SLAResolutionTime
	}
	if patch.CITypeID != nil {
		if *patch.CITypeID < 0 {
			return nil, common.NewBadRequestError("CI 类型 ID 无效", nil)
		}
		current.CITypeID = *patch.CITypeID
	}
	if patch.CloudServiceID != nil {
		if *patch.CloudServiceID < 0 {
			return nil, common.NewBadRequestError("云服务 ID 无效", nil)
		}
		current.CloudServiceID = *patch.CloudServiceID
	}
	if patch.FormSchema != nil {
		current.FormSchema = *patch.FormSchema
	}
	if patch.SortOrder != nil {
		if *patch.SortOrder < 0 {
			return nil, common.NewBadRequestError("排序不能为负数", nil)
		}
		current.SortOrder = *patch.SortOrder
	}
	if len(patch.Approvers) > 0 {
		current.Approvers = patch.Approvers
	}
	if len(patch.AvailableRegions) > 0 {
		current.AvailableRegions = patch.AvailableRegions
	}
	if len(patch.AvailableSpecs) > 0 {
		current.AvailableSpecs = patch.AvailableSpecs
	}

	// 云服务依赖 CI 类型：解绑 CI 会让已关联的云服务失去归属，必须一起拒绝。
	if current.CloudServiceID > 0 && current.CITypeID == 0 {
		return nil, common.NewBadRequestError("关联云服务时必须选择 CI 类型", nil)
	}
	if patch.Status != nil {
		status := strings.TrimSpace(*patch.Status)
		if !isValidCatalogStatus(status) {
			return nil, common.NewBadRequestError(msgInvalidStatus, nil)
		}
		current.Status = status
	}
	current.IsActive = current.Status == "enabled"

	exists, err := s.repo.NameExists(ctx, tenantID, current.Name, id)
	if err != nil {
		return nil, common.NewInternalError("Failed to validate service catalog name", err)
	}
	if exists {
		return nil, nameConflict(current.Name)
	}
	if err := validateReferencesFor(ctx, s.repo, tenantID, current.CITypeID, current.CloudServiceID); err != nil {
		return nil, err
	}

	return s.repo.Update(ctx, tenantID, current)
}

func (s *Service) Delete(ctx context.Context, tenantID int, id int) error {
	if _, err := s.repo.Get(ctx, tenantID, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *Service) Search(ctx context.Context, tenantID int, keyword string, filters ListFilters) ([]*ServiceCatalog, int, error) {
	// 页长夹紧见 List 上的说明：唯一的 HTTP 所有者在 handler 入口，兜底在 repository。
	return s.repo.Search(ctx, tenantID, strings.TrimSpace(keyword), filters)
}

func isValidCatalogStatus(status string) bool {
	return status == "enabled" || status == "disabled"
}

const msgInvalidStatus = "状态只能是 enabled 或 disabled"

var catalogServiceTypes = map[string]bool{
	"vm": true, "rds": true, "oss": true, "network": true,
	"storage": true, "security": true, "custom": true,
}

var catalogUnits = map[string]bool{"月": true, "次": true, "用户": true}

func isValidServiceType(serviceType string) bool {
	return catalogServiceTypes[serviceType]
}

func isValidUnit(unit string) bool {
	return catalogUnits[unit]
}

func (s *Service) GetStats(ctx context.Context, tenantID int) (*ServiceStats, error) {
	// Count total services
	total, err := s.repo.Count(ctx, tenantID, ListFilters{})
	if err != nil {
		return nil, err
	}

	// Count published (enabled) services
	enabled, err := s.repo.Count(ctx, tenantID, ListFilters{Status: "enabled"})
	if err != nil {
		return nil, err
	}

	// Count by category
	byCategory, err := s.repo.CountByCategory(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &ServiceStats{
		TotalServices:     total,
		PublishedServices: enabled,
		Categories:        byCategory,
	}, nil
}
