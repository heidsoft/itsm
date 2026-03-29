package dto

import (
	"time"
)

// 结构化字段定义

type ImpactAnalysis struct {
	BusinessImpact  *BusinessImpact `json:"business_impact,omitempty"`
	TechnicalImpact string          `json:"technical_impact,omitempty"`
	AffectedUsers   int             `json:"affected_users,omitempty"`
	TimeImpact      *TimeImpact     `json:"time_impact,omitempty"`
}

type BusinessImpact struct {
	AffectedUsers       int     `json:"affected_users,omitempty"`
	RevenueImpact       float64 `json:"revenue_impact,omitempty"`
	ServiceAvailability float64 `json:"service_availability,omitempty"`
}

type TimeImpact struct {
	IsOverdue          bool   `json:"is_overdue,omitempty"`
	HoursSinceCreation int    `json:"hours_since_creation,omitempty"`
	ResponseDeadline   string `json:"response_deadline,omitempty"`
	ResolutionDeadline string `json:"resolution_deadline,omitempty"`
}

type RootCause struct {
	AnalysisMethod      string   `json:"analysis_method,omitempty"`
	RootCause           string   `json:"root_cause,omitempty"`
	ContributingFactors []string `json:"contributing_factors,omitempty"`
	Evidence            []string `json:"evidence,omitempty"`
	PreventiveActions   []string `json:"preventive_actions,omitempty"`
	Status              string   `json:"status,omitempty"`
}

type ResolutionStep struct {
	Step        int       `json:"step"`
	Description string    `json:"description"`
	ExecutedBy  string    `json:"executed_by"`
	ExecutedAt  time.Time `json:"executed_at"`
	Status      string    `json:"status"` // pending, in_progress, completed, failed
}

// 事件管理相关DTO
type CreateIncidentRequest struct {
	Title               string                 `json:"title" binding:"required" example:"服务器CPU使用率过高"`
	Description         string                 `json:"description" example:"生产环境Web服务器CPU使用率持续超过90%"`
	Type                string                 `json:"type" binding:"omitempty,oneof=incident service_request security_event alert" example:"incident"` // 事件类型
	Priority            string                 `json:"priority" binding:"omitempty,oneof=low medium high critical" example:"high"`
	Severity            string                 `json:"severity" binding:"omitempty,oneof=low medium high critical" example:"high"`
	Category            string                 `json:"category" example:"performance"`
	Subcategory         string                 `json:"subcategory" example:"cpu"`
	ConfigurationItemID *int                   `json:"configurationItemId" example:"1"`
	AssigneeID          *int                   `json:"assigneeId" example:"1"`
	ImpactAnalysis      *ImpactAnalysis        `json:"impactAnalysis"`
	Source              string                 `json:"source" binding:"omitempty,oneof=manual monitoring system user" example:"monitoring"`
	Metadata            map[string]interface{} `json:"metadata"`
	DetectedAt          *time.Time             `json:"detectedAt" example:"2024-01-01T00:00:00Z"`
}

type UpdateIncidentRequest struct {
	Title           *string                `json:"title,omitempty"`
	Description     *string                `json:"description,omitempty"`
	Status          *string                `json:"status,omitempty" binding:"omitempty,oneof=new assigned in_progress on_hold resolved closed cancelled"`
	Priority        *string                `json:"priority,omitempty" binding:"omitempty,oneof=low medium high critical"`
	Severity        *string                `json:"severity,omitempty" binding:"omitempty,oneof=low medium high critical"`
	Category        *string                `json:"category,omitempty"`
	Subcategory     *string                `json:"subcategory,omitempty"`
	AssigneeID      *int                   `json:"assigneeId,omitempty"`
	ImpactAnalysis  *ImpactAnalysis        `json:"impactAnalysis,omitempty"`
	RootCause       *RootCause             `json:"rootCause,omitempty"`
	ResolutionSteps []ResolutionStep       `json:"resolutionSteps,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type IncidentResponse struct {
	ID                  int                    `json:"id" example:"1"`
	Title               string                 `json:"title" example:"服务器CPU使用率过高"`
	Description         string                 `json:"description" example:"生产环境Web服务器CPU使用率持续超过90%"`
	Status              string                 `json:"status" example:"new"`
	Priority            string                 `json:"priority" example:"high"`
	Severity            string                 `json:"severity" example:"high"`
	IncidentNumber      string                 `json:"incidentNumber" example:"INC-000001"`
	ReporterID          int                    `json:"reporterId" example:"1"`
	AssigneeID          *int                   `json:"assigneeId" example:"2"`
	ConfigurationItemID *int                   `json:"configurationItemId" example:"1"`
	Category            string                 `json:"category" example:"performance"`
	Subcategory         string                 `json:"subcategory" example:"cpu"`
	ImpactAnalysis      *ImpactAnalysis        `json:"impactAnalysis"`
	RootCause           *RootCause             `json:"rootCause"`
	ResolutionSteps     []ResolutionStep       `json:"resolutionSteps"`
	DetectedAt          time.Time              `json:"detectedAt" example:"2024-01-01T00:00:00Z"`
	ResolvedAt          *time.Time             `json:"resolvedAt,omitempty" example:"2024-01-01T12:00:00Z"`
	ClosedAt            *time.Time             `json:"closedAt,omitempty" example:"2024-01-01T18:00:00Z"`
	EscalatedAt         *time.Time             `json:"escalatedAt,omitempty" example:"2024-01-01T06:00:00Z"`
	EscalationLevel     int                    `json:"escalationLevel" example:"1"`
	IsAutomated         bool                   `json:"isAutomated" example:"false"`
	IsMajorIncident     bool                   `json:"isMajorIncident" example:"false"`
	Source              string                 `json:"source" example:"monitoring"`
	Metadata            map[string]interface{} `json:"metadata"`
	TenantID            int                    `json:"tenantId" example:"1"`
	CreatedAt           time.Time              `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt           time.Time              `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// 事件活动记录DTO
type CreateIncidentEventRequest struct {
	IncidentID  int                    `json:"incidentId" binding:"required" example:"1"`
	EventType   string                 `json:"eventType" binding:"required" example:"status_change"`
	EventName   string                 `json:"eventName" binding:"required" example:"状态变更"`
	Description string                 `json:"description" example:"事件状态从new变更为in_progress"`
	Status      string                 `json:"status" example:"active"`
	Severity    string                 `json:"severity" example:"medium"`
	Data        map[string]interface{} `json:"data"`
	OccurredAt  *time.Time             `json:"occurredAt" example:"2024-01-01T00:00:00Z"`
	UserID      *int                   `json:"userId" example:"1"`
	Source      string                 `json:"source" example:"system"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type IncidentEventResponse struct {
	ID          int                    `json:"id" example:"1"`
	IncidentID  int                    `json:"incidentId" example:"1"`
	EventType   string                 `json:"eventType" example:"status_change"`
	EventName   string                 `json:"eventName" example:"状态变更"`
	Description string                 `json:"description" example:"事件状态从new变更为in_progress"`
	Status      string                 `json:"status" example:"active"`
	Severity    string                 `json:"severity" example:"medium"`
	Data        map[string]interface{} `json:"data"`
	OccurredAt  time.Time              `json:"occurredAt" example:"2024-01-01T00:00:00Z"`
	UserID      *int                   `json:"userId" example:"1"`
	Source      string                 `json:"source" example:"system"`
	Metadata    map[string]interface{} `json:"metadata"`
	TenantID    int                    `json:"tenantId" example:"1"`
	CreatedAt   time.Time              `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt   time.Time              `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// 事件告警DTO
type CreateIncidentAlertRequest struct {
	IncidentID  int                    `json:"incidentId" binding:"required" example:"1"`
	AlertType   string                 `json:"alertType" binding:"required" example:"escalation"`
	AlertName   string                 `json:"alertName" binding:"required" example:"事件升级告警"`
	Message     string                 `json:"message" binding:"required" example:"事件已升级到下一级别"`
	Severity    string                 `json:"severity" example:"high"`
	Channels    []string               `json:"channels" example:"[\"email\",\"sms\"]"`
	Recipients  []string               `json:"recipients" example:"[\"manager@company.com\"]"`
	TriggeredAt *time.Time             `json:"triggeredAt" example:"2024-01-01T00:00:00Z"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type IncidentAlertResponse struct {
	ID             int                    `json:"id" example:"1"`
	IncidentID     int                    `json:"incidentId" example:"1"`
	AlertType      string                 `json:"alertType" example:"escalation"`
	AlertName      string                 `json:"alertName" example:"事件升级告警"`
	Message        string                 `json:"message" example:"事件已升级到下一级别"`
	Severity       string                 `json:"severity" example:"high"`
	Status         string                 `json:"status" example:"active"`
	Channels       []string               `json:"channels" example:"[\"email\",\"sms\"]"`
	Recipients     []string               `json:"recipients" example:"[\"manager@company.com\"]"`
	TriggeredAt    time.Time              `json:"triggeredAt" example:"2024-01-01T00:00:00Z"`
	AcknowledgedAt *time.Time             `json:"acknowledgedAt,omitempty" example:"2024-01-01T01:00:00Z"`
	ResolvedAt     *time.Time             `json:"resolvedAt,omitempty" example:"2024-01-01T02:00:00Z"`
	AcknowledgedBy *int                   `json:"acknowledgedBy" example:"1"`
	Metadata       map[string]interface{} `json:"metadata"`
	TenantID       int                    `json:"tenantId" example:"1"`
	CreatedAt      time.Time              `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt      time.Time              `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// 事件指标DTO
type CreateIncidentMetricRequest struct {
	IncidentID  int                    `json:"incidentId" binding:"required" example:"1"`
	MetricType  string                 `json:"metricType" binding:"required" example:"response_time"`
	MetricName  string                 `json:"metricName" binding:"required" example:"平均响应时间"`
	MetricValue float64                `json:"metricValue" binding:"required" example:"2.5"`
	Unit        string                 `json:"unit" example:"秒"`
	MeasuredAt  *time.Time             `json:"measuredAt" example:"2024-01-01T00:00:00Z"`
	Tags        map[string]string      `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type IncidentMetricResponse struct {
	ID          int                    `json:"id" example:"1"`
	IncidentID  int                    `json:"incidentId" example:"1"`
	MetricType  string                 `json:"metricType" example:"response_time"`
	MetricName  string                 `json:"metricName" example:"平均响应时间"`
	MetricValue float64                `json:"metricValue" example:"2.5"`
	Unit        string                 `json:"unit" example:"秒"`
	MeasuredAt  time.Time              `json:"measuredAt" example:"2024-01-01T00:00:00Z"`
	Tags        map[string]string      `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	TenantID    int                    `json:"tenantId" example:"1"`
	CreatedAt   time.Time              `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt   time.Time              `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// 事件规则DTO
type CreateIncidentRuleRequest struct {
	Name        string                   `json:"name" binding:"required" example:"高优先级事件自动升级"`
	Description string                   `json:"description" example:"当事件优先级为high或urgent时，自动升级到下一级别"`
	RuleType    string                   `json:"ruleType" binding:"required" example:"escalation"`
	Conditions  map[string]interface{}   `json:"conditions"`
	Actions     []map[string]interface{} `json:"actions"`
	Priority    string                   `json:"priority" example:"high"`
	IsActive    bool                     `json:"isActive" example:"true"`
	Metadata    map[string]interface{}   `json:"metadata"`
}

type UpdateIncidentRuleRequest struct {
	Name        *string                  `json:"name,omitempty"`
	Description *string                  `json:"description,omitempty"`
	RuleType    *string                  `json:"ruleType,omitempty"`
	Conditions  map[string]interface{}   `json:"conditions,omitempty"`
	Actions     []map[string]interface{} `json:"actions,omitempty"`
	Priority    *string                  `json:"priority,omitempty"`
	IsActive    *bool                    `json:"isActive,omitempty"`
	Metadata    map[string]interface{}   `json:"metadata,omitempty"`
}

type IncidentRuleResponse struct {
	ID             int                      `json:"id" example:"1"`
	Name           string                   `json:"name" example:"高优先级事件自动升级"`
	Description    string                   `json:"description" example:"当事件优先级为high或urgent时，自动升级到下一级别"`
	RuleType       string                   `json:"ruleType" example:"escalation"`
	Conditions     map[string]interface{}   `json:"conditions"`
	Actions        []map[string]interface{} `json:"actions"`
	Priority       string                   `json:"priority" example:"high"`
	IsActive       bool                     `json:"isActive" example:"true"`
	ExecutionCount int                      `json:"executionCount" example:"5"`
	LastExecutedAt *time.Time               `json:"lastExecutedAt,omitempty" example:"2024-01-01T00:00:00Z"`
	Metadata       map[string]interface{}   `json:"metadata"`
	TenantID       int                      `json:"tenantId" example:"1"`
	CreatedAt      time.Time                `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt      time.Time                `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// 事件监控DTO
type IncidentMonitoringRequest struct {
	IncidentID *int    `json:"incidentId,omitempty" example:"1"`
	Category   *string `json:"category,omitempty" example:"performance"`
	Priority   *string `json:"priority,omitempty" example:"high"`
	Status     *string `json:"status,omitempty" example:"new"`
	StartTime  string  `json:"startTime" example:"2024-01-01T00:00:00Z"`
	EndTime    string  `json:"endTime" example:"2024-01-31T23:59:59Z"`
}

type IncidentMonitoringResponse struct {
	TotalIncidents        int                      `json:"totalIncidents" example:"100"`
	OpenIncidents         int                      `json:"openIncidents" example:"25"`
	ResolvedIncidents     int                      `json:"resolvedIncidents" example:"70"`
	ClosedIncidents       int                      `json:"closedIncidents" example:"5"`
	CriticalIncidents     int                      `json:"criticalIncidents" example:"10"`
	HighPriorityIncidents int                      `json:"highPriorityIncidents" example:"20"`
	AverageResolutionTime float64                  `json:"averageResolutionTime" example:"4.5"`
	ResolutionRate        float64                  `json:"resolutionRate" example:"95.0"`
	EscalationRate        float64                  `json:"escalationRate" example:"15.0"`
	Incidents             []IncidentResponse       `json:"incidents"`
	Metrics               []IncidentMetricResponse `json:"metrics"`
	Alerts                []IncidentAlertResponse  `json:"alerts"`
}

// 事件升级DTO
type IncidentEscalationRequest struct {
	IncidentID      int    `json:"incidentId" binding:"required" example:"1"`
	EscalationLevel int    `json:"escalationLevel" binding:"required" example:"1"`
	Reason          string `json:"reason" example:"事件处理时间过长"`
	NotifyUsers     []int  `json:"notifyUsers"`
	AutoAssign      bool   `json:"autoAssign" example:"true"`
}

type IncidentEscalationResponse struct {
	ID              int       `json:"id" example:"1"`
	IncidentID      int       `json:"incidentId" example:"1"`
	EscalationLevel int       `json:"escalationLevel" example:"1"`
	Reason          string    `json:"reason" example:"事件处理时间过长"`
	Status          string    `json:"status" example:"active"`
	NotifiedUsers   []int     `json:"notifiedUsers"`
	AutoAssigned    bool      `json:"autoAssigned" example:"true"`
	CreatedAt       time.Time `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt       time.Time `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// 事件关联分析DTO
type IncidentCorrelationRequest struct {
	IncidentID          int     `json:"incidentId" binding:"required" example:"1"`
	SimilarityThreshold float64 `json:"similarityThreshold" example:"0.8"`
	TimeWindowHours     int     `json:"timeWindowHours" example:"24"`
	MaxResults          int     `json:"maxResults" example:"10"`
}

type IncidentCorrelationResponse struct {
	IncidentID          int                    `json:"incidentId" example:"1"`
	CorrelatedIncidents []CorrelatedIncident   `json:"correlatedIncidents"`
	CorrelationScore    float64                `json:"correlationScore" example:"0.85"`
	AnalysisResult      map[string]interface{} `json:"analysisResult"`
}

type CorrelatedIncident struct {
	IncidentID        int       `json:"incidentId" example:"2"`
	IncidentNumber    string    `json:"incidentNumber" example:"INC-000002"`
	Title             string    `json:"title" example:"数据库连接超时"`
	SimilarityScore   float64   `json:"similarityScore" example:"0.85"`
	CorrelationReason string    `json:"correlationReason" example:"相似的错误模式和影响范围"`
	CreatedAt         time.Time `json:"createdAt" example:"2024-01-01T00:00:00Z"`
}

// IncidentStatsResponse 事件统计响应
type IncidentStatsResponse struct {
	TotalIncidents    int     `json:"totalIncidents"`
	OpenIncidents     int     `json:"openIncidents"`
	CriticalIncidents int     `json:"criticalIncidents"`
	MajorIncidents    int     `json:"majorIncidents"`
	AvgResolutionTime float64 `json:"avgResolutionTime"`
	MTTA              float64 `json:"mtta"` // Mean Time To Acknowledge
	MTTR              float64 `json:"mttr"` // Mean Time To Resolve
}

// ConvertIncidentToProblemRequest 将事件转换为问题的请求
type ConvertIncidentToProblemRequest struct {
	Title       string `json:"title" binding:"omitempty"`        // 可选自定义标题
	Description string `json:"description" binding:"omitempty"` // 可选自定义描述
	RootCause   string `json:"rootCause" binding:"omitempty"`   // 根因分析
}
