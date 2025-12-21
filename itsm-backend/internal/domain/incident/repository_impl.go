package incident

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/incidentevent"
	"itsm-backend/ent/incidentrule"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

// toDomain converts ent.Incident to domain Incident
func (r *EntRepository) toDomain(e *ent.Incident) *Incident {
	if e == nil {
		return nil
	}
	return &Incident{
		ID:                  e.ID,
		Title:               e.Title,
		Description:         e.Description,
		Status:              e.Status,
		Priority:            e.Priority,
		Severity:            e.Severity,
		IncidentNumber:      e.IncidentNumber,
		ReporterID:          e.ReporterID,
		AssigneeID:          &e.AssigneeID,
		ConfigurationItemID: &e.ConfigurationItemID,
		Category:            e.Category,
		Subcategory:         e.Subcategory,
		ImpactAnalysis:      e.ImpactAnalysis,
		RootCause:           e.RootCause,
		ResolutionSteps:     e.ResolutionSteps,
		Metadata:            e.Metadata,
		DetectedAt:          e.DetectedAt,
		ResolvedAt:          &e.ResolvedAt,
		ClosedAt:            &e.ClosedAt,
		EscalatedAt:         &e.EscalatedAt,
		EscalationLevel:     e.EscalationLevel,
		IsAutomated:         e.IsAutomated,
		Source:              e.Source,
		TenantID:            e.TenantID,
		CreatedAt:           e.CreatedAt,
		UpdatedAt:           e.UpdatedAt,
	}
}

// toDomainEvent converts ent.IncidentEvent to domain IncidentEvent
func (r *EntRepository) toDomainEvent(e *ent.IncidentEvent) *IncidentEvent {
	if e == nil {
		return nil
	}
	return &IncidentEvent{
		ID:          e.ID,
		IncidentID:  e.IncidentID,
		EventType:   e.EventType,
		EventName:   e.EventName,
		Description: e.Description,
		Status:      e.Status,
		Severity:    e.Severity,
		Data:        e.Data,
		OccurredAt:  e.OccurredAt,
		UserID:      e.UserID,
		Source:      e.Source,
		Metadata:    e.Metadata,
		TenantID:    e.TenantID,
		CreatedAt:   e.CreatedAt,
	}
}

// toDomainRule converts ent.IncidentRule to domain IncidentRule
func (r *EntRepository) toDomainRule(e *ent.IncidentRule) *IncidentRule {
	if e == nil {
		return nil
	}
	return &IncidentRule{
		ID:             e.ID,
		Name:           e.Name,
		Description:    e.Description,
		Conditions:     e.Conditions,
		Actions:        e.Actions,
		IsActive:       e.IsActive,
		Priority:       e.Priority, // Now string
		ExecutionCount: e.ExecutionCount,
		LastExecutedAt: &e.LastExecutedAt,
		TenantID:       e.TenantID,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func (r *EntRepository) Create(ctx context.Context, i *Incident) (*Incident, error) {
	query := r.client.Incident.Create().
		SetTitle(i.Title).
		SetDescription(i.Description).
		SetStatus(i.Status).
		SetPriority(i.Priority).
		SetSeverity(i.Severity).
		SetIncidentNumber(i.IncidentNumber).
		SetReporterID(i.ReporterID).
		SetCategory(i.Category).
		SetSubcategory(i.Subcategory).
		SetImpactAnalysis(i.ImpactAnalysis).
		SetSource(i.Source).
		SetMetadata(i.Metadata).
		SetDetectedAt(i.DetectedAt).
		SetIsAutomated(i.IsAutomated).
		SetTenantID(i.TenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now())

	if i.AssigneeID != nil {
		query.SetAssigneeID(*i.AssigneeID)
	}
	if i.ConfigurationItemID != nil {
		query.SetConfigurationItemID(*i.ConfigurationItemID)
	}
	if i.ResolvedAt != nil {
		query.SetResolvedAt(*i.ResolvedAt)
	}
	if i.ClosedAt != nil {
		query.SetClosedAt(*i.ClosedAt)
	}
	if i.EscalatedAt != nil {
		query.SetEscalatedAt(*i.EscalatedAt)
	}

	saved, err := query.Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomain(saved), nil
}

func (r *EntRepository) Get(ctx context.Context, id int, tenantID int) (*Incident, error) {
	i, err := r.client.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomain(i), nil
}

func (r *EntRepository) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*Incident, int, error) {
	query := r.client.Incident.Query().Where(incident.TenantIDEQ(tenantID))

	if v, ok := filters["status"].(string); ok && v != "" {
		query.Where(incident.StatusEQ(v))
	}
	if v, ok := filters["priority"].(string); ok && v != "" {
		query.Where(incident.PriorityEQ(v))
	}
	if v, ok := filters["keyword"].(string); ok && v != "" {
		query.Where(incident.Or(
			incident.TitleContains(v),
			incident.DescriptionContains(v),
			incident.IncidentNumberContains(v),
		))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	list, err := query.
		Offset((page - 1) * size).
		Limit(size).
		Order(ent.Desc(incident.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var result []*Incident
	for _, item := range list {
		result = append(result, r.toDomain(item))
	}
	return result, total, nil
}

func (r *EntRepository) Update(ctx context.Context, i *Incident) (*Incident, error) {
	u := r.client.Incident.UpdateOneID(i.ID).
		SetUpdatedAt(time.Now()).
		SetTitle(i.Title).
		SetDescription(i.Description).
		SetStatus(i.Status).
		SetPriority(i.Priority).
		SetSeverity(i.Severity).
		SetCategory(i.Category).
		SetSubcategory(i.Subcategory).
		SetImpactAnalysis(i.ImpactAnalysis).
		SetRootCause(i.RootCause).
		SetResolutionSteps(i.ResolutionSteps).
		SetMetadata(i.Metadata).
		SetEscalationLevel(i.EscalationLevel)

	if i.AssigneeID != nil {
		u.SetAssigneeID(*i.AssigneeID)
	}
	if i.ResolvedAt != nil {
		u.SetResolvedAt(*i.ResolvedAt)
	}
	if i.ClosedAt != nil {
		u.SetClosedAt(*i.ClosedAt)
	}
	if i.EscalatedAt != nil {
		u.SetEscalatedAt(*i.EscalatedAt)
	}

	saved, err := u.Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomain(saved), nil
}

func (r *EntRepository) Delete(ctx context.Context, id int, tenantID int) error {
	return r.client.Incident.DeleteOneID(id).
		Where(incident.TenantIDEQ(tenantID)).
		Exec(ctx)
}

func (r *EntRepository) CountByPeriod(ctx context.Context, tenantID int, start, end time.Time) (int, error) {
	return r.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.CreatedAtGTE(start),
			incident.CreatedAtLT(end),
		).
		Count(ctx)
}

func (r *EntRepository) GenerateIncidentNumber(ctx context.Context, tenantID int, year int, month int) (string, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	count, err := r.CountByPeriod(ctx, tenantID, start, end)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("INC-%04d%02d-%06d", year, month, count+1), nil
}

func (r *EntRepository) CreateEvent(ctx context.Context, e *IncidentEvent) (*IncidentEvent, error) {
	saved, err := r.client.IncidentEvent.Create().
		SetIncidentID(e.IncidentID).
		SetEventType(e.EventType).
		SetEventName(e.EventName).
		SetDescription(e.Description).
		SetStatus(e.Status).
		SetSeverity(e.Severity).
		SetData(e.Data).
		SetOccurredAt(e.OccurredAt).
		SetUserID(e.UserID).
		SetSource(e.Source).
		SetMetadata(e.Metadata).
		SetTenantID(e.TenantID).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomainEvent(saved), nil
}

func (r *EntRepository) ListEvents(ctx context.Context, incidentID int, tenantID int) ([]*IncidentEvent, error) {
	list, err := r.client.IncidentEvent.Query().
		Where(
			incidentevent.IncidentIDEQ(incidentID),
			incidentevent.TenantIDEQ(tenantID),
		).
		Order(ent.Desc(incidentevent.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	var result []*IncidentEvent
	for _, item := range list {
		result = append(result, r.toDomainEvent(item))
	}
	return result, nil
}

func (r *EntRepository) ListActiveRules(ctx context.Context, tenantID int) ([]*IncidentRule, error) {
	list, err := r.client.IncidentRule.Query().
		Where(
			incidentrule.TenantIDEQ(tenantID),
			incidentrule.IsActiveEQ(true),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	var result []*IncidentRule
	for _, item := range list {
		result = append(result, r.toDomainRule(item))
	}
	return result, nil
}

func (r *EntRepository) UpdateRuleStats(ctx context.Context, ruleID int, count int, lastExecutedAt time.Time) error {
	return r.client.IncidentRule.UpdateOneID(ruleID).
		SetExecutionCount(count).
		SetLastExecutedAt(lastExecutedAt).
		Exec(ctx)
}
