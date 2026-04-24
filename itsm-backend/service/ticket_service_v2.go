package service

import (
	"context"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/repository/ticket"

	"go.uber.org/zap"
)

// TicketServiceV2 改进版的工单服务
// 使用构造函数注入和 Repository 模式
type TicketServiceV2 struct {
	repo              ticket.Repository
	logger            *zap.SugaredLogger
	notificationSvc   *TicketNotificationService
	approvalSvc       *ApprovalService
	automationRuleSvc *TicketAutomationRuleService
	slaSvc            *TicketSLAService
}

// TicketServiceV2Config 工单服务配置
// 所有依赖都在配置中明确声明
type TicketServiceV2Config struct {
	Repository            ticket.Repository
	Logger                *zap.SugaredLogger
	NotificationService   *TicketNotificationService
	ApprovalService       *ApprovalService
	AutomationRuleService *TicketAutomationRuleService
	SLAService            *TicketSLAService
}

// NewTicketServiceV2 创建工单服务
// 使用构造函数注入，所有依赖必须显式传入
func NewTicketServiceV2(cfg *TicketServiceV2Config) *TicketServiceV2 {
	if cfg.Repository == nil {
		panic("Repository is required")
	}
	if cfg.Logger == nil {
		panic("Logger is required")
	}

	return &TicketServiceV2{
		repo:              cfg.Repository,
		logger:            cfg.Logger,
		notificationSvc:   cfg.NotificationService,
		approvalSvc:       cfg.ApprovalService,
		automationRuleSvc: cfg.AutomationRuleService,
		slaSvc:            cfg.SLAService,
	}
}

// CreateTicket 创建工单
func (s *TicketServiceV2) CreateTicket(ctx context.Context, req *dto.CreateTicketRequest, tenantID int) (*ticket.Ticket, error) {
	s.logger.Infow("Creating ticket", "tenant_id", tenantID, "title", req.Title)

	// 转换 DTO 到领域参数
	params := &ticket.CreateParams{
		Title:       req.Title,
		Description: req.Description,
		Type:        ticket.Type(req.Type),
		Priority:    ticket.Priority(req.Priority),
		RequesterID: req.RequesterID,
	}

	if req.AssigneeID != nil {
		params.AssigneeID = req.AssigneeID
	}
	if req.CategoryID != nil {
		params.CategoryID = req.CategoryID
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

	// 异步触发审批（紧急和关键优先级）
	if s.approvalSvc != nil && (req.Priority == "urgent" || req.Priority == "critical") {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, err := s.approvalSvc.TriggerApproval(ctx2, &ApprovalTriggerRequest{
				TicketID:     tkt.ID,
				TicketNumber: tkt.TicketNumber,
				TicketTitle:  tkt.Title,
				TicketType:   string(tkt.Type),
				Priority:     string(tkt.Priority),
				RequesterID:  tkt.RequesterID,
				TenantID:     tenantID,
			})
			if err != nil {
				s.logger.Warnw("Approval trigger failed", "error", err)
			}
		}()
	}

	// 异步发送通知（如果分配了处理人）
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
	if s.automationRuleSvc != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.automationRuleSvc.ExecuteRulesForTicket(ctx2, tkt.ID, tenantID); err != nil {
				s.logger.Warnw("Automation rules failed", "error", err)
			}
		}()
	}

	s.logger.Infow("Ticket created", "ticket_id", tkt.ID, "ticket_number", tkt.TicketNumber)
	return tkt, nil
}

// GetTicket 获取工单
func (s *TicketServiceV2) GetTicket(ctx context.Context, id int, tenantID int) (*ticket.Ticket, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// GetTicketByNumber 根据编号获取工单
func (s *TicketServiceV2) GetTicketByNumber(ctx context.Context, ticketNumber string, tenantID int) (*ticket.Ticket, error) {
	return s.repo.GetByNumber(ctx, ticketNumber, tenantID)
}

// UpdateTicket 更新工单
func (s *TicketServiceV2) UpdateTicket(ctx context.Context, id int, req *dto.UpdateTicketRequest, tenantID int) (*ticket.Ticket, error) {
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

	// 转换更新参数
	params := &ticket.UpdateParams{
		Version: current.Version, // 乐观锁
	}

	if req.Title != nil {
		params.Title = req.Title
	}
	if req.Description != nil {
		params.Description = req.Description
	}
	if req.Status != "" {
		status := ticket.Status(req.Status)
		params.Status = &status
	}
	if req.Priority != "" {
		priority := ticket.Priority(req.Priority)
		params.Priority = &priority
	}
	if req.AssigneeID != nil {
		params.AssigneeID = req.AssigneeID
	}
	if req.Resolution != nil {
		params.Resolution = req.Resolution
	}

	// 更新工单
	updated, err := s.repo.Update(ctx, id, params, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to update ticket", "error", err)
		return nil, err
	}

	s.logger.Infow("Ticket updated", "ticket_id", id)
	return updated, nil
}

// DeleteTicket 删除工单
func (s *TicketServiceV2) DeleteTicket(ctx context.Context, id int, tenantID int) error {
	return s.repo.Delete(ctx, id, tenantID)
}

// ListTickets 列表查询工单
func (s *TicketServiceV2) ListTickets(ctx context.Context, req *dto.ListTicketsRequest, tenantID int) (*dto.ListTicketsResponse, error) {
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
	pagination := &QueryParams{
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
func (s *TicketServiceV2) AssignTicket(ctx context.Context, ticketID int, assigneeID int, tenantID int) (*ticket.Ticket, error) {
	s.logger.Infow("Assigning ticket", "ticket_id", ticketID, "assignee_id", assigneeID)

	updated, err := s.repo.AssignTicket(ctx, ticketID, assigneeID, tenantID)
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

	return updated, nil
}

// ResolveTicket 解决工单
func (s *TicketServiceV2) ResolveTicket(ctx context.Context, ticketID int, resolution string, tenantID int) (*ticket.Ticket, error) {
	s.logger.Infow("Resolving ticket", "ticket_id", ticketID)

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

	// 更新状态
	updated, err := s.repo.UpdateStatus(ctx, ticketID, ticket.StatusResolved, tenantID)
	if err != nil {
		return nil, err
	}

	s.logger.Infow("Ticket resolved", "ticket_id", ticketID)
	return updated, nil
}

// CloseTicket 关闭工单
func (s *TicketServiceV2) CloseTicket(ctx context.Context, ticketID int, tenantID int) (*ticket.Ticket, error) {
	s.logger.Infow("Closing ticket", "ticket_id", ticketID)

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

	// 更新状态
	updated, err := s.repo.UpdateStatus(ctx, ticketID, ticket.StatusClosed, tenantID)
	if err != nil {
		return nil, err
	}

	s.logger.Infow("Ticket closed", "ticket_id", ticketID)
	return updated, nil
}

// GetTicketStats 获取工单统计
func (s *TicketServiceV2) GetTicketStats(ctx context.Context, tenantID int) (*dto.TicketStatsResponse, error) {
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
		HighPriority: 0, // 需要单独查询
		Overdue:      len(overdue),
	}, nil
}

// ==================== 辅助方法 ====================

// toTicketResponse 转换为 DTO 响应
func (s *TicketServiceV2) toTicketResponse(t *ticket.Ticket) *dto.TicketResponse {
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
		resp.AssigneeID = t.AssigneeID
	}
	if t.CategoryID != nil {
		resp.CategoryID = t.CategoryID
	}
	if t.ResolvedAt != nil {
		resp.ResolvedAt = t.ResolvedAt
	}

	return resp
}

// toEntTicket 转换为 Ent 工单（临时方案，用于兼容现有通知服务）
func (s *TicketServiceV2) toEntTicket(t *ticket.Ticket) *ent.Ticket {
	// 注意：这是一个临时方案
	// 理想情况下，通知服务应该接受领域模型
	// 这里创建一个最小化的 Ent 工单对象用于传递必要信息
	return &ent.Ticket{
		ID:           t.ID,
		TicketNumber: t.TicketNumber,
		Title:        t.Title,
		Status:       string(t.Status),
		Priority:     string(t.Priority),
		RequesterID:  t.RequesterID,
		TenantID:     t.TenantID,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}

// QueryParams 查询参数（从 repository/base 引用）
type QueryParams struct {
	Page     int
	PageSize int
	OrderBy  string
	OrderDir string
}

// CalculateOffset 计算分页偏移量
func (p QueryParams) CalculateOffset() int {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	return (p.Page - 1) * p.PageSize
}

// GetLimit 获取分页大小
func (p QueryParams) GetLimit() int {
	if p.PageSize <= 0 {
		return 20
	}
	return p.PageSize
}
