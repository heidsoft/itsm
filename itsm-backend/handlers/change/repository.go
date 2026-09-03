package change

import (
	"context"

	"itsm-backend/handlers/common/datascope"
)

// Repository interface for Change domain
type Repository interface {
	// Change CRUD
	Create(ctx context.Context, c *Change) (*Change, error)
	Get(ctx context.Context, id int, tenantID int) (*Change, error)
	List(ctx context.Context, tenantID int, page, size int, status, search, riskLevel string, dataScope datascope.DataScope, currentUserID int) ([]*Change, int, error)
	Update(ctx context.Context, c *Change) (*Change, error)
	// UpdateStatusCAS 条件更新变更状态（乐观锁）：仅当记录当前状态等于
	// expectedStatus 时才推进到 targetStatus。返回 false 表示状态已被并发
	// 修改（或记录不存在/不属于该租户），调用方据此返回 409 冲突。
	// 这是状态转换消除 TOCTOU 竞态的关键原语，禁止绕过。
	UpdateStatusCAS(ctx context.Context, id, tenantID int, expectedStatus, targetStatus string) (bool, error)
	Delete(ctx context.Context, id int, tenantID int) error
	GetStats(ctx context.Context, tenantID int) (*Stats, error)
	SubmitForApproval(ctx context.Context, changeID, tenantID int, plan []ApprovalLevelPlan, comment string) error

	// Approvals
	CreateApprovalRecord(ctx context.Context, r *ApprovalRecord) (*ApprovalRecord, error)
	UpdateApprovalRecord(ctx context.Context, r *ApprovalRecord) (*ApprovalRecord, error)
	GetApprovalHistory(ctx context.Context, changeID int, tenantID int) ([]*ApprovalRecord, error)

	// Approval Workflow/Chain
	GetApprovalChain(ctx context.Context, changeID int, tenantID int) ([]*ApprovalChain, error)
	DeleteApprovalChain(ctx context.Context, changeID int, tenantID int) error
	ReplaceApprovalChain(ctx context.Context, changeID, tenantID int, chain []*ApprovalChain) error

	// Risk Assessment
	CreateRiskAssessment(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error)
	GetRiskAssessment(ctx context.Context, changeID int, tenantID int) (*RiskAssessment, error)
	UpdateRiskAssessment(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error)

	// Tenant validation
	ValidateApproverBelongsToTenant(ctx context.Context, approverID, tenantID int) (bool, error)

	// Calendar view
	ListByDateRange(ctx context.Context, tenantID int, startDate, endDate, status string) ([]*Change, error)
}
