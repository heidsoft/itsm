package dto

import (
	"time"
)

// 事件管理相关DTO
type CreateIncidentRequest struct {
	Title               string                 `json:"title" binding:"required" example:"服务器CPU使用率过高"`
	Description         string                 `json:"description" example:"生产环境Web服务器CPU使用率持续超过90%"`
	Priority            string                 `json:"priority" example:"high"`
	Severity            string                 `json:"severity" example:"high"`
	Category            string                 `json:"category" example:"performance"`
	Subcategory         string                 `json:"subcategory" example:"cpu"`
	ConfigurationItemID *int                   `json:"configuration_item_id" example:"1"`
	AssigneeID          *int                   `json:"assignee_id" example:"1"`
	ImpactAnalysis      map[string]interface{} `json:"impact_analysis" example:"{\"affected_users\":100}"`
	Source              string                 `json:"source" example:"monitoring"`
	Metadata            map[string]interface{} `json:"metadata" example:"{\"alert_id\":\"alert_001\"}"`
	DetectedAt          *time.Time             `json:"detected_at" example:"2024-01-01T00:00:00Z"`
}

type UpdateIncidentRequest struct {
	Title           *string                  `json:"title,omitempty"`
	Description     *string                  `json:"description,omitempty"`
	Status          *string                  `json:"status,omitempty"`
	Priority        *string                  `json:"priority,omitempty"`
	Severity        *string                  `json:"severity,omitempty"`
	Category        *string                  `json:"category,omitempty"`
	Subcategory     *string                  `json:"subcategory,omitempty"`
	AssigneeID      *int                     `json:"assignee_id,omitempty"`
	ImpactAnalysis  map[string]interface{}   `json:"impact_analysis,omitempty"`
	RootCause       map[string]interface{}   `json:"root_cause,omitempty"`
	ResolutionSteps []map[string]interface{} `json:"resolution_steps,omitempty"`
	Metadata        map[string]interface{}   `json:"metadata,omitempty"`
}

type IncidentResponse struct {
	ID                  int                      `json:"id" example:"1"`
	Title               string                   `json:"title" example:"服务器CPU使用率过高"`
	Description         string                   `json:"description" example:"生产环境Web服务器CPU使用率持续超过90%"`
	Status              string                   `json:"status" example:"new"`
	Priority            string                   `json:"priority" example:"high"`
	Severity            string                   `json:"severity" example:"high"`
	IncidentNumber      string                   `json:"incident_number" example:"INC-000001"`
	ReporterID          int                      `json:"reporter_id" example:"1"`
	AssigneeID          *int                     `json:"assignee_id" example:"2"`
	ConfigurationItemID *int                     `json:"configuration_item_id" example:"1"`
	Category            string                   `json:"category" example:"performance"`
	Subcategory         string                   `json:"subcategory" example:"cpu"`
	ImpactAnalysis      map[string]interface{}   `json:"impact_analysis"`
	RootCause           map[string]interface{}   `json:"root_cause"`
	ResolutionSteps     []map[string]interface{} `json:"resolution_steps"`
	DetectedAt          time.Time                `json:"detected_at" example:"2024-01-01T00:00:00Z"`
	ResolvedAt          *time.Time               `json:"resolved_at,omitempty" example:"2024-01-01T12:00:00Z"`
	ClosedAt            *time.Time               `json:"closed_at,omitempty" example:"2024-01-01T18:00:00Z"`
	EscalatedAt         *time.Time               `json:"escalated_at,omitempty" example:"2024-01-01T06:00:00Z"`
	EscalationLevel     int                      `json:"escalation_level" example:"1"`
	IsAutomated         bool                     `json:"is_automated" example:"false"`
	Source              string                   `json:"source" example:"monitoring"`
	Metadata            map[string]interface{}   `json:"metadata"`
	TenantID            int                      `json:"tenant_id" example:"1"`
	CreatedAt           time.Time                `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt           time.Time                `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// 事件活动记录DTO
type CreateIncidentEventRequest struct {
	IncidentID  int                    `json:"incident_id" binding:"required" example:"1"`
	EventType   string                 `json:"event_type" binding:"required" example:"status_change"`
	EventName   string                 `json:"event_name" binding:"required" example:"状态变更"`
	Description string                 `json:"description" example:"事件状态从new变更为in_progress"`
	Status      string                 `json:"status" example:"active"`
	Severity    string                 `json:"severity" example:"medium"`
	Data        map[string]interface{} `json:"data" example:"{\"old_status\":\"new\",\"new_status\":\"in_progress\"}"`
	OccurredAt  *time.Time             `json:"occurred_at" example:"2024-01-01T00:00:00Z"`
	UserID      *int                   `json:"user_id" example:"1"`
	Source      string                 `json:"source" example:"system"`
	Metadata    map[string]interface{} `json:"metadata" example:"{\"ip_address\":\"192.168.1.1\"}"`
}

type IncidentEventResponse struct {
	ID          int                    `json:"id" example:"1"`
	IncidentID  int                    `json:"incident_id" example:"1"`
	EventType   string                 `json:"event_type" example:"status_change"`
	EventName   string                 `json:"event_name" example:"状态变更"`
	Description string                 `json:"description" example:"事件状态从new变更为in_progress"`
	Status      string                 `json:"status" example:"active"`
	Severity    string                 `json:"severity" example:"medium"`
	Data        map[string]interface{} `json:"data"`
	OccurredAt  time.Time              `json:"occurred_at" example:"2024-01-01T00:00:00Z"`
	UserID      *int                   `json:"user_id" example:"1"`
	Source      string                 `json:"source" example:"system"`
	Metadata    map[string]interface{} `json:"metadata"`
	TenantID    int                    `json:"tenant_id" example:"1"`
	CreatedAt   time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt   time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// 事件告警DTO
type CreateIncidentAlertRequest struct {
	IncidentID  int                    `json:"incident_id" binding:"required" example:"1"`
	AlertType   string                 `json:"alert_type" binding:"required" example:"escalation"`
	AlertName   string                 `json:"alert_name" binding:"required" example:"事件升级告警"`
	Message     string                 `json:"message" binding:"required" example:"事件已升级到下一级别"`
	Severity    string                 `json:"severity" example:"high"`
	Channels    []string               `json:"channels" example:"[\"email\",\"sms\"]"`
	Recipients  []string               `json:"recipients" example:"[\"manager@company.com\"]"`
	TriggeredAt *time.Time             `json:"triggered_at" example:"2024-01-01T00:00:00Z"`
	Metadata    map[string]interface{} `json:"metadata" example:"{\"escalation_level\":1}"`
}

type IncidentAlertResponse struct {
	ID             int                    `json:"id" example:"1"`
	IncidentID     int                    `json:"incident_id" example:"1"`
	AlertType      string                 `json:"alert_type" example:"escalation"`
	AlertName      string                 `json:"alert_name" example:"事件升级告警"`
	Message        string                 `json:"message" example:"事件已升级到下一级别"`
	Severity       string                 `json:"severity" example:"high"`
	Status         string                 `json:"status" example:"active"`
	Channels       []string               `json:"channels" example:"[\"email\",\"sms\"]"`
	Recipients     []string               `json:"recipients" example:"[\"manager@company.com\"]"`
	TriggeredAt    time.Time              `json:"triggered_at" example:"2024-01-01T00:00:00Z"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty" example:"2024-01-01T01:00:00Z"`
	ResolvedAt     *time.Time             `json:"resolved_at,omitempty" example:"2024-01-01T02:00:00Z"`
	AcknowledgedBy *int                   `json:"acknowledged_by" example:"1"`
	Metadata       map[string]interface{} `json:"metadata"`
	TenantID       int                    `json:"tenant_id" example:"1"`
	CreatedAt      time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt      time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// 事件指标DTO
type CreateIncidentMetricRequest struct {
	IncidentID  int                    `json:"incident_id" binding:"required" example:"1"`
	MetricType  string                 `json:"metric_type" binding:"required" example:"response_time"`
	MetricName  string                 `json:"metric_name" binding:"required" example:"平均响应时间"`
	MetricValue float64                `json:"metric_value" binding:"required" example:"2.5"`
	Unit        string                 `json:"unit" example:"秒"`
	MeasuredAt  *time.Time             `json:"measured_at" example:"2024-01-01T00:00:00Z"`
	Tags        map[string]string      `json:"tags" example:"{\"environment\":\"production\"}"`
	Metadata    map[string]interface{} `json:"metadata" example:"{\"source\":\"monitoring\"}"`
}

type IncidentMetricResponse struct {
	ID          int                    `json:"id" example:"1"`
	IncidentID  int                    `json:"incident_id" example:"1"`
	MetricType  string                 `json:"metric_type" example:"response_time"`
	MetricName  string                 `json:"metric_name" example:"平均响应时间"`
	MetricValue float64                `json:"metric_value" example:"2.5"`
	Unit        string                 `json:"unit" example:"秒"`
	MeasuredAt  time.Time              `json:"measured_at" example:"2024-01-01T00:00:00Z"`
	Tags        map[string]string      `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	TenantID    int                    `json:"tenant_id" example:"1"`
	CreatedAt   time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt   time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// 事件规则DTO
type CreateIncidentRuleRequest struct {
	Name        string                   `json:"name" binding:"required" example:"高优先级事件自动升级"`
	Description string                   `json:"description" example:"当事件优先级为high或urgent时，自动升级到下一级别"`
	RuleType    string                   `json:"rule_type" binding:"required" example:"escalation"`
	Conditions  map[string]interface{}   `json:"conditions" example:"{\"priority\":[\"high\",\"urgent\"]}"`
	Actions     []map[string]interface{} `json:"actions" example:"[{\"type\":\"escalate\",\"level\":1}]"`
	Priority    string                   `json:"priority" example:"high"`
	IsActive    bool                     `json:"is_active" example:"true"`
	Metadata    map[string]interface{}   `json:"metadata" example:"{\"version\":\"1.0\"}"`
}

type UpdateIncidentRuleRequest struct {
	Name        *string                  `json:"name,omitempty"`
	Description *string                  `json:"description,omitempty"`
	RuleType    *string                  `json:"rule_type,omitempty"`
	Conditions  map[string]interface{}   `json:"conditions,omitempty"`
	Actions     []map[string]interface{} `json:"actions,omitempty"`
	Priority    *string                  `json:"priority,omitempty"`
	IsActive    *bool                    `json:"is_active,omitempty"`
	Metadata    map[string]interface{}   `json:"metadata,omitempty"`
}

type IncidentRuleResponse struct {
	ID             int                      `json:"id" example:"1"`
	Name           string                   `json:"name" example:"高优先级事件自动升级"`
	Description    string                   `json:"description" example:"当事件优先级为high或urgent时，自动升级到下一级别"`
	RuleType       string                   `json:"rule_type" example:"escalation"`
	Conditions     map[string]interface{}   `json:"conditions"`
	Actions        []map[string]interface{} `json:"actions"`
	Priority       string                   `json:"priority" example:"high"`
	IsActive       bool                     `json:"is_active" example:"true"`
	ExecutionCount int                      `json:"execution_count" example:"5"`
	LastExecutedAt *time.Time               `json:"last_executed_at,omitempty" example:"2024-01-01T00:00:00Z"`
	Metadata       map[string]interface{}   `json:"metadata"`
	TenantID       int                      `json:"tenant_id" example:"1"`
	CreatedAt      time.Time                `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt      time.Time                `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// 事件监控DTO
type IncidentMonitoringRequest struct {
	IncidentID *int    `json:"incident_id,omitempty" example:"1"`
	Category   *string `json:"category,omitempty" example:"performance"`
	Priority   *string `json:"priority,omitempty" example:"high"`
	Status     *string `json:"status,omitempty" example:"new"`
	StartTime  string  `json:"start_time" example:"2024-01-01T00:00:00Z"`
	EndTime    string  `json:"end_time" example:"2024-01-31T23:59:59Z"`
}

type IncidentMonitoringResponse struct {
	TotalIncidents        int                      `json:"total_incidents" example:"100"`
	OpenIncidents         int                      `json:"open_incidents" example:"25"`
	ResolvedIncidents     int                      `json:"resolved_incidents" example:"70"`
	ClosedIncidents       int                      `json:"closed_incidents" example:"5"`
	CriticalIncidents     int                      `json:"critical_incidents" example:"10"`
	HighPriorityIncidents int                      `json:"high_priority_incidents" example:"20"`
	AverageResolutionTime float64                  `json:"average_resolution_time" example:"4.5"`
	ResolutionRate        float64                  `json:"resolution_rate" example:"95.0"`
	EscalationRate        float64                  `json:"escalation_rate" example:"15.0"`
	Incidents             []IncidentResponse       `json:"incidents"`
	Metrics               []IncidentMetricResponse `json:"metrics"`
	Alerts                []IncidentAlertResponse  `json:"alerts"`
}

// 事件升级DTO
type IncidentEscalationRequest struct {
	IncidentID      int    `json:"incident_id" binding:"required" example:"1"`
	EscalationLevel int    `json:"escalation_level" binding:"required" example:"1"`
	Reason          string `json:"reason" example:"事件处理时间过长"`
	NotifyUsers     []int  `json:"notify_users" example:"[1,2,3]"`
	AutoAssign      bool   `json:"auto_assign" example:"true"`
}

type IncidentEscalationResponse struct {
	ID              int       `json:"id" example:"1"`
	IncidentID      int       `json:"incident_id" example:"1"`
	EscalationLevel int       `json:"escalation_level" example:"1"`
	Reason          string    `json:"reason" example:"事件处理时间过长"`
	Status          string    `json:"status" example:"active"`
	NotifiedUsers   []int     `json:"notified_users" example:"[1,2,3]"`
	AutoAssigned    bool      `json:"auto_assigned" example:"true"`
	CreatedAt       time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt       time.Time `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// 事件关联分析DTO
type IncidentCorrelationRequest struct {
	IncidentID          int     `json:"incident_id" binding:"required" example:"1"`
	SimilarityThreshold float64 `json:"similarity_threshold" example:"0.8"`
	TimeWindowHours     int     `json:"time_window_hours" example:"24"`
	MaxResults          int     `json:"max_results" example:"10"`
}

type IncidentCorrelationResponse struct {
	IncidentID          int                    `json:"incident_id" example:"1"`
	CorrelatedIncidents []CorrelatedIncident   `json:"correlated_incidents"`
	CorrelationScore    float64                `json:"correlation_score" example:"0.85"`
	AnalysisResult      map[string]interface{} `json:"analysis_result"`
}

type CorrelatedIncident struct {
	IncidentID        int       `json:"incident_id" example:"2"`
	IncidentNumber    string    `json:"incident_number" example:"INC-000002"`
	Title             string    `json:"title" example:"数据库连接超时"`
	SimilarityScore   float64   `json:"similarity_score" example:"0.85"`
	CorrelationReason string    `json:"correlation_reason" example:"相似的错误模式和影响范围"`
	CreatedAt         time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
}

// IncidentStatsResponse 事件统计响应
type IncidentStatsResponse struct {
	TotalIncidents    int     `json:"total_incidents"`
	OpenIncidents     int     `json:"open_incidents"`
	CriticalIncidents int     `json:"critical_incidents"`
	MajorIncidents    int     `json:"major_incidents"`
	AvgResolutionTime float64 `json:"avg_resolution_time"`
	MTTA              float64 `json:"mtta"` // Mean Time To Acknowledge
	MTTR              float64 `json:"mttr"` // Mean Time To Resolve
}
