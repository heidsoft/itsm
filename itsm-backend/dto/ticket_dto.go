package dto

import (
	"time"
)

// UserBasicInfo 用户基本信息
type UserBasicInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// CreateTicketRequest 创建工单请求
type CreateTicketRequest struct {
	Title                 string                 `json:"title" binding:"required,min=2,max=200"`
	Description           string                 `json:"description" binding:"required,min=0,max=5000"`
	Priority              string                 `json:"priority" binding:"required,oneof=low medium high critical urgent"`
	Type                  string                 `json:"type" binding:"omitempty,oneof=incident service_request change ticket problem"` // 工单类型
	TypeID                string                 `json:"typeId,omitempty"`                                                              
	Category              string                 `json:"category"`                                                                      // 分类名称（可选，前端传入）
	CategoryID            *int                   `json:"categoryId,omitempty"`                                                          // 分类ID（优先使用）
	TemplateID            *int                   `json:"templateId,omitempty"`                                                          // 模板ID
	RequesterID           int                    `json:"requesterId" binding:"omitempty"`                                               // 从认证上下文中获取，前端可不传
	AssigneeID            int                    `json:"assigneeId"`
	ParentTicketID        *int                   `json:"parentTicketId,omitempty"`
	TagIDs                []int                  `json:"tagIds,omitempty"` // 标签ID列表
	Tags                  []string               `json:"tags"`             
	FormFields            map[string]interface{} `json:"formFields"`
	Attachments           []string               `json:"attachments"`
	WorkflowDefinitionKey string                 `json:"workflowDefinitionKey"` // 工作流定义Key（可选，优先级高于自动选择）
}

// UpdateTicketRequest 更新工单请求
type UpdateTicketRequest struct {
	Title       string                 `json:"title" binding:"omitempty,min=2,max=200"`
	Description string                 `json:"description" binding:"omitempty,min=10,max=5000"`
	Priority    string                 `json:"priority" binding:"omitempty,oneof=low medium high critical"`
	Status      string                 `json:"status" binding:"omitempty,oneof=new open assigned in_progress pending resolved closed cancelled approved rejected"`
	Type        string                 `json:"type" binding:"omitempty,oneof=incident service_request change ticket problem"`
	Category    string                 `json:"category" binding:"omitempty"`
	AssigneeID  int                    `json:"assigneeId"`
	RequesterID int                    `json:"requesterId"` // 创建人ID
	Tags        []string               `json:"tags"`
	Resolution  string                 `json:"resolution" binding:"omitempty"`
	FormFields  map[string]interface{} `json:"formFields"`
	UserID      int                    `json:"userId" binding:"omitempty"` // 操作用户ID (后端自动填充)
	Version     int                    `json:"version"`                    // 版本号（乐观锁）
	Force       bool                   `json:"force"`                      // 是否强制更新（忽略版本检查）
}

// ListTicketsRequest 获取工单列表请求
type ListTicketsRequest struct {
	Page           int        `json:"page" form:"page"`
	PageSize       int        `json:"pageSize" form:"pageSize"`
	Status         string     `json:"status" form:"status"`
	Priority       string     `json:"priority" form:"priority"`
	Type           string     `json:"type" form:"type"`               // 工单类型筛选
	Category       string     `json:"category" form:"category"`       // 分类名称
	CategoryID     *int       `json:"categoryId" form:"categoryId"`   // 分类ID（优先）
	AssigneeID     *int       `json:"assigneeId" form:"assigneeId"`   // 处理人ID（可选）
	RequesterID    *int       `json:"requesterId" form:"requesterId"` // 创建人ID（可选）
	ParentTicketID *int       `json:"parentTicketId" form:"parentTicketId"`
	Keyword        string     `json:"keyword" form:"keyword"`
	DateFrom       *time.Time `json:"dateFrom" form:"dateFrom"` // 创建时间起始
	DateTo         *time.Time `json:"dateTo" form:"dateTo"`     // 创建时间截止
	IsOverdue      bool       `json:"isOverdue" form:"isOverdue"`
	SortBy         string     `json:"sortBy" form:"sortBy"`
	SortOrder      string     `json:"sortOrder" form:"sortOrder"`
}

// TicketResponse 工单响应
type TicketResponse struct {
	ID                    int            `json:"id"`
	Title                 string         `json:"title"`
	Description           string         `json:"description"`
	Status                string         `json:"status"`
	Priority              string         `json:"priority"`
	Type                  string         `json:"type"`
	TicketNumber          string         `json:"ticketNumber"`
	RequesterID           int            `json:"requesterId"`
	AssigneeID            int            `json:"assigneeId,omitempty"`
	TenantID              int            `json:"tenantId"`
	CategoryID            int            `json:"categoryId,omitempty"`
	DepartmentID          int            `json:"departmentId,omitempty"`
	ParentTicketID        int            `json:"parentTicketId,omitempty"`
	Version               int            `json:"version"`
	CreatedAt             time.Time      `json:"createdAt"`
	UpdatedAt             time.Time      `json:"updatedAt"`
	Requester             *UserBasicInfo `json:"requester,omitempty"`
	Assignee              *UserBasicInfo `json:"assignee,omitempty"`
	Resolution            string         `json:"resolution,omitempty"`
	ResolutionCategory    string         `json:"resolutionCategory,omitempty"`
	ResolvedAt            *time.Time     `json:"resolvedAt,omitempty"`
	ClosedAt              *time.Time     `json:"closedAt,omitempty"`
	FirstResponseAt       *time.Time     `json:"firstResponseAt,omitempty"`
	SLAResponseDeadline   *time.Time     `json:"slaResponseDeadline,omitempty"`
	SLAResolutionDeadline *time.Time     `json:"slaResolutionDeadline,omitempty"`
	Rating                int            `json:"rating,omitempty"`
}

// ListTicketsResponse 工单列表响应
type ListTicketsResponse struct {
	Tickets    []*TicketResponse `json:"tickets"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"pageSize"`
	TotalPages int               `json:"totalPages"`
}

// TicketStatsResponse 工单统计响应
type TicketStatsResponse struct {
	Total        int `json:"total"`
	Open         int `json:"open"`
	InProgress   int `json:"inProgress"`
	Resolved     int `json:"resolved"`
	Pending      int `json:"pending"`
	HighPriority int `json:"highPriority"`
	Overdue      int `json:"overdue"`
}

// TicketDetailResponse 工单详情响应
type TicketDetailResponse struct {
	Ticket     *TicketResponse   `json:"ticket"`
	SLAMetrics *TicketSLAMetrics `json:"slaMetrics"`
}

// TicketSLAMetrics 工单SLA指标
type TicketSLAMetrics struct {
	ID              int       `json:"id"`
	SLADefinitionID int       `json:"slaDefinitionId"`
	TicketID        int       `json:"ticketId"`
	MetricType      string    `json:"metricType"`
	TargetValue     float64   `json:"targetValue"`
	ActualValue     *float64  `json:"actualValue,omitempty"`
	IsMet           bool      `json:"isMet"`
	MeasurementTime time.Time `json:"measurementTime"`
	TenantID        int       `json:"tenantId"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	TicketIDs []int `json:"ticketIds" binding:"required"`
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
	ID            int                      `json:"id"`
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	Category      string                   `json:"category"`
	Priority      string                   `json:"priority"`
	Fields        []map[string]interface{} `json:"fields,omitempty"`
	FormFields    map[string]interface{}   `json:"formFields"`
	WorkflowSteps []map[string]interface{} `json:"workflowSteps,omitempty"`
	IsActive      bool                     `json:"isActive"`
	IsActiveAlt   *bool                    `json:"-"` // internal flag: force-update isActive (not JSON-bound)
	CreatedAt     time.Time                `json:"createdAt"`
	UpdatedAt     time.Time                `json:"updatedAt"`
}

// TicketWorkflowRequest 工单工作流请求
type TicketWorkflowRequest struct {
	TicketID       int    `json:"ticketId" binding:"required"`
	Action         string `json:"action" binding:"required"`
	Comment        string `json:"comment"`
	UserID         int    `json:"userId" binding:"required"`
	NextAssigneeID int    `json:"nextAssigneeId"`
}

// TicketAssignmentRequest 工单分配请求
type TicketAssignmentRequest struct {
	TicketIDs      []int  `json:"ticketIds" binding:"required"`
	AssigneeID     int    `json:"assigneeId" binding:"required"`
	Reason         string `json:"reason"`
	NotifyAssignee bool   `json:"notifyAssignee"`
}

// TicketEscalationRequest 工单升级请求
type TicketEscalationRequest struct {
	TicketID      int    `json:"ticketId" binding:"required"`
	Reason        string `json:"reason" binding:"required"`
	NewPriority   string `json:"newPriority"`
	NewAssigneeID int    `json:"newAssigneeId"`
	UserID        int    `json:"userId" binding:"required"`
}

// TicketSLARequest SLA设置请求
type TicketSLARequest struct {
	Category       string `json:"category" binding:"required"`
	Priority       string `json:"priority" binding:"required"`
	ResponseTime   int    `json:"responseTime" binding:"required"`   // 响应时间（小时）
	ResolutionTime int    `json:"resolutionTime" binding:"required"` // 解决时间（小时）
	BusinessHours  bool   `json:"businessHours"`                     // 是否仅计算工作时间
}

// TicketNotificationRequest 工单通知请求
type TicketNotificationRequest struct {
	TicketID   int                    `json:"ticketId" binding:"required"`
	Type       string                 `json:"type" binding:"required,oneof=email sms webhook"`
	Recipients []string               `json:"recipients"`
	Template   string                 `json:"template"`
	Data       map[string]interface{} `json:"data"`
}

// TicketAnalyticsRequest 工单分析请求
type TicketAnalyticsRequest struct {
	DateFrom time.Time              `json:"dateFrom" binding:"required"`
	DateTo   time.Time              `json:"dateTo" binding:"required"`
	GroupBy  string                 `json:"groupBy" binding:"required,oneof=day week month category priority assignee"`
	Metrics  []string               `json:"metrics" binding:"required"`
	Filters  map[string]interface{} `json:"filters"`
}

// TicketAnalyticsResponse 工单分析响应
type TicketAnalyticsResponse struct {
	Data        []map[string]interface{} `json:"data"`
	Summary     map[string]interface{}   `json:"summary"`
	Trends      []map[string]interface{} `json:"trends"`
	GeneratedAt time.Time                `json:"generatedAt"`
}

// AssignTicketRequest 分配工单请求
// 由控制器进行非零校验
type AssignTicketRequest struct {
	AssigneeID int `json:"assigneeId"`
}

// EscalateTicketRequest 升级工单请求
type EscalateTicketRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// ResolveTicketRequest 解决工单请求
type ResolveTicketRequest struct {
	TicketID           int    `json:"ticketId"`
	Resolution         string `json:"resolution"` // 解决方案（可选，兼容前端发送 solution）
	Solution           string `json:"solution"`   // 兼容前端字段名
	ResolutionCategory string `json:"resolutionCategory,omitempty"`
	WorkNotes          string `json:"workNotes,omitempty"`
}

// CloseTicketRequest 关闭工单请求
type CloseTicketRequest struct {
	TicketID    int    `json:"ticketId"`
	CloseReason string `json:"closeReason,omitempty"`
	CloseNotes  string `json:"closeNotes,omitempty"`
	Feedback    string `json:"feedback,omitempty"`
}

// BatchCloseRequest 批量关闭工单请求
type BatchCloseRequest struct {
	TicketIDs   []int  `json:"ticketIds" binding:"required,min=1"`
	CloseReason string `json:"closeReason,omitempty"`
}

// BatchPriorityRequest 批量更新优先级请求
type BatchPriorityRequest struct {
	TicketIDs []int  `json:"ticketIds" binding:"required,min=1"`
	Priority  string `json:"priority" binding:"required,oneof=low medium high critical urgent"`
}

// CreateSubtaskRequest 创建子任务请求（轻量级，不强制要求 Priority 等父工单字段）
type CreateSubtaskRequest struct {
	Title       string                 `json:"title" binding:"required,min=1,max=200"`
	Description string                 `json:"description"`
	Priority    string                 `json:"priority" binding:"omitempty,oneof=low medium high critical urgent"`
	Type        string                 `json:"type" binding:"omitempty,oneof=incident service_request change ticket problem"`
	AssigneeID  int                    `json:"assigneeId"`
	FormFields  map[string]interface{} `json:"formFields"`
}
