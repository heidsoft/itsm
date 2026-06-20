package problem

import (
	"context"
)

// Repository interface for Problem domain
type Repository interface {
	Create(ctx context.Context, p *Problem) (*Problem, error)
	Get(ctx context.Context, id int, tenantID int) (*Problem, error)
	GetWithAssociations(ctx context.Context, id int, tenantID int) (*Problem, error)
	List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*Problem, int, error)
	Update(ctx context.Context, p *Problem) (*Problem, error)
	Delete(ctx context.Context, id int, tenantID int) error
	GetStats(ctx context.Context, tenantID int) (*ProblemStats, error)
	AddAssociations(ctx context.Context, problemID int, relatedType string, relatedIDs []int) error
	RemoveAssociation(ctx context.Context, problemID int, relatedType string, relatedID int) error
}
