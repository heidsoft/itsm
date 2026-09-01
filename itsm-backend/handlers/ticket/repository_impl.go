package ticket

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/handlers/common/datascope"
	"itsm-backend/repository/base"
	"itsm-backend/repository/ticket"
)

// EntRepository wraps the existing repository/ticket implementation.
// All DB operations go through the shared repository to respect tenant isolation
// and existing business logic.
type EntRepository struct {
	repo ticket.Repository
}

// NewEntRepository creates a handlers/ticket EntRepository backed by the shared repository/ticket implementation.
func NewEntRepository(repo ticket.Repository) *EntRepository {
	return &EntRepository{repo: repo}
}

// toDomain converts repository/ticket.Ticket to handlers/ticket.Ticket
func (r *EntRepository) toDomain(t *ticket.Ticket) *Ticket {
	if t == nil {
		return nil
	}
	return &Ticket{
		ID:                    t.ID,
		TicketNumber:          t.TicketNumber,
		Title:                 t.Title,
		Description:           t.Description,
		Status:                string(t.Status),
		Priority:              string(t.Priority),
		Type:                  string(t.Type),
		TicketTypeID:          t.TicketTypeID,
		TicketTypeCode:        t.TicketTypeCode,
		TicketTypeName:        t.TicketTypeName,
		FormFields:            t.FormFields,
		RequesterID:           t.RequesterID,
		AssigneeID:            t.AssigneeID,
		TenantID:              t.TenantID,
		TemplateID:            t.TemplateID,
		CategoryID:            t.CategoryID,
		DepartmentID:          t.DepartmentID,
		ParentTicketID:        t.ParentTicketID,
		SLADefinitionID:       t.SLADefinitionID,
		SLAResponseDeadline:   t.SLAResponseDeadline,
		SLAResolutionDeadline: t.SLAResolutionDeadline,
		FirstResponseAt:       t.FirstResponseAt,
		ResolvedAt:            t.ResolvedAt,
		ClosedAt:              t.ClosedAt,
		Resolution:            t.Resolution,
		Rating:                t.Rating,
		RatingComment:         t.RatingComment,
		RatedAt:               t.RatedAt,
		RatedBy:               t.RatedBy,
		Version:                t.Version,
		IsManagedByMSP:        t.IsManagedByMSP,
		MSPProviderID:         t.MSPProviderID,
		ManagedByUserID:       t.ManagedByUserID,
		MSPTicketID:           t.MSPTicketID,
		CreatedAt:             t.CreatedAt,
		UpdatedAt:             t.UpdatedAt,
		DeletedAt:             t.DeletedAt,
	}
}

// toRepoCreateParams converts handlers/ticket CreateParams to repository/ticket CreateParams
func toRepoCreateParams(p *CreateParams) *ticket.CreateParams {
	if p == nil {
		return nil
	}
	return &ticket.CreateParams{
		Title:          p.Title,
		Description:    p.Description,
		Type:           ticket.Type(p.Type),
		Priority:       ticket.Priority(p.Priority),
		RequesterID:    p.RequesterID,
		AssigneeID:     p.AssigneeID,
		TicketTypeID:   p.TicketTypeID,
		TicketTypeCode: p.TicketTypeCode,
		TicketTypeName: p.TicketTypeName,
		CategoryID:     p.CategoryID,
		TemplateID:     p.TemplateID,
		ParentTicketID: p.ParentTicketID,
		FormFields:     p.FormFields,
		TagIDs:         p.TagIDs,
	}
}

// toRepoUpdateParams converts handlers/ticket UpdateParams to repository/ticket UpdateParams
func toRepoUpdateParams(p *UpdateParams) *ticket.UpdateParams {
	if p == nil {
		return nil
	}
	rp := &ticket.UpdateParams{
		Title:       p.Title,
		Description: p.Description,
		AssigneeID:  p.AssigneeID,
		CategoryID:  p.CategoryID,
		Resolution:  p.Resolution,
		FormFields:  p.FormFields,
		Version:     p.Version,
		ReplaceTags: p.ReplaceTags,
		TagIDs:      p.TagIDs,
	}
	if p.Status != nil {
		s := ticket.Status(*p.Status)
		rp.Status = &s
	}
	if p.Type != nil {
		t := ticket.Type(*p.Type)
		rp.Type = &t
	}
	if p.Priority != nil {
		pr := ticket.Priority(*p.Priority)
		rp.Priority = &pr
	}
	return rp
}

// toDomainList converts a list of repository/ticket.Ticket to handlers/ticket.Ticket
func (r *EntRepository) toDomainList(ts []*ticket.Ticket) []*Ticket {
	result := make([]*Ticket, 0, len(ts))
	for _, t := range ts {
		result = append(result, r.toDomain(t))
	}
	return result
}

func (r *EntRepository) Create(ctx context.Context, params *CreateParams, tenantID int) (*Ticket, error) {
	t, err := r.repo.Create(ctx, toRepoCreateParams(params), tenantID)
	if err != nil {
		return nil, err
	}
	return r.toDomain(t), nil
}

func (r *EntRepository) GetByID(ctx context.Context, id int, tenantID int) (*Ticket, error) {
	t, err := r.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	return r.toDomain(t), nil
}

func (r *EntRepository) GetByNumber(ctx context.Context, ticketNumber string, tenantID int) (*Ticket, error) {
	t, err := r.repo.GetByNumber(ctx, ticketNumber, tenantID)
	if err != nil {
		return nil, err
	}
	return r.toDomain(t), nil
}

func (r *EntRepository) Update(ctx context.Context, id int, params *UpdateParams, tenantID int) (*Ticket, error) {
	t, err := r.repo.Update(ctx, id, toRepoUpdateParams(params), tenantID)
	if err != nil {
		return nil, err
	}
	return r.toDomain(t), nil
}

func (r *EntRepository) Delete(ctx context.Context, id int, tenantID int) error {
	return r.repo.Delete(ctx, id, tenantID)
}

func (r *EntRepository) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, dataScope datascope.DataScope, currentUserID int) ([]*Ticket, int, error) {
	fp := &ticket.FilterParams{}
	if filters != nil {
		if v, ok := filters["status"].(string); ok && v != "" {
			s := ticket.Status(v)
			fp.Status = &s
		}
		if v, ok := filters["priority"].(string); ok && v != "" {
			pt := ticket.Priority(v)
			fp.Priority = &pt
		}
		if v, ok := filters["type"].(string); ok && v != "" {
			t := ticket.Type(v)
			fp.Type = &t
		}
		if v, ok := filters["assignee_id"].(int); ok && v > 0 {
			fp.AssigneeID = &v
		}
		if v, ok := filters["requester_id"].(int); ok && v > 0 {
			fp.RequesterID = &v
		}
		if v, ok := filters["category_id"].(int); ok && v > 0 {
			fp.CategoryID = &v
		}
		if v, ok := filters["parent_ticket_id"].(int); ok && v > 0 {
			fp.ParentTicketID = &v
		}
		if v, ok := filters["template_id"].(int); ok && v > 0 {
			fp.TemplateID = &v
		}
		if v, ok := filters["is_overdue"].(bool); ok {
			fp.IsOverdue = v
		}
		if v, ok := filters["keyword"].(string); ok {
			fp.Keyword = v
		}
	}
	fp.DataScope = ticket.DataScope(dataScope)
	fp.CurrentUserID = currentUserID

	result, err := r.repo.List(ctx, tenantID, fp, &base.QueryParams{Page: page, PageSize: size, OrderBy: "created_at", OrderDir: "desc"})
	if err != nil {
		return nil, 0, err
	}
	return r.toDomainList(result.Data), result.Total, nil
}

func (r *EntRepository) BatchDelete(ctx context.Context, ids []int, tenantID int) error {
	return r.repo.BatchDelete(ctx, ids, tenantID)
}

func (r *EntRepository) Exists(ctx context.Context, id int, tenantID int) (bool, error) {
	return r.repo.Exists(ctx, id, tenantID)
}

func (r *EntRepository) FindByAssignee(ctx context.Context, assigneeID int, tenantID int) ([]*Ticket, error) {
	ts, err := r.repo.FindByAssignee(ctx, assigneeID, tenantID)
	if err != nil {
		return nil, err
	}
	return r.toDomainList(ts), nil
}

func (r *EntRepository) FindOverdue(ctx context.Context, tenantID int) ([]*Ticket, error) {
	ts, err := r.repo.FindOverdue(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return r.toDomainList(ts), nil
}

func (r *EntRepository) Search(ctx context.Context, keyword string, tenantID int) ([]*Ticket, error) {
	fp := &ticket.FilterParams{Keyword: keyword}
	result, err := r.repo.List(ctx, tenantID, fp, &base.QueryParams{Page: 1, PageSize: 100})
	if err != nil {
		return nil, err
	}
	return r.toDomainList(result.Data), nil
}

func (r *EntRepository) GetStats(ctx context.Context, tenantID int) (*TicketStats, error) {
	statusCounts, err := r.repo.CountByStatus(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get status counts: %w", err)
	}
	priorityCounts, err := r.repo.CountByPriority(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get priority counts: %w", err)
	}

	stats := &TicketStats{}
	for status, count := range statusCounts {
		stats.TotalTickets += count
		switch status {
		case ticket.StatusNew, ticket.StatusOpen:
			stats.OpenTickets += count
		case ticket.StatusInProgress:
			stats.InProgressTickets += count
		case ticket.StatusPending:
			stats.PendingTickets += count
		case ticket.StatusResolved:
			stats.ResolvedTickets += count
		case ticket.StatusClosed:
			stats.ClosedTickets += count
		}
	}
	for priority, count := range priorityCounts {
		switch priority {
		case ticket.PriorityCritical:
			stats.CriticalTickets += count
		case ticket.PriorityHigh:
			stats.HighTickets += count
		}
	}
	return stats, nil
}

func (r *EntRepository) GenerateTicketNumber(ctx context.Context, tenantID int) (string, error) {
	return r.repo.GenerateTicketNumber(ctx, tenantID)
}

func (r *EntRepository) UpdateStatus(ctx context.Context, id int, status string, tenantID int) (*Ticket, error) {
	t, err := r.repo.UpdateStatus(ctx, id, ticket.Status(status), tenantID)
	if err != nil {
		return nil, err
	}
	return r.toDomain(t), nil
}

func (r *EntRepository) AssignTicket(ctx context.Context, id int, assigneeID int, tenantID int) (*Ticket, error) {
	t, err := r.repo.AssignTicket(ctx, id, assigneeID, tenantID)
	if err != nil {
		return nil, err
	}
	return r.toDomain(t), nil
}

func (r *EntRepository) ResolveTicket(ctx context.Context, id int, resolution string, tenantID int) (*Ticket, error) {
	current, err := r.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	params := &ticket.UpdateParams{
		Status:     ptrTicketStatus(ticket.StatusResolved),
		Resolution: &resolution,
		Version:    current.Version,
	}
	t, err := r.repo.Update(ctx, id, params, tenantID)
	if err != nil {
		return nil, err
	}
	return r.toDomain(t), nil
}

func (r *EntRepository) CloseTicket(ctx context.Context, id int, tenantID int) (*Ticket, error) {
	t, err := r.repo.UpdateStatus(ctx, id, ticket.StatusClosed, tenantID)
	if err != nil {
		return nil, err
	}
	return r.toDomain(t), nil
}

func (r *EntRepository) EscalateTicket(ctx context.Context, id int, reason string, tenantID int, escalatedBy int) (*Ticket, error) {
	current, err := r.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	escalated := ticket.StatusInProgress
	params := &ticket.UpdateParams{Status: &escalated, Version: current.Version}
	t, err := r.repo.Update(ctx, id, params, tenantID)
	if err != nil {
		return nil, err
	}
	return r.toDomain(t), nil
}

func (r *EntRepository) UpdateSLADeadlines(ctx context.Context, id int, responseDeadline, resolutionDeadline *time.Time, slaDefinitionID *int, tenantID int) error {
	return r.repo.UpdateSLADeadlines(ctx, id, responseDeadline, resolutionDeadline, slaDefinitionID, tenantID)
}

// Template operations
func (r *EntRepository) CreateTemplate(ctx context.Context, tmpl *TicketTemplate, tenantID int) (*TicketTemplate, error) {
	return tmpl, nil
}

func (r *EntRepository) UpdateTemplate(ctx context.Context, id int, tmpl *TicketTemplate, tenantID int) (*TicketTemplate, error) {
	return tmpl, nil
}

func (r *EntRepository) DeleteTemplate(ctx context.Context, id int, tenantID int) error {
	return nil
}

func (r *EntRepository) GetTemplate(ctx context.Context, id int, tenantID int) (*TicketTemplate, error) {
	return nil, fmt.Errorf("template not found")
}

func (r *EntRepository) ListTemplates(ctx context.Context, tenantID int) ([]*TicketTemplate, error) {
	return []*TicketTemplate{}, nil
}

func (r *EntRepository) UpdateTemplateStatus(ctx context.Context, id int, isActive bool, tenantID int) (*TicketTemplate, error) {
	return &TicketTemplate{ID: id, IsActive: isActive}, nil
}

func (r *EntRepository) CopyTemplate(ctx context.Context, id int, newName string, tenantID int) (*TicketTemplate, error) {
	return &TicketTemplate{ID: id, Name: newName}, nil
}

func (r *EntRepository) GetTemplateCategories(ctx context.Context, tenantID int) ([]string, error) {
	return []string{}, nil
}

func ptrTicketStatus(v ticket.Status) *ticket.Status {
	return &v
}
