// Package persistence provides infrastructure for data persistence.
// This implementation is an in-memory placeholder to keep the DDD layer compiling
// while Ent-based mappings are still under construction.
package persistence

import (
	"errors"
	"strings"

	"go.uber.org/zap"
	"itsm-backend/internal/domain/shared"
	ticketdomain "itsm-backend/internal/domain/ticket"
)

// EntTicketRepository implements ticket.Repository using an in-memory store.
type EntTicketRepository struct {
	store  map[string]*ticketdomain.Ticket
	logger *zap.SugaredLogger
}

// NewEntTicketRepository creates a new in-memory repository instance.
// The client argument is unused to maintain compatibility with previous callers.
func NewEntTicketRepository(_ interface{}, logger *zap.SugaredLogger) *EntTicketRepository {
	return &EntTicketRepository{
		store:  make(map[string]*ticketdomain.Ticket),
		logger: logger,
	}
}

// Save stores or updates a ticket aggregate.
func (r *EntTicketRepository) Save(aggregate *ticketdomain.Ticket) error {
	if aggregate == nil {
		return errors.New("aggregate is nil")
	}
	r.store[aggregate.GetID()] = aggregate
	return nil
}

// GetByID retrieves a ticket by ID.
func (r *EntTicketRepository) GetByID(id string) (*ticketdomain.Ticket, error) {
	if t, ok := r.store[id]; ok {
		return t, nil
	}
	return nil, errors.New("ticket not found")
}

// GetByNumber retrieves a ticket by ticket number scoped to tenant.
func (r *EntTicketRepository) GetByNumber(number string, tenantID shared.TenantID) (*ticketdomain.Ticket, error) {
	for _, t := range r.store {
		if t.GetNumber() == number && t.GetTenantID() == tenantID {
			return t, nil
		}
	}
	return nil, errors.New("ticket not found")
}

// GetByAssignee retrieves tickets assigned to a specific user.
func (r *EntTicketRepository) GetByAssignee(assigneeID shared.UserID) ([]*ticketdomain.Ticket, error) {
	results := make([]*ticketdomain.Ticket, 0)
	for _, t := range r.store {
		if assignment := t.GetAssignment(); assignment != nil && assignment.AssignedTo == assigneeID {
			results = append(results, t)
		}
	}
	return results, nil
}

// GetByStatus retrieves tickets by status within a tenant.
func (r *EntTicketRepository) GetByStatus(status ticketdomain.Status, tenantID shared.TenantID) ([]*ticketdomain.Ticket, error) {
	results := make([]*ticketdomain.Ticket, 0)
	for _, t := range r.store {
		if t.GetStatus() == status && t.GetTenantID() == tenantID {
			results = append(results, t)
		}
	}
	return results, nil
}

// Search performs an in-memory search based on the given criteria.
func (r *EntTicketRepository) Search(criteria ticketdomain.SearchCriteria) ([]*ticketdomain.Ticket, int, error) {
	filtered := make([]*ticketdomain.Ticket, 0)
	for _, t := range r.store {
		if t.GetTenantID() != criteria.TenantID {
			continue
		}

		if len(criteria.Status) > 0 {
			matched := false
			for _, s := range criteria.Status {
				if t.GetStatus() == s {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		if len(criteria.Priority) > 0 {
			matched := false
			for _, p := range criteria.Priority {
				if t.GetPriority() == p {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		if criteria.AssignedTo != nil {
			assignment := t.GetAssignment()
			if assignment == nil || assignment.AssignedTo != *criteria.AssignedTo {
				continue
			}
		}

		if criteria.CreatedBy != nil && t.GetCreatedBy() != *criteria.CreatedBy {
			continue
		}

		if criteria.Keywords != "" {
			title := strings.ToLower(t.GetTitle())
			desc := strings.ToLower(t.GetDescription())
			kw := strings.ToLower(criteria.Keywords)
			if !strings.Contains(title, kw) && !strings.Contains(desc, kw) {
				continue
			}
		}

		if criteria.DateRange != nil {
			created := t.GetCreatedAt()
			if created.Before(criteria.DateRange.Start) || created.After(criteria.DateRange.End) {
				continue
			}
		}

		filtered = append(filtered, t)
	}

	total := len(filtered)

	// Simple pagination
	if criteria.PageSize > 0 {
		start := 0
		if criteria.Page > 1 {
			start = (criteria.Page - 1) * criteria.PageSize
		}
		if start > total {
			return []*ticketdomain.Ticket{}, total, nil
		}
		end := start + criteria.PageSize
		if end > total {
			end = total
		}
		return filtered[start:end], total, nil
	}

	return filtered, total, nil
}
