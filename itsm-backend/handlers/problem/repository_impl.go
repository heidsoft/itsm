package problem

import (
	"context"
	"fmt"
	"sort"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/incident"
	entpredicate "itsm-backend/ent/predicate"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/user"
	"itsm-backend/handlers/common/datascope"
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
		ID:            e.ID,
		ProblemNumber: e.ProblemNumber,
		Title:         e.Title,
		Description:   e.Description,
		Status:        e.Status,
		Priority:      e.Priority,
		Category:      e.Category,
		RootCause:     e.RootCause,
		Workaround:    e.Workaround,
		Resolution:    e.Resolution,
		Impact:        e.Impact,
		CreatedBy:     e.CreatedBy,
		TenantID:      e.TenantID,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
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
				Status: string(ch.Status),
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
		SetWorkaround(p.Workaround).
		SetResolution(p.Resolution).
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
	// 生成问题编号（PRB-YYYYMMDD-XXXX，租户内日序列）
	number := fmt.Sprintf("PRB-%s-%04d", time.Now().Format("20060102"), saved.ID)
	if _, err := r.client.Problem.UpdateOneID(saved.ID).SetProblemNumber(number).Save(ctx); err != nil {
		// 编号写失败不阻断创建（可后台补偿），仅记录
		return r.toDomain(saved), nil
	}
	saved.ProblemNumber = number
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

func (r *EntRepository) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, dataScope datascope.DataScope, currentUserID int) ([]*Problem, int, error) {
	// 分页参数由 HTTP 入口的 common.GetPaginationFromQuery 夹紧后再传下来，这里不再写第二套
	// 规则——原来 repo 自己按 page<1→1、size<1→10、size>200→200 夹紧，而响应用未夹紧的原值
	// 回显 pageSize，两侧对同一个请求给两个答案，翻页会整段跳过数据。
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

	// 行级数据权限（推广自 ticket DataScope 模式）：
	// OwnedOrAssigned 时强制追加 Or(CreatedByEQ(uid), AssigneeIDEQ(uid))，
	// 使普通用户只能看到自己创建或分配给自己的问题单。
	// CurrentUserID<=0 时 fail-closed，返回空集而非全量。
	if dataScope == datascope.DataScopeOwnedOrAssigned {
		if currentUserID <= 0 {
			query = query.Where(problem.IDEQ(-1))
		} else {
			query = query.Where(problem.Or(
				problem.CreatedByEQ(currentUserID),
				problem.AssigneeIDEQ(currentUserID),
			))
		}
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// created_at 不是唯一列：同一秒创建的问题在页边界的归属不确定，用 ID 兜底成全序。
	list, err := query.
		Offset((page-1)*size).
		Limit(size).
		Order(ent.Desc(problem.FieldCreatedAt), ent.Asc(problem.FieldID)).
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
		SetWorkaround(p.Workaround).
		SetResolution(p.Resolution).
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

// GetAllForAnalytics 拉取 [start, end) 区间内的问题单用于趋势/热点统计。
// P1 修复：此前只用 start 做下界（CreatedAtGTE），导致区间之后的全量数据被计入，
// 月度趋势与计数失真。这里同时约束上界 CreatedAtLT(end)，区间外数据不再混入。
func (r *EntRepository) GetAllForAnalytics(ctx context.Context, tenantID int, start, end time.Time) ([]*Problem, error) {
	list, err := r.client.Problem.Query().
		Where(problem.TenantIDEQ(tenantID), problem.DeletedAtIsNil(), problem.CreatedAtGTE(start), problem.CreatedAtLT(end)).
		Order(ent.Desc(problem.FieldCreatedAt)).
		Limit(1000).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*Problem, 0, len(list))
	for _, item := range list {
		result = append(result, r.toDomain(item))
	}
	return result, nil
}

func (r *EntRepository) GetStats(ctx context.Context, tenantID int) (*ProblemStats, error) {
	base := []entpredicate.Problem{problem.TenantIDEQ(tenantID), problem.DeletedAtIsNil()}

	total, err := r.client.Problem.Query().Where(base...).Count(ctx)
	if err != nil {
		return nil, err
	}

	statusRows := []statusCountRow{}
	if err := r.client.Problem.Query().
		Where(base...).
		GroupBy(problem.FieldStatus).
		Aggregate(ent.Count()).
		Scan(ctx, &statusRows); err != nil {
		return nil, err
	}

	priorityRows := []priorityCountRow{}
	if err := r.client.Problem.Query().
		Where(base...).
		GroupBy(problem.FieldPriority).
		Aggregate(ent.Count()).
		Scan(ctx, &priorityRows); err != nil {
		return nil, err
	}

	stats := &ProblemStats{
		Total:      total,
		ByStatus:   orderStatusCounts(statusRows),
		ByPriority: orderPriorityCounts(priorityRows),
	}
	// 折叠口径保持与修复前一致，报表才不会出现读数突变：
	// open 单独一桶，investigating 与 in_progress 并进 InProgress，
	// identified 历史上不属于任何单值桶（因此单值之和小于 total），
	// 但它在 ByStatus 里必须可见，否则问题被静默漏计。
	for _, row := range statusRows {
		switch row.Status {
		case "open":
			stats.Open += row.Count
		case "investigating", "in_progress":
			stats.InProgress += row.Count
		case "resolved":
			stats.Resolved += row.Count
		case "closed":
			stats.Closed += row.Count
		}
	}
	for _, row := range priorityRows {
		if row.Priority == "high" || row.Priority == "critical" {
			stats.HighPriority += row.Count
		}
	}
	return stats, nil
}

// statusCountRow 是 GroupBy(status) Scan 的目标结构。
type statusCountRow struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// priorityCountRow 是 GroupBy(priority) Scan 的目标结构。
type priorityCountRow struct {
	Priority string `json:"priority"`
	Count    int    `json:"count"`
}

// 分布的展示基准顺序，与后端问题状态机/优先级词表一致。
// ent 的 status/priority 是无强校验字符串字段，词表外的历史值按字典序追加而不是丢弃，
// 这样报表总数能与 total 对账，且输出顺序确定、不会抖动。
var (
	problemStatusOrder   = []string{"open", "investigating", "identified", "in_progress", "resolved", "closed"}
	problemPriorityOrder = []string{"low", "medium", "high", "critical"}
)

func orderStatusCounts(rows []statusCountRow) []StatusCount {
	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.Status] += row.Count
	}
	out := []StatusCount{}
	for _, name := range orderedVocabulary(counts, problemStatusOrder) {
		out = append(out, StatusCount{Status: name, Count: counts[name]})
	}
	return out
}

func orderPriorityCounts(rows []priorityCountRow) []PriorityCount {
	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.Priority] += row.Count
	}
	out := []PriorityCount{}
	for _, name := range orderedVocabulary(counts, problemPriorityOrder) {
		out = append(out, PriorityCount{Priority: name, Count: counts[name]})
	}
	return out
}

// orderedVocabulary 只保留真实存在的取值（count>0），先按词表基准顺序，再按字典序追加词表外值。
func orderedVocabulary(counts map[string]int, order []string) []string {
	out := make([]string, 0, len(counts))
	inOrder := make(map[string]bool, len(order))
	for _, name := range order {
		inOrder[name] = true
		if counts[name] > 0 {
			out = append(out, name)
		}
	}
	extra := make([]string, 0, len(counts))
	for name, count := range counts {
		if count > 0 && !inOrder[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}

// LoadUserNames 批量加载 user 显示名（id -> name）。当前租户范围，name 为空回退 username。
// 失败返回部分映射 + error，由调用方决定是否降级（一般仅记日志不阻断主流程）。
func (r *EntRepository) LoadUserNames(ctx context.Context, tenantID int, ids []int) (map[int]string, error) {
	out := map[int]string{}
	if len(ids) == 0 {
		return out, nil
	}
	seen := map[int]struct{}{}
	uniq := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return out, nil
	}
	users, err := r.client.User.Query().
		Where(user.TenantIDEQ(tenantID), user.IDIn(uniq...)).
		Select(user.FieldID, user.FieldName, user.FieldUsername).
		All(ctx)
	if err != nil {
		return out, err
	}
	for _, u := range users {
		name := u.Name
		if name == "" {
			name = u.Username
		}
		if name != "" {
			out[u.ID] = name
		}
	}
	return out, nil
}
