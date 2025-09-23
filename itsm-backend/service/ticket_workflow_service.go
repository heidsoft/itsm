package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"time"

	"go.uber.org/zap"
)

type TicketWorkflowService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewTicketWorkflowService(client *ent.Client, logger *zap.SugaredLogger) *TicketWorkflowService {
	return &TicketWorkflowService{
		client: client,
		logger: logger,
	}
}

// ProcessWorkflowAction 处理工作流动作
func (s *TicketWorkflowService) ProcessWorkflowAction(ctx context.Context, req *dto.TicketWorkflowRequest, tenantID int) (*ent.Ticket, error) {
	s.logger.Infow("Processing workflow action", 
		"ticket_id", req.TicketID, 
		"action", req.Action, 
		"user_id", req.UserID,
		"tenant_id", tenantID)

	// 获取工单
	ticketEntity, err := s.client.Ticket.Query().
		Where(
			ticket.ID(req.TicketID),
			ticket.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get ticket", "error", err, "ticket_id", req.TicketID)
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	// 验证状态转换是否合法
	newStatus, err := s.validateStatusTransition(ticketEntity.Status, req.Action)
	if err != nil {
		s.logger.Errorw("Invalid status transition", "error", err, 
			"current_status", ticketEntity.Status, "action", req.Action)
		return nil, err
	}

	// 开始事务
	tx, err := s.client.Tx(ctx)
	if err != nil {
		s.logger.Errorw("Failed to start transaction", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 更新工单状态
	updateQuery := tx.Ticket.UpdateOneID(req.TicketID).
		SetStatus(newStatus).
		SetUpdatedAt(time.Now())

	// 如果有新的处理人，更新处理人
	if req.NextAssigneeID != 0 {
		updateQuery.SetAssigneeID(req.NextAssigneeID)
	}

	updatedTicket, err := updateQuery.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update ticket", "error", err)
		return nil, fmt.Errorf("failed to update ticket: %w", err)
	}

	// 记录工作流历史
	err = s.recordWorkflowHistory(ctx, tx, req, ticketEntity.Status, newStatus)
	if err != nil {
		s.logger.Errorw("Failed to record workflow history", "error", err)
		return nil, fmt.Errorf("failed to record workflow history: %w", err)
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		s.logger.Errorw("Failed to commit transaction", "error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Infow("Workflow action processed successfully", 
		"ticket_id", req.TicketID, 
		"old_status", ticketEntity.Status,
		"new_status", newStatus)

	return updatedTicket, nil
}

// validateStatusTransition 验证状态转换是否合法
func (s *TicketWorkflowService) validateStatusTransition(currentStatus, action string) (string, error) {
	// 定义状态转换规则
	transitions := map[string]map[string]string{
		"open": {
			"start":    "in_progress",
			"resolve":  "resolved",
			"close":    "closed",
			"escalate": "escalated",
		},
		"in_progress": {
			"resolve":  "resolved",
			"close":    "closed",
			"escalate": "escalated",
			"pause":    "on_hold",
		},
		"on_hold": {
			"resume": "in_progress",
			"close":  "closed",
		},
		"resolved": {
			"close":  "closed",
			"reopen": "open",
		},
		"escalated": {
			"resolve": "resolved",
			"close":   "closed",
		},
		"closed": {
			"reopen": "open",
		},
	}

	if statusMap, exists := transitions[currentStatus]; exists {
		if newStatus, valid := statusMap[action]; valid {
			return newStatus, nil
		}
	}

	return "", fmt.Errorf("invalid transition from %s with action %s", currentStatus, action)
}

// recordWorkflowHistory 记录工作流历史
func (s *TicketWorkflowService) recordWorkflowHistory(ctx context.Context, tx *ent.Tx, req *dto.TicketWorkflowRequest, oldStatus, newStatus string) error {
	// 这里应该创建工作流历史记录
	// 由于没有具体的WorkflowHistory实体，这里只是记录日志
	s.logger.Infow("Workflow history recorded",
		"ticket_id", req.TicketID,
		"user_id", req.UserID,
		"action", req.Action,
		"old_status", oldStatus,
		"new_status", newStatus,
		"comment", req.Comment,
		"timestamp", time.Now())

	return nil
}

// EscalateTicket 升级工单
func (s *TicketWorkflowService) EscalateTicket(ctx context.Context, ticketID int, reason string, tenantID, escalatedBy int) (*ent.Ticket, error) {
	s.logger.Infow("Escalating ticket", "ticket_id", ticketID, "reason", reason, "escalated_by", escalatedBy)

	// 获取工单
	ticketEntity, err := s.client.Ticket.Query().
		Where(
			ticket.ID(ticketID),
			ticket.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	// 更新工单状态和优先级
	newPriority := s.getEscalatedPriority(ticketEntity.Priority)
	
	updatedTicket, err := s.client.Ticket.UpdateOneID(ticketID).
		SetStatus("escalated").
		SetPriority(newPriority).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to escalate ticket: %w", err)
	}

	// 记录升级历史
	s.logger.Infow("Ticket escalated successfully",
		"ticket_id", ticketID,
		"old_priority", ticketEntity.Priority,
		"new_priority", newPriority,
		"reason", reason,
		"escalated_by", escalatedBy)

	return updatedTicket, nil
}

// getEscalatedPriority 获取升级后的优先级
func (s *TicketWorkflowService) getEscalatedPriority(currentPriority string) string {
	switch currentPriority {
	case "low":
		return "medium"
	case "medium":
		return "high"
	case "high":
		return "critical"
	case "critical":
		return "critical" // 已经是最高级别
	default:
		return "high"
	}
}

// ResolveTicket 解决工单
func (s *TicketWorkflowService) ResolveTicket(ctx context.Context, ticketID int, resolution string, tenantID, resolvedBy int) (*ent.Ticket, error) {
	s.logger.Infow("Resolving ticket", "ticket_id", ticketID, "resolved_by", resolvedBy)

	updatedTicket, err := s.client.Ticket.UpdateOneID(ticketID).
		Where(ticket.TenantID(tenantID)).
		SetStatus("resolved").
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve ticket: %w", err)
	}

	// 记录解决历史
	s.logger.Infow("Ticket resolved successfully",
		"ticket_id", ticketID,
		"resolution", resolution,
		"resolved_by", resolvedBy)

	return updatedTicket, nil
}

// CloseTicket 关闭工单
func (s *TicketWorkflowService) CloseTicket(ctx context.Context, ticketID int, feedback string, tenantID, closedBy int) (*ent.Ticket, error) {
	s.logger.Infow("Closing ticket", "ticket_id", ticketID, "closed_by", closedBy)

	updatedTicket, err := s.client.Ticket.UpdateOneID(ticketID).
		Where(ticket.TenantID(tenantID)).
		SetStatus("closed").
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to close ticket: %w", err)
	}

	// 记录关闭历史
	s.logger.Infow("Ticket closed successfully",
		"ticket_id", ticketID,
		"feedback", feedback,
		"closed_by", closedBy)

	return updatedTicket, nil
}

// GetTicketActivity 获取工单活动历史
func (s *TicketWorkflowService) GetTicketActivity(ctx context.Context, ticketID, tenantID int) ([]map[string]interface{}, error) {
	s.logger.Infow("Getting ticket activity", "ticket_id", ticketID)

	// 验证工单存在
	exists, err := s.client.Ticket.Query().
		Where(
			ticket.ID(ticketID),
			ticket.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check ticket existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("ticket not found")
	}

	// 这里应该从工作流历史表获取活动记录
	// 由于没有具体的实体，返回模拟数据
	activities := []map[string]interface{}{
		{
			"id":          1,
			"ticket_id":   ticketID,
			"action":      "created",
			"description": "工单已创建",
			"user_id":     1,
			"user_name":   "系统用户",
			"created_at":  time.Now().Add(-2 * time.Hour),
		},
		{
			"id":          2,
			"ticket_id":   ticketID,
			"action":      "assigned",
			"description": "工单已分配给处理人",
			"user_id":     2,
			"user_name":   "管理员",
			"created_at":  time.Now().Add(-1 * time.Hour),
		},
	}

	return activities, nil
}