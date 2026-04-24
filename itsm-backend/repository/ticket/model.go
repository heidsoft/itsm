// Package ticket 提供工单领域模型和 Repository 接口
package ticket

import (
	"time"
)

// Status 工单状态类型
type Status string

const (
	StatusNew        Status = "new"
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusPending    Status = "pending"
	StatusResolved   Status = "resolved"
	StatusClosed     Status = "closed"
	StatusCancelled  Status = "cancelled"
)

// Priority 工单优先级类型
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityUrgent   Priority = "urgent"
	PriorityCritical Priority = "critical"
)

// Type 工单类型
type Type string

const (
	TypeIncident      Type = "incident"
	TypeProblem       Type = "problem"
	TypeChange        Type = "change"
	TypeServiceRequest Type = "service_request"
)

// Ticket 工单领域模型
// 表示 ITSM 系统中的工单实体
type Ticket struct {
	ID                   int
	TicketNumber         string
	Title                string
	Description          string
	Status               Status
	Type                 Type
	Priority             Priority
	RequesterID          int
	AssigneeID           *int
	TenantID             int
	TemplateID           *int
	CategoryID           *int
	DepartmentID         *int
	ParentTicketID       *int
	SLADefinitionID      *int
	SLAResponseDeadline  *time.Time
	SLAResolutionDeadline *time.Time
	FirstResponseAt      *time.Time
	ResolvedAt           *time.Time
	Resolution           *string
	Rating               *int
	RatingComment        *string
	RatedAt              *time.Time
	RatedBy              *int
	Version              int
	IsManagedByMSP       bool
	MSPProviderID        *int
	ManagedByUserID      *int
	MSPTicketID          *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            *time.Time
}

// IsFinalState 判断是否为终态
func (t *Ticket) IsFinalState() bool {
	return t.Status == StatusClosed || t.Status == StatusCancelled
}

// CanTransitionTo 判断是否可以转换到目标状态
func (t *Ticket) CanTransitionTo(target Status) bool {
	transitions := map[Status][]Status{
		StatusNew:        {StatusOpen, StatusCancelled},
		StatusOpen:       {StatusInProgress, StatusPending, StatusResolved, StatusCancelled},
		StatusInProgress: {StatusPending, StatusResolved, StatusCancelled},
		StatusPending:    {StatusInProgress, StatusResolved, StatusCancelled},
		StatusResolved:   {StatusClosed, StatusOpen}, // 可重开
		StatusClosed:     {},                         // 终态
		StatusCancelled:  {},                         // 终态
	}

	allowed, exists := transitions[t.Status]
	if !exists {
		return false
	}

	for _, s := range allowed {
		if s == target {
			return true
		}
	}
	return false
}

// Assign 分配工单给处理人
func (t *Ticket) Assign(assigneeID int) error {
	if t.IsFinalState() {
		return &StateError{
			CurrentStatus: t.Status,
			Message:       "cannot assign ticket in final state",
		}
	}
	t.AssigneeID = &assigneeID
	if t.Status == StatusNew {
		t.Status = StatusOpen
	}
	return nil
}

// Resolve 解决工单
func (t *Ticket) Resolve(resolution string) error {
	if !t.CanTransitionTo(StatusResolved) {
		return &StateError{
			CurrentStatus: t.Status,
			Message:       "cannot resolve ticket from current status",
		}
	}
	now := time.Now()
	t.Status = StatusResolved
	t.Resolution = &resolution
	t.ResolvedAt = &now
	return nil
}

// Close 关闭工单
func (t *Ticket) Close() error {
	if !t.CanTransitionTo(StatusClosed) {
		return &StateError{
			CurrentStatus: t.Status,
			Message:       "cannot close ticket from current status",
		}
	}
	t.Status = StatusClosed
	return nil
}

// Reopen 重开工单
func (t *Ticket) Reopen() error {
	if t.Status != StatusResolved {
		return &StateError{
			CurrentStatus: t.Status,
			Message:       "can only reopen resolved tickets",
		}
	}
	t.Status = StatusOpen
	t.ResolvedAt = nil
	return nil
}

// StateError 状态错误
type StateError struct {
	CurrentStatus Status
	Message       string
}

func (e *StateError) Error() string {
	return e.Message
}

// FilterParams 工单查询过滤参数
type FilterParams struct {
	Status       *Status
	Priority     *Priority
	Type         *Type
	RequesterID  *int
	AssigneeID   *int
	CategoryID   *int
	DepartmentID *int
	Keyword      string
	DateFrom     *time.Time
	DateTo       *time.Time
}

// CreateParams 工单创建参数
type CreateParams struct {
	Title       string
	Description string
	Type        Type
	Priority    Priority
	RequesterID int
	AssigneeID  *int
	CategoryID  *int
	Tags        []string
}

// UpdateParams 工单更新参数
type UpdateParams struct {
	Title       *string
	Description *string
	Status      *Status
	Priority    *Priority
	AssigneeID  *int
	Resolution  *string
	Version     int // 乐观锁版本号
}
