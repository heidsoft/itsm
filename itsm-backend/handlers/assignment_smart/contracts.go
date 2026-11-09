package assignment_smart

import (
	"context"

	"itsm-backend/dto"
)

// 本文件声明智能分配域真正消费的最小能力面。方法集按整个域包的生产文件
// 对字段的调用并集整理（不能只看持有字段的那个文件；路由文件常是盲区）。
// 组合根注入的共享层具体指针自动满足这些接口，装配代码无需改动。
// 新增方法前先确认该域确实需要，不要把共享层的大接口原样搬回来。

// SmartAssignmentService 是自动分配与推荐所需的能力。
type SmartAssignmentService interface {
	AutoAssign(ctx context.Context, ticketID, tenantID int) (*dto.AutoAssignResponse, error)
	GetAssignRecommendations(ctx context.Context, ticketID, tenantID int) ([]*dto.AssignmentRecommendation, error)
}

// AssignmentRuleService 是分配规则 CRUD 与试运行所需的能力。
type AssignmentRuleService interface {
	CreateAssignmentRule(ctx context.Context, req *dto.CreateAssignmentRuleRequest, tenantID int) (*dto.AssignmentRuleResponse, error)
	DeleteAssignmentRule(ctx context.Context, ruleID, tenantID int) error
	GetAssignmentRule(ctx context.Context, ruleID, tenantID int) (*dto.AssignmentRuleResponse, error)
	ListAssignmentRules(ctx context.Context, tenantID int) ([]*dto.AssignmentRuleResponse, error)
	TestAssignmentRule(ctx context.Context, req *dto.TestAssignmentRuleRequest, tenantID int) (*dto.TestAssignmentRuleResponse, error)
	UpdateAssignmentRule(ctx context.Context, ruleID int, req *dto.UpdateAssignmentRuleRequest, tenantID int) (*dto.AssignmentRuleResponse, error)
}
