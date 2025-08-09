package dto

import "time"

// Notification 通知DTO
type Notification struct {
	ID         int       `json:"id"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	Type       string    `json:"type"`
	Read       bool      `json:"read"`
	ActionURL  *string   `json:"action_url,omitempty"`
	ActionText *string   `json:"action_text,omitempty"`
	UserID     int       `json:"user_id"`
	TenantID   int       `json:"tenant_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CreateNotificationRequest 创建通知请求
type CreateNotificationRequest struct {
	Title      string  `json:"title" binding:"required"`
	Message    string  `json:"message" binding:"required"`
	Type       string  `json:"type" binding:"required,oneof=info success warning error"`
	ActionURL  *string `json:"action_url,omitempty"`
	ActionText *string `json:"action_text,omitempty"`
	UserID     int     `json:"user_id" binding:"required"`
	TenantID   int     `json:"tenant_id" binding:"required"`
}

// UpdateNotificationRequest 更新通知请求
type UpdateNotificationRequest struct {
	Title      *string `json:"title,omitempty"`
	Message    *string `json:"message,omitempty"`
	Type       *string `json:"type,omitempty" binding:"omitempty,oneof=info success warning error"`
	Read       *bool   `json:"read,omitempty"`
	ActionURL  *string `json:"action_url,omitempty"`
	ActionText *string `json:"action_text,omitempty"`
}

// GetNotificationsRequest 获取通知列表请求
type GetNotificationsRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	Size     int    `form:"size" binding:"min=1,max=100"`
	Type     string `form:"type"`
	Read     *bool  `form:"read"`
	UserID   int    `form:"user_id" binding:"required"`
	TenantID int    `form:"tenant_id" binding:"required"`
}

// NotificationListResponse 通知列表响应
type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	Total         int            `json:"total"`
	Page          int            `json:"page"`
	Size          int            `json:"size"`
}

// MarkNotificationReadRequest 标记通知已读请求
type MarkNotificationReadRequest struct {
	NotificationID int `json:"notification_id" binding:"required"`
	UserID         int `json:"user_id" binding:"required"`
	TenantID       int `json:"tenant_id" binding:"required"`
}

// MarkAllNotificationsReadRequest 标记所有通知已读请求
type MarkAllNotificationsReadRequest struct {
	UserID   int `json:"user_id" binding:"required"`
	TenantID int `json:"tenant_id" binding:"required"`
}

// DeleteNotificationRequest 删除通知请求
type DeleteNotificationRequest struct {
	NotificationID int `json:"notification_id" binding:"required"`
	UserID         int `json:"user_id" binding:"required"`
	TenantID       int `json:"tenant_id" binding:"required"`
}
