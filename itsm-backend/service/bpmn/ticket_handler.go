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

// TicketServiceTaskHandler 工单服务任务处理器
//
// 通知一律走 NotificationHandler 的可靠出箱（operational_commands +
// notification.deliver worker），不注入「可替换的通知服务」：此前的
// TicketNotificationServiceInterface 默认实现恒返回 nil，生产从未被注入过
// 真实现（RegisterTicketHandlerWithNotification 零调用方），于是通知失败
// 也记成 ServiceTask 成功、worker 不重试（codex 审计 P2-08）。
type TicketServiceTaskHandler struct {
	HandlerBase
	client   *ent.Client
	logger   *zap.SugaredLogger
	notifier *NotificationHandler
}

// NewTicketServiceTaskHandler 创建工单处理器
func NewTicketServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *TicketServiceTaskHandler {
	handler := &TicketServiceTaskHandler{
		client:   client,
		logger:   logger,
		notifier: NewNotificationHandler(client, logger),
	}
	return handler
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
		return h.notifyRequester(ctx, task, businessID, variables)
	case "notify_handler":
		// 通知处理人
		return h.notifyHandler(ctx, task, businessID, variables)
	case "escalate":
		// 升级处理
		return h.escalateTicket(ctx, task, businessID, variables)
	case "assign":
		// 分配任务
		return h.assignTicket(ctx, task, businessID, variables)
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

// notifyFailure 输出结构化错误日志并返回明确失败的服务任务结果。
// 语义与 NotificationHandler.failure 一致：调用方（command worker）只以
// error 判定失败并清除执行意图、安排重试，Success=false 同时把「没投递」
// 这个事实写进回执，绝不与「已通知」共用一种成功返回。
func (h *TicketServiceTaskHandler) notifyFailure(cause error, action string, ticketID int) (*dto.ServiceTaskResult, error) {
	h.logger.Errorw("BPMN ticket_task notification failed",
		"action", action, "ticket_id", ticketID, "error", cause)
	return &dto.ServiceTaskResult{
		Success: false,
		Message: fmt.Sprintf("通知发送失败: %v", cause),
	}, cause
}

// notifyTicketRecipients 把一次工单通知写入可靠投递队列（operational_commands
// + notification.deliver worker），与 notification_task 共用同一条出箱通道。
//
// 失败必须上抛：入队失败说明通知没有发生，ServiceTask 需要失败让 worker
// 重试；返回成功会让「完成凭据」证明一件没发生的事（codex 审计 P2-08）。
func (h *TicketServiceTaskHandler) notifyTicketRecipients(
	ctx context.Context,
	task *ent.ProcessTask,
	variables map[string]interface{},
	ticketEntity *ent.Ticket,
	recipientIDs []int,
	channel, notificationType, content, action string,
) (int, error) {
	if ticketEntity.TenantID <= 0 {
		return 0, fmt.Errorf("工单 %d 缺少租户归属，无法发送通知", ticketEntity.ID)
	}
	if len(recipientIDs) == 0 {
		return 0, fmt.Errorf("工单 %d 没有可通知的收件人", ticketEntity.ID)
	}
	if strings.TrimSpace(content) == "" {
		return 0, fmt.Errorf("工单 %d 的通知内容为空", ticketEntity.ID)
	}
	return h.notifier.enqueueDeliveries(ctx, deliveryRequest{
		TenantID:         ticketEntity.TenantID,
		ResourceType:     "ticket",
		ResourceID:       ticketEntity.ID,
		Channel:          channel,
		NotificationType: notificationType,
		Content:          strings.TrimSpace(content),
		OccurrenceKey:    occurrenceKey(task, variables, action),
		RecipientIDs:     recipientIDs,
	})
}

// notifyRequester 通知请求人
func (h *TicketServiceTaskHandler) notifyRequester(ctx context.Context, task *ent.ProcessTask, ticketID int, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
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

	if ticketEntity.RequesterID <= 0 {
		return h.notifyFailure(fmt.Errorf("工单 %d 未关联请求人", ticketID), "notify_requester", ticketID)
	}

	enqueued, err := h.notifyTicketRecipients(ctx, task, variables, ticketEntity,
		[]int{ticketEntity.RequesterID}, channel, notificationType, content, "notify_requester")
	if err != nil {
		return h.notifyFailure(err, "notify_requester", ticketID)
	}

	h.logger.Infow("Requester notification enqueued via BPMN",
		"ticket_id", ticketID, "requester_id", ticketEntity.RequesterID, "enqueued", enqueued)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("请求人 %d 的通知已入队投递", ticketEntity.RequesterID),
		OutputVars: map[string]interface{}{
			"recipient_ids":  []int{ticketEntity.RequesterID},
			"enqueued_count": enqueued,
		},
	}, nil
}

// notifyHandler 通知处理人
func (h *TicketServiceTaskHandler) notifyHandler(ctx context.Context, task *ent.ProcessTask, ticketID int, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
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

	if ticketEntity.AssigneeID <= 0 {
		return h.notifyFailure(fmt.Errorf("工单 %d 未分配处理人", ticketID), "notify_handler", ticketID)
	}

	enqueued, err := h.notifyTicketRecipients(ctx, task, variables, ticketEntity,
		[]int{ticketEntity.AssigneeID}, channel, notificationType, content, "notify_handler")
	if err != nil {
		return h.notifyFailure(err, "notify_handler", ticketID)
	}

	h.logger.Infow("Handler notification enqueued via BPMN",
		"ticket_id", ticketID, "handler_id", ticketEntity.AssigneeID, "enqueued", enqueued)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("处理人 %d 的通知已入队投递", ticketEntity.AssigneeID),
		OutputVars: map[string]interface{}{
			"recipient_ids":  []int{ticketEntity.AssigneeID},
			"enqueued_count": enqueued,
		},
	}, nil
}

// escalateTicket 升级工单
func (h *TicketServiceTaskHandler) escalateTicket(ctx context.Context, task *ent.ProcessTask, ticketID int, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
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

	// 升级 = 提高优先级 + 通知，不改工单状态。
	// tickets.status 的权威词表没有 "escalated"（那是 incident 的状态），
	// 数据库 ck_tickets_status_vocabulary 约束会直接拒绝这个写入——旧实现
	// 在生产上必然失败，只有不带 CHECK 的 SQLite 测试才绿。要让工单进入
	// 可见的「已升级」态需要新增状态值（词表 + 迁移 + 状态机 + 前端），
	// 属于产品决策，不在本次收口范围内；这里先保证动作可用。
	update := h.client.Ticket.UpdateOneID(ticketID)
	if tenantID > 0 {
		update = update.Where(ticket.TenantID(tenantID))
	}
	_, err = update.
		SetPriority(escalateTo).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("升级工单失败: %w", err)
	}

	// 通知管理员或升级处理人：入队失败必须让服务任务失败，
	// 不能像旧实现那样 Warn 之后照常返回成功（codex 审计 P2-08）。
	adminIDs, _ := variables["notify_admin_ids"].([]int)
	if len(adminIDs) > 0 {
		content := fmt.Sprintf("工单 %s (#%s) 已升级，原因：%s", ticketEntity.Title, ticketEntity.TicketNumber, escalationReason)
		if _, err := h.notifyTicketRecipients(ctx, task, variables, ticketEntity,
			adminIDs, "in_app", "escalation", content, "escalate"); err != nil {
			return h.notifyFailure(err, "escalate", ticketID)
		}
	}

	h.logger.Infow("Ticket escalated via BPMN", "ticket_id", ticketID, "escalated_to", escalateTo, "reason", escalationReason)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("工单 %d 已升级为 %s", ticketID, escalateTo),
	}, nil
}

// assignTicket 分配工单
func (h *TicketServiceTaskHandler) assignTicket(ctx context.Context, task *ent.ProcessTask, ticketID int, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
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

	// 更新工单分配。状态迁移必须过权威状态机：终态（closed/cancelled）或
	// 非法来源不得因为一次分配被改回 "assigned"——那等于分配动作复活了
	// 已关闭的工单。迁移不合法时只更新分配人、保留原状态，分配本身仍成功。
	update := h.client.Ticket.UpdateOneID(ticketID)
	if tenantID > 0 {
		update = update.Where(ticket.TenantID(tenantID))
	}
	update = update.
		SetAssigneeID(assigneeID).
		SetUpdatedAt(time.Now())
	if common.IsValidTicketStatusTransition(ticketEntity.Status, common.TicketStatusAssigned) {
		update = update.SetStatus(common.TicketStatusAssigned)
	} else {
		h.logger.Infow("Ticket assign keeps current status (illegal transition to assigned)",
			"ticket_id", ticketID, "current_status", ticketEntity.Status)
	}
	_, err = update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("分配工单失败: %w", err)
	}

	// 发送通知给新的处理人：入队失败必须让服务任务失败（codex 审计 P2-08）。
	notifyContent, _ := variables["notify_content"].(string)
	if notifyContent == "" {
		notifyContent = fmt.Sprintf("您被分配了一个新工单：%s (#%s)", ticketEntity.Title, ticketEntity.TicketNumber)
	}
	if _, err := h.notifyTicketRecipients(ctx, task, variables, ticketEntity,
		[]int{assigneeID}, "in_app", "assignment", notifyContent, "assign"); err != nil {
		return h.notifyFailure(err, "assign", ticketID)
	}

	h.logger.Infow("Ticket assigned via BPMN", "ticket_id", ticketID, "assignee_id", assigneeID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("工单 %d 已分配给用户 %d", ticketID, assigneeID),
	}, nil
}

// 确保 TicketServiceTaskHandler 实现了 ServiceTaskHandlerInterface
var _ ServiceTaskHandlerInterface = (*TicketServiceTaskHandler)(nil)
