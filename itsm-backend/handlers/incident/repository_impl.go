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
		ID:                    e.ID,
		Title:                 e.Title,
		Description:           e.Description,
		Status:                e.Status,
		Priority:              e.Priority,
		Severity:              e.Severity,
		IncidentNumber:        e.IncidentNumber,
		ReporterID:            e.ReporterID,
		AssigneeID:            &e.AssigneeID,
		ConfigurationItemID:   &e.ConfigurationItemID,
		Category:              e.Category,
		Subcategory:           e.Subcategory,
		ImpactAnalysis:        e.ImpactAnalysis,
		RootCause:             e.RootCause,
		ResolutionSteps:       e.ResolutionSteps,
		Metadata:              e.Metadata,
		DetectedAt:            e.DetectedAt,
		ResolvedAt:            &e.ResolvedAt,
		SLADefinitionID:       &e.SLADefinitionID,
		SLAResponseDeadline:   &e.SLAResponseDeadline,
		SLAResolutionDeadline: &e.SLAResolutionDeadline,
		SLAFirstResponseAt:    &e.SLAFirstResponseAt,
		SLAResolvedAt:         &e.SLAResolvedAt,
		SLAStatus:             e.SLAStatus,
		SLAPausedAt:           &e.SLAPausedAt,
		SLAPauseReason:        e.SLAPauseReason,
		ClosedAt:              &e.ClosedAt,
		EscalatedAt:           &e.EscalatedAt,
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
	u := r.client.Incident.UpdateOneID(i.ID).
		Where(
			incident.TenantIDEQ(i.TenantID),
			incident.DeletedAtIsNil(),
		).
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
// 租户隔离：所有 SQL 均绑定 tenantID = $1，避免跨租户泄露。
// 指标说明：
//   - total/open/critical/major/resolved: COUNT(*) FILTER 一次完成
//   - avgResolutionTime: 已解决/已关闭事件的平均 (resolved_at - created_at) 分钟数；
//     未解决事件排除；空集通过 COALESCE 返回 0 而非 NULL。
func (r *EntRepository) GetStats(ctx context.Context, tenantID int) (*IncidentStats, error) {
	stats := &IncidentStats{}

	row := database.GetRawDB().QueryRowContext(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE TRUE) AS total,
		  COUNT(*) FILTER (WHERE status IN ('open','in_progress')) AS open,
		  COUNT(*) FILTER (WHERE priority = 'critical') AS critical,
		  COUNT(*) FILTER (WHERE priority = 'high') AS major,
		  COUNT(*) FILTER (WHERE status IN ('resolved','closed')) AS resolved,
		  COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60.0)
		           FILTER (WHERE status IN ('resolved','closed') AND resolved_at IS NOT NULL),
		           0)::int AS avg_minutes
		FROM incidents
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`, tenantID)

	if err := row.Scan(
		&stats.TotalIncidents,
		&stats.OpenIncidents,
		&stats.CriticalIncidents,
		&stats.MajorIncidents,
		&stats.ResolvedIncidents,
		&stats.AvgResolutionTime,
	); err != nil {
		return nil, fmt.Errorf("scan incident stats: %w", err)
	}

	return stats, nil
}

func (r *EntRepository) GenerateIncidentNumber(ctx context.Context, tenantID int, year int, month int) (string, error) {
	// P0-2 修复：编号改为由数据库序列 incident_number_seq 原子生成，彻底消除
	// 「COUNT+1 并发复用 / 删除回退 / UTC 窗口错乱」三类编号冲突。
	//   1) 序列单调递增且全局唯一，后缀永不重复，跨月也不会与历史编号碰撞；
	//   2) 前缀 INC-YYYYMM 使用调用方传入的本地年月（service.go 用 time.Now()
	//      本地时区），与统计窗口口径一致，不再出现 UTC 空窗 INC-...-000001；
	//   3) (tenant_id, incident_number) 唯一索引作为最后兜底，即便异常也不会
	//      落库重复编号。
	// 序列在 database.InitDatabase 中创建并播种为「历史最大后缀+1」。
	var seq int64
	if err := database.GetRawDB().QueryRowContext(ctx,
		"SELECT nextval('incident_number_seq')",
	).Scan(&seq); err != nil {
		return "", fmt.Errorf("generate incident number: %w", err)
	}
	return fmt.Sprintf("INC-%04d%02d-%06d", year, month, seq), nil
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
