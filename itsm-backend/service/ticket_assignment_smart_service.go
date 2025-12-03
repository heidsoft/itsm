package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticketassignmentrule"
	"sort"
	"time"

	"go.uber.org/zap"
)

type TicketAssignmentSmartService struct {
	client              *ent.Client
	logger              *zap.SugaredLogger
	assignmentService   *TicketAssignmentService
	ruleService         *TicketAssignmentRuleService
}

func NewTicketAssignmentSmartService(
	client *ent.Client,
	logger *zap.SugaredLogger,
	assignmentService *TicketAssignmentService,
	ruleService *TicketAssignmentRuleService,
) *TicketAssignmentSmartService {
	return &TicketAssignmentSmartService{
		client:            client,
		logger:            logger,
		assignmentService: assignmentService,
		ruleService:       ruleService,
	}
}

// AutoAssign 自动分配工单
func (s *TicketAssignmentSmartService) AutoAssign(
	ctx context.Context,
	ticketID, tenantID int,
) (*dto.AutoAssignResponse, error) {
	s.logger.Infow("Auto assigning ticket", "ticket_id", ticketID, "tenant_id", tenantID)

	// 获取工单信息
	ticketEntity, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	if ticketEntity.TenantID != tenantID {
		return nil, fmt.Errorf("ticket not found")
	}

	// 1. 先尝试基于规则分配
	ruleAssigned, err := s.tryRuleBasedAssignment(ctx, ticketEntity, tenantID)
	if err == nil && ruleAssigned != nil {
		return ruleAssigned, nil
	}

	// 2. 如果没有匹配的规则，使用智能分配
	var categoryID *int
	if ticketEntity.CategoryID > 0 {
		categoryID = &ticketEntity.CategoryID
	}
	req := &AssignmentRequest{
		TicketID:   ticketID,
		CategoryID: categoryID,
		Priority:   ticketEntity.Priority,
		TenantID:   tenantID,
		AutoAssign: true,
	}

	assignment, err := s.assignmentService.AssignTicket(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to auto assign: %w", err)
	}

	return &dto.AutoAssignResponse{
		TicketID:       assignment.TicketID,
		AssignedTo:     assignment.AssignedTo,
		AssignmentType: assignment.AssignmentType,
		Reason:         assignment.Reason,
		Score:          assignment.Score,
	}, nil
}

// GetAssignRecommendations 获取分配推荐
func (s *TicketAssignmentSmartService) GetAssignRecommendations(
	ctx context.Context,
	ticketID, tenantID int,
) ([]*dto.AssignmentRecommendation, error) {
	s.logger.Infow("Getting assignment recommendations", "ticket_id", ticketID, "tenant_id", tenantID)

	// 获取工单信息
	ticketEntity, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	if ticketEntity.TenantID != tenantID {
		return nil, fmt.Errorf("ticket not found")
	}

	// 构建分配请求
	var categoryID *int
	if ticketEntity.CategoryID > 0 {
		categoryID = &ticketEntity.CategoryID
	}
	req := &AssignmentRequest{
		TicketID:   ticketID,
		CategoryID: categoryID,
		Priority:   ticketEntity.Priority,
		TenantID:   tenantID,
		AutoAssign: false, // 不实际分配，只获取推荐
	}

	// 获取可用用户
	availableUsers, err := s.assignmentService.getAvailableUsers(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get available users: %w", err)
	}

	// 计算每个用户的评分
	for i := range availableUsers {
		availableUsers[i].Score = s.assignmentService.calculateUserScore(ctx, &availableUsers[i], req)
	}

	// 按评分排序
	sort.Slice(availableUsers, func(i, j int) bool {
		return availableUsers[i].Score > availableUsers[j].Score
	})

	// 转换为推荐响应
	recommendations := make([]*dto.AssignmentRecommendation, 0, len(availableUsers))
	for _, user := range availableUsers {
		// 获取用户详细信息
		userEntity, err := s.client.User.Get(ctx, user.UserID)
		if err != nil {
			continue
		}

		// 生成推荐理由
		reason := s.generateRecommendationReason(&user, req)

		recommendations = append(recommendations, &dto.AssignmentRecommendation{
			UserID:     user.UserID,
			Username:   userEntity.Username,
			Name:       userEntity.Name,
			Email:      userEntity.Email,
			Score:      user.Score,
			Reason:     reason,
			Workload:   user.ActiveTickets,
			Skills:     user.Skills,
			Categories: user.Categories,
		})
	}

	return recommendations, nil
}

// tryRuleBasedAssignment 尝试基于规则分配
func (s *TicketAssignmentSmartService) tryRuleBasedAssignment(
	ctx context.Context,
	ticketEntity *ent.Ticket,
	tenantID int,
) (*dto.AutoAssignResponse, error) {
	// 获取所有激活的规则，按优先级排序
	rules, err := s.client.TicketAssignmentRule.Query().
		Where(
			ticketassignmentrule.TenantID(tenantID),
			ticketassignmentrule.IsActive(true),
		).
		Order(ent.Desc(ticketassignmentrule.FieldPriority)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// 遍历规则，找到第一个匹配的
	for _, rule := range rules {
		matched, _ := s.ruleService.matchRule(ctx, rule, ticketEntity)
		if matched {
			// 执行规则动作
			assignedTo, score, err := s.ruleService.executeRuleAction(ctx, rule, ticketEntity)
			if err != nil {
				s.logger.Warnw("Failed to execute rule action", "error", err, "rule_id", rule.ID)
				continue
			}

			// 更新工单
			_, err = s.client.Ticket.UpdateOneID(ticketEntity.ID).
				SetAssigneeID(*assignedTo).
				Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to assign ticket: %w", err)
			}

			// 更新规则执行统计
			now := time.Now()
			_, err = s.client.TicketAssignmentRule.UpdateOneID(rule.ID).
				AddExecutionCount(1).
				SetLastExecutedAt(now).
				Save(ctx)
			if err != nil {
				s.logger.Warnw("Failed to update rule execution count", "error", err)
			}

			return &dto.AutoAssignResponse{
				TicketID:       ticketEntity.ID,
				AssignedTo:     assignedTo,
				AssignmentType: "rule",
				Reason:         fmt.Sprintf("基于规则 '%s' 自动分配", rule.Name),
				Score:          score,
			}, nil
		}
	}

	return nil, fmt.Errorf("no matching rule found")
}

// generateRecommendationReason 生成推荐理由
func (s *TicketAssignmentSmartService) generateRecommendationReason(
	user *UserWorkload,
	req *AssignmentRequest,
) string {
	reasons := []string{}

	if len(user.Skills) > 0 {
		reasons = append(reasons, fmt.Sprintf("具备相关技能（%d项）", len(user.Skills)))
	}

	if user.ActiveTickets == 0 {
		reasons = append(reasons, "当前无活跃工单")
	} else if user.ActiveTickets < 3 {
		reasons = append(reasons, fmt.Sprintf("工作负载较轻（%d个活跃工单）", user.ActiveTickets))
	}

	if len(user.Categories) > 0 && req.CategoryID != nil {
		for _, catID := range user.Categories {
			if catID == *req.CategoryID {
				reasons = append(reasons, "有该分类的处理经验")
				break
			}
		}
	}

	if user.AvgResolution > 0 && user.AvgResolution < 24*time.Hour {
		reasons = append(reasons, "平均解决时间较短")
	}

	if len(reasons) == 0 {
		return "符合基本分配条件"
	}

	return fmt.Sprintf("推荐理由: %s", joinStrings(reasons, "，"))
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

