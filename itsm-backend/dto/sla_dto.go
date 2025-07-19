package dto

import "time"

// SLADefinitionRequest SLA定义请求
type SLADefinitionRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	ServiceType    string `json:"service_type" binding:"required,oneof=incident service_request problem change"`
	Priority       string `json:"priority" binding:"required,oneof=low medium high critical"`
	Impact         string `json:"impact" binding:"required,oneof=low medium high"`
	ResponseTime   int    `json:"response_time" binding:"required,min=1"`
	ResolutionTime int    `json:"resolution_time" binding:"required,min=1"`
	BusinessHours  string `json:"business_hours"` // JSON格式的工作时间配置
	Holidays       string `json:"holidays"`       // JSON格式的节假日配置
	IsActive       bool   `json:"is_active"`
}

// SLADefinitionResponse SLA定义响应
type SLADefinitionResponse struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	ServiceType    string    `json:"service_type"`
	Priority       string    `json:"priority"`
	Impact         string    `json:"impact"`
	ResponseTime   int       `json:"response_time"`
	ResolutionTime int       `json:"resolution_time"`
	BusinessHours  string    `json:"business_hours"`
	Holidays       string    `json:"holidays"`
	IsActive       bool      `json:"is_active"`
	TenantID       int       `json:"tenant_id"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SLAViolationResponse SLA违规响应
type SLAViolationResponse struct {
	ID                  int        `json:"id"`
	TicketID            int        `json:"ticket_id"`
	TicketType          string     `json:"ticket_type"`
	ViolationType       string     `json:"violation_type"`
	SLADefinitionID     int        `json:"sla_definition_id"`
	SLAName             string     `json:"sla_name"`
	ExpectedTime        int        `json:"expected_time"`
	ActualTime          int        `json:"actual_time"`
	OverdueMinutes      int        `json:"overdue_minutes"`
	Status              string     `json:"status"`
	AssignedTo          *string    `json:"assigned_to"`
	ViolationOccurredAt time.Time  `json:"violation_occurred_at"`
	ResolvedAt          *time.Time `json:"resolved_at"`
	ResolutionNote      *string    `json:"resolution_note"`
	TenantID            int        `json:"tenant_id"`
	CreatedBy           string     `json:"created_by"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// SLAMetricsResponse SLA指标响应
type SLAMetricsResponse struct {
	ID                 int       `json:"id"`
	ServiceType        string    `json:"service_type"`
	Priority           string    `json:"priority"`
	Impact             string    `json:"impact"`
	TotalTickets       int       `json:"total_tickets"`
	MetSLATickets      int       `json:"met_sla_tickets"`
	ViolatedSLATickets int       `json:"violated_sla_tickets"`
	SLAComplianceRate  float64   `json:"sla_compliance_rate"`
	AvgResponseTime    float64   `json:"avg_response_time"`
	AvgResolutionTime  float64   `json:"avg_resolution_time"`
	Period             string    `json:"period"`
	PeriodStart        time.Time `json:"period_start"`
	PeriodEnd          time.Time `json:"period_end"`
	TenantID           int       `json:"tenant_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// SLADefinitionListResponse SLA定义列表响应
type SLADefinitionListResponse struct {
	Definitions []SLADefinitionResponse `json:"definitions"`
	Total       int                     `json:"total"`
	Page        int                     `json:"page"`
	PageSize    int                     `json:"page_size"`
}

// SLAViolationListResponse SLA违规列表响应
type SLAViolationListResponse struct {
	Violations []SLAViolationResponse `json:"violations"`
	Total      int                    `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
}

// SLAMetricsListResponse SLA指标列表响应
type SLAMetricsListResponse struct {
	Metrics  []SLAMetricsResponse `json:"metrics"`
	Total    int                  `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

// SLAComplianceReportResponse SLA达标率报表响应
type SLAComplianceReportResponse struct {
	OverallComplianceRate float64                    `json:"overall_compliance_rate"`
	ServiceTypeMetrics    map[string]ServiceMetrics  `json:"service_type_metrics"`
	PriorityMetrics       map[string]PriorityMetrics `json:"priority_metrics"`
	TrendData             []TrendData                `json:"trend_data"`
	Period                string                     `json:"period"`
	PeriodStart           time.Time                  `json:"period_start"`
	PeriodEnd             time.Time                  `json:"period_end"`
}

// ServiceMetrics 服务类型指标
type ServiceMetrics struct {
	TotalTickets       int     `json:"total_tickets"`
	MetSLATickets      int     `json:"met_sla_tickets"`
	ViolatedSLATickets int     `json:"violated_sla_tickets"`
	ComplianceRate     float64 `json:"compliance_rate"`
	AvgResponseTime    float64 `json:"avg_response_time"`
	AvgResolutionTime  float64 `json:"avg_resolution_time"`
}

// PriorityMetrics 优先级指标
type PriorityMetrics struct {
	TotalTickets       int     `json:"total_tickets"`
	MetSLATickets      int     `json:"met_sla_tickets"`
	ViolatedSLATickets int     `json:"violated_sla_tickets"`
	ComplianceRate     float64 `json:"compliance_rate"`
	AvgResponseTime    float64 `json:"avg_response_time"`
	AvgResolutionTime  float64 `json:"avg_resolution_time"`
}

// TrendData 趋势数据
type TrendData struct {
	Date            time.Time `json:"date"`
	ComplianceRate  float64   `json:"compliance_rate"`
	TotalTickets    int       `json:"total_tickets"`
	ViolatedTickets int       `json:"violated_tickets"`
}

// BusinessHoursConfig 工作时间配置
type BusinessHoursConfig struct {
	Monday    DaySchedule `json:"monday"`
	Tuesday   DaySchedule `json:"tuesday"`
	Wednesday DaySchedule `json:"wednesday"`
	Thursday  DaySchedule `json:"thursday"`
	Friday    DaySchedule `json:"friday"`
	Saturday  DaySchedule `json:"saturday"`
	Sunday    DaySchedule `json:"sunday"`
}

// DaySchedule 每日工作时间
type DaySchedule struct {
	IsWorkingDay bool   `json:"is_working_day"`
	StartTime    string `json:"start_time"` // HH:MM格式
	EndTime      string `json:"end_time"`   // HH:MM格式
}

// HolidayConfig 节假日配置
type HolidayConfig struct {
	Holidays []Holiday `json:"holidays"`
}

// Holiday 节假日
type Holiday struct {
	Date         time.Time `json:"date"`
	Name         string    `json:"name"`
	IsWorkingDay bool      `json:"is_working_day"`
}
