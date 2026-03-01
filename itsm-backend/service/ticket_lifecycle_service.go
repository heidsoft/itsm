package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/ticket"

	"go.uber.org/zap"
)

// TicketLifecycleServiceInterface 工单生命周期服务接口
type TicketLifecycleServiceInterface interface {
	// ResolveTicket 解决工单
	ResolveTicket(ctx context.Context, ticketID int, resolution string, tenantID int, resolvedBy int) (*ent.Ticket, error)
	// CloseTicket 关闭工单
	CloseTicket(ctx context.Context, ticketID int, feedback string, tenantID int, closedBy int) (*ent.Ticket, error)
	// EscalateTicket 升级工单
	EscalateTicket(ctx context.Context, ticketID int, reason string, tenantID int, escalatedBy int) (*ent.Ticket, error)
	// UpdateTicketStatus 更新工单状态
	UpdateTicketStatus(ctx context.Context, ticketID int, status string, tenantID int, operatorID int) (*ent.Ticket, error)
	// CancelWorkflow 取消工作流
	CancelWorkflow(ctx context.Context, ticketID int, tenantID int, reason string) error
	// SyncTicketStatusWithWorkflow 同步工单状态和工作流
	SyncTicketStatusWithWorkflow(ctx context.Context, ticketID int, tenantID int) error
}

// TicketLifecycleService 工单生命周期服务
type TicketLifecycleService struct {
	client              *ent.Client
	logger              *zap.SugaredLogger
	notificationService *TicketNotificationService
}

// NewTicketLifecycleService 创建工单生命周期服务
func NewTicketLifecycleService(client *ent.Client, logger *zap.SugaredLogger) *TicketLifecycleService {
	return &TicketLifecycleService{
		client: client,
		logger: logger,
	}
}

// SetNotificationService 设置通知服务
func (s *TicketLifecycleService) SetNotificationService(notificationService *TicketNotificationService) {
	s.notificationService = notificationService
}

// ResolveTicket 解决工单
func (s *TicketLifecycleService) ResolveTicket(ctx context.Context, ticketID int, resolution string, tenantID int, resolvedBy int) (*ent.Ticket, error) {
	// 验证工单存在
	t, err := s.client.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to find ticket", "ticketID", ticketID, "error", err)
		return nil, err
	}

	// 只有处于开放状态的工单才能被解决
	if t.Status != common.TicketStatusOpen && t.Status != common.TicketStatusInProgress && t.Status != common.TicketStatusPending {
		return nil, ErrInvalidTicketStatus
	}

	// 更新工单状态为已解决
	updatedTicket, err := s.client.Ticket.UpdateOne(t).
		SetStatus(common.TicketStatusResolved).
		SetResolvedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to resolve ticket", "ticketID", ticketID, "error", err)
		return nil, err
	}

	// 记录审计日志
	s.logAuditEvent(ctx, "ticket_resolved", ticketID, tenantID, map[string]interface{}{
		"resolvedBy":    resolvedBy,
		"resolution":   resolution,
		"previousStatus": t.Status,
	})

	// 发送解决通知
	if s.notificationService != nil {
		s.notificationService.SendResolutionNotification(ticketID, t.RequesterID, resolvedBy)
	}

	s.logger.Infow("Ticket resolved", "ticketID", ticketID, "resolvedBy", resolvedBy)
	return updatedTicket, nil
}

// CloseTicket 关闭工单
func (s *TicketLifecycleService) CloseTicket(ctx context.Context, ticketID int, feedback string, tenantID int, closedBy int) (*ent.Ticket, error) {
	// 验证工单存在
	t, err := s.client.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to find ticket", "ticketID", ticketID, "error", err)
		return nil, err
	}

	// 只有已解决或已关闭的工单才能被关闭
	if t.Status != common.TicketStatusResolved && t.Status != common.TicketStatusClosed {
		return nil, ErrInvalidTicketStatus
	}

	// 更新工单状态为已关闭
	updatedTicket, err := s.client.Ticket.UpdateOne(t).
		SetStatus(common.TicketStatusClosed).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to close ticket", "ticketID", ticketID, "error", err)
		return nil, err
	}

	// 记录审计日志
	s.logAuditEvent(ctx, "ticket_closed", ticketID, tenantID, map[string]interface{}{
		"closedBy":       closedBy,
		"feedback":       feedback,
		"previousStatus": t.Status,
	})

	s.logger.Infow("Ticket closed", "ticketID", ticketID, "closedBy", closedBy)
	return updatedTicket, nil
}

// EscalateTicket 升级工单
func (s *TicketLifecycleService) EscalateTicket(ctx context.Context, ticketID int, reason string, tenantID int, escalatedBy int) (*ent.Ticket, error) {
	// 验证工单存在
	t, err := s.client.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to find ticket", "ticketID", ticketID, "error", err)
		return nil, err
	}

	// 计算升级后的优先级
	newPriority := s.getEscalatedPriority(t.Priority)

	// 获取升级后的处理人
	newAssigneeID := s.getEscalationAssignee(newPriority, tenantID)

	// 构建更新操作
	update := s.client.Ticket.UpdateOne(t).
		SetPriority(newPriority).
		SetStatus("escalated")

	// 如果获取到了新的处理人，则分配给他
	if newAssigneeID > 0 {
		update = update.SetAssigneeID(newAssigneeID)
	}

	// 更新工单
	updatedTicket, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to escalate ticket", "ticketID", ticketID, "error", err)
		return nil, err
	}

	// 记录审计日志
	s.logAuditEvent(ctx, "ticket_escalated", ticketID, tenantID, map[string]interface{}{
		"escalatedBy":     escalatedBy,
		"reason":          reason,
		"previousPriority": t.Priority,
		"newPriority":     newPriority,
		"newAssignee":     newAssigneeID,
	})

	// 发送升级通知
	if s.notificationService != nil && newAssigneeID > 0 {
		s.notificationService.SendEscalationNotification(ticketID, newAssigneeID, escalatedBy, reason)
	}

	s.logger.Infow("Ticket escalated", "ticketID", ticketID, "newPriority", newPriority, "newAssignee", newAssigneeID)
	return updatedTicket, nil
}

// UpdateTicketStatus 更新工单状态
func (s *TicketLifecycleService) UpdateTicketStatus(ctx context.Context, ticketID int, status string, tenantID int, operatorID int) (*ent.Ticket, error) {
	// 验证工单存在
	t, err := s.client.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to find ticket", "ticketID", ticketID, "error", err)
		return nil, err
	}

	// 验证状态转换是否合法
	if !s.isValidStatusTransition(t.Status, status) {
		return nil, ErrInvalidTicketStatus
	}

	// 构建更新操作
	update := s.client.Ticket.UpdateOne(t).SetStatus(status)

	// 根据新状态设置相关时间
	switch status {
	case common.TicketStatusResolved:
		update = update.SetResolvedAt(time.Now())
	}

	updatedTicket, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update ticket status", "ticketID", ticketID, "status", status, "error", err)
		return nil, err
	}

	// 记录审计日志
	s.logAuditEvent(ctx, "ticket_status_updated", ticketID, tenantID, map[string]interface{}{
		"operatorID":      operatorID,
		"previousStatus":  t.Status,
		"newStatus":       status,
	})

	s.logger.Infow("Ticket status updated", "ticketID", ticketID, "newStatus", status, "operatorID", operatorID)
	return updatedTicket, nil
}

// CancelWorkflow 取消工作流
func (s *TicketLifecycleService) CancelWorkflow(ctx context.Context, ticketID int, tenantID int, reason string) error {
	// 使用 BusinessKey 查找流程实例
	businessKey := fmt.Sprintf("ticket:%d", ticketID)
	instance, err := s.client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
			processinstance.StatusNEQ("completed"),
			processinstance.StatusNEQ("terminated"),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			s.logger.Warnw("No active workflow found for ticket", "ticketID", ticketID)
			return nil
		}
		s.logger.Errorw("Failed to query workflow instance", "ticketID", ticketID, "error", err)
		return err
	}

	// 更新流程实例状态为终止
	_, err = s.client.ProcessInstance.UpdateOne(instance).
		SetStatus("terminated").
		SetEndTime(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to cancel workflow", "instanceID", instance.ID, "error", err)
		return err
	}

	s.logger.Infow("Workflow cancelled", "ticketID", ticketID, "instanceID", instance.ID, "reason", reason)
	return nil
}

// SyncTicketStatusWithWorkflow 同步工单状态和工作流状态
func (s *TicketLifecycleService) SyncTicketStatusWithWorkflow(ctx context.Context, ticketID int, tenantID int) error {
	// 获取工单
	t, err := s.client.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to find ticket", "ticketID", ticketID, "error", err)
		return err
	}

	// 使用 BusinessKey 查找流程实例
	businessKey := fmt.Sprintf("ticket:%d", ticketID)
	instance, err := s.client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
		).
		Order(ent.Desc("created_at")).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// 没有流程实例，保持工单当前状态
			return nil
		}
		s.logger.Errorw("Failed to query workflow instance", "ticketID", ticketID, "error", err)
		return err
	}

	// 根据流程状态更新工单状态
	newStatus := s.mapProcessStatus(instance.Status)
	if newStatus != "" && newStatus != t.Status {
		_, err = s.client.Ticket.UpdateOne(t).
			SetStatus(newStatus).
			Save(ctx)
		if err != nil {
			s.logger.Errorw("Failed to sync ticket status", "ticketID", ticketID, "error", err)
			return err
		}
		s.logger.Infow("Ticket status synced with workflow", "ticketID", ticketID, "workflowStatus", instance.Status, "ticketStatus", newStatus)
	}

	return nil
}

// mapProcessStatus 将流程状态映射为工单状态
func (s *TicketLifecycleService) mapProcessStatus(status string) string {
	switch status {
	case "completed":
		return common.TicketStatusResolved
	case "terminated", "cancelled":
		return common.TicketStatusClosed
	case "running", "active":
		return common.TicketStatusInProgress
	default:
		return ""
	}
}

// isValidStatusTransition 检查状态转换是否合法
func (s *TicketLifecycleService) isValidStatusTransition(currentStatus, newStatus string) bool {
	validTransitions := map[string][]string{
		common.TicketStatusOpen:         {common.TicketStatusInProgress, common.TicketStatusPending, common.TicketStatusClosed},
		common.TicketStatusInProgress:   {common.TicketStatusResolved, common.TicketStatusPending, common.TicketStatusOpen},
		common.TicketStatusPending:      {common.TicketStatusInProgress, common.TicketStatusResolved, common.TicketStatusOpen},
		common.TicketStatusResolved:     {common.TicketStatusClosed, common.TicketStatusOpen},
		common.TicketStatusClosed:        {}, // 已关闭的工单不能转换到其他状态
	}

	allowed, ok := validTransitions[currentStatus]
	if !ok {
		return false
	}

	for _, status := range allowed {
		if status == newStatus {
			return true
		}
	}

	return false
}

// getEscalatedPriority 获取升级后的优先级
func (s *TicketLifecycleService) getEscalatedPriority(currentPriority string) string {
	priorityLevels := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	currentLevel, ok := priorityLevels[currentPriority]
	if !ok {
		return currentPriority
	}

	// 升级到更高优先级，最多到 critical
	if currentLevel < 4 {
		for p, level := range priorityLevels {
			if level == currentLevel+1 {
				return p
			}
		}
	}

	return currentPriority
}

// getEscalationAssignee 获取升级后的处理人
func (s *TicketLifecycleService) getEscalationAssignee(priority string, tenantID int) int {
	// 这里应该查询升级规则配置，返回对应的处理人
	// 暂时返回0，表示不自动分配处理人
	return 0
}

// logAuditEvent 记录审计日志
func (s *TicketLifecycleService) logAuditEvent(ctx context.Context, event string, ticketID int, tenantID int, metadata map[string]interface{}) {
	// 实现审计日志记录逻辑
	s.logger.Infow("Audit event",
		"event", event,
		"ticketID", ticketID,
		"tenantID", tenantID,
		"metadata", metadata,
	)
}

// ErrInvalidTicketStatus 无效的工单状态错误
var ErrInvalidTicketStatus = fmt.Errorf("invalid ticket status transition")
