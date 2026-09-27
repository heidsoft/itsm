package workbench

import "context"

// Service provides workbench business logic.
type Service interface {
	// Query returns paginated workbench items for the current user.
	Query(ctx context.Context, query WorkbenchQuery) (*WorkbenchResponse, error)
}

type serviceImpl struct {
	repo Repository
}

// NewService creates a new workbench service.
func NewService(repo Repository) Service {
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) Query(ctx context.Context, query WorkbenchQuery) (*WorkbenchResponse, error) {
	// TODO: Add permission checks to filter domains based on user's RBAC permissions
	// For now, delegate directly to repository
	return s.repo.Query(ctx, query)
}
