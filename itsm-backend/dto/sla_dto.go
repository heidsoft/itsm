package dto

import (
	"time"
)

// SLA定义相关DTO
type CreateSLADefinitionRequest struct {
	Name            string                 `json:"name" binding:"required" example:"标准服务SLA"`
	Description     string                 `json:"description" example:"标准IT服务的SLA定义"`
	ServiceType     string                 `json:"serviceType" example:"standard"`
	Priority        string                 `json:"priority" example:"medium"`
	ResponseTime    int                    `json:"responseTime" binding:"required,min=1" example:"30"`
	ResolutionTime  int                    `json:"resolutionTime" binding:"required,min=1" example:"240"`
	BusinessHours   map[string]interface{} `json:"businessHours" example:"{\"timezone\":\"Asia/Shanghai\"}"`
	EscalationRules map[string]interface{} `json:"escalationRules" example:"{\"levels\":[]}"`
	Conditions      map[string]interface{} `json:"conditions" example:"{\"priority\":[\"low\",\"medium\"]}"`
	IsActive        bool                   `json:"isActive" example:"true"`
}

type UpdateSLADefinitionRequest struct {
	Name            *string                `json:"name,omitempty"`
	Description     *string                `json:"description,omitempty"`
	ServiceType     *string                `json:"serviceType,omitempty"`
	Priority        *string                `json:"priority,omitempty"`
	ResponseTime    *int                   `json:"responseTime,omitempty"`
	ResolutionTime  *int                   `json:"resolutionTime,omitempty"`
	BusinessHours   map[string]interface{} `json:"businessHours,omitempty"`
	EscalationRules map[string]interface{} `json:"escalationRules,omitempty"`
	Conditions      map[string]interface{} `json:"conditions,omitempty"`
	IsActive        *bool                  `json:"isActive,omitempty"`
}

// SLABusinessHours / SLAEscalationLevel 把存储层的 snake_case JSON 键投影成 API 的 camelCase，
// 前端不再直接读 business_hours / escalation_rules 的原始键。
type SLABusinessHours struct {
	TimeZone  string `json:"timeZone,omitempty"`
	AlwaysOn  bool   `json:"alwaysOn"`
	WorkDays  []int  `json:"workDays,omitempty"`
	StartTime string `json:"startTime,omitempty"`
	EndTime   string `json:"endTime,omitempty"`
}

// SLAEscalationLevel 是 escalation_rules.levels[] 在 API 边界的 camelCase 视图。
type SLAEscalationLevel struct {
	Level            int      `json:"level"`
	ThresholdMinutes int      `json:"thresholdMinutes"`
	NotifyRoles      []string `json:"notifyRoles,omitempty"`
}

type SLADefinitionResponse struct {
	ID               int                    `json:"id" example:"1"`
	Name             string                 `json:"name" example:"标准服务SLA"`
	Description      string                 `json:"description" example:"标准IT服务的SLA定义"`
	ServiceType      string                 `json:"serviceType" example:"standard"`
	Priority         string                 `json:"priority" example:"medium"`
	ResponseTime     int                    `json:"responseTime" example:"30"`
	ResolutionTime   int                    `json:"resolutionTime" example:"240"`
	BusinessHours    *SLABusinessHours      `json:"businessHours,omitempty"`
	EscalationLevels []SLAEscalationLevel   `json:"escalationLevels,omitempty"`
	Conditions       map[string]interface{} `json:"conditions"`
	IsActive         bool                   `json:"isActive" example:"true"`
	TenantID         int                    `json:"tenantId" example:"1"`
	CreatedAt        time.Time              `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt        time.Time              `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// SLA违规相关DTO
type CreateSLAViolationRequest struct {
	SLADefinitionID int    `json:"slaDefinitionId" binding:"required" example:"1"`
	TicketID        int    `json:"ticketId" binding:"required" example:"1"`
	ViolationType   string `json:"violationType" binding:"required" example:"response_time"`
	Description     string `json:"description" example:"响应时间超时"`
	Severity        string `json:"severity" example:"medium"`
}

type UpdateSLAViolationRequest struct {
	Description     *string `json:"description,omitempty"`
	Severity        *string `json:"severity,omitempty"`
	IsResolved      *bool   `json:"isResolved,omitempty"`
	ResolutionNotes *string `json:"resolutionNotes,omitempty"`
}

type SLAViolationResponse struct {
	ID              int        `json:"id" example:"1"`
	SLADefinitionID int        `json:"slaDefinitionId" example:"1"`
	TicketID        int        `json:"ticketId" example:"1"`
	ViolationType   string     `json:"violationType" example:"response_time"`
	ViolationTime   time.Time  `json:"violationTime" example:"2024-01-01T00:00:00Z"`
	Description     string     `json:"description" example:"响应时间超时"`
	Severity        string     `json:"severity" example:"medium"`
	IsResolved      bool       `json:"isResolved" example:"false"`
	ResolvedAt      *time.Time `json:"resolvedAt,omitempty" example:"2024-01-01T00:00:00Z"`
	ResolutionNotes string     `json:"resolutionNotes" example:"已解决"`
	TenantID        int        `json:"tenantId" example:"1"`
	CreatedAt       time.Time  `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt       time.Time  `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// SLAMonitoringRequest 监控大屏指标查询。窗口缺省时服务端使用最近 30 天。
// 监控接口只按认证中间件注入的租户上下文 + 时间窗口聚合，不接受请求体里的
// slaDefinitionId / ticketId 等自报字段，避免跨范围注入。
type SLAMonitoringRequest struct {
	StartTime string `json:"startTime" example:"2024-01-01T00:00:00Z"`
	EndTime   string `json:"endTime" example:"2024-01-31T23:59:59Z"`
}

type SLAReportPeriod struct {
	StartDate string `json:"startDate" example:"2024-01-01T00:00:00Z"`
	EndDate   string `json:"endDate" example:"2024-01-31T23:59:59Z"`
}

// SLAComplianceReport represents SLA compliance report
type SLAComplianceReport struct {
	TotalTickets      int             `json:"totalTickets" example:"100"`
	MetSLA            int             `json:"metSla" example:"95"`
	ViolatedSLA       int             `json:"violatedSla" example:"5"`
	ComplianceRate    float64         `json:"complianceRate" example:"95.0"`
	AvgResponseTime   float64         `json:"avgResponseTime" example:"28.5"`
	AvgResolutionTime float64         `json:"avgResolutionTime" example:"180.5"`
	ReportPeriod      SLAReportPeriod `json:"reportPeriod"`
}

// SLAStatsResponse 是 GET /sla/stats 的契约。字段必须是 camelCase：
// 前端 sla-api.ts 早已按 totalDefinitions/activeDefinitions 读取，
// 后端此前返回 snake_case map，页面拿到的永远是 undefined。
// 计数来自 Count 查询，不受列表分页上限影响（旧实现最多只看到 1000 行）。
type SLAStatsResponse struct {
	TotalDefinitions      int     `json:"totalDefinitions" example:"12"`
	ActiveDefinitions     int     `json:"activeDefinitions" example:"10"`
	TotalViolations       int     `json:"totalViolations" example:"37"`
	OpenViolations        int     `json:"openViolations" example:"4"`
	OverallComplianceRate float64 `json:"overallComplianceRate" example:"92.5"`
}

// Existing SLA report DTOs remain unchanged...

type SLASummary struct {
	TotalTickets             int     `json:"totalTickets" example:"100"`
	CompliantTickets         int     `json:"compliantTickets" example:"95"`
	ViolatedTickets          int     `json:"violatedTickets" example:"5"`
	ComplianceRate           float64 `json:"complianceRate" example:"95.0"`
	AverageResponseTime      float64 `json:"averageResponseTime" example:"28.5"`
	AverageResolutionTime    float64 `json:"averageResolutionTime" example:"180.5"`
	ResponseTimeCompliance   float64 `json:"responseTimeCompliance" example:"96.0"`
	ResolutionTimeCompliance float64 `json:"resolutionTimeCompliance" example:"94.0"`
}

type SLADailyMetric struct {
	Date                  string  `json:"date" example:"2024-01-01"`
	Tickets               int     `json:"tickets" example:"10"`
	CompliantTickets      int     `json:"compliantTickets" example:"9"`
	ViolatedTickets       int     `json:"violatedTickets" example:"1"`
	ComplianceRate        float64 `json:"complianceRate" example:"90.0"`
	AverageResponseTime   float64 `json:"averageResponseTime" example:"25.5"`
	AverageResolutionTime float64 `json:"averageResolutionTime" example:"200.5"`
}

type SLATrends struct {
	ComplianceTrend     string  `json:"complianceTrend" example:"improving"` // improving, declining, stable
	ResponseTimeTrend   string  `json:"responseTimeTrend" example:"stable"`
	ResolutionTimeTrend string  `json:"resolutionTimeTrend" example:"improving"`
	TrendPercentage     float64 `json:"trendPercentage" example:"5.2"`
}

// SLA升级相关DTO
type SLAEscalationRequest struct {
	TicketID        int    `json:"ticketId" binding:"required" example:"1"`
	EscalationLevel int    `json:"escalationLevel" binding:"required" example:"1"`
	Reason          string `json:"reason" example:"响应时间即将超时"`
	NotifyUsers     []int  `json:"notifyUsers" example:"[1,2,3]"`
}

type SLAEscalationResponse struct {
	ID              int       `json:"id" example:"1"`
	TicketID        int       `json:"ticketId" example:"1"`
	EscalationLevel int       `json:"escalationLevel" example:"1"`
	Reason          string    `json:"reason" example:"响应时间即将超时"`
	Status          string    `json:"status" example:"active"`
	NotifiedUsers   []int     `json:"notifiedUsers" example:"[1,2,3]"`
	CreatedAt       time.Time `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt       time.Time `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// SLAMonitoringDashboard represents comprehensive SLA monitoring dashboard
type SLAMonitoringDashboard struct {
	ComplianceRate    float64            `json:"complianceRate"`
	ViolationRate     float64            `json:"violationRate"`
	TotalTickets      int                `json:"totalTickets"`
	AtRiskTickets     int                `json:"atRiskTickets"`
	BreachedTickets   int                `json:"breachedTickets"`
	UpcomingDeadlines []SLADeadline      `json:"upcomingDeadlines"`
	TopViolations     []SLAViolationItem `json:"topViolations"`
	SLAByPriority     map[string]float64 `json:"slaByPriority"`
	TrendData         []SLATrendPoint    `json:"trendData"`
}

type SLADeadline struct {
	TicketID    int       `json:"ticketId"`
	TicketTitle string    `json:"ticketTitle"`
	Deadline    time.Time `json:"deadline"`
	SLAPolicy   string    `json:"slaPolicy"`
	TimeLeft    string    `json:"timeLeft"`
}

type SLAViolationItem struct {
	TicketID    int    `json:"ticketId"`
	TicketTitle string `json:"ticketTitle"`
	SLAPolicy   string `json:"slaPolicy"`
	ViolatedAt  string `json:"violatedAt"`
	Delay       int    `json:"delayMinutes"`
}

type SLATrendPoint struct {
	Date           string  `json:"date"`
	ComplianceRate float64 `json:"complianceRate"`
	TicketCount    int     `json:"ticketCount"`
}
