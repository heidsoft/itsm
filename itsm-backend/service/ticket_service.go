package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"itsm-backend/connector"
	feishuConnector "itsm-backend/connector/builtin/feishu"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/processinstance"
	entTicket "itsm-backend/ent/ticket"
	"itsm-backend/ent/ticketcategory"
	entTicketComment "itsm-backend/ent/ticketcomment"
	"itsm-backend/ent/tickettag"
	"itsm-backend/ent/tickettemplate"
	"itsm-backend/ent/user"
	"itsm-backend/repository/base"
	"itsm-backend/repository/ticket"

	"go.uber.org/zap"
)

// TicketService 改进版的工单服务
// 使用构造函数注入和 Repository 模式
type TicketService struct {
	repo              ticket.Repository
	client            *ent.Client // 用于 ProcessInstance 等系统级查询（不走 Repository）
	logger            *zap.SugaredLogger
	notificationSvc   *TicketNotificationService
	approvalSvc       *ApprovalService
	automationRuleSvc *TicketAutomationRuleService
	slaSvc            *TicketSLAService
	connectorManager  *connector.Manager // 连接器管理器，用于飞书等外部集成

	// 流程触发（V1 兼容语义）
	processTriggerSvc ProcessTriggerServiceInterface
	processResolver   *ProcessResolver
}

// TicketServiceConfig 工单服务配置
// 所有依赖都在配置中明确声明
type TicketServiceConfig struct {
	Repository            ticket.Repository
	Client                *ent.Client // 可选；传入后可用作 ProcessInstance 等系统级查询
	Logger                *zap.SugaredLogger
	NotificationService   *TicketNotificationService
	ApprovalService       *ApprovalService
	AutomationRuleService *TicketAutomationRuleService
	SLAService            *TicketSLAService
	ProcessTriggerService ProcessTriggerServiceInterface
	ProcessResolver       *ProcessResolver
	ConnectorManager      *connector.Manager // 连接器管理器
}

// NewTicketService 创建工单服务
// 使用构造函数注入，所有依赖必须显式传入
func NewTicketService(cfg *TicketServiceConfig) *TicketService {
	if cfg.Repository == nil {
		panic("Repository is required")
	}
	if cfg.Logger == nil {
		panic("Logger is required")
	}

	return &TicketService{
		repo:              cfg.Repository,
		client:            cfg.Client,
		logger:            cfg.Logger,
		notificationSvc:   cfg.NotificationService,
		approvalSvc:       cfg.ApprovalService,
		automationRuleSvc: cfg.AutomationRuleService,
		slaSvc:            cfg.SLAService,
		processTriggerSvc: cfg.ProcessTriggerService,
		processResolver:   cfg.ProcessResolver,
		connectorManager:  cfg.ConnectorManager,
	}
}

// NewTicketServiceForTest 构造一个最小可运行的 TicketService（仅用于测试）
// 自动构造一个 EntRepository，避免每个测试都要写完整配置
func NewTicketServiceForTest(client *ent.Client, logger *zap.SugaredLogger) *TicketService {
	return NewTicketService(&TicketServiceConfig{
		Repository: ticket.NewEntRepository(client, logger),
		Client:     client,
		Logger:     logger,
	})
}

// SetNotificationService 注入通知服务（运行时依赖注入）
func (s *TicketService) SetNotificationService(n *TicketNotificationService) {
	s.notificationSvc = n
}

// SetApprovalService 注入审批服务（运行时依赖注入）
func (s *TicketService) SetApprovalService(a *ApprovalService) {
	s.approvalSvc = a
}

// SetProcessTriggerService 注入流程触发服务（运行时依赖注入）
func (s *TicketService) SetProcessTriggerService(p ProcessTriggerServiceInterface) {
	s.processTriggerSvc = p
}

// SetProcessResolver 注入流程解析器（运行时依赖注入）
func (s *TicketService) SetProcessResolver(r *ProcessResolver) {
	s.processResolver = r
}

// CreateTicket 创建工单
func (s *TicketService) CreateTicket(ctx context.Context, req *dto.CreateTicketRequest, tenantID int) (*ticket.Ticket, error) {
	s.logger.Infow("Creating ticket", "tenant_id", tenantID, "title", req.Title)
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("title不能为空")
	}
	switch ticket.Priority(req.Priority) {
	case ticket.PriorityLow, ticket.PriorityMedium, ticket.PriorityHigh, ticket.PriorityUrgent, ticket.PriorityCritical:
	default:
		return nil, fmt.Errorf("无效的工单优先级: %s", req.Priority)
	}
	if err := s.validateCreateTicketReferences(ctx, req, tenantID); err != nil {
		return nil, err
	}

	ticketType := normalizeCreateTicketType(req.Type, req.FormFields)
	assigneeID := req.AssigneeID
	categoryID := req.CategoryID
	workflowDefinitionKey := req.WorkflowDefinitionKey

	// 转换 DTO 到领域参数
	params := &ticket.CreateParams{
		Title:          req.Title,
		Description:    req.Description,
		Type:           ticketType,
		Priority:       ticket.Priority(req.Priority),
		RequesterID:    req.RequesterID,
		TemplateID:     req.TemplateID,
		ParentTicketID: req.ParentTicketID,
		TagIDs:         uniqueIDs(req.TagIDs),
	}

	if assigneeID != 0 {
		params.AssigneeID = &assigneeID
	}
	if categoryID != nil {
		params.CategoryID = categoryID
	}

	// 通过 Repository 创建工单
	tkt, err := s.repo.Create(ctx, params, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to create ticket", "error", err)
		return nil, err
	}

	// 计算 SLA（如果配置了 SLA 服务）
	if s.slaSvc != nil {
		slaResult, err := s.slaSvc.CalculateSLADeadlineFromRequest(ctx, tenantID, string(tkt.Type), string(tkt.Priority))
		if err != nil {
			s.logger.Warnw("Failed to calculate SLA", "error", err)
		} else {
			err = s.repo.UpdateSLADeadlines(ctx, tkt.ID, slaResult.ResponseDeadline, slaResult.ResolutionDeadline, &slaResult.SLADefinitionID, tenantID)
			if err != nil {
				s.logger.Warnw("Failed to update SLA deadlines", "error", err)
			}
		}
	}

	// 触发审批（同步，走 ApprovalService，查找匹配工作流并创建 ApprovalRecord）
	// 这是 V1 缺失的 Phase 1 #1 缺陷修复：V2 必须让工单进入审批链路
	if s.approvalSvc != nil {
		if _, err := s.approvalSvc.TriggerApproval(ctx, &ApprovalTriggerRequest{
			TicketID:     tkt.ID,
			TicketNumber: tkt.TicketNumber,
			TicketTitle:  tkt.Title,
			TicketType:   string(tkt.Type),
			Priority:     string(tkt.Priority),
			RequesterID:  tkt.RequesterID,
			TenantID:     tenantID,
		}); err != nil {
			s.logger.Warnw("Approval trigger failed", "error", err, "ticket_id", tkt.ID)
		}
	}

	// 异步发送通知（如果分配了处理人）
	// 注意：必须使用 context.Background()，不能复用请求 ctx。
	// 否则 HTTP 响应返回后 ctx 立即取消，异步任务会失败（context canceled）。
	if s.notificationSvc != nil && tkt.AssigneeID != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// 获取 ent.Ticket 用于通知（临时方案）
			// 理想情况下应该传递领域模型
			entTicket := s.toEntTicket(tkt)
			if err := s.notificationSvc.NotifyTicketCreated(ctx2, entTicket); err != nil {
				s.logger.Warnw("Notification failed", "error", err)
			}
		}()
	}

	// 异步执行自动化规则
	// 同样使用独立 ctx，避免请求生命周期结束导致异步任务中断。
	if s.automationRuleSvc != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.automationRuleSvc.ExecuteRulesForTicket(ctx2, tkt.ID, tenantID); err != nil {
				s.logger.Warnw("Automation rules failed", "error", err)
			}
		}()
	}

	// 异步触发 BPMN 流程（V1 兼容语义）
	// 这是 V1 缺失的 Phase 1 #1 缺陷修复：V2 必须让工单进入 BPMN 引擎
	// 使用独立 ctx，否则控制器响应后 workflow 触发立即被取消。
	if s.processTriggerSvc != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.triggerWorkflowForTicket(ctx2, tkt, tenantID, workflowDefinitionKey); err != nil {
				s.logger.Warnw("Workflow trigger failed", "error", err, "ticket_id", tkt.ID)
			}
		}()
	}

	s.logger.Infow("Ticket created", "ticket_id", tkt.ID, "ticket_number", tkt.TicketNumber)

	// 异步同步工单到飞书
	// 必须使用独立 ctx，否则飞书同步在响应返回后立即失败。
	if s.connectorManager != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// 获取Feishu连接器
			conn, ok := s.connectorManager.Get(tenantID, "feishu")
			if !ok {
				// 飞书连接器未配置，忽略
				return
			}
			feishuConn, ok := conn.(*feishuConnector.Feishu)
			if !ok {
				return
			}
			// 开启事务
			tx, err := s.client.Tx(ctx2)
			if err != nil {
				s.logger.Warnw("Failed to start transaction for feishu sync", "error", err, "ticket_id", tkt.ID)
				return
			}
			defer tx.Rollback()
			// 同步工单到飞书
			_, err = feishuConn.SyncTicketToFeishu(ctx2, tx, s.toEntTicket(tkt))
			if err != nil {
				s.logger.Warnw("Failed to sync ticket to feishu", "error", err, "ticket_id", tkt.ID)
				return
			}
			// 提交事务
			if err := tx.Commit(); err != nil {
				s.logger.Warnw("Failed to commit transaction for feishu sync", "error", err, "ticket_id", tkt.ID)
				return
			}
		}()
	}

	return tkt, nil
}

func (s *TicketService) validateCreateTicketReferences(ctx context.Context, req *dto.CreateTicketRequest, tenantID int) error {
	// Mock-backed unit tests may intentionally construct the service without an
	// Ent client. Production wiring and integration tests always provide it.
	if s.client == nil {
		return nil
	}
	if req.RequesterID <= 0 {
		return fmt.Errorf("requesterId必须为当前租户中的有效用户")
	}
	requesterExists, err := s.client.User.Query().
		Where(user.IDEQ(req.RequesterID), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("验证申请人失败: %w", err)
	}
	if !requesterExists {
		return fmt.Errorf("申请人不存在或不可用")
	}
	if req.AssigneeID > 0 {
		assigneeExists, err := s.client.User.Query().
			Where(user.IDEQ(req.AssigneeID), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("验证处理人失败: %w", err)
		}
		if !assigneeExists {
			return fmt.Errorf("处理人不存在或不可用")
		}
	}
	if req.CategoryID != nil {
		categoryExists, err := s.client.TicketCategory.Query().
			Where(ticketcategory.IDEQ(*req.CategoryID), ticketcategory.TenantIDEQ(tenantID), ticketcategory.IsActiveEQ(true)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("验证工单分类失败: %w", err)
		}
		if !categoryExists {
			return fmt.Errorf("工单分类不存在或不可用")
		}
	}
	if req.TemplateID != nil {
		templateExists, err := s.client.TicketTemplate.Query().
			Where(tickettemplate.IDEQ(*req.TemplateID), tickettemplate.TenantIDEQ(tenantID), tickettemplate.IsActiveEQ(true)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("验证工单模板失败: %w", err)
		}
		if !templateExists {
			return fmt.Errorf("工单模板不存在或不可用")
		}
	}
	if req.ParentTicketID != nil {
		parentExists, err := s.client.Ticket.Query().
			Where(entTicket.IDEQ(*req.ParentTicketID), entTicket.TenantIDEQ(tenantID), entTicket.DeletedAtIsNil()).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("验证父工单失败: %w", err)
		}
		if !parentExists {
			return fmt.Errorf("父工单不存在")
		}
	}
	tagIDs := uniqueIDs(req.TagIDs)
	if len(tagIDs) > 0 {
		count, err := s.client.TicketTag.Query().
			Where(tickettag.IDIn(tagIDs...), tickettag.TenantIDEQ(tenantID), tickettag.IsActiveEQ(true)).
			Count(ctx)
		if err != nil {
			return fmt.Errorf("验证工单标签失败: %w", err)
		}
		if count != len(tagIDs) {
			return fmt.Errorf("工单标签不存在或不可用")
		}
	}
	return nil
}

func normalizeCreateTicketType(reqType string, formFields ...map[string]interface{}) ticket.Type {
	if isSupportedTicketType(reqType) {
		return ticket.Type(reqType)
	}

	for _, fields := range formFields {
		if fields == nil {
			continue
		}
		if value, ok := fields["type"].(string); ok && isSupportedTicketType(value) {
			return ticket.Type(value)
		}
	}

	return ticket.TypeIncident
}

func isSupportedTicketType(value string) bool {
	switch ticket.Type(value) {
	case ticket.TypeIncident, ticket.TypeProblem, ticket.TypeChange, ticket.TypeServiceRequest, "improvement", "ticket":
		return true
	default:
		return false
	}
}

// triggerWorkflowForTicket 异步触发工单关联的 BPMN 流程
// 逻辑参考 V1 (ticket_service.go:221-279)，适配 V2 的 DDD 领域模型
func (s *TicketService) triggerWorkflowForTicket(ctx context.Context, tkt *ticket.Ticket, tenantID int, workflowDefinitionKey string) error {
	// 构造流程变量
	variables := map[string]interface{}{
		"ticket_id":     tkt.ID,
		"ticket_number": tkt.TicketNumber,
		"title":         tkt.Title,
		"description":   tkt.Description,
		"priority":      string(tkt.Priority),
		"status":        string(tkt.Status),
		"requester_id":  tkt.RequesterID,
	}
	if tkt.AssigneeID != nil {
		variables["assignee_id"] = *tkt.AssigneeID
	}

	// 解析 process key：1.请求指定 2.Resolver 3.兜底
	processKey := workflowDefinitionKey
	if processKey == "" && s.processResolver != nil {
		// ProcessResolver 当前接口需要 *ent.Ticket，临时转换为 ent 适配
		resolved, err := s.processResolver.ResolveWithPriority(ctx, s.toEntTicket(tkt), workflowDefinitionKey)
		if err != nil {
			return fmt.Errorf("failed to resolve process key: %w", err)
		}
		processKey = resolved
	}
	if processKey == "" {
		processKey = "ticket_general_flow"
	}

	triggerReq := &dto.ProcessTriggerRequest{
		BusinessType:         dto.BusinessTypeTicket,
		BusinessID:           tkt.ID,
		ProcessDefinitionKey: processKey,
		Variables:            variables,
		TriggeredBy:          fmt.Sprintf("%d", tkt.RequesterID),
		TriggeredAt:          time.Now(),
		TenantID:             tenantID,
	}

	resp, err := s.processTriggerSvc.TriggerProcess(ctx, triggerReq)
	if err != nil {
		return fmt.Errorf("failed to trigger workflow: %w", err)
	}

	s.logger.Infow(
		"Workflow triggered for ticket",
		"ticket_id", tkt.ID,
		"process_instance_id", resp.ProcessInstanceID,
		"process_key", processKey,
		"business_key", resp.BusinessKey,
	)
	return nil
}

// GetWorkflowStatus 获取工单关联的流程状态
// 与 V1 (ticket_service.go:282-319) 等价
func (s *TicketService) GetWorkflowStatus(ctx context.Context, ticketID int, tenantID int) (*dto.ProcessTriggerResponse, error) {
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for workflow status query")
	}
	businessKey := fmt.Sprintf("ticket:%d", ticketID)

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

	processDefName := ""
	if processInstance.Edges.Definition != nil {
		processDefName = processInstance.Edges.Definition.Name
	}

	return &dto.ProcessTriggerResponse{
		ProcessInstanceID:     processInstance.ID,
		ProcessDefinitionKey:  processInstance.ProcessDefinitionKey,
		ProcessDefinitionName: processDefName,
		BusinessKey:           processInstance.BusinessKey,
		Status:                mapProcessStatusToDTO(processInstance.Status),
		CurrentActivityID:     processInstance.CurrentActivityID,
		CurrentActivityName:   processInstance.CurrentActivityName,
		StartTime:             processInstance.StartTime,
		EndTime:               &processInstance.EndTime,
	}, nil
}

// CancelWorkflow 取消工单关联的流程
func (s *TicketService) CancelWorkflow(ctx context.Context, ticketID int, tenantID int, reason string) error {
	if s.client == nil {
		return fmt.Errorf("ent client not available for workflow cancel")
	}
	businessKey := fmt.Sprintf("ticket:%d", ticketID)

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

	if s.processTriggerSvc != nil {
		return s.processTriggerSvc.CancelProcess(ctx, processInstance.ID, reason, tenantID)
	}
	return fmt.Errorf("流程触发服务未配置")
}

// SyncTicketStatusWithWorkflow 同步工单状态与流程状态
func (s *TicketService) SyncTicketStatusWithWorkflow(ctx context.Context, ticketID int, tenantID int) error {
	workflowStatus, err := s.GetWorkflowStatus(ctx, ticketID, tenantID)
	if err != nil {
		s.logger.Warnw("Failed to get workflow status for sync", "error", err, "ticket_id", ticketID)
		return err
	}

	var newStatus ticket.Status
	switch workflowStatus.Status {
	case dto.ProcessStatusCompleted:
		newStatus = ticket.StatusResolved
	case dto.ProcessStatusTerminated, dto.ProcessStatusSuspended:
		newStatus = ticket.StatusPending
	default:
		return nil
	}

	if _, err := s.repo.UpdateStatus(ctx, ticketID, newStatus, tenantID); err != nil {
		return fmt.Errorf("同步工单状态失败: %w", err)
	}

	s.logger.Infow(
		"Ticket status synced with workflow",
		"ticket_id", ticketID,
		"workflow_status", workflowStatus.Status,
		"ticket_status", string(newStatus),
	)
	return nil
}

// mapProcessStatusToDTO 映射流程状态（与 V1 mapProcessStatus 等价）
func mapProcessStatusToDTO(status string) dto.ProcessStatus {
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

// GetTicket 获取工单
func (s *TicketService) GetTicket(ctx context.Context, id int, tenantID int) (*ticket.Ticket, error) {
	updated, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// 异步同步工单到飞书
	if s.connectorManager != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// 获取Feishu连接器
			conn, ok := s.connectorManager.Get(tenantID, "feishu")
			if !ok {
				// 飞书连接器未配置，忽略
				return
			}
			feishuConn, ok := conn.(*feishuConnector.Feishu)
			if !ok {
				return
			}
			// 开启事务
			tx, err := s.client.Tx(ctx2)
			if err != nil {
				s.logger.Warnw("Failed to start transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
			defer tx.Rollback()
			// 同步工单到飞书
			_, err = feishuConn.SyncTicketToFeishu(ctx2, tx, s.toEntTicket(updated))
			if err != nil {
				s.logger.Warnw("Failed to sync ticket to feishu", "error", err, "ticket_id", updated.ID)
				return
			}
			// 提交事务
			if err := tx.Commit(); err != nil {
				s.logger.Warnw("Failed to commit transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
		}()
	}

	return updated, nil
}

// GetTicketByNumber 根据编号获取工单
func (s *TicketService) GetTicketByNumber(ctx context.Context, ticketNumber string, tenantID int) (*ticket.Ticket, error) {
	return s.repo.GetByNumber(ctx, ticketNumber, tenantID)
}

// UpdateTicket 更新工单
func (s *TicketService) UpdateTicket(ctx context.Context, id int, req *dto.UpdateTicketRequest, tenantID int) (*ticket.Ticket, error) {
	s.logger.Infow("Updating ticket", "ticket_id", id, "tenant_id", tenantID)

	// 获取当前工单
	current, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// 状态转换验证
	if req.Status != "" && ticket.Status(req.Status) != current.Status {
		if !current.CanTransitionTo(ticket.Status(req.Status)) {
			return nil, &ticket.StateError{
				CurrentStatus: current.Status,
				Message:       "invalid state transition",
			}
		}
	}
	if req.Status == string(ticket.StatusResolved) && strings.TrimSpace(req.Resolution) == "" &&
		(current.Resolution == nil || strings.TrimSpace(*current.Resolution) == "") {
		return nil, fmt.Errorf("解决工单时必须填写解决方案")
	}

	// 转换更新参数
	params := &ticket.UpdateParams{
		Version: current.Version,
	}
	if req.Version > 0 {
		params.Version = req.Version
	}

	if req.Title != "" {
		params.Title = &req.Title
	}
	if req.Description != "" {
		params.Description = &req.Description
	}
	if req.Status != "" {
		status := ticket.Status(req.Status)
		params.Status = &status
	}
	if req.Type != "" {
		if req.Type != "ticket" && !isSupportedTicketType(req.Type) {
			return nil, fmt.Errorf("无效的工单类型: %s", req.Type)
		}
		ticketType := normalizeCreateTicketType(req.Type)
		params.Type = &ticketType
	}
	if req.Priority != "" {
		priority := ticket.Priority(req.Priority)
		params.Priority = &priority
	}
	if req.AssigneeID != 0 {
		if s.client != nil {
			assigneeExists, err := s.client.User.Query().
				Where(user.IDEQ(req.AssigneeID), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).
				Exist(ctx)
			if err != nil {
				return nil, fmt.Errorf("验证处理人失败: %w", err)
			}
			if !assigneeExists {
				return nil, fmt.Errorf("处理人不存在或不可用")
			}
		}
		params.AssigneeID = &req.AssigneeID
	}
	categoryID := req.CategoryID
	if categoryID == nil && strings.TrimSpace(req.Category) != "" {
		if s.client == nil {
			return nil, fmt.Errorf("无法解析工单分类")
		}
		category, err := s.client.TicketCategory.Query().
			Where(ticketcategory.NameEQ(strings.TrimSpace(req.Category)), ticketcategory.TenantIDEQ(tenantID), ticketcategory.IsActiveEQ(true)).
			Only(ctx)
		if err != nil {
			return nil, fmt.Errorf("工单分类不存在或不可用")
		}
		categoryID = &category.ID
	}
	if categoryID != nil {
		if *categoryID != 0 && s.client != nil {
			exists, err := s.client.TicketCategory.Query().
				Where(ticketcategory.IDEQ(*categoryID), ticketcategory.TenantIDEQ(tenantID), ticketcategory.IsActiveEQ(true)).
				Exist(ctx)
			if err != nil {
				return nil, fmt.Errorf("验证工单分类失败: %w", err)
			}
			if !exists {
				return nil, fmt.Errorf("工单分类不存在或不可用")
			}
		}
		params.CategoryID = categoryID
	}
	if req.Tags != nil {
		params.ReplaceTags = true
		if len(req.Tags) > 0 {
			if s.client == nil {
				return nil, fmt.Errorf("无法解析工单标签")
			}
			tagIDs, err := NewTicketTagService(s.client).ResolveTagIDsByNames(ctx, req.Tags, tenantID, true)
			if err != nil {
				return nil, fmt.Errorf("解析工单标签失败: %w", err)
			}
			params.TagIDs = tagIDs
		}
	}
	if req.Resolution != "" {
		params.Resolution = &req.Resolution
	}

	// 更新工单
	updated, err := s.repo.Update(ctx, id, params, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to update ticket", "error", err)
		return nil, err
	}

	s.logger.Infow("Ticket updated", "ticket_id", id)

	// 异步同步工单到飞书
	if s.connectorManager != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// 获取Feishu连接器
			conn, ok := s.connectorManager.Get(tenantID, "feishu")
			if !ok {
				// 飞书连接器未配置，忽略
				return
			}
			feishuConn, ok := conn.(*feishuConnector.Feishu)
			if !ok {
				return
			}
			// 开启事务
			tx, err := s.client.Tx(ctx2)
			if err != nil {
				s.logger.Warnw("Failed to start transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
			defer tx.Rollback()
			// 同步工单到飞书
			_, err = feishuConn.SyncTicketToFeishu(ctx2, tx, s.toEntTicket(updated))
			if err != nil {
				s.logger.Warnw("Failed to sync ticket to feishu", "error", err, "ticket_id", updated.ID)
				return
			}
			// 提交事务
			if err := tx.Commit(); err != nil {
				s.logger.Warnw("Failed to commit transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
		}()
	}

	return updated, nil
}

// DeleteTicket 删除工单
func (s *TicketService) DeleteTicket(ctx context.Context, id int, tenantID int) error {
	return s.repo.Delete(ctx, id, tenantID)
}

// ListTickets 列表查询工单
func (s *TicketService) ListTickets(ctx context.Context, req *dto.ListTicketsRequest, tenantID int) (*dto.ListTicketsResponse, error) {
	// 构建过滤参数
	filters := &ticket.FilterParams{}
	if req.Status != "" {
		status := ticket.Status(req.Status)
		filters.Status = &status
	}
	if req.Priority != "" {
		priority := ticket.Priority(req.Priority)
		filters.Priority = &priority
	}
	if req.RequesterID != nil {
		filters.RequesterID = req.RequesterID
	}
	if req.AssigneeID != nil {
		filters.AssigneeID = req.AssigneeID
	}
	if req.Type != "" {
		ticketType := ticket.Type(req.Type)
		filters.Type = &ticketType
	}
	if req.CategoryID != nil {
		filters.CategoryID = req.CategoryID
	}
	if req.ParentTicketID != nil {
		filters.ParentTicketID = req.ParentTicketID
	}
	filters.IsOverdue = req.IsOverdue
	if req.Keyword != "" {
		filters.Keyword = req.Keyword
	}
	if req.DateFrom != nil {
		filters.DateFrom = req.DateFrom
	}
	if req.DateTo != nil {
		filters.DateTo = req.DateTo
	}

	// 分页参数
	pagination := &base.QueryParams{
		Page:     req.Page,
		PageSize: req.PageSize,
		OrderBy:  req.SortBy,
		OrderDir: req.SortOrder,
	}

	// 查询
	result, err := s.repo.List(ctx, tenantID, filters, pagination)
	if err != nil {
		return nil, err
	}

	// 转换为 DTO
	response := &dto.ListTicketsResponse{
		Total:    result.Total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Tickets:  make([]*dto.TicketResponse, len(result.Data)),
	}

	for i, t := range result.Data {
		response.Tickets[i] = s.toTicketResponse(t)
	}

	return response, nil
}

// AssignTicket 分配工单
func (s *TicketService) AssignTicket(ctx context.Context, ticketID int, assigneeID int, tenantID int) (*ticket.Ticket, error) {
	s.logger.Infow("Assigning ticket", "ticket_id", ticketID, "assignee_id", assigneeID)
	current, err := s.repo.GetByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}
	if err := current.Assign(assigneeID); err != nil {
		return nil, err
	}
	if s.client != nil {
		exists, err := s.client.User.Query().
			Where(user.IDEQ(assigneeID), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("验证处理人失败: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("处理人不存在或不可用")
		}
	}
	status := current.Status
	updated, err := s.repo.Update(ctx, ticketID, &ticket.UpdateParams{
		AssigneeID: &assigneeID,
		Status:     &status,
		Version:    current.Version,
	}, tenantID)
	if err != nil {
		return nil, err
	}

	// 发送通知
	if s.notificationSvc != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.notificationSvc.NotifyTicketAssigned(ctx2, ticketID, assigneeID, tenantID); err != nil {
				s.logger.Warnw("Assignment notification failed", "error", err)
			}
		}()
	}

	// 异步同步工单到飞书
	if s.connectorManager != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// 获取Feishu连接器
			conn, ok := s.connectorManager.Get(tenantID, "feishu")
			if !ok {
				// 飞书连接器未配置，忽略
				return
			}
			feishuConn, ok := conn.(*feishuConnector.Feishu)
			if !ok {
				return
			}
			// 开启事务
			tx, err := s.client.Tx(ctx2)
			if err != nil {
				s.logger.Warnw("Failed to start transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
			defer tx.Rollback()
			// 同步工单到飞书
			_, err = feishuConn.SyncTicketToFeishu(ctx2, tx, s.toEntTicket(updated))
			if err != nil {
				s.logger.Warnw("Failed to sync ticket to feishu", "error", err, "ticket_id", updated.ID)
				return
			}
			// 提交事务
			if err := tx.Commit(); err != nil {
				s.logger.Warnw("Failed to commit transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
		}()
	}

	return updated, nil
}

// ResolveTicket 解决工单
func (s *TicketService) ResolveTicket(ctx context.Context, ticketID int, resolution string, tenantID int) (*ticket.Ticket, error) {
	s.logger.Infow("Resolving ticket", "ticket_id", ticketID)
	resolution = strings.TrimSpace(resolution)
	if resolution == "" {
		return nil, fmt.Errorf("解决方案不能为空")
	}

	// 获取工单
	tkt, err := s.repo.GetByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}

	// 状态转换验证
	if !tkt.CanTransitionTo(ticket.StatusResolved) {
		return nil, &ticket.StateError{
			CurrentStatus: tkt.Status,
			Message:       "cannot resolve ticket from current status",
		}
	}

	status := ticket.StatusResolved
	updated, err := s.repo.Update(ctx, ticketID, &ticket.UpdateParams{
		Status:     &status,
		Resolution: &resolution,
		Version:    tkt.Version,
	}, tenantID)
	if err != nil {
		return nil, err
	}

	s.logger.Infow("Ticket resolved", "ticket_id", ticketID)

	// 异步同步工单到飞书
	if s.connectorManager != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// 获取Feishu连接器
			conn, ok := s.connectorManager.Get(tenantID, "feishu")
			if !ok {
				// 飞书连接器未配置，忽略
				return
			}
			feishuConn, ok := conn.(*feishuConnector.Feishu)
			if !ok {
				return
			}
			// 开启事务
			tx, err := s.client.Tx(ctx2)
			if err != nil {
				s.logger.Warnw("Failed to start transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
			defer tx.Rollback()
			// 同步工单到飞书
			_, err = feishuConn.SyncTicketToFeishu(ctx2, tx, s.toEntTicket(updated))
			if err != nil {
				s.logger.Warnw("Failed to sync ticket to feishu", "error", err, "ticket_id", updated.ID)
				return
			}
			// 提交事务
			if err := tx.Commit(); err != nil {
				s.logger.Warnw("Failed to commit transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
		}()
	}

	return updated, nil
}

// CloseTicket 关闭工单
func (s *TicketService) CloseTicket(ctx context.Context, ticketID int, tenantID int, feedback string) (*ticket.Ticket, error) {
	s.logger.Infow("Closing ticket", "ticket_id", ticketID, "tenant_id", tenantID)

	// 获取工单
	tkt, err := s.repo.GetByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}

	// 状态转换验证
	if !tkt.CanTransitionTo(ticket.StatusClosed) {
		return nil, &ticket.StateError{
			CurrentStatus: tkt.Status,
			Message:       "cannot close ticket from current status",
		}
	}

	status := ticket.StatusClosed
	params := &ticket.UpdateParams{Status: &status, Version: tkt.Version}
	if strings.TrimSpace(feedback) != "" {
		feedback = strings.TrimSpace(feedback)
		params.Resolution = &feedback
	}
	updated, err := s.repo.Update(ctx, ticketID, params, tenantID)
	if err != nil {
		return nil, err
	}

	s.logger.Infow("Ticket closed", "ticket_id", ticketID)

	// 异步同步工单到飞书
	if s.connectorManager != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// 获取Feishu连接器
			conn, ok := s.connectorManager.Get(tenantID, "feishu")
			if !ok {
				// 飞书连接器未配置，忽略
				return
			}
			feishuConn, ok := conn.(*feishuConnector.Feishu)
			if !ok {
				return
			}
			// 开启事务
			tx, err := s.client.Tx(ctx2)
			if err != nil {
				s.logger.Warnw("Failed to start transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
			defer tx.Rollback()
			// 同步工单到飞书
			_, err = feishuConn.SyncTicketToFeishu(ctx2, tx, s.toEntTicket(updated))
			if err != nil {
				s.logger.Warnw("Failed to sync ticket to feishu", "error", err, "ticket_id", updated.ID)
				return
			}
			// 提交事务
			if err := tx.Commit(); err != nil {
				s.logger.Warnw("Failed to commit transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
		}()
	}

	return updated, nil
}

// GetTicketStats 获取工单统计
func (s *TicketService) GetTicketStats(ctx context.Context, tenantID int) (*dto.TicketStatsResponse, error) {
	statusCounts, err := s.repo.CountByStatus(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	overdue, err := s.repo.FindOverdue(ctx, tenantID)
	if err != nil {
		s.logger.Warnw("Failed to get overdue tickets", "error", err)
	}

	total := 0
	for _, count := range statusCounts {
		total += count
	}

	return &dto.TicketStatsResponse{
		Total:        total,
		Open:         statusCounts[ticket.StatusNew] + statusCounts[ticket.StatusOpen],
		InProgress:   statusCounts[ticket.StatusInProgress],
		Resolved:     statusCounts[ticket.StatusResolved],
		Pending:      statusCounts[ticket.StatusNew] + statusCounts[ticket.StatusPending],
		HighPriority: 0, // 需要单独查询
		Overdue:      len(overdue),
	}, nil
}

// ==================== 辅助方法 ====================

// toTicketResponse 转换为 DTO 响应
func (s *TicketService) toTicketResponse(t *ticket.Ticket) *dto.TicketResponse {
	resp := &dto.TicketResponse{
		ID:           t.ID,
		TicketNumber: t.TicketNumber,
		Title:        t.Title,
		Description:  t.Description,
		Status:       string(t.Status),
		Priority:     string(t.Priority),
		Type:         string(t.Type),
		RequesterID:  t.RequesterID,
		TenantID:     t.TenantID,
		Version:      t.Version,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}

	if t.AssigneeID != nil {
		resp.AssigneeID = *t.AssigneeID
	}
	if t.CategoryID != nil {
		resp.CategoryID = *t.CategoryID
	}
	if t.DepartmentID != nil {
		resp.DepartmentID = *t.DepartmentID
	}
	if t.ParentTicketID != nil {
		resp.ParentTicketID = *t.ParentTicketID
	}
	if t.Resolution != nil {
		resp.Resolution = *t.Resolution
	}
	resp.ResolvedAt = t.ResolvedAt
	resp.ClosedAt = t.ClosedAt
	resp.FirstResponseAt = t.FirstResponseAt
	resp.SLAResponseDeadline = t.SLAResponseDeadline
	resp.SLAResolutionDeadline = t.SLAResolutionDeadline

	return resp
}

// toEntTicket 转换为 Ent 工单（兼容现有 ProcessResolver / BPMN 触发）
// 用于 BPMN 流程解析、触发、状态同步等需要走 Ent 查询的场景。
// 这是一个临时方案：理想情况下 ProcessResolver 应该接受领域模型。
func (s *TicketService) toEntTicket(t *ticket.Ticket) *ent.Ticket {
	entTicket := &ent.Ticket{
		ID:           t.ID,
		TicketNumber: t.TicketNumber,
		Title:        t.Title,
		Description:  t.Description,
		Status:       string(t.Status),
		Type:         string(t.Type),
		Priority:     string(t.Priority),
		RequesterID:  t.RequesterID,
		TenantID:     t.TenantID,
		Version:      t.Version,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
	if t.AssigneeID != nil {
		entTicket.AssigneeID = *t.AssigneeID
	}
	if t.CategoryID != nil {
		entTicket.CategoryID = *t.CategoryID
	}
	if t.Resolution != nil {
		entTicket.Resolution = *t.Resolution
	}
	if t.ResolvedAt != nil {
		entTicket.ResolvedAt = *t.ResolvedAt
	}
	entTicket.ClosedAt = t.ClosedAt
	return entTicket
}

// ==================== 状态变更 / SLA / 批量 / 升级 / 查询（V1 兼容） ====================

// UpdateTicketStatus 更新工单状态（等价 V1.TicketService.UpdateTicketStatus）
func (s *TicketService) UpdateTicketStatus(ctx context.Context, ticketID int, status string, tenantID int, operatorID int) (*ticket.Ticket, error) {
	s.logger.Infow("Updating ticket status", "ticket_id", ticketID, "status", status, "tenant_id", tenantID, "operator_id", operatorID)

	current, err := s.repo.GetByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	if !IsValidTicketStatusTransition(string(current.Status), status) {
		return nil, fmt.Errorf("invalid status transition: %s -> %s", current.Status, status)
	}
	if status == string(ticket.StatusResolved) && (current.Resolution == nil || strings.TrimSpace(*current.Resolution) == "") {
		return nil, fmt.Errorf("解决工单必须通过 ResolveTicket 提交解决方案")
	}

	updated, err := s.repo.UpdateStatus(ctx, ticketID, ticket.Status(status), tenantID)
	if err != nil {
		s.logger.Errorw("Failed to update ticket status", "error", err, "ticket_id", ticketID)
		return nil, fmt.Errorf("failed to update ticket status: %w", err)
	}

	// 如果是解决或关闭状态，标记 FirstResponse / Resolved 时间
	if status == string(ticket.StatusResolved) || status == string(ticket.StatusClosed) {
		if updated.FirstResponseAt == nil {
			_ = s.repo.MarkFirstResponse(ctx, ticketID, tenantID)
		}
	}

	s.logger.Infow("Ticket status updated", "ticket_id", ticketID, "new_status", status)
	updated, err = s.repo.GetByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}

	// 异步同步工单到飞书
	if s.connectorManager != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// 获取Feishu连接器
			conn, ok := s.connectorManager.Get(tenantID, "feishu")
			if !ok {
				// 飞书连接器未配置，忽略
				return
			}
			feishuConn, ok := conn.(*feishuConnector.Feishu)
			if !ok {
				return
			}
			// 开启事务
			tx, err := s.client.Tx(ctx2)
			if err != nil {
				s.logger.Warnw("Failed to start transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
			defer tx.Rollback()
			// 同步工单到飞书
			_, err = feishuConn.SyncTicketToFeishu(ctx2, tx, s.toEntTicket(updated))
			if err != nil {
				s.logger.Warnw("Failed to sync ticket to feishu", "error", err, "ticket_id", updated.ID)
				return
			}
			// 提交事务
			if err := tx.Commit(); err != nil {
				s.logger.Warnw("Failed to commit transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
		}()
	}

	return updated, nil
}

// TicketSLAInfo 工单 SLA 信息（V2 内联定义，避免与 V1 重复）
type TicketSLAInfo struct {
	TicketID             int        `json:"ticketId"`
	TicketNumber         string     `json:"ticketNumber"`
	Priority             string     `json:"priority"`
	SLADefinitionID      int        `json:"slaDefinitionId"`
	SLADefinitionName    string     `json:"slaDefinitionName"`
	ResponseDeadline     time.Time  `json:"responseDeadline"`
	ResolutionDeadline   time.Time  `json:"resolutionDeadline"`
	ResponseTimeLeft     int        `json:"responseTimeLeftMinutes"`
	IsResponseBreached   bool       `json:"isResponseBreached"`
	ResolutionTimeLeft   int        `json:"resolutionTimeLeftMinutes"`
	IsResolutionBreached bool       `json:"isResolutionBreached"`
	FirstResponseAt      *time.Time `json:"firstResponseAt,omitempty"`
	ResolvedAt           *time.Time `json:"resolvedAt,omitempty"`
}

// GetTicketSLAInfo 获取工单 SLA 信息
func (s *TicketService) GetTicketSLAInfo(ctx context.Context, ticketID int, tenantID int) (*TicketSLAInfo, error) {
	tkt, err := s.repo.GetByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	info := &TicketSLAInfo{
		TicketID:         tkt.ID,
		TicketNumber:     tkt.TicketNumber,
		Priority:         string(tkt.Priority),
		SLADefinitionID:  0,
		ResponseDeadline: time.Time{},
	}
	if tkt.SLADefinitionID != nil {
		info.SLADefinitionID = *tkt.SLADefinitionID
	}
	if tkt.SLAResponseDeadline != nil {
		info.ResponseDeadline = *tkt.SLAResponseDeadline
	}
	if tkt.SLAResolutionDeadline != nil {
		info.ResolutionDeadline = *tkt.SLAResolutionDeadline
	}
	if tkt.FirstResponseAt != nil {
		info.FirstResponseAt = tkt.FirstResponseAt
	}
	if tkt.ResolvedAt != nil {
		info.ResolvedAt = tkt.ResolvedAt
	}

	// 获取 SLA 定义名称（通过 ent 客户端查询）
	if info.SLADefinitionID > 0 && s.client != nil {
		sla, err := s.client.SLADefinition.Get(ctx, info.SLADefinitionID)
		if err == nil && sla != nil {
			info.SLADefinitionName = sla.Name
		}
	}

	now := time.Now()
	if info.FirstResponseAt != nil {
		info.ResponseTimeLeft = 0
		info.IsResponseBreached = false
	} else if now.After(info.ResponseDeadline) && !info.ResponseDeadline.IsZero() {
		info.ResponseTimeLeft = 0
		info.IsResponseBreached = true
	} else if !info.ResponseDeadline.IsZero() {
		info.ResponseTimeLeft = int(info.ResponseDeadline.Sub(now).Minutes())
	}

	if info.ResolvedAt != nil {
		info.ResolutionTimeLeft = 0
		info.IsResolutionBreached = false
	} else if now.After(info.ResolutionDeadline) && !info.ResolutionDeadline.IsZero() {
		info.ResolutionTimeLeft = 0
		info.IsResolutionBreached = true
	} else if !info.ResolutionDeadline.IsZero() {
		info.ResolutionTimeLeft = int(info.ResolutionDeadline.Sub(now).Minutes())
	}

	return info, nil
}

// BatchDeleteTickets 批量删除工单
func (s *TicketService) BatchDeleteTickets(ctx context.Context, ticketIDs []int, tenantID int) error {
	s.logger.Infow("Batch deleting tickets", "ticket_ids", ticketIDs, "tenant_id", tenantID)
	if len(ticketIDs) == 0 {
		return nil
	}
	return s.repo.BatchDelete(ctx, ticketIDs, tenantID)
}

// EscalateTicket 升级工单
func (s *TicketService) EscalateTicket(ctx context.Context, ticketID int, reason string, tenantID int, escalatedBy int) (*ticket.Ticket, error) {
	s.logger.Infow("Escalating ticket", "ticket_id", ticketID, "reason", reason, "tenant_id", tenantID)

	current, err := s.repo.GetByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}

	newPriority := s.getEscalatedPriority(string(current.Priority))
	newAssignee := s.getEscalationAssignee(newPriority, tenantID)

	params := &ticket.UpdateParams{
		Version: current.Version,
		Priority: func() *ticket.Priority {
			p := ticket.Priority(newPriority)
			return &p
		}(),
		AssigneeID: &newAssignee,
		Status: func() *ticket.Status {
			st := ticket.StatusInProgress
			return &st
		}(),
	}

	updated, err := s.repo.Update(ctx, ticketID, params, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to escalate ticket", "error", err, "ticket_id", ticketID)
		return nil, fmt.Errorf("failed to escalate ticket: %w", err)
	}

	if s.notificationSvc != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = s.notificationSvc.NotifyTicketAssigned(ctx2, ticketID, newAssignee, tenantID)
		}()
	}

	s.logger.Infow("Ticket escalated", "ticket_id", ticketID, "new_priority", newPriority, "new_assignee", newAssignee)

	// 异步同步工单到飞书
	if s.connectorManager != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// 获取Feishu连接器
			conn, ok := s.connectorManager.Get(tenantID, "feishu")
			if !ok {
				// 飞书连接器未配置，忽略
				return
			}
			feishuConn, ok := conn.(*feishuConnector.Feishu)
			if !ok {
				return
			}
			// 开启事务
			tx, err := s.client.Tx(ctx2)
			if err != nil {
				s.logger.Warnw("Failed to start transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
			defer tx.Rollback()
			// 同步工单到飞书
			_, err = feishuConn.SyncTicketToFeishu(ctx2, tx, s.toEntTicket(updated))
			if err != nil {
				s.logger.Warnw("Failed to sync ticket to feishu", "error", err, "ticket_id", updated.ID)
				return
			}
			// 提交事务
			if err := tx.Commit(); err != nil {
				s.logger.Warnw("Failed to commit transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
		}()
	}

	return updated, nil
}

// SearchTickets 高级搜索工单
func (s *TicketService) SearchTickets(ctx context.Context, searchTerm string, tenantID int) ([]*ticket.Ticket, error) {
	s.logger.Infow("Searching tickets", "search_term", searchTerm, "tenant_id", tenantID)
	term := strings.TrimSpace(searchTerm)
	if term == "" {
		return []*ticket.Ticket{}, nil
	}
	// V2 Repository 暂不提供全文搜索，走 ent 客户端查询并转为领域模型
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for search")
	}
	ents, err := s.client.Ticket.Query().
		Where(
			entTicket.TenantID(tenantID),
			entTicket.Or(
				entTicket.TitleContains(strings.ToLower(term)),
				entTicket.DescriptionContains(strings.ToLower(term)),
			),
		).
		Order(ent.Desc(entTicket.FieldCreatedAt)).
		Limit(100).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to search tickets: %w", err)
	}
	result := make([]*ticket.Ticket, len(ents))
	for i, e := range ents {
		result[i] = s.entToDomain(e)
	}
	return result, nil
}

// GetOverdueTickets 获取逾期工单（V2 走 SLA 服务或 Repository 兜底）
func (s *TicketService) GetOverdueTickets(ctx context.Context, tenantID int) ([]*ticket.Ticket, error) {
	s.logger.Infow("Getting overdue tickets", "tenant_id", tenantID)
	if s.slaSvc != nil {
		ents, err := s.slaSvc.GetOverdueTickets(ctx, tenantID)
		if err == nil {
			result := make([]*ticket.Ticket, len(ents))
			for i, e := range ents {
				result[i] = s.entToDomain(e)
			}
			return result, nil
		}
		s.logger.Warnw("slaSvc.GetOverdueTickets failed, falling back", "error", err)
	}
	ents, err := s.repo.FindOverdue(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue tickets: %w", err)
	}
	return ents, nil
}

// GetTicketsByAssignee 获取指定处理人的工单
func (s *TicketService) GetTicketsByAssignee(ctx context.Context, assigneeID int, tenantID int) ([]*ticket.Ticket, error) {
	s.logger.Infow("Getting tickets by assignee", "assignee_id", assigneeID, "tenant_id", tenantID)
	return s.repo.FindByAssignee(ctx, assigneeID, tenantID)
}

// GetTicketActivity 获取工单活动日志（合并 comments、attachments、状态变更）
func (s *TicketService) GetTicketActivity(ctx context.Context, ticketID int, tenantID int) ([]map[string]interface{}, error) {
	s.logger.Infow("Getting ticket activity", "ticket_id", ticketID, "tenant_id", tenantID)
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for activity query")
	}
	tkt, err := s.repo.GetByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	activities := make([]map[string]interface{}, 0)

	activities = append(activities, map[string]interface{}{
		"action":    "created",
		"timestamp": tkt.CreatedAt,
		"user_id":   tkt.RequesterID,
		"user_name": "",
		"details":   "工单已创建",
		"old_value": nil,
		"new_value": tkt.Title,
	})

	comments, err := s.client.TicketComment.Query().
		Where(entTicketComment.TicketID(ticketID)).
		WithUser().
		Order(ent.Asc(entTicketComment.FieldCreatedAt)).
		All(ctx)
	if err == nil {
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
	} else {
		s.logger.Warnw("Failed to get comments for activity", "error", err)
	}

	if tkt.AssigneeID != nil && *tkt.AssigneeID > 0 {
		activities = append(activities, map[string]interface{}{
			"action":    "assigned",
			"timestamp": tkt.UpdatedAt,
			"user_id":   *tkt.AssigneeID,
			"user_name": "",
			"details":   "工单已分配",
			"old_value": nil,
			"new_value": *tkt.AssigneeID,
		})
	}
	if tkt.FirstResponseAt != nil {
		activities = append(activities, map[string]interface{}{
			"action":    "first_response",
			"timestamp": *tkt.FirstResponseAt,
			"user_id":   0,
			"details":   "首次响应工单",
		})
	}
	if tkt.ResolvedAt != nil {
		activities = append(activities, map[string]interface{}{
			"action":    "resolved",
			"timestamp": *tkt.ResolvedAt,
			"user_id":   0,
			"details":   "工单已解决",
		})
	}

	// 倒序
	for i, j := 0, len(activities)-1; i < j; i, j = i+1, j-1 {
		activities[i], activities[j] = activities[j], activities[i]
	}
	return activities, nil
}

// ==================== 辅助函数 ====================

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

// getEscalationAssignee 获取升级后的处理人
func (s *TicketService) getEscalationAssignee(priority string, tenantID int) int {
	switch priority {
	case "critical":
		return 1
	case "high":
		return 2
	default:
		return 3
	}
}

// entToDomain 将 ent.Ticket 转为领域模型（用于 SearchTickets / GetOverdueTickets 等结果适配）
func (s *TicketService) entToDomain(e *ent.Ticket) *ticket.Ticket {
	if e == nil {
		return nil
	}
	t := &ticket.Ticket{
		ID:           e.ID,
		TicketNumber: e.TicketNumber,
		Title:        e.Title,
		Description:  e.Description,
		Status:       ticket.Status(e.Status),
		Type:         ticket.Type(e.Type),
		Priority:     ticket.Priority(e.Priority),
		RequesterID:  e.RequesterID,
		TenantID:     e.TenantID,
		Version:      e.Version,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
	if e.AssigneeID > 0 {
		aid := e.AssigneeID
		t.AssigneeID = &aid
	}
	if e.CategoryID > 0 {
		cid := e.CategoryID
		t.CategoryID = &cid
	}
	if e.Resolution != "" {
		r := e.Resolution
		t.Resolution = &r
	}
	if !e.FirstResponseAt.IsZero() {
		ft := e.FirstResponseAt
		t.FirstResponseAt = &ft
	}
	if !e.ResolvedAt.IsZero() {
		rt := e.ResolvedAt
		t.ResolvedAt = &rt
	}
	return t
}

// ==================== 导出/导入/批量分配/分析 ====================

// ExportTickets 导出工单
func (s *TicketService) ExportTickets(ctx context.Context, tenantID int, filters map[string]interface{}, format string) ([]byte, error) {
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for export")
	}
	query := s.client.Ticket.Query().Where(entTicket.TenantID(tenantID))
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where(entTicket.StatusEQ(status))
	}
	if priority, ok := filters["priority"].(string); ok && priority != "" {
		query = query.Where(entTicket.PriorityEQ(priority))
	}
	tickets, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	exportData := make([]map[string]interface{}, 0, len(tickets))
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
	switch format {
	case "csv":
		parsed, err := s.parseCSV(data)
		if err != nil {
			return err
		}
		tickets = parsed
	case "excel":
		parsed, err := s.parseExcel(data)
		if err != nil {
			return err
		}
		tickets = parsed
	case "json":
		if err := json.Unmarshal(data, &tickets); err != nil {
			return err
		}
	default:
		return fmt.Errorf("不支持的导入格式: %s", format)
	}
	for _, ticketData := range tickets {
		title, _ := ticketData["标题"].(string)
		desc, _ := ticketData["描述"].(string)
		priority, _ := ticketData["优先级"].(string)
		_, err := s.CreateTicket(ctx, &dto.CreateTicketRequest{
			Title:       title,
			Description: desc,
			Priority:    priority,
			RequesterID: 1,
		}, tenantID)
		if err != nil {
			return fmt.Errorf("导入工单失败: %v", err)
		}
	}
	return nil
}

// AssignTickets 批量分配工单
func (s *TicketService) AssignTickets(ctx context.Context, tenantID int, ticketIDs []int, assigneeID int) error {
	if s.client == nil {
		return fmt.Errorf("ent client not available for assign")
	}
	if _, err := s.client.User.Get(ctx, assigneeID); err != nil {
		return fmt.Errorf("分配者不存在: %v", err)
	}
	for _, ticketID := range ticketIDs {
		if _, err := s.repo.AssignTicket(ctx, ticketID, assigneeID, tenantID); err != nil {
			return fmt.Errorf("分配工单 %d 失败: %v", ticketID, err)
		}
	}
	return nil
}

// BatchCloseTickets 批量关闭工单
func (s *TicketService) BatchCloseTickets(ctx context.Context, ticketIDs []int, tenantID int, closeReason string) error {
	for _, ticketID := range ticketIDs {
		if _, err := s.CloseTicket(ctx, ticketID, tenantID, closeReason); err != nil {
			return fmt.Errorf("关闭工单 %d 失败: %v", ticketID, err)
		}
	}
	return nil
}

// BatchUpdatePriority 批量更新优先级
func (s *TicketService) BatchUpdatePriority(ctx context.Context, ticketIDs []int, priority string, tenantID int) error {
	for _, ticketID := range ticketIDs {
		p := ticket.Priority(priority)
		_, err := s.repo.Update(ctx, ticketID, &ticket.UpdateParams{Priority: &p}, tenantID)
		if err != nil {
			return fmt.Errorf("更新工单 %d 优先级失败: %v", ticketID, err)
		}
	}
	return nil
}

// GetTicketAnalytics 获取工单分析数据
func (s *TicketService) GetTicketAnalytics(ctx context.Context, tenantID int, dateFrom, dateTo time.Time) (*dto.TicketAnalyticsResponse, error) {
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for analytics")
	}
	query := s.client.Ticket.Query().Where(entTicket.TenantID(tenantID))
	if !dateFrom.IsZero() {
		query = query.Where(entTicket.CreatedAtGTE(dateFrom))
	}
	if !dateTo.IsZero() {
		query = query.Where(entTicket.CreatedAtLTE(dateTo))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	tickets, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	statusStats := make(map[string]int)
	priorityStats := make(map[string]int)
	for _, t := range tickets {
		statusStats[t.Status]++
		priorityStats[t.Priority]++
	}
	resolvedTickets, err := query.Where(entTicket.StatusEQ("resolved")).All(ctx)
	if err != nil {
		return nil, err
	}
	var totalResolutionTime time.Duration
	resolvedCount := 0
	for _, t := range resolvedTickets {
		if !t.UpdatedAt.IsZero() {
			totalResolutionTime += t.UpdatedAt.Sub(t.CreatedAt)
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

// ==================== 模板 CRUD ====================

// CreateTicketTemplate 创建工单模板
func (s *TicketService) CreateTicketTemplate(ctx context.Context, tenantID int, req interface{}) (interface{}, error) {
	createReq, ok := req.(*dto.TicketTemplate)
	if !ok {
		return nil, fmt.Errorf("无效的请求参数类型")
	}
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for template")
	}
	formFields := createReq.FormFields
	if formFields == nil {
		formFields = make(map[string]interface{})
	}
	if len(createReq.Fields) > 0 {
		formFields["fields"] = createReq.Fields
	}
	isActive := true
	priority := strings.TrimSpace(createReq.Priority)
	if priority == "" {
		priority = "medium"
	}

	templateService := NewTicketTemplateService(s.client)
	serviceReq := &CreateTemplateRequest{
		Name:          createReq.Name,
		Description:   createReq.Description,
		Category:      createReq.Category,
		Priority:      priority,
		FormFields:    formFields,
		WorkflowSteps: createReq.WorkflowSteps,
		IsActive:      isActive,
		TenantID:      tenantID,
	}
	template, err := templateService.CreateTemplate(ctx, serviceReq)
	if err != nil {
		return nil, err
	}
	return s.toTicketTemplateDTO(template)
}

// UpdateTicketTemplate 更新工单模板
func (s *TicketService) UpdateTicketTemplate(ctx context.Context, tenantID int, templateID int, req interface{}) (interface{}, error) {
	updateReq, ok := req.(*dto.TicketTemplate)
	if !ok {
		return nil, fmt.Errorf("无效的请求参数类型")
	}
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for template")
	}
	formFields := updateReq.FormFields
	if formFields == nil && len(updateReq.Fields) > 0 {
		formFields = map[string]interface{}{"fields": updateReq.Fields}
	} else if len(updateReq.Fields) > 0 {
		formFields["fields"] = updateReq.Fields
	}
	var isActive *bool
	if updateReq.IsActiveAlt != nil {
		isActive = updateReq.IsActiveAlt
	}
	priority := strings.TrimSpace(updateReq.Priority)
	templateService := NewTicketTemplateService(s.client)
	serviceReq := &UpdateTemplateRequest{
		Name:          updateReq.Name,
		Description:   updateReq.Description,
		Category:      updateReq.Category,
		Priority:      priority,
		FormFields:    formFields,
		WorkflowSteps: updateReq.WorkflowSteps,
		IsActive:      isActive,
	}
	template, err := templateService.UpdateTemplate(ctx, templateID, serviceReq, tenantID)
	if err != nil {
		return nil, err
	}
	return s.toTicketTemplateDTO(template)
}

// DeleteTicketTemplate 删除工单模板
func (s *TicketService) DeleteTicketTemplate(ctx context.Context, tenantID int, templateID int) error {
	if s.client == nil {
		return fmt.Errorf("ent client not available for template")
	}
	templateService := NewTicketTemplateService(s.client)
	return templateService.DeleteTemplate(ctx, templateID, tenantID)
}

// GetTicketTemplates 获取工单模板列表
func (s *TicketService) GetTicketTemplates(ctx context.Context, tenantID int) ([]interface{}, error) {
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for template")
	}
	templateService := NewTicketTemplateService(s.client)
	templates, _, err := templateService.ListTemplates(ctx, &ListTemplatesRequest{
		Page:      1,
		PageSize:  100,
		TenantID:  tenantID,
		SortBy:    "created_at",
		SortOrder: "desc",
	})
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, 0, len(templates))
	for _, template := range templates {
		templateDTO, err := s.toTicketTemplateDTO(template)
		if err != nil {
			return nil, err
		}
		result = append(result, templateDTO)
	}
	return result, nil
}

// GetTicketTemplate 获取工单模板详情
func (s *TicketService) GetTicketTemplate(ctx context.Context, tenantID int, templateID int) (*dto.TicketTemplate, error) {
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for template")
	}
	templateService := NewTicketTemplateService(s.client)
	template, err := templateService.GetTemplate(ctx, templateID, tenantID)
	if err != nil {
		return nil, err
	}
	return s.toTicketTemplateDTO(template)
}

// UpdateTicketTemplateStatus 启用或停用工单模板
func (s *TicketService) UpdateTicketTemplateStatus(ctx context.Context, tenantID int, templateID int, isActive bool) (*dto.TicketTemplate, error) {
	returned, err := s.UpdateTicketTemplate(ctx, tenantID, templateID, &dto.TicketTemplate{
		IsActiveAlt: &isActive,
	})
	if err != nil {
		return nil, err
	}
	template, ok := returned.(*dto.TicketTemplate)
	if !ok {
		return nil, fmt.Errorf("invalid template response type")
	}
	return template, nil
}

// CopyTicketTemplate 复制工单模板
func (s *TicketService) CopyTicketTemplate(ctx context.Context, tenantID int, templateID int, newName string) (*dto.TicketTemplate, error) {
	source, err := s.GetTicketTemplate(ctx, tenantID, templateID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(newName) == "" {
		newName = source.Name + " - 副本"
	}
	copied, err := s.CreateTicketTemplate(ctx, tenantID, &dto.TicketTemplate{
		Name:          newName,
		Description:   source.Description,
		Category:      source.Category,
		Priority:      source.Priority,
		Fields:        source.Fields,
		FormFields:    source.FormFields,
		WorkflowSteps: source.WorkflowSteps,
		IsActive:      source.IsActive,
	})
	if err != nil {
		return nil, err
	}
	template, ok := copied.(*dto.TicketTemplate)
	if !ok {
		return nil, fmt.Errorf("invalid template response type")
	}
	return template, nil
}

// GetTicketTemplateCategories 获取模板分类
func (s *TicketService) GetTicketTemplateCategories(ctx context.Context, tenantID int) ([]string, error) {
	templates, err := s.GetTicketTemplates(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	categories := make([]string, 0)
	for _, item := range templates {
		template, ok := item.(*dto.TicketTemplate)
		if !ok || strings.TrimSpace(template.Category) == "" {
			continue
		}
		if !seen[template.Category] {
			seen[template.Category] = true
			categories = append(categories, template.Category)
		}
	}
	return categories, nil
}

func (s *TicketService) toTicketTemplateDTO(template *ent.TicketTemplate) (*dto.TicketTemplate, error) {
	var formFields map[string]interface{}
	if len(template.FormFields) > 0 {
		if err := json.Unmarshal(template.FormFields, &formFields); err != nil {
			s.logger.Warnw("反序列化表单字段失败", "error", err, "template_id", template.ID)
			formFields = make(map[string]interface{})
		}
	} else {
		formFields = make(map[string]interface{})
	}

	var workflowSteps []map[string]interface{}
	if len(template.WorkflowSteps) > 0 {
		if err := json.Unmarshal(template.WorkflowSteps, &workflowSteps); err != nil {
			s.logger.Warnw("反序列化工作流步骤失败", "error", err, "template_id", template.ID)
			workflowSteps = nil
		}
	}

	fields := make([]map[string]interface{}, 0)
	if rawFields, ok := formFields["fields"]; ok {
		if encoded, err := json.Marshal(rawFields); err == nil {
			_ = json.Unmarshal(encoded, &fields)
		}
	}

	isActive := template.IsActive
	return &dto.TicketTemplate{
		ID:            template.ID,
		Name:          template.Name,
		Description:   template.Description,
		Category:      template.Category,
		Priority:      template.Priority,
		Fields:        fields,
		FormFields:    formFields,
		WorkflowSteps: workflowSteps,
		IsActive:      isActive,
		CreatedAt:     template.CreatedAt,
		UpdatedAt:     template.UpdatedAt,
	}, nil
}

// ==================== CSV / Excel / JSON 独立实现（V2 不依赖 V1） ====================

// generateCSV 生成 CSV
func (s *TicketService) generateCSV(data []map[string]interface{}) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	var headers []string
	for key := range data[0] {
		headers = append(headers, key)
	}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}
	for _, row := range data {
		record := make([]string, 0, len(headers))
		for _, header := range headers {
			value := row[header]
			if value == nil {
				record = append(record, "")
			} else {
				record = append(record, fmt.Sprintf("%v", value))
			}
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// generateExcel 生成 Excel（实际上返回 CSV，保持与 V1 一致的行为）
func (s *TicketService) generateExcel(data []map[string]interface{}) ([]byte, error) {
	return s.generateCSV(data)
}

// parseCSV 解析 CSV
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
	result := make([]map[string]interface{}, 0, len(records)-1)
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

// parseExcel 解析 Excel（与 V1 一致：暂返回空结果）
func (s *TicketService) parseExcel(data []byte) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

// ==================== MSP 相关方法 ====================

// GetCustomerTicketsForMSP 获取 MSP 视角下的客户工单
func (s *TicketService) GetCustomerTicketsForMSP(ctx context.Context, userID, customerTenantID int, status *string, page, pageSize int) ([]*ticket.Ticket, error) {
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for MSP query")
	}
	query := s.client.Ticket.Query().Where(entTicket.TenantIDEQ(customerTenantID))
	if status != nil && *status != "" {
		query = query.Where(entTicket.StatusEQ(*status))
	}
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}
	query = query.Order(ent.Desc(entTicket.FieldCreatedAt))
	ents, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer tickets for MSP: %w", err)
	}
	result := make([]*ticket.Ticket, len(ents))
	for i, e := range ents {
		result[i] = s.entToDomain(e)
	}
	return result, nil
}

// AssignMSPTechnician 为工单分配 MSP 技术员
func (s *TicketService) AssignMSPTechnician(ctx context.Context, ticketID, customerTenantID, assignerID int) (*ticket.Ticket, error) {
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for MSP assign")
	}
	t, err := s.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("工单不存在")
		}
		return nil, err
	}
	if t.TenantID != customerTenantID {
		return nil, fmt.Errorf("工单不属于指定客户租户")
	}
	// 分配给 MSP 技术员（这里 assignerID 作为目标处理人；可后续扩展为查表分配）
	if _, err := s.repo.AssignTicket(ctx, ticketID, assignerID, customerTenantID); err != nil {
		return nil, fmt.Errorf("failed to assign MSP technician: %w", err)
	}
	updated, err := s.repo.GetByID(ctx, ticketID, customerTenantID)
	if err != nil {
		return nil, err
	}

	// 异步同步工单到飞书
	if s.connectorManager != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// 获取Feishu连接器
			conn, ok := s.connectorManager.Get(customerTenantID, "feishu")
			if !ok {
				// 飞书连接器未配置，忽略
				return
			}
			feishuConn, ok := conn.(*feishuConnector.Feishu)
			if !ok {
				return
			}
			// 开启事务
			tx, err := s.client.Tx(ctx2)
			if err != nil {
				s.logger.Warnw("Failed to start transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
			defer tx.Rollback()
			// 同步工单到飞书
			_, err = feishuConn.SyncTicketToFeishu(ctx2, tx, s.toEntTicket(updated))
			if err != nil {
				s.logger.Warnw("Failed to sync ticket to feishu", "error", err, "ticket_id", updated.ID)
				return
			}
			// 提交事务
			if err := tx.Commit(); err != nil {
				s.logger.Warnw("Failed to commit transaction for feishu sync", "error", err, "ticket_id", updated.ID)
				return
			}
		}()
	}

	return updated, nil
}

// GetMSPCustomerReports 获取 MSP 客户报告
func (s *TicketService) GetMSPCustomerReports(ctx context.Context, mspTenantID int, dateFrom, dateTo time.Time) ([]map[string]interface{}, error) {
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for MSP reports")
	}
	query := s.client.Ticket.Query().Where(entTicket.TenantID(mspTenantID))
	if !dateFrom.IsZero() {
		query = query.Where(entTicket.CreatedAtGTE(dateFrom))
	}
	if !dateTo.IsZero() {
		query = query.Where(entTicket.CreatedAtLTE(dateTo))
	}
	tickets, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get MSP customer reports: %w", err)
	}
	reports := make([]map[string]interface{}, 0, len(tickets))
	statusCount := make(map[string]int)
	for _, t := range tickets {
		statusCount[t.Status]++
	}
	reports = append(reports, map[string]interface{}{
		"total_tickets":  len(tickets),
		"status_summary": statusCount,
		"date_from":      dateFrom,
		"date_to":        dateTo,
	})
	return reports, nil
}

// GetMSPPerformanceReports 获取 MSP 性能报告
func (s *TicketService) GetMSPPerformanceReports(ctx context.Context, mspTenantID int, dateFrom, dateTo time.Time) ([]map[string]interface{}, error) {
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for MSP performance")
	}
	query := s.client.Ticket.Query().Where(entTicket.TenantID(mspTenantID))
	if !dateFrom.IsZero() {
		query = query.Where(entTicket.CreatedAtGTE(dateFrom))
	}
	if !dateTo.IsZero() {
		query = query.Where(entTicket.CreatedAtLTE(dateTo))
	}
	tickets, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	resolvedCount := 0
	var totalResolutionTime time.Duration
	for _, t := range tickets {
		if t.Status == "resolved" {
			resolvedCount++
			if !t.UpdatedAt.IsZero() {
				totalResolutionTime += t.UpdatedAt.Sub(t.CreatedAt)
			}
		}
	}
	avgResolution := time.Duration(0)
	if resolvedCount > 0 {
		avgResolution = totalResolutionTime / time.Duration(resolvedCount)
	}
	return []map[string]interface{}{
		{
			"msp_tenant_id":       mspTenantID,
			"total_tickets":       len(tickets),
			"resolved_tickets":    resolvedCount,
			"avg_resolution_time": avgResolution.Hours(),
			"date_from":           dateFrom,
			"date_to":             dateTo,
		},
	}, nil
}
