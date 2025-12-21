package problem

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

func (s *Service) Create(ctx context.Context, tenantID int, p *Problem) (*Problem, error) {
	p.TenantID = tenantID
	return s.repo.Create(ctx, p)
}

func (s *Service) Get(ctx context.Context, id int, tenantID int) (*Problem, error) {
	return s.repo.Get(ctx, id, tenantID)
}

func (s *Service) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*Problem, int, error) {
	return s.repo.List(ctx, tenantID, page, size, filters)
}

func (s *Service) Update(ctx context.Context, tenantID int, id int, p *Problem) (*Problem, error) {
	existing, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Update fields if they are set (non-zero/non-empty check in Handler or here)
	// Here assuming 'p' contains only fields to update usually, but domain entity isn't partial.
	// We merge changes here.
	if p.Title != "" {
		existing.Title = p.Title
	}
	if p.Description != "" {
		existing.Description = p.Description
	}
	if p.Status != "" {
		existing.Status = p.Status
	}
	if p.Priority != "" {
		existing.Priority = p.Priority
	}
	if p.Category != "" {
		existing.Category = p.Category
	}
	if p.RootCause != "" {
		existing.RootCause = p.RootCause
	}
	if p.Impact != "" {
		existing.Impact = p.Impact
	}
	if p.AssigneeID != nil {
		existing.AssigneeID = p.AssigneeID
	}

	return s.repo.Update(ctx, existing)
}

func (s *Service) Delete(ctx context.Context, id int, tenantID int) error {
	return s.repo.Delete(ctx, id, tenantID)
}

func (s *Service) GetStats(ctx context.Context, tenantID int) (*ProblemStats, error) {
	return s.repo.GetStats(ctx, tenantID)
}
