package change

import (
	"context"
)

// Repository interface for Change domain
type Repository interface {
	// Change CRUD
	Create(ctx context.Context, c *Change) (*Change, error)
	Get(ctx context.Context, id int, tenantID int) (*Change, error)
	List(ctx context.Context, tenantID int, page, size int, status, search string) ([]*Change, int, error)
	Update(ctx context.Context, c *Change) (*Change, error)
	Delete(ctx context.Context, id int, tenantID int) error
	GetStats(ctx context.Context, tenantID int) (*Stats, error)

	// Approvals
	CreateApprovalRecord(ctx context.Context, r *ApprovalRecord) (*ApprovalRecord, error)
	UpdateApprovalRecord(ctx context.Context, r *ApprovalRecord) (*ApprovalRecord, error)
	GetApprovalHistory(ctx context.Context, changeID int) ([]*ApprovalRecord, error)

	// Approval Workflow/Chain
	CreateApprovalChain(ctx context.Context, chain []*ApprovalChain) error
	GetApprovalChain(ctx context.Context, changeID int) ([]*ApprovalChain, error)
	DeleteApprovalChain(ctx context.Context, changeID int) error

	// Risk Assessment
	CreateRiskAssessment(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error)
	GetRiskAssessment(ctx context.Context, changeID int) (*RiskAssessment, error)
}
