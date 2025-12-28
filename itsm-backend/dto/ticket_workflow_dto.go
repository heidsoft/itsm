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
	FullName   string `json:"full_name"`
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
	TicketID    int                    `json:"ticket_id"`
	Action      TicketWorkflowAction   `json:"action"`
	FromStatus  *string                `json:"from_status,omitempty"`
	ToStatus    *string                `json:"to_status,omitempty"`
	Operator    WorkflowUserInfo       `json:"operator"`
	FromUser    *WorkflowUserInfo      `json:"from_user,omitempty"`
	ToUser      *WorkflowUserInfo      `json:"to_user,omitempty"`
	Comment     string                 `json:"comment,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
	Attachments []AttachmentInfo       `json:"attachments,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// ApprovalRecord 审批记录
type ApprovalRecord struct {
	ID          int               `json:"id"`
	TicketID    int               `json:"ticket_id"`
	Level       int               `json:"level"`
	LevelName   string            `json:"level_name"`
	Approver    WorkflowUserInfo  `json:"approver"`
	Status      ApprovalStatus    `json:"status"`
	Action      *string           `json:"action,omitempty"` // approve, reject, delegate
	Comment     string            `json:"comment,omitempty"`
	Attachments []AttachmentInfo  `json:"attachments,omitempty"`
	DelegateTo  *WorkflowUserInfo `json:"delegate_to,omitempty"`
	ProcessedAt *time.Time        `json:"processed_at,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// TicketWorkflowState 工单流转状态
type TicketWorkflowState struct {
	TicketID             int                    `json:"ticket_id"`
	CurrentStatus        string                 `json:"current_status"`
	CurrentAssignee      *WorkflowUserInfo      `json:"current_assignee,omitempty"`
	ApprovalStatus       *ApprovalStatus        `json:"approval_status,omitempty"`
	CurrentApprovalLevel *int                   `json:"current_approval_level,omitempty"`
	TotalApprovalLevels  *int                   `json:"total_approval_levels,omitempty"`
	PendingApprovers     []WorkflowUserInfo     `json:"pending_approvers,omitempty"`
	CompletedApprovals   []ApprovalRecord       `json:"completed_approvals,omitempty"`
	CanAccept            bool                   `json:"can_accept"`
	CanReject            bool                   `json:"can_reject"`
	CanWithdraw          bool                   `json:"can_withdraw"`
	CanForward           bool                   `json:"can_forward"`
	CanCC                bool                   `json:"can_cc"`
	CanApprove           bool                   `json:"can_approve"`
	CanResolve           bool                   `json:"can_resolve"`
	CanClose             bool                   `json:"can_close"`
	AvailableActions     []TicketWorkflowAction `json:"available_actions"`
}

// AcceptTicketRequest 接单请求
type AcceptTicketRequest struct {
	TicketID int    `json:"ticket_id" binding:"required"`
	Comment  string `json:"comment"`
}

// RejectTicketRequest 驳回请求
type RejectTicketRequest struct {
	TicketID       int     `json:"ticket_id" binding:"required"`
	Reason         string  `json:"reason" binding:"required"`
	Comment        string  `json:"comment" binding:"required"`
	ReturnToStatus *string `json:"return_to_status"`
}

// WithdrawTicketRequest 撤回请求
type WithdrawTicketRequest struct {
	TicketID int    `json:"ticket_id" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

// ForwardTicketRequest 转发请求
type ForwardTicketRequest struct {
	TicketID          int    `json:"ticket_id" binding:"required"`
	ToUserID          int    `json:"to_user_id" binding:"required"`
	Comment           string `json:"comment" binding:"required"`
	TransferOwnership bool   `json:"transfer_ownership"`
}

// CCTicketRequest 抄送请求
type CCTicketRequest struct {
	TicketID int    `json:"ticket_id" binding:"required"`
	CCUsers  []int  `json:"cc_users" binding:"required,min=1"`
	Comment  string `json:"comment"`
}

// ApproveTicketRequest 审批请求
type ApproveTicketRequest struct {
	TicketID         int    `json:"ticket_id" binding:"required"`
	ApprovalID       int    `json:"approval_id" binding:"required"`
	Action           string `json:"action" binding:"required,oneof=approve reject delegate"`
	Comment          string `json:"comment" binding:"required"`
	DelegateToUserID *int   `json:"delegate_to_user_id"`
}

// ReopenTicketRequest 重开工单请求
type ReopenTicketRequest struct {
	TicketID int    `json:"ticket_id" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

// TicketCC 抄送人
type TicketCC struct {
	ID       int              `json:"id"`
	TicketID int              `json:"ticket_id"`
	User     WorkflowUserInfo `json:"user"`
	AddedBy  WorkflowUserInfo `json:"added_by"`
	AddedAt  time.Time        `json:"added_at"`
	IsActive bool             `json:"is_active"`
}

// TicketWorkflowStats 工单流转统计
type TicketWorkflowStats struct {
	TotalTransitions      int                          `json:"total_transitions"`
	AverageTransitionTime float64                      `json:"average_transition_time"` // 小时
	ByAction              map[TicketWorkflowAction]int `json:"by_action"`
	ByStatus              map[string]int               `json:"by_status"`
	ApprovalStats         ApprovalStatistics           `json:"approval_stats"`
}

// ApprovalStatistics 审批统计
type ApprovalStatistics struct {
	TotalApprovals      int     `json:"total_approvals"`
	ApprovedCount       int     `json:"approved_count"`
	RejectedCount       int     `json:"rejected_count"`
	AverageApprovalTime float64 `json:"average_approval_time"` // 小时
	ApprovalRate        float64 `json:"approval_rate"`         // 百分比
}

// TicketActionPermissions 工单操作权限
type TicketActionPermissions struct {
	CanAccept       bool `json:"can_accept"`
	CanReject       bool `json:"can_reject"`
	CanWithdraw     bool `json:"can_withdraw"`
	CanForward      bool `json:"can_forward"`
	CanCC           bool `json:"can_cc"`
	CanApprove      bool `json:"can_approve"`
	CanResolve      bool `json:"can_resolve"`
	CanClose        bool `json:"can_close"`
	CanReopen       bool `json:"can_reopen"`
	CanEdit         bool `json:"can_edit"`
	CanDelete       bool `json:"can_delete"`
	CanComment      bool `json:"can_comment"`
	CanViewInternal bool `json:"can_view_internal"`
}
