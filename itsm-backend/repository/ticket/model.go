// Package ticket 提供工单领域模型和 Repository 接口
package ticket

import (
	"time"

	"itsm-backend/common"
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
	TypeIncident       Type = "incident"
	TypeProblem        Type = "problem"
	TypeChange         Type = "change"
	TypeServiceRequest Type = "service_request"
)

// Ticket 工单领域模型
// 表示 ITSM 系统中的工单实体
type Ticket struct {
	ID                    int
	TicketNumber          string
	Title                 string
	Description           string
	Status                Status
	Type                  Type
	TicketTypeID          *int
	TicketTypeCode        string
	TicketTypeName        string
	FormFields            map[string]interface{}
	Priority              Priority
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

// IsFinalState 判断是否为终态
func (t *Ticket) IsFinalState() bool {
	return t.Status == StatusClosed || t.Status == StatusCancelled
}

// CanTransitionTo 判断是否可以转换到目标状态。
// P1-1：统一委托 common.IsValidTicketStatusTransition，使其成为全系统唯一的工单状态机事实来源。
// 禁止在此处或其他层内联状态转换表（否则会出现不同入口判定不一致的漏洞）。
func (t *Ticket) CanTransitionTo(target Status) bool {
	return common.IsValidTicketStatusTransition(string(t.Status), string(target))
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

// DataScope 行级数据权限范围。
// 阻断8 修复：原 ListTickets 仅按 tenantID 过滤，普通员工（end_user）可读取全租户工单，
// 含 HR/薪酬/安全事件工单。引入 DataScope 由中间件解析角色后注入，service 层统一消费。
type DataScope int

const (
	// DataScopeAll 全租户可见（admin/manager/super_admin 等管理角色）。
	DataScopeAll DataScope = iota
	// DataScopeOwnedOrAssigned 仅可见本人创建或分配给本人的工单（end_user/agent）。
	DataScopeOwnedOrAssigned
)

// FilterParams 工单查询过滤参数
type FilterParams struct {
	Status         *Status
	Priority       *Priority
	Type           *Type
	RequesterID    *int
	AssigneeID     *int
	CategoryID     *int
	DepartmentID   *int
	ParentTicketID *int
	TemplateID     *int
	IsOverdue      bool
	Keyword        string
	DateFrom       *time.Time
	DateTo         *time.Time
	// DataScope 行级数据权限（阻断8）。
	// DataScopeOwnedOrAssigned 时，CurrentUserID 必须非零，
	// repository 会强制追加 Or(RequesterIDEQ(uid), AssigneeIDEQ(uid)) 谓词。
	DataScope     DataScope
	CurrentUserID int
}

// CreateParams 工单创建参数
type CreateParams struct {
	Title          string
	Description    string
	Type           Type
	TicketTypeID   *int
	TicketTypeCode string
	TicketTypeName string
	FormFields     map[string]interface{}
	Priority       Priority
	RequesterID    int
	AssigneeID     *int
	CategoryID     *int
	TemplateID     *int
	ParentTicketID *int
	TagIDs         []int
	Tags           []string
}

// UpdateParams 工单更新参数
type UpdateParams struct {
	Title       *string
	Description *string
	Status      *Status
	Type        *Type
	Priority    *Priority
	AssigneeID  *int
	CategoryID  *int
	ReplaceTags bool
	TagIDs      []int
	Resolution  *string
	FormFields  *map[string]interface{}
	Version     int // 乐观锁版本号
}
