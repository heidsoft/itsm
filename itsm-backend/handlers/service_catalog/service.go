package service_catalog

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"itsm-backend/common"
)

// Service defines the business logic
type Service struct {
	repo   Repository
	logger *zap.SugaredLogger
}

// NewService creates a new Service
func NewService(repo Repository, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

func (s *Service) Create(ctx context.Context, catalog *ServiceCatalog) (*ServiceCatalog, error) {
	catalog.Name = strings.TrimSpace(catalog.Name)
	catalog.Category = strings.TrimSpace(catalog.Category)
	if catalog.Name == "" || catalog.Category == "" {
		return nil, common.NewBadRequestError("Service name and category are required", nil)
	}
	if catalog.DeliveryTime == 0 {
		catalog.DeliveryTime = 1
	}
	if catalog.DeliveryTime < 1 || catalog.DeliveryTime > 3650 {
		return nil, common.NewBadRequestError("Delivery time must be between 1 and 3650 days", nil)
	}
	if catalog.Status == "" {
		catalog.Status = "enabled"
	}
	if !isValidCatalogStatus(catalog.Status) {
		return nil, common.NewBadRequestError("Invalid service catalog status", nil)
	}
	if catalog.CloudServiceID > 0 && catalog.CITypeID == 0 {
		return nil, common.NewBadRequestError("CI type is required when linking a cloud service", nil)
	}
	exists, err := s.repo.NameExists(ctx, catalog.TenantID, catalog.Name, 0)
	if err != nil {
		return nil, common.NewInternalError("Failed to validate service catalog name", err)
	}
	if exists {
		return nil, common.NewConflictError("Service catalog name", catalog.Name)
	}
	if err := s.repo.ValidateReferences(ctx, catalog.TenantID, catalog.CITypeID, catalog.CloudServiceID); err != nil {
		return nil, common.NewBadRequestError(err.Error(), err)
	}
	return s.repo.Create(ctx, catalog)
}

func (s *Service) Get(ctx context.Context, tenantID int, id int) (*ServiceCatalog, error) {
	return s.repo.Get(ctx, tenantID, id)
}

func (s *Service) List(ctx context.Context, tenantID int, filters ListFilters) ([]*ServiceCatalog, int, error) {
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Size < 1 {
		filters.Size = 10
	}
	if filters.Size > 100 {
		filters.Size = 100
	}
	return s.repo.List(ctx, tenantID, filters)
}

func (s *Service) Update(ctx context.Context, tenantID int, catalog *ServiceCatalog) (*ServiceCatalog, error) {
	// First check if exists
	current, err := s.repo.Get(ctx, tenantID, catalog.ID)
	if err != nil {
		return nil, err
	}

	catalog.Name = strings.TrimSpace(catalog.Name)
	catalog.Category = strings.TrimSpace(catalog.Category)
	if catalog.DeliveryTime < 0 || catalog.DeliveryTime > 3650 {
		return nil, common.NewBadRequestError("Delivery time must be between 0 and 3650 days", nil)
	}
	if catalog.Status != "" && !isValidCatalogStatus(catalog.Status) {
		return nil, common.NewBadRequestError("Invalid service catalog status", nil)
	}
	effectiveName := current.Name
	if catalog.Name != "" {
		effectiveName = catalog.Name
	}
	exists, err := s.repo.NameExists(ctx, tenantID, effectiveName, catalog.ID)
	if err != nil {
		return nil, common.NewInternalError("Failed to validate service catalog name", err)
	}
	if exists {
		return nil, common.NewConflictError("Service catalog name", effectiveName)
	}
	effectiveCITypeID := current.CITypeID
	if catalog.CITypeID > 0 {
		effectiveCITypeID = catalog.CITypeID
	}
	effectiveCloudServiceID := current.CloudServiceID
	if catalog.CloudServiceID > 0 {
		effectiveCloudServiceID = catalog.CloudServiceID
	}
	if effectiveCloudServiceID > 0 && effectiveCITypeID == 0 {
		return nil, common.NewBadRequestError("CI type is required when linking a cloud service", nil)
	}
	if err := s.repo.ValidateReferences(ctx, tenantID, effectiveCITypeID, effectiveCloudServiceID); err != nil {
		return nil, common.NewBadRequestError(err.Error(), err)
	}

	// Apply updates
	if catalog.Name != "" {
		current.Name = catalog.Name
	}
	if catalog.Category != "" {
		current.Category = catalog.Category
	}
	if catalog.Description != "" {
		current.Description = catalog.Description
	}
	if catalog.DeliveryTime > 0 {
		current.DeliveryTime = catalog.DeliveryTime
	}
	if catalog.Status != "" {
		current.Status = catalog.Status
	}
	if catalog.CITypeID > 0 {
		current.CITypeID = catalog.CITypeID
	}
	if catalog.CloudServiceID > 0 {
		current.CloudServiceID = catalog.CloudServiceID
	}
	// New fields
	if catalog.Icon != "" {
		current.Icon = catalog.Icon
	}
	if catalog.ServiceType != "" {
		current.ServiceType = catalog.ServiceType
	}
	if catalog.Price > 0 {
		current.Price = catalog.Price
	}
	if catalog.Unit != "" {
		current.Unit = catalog.Unit
	}
	current.RequiresApproval = catalog.RequiresApproval
	if catalog.ApprovalLevel > 0 {
		current.ApprovalLevel = catalog.ApprovalLevel
	}
	if len(catalog.Approvers) > 0 {
		current.Approvers = catalog.Approvers
	}
	if catalog.SLAResponseTime > 0 {
		current.SLAResponseTime = catalog.SLAResponseTime
	}
	if catalog.SLAResolutionTime > 0 {
		current.SLAResolutionTime = catalog.SLAResolutionTime
	}
	if catalog.FormSchema != nil {
		current.FormSchema = catalog.FormSchema
	}
	if len(catalog.AvailableRegions) > 0 {
		current.AvailableRegions = catalog.AvailableRegions
	}
	if len(catalog.AvailableSpecs) > 0 {
		current.AvailableSpecs = catalog.AvailableSpecs
	}
	if catalog.SortOrder > 0 {
		current.SortOrder = catalog.SortOrder
	}
	current.IsActive = catalog.Status == "enabled"

	return s.repo.Update(ctx, tenantID, current)
}

func (s *Service) Delete(ctx context.Context, tenantID int, id int) error {
	if _, err := s.repo.Get(ctx, tenantID, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *Service) Search(ctx context.Context, tenantID int, keyword string, filters ListFilters) ([]*ServiceCatalog, int, error) {
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Size < 1 {
		filters.Size = 20
	}
	if filters.Size > 100 {
		filters.Size = 100
	}
	return s.repo.Search(ctx, tenantID, strings.TrimSpace(keyword), filters)
}

func isValidCatalogStatus(status string) bool {
	return status == "enabled" || status == "disabled"
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
