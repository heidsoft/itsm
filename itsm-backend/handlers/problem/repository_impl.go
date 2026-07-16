package problem

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/incident"
	entpredicate "itsm-backend/ent/predicate"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/ticket"
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
	if e.ResolvedAt != nil {
		p.ResolvedAt = e.ResolvedAt
	}
	if e.ClosedAt != nil {
		p.ClosedAt = e.ClosedAt
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

func (r *EntRepository) AddAssociations(ctx context.Context, tenantID, problemID int, relatedType string, relatedIDs []int) error {
	exists, err := r.client.Problem.Query().
		Where(problem.IDEQ(problemID), problem.TenantIDEQ(tenantID), problem.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("problem not found")
	}

	switch relatedType {
	case "ticket":
		count, err := r.client.Ticket.Query().
			Where(ticket.IDIn(relatedIDs...), ticket.TenantIDEQ(tenantID), ticket.DeletedAtIsNil()).
			Count(ctx)
		if err != nil {
			return err
		}
		if count != len(relatedIDs) {
			return fmt.Errorf("one or more tickets do not belong to the current tenant")
		}
	case "incident":
		count, err := r.client.Incident.Query().
			Where(incident.IDIn(relatedIDs...), incident.TenantIDEQ(tenantID)).
			Count(ctx)
		if err != nil {
			return err
		}
		if count != len(relatedIDs) {
			return fmt.Errorf("one or more incidents do not belong to the current tenant")
		}
	case "change":
		count, err := r.client.Change.Query().
			Where(change.IDIn(relatedIDs...), change.TenantIDEQ(tenantID)).
			Count(ctx)
		if err != nil {
			return err
		}
		if count != len(relatedIDs) {
			return fmt.Errorf("one or more changes do not belong to the current tenant")
		}
	default:
		return fmt.Errorf("unsupported related type: %s", relatedType)
	}

	update := r.client.Problem.Update().
		Where(problem.IDEQ(problemID), problem.TenantIDEQ(tenantID), problem.DeletedAtIsNil())
	switch relatedType {
	case "ticket":
		update.AddTicketIDs(relatedIDs...)
	case "incident":
		update.AddIncidentIDs(relatedIDs...)
	case "change":
		update.AddChangeIDs(relatedIDs...)
	}
	updated, err := update.Save(ctx)
	if err == nil && updated != 1 {
		return fmt.Errorf("problem not found")
	}
	return err
}

func (r *EntRepository) RemoveAssociation(ctx context.Context, tenantID, problemID int, relatedType string, relatedID int) error {
	update := r.client.Problem.Update().
		Where(problem.IDEQ(problemID), problem.TenantIDEQ(tenantID), problem.DeletedAtIsNil())

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

	updated, err := update.Save(ctx)
	if err == nil && updated != 1 {
		return fmt.Errorf("problem not found")
	}
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
		Where(problem.ID(id), problem.TenantID(tenantID), problem.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomain(e), nil
}

func (r *EntRepository) GetWithAssociations(ctx context.Context, id int, tenantID int) (*Problem, error) {
	e, err := r.client.Problem.Query().
		Where(problem.ID(id), problem.TenantID(tenantID), problem.DeletedAtIsNil()).
		WithTickets(func(q *ent.TicketQuery) {
			q.Where(ticket.TenantIDEQ(tenantID), ticket.DeletedAtIsNil()).
				Select("id", "title", "status", "ticket_number")
		}).
		WithIncidents(func(q *ent.IncidentQuery) {
			q.Where(incident.TenantIDEQ(tenantID)).
				Select("id", "title", "status", "incident_number")
		}).
		WithChanges(func(q *ent.ChangeQuery) {
			q.Where(change.TenantIDEQ(tenantID)).
				Select("id", "title", "status")
		}).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomainWithAssociations(e), nil
}

func (r *EntRepository) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*Problem, int, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 200 {
		size = 200
	}
	query := r.client.Problem.Query().Where(problem.TenantID(tenantID), problem.DeletedAtIsNil())

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
		Where(problem.TenantIDEQ(p.TenantID), problem.DeletedAtIsNil()).
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
	if p.ResolvedAt != nil {
		update.SetResolvedAt(*p.ResolvedAt)
	} else {
		update.ClearResolvedAt()
	}
	if p.ClosedAt != nil {
		update.SetClosedAt(*p.ClosedAt)
	} else {
		update.ClearClosedAt()
	}

	saved, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomain(saved), nil
}

func (r *EntRepository) Delete(ctx context.Context, id int, tenantID int) error {
	updated, err := r.client.Problem.Update().
		Where(problem.IDEQ(id), problem.TenantIDEQ(tenantID), problem.DeletedAtIsNil()).
		SetDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return err
	}
	if updated != 1 {
		return fmt.Errorf("problem not found")
	}
	return nil
}

func (r *EntRepository) GetStats(ctx context.Context, tenantID int) (*ProblemStats, error) {
	base := []entpredicate.Problem{problem.TenantIDEQ(tenantID), problem.DeletedAtIsNil()}
	query := r.client.Problem.Query().Where(base...)

	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Simple count queries. Optimization: group by status/priority?
	// For now keeping it simple as per original service.
	count := func(pred entpredicate.Problem) (int, error) {
		return r.client.Problem.Query().Where(problem.TenantIDEQ(tenantID), problem.DeletedAtIsNil(), pred).Count(ctx)
	}
	open, err := count(problem.StatusEQ("open"))
	if err != nil {
		return nil, err
	}
	inProgress, err := count(problem.StatusIn("investigating", "in_progress"))
	if err != nil {
		return nil, err
	}
	resolved, err := count(problem.StatusEQ("resolved"))
	if err != nil {
		return nil, err
	}
	closed, err := count(problem.StatusEQ("closed"))
	if err != nil {
		return nil, err
	}
	high, err := count(problem.PriorityIn("high", "critical"))
	if err != nil {
		return nil, err
	}

	return &ProblemStats{
		Total:        total,
		Open:         open,
		InProgress:   inProgress,
		Resolved:     resolved,
		Closed:       closed,
		HighPriority: high,
	}, nil
}
