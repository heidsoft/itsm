// Package ticket contains the ticket aggregate and related domain logic
package ticket

import (
	"errors"
	"time"

	"itsm-backend/internal/domain/shared"
)

// Priority represents ticket priority levels
type Priority int

const (
	PriorityLow Priority = iota + 1
	PriorityNormal
	PriorityHigh
	PriorityUrgent
	PriorityCritical
)

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityUrgent:
		return "urgent"
	case PriorityCritical:
		return "critical"
	default:
		return "normal"
	}
}

// Status represents ticket status
type Status string

const (
	StatusNew        Status = "new"
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusPending    Status = "pending"
	StatusResolved   Status = "resolved"
	StatusClosed     Status = "closed"
	StatusCancelled  Status = "cancelled"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusNew, StatusOpen, StatusInProgress, StatusPending,
		StatusResolved, StatusClosed, StatusCancelled:
		return true
	}
	return false
}

// TicketNumber is a value object representing a ticket number
type TicketNumber struct {
	value string
}

func NewTicketNumber(number string) (*TicketNumber, error) {
	if number == "" {
		return nil, errors.New("ticket number cannot be empty")
	}
	return &TicketNumber{value: number}, nil
}

func (tn *TicketNumber) Value() string {
	return tn.value
}

func (tn *TicketNumber) Equals(other shared.ValueObject) bool {
	if otherTN, ok := other.(*TicketNumber); ok {
		return tn.value == otherTN.value
	}
	return false
}

func (tn *TicketNumber) Validate() error {
	if tn.value == "" {
		return errors.New("ticket number cannot be empty")
	}
	return nil
}

// TicketAssignment represents ticket assignment information
type TicketAssignment struct {
	AssignedTo   shared.UserID
	AssignedBy   shared.UserID
	AssignedAt   time.Time
	TeamID       *shared.TeamID
	Instructions string
}

// Comment represents a ticket comment
type Comment struct {
	ID        string
	AuthorID  shared.UserID
	Content   string
	CreatedAt time.Time
	IsPrivate bool
}

// Attachment represents a ticket attachment
type Attachment struct {
	ID         string
	Filename   string
	FileSize   int64
	MimeType   string
	URL        string
	UploadedBy shared.UserID
	UploadedAt time.Time
}

// TicketCreatedEvent represents the event when a ticket is created
type TicketCreatedEvent struct {
	BaseEvent
	TicketID  string
	Title     string
	Priority  Priority
	CreatedBy shared.UserID
	TenantID  shared.TenantID
}

func (e *TicketCreatedEvent) GetEventType() string {
	return "ticket.created"
}

// TicketAssignedEvent represents the event when a ticket is assigned
type TicketAssignedEvent struct {
	BaseEvent
	TicketID   string
	AssignedTo shared.UserID
	AssignedBy shared.UserID
	TeamID     *shared.TeamID
}

func (e *TicketAssignedEvent) GetEventType() string {
	return "ticket.assigned"
}

// TicketStatusChangedEvent represents the event when ticket status changes
type TicketStatusChangedEvent struct {
	BaseEvent
	TicketID  string
	OldStatus Status
	NewStatus Status
	ChangedBy shared.UserID
	Reason    string
}

func (e *TicketStatusChangedEvent) GetEventType() string {
	return "ticket.status_changed"
}

// TicketCommentAddedEvent represents the event when a comment is added
type TicketCommentAddedEvent struct {
	BaseEvent
	TicketID  string
	CommentID string
	AuthorID  shared.UserID
	Content   string
	IsPrivate bool
}

func (e *TicketCommentAddedEvent) GetEventType() string {
	return "ticket.comment_added"
}

// TicketPriorityUpdatedEvent represents priority change events
type TicketPriorityUpdatedEvent struct {
	BaseEvent
	TicketID   string
	Priority   Priority
	UpdatedBy  shared.UserID
	OccurredAt time.Time
}

func (e *TicketPriorityUpdatedEvent) GetEventType() string {
	return "ticket.priority_updated"
}

// BaseEvent provides common event fields
type BaseEvent struct {
	EventID     string
	AggregateID string
	Timestamp   time.Time
}

func (e *BaseEvent) GetEventID() string {
	return e.EventID
}

func (e *BaseEvent) GetAggregateID() string {
	return e.AggregateID
}

func (e *BaseEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

func (e *BaseEvent) GetEventData() interface{} {
	return e
}

// Ticket aggregate root
type Ticket struct {
	id      string
	version int
	events  []shared.DomainEvent

	// Core fields
	number      *TicketNumber
	title       string
	description string
	priority    Priority
	status      Status
	category    string

	// Relationships
	createdBy  shared.UserID
	assignment *TicketAssignment
	tenantID   shared.TenantID

	// Metadata
	createdAt time.Time
	updatedAt time.Time

	// Collections
	comments    []*Comment
	attachments []*Attachment
	tags        []string
}

// NewTicket creates a new ticket aggregate
func NewTicket(
	id string,
	number *TicketNumber,
	title, description string,
	priority Priority,
	category string,
	createdBy shared.UserID,
	tenantID shared.TenantID,
) (*Ticket, error) {
	if id == "" {
		return nil, errors.New("ticket ID cannot be empty")
	}
	if title == "" {
		return nil, errors.New("ticket title cannot be empty")
	}
	if number == nil {
		return nil, errors.New("ticket number cannot be nil")
	}

	ticket := &Ticket{
		id:          id,
		version:     1,
		events:      make([]shared.DomainEvent, 0),
		number:      number,
		title:       title,
		description: description,
		priority:    priority,
		status:      StatusNew,
		category:    category,
		createdBy:   createdBy,
		tenantID:    tenantID,
		createdAt:   time.Now(),
		updatedAt:   time.Now(),
		comments:    make([]*Comment, 0),
		attachments: make([]*Attachment, 0),
		tags:        make([]string, 0),
	}

	// Apply domain event
	event := &TicketCreatedEvent{
		BaseEvent: BaseEvent{
			EventID:     generateEventID(),
			AggregateID: id,
			Timestamp:   time.Now(),
		},
		TicketID:  id,
		Title:     title,
		Priority:  priority,
		CreatedBy: createdBy,
		TenantID:  tenantID,
	}

	ticket.ApplyEvent(event)
	return ticket, nil
}

// Domain methods
func (t *Ticket) GetID() string {
	return t.id
}

func (t *Ticket) GetVersion() int {
	return t.version
}

func (t *Ticket) GetEvents() []shared.DomainEvent {
	return t.events
}

func (t *Ticket) ClearEvents() {
	t.events = make([]shared.DomainEvent, 0)
}

func (t *Ticket) ApplyEvent(event shared.DomainEvent) {
	t.events = append(t.events, event)
	t.version++
	t.updatedAt = time.Now()
}

// Business methods
func (t *Ticket) AssignTo(assignedTo shared.UserID, assignedBy shared.UserID, teamID *shared.TeamID, instructions string) error {
	if t.status == StatusClosed || t.status == StatusCancelled {
		return errors.New("cannot assign closed or cancelled ticket")
	}

	t.assignment = &TicketAssignment{
		AssignedTo:   assignedTo,
		AssignedBy:   assignedBy,
		AssignedAt:   time.Now(),
		TeamID:       teamID,
		Instructions: instructions,
	}

	if t.status == StatusNew {
		t.status = StatusOpen
	}

	event := &TicketAssignedEvent{
		BaseEvent: BaseEvent{
			EventID:     generateEventID(),
			AggregateID: t.id,
			Timestamp:   time.Now(),
		},
		TicketID:   t.id,
		AssignedTo: assignedTo,
		AssignedBy: assignedBy,
		TeamID:     teamID,
	}

	t.ApplyEvent(event)
	return nil
}

func (t *Ticket) ChangeStatus(newStatus Status, changedBy shared.UserID, reason string) error {
	if !newStatus.IsValid() {
		return errors.New("invalid status")
	}

	oldStatus := t.status

	// Business rules for status transitions
	if !t.isValidStatusTransition(oldStatus, newStatus) {
		return errors.New("invalid status transition")
	}

	t.status = newStatus

	event := &TicketStatusChangedEvent{
		BaseEvent: BaseEvent{
			EventID:     generateEventID(),
			AggregateID: t.id,
			Timestamp:   time.Now(),
		},
		TicketID:  t.id,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		ChangedBy: changedBy,
		Reason:    reason,
	}

	t.ApplyEvent(event)
	return nil
}

func (t *Ticket) AddComment(commentID string, authorID shared.UserID, content string, isPrivate bool) error {
	if content == "" {
		return errors.New("comment content cannot be empty")
	}

	comment := &Comment{
		ID:        commentID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: time.Now(),
		IsPrivate: isPrivate,
	}

	t.comments = append(t.comments, comment)

	event := &TicketCommentAddedEvent{
		BaseEvent: BaseEvent{
			EventID:     generateEventID(),
			AggregateID: t.id,
			Timestamp:   time.Now(),
		},
		TicketID:  t.id,
		CommentID: commentID,
		AuthorID:  authorID,
		Content:   content,
		IsPrivate: isPrivate,
	}

	t.ApplyEvent(event)
	return nil
}

func (t *Ticket) UpdatePriority(newPriority Priority, updatedBy shared.UserID) error {
	if t.status == StatusClosed || t.status == StatusCancelled {
		return errors.New("cannot update priority of closed or cancelled ticket")
	}

	t.priority = newPriority
	t.ApplyEvent(&TicketPriorityUpdatedEvent{
		BaseEvent: BaseEvent{
			EventID:     generateEventID(),
			AggregateID: t.id,
			Timestamp:   time.Now(),
		},
		TicketID:   t.id,
		Priority:   newPriority,
		UpdatedBy:  updatedBy,
		OccurredAt: time.Now(),
	})

	return nil
}

// Helper methods
func (t *Ticket) isValidStatusTransition(from, to Status) bool {
	validTransitions := map[Status][]Status{
		StatusNew:        {StatusOpen, StatusCancelled},
		StatusOpen:       {StatusInProgress, StatusPending, StatusResolved, StatusCancelled},
		StatusInProgress: {StatusPending, StatusResolved, StatusOpen, StatusCancelled},
		StatusPending:    {StatusInProgress, StatusOpen, StatusResolved, StatusCancelled},
		StatusResolved:   {StatusClosed, StatusOpen},
		StatusClosed:     {},
		StatusCancelled:  {},
	}

	allowedStatuses, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, status := range allowedStatuses {
		if status == to {
			return true
		}
	}
	return false
}

// Getters for external access
func (t *Ticket) GetNumber() string {
	if t.number == nil {
		return ""
	}
	return t.number.Value()
}

func (t *Ticket) GetTitle() string {
	return t.title
}

func (t *Ticket) GetDescription() string {
	return t.description
}

func (t *Ticket) GetStatus() Status {
	return t.status
}

func (t *Ticket) GetPriority() Priority {
	return t.priority
}

func (t *Ticket) GetAssignment() *TicketAssignment {
	return t.assignment
}

func (t *Ticket) GetComments() []*Comment {
	return t.comments
}

func (t *Ticket) GetAttachments() []*Attachment {
	return t.attachments
}

func (t *Ticket) GetCreatedAt() time.Time {
	return t.createdAt
}

func (t *Ticket) GetUpdatedAt() time.Time {
	return t.updatedAt
}

func (t *Ticket) GetCreatedBy() shared.UserID {
	return t.createdBy
}

func (t *Ticket) GetTenantID() shared.TenantID {
	return t.tenantID
}

// Repository interface for tickets
type Repository interface {
	shared.Repository[*Ticket]
	GetByNumber(number string, tenantID shared.TenantID) (*Ticket, error)
	GetByAssignee(assigneeID shared.UserID) ([]*Ticket, error)
	GetByStatus(status Status, tenantID shared.TenantID) ([]*Ticket, error)
	Search(criteria SearchCriteria) ([]*Ticket, int, error)
}

// SearchCriteria for ticket queries
type SearchCriteria struct {
	TenantID   shared.TenantID
	Status     []Status
	Priority   []Priority
	AssignedTo *shared.UserID
	CreatedBy  *shared.UserID
	Category   string
	Keywords   string
	DateRange  *shared.DateRange
	Page       int
	PageSize   int
	SortBy     string
	SortOrder  string
}

// Domain service for ticket operations
type TicketDomainService struct {
	repository Repository
	eventBus   shared.EventBus
}

func NewTicketDomainService(repo Repository, eventBus shared.EventBus) *TicketDomainService {
	return &TicketDomainService{
		repository: repo,
		eventBus:   eventBus,
	}
}

func (s *TicketDomainService) GetServiceName() string {
	return "TicketDomainService"
}

func (s *TicketDomainService) CreateTicket(cmd CreateTicketCommand) (*Ticket, error) {
	// Business validation
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	// Generate ticket number
	number, err := s.generateTicketNumber(cmd.TenantID)
	if err != nil {
		return nil, err
	}

	// Create aggregate
	ticket, err := NewTicket(
		cmd.ID,
		number,
		cmd.Title,
		cmd.Description,
		cmd.Priority,
		cmd.Category,
		cmd.CreatedBy,
		cmd.TenantID,
	)
	if err != nil {
		return nil, err
	}

	// Auto-assign if rules exist
	if assignee := s.findAutoAssignee(cmd); assignee != nil {
		err = ticket.AssignTo(*assignee, shared.SystemUserID, nil, "Auto-assigned by rules")
		if err != nil {
			return nil, err
		}
	}

	// Save to repository
	if err := s.repository.Save(ticket); err != nil {
		return nil, err
	}

	// Publish domain events
	for _, event := range ticket.GetEvents() {
		if err := s.eventBus.Publish(event); err != nil {
			// Log error but don't fail the operation
		}
	}
	ticket.ClearEvents()

	return ticket, nil
}

func (s *TicketDomainService) generateTicketNumber(tenantID shared.TenantID) (*TicketNumber, error) {
	// Implementation would generate unique ticket number
	// For now, return a simple implementation
	timestamp := time.Now().Format("20060102150405")
	number := "TK-" + timestamp
	return NewTicketNumber(number)
}

func (s *TicketDomainService) findAutoAssignee(cmd CreateTicketCommand) *shared.UserID {
	// Implementation would check assignment rules
	return nil
}

// Commands
type CreateTicketCommand struct {
	ID          string
	Title       string
	Description string
	Priority    Priority
	Category    string
	CreatedBy   shared.UserID
	TenantID    shared.TenantID
}

func (cmd *CreateTicketCommand) Validate() error {
	if cmd.ID == "" {
		return errors.New("ticket ID is required")
	}
	if cmd.Title == "" {
		return errors.New("ticket title is required")
	}
	if cmd.CreatedBy == "" {
		return errors.New("created by is required")
	}
	if cmd.TenantID == "" {
		return errors.New("tenant ID is required")
	}
	return nil
}

// Helper function to generate event IDs
func generateEventID() string {
	// Implementation would generate unique event ID
	return time.Now().Format("20060102150405") + "-" + "event"
}
