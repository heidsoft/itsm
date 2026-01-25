package service

import (
	"context"
	"fmt"
	"itsm-backend/ent"
	"itsm-backend/ent/workflowtask"
	"time"

	"go.uber.org/zap"
)

// 自动分配规则
type AutoAssignmentRule struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Conditions  map[string]interface{} `json:"conditions"`
	Actions     map[string]interface{} `json:"actions"`
	Priority    int                    `json:"priority"`
	IsActive    bool                   `json:"is_active"`
}

// 智能路由规则
type SmartRoutingRule struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Conditions  map[string]interface{} `json:"conditions"`
	RouteTo     string                 `json:"route_to"`
	Priority    int                    `json:"priority"`
	IsActive    bool                   `json:"is_active"`
}

// 自动升级规则
type AutoEscalationRule struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Conditions  map[string]interface{} `json:"conditions"`
	EscalateTo  string                 `json:"escalate_to"`
	TimeLimit   time.Duration          `json:"time_limit"`
	Priority    int                    `json:"priority"`
	IsActive    bool                   `json:"is_active"`
}

type WorkflowAutomationService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewWorkflowAutomationService(client *ent.Client, logger *zap.SugaredLogger) *WorkflowAutomationService {
	return &WorkflowAutomationService{
		client: client,
		logger: logger,
	}
}

// AutoAssignTask 自动分配任务
func (was *WorkflowAutomationService) AutoAssignTask(ctx context.Context, task *ent.WorkflowTask, tenantID int) error {
	was.logger.Infow("Auto assigning task", "task_id", task.TaskID, "tenant_id", tenantID)

	// 获取自动分配规则
	rules, err := was.getAutoAssignmentRules(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get auto assignment rules: %w", err)
	}

	// 按优先级排序规则
	sortedRules := was.sortAutoAssignmentRulesByPriority(rules)

	// 应用规则
	for _, rule := range sortedRules {
		if !rule.IsActive {
			continue
		}

		if was.matchesConditions(task, rule.Conditions) {
			assignee, err := was.executeAssignmentAction(task, rule.Actions)
			if err != nil {
				was.logger.Errorw("Failed to execute assignment action", "error", err, "rule_id", rule.ID)
				continue
			}

			if assignee != "" {
				// 更新任务分配
				_, err = was.client.WorkflowTask.UpdateOne(task).
					SetAssignee(assignee).
					SetStatus("pending").
					Save(ctx)
				if err != nil {
					return fmt.Errorf("failed to update task assignment: %w", err)
				}

				was.logger.Infow("Task auto assigned", "task_id", task.TaskID, "assignee", assignee, "rule_id", rule.ID)
				return nil
			}
		}
	}

	// 如果没有匹配的规则，使用默认分配策略
	return was.applyDefaultAssignment(ctx, task, tenantID)
}

// SmartRouteTask 智能路由任务
func (was *WorkflowAutomationService) SmartRouteTask(ctx context.Context, task *ent.WorkflowTask, tenantID int) error {
	was.logger.Infow("Smart routing task", "task_id", task.TaskID, "tenant_id", tenantID)

	// 获取智能路由规则
	rules, err := was.getSmartRoutingRules(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get smart routing rules: %w", err)
	}

	// 按优先级排序规则
	sortedRules := was.sortSmartRoutingRulesByPriority(rules)

	// 应用规则
	for _, rule := range sortedRules {
		if !rule.IsActive {
			continue
		}

		if was.matchesConditions(task, rule.Conditions) {
			// 执行路由动作
			err := was.executeRoutingAction(ctx, task, rule.RouteTo)
			if err != nil {
				was.logger.Errorw("Failed to execute routing action", "error", err, "rule_id", rule.ID)
				continue
			}

			was.logger.Infow("Task smart routed", "task_id", task.TaskID, "route_to", rule.RouteTo, "rule_id", rule.ID)
			return nil
		}
	}

	return nil
}

// CheckAutoEscalation 检查自动升级
func (was *WorkflowAutomationService) CheckAutoEscalation(ctx context.Context, tenantID int) error {
	was.logger.Infow("Checking auto escalation", "tenant_id", tenantID)

	// 查询待处理的任务
	tasks, err := was.client.WorkflowTask.Query().
		Where(workflowtask.TenantID(tenantID)).
		Where(workflowtask.StatusIn("pending", "in_progress")).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending tasks: %w", err)
	}

	if len(tasks) == 0 {
		was.logger.Infow("No pending tasks found for escalation check", "tenant_id", tenantID)
		return nil
	}

	// 获取自动升级规则
	rules, err := was.getAutoEscalationRules(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get auto escalation rules: %w", err)
	}

	// 按优先级排序规则
	sortedRules := was.sortAutoEscalationRulesByPriority(rules)

	// 已升级任务计数
	escalatedCount := 0

	// 检查每个任务是否需要升级
	for _, task := range tasks {
		for _, rule := range sortedRules {
			if !rule.IsActive {
				continue
			}

			if was.shouldEscalate(task, rule) {
				err := was.executeEscalation(ctx, task, rule)
				if err != nil {
					was.logger.Errorw("Failed to execute escalation", "error", err, "task_id", task.TaskID, "rule_id", rule.ID)
					continue
				}

				escalatedCount++
				was.logger.Infow("Task escalated", "task_id", task.TaskID, "escalate_to", rule.EscalateTo, "rule_id", rule.ID)
				break // 只应用第一个匹配的规则
			}
		}
	}

	was.logger.Infow("Auto escalation check completed", "escalated_count", escalatedCount, "tenant_id", tenantID)
	return nil
}

// 获取自动分配规则
func (was *WorkflowAutomationService) getAutoAssignmentRules(ctx context.Context, tenantID int) ([]AutoAssignmentRule, error) {
	// 模拟规则数据
	rules := []AutoAssignmentRule{
		{
			ID:          1,
			Name:        "高优先级任务分配给专家",
			Description: "将高优先级任务自动分配给专家用户",
			Conditions: map[string]interface{}{
				"priority": "high",
				"category": "technical",
			},
			Actions: map[string]interface{}{
				"assign_to": "expert",
				"method":    "round_robin",
			},
			Priority: 1,
			IsActive: true,
		},
		{
			ID:          2,
			Name:        "按技能分配",
			Description: "根据任务类型和用户技能进行分配",
			Conditions: map[string]interface{}{
				"type": "user_task",
			},
			Actions: map[string]interface{}{
				"assign_to": "skill_based",
				"method":    "least_busy",
			},
			Priority: 2,
			IsActive: true,
		},
	}

	return rules, nil
}

// 获取智能路由规则
func (was *WorkflowAutomationService) getSmartRoutingRules(ctx context.Context, tenantID int) ([]SmartRoutingRule, error) {
	// 模拟规则数据
	rules := []SmartRoutingRule{
		{
			ID:          1,
			Name:        "技术问题路由到技术组",
			Description: "技术相关问题自动路由到技术支持组",
			Conditions: map[string]interface{}{
				"category": "technical",
				"priority": "high",
			},
			RouteTo:  "tech_support",
			Priority: 1,
			IsActive: true,
		},
		{
			ID:          2,
			Name:        "财务问题路由到财务组",
			Description: "财务相关问题自动路由到财务组",
			Conditions: map[string]interface{}{
				"category": "finance",
			},
			RouteTo:  "finance_team",
			Priority: 2,
			IsActive: true,
		},
	}

	return rules, nil
}

// 获取自动升级规则
func (was *WorkflowAutomationService) getAutoEscalationRules(ctx context.Context, tenantID int) ([]AutoEscalationRule, error) {
	// 模拟规则数据
	rules := []AutoEscalationRule{
		{
			ID:          1,
			Name:        "高优先级任务4小时升级",
			Description: "高优先级任务4小时内未处理自动升级",
			Conditions: map[string]interface{}{
				"priority": "high",
			},
			EscalateTo: "manager",
			TimeLimit:  4 * time.Hour,
			Priority:   1,
			IsActive:   true,
		},
		{
			ID:          2,
			Name:        "普通任务24小时升级",
			Description: "普通任务24小时内未处理自动升级",
			Conditions: map[string]interface{}{
				"priority": "normal",
			},
			EscalateTo: "supervisor",
			TimeLimit:  24 * time.Hour,
			Priority:   2,
			IsActive:   true,
		},
	}

	return rules, nil
}

// 检查条件匹配
func (was *WorkflowAutomationService) matchesConditions(task *ent.WorkflowTask, conditions map[string]interface{}) bool {
	for key, value := range conditions {
		switch key {
		case "priority":
			if task.Priority != value {
				return false
			}
		case "type":
			if task.Type != value {
				return false
			}
		case "category":
			// 这里需要从任务变量中获取分类信息
			// 暂时返回true
			return true
		}
	}
	return true
}

// 执行分配动作
func (was *WorkflowAutomationService) executeAssignmentAction(task *ent.WorkflowTask, actions map[string]interface{}) (string, error) {
	method, ok := actions["method"].(string)
	if !ok {
		return "", fmt.Errorf("invalid assignment method")
	}

	switch method {
	case "round_robin":
		return was.roundRobinAssignment()
	case "least_busy":
		return was.leastBusyAssignment()
	case "skill_based":
		return was.skillBasedAssignment(task)
	default:
		return "", fmt.Errorf("unknown assignment method: %s", method)
	}
}

// 轮询分配
func (was *WorkflowAutomationService) roundRobinAssignment() (string, error) {
	// 模拟轮询分配
	users := []string{"user1", "user2", "user3", "user4"}
	// 这里应该实现真正的轮询逻辑
	return users[0], nil
}

// 最少忙碌分配
func (was *WorkflowAutomationService) leastBusyAssignment() (string, error) {
	// 模拟最少忙碌分配
	// 这里应该查询用户当前任务数量
	return "user2", nil
}

// 基于技能分配
func (was *WorkflowAutomationService) skillBasedAssignment(task *ent.WorkflowTask) (string, error) {
	// 模拟基于技能分配
	// 这里应该根据任务类型和用户技能进行匹配
	return "expert1", nil
}

// 执行路由动作
func (was *WorkflowAutomationService) executeRoutingAction(ctx context.Context, task *ent.WorkflowTask, routeTo string) error {
	// 更新任务的路由信息
	_, err := was.client.WorkflowTask.UpdateOne(task).
		SetCandidateGroups(routeTo).
		Save(ctx)
	return err
}

// 检查是否应该升级
func (was *WorkflowAutomationService) shouldEscalate(task *ent.WorkflowTask, rule AutoEscalationRule) bool {
	// 检查任务是否超过时间限制
	createdAt := task.CreatedAt
	timeLimit := rule.TimeLimit
	now := time.Now()

	return now.Sub(createdAt) > timeLimit
}

// 执行升级
func (was *WorkflowAutomationService) executeEscalation(ctx context.Context, task *ent.WorkflowTask, rule AutoEscalationRule) error {
	// 更新任务分配
	_, err := was.client.WorkflowTask.UpdateOne(task).
		SetAssignee(rule.EscalateTo).
		SetComment(fmt.Sprintf("自动升级到 %s", rule.EscalateTo)).
		Save(ctx)
	return err
}

// 应用默认分配策略
func (was *WorkflowAutomationService) applyDefaultAssignment(ctx context.Context, task *ent.WorkflowTask, tenantID int) error {
	// 默认分配策略：分配给当前最空闲的用户
	defaultAssignee := "default_user"

	_, err := was.client.WorkflowTask.UpdateOne(task).
		SetAssignee(defaultAssignee).
		SetStatus("pending").
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to apply default assignment: %w", err)
	}

	was.logger.Infow("Applied default assignment", "task_id", task.TaskID, "assignee", defaultAssignee)
	return nil
}

// 按优先级排序规则
func (was *WorkflowAutomationService) sortAutoAssignmentRulesByPriority(rules []AutoAssignmentRule) []AutoAssignmentRule {
	// 使用插入排序按优先级排序
	for i := 1; i < len(rules); i++ {
		key := rules[i]
		j := i - 1
		for j >= 0 && rules[j].Priority > key.Priority {
			rules[j+1] = rules[j]
			j--
		}
		rules[j+1] = key
	}
	return rules
}

func (was *WorkflowAutomationService) sortSmartRoutingRulesByPriority(rules []SmartRoutingRule) []SmartRoutingRule {
	// 使用插入排序按优先级排序
	for i := 1; i < len(rules); i++ {
		key := rules[i]
		j := i - 1
		for j >= 0 && rules[j].Priority > key.Priority {
			rules[j+1] = rules[j]
			j--
		}
		rules[j+1] = key
	}
	return rules
}

func (was *WorkflowAutomationService) sortAutoEscalationRulesByPriority(rules []AutoEscalationRule) []AutoEscalationRule {
	// 使用插入排序按优先级排序
	for i := 1; i < len(rules); i++ {
		key := rules[i]
		j := i - 1
		for j >= 0 && rules[j].Priority > key.Priority {
			rules[j+1] = rules[j]
			j--
		}
		rules[j+1] = key
	}
	return rules
}

// 启动自动升级检查定时器
func (was *WorkflowAutomationService) StartAutoEscalationTimer(ctx context.Context, tenantID int) {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟检查一次
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := was.CheckAutoEscalation(ctx, tenantID); err != nil {
					was.logger.Errorw("Auto escalation check failed", "error", err, "tenant_id", tenantID)
				}
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// GetEscalationCandidates 获取可升级的任务候选人
func (was *WorkflowAutomationService) GetEscalationCandidates(ctx context.Context, tenantID int) ([]*ent.WorkflowTask, error) {
	// 获取所有待处理和进行中的任务
	tasks, err := was.client.WorkflowTask.Query().
		Where(workflowtask.TenantID(tenantID)).
		Where(workflowtask.StatusIn("pending", "in_progress")).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// 获取升级规则
	rules, err := was.getAutoEscalationRules(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get escalation rules: %w", err)
	}

	// 过滤出需要升级的任务
	var candidates []*ent.WorkflowTask
	for _, task := range tasks {
		for _, rule := range rules {
			if rule.IsActive && was.shouldEscalate(task, rule) {
				candidates = append(candidates, task)
				break
			}
		}
	}

	return candidates, nil
}

// ManualEscalate 手动触发升级
func (was *WorkflowAutomationService) ManualEscalate(ctx context.Context, taskID string, escalateTo string, reason string, tenantID int) error {
	was.logger.Infow("Manual escalation triggered", "task_id", taskID, "escalate_to", escalateTo, "tenant_id", tenantID)

	// 查找任务
	task, err := was.client.WorkflowTask.Query().
		Where(workflowtask.TaskIDEQ(taskID)).
		Where(workflowtask.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("task not found: %s", taskID)
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 更新任务分配
	_, err = was.client.WorkflowTask.UpdateOne(task).
		SetAssignee(escalateTo).
		SetComment(fmt.Sprintf("手动升级到 %s: %s", escalateTo, reason)).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	was.logger.Infow("Manual escalation completed", "task_id", taskID, "escalate_to", escalateTo)
	return nil
}
