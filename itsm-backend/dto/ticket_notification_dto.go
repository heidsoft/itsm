package dto

import (
	"itsm-backend/ent"
	"time"
)

// TicketNotificationResponse 工单通知响应
type TicketNotificationResponse struct {
	ID        int       `json:"id"`
	TicketID  int       `json:"ticket_id"`
	UserID    int       `json:"user_id"`
	Type      string    `json:"type"`      // created, assigned, status_changed, commented, sla_warning, resolved, closed
	Channel   string    `json:"channel"`   // email, in_app, sms
	Content   string    `json:"content"`
	SentAt    *time.Time `json:"sent_at,omitempty"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	Status    string    `json:"status"`   // pending, sent, read
	CreatedAt time.Time `json:"created_at"`
	User      *UserInfo `json:"user,omitempty"` // 接收人信息
}

// ListTicketNotificationsResponse 工单通知列表响应
type ListTicketNotificationsResponse struct {
	Notifications []*TicketNotificationResponse `json:"notifications"`
	Total         int                           `json:"total"`
}

// SendTicketNotificationRequest 发送工单通知请求
type SendTicketNotificationRequest struct {
	UserIDs []int  `json:"user_ids" binding:"required,min=1"` // 接收人ID列表
	Type    string `json:"type" binding:"required"`           // 通知类型
	Channel string `json:"channel" binding:"required,oneof=email in_app sms"` // 通知渠道
	Content string `json:"content" binding:"required"`        // 通知内容
}

// UpdateNotificationPreferencesRequest 更新通知偏好请求
type UpdateNotificationPreferencesRequest struct {
	EmailEnabled   bool `json:"email_enabled"`   // 是否启用邮件通知
	InAppEnabled   bool `json:"in_app_enabled"`  // 是否启用站内消息通知
	SmsEnabled     bool `json:"sms_enabled"`    // 是否启用短信通知（可选）
	SlaWarningTime int  `json:"sla_warning_time"` // SLA警告提前时间（分钟）
}

// NotificationPreferencesResponse 通知偏好响应
type NotificationPreferencesResponse struct {
	UserID         int    `json:"user_id"`
	EmailEnabled   bool   `json:"email_enabled"`
	InAppEnabled   bool   `json:"in_app_enabled"`
	SmsEnabled     bool   `json:"sms_enabled"`
	SlaWarningTime int    `json:"sla_warning_time"`
}

// ToTicketNotificationResponse 将 Ent 实体转换为 DTO
func ToTicketNotificationResponse(notification *ent.TicketNotification, user *ent.User) *TicketNotificationResponse {
	resp := &TicketNotificationResponse{
		ID:        notification.ID,
		TicketID:  notification.TicketID,
		UserID:    notification.UserID,
		Type:      notification.Type,
		Channel:   notification.Channel,
		Content:   notification.Content,
		Status:    notification.Status,
		CreatedAt: notification.CreatedAt,
	}

	// SentAt 和 ReadAt 在 Ent 中可能是 time.Time 或 *time.Time
	// 检查是否为零值来判断是否已设置
	if !notification.SentAt.IsZero() {
		resp.SentAt = &notification.SentAt
	}
	if !notification.ReadAt.IsZero() {
		resp.ReadAt = &notification.ReadAt
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

