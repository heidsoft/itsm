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

// DailyTrendPoint 按天的新建/解决读数，日期为本地时区的 YYYY-MM-DD。
type DailyTrendPoint struct {
	Date     string `json:"date"`
	Created  int    `json:"created"`
	Resolved int    `json:"resolved"`
}

// TicketSLACompliance 是统计窗口内的 SLA 达标读数。
//
// 它是指针：nil 表示 SLA 计时服务未接入或读数失败，属于「没取到」，
// 不能塌成 ComplianceRate=0 —— 0% 会被读成「全部工单都超时」，与故障无法区分。
//
// Scope 固定为 tenant：SLA 达标率按租户全量计算，不跟随登录者的行级数据范围。
// 页面必须把这句口径显示出来，否则同一屏会出现两种范围。
type TicketSLACompliance struct {
	ComplianceRate     float64 `json:"complianceRate"`
	SampledTickets     int     `json:"sampledTickets"`
	ViolatedTickets    int     `json:"violatedTickets"`
	AvgResponseMinutes float64 `json:"avgResponseMinutes"`
	AvgResolutionHours float64 `json:"avgResolutionHours"`
	WindowDays         int     `json:"windowDays"`
	Scope              string  `json:"scope"`
}

// SLAScopeTenant 是 SLA 达标读数唯一的口径取值（租户口径）。
const SLAScopeTenant = "tenant"

// TicketStatsWindow 回显本次统计实际使用的时间窗口，供页面标注读数口径。
// DateFrom/DateTo 是闭区间日（YYYY-MM-DD），Days 是天数（含首尾两天）。
type TicketStatsWindow struct {
	DateFrom string `json:"dateFrom"`
	DateTo   string `json:"dateTo"`
	Days     int    `json:"days"`
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
	// 两者与上面的计数器共用同一套行级数据范围与时间窗口。
	ByStatus   []StatusCount   `json:"byStatus"`
	ByPriority []PriorityCount `json:"byPriority"`
	// DailyTrend 仅在请求带时间窗口时填充；nil 表示本次未取趋势。
	DailyTrend []DailyTrendPoint    `json:"dailyTrend,omitempty"`
	SLA        *TicketSLACompliance `json:"sla,omitempty"`
	// Window 回显本次实际生效的统计窗口；nil 表示未限制窗口。
	// 页面必须把它显示出来，否则「最近 30 天」和「全部」在屏幕上长得一样。
	Window *TicketStatsWindow `json:"window,omitempty"`
	// Scope 是计数器与分布跟随的行级数据范围（all/department/owner），
	// 与 GET /api/v1/tickets 列表使用同一判定。
	Scope string `json:"scope"`
}
