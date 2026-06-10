package dto

import (
	"time"

	"itsm-backend/ent"
)

// CreateIncidentCommentRequest 创建事件评论请求
type CreateIncidentCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=5000"`
	IsInternal  bool   `json:"isInternal"`
	Mentions    []int  `json:"mentions"`
	Attachments []int  `json:"attachments"`
}

// IncidentCommentResponse 事件评论响应
type IncidentCommentResponse struct {
	ID          int       `json:"id"`
	IncidentID  int       `json:"incidentId"`
	UserID      int       `json:"userId"`
	Content     string    `json:"content"`
	IsInternal  bool      `json:"isInternal"`
	Mentions    []int `json:"mentions"`
	Attachments []int     `json:"attachments"`
	User        *UserInfo `json:"user,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ToIncidentCommentResponse 将 Ent IncidentEvent 实体转换为 DTO
func ToIncidentCommentResponse(event *ent.IncidentEvent, user *ent.User) *IncidentCommentResponse {
	resp := &IncidentCommentResponse{
		ID:         event.ID,
		IncidentID: event.IncidentID,
		UserID:     event.UserID,
		Content:    event.Description,
		IsInternal: false,
		CreatedAt:  event.CreatedAt,
		UpdatedAt:  event.UpdatedAt,
	}

	// 从 data字段解析 isInternal, mentions, attachments
	if event.Data != nil {
		if v, ok := event.Data["isInternal"].(bool); ok {
			resp.IsInternal = v
		}
		if v, ok := event.Data["mentions"].([]int); ok {
			resp.Mentions = v
		}
		if v, ok := event.Data["attachments"].([]int); ok {
			resp.Attachments = v
		}
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