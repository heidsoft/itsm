package dto

import (
	"itsm-backend/ent"
	"time"
)

// CreateTicketRequest 创建工单请求
type CreateTicketRequest struct {
	Title          string                 `json:"title" binding:"required,min=2,max=200"`
	Description    string                 `json:"description" binding:"required,min=10,max=5000"`
	Priority       string                 `json:"priority" binding:"required,oneof=low medium high critical"`
	Category       string                 `json:"category" binding:"required"`
	RequesterID    int                    `json:"requester_id" binding:"required"`
	AssigneeID     int                    `json:"assignee_id"`
	ParentTicketID *int                   `json:"parent_ticket_id,omitempty"`
	Tags           []string              `json:"tags"`
	FormFields     map[string]interface{} `json:"form_fields"`
	Attachments    []string               `json:"attachments"`
}

// UpdateTicketRequest 更新工单请求
type UpdateTicketRequest struct {
	Title       string                 `json:"title" binding:"omitempty,min=2,max=200"`
	Description string                 `json:"description" binding:"omitempty,min=10,max=5000"`
	Priority    string                 `json:"priority" binding:"omitempty,oneof=low medium high critical"`
	Status      string                 `json:"status" binding:"omitempty,oneof=open in_progress resolved closed"`
	Category    string                 `json:"category" binding:"omitempty"`
	AssigneeID  int                    `json:"assignee_id"`
	Tags        []string               `json:"tags"`
	FormFields  map[string]interface{} `json:"form_fields"`
	UserID      int                    `json:"user_id" binding:"required"` // 操作用户ID
}

// ListTicketsRequest 获取工单列表请求
type ListTicketsRequest struct {
	Page          int        `json:"page" form:"page"`
	PageSize      int        `json:"page_size" form:"page_size"`
	Status        string     `json:"status" form:"status"`
	Priority      string     `json:"priority" form:"priority"`
	Category      string     `json:"category" form:"category"`
	AssigneeID    int        `json:"assignee_id" form:"assignee_id"`
	RequesterID   int        `json:"requester_id" form:"requester_id"`
	ParentTicketID *int      `json:"parent_ticket_id" form:"parent_ticket_id"`
	Keyword       string     `json:"keyword" form:"keyword"`
	DateFrom      *time.Time `json:"date_from" form:"date_from"`
	DateTo        *time.Time `json:"date_to" form:"date_to"`
	IsOverdue     bool       `json:"is_overdue" form:"is_overdue"`
	SortBy        string     `json:"sort_by" form:"sort_by"`
	SortOrder     string     `json:"sort_order" form:"sort_order"`
}

// TicketResponse 工单响应
type TicketResponse struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	Priority     string    `json:"priority"`
	TicketNumber string    `json:"ticket_number"`
	RequesterID  int       `json:"requester_id"`
	AssigneeID   int       `json:"assignee_id,omitempty"`
	TenantID     int       `json:"tenant_id"`
	CategoryID   int       `json:"category_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ListTicketsResponse 工单列表响应
type ListTicketsResponse struct {
	Tickets  []*ent.Ticket `json:"tickets"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// TicketStatsResponse 工单统计响应
type TicketStatsResponse struct {
	Total        int `json:"total"`
	Open         int `json:"open"`
	InProgress   int `json:"in_progress"`
	Resolved     int `json:"resolved"`
	HighPriority int `json:"high_priority"`
	Overdue      int `json:"overdue"`
}

// TicketDetailResponse 工单详情响应
type TicketDetailResponse struct {
	Ticket     *ent.Ticket       `json:"ticket"`
	SLAMetrics *TicketSLAMetrics `json:"sla_metrics"`
}

// TicketSLAMetrics 工单SLA指标
type TicketSLAMetrics struct {
	ID              int       `json:"id"`
	SLADefinitionID int       `json:"sla_definition_id"`
	TicketID        int       `json:"ticket_id"`
	MetricType      string    `json:"metric_type"`
	TargetValue     float64   `json:"target_value"`
	ActualValue     *float64  `json:"actual_value,omitempty"`
	IsMet           bool      `json:"is_met"`
	MeasurementTime time.Time `json:"measurement_time"`
	TenantID        int       `json:"tenant_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	TicketIDs []int `json:"ticket_ids" binding:"required"`
}

// TicketExportRequest 工单导出请求
type TicketExportRequest struct {
	Format  string                 `json:"format" binding:"required,oneof=csv excel pdf"`
	Filters ListTicketsRequest     `json:"filters"`
	Fields  []string               `json:"fields"`
	Options map[string]interface{} `json:"options"`
}

// TicketImportRequest 工单导入请求
type TicketImportRequest struct {
	File     string                 `json:"file" binding:"required"`
	Format   string                 `json:"format" binding:"required,oneof=csv excel"`
	Options  map[string]interface{} `json:"options"`
	Validate bool                   `json:"validate"`
}

// TicketTemplate 工单模板
type TicketTemplate struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Priority    string                 `json:"priority"`
	FormFields  map[string]interface{} `json:"form_fields"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// TicketWorkflowRequest 工单工作流请求
type TicketWorkflowRequest struct {
	TicketID       int    `json:"ticket_id" binding:"required"`
	Action         string `json:"action" binding:"required"`
	Comment        string `json:"comment"`
	UserID         int    `json:"user_id" binding:"required"`
	NextAssigneeID int    `json:"next_assignee_id"`
}

// TicketAssignmentRequest 工单分配请求
type TicketAssignmentRequest struct {
	TicketIDs      []int  `json:"ticket_ids" binding:"required"`
	AssigneeID     int    `json:"assignee_id" binding:"required"`
	Reason         string `json:"reason"`
	NotifyAssignee bool   `json:"notify_assignee"`
}

// TicketEscalationRequest 工单升级请求
type TicketEscalationRequest struct {
	TicketID      int    `json:"ticket_id" binding:"required"`
	Reason        string `json:"reason" binding:"required"`
	NewPriority   string `json:"new_priority"`
	NewAssigneeID int    `json:"new_assignee_id"`
	UserID        int    `json:"user_id" binding:"required"`
}

// TicketSLARequest SLA设置请求
type TicketSLARequest struct {
	Category       string `json:"category" binding:"required"`
	Priority       string `json:"priority" binding:"required"`
	ResponseTime   int    `json:"response_time" binding:"required"`   // 响应时间（小时）
	ResolutionTime int    `json:"resolution_time" binding:"required"` // 解决时间（小时）
	BusinessHours  bool   `json:"business_hours"`                     // 是否仅计算工作时间
}

// TicketNotificationRequest 工单通知请求
type TicketNotificationRequest struct {
	TicketID   int                    `json:"ticket_id" binding:"required"`
	Type       string                 `json:"type" binding:"required,oneof=email sms webhook"`
	Recipients []string               `json:"recipients"`
	Template   string                 `json:"template"`
	Data       map[string]interface{} `json:"data"`
}

// TicketAnalyticsRequest 工单分析请求
type TicketAnalyticsRequest struct {
	DateFrom time.Time              `json:"date_from" binding:"required"`
	DateTo   time.Time              `json:"date_to" binding:"required"`
	GroupBy  string                 `json:"group_by" binding:"required,oneof=day week month category priority assignee"`
	Metrics  []string               `json:"metrics" binding:"required"`
	Filters  map[string]interface{} `json:"filters"`
}

// TicketAnalyticsResponse 工单分析响应
type TicketAnalyticsResponse struct {
	Data        []map[string]interface{} `json:"data"`
	Summary     map[string]interface{}   `json:"summary"`
	Trends      []map[string]interface{} `json:"trends"`
	GeneratedAt time.Time                `json:"generated_at"`
}
