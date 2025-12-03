package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/ticketautomationrule"
	"strings"
	"time"

	"go.uber.org/zap"
)

type TicketAutomationRuleService struct {
	client              *ent.Client
	logger              *zap.SugaredLogger
	ticketService       *TicketService
	assignmentService   *TicketAssignmentService
	notificationService *TicketNotificationService
}

func NewTicketAutomationRuleService(
	client *ent.Client,
	logger *zap.SugaredLogger,
) *TicketAutomationRuleService {
	return &TicketAutomationRuleService{
		client: client,
		logger: logger,
	}
}

// SetTicketService 设置工单服务（用于依赖注入）
func (s *TicketAutomationRuleService) SetTicketService(ticketService *TicketService) {
	s.ticketService = ticketService
}

// SetAssignmentService 设置分配服务（用于依赖注入）
func (s *TicketAutomationRuleService) SetAssignmentService(assignmentService *TicketAssignmentService) {
	s.assignmentService = assignmentService
}

// SetNotificationService 设置通知服务（用于依赖注入）
func (s *TicketAutomationRuleService) SetNotificationService(notificationService *TicketNotificationService) {
	s.notificationService = notificationService
}

// ListAutomationRules 获取自动化规则列表
func (s *TicketAutomationRuleService) ListAutomationRules(
	ctx context.Context,
	tenantID int,
) ([]*dto.AutomationRuleResponse, error) {
	rules, err := s.client.TicketAutomationRule.Query().
		Where(ticketautomationrule.TenantID(tenantID)).
		Order(ent.Desc(ticketautomationrule.FieldPriority)).
		Order(ent.Desc(ticketautomationrule.FieldCreatedAt)).
		WithCreator().
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list automation rules", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to list automation rules: %w", err)
	}

	responses := make([]*dto.AutomationRuleResponse, 0, len(rules))
	for _, rule := range rules {
		var creator *ent.User
		if rule.Edges.Creator != nil {
			creator = rule.Edges.Creator
		} else {
			creator, _ = s.client.User.Get(ctx, rule.CreatedBy)
		}
		responses = append(responses, dto.ToAutomationRuleResponse(rule, creator))
	}

	return responses, nil
}

// GetAutomationRule 获取自动化规则详情
func (s *TicketAutomationRuleService) GetAutomationRule(
	ctx context.Context,
	ruleID, tenantID int,
) (*dto.AutomationRuleResponse, error) {
	rule, err := s.client.TicketAutomationRule.Query().
		Where(
			ticketautomationrule.ID(ruleID),
			ticketautomationrule.TenantID(tenantID),
		).
		WithCreator().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("automation rule not found: %w", err)
	}

	var creator *ent.User
	if rule.Edges.Creator != nil {
		creator = rule.Edges.Creator
	} else {
		creator, _ = s.client.User.Get(ctx, rule.CreatedBy)
	}

	return dto.ToAutomationRuleResponse(rule, creator), nil
}

// CreateAutomationRule 创建自动化规则
func (s *TicketAutomationRuleService) CreateAutomationRule(
	ctx context.Context,
	req *dto.CreateAutomationRuleRequest,
	userID, tenantID int,
) (*dto.AutomationRuleResponse, error) {
	s.logger.Infow("Creating automation rule", "name", req.Name, "user_id", userID, "tenant_id", tenantID)

	rule, err := s.client.TicketAutomationRule.Create().
		SetName(req.Name).
		SetNillableDescription(req.Description).
		SetPriority(req.Priority).
		SetConditions(req.Conditions).
		SetActions(req.Actions).
		SetIsActive(req.IsActive).
		SetCreatedBy(userID).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create automation rule", "error", err)
		return nil, fmt.Errorf("failed to create automation rule: %w", err)
	}

	creator, _ := s.client.User.Get(ctx, userID)
	return dto.ToAutomationRuleResponse(rule, creator), nil
}

// UpdateAutomationRule 更新自动化规则
func (s *TicketAutomationRuleService) UpdateAutomationRule(
	ctx context.Context,
	ruleID int,
	req *dto.UpdateAutomationRuleRequest,
	tenantID int,
) (*dto.AutomationRuleResponse, error) {
	s.logger.Infow("Updating automation rule", "rule_id", ruleID, "tenant_id", tenantID)

	updateQuery := s.client.TicketAutomationRule.UpdateOneID(ruleID).
		Where(ticketautomationrule.TenantID(tenantID))

	if req.Name != nil {
		updateQuery.SetName(*req.Name)
	}
	if req.Description != nil {
		updateQuery.SetNillableDescription(req.Description)
	}
	if req.Priority != nil {
		updateQuery.SetPriority(*req.Priority)
	}
	if req.Conditions != nil {
		updateQuery.SetConditions(req.Conditions)
	}
	if req.Actions != nil {
		updateQuery.SetActions(req.Actions)
	}
	if req.IsActive != nil {
		updateQuery.SetIsActive(*req.IsActive)
	}

	updateQuery.SetUpdatedAt(time.Now())

	updatedRule, err := updateQuery.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update automation rule", "error", err)
		return nil, fmt.Errorf("failed to update automation rule: %w", err)
	}

	creator, _ := s.client.User.Get(ctx, updatedRule.CreatedBy)
	return dto.ToAutomationRuleResponse(updatedRule, creator), nil
}

// DeleteAutomationRule 删除自动化规则
func (s *TicketAutomationRuleService) DeleteAutomationRule(
	ctx context.Context,
	ruleID, tenantID int,
) error {
	s.logger.Infow("Deleting automation rule", "rule_id", ruleID, "tenant_id", tenantID)

	err := s.client.TicketAutomationRule.DeleteOneID(ruleID).
		Where(ticketautomationrule.TenantID(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete automation rule", "error", err)
		return fmt.Errorf("failed to delete automation rule: %w", err)
	}

	return nil
}

// TestAutomationRule 测试自动化规则
func (s *TicketAutomationRuleService) TestAutomationRule(
	ctx context.Context,
	req *dto.TestAutomationRuleRequest,
	tenantID int,
) (*dto.TestAutomationRuleResponse, error) {
	s.logger.Infow("Testing automation rule", "rule_id", req.RuleID, "ticket_id", req.TicketID)

	// 获取规则
	rule, err := s.client.TicketAutomationRule.Query().
		Where(
			ticketautomationrule.ID(req.RuleID),
			ticketautomationrule.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("automation rule not found: %w", err)
	}

	// 获取工单
	ticketEntity, err := s.client.Ticket.Query().
		Where(
			ticket.ID(req.TicketID),
			ticket.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	// 评估条件
	matched, reason := s.evaluateConditions(ctx, rule.Conditions, ticketEntity)
	if !matched {
		return &dto.TestAutomationRuleResponse{
			Matched: false,
			Reason:  reason,
		}, nil
	}

	// 如果匹配，返回将执行的动作
	actionDescriptions := s.getActionDescriptions(rule.Actions)
	return &dto.TestAutomationRuleResponse{
		Matched: true,
		Actions: actionDescriptions,
		Reason:  "规则条件匹配，将执行以下动作",
	}, nil
}

// ExecuteRulesForTicket 为工单执行所有适用的规则
func (s *TicketAutomationRuleService) ExecuteRulesForTicket(
	ctx context.Context,
	ticketID, tenantID int,
) error {
	s.logger.Infow("Executing automation rules for ticket", "ticket_id", ticketID, "tenant_id", tenantID)

	// 获取工单
	ticketEntity, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("ticket not found: %w", err)
	}

	if ticketEntity.TenantID != tenantID {
		return fmt.Errorf("ticket not found")
	}

	// 获取所有激活的规则，按优先级排序
	rules, err := s.client.TicketAutomationRule.Query().
		Where(
			ticketautomationrule.TenantID(tenantID),
			ticketautomationrule.IsActive(true),
		).
		Order(ent.Desc(ticketautomationrule.FieldPriority)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get automation rules", "error", err)
		return fmt.Errorf("failed to get automation rules: %w", err)
	}

	// 执行每个规则
	for _, rule := range rules {
		matched, _ := s.evaluateConditions(ctx, rule.Conditions, ticketEntity)
		if matched {
			if err := s.executeActions(ctx, rule, ticketEntity); err != nil {
				s.logger.Warnw("Failed to execute rule actions", "error", err, "rule_id", rule.ID)
				// 继续执行其他规则，不中断
				continue
			}

			// 更新规则执行统计
			now := time.Now()
			_, err = s.client.TicketAutomationRule.UpdateOneID(rule.ID).
				AddExecutionCount(1).
				SetLastExecutedAt(now).
				Save(ctx)
			if err != nil {
				s.logger.Warnw("Failed to update rule execution count", "error", err)
			}

			// 记录执行日志
			s.logger.Infow("Automation rule executed", "rule_id", rule.ID, "rule_name", rule.Name, "ticket_id", ticketID)
		}
	}

	return nil
}

// evaluateConditions 评估规则条件
func (s *TicketAutomationRuleService) evaluateConditions(
	ctx context.Context,
	conditions []map[string]interface{},
	ticketEntity *ent.Ticket,
) (bool, string) {
	if len(conditions) == 0 {
		return true, "无条件（总是匹配）"
	}

	for _, condition := range conditions {
		field, ok := condition["field"].(string)
		if !ok {
			continue
		}

		operator, _ := condition["operator"].(string)
		value := condition["value"]

		matched := s.evaluateCondition(field, operator, value, ticketEntity)
		if !matched {
			return false, fmt.Sprintf("条件不匹配: %s %s %v", field, operator, value)
		}
	}

	return true, "所有条件匹配"
}

// evaluateCondition 评估单个条件
func (s *TicketAutomationRuleService) evaluateCondition(
	field, operator string,
	value interface{},
	ticketEntity *ent.Ticket,
) bool {
	var fieldValue interface{}

	switch field {
	case "status":
		fieldValue = ticketEntity.Status
	case "priority":
		fieldValue = ticketEntity.Priority
	case "category_id":
		fieldValue = ticketEntity.CategoryID
	case "department_id":
		fieldValue = ticketEntity.DepartmentID
	case "requester_id":
		fieldValue = ticketEntity.RequesterID
	case "assignee_id":
		fieldValue = ticketEntity.AssigneeID
	default:
		return false
	}

	return s.compareValues(fieldValue, operator, value)
}

// compareValues 比较值
func (s *TicketAutomationRuleService) compareValues(fieldValue interface{}, operator string, conditionValue interface{}) bool {
	switch operator {
	case "equals":
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", conditionValue)
	case "not_equals":
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", conditionValue)
	case "contains":
		fieldStr := fmt.Sprintf("%v", fieldValue)
		valueStr := fmt.Sprintf("%v", conditionValue)
		return strings.Contains(fieldStr, valueStr)
	case "in":
		if arr, ok := conditionValue.([]interface{}); ok {
			for _, v := range arr {
				if fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", v) {
					return true
				}
			}
		}
		return false
	case "not_in":
		if arr, ok := conditionValue.([]interface{}); ok {
			for _, v := range arr {
				if fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", v) {
					return false
				}
			}
		}
		return true
	case "greater_than":
		return s.compareNumbersAuto(fieldValue, conditionValue) > 0
	case "less_than":
		return s.compareNumbersAuto(fieldValue, conditionValue) < 0
	default:
		return false
	}
}

// executeActions 执行规则动作
func (s *TicketAutomationRuleService) executeActions(
	ctx context.Context,
	rule *ent.TicketAutomationRule,
	ticketEntity *ent.Ticket,
) error {
	for _, action := range rule.Actions {
		actionType, ok := action["type"].(string)
		if !ok {
			continue
		}

		switch actionType {
		case "set_category":
			if categoryID, ok := action["category_id"].(float64); ok {
				_, err := s.client.Ticket.UpdateOneID(ticketEntity.ID).
					SetCategoryID(int(categoryID)).
					Save(ctx)
				if err != nil {
					s.logger.Warnw("Failed to set category", "error", err)
				}
			}

		case "set_priority":
			if priority, ok := action["priority"].(string); ok {
				_, err := s.client.Ticket.UpdateOneID(ticketEntity.ID).
					SetPriority(priority).
					Save(ctx)
				if err != nil {
					s.logger.Warnw("Failed to set priority", "error", err)
				}
			}

		case "assign":
			if userID, ok := action["user_id"].(float64); ok {
				_, err := s.client.Ticket.UpdateOneID(ticketEntity.ID).
					SetAssigneeID(int(userID)).
					Save(ctx)
				if err != nil {
					s.logger.Warnw("Failed to assign ticket", "error", err)
				}
			}

		case "auto_assign":
			if s.assignmentService != nil {
				var categoryID *int
				if ticketEntity.CategoryID > 0 {
					categoryID = &ticketEntity.CategoryID
				}
				_, err := s.assignmentService.AssignTicket(ctx, &AssignmentRequest{
					TicketID:   ticketEntity.ID,
					CategoryID: categoryID,
					Priority:   ticketEntity.Priority,
					TenantID:   ticketEntity.TenantID,
					AutoAssign: true,
				})
				if err != nil {
					s.logger.Warnw("Failed to auto assign ticket", "error", err)
				}
			}

		case "escalate":
			// 升级优先级
			priorityMap := map[string]string{
				"low":    "medium",
				"medium": "high",
				"high":   "urgent",
				"urgent": "urgent",
			}
			if newPriority, ok := priorityMap[ticketEntity.Priority]; ok {
				_, err := s.client.Ticket.UpdateOneID(ticketEntity.ID).
					SetPriority(newPriority).
					Save(ctx)
				if err != nil {
					s.logger.Warnw("Failed to escalate ticket", "error", err)
				}
			}

		case "send_notification":
			if s.notificationService != nil {
				content, _ := action["content"].(string)
				userIDs := []int{ticketEntity.RequesterID}
				if ticketEntity.AssigneeID > 0 {
					userIDs = append(userIDs, ticketEntity.AssigneeID)
				}
				_ = s.notificationService.SendNotification(ctx, ticketEntity.ID, &dto.SendTicketNotificationRequest{
					UserIDs: userIDs,
					Type:    "automation",
					Channel: "in_app",
					Content: content,
				}, ticketEntity.TenantID)
			}

		case "set_status":
			if status, ok := action["status"].(string); ok {
				_, err := s.client.Ticket.UpdateOneID(ticketEntity.ID).
					SetStatus(status).
					Save(ctx)
				if err != nil {
					s.logger.Warnw("Failed to set status", "error", err)
				}
			}
		}
	}

	return nil
}

// getActionDescriptions 获取动作描述
func (s *TicketAutomationRuleService) getActionDescriptions(actions []map[string]interface{}) []string {
	descriptions := make([]string, 0, len(actions))
	for _, action := range actions {
		actionType, _ := action["type"].(string)
		switch actionType {
		case "set_category":
			descriptions = append(descriptions, "设置分类")
		case "set_priority":
			descriptions = append(descriptions, "设置优先级")
		case "assign":
			descriptions = append(descriptions, "分配给用户")
		case "auto_assign":
			descriptions = append(descriptions, "自动分配")
		case "escalate":
			descriptions = append(descriptions, "升级优先级")
		case "send_notification":
			descriptions = append(descriptions, "发送通知")
		case "set_status":
			descriptions = append(descriptions, "设置状态")
		default:
			descriptions = append(descriptions, fmt.Sprintf("执行动作: %s", actionType))
		}
	}
	return descriptions
}

// compareNumbersAuto 比较数字
func (s *TicketAutomationRuleService) compareNumbersAuto(a, b interface{}) int {
	af, ok1 := s.toFloat64Auto(a)
	bf, ok2 := s.toFloat64Auto(b)
	if !ok1 || !ok2 {
		return 0
	}
	if af > bf {
		return 1
	}
	if af < bf {
		return -1
	}
	return 0
}

// toFloat64Auto 转换为float64
func (s *TicketAutomationRuleService) toFloat64Auto(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}
