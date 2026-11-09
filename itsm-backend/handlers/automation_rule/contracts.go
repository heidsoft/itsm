package automation_rule

// 域内窄契约：工单自动化规则 CRUD、列表与试运行。
// 生产装配传入具体实现，编译期断言见 internal/layering/domain_contracts_test.go。

import (
	"context"

	"itsm-backend/dto"
)

type AutomationRuleService interface {
	CreateAutomationRule(ctx context.Context, req *dto.CreateAutomationRuleRequest, userID, tenantID int) (*dto.AutomationRuleResponse, error)
	UpdateAutomationRule(ctx context.Context, ruleID int, req *dto.UpdateAutomationRuleRequest, tenantID int) (*dto.AutomationRuleResponse, error)
	DeleteAutomationRule(ctx context.Context, ruleID, tenantID int) error
	GetAutomationRule(ctx context.Context, ruleID, tenantID int) (*dto.AutomationRuleResponse, error)
	ListAutomationRules(ctx context.Context, tenantID int) ([]*dto.AutomationRuleResponse, error)
	TestAutomationRule(ctx context.Context, req *dto.TestAutomationRuleRequest, tenantID int) (*dto.TestAutomationRuleResponse, error)
}
