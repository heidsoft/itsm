package problem

import (
	"context"
	"time"

	"itsm-backend/handlers/common/datascope"
)

// Repository interface for Problem domain
type Repository interface {
	Create(ctx context.Context, p *Problem) (*Problem, error)
	Get(ctx context.Context, id int, tenantID int) (*Problem, error)
	GetWithAssociations(ctx context.Context, id int, tenantID int) (*Problem, error)
	List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, dataScope datascope.DataScope, currentUserID int) ([]*Problem, int, error)
	GetAllForAnalytics(ctx context.Context, tenantID int, start, end time.Time) ([]*Problem, error)
	Update(ctx context.Context, p *Problem) (*Problem, error)
	Delete(ctx context.Context, id int, tenantID int) error
	GetStats(ctx context.Context, tenantID int) (*ProblemStats, error)
	AddAssociations(ctx context.Context, tenantID, problemID int, relatedType string, relatedIDs []int) error
	RemoveAssociation(ctx context.Context, tenantID, problemID int, relatedType string, relatedID int) error
	// LoadUserNames 批量加载 user 显示名（id -> name），用于在 ProblemResponse 中
	// 返回 createdBy/assignee 的中文姓名。仅查询 name/username 字段，name 为空时回退 username。
	LoadUserNames(ctx context.Context, tenantID int, ids []int) (map[int]string, error)
}
