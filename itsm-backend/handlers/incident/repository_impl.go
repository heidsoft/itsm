package incident

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/database"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/incidentevent"
	"itsm-backend/ent/incidentrule"
	"itsm-backend/ent/slaviolation"
	"itsm-backend/ent/user"
	"itsm-backend/handlers/common/datascope"
)

type EntRepository struct {
	client *ent.Client
	stats  *incidentStatsRepository
	number *incidentNumberRepository
}

func NewEntRepository(client *ent.Client) *EntRepository {
	// 共享 *sql.DB：用于 stats 与 number 两个 PG 专用聚合/序列调用。
	// 经由 database.GetRawDB() 取值，便于 bootstrap 不传第二个参数；
	// 该 DB 仅在本仓库承担 PostgreSQL 序列与 FILTER 聚合表达。
	return &EntRepository{
		client: client,
		stats:  newIncidentStatsRepository(database.GetRawDB()),
		number: newIncidentNumberRepository(database.GetRawDB()),
	}
}

// optionalTime 保留 NULL 语义：ent 的 Optional 非 Nillable 时间字段把 DB NULL
// 读成零值，直接取地址会伪造出「指向 0001-01-01 的非 nil 指针」，写路径据 != nil
// 判断再把零值落库，从而不可逆地污染 resolved_at / closed_at。
func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// optionalID 同理保留「未设置」语义，避免 NULL 外键被读成指向 0 的指针，
// 使「未分配」与「分配给用户 0」不可区分。
func optionalID(id int) *int {
	if id <= 0 {
		return nil
	}
	return &id
}

// toDomain converts ent.Incident to domain Incident
func (r *EntRepository) toDomain(e *ent.Incident) *Incident {
	if e == nil {
		return nil
	}
	return &Incident{
		ID:                    e.ID,
		Title:                 e.Title,
		Description:           e.Description,
		Status:                e.Status,
		Priority:              e.Priority,
		Severity:              e.Severity,
		Impact:                e.Impact,
		Urgency:               e.Urgency,
		IncidentNumber:        e.IncidentNumber,
		ReporterID:            e.ReporterID,
		AssigneeID:            optionalID(e.AssigneeID),
		ConfigurationItemID:   optionalID(e.ConfigurationItemID),
		Version:               e.Version,
		IsMajorIncident:       e.IsMajorIncident,
		Category:              e.Category,
		Subcategory:           e.Subcategory,
		ImpactAnalysis:        e.ImpactAnalysis,
		RootCause:             e.RootCause,
		ResolutionSteps:       e.ResolutionSteps,
		Metadata:              e.Metadata,
		DetectedAt:            e.DetectedAt,
		ResolvedAt:            optionalTime(e.ResolvedAt),
		SLADefinitionID:       optionalID(e.SLADefinitionID),
		SLAResponseDeadline:   optionalTime(e.SLAResponseDeadline),
		SLAResolutionDeadline: optionalTime(e.SLAResolutionDeadline),
		SLAFirstResponseAt:    optionalTime(e.SLAFirstResponseAt),
		SLAResolvedAt:         optionalTime(e.SLAResolvedAt),
		SLAStatus:             e.SLAStatus,
		SLAPausedAt:           optionalTime(e.SLAPausedAt),
		SLAPauseReason:        e.SLAPauseReason,
		ClosedAt:              optionalTime(e.ClosedAt),
		EscalatedAt:           optionalTime(e.EscalatedAt),
		EscalationLevel:       e.EscalationLevel,
		IsAutomated:           e.IsAutomated,
		Source:                e.Source,
		TenantID:              e.TenantID,
		CreatedAt:             e.CreatedAt,
		UpdatedAt:             e.UpdatedAt,
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
		SetIsMajorIncident(i.IsMajorIncident).
		SetTenantID(i.TenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now())

	if i.AssigneeID != nil {
		query.SetAssigneeID(*i.AssigneeID)
	}
	if i.ConfigurationItemID != nil {
		query.SetConfigurationItemID(*i.ConfigurationItemID)
	}
	// ent schema 对 impact/urgency 的 Validate 拒绝空串，留空时必须让 Default("medium")
	// 生效，因此仅在调用方显式给出值时写入。
	if i.Impact != "" {
		query.SetImpact(i.Impact)
	}
	if i.Urgency != "" {
		query.SetUrgency(i.Urgency)
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

func (r *EntRepository) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, dataScope datascope.DataScope, currentUserID int) ([]*Incident, int, error) {
	query := r.client.Incident.Query().Where(incident.TenantIDEQ(tenantID))

	if v, ok := filters["status"].(string); ok && v != "" {
		query = query.Where(incident.StatusEQ(v))
	}
	if v, ok := filters["priority"].(string); ok && v != "" {
		query = query.Where(incident.PriorityEQ(v))
	}
	if v, ok := filters["keyword"].(string); ok && v != "" {
		query = query.Where(incident.Or(
			incident.TitleContains(v),
			incident.DescriptionContains(v),
			incident.IncidentNumberContains(v),
		))
	}
	if v, ok := filters["source"].(string); ok && v != "" {
		query = query.Where(incident.Source(v))
	}
	if v, ok := filters["type"].(string); ok && v != "" {
		query = query.Where(incident.TypeEQ(v))
	}
	if v, ok := filters["category"].(string); ok && v != "" {
		query = query.Where(incident.Category(v))
	}
	// 布尔/整型只在 key 存在时过滤，避免把 false / 0 当成「不过滤」。
	if v, ok := filters["is_major_incident"].(bool); ok {
		query = query.Where(incident.IsMajorIncident(v))
	}
	if v, ok := filters["assignee_id"].(int); ok && v > 0 {
		query = query.Where(incident.AssigneeIDEQ(v))
	}
	if v, ok := filters["date_from"].(time.Time); ok && !v.IsZero() {
		query = query.Where(incident.CreatedAtGTE(v))
	}
	if v, ok := filters["date_to"].(time.Time); ok && !v.IsZero() {
		query = query.Where(incident.CreatedAtLTE(v))
	}

	// 行级数据权限（推广自 ticket DataScope 模式）：
	// OwnedOrAssigned 时强制追加 Or(ReporterIDEQ(uid), AssigneeIDEQ(uid))，
	// 使普通用户只能看到自己创建或分配给自己的事件单。
	// CurrentUserID<=0 时 fail-closed，返回空集而非全量。
	if dataScope == datascope.DataScopeOwnedOrAssigned {
		if currentUserID <= 0 {
			query = query.Where(incident.IDEQ(-1))
		} else {
			query = query.Where(incident.Or(
				incident.ReporterIDEQ(currentUserID),
				incident.AssigneeIDEQ(currentUserID),
			))
		}
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
	// P1-infra 修复：写路径强制租户隔离 + 软删守卫，避免越权更新与已软删记录被
	// 覆盖（此前缺 TenantIDEQ / DeletedAtIsNil，并发陈旧快照上状态机判定失效）。
	// P1-乐观锁修复：追加 VersionEQ 条件并 AddVersion(1)。此前是 read-then-
	// unconditional-write，UpdateIncidentRequest.Version 从未被消费，陈旧客户端可
	// 静默覆盖他人写入；且版本不自增，使 lifecycle 操作的 CAS 也失去意义。
	// impact/urgency/is_major_incident 不在此写回：ent schema 的 Validate 拒绝空串，
	// 且本包没有任何路径经 Update 修改它们（SetIsMajorIncident 在 legacy service 内
	// 直接操作 ent），原样回写只会引入校验失败风险。
	u := r.client.Incident.UpdateOneID(i.ID).
		Where(
			incident.TenantIDEQ(i.TenantID),
			incident.DeletedAtIsNil(),
			incident.VersionEQ(i.Version),
		).
		SetUpdatedAt(time.Now()).
		AddVersion(1).
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
		// 调用方已先经 Get 校验过存在性与租户归属，此处未命中只可能是版本被并发
		// 推进或记录在读写之间被软删，两者都属于陈旧快照冲突而非资源不存在。
		if ent.IsNotFound(err) {
			return nil, ErrStaleVersion
		}
		return nil, err
	}
	return r.toDomain(saved), nil
}

func (r *EntRepository) Delete(ctx context.Context, id int, tenantID int) error {
	// P0-2 修复：改为软删除，避免物理删除导致 CountByPeriod 计数回退、历史
	// 编号被复用，并使 incident_events / incident_metrics / problem↔incident
	// 关联成为孤儿。软删后读路径由全局拦截器自动过滤。
	return r.client.Incident.UpdateOneID(id).
		Where(incident.TenantIDEQ(tenantID)).
		SetDeletedAt(time.Now()).
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

// GetStats 单次聚合查询返回全量指标。
// 已封装到 incidentStatsRepository：保留 PostgreSQL FILTER 聚合与 EXTRACT(EPOCH)
// 语法以便单查询完成多指标统计，调用方不再接触 SQL。
func (r *EntRepository) GetStats(ctx context.Context, tenantID int) (*IncidentStats, error) {
	if r.stats == nil {
		return nil, fmt.Errorf("incident stats repository not initialised")
	}
	return r.stats.GetStats(ctx, tenantID)
}

func (r *EntRepository) GenerateIncidentNumber(ctx context.Context, tenantID int, year int, month int) (string, error) {
	// tenantID 入参保留以兼容 Repository 接口契约，但编号生成不依赖租户：
	// incident_number_seq 是全局序列，租户隔离由
	//   (tenant_id, incident_number) 唯一索引 与 service 层校验 兜底。
	if r.number == nil {
		return "", fmt.Errorf("incident number repository not initialised")
	}
	return r.number.Generate(ctx, year, month)
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

// GetIncidentWithEdges 按 id+tenant 加载 Incident 并带上指定 eager-loading 边。
// edges 取值："events" | "alerts" | "metrics"；未知值忽略。
func (r *EntRepository) GetIncidentWithEdges(ctx context.Context, id, tenantID int, edges ...string) (*ent.Incident, error) {
	query := r.client.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID))
	for _, e := range edges {
		switch e {
		case "events":
			query = query.WithIncidentEvents()
		case "alerts":
			query = query.WithIncidentAlerts()
		case "metrics":
			query = query.WithIncidentMetrics()
		}
	}
	return query.Only(ctx)
}

// ListIncidentComments 返回 event_type=comment 的事件（即事件评论）。
func (r *EntRepository) ListIncidentComments(ctx context.Context, incidentID, tenantID int) ([]*ent.IncidentEvent, error) {
	return r.client.IncidentEvent.Query().
		Where(
			incidentevent.IncidentIDEQ(incidentID),
			incidentevent.TenantIDEQ(tenantID),
			incidentevent.EventType("comment"),
		).
		WithIncident().
		All(ctx)
}

// CreateIncidentCommentEvent 写入一条评论型 IncidentEvent。
func (r *EntRepository) CreateIncidentCommentEvent(ctx context.Context, event *ent.IncidentEvent) (*ent.IncidentEvent, error) {
	return r.client.IncidentEvent.Create().
		SetIncidentID(event.IncidentID).
		SetEventType(event.EventType).
		SetEventName(event.EventName).
		SetDescription(event.Description).
		SetStatus(event.Status).
		SetUserID(event.UserID).
		SetSource(event.Source).
		SetData(event.Data).
		SetTenantID(event.TenantID).
		SetOccurredAt(event.OccurredAt).
		Save(ctx)
}

// CountTenantSLAViolations 统计租户级 SLA 违规数（供指标接口）。
func (r *EntRepository) CountTenantSLAViolations(ctx context.Context, tenantID int) (int, error) {
	return r.client.SLAViolation.Query().
		Where(slaviolation.TenantIDEQ(tenantID)).
		Count(ctx)
}

// GetUserNamesByIDs 批量查询用户姓名（id → name）。租户隔离 + IN 一次查询。
// 查不到的用户不进 map（前端回退显示 ID）。
func (r *EntRepository) GetUserNamesByIDs(ctx context.Context, tenantID int, ids []int) (map[int]string, error) {
	names := make(map[int]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}
	rows, err := r.client.User.Query().
		Where(
			user.TenantIDEQ(tenantID),
			user.IDIn(ids...),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("batch query user names: %w", err)
	}
	for _, u := range rows {
		names[u.ID] = u.Name
	}
	return names, nil
}
