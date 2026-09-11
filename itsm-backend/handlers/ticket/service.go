package ticket

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/handlers/common/datascope"
	"itsm-backend/service"

	"go.uber.org/zap"
)

// Service handles ticket business logic.
// All operations go through the Repository interface (tenant-isolated).
type Service struct {
	repo            Repository
	productionSvc   *service.TicketService
	logger          *zap.SugaredLogger
}

// NewService creates a new ticket service.
func NewService(repo Repository, productionSvc *service.TicketService, logger *zap.SugaredLogger) *Service {
	return &Service{repo: repo, productionSvc: productionSvc, logger: logger}
}

// Create creates a new ticket.
// Delegates to productionSvc.CreateTicket which handles workflow outbox,
// SLA, notifications, automation rules, and other side effects.
func (s *Service) Create(ctx context.Context, tenantID int, params *CreateParams) (*Ticket, error) {
	s.logger.Infow("Creating ticket", "title", params.Title, "tenant_id", tenantID)

	if params.RequesterID == 0 {
		return nil, common.NewBadRequestError("requester_id is required", nil)
	}
	if params.Priority == "" {
		params.Priority = "medium"
	}
	if params.Type == "" {
		params.Type = "incident"
	}

	// Build the DTO for the production service which owns workflow/SLA/outbox logic.
	req := &dto.CreateTicketRequest{
		Title:                 params.Title,
		Description:           params.Description,
		Type:                  params.Type,
		Priority:              params.Priority,
		RequesterID:           params.RequesterID,
		FormFields:            params.FormFields,
		TagIDs:                params.TagIDs,
		WorkflowDefinitionKey: params.WorkflowDefinitionKey,
	}
	if params.AssigneeID != nil {
		req.AssigneeID = *params.AssigneeID
	}
	if params.TicketTypeID != nil {
		req.TicketTypeID = params.TicketTypeID
	}
	if params.CategoryID != nil {
		req.CategoryID = params.CategoryID
	}
	if params.TemplateID != nil {
		req.TemplateID = params.TemplateID
	}
	if params.ParentTicketID != nil {
		req.ParentTicketID = params.ParentTicketID
	}

	// productionSvc 为 nil 时（单元测试 harness 场景）降级为纯 repo 创建，
	// 跳过工作流/SLA/通知等生产侧副作用——生产装配恒传非 nil。
	if s.productionSvc == nil {
		return s.repo.Create(ctx, params, tenantID)
	}

	created, err := s.productionSvc.CreateTicket(ctx, req, tenantID)
	if err != nil {
		return nil, err
	}

	// Convert domain ticket back to handler-layer ticket for response mapping.
	return &Ticket{
		ID:             created.ID,
		TicketNumber:   created.TicketNumber,
		Title:          created.Title,
		Description:    created.Description,
		Status:         string(created.Status),
		Priority:       string(created.Priority),
		Type:           string(created.Type),
		TicketTypeCode: created.TicketTypeCode,
		TicketTypeName: created.TicketTypeName,
		FormFields:     created.FormFields,
		RequesterID:    created.RequesterID,
		AssigneeID:     created.AssigneeID,
		TenantID:       created.TenantID,
		Version:        created.Version,
		CreatedAt:      created.CreatedAt,
		UpdatedAt:      created.UpdatedAt,
	}, nil
}

// Get retrieves a ticket by ID.
func (s *Service) Get(ctx context.Context, id int, tenantID int) (*Ticket, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// GetByNumber retrieves a ticket by its business ticket number (e.g. TKT-202609-000010).
// 用于让 GET /api/v1/tickets/:id 同时支持数字 ID 与业务工单号。
func (s *Service) GetByNumber(ctx context.Context, ticketNumber string, tenantID int) (*Ticket, error) {
	return s.repo.GetByNumber(ctx, ticketNumber, tenantID)
}

// List lists tickets with pagination and filtering.
func (s *Service) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, currentUserID int, currentRole string) ([]*Ticket, int, error) {
	dataScope := datascope.DataScopeAll
	if !IsDataScopeAllRole(currentRole) {
		dataScope = datascope.DataScopeOwnedOrAssigned
	}
	return s.repo.List(ctx, tenantID, page, size, filters, dataScope, currentUserID)
}

// Update updates a ticket.
// P1-DataScope：写路径行级校验——写权限 ⊆ 读权限，普通角色仅可修改
// 本人创建或受理的工单（datascope.CanWriteResource）。
func (s *Service) Update(ctx context.Context, tenantID int, id int, params *UpdateParams, actorID int, actorRole string) (*Ticket, error) {
	// Fetch current ticket to get version for optimistic locking
	current, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if !datascope.CanWriteResource(actorID, actorRole, current.RequesterID, current.AssigneeID) {
		return nil, common.NewForbiddenError("无权限修改该工单：仅创建人、受理人或管理员可操作")
	}
	if params.Version == 0 {
		params.Version = current.Version
	}

	updated, err := s.repo.Update(ctx, id, params, tenantID)
	if err != nil {
		return nil, fmt.Errorf("update ticket: %w", err)
	}
	return updated, nil
}

// Delete soft-deletes a ticket.
// P1-DataScope：删除同样受行级写权限约束（原先仅校验租户隔离）。
func (s *Service) Delete(ctx context.Context, id int, tenantID int, actorID int, actorRole string) error {
	current, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if !datascope.CanWriteResource(actorID, actorRole, current.RequesterID, current.AssigneeID) {
		return common.NewForbiddenError("无权限删除该工单：仅创建人、受理人或管理员可操作")
	}
	return s.repo.Delete(ctx, id, tenantID)
}

// BatchDelete deletes multiple tickets.
// P1-DataScope：逐单行级校验，任一单据无权限则整体拒绝（全有或全无），
// 避免批量接口被用来越权删除他人工单。
func (s *Service) BatchDelete(ctx context.Context, ids []int, tenantID int, actorID int, actorRole string) error {
	if !datascope.IsDataScopeAllRole(actorRole) {
		for _, id := range ids {
			current, err := s.repo.GetByID(ctx, id, tenantID)
			if err != nil {
				return err
			}
			if !datascope.CanWriteResource(actorID, actorRole, current.RequesterID, current.AssigneeID) {
				return common.NewForbiddenError(fmt.Sprintf("无权限删除工单 %d：仅创建人、受理人或管理员可操作", id))
			}
		}
	}
	return s.repo.BatchDelete(ctx, ids, tenantID)
}

// AssignTicket assigns a ticket to a user.
func (s *Service) AssignTicket(ctx context.Context, ticketID int, assigneeID int, tenantID int) (*Ticket, error) {
	return s.repo.AssignTicket(ctx, ticketID, assigneeID, tenantID)
}

// EscalateTicket escalates a ticket.
func (s *Service) EscalateTicket(ctx context.Context, ticketID int, reason string, tenantID int, escalatedBy int) (*Ticket, error) {
	return s.repo.EscalateTicket(ctx, ticketID, reason, tenantID, escalatedBy)
}

// ResolveTicket resolves a ticket with a resolution note.
func (s *Service) ResolveTicket(ctx context.Context, ticketID int, resolution string, tenantID int) (*Ticket, error) {
	return s.repo.ResolveTicket(ctx, ticketID, resolution, tenantID)
}

// CloseTicket closes a ticket.
func (s *Service) CloseTicket(ctx context.Context, ticketID int, tenantID int) (*Ticket, error) {
	return s.repo.CloseTicket(ctx, ticketID, tenantID)
}

// UpdateStatus updates the status of a ticket.
func (s *Service) UpdateStatus(ctx context.Context, ticketID int, status string, tenantID int, userID int) (*Ticket, error) {
	return s.repo.UpdateStatus(ctx, ticketID, status, tenantID)
}

// Search searches tickets by keyword.
func (s *Service) Search(ctx context.Context, keyword string, tenantID int) ([]*Ticket, error) {
	return s.repo.Search(ctx, keyword, tenantID)
}

// GetOverdueTickets returns overdue tickets.
func (s *Service) GetOverdueTickets(ctx context.Context, tenantID int) ([]*Ticket, error) {
	return s.repo.FindOverdue(ctx, tenantID)
}

// GetTicketsByAssignee returns tickets assigned to a specific user.
func (s *Service) GetTicketsByAssignee(ctx context.Context, assigneeID int, tenantID int) ([]*Ticket, error) {
	return s.repo.FindByAssignee(ctx, assigneeID, tenantID)
}

// GetStats returns ticket statistics.
func (s *Service) GetStats(ctx context.Context, tenantID int) (*TicketStats, error) {
	return s.repo.GetStats(ctx, tenantID)
}

// ExportTickets exports tickets (placeholder — delegates to existing service layer).
func (s *Service) ExportTickets(ctx context.Context, tenantID int, filters map[string]interface{}, format string) ([]byte, error) {
	return nil, fmt.Errorf("export not implemented in handlers layer")
}

// ImportTickets imports tickets (placeholder).
func (s *Service) ImportTickets(ctx context.Context, tenantID int, data []byte, format string) error {
	return fmt.Errorf("import not implemented in handlers layer")
}

// AssignTickets assigns multiple tickets to a user.
func (s *Service) AssignTickets(ctx context.Context, tenantID int, ticketIDs []int, assigneeID int) error {
	for _, id := range ticketIDs {
		_, err := s.repo.AssignTicket(ctx, id, assigneeID, tenantID)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetTicketAnalytics returns analytics data.
func (s *Service) GetTicketAnalytics(ctx context.Context, tenantID int, dateFrom, dateTo string) (map[string]interface{}, error) {
	stats, err := s.repo.GetStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"stats":     stats,
		"date_from": dateFrom,
		"date_to":   dateTo,
	}, nil
}

// GetTicketTemplates returns all ticket templates.
func (s *Service) GetTicketTemplates(ctx context.Context, tenantID int) ([]*TicketTemplate, error) {
	return s.repo.ListTemplates(ctx, tenantID)
}

// GetTicketTemplate returns a single template.
func (s *Service) GetTicketTemplate(ctx context.Context, tenantID int, templateID int) (*TicketTemplate, error) {
	return s.repo.GetTemplate(ctx, templateID, tenantID)
}

// CreateTicketTemplate creates a new template.
func (s *Service) CreateTicketTemplate(ctx context.Context, tenantID int, tmpl *TicketTemplate) (*TicketTemplate, error) {
	return s.repo.CreateTemplate(ctx, tmpl, tenantID)
}

// UpdateTicketTemplate updates a template.
func (s *Service) UpdateTicketTemplate(ctx context.Context, tenantID int, templateID int, tmpl *TicketTemplate) (*TicketTemplate, error) {
	return s.repo.UpdateTemplate(ctx, templateID, tmpl, tenantID)
}

// DeleteTicketTemplate deletes a template.
func (s *Service) DeleteTicketTemplate(ctx context.Context, tenantID int, templateID int) error {
	return s.repo.DeleteTemplate(ctx, templateID, tenantID)
}

// UpdateTicketTemplateStatus enables or disables a template.
func (s *Service) UpdateTicketTemplateStatus(ctx context.Context, tenantID int, templateID int, isActive bool) (*TicketTemplate, error) {
	return s.repo.UpdateTemplateStatus(ctx, templateID, isActive, tenantID)
}

// CopyTicketTemplate copies a template with a new name.
func (s *Service) CopyTicketTemplate(ctx context.Context, tenantID int, templateID int, newName string) (*TicketTemplate, error) {
	return s.repo.CopyTemplate(ctx, templateID, newName, tenantID)
}

// GetTicketTemplateCategories returns template categories.
func (s *Service) GetTicketTemplateCategories(ctx context.Context, tenantID int) ([]string, error) {
	return s.repo.GetTemplateCategories(ctx, tenantID)
}

// GetSubtasks returns child tickets of a parent ticket.
func (s *Service) GetSubtasks(ctx context.Context, parentID int, tenantID int, currentUserID int, currentRole string) ([]*Ticket, int, error) {
	filters := map[string]interface{}{
		"parent_ticket_id": parentID,
	}
	return s.repo.List(ctx, tenantID, 1, 100, filters, datascope.DataScopeOwnedOrAssigned, currentUserID)
}

// CreateSubtask creates a child ticket.
func (s *Service) CreateSubtask(ctx context.Context, tenantID int, parentID int, params *CreateParams) (*Ticket, error) {
	params.ParentTicketID = &parentID
	return s.repo.Create(ctx, params, tenantID)
}

// UpdateSubtask updates a child ticket.
// P1-DataScope：子任务写路径同样受行级写权限约束。
func (s *Service) UpdateSubtask(ctx context.Context, tenantID int, subtaskID int, params *UpdateParams, actorID int, actorRole string) (*Ticket, error) {
	// Verify subtask belongs to parent
	current, err := s.repo.GetByID(ctx, subtaskID, tenantID)
	if err != nil {
		return nil, err
	}
	if !datascope.CanWriteResource(actorID, actorRole, current.RequesterID, current.AssigneeID) {
		return nil, common.NewForbiddenError("无权限修改该子任务：仅创建人、受理人或管理员可操作")
	}
	// 修复：原代码比较 *current.ParentTicketID != tenantID —— ParentTicketID 是父工单 ID，
	// 恒不等于 tenantID，导致所有子任务更新都被误判为"不属于父工单"。正确语义是：
	// 仅当该工单不是子任务（无父工单）时拒绝。
	if current.ParentTicketID == nil {
		return nil, common.NewBadRequestError("subtask does not belong to parent", nil)
	}
	return s.repo.Update(ctx, subtaskID, params, tenantID)
}

// DeleteSubtask deletes a child ticket.
// P1-DataScope：子任务删除同样受行级写权限约束。
func (s *Service) DeleteSubtask(ctx context.Context, subtaskID int, tenantID int, actorID int, actorRole string) error {
	current, err := s.repo.GetByID(ctx, subtaskID, tenantID)
	if err != nil {
		return err
	}
	if !datascope.CanWriteResource(actorID, actorRole, current.RequesterID, current.AssigneeID) {
		return common.NewForbiddenError("无权限删除该子任务：仅创建人、受理人或管理员可操作")
	}
	return s.repo.Delete(ctx, subtaskID, tenantID)
}

// GetTicketSLAInfo returns SLA info for a ticket.
func (s *Service) GetTicketSLAInfo(ctx context.Context, ticketID int, tenantID int) (map[string]interface{}, error) {
	t, err := s.repo.GetByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"ticket_id":                     t.ID,
		"sla_definition_id":            t.SLADefinitionID,
		"sla_response_deadline":        t.SLAResponseDeadline,
		"sla_resolution_deadline":      t.SLAResolutionDeadline,
		"first_response_at":            t.FirstResponseAt,
		"status":                       t.Status,
		"resolution_deadline_breached": t.SLAResolutionDeadline != nil && time.Now().After(*t.SLAResolutionDeadline),
	}, nil
}

// IsDataScopeAllRole returns true if the role has access to all tickets.
func IsDataScopeAllRole(role string) bool {
	switch role {
	case "admin", "super_admin", "manager":
		return true
	default:
		return false
	}
}

// GetTicketActivity retrieves ticket activity log (merged comments, attachments, status changes)
func (s *Service) GetTicketActivity(ctx context.Context, ticketID, tenantID int) ([]map[string]interface{}, error) {
	return s.productionSvc.GetTicketActivity(ctx, ticketID, tenantID)
}
