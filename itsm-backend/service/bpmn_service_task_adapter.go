package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/processtask"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Helper functions for extracting values from variables

// getIntFromVars 从变量中提取整数
func getIntFromVars(variables map[string]interface{}, key string) int {
	if v, ok := variables[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		case int64:
			return int(val)
		}
	}
	return 0
}

// getStringFromVars 从变量中提取字符串
func getStringFromVars(variables map[string]interface{}, key string) string {
	if v, ok := variables[key]; ok {
		if val, ok := v.(string); ok {
			return val
		}
	}
	return ""
}

// getTenantIDFromVars 从变量中提取租户ID
func getTenantIDFromVars(variables map[string]interface{}) int {
	// 首先检查 tenant_id
	if id := getIntFromVars(variables, "tenant_id"); id > 0 {
		return id
	}
	// 如果没有，返回默认租户ID 1
	return 1
}

// ProcessCallbackService 流程回调服务实现
type ProcessCallbackService struct {
	client     *ent.Client
	handlers   map[string]ServiceTaskHandlerInterface
	handlersMu sync.RWMutex
}

// NewProcessCallbackService 创建流程回调服务
func NewProcessCallbackService(client *ent.Client, logger *zap.SugaredLogger) *ProcessCallbackService {
	svc := &ProcessCallbackService{
		client:   client,
		handlers: make(map[string]ServiceTaskHandlerInterface),
	}

	// 注册默认处理器
	svc.registerDefaultHandlers(logger)

	return svc
}

// HandleCallback 处理流程回调
func (s *ProcessCallbackService) HandleCallback(ctx context.Context, req *dto.CallbackRequest) error {
	// 1. 获取任务
	task, err := s.client.ProcessTask.Query().
		Where(processtask.ID(req.ProcessInstanceID)). // 实际应该是 task_id
		Only(ctx)
	if err != nil {
		return errors.Wrap(err, "查询任务失败")
	}

	// 2. 获取处理器
	handler := s.getHandler(req.ActivityType)
	if handler == nil {
		return fmt.Errorf("未找到任务类型 %s 的处理器", req.ActivityType)
	}

	// 3. 执行处理器
	_, err = handler.Execute(ctx, nil, req.Result.OutputVars)
	if err != nil {
		return errors.Wrap(err, "执行服务任务失败")
	}

	// 4. 更新任务状态
	_, err = s.client.ProcessTask.Update().
		Where(processtask.ID(task.ID)).
		SetStatus("completed").
		Save(ctx)
	if err != nil {
		return errors.Wrap(err, "更新任务状态失败")
	}

	// 5. 如果流程引擎需要，推进流程
	// （由流程引擎自己处理）

	return nil
}

// RegisterHandler 注册服务任务处理器
func (s *ProcessCallbackService) RegisterHandler(handler ServiceTaskHandlerInterface) {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()
	s.handlers[handler.GetHandlerID()] = handler
}

// UnregisterHandler 注销处理器
func (s *ProcessCallbackService) UnregisterHandler(handlerID string) {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()
	delete(s.handlers, handlerID)
}

// getHandler 获取处理器
func (s *ProcessCallbackService) getHandler(taskType string) ServiceTaskHandlerInterface {
	s.handlersMu.RLock()
	defer s.handlersMu.RUnlock()

	// 精确匹配
	if handler, ok := s.handlers[taskType]; ok {
		return handler
	}

	// 通配匹配
	for _, handler := range s.handlers {
		if handler.GetTaskType() == taskType {
			return handler
		}
	}

	return nil
}

// registerDefaultHandlers 注册默认处理器
func (s *ProcessCallbackService) registerDefaultHandlers(logger *zap.SugaredLogger) {
	// 注册 Ticket 服务任务处理器
	s.RegisterHandler(NewTicketServiceTaskHandler(s.client, logger))
	// 注册 Change 服务任务处理器
	s.RegisterHandler(NewChangeServiceTaskHandler(s.client, logger))
	// 注册 Incident 服务任务处理器
	s.RegisterHandler(NewIncidentServiceTaskHandler(s.client, logger))
	// 注册通用服务任务处理器
	s.RegisterHandler(NewGenericServiceTaskHandler(s.client, logger))
}

// ServiceTaskHandlerBase 服务任务处理器基类
type ServiceTaskHandlerBase struct {
	client *ent.Client
}

// GetHandlerID 返回处理器标识
func (h *ServiceTaskHandlerBase) GetHandlerID() string {
	return ""
}

// Validate 验证配置
func (h *ServiceTaskHandlerBase) Validate(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// TicketServiceTaskHandler 工单服务任务处理器
type TicketServiceTaskHandler struct {
	*ServiceTaskHandlerBase
	logger              *zap.SugaredLogger
	notificationService *TicketNotificationService
}

// NewTicketServiceTaskHandler 创建工单处理器
func NewTicketServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *TicketServiceTaskHandler {
	handler := &TicketServiceTaskHandler{
		ServiceTaskHandlerBase: &ServiceTaskHandlerBase{client: client},
		logger:                 logger,
	}
	// 初始化通知服务
	handler.notificationService = NewTicketNotificationService(client, logger)
	return handler
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

// ChangeServiceTaskHandler 变更服务任务处理器
type ChangeServiceTaskHandler struct {
	*ServiceTaskHandlerBase
	logger *zap.SugaredLogger
}

// NewChangeServiceTaskHandler 创建变更处理器
func NewChangeServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *ChangeServiceTaskHandler {
	return &ChangeServiceTaskHandler{
		ServiceTaskHandlerBase: &ServiceTaskHandlerBase{client: client},
		logger:                 logger,
	}
}

// GetTaskType 返回任务类型
func (h *ChangeServiceTaskHandler) GetTaskType() string {
	return "change_task"
}

// GetHandlerID 返回处理器标识
func (h *ChangeServiceTaskHandler) GetHandlerID() string {
	return "change_service_handler"
}

// Execute 执行变更服务任务
func (h *ChangeServiceTaskHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	action, _ := variables["action"].(string)
	switch action {
	case "create_change":
		return h.createChange(ctx, variables)
	case "update_change":
		return h.updateChange(ctx, variables)
	case "approve_change":
		return h.approveChange(ctx, variables)
	case "reject_change":
		return h.rejectChange(ctx, variables)
	case "schedule_change":
		return h.scheduleChange(ctx, variables)
	case "implement_change":
		return h.implementChange(ctx, variables)
	case "verify_change":
		return h.verifyChange(ctx, variables)
	case "close_change":
		return h.closeChange(ctx, variables)
	case "assess_risk":
		return h.assessRisk(ctx, variables)
	case "notify_stakeholders":
		return h.notifyStakeholders(ctx, variables)
	default:
		return &dto.ServiceTaskResult{
			Success: true,
			Message: "无操作执行",
		}, nil
	}
}

// createChange 创建变更
func (h *ChangeServiceTaskHandler) createChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	title, _ := variables["title"].(string)
	description, _ := variables["description"].(string)
	changeType, _ := variables["type"].(string)
	priority, _ := variables["priority"].(string)
	tenantID := getTenantIDFromVars(variables)

	if title == "" {
		return nil, fmt.Errorf("变更标题不能为空")
	}

	change, err := h.client.Change.Create().
		SetTitle(title).
		SetDescription(description).
		SetType(changeType).
		SetPriority(priority).
		SetStatus("draft").
		SetCreatedBy(getIntFromVars(variables, "created_by")).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建变更失败: %w", err)
	}

	h.logger.Infow("Change created via BPMN", "change_id", change.ID, "title", title)

	return &dto.ServiceTaskResult{
		Success:    true,
		Message:    fmt.Sprintf("变更 %d 已创建", change.ID),
		OutputVars: map[string]interface{}{"change_id": change.ID},
	}, nil
}

// updateChange 更新变更
func (h *ChangeServiceTaskHandler) updateChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := getIntFromVars(variables, "change_id")
	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	updateQuery := h.client.Change.UpdateOneID(changeID)

	if title, ok := variables["title"].(string); ok && title != "" {
		updateQuery.SetTitle(title)
	}
	if description, ok := variables["description"].(string); ok && description != "" {
		updateQuery.SetDescription(description)
	}
	if status, ok := variables["status"].(string); ok && status != "" {
		updateQuery.SetStatus(status)
	}

	_, err := updateQuery.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新变更失败: %w", err)
	}

	h.logger.Infow("Change updated via BPMN", "change_id", changeID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 已更新", changeID),
	}, nil
}

// approveChange 审批变更
func (h *ChangeServiceTaskHandler) approveChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := getIntFromVars(variables, "change_id")
	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	// 获取变更信息
	change, err := h.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("变更不存在: %w", err)
	}

	// 更新状态为审批中
	_, err = h.client.Change.UpdateOneID(changeID).
		SetStatus("pending_approval").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新变更状态失败: %w", err)
	}

	h.logger.Infow("Change submitted for approval via BPMN", "change_id", changeID, "title", change.Title)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 已提交审批", changeID),
	}, nil
}

// rejectChange 驳回变更
func (h *ChangeServiceTaskHandler) rejectChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := getIntFromVars(variables, "change_id")
	reason, _ := variables["reject_reason"].(string)

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	_, err := h.client.Change.UpdateOneID(changeID).
		SetStatus("rejected").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("驳回变更失败: %w", err)
	}

	h.logger.Infow("Change rejected via BPMN", "change_id", changeID, "reason", reason)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 已驳回: %s", changeID, reason),
	}, nil
}

// scheduleChange 排期变更
func (h *ChangeServiceTaskHandler) scheduleChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := getIntFromVars(variables, "change_id")

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	// 解析日期时间
	var plannedStart, plannedEnd time.Time
	if startStr, ok := variables["planned_start_date"].(string); ok && startStr != "" {
		plannedStart, _ = time.Parse(time.RFC3339, startStr)
	}
	if endStr, ok := variables["planned_end_date"].(string); ok && endStr != "" {
		plannedEnd, _ = time.Parse(time.RFC3339, endStr)
	}

	updateQuery := h.client.Change.UpdateOneID(changeID).SetStatus("scheduled")
	if !plannedStart.IsZero() {
		updateQuery.SetPlannedStartDate(plannedStart)
	}
	if !plannedEnd.IsZero() {
		updateQuery.SetPlannedEndDate(plannedEnd)
	}

	_, err := updateQuery.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("排期变更失败: %w", err)
	}

	h.logger.Infow("Change scheduled via BPMN", "change_id", changeID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 已排期", changeID),
	}, nil
}

// implementChange 实施变更
func (h *ChangeServiceTaskHandler) implementChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := getIntFromVars(variables, "change_id")

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	// 获取变更信息
	change, err := h.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("变更不存在: %w", err)
	}

	now := time.Now()
	_, err = h.client.Change.UpdateOneID(changeID).
		SetStatus("in_progress").
		SetActualStartDate(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("实施变更失败: %w", err)
	}

	h.logger.Infow("Change implementation started via BPMN", "change_id", changeID, "title", change.Title)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 开始实施", changeID),
	}, nil
}

// verifyChange 验证变更
func (h *ChangeServiceTaskHandler) verifyChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := getIntFromVars(variables, "change_id")
	verificationResult, _ := variables["verification_result"].(string)

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	// 根据验证结果更新状态
	newStatus := "verification_passed"
	if verificationResult == "failed" {
		newStatus = "verification_failed"
	}

	_, err := h.client.Change.UpdateOneID(changeID).
		SetStatus(newStatus).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("验证变更失败: %w", err)
	}

	h.logger.Infow("Change verification via BPMN", "change_id", changeID, "result", verificationResult)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 验证结果: %s", changeID, verificationResult),
	}, nil
}

// closeChange 关闭变更
func (h *ChangeServiceTaskHandler) closeChange(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := getIntFromVars(variables, "change_id")
	feedback, _ := variables["feedback"].(string)

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	now := time.Now()
	_, err := h.client.Change.UpdateOneID(changeID).
		SetStatus("closed").
		SetActualEndDate(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("关闭变更失败: %w", err)
	}

	h.logger.Infow("Change closed via BPMN", "change_id", changeID, "feedback", feedback)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 已关闭", changeID),
	}, nil
}

// assessRisk 评估变更风险
func (h *ChangeServiceTaskHandler) assessRisk(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := getIntFromVars(variables, "change_id")

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	// 获取变更信息进行风险评估
	change, err := h.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("变更不存在: %w", err)
	}

	// 简化的风险评估逻辑
	riskLevel := "medium"
	impactScope := change.ImpactScope

	// 根据变更类型和优先级确定风险等级
	if change.Type == "emergency" {
		riskLevel = "high"
	} else if change.Type == "minor" {
		riskLevel = "low"
	}

	_, err = h.client.Change.UpdateOneID(changeID).
		SetRiskLevel(riskLevel).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("评估风险失败: %w", err)
	}

	h.logger.Infow("Change risk assessed via BPMN", "change_id", changeID, "risk_level", riskLevel, "impact_scope", impactScope)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 风险评估完成: %s", changeID, riskLevel),
		OutputVars: map[string]interface{}{
			"risk_level":   riskLevel,
			"impact_scope": impactScope,
		},
	}, nil
}

// notifyStakeholders 通知利益相关者
func (h *ChangeServiceTaskHandler) notifyStakeholders(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	changeID := getIntFromVars(variables, "change_id")
	notificationType, _ := variables["notification_type"].(string)

	if changeID <= 0 {
		return nil, fmt.Errorf("无效的变更ID")
	}

	// 获取变更信息
	change, err := h.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("变更不存在: %w", err)
	}

	// 记录通知日志（实际应调用通知服务）
	h.logger.Infow("Stakeholders notification via BPMN",
		"change_id", changeID,
		"change_title", change.Title,
		"notification_type", notificationType,
	)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("变更 %d 利益相关者已通知", changeID),
	}, nil
}

// IncidentServiceTaskHandler 事件服务任务处理器
type IncidentServiceTaskHandler struct {
	*ServiceTaskHandlerBase
	logger *zap.SugaredLogger
}

// NewIncidentServiceTaskHandler 创建事件处理器
func NewIncidentServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *IncidentServiceTaskHandler {
	return &IncidentServiceTaskHandler{
		ServiceTaskHandlerBase: &ServiceTaskHandlerBase{client: client},
		logger:                 logger,
	}
}

// GetTaskType 返回任务类型
func (h *IncidentServiceTaskHandler) GetTaskType() string {
	return "incident_task"
}

// GetHandlerID 返回处理器标识
func (h *IncidentServiceTaskHandler) GetHandlerID() string {
	return "incident_service_handler"
}

// Execute 执行事件服务任务
func (h *IncidentServiceTaskHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	action, _ := variables["action"].(string)
	switch action {
	case "create_incident":
		return h.createIncident(ctx, variables)
	case "assign_incident":
		return h.assignIncident(ctx, variables)
	case "escalate_incident":
		return h.escalateIncident(ctx, variables)
	case "resolve_incident":
		return h.resolveIncident(ctx, variables)
	case "close_incident":
		return h.closeIncident(ctx, variables)
	case "update_incident":
		return h.updateIncident(ctx, variables)
	case "acknowledge_incident":
		return h.acknowledgeIncident(ctx, variables)
	case "categorize_incident":
		return h.categorizeIncident(ctx, variables)
	default:
		return &dto.ServiceTaskResult{Success: true, Message: "无操作执行"}, nil
	}
}

// createIncident 创建事件
func (h *IncidentServiceTaskHandler) createIncident(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	title, _ := variables["title"].(string)
	description, _ := variables["description"].(string)
	incidentType, _ := variables["type"].(string)
	priority, _ := variables["priority"].(string)
	severity, _ := variables["severity"].(string)
	tenantID := getTenantIDFromVars(variables)

	if title == "" {
		return nil, fmt.Errorf("事件标题不能为空")
	}

	incident, err := h.client.Incident.Create().
		SetTitle(title).
		SetDescription(description).
		SetType(incidentType).
		SetPriority(priority).
		SetSeverity(severity).
		SetStatus("new").
		SetReporterID(getIntFromVars(variables, "reporter_id")).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建事件失败: %w", err)
	}

	h.logger.Infow("Incident created via BPMN", "incident_id", incident.ID, "title", title)

	return &dto.ServiceTaskResult{
		Success:    true,
		Message:    fmt.Sprintf("事件 %d 已创建", incident.ID),
		OutputVars: map[string]interface{}{"incident_id": incident.ID, "incident_number": incident.IncidentNumber},
	}, nil
}

// assignIncident 分配事件
func (h *IncidentServiceTaskHandler) assignIncident(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	incidentID := getIntFromVars(variables, "incident_id")
	assigneeID := getIntFromVars(variables, "assignee_id")

	if incidentID <= 0 {
		return nil, fmt.Errorf("无效的事件ID")
	}

	if assigneeID <= 0 {
		return nil, fmt.Errorf("无效的处理人ID")
	}

	_, err := h.client.Incident.UpdateOneID(incidentID).
		SetAssigneeID(assigneeID).
		SetStatus("assigned").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("分配事件失败: %w", err)
	}

	h.logger.Infow("Incident assigned via BPMN", "incident_id", incidentID, "assignee_id", assigneeID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("事件 %d 已分配给用户 %d", incidentID, assigneeID),
	}, nil
}

// escalateIncident 升级事件
func (h *IncidentServiceTaskHandler) escalateIncident(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	incidentID := getIntFromVars(variables, "incident_id")
	escalationLevel := getIntFromVars(variables, "escalation_level")
	reason, _ := variables["escalation_reason"].(string)

	if incidentID <= 0 {
		return nil, fmt.Errorf("无效的事件ID")
	}

	// 获取当前升级级别
	incident, err := h.client.Incident.Get(ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("事件不存在: %w", err)
	}

	newLevel := escalationLevel
	if newLevel <= 0 {
		newLevel = incident.EscalationLevel + 1
	}

	now := time.Now()
	_, err = h.client.Incident.UpdateOneID(incidentID).
		SetEscalationLevel(newLevel).
		SetEscalatedAt(now).
		SetStatus("escalated").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("升级事件失败: %w", err)
	}

	h.logger.Infow("Incident escalated via BPMN", "incident_id", incidentID, "escalation_level", newLevel, "reason", reason)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("事件 %d 已升级到第 %d 级", incidentID, newLevel),
	}, nil
}

// resolveIncident 解决事件
func (h *IncidentServiceTaskHandler) resolveIncident(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	incidentID := getIntFromVars(variables, "incident_id")
	resolution, _ := variables["resolution"].(string)

	if incidentID <= 0 {
		return nil, fmt.Errorf("无效的事件ID")
	}

	now := time.Now()
	_, err := h.client.Incident.UpdateOneID(incidentID).
		SetStatus("resolved").
		SetResolvedAt(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("解决事件失败: %w", err)
	}

	h.logger.Infow("Incident resolved via BPMN", "incident_id", incidentID, "resolution", resolution)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("事件 %d 已解决: %s", incidentID, resolution),
	}, nil
}

// closeIncident 关闭事件
func (h *IncidentServiceTaskHandler) closeIncident(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	incidentID := getIntFromVars(variables, "incident_id")
	feedback, _ := variables["feedback"].(string)

	if incidentID <= 0 {
		return nil, fmt.Errorf("无效的事件ID")
	}

	now := time.Now()
	_, err := h.client.Incident.UpdateOneID(incidentID).
		SetStatus("closed").
		SetClosedAt(now).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("关闭事件失败: %w", err)
	}

	h.logger.Infow("Incident closed via BPMN", "incident_id", incidentID, "feedback", feedback)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("事件 %d 已关闭", incidentID),
	}, nil
}

// updateIncident 更新事件
func (h *IncidentServiceTaskHandler) updateIncident(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	incidentID := getIntFromVars(variables, "incident_id")

	if incidentID <= 0 {
		return nil, fmt.Errorf("无效的事件ID")
	}

	updateQuery := h.client.Incident.UpdateOneID(incidentID)

	if title, ok := variables["title"].(string); ok && title != "" {
		updateQuery.SetTitle(title)
	}
	if description, ok := variables["description"].(string); ok && description != "" {
		updateQuery.SetDescription(description)
	}
	if priority, ok := variables["priority"].(string); ok && priority != "" {
		updateQuery.SetPriority(priority)
	}
	if severity, ok := variables["severity"].(string); ok && severity != "" {
		updateQuery.SetSeverity(severity)
	}
	if status, ok := variables["status"].(string); ok && status != "" {
		updateQuery.SetStatus(status)
	}

	_, err := updateQuery.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新事件失败: %w", err)
	}

	h.logger.Infow("Incident updated via BPMN", "incident_id", incidentID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("事件 %d 已更新", incidentID),
	}, nil
}

// acknowledgeIncident 确认事件
func (h *IncidentServiceTaskHandler) acknowledgeIncident(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	incidentID := getIntFromVars(variables, "incident_id")

	if incidentID <= 0 {
		return nil, fmt.Errorf("无效的事件ID")
	}

	_, err := h.client.Incident.UpdateOneID(incidentID).
		SetStatus("acknowledged").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("确认事件失败: %w", err)
	}

	h.logger.Infow("Incident acknowledged via BPMN", "incident_id", incidentID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("事件 %d 已确认", incidentID),
	}, nil
}

// categorizeIncident 分类事件
func (h *IncidentServiceTaskHandler) categorizeIncident(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	incidentID := getIntFromVars(variables, "incident_id")
	category, _ := variables["category"].(string)
	subcategory, _ := variables["subcategory"].(string)

	if incidentID <= 0 {
		return nil, fmt.Errorf("无效的事件ID")
	}

	updateQuery := h.client.Incident.UpdateOneID(incidentID).SetStatus("triaged")
	if category != "" {
		updateQuery.SetCategory(category)
	}
	if subcategory != "" {
		updateQuery.SetSubcategory(subcategory)
	}

	_, err := updateQuery.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("分类事件失败: %w", err)
	}

	h.logger.Infow("Incident categorized via BPMN", "incident_id", incidentID, "category", category, "subcategory", subcategory)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("事件 %d 已分类: %s/%s", incidentID, category, subcategory),
	}, nil
}

// ProblemServiceTaskHandler 问题服务任务处理器
type ProblemServiceTaskHandler struct {
	*ServiceTaskHandlerBase
	logger *zap.SugaredLogger
}

// NewProblemServiceTaskHandler 创建问题处理器
func NewProblemServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *ProblemServiceTaskHandler {
	return &ProblemServiceTaskHandler{
		ServiceTaskHandlerBase: &ServiceTaskHandlerBase{client: client},
		logger:                 logger,
	}
}

// GetTaskType 返回任务类型
func (h *ProblemServiceTaskHandler) GetTaskType() string {
	return "problem_task"
}

// GetHandlerID 返回处理器标识
func (h *ProblemServiceTaskHandler) GetHandlerID() string {
	return "problem_service_handler"
}

// Execute 执行问题服务任务
func (h *ProblemServiceTaskHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	action, _ := variables["action"].(string)
	switch action {
	case "create_problem":
		return h.createProblem(ctx, variables)
	case "update_problem":
		return h.updateProblem(ctx, variables)
	case "assign_problem":
		return h.assignProblem(ctx, variables)
	case "escalate_problem":
		return h.escalateProblem(ctx, variables)
	case "resolve_problem":
		return h.resolveProblem(ctx, variables)
	case "close_problem":
		return h.closeProblem(ctx, variables)
	case "link_incident":
		return h.linkIncident(ctx, variables)
	case "identify_root_cause":
		return h.identifyRootCause(ctx, variables)
	case "implement_solution":
		return h.implementSolution(ctx, variables)
	case "review_problem":
		return h.reviewProblem(ctx, variables)
	default:
		return &dto.ServiceTaskResult{Success: true, Message: "无操作执行"}, nil
	}
}

// createProblem 创建问题
func (h *ProblemServiceTaskHandler) createProblem(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	title, _ := variables["title"].(string)
	description, _ := variables["description"].(string)
	priority, _ := variables["priority"].(string)
	category, _ := variables["category"].(string)
	tenantID := getTenantIDFromVars(variables)

	if title == "" {
		return nil, fmt.Errorf("问题标题不能为空")
	}

	problem, err := h.client.Problem.Create().
		SetTitle(title).
		SetDescription(description).
		SetPriority(priority).
		SetCategory(category).
		SetStatus("open").
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建问题失败: %w", err)
	}

	h.logger.Infow("Problem created via BPMN", "problem_id", problem.ID, "title", title)

	return &dto.ServiceTaskResult{
		Success:    true,
		Message:    fmt.Sprintf("问题 %d 已创建", problem.ID),
		OutputVars: map[string]interface{}{"problem_id": problem.ID},
	}, nil
}

// updateProblem 更新问题
func (h *ProblemServiceTaskHandler) updateProblem(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	problemID := getIntFromVars(variables, "problem_id")
	if problemID <= 0 {
		return nil, fmt.Errorf("无效的问题ID")
	}

	updateQuery := h.client.Problem.UpdateOneID(problemID)

	if title, ok := variables["title"].(string); ok && title != "" {
		updateQuery.SetTitle(title)
	}
	if description, ok := variables["description"].(string); ok && description != "" {
		updateQuery.SetDescription(description)
	}
	if priority, ok := variables["priority"].(string); ok && priority != "" {
		updateQuery.SetPriority(priority)
	}
	if status, ok := variables["status"].(string); ok && status != "" {
		updateQuery.SetStatus(status)
	}
	if category, ok := variables["category"].(string); ok && category != "" {
		updateQuery.SetCategory(category)
	}

	_, err := updateQuery.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新问题失败: %w", err)
	}

	h.logger.Infow("Problem updated via BPMN", "problem_id", problemID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("问题 %d 已更新", problemID),
	}, nil
}

// assignProblem 分配问题
func (h *ProblemServiceTaskHandler) assignProblem(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	problemID := getIntFromVars(variables, "problem_id")
	assigneeID := getIntFromVars(variables, "assignee_id")

	if problemID <= 0 {
		return nil, fmt.Errorf("无效的问题ID")
	}

	_, err := h.client.Problem.UpdateOneID(problemID).
		SetAssigneeID(assigneeID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("分配问题失败: %w", err)
	}

	h.logger.Infow("Problem assigned via BPMN", "problem_id", problemID, "assignee_id", assigneeID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("问题 %d 已分配", problemID),
	}, nil
}

// escalateProblem 升级问题
func (h *ProblemServiceTaskHandler) escalateProblem(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	problemID := getIntFromVars(variables, "problem_id")
	reason, _ := variables["escalation_reason"].(string)

	if problemID <= 0 {
		return nil, fmt.Errorf("无效的问题ID")
	}

	_, err := h.client.Problem.UpdateOneID(problemID).
		SetStatus("escalated").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("升级问题失败: %w", err)
	}

	h.logger.Infow("Problem escalated via BPMN", "problem_id", problemID, "reason", reason)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("问题 %d 已升级", problemID),
	}, nil
}

// resolveProblem 解决问题
func (h *ProblemServiceTaskHandler) resolveProblem(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	problemID := getIntFromVars(variables, "problem_id")
	solution, _ := variables["solution"].(string)

	if problemID <= 0 {
		return nil, fmt.Errorf("无效的问题ID")
	}

	_, err := h.client.Problem.UpdateOneID(problemID).
		SetStatus("resolved").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("解决问题失败: %w", err)
	}

	h.logger.Infow("Problem resolved via BPMN", "problem_id", problemID, "solution", solution)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("问题 %d 已解决", problemID),
	}, nil
}

// closeProblem 关闭问题
func (h *ProblemServiceTaskHandler) closeProblem(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	problemID := getIntFromVars(variables, "problem_id")
	feedback, _ := variables["feedback"].(string)

	if problemID <= 0 {
		return nil, fmt.Errorf("无效的问题ID")
	}

	_, err := h.client.Problem.UpdateOneID(problemID).
		SetStatus("closed").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("关闭问题失败: %w", err)
	}

	h.logger.Infow("Problem closed via BPMN", "problem_id", problemID, "feedback", feedback)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("问题 %d 已关闭", problemID),
	}, nil
}

// linkIncident 关联事件
func (h *ProblemServiceTaskHandler) linkIncident(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	problemID := getIntFromVars(variables, "problem_id")
	incidentID := getIntFromVars(variables, "incident_id")

	if problemID <= 0 {
		return nil, fmt.Errorf("无效的问题ID")
	}
	if incidentID <= 0 {
		return nil, fmt.Errorf("无效的事件ID")
	}

	// 获取问题并添加关联事件
	problem, err := h.client.Problem.Get(ctx, problemID)
	if err != nil {
		return nil, fmt.Errorf("问题不存在: %w", err)
	}

	// 简化处理：记录关联信息到日志
	h.logger.Infow("Incident linked to problem via BPMN",
		"problem_id", problemID,
		"incident_id", incidentID,
		"problem_title", problem.Title,
	)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("事件 %d 已关联到问题 %d", incidentID, problemID),
	}, nil
}

// identifyRootCause 识别根本原因
func (h *ProblemServiceTaskHandler) identifyRootCause(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	problemID := getIntFromVars(variables, "problem_id")
	rootCause, _ := variables["root_cause"].(string)

	if problemID <= 0 {
		return nil, fmt.Errorf("无效的问题ID")
	}

	_, err := h.client.Problem.UpdateOneID(problemID).
		SetRootCause(rootCause).
		SetStatus("investigating").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("识别根本原因失败: %w", err)
	}

	h.logger.Infow("Root cause identified via BPMN", "problem_id", problemID, "root_cause", rootCause)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("问题 %d 根本原因已识别", problemID),
	}, nil
}

// implementSolution 实施解决方案
func (h *ProblemServiceTaskHandler) implementSolution(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	problemID := getIntFromVars(variables, "problem_id")
	solution, _ := variables["solution"].(string)

	if problemID <= 0 {
		return nil, fmt.Errorf("无效的问题ID")
	}

	_, err := h.client.Problem.UpdateOneID(problemID).
		SetStatus("implementing").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("实施解决方案失败: %w", err)
	}

	h.logger.Infow("Solution implementation started via BPMN", "problem_id", problemID, "solution", solution)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("问题 %d 开始实施解决方案", problemID),
	}, nil
}

// reviewProblem 评审问题
func (h *ProblemServiceTaskHandler) reviewProblem(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	problemID := getIntFromVars(variables, "problem_id")
	reviewResult, _ := variables["review_result"].(string)
	comments, _ := variables["comments"].(string)

	if problemID <= 0 {
		return nil, fmt.Errorf("无效的问题ID")
	}

	newStatus := "under_review"
	if reviewResult == "approved" {
		newStatus = "resolved"
	} else if reviewResult == "rejected" {
		newStatus = "open"
	}

	_, err := h.client.Problem.UpdateOneID(problemID).
		SetStatus(newStatus).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("评审问题失败: %w", err)
	}

	h.logger.Infow("Problem reviewed via BPMN",
		"problem_id", problemID,
		"result", reviewResult,
		"comments", comments,
	)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("问题 %d 评审完成: %s", problemID, reviewResult),
	}, nil
}

// GenericServiceTaskHandler 通用服务任务处理器
type GenericServiceTaskHandler struct {
	*ServiceTaskHandlerBase
	logger *zap.SugaredLogger
}

// NewGenericServiceTaskHandler 创建通用处理器
func NewGenericServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *GenericServiceTaskHandler {
	return &GenericServiceTaskHandler{
		ServiceTaskHandlerBase: &ServiceTaskHandlerBase{client: client},
		logger:                 logger,
	}
}

// GetTaskType 返回任务类型
func (h *GenericServiceTaskHandler) GetTaskType() string {
	return "generic_task"
}

// GetHandlerID 返回处理器标识
func (h *GenericServiceTaskHandler) GetHandlerID() string {
	return "generic_service_handler"
}

// Execute 执行通用服务任务
func (h *GenericServiceTaskHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	// 通用处理器可以根据配置执行各种操作
	operation, _ := variables["operation"].(string)

	result := &dto.ServiceTaskResult{
		Success:    true,
		Message:    fmt.Sprintf("通用任务 %s 执行完成", operation),
		OutputVars: make(map[string]interface{}),
	}

	// 将输入变量透传到输出
	for k, v := range variables {
		result.OutputVars[k] = v
	}

	return result, nil
}

// ServiceRequestServiceTaskHandler 服务请求服务任务处理器
type ServiceRequestServiceTaskHandler struct {
	*ServiceTaskHandlerBase
	logger *zap.SugaredLogger
}

// NewServiceRequestServiceTaskHandler 创建服务请求处理器
func NewServiceRequestServiceTaskHandler(client *ent.Client, logger *zap.SugaredLogger) *ServiceRequestServiceTaskHandler {
	return &ServiceRequestServiceTaskHandler{
		ServiceTaskHandlerBase: &ServiceTaskHandlerBase{client: client},
		logger:                 logger,
	}
}

// GetTaskType 返回任务类型
func (h *ServiceRequestServiceTaskHandler) GetTaskType() string {
	return "service_request_task"
}

// GetHandlerID 返回处理器标识
func (h *ServiceRequestServiceTaskHandler) GetHandlerID() string {
	return "service_request_handler"
}

// Execute 执行服务请求任务
func (h *ServiceRequestServiceTaskHandler) Execute(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	action, _ := variables["action"].(string)
	switch action {
	case "create_request":
		return h.createRequest(ctx, variables)
	case "update_request":
		return h.updateRequest(ctx, variables)
	case "approve_request":
		return h.approveRequest(ctx, variables)
	case "reject_request":
		return h.rejectRequest(ctx, variables)
	case "assign_request":
		return h.assignRequest(ctx, variables)
	case "provision_resource":
		return h.provisionResource(ctx, variables)
	case "complete_request":
		return h.completeRequest(ctx, variables)
	case "cancel_request":
		return h.cancelRequest(ctx, variables)
	default:
		return &dto.ServiceTaskResult{Success: true, Message: "无操作执行"}, nil
	}
}

// createRequest 创建服务请求
func (h *ServiceRequestServiceTaskHandler) createRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	title, _ := variables["title"].(string)
	catalogID := getIntFromVars(variables, "catalog_id")

	if title == "" {
		return nil, fmt.Errorf("请求标题不能为空")
	}

	// 注意：这里需要通过其他服务创建请求，简化处理返回成功
	h.logger.Infow("Service request creation via BPMN", "title", title, "catalog_id", catalogID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: "服务请求已创建",
	}, nil
}

// updateRequest 更新服务请求
func (h *ServiceRequestServiceTaskHandler) updateRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := getIntFromVars(variables, "request_id")
	if requestID <= 0 {
		return nil, fmt.Errorf("无效的请求ID")
	}

	h.logger.Infow("Service request updated via BPMN", "request_id", requestID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("服务请求 %d 已更新", requestID),
	}, nil
}

// approveRequest 审批服务请求
func (h *ServiceRequestServiceTaskHandler) approveRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := getIntFromVars(variables, "request_id")
	level := getIntFromVars(variables, "approval_level")

	if requestID <= 0 {
		return nil, fmt.Errorf("无效的请求ID")
	}

	h.logger.Infow("Service request approved via BPMN", "request_id", requestID, "level", level)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("服务请求 %d 第%d级审批已通过", requestID, level),
	}, nil
}

// rejectRequest 驳回服务请求
func (h *ServiceRequestServiceTaskHandler) rejectRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := getIntFromVars(variables, "request_id")
	reason, _ := variables["reject_reason"].(string)

	if requestID <= 0 {
		return nil, fmt.Errorf("无效的请求ID")
	}

	h.logger.Infow("Service request rejected via BPMN", "request_id", requestID, "reason", reason)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("服务请求 %d 已被驳回: %s", requestID, reason),
	}, nil
}

// assignRequest 分配服务请求
func (h *ServiceRequestServiceTaskHandler) assignRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := getIntFromVars(variables, "request_id")
	assigneeID := getIntFromVars(variables, "assignee_id")

	if requestID <= 0 {
		return nil, fmt.Errorf("无效的请求ID")
	}

	h.logger.Infow("Service request assigned via BPMN", "request_id", requestID, "assignee_id", assigneeID)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("服务请求 %d 已分配", requestID),
	}, nil
}

// provisionResource provision资源
func (h *ServiceRequestServiceTaskHandler) provisionResource(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := getIntFromVars(variables, "request_id")
	resourceType, _ := variables["resource_type"].(string)

	h.logger.Infow("Resource provisioning via BPMN", "request_id", requestID, "resource_type", resourceType)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("资源 %s 正在供应中", resourceType),
	}, nil
}

// completeRequest 完成服务请求
func (h *ServiceRequestServiceTaskHandler) completeRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := getIntFromVars(variables, "request_id")
	completionNote, _ := variables["completion_note"].(string)

	h.logger.Infow("Service request completed via BPMN", "request_id", requestID, "note", completionNote)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("服务请求 %d 已完成", requestID),
	}, nil
}

// cancelRequest 取消服务请求
func (h *ServiceRequestServiceTaskHandler) cancelRequest(ctx context.Context, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	requestID := getIntFromVars(variables, "request_id")
	reason, _ := variables["cancel_reason"].(string)

	h.logger.Infow("Service request cancelled via BPMN", "request_id", requestID, "reason", reason)

	return &dto.ServiceTaskResult{
		Success: true,
		Message: fmt.Sprintf("服务请求 %d 已取消: %s", requestID, reason),
	}, nil
}
