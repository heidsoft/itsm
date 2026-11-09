package dto

import "time"

// SLAMonitorMetrics 是 SLA 监控服务计算窗口内指标的结果。
// 注意与 dashboard_dto.go 的 SLAMetrics（仪表盘达成率视图）不是同一个结构，
// 故独立命名；service 侧以别名保留 SLAMetrics 这一公开名。
type SLAMonitorMetrics struct {
	ResponseTime       float64 `json:"responseTime"`
	ResolutionTime     float64 `json:"resolutionTime"`
	SLACompliance      float64 `json:"slaCompliance"`
	TotalTickets       int     `json:"totalTickets"`
	ViolatedTickets    int     `json:"violatedTickets"`
	AvgResponseMinutes float64 `json:"avgResponseMinutes"`
	AvgResolutionHours float64 `json:"avgResolutionHours"`
}

// TicketSLAInfo 工单 SLA 信息（V2 内联定义，避免与 V1 重复）。
// 定义下沉到 dto，使 handlers/ticket 的窄契约无需引用共享层具体类型；
// service 侧以类型别名保留原公开名，JSON 字段与 tag 完全不变。
type TicketSLAInfo struct {
	TicketID             int        `json:"ticketId"`
	TicketNumber         string     `json:"ticketNumber"`
	Priority             string     `json:"priority"`
	SLADefinitionID      int        `json:"slaDefinitionId"`
	SLADefinitionName    string     `json:"slaDefinitionName"`
	ResponseDeadline     time.Time  `json:"responseDeadline"`
	ResolutionDeadline   time.Time  `json:"resolutionDeadline"`
	ResponseTimeLeft     int        `json:"responseTimeLeftMinutes"`
	IsResponseBreached   bool       `json:"isResponseBreached"`
	ResolutionTimeLeft   int        `json:"resolutionTimeLeftMinutes"`
	IsResolutionBreached bool       `json:"isResolutionBreached"`
	FirstResponseAt      *time.Time `json:"firstResponseAt,omitempty"`
	ResolvedAt           *time.Time `json:"resolvedAt,omitempty"`
}
