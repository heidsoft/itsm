package dto

import (
	"time"
)

// NotificationPreferenceRequest 通知偏好请求
type NotificationPreferenceRequest struct {
	EventType        string  `json:"event_type" binding:"required"`
	EmailEnabled     *bool   `json:"email_enabled"`
	SMSEnabled       *bool   `json:"sms_enabled"`
	InAppEnabled     *bool   `json:"in_app_enabled"`
	PushEnabled      *bool   `json:"push_enabled"`
	Frequency        string  `json:"frequency"`
	QuietHoursStart  *string `json:"quiet_hours_start"`
	QuietHoursEnd    *string `json:"quiet_hours_end"`
	Timezone         string  `json:"timezone"`
}

// NotificationPreferenceResponse 通知偏好响应
type NotificationPreferenceResponse struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	EventType       string    `json:"event_type"`
	EmailEnabled    bool      `json:"email_enabled"`
	SmsEnabled      bool      `json:"sms_enabled"`
	InAppEnabled    bool      `json:"in_app_enabled"`
	PushEnabled     bool      `json:"push_enabled"`
	Frequency       string    `json:"frequency"`
	QuietHoursStart time.Time `json:"quiet_hours_start"`
	QuietHoursEnd   time.Time `json:"quiet_hours_end"`
	Timezone        string    `json:"timezone"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// BulkNotificationPreferenceRequest 批量更新通知偏好请求
type BulkNotificationPreferenceRequest struct {
	Preferences []NotificationPreferenceRequest `json:"preferences" binding:"required,dive"`
}

// NotificationEventType 通知事件类型
type NotificationEventType struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ListNotificationEventTypes 获取所有通知事件类型
func ListNotificationEventTypes() []NotificationEventType {
	return []NotificationEventType{
		{Code: "ticket_created", Name: "工单创建", Description: "当工单被创建时"},
		{Code: "ticket_assigned", Name: "工单分配", Description: "当工单被分配给自己时"},
		{Code: "ticket_updated", Name: "工单更新", Description: "当工单被更新时"},
		{Code: "ticket_resolved", Name: "工单解决", Description: "当工单被解决时"},
		{Code: "ticket_closed", Name: "工单关闭", Description: "当工单被关闭时"},
		{Code: "sla_warning", Name: "SLA警告", Description: "当SLA即将超时时"},
		{Code: "sla_violated", Name: "SLA违规", Description: "当SLA超时时"},
		{Code: "comment_added", Name: "新增评论", Description: "当工单新增评论时"},
		{Code: "approval_required", Name: "需要审批", Description: "当需要审批时"},
		{Code: "mention", Name: "被提及", Description: "当被@提及 时"},
		{Code: "incident_created", Name: "事件创建", Description: "当事件被创建时"},
		{Code: "incident_escalated", Name: "事件升级", Description: "当事件被升级时"},
		{Code: "change_approved", Name: "变更批准", Description: "当变更被批准时"},
		{Code: "change_rejected", Name: "变更拒绝", Description: "当变更被拒绝时"},
		{Code: "problem_identified", Name: "问题识别", Description: "当问题被识别时"},
	}
}
