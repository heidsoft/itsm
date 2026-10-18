package problem

import (
	"context"
	"errors"
	"time"

	"itsm-backend/internal/datascope"
)

// Repository interface for Problem domain
type Repository interface {
	Create(ctx context.Context, p *Problem) (*Problem, error)
	Get(ctx context.Context, id int, tenantID int) (*Problem, error)
	GetWithAssociations(ctx context.Context, id int, tenantID int) (*Problem, error)
	List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, dataScope datascope.DataScope, currentUserID int) ([]*Problem, int, error)
	GetAllForAnalytics(ctx context.Context, tenantID int, start, end time.Time) ([]*Problem, error)
	// Update 以 expectedStatus（读取时的状态）作 CAS 预期：problem 无 version 列，
	// 用读取时状态锚定「领域 service 校验与仓储写入之间的并发窗口」。
	// 预期不匹配返回 ErrProblemStatusMoved，service 层映射为 409 冲突语义。
	Update(ctx context.Context, p *Problem, expectedStatus string) (*Problem, error)
	Delete(ctx context.Context, id int, tenantID int) error
	GetStats(ctx context.Context, tenantID int) (*ProblemStats, error)
	AddAssociations(ctx context.Context, tenantID, problemID int, relatedType string, relatedIDs []int) error
	RemoveAssociation(ctx context.Context, tenantID, problemID int, relatedType string, relatedID int) error
	// LoadUserNames 批量加载 user 显示名（id -> name），用于在 ProblemResponse 中
	// 返回 createdBy/assignee 的中文姓名。仅查询 name/username 字段，name 为空时回退 username。
	LoadUserNames(ctx context.Context, tenantID int, ids []int) (map[int]string, error)
}

// ErrProblemStatusMoved 表示 Update 的 CAS 预期状态与真实状态不一致
// （校验与写入的并发窗口内状态已被其他请求推进）。
var ErrProblemStatusMoved = errors.New("problem status changed concurrently")
