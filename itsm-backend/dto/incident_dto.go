package dto

import (
	"time"
)

// CreateIncidentRequest 创建事件请求
type CreateIncidentRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Priority    string `json:"priority" binding:"required"`
	Source      string `json:"source" binding:"required"`
	Type        string `json:"type" binding:"required"`
	IsMajor     bool   `json:"is_major_incident"`
	AssigneeID  *int   `json:"assignee_id"`
	// 阿里云相关字段
	AlibabaCloudInstanceID string                 `json:"alibaba_cloud_instance_id"`
	AlibabaCloudRegion     string                 `json:"alibaba_cloud_region"`
	AlibabaCloudService    string                 `json:"alibaba_cloud_service"`
	AlibabaCloudAlertData  map[string]interface{} `json:"alibaba_cloud_alert_data"`
	AlibabaCloudMetrics    map[string]interface{} `json:"alibaba_cloud_metrics"`
	// 安全事件相关字段
	SecurityEventType     string                 `json:"security_event_type"`
	SecurityEventSourceIP string                 `json:"security_event_source_ip"`
	SecurityEventTarget   string                 `json:"security_event_target"`
	SecurityEventDetails  map[string]interface{} `json:"security_event_details"`
	// 关联的配置项
	AffectedConfigurationItemIDs []int `json:"affected_configuration_item_ids"`
}

// UpdateIncidentRequest 更新事件请求
type UpdateIncidentRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Priority    *string `json:"priority"`
	Type        *string `json:"type"`
	AssigneeID  *int    `json:"assignee_id"`
	IsMajor     *bool   `json:"is_major_incident"`
}

// UpdateIncidentStatusRequest 更新事件状态请求
type UpdateIncidentStatusRequest struct {
	Status         string `json:"status" binding:"required"`
	ResolutionNote string `json:"resolution_note"`
	SuspendReason  string `json:"suspend_reason"`
}

// IncidentResponse 事件响应
type IncidentResponse struct {
	ID              int           `json:"id"`
	Title           string        `json:"title"`
	Description     string        `json:"description"`
	Status          string        `json:"status"`
	Priority        string        `json:"priority"`
	Source          string        `json:"source"`
	Type            string        `json:"type"`
	IncidentNumber  string        `json:"incident_number"`
	IsMajorIncident bool          `json:"is_major_incident"`
	Reporter        *UserResponse `json:"reporter"`
	Assignee        *UserResponse `json:"assignee"`
	// 阿里云相关字段
	AlibabaCloudInstanceID string                 `json:"alibaba_cloud_instance_id"`
	AlibabaCloudRegion     string                 `json:"alibaba_cloud_region"`
	AlibabaCloudService    string                 `json:"alibaba_cloud_service"`
	AlibabaCloudAlertData  map[string]interface{} `json:"alibaba_cloud_alert_data"`
	AlibabaCloudMetrics    map[string]interface{} `json:"alibaba_cloud_metrics"`
	// 安全事件相关字段
	SecurityEventType     string                 `json:"security_event_type"`
	SecurityEventSourceIP string                 `json:"security_event_source_ip"`
	SecurityEventTarget   string                 `json:"security_event_target"`
	SecurityEventDetails  map[string]interface{} `json:"security_event_details"`
	// 时间字段
	DetectedAt  *time.Time `json:"detected_at"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	// 关联数据
	AffectedConfigurationItems []ConfigurationItemResponse `json:"affected_configuration_items"`
	StatusLogs                 []interface{}               `json:"status_logs"`
	Comments                   []interface{}               `json:"comments"`
}

// ListIncidentsRequest 获取事件列表请求
type ListIncidentsRequest struct {
	Page       int    `form:"page" binding:"min=1"`
	PageSize   int    `form:"page_size" binding:"min=1,max=100"`
	Status     string `form:"status"`
	Priority   string `form:"priority"`
	Source     string `form:"source"`
	Type       string `form:"type"`
	AssigneeID int    `form:"assignee_id"`
	IsMajor    *bool  `form:"is_major_incident"`
	Keyword    string `form:"keyword"`
}

// ListIncidentsResponse 事件列表响应
type ListIncidentsResponse struct {
	Incidents []IncidentResponse `json:"incidents"`
	Total     int                `json:"total"`
	Page      int                `json:"page"`
	PageSize  int                `json:"page_size"`
}

// AlibabaCloudAlertRequest 阿里云告警请求
type AlibabaCloudAlertRequest struct {
	AlertID          string                 `json:"alert_id"`
	AlertName        string                 `json:"alert_name"`
	AlertDescription string                 `json:"alert_description"`
	AlertLevel       string                 `json:"alert_level"`
	InstanceID       string                 `json:"instance_id"`
	Region           string                 `json:"region"`
	Service          string                 `json:"service"`
	Metrics          map[string]interface{} `json:"metrics"`
	AlertData        map[string]interface{} `json:"alert_data"`
	DetectedAt       time.Time              `json:"detected_at"`
}

// SecurityEventRequest 安全事件请求
type SecurityEventRequest struct {
	EventID          string                 `json:"event_id"`
	EventType        string                 `json:"event_type"`
	EventName        string                 `json:"event_name"`
	EventDescription string                 `json:"event_description"`
	SourceIP         string                 `json:"source_ip"`
	Target           string                 `json:"target"`
	Severity         string                 `json:"severity"`
	EventDetails     map[string]interface{} `json:"event_details"`
	DetectedAt       time.Time              `json:"detected_at"`
}

// CloudProductEventRequest 云产品事件请求
type CloudProductEventRequest struct {
	EventID          string                 `json:"event_id"`
	EventType        string                 `json:"event_type"`
	EventName        string                 `json:"event_name"`
	EventDescription string                 `json:"event_description"`
	Product          string                 `json:"product"`
	InstanceID       string                 `json:"instance_id"`
	Region           string                 `json:"region"`
	EventData        map[string]interface{} `json:"event_data"`
	DetectedAt       time.Time              `json:"detected_at"`
}

// IncidentManagementMetrics 事件管理指标
type IncidentManagementMetrics struct {
	TotalIncidents    int     `json:"total_incidents"`
	OpenIncidents     int     `json:"open_incidents"`
	CriticalIncidents int     `json:"critical_incidents"`
	MajorIncidents    int     `json:"major_incidents"`
	AvgResolutionTime float64 `json:"avg_resolution_time"` // 小时
	MTTA              float64 `json:"mtta"`                // 平均确认时间（小时）
	MTTR              float64 `json:"mttr"`                // 平均恢复时间（小时）
}

// IncidentSourceStats 事件来源统计
type IncidentSourceStats struct {
	Source        string `json:"source"`
	Count         int    `json:"count"`
	CriticalCount int    `json:"critical_count"`
	MajorCount    int    `json:"major_count"`
}

// IncidentTypeStats 事件类型统计
type IncidentTypeStats struct {
	Type          string `json:"type"`
	Count         int    `json:"count"`
	CriticalCount int    `json:"critical_count"`
	MajorCount    int    `json:"major_count"`
}

// IncidentTrend 事件趋势
type IncidentTrend struct {
	Date          string `json:"date"`
	TotalCount    int    `json:"total_count"`
	ResolvedCount int    `json:"resolved_count"`
	CriticalCount int    `json:"critical_count"`
	MajorCount    int    `json:"major_count"`
}
