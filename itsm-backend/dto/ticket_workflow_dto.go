package dto

import "time"

// TicketWorkflowAction 工单流转操作类型
type TicketWorkflowAction string

const (
	WorkflowActionAccept        TicketWorkflowAction = "accept"
	WorkflowActionReject        TicketWorkflowAction = "reject"
	WorkflowActionWithdraw      TicketWorkflowAction = "withdraw"
	WorkflowActionForward       TicketWorkflowAction = "forward"
	WorkflowActionCC            TicketWorkflowAction = "cc"
	WorkflowActionEscalate      TicketWorkflowAction = "escalate"
	WorkflowActionApprove       TicketWorkflowAction = "approve"
	WorkflowActionApproveReject TicketWorkflowAction = "approve_reject"
	WorkflowActionDelegate      TicketWorkflowAction = "delegate"
	WorkflowActionResolve       TicketWorkflowAction = "resolve"
	WorkflowActionClose         TicketWorkflowAction = "close"
	WorkflowActionReopen        TicketWorkflowAction = "reopen"
)

// ApprovalStatus 审批状态
type ApprovalStatus string

const (
	ApprovalStatusPending    ApprovalStatus = "pending"
	ApprovalStatusInProgress ApprovalStatus = "in_progress"
	ApprovalStatusApproved   ApprovalStatus = "approved"
	ApprovalStatusRejected   ApprovalStatus = "rejected"
	ApprovalStatusCancelled  ApprovalStatus = "cancelled"
)

// WorkflowUserInfo 用户信息
type WorkflowUserInfo struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	FullName   string `json:"fullName"`
	Email      string `json:"email"`
	Avatar     string `json:"avatar,omitempty"`
	Department string `json:"department,omitempty"`
	Role       string `json:"role"`
}

// AttachmentInfo 附件信息
type AttachmentInfo struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

// TicketWorkflowRecord 工单流转记录
type TicketWorkflowRecord struct {
	ID          int                    `json:"id"`
	TicketID    int                    `json:"ticketId"`
	Action      TicketWorkflowAction   `json:"action"`
	FromStatus  *string                `json:"fromStatus,omitempty"`
	ToStatus    *string                `json:"toStatus,omitempty"`
	Operator    WorkflowUserInfo       `json:"operator"`
	FromUser    *WorkflowUserInfo      `json:"fromUser,omitempty"`
	ToUser      *WorkflowUserInfo      `json:"toUser,omitempty"`
	Comment     string                 `json:"comment,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
	Attachments []AttachmentInfo       `json:"attachments,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
}

// ApprovalRecord 审批记录
type ApprovalRecord struct {
	ID          int               `json:"id"`
	TicketID    int               `json:"ticketId"`
	Level       int               `json:"level"`
	LevelName   string            `json:"levelName"`
	Approver    WorkflowUserInfo  `json:"approver"`
	Status      ApprovalStatus    `json:"status"`
	Action      *string           `json:"action,omitempty"` // approve, reject, delegate
	Comment     string            `json:"comment,omitempty"`
	Attachments []AttachmentInfo  `json:"attachments,omitempty"`
	DelegateTo  *WorkflowUserInfo `json:"delegateTo,omitempty"`
	ProcessedAt *time.Time        `json:"processedAt,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
}

// TicketWorkflowState 工单流转状态
type TicketWorkflowState struct {
	TicketID             int                    `json:"ticketId"`
	CurrentStatus        string                 `json:"currentStatus"`
	CurrentAssignee      *WorkflowUserInfo      `json:"currentAssignee,omitempty"`
	ApprovalStatus       *ApprovalStatus        `json:"approvalStatus,omitempty"`
	CurrentApprovalLevel *int                   `json:"currentApprovalLevel,omitempty"`
	TotalApprovalLevels  *int                   `json:"totalApprovalLevels,omitempty"`
	PendingApprovers     []WorkflowUserInfo     `json:"pendingApprovers,omitempty"`
	CompletedApprovals   []ApprovalRecord       `json:"completedApprovals,omitempty"`
	CanAccept            bool                   `json:"canAccept"`
	CanReject            bool                   `json:"canReject"`
	CanWithdraw          bool                   `json:"canWithdraw"`
	CanForward           bool                   `json:"canForward"`
	CanCC                bool                   `json:"canCc"`
	CanApprove           bool                   `json:"canApprove"`
	CanResolve           bool                   `json:"canResolve"`
	CanClose             bool                   `json:"canClose"`
	AvailableActions     []TicketWorkflowAction `json:"availableActions"`
	// BpmnProcessState 聚合 BPMN 真实节点状态；未启动 / 终态时仍返回结构体，
	// 由 bpmnStatus 区分（not_started / running / completed / suspended / terminated）。
	BpmnProcessState *BpmnProcessState `json:"bpmnProcessState,omitempty"`
}

// BpmnProcessState 聚合 BPMN 流程实例的当前/下一/历史节点，供工单详情页直接渲染。
//
// 设计要点：
//   - 单结构体覆盖全部状态，调用方按 bpmnStatus 分支；不引入二级 data 嵌套。
//   - currentActivity* 在 running 时有值；completed/terminated 时省略，调用方按空态处理。
//   - nextActivities 由 outgoing sequence flows 解析得到，对网关节点按 isGateway=true 标记。
//   - history 按 BPMN 引擎 process_tasks 排序输出，包含人/服务节点的实际处理结果。
type BpmnProcessState struct {
	ProcessInstanceID     string             `json:"processInstanceId"`
	ProcessDefinitionKey  string             `json:"processDefinitionKey"`
	ProcessDefinitionName string             `json:"processDefinitionName"`
	BpmnStatus            string             `json:"bpmnStatus"`
	CurrentActivityID     string             `json:"currentActivityId,omitempty"`
	CurrentActivityName   string             `json:"currentActivityName,omitempty"`
	CurrentActivityType   string             `json:"currentActivityType,omitempty"`
	CurrentAssignees      []WorkflowUserInfo `json:"currentAssignees,omitempty"`
	NextActivities        []NextActivityInfo `json:"nextActivities,omitempty"`
	History               []BpmnHistoryItem  `json:"history,omitempty"`
	StartedAt             *time.Time         `json:"startedAt,omitempty"`
	EndedAt               *time.Time         `json:"endedAt,omitempty"`
}

// NextActivityInfo 描述当前节点的下一步候选活动。
// IsGateway=true 表示该活动为网关节点（exclusive/parallel/inclusive），
// 调用方应根据此标志决定是否展开分支说明。
type NextActivityInfo struct {
	ActivityID   string             `json:"activityId"`
	ActivityName string             `json:"activityName"`
	ActivityType string             `json:"activityType"`
	Assignees    []WorkflowUserInfo `json:"assignees,omitempty"`
	IsGateway    bool               `json:"isGateway"`
}

// BpmnHistoryItem 描述单个节点的历史快照。
// Outcome 字段取自 process_tasks 的 status 与业务变量（approvalResult/approvalAction），
// 例如 approved / rejected / completed / skipped。
type BpmnHistoryItem struct {
	ActivityID   string            `json:"activityId"`
	ActivityName string            `json:"activityName"`
	ActivityType string            `json:"activityType"`
	StartTime    time.Time         `json:"startTime"`
	EndTime      *time.Time        `json:"endTime,omitempty"`
	Assignee     *WorkflowUserInfo `json:"assignee,omitempty"`
	Outcome      string            `json:"outcome,omitempty"`
}

// AcceptTicketRequest 接单请求
type AcceptTicketRequest struct {
	TicketID int    `json:"ticketId"`
	Comment  string `json:"comment"`
}

// RejectTicketRequest 驳回请求
type RejectTicketRequest struct {
	TicketID       int     `json:"ticketId"`
	Reason         string  `json:"reason" binding:"required"`
	Comment        string  `json:"comment" binding:"required"`
	ReturnToStatus *string `json:"returnToStatus"`
}

// WithdrawTicketRequest 撤回请求
type WithdrawTicketRequest struct {
	TicketID int    `json:"ticketId"`
	Reason   string `json:"reason" binding:"required"`
}

// ForwardTicketRequest 转发请求
type ForwardTicketRequest struct {
	TicketID          int    `json:"ticketId"`
	ToUserID          int    `json:"toUserId" binding:"required"`
	Comment           string `json:"comment" binding:"required"`
	TransferOwnership bool   `json:"transferOwnership"`
}

// CCTicketRequest 抄送请求
type CCTicketRequest struct {
	TicketID       int      `json:"ticketId"`
	CCUsers        []int    `json:"ccUsers" binding:"required,min=1"`
	Comment        string   `json:"comment"`
	NotifyChannels []string `json:"notifyChannels"`
}

// ApproveTicketRequest 审批请求
type ApproveTicketRequest struct {
	TicketID         int    `json:"ticketId"`
	ApprovalID       int    `json:"approvalId" binding:"required"`
	Action           string `json:"action" binding:"required,oneof=approve reject delegate"`
	Comment          string `json:"comment"`
	DelegateToUserID *int   `json:"delegateToUserId"`
}

// ReopenTicketRequest 重开工单请求
type ReopenTicketRequest struct {
	TicketID int    `json:"ticketId"`
	Reason   string `json:"reason" binding:"required"`
}

// TicketCC 抄送人
type TicketCC struct {
	ID       int              `json:"id"`
	TicketID int              `json:"ticketId"`
	User     WorkflowUserInfo `json:"user"`
	AddedBy  WorkflowUserInfo `json:"addedBy"`
	AddedAt  time.Time        `json:"addedAt"`
	IsActive bool             `json:"isActive"`
}

// TicketCCRecordResponse 抄送记录响应
type TicketCCRecordResponse struct {
	ID           int              `json:"id"`
	TicketID     int              `json:"ticketId"`
	TicketNumber string           `json:"ticketNumber"`
	Title        string           `json:"title"`
	Status       string           `json:"status"`
	Priority     string           `json:"priority"`
	User         WorkflowUserInfo `json:"user"`
	AddedBy      WorkflowUserInfo `json:"addedBy"`
	AddedAt      time.Time        `json:"addedAt"`
	IsActive     bool             `json:"isActive"`
}

// TicketCCListResponse 抄送记录列表响应
type TicketCCListResponse struct {
	Records []TicketCCRecordResponse `json:"records"`
	Total   int                      `json:"total"`
}

// TicketWorkflowStats 工单流转统计
type TicketWorkflowStats struct {
	TotalTransitions      int                          `json:"totalTransitions"`
	AverageTransitionTime float64                      `json:"averageTransitionTime"` // 小时
	ByAction              map[TicketWorkflowAction]int `json:"byAction"`
	ByStatus              map[string]int               `json:"byStatus"`
	ApprovalStats         ApprovalStatistics           `json:"approvalStats"`
}

// ApprovalStatistics 审批统计
type ApprovalStatistics struct {
	TotalApprovals      int     `json:"totalApprovals"`
	ApprovedCount       int     `json:"approvedCount"`
	RejectedCount       int     `json:"rejectedCount"`
	AverageApprovalTime float64 `json:"averageApprovalTime"` // 小时
	ApprovalRate        float64 `json:"approvalRate"`        // 百分比
}

// TicketActionPermissions 工单操作权限
type TicketActionPermissions struct {
	CanAccept       bool `json:"canAccept"`
	CanReject       bool `json:"canReject"`
	CanWithdraw     bool `json:"canWithdraw"`
	CanForward      bool `json:"canForward"`
	CanCC           bool `json:"canCc"`
	CanApprove      bool `json:"canApprove"`
	CanResolve      bool `json:"canResolve"`
	CanClose        bool `json:"canClose"`
	CanReopen       bool `json:"canReopen"`
	CanEdit         bool `json:"canEdit"`
	CanDelete       bool `json:"canDelete"`
	CanComment      bool `json:"canComment"`
	CanViewInternal bool `json:"canViewInternal"`
}
