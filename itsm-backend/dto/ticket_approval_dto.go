package dto

import "time"

// 审批工作流相关DTO
type CreateApprovalWorkflowRequest struct {
	Name        string                `json:"name" binding:"required" example:"标准工单审批流程"`
	Description string                `json:"description" example:"适用于一般工单的标准审批流程"`
	TicketType  *string               `json:"ticketType,omitempty" example:"service_request"`
	Priority    *string               `json:"priority,omitempty" example:"medium"`
	Nodes       []ApprovalNodeRequest `json:"nodes" binding:"required,min=1"`
	IsActive    bool                  `json:"isActive" example:"true"`
}

type UpdateApprovalWorkflowRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	TicketType  *string                `json:"ticketType,omitempty"`
	Priority    *string                `json:"priority,omitempty"`
	Nodes       *[]ApprovalNodeRequest `json:"nodes,omitempty"`
	IsActive    *bool                  `json:"isActive,omitempty"`
}

type ApprovalNodeRequest struct {
	Level            int                       `json:"level" binding:"required,min=1" example:"1"`
	Name             string                    `json:"name" binding:"required" example:"直属主管审批"`
	ApproverType     string                    `json:"approverType" binding:"required,oneof=user role department dynamic dept_manager team_leader project_manager temp_team_leader amount_based" example:"role"`
	ApproverIDs      []int                     `json:"approverIds,omitempty" example:"[1,2]"`
	AssigneeType     string                    `json:"assigneeType,omitempty" example:"dept_manager"`
	AssigneeValue    string                    `json:"assigneeValue,omitempty" example:"1"`
	ApprovalMode     string                    `json:"approvalMode" binding:"required,oneof=sequential parallel any all" example:"any"`
	MinimumApprovals *int                      `json:"minimumApprovals,omitempty"`
	TimeoutHours     *int                      `json:"timeoutHours,omitempty" example:"24"`
	Conditions       []ApprovalConditionConfig `json:"conditions,omitempty"`
	AllowReject      bool                      `json:"allowReject" example:"true"`
	AllowDelegate    bool                      `json:"allowDelegate" example:"false"`
	RejectAction     string                    `json:"rejectAction" binding:"required,oneof=end return custom" example:"end"`
	ReturnToLevel    *int                      `json:"returnToLevel,omitempty"`
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
	TicketType  *string                `json:"ticketType,omitempty"`
	Priority    *string                `json:"priority,omitempty"`
	Nodes       []ApprovalNodeResponse `json:"nodes"`
	IsActive    bool                   `json:"isActive" example:"true"`
	TenantID    int                    `json:"tenantId" example:"1"`
	CreatedAt   time.Time              `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt   time.Time              `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

type ApprovalNodeResponse struct {
	ID               string                    `json:"id" example:"node1"`
	Level            int                       `json:"level" example:"1"`
	Name             string                    `json:"name" example:"直属主管审批"`
	ApproverType     string                    `json:"approverType" example:"role"`
	ApproverIDs      []int                     `json:"approverIds" example:"[1,2]"`
	AssigneeType     string                    `json:"assigneeType,omitempty" example:"dept_manager"`
	AssigneeValue    string                    `json:"assigneeValue,omitempty" example:"1"`
	ApproverNames    []string                  `json:"approverNames" example:"[\"张三\",\"李四\"]"`
	ApprovalMode     string                    `json:"approvalMode" example:"any"`
	MinimumApprovals *int                      `json:"minimumApprovals,omitempty"`
	TimeoutHours     *int                      `json:"timeoutHours,omitempty" example:"24"`
	Conditions       []ApprovalConditionConfig `json:"conditions,omitempty"`
	AllowReject      bool                      `json:"allowReject" example:"true"`
	AllowDelegate    bool                      `json:"allowDelegate" example:"false"`
	RejectAction     string                    `json:"rejectAction" example:"end"`
	ReturnToLevel    *int                      `json:"returnToLevel,omitempty"`
}

// 审批记录相关DTO
type ApprovalRecordResponse struct {
	ID           int        `json:"id" example:"1"`
	TicketID     int        `json:"ticketId" example:"1001"`
	TicketNumber string     `json:"ticketNumber" example:"T-2024-001"`
	TicketTitle  string     `json:"ticketTitle" example:"系统响应缓慢"`
	WorkflowID   int        `json:"workflowId" example:"1"`
	WorkflowName string     `json:"workflowName" example:"标准工单审批流程"`
	CurrentLevel int        `json:"currentLevel" example:"1"`
	TotalLevels  int        `json:"totalLevels" example:"2"`
	ApproverID   int        `json:"approverId" example:"1"`
	ApproverName string     `json:"approverName" example:"张三"`
	Status       string     `json:"status" example:"pending"`
	Action       *string    `json:"action,omitempty" example:"approve"`
	Comment      *string    `json:"comment,omitempty"`
	CreatedAt    time.Time  `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	ProcessedAt  *time.Time `json:"processedAt,omitempty"`
}

type GetApprovalRecordsRequest struct {
	TicketID   *int    `json:"ticketId,omitempty"`
	WorkflowID *int    `json:"workflowId,omitempty"`
	Status     *string `json:"status,omitempty"`
	Page       int     `json:"page" example:"1"`
	PageSize   int     `json:"pageSize" example:"20"`
}
