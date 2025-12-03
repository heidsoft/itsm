package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"strings"
	"time"

	"go.uber.org/zap"
)

type TicketService struct {
	client              *ent.Client
	logger              *zap.SugaredLogger
	notificationService *TicketNotificationService // 可选的通知服务
	automationRuleService *TicketAutomationRuleService // 可选的自动化规则服务
}

func NewTicketService(client *ent.Client, logger *zap.SugaredLogger) *TicketService {
	return &TicketService{
		client: client,
		logger: logger,
	}
}

// SetNotificationService 设置通知服务（用于依赖注入）
func (s *TicketService) SetNotificationService(notificationService *TicketNotificationService) {
	s.notificationService = notificationService
}

// SetAutomationRuleService 设置自动化规则服务（用于依赖注入）
func (s *TicketService) SetAutomationRuleService(automationRuleService *TicketAutomationRuleService) {
	s.automationRuleService = automationRuleService
}

// CreateTicket 创建工单
func (s *TicketService) CreateTicket(ctx context.Context, req *dto.CreateTicketRequest, tenantID int) (*ent.Ticket, error) {
	s.logger.Infow("Creating ticket", "tenant_id", tenantID, "title", req.Title)

	ticket, err := s.client.Ticket.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetPriority(req.Priority).
		SetStatus("submitted").
		SetTenantID(tenantID).
		SetRequesterID(req.RequesterID).
		SetAssigneeID(req.AssigneeID).
		Save(ctx)

	if err != nil {
		s.logger.Errorw("Failed to create ticket", "error", err)
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	// 记录审计日志
	s.logAuditEvent(ctx, "ticket_created", ticket.ID, tenantID, map[string]interface{}{
		"title":    req.Title,
		"priority": req.Priority,
		"category": req.Category,
	})

	// 发送通知给处理人
	if s.notificationService != nil && ticket.AssigneeID > 0 {
		if err := s.notificationService.NotifyTicketCreated(ctx, ticket); err != nil {
			s.logger.Warnw("Failed to send ticket created notification", "error", err)
		}
	}

	// 执行自动化规则（异步执行，不阻塞工单创建）
	if s.automationRuleService != nil {
		go func() {
			// 使用新的context避免超时影响
			ruleCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.automationRuleService.ExecuteRulesForTicket(ruleCtx, ticket.ID, tenantID); err != nil {
				s.logger.Warnw("Failed to execute automation rules", "error", err, "ticket_id", ticket.ID)
			}
		}()
	}

	return ticket, nil
}

// UpdateTicket 更新工单
func (s *TicketService) UpdateTicket(ctx context.Context, ticketID int, req *dto.UpdateTicketRequest, tenantID int) (*ent.Ticket, error) {
	s.logger.Infow("Updating ticket", "ticket_id", ticketID, "tenant_id", tenantID)

	// 检查工单是否存在且属于当前租户
	existingTicket, err := s.client.Ticket.Query().
		Where(ticket.ID(ticketID), ticket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	// 检查权限
	if !s.canUpdateTicket(existingTicket, req.UserID) {
		return nil, fmt.Errorf("insufficient permissions")
	}

	updateQuery := s.client.Ticket.UpdateOne(existingTicket)

	if req.Title != "" {
		updateQuery.SetTitle(req.Title)
	}
	if req.Description != "" {
		updateQuery.SetDescription(req.Description)
	}
	if req.Priority != "" {
		updateQuery.SetPriority(req.Priority)
	}
	if req.Status != "" {
		updateQuery.SetStatus(req.Status)
	}
	if req.AssigneeID != 0 {
		updateQuery.SetAssigneeID(req.AssigneeID)
	}

	ticket, err := updateQuery.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update ticket", "error", err)
		return nil, fmt.Errorf("failed to update ticket: %w", err)
	}

	// 执行自动化规则（异步执行，不阻塞工单更新）
	if s.automationRuleService != nil {
		go func() {
			// 使用新的context避免超时影响
			ruleCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.automationRuleService.ExecuteRulesForTicket(ruleCtx, ticket.ID, tenantID); err != nil {
				s.logger.Warnw("Failed to execute automation rules", "error", err, "ticket_id", ticket.ID)
			}
		}()
	}

	// 记录审计日志
	s.logAuditEvent(ctx, "ticket_updated", ticketID, tenantID, map[string]interface{}{
		"updated_fields": req,
	})

	return ticket, nil
}

// ListTickets 获取工单列表（支持高级查询）
func (s *TicketService) ListTickets(ctx context.Context, req *dto.ListTicketsRequest, tenantID int) (*dto.ListTicketsResponse, error) {
	s.logger.Infow("Listing tickets", "tenant_id", tenantID, "filters", req)

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	query := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID))

	// 应用筛选条件
	if req.Status != "" {
		query.Where(ticket.StatusEQ(req.Status))
	}
	if req.Priority != "" {
		query.Where(ticket.PriorityEQ(req.Priority))
	}
	if req.AssigneeID != 0 {
		query.Where(ticket.AssigneeID(req.AssigneeID))
	}
	if req.RequesterID != 0 {
		query.Where(ticket.RequesterID(req.RequesterID))
	}

	// 关键词搜索
	if req.Keyword != "" {
		query.Where(ticket.Or(
			ticket.TitleContains(req.Keyword),
			ticket.DescriptionContains(req.Keyword),
		))
	}

	// 日期范围筛选
	if req.DateFrom != nil {
		query.Where(ticket.CreatedAtGTE(*req.DateFrom))
	}
	if req.DateTo != nil {
		query.Where(ticket.CreatedAtLTE(*req.DateTo))
	}

	// 排序
	sortField := ticket.FieldCreatedAt
	if req.SortBy != "" {
		switch req.SortBy {
		case "title":
			sortField = ticket.FieldTitle
		case "priority":
			sortField = ticket.FieldPriority
		case "status":
			sortField = ticket.FieldStatus
		case "updated_at":
			sortField = ticket.FieldUpdatedAt
		}
	}

	if req.SortOrder == "asc" {
		query.Order(ent.Asc(sortField))
	} else {
		query.Order(ent.Desc(sortField))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count tickets", "error", err)
		return nil, fmt.Errorf("failed to count tickets: %w", err)
	}

	// 分页查询
	tickets, err := query.
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list tickets", "error", err)
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}

	return &dto.ListTicketsResponse{
		Tickets:  tickets,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetTicket 获取工单详情
func (s *TicketService) GetTicket(ctx context.Context, ticketID int, tenantID int) (*ent.Ticket, error) {
	s.logger.Infow("Getting ticket", "ticket_id", ticketID, "tenant_id", tenantID)

	ticket, err := s.client.Ticket.Query().
		Where(ticket.ID(ticketID), ticket.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("ticket not found")
		}
		s.logger.Errorw("Failed to get ticket", "error", err)
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return ticket, nil
}

// DeleteTicket 删除工单
func (s *TicketService) DeleteTicket(ctx context.Context, ticketID int, tenantID int) error {
	s.logger.Infow("Deleting ticket", "ticket_id", ticketID, "tenant_id", tenantID)

	// 检查工单是否存在且属于当前租户
	exists, err := s.client.Ticket.Query().
		Where(ticket.ID(ticketID), ticket.TenantID(tenantID)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check ticket existence", "error", err)
		return fmt.Errorf("failed to check ticket existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("ticket not found")
	}

	// 删除工单
	err = s.client.Ticket.DeleteOneID(ticketID).Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete ticket", "error", err)
		return fmt.Errorf("failed to delete ticket: %w", err)
	}

	// 记录审计日志
	s.logAuditEvent(ctx, "ticket_deleted", ticketID, tenantID, nil)

	return nil
}

// BatchDeleteTickets 批量删除工单
func (s *TicketService) BatchDeleteTickets(ctx context.Context, ticketIDs []int, tenantID int) error {
	s.logger.Infow("Batch deleting tickets", "ticket_ids", ticketIDs, "tenant_id", tenantID)

	// 验证所有工单都属于当前租户
	count, err := s.client.Ticket.Query().
		Where(ticket.IDIn(ticketIDs...), ticket.TenantID(tenantID)).
		Count(ctx)

	if err != nil {
		return fmt.Errorf("failed to validate tickets: %w", err)
	}

	if count != len(ticketIDs) {
		return fmt.Errorf("some tickets not found or not accessible")
	}

	// 批量硬删除
	_, err = s.client.Ticket.Delete().Where(ticket.IDIn(ticketIDs...)).Exec(ctx)

	if err != nil {
		s.logger.Errorw("Failed to batch delete tickets", "error", err)
		return fmt.Errorf("failed to batch delete tickets: %w", err)
	}

	// 记录审计日志
	s.logAuditEvent(ctx, "tickets_batch_deleted", 0, tenantID, map[string]interface{}{
		"ticket_ids": ticketIDs,
		"count":      len(ticketIDs),
	})

	return nil
}

// GetTicketStats 获取工单统计信息
func (s *TicketService) GetTicketStats(ctx context.Context, tenantID int) (*dto.TicketStatsResponse, error) {
	s.logger.Infow("Getting ticket stats", "tenant_id", tenantID)

	// 获取总工单数
	totalTickets, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID)).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count total tickets", "error", err)
		return nil, fmt.Errorf("failed to count total tickets: %w", err)
	}

	// 获取已提交工单数
	submittedTickets, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.StatusEQ("submitted")).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count submitted tickets", "error", err)
		return nil, fmt.Errorf("failed to count submitted tickets: %w", err)
	}

	// 获取处理中工单数
	inProgressTickets, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.StatusEQ("in_progress")).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count in-progress tickets", "error", err)
		return nil, fmt.Errorf("failed to count in-progress tickets: %w", err)
	}

	// 获取已关闭工单数
	closedTickets, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.StatusEQ("closed")).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count closed tickets", "error", err)
		return nil, fmt.Errorf("failed to count closed tickets: %w", err)
	}

	// 获取高优先级工单数
	highPriorityTickets, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.PriorityIn("high", "critical")).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count high priority tickets", "error", err)
		return nil, fmt.Errorf("failed to count high priority tickets: %w", err)
	}

	return &dto.TicketStatsResponse{
		Total:        totalTickets,
		Open:         submittedTickets,
		InProgress:   inProgressTickets,
		Resolved:     closedTickets,
		HighPriority: highPriorityTickets,
		Overdue:      0, // 暂时设为0，因为需要SLA计算
	}, nil
}

// calculateSLADeadline 计算SLA截止时间
func (s *TicketService) calculateSLADeadline(priority, category string) (time.Time, error) {
	now := time.Now()

	// 根据优先级和分类计算SLA时间
	var slaHours int
	switch priority {
	case "critical":
		slaHours = 2
	case "high":
		slaHours = 4
	case "medium":
		slaHours = 8
	case "low":
		slaHours = 24
	default:
		slaHours = 8
	}

	// 考虑工作时间（简单实现，实际应该更复杂）
	deadline := now.Add(time.Duration(slaHours) * time.Hour)

	// 如果是周末，延后到周一
	if deadline.Weekday() == time.Saturday {
		deadline = deadline.AddDate(0, 0, 2)
	} else if deadline.Weekday() == time.Sunday {
		deadline = deadline.AddDate(0, 0, 1)
	}

	return deadline, nil
}

// canUpdateTicket 检查是否可以更新工单
func (s *TicketService) canUpdateTicket(ticket *ent.Ticket, userID int) bool {
	// 简化权限检查：工单创建者或处理人可以更新
	return ticket.RequesterID == userID || ticket.AssigneeID == userID
}

// AssignTicket 分配工单
func (s *TicketService) AssignTicket(ctx context.Context, ticketID int, assigneeID int, tenantID int, assignedBy int) (*ent.Ticket, error) {
	s.logger.Infow("Assigning ticket", "ticket_id", ticketID, "assignee_id", assigneeID, "tenant_id", tenantID)

	ticket, err := s.client.Ticket.UpdateOneID(ticketID).
		Where(ticket.TenantID(tenantID)).
		SetAssigneeID(assigneeID).
		SetStatus("assigned").
		Save(ctx)

	if err != nil {
		s.logger.Errorw("Failed to assign ticket", "error", err)
		return nil, fmt.Errorf("failed to assign ticket: %w", err)
	}

	// 记录审计日志
	s.logAuditEvent(ctx, "ticket_assigned", ticketID, tenantID, map[string]interface{}{
		"assignee_id": assigneeID,
		"assigned_by": assignedBy,
		"old_status":  "submitted",
		"new_status":  "assigned",
	})

	// 发送分配通知
	if s.notificationService != nil {
		if err := s.notificationService.NotifyTicketAssigned(ctx, ticketID, assigneeID, tenantID); err != nil {
			s.logger.Warnw("Failed to send assignment notification", "error", err)
		}
	}

	return ticket, nil
}

// EscalateTicket 升级工单
func (s *TicketService) EscalateTicket(ctx context.Context, ticketID int, reason string, tenantID int, escalatedBy int) (*ent.Ticket, error) {
	s.logger.Infow("Escalating ticket", "ticket_id", ticketID, "reason", reason, "tenant_id", tenantID)

	// 获取当前工单信息
	currentTicket, err := s.GetTicket(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}

	// 确定升级规则
	newPriority := s.getEscalatedPriority(currentTicket.Priority)
	newAssignee := s.getEscalationAssignee(currentTicket.Priority, tenantID)

	// 更新工单
	ticket, err := s.client.Ticket.UpdateOneID(ticketID).
		SetPriority(newPriority).
		SetAssigneeID(newAssignee).
		SetStatus("escalated").
		Save(ctx)

	if err != nil {
		s.logger.Errorw("Failed to escalate ticket", "error", err)
		return nil, fmt.Errorf("failed to escalate ticket: %w", err)
	}

	// 记录升级日志
	s.logAuditEvent(ctx, "ticket_escalated", ticketID, tenantID, map[string]interface{}{
		"old_priority":    currentTicket.Priority,
		"new_priority":    newPriority,
		"old_assignee_id": currentTicket.AssigneeID,
		"new_assignee_id": newAssignee,
		"escalated_by":    escalatedBy,
		"reason":          reason,
	})

	// 发送升级通知
	go s.sendEscalationNotification(ticketID, newAssignee, escalatedBy, reason)

	return ticket, nil
}

// ResolveTicket 解决工单
func (s *TicketService) ResolveTicket(ctx context.Context, ticketID int, resolution string, tenantID int, resolvedBy int) (*ent.Ticket, error) {
	s.logger.Infow("Resolving ticket", "ticket_id", ticketID, "tenant_id", tenantID)

	ticket, err := s.client.Ticket.UpdateOneID(ticketID).
		Where(ticket.TenantID(tenantID)).
		SetStatus("resolved").
		Save(ctx)

	if err != nil {
		s.logger.Errorw("Failed to resolve ticket", "error", err)
		return nil, fmt.Errorf("failed to resolve ticket: %w", err)
	}

	// 记录审计日志
	s.logAuditEvent(ctx, "ticket_resolved", ticketID, tenantID, map[string]interface{}{
		"resolved_by": resolvedBy,
		"resolution":  resolution,
		"resolved_at": time.Now(),
	})

	// 发送状态变更通知
	if s.notificationService != nil {
		oldStatus := "in_progress" // 假设之前是进行中状态
		if err := s.notificationService.NotifyTicketStatusChanged(ctx, ticketID, oldStatus, "resolved", tenantID); err != nil {
			s.logger.Warnw("Failed to send resolution notification", "error", err)
		}
	}

	return ticket, nil
}

// CloseTicket 关闭工单
func (s *TicketService) CloseTicket(ctx context.Context, ticketID int, feedback string, tenantID int, closedBy int) (*ent.Ticket, error) {
	s.logger.Infow("Closing ticket", "ticket_id", ticketID, "tenant_id", tenantID)

	ticket, err := s.client.Ticket.UpdateOneID(ticketID).
		Where(ticket.TenantID(tenantID)).
		SetStatus("closed").
		Save(ctx)

	if err != nil {
		s.logger.Errorw("Failed to close ticket", "error", err)
		return nil, fmt.Errorf("failed to close ticket: %w", err)
	}

	// 记录审计日志
	s.logAuditEvent(ctx, "ticket_closed", ticketID, tenantID, map[string]interface{}{
		"closed_by": closedBy,
		"feedback":  feedback,
		"closed_at": time.Now(),
	})

	// 发送状态变更通知
	if s.notificationService != nil {
		oldStatus := ticket.Status
		if err := s.notificationService.NotifyTicketStatusChanged(ctx, ticketID, oldStatus, "closed", tenantID); err != nil {
			s.logger.Warnw("Failed to send close notification", "error", err)
		}
	}

	return ticket, nil
}

// SearchTickets 高级搜索工单
func (s *TicketService) SearchTickets(ctx context.Context, searchTerm string, tenantID int) ([]*ent.Ticket, error) {
	s.logger.Infow("Searching tickets", "search_term", searchTerm, "tenant_id", tenantID)

	// 构建搜索查询（简化版，实际应使用全文搜索引擎）
	query := s.client.Ticket.Query().Where(ticket.TenantID(tenantID))

	// 在标题和描述中搜索
	searchLower := strings.ToLower(searchTerm)
	query = query.Where(
		ticket.Or(
			ticket.TitleContains(searchLower),
			ticket.DescriptionContains(searchLower),
		),
	)

	tickets, err := query.
		Order(ent.Desc(ticket.FieldCreatedAt)).
		Limit(100). // 限制搜索结果
		All(ctx)

	if err != nil {
		s.logger.Errorw("Failed to search tickets", "error", err)
		return nil, fmt.Errorf("failed to search tickets: %w", err)
	}

	return tickets, nil
}

// GetOverdueTickets 获取逾期工单
func (s *TicketService) GetOverdueTickets(ctx context.Context, tenantID int) ([]*ent.Ticket, error) {
	s.logger.Infow("Getting overdue tickets", "tenant_id", tenantID)

	now := time.Now()

	// 查找创建时间超过SLA时限的工单（简化逻辑）
	tickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.StatusNotIn("closed", "resolved"),
			ticket.CreatedAtLT(now.Add(-24*time.Hour)), // 24小时前创建的未解决工单
		).
		Order(ent.Desc(ticket.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		s.logger.Errorw("Failed to get overdue tickets", "error", err)
		return nil, fmt.Errorf("failed to get overdue tickets: %w", err)
	}

	return tickets, nil
}

// GetTicketsByAssignee 获取指定处理人的工单
func (s *TicketService) GetTicketsByAssignee(ctx context.Context, assigneeID int, tenantID int) ([]*ent.Ticket, error) {
	s.logger.Infow("Getting tickets by assignee", "assignee_id", assigneeID, "tenant_id", tenantID)

	tickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.AssigneeID(assigneeID),
			ticket.StatusNotIn("closed"),
		).
		Order(ent.Desc(ticket.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		s.logger.Errorw("Failed to get tickets by assignee", "error", err)
		return nil, fmt.Errorf("failed to get tickets by assignee: %w", err)
	}

	return tickets, nil
}

// GetTicketActivity 获取工单活动日志
func (s *TicketService) GetTicketActivity(ctx context.Context, ticketID int, tenantID int) ([]map[string]interface{}, error) {
	s.logger.Infow("Getting ticket activity", "ticket_id", ticketID, "tenant_id", tenantID)

	// 这里应该从审计日志表中查询，暂时返回模拟数据
	activities := []map[string]interface{}{
		{
			"action":    "created",
			"timestamp": time.Now().Add(-2 * time.Hour),
			"user_id":   1,
			"details":   "工单已创建",
		},
		{
			"action":    "assigned",
			"timestamp": time.Now().Add(-1 * time.Hour),
			"user_id":   2,
			"details":   "工单已分配给技术支持",
		},
	}

	return activities, nil
}

// getEscalatedPriority 获取升级后的优先级
func (s *TicketService) getEscalatedPriority(currentPriority string) string {
	switch currentPriority {
	case "low":
		return "medium"
	case "medium":
		return "high"
	case "high":
		return "critical"
	default:
		return "high"
	}
}

// getEscalationAssignee 获取升级处理人
func (s *TicketService) getEscalationAssignee(priority string, tenantID int) int {
	// 简化逻辑：根据优先级分配不同级别的处理人
	// 实际应该查询用户角色和技能匹配
	switch priority {
	case "critical":
		return 1 // 高级工程师
	case "high":
		return 2 // 资深工程师
	default:
		return 3 // 普通工程师
	}
}

// sendAssignmentNotification 发送分配通知
func (s *TicketService) sendAssignmentNotification(ticketID, assigneeID, assignedBy int) {
	s.logger.Infow("Sending assignment notification",
		"ticket_id", ticketID,
		"assignee_id", assigneeID,
		"assigned_by", assignedBy,
	)
	// 实现通知逻辑
}

// sendEscalationNotification 发送升级通知
func (s *TicketService) sendEscalationNotification(ticketID, newAssignee, escalatedBy int, reason string) {
	s.logger.Infow("Sending escalation notification",
		"ticket_id", ticketID,
		"new_assignee", newAssignee,
		"escalated_by", escalatedBy,
		"reason", reason,
	)
	// 实现通知逻辑
}

// sendResolutionNotification 发送解决通知
func (s *TicketService) sendResolutionNotification(ticketID, requesterID, resolvedBy int) {
	s.logger.Infow("Sending resolution notification",
		"ticket_id", ticketID,
		"requester_id", requesterID,
		"resolved_by", resolvedBy,
	)
	// 实现通知逻辑
}

// logAuditEvent 记录审计事件
func (s *TicketService) logAuditEvent(ctx context.Context, event string, ticketID int, tenantID int, metadata map[string]interface{}) {
	s.logger.Infow("Audit event",
		"event", event,
		"ticket_id", ticketID,
		"tenant_id", tenantID,
		"metadata", metadata,
	)

	// 这里应该将审计日志保存到数据库
	// 暂时只记录到日志中
}

// ExportTickets 导出工单
func (s *TicketService) ExportTickets(ctx context.Context, tenantID int, filters map[string]interface{}, format string) ([]byte, error) {
	query := s.client.Ticket.Query().Where(ticket.TenantID(tenantID))

	// 应用过滤条件
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where(ticket.StatusEQ(status))
	}
	if priority, ok := filters["priority"].(string); ok && priority != "" {
		query = query.Where(ticket.PriorityEQ(priority))
	}

	tickets, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	// 转换为导出格式
	var exportData []map[string]interface{}
	for _, t := range tickets {
		exportData = append(exportData, map[string]interface{}{
			"工单编号": t.TicketNumber,
			"标题":   t.Title,
			"描述":   t.Description,
			"状态":   t.Status,
			"优先级":  t.Priority,
			"创建时间": t.CreatedAt.Format("2006-01-02 15:04:05"),
			"更新时间": t.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	// 根据格式生成数据
	switch format {
	case "csv":
		return s.generateCSV(exportData)
	case "excel":
		return s.generateExcel(exportData)
	case "json":
		return json.Marshal(exportData)
	default:
		return nil, fmt.Errorf("不支持的导出格式: %s", format)
	}
}

// ImportTickets 导入工单
func (s *TicketService) ImportTickets(ctx context.Context, tenantID int, data []byte, format string) error {
	var tickets []map[string]interface{}

	// 解析数据
	switch format {
	case "csv":
		var err error
		tickets, err = s.parseCSV(data)
		if err != nil {
			return err
		}
	case "excel":
		var err error
		tickets, err = s.parseExcel(data)
		if err != nil {
			return err
		}
	case "json":
		if err := json.Unmarshal(data, &tickets); err != nil {
			return err
		}
	default:
		return fmt.Errorf("不支持的导入格式: %s", format)
	}

	// 批量创建工单
	for _, ticketData := range tickets {
		_, err := s.CreateTicket(ctx, &dto.CreateTicketRequest{
			Title:       ticketData["标题"].(string),
			Description: ticketData["描述"].(string),
			Priority:    ticketData["优先级"].(string),
			Category:    "导入",
			RequesterID: 1, // 默认用户
		}, tenantID)
		if err != nil {
			return fmt.Errorf("导入工单失败: %v", err)
		}
	}

	return nil
}

// AssignTickets 批量分配工单
func (s *TicketService) AssignTickets(ctx context.Context, tenantID int, ticketIDs []int, assigneeID int) error {
	// 验证分配者是否存在
	_, err := s.client.User.Get(ctx, assigneeID)
	if err != nil {
		return fmt.Errorf("分配者不存在: %v", err)
	}

	// 批量更新工单
	for _, ticketID := range ticketIDs {
		_, err := s.client.Ticket.UpdateOneID(ticketID).
			SetAssigneeID(assigneeID).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("分配工单 %d 失败: %v", ticketID, err)
		}
	}

	return nil
}

// GetTicketAnalytics 获取工单分析数据
func (s *TicketService) GetTicketAnalytics(ctx context.Context, tenantID int, dateFrom, dateTo time.Time) (*dto.TicketAnalyticsResponse, error) {
	// 基础查询
	query := s.client.Ticket.Query().Where(ticket.TenantID(tenantID))

	// 时间范围过滤
	if !dateFrom.IsZero() {
		query = query.Where(ticket.CreatedAtGTE(dateFrom))
	}
	if !dateTo.IsZero() {
		query = query.Where(ticket.CreatedAtLTE(dateTo))
	}

	// 获取统计数据
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	// 按状态统计
	statusStats := make(map[string]int)
	tickets, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	for _, t := range tickets {
		statusStats[t.Status]++
	}

	// 按优先级统计
	priorityStats := make(map[string]int)
	for _, t := range tickets {
		priorityStats[t.Priority]++
	}

	// 计算平均解决时间
	resolvedTickets, err := query.Where(ticket.StatusEQ("resolved")).All(ctx)
	if err != nil {
		return nil, err
	}

	var totalResolutionTime time.Duration
	resolvedCount := 0
	for _, ticket := range resolvedTickets {
		if !ticket.UpdatedAt.IsZero() {
			totalResolutionTime += ticket.UpdatedAt.Sub(ticket.CreatedAt)
			resolvedCount++
		}
	}

	avgResolutionTime := time.Duration(0)
	if resolvedCount > 0 {
		avgResolutionTime = totalResolutionTime / time.Duration(resolvedCount)
	}

	return &dto.TicketAnalyticsResponse{
		Data: []map[string]interface{}{
			{"total": total},
			{"status_distribution": statusStats},
			{"priority_distribution": priorityStats},
			{"avg_resolution_time": avgResolutionTime.Hours()},
			{"resolved_count": resolvedCount},
		},
		Summary: map[string]interface{}{
			"total":    total,
			"resolved": resolvedCount,
		},
		GeneratedAt: time.Now(),
	}, nil
}

// CreateTicketTemplate 创建工单模板
func (s *TicketService) CreateTicketTemplate(ctx context.Context, tenantID int, req interface{}) (interface{}, error) {
	// 调用专门的工单模板服务
	templateService := NewTicketTemplateService(s.client)

	// 类型断言
	createReq, ok := req.(*dto.TicketTemplate)
	if !ok {
		return nil, fmt.Errorf("无效的请求参数类型")
	}

	// 转换为服务请求格式
	serviceReq := &CreateTemplateRequest{
		Name:          createReq.Name,
		Description:   createReq.Description,
		Category:      createReq.Category,
		Priority:      createReq.Priority,
		FormFields:    createReq.FormFields,
		WorkflowSteps: nil, // 暂时设为nil，后续可以扩展
		IsActive:      createReq.IsActive,
		TenantID:      tenantID,
	}

	template, err := templateService.CreateTemplate(ctx, serviceReq)
	if err != nil {
		return nil, err
	}

	return template, nil
}

// UpdateTicketTemplate 更新工单模板
func (s *TicketService) UpdateTicketTemplate(ctx context.Context, templateID int, req interface{}) (interface{}, error) {
	// 调用专门的工单模板服务
	templateService := NewTicketTemplateService(s.client)

	// 类型断言
	updateReq, ok := req.(*dto.TicketTemplate)
	if !ok {
		return nil, fmt.Errorf("无效的请求参数类型")
	}

	// 转换为服务请求格式
	serviceReq := &UpdateTemplateRequest{
		Name:          updateReq.Name,
		Description:   updateReq.Description,
		Category:      updateReq.Category,
		Priority:      updateReq.Priority,
		FormFields:    updateReq.FormFields,
		WorkflowSteps: nil, // 暂时设为nil，后续可以扩展
		IsActive:      &updateReq.IsActive,
	}

	template, err := templateService.UpdateTemplate(ctx, templateID, serviceReq)
	if err != nil {
		return nil, err
	}

	return template, nil
}

// DeleteTicketTemplate 删除工单模板
func (s *TicketService) DeleteTicketTemplate(ctx context.Context, templateID int) error {
	// 调用专门的工单模板服务
	templateService := NewTicketTemplateService(s.client)
	return templateService.DeleteTemplate(ctx, templateID)
}

// GetTicketTemplates 获取工单模板列表
func (s *TicketService) GetTicketTemplates(ctx context.Context, tenantID int) ([]interface{}, error) {
	// 调用专门的工单模板服务
	templateService := NewTicketTemplateService(s.client)

	// 构建请求参数
	req := &ListTemplatesRequest{
		Page:      1,
		PageSize:  100,
		TenantID:  tenantID,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	templates, _, err := templateService.ListTemplates(ctx, req)
	if err != nil {
		return nil, err
	}

	// 转换为DTO格式
	var result []interface{}
	for _, template := range templates {
		// 反序列化表单字段
		var formFields map[string]interface{}
		if len(template.FormFields) > 0 {
			if err := json.Unmarshal(template.FormFields, &formFields); err != nil {
				s.logger.Warnw("反序列化表单字段失败", "error", err, "template_id", template.ID)
				formFields = make(map[string]interface{})
			}
		} else {
			formFields = make(map[string]interface{})
		}

		dtoTemplate := &dto.TicketTemplate{
			ID:          template.ID,
			Name:        template.Name,
			Description: template.Description,
			Category:    template.Category,
			Priority:    template.Priority,
			FormFields:  formFields,
			IsActive:    template.IsActive,
			CreatedAt:   template.CreatedAt,
			UpdatedAt:   template.UpdatedAt,
		}
		result = append(result, dtoTemplate)
	}

	return result, nil
}

// 辅助方法
func (s *TicketService) generateCSV(data []map[string]interface{}) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// 写入表头
	var headers []string
	for key := range data[0] {
		headers = append(headers, key)
	}
	writer.Write(headers)

	// 写入数据
	for _, row := range data {
		var record []string
		for _, header := range headers {
			value := row[header]
			if value == nil {
				record = append(record, "")
			} else {
				record = append(record, fmt.Sprintf("%v", value))
			}
		}
		writer.Write(record)
	}

	writer.Flush()
	return buf.Bytes(), writer.Error()
}

func (s *TicketService) generateExcel(data []map[string]interface{}) ([]byte, error) {
	// 这里应该使用Excel库生成Excel文件
	// 暂时返回CSV格式
	return s.generateCSV(data)
}

func (s *TicketService) parseCSV(data []byte) ([]map[string]interface{}, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV文件格式错误")
	}

	headers := records[0]
	var result []map[string]interface{}

	for i := 1; i < len(records); i++ {
		row := make(map[string]interface{})
		for j, header := range headers {
			if j < len(records[i]) {
				row[header] = records[i][j]
			}
		}
		result = append(result, row)
	}

	return result, nil
}

func (s *TicketService) parseExcel(data []byte) ([]map[string]interface{}, error) {
	// 这里应该使用Excel库解析Excel文件
	// 暂时返回空结果
	return []map[string]interface{}{}, nil
}
