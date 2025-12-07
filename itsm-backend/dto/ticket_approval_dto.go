package dto

import "time"

// 审批工作流相关DTO
type CreateApprovalWorkflowRequest struct {
	Name        string                 `json:"name" binding:"required" example:"标准工单审批流程"`
	Description string                 `json:"description" example:"适用于一般工单的标准审批流程"`
	TicketType  *string                `json:"ticket_type,omitempty" example:"service_request"`
	Priority    *string                `json:"priority,omitempty" example:"medium"`
	Nodes       []ApprovalNodeRequest  `json:"nodes" binding:"required,min=1"`
	IsActive    bool                   `json:"is_active" example:"true"`
}

type UpdateApprovalWorkflowRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	TicketType  *string                `json:"ticket_type,omitempty"`
	Priority    *string                `json:"priority,omitempty"`
	Nodes       *[]ApprovalNodeRequest `json:"nodes,omitempty"`
	IsActive    *bool                  `json:"is_active,omitempty"`
}

type ApprovalNodeRequest struct {
	Level          int                      `json:"level" binding:"required,min=1" example:"1"`
	Name           string                   `json:"name" binding:"required" example:"直属主管审批"`
	ApproverType   string                   `json:"approver_type" binding:"required,oneof=user role department dynamic" example:"role"`
	ApproverIDs    []int                    `json:"approver_ids" binding:"required,min=1" example:"[1,2]"`
	ApprovalMode   string                   `json:"approval_mode" binding:"required,oneof=sequential parallel any all" example:"any"`
	MinimumApprovals *int                   `json:"minimum_approvals,omitempty"`
	TimeoutHours   *int                     `json:"timeout_hours,omitempty" example:"24"`
	Conditions     []ApprovalConditionConfig     `json:"conditions,omitempty"`
	AllowReject    bool                     `json:"allow_reject" example:"true"`
	AllowDelegate  bool                     `json:"allow_delegate" example:"false"`
	RejectAction   string                   `json:"reject_action" binding:"required,oneof=end return custom" example:"end"`
	ReturnToLevel  *int                     `json:"return_to_level,omitempty"`
}

// ApprovalConditionConfig 审批条件配置（避免与ticket_type_dto.go中的ApprovalCondition冲突）
type ApprovalConditionConfig struct {
	Field    string      `json:"field" example:"amount"`
	Operator string      `json:"operator" example:"greater_than"`
	Value    interface{} `json:"value" example:"10000"`
}

type ApprovalWorkflowResponse struct {
	ID          int                    `json:"id" example:"1"`
	Name        string                 `json:"name" example:"标准工单审批流程"`
	Description string                 `json:"description"`
	TicketType  *string                `json:"ticket_type,omitempty"`
	Priority    *string                `json:"priority,omitempty"`
	Nodes       []ApprovalNodeResponse `json:"nodes"`
	IsActive    bool                   `json:"is_active" example:"true"`
	TenantID    int                    `json:"tenant_id" example:"1"`
	CreatedAt   time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt   time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

type ApprovalNodeResponse struct {
	ID              string                 `json:"id" example:"node1"`
	Level           int                    `json:"level" example:"1"`
	Name            string                 `json:"name" example:"直属主管审批"`
	ApproverType    string                 `json:"approver_type" example:"role"`
	ApproverIDs     []int                  `json:"approver_ids" example:"[1,2]"`
	ApproverNames   []string               `json:"approver_names" example:"[\"张三\",\"李四\"]"`
	ApprovalMode    string                 `json:"approval_mode" example:"any"`
	MinimumApprovals *int                  `json:"minimum_approvals,omitempty"`
	TimeoutHours    *int                  `json:"timeout_hours,omitempty" example:"24"`
	Conditions      []ApprovalConditionConfig   `json:"conditions,omitempty"`
	AllowReject     bool                  `json:"allow_reject" example:"true"`
	AllowDelegate   bool                  `json:"allow_delegate" example:"false"`
	RejectAction    string                `json:"reject_action" example:"end"`
	ReturnToLevel   *int                  `json:"return_to_level,omitempty"`
}

// 审批记录相关DTO
type ApprovalRecordResponse struct {
	ID            int        `json:"id" example:"1"`
	TicketID      int        `json:"ticket_id" example:"1001"`
	TicketNumber  string     `json:"ticket_number" example:"T-2024-001"`
	TicketTitle   string     `json:"ticket_title" example:"系统响应缓慢"`
	WorkflowID    int        `json:"workflow_id" example:"1"`
	WorkflowName  string     `json:"workflow_name" example:"标准工单审批流程"`
	CurrentLevel  int        `json:"current_level" example:"1"`
	TotalLevels   int        `json:"total_levels" example:"2"`
	ApproverID    int        `json:"approver_id" example:"1"`
	ApproverName  string     `json:"approver_name" example:"张三"`
	Status        string     `json:"status" example:"pending"`
	Action        *string    `json:"action,omitempty" example:"approve"`
	Comment       *string    `json:"comment,omitempty"`
	CreatedAt     time.Time  `json:"created_at" example:"2024-01-01T00:00:00Z"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
}

type GetApprovalRecordsRequest struct {
	TicketID    *int    `json:"ticket_id,omitempty"`
	WorkflowID  *int    `json:"workflow_id,omitempty"`
	Status      *string `json:"status,omitempty"`
	Page        int     `json:"page" example:"1"`
	PageSize    int     `json:"page_size" example:"20"`
}

