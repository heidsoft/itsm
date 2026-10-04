package incident

import (
	"time"
)

// Incident represents the core incident entity
type Incident struct {
	ID                    int
	Title                 string
	Description           string
	Status                string // 取值见 common.IncidentStatus*，事件域没有 open
	Priority              string // ent/schema/incident.go 校验：low/medium/high/critical
	Severity              string // low, medium, high, critical
	Impact                string // low, medium, high, critical
	Urgency               string // low, medium, high, critical
	IncidentNumber        string
	ReporterID            int
	AssigneeID            *int
	ConfigurationItemID   *int
	Version               int
	IsMajorIncident       bool
	Category              string `json:"category"`
	Subcategory           string `json:"subcategory"`
	ServiceType           string `json:"serviceType"`
	FailureType           string `json:"failureType"`
	ImpactAnalysis        map[string]interface{}
	RootCause             map[string]interface{}
	ResolutionSteps       []map[string]interface{}
	Metadata              map[string]interface{}
	DetectedAt            time.Time
	ResolvedAt            *time.Time
	SLADefinitionID       *int
	SLAResponseDeadline   *time.Time
	SLAResolutionDeadline *time.Time
	SLAFirstResponseAt    *time.Time
	SLAResolvedAt         *time.Time
	SLAStatus             string
	SLAPausedAt           *time.Time
	SLAPauseReason        string
	ClosedAt              *time.Time
	EscalatedAt           *time.Time
	EscalationLevel       int
	IsAutomated           bool
	Source                string
	TenantID              int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// IncidentEvent represents an audit or activity log for an incident
type IncidentEvent struct {
	ID          int
	IncidentID  int
	EventType   string // e.g., creation, update, escalation, comment
	EventName   string
	Description string
	Status      string
	Severity    string
	Data        map[string]interface{}
	OccurredAt  time.Time
	UserID      int
	Source      string
	Metadata    map[string]interface{}
	TenantID    int
	CreatedAt   time.Time
}

// IncidentRule represents an automation rule
type IncidentRule struct {
	ID             int
	Name           string
	Description    string
	Conditions     map[string]interface{}
	Actions        []map[string]interface{}
	IsActive       bool
	Priority       string // Ent has this as string
	ExecutionCount int
	LastExecutedAt *time.Time
	TenantID       int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IncidentStatCount 是报表分组计数项。Value 保留数据库原始取值：事件 status 是无约束
// 字符串列，词表外的历史取值必须原样返回（既不丢计数，也不静默并进别的桶），
// 由前端按常量表映射标签、缺失时显示原始值。
type IncidentStatCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// IncidentTrendPoint 是窗口内单日的事件量。Created 按 created_at 归日，
// Resolved 按 resolved_at 归日：两者是不同口径的独立集合，不满足逐日相等关系。
type IncidentTrendPoint struct {
	Date     string `json:"date"`
	Created  int    `json:"created"`
	Resolved int    `json:"resolved"`
}

// IncidentReportWindow 回显报表实际使用的统计窗口。日期按服务器本地时区整日边界解释，
// DateTo 为闭区间（包含当天）；前端不得自行推断或改写窗口。
type IncidentReportWindow struct {
	DateFrom string `json:"dateFrom"`
	DateTo   string `json:"dateTo"`
	Days     int    `json:"days"`
}

// IncidentReport 是事件趋势报表读模型（GET /api/v1/incidents/stats/report）。
//
// 与 GET /api/v1/incidents/stats 的租户全量标量是两套口径，不可混用：
//   - CreatedInWindow/ByStatus/ByPriority 按 created_at 限定在 Window 内；
//     ByStatus、ByPriority 各自之和恒等于 CreatedInWindow。
//   - ResolvedInWindow 按 resolved_at 落在 Window 内计数，包含窗口之前就已创建的事件。
//   - AvgResolutionMinutes 是 ResolvedInWindow 集合的 (resolved_at - created_at) 均值，
//     单位写在字段名里，窗口内没有已解决事件时为 0。
type IncidentReport struct {
	Window               IncidentReportWindow `json:"window"`
	CreatedInWindow      int                  `json:"createdInWindow"`
	ResolvedInWindow     int                  `json:"resolvedInWindow"`
	AvgResolutionMinutes int                  `json:"avgResolutionMinutes"`
	ByStatus             []IncidentStatCount  `json:"byStatus"`
	ByPriority           []IncidentStatCount  `json:"byPriority"`
	DailyTrend           []IncidentTrendPoint `json:"dailyTrend"`
}
