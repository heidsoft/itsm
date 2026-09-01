package ticket

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/handlers/common/datascope"

	"go.uber.org/zap"
)

// Service handles ticket business logic.
// All operations go through the Repository interface (tenant-isolated).
type Service struct {
	repo   Repository
	logger *zap.SugaredLogger
}

// NewService creates a new ticket service.
func NewService(repo Repository, logger *zap.SugaredLogger) *Service {
	return &Service{repo: repo, logger: logger}
}

// Create creates a new ticket.
func (s *Service) Create(ctx context.Context, tenantID int, params *CreateParams) (*Ticket, error) {
	s.logger.Infow("Creating ticket", "title", params.Title, "tenant_id", tenantID)

	if params.RequesterID == 0 {
		return nil, fmt.Errorf("requester_id is required")
	}
	if params.Priority == "" {
		params.Priority = "medium"
	}
	if params.Type == "" {
		params.Type = "incident"
	}

	created, err := s.repo.Create(ctx, params, tenantID)
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}

	return created, nil
}

// Get retrieves a ticket by ID.
func (s *Service) Get(ctx context.Context, id int, tenantID int) (*Ticket, error) {
	return s.repo.GetByID(ctx, id, tenantID)
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
func (s *Service) Update(ctx context.Context, tenantID int, id int, params *UpdateParams) (*Ticket, error) {
	// Fetch current ticket to get version for optimistic locking
	current, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
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
func (s *Service) Delete(ctx context.Context, id int, tenantID int) error {
	return s.repo.Delete(ctx, id, tenantID)
}

// BatchDelete deletes multiple tickets.
func (s *Service) BatchDelete(ctx context.Context, ids []int, tenantID int) error {
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
func (s *Service) UpdateSubtask(ctx context.Context, tenantID int, subtaskID int, params *UpdateParams) (*Ticket, error) {
	// Verify subtask belongs to parent
	current, err := s.repo.GetByID(ctx, subtaskID, tenantID)
	if err != nil {
		return nil, err
	}
	if current.ParentTicketID == nil || *current.ParentTicketID != tenantID {
		return nil, fmt.Errorf("subtask does not belong to parent")
	}
	return s.repo.Update(ctx, subtaskID, params, tenantID)
}

// DeleteSubtask deletes a child ticket.
func (s *Service) DeleteSubtask(ctx context.Context, subtaskID int, tenantID int) error {
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
