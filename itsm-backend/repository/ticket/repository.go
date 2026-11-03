package ticket

import (
	"context"
	"time"

	"itsm-backend/ent"
	"itsm-backend/internal/datascope"
	"itsm-backend/repository/base"
)

// Repository 工单仓储接口
// 定义工单数据访问的所有操作
type Repository interface {
	// 基础 CRUD
	Create(ctx context.Context, params *CreateParams, tenantID int) (*Ticket, error)
	// CreateWithTx 在调用方提供的 *ent.Tx 内创建工单，用于事务性通知入箱：
	// 调用方负责 commit/rollback，确保 ticket INSERT 与 operational_command 写入同生同死。
	CreateWithTx(ctx context.Context, tx *ent.Tx, params *CreateParams, tenantID int) (*Ticket, error)
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
	// FindOverduePage 是 FindOverdue 的分页形态，total 为未截断的逾期总数。
	// scope/viewerID 与 List 共用同一套行级数据范围：逾期列表是工单列表的一个
	// 过滤视图，不是新的读取通道，否则普通角色可绕过 DataScope 读到全租户数据。
	FindOverduePage(ctx context.Context, tenantID int, page, size int, scope datascope.DataScope, viewerID int) ([]*Ticket, int, error)
	// CountByStatus / CountByPriority 接受与 List 相同的行级数据范围，
	// 让工作台卡片读数与列表结果共用一个口径；scope 语义见 dataScopePredicates。
	// window 为 nil 时统计可见范围内全部工单，非 nil 时按 created_at 落在窗口内收窄。
	CountByStatus(ctx context.Context, tenantID int, scope datascope.DataScope, viewerID int, window *StatsWindow) (map[Status]int, error)
	CountByPriority(ctx context.Context, tenantID int, scope datascope.DataScope, viewerID int, window *StatsWindow) (map[Priority]int, error)
	// TrendByDay 按天返回窗口内的新建数与解决数。两条序列各按自己的事件时间落桶
	// （created_at / resolved_at），因此窗口外的老工单在窗口内被解决时，仍会计入
	// 当日 resolved —— 否则趋势图会把「本周解决」画成「本周新建且本周解决」。
	TrendByDay(ctx context.Context, tenantID int, scope datascope.DataScope, viewerID int, window StatsWindow) ([]TrendPoint, error)

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

// TransactionalUpdater lets application services append durable outbox writes
// to the same transaction as an optimistic-lock ticket update.
type TransactionalUpdater interface {
	UpdateWithTxHook(ctx context.Context, id int, params *UpdateParams, tenantID int, hook func(*ent.Tx, *Ticket) error) (*Ticket, error)
}
