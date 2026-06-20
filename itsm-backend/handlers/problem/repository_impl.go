package problem

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/problem"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) toDomain(e *ent.Problem) *Problem {
	if e == nil {
		return nil
	}
	p := &Problem{
		ID:          e.ID,
		Title:       e.Title,
		Description: e.Description,
		Status:      e.Status,
		Priority:    e.Priority,
		Category:    e.Category,
		RootCause:   e.RootCause,
		Impact:      e.Impact,
		CreatedBy:   e.CreatedBy,
		TenantID:    e.TenantID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
	// Handle optional fields
	// Ent fields might be zero value if not set, or pointer depending on schema.
	// Schema says: AssigneeID optional.
	if e.AssigneeID != 0 {
		id := e.AssigneeID
		p.AssigneeID = &id
	}
	return p
}

func (r *EntRepository) toDomainWithAssociations(e *ent.Problem) *Problem {
	p := r.toDomain(e)

	if e.Edges.Tickets != nil {
		p.Tickets = make([]*AssociatedItem, 0, len(e.Edges.Tickets))
		for _, t := range e.Edges.Tickets {
			p.Tickets = append(p.Tickets, &AssociatedItem{
				ID:     t.ID,
				Title:  t.Title,
				Status: t.Status,
				Number: t.TicketNumber,
				Type:   "ticket",
			})
		}
	}
	if e.Edges.Incidents != nil {
		p.Incidents = make([]*AssociatedItem, 0, len(e.Edges.Incidents))
		for _, inc := range e.Edges.Incidents {
			p.Incidents = append(p.Incidents, &AssociatedItem{
				ID:     inc.ID,
				Title:  inc.Title,
				Status: inc.Status,
				Number: inc.IncidentNumber,
				Type:   "incident",
			})
		}
	}
	if e.Edges.Changes != nil {
		p.Changes = make([]*AssociatedItem, 0, len(e.Edges.Changes))
		for _, ch := range e.Edges.Changes {
			p.Changes = append(p.Changes, &AssociatedItem{
				ID:     ch.ID,
				Title:  ch.Title,
				Status: ch.Status,
				Type:   "change",
			})
		}
	}

	return p
}

func (r *EntRepository) AddAssociations(ctx context.Context, problemID int, relatedType string, relatedIDs []int) error {
	update := r.client.Problem.UpdateOneID(problemID)

	switch relatedType {
	case "ticket":
		update.AddTicketIDs(relatedIDs...)
	case "incident":
		update.AddIncidentIDs(relatedIDs...)
	case "change":
		update.AddChangeIDs(relatedIDs...)
	default:
		return fmt.Errorf("unsupported related type: %s", relatedType)
	}

	_, err := update.Save(ctx)
	return err
}

func (r *EntRepository) RemoveAssociation(ctx context.Context, problemID int, relatedType string, relatedID int) error {
	update := r.client.Problem.UpdateOneID(problemID)

	switch relatedType {
	case "ticket":
		update.RemoveTicketIDs(relatedID)
	case "incident":
		update.RemoveIncidentIDs(relatedID)
	case "change":
		update.RemoveChangeIDs(relatedID)
	default:
		return fmt.Errorf("unsupported related type: %s", relatedType)
	}

	_, err := update.Save(ctx)
	return err
}

func (r *EntRepository) Create(ctx context.Context, p *Problem) (*Problem, error) {
	create := r.client.Problem.Create().
		SetTitle(p.Title).
		SetDescription(p.Description).
		SetStatus(p.Status).
		SetPriority(p.Priority).
		SetCategory(p.Category).
		SetRootCause(p.RootCause).
		SetImpact(p.Impact).
		SetCreatedBy(p.CreatedBy).
		SetTenantID(p.TenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now())

	if p.AssigneeID != nil {
		create.SetAssigneeID(*p.AssigneeID)
	}

	saved, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomain(saved), nil
}

func (r *EntRepository) Get(ctx context.Context, id int, tenantID int) (*Problem, error) {
	e, err := r.client.Problem.Query().
		Where(problem.ID(id), problem.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomain(e), nil
}

func (r *EntRepository) GetWithAssociations(ctx context.Context, id int, tenantID int) (*Problem, error) {
	e, err := r.client.Problem.Query().
		Where(problem.ID(id), problem.TenantID(tenantID)).
		WithTickets(func(q *ent.TicketQuery) {
			q.Select("id", "title", "status", "ticket_number")
		}).
		WithIncidents(func(q *ent.IncidentQuery) {
			q.Select("id", "title", "status", "incident_number")
		}).
		WithChanges(func(q *ent.ChangeQuery) {
			q.Select("id", "title", "status")
		}).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomainWithAssociations(e), nil
}

func (r *EntRepository) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*Problem, int, error) {
	query := r.client.Problem.Query().Where(problem.TenantID(tenantID))

	if v, ok := filters["status"].(string); ok && v != "" {
		query = query.Where(problem.StatusEQ(v))
	}
	if v, ok := filters["priority"].(string); ok && v != "" {
		query = query.Where(problem.PriorityEQ(v))
	}
	if v, ok := filters["category"].(string); ok && v != "" {
		query = query.Where(problem.CategoryEQ(v))
	}
	if v, ok := filters["keyword"].(string); ok && v != "" {
		query = query.Where(problem.Or(
			problem.TitleContains(v),
			problem.DescriptionContains(v),
		))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	list, err := query.
		Offset((page - 1) * size).
		Limit(size).
		Order(ent.Desc(problem.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var result []*Problem
	for _, item := range list {
		result = append(result, r.toDomain(item))
	}
	return result, total, nil
}

func (r *EntRepository) Update(ctx context.Context, p *Problem) (*Problem, error) {
	update := r.client.Problem.UpdateOneID(p.ID).
		SetTitle(p.Title).
		SetDescription(p.Description).
		SetStatus(p.Status).
		SetPriority(p.Priority).
		SetCategory(p.Category).
		SetRootCause(p.RootCause).
		SetImpact(p.Impact).
		SetUpdatedAt(time.Now())

	if p.AssigneeID != nil {
		update.SetAssigneeID(*p.AssigneeID)
	} else {
		update.ClearAssigneeID()
	}

	saved, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomain(saved), nil
}

func (r *EntRepository) Delete(ctx context.Context, id int, tenantID int) error {
	return r.client.Problem.DeleteOneID(id).
		Where(problem.TenantID(tenantID)).
		Exec(ctx)
}

func (r *EntRepository) GetStats(ctx context.Context, tenantID int) (*ProblemStats, error) {
	query := r.client.Problem.Query().Where(problem.TenantID(tenantID))

	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Simple count queries. Optimization: group by status/priority?
	// For now keeping it simple as per original service.
	open, _ := r.client.Problem.Query().Where(problem.TenantID(tenantID), problem.StatusEQ("open")).Count(ctx)
	inProgress, _ := r.client.Problem.Query().Where(problem.TenantID(tenantID), problem.StatusEQ("in_progress")).Count(ctx)
	resolved, _ := r.client.Problem.Query().Where(problem.TenantID(tenantID), problem.StatusEQ("resolved")).Count(ctx)
	closed, _ := r.client.Problem.Query().Where(problem.TenantID(tenantID), problem.StatusEQ("closed")).Count(ctx)
	high, _ := r.client.Problem.Query().Where(problem.TenantID(tenantID), problem.PriorityIn("high", "critical")).Count(ctx)

	return &ProblemStats{
		Total:        total,
		Open:         open,
		InProgress:   inProgress,
		Resolved:     resolved,
		Closed:       closed,
		HighPriority: high,
	}, nil
}
