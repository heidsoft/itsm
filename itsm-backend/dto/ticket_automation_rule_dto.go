package dto

import (
	"itsm-backend/ent"
	"time"
)

// CreateAutomationRuleRequest 创建自动化规则请求
type CreateAutomationRuleRequest struct {
	Name        string                   `json:"name" binding:"required"`
	Description *string                  `json:"description"`
	Priority    int                      `json:"priority"`
	Conditions  []map[string]interface{} `json:"conditions" binding:"required"`
	Actions     []map[string]interface{} `json:"actions" binding:"required"`
	IsActive    bool                     `json:"is_active"`
}

// UpdateAutomationRuleRequest 更新自动化规则请求
type UpdateAutomationRuleRequest struct {
	Name        *string                  `json:"name"`
	Description *string                  `json:"description"`
	Priority    *int                     `json:"priority"`
	Conditions  []map[string]interface{} `json:"conditions"`
	Actions     []map[string]interface{} `json:"actions"`
	IsActive    *bool                    `json:"is_active"`
}

// TestAutomationRuleRequest 测试自动化规则请求
type TestAutomationRuleRequest struct {
	RuleID   int `json:"rule_id" binding:"required"`
	TicketID int `json:"ticket_id" binding:"required"`
}

// TestAutomationRuleResponse 测试自动化规则响应
type TestAutomationRuleResponse struct {
	Matched    bool     `json:"matched"`
	Actions    []string `json:"actions,omitempty"`
	Reason     string   `json:"reason,omitempty"`
	Error      string   `json:"error,omitempty"`
}

// AutomationRuleResponse 自动化规则响应
type AutomationRuleResponse struct {
	ID             int                      `json:"id"`
	Name           string                   `json:"name"`
	Description    *string                  `json:"description"`
	Priority       int                      `json:"priority"`
	Conditions     []map[string]interface{} `json:"conditions"`
	Actions        []map[string]interface{} `json:"actions"`
	IsActive       bool                     `json:"is_active"`
	ExecutionCount int                      `json:"execution_count"`
	LastExecutedAt *time.Time               `json:"last_executed_at,omitempty"`
	CreatedBy      int                      `json:"created_by"`
	Creator        *UserResponse             `json:"creator,omitempty"`
	TenantID       int                      `json:"tenant_id"`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
}

// ListAutomationRulesResponse 自动化规则列表响应
type ListAutomationRulesResponse struct {
	Rules []*AutomationRuleResponse `json:"rules"`
	Total int                       `json:"total"`
}

// ToAutomationRuleResponse 转换为自动化规则响应
func ToAutomationRuleResponse(rule *ent.TicketAutomationRule, creator *ent.User) *AutomationRuleResponse {
	var description *string
	if rule.Description != "" {
		description = &rule.Description
	}

	response := &AutomationRuleResponse{
		ID:             rule.ID,
		Name:           rule.Name,
		Description:    description,
		Priority:       rule.Priority,
		Conditions:     rule.Conditions,
		Actions:        rule.Actions,
		IsActive:       rule.IsActive,
		ExecutionCount: rule.ExecutionCount,
		CreatedBy:      rule.CreatedBy,
		TenantID:       rule.TenantID,
		CreatedAt:      rule.CreatedAt,
		UpdatedAt:      rule.UpdatedAt,
	}

	if !rule.LastExecutedAt.IsZero() {
		response.LastExecutedAt = &rule.LastExecutedAt
	}

	if creator != nil {
		response.Creator = &UserResponse{
			ID:       creator.ID,
			Username: creator.Username,
			Email:    creator.Email,
			Name:     creator.Name,
			Role:     string(creator.Role),
		}
	}

	return response
}

