package ticket

import (
	"context"
	"time"

	"itsm-backend/internal/datascope"
)

// Repository defines the interface for ticket data access
type Repository interface {
	// Ticket operations
	Create(ctx context.Context, params *CreateParams, tenantID int) (*Ticket, error)
	GetByID(ctx context.Context, id int, tenantID int) (*Ticket, error)
	GetByNumber(ctx context.Context, ticketNumber string, tenantID int) (*Ticket, error)
	Update(ctx context.Context, id int, params *UpdateParams, tenantID int) (*Ticket, error)
	Delete(ctx context.Context, id int, tenantID int) error
	List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, dataScope datascope.DataScope, currentUserID int) ([]*Ticket, int, error)
	BatchDelete(ctx context.Context, ids []int, tenantID int) error
	Exists(ctx context.Context, id int, tenantID int) (bool, error)

	// Business queries
	FindByAssignee(ctx context.Context, assigneeID int, tenantID int) ([]*Ticket, error)
	FindOverdue(ctx context.Context, tenantID int) ([]*Ticket, error)
	Search(ctx context.Context, keyword string, tenantID int) ([]*Ticket, error)

	// Stats
	GetStats(ctx context.Context, tenantID int) (*TicketStats, error)

	// Number generation
	GenerateTicketNumber(ctx context.Context, tenantID int) (string, error)

	// Status transitions
	UpdateStatus(ctx context.Context, id int, status string, tenantID int) (*Ticket, error)
	AssignTicket(ctx context.Context, id int, assigneeID int, tenantID int) (*Ticket, error)
	ResolveTicket(ctx context.Context, id int, resolution string, tenantID int) (*Ticket, error)
	CloseTicket(ctx context.Context, id int, tenantID int) (*Ticket, error)
	EscalateTicket(ctx context.Context, id int, reason string, tenantID int, escalatedBy int) (*Ticket, error)

	// SLA
	UpdateSLADeadlines(ctx context.Context, id int, responseDeadline, resolutionDeadline *time.Time, slaDefinitionID *int, tenantID int) error

	// 注：模板 CRUD（CreateTemplate 等）不在此接口——它们的规则与持久化
	// 唯一所有者是 productionSvc（service.TicketService 模板 CRUD）；
	// 历史上这里挂在接口上时由返回伪造数据的 stub 实现，属于"伪成功"，已移除。
}

// CreateParams holds parameters for creating a ticket
type CreateParams struct {
	Title          string
	Description    string
	Type           string
	Priority       string
	RequesterID    int
	AssigneeID     *int
	TicketTypeID   *int
	TicketTypeCode string
	TicketTypeName string
	CategoryID     *int
	TemplateID     *int
	ParentTicketID *int
	FormFields     map[string]interface{}
	TagIDs         []int
	// WorkflowDefinitionKey 调用方显式指定的 BPMN 流程 Key（可选）。
	// handler→service→repository 任一层丢参，都会让 workflow_start 命令的 payload
	// 恒为空串，流程只能退到默认绑定。
	WorkflowDefinitionKey string
}

// UpdateParams holds parameters for updating a ticket
type UpdateParams struct {
	Title       *string
	Description *string
	Status      *string
	Priority    *string
	Type        *string
	AssigneeID  *int
	CategoryID  *int
	Resolution  *string
	FormFields  *map[string]interface{}
	Version     int
	ReplaceTags bool
	TagIDs      []int
}
