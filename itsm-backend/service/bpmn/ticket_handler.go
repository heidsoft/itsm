package bpmn

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"

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
	client              *ent.Client
	logger              *zap.SugaredLogger
	notificationService TicketNotificationServiceInterface
}

// NewTicketServiceTaskHandler 创建工单处理器
func NewTicketServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *TicketServiceTaskHandler {
	handler := &TicketServiceTaskHandler{
		client:              client,
		logger:              logger,
		notificationService: &DefaultTicketNotificationService{},
	}
	return handler
}

// SetNotificationService 设置通知服务
func (h *TicketServiceTaskHandler) SetNotificationService(svc TicketNotificationServiceInterface) {
	h.notificationService = svc
}

func (h *TicketServiceTaskHandler) GetTaskType() string {
	return "ticket_task"
}

func (h *TicketServiceTaskHandler) GetHandlerID() string {
	return "ticket_service_handler"
}

func (h *TicketServiceTaskHandler) getTenantID(ctx context.Context, _ map[string]interface{}) int {
	if tenantID, ok := ctx.Value(BPMNTenantIDContextKey).(int); ok && tenantID > 0 {
		return tenantID
	}
	return 0
}

func (h *TicketServiceTaskHandler) getTicket(ctx context.Context, ticketID int, tenantID int) (*ent.Ticket, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("getTicket: 租户ID无效 (ticketID=%d, tenantID=%d)", ticketID, tenantID)
	}
	return h.client.Ticket.Query().
		Where(ticket.ID(ticketID), ticket.TenantID(tenantID)).
		Only(ctx)
}

// Execute 执行工单服务任务
func (h *TicketServiceTaskHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	// 提取业务ID
	var businessID int
	switch v := variables["business_id"].(type) {
	case int:
		businessID = v
	case float64:
		businessID = int(v)
	default:
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
		return nil, serviceTaskActionError("ticket_task", task, action, ticketServiceTaskActions)
	}
}

// ticketServiceTaskActions 是 ticket_task 支持的全部动作，供错误信息与 Validate 共用。
var ticketServiceTaskActions = []string{
	"update_status", "notify_requester", "notify_handler", "escalate", "assign",
}

// Validate 验证配置
func (h *TicketServiceTaskHandler) Validate(ctx context.Context, config map[string]interface{}) error {
	return validateServiceTaskAction("ticket_task", config, ticketServiceTaskActions)
}

// updateTicketStatus 更新工单状态
func (h *TicketServiceTaskHandler) updateTicketStatus(ctx context.Context, ticketID int, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	// 获取新状态
	newStatus, _ := variables["new_status"].(string)
	if newStatus == "" {
		newStatus = common.TicketStatusInProgress
	}

	// 解析附加字段
	additionalData := make(map[string]interface{})
	if formFields, ok := variables["form_fields"].(map[string]interface{}); ok {
		additionalData["form_fields"] = formFields
	}

	// 租户 fail-closed：拿不到租户上下文不得改工单。
	tenantID := h.getTenantID(ctx, variables)
	if tenantID <= 0 {
		return nil, fmt.Errorf("缺少租户上下文，无法更新工单状态")
	}

	// 状态机权威闸门：BPMN 服务任务与人工流转共用同一张迁移表，
	// 流程配置错误（如把已关闭工单改回 in_progress）必须在服务任务边界被拒。
	current, err := h.getTicket(ctx, ticketID, tenantID)
	if err != nil {
		h.logger.Errorw("Failed to load ticket for status update", "ticket_id", ticketID, "tenant_id", tenantID, "error", err)
		return nil, fmt.Errorf("工单不存在或不属于当前租户: %w", err)
	}
	if !common.IsValidTicketStatusTransition(current.Status, newStatus) {
		h.logger.Errorw("Illegal ticket status transition via BPMN service task",
			"ticket_id", ticketID, "current_status", current.Status, "target_status", newStatus)
		return nil, fmt.Errorf("非法工单状态转换: %s -> %s", current.Status, newStatus)
	}
	if newStatus == common.TicketStatusResolved && strings.TrimSpace(current.Resolution) == "" {
		return nil, fmt.Errorf("解决工单前必须填写解决方案")
	}
	if current.Status == newStatus {
		return &dto.ServiceTaskResult{
			Success:     true,
			Message:     fmt.Sprintf("工单 %d 已处于状态 %s，无需变更", ticketID, newStatus),
			UpdatedData: additionalData,
		}, nil
	}

	// 条件更新（version CAS）：与人工流转一致，读后并发修改按冲突拒绝，
	// 不得无条件覆盖。时间戳语义与 TicketLifecycleService 对齐。
	now := time.Now()
	updater := h.client.Ticket.Update().
		Where(
			ticket.ID(ticketID),
			ticket.TenantIDEQ(tenantID),
			ticket.DeletedAtIsNil(),
			ticket.VersionEQ(current.Version),
		).
		SetStatus(newStatus).
		SetVersion(current.Version + 1).
		SetUpdatedAt(now)
	switch newStatus {
	case common.TicketStatusResolved:
		updater = updater.SetResolvedAt(now).ClearClosedAt()
	case common.TicketStatusClosed:
		updater = updater.SetClosedAt(now)
	default:
		updater = updater.ClearResolvedAt().ClearClosedAt()
	}
	affected, err := updater.Save(ctx)
	if err != nil {
		h.logger.Errorw("Failed to update ticket status", "ticket_id", ticketID, "error", err)
		return nil, fmt.Errorf("更新工单状态失败: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("工单状态并发冲突，请重试")
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
	tenantID := h.getTenantID(ctx, variables)
	ticketEntity, err := h.getTicket(ctx, ticketID, tenantID)
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
	tenantID := h.getTenantID(ctx, variables)
	ticketEntity, err := h.getTicket(ctx, ticketID, tenantID)
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
	tenantID := h.getTenantID(ctx, variables)
	ticketEntity, err := h.getTicket(ctx, ticketID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	// 升级工单：更新优先级和状态
	update := h.client.Ticket.UpdateOneID(ticketID)
	if tenantID > 0 {
		update = update.Where(ticket.TenantID(tenantID))
	}
	_, err = update.
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
			if err := h.notificationService.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
				UserIDs: []int{adminID},
				Type:    "escalation",
				Channel: "in_app",
				Content: content,
			}, ticketEntity.TenantID); err != nil {
				h.logger.Warnw("failed to send escalation notification", "error", err, "ticket_id", ticketID, "admin_id", adminID)
			}
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
	tenantID := h.getTenantID(ctx, variables)
	ticketEntity, err := h.getTicket(ctx, ticketID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	// 更新工单分配
	update := h.client.Ticket.UpdateOneID(ticketID)
	if tenantID > 0 {
		update = update.Where(ticket.TenantID(tenantID))
	}
	_, err = update.
		SetAssigneeID(assigneeID).
		SetStatus(common.TicketStatusAssigned).
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
	if err := h.notificationService.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: []int{assigneeID},
		Type:    "assignment",
		Channel: "in_app",
		Content: notifyContent,
	}, ticketEntity.TenantID); err != nil {
		h.logger.Warnw("failed to send assignment notification", "error", err, "ticket_id", ticketID, "assignee_id", assigneeID)
	}

	h.logger.Infow("Ticket assigned via BPMN", "ticket_id", ticketID, "assignee_id", assigneeID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("工单 %d 已分配给用户 %d", ticketID, assigneeID),
	}, nil
}

// 确保 TicketServiceTaskHandler 实现了 ServiceTaskHandlerInterface
var _ ServiceTaskHandlerInterface = (*TicketServiceTaskHandler)(nil)
