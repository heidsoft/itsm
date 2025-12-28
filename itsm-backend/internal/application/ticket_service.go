// Package application contains application services that orchestrate domain logic
package application

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"itsm-backend/internal/domain/shared"
	"itsm-backend/internal/domain/ticket"
)

// TicketService is the application service for ticket operations
type TicketService struct {
	ticketRepo ticket.Repository
	domainSvc  *ticket.TicketDomainService
	eventBus   shared.EventBus
	logger     *zap.SugaredLogger
}

// NewTicketService creates a new ticket application service
func NewTicketService(
	ticketRepo ticket.Repository,
	domainSvc *ticket.TicketDomainService,
	eventBus shared.EventBus,
	logger *zap.SugaredLogger,
) *TicketService {
	return &TicketService{
		ticketRepo: ticketRepo,
		domainSvc:  domainSvc,
		eventBus:   eventBus,
		logger:     logger,
	}
}

// CreateTicketCommand represents the command to create a ticket
type CreateTicketCommand struct {
	Title       string          `json:"title" validate:"required,max=200"`
	Description string          `json:"description" validate:"max=2000"`
	Priority    string          `json:"priority" validate:"required,oneof=low normal high urgent critical"`
	Category    string          `json:"category" validate:"required"`
	CreatedBy   shared.UserID   `json:"created_by" validate:"required"`
	TenantID    shared.TenantID `json:"tenant_id" validate:"required"`
}

// CreateTicketResult represents the result of creating a ticket
type CreateTicketResult struct {
	ID           string `json:"id"`
	TicketNumber string `json:"ticket_number"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

// CreateTicket creates a new ticket
func (s *TicketService) CreateTicket(ctx context.Context, cmd CreateTicketCommand) (*CreateTicketResult, error) {
	s.logger.Infow("Creating ticket", "title", cmd.Title, "tenant", cmd.TenantID)

	// Generate unique ID
	ticketID := generateID()

	// Convert priority string to domain type
	priority, err := s.parsePriority(cmd.Priority)
	if err != nil {
		return nil, err
	}

	// Create domain command
	domainCmd := ticket.CreateTicketCommand{
		ID:          ticketID,
		Title:       cmd.Title,
		Description: cmd.Description,
		Priority:    priority,
		Category:    cmd.Category,
		CreatedBy:   cmd.CreatedBy,
		TenantID:    cmd.TenantID,
	}

	// Execute domain operation
	ticketAggregate, err := s.domainSvc.CreateTicket(domainCmd)
	if err != nil {
		s.logger.Errorw("Failed to create ticket", "error", err)
		return nil, err
	}

	// Return result
	return &CreateTicketResult{
		ID:           ticketAggregate.GetID(),
		TicketNumber: ticketAggregate.GetNumber(),
		Status:       string(ticketAggregate.GetStatus()),
		CreatedAt:    ticketAggregate.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
	}, nil
}

// AssignTicketCommand represents the command to assign a ticket
type AssignTicketCommand struct {
	TicketID     string          `json:"ticket_id" validate:"required"`
	AssignedTo   shared.UserID   `json:"assigned_to" validate:"required"`
	AssignedBy   shared.UserID   `json:"assigned_by" validate:"required"`
	TeamID       *shared.TeamID  `json:"team_id,omitempty"`
	Instructions string          `json:"instructions"`
	TenantID     shared.TenantID `json:"tenant_id" validate:"required"`
}

// AssignTicket assigns a ticket to a user
func (s *TicketService) AssignTicket(ctx context.Context, cmd AssignTicketCommand) error {
	s.logger.Infow("Assigning ticket", "ticket_id", cmd.TicketID, "assigned_to", cmd.AssignedTo)

	// Get ticket aggregate
	ticketAggregate, err := s.ticketRepo.GetByID(cmd.TicketID)
	if err != nil {
		return err
	}

	// Verify tenant access
	if ticketAggregate.GetTenantID() != cmd.TenantID {
		return errors.New("ticket not found in tenant")
	}

	// Execute domain operation
	err = ticketAggregate.AssignTo(cmd.AssignedTo, cmd.AssignedBy, cmd.TeamID, cmd.Instructions)
	if err != nil {
		return err
	}

	// Save aggregate
	if err := s.ticketRepo.Save(ticketAggregate); err != nil {
		return err
	}

	// Publish events
	for _, event := range ticketAggregate.GetEvents() {
		if err := s.eventBus.Publish(event); err != nil {
			s.logger.Warnw("Failed to publish event", "event", event.GetEventType(), "error", err)
		}
	}
	ticketAggregate.ClearEvents()

	return nil
}

// UpdateTicketStatusCommand represents the command to update ticket status
type UpdateTicketStatusCommand struct {
	TicketID  string          `json:"ticket_id" validate:"required"`
	NewStatus string          `json:"new_status" validate:"required"`
	Reason    string          `json:"reason"`
	ChangedBy shared.UserID   `json:"changed_by" validate:"required"`
	TenantID  shared.TenantID `json:"tenant_id" validate:"required"`
}

// UpdateTicketStatus updates the status of a ticket
func (s *TicketService) UpdateTicketStatus(ctx context.Context, cmd UpdateTicketStatusCommand) error {
	s.logger.Infow("Updating ticket status", "ticket_id", cmd.TicketID, "new_status", cmd.NewStatus)

	// Get ticket aggregate
	ticketAggregate, err := s.ticketRepo.GetByID(cmd.TicketID)
	if err != nil {
		return err
	}

	// Verify tenant access
	if ticketAggregate.GetTenantID() != cmd.TenantID {
		return errors.New("ticket not found in tenant")
	}

	// Convert status string to domain type
	newStatus := ticket.Status(cmd.NewStatus)
	if !newStatus.IsValid() {
		return errors.New("invalid status")
	}

	// Execute domain operation
	err = ticketAggregate.ChangeStatus(newStatus, cmd.ChangedBy, cmd.Reason)
	if err != nil {
		return err
	}

	// Save aggregate
	if err := s.ticketRepo.Save(ticketAggregate); err != nil {
		return err
	}

	// Publish events
	for _, event := range ticketAggregate.GetEvents() {
		if err := s.eventBus.Publish(event); err != nil {
			s.logger.Warnw("Failed to publish event", "event", event.GetEventType(), "error", err)
		}
	}
	ticketAggregate.ClearEvents()

	return nil
}

// AddCommentCommand represents the command to add a comment
type AddCommentCommand struct {
	TicketID  string          `json:"ticket_id" validate:"required"`
	Content   string          `json:"content" validate:"required,max=2000"`
	AuthorID  shared.UserID   `json:"author_id" validate:"required"`
	IsPrivate bool            `json:"is_private"`
	TenantID  shared.TenantID `json:"tenant_id" validate:"required"`
}

// AddComment adds a comment to a ticket
func (s *TicketService) AddComment(ctx context.Context, cmd AddCommentCommand) error {
	s.logger.Infow("Adding comment to ticket", "ticket_id", cmd.TicketID, "author", cmd.AuthorID)

	// Get ticket aggregate
	ticketAggregate, err := s.ticketRepo.GetByID(cmd.TicketID)
	if err != nil {
		return err
	}

	// Verify tenant access
	if ticketAggregate.GetTenantID() != cmd.TenantID {
		return errors.New("ticket not found in tenant")
	}

	// Generate comment ID
	commentID := generateID()

	// Execute domain operation
	err = ticketAggregate.AddComment(commentID, cmd.AuthorID, cmd.Content, cmd.IsPrivate)
	if err != nil {
		return err
	}

	// Save aggregate
	if err := s.ticketRepo.Save(ticketAggregate); err != nil {
		return err
	}

	// Publish events
	for _, event := range ticketAggregate.GetEvents() {
		if err := s.eventBus.Publish(event); err != nil {
			s.logger.Warnw("Failed to publish event", "event", event.GetEventType(), "error", err)
		}
	}
	ticketAggregate.ClearEvents()

	return nil
}

// GetTicketQuery represents a query for a single ticket
type GetTicketQuery struct {
	TicketID string          `json:"ticket_id" validate:"required"`
	TenantID shared.TenantID `json:"tenant_id" validate:"required"`
}

// TicketDetails represents detailed ticket information
type TicketDetails struct {
	ID           string                `json:"id"`
	TicketNumber string                `json:"ticket_number"`
	Title        string                `json:"title"`
	Description  string                `json:"description"`
	Status       string                `json:"status"`
	Priority     string                `json:"priority"`
	Category     string                `json:"category"`
	CreatedBy    string                `json:"created_by"`
	Assignment   *TicketAssignmentInfo `json:"assignment,omitempty"`
	Comments     []*CommentInfo        `json:"comments"`
	Attachments  []*AttachmentInfo     `json:"attachments"`
	CreatedAt    string                `json:"created_at"`
	UpdatedAt    string                `json:"updated_at"`
}

type TicketAssignmentInfo struct {
	AssignedTo   string `json:"assigned_to"`
	AssignedBy   string `json:"assigned_by"`
	AssignedAt   string `json:"assigned_at"`
	TeamID       string `json:"team_id,omitempty"`
	Instructions string `json:"instructions"`
}

type CommentInfo struct {
	ID        string `json:"id"`
	AuthorID  string `json:"author_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	IsPrivate bool   `json:"is_private"`
}

type AttachmentInfo struct {
	ID         string `json:"id"`
	Filename   string `json:"filename"`
	FileSize   int64  `json:"file_size"`
	MimeType   string `json:"mime_type"`
	URL        string `json:"url"`
	UploadedBy string `json:"uploaded_by"`
	UploadedAt string `json:"uploaded_at"`
}

// GetTicket retrieves a ticket by ID
func (s *TicketService) GetTicket(ctx context.Context, query GetTicketQuery) (*TicketDetails, error) {
	// Get ticket aggregate
	ticketAggregate, err := s.ticketRepo.GetByID(query.TicketID)
	if err != nil {
		return nil, err
	}

	// Verify tenant access
	if ticketAggregate.GetTenantID() != query.TenantID {
		return nil, errors.New("ticket not found")
	}

	// Map to DTO
	return s.mapTicketToDetails(ticketAggregate), nil
}

// SearchTicketsQuery represents a search query for tickets
type SearchTicketsQuery struct {
	TenantID   shared.TenantID `json:"tenant_id" validate:"required"`
	Status     []string        `json:"status,omitempty"`
	Priority   []string        `json:"priority,omitempty"`
	AssignedTo *string         `json:"assigned_to,omitempty"`
	CreatedBy  *string         `json:"created_by,omitempty"`
	Category   string          `json:"category,omitempty"`
	Keywords   string          `json:"keywords,omitempty"`
	DateFrom   *string         `json:"date_from,omitempty"`
	DateTo     *string         `json:"date_to,omitempty"`
	Page       int             `json:"page" validate:"min=1"`
	PageSize   int             `json:"page_size" validate:"min=1,max=100"`
	SortBy     string          `json:"sort_by,omitempty"`
	SortOrder  string          `json:"sort_order,omitempty"`
}

// SearchTicketsResult represents the result of a ticket search
type SearchTicketsResult struct {
	Tickets    []*TicketSummary `json:"tickets"`
	TotalCount int              `json:"total_count"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
}

// TicketSummary represents a summary view of a ticket
type TicketSummary struct {
	ID           string  `json:"id"`
	TicketNumber string  `json:"ticket_number"`
	Title        string  `json:"title"`
	Status       string  `json:"status"`
	Priority     string  `json:"priority"`
	Category     string  `json:"category"`
	AssignedTo   *string `json:"assigned_to,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// SearchTickets searches for tickets based on criteria
func (s *TicketService) SearchTickets(ctx context.Context, query SearchTicketsQuery) (*SearchTicketsResult, error) {
	// Build search criteria
	criteria := s.buildSearchCriteria(query)

	// Execute search
	tickets, totalCount, err := s.ticketRepo.Search(criteria)
	if err != nil {
		return nil, err
	}

	// Map to DTOs
	summaries := make([]*TicketSummary, len(tickets))
	for i, ticket := range tickets {
		summaries[i] = s.mapTicketToSummary(ticket)
	}

	return &SearchTicketsResult{
		Tickets:    summaries,
		TotalCount: totalCount,
		Page:       query.Page,
		PageSize:   query.PageSize,
	}, nil
}

// Helper methods
func (s *TicketService) parsePriority(priority string) (ticket.Priority, error) {
	switch priority {
	case "low":
		return ticket.PriorityLow, nil
	case "normal":
		return ticket.PriorityNormal, nil
	case "high":
		return ticket.PriorityHigh, nil
	case "urgent":
		return ticket.PriorityUrgent, nil
	case "critical":
		return ticket.PriorityCritical, nil
	default:
		return ticket.PriorityNormal, errors.New("invalid priority")
	}
}

func (s *TicketService) mapTicketToDetails(t *ticket.Ticket) *TicketDetails {
	details := &TicketDetails{
		ID:           t.GetID(),
		TicketNumber: t.GetNumber(),
		Title:        t.GetTitle(),
		Description:  t.GetDescription(),
		Status:       string(t.GetStatus()),
		Priority:     t.GetPriority().String(),
		Category:     "", // Get from aggregate
		CreatedBy:    t.GetCreatedBy().String(),
		Comments:     make([]*CommentInfo, 0),
		Attachments:  make([]*AttachmentInfo, 0),
		CreatedAt:    t.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    t.GetUpdatedAt().Format("2006-01-02T15:04:05Z"),
	}

	// Map assignment if exists
	if assignment := t.GetAssignment(); assignment != nil {
		details.Assignment = &TicketAssignmentInfo{
			AssignedTo:   assignment.AssignedTo.String(),
			AssignedBy:   assignment.AssignedBy.String(),
			AssignedAt:   assignment.AssignedAt.Format("2006-01-02T15:04:05Z"),
			Instructions: assignment.Instructions,
		}
		if assignment.TeamID != nil {
			details.Assignment.TeamID = assignment.TeamID.String()
		}
	}

	// Map comments
	for _, comment := range t.GetComments() {
		details.Comments = append(details.Comments, &CommentInfo{
			ID:        comment.ID,
			AuthorID:  comment.AuthorID.String(),
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt.Format("2006-01-02T15:04:05Z"),
			IsPrivate: comment.IsPrivate,
		})
	}

	// Map attachments
	for _, attachment := range t.GetAttachments() {
		details.Attachments = append(details.Attachments, &AttachmentInfo{
			ID:         attachment.ID,
			Filename:   attachment.Filename,
			FileSize:   attachment.FileSize,
			MimeType:   attachment.MimeType,
			URL:        attachment.URL,
			UploadedBy: attachment.UploadedBy.String(),
			UploadedAt: attachment.UploadedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return details
}

func (s *TicketService) mapTicketToSummary(t *ticket.Ticket) *TicketSummary {
	summary := &TicketSummary{
		ID:           t.GetID(),
		TicketNumber: t.GetNumber(),
		Title:        t.GetTitle(),
		Status:       string(t.GetStatus()),
		Priority:     t.GetPriority().String(),
		Category:     "", // Get from aggregate
		CreatedAt:    t.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    t.GetUpdatedAt().Format("2006-01-02T15:04:05Z"),
	}

	// Add assigned user if exists
	if assignment := t.GetAssignment(); assignment != nil {
		assignedTo := assignment.AssignedTo.String()
		summary.AssignedTo = &assignedTo
	}

	return summary
}

func (s *TicketService) buildSearchCriteria(query SearchTicketsQuery) ticket.SearchCriteria {
	criteria := ticket.SearchCriteria{
		TenantID:  query.TenantID,
		Page:      query.Page,
		PageSize:  query.PageSize,
		SortBy:    query.SortBy,
		SortOrder: query.SortOrder,
	}

	// Convert status strings to domain types
	if len(query.Status) > 0 {
		criteria.Status = make([]ticket.Status, len(query.Status))
		for i, status := range query.Status {
			criteria.Status[i] = ticket.Status(status)
		}
	}

	// Convert priority strings to domain types
	if len(query.Priority) > 0 {
		criteria.Priority = make([]ticket.Priority, len(query.Priority))
		for i, priority := range query.Priority {
			switch priority {
			case "low":
				criteria.Priority[i] = ticket.PriorityLow
			case "normal":
				criteria.Priority[i] = ticket.PriorityNormal
			case "high":
				criteria.Priority[i] = ticket.PriorityHigh
			case "urgent":
				criteria.Priority[i] = ticket.PriorityUrgent
			case "critical":
				criteria.Priority[i] = ticket.PriorityCritical
			}
		}
	}

	// Set other criteria
	if query.AssignedTo != nil {
		assignedTo := shared.UserID(*query.AssignedTo)
		criteria.AssignedTo = &assignedTo
	}

	if query.CreatedBy != nil {
		createdBy := shared.UserID(*query.CreatedBy)
		criteria.CreatedBy = &createdBy
	}

	criteria.Category = query.Category
	criteria.Keywords = query.Keywords

	// Parse date range
	if query.DateFrom != nil && query.DateTo != nil {
		// Parse dates and set criteria.DateRange
		// Implementation would parse the date strings
	}

	return criteria
}

// Helper function to generate IDs
func generateID() string {
	// Implementation would generate unique ID
	return "generated-id"
}
