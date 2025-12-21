package cmdb

import (
	"context"

	"go.uber.org/zap"
)

type Service struct {
	repo   Repository
	logger *zap.SugaredLogger
}

func NewService(repo Repository, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// CI operations
func (s *Service) CreateCI(ctx context.Context, ci *ConfigurationItem) (*ConfigurationItem, error) {
	s.logger.Infow("Creating CI", "name", ci.Name, "type_id", ci.CITypeID)
	return s.repo.CreateCI(ctx, ci)
}

func (s *Service) GetCI(ctx context.Context, id int, tenantID int) (*ConfigurationItem, error) {
	return s.repo.GetCI(ctx, id, tenantID)
}

func (s *Service) ListCIs(ctx context.Context, tenantID int, page, size int, ciTypeID int, status string) ([]*ConfigurationItem, int, error) {
	return s.repo.ListCIs(ctx, tenantID, page, size, ciTypeID, status)
}

func (s *Service) UpdateCI(ctx context.Context, ci *ConfigurationItem) (*ConfigurationItem, error) {
	return s.repo.UpdateCI(ctx, ci)
}

func (s *Service) DeleteCI(ctx context.Context, id int, tenantID int) error {
	return s.repo.DeleteCI(ctx, id, tenantID)
}

func (s *Service) GetStats(ctx context.Context, tenantID int) (*Stats, error) {
	return s.repo.GetStats(ctx, tenantID)
}

// CIType operations
func (s *Service) ListTypes(ctx context.Context, tenantID int) ([]*CIType, error) {
	return s.repo.ListCITypes(ctx, tenantID)
}

func (s *Service) CreateType(ctx context.Context, ct *CIType) (*CIType, error) {
	return s.repo.CreateCIType(ctx, ct)
}

// Relationship operations
func (s *Service) CreateRelationship(ctx context.Context, rel *CIRelationship) (*CIRelationship, error) {
	return s.repo.CreateRelationship(ctx, rel)
}

func (s *Service) GetCIRelationships(ctx context.Context, ciID int) ([]*CIRelationship, error) {
	return s.repo.GetRelationships(ctx, ciID)
}
