package sla

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/predicate"
	"itsm-backend/ent/slaalerthistory"
	"itsm-backend/ent/slaalertrule"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/slametric"
	"itsm-backend/ent/slaviolation"
	"itsm-backend/ent/ticket"
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
		Where(sladefinition.TenantID(s.TenantID)).
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
		CreatedBy:       e.CreatedBy,
		TicketID:        e.TicketID,
		SLADefinitionID: e.SLADefinitionID,
		SLAName:         e.SLAName,
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
		SetCreatedBy(v.CreatedBy).
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

	es, err := q.Limit(size).Offset((page - 1) * size).Order(ent.Desc(slaviolation.FieldViolationTime)).
		WithTicket(func(tq *ent.TicketQuery) {
			tq.Select(ticket.FieldTitle).Select(ticket.FieldTicketNumber).Select(ticket.FieldPriority)
		}).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	var res []*SLAViolation
	for _, e := range es {
		d := toSLAViolationDomain(e)
		// 工单标题/编号/优先级来自 ticket edge（监控大屏列表展示需要）
		if t := e.Edges.Ticket; t != nil {
			d.TicketTitle = t.Title
			d.TicketNumber = t.TicketNumber
			d.TicketPriority = t.Priority
		}
		res = append(res, d)
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
		Where(slaalertrule.TenantID(ar.TenantID)).
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

// monitoringScanLimit 是单次聚合允许扫描的工单上限。命中上限时 truncated=true，
// 大屏必须提示样本不完整，而不是把偏小的分母当成全量真相。
const monitoringScanLimit = 20000

// monitoringAlertListLimit 大屏告警表最多返回的明细条数；activeAlerts 仍是全量计数。
const monitoringAlertListLimit = 50

// ticketResolvedStatuses 计入「已解决」的工单状态。以生命周期状态为权威口径，
// 不用 resolved_at 猜测，因为工单可以从 resolved 回到 in_progress 而时间戳仍保留。
var ticketResolvedStatuses = map[string]bool{
	common.TicketStatusResolved: true,
	common.TicketStatusClosed:   true,
}

// ticketFinishedStatuses 表示工单已无需继续响应/解决，用于风险判定。
var ticketFinishedStatuses = map[string]bool{
	common.TicketStatusResolved:  true,
	common.TicketStatusClosed:    true,
	common.TicketStatusCancelled: true,
}

// slaCohortTicket 是聚合所需的最小工单列集合。
type slaCohortTicket struct {
	id                    int
	status                string
	priority              string
	slaDefinitionID       int
	createdAt             time.Time
	firstResponseAt       time.Time
	resolvedAt            time.Time
	slaResponseDeadline   time.Time
	slaResolutionDeadline time.Time
}

// slaCohort 是监控大屏与绩效聚合共用的统计种群快照。
type slaCohort struct {
	tickets      []slaCohortTicket
	serviceTypes map[int]string // slaDefinitionID -> service_type
	violated     map[int]bool   // 当前仍有未解决违约的工单ID
	truncated    bool
}

// loadSLACohort 按租户 + 时间窗口加载统计种群，并可选按服务类型 / 优先级预过滤。
//
// 服务类型不是工单字段，它来自工单绑定的 SLA 定义，因此服务类型过滤会先解析出该
// 类型下的 SLA 定义ID集合，再作为工单谓词下推；本租户不存在该服务类型时返回空种群
// （而不是回退到全部工单）。
func (r *EntRepository) loadSLACohort(
	ctx context.Context,
	tenantID int,
	start, end time.Time,
	serviceTypeFilter, priorityFilter string,
) (*slaCohort, error) {
	defRows, err := r.client.SLADefinition.Query().
		Where(sladefinition.TenantID(tenantID)).
		Select(sladefinition.FieldID, sladefinition.FieldServiceType).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("load sla definitions: %w", err)
	}

	serviceTypes := make(map[int]string, len(defRows))
	var filterDefinitionIDs []int
	for _, d := range defRows {
		serviceTypes[d.ID] = strings.TrimSpace(d.ServiceType)
		if serviceTypeFilter != "" && serviceTypes[d.ID] == serviceTypeFilter {
			filterDefinitionIDs = append(filterDefinitionIDs, d.ID)
		}
	}

	cohort := &slaCohort{serviceTypes: serviceTypes, violated: map[int]bool{}}

	predicates := []predicate.Ticket{
		ticket.TenantID(tenantID),
		ticket.DeletedAtIsNil(),
		ticket.CreatedAtGTE(start),
		ticket.CreatedAtLTE(end),
	}
	if priorityFilter != "" {
		predicates = append(predicates, ticket.PriorityEQ(priorityFilter))
	}
	if serviceTypeFilter != "" {
		if serviceTypeFilter == SLAPerformanceUnassignedKey {
			predicates = append(predicates, ticket.SLADefinitionIDIsNil())
		} else {
			if len(filterDefinitionIDs) == 0 {
				return cohort, nil
			}
			predicates = append(predicates, ticket.SLADefinitionIDIn(filterDefinitionIDs...))
		}
	}

	rows, err := r.client.Ticket.Query().
		Where(predicates...).
		Select(
			ticket.FieldID,
			ticket.FieldStatus,
			ticket.FieldPriority,
			ticket.FieldSLADefinitionID,
			ticket.FieldCreatedAt,
			ticket.FieldFirstResponseAt,
			ticket.FieldResolvedAt,
			ticket.FieldSLAResponseDeadline,
			ticket.FieldSLAResolutionDeadline,
		).
		Limit(monitoringScanLimit + 1).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("load ticket cohort: %w", err)
	}

	cohort.tickets = make([]slaCohortTicket, 0, len(rows))
	for _, t := range rows {
		cohort.tickets = append(cohort.tickets, slaCohortTicket{
			id:                    t.ID,
			status:                t.Status,
			priority:              t.Priority,
			slaDefinitionID:       t.SLADefinitionID,
			createdAt:             t.CreatedAt,
			firstResponseAt:       t.FirstResponseAt,
			resolvedAt:            t.ResolvedAt,
			slaResponseDeadline:   t.SLAResponseDeadline,
			slaResolutionDeadline: t.SLAResolutionDeadline,
		})
	}
	if len(cohort.tickets) > monitoringScanLimit {
		cohort.tickets = cohort.tickets[:monitoringScanLimit]
		cohort.truncated = true
	}
	if len(cohort.tickets) == 0 {
		return cohort, nil
	}

	ids := make([]int, 0, len(cohort.tickets))
	for _, t := range cohort.tickets {
		ids = append(ids, t.id)
	}
	var groups []struct {
		TicketID int `json:"ticket_id"` // json tag matches SQL column for ent GroupBy().Scan()
	}
	if err := r.client.SLAViolation.Query().
		Where(
			slaviolation.TenantID(tenantID),
			slaviolation.IsResolved(false),
			slaviolation.TicketIDIn(ids...),
		).
		GroupBy(slaviolation.FieldTicketID).
		Scan(ctx, &groups); err != nil {
		return nil, fmt.Errorf("load violated tickets: %w", err)
	}
	for _, g := range groups {
		cohort.violated[g.TicketID] = true
	}

	return cohort, nil
}

// slaAggregation 是种群上的可加计数；按维度分组时每个分组一份。
type slaAggregation struct {
	total       int
	resolved    int
	violated    int
	atRisk      int
	respSamples int
	respMet     int
	resSamples  int
	resMet      int
	respDur     time.Duration
	respDurN    int
	resDur      time.Duration
	resDurN     int
}

// add 把一张工单计入聚合。所有判定只依赖数据库中的权威时间戳与状态，
// 截止时间缺失时不计入对应样本，避免把“未配置 SLA”当成违约或达标。
func (a *slaAggregation) add(t slaCohortTicket, violated bool, now time.Time) {
	a.total++
	if violated {
		a.violated++
	}
	if ticketResolvedStatuses[t.status] {
		a.resolved++
	}

	if !t.slaResponseDeadline.IsZero() && !t.firstResponseAt.IsZero() {
		a.respSamples++
		if !t.firstResponseAt.After(t.slaResponseDeadline) {
			a.respMet++
		}
	}
	if !t.slaResolutionDeadline.IsZero() && !t.resolvedAt.IsZero() {
		a.resSamples++
		if !t.resolvedAt.After(t.slaResolutionDeadline) {
			a.resMet++
		}
	}
	if !t.createdAt.IsZero() && !t.firstResponseAt.IsZero() {
		a.respDur += t.firstResponseAt.Sub(t.createdAt)
		a.respDurN++
	}
	if !t.createdAt.IsZero() && !t.resolvedAt.IsZero() {
		a.resDur += t.resolvedAt.Sub(t.createdAt)
		a.resDurN++
	}

	if ticketFinishedStatuses[t.status] {
		return
	}
	overdueResponse := !t.slaResponseDeadline.IsZero() &&
		t.slaResponseDeadline.Before(now) &&
		(t.firstResponseAt.IsZero() || t.firstResponseAt.After(t.slaResponseDeadline))
	overdueResolution := !t.slaResolutionDeadline.IsZero() && t.slaResolutionDeadline.Before(now)
	if overdueResponse || overdueResolution {
		a.atRisk++
	}
}

func (a *slaAggregation) toMonitoringMetrics(data *SLAMonitoringData) {
	data.TotalTickets = a.total
	data.ResolvedTickets = a.resolved
	data.ResolutionRate = ratePercent(a.resolved, a.total)
	data.ViolatedTickets = a.violated
	data.MetSlaTickets = a.total - a.violated
	data.ComplianceRate = ratePercent(a.total-a.violated, a.total)
	data.ViolationRate = ratePercent(a.violated, a.total)
	data.AtRiskTickets = a.atRisk
	data.ResponseTimeSamples = a.respSamples
	data.ResponseTimeMet = a.respMet
	data.ResponseTimeCompliance = ratePercent(a.respMet, a.respSamples)
	data.ResolutionTimeSamples = a.resSamples
	data.ResolutionTimeMet = a.resMet
	data.ResolutionTimeCompliance = ratePercent(a.resMet, a.resSamples)
	data.AverageResponseMinutes = averageMinutes(a.respDur, a.respDurN)
	data.AverageResolutionMinutes = averageMinutes(a.resDur, a.resDurN)
}

// ratePercent 计算百分数并保留一位小数；样本为 0 时诚实返回 0，
// 由前端根据对应的样本数量字段渲染“暂无样本”，不得回退成 100%。
func ratePercent(part, whole int) float64 {
	if whole <= 0 {
		return 0
	}
	return math.Round(float64(part)/float64(whole)*1000) / 10
}

// averageMinutes 把累计时长换算成分钟平均值并保留一位小数；无样本时返回 0。
func averageMinutes(total time.Duration, count int) float64 {
	if count <= 0 {
		return 0
	}
	perTicket := float64(total) / float64(count)
	return math.Round((perTicket/float64(time.Minute))*10) / 10
}

// GetSLAMonitoring 返回监控大屏的完整指标快照。
//
// 与旧实现的区别：严格遵守传入的时间窗口（旧实现接受了但从未使用），
// 合规率不再在零样本时返回 1.0，告警数量来自真实的未解决 SLAAlertHistory。
func (r *EntRepository) GetSLAMonitoring(ctx context.Context, tenantID int, start, end time.Time) (*SLAMonitoringData, error) {
	cohort, err := r.loadSLACohort(ctx, tenantID, start, end, "", "")
	if err != nil {
		return nil, err
	}

	agg := &slaAggregation{}
	for _, t := range cohort.tickets {
		agg.add(t, cohort.violated[t.id], time.Now())
	}

	data := &SLAMonitoringData{
		StartTime: start.UTC().Format(time.RFC3339),
		EndTime:   end.UTC().Format(time.RFC3339),
		Truncated: cohort.truncated,
		Alerts:    []*SLAAlertItem{},
	}
	agg.toMonitoringMetrics(data)

	// 违约记录计数（记录数，与上面的工单数口径不同）
	totalViolations, err := r.client.SLAViolation.Query().Where(
		slaviolation.TenantID(tenantID),
		slaviolation.ViolationTimeGTE(start),
		slaviolation.ViolationTimeLTE(end),
	).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count violations: %w", err)
	}
	resolvedViolations, err := r.client.SLAViolation.Query().Where(
		slaviolation.TenantID(tenantID),
		slaviolation.ViolationTimeGTE(start),
		slaviolation.ViolationTimeLTE(end),
		slaviolation.IsResolved(true),
	).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count resolved violations: %w", err)
	}
	data.TotalViolations = totalViolations
	data.ResolvedViolations = resolvedViolations
	data.ActiveViolations = totalViolations - resolvedViolations

	activeSlas, err := r.client.SLADefinition.Query().Where(
		sladefinition.TenantID(tenantID),
		sladefinition.IsActive(true),
	).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count active sla definitions: %w", err)
	}
	data.ActiveSlas = activeSlas

	activeAlertRules, err := r.client.SLAAlertRule.Query().Where(
		slaalertrule.TenantID(tenantID),
		slaalertrule.IsActive(true),
	).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count alert rules: %w", err)
	}
	data.ActiveAlertRules = activeAlertRules

	// 告警一经触发就是“当前活跃”，因此只按窗口结束时刻截断，
	// 不用开始时间把更早触发但仍未解决的告警藏起来。
	alertPredicates := []predicate.SLAAlertHistory{
		slaalerthistory.TenantID(tenantID),
		slaalerthistory.ResolvedAtIsNil(),
		slaalerthistory.CreatedAtLTE(end),
	}
	activeAlerts, err := r.client.SLAAlertHistory.Query().Where(alertPredicates...).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count active alerts: %w", err)
	}
	data.ActiveAlerts = activeAlerts

	alerts, err := r.listSLAAlertItems(ctx, tenantID, alertPredicates, monitoringAlertListLimit)
	if err != nil {
		return nil, err
	}
	data.Alerts = alerts

	return data, nil
}

// listSLAAlertItems 把告警历史映射为大屏告警行，并补齐工单优先级与剩余时间。
func (r *EntRepository) listSLAAlertItems(
	ctx context.Context,
	tenantID int,
	alertPredicates []predicate.SLAAlertHistory,
	limit int,
) ([]*SLAAlertItem, error) {
	es, err := r.client.SLAAlertHistory.Query().
		Where(alertPredicates...).
		Order(ent.Desc(slaalerthistory.FieldCreatedAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("load active alerts: %w", err)
	}
	items := make([]*SLAAlertItem, 0, len(es))
	if len(es) == 0 {
		return items, nil
	}

	ticketIDs := make([]int, 0, len(es))
	seen := make(map[int]bool, len(es))
	for _, e := range es {
		if !seen[e.TicketID] {
			seen[e.TicketID] = true
			ticketIDs = append(ticketIDs, e.TicketID)
		}
	}
	// 只加载本租户的工单列，跨租户的脏数据不得泄露优先级与截止时间。
	tks, err := r.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.IDIn(ticketIDs...)).
		Select(ticket.FieldID, ticket.FieldPriority, ticket.FieldSLAResolutionDeadline).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("load alert tickets: %w", err)
	}
	ticketByID := make(map[int]*ent.Ticket, len(tks))
	for _, t := range tks {
		ticketByID[t.ID] = t
	}

	now := time.Now()
	for _, e := range es {
		item := &SLAAlertItem{
			ID:            e.ID,
			TicketID:      e.TicketID,
			TicketNumber:  e.TicketNumber,
			TicketTitle:   e.TicketTitle,
			AlertLevel:    e.AlertLevel,
			AlertRuleName: e.AlertRuleName,
			ThresholdPct:  e.ThresholdPercentage,
			ActualPct:     e.ActualPercentage,
			CreatedAt:     e.CreatedAt,
			Priority:      "",
		}
		if t, ok := ticketByID[e.TicketID]; ok {
			item.Priority = t.Priority
			if !t.SLAResolutionDeadline.IsZero() {
				item.TimeRemaining = &SLATimeRemaining{
					Hours:    math.Round(t.SLAResolutionDeadline.Sub(now).Hours()*10) / 10,
					Deadline: t.SLAResolutionDeadline.UTC().Format(time.RFC3339),
				}
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// performanceKeyOf 计算工单在指定维度下的分组 key。
// 未绑定 SLA 定义或未填写服务类型的工单归入 unassigned，禁止静默丢弃。
func performanceKeyOf(dimension string, t slaCohortTicket, serviceTypes map[int]string) string {
	if dimension == SLADimensionPriority {
		if strings.TrimSpace(t.priority) == "" {
			return common.PriorityMedium
		}
		return t.priority
	}
	if t.slaDefinitionID == 0 {
		return SLAPerformanceUnassignedKey
	}
	if st := strings.TrimSpace(serviceTypes[t.slaDefinitionID]); st != "" {
		return st
	}
	return SLAPerformanceUnassignedKey
}

// ListSLAPerformance 按维度聚合窗口内工单绩效，返回按工单数降序、key 升序排列的行。
func (r *EntRepository) ListSLAPerformance(
	ctx context.Context,
	tenantID int,
	dimension string,
	start, end time.Time,
	serviceTypeFilter, priorityFilter string,
) ([]*SLAPerformanceRow, bool, error) {
	cohort, err := r.loadSLACohort(ctx, tenantID, start, end, serviceTypeFilter, priorityFilter)
	if err != nil {
		return nil, false, err
	}

	now := time.Now()
	grouped := make(map[string]*slaAggregation, len(cohort.tickets))
	for _, t := range cohort.tickets {
		key := performanceKeyOf(dimension, t, cohort.serviceTypes)
		agg, ok := grouped[key]
		if !ok {
			agg = &slaAggregation{}
			grouped[key] = agg
		}
		agg.add(t, cohort.violated[t.id], now)
	}

	rows := make([]*SLAPerformanceRow, 0, len(grouped))
	for key, agg := range grouped {
		rows = append(rows, &SLAPerformanceRow{
			Key:                       key,
			TotalTickets:              agg.total,
			ResolvedTickets:           agg.resolved,
			ResolutionRate:            ratePercent(agg.resolved, agg.total),
			ViolatedTickets:           agg.violated,
			MetSlaTickets:             agg.total - agg.violated,
			ComplianceRate:            ratePercent(agg.total-agg.violated, agg.total),
			ResponseSamples:           agg.respSamples,
			ResponseAchievementRate:   ratePercent(agg.respMet, agg.respSamples),
			ResolutionSamples:         agg.resSamples,
			ResolutionAchievementRate: ratePercent(agg.resMet, agg.resSamples),
			AverageResponseMinutes:    averageMinutes(agg.respDur, agg.respDurN),
			AverageResolutionMinutes:  averageMinutes(agg.resDur, agg.resDurN),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TotalTickets != rows[j].TotalTickets {
			return rows[i].TotalTickets > rows[j].TotalTickets
		}
		return rows[i].Key < rows[j].Key
	})

	return rows, cohort.truncated, nil
}

// GetComplianceReportData retrieves data for SLA compliance report
func (r *EntRepository) GetComplianceReportData(ctx context.Context, tenantID int, startDate, endDate time.Time) (totalTickets, metSLA, violatedSLA int, avgResponseTime, avgResolutionTime float64, err error) {
	// 1. 统计「窗口内创建」的工单，作为合规率统计的统一口径种群。
	//    总数与达标数必须基于同一批工单，否则会出现负数合规率（详见 R-003）。
	cohort, err := r.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.CreatedAtGTE(startDate), ticket.CreatedAtLTE(endDate)).
		Select(ticket.FieldID).
		All(ctx)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	totalTickets = len(cohort)
	if totalTickets == 0 {
		return 0, 0, 0, 0, 0, nil
	}
	cohortIDs := make([]int, 0, totalTickets)
	for _, t := range cohort {
		cohortIDs = append(cohortIDs, t.ID)
	}

	// 2. 统计同一批（窗口内创建）工单中、在窗口内发生未解决 SLA 违约的数量。
	//    通过 TicketIDIn(cohortIDs) 将违约统计限定在上述种群内，
	//    保证 violatedSLA <= totalTickets，从根本上消除负数合规率。
	var groups []struct {
		TicketID int `json:"ticket_id"` // json tag matches SQL column for ent GroupBy().Scan()
	}
	scanErr := r.client.SLAViolation.Query().
		Where(
			slaviolation.TenantID(tenantID),
			slaviolation.IsResolved(false),
			slaviolation.TicketIDIn(cohortIDs...),
			slaviolation.ViolationTimeGTE(startDate),
			slaviolation.ViolationTimeLTE(endDate),
		).
		GroupBy(slaviolation.FieldTicketID).
		Scan(ctx, &groups)
	if scanErr != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("compliance scan query failed: %w", scanErr)
	}
	violatedSLA = len(groups)
	metSLA = totalTickets - violatedSLA

	// 3. Average response time (minutes) for tickets with first_response_at
	responseRows, respErr := r.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.CreatedAtGTE(startDate), ticket.CreatedAtLTE(endDate)).
		Select(ticket.FieldCreatedAt, ticket.FieldFirstResponseAt).
		All(ctx)
	if respErr == nil {
		var totalDur time.Duration
		count := 0
		for _, t := range responseRows {
			if !t.CreatedAt.IsZero() && !t.FirstResponseAt.IsZero() {
				totalDur += t.FirstResponseAt.Sub(t.CreatedAt)
				count++
			}
		}
		if count > 0 {
			avgResponseTime = float64(totalDur) / float64(time.Minute) / float64(count)
		}
	}

	// 4. Average resolution time (minutes) for tickets with resolved_at
	resolutionRows, resErr := r.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.CreatedAtGTE(startDate), ticket.CreatedAtLTE(endDate)).
		Select(ticket.FieldCreatedAt, ticket.FieldResolvedAt).
		All(ctx)
	if resErr == nil {
		var totalDur time.Duration
		count := 0
		for _, t := range resolutionRows {
			if !t.CreatedAt.IsZero() && !t.ResolvedAt.IsZero() {
				totalDur += t.ResolvedAt.Sub(t.CreatedAt)
				count++
			}
		}
		if count > 0 {
			avgResolutionTime = float64(totalDur) / float64(time.Minute) / float64(count)
		}
	}

	return totalTickets, metSLA, violatedSLA, avgResponseTime, avgResolutionTime, nil
}

// GetTicketStats retrieves basic ticket statistics for compliance calculation
func (r *EntRepository) GetTicketStats(ctx context.Context, tenantID int) (total int, metSLA int, err error) {
	// 获取总工单数
	total, err = r.client.Ticket.Query().
		Where(ticket.TenantID(tenantID)).
		Count(ctx)
	if err != nil {
		return 0, 0, err
	}

	// 获取有SLA违规的工单数（被视为未达标）
	violatedTickets, err := r.client.SLAViolation.Query().
		Where(slaviolation.TenantID(tenantID)).
		Select(slaviolation.FieldTicketID).
		All(ctx)
	if err != nil {
		return total, 0, err
	}

	// 使用违规工单的独特ID数量
	uniqueViolated := make(map[int]bool)
	for _, v := range violatedTickets {
		uniqueViolated[v.TicketID] = true
	}

	metSLA = total - len(uniqueViolated)
	if metSLA < 0 {
		metSLA = 0
	}

	return total, metSLA, nil
}

// GetTicketSLA retrieves per-ticket SLA timing fields used by check-compliance.
// P1-07 修复：返回创建的首次响应 / 解决时间点，让 Service 能计算 actual_response_minutes。
// 修复：同时返回 SLA 截止时间，让 Service 判断合规时对比 deadline 而非仅检查是否有响应。
func (r *EntRepository) GetTicketSLA(ctx context.Context, ticketID int, tenantID int) (createdAt, firstResponseAt, resolvedAt, slaResponseDeadline, slaResolutionDeadline time.Time, found bool, err error) {
	t, err := r.client.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantID(tenantID)).
		Select(ticket.FieldCreatedAt, ticket.FieldFirstResponseAt, ticket.FieldResolvedAt,
			ticket.FieldSLAResponseDeadline, ticket.FieldSLAResolutionDeadline).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}, false, nil
		}
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}, false, err
	}
	return t.CreatedAt, t.FirstResponseAt, t.ResolvedAt, t.SLAResponseDeadline, t.SLAResolutionDeadline, true, nil
}
