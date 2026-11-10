package dto

import "time"

// 工作台概览与管理者服务质量视图的值类型。
// 原定义在 service/dashboard_service.go 与 service/service_quality_report_service.go；
// service 侧保留同名类型别名，保持公开 API 与类型身份不变。
// 域内窄契约的签名需要这些类型，故按类型闭包成组下沉到本包。

// MaxQualityWindowDays 管理者视图的统计窗口上限，超出请走各单域报表或导出通道。
const MaxQualityWindowDays = 90

type DashboardOverviewStats struct {
	TotalTickets      int
	PendingTickets    int
	InProgressTickets int
	ResolvedToday     int
	AvgResponseTime   float64
	AvgResolutionTime float64
}

type SLAComplianceData struct {
	ComplianceRate           float64 `json:"complianceRate"`
	ResponseTimeCompliance   float64 `json:"responseTimeCompliance"`
	ResolutionTimeCompliance float64 `json:"resolutionTimeCompliance"`
	AtRiskTickets            int     `json:"atRiskTickets"`
	BreachedTickets          int     `json:"breachedTickets"`
	TotalTickets             int     `json:"totalTickets"`
	CompliantTickets         int     `json:"compliantTickets"`
}

type DashboardOverviewData struct {
	KPIMetrics               []KPIMetricData                `json:"kpiMetrics"`
	TicketTrend              []TicketTrendData              `json:"ticketTrend"`
	IncidentDistribution     []IncidentDistributionData     `json:"incidentDistribution"`
	SLAData                  *SLAComplianceData             `json:"slaData"`
	SatisfactionData         []SatisfactionData             `json:"satisfactionData"`
	QuickActions             []QuickActionData              `json:"quickActions"`
	RecentActivities         []RecentActivityData           `json:"recentActivities"`
	ResponseTimeDistribution []ResponseTimeDistributionData `json:"responseTimeDistribution,omitempty"`
	TeamWorkload             []TeamWorkloadData             `json:"teamWorkload,omitempty"`
	PeakHours                []PeakHourData                 `json:"peakHours,omitempty"`
}

type KPIMetricData struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Color       string  `json:"color"`
	Trend       string  `json:"trend"`
	Change      float64 `json:"change"`
	ChangeType  string  `json:"changeType"`
	Description string  `json:"description,omitempty"`
	Target      float64 `json:"target,omitempty"`
	Alert       string  `json:"alert,omitempty"`
	// Comparable 标识环比是否可信：false 表示读取者范围内没有可比的上期基线，
	// 此时 Change 恒为 0，前端不得渲染"相比上期"。
	Comparable bool `json:"comparable"`
	// Scope 标识本组指标的行级范围：tenant = 全租户，own = 本人创建或受理。
	Scope string `json:"scope"`
}

type TicketTrendData struct {
	Date             string `json:"date"`
	Open             int    `json:"open"`
	InProgress       int    `json:"inProgress"`
	Resolved         int    `json:"resolved"`
	Closed           int    `json:"closed"`
	NewTickets       int    `json:"newTickets,omitempty"`
	CompletedTickets int    `json:"completedTickets,omitempty"`
	PendingTickets   int    `json:"pendingTickets,omitempty"`
}

type IncidentDistributionData struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
	Color    string `json:"color"`
}

type SatisfactionData struct {
	Month     string  `json:"month"`
	Rating    float64 `json:"rating"`
	Responses int     `json:"responses"`
}

type QuickActionData struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Color       string `json:"color"`
	Permission  string `json:"permission,omitempty"`
}

type RecentActivityData struct {
	ID           int       `json:"id"`
	Type         string    `json:"type"`
	TicketID     int       `json:"ticketId"`
	TicketNumber string    `json:"ticketNumber"`
	TicketTitle  string    `json:"ticketTitle"`
	Status       string    `json:"status"`
	StatusName   string    `json:"statusName"`
	Operator     string    `json:"operator"`
	Assignee     string    `json:"assignee"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ResponseTimeDistributionData struct {
	TimeRange  string  `json:"timeRange"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
	AvgTime    float64 `json:"avgTime,omitempty"`
}

type TeamWorkloadData struct {
	Assignee        string  `json:"assignee"`
	TicketCount     int     `json:"ticketCount"`
	AvgResponseTime float64 `json:"avgResponseTime"`
	CompletionRate  float64 `json:"completionRate"`
	ActiveTickets   int     `json:"activeTickets,omitempty"`
}

type PeakHourData struct {
	Hour            string  `json:"hour"`
	Count           int     `json:"count"`
	AvgResponseTime float64 `json:"avgResponseTime,omitempty"`
}

type ServiceQualityWindow struct {
	Days int       `json:"days"`
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type ServiceQualityReport struct {
	MetricVersion  string               `json:"metricVersion"`
	MetricVersions map[string]string    `json:"metricVersions"`
	GeneratedAt    time.Time            `json:"generatedAt"`
	Window         ServiceQualityWindow `json:"window"`
	Scope          string               `json:"scope"` // KPI 行级口径：tenant | own

	KPI                  []KPIMetricData               `json:"kpi"`
	SLA                  *SLAComplianceData            `json:"sla"`
	IncidentDistribution []IncidentDistributionData    `json:"incidentDistribution"`
	Satisfaction         []SatisfactionData            `json:"satisfaction"`
	TicketTrend          []TicketTrendData             `json:"ticketTrend"`
	TicketStats          *AnalyticsTicketStatsResponse `json:"ticketStats,omitempty"`  // analytics 未装配时缺省
	ProblemStats         *ProblemStatsResponse         `json:"problemStats,omitempty"` // problem 域未装配时缺省
	ChangeStats          *ChangeStatsResponse          `json:"changeStats,omitempty"`  // change 域未装配时缺省
}
