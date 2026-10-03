package ticket

import (
	"time"
)

// Ticket represents the core ticket entity
type Ticket struct {
	ID                    int
	TicketNumber          string
	Title                 string
	Description           string
	Status                string
	Priority              string
	Type                  string
	TicketTypeID          *int
	TicketTypeCode        string
	TicketTypeName        string
	FormFields            map[string]interface{}
	RequesterID           int
	AssigneeID            *int
	TenantID              int
	TemplateID            *int
	CategoryID            *int
	DepartmentID          *int
	ParentTicketID        *int
	SLADefinitionID       *int
	SLAResponseDeadline   *time.Time
	SLAResolutionDeadline *time.Time
	FirstResponseAt       *time.Time
	ResolvedAt            *time.Time
	ClosedAt              *time.Time
	Resolution            *string
	Rating                *int
	RatingComment         *string
	RatedAt               *time.Time
	RatedBy               *int
	Version               int
	IsManagedByMSP        bool
	MSPProviderID         *int
	ManagedByUserID       *int
	MSPTicketID           *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             *time.Time
}

// TicketEvent represents an audit or activity log for a ticket
type TicketEvent struct {
	ID          int
	TicketID    int
	EventType   string
	EventName   string
	Description string
	Status      string
	Data        map[string]interface{}
	OccurredAt  time.Time
	UserID      int
	Source      string
	Metadata    map[string]interface{}
	TenantID    int
	CreatedAt   time.Time
}

// TicketTemplate represents a ticket template
type TicketTemplate struct {
	ID            int
	Name          string
	Description   string
	Category      string
	Priority      string
	Fields        []map[string]interface{}
	FormFields    map[string]interface{}
	WorkflowSteps []map[string]interface{}
	IsActive      bool
	TenantID      int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// StatusCount 和 PriorityCount 保留分布维度：报表要的是按词表排序的完整分布，
// 而不是折成 highPriority 这类单值后无法还原的计数。
type StatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type PriorityCount struct {
	Priority string `json:"priority"`
	Count    int    `json:"count"`
}

// TicketStats holds aggregated ticket statistics per tenant
type TicketStats struct {
	OverdueTickets    int `json:"overdueTickets"`
	TotalTickets      int `json:"totalTickets"`
	OpenTickets       int `json:"openTickets"`
	InProgressTickets int `json:"inProgressTickets"`
	PendingTickets    int `json:"pendingTickets"`
	ResolvedTickets   int `json:"resolvedTickets"`
	ClosedTickets     int `json:"closedTickets"`
	CriticalTickets   int `json:"criticalTickets"`
	HighTickets       int `json:"highTickets"`
	AvgResolutionMin  int `json:"avgResolutionMin"`
	// ByStatus/ByPriority 只包含真实存在的取值（来自分组计数），
	// 顺序按词表基准，词表外的历史值按字母序追加。
	ByStatus   []StatusCount   `json:"byStatus"`
	ByPriority []PriorityCount `json:"byPriority"`
}
