package bpmn

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"

	"go.uber.org/zap"
)

// TicketNotificationServiceInterface 通知服务接口
// 用于避免循环依赖
type TicketNotificationServiceInterface interface {
	SendNotification(ctx context.Context, ticketID int, req *dto.SendTicketNotificationRequest, tenantID int) error
}

// DefaultTicketNotificationService 默认通知服务（空实现）
type DefaultTicketNotificationService struct{}

// SendNotification 默认不发送通知
func (d *DefaultTicketNotificationService) SendNotification(ctx context.Context, ticketID int, req *dto.SendTicketNotificationRequest, tenantID int) error {
	// 默认实现不执行任何操作
	return nil
}

// TicketServiceTaskHandler 工单服务任务处理器
type TicketServiceTaskHandler struct {
	HandlerBase
	client             *ent.Client
	logger             *zap.SugaredLogger
	notificationService TicketNotificationServiceInterface
}

// NewTicketServiceTaskHandler 创建工单处理器
func NewTicketServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *TicketServiceTaskHandler {
	handler := &TicketServiceTaskHandler{
		client:               client,
		logger:              logger,
		notificationService: &DefaultTicketNotificationService{},
	}
	return handler
}

// SetNotificationService 设置通知服务
func (h *TicketServiceTaskHandler) SetNotificationService(svc TicketNotificationServiceInterface) {
	h.notificationService = svc
}

// GetTaskType 返回任务类型
func (h *TicketServiceTaskHandler) GetTaskType() string {
	return "ticket_task"
}

// GetHandlerID 返回处理器标识
func (h *TicketServiceTaskHandler) GetHandlerID() string {
	return "ticket_service_handler"
}

// Execute 执行工单服务任务
func (h *TicketServiceTaskHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	// 提取业务ID
	businessID, ok := variables["business_id"].(int)
	if !ok {
		return nil, fmt.Errorf("无效的 business_id")
	}

	// 根据任务类型执行不同操作
	action, _ := variables["action"].(string)
	switch action {
	case "update_status":
		// 更新状态
		return h.updateTicketStatus(ctx, businessID, variables)
	case "notify_requester":
		// 通知请求人
		return h.notifyRequester(ctx, businessID, variables)
	case "notify_handler":
		// 通知处理人
		return h.notifyHandler(ctx, businessID, variables)
	case "escalate":
		// 升级处理
		return h.escalateTicket(ctx, businessID, variables)
	case "assign":
		// 分配任务
		return h.assignTicket(ctx, businessID, variables)
	default:
		return &dto.ServiceTaskResult{
			Success: true,
			Message: "无操作执行",
		}, nil
	}
}

// Validate 验证配置
func (h *TicketServiceTaskHandler) Validate(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// updateTicketStatus 更新工单状态
func (h *TicketServiceTaskHandler) updateTicketStatus(ctx context.Context, ticketID int, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	// 获取新状态
	newStatus, _ := variables["new_status"].(string)
	if newStatus == "" {
		newStatus = "in_progress"
	}

	// 解析附加字段
	additionalData := make(map[string]interface{})
	if formFields, ok := variables["form_fields"].(map[string]interface{}); ok {
		additionalData["form_fields"] = formFields
	}

	// 更新工单状态
	updates := map[string]interface{}{
		"status": newStatus,
	}

	// 如果有解决时间，设置解决时间
	if newStatus == "resolved" || newStatus == "closed" {
		now := time.Now()
		updates["resolved_at"] = now
	}

	// 执行更新
	_, err := h.client.Ticket.UpdateOneID(ticketID).
		SetStatus(newStatus).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		h.logger.Errorw("Failed to update ticket status", "ticket_id", ticketID, "error", err)
		return nil, fmt.Errorf("更新工单状态失败: %w", err)
	}

	h.logger.Infow("Ticket status updated via BPMN", "ticket_id", ticketID, "new_status", newStatus)

	return &dto.ServiceTaskResult{
		Success:     true,
		Message:     fmt.Sprintf("工单 %d 状态已更新为 %s", ticketID, newStatus),
		UpdatedData: additionalData,
	}, nil
}

// notifyRequester 通知请求人
func (h *TicketServiceTaskHandler) notifyRequester(ctx context.Context, ticketID int, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	// 获取通知内容
	notificationType, _ := variables["notification_type"].(string)
	if notificationType == "" {
		notificationType = "status_update"
	}
	content, _ := variables["content"].(string)
	channel, _ := variables["channel"].(string)
	if channel == "" {
		channel = "in_app"
	}

	// 获取工单信息
	ticketEntity, err := h.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	// 发送通知给请求人
	if ticketEntity.RequesterID > 0 {
		err = h.notificationService.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
			UserIDs: []int{ticketEntity.RequesterID},
			Type:    notificationType,
			Channel: channel,
			Content: content,
		}, ticketEntity.TenantID)
		if err != nil {
			h.logger.Warnw("Failed to notify requester", "ticket_id", ticketID, "requester_id", ticketEntity.RequesterID, "error", err)
		}
	}

	h.logger.Infow("Requester notified via BPMN", "ticket_id", ticketID, "requester_id", ticketEntity.RequesterID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("已通知请求人 %d", ticketEntity.RequesterID),
	}, nil
}

// notifyHandler 通知处理人
func (h *TicketServiceTaskHandler) notifyHandler(ctx context.Context, ticketID int, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	// 获取通知内容
	notificationType, _ := variables["notification_type"].(string)
	if notificationType == "" {
		notificationType = "assignment"
	}
	content, _ := variables["content"].(string)
	channel, _ := variables["channel"].(string)
	if channel == "" {
		channel = "in_app"
	}

	// 获取工单信息
	ticketEntity, err := h.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	// 发送通知给处理人
	if ticketEntity.AssigneeID > 0 {
		err = h.notificationService.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
			UserIDs: []int{ticketEntity.AssigneeID},
			Type:    notificationType,
			Channel: channel,
			Content: content,
		}, ticketEntity.TenantID)
		if err != nil {
			h.logger.Warnw("Failed to notify handler", "ticket_id", ticketID, "handler_id", ticketEntity.AssigneeID, "error", err)
		}
	}

	h.logger.Infow("Handler notified via BPMN", "ticket_id", ticketID, "handler_id", ticketEntity.AssigneeID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("已通知处理人 %d", ticketEntity.AssigneeID),
	}, nil
}

// escalateTicket 升级工单
func (h *TicketServiceTaskHandler) escalateTicket(ctx context.Context, ticketID int, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	// 获取升级优先级
	escalateTo, _ := variables["escalate_to"].(string)
	if escalateTo == "" {
		escalateTo = "high"
	}
	escalationReason, _ := variables["escalation_reason"].(string)

	// 获取工单信息
	ticketEntity, err := h.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	// 升级工单：更新优先级和状态
	_, err = h.client.Ticket.UpdateOneID(ticketID).
		SetPriority(escalateTo).
		SetStatus("escalated").
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("升级工单失败: %w", err)
	}

	// 通知管理员或升级处理人
	adminIDs, _ := variables["notify_admin_ids"].([]int)
	if len(adminIDs) > 0 {
		content := fmt.Sprintf("工单 %s (#%s) 已升级，原因：%s", ticketEntity.Title, ticketEntity.TicketNumber, escalationReason)
		for _, adminID := range adminIDs {
			_ = h.notificationService.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
				UserIDs: []int{adminID},
				Type:    "escalation",
				Channel: "in_app",
				Content: content,
			}, ticketEntity.TenantID)
		}
	}

	h.logger.Infow("Ticket escalated via BPMN", "ticket_id", ticketID, "escalated_to", escalateTo, "reason", escalationReason)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("工单 %d 已升级为 %s", ticketID, escalateTo),
	}, nil
}

// assignTicket 分配工单
func (h *TicketServiceTaskHandler) assignTicket(ctx context.Context, ticketID int, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	// 获取分配的处理人ID
	assigneeIDFloat, ok := variables["assignee_id"].(float64)
	assigneeID := int(assigneeIDFloat)
	if !ok || assigneeID == 0 {
		// 尝试从变量中获取
		assigneeID, _ = variables["assignee_id"].(int)
	}

	if assigneeID == 0 {
		return nil, fmt.Errorf("分配失败: 未指定处理人ID")
	}

	// 获取工单信息
	ticketEntity, err := h.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	// 更新工单分配
	_, err = h.client.Ticket.UpdateOneID(ticketID).
		SetAssigneeID(assigneeID).
		SetStatus("assigned").
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("分配工单失败: %w", err)
	}

	// 发送通知给新的处理人
	notifyContent, _ := variables["notify_content"].(string)
	if notifyContent == "" {
		notifyContent = fmt.Sprintf("您被分配了一个新工单：%s (#%s)", ticketEntity.Title, ticketEntity.TicketNumber)
	}
	_ = h.notificationService.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: []int{assigneeID},
		Type:    "assignment",
		Channel: "in_app",
		Content: notifyContent,
	}, ticketEntity.TenantID)

	h.logger.Infow("Ticket assigned via BPMN", "ticket_id", ticketID, "assignee_id", assigneeID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("工单 %d 已分配给用户 %d", ticketID, assigneeID),
	}, nil
}

// 确保 TicketServiceTaskHandler 实现了 ServiceTaskHandlerInterface
var _ ServiceTaskHandlerInterface = (*TicketServiceTaskHandler)(nil)
