package dto

import (
	"time"
)

// CreateTicketRequest 创建工单请求
type CreateTicketRequest struct {
	Title                 string                 `json:"title" binding:"required,min=2,max=200"`
	Description           string                 `json:"description" binding:"omitempty,max=5000"`
	Priority              string                 `json:"priority" binding:"required,oneof=low medium high critical urgent"`
	Type                  string                 `json:"type" binding:"omitempty,oneof=incident service_request change ticket problem improvement"` // 工单类型
	TicketTypeID          *int                   `json:"ticketTypeId,omitempty"`
	TypeID                string                 `json:"typeId,omitempty"`                // legacy configured type code
	Category              string                 `json:"category"`                        // 分类名称（可选，前端传入）
	CategoryID            *int                   `json:"categoryId,omitempty"`            // 分类ID（优先使用）
	TemplateID            *int                   `json:"templateId,omitempty"`            // 模板ID
	RequesterID           int                    `json:"requesterId" binding:"omitempty"` // 从认证上下文中获取，前端可不传
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
	Description string                 `json:"description" binding:"omitempty,max=5000"`
	Priority    string                 `json:"priority" binding:"omitempty,oneof=low medium high critical urgent"`
	Status      string                 `json:"status" binding:"omitempty,oneof=new open assigned in_progress pending resolved closed cancelled approved rejected"`
	Type        string                 `json:"type" binding:"omitempty,oneof=incident service_request change ticket problem improvement"`
	Category    string                 `json:"category" binding:"omitempty"`
	CategoryID  *int                   `json:"categoryId,omitempty"`
	AssigneeID  int                    `json:"assigneeId"`
	RequesterID int                    `json:"requesterId"` // 创建人ID
	Tags        []string               `json:"tags"`
	Resolution  string                 `json:"resolution" binding:"omitempty"`
	FormFields  map[string]interface{} `json:"formFields,omitempty"`
	UserID      int                    `json:"userId" binding:"omitempty"` // 操作用户ID (后端自动填充)
	Version     int                    `json:"version"`                    // 版本号（乐观锁）
	Force       bool                   `json:"-"`                          // 仅限内部受信调用，禁止客户端绕过乐观锁
}

// ListTicketsRequest 获取工单列表请求
type ListTicketsRequest struct {
	Page           int        `json:"page" form:"page"`
	PageSize       int        `json:"pageSize" form:"pageSize"`
	Status         string     `json:"status" form:"status" binding:"omitempty,oneof=new open assigned in_progress pending resolved closed cancelled approved rejected"`
	Priority       string     `json:"priority" form:"priority" binding:"omitempty,oneof=low medium high critical urgent"`
	Type           string     `json:"type" form:"type" binding:"omitempty,oneof=incident service_request change ticket problem improvement"` // 工单类型筛选
	Category       string     `json:"category" form:"category"`                                                                              // 分类名称
	CategoryID     *int       `json:"categoryId" form:"categoryId"`                                                                          // 分类ID（优先）
	AssigneeID     *int       `json:"assigneeId" form:"assigneeId"`                                                                          // 处理人ID（可选）
	RequesterID    *int       `json:"requesterId" form:"requesterId"`                                                                        // 创建人ID（可选）
	ParentTicketID *int       `json:"parentTicketId" form:"parentTicketId"`
	TemplateID     *int       `json:"templateId" form:"templateId"`
	Keyword        string     `json:"keyword" form:"keyword"`
	DateFrom       *time.Time `json:"dateFrom" form:"dateFrom"` // 创建时间起始
	DateTo         *time.Time `json:"dateTo" form:"dateTo"`     // 创建时间截止
	IsOverdue      bool       `json:"isOverdue" form:"isOverdue"`
	SortBy         string     `json:"sortBy" form:"sortBy" binding:"omitempty,oneof=created_at updated_at title status priority ticket_number"`
	SortOrder      string     `json:"sortOrder" form:"sortOrder" binding:"omitempty,oneof=asc desc"`
}

// TicketResponse 工单响应
type TicketResponse struct {
	ID             int                    `json:"id"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Status         string                 `json:"status"`
	Priority       string                 `json:"priority"`
	Type           string                 `json:"type"`
	TicketTypeID   int                    `json:"ticketTypeId,omitempty"`
	TicketTypeCode string                 `json:"ticketTypeCode,omitempty"`
	TicketTypeName string                 `json:"ticketTypeName,omitempty"`
	FormFields     map[string]interface{} `json:"formFields,omitempty"`
	TicketNumber   string                 `json:"ticketNumber"`
	RequesterID    int                    `json:"requesterId"`
	AssigneeID     int                    `json:"assigneeId,omitempty"`
	// 显示名由服务端按当前租户投影（handlers/ticket attachUserNames）。
	// 历史上这里挂的是 requester/assignee 对象却从未赋值，前端因此把已派工单
	// 渲染成「未分配」；姓名以外的用户字段不在这个 DTO 里返回。
	RequesterName         string     `json:"requesterName,omitempty"`
	AssigneeName          string     `json:"assigneeName,omitempty"`
	TenantID              int        `json:"tenantId"`
	CategoryID            int        `json:"categoryId,omitempty"`
	DepartmentID          int        `json:"departmentId,omitempty"`
	ParentTicketID        int        `json:"parentTicketId,omitempty"`
	TemplateID            *int       `json:"templateId,omitempty"`
	Version               int        `json:"version"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
	Resolution            string     `json:"resolution,omitempty"`
	ResolutionCategory    string     `json:"resolutionCategory,omitempty"`
	ResolvedAt            *time.Time `json:"resolvedAt,omitempty"`
	ClosedAt              *time.Time `json:"closedAt,omitempty"`
	FirstResponseAt       *time.Time `json:"firstResponseAt,omitempty"`
	SLAResponseDeadline   *time.Time `json:"slaResponseDeadline,omitempty"`
	SLAResolutionDeadline *time.Time `json:"slaResolutionDeadline,omitempty"`
	Rating                int        `json:"rating,omitempty"`
}

// ListTicketsResponse 工单列表响应
type ListTicketsResponse struct {
	Items      []*TicketResponse `json:"items"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"pageSize"`
	TotalPages int               `json:"totalPages"`
}

// TicketStatusCount 工单状态分布项
type TicketStatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// TicketPriorityCount 工单优先级分布项
type TicketPriorityCount struct {
	Priority string `json:"priority"`
	Count    int    `json:"count"`
}

// TicketStatsResponse 工单统计响应
//
// 这是工单统计的唯一契约：计数器、分布、趋势、SLA 读数共用同一次请求的
// 行级数据范围与时间窗口，禁止前端再拼第二个数据源补齐页面上的卡片。
type TicketStatsResponse struct {
	Total        int `json:"total"`
	Open         int `json:"open"`
	InProgress   int `json:"inProgress"`
	Resolved     int `json:"resolved"`
	Closed       int `json:"closed"`
	Pending      int `json:"pending"`
	HighPriority int `json:"highPriority"`
	Overdue      int `json:"overdue"`
	// ByStatus/ByPriority 是与计数器同一范围、同一窗口的分组计数，
	// 只含真实存在的取值；报表分布必须用它，禁止改成前端拉一页数据自己数
	// （页长外的工作量会被静默截断）。紧急/关键这类单点 KPI 也从这里取，
	// 不再另立 urgent/critical 标量字段，避免同一数字出现两套口径。
	ByStatus   []TicketStatusCount   `json:"byStatus"`
	ByPriority []TicketPriorityCount `json:"byPriority"`
	// Scope 是本次统计跟随的行级数据范围，词表与 datascope 一致：
	// all（租户全量）/ department / owner（本人创建或受理）。
	Scope string `json:"scope"`
	// Window 回显实际生效的时间窗口；请求未带日期时缺省，表示不限窗口。
	// 页面必须显示它，否则「最近 30 天」和「全部」在屏幕上长得一样。
	Window *TicketStatsWindowResponse `json:"window,omitempty"`
	// DailyTrend 仅在带窗口的请求里返回；缺省表示本次未取趋势，
	// 不是「窗口内没有工单」——两者在前端必须能区分。
	DailyTrend []TicketDailyTrendPoint `json:"dailyTrend,omitempty"`
	// SLA 是窗口内的 SLA 达标读数（租户口径，见 sla.scope）。缺省表示
	// SLA 计时服务未接入或读数失败，属于「没取到」，前端不得渲染成 0%。
	SLA *TicketSLAComplianceResponse `json:"sla,omitempty"`
}

// TicketStatsWindowResponse 统计窗口回显（闭区间日 + 天数）
type TicketStatsWindowResponse struct {
	DateFrom string `json:"dateFrom"`
	DateTo   string `json:"dateTo"`
	Days     int    `json:"days"`
}

// TicketDailyTrendPoint 按天的新建/解决读数
type TicketDailyTrendPoint struct {
	Date     string `json:"date"`
	Created  int    `json:"created"`
	Resolved int    `json:"resolved"`
}

// TicketSLAComplianceResponse 窗口内的 SLA 达标与处理时长读数
type TicketSLAComplianceResponse struct {
	ComplianceRate     float64 `json:"complianceRate"`
	SampledTickets     int     `json:"sampledTickets"`
	ViolatedTickets    int     `json:"violatedTickets"`
	AvgResponseMinutes float64 `json:"avgResponseMinutes"`
	AvgResolutionHours float64 `json:"avgResolutionHours"`
	WindowDays         int     `json:"windowDays"`
	// Scope 固定为 tenant：SLA 达标率按租户全量计算，不跟随登录者的数据范围。
	Scope string `json:"scope"`
}

// AnalyticsTicketStatsResponse 工单分析统计响应（/api/v1/analytics/tickets）
type AnalyticsTicketStatsResponse struct {
	Total          int                   `json:"total"`
	StatusGroups   []TicketStatusGroup   `json:"statusGroups"`
	PriorityGroups []TicketPriorityGroup `json:"priorityGroups"`
	Trend30d       []TicketTrendPoint    `json:"trend30d"`
	GeneratedAt    string                `json:"generatedAt"`
}

type TicketStatusGroup struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type TicketPriorityGroup struct {
	Priority string `json:"priority"`
	Count    int    `json:"count"`
}

type TicketTrendPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
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
// format 只声明 service 层真实实现的格式；pdf 从未实现，请求它会得到 400 而不是 500。
type TicketExportRequest struct {
	Format  string                 `json:"format" binding:"required,oneof=csv excel"`
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
	// 指针用于区分「未传」与「显式 false」：创建时显式停用不能被默认成启用。
	IsActive  *bool     `json:"isActive,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TicketAssignmentRequest 工单分配请求
type TicketAssignmentRequest struct {
	TicketIDs      []int  `json:"ticketIds" binding:"required"`
	AssigneeID     int    `json:"assigneeId" binding:"required"`
	Reason         string `json:"reason"`
	NotifyAssignee bool   `json:"notifyAssignee"`
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
	Type        string                 `json:"type" binding:"omitempty,oneof=incident service_request change ticket problem improvement"`
	AssigneeID  int                    `json:"assigneeId"`
	FormFields  map[string]interface{} `json:"formFields"`
}
