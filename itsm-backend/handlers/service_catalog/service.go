package service_catalog

import (
	"context"

	"go.uber.org/zap"
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

func (s *Service) Create(ctx context.Context, name, category, description string, deliveryTime, tenantID int, status string, ciTypeID, cloudServiceID int) (*ServiceCatalog, error) {
	// Business validation could go here
	catalog := &ServiceCatalog{
		Name:           name,
		Category:       category,
		Description:    description,
		DeliveryTime:   deliveryTime,
		CITypeID:       ciTypeID,
		CloudServiceID: cloudServiceID,
		Status:         status,
		TenantID:       tenantID,
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
	return s.repo.List(ctx, tenantID, filters)
}

func (s *Service) Update(ctx context.Context, tenantID int, id int, name, category, description string, deliveryTime int, status string, ciTypeID, cloudServiceID int) (*ServiceCatalog, error) {
	// First check if exists
	current, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if name != "" {
		current.Name = name
	}
	if category != "" {
		current.Category = category
	}
	if description != "" {
		current.Description = description
	}
	if deliveryTime > 0 {
		current.DeliveryTime = deliveryTime
	}
	if status != "" {
		current.Status = status
	}
	if ciTypeID > 0 {
		current.CITypeID = ciTypeID
	}
	if cloudServiceID > 0 {
		current.CloudServiceID = cloudServiceID
	}

	return s.repo.Update(ctx, tenantID, current)
}

func (s *Service) Delete(ctx context.Context, tenantID int, id int) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *Service) Search(ctx context.Context, tenantID int, keyword string, filters ListFilters) ([]*ServiceCatalog, int, error) {
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Size < 1 {
		filters.Size = 20
	}
	return s.repo.Search(ctx, tenantID, keyword, filters)
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
