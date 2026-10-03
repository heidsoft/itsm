package dto

import "time"

// Notification 通知DTO
type Notification struct {
	ID         int       `json:"id"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	Type       string    `json:"type"`
	Read       bool      `json:"read"`
	ActionURL  *string   `json:"actionUrl,omitempty"`
	ActionText *string   `json:"actionText,omitempty"`
	UserID     int       `json:"userId"`
	TenantID   int       `json:"tenantId"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// CreateNotificationRequest 创建通知请求
type CreateNotificationRequest struct {
	Title      string  `json:"title" binding:"required"`
	Message    string  `json:"message" binding:"required"`
	Type       string  `json:"type" binding:"required,oneof=info success warning error"`
	ActionURL  *string `json:"actionUrl,omitempty"`
	ActionText *string `json:"actionText,omitempty"`
	// UserID and TenantID are extracted from auth token by the handler; they are
	// accepted in the body only for backwards compatibility but not required.
	UserID   int `json:"userId"`
	TenantID int `json:"tenantId"`
}

// UpdateNotificationRequest 更新通知请求
type UpdateNotificationRequest struct {
	Title      *string `json:"title,omitempty"`
	Message    *string `json:"message,omitempty"`
	Type       *string `json:"type,omitempty" binding:"omitempty,oneof=info success warning error"`
	Read       *bool   `json:"read,omitempty"`
	ActionURL  *string `json:"actionUrl,omitempty"`
	ActionText *string `json:"actionText,omitempty"`
}

// GetNotificationsRequest 获取通知列表请求。
//
// 分页只认 page/pageSize，且不在 binding 里写 max：越界由 common 的分页通道回落到
// 默认页长，binding 再判一次就把同一条规则写成了两处。
// userId/tenantId 用 form:"-" 拒绝绑定：它们只能来自认证上下文（handler 覆盖），
// 客户端自报的身份不参与查询。
type GetNotificationsRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Type     string `form:"type"`
	Read     *bool  `form:"read"`
	UserID   int    `form:"-"`
	TenantID int    `form:"-"`
}

// NotificationListResponse 通知列表响应。
//
// 集合键固定为 items（`notifications` 是 AGENTS.md 禁止的领域名第二键），分页键由
// common.NewPaginationResponse 一次算出：本端点真的做 Count + Offset/Limit，
// 少一个键调用方就无法核对是否还有下一页。
type NotificationListResponse struct {
	Items      []Notification `json:"items"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	TotalPages int            `json:"totalPages"`
}

// MarkNotificationReadRequest 标记通知已读请求
type MarkNotificationReadRequest struct {
	NotificationID int `json:"notificationId" binding:"required"`
	UserID         int `json:"userId" binding:"required"`
	TenantID       int `json:"tenantId" binding:"required"`
}

// MarkAllNotificationsReadRequest 标记所有通知已读请求
type MarkAllNotificationsReadRequest struct {
	UserID   int `json:"userId" binding:"required"`
	TenantID int `json:"tenantId" binding:"required"`
}

// DeleteNotificationRequest 删除通知请求
type DeleteNotificationRequest struct {
	NotificationID int `json:"notificationId" binding:"required"`
	UserID         int `json:"userId" binding:"required"`
	TenantID       int `json:"tenantId" binding:"required"`
}

// BatchNotificationRequest 批量操作当前用户的通知。
type BatchNotificationRequest struct {
	NotificationIDs []int `json:"notificationIds" binding:"required,min=1,max=100"`
}
