package sla

import (
	"context"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/slaalerthistory"
	"itsm-backend/ent/slaalertrule"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/slametric"
	"itsm-backend/ent/slaviolation"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

// Map ent SLADefinition to domain SLADefinition
func toSLADefinitionDomain(e *ent.SLADefinition) *SLADefinition {
	if e == nil {
		return nil
	}
	return &SLADefinition{
		ID:              e.ID,
		Name:            e.Name,
		Description:     e.Description,
		ServiceType:     e.ServiceType,
		Priority:        e.Priority,
		ResponseTime:    e.ResponseTime,
		ResolutionTime:  e.ResolutionTime,
		BusinessHours:   e.BusinessHours,
		EscalationRules: e.EscalationRules,
		Conditions:      e.Conditions,
		IsActive:        e.IsActive,
		TenantID:        e.TenantID,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func (r *EntRepository) CreateDefinition(ctx context.Context, s *SLADefinition) (*SLADefinition, error) {
	e, err := r.client.SLADefinition.Create().
		SetName(s.Name).
		SetDescription(s.Description).
		SetServiceType(s.ServiceType).
		SetPriority(s.Priority).
		SetResponseTime(s.ResponseTime).
		SetResolutionTime(s.ResolutionTime).
		SetBusinessHours(s.BusinessHours).
		SetEscalationRules(s.EscalationRules).
		SetConditions(s.Conditions).
		SetIsActive(s.IsActive).
		SetTenantID(s.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toSLADefinitionDomain(e), nil
}

func (r *EntRepository) GetDefinition(ctx context.Context, id int, tenantID int) (*SLADefinition, error) {
	e, err := r.client.SLADefinition.Query().
		Where(sladefinition.ID(id), sladefinition.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toSLADefinitionDomain(e), nil
}

func (r *EntRepository) ListDefinitions(ctx context.Context, tenantID int, page, size int) ([]*SLADefinition, int, error) {
	q := r.client.SLADefinition.Query().Where(sladefinition.TenantID(tenantID))
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	es, err := q.Limit(size).Offset((page - 1) * size).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var results []*SLADefinition
	for _, e := range es {
		results = append(results, toSLADefinitionDomain(e))
	}
	return results, total, nil
}

func (r *EntRepository) UpdateDefinition(ctx context.Context, s *SLADefinition) (*SLADefinition, error) {
	e, err := r.client.SLADefinition.UpdateOneID(s.ID).
		SetName(s.Name).
		SetDescription(s.Description).
		SetServiceType(s.ServiceType).
		SetPriority(s.Priority).
		SetResponseTime(s.ResponseTime).
		SetResolutionTime(s.ResolutionTime).
		SetBusinessHours(s.BusinessHours).
		SetEscalationRules(s.EscalationRules).
		SetConditions(s.Conditions).
		SetIsActive(s.IsActive).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toSLADefinitionDomain(e), nil
}

func (r *EntRepository) DeleteDefinition(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.SLADefinition.Delete().
		Where(sladefinition.ID(id), sladefinition.TenantID(tenantID)).
		Exec(ctx)
	return err
}

// --- Violations ---

func toSLAViolationDomain(e *ent.SLAViolation) *SLAViolation {
	if e == nil {
		return nil
	}
	var resolvedAt *time.Time
	if !e.ResolvedAt.IsZero() {
		t := e.ResolvedAt
		resolvedAt = &t
	}
	return &SLAViolation{
		ID:              e.ID,
		TicketID:        e.TicketID,
		SLADefinitionID: e.SLADefinitionID,
		ViolationType:   e.ViolationType,
		ViolationTime:   e.ViolationTime,
		Description:     e.Description,
		Severity:        e.Severity,
		IsResolved:      e.IsResolved,
		ResolvedAt:      resolvedAt,
		ResolutionNotes: e.ResolutionNotes,
		TenantID:        e.TenantID,
	}
}

func (r *EntRepository) CreateViolation(ctx context.Context, v *SLAViolation) (*SLAViolation, error) {
	creator := r.client.SLAViolation.Create().
		SetTicketID(v.TicketID).
		SetSLADefinitionID(v.SLADefinitionID).
		SetViolationType(v.ViolationType).
		SetViolationTime(v.ViolationTime).
		SetDescription(v.Description).
		SetSeverity(v.Severity).
		SetIsResolved(v.IsResolved).
		SetResolutionNotes(v.ResolutionNotes).
		SetTenantID(v.TenantID)

	if v.ResolvedAt != nil {
		creator.SetResolvedAt(*v.ResolvedAt)
	}

	e, err := creator.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toSLAViolationDomain(e), nil
}

func (r *EntRepository) ListViolations(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*SLAViolation, int, error) {
	q := r.client.SLAViolation.Query().Where(slaviolation.TenantID(tenantID))
	if val, ok := filters["is_resolved"]; ok {
		q = q.Where(slaviolation.IsResolved(val.(bool)))
	}
	if val, ok := filters["severity"]; ok {
		q = q.Where(slaviolation.Severity(val.(string)))
	}
	if val, ok := filters["violation_type"]; ok {
		q = q.Where(slaviolation.ViolationType(val.(string)))
	}
	if val, ok := filters["sla_definition_id"]; ok {
		q = q.Where(slaviolation.SLADefinitionID(val.(int)))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	es, err := q.Limit(size).Offset((page - 1) * size).Order(ent.Desc(slaviolation.FieldViolationTime)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	var res []*SLAViolation
	for _, e := range es {
		res = append(res, toSLAViolationDomain(e))
	}
	return res, total, nil
}

func (r *EntRepository) UpdateViolationStatus(ctx context.Context, id int, isResolved bool, notes string, tenantID int) error {
	update := r.client.SLAViolation.UpdateOneID(id).
		Where(slaviolation.TenantID(tenantID)).
		SetIsResolved(isResolved).
		SetResolutionNotes(notes)
	if isResolved {
		update.SetResolvedAt(time.Now())
	}
	return update.Exec(ctx)
}

// --- Alert Rules ---

func toSLAAlertRuleDomain(e *ent.SLAAlertRule) *SLAAlertRule {
	if e == nil {
		return nil
	}
	return &SLAAlertRule{
		ID:                   e.ID,
		SLADefinitionID:      e.SLADefinitionID,
		Name:                 e.Name,
		ThresholdPercentage:  e.ThresholdPercentage,
		AlertLevel:           e.AlertLevel,
		NotificationChannels: e.NotificationChannels,
		IsActive:             e.IsActive,
		TenantID:             e.TenantID,
	}
}

func (r *EntRepository) CreateAlertRule(ctx context.Context, ar *SLAAlertRule) (*SLAAlertRule, error) {
	e, err := r.client.SLAAlertRule.Create().
		SetSLADefinitionID(ar.SLADefinitionID).
		SetName(ar.Name).
		SetThresholdPercentage(ar.ThresholdPercentage).
		SetAlertLevel(ar.AlertLevel).
		SetNotificationChannels(ar.NotificationChannels).
		SetIsActive(ar.IsActive).
		SetTenantID(ar.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toSLAAlertRuleDomain(e), nil
}

func (r *EntRepository) GetAlertRule(ctx context.Context, id int, tenantID int) (*SLAAlertRule, error) {
	e, err := r.client.SLAAlertRule.Query().Where(slaalertrule.ID(id), slaalertrule.TenantID(tenantID)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return toSLAAlertRuleDomain(e), nil
}

func (r *EntRepository) ListAlertRules(ctx context.Context, tenantID int, filters map[string]interface{}) ([]*SLAAlertRule, error) {
	q := r.client.SLAAlertRule.Query().Where(slaalertrule.TenantID(tenantID))
	if val, ok := filters["sla_definition_id"]; ok {
		q = q.Where(slaalertrule.SLADefinitionID(val.(int)))
	}
	es, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	var res []*SLAAlertRule
	for _, e := range es {
		res = append(res, toSLAAlertRuleDomain(e))
	}
	return res, nil
}

func (r *EntRepository) UpdateAlertRule(ctx context.Context, ar *SLAAlertRule) (*SLAAlertRule, error) {
	e, err := r.client.SLAAlertRule.UpdateOneID(ar.ID).
		SetName(ar.Name).
		SetThresholdPercentage(ar.ThresholdPercentage).
		SetAlertLevel(ar.AlertLevel).
		SetNotificationChannels(ar.NotificationChannels).
		SetIsActive(ar.IsActive).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toSLAAlertRuleDomain(e), nil
}

func (r *EntRepository) DeleteAlertRule(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.SLAAlertRule.Delete().Where(slaalertrule.ID(id), slaalertrule.TenantID(tenantID)).Exec(ctx)
	return err
}

// --- Alert History ---

func toSLAAlertHistoryDomain(e *ent.SLAAlertHistory) *SLAAlertHistory {
	if e == nil {
		return nil
	}
	var resolvedAt *time.Time
	if !e.ResolvedAt.IsZero() {
		t := e.ResolvedAt
		resolvedAt = &t
	}
	return &SLAAlertHistory{
		ID:                  e.ID,
		AlertRuleID:         e.AlertRuleID,
		TicketID:            e.TicketID,
		TicketNumber:        e.TicketNumber,
		TicketTitle:         e.TicketTitle,
		AlertLevel:          e.AlertLevel,
		ThresholdPercentage: e.ThresholdPercentage,
		ActualPercentage:    e.ActualPercentage,
		NotificationSent:    e.NotificationSent,
		CreatedAt:           e.CreatedAt,
		ResolvedAt:          resolvedAt,
		TenantID:            e.TenantID,
	}
}

func (r *EntRepository) CreateAlertHistory(ctx context.Context, h *SLAAlertHistory) (*SLAAlertHistory, error) {
	creator := r.client.SLAAlertHistory.Create().
		SetAlertRuleID(h.AlertRuleID).
		SetTicketID(h.TicketID).
		SetTicketNumber(h.TicketNumber).
		SetTicketTitle(h.TicketTitle).
		SetAlertLevel(h.AlertLevel).
		SetThresholdPercentage(h.ThresholdPercentage).
		SetActualPercentage(h.ActualPercentage).
		SetNotificationSent(h.NotificationSent).
		SetTenantID(h.TenantID)

	if h.ResolvedAt != nil {
		creator.SetResolvedAt(*h.ResolvedAt)
	}

	e, err := creator.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toSLAAlertHistoryDomain(e), nil
}

func (r *EntRepository) ListAlertHistory(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}) ([]*SLAAlertHistory, int, error) {
	q := r.client.SLAAlertHistory.Query().Where(slaalerthistory.TenantID(tenantID))
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	es, err := q.Limit(size).Offset((page - 1) * size).Order(ent.Desc(slaalerthistory.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	var res []*SLAAlertHistory
	for _, e := range es {
		res = append(res, toSLAAlertHistoryDomain(e))
	}
	return res, total, nil
}

// --- Metrics ---

func toSLAMetricDomain(e *ent.SLAMetric) *SLAMetric {
	if e == nil {
		return nil
	}
	return &SLAMetric{
		ID:              e.ID,
		SLADefinitionID: e.SLADefinitionID,
		MetricType:      e.MetricType,
		MetricName:      e.MetricName,
		MetricValue:     e.MetricValue,
		Unit:            e.Unit,
		MeasurementTime: e.MeasurementTime,
		TenantID:        e.TenantID,
	}
}

func (r *EntRepository) CreateMetric(ctx context.Context, m *SLAMetric) (*SLAMetric, error) {
	e, err := r.client.SLAMetric.Create().
		SetSLADefinitionID(m.SLADefinitionID).
		SetMetricType(m.MetricType).
		SetMetricName(m.MetricName).
		SetMetricValue(m.MetricValue).
		SetUnit(m.Unit).
		SetMeasurementTime(m.MeasurementTime).
		SetTenantID(m.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toSLAMetricDomain(e), nil
}

func (r *EntRepository) GetMetrics(ctx context.Context, tenantID int, filters map[string]interface{}) ([]*SLAMetric, error) {
	q := r.client.SLAMetric.Query().Where(slametric.TenantID(tenantID))
	if val, ok := filters["sla_definition_id"]; ok {
		q = q.Where(slametric.SLADefinitionID(val.(int)))
	}
	if val, ok := filters["metric_type"]; ok {
		q = q.Where(slametric.MetricType(val.(string)))
	}
	es, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	var res []*SLAMetric
	for _, e := range es {
		res = append(res, toSLAMetricDomain(e))
	}
	return res, nil
}

func (r *EntRepository) GetSLAMonitoring(ctx context.Context, tenantID int, startTime, endTime string) (map[string]interface{}, error) {
	// Count violations
	totalViolations, _ := r.client.SLAViolation.Query().
		Where(slaviolation.TenantID(tenantID)).
		Count(ctx)

	resolvedViolations, _ := r.client.SLAViolation.Query().
		Where(slaviolation.TenantID(tenantID), slaviolation.IsResolved(true)).
		Count(ctx)

	// Calculate compliance rate
	var complianceRate float64
	if totalViolations > 0 {
		complianceRate = float64(totalViolations-resolvedViolations) / float64(totalViolations)
	}

	// Get active SLA definitions
	activeSLAs, _ := r.client.SLADefinition.Query().
		Where(sladefinition.TenantID(tenantID), sladefinition.IsActive(true)).
		Count(ctx)

	// Get alert rules count
	alertRules, _ := r.client.SLAAlertRule.Query().
		Where(slaalertrule.TenantID(tenantID), slaalertrule.IsActive(true)).
		Count(ctx)

	return map[string]interface{}{
		"total_violations":    totalViolations,
		"resolved_violations": resolvedViolations,
		"active_violations":   totalViolations - resolvedViolations,
		"compliance_rate":     complianceRate,
		"active_slas":         activeSLAs,
		"active_alert_rules":  alertRules,
	}, nil
}
