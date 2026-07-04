package dto

import "time"

// SLA预警规则相关DTO
type CreateSLAAlertRuleRequest struct {
	Name                 string                  `json:"name" binding:"required" example:"P1工单-严重预警"`
	SLADefinitionID      int                     `json:"slaDefinitionId" binding:"required" example:"1"`
	AlertLevel           string                  `json:"alertLevel" binding:"required,oneof=warning critical severe" example:"severe"`
	ThresholdPercentage  int                     `json:"thresholdPercentage" binding:"required,min=0,max=100" example:"95"`
	NotificationChannels []string                `json:"notificationChannels" binding:"required,min=1" example:"[\"email\",\"sms\",\"in_app\"]"`
	EscalationEnabled    bool                    `json:"escalationEnabled" example:"true"`
	EscalationLevels     []EscalationLevelConfig `json:"escalationLevels" example:"[]"`
	IsActive             bool                    `json:"isActive" example:"true"`
}

type UpdateSLAAlertRuleRequest struct {
	Name                 *string                  `json:"name,omitempty"`
	AlertLevel           *string                  `json:"alertLevel,omitempty"`
	ThresholdPercentage  *int                     `json:"thresholdPercentage,omitempty"`
	NotificationChannels *[]string                `json:"notificationChannels,omitempty"`
	EscalationEnabled    *bool                    `json:"escalationEnabled,omitempty"`
	EscalationLevels     *[]EscalationLevelConfig `json:"escalationLevels,omitempty"`
	IsActive             *bool                    `json:"isActive,omitempty"`
}

type EscalationLevelConfig struct {
	Level       int   `json:"level" example:"1"`
	Threshold   int   `json:"threshold" example:"95"`
	NotifyUsers []int `json:"notifyUsers" example:"[1,2]"`
}

type SLAAlertRuleResponse struct {
	ID                   int                     `json:"id" example:"1"`
	Name                 string                  `json:"name" example:"P1工单-严重预警"`
	SLADefinitionID      int                     `json:"slaDefinitionId" example:"1"`
	AlertLevel           string                  `json:"alertLevel" example:"severe"`
	ThresholdPercentage  int                     `json:"thresholdPercentage" example:"95"`
	NotificationChannels []string                `json:"notificationChannels" example:"[\"email\",\"sms\",\"in_app\"]"`
	EscalationEnabled    bool                    `json:"escalationEnabled" example:"true"`
	EscalationLevels     []EscalationLevelConfig `json:"escalationLevels"`
	IsActive             bool                    `json:"isActive" example:"true"`
	TenantID             int                     `json:"tenantId" example:"1"`
	CreatedAt            time.Time               `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt            time.Time               `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// SLA预警历史相关DTO
type SLAAlertHistoryResponse struct {
	ID                       int        `json:"id" example:"1"`
	TicketID                 int        `json:"ticketId" example:"1001"`
	TicketNumber             string     `json:"ticketNumber" example:"T-2024-001"`
	TicketTitle              string     `json:"ticketTitle" example:"数据库连接超时"`
	AlertRuleID              int        `json:"alertRuleId" example:"1"`
	AlertRuleName            string     `json:"alertRuleName" example:"P1工单-严重预警"`
	AlertLevel               string     `json:"alertLevel" example:"severe"`
	ThresholdPercentage      int        `json:"thresholdPercentage" example:"95"`
	ActualPercentage         float64    `json:"actualPercentage" example:"96.5"`
	NotificationSent         bool       `json:"notificationSent" example:"true"`
	EscalationLevel          int        `json:"escalationLevel" example:"1"`
	CreatedAt                time.Time  `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	ResolvedAt               *time.Time `json:"resolvedAt,omitempty"`
	CooldownRemainingSeconds int        `json:"cooldownRemainingSeconds" example:"0"`
	CooldownMinutes          int        `json:"cooldownMinutes" example:"15"`
	SuppressedByCooldown     bool       `json:"suppressedByCooldown" example:"false"`
}

type GetSLAAlertHistoryRequest struct {
	SLADefinitionID *int    `json:"slaDefinitionId,omitempty"`
	AlertRuleID     *int    `json:"alertRuleId,omitempty"`
	TicketID        *int    `json:"ticketId,omitempty"`
	AlertLevel      *string `json:"alertLevel,omitempty"`
	StartTime       string  `json:"startTime" example:"2024-01-01T00:00:00Z"`
	EndTime         string  `json:"endTime" example:"2024-01-31T23:59:59Z"`
	Page            int     `json:"page" example:"1"`
	PageSize        int     `json:"pageSize" example:"20"`
}
