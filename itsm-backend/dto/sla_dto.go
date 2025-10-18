package dto

import (
	"time"
)

// SLA定义相关DTO
type CreateSLADefinitionRequest struct {
	Name            string                 `json:"name" binding:"required" example:"标准服务SLA"`
	Description     string                 `json:"description" example:"标准IT服务的SLA定义"`
	ServiceType     string                 `json:"service_type" example:"standard"`
	Priority        string                 `json:"priority" example:"medium"`
	ResponseTime    int                    `json:"response_time" binding:"required,min=1" example:"30"`
	ResolutionTime  int                    `json:"resolution_time" binding:"required,min=1" example:"240"`
	BusinessHours   map[string]interface{} `json:"business_hours" example:"{\"timezone\":\"Asia/Shanghai\"}"`
	EscalationRules map[string]interface{} `json:"escalation_rules" example:"{\"levels\":[]}"`
	Conditions      map[string]interface{} `json:"conditions" example:"{\"priority\":[\"low\",\"medium\"]}"`
	IsActive        bool                   `json:"is_active" example:"true"`
}

type UpdateSLADefinitionRequest struct {
	Name            *string                `json:"name,omitempty"`
	Description     *string                `json:"description,omitempty"`
	ServiceType     *string                `json:"service_type,omitempty"`
	Priority        *string                `json:"priority,omitempty"`
	ResponseTime    *int                   `json:"response_time,omitempty"`
	ResolutionTime  *int                   `json:"resolution_time,omitempty"`
	BusinessHours   map[string]interface{} `json:"business_hours,omitempty"`
	EscalationRules map[string]interface{} `json:"escalation_rules,omitempty"`
	Conditions      map[string]interface{} `json:"conditions,omitempty"`
	IsActive        *bool                  `json:"is_active,omitempty"`
}

type SLADefinitionResponse struct {
	ID              int                    `json:"id" example:"1"`
	Name            string                 `json:"name" example:"标准服务SLA"`
	Description     string                 `json:"description" example:"标准IT服务的SLA定义"`
	ServiceType     string                 `json:"service_type" example:"standard"`
	Priority        string                 `json:"priority" example:"medium"`
	ResponseTime    int                    `json:"response_time" example:"30"`
	ResolutionTime  int                    `json:"resolution_time" example:"240"`
	BusinessHours   map[string]interface{} `json:"business_hours"`
	EscalationRules map[string]interface{} `json:"escalation_rules"`
	Conditions      map[string]interface{} `json:"conditions"`
	IsActive        bool                   `json:"is_active" example:"true"`
	TenantID        int                    `json:"tenant_id" example:"1"`
	CreatedAt       time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt       time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// SLA违规相关DTO
type CreateSLAViolationRequest struct {
	SLADefinitionID int    `json:"sla_definition_id" binding:"required" example:"1"`
	TicketID        int    `json:"ticket_id" binding:"required" example:"1"`
	ViolationType   string `json:"violation_type" binding:"required" example:"response_time"`
	Description     string `json:"description" example:"响应时间超时"`
	Severity        string `json:"severity" example:"medium"`
}

type UpdateSLAViolationRequest struct {
	Description     *string `json:"description,omitempty"`
	Severity        *string `json:"severity,omitempty"`
	IsResolved      *bool   `json:"is_resolved,omitempty"`
	ResolutionNotes *string `json:"resolution_notes,omitempty"`
}

type SLAViolationResponse struct {
	ID              int        `json:"id" example:"1"`
	SLADefinitionID int        `json:"sla_definition_id" example:"1"`
	TicketID        int        `json:"ticket_id" example:"1"`
	ViolationType   string     `json:"violation_type" example:"response_time"`
	ViolationTime   time.Time  `json:"violation_time" example:"2024-01-01T00:00:00Z"`
	Description     string     `json:"description" example:"响应时间超时"`
	Severity        string     `json:"severity" example:"medium"`
	IsResolved      bool       `json:"is_resolved" example:"false"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty" example:"2024-01-01T00:00:00Z"`
	ResolutionNotes string     `json:"resolution_notes" example:"已解决"`
	TenantID        int        `json:"tenant_id" example:"1"`
	CreatedAt       time.Time  `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt       time.Time  `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// SLA指标相关DTO
type CreateSLAMetricRequest struct {
	SLADefinitionID int                    `json:"sla_definition_id" binding:"required" example:"1"`
	MetricType      string                 `json:"metric_type" binding:"required" example:"response_time"`
	MetricName      string                 `json:"metric_name" binding:"required" example:"平均响应时间"`
	MetricValue     float64                `json:"metric_value" binding:"required" example:"25.5"`
	Unit            string                 `json:"unit" example:"分钟"`
	Metadata        map[string]interface{} `json:"metadata" example:"{\"source\":\"system\"}"`
}

type SLAMetricResponse struct {
	ID              int                    `json:"id" example:"1"`
	SLADefinitionID int                    `json:"sla_definition_id" example:"1"`
	MetricType      string                 `json:"metric_type" example:"response_time"`
	MetricName      string                 `json:"metric_name" example:"平均响应时间"`
	MetricValue     float64                `json:"metric_value" example:"25.5"`
	Unit            string                 `json:"unit" example:"分钟"`
	MeasurementTime time.Time              `json:"measurement_time" example:"2024-01-01T00:00:00Z"`
	Metadata        map[string]interface{} `json:"metadata"`
	TenantID        int                    `json:"tenant_id" example:"1"`
	CreatedAt       time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt       time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// SLA监控相关DTO
type SLAMonitoringRequest struct {
	SLADefinitionID *int   `json:"sla_definition_id,omitempty" example:"1"`
	TicketID        *int   `json:"ticket_id,omitempty" example:"1"`
	StartTime       string `json:"start_time" example:"2024-01-01T00:00:00Z"`
	EndTime         string `json:"end_time" example:"2024-01-31T23:59:59Z"`
}

type SLAMonitoringResponse struct {
	SLADefinitionID       int                    `json:"sla_definition_id" example:"1"`
	SLAInfo               SLADefinitionResponse  `json:"sla_info"`
	Metrics               []SLAMetricResponse    `json:"metrics"`
	Violations            []SLAViolationResponse `json:"violations"`
	ComplianceRate        float64                `json:"compliance_rate" example:"95.5"`
	AverageResponseTime   float64                `json:"average_response_time" example:"28.5"`
	AverageResolutionTime float64                `json:"average_resolution_time" example:"180.5"`
	TotalTickets          int                    `json:"total_tickets" example:"100"`
	ViolatedTickets       int                    `json:"violated_tickets" example:"5"`
}

// SLA报告相关DTO
type SLAReportRequest struct {
	SLADefinitionID *int   `json:"sla_definition_id,omitempty" example:"1"`
	StartTime       string `json:"start_time" binding:"required" example:"2024-01-01T00:00:00Z"`
	EndTime         string `json:"end_time" binding:"required" example:"2024-01-31T23:59:59Z"`
	GroupBy         string `json:"group_by" example:"day"` // day, week, month
}

type SLAReportResponse struct {
	Period       string                 `json:"period" example:"2024-01-01 to 2024-01-31"`
	SLAInfo      SLADefinitionResponse  `json:"sla_info"`
	Summary      SLASummary             `json:"summary"`
	DailyMetrics []SLADailyMetric       `json:"daily_metrics"`
	Violations   []SLAViolationResponse `json:"violations"`
	Trends       SLATrends              `json:"trends"`
}

type SLASummary struct {
	TotalTickets             int     `json:"total_tickets" example:"100"`
	CompliantTickets         int     `json:"compliant_tickets" example:"95"`
	ViolatedTickets          int     `json:"violated_tickets" example:"5"`
	ComplianceRate           float64 `json:"compliance_rate" example:"95.0"`
	AverageResponseTime      float64 `json:"average_response_time" example:"28.5"`
	AverageResolutionTime    float64 `json:"average_resolution_time" example:"180.5"`
	ResponseTimeCompliance   float64 `json:"response_time_compliance" example:"96.0"`
	ResolutionTimeCompliance float64 `json:"resolution_time_compliance" example:"94.0"`
}

type SLADailyMetric struct {
	Date                  string  `json:"date" example:"2024-01-01"`
	Tickets               int     `json:"tickets" example:"10"`
	CompliantTickets      int     `json:"compliant_tickets" example:"9"`
	ViolatedTickets       int     `json:"violated_tickets" example:"1"`
	ComplianceRate        float64 `json:"compliance_rate" example:"90.0"`
	AverageResponseTime   float64 `json:"average_response_time" example:"25.5"`
	AverageResolutionTime float64 `json:"average_resolution_time" example:"200.5"`
}

type SLATrends struct {
	ComplianceTrend     string  `json:"compliance_trend" example:"improving"` // improving, declining, stable
	ResponseTimeTrend   string  `json:"response_time_trend" example:"stable"`
	ResolutionTimeTrend string  `json:"resolution_time_trend" example:"improving"`
	TrendPercentage     float64 `json:"trend_percentage" example:"5.2"`
}

// SLA升级相关DTO
type SLAEscalationRequest struct {
	TicketID        int    `json:"ticket_id" binding:"required" example:"1"`
	EscalationLevel int    `json:"escalation_level" binding:"required" example:"1"`
	Reason          string `json:"reason" example:"响应时间即将超时"`
	NotifyUsers     []int  `json:"notify_users" example:"[1,2,3]"`
}

type SLAEscalationResponse struct {
	ID              int       `json:"id" example:"1"`
	TicketID        int       `json:"ticket_id" example:"1"`
	EscalationLevel int       `json:"escalation_level" example:"1"`
	Reason          string    `json:"reason" example:"响应时间即将超时"`
	Status          string    `json:"status" example:"active"`
	NotifiedUsers   []int     `json:"notified_users" example:"[1,2,3]"`
	CreatedAt       time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt       time.Time `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}
