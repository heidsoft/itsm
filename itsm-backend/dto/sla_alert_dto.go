package dto

import "time"

// SLA预警规则相关DTO
type CreateSLAAlertRuleRequest struct {
	Name                string   `json:"name" binding:"required" example:"P1工单-严重预警"`
	SLADefinitionID     int      `json:"sla_definition_id" binding:"required" example:"1"`
	AlertLevel          string   `json:"alert_level" binding:"required,oneof=warning critical severe" example:"severe"`
	ThresholdPercentage int      `json:"threshold_percentage" binding:"required,min=0,max=100" example:"95"`
	NotificationChannels []string `json:"notification_channels" binding:"required,min=1" example:"[\"email\",\"sms\",\"in_app\"]"`
	EscalationEnabled   bool     `json:"escalation_enabled" example:"true"`
	EscalationLevels    []EscalationLevelConfig `json:"escalation_levels" example:"[]"`
	IsActive            bool     `json:"is_active" example:"true"`
}

type UpdateSLAAlertRuleRequest struct {
	Name                *string   `json:"name,omitempty"`
	AlertLevel          *string   `json:"alert_level,omitempty"`
	ThresholdPercentage *int      `json:"threshold_percentage,omitempty"`
	NotificationChannels *[]string `json:"notification_channels,omitempty"`
	EscalationEnabled   *bool     `json:"escalation_enabled,omitempty"`
	EscalationLevels    *[]EscalationLevelConfig `json:"escalation_levels,omitempty"`
	IsActive            *bool     `json:"is_active,omitempty"`
}

type EscalationLevelConfig struct {
	Level       int   `json:"level" example:"1"`
	Threshold   int   `json:"threshold" example:"95"`
	NotifyUsers []int `json:"notify_users" example:"[1,2]"`
}

type SLAAlertRuleResponse struct {
	ID                  int                    `json:"id" example:"1"`
	Name                string                 `json:"name" example:"P1工单-严重预警"`
	SLADefinitionID     int                    `json:"sla_definition_id" example:"1"`
	AlertLevel          string                 `json:"alert_level" example:"severe"`
	ThresholdPercentage int                    `json:"threshold_percentage" example:"95"`
	NotificationChannels []string               `json:"notification_channels" example:"[\"email\",\"sms\",\"in_app\"]"`
	EscalationEnabled   bool                   `json:"escalation_enabled" example:"true"`
	EscalationLevels    []EscalationLevelConfig `json:"escalation_levels"`
	IsActive            bool                   `json:"is_active" example:"true"`
	TenantID            int                    `json:"tenant_id" example:"1"`
	CreatedAt           time.Time              `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt           time.Time              `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// SLA预警历史相关DTO
type SLAAlertHistoryResponse struct {
	ID                int       `json:"id" example:"1"`
	TicketID          int       `json:"ticket_id" example:"1001"`
	TicketNumber      string    `json:"ticket_number" example:"T-2024-001"`
	TicketTitle       string    `json:"ticket_title" example:"数据库连接超时"`
	AlertRuleID       int       `json:"alert_rule_id" example:"1"`
	AlertRuleName     string    `json:"alert_rule_name" example:"P1工单-严重预警"`
	AlertLevel        string    `json:"alert_level" example:"severe"`
	ThresholdPercentage int     `json:"threshold_percentage" example:"95"`
	ActualPercentage  float64   `json:"actual_percentage" example:"96.5"`
	NotificationSent  bool     `json:"notification_sent" example:"true"`
	EscalationLevel   int       `json:"escalation_level" example:"1"`
	CreatedAt         time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`
}

type GetSLAAlertHistoryRequest struct {
	SLADefinitionID *int   `json:"sla_definition_id,omitempty"`
	AlertRuleID     *int   `json:"alert_rule_id,omitempty"`
	TicketID        *int   `json:"ticket_id,omitempty"`
	AlertLevel      *string `json:"alert_level,omitempty"`
	StartTime       string  `json:"start_time" example:"2024-01-01T00:00:00Z"`
	EndTime         string  `json:"end_time" example:"2024-01-31T23:59:59Z"`
	Page            int     `json:"page" example:"1"`
	PageSize        int     `json:"page_size" example:"20"`
}

