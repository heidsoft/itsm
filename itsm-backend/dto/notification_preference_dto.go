package dto

import (
	"time"
)

// NotificationPreferenceRequest 通知偏好请求
type NotificationPreferenceRequest struct {
	EventType       string  `json:"eventType" binding:"required"`
	EmailEnabled    *bool   `json:"emailEnabled"`
	SMSEnabled      *bool   `json:"smsEnabled"`
	InAppEnabled    *bool   `json:"inAppEnabled"`
	PushEnabled     *bool   `json:"pushEnabled"`
	Frequency       string  `json:"frequency"`
	QuietHoursStart *string `json:"quietHoursStart"`
	QuietHoursEnd   *string `json:"quietHoursEnd"`
	Timezone        string  `json:"timezone"`
}

// NotificationPreferenceResponse 通知偏好响应
type NotificationPreferenceResponse struct {
	ID              int       `json:"id"`
	UserID          int       `json:"userId"`
	EventType       string    `json:"eventType"`
	EmailEnabled    bool      `json:"emailEnabled"`
	SmsEnabled      bool      `json:"smsEnabled"`
	InAppEnabled    bool      `json:"inAppEnabled"`
	PushEnabled     bool      `json:"pushEnabled"`
	Frequency       string    `json:"frequency"`
	QuietHoursStart time.Time `json:"quietHoursStart"`
	QuietHoursEnd   time.Time `json:"quietHoursEnd"`
	Timezone        string    `json:"timezone"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
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
		{Code: "ticket_created", Name: "Ticket Created", Description: "When a ticket is created"},
		{Code: "ticket_assigned", Name: "Ticket Assigned", Description: "When a ticket is assigned to you"},
		{Code: "ticket_updated", Name: "Ticket Updated", Description: "When a ticket is updated"},
		{Code: "ticket_resolved", Name: "Ticket Resolved", Description: "When a ticket is resolved"},
		{Code: "ticket_closed", Name: "Ticket Closed", Description: "When a ticket is closed"},
		{Code: "sla_warning", Name: "SLA Warning", Description: "When an SLA is approaching its deadline"},
		{Code: "sla_violated", Name: "SLA Breach", Description: "When an SLA deadline is breached"},
		{Code: "comment_added", Name: "Comment Added", Description: "When a comment is added to a ticket"},
		{Code: "approval_required", Name: "Approval Required", Description: "When approval is required"},
		{Code: "mention", Name: "Mentioned", Description: "When you are @mentioned"},
		{Code: "incident_created", Name: "Incident Created", Description: "When an incident is created"},
		{Code: "incident_escalated", Name: "Incident Escalated", Description: "When an incident is escalated"},
		{Code: "change_approved", Name: "Change Approved", Description: "When a change is approved"},
		{Code: "change_rejected", Name: "Change Rejected", Description: "When a change is rejected"},
		{Code: "problem_identified", Name: "Problem Identified", Description: "When a problem is identified"},
	}
}
