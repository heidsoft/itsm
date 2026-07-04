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
	TicketID int    `json:"ticketId"`
	CCUsers  []int  `json:"ccUsers" binding:"required,min=1"`
	Comment  string `json:"comment"`
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
