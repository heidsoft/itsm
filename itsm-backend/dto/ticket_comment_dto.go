package dto

import (
	"itsm-backend/ent"
	"time"
)

// CreateTicketCommentRequest 创建工单评论请求
type CreateTicketCommentRequest struct {
	Content    string  `json:"content" binding:"required,min=1,max=5000"`
	IsInternal bool    `json:"is_internal"` // 是否内部备注
	Mentions   []int   `json:"mentions"`    // @的用户ID列表
	Attachments []int  `json:"attachments"`  // 附件ID列表（后续实现）
}

// UpdateTicketCommentRequest 更新工单评论请求
type UpdateTicketCommentRequest struct {
	Content    string  `json:"content" binding:"omitempty,min=1,max=5000"`
	IsInternal *bool   `json:"is_internal"` // 是否内部备注
	Mentions   []int   `json:"mentions"`    // @的用户ID列表
}

// TicketCommentResponse 工单评论响应
type TicketCommentResponse struct {
	ID          int       `json:"id"`
	TicketID    int       `json:"ticket_id"`
	UserID      int       `json:"user_id"`
	Content     string    `json:"content"`
	IsInternal  bool      `json:"is_internal"`
	Mentions    []int     `json:"mentions"`
	Attachments []int     `json:"attachments"`
	User        *UserInfo `json:"user,omitempty"` // 评论人信息
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListTicketCommentsResponse 工单评论列表响应
type ListTicketCommentsResponse struct {
	Comments []*TicketCommentResponse `json:"comments"`
	Total    int                        `json:"total"`
}

// ToTicketCommentResponse 将 Ent 实体转换为 DTO
func ToTicketCommentResponse(comment *ent.TicketComment, user *ent.User) *TicketCommentResponse {
	resp := &TicketCommentResponse{
		ID:         comment.ID,
		TicketID:   comment.TicketID,
		UserID:     comment.UserID,
		Content:    comment.Content,
		IsInternal: comment.IsInternal,
		CreatedAt:  comment.CreatedAt,
		UpdatedAt:  comment.UpdatedAt,
	}

	// 设置 mentions
	if comment.Mentions != nil {
		resp.Mentions = comment.Mentions
	}

	// 设置 attachments
	if comment.Attachments != nil {
		resp.Attachments = comment.Attachments
	}

	// 设置用户信息
	if user != nil {
		resp.User = &UserInfo{
			ID:         user.ID,
			Username:   user.Username,
			Name:       user.Name,
			Email:      user.Email,
			Role:       string(user.Role),
			Department: user.Department,
			TenantID:   user.TenantID,
		}
	}

	return resp
}

