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

func (s *Service) Get(ctx context.Context, id int) (*ServiceCatalog, error) {
	return s.repo.Get(ctx, id)
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

func (s *Service) Update(ctx context.Context, id int, name, category, description string, deliveryTime int, status string, ciTypeID, cloudServiceID int) (*ServiceCatalog, error) {
	// First check if exists
	current, err := s.repo.Get(ctx, id)
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

	return s.repo.Update(ctx, current)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
