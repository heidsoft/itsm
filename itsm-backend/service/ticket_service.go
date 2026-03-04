package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/ticketattachment"
	"itsm-backend/ent/ticketcategory"
	"itsm-backend/ent/ticketcomment"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

type TicketService struct {
	config *config.Config
	client                *ent.Client
	logger                *zap.SugaredLogger
	notificationService   *TicketNotificationService
	automationRuleService *TicketAutomationRuleService
	// 流程触发服务（用于工单创建时自动触发工作流）
	processTriggerService ProcessTriggerServiceInterface
	// 审批服务
	approvalService *ApprovalService
	// 子服务
	lifecycleService  *TicketLifecycleService
	assignmentService *TicketAssignmentService
	slaService        *TicketSLAService
}

func NewTicketService(client *ent.Client, logger *zap.SugaredLogger) *TicketService {
	svc := &TicketService{
		client: client,
		logger: logger,
	}
	// 初始化子服务
	svc.lifecycleService = NewTicketLifecycleService(client, logger)
	svc.assignmentService = NewTicketAssignmentService(client, logger)
	svc.slaService = NewTicketSLAService(client, logger)
	return svc
}

// SetNotificationService 设置通知服务（用于依赖注入）
func (s *TicketService) SetNotificationService(notificationService *TicketNotificationService) {
	s.notificationService = notificationService
}

// SetAutomationRuleService 设置自动化规则服务（用于依赖注入）
func (s *TicketService) SetAutomationRuleService(automationRuleService *TicketAutomationRuleService) {
	s.automationRuleService = automationRuleService
}

// SetProcessTriggerService 设置流程触发服务（用于工单创建时自动触发工作流）
func (s *TicketService) SetProcessTriggerService(triggerService ProcessTriggerServiceInterface) {
	s.processTriggerService = triggerService
}

// SetApprovalService 设置审批服务
func (s *TicketService) SetApprovalService(approvalService *ApprovalService) {
	s.approvalService = approvalService
}

// SetLifecycleService 设置生命周期服务
func (s *TicketService) SetLifecycleService(lifecycleService *TicketLifecycleService) {
	s.lifecycleService = lifecycleService
	if lifecycleService != nil && s.notificationService != nil {
		lifecycleService.SetNotificationService(s.notificationService)
	}
}

// SetAssignmentService 设置分配服务
func (s *TicketService) SetAssignmentService(assignmentService *TicketAssignmentService) {
	s.assignmentService = assignmentService
}

// SetSLAService 设置SLA服务
func (s *TicketService) SetSLAService(slaService *TicketSLAService) {
	s.slaService = slaService
}

// CreateTicket 创建工单
func (s *TicketService) CreateTicket(ctx context.Context, req *dto.CreateTicketRequest, tenantID int) (*ent.Ticket, error) {
	s.logger.Infow("Creating ticket", "tenant_id", tenantID, "title", req.Title)

	// V0：最小校验（对齐测试与产品规则）
	if strings.TrimSpace(req.Description) == "" {
		return nil, fmt.Errorf("描述不能为空")
	}
	switch strings.ToLower(strings.TrimSpace(req.Priority)) {
	case "", "low", "medium", "high", "urgent":
		// ok（空值交给 schema 默认 medium）
	default:
		return nil, fmt.Errorf("无效的优先级")
	}

	// 验证 requester_id 是否有效
	if req.RequesterID <= 0 {
		return nil, fmt.Errorf("创建人工单无效")
	}
	requesterExists, err := s.client.User.Query().
		Where(user.ID(req.RequesterID), user.TenantID(tenantID)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to verify requester", "error", err)
		return nil, fmt.Errorf("验证创建人失败")
	}
	if !requesterExists {
		return nil, fmt.Errorf("创建的用户不存在或不属于当前租户")
	}

	// 验证 assignee_id 是否有效（如果是正数，必须是有效的用户ID）
	if req.AssigneeID > 0 {
		exists, err := s.client.User.Query().Where(user.ID(req.AssigneeID), user.TenantID(tenantID)).Exist(ctx)
		if err != nil {
			s.logger.Errorw("Failed to verify assignee", "error", err)
			return nil, fmt.Errorf("验证分配人失败")
		}
		if !exists {
			return nil, fmt.Errorf("分配的用户不存在或不属于当前租户")
		}
	}

	// 计算SLA截止时间
	ticketType := req.Type
	if ticketType == "" {
		ticketType = "ticket" // 默认类型
	}
	slaResult, err := s.slaService.CalculateSLADeadlineFromRequest(ctx, tenantID, ticketType, req.Priority)
	if err != nil {
		s.logger.Warnw("Failed to calculate SLA deadline", "error", err)
		// SLA计算失败不阻止工单创建，使用默认值（响应8小时，解决24小时）
		defaultRespDeadline := time.Now().Add(8 * time.Hour)
		defaultResDeadline := time.Now().Add(24 * time.Hour)
		slaResult = &SLADeadlineResult{
			SLADefinitionID:    0,
			ResponseDeadline:   &defaultRespDeadline,
			ResolutionDeadline: &defaultResDeadline,
		}
	}

	// 生成工单编号
	ticketNumber, err := s.generateTicketNumber(ctx, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to generate ticket number", "error", err)
		return nil, fmt.Errorf("failed to generate ticket number: %w", err)
	}

	// 处理 SLA 截止时间指针
	var respDeadline, resDeadline time.Time
	if slaResult.ResponseDeadline != nil {
		respDeadline = *slaResult.ResponseDeadline
	} else {
		respDeadline = time.Now().Add(8 * time.Hour)
	}
	if slaResult.ResolutionDeadline != nil {
		resDeadline = *slaResult.ResolutionDeadline
	} else {
		resDeadline = time.Now().Add(24 * time.Hour)
	}

	createBuilder := s.client.Ticket.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetPriority(req.Priority).
		SetType(ticketType).
		// 工单默认状态：open（与 schema 默认一致）
		SetStatus("open").
		SetTicketNumber(ticketNumber).
		SetTenantID(tenantID).
		SetRequesterID(req.RequesterID).
		SetSLAResponseDeadline(respDeadline).
		SetSLAResolutionDeadline(resDeadline)

	// 只在指定了有效的分配人ID时设置（避免SQLite FK约束问题）
	if req.AssigneeID > 0 {
		createBuilder = createBuilder.SetAssigneeID(req.AssigneeID)
	}
	// 只在有有效的SLA定义ID时设置（避免FK约束问题）
	if slaResult.SLADefinitionID > 0 {
		createBuilder = createBuilder.SetSLADefinitionID(slaResult.SLADefinitionID)
	}

	// 如果指定了父工单ID，设置父工单关系
	if req.ParentTicketID != nil && *req.ParentTicketID > 0 {
		createBuilder = createBuilder.SetParentTicketID(*req.ParentTicketID)
	}

	// 如果指定了分类ID，设置分类
	if req.CategoryID != nil && *req.CategoryID > 0 {
		createBuilder = createBuilder.SetCategoryID(*req.CategoryID)
	} else if req.Category != "" {
		// 尝试根据名称查找分类
		cat, err := s.client.TicketCategory.Query().
			Where(ticketcategory.NameEQ(req.Category), ticketcategory.TenantID(tenantID)).
			First(ctx)
		if err == nil {
			createBuilder = createBuilder.SetCategoryID(cat.ID)
	} else if ent.IsNotFound(err) {
		// 查找默认分类
		defaultCat, err := s.client.TicketCategory.Query().
			Where(ticketcategory.IsActive(true), ticketcategory.TenantID(tenantID)).
			First(ctx)
		if err == nil {
			createBuilder = createBuilder.SetCategoryID(defaultCat.ID)
		} else {
			s.logger.Warnw("Default category not found", "error", err)
		}
	}
	}

	// 如果指定了模板ID，设置模板
	if req.TemplateID != nil && *req.TemplateID > 0 {
		createBuilder = createBuilder.SetTemplateID(*req.TemplateID)
	}

	ticket, err := createBuilder.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create ticket", "error", err)
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	// 如果指定了标签ID，添加标签关联
	if len(req.TagIDs) > 0 {
		_, err = ticket.Update().
			AddTagIDs(req.TagIDs...).
			Save(ctx)
		if err != nil {
			s.logger.Warnw("Failed to add tags to ticket", "error", err, "ticket_id", ticket.ID)
			// 不返回错误，因为工单已创建成功
		}
	}

	if err != nil {
		s.logger.Errorw("Failed to create ticket", "error", err)
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	// 记录审计日志
	s.logAuditEvent(ctx, "ticket_created", ticket.ID, tenantID, map[string]interface{}{
		"title":    req.Title,
		"priority": req.Priority,
		"category": req.Category,
		"type":     ticketType,
	})

	// 触发审批流程（仅针对需要审批的工单类型）
	if s.approvalService != nil {
		ticketType := req.Type
		if ticketType == "" {
			ticketType = "ticket"
		}
		// 只有紧急和高优先级工单需要审批
		if req.Priority == "urgent" || req.Priority == "critical" {
			approvalReq := &ApprovalTriggerRequest{
				TicketID:     ticket.ID,
				TicketNumber: ticket.TicketNumber,
				TicketTitle:  ticket.Title,
				TicketType:   ticketType,
				Priority:     req.Priority,
				RequesterID:  req.RequesterID,
				TenantID:     tenantID,
			}
			records, err := s.approvalService.TriggerApproval(ctx, approvalReq)
			if err != nil {
				s.logger.Warnw("Failed to trigger approval", "error", err, "ticket_id", ticket.ID)
			} else if len(records) > 0 {
				s.logger.Infow("Approval workflow triggered", "ticket_id", ticket.ID, "records", len(records))
			}
		}
	}

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

	// 触发BPMN工作流（异步执行，不阻塞工单创建）
	if s.processTriggerService != nil {
		go func() {
			// 使用新的context避免超时影响
			workflowCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.triggerWorkflowForTicket(workflowCtx, ticket.ID, tenantID); err != nil {
				s.logger.Warnw("Failed to trigger workflow for ticket", "error", err, "ticket_id", ticket.ID)
			}
		}()
	}

	return ticket, nil
}

// triggerWorkflowForTicket 为工单触发工作流
func (s *TicketService) triggerWorkflowForTicket(ctx context.Context, ticketID int, tenantID int) error {
	// 获取工单信息
	ticket, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	// 构建流程变量
	variables := map[string]interface{}{
		"ticket_id":     ticket.ID,
		"ticket_number": ticket.TicketNumber,
		"title":         ticket.Title,
		"description":   ticket.Description,
		"priority":      ticket.Priority,
		"status":        ticket.Status,
		"requester_id":  ticket.RequesterID,
		"assignee_id":   ticket.AssigneeID,
	}

	// 根据优先级选择不同的流程
	processKey := "ticket_general_flow"
	if ticket.Priority == "high" || ticket.Priority == "urgent" {
		processKey = "ticket_urgent_flow"
	}

	// 触发流程
	triggerReq := &dto.ProcessTriggerRequest{
		BusinessType:         dto.BusinessTypeTicket,
		BusinessID:           ticket.ID,
		ProcessDefinitionKey: processKey,
		Variables:            variables,
		TriggeredBy:          fmt.Sprintf("%d", ticket.RequesterID),
		TriggeredAt:          time.Now(),
		TenantID:             tenantID,
	}

	resp, err := s.processTriggerService.TriggerProcess(ctx, triggerReq)
	if err != nil {
		return fmt.Errorf("failed to trigger workflow: %w", err)
	}

	s.logger.Infow("Workflow triggered for ticket",
		"ticket_id", ticketID,
		"process_instance_id", resp.ProcessInstanceID,
		"process_key", processKey,
		"business_key", resp.BusinessKey,
	)

	return nil
}

// GetWorkflowStatus 获取工单关联的流程状态
func (s *TicketService) GetWorkflowStatus(ctx context.Context, ticketID int, tenantID int) (*dto.ProcessTriggerResponse, error) {
	// 构建 businessKey
	businessKey := fmt.Sprintf("ticket:%d", ticketID)

	// 通过 ProcessTriggerService 获取流程状态
	// 由于 ProcessTriggerService 没有直接查询方法，我们直接查询数据库
	processInstance, err := s.client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
		).
		WithDefinition().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("未找到工单关联的流程实例")
		}
		return nil, fmt.Errorf("查询流程实例失败: %w", err)
	}

	// 获取流程定义名称
	processDefName := ""
	if processInstance.Edges.Definition != nil {
		processDefName = processInstance.Edges.Definition.Name
	}

	return &dto.ProcessTriggerResponse{
		ProcessInstanceID:     processInstance.ID,
		ProcessDefinitionKey:  processInstance.ProcessDefinitionKey,
		ProcessDefinitionName: processDefName,
		BusinessKey:           processInstance.BusinessKey,
		Status:                s.mapProcessStatus(processInstance.Status),
		CurrentActivityID:     processInstance.CurrentActivityID,
		CurrentActivityName:   processInstance.CurrentActivityName,
		StartTime:             processInstance.StartTime,
		EndTime:               &processInstance.EndTime,
	}, nil
}

// mapProcessStatus 映射流程状态
func (s *TicketService) mapProcessStatus(status string) dto.ProcessStatus {
	switch status {
	case "running", "active":
		return dto.ProcessStatusRunning
	case "completed":
		return dto.ProcessStatusCompleted
	case "suspended":
		return dto.ProcessStatusSuspended
	case "terminated", "cancelled":
		return dto.ProcessStatusTerminated
	default:
		return dto.ProcessStatusPending
	}
}

// CancelWorkflow 取消工单关联的流程
func (s *TicketService) CancelWorkflow(ctx context.Context, ticketID int, tenantID int, reason string) error {
	businessKey := fmt.Sprintf("ticket:%d", ticketID)

	// 查找流程实例
	processInstance, err := s.client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("未找到工单关联的流程实例")
		}
		return fmt.Errorf("查询流程实例失败: %w", err)
	}

	// 取消流程
	if s.processTriggerService != nil {
		return s.processTriggerService.CancelProcess(ctx, processInstance.ID, reason, tenantID)
	}

	return fmt.Errorf("流程触发服务未配置")
}

// SyncTicketStatusWithWorkflow 同步工单状态与流程状态
func (s *TicketService) SyncTicketStatusWithWorkflow(ctx context.Context, ticketID int, tenantID int) error {
	// 获取流程状态
	workflowStatus, err := s.GetWorkflowStatus(ctx, ticketID, tenantID)
	if err != nil {
		s.logger.Warnw("Failed to get workflow status for sync", "error", err, "ticket_id", ticketID)
		return err
	}

	// 根据流程状态更新工单状态
	var newTicketStatus string
	switch workflowStatus.Status {
	case dto.ProcessStatusCompleted:
		// 流程完成时，工单状态设为已解决
		newTicketStatus = "resolved"
	case dto.ProcessStatusTerminated, dto.ProcessStatusSuspended:
		// 流程终止或暂停时，工单状态设为暂停
		newTicketStatus = "pending"
	default:
		// 流程进行中，不自动更新工单状态
		return nil
	}

	// 更新工单状态
	_, err = s.client.Ticket.UpdateOneID(ticketID).
		SetStatus(newTicketStatus).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("同步工单状态失败: %w", err)
	}

	s.logger.Infow("Ticket status synced with workflow",
		"ticket_id", ticketID,
		"workflow_status", workflowStatus.Status,
		"ticket_status", newTicketStatus,
	)

	return nil
}

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
	// 验证 requester_id 是否有效
	if req.RequesterID <= 0 {
		return nil, fmt.Errorf("创建人工单无效")
	}
	requesterExists, err := s.client.User.Query().
		Where(user.ID(req.RequesterID), user.TenantID(tenantID)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to verify requester", "error", err)
		return nil, fmt.Errorf("验证创建人失败")
	}
	if !requesterExists {
		return nil, fmt.Errorf("创建的用户不存在或不属于当前租户")
	}

	// 验证 assignee_id 是否有效（如果是正数，必须是有效的用户ID）
	if req.AssigneeID > 0 {
		exists, err := s.client.User.Query().Where(user.ID(req.AssigneeID), user.TenantID(tenantID)).Exist(ctx)
		if err != nil {
			s.logger.Errorw("Failed to verify assignee", "error", err)
			return nil, fmt.Errorf("验证分配人失败")
		}
		if !exists {
			return nil, fmt.Errorf("分配的用户不存在或不属于当前租户")
		}
		updateQuery.SetAssigneeID(req.AssigneeID)
	} else if req.AssigneeID == 0 {
		// 允许将 assignee 设为 0（表示取消分配）
		updateQuery.SetAssigneeID(0)
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

// UpdateTicketStatus 更新工单状态
func (s *TicketService) UpdateTicketStatus(ctx context.Context, ticketID int, status string, tenantID int, operatorID int) (*ent.Ticket, error) {
	s.logger.Infow("Updating ticket status", "ticket_id", ticketID, "status", status, "tenant_id", tenantID, "operator_id", operatorID)

	// 检查工单是否存在
	ticket, err := s.client.Ticket.Query().
		Where(ticket.ID(ticketID), ticket.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	// 不能审批/拒绝自己提交的工单
	// if (status == "approved" || status == "rejected") && ticket.RequesterID == operatorID {
	// 	return nil, fmt.Errorf("cannot approve or reject your own ticket")
	// }

	// 验证状态值
	validStatuses := map[string]bool{
		"new": true, "open": true, "in_progress": true, "pending": true,
		"resolved": true, "closed": true, "cancelled": true, "approved": true, "rejected": true,
	}
	if !validStatuses[status] {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	// 更新状态
	ticket, err = s.client.Ticket.UpdateOneID(ticketID).
		SetStatus(status).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update ticket status", "error", err, "ticket_id", ticketID)
		return nil, fmt.Errorf("failed to update ticket status: %w", err)
	}

	// 如果是解决或关闭状态，设置解决时间
	if status == "resolved" || status == "closed" {
		now := time.Now()
		ticket, err = s.client.Ticket.UpdateOneID(ticketID).
			SetResolvedAt(now).
			SetUpdatedAt(now).
			Save(ctx)
		if err != nil {
			s.logger.Warnw("Failed to set resolved time", "error", err, "ticket_id", ticketID)
		}
	}

	s.logger.Infow("Ticket status updated", "ticket_id", ticketID, "new_status", status)
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
	if req.ParentTicketID != nil && *req.ParentTicketID > 0 {
		query.Where(ticket.ParentTicketID(*req.ParentTicketID))
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

	// 收集所有需要的用户ID
	userIDMap := make(map[int]bool)
	for _, t := range tickets {
		if t.RequesterID > 0 {
			userIDMap[t.RequesterID] = true
		}
		if t.AssigneeID > 0 {
			userIDMap[t.AssigneeID] = true
		}
	}

	// 批量查询用户
	userMap := make(map[int]*ent.User)
	if len(userIDMap) > 0 {
		userIDs := make([]int, 0, len(userIDMap))
		for id := range userIDMap {
			userIDs = append(userIDs, id)
		}
		users, err := s.client.User.Query().Where(user.IDIn(userIDs...)).All(ctx)
		if err != nil {
			s.logger.Warnw("Failed to query users", "error", err)
		} else {
			for _, u := range users {
				userMap[u.ID] = u
			}
		}
	}

	// 转换为 DTO 响应格式，确保 ticket_number 字段正确返回
	ticketResponses := make([]*dto.TicketResponse, len(tickets))
	for i, t := range tickets {
		var requester, assignee *ent.User
		if t.RequesterID > 0 {
			requester = userMap[t.RequesterID]
		}
		if t.AssigneeID > 0 {
			assignee = userMap[t.AssigneeID]
		}
		ticketResponses[i] = dto.ToTicketResponseWithUsers(t, requester, assignee)
	}

	return &dto.ListTicketsResponse{
		Tickets:  ticketResponses,
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

// TicketSLAInfo 工单SLA信息
type TicketSLAInfo struct {
	TicketID             int        `json:"ticket_id"`
	TicketNumber         string     `json:"ticket_number"`
	Priority             string     `json:"priority"`
	SLADefinitionID      int        `json:"sla_definition_id"`
	SLADefinitionName    string     `json:"sla_definition_name"`
	ResponseDeadline     time.Time  `json:"response_deadline"`
	ResolutionDeadline   time.Time  `json:"resolution_deadline"`
	FirstResponseAt      *time.Time `json:"first_response_at"`
	ResolvedAt           *time.Time `json:"resolved_at"`
	ResponseTimeLeft     int        `json:"response_time_left"`   // 剩余响应时间（分钟）
	ResolutionTimeLeft   int        `json:"resolution_time_left"` // 剩余解决时间（分钟）
	IsResponseBreached   bool       `json:"is_response_breached"`
	IsResolutionBreached bool       `json:"is_resolution_breached"`
}

// GetTicketSLAInfo 获取工单SLA信息
func (s *TicketService) GetTicketSLAInfo(ctx context.Context, ticketID int, tenantID int) (*TicketSLAInfo, error) {
	ticket, err := s.client.Ticket.Query().
		Where(ticket.ID(ticketID), ticket.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	info := &TicketSLAInfo{
		TicketID:           ticket.ID,
		TicketNumber:       ticket.TicketNumber,
		Priority:           ticket.Priority,
		SLADefinitionID:    ticket.SLADefinitionID,
		ResponseDeadline:   ticket.SLAResponseDeadline,
		ResolutionDeadline: ticket.SLAResolutionDeadline,
	}

	// 处理 first_response_at
	if !ticket.FirstResponseAt.IsZero() {
		info.FirstResponseAt = &ticket.FirstResponseAt
	}

	// 处理 resolved_at
	if !ticket.ResolvedAt.IsZero() {
		info.ResolvedAt = &ticket.ResolvedAt
	}

	// 获取SLA定义名称
	if ticket.SLADefinitionID > 0 {
		sla, err := s.client.SLADefinition.Query().
			Where(sladefinition.ID(ticket.SLADefinitionID)).
			First(ctx)
		if err == nil && sla != nil {
			info.SLADefinitionName = sla.Name
		}
	}

	// 计算剩余时间
	now := time.Now()

	// 响应时间剩余
	if !ticket.FirstResponseAt.IsZero() {
		info.ResponseTimeLeft = 0
		info.IsResponseBreached = false
	} else if now.After(ticket.SLAResponseDeadline) {
		info.ResponseTimeLeft = 0
		info.IsResponseBreached = true
	} else {
		info.ResponseTimeLeft = int(ticket.SLAResponseDeadline.Sub(now).Minutes())
	}

	// 解决时间剩余
	if !ticket.ResolvedAt.IsZero() {
		info.ResolutionTimeLeft = 0
		info.IsResolutionBreached = false
	} else if now.After(ticket.SLAResolutionDeadline) {
		info.ResolutionTimeLeft = 0
		info.IsResolutionBreached = true
	} else {
		info.ResolutionTimeLeft = int(ticket.SLAResolutionDeadline.Sub(now).Minutes())
	}

	return info, nil
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

	// 批量硬删除（注意：生产环境通常建议软删除）
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

	// 获取逾期工单数（使用SLA服务）
	overdueCount := 0
	if s.slaService != nil {
		overdueTickets, err := s.slaService.GetOverdueTickets(ctx, tenantID)
		if err == nil {
			overdueCount = len(overdueTickets)
		}
	}

	return &dto.TicketStatsResponse{
		Total:        totalTickets,
		Open:         submittedTickets,
		InProgress:   inProgressTickets,
		Resolved:     closedTickets,
		HighPriority: highPriorityTickets,
		Overdue:      overdueCount,
	}, nil
}

// TicketSLADeadlineResult 工单SLA截止时间结果（用于工单创建）
type TicketSLADeadlineResult struct {
	SLADefinitionID    int       `json:"sla_definition_id"`
	ResponseDeadline   time.Time `json:"response_deadline"`
	ResolutionDeadline time.Time `json:"resolution_deadline"`
}

// canUpdateTicket 检查是否可以更新工单
func (s *TicketService) canUpdateTicket(ticket *ent.Ticket, userID int) bool {
	// 简化权限检查：工单创建者或处理人可以更新
	return ticket.RequesterID == userID || ticket.AssigneeID == userID
}

// AssignTicket 分配工单
func (s *TicketService) AssignTicket(ctx context.Context, ticketID int, assigneeID int, tenantID int, assignedBy int) (*ent.Ticket, error) {
	s.logger.Infow("Assigning ticket", "ticket_id", ticketID, "assignee_id", assigneeID, "tenant_id", tenantID)

	// 直接更新工单分配信息
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

	// 如果有生命周期服务，委托给它
	if s.lifecycleService != nil {
		return s.lifecycleService.EscalateTicket(ctx, ticketID, reason, tenantID, escalatedBy)
	}

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

	// 如果有生命周期服务，委托给它
	if s.lifecycleService != nil {
		return s.lifecycleService.ResolveTicket(ctx, ticketID, resolution, tenantID, resolvedBy)
	}

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

	// 如果有生命周期服务，委托给它
	if s.lifecycleService != nil {
		return s.lifecycleService.CloseTicket(ctx, ticketID, feedback, tenantID, closedBy)
	}

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

	if strings.TrimSpace(searchTerm) == "" {
		// 空搜索词：返回空结果（避免无条件返回全量）
		return []*ent.Ticket{}, nil
	}

	// 构建搜索查询（简化版，实际应使用全文搜索引擎）
	query := s.client.Ticket.Query().Where(ticket.TenantID(tenantID))

	// 在标题和描述中搜索
	searchLower := strings.ToLower(strings.TrimSpace(searchTerm))
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

	// 如果有SLA服务，委托给它
	if s.slaService != nil {
		return s.slaService.GetOverdueTickets(ctx, tenantID)
	}

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

	// 如果有分配服务，委托给它
	if s.assignmentService != nil {
		return s.assignmentService.GetTicketsByAssignee(ctx, assigneeID, tenantID)
	}

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

	// 验证工单存在
	ticket, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	activities := make([]map[string]interface{}, 0)

	// 1. 添加工单创建活动
	activities = append(activities, map[string]interface{}{
		"action":    "created",
		"timestamp": ticket.CreatedAt,
		"user_id":   ticket.RequesterID,
		"user_name": "",
		"details":   "工单已创建",
		"old_value": nil,
		"new_value": ticket.Title,
	})

	// 2. 查询评论作为活动记录
	comments, err := s.client.TicketComment.Query().
		Where(
			ticketcomment.TicketID(ticketID),
		).
		WithUser().
		Order(ent.Asc(ticketcomment.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Warnw("Failed to get comments for activity", "error", err)
	} else {
		for _, c := range comments {
			userName := ""
			if c.Edges.User != nil {
				userName = c.Edges.User.Name
				if userName == "" {
					userName = c.Edges.User.Username
				}
			}
			activities = append(activities, map[string]interface{}{
				"action":    "commented",
				"timestamp": c.CreatedAt,
				"user_id":   c.UserID,
				"user_name": userName,
				"details":   "添加了评论",
				"old_value": nil,
				"new_value": nil,
			})
		}
	}

	// 3. 查询附件作为活动记录
	attachments, err := s.client.TicketAttachment.Query().
		Where(
			ticketattachment.TicketID(ticketID),
		).
		Order(ent.Asc(ticketattachment.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Warnw("Failed to get attachments for activity", "error", err)
	} else {
		for _, a := range attachments {
			activities = append(activities, map[string]interface{}{
				"action":    "attachment",
				"timestamp": a.CreatedAt,
				"user_id":   a.UploadedBy,
				"user_name": "",
				"details":   fmt.Sprintf("添加了附件: %s", a.FileName),
				"old_value": nil,
				"new_value": nil,
			})
		}
	}

	// 4. 如果有分配历史，添加分配活动
	if ticket.AssigneeID > 0 {
		activities = append(activities, map[string]interface{}{
			"action":    "assigned",
			"timestamp": ticket.UpdatedAt,
			"user_id":   ticket.AssigneeID,
			"user_name": "",
			"details":   "工单已分配",
			"old_value": nil,
			"new_value": ticket.AssigneeID,
		})
	}

	// 5. 如果有首次响应时间，添加响应活动
	if !ticket.FirstResponseAt.IsZero() {
		activities = append(activities, map[string]interface{}{
			"action":    "first_response",
			"timestamp": ticket.FirstResponseAt,
			"user_id":   ticket.AssigneeID,
			"user_name": "",
			"details":   "首次响应工单",
			"old_value": nil,
			"new_value": nil,
		})
	}

	// 6. 如果有解决时间，添加解决活动
	if !ticket.ResolvedAt.IsZero() {
		activities = append(activities, map[string]interface{}{
			"action":    "resolved",
			"timestamp": ticket.ResolvedAt,
			"user_id":   ticket.AssigneeID,
			"user_name": "",
			"details":   "工单已解决",
			"old_value": nil,
			"new_value": nil,
		})
	}

	// 7. 如果有评分，添加评分活动
	if ticket.Rating > 0 {
		activities = append(activities, map[string]interface{}{
			"action":    "rated",
			"timestamp": ticket.RatedAt,
			"user_id":   ticket.RatedBy,
			"user_name": "",
			"details":   fmt.Sprintf("用户评分: %d星", ticket.Rating),
			"old_value": nil,
			"new_value": ticket.Rating,
		})
	}

	// 按时间倒序排列
	for i, j := 0, len(activities)-1; i < j; i, j = i+1, j-1 {
		activities[i], activities[j] = activities[j], activities[i]
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
	// 如果有分配服务，委托给它
	if s.assignmentService != nil {
		return s.assignmentService.AssignTickets(ctx, tenantID, ticketIDs, assigneeID)
	}

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

// generateTicketNumber 生成工单编号
func (s *TicketService) generateTicketNumber(ctx context.Context, tenantID int) (string, error) {
	// 获取当前年份和月份
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// 查询当月的工单数量
	count, err := s.client.Ticket.Query().
		Where(
			ticket.TenantIDEQ(tenantID),
			ticket.CreatedAtGTE(time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)),
			ticket.CreatedAtLT(time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)),
		).
		Count(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to count tickets: %w", err)
	}

	// 生成工单编号格式: TKT-YYYYMM-XXXXXX
	ticketNumber := fmt.Sprintf("TKT-%04d%02d-%06d", year, month, count+1)
	return ticketNumber, nil
}
