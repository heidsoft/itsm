package dto

import (
	"time"
	"itsm-backend/ent"
)

// CreateTicketRequest 创建工单请求
type CreateTicketRequest struct {
	Title       string                 `json:"title" binding:"required,max=255"`
	Description string                 `json:"description"`
	Priority    string                 `json:"priority" binding:"required,oneof=low medium high critical"`
	FormFields  map[string]interface{} `json:"form_fields"`
	AssigneeID  *int                   `json:"assignee_id"`
}

// UpdateTicketRequest 更新工单请求
type UpdateTicketRequest struct {
	Status     *string                `json:"status" binding:"omitempty,oneof=draft submitted in_progress pending approved rejected closed cancelled"`
	Priority   *string                `json:"priority" binding:"omitempty,oneof=low medium high critical"`
	FormFields map[string]interface{} `json:"form_fields"`
	AssigneeID *int                   `json:"assignee_id"`
}

// ApprovalRequest 审批请求
type ApprovalRequest struct {
	Action   string `json:"action" binding:"required,oneof=approve reject"`
	Comment  string `json:"comment"`
	StepName string `json:"step_name"`
}

// CommentRequest 评论请求
type CommentRequest struct {
	Content string `json:"content" binding:"required"`
}

// TicketResponse 工单响应
type TicketResponse struct {
	ID           int                    `json:"id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Status       string                 `json:"status"`
	Priority     string                 `json:"priority"`
	FormFields   map[string]interface{} `json:"form_fields"`
	TicketNumber string                 `json:"ticket_number"`
	RequesterID  int                    `json:"requester_id"`
	AssigneeID   *int                   `json:"assignee_id"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Requester    *UserResponse          `json:"requester,omitempty"`
	Assignee     *UserResponse          `json:"assignee,omitempty"`
}

// UserResponse 用户响应
type UserResponse struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Department string `json:"department"`
}

// ApprovalLogResponse 审批记录响应
type ApprovalLogResponse struct {
	ID         int                    `json:"id"`
	ApproverID int                    `json:"approver_id"`
	Comment    string                 `json:"comment"`
	Status     string                 `json:"status"`
	StepOrder  int                    `json:"step_order"`
	StepName   string                 `json:"step_name"`
	Metadata   map[string]interface{} `json:"metadata"`
	ApprovedAt *time.Time             `json:"approved_at"`
	CreatedAt  time.Time              `json:"created_at"`
	Approver   *UserResponse          `json:"approver,omitempty"`
}

// ToTicketResponse 转换为工单响应
func ToTicketResponse(ticket *ent.Ticket) *TicketResponse {
	resp := &TicketResponse{
		ID:           ticket.ID,
		Title:        ticket.Title,
		Description:  ticket.Description,
		Status:       string(ticket.Status),
		Priority:     string(ticket.Priority),
		FormFields:   ticket.FormFields,
		TicketNumber: ticket.TicketNumber,
		RequesterID:  ticket.RequesterID,
		AssigneeID:   ticket.AssigneeID,
		CreatedAt:    ticket.CreatedAt,
		UpdatedAt:    ticket.UpdatedAt,
	}

	// 加载关联用户信息
	if ticket.Edges.Requester != nil {
		resp.Requester = &UserResponse{
			ID:         ticket.Edges.Requester.ID,
			Username:   ticket.Edges.Requester.Username,
			Name:       ticket.Edges.Requester.Name,
			Email:      ticket.Edges.Requester.Email,
			Department: ticket.Edges.Requester.Department,
		}
	}

	if ticket.Edges.Assignee != nil {
		resp.Assignee = &UserResponse{
			ID:         ticket.Edges.Assignee.ID,
			Username:   ticket.Edges.Assignee.Username,
			Name:       ticket.Edges.Assignee.Name,
			Email:      ticket.Edges.Assignee.Email,
			Department: ticket.Edges.Assignee.Department,
		}
	}

	return resp
}


// GetTicketsRequest 获取工单列表请求
type GetTicketsRequest struct {
	Page     int    `json:"page" form:"page"`
	Size     int    `json:"size" form:"size"`
	Status   string `json:"status" form:"status"`
	Priority string `json:"priority" form:"priority"`
	UserID   int    `json:"-"` // 从认证中间件获取
}

// TicketListResponse 工单列表响应
type TicketListResponse struct {
	Tickets []TicketResponse `json:"tickets"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	Size    int              `json:"size"`
}

// UpdateStatusRequest 更新状态请求
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}