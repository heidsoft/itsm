package ticket

import (
	"context"
	"time"

	"itsm-backend/repository/base"
)

// Repository 工单仓储接口
// 定义工单数据访问的所有操作
type Repository interface {
	// 基础 CRUD
	Create(ctx context.Context, params *CreateParams, tenantID int) (*Ticket, error)
	GetByID(ctx context.Context, id int, tenantID int) (*Ticket, error)
	GetByNumber(ctx context.Context, ticketNumber string, tenantID int) (*Ticket, error)
	Update(ctx context.Context, id int, params *UpdateParams, tenantID int) (*Ticket, error)
	Delete(ctx context.Context, id int, tenantID int) error

	// 列表查询
	List(ctx context.Context, tenantID int, filters *FilterParams, pagination *base.QueryParams) (*base.ListResult[Ticket], error)

	// 批量操作
	BatchDelete(ctx context.Context, ids []int, tenantID int) error
	Exists(ctx context.Context, id int, tenantID int) (bool, error)

	// 业务查询
	FindByAssignee(ctx context.Context, assigneeID int, tenantID int) ([]*Ticket, error)
	FindByRequester(ctx context.Context, requesterID int, tenantID int) ([]*Ticket, error)
	FindOverdue(ctx context.Context, tenantID int) ([]*Ticket, error)
	CountByStatus(ctx context.Context, tenantID int) (map[Status]int, error)
	CountByPriority(ctx context.Context, tenantID int) (map[Priority]int, error)

	// 编号生成
	GenerateTicketNumber(ctx context.Context, tenantID int) (string, error)

	// 状态变更
	UpdateStatus(ctx context.Context, id int, status Status, tenantID int) (*Ticket, error)
	AssignTicket(ctx context.Context, id int, assigneeID int, tenantID int) (*Ticket, error)

	// SLA 相关
	UpdateSLADeadlines(ctx context.Context, id int, responseDeadline, resolutionDeadline *time.Time, slaDefinitionID *int, tenantID int) error
	MarkFirstResponse(ctx context.Context, id int, tenantID int) error

	// 版本控制
	GetVersion(ctx context.Context, id int, tenantID int) (int, error)
}
