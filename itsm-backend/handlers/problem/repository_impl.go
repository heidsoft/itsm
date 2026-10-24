package problem

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/incident"
	entpredicate "itsm-backend/ent/predicate"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/problemcomment"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/user"
	"itsm-backend/internal/datascope"
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
	// 编号与主体行必须同生共死。旧实现是「先插入、再单独 UPDATE 编号」，且
	// UPDATE 失败被静默吞掉（注释声称可后台补偿，但代码库里没有补偿任务），
	// 结果是留下一条没有编号的问题单：列表/详情编号为空白，按编号检索永远
	// 命中不到，而调用方拿到的是 200。改为同一事务内完成。
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(cause error) (*Problem, error) {
		if rbErr := tx.Rollback(); rbErr != nil {
			return nil, fmt.Errorf("%w; rollback: %v", cause, rbErr)
		}
		return nil, cause
	}

	create := tx.Problem.Create().
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
		return rollback(err)
	}
	// 生成问题编号（PRB-YYYYMMDD-XXXX，租户内日序列）
	number := fmt.Sprintf("PRB-%s-%04d", time.Now().Format("20060102"), saved.ID)
	if _, err := tx.Problem.UpdateOneID(saved.ID).SetProblemNumber(number).Save(ctx); err != nil {
		return rollback(fmt.Errorf("写入问题编号失败: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return rollback(err)
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

func (r *EntRepository) Update(ctx context.Context, p *Problem, expectedStatus string) (*Problem, error) {
	return r.UpdateTx(ctx, r.client, p, expectedStatus)
}

// UpdateTx 与 Update 同语义，但写入走调用方事务绑定的 Ent client，
// 供「业务写入 + BPMN 桥接」组成同一事务（问题域生命周期原子推进）。
// 不创建/提交/回滚事务，最终结果由事务所有者决定。
func (r *EntRepository) UpdateTx(ctx context.Context, txc *ent.Client, p *Problem, expectedStatus string) (*Problem, error) {
	// CAS：problem 无 version 列，用读取时状态锚定校验-写入并发窗口。
	// 注意区分两种 NotFound 成因：工件真的不存在/已软删（透传原始错误，
	// service 映射 404）vs 工件还在但状态已被并发推进（类型化冲突，409）。
	update := txc.Problem.UpdateOneID(p.ID).
		Where(problem.TenantIDEQ(p.TenantID), problem.DeletedAtIsNil(), problem.StatusEQ(expectedStatus)).
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
		if ent.IsNotFound(err) {
			// 区分成因：仍能按 id+tenant 读到 → 状态被并发推进（CAS 冲突）；
			// 读不到 → 原样透传（service 按 not-found 语义处理）。
			if _, lerr := r.client.Problem.Query().
				Where(problem.IDEQ(p.ID), problem.TenantIDEQ(p.TenantID)).
				First(ctx); lerr == nil {
				return nil, fmt.Errorf("%w: 期望状态 '%s'", ErrProblemStatusMoved, expectedStatus)
			}
			return nil, err
		}
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
// 只用 start 做下界会把区间之后的全量数据计入，月度趋势与计数失真，
// 因此上界 CreatedAtLT(end) 必须同时约束。
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
	// 折叠口径（改动会造成报表读数突变）：
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

// ─────────────────────────────────────────────────────────────────────────────
// 评论（2026-10-21 GA 阻断项落地）
//
// 隔离口径：评论表自带 tenant_id，但 problem_id 是跨租户可猜的外键——
// 写入/读取前一律用 JOIN problems 校验问题归属，防止别家租户的 problem_id
// 被用来写评论或探测评论存在性（跨租户统一表现为 not-found，不暴露 ID 可枚举信号）。
// ─────────────────────────────────────────────────────────────────────────────

// problemVisible 校验问题存在、未删除且属于该租户。
func (r *EntRepository) problemVisible(ctx context.Context, tenantID, problemID int) (bool, error) {
	if problemID <= 0 {
		return false, nil
	}
	return r.client.Problem.Query().
		Where(problem.ID(problemID), problem.TenantID(tenantID), problem.DeletedAtIsNil()).
		Exist(ctx)
}

func (r *EntRepository) commentToDomain(e *ent.ProblemComment) *ProblemComment {
	if e == nil {
		return nil
	}
	c := &ProblemComment{
		ID:         e.ID,
		ProblemID:  e.ProblemID,
		UserID:     e.UserID,
		Content:    e.Content,
		IsInternal: e.IsInternal,
		TenantID:   e.TenantID,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
	if len(e.Mentions) > 0 {
		c.Mentions = append([]int(nil), e.Mentions...)
	}
	if len(e.Attachments) > 0 {
		c.Attachments = append([]int(nil), e.Attachments...)
	}
	return c
}

func (r *EntRepository) CreateComment(ctx context.Context, c *ProblemComment) (*ProblemComment, error) {
	if strings.TrimSpace(c.Content) == "" {
		return nil, fmt.Errorf("comment content is required")
	}
	visible, err := r.problemVisible(ctx, c.TenantID, c.ProblemID)
	if err != nil {
		return nil, err
	}
	if !visible {
		// 跨租户/不存在/已删除统一 not-found：不暴露问题 ID 可枚举信号。
		return nil, ErrProblemNotVisible
	}

	create := r.client.ProblemComment.Create().
		SetProblemID(c.ProblemID).
		SetUserID(c.UserID).
		SetContent(strings.TrimSpace(c.Content)).
		SetIsInternal(c.IsInternal).
		SetTenantID(c.TenantID)
	if len(c.Mentions) > 0 {
		create.SetMentions(c.Mentions)
	}
	if len(c.Attachments) > 0 {
		create.SetAttachments(c.Attachments)
	}
	saved, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	out := r.commentToDomain(saved)
	// 创建响应也投影显示名：前端拿到即可直接渲染，不必再查通讯录。
	// 投影失败不阻断创建（响应缺 userName 时前端回退 `用户 #id`）。
	if names, nerr := r.LoadUserNames(ctx, c.TenantID, []int{c.UserID}); nerr == nil {
		if name, ok := names[c.UserID]; ok {
			out.UserName = name
		}
	}
	return out, nil
}

func (r *EntRepository) ListComments(ctx context.Context, tenantID, problemID int, page, size int) ([]*ProblemComment, int, error) {
	visible, err := r.problemVisible(ctx, tenantID, problemID)
	if err != nil {
		return nil, 0, err
	}
	if !visible {
		return nil, 0, ErrProblemNotVisible
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	q := r.client.ProblemComment.Query().
		Where(problemcomment.TenantID(tenantID), problemcomment.ProblemID(problemID))

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*ProblemComment{}, 0, nil
	}

	rows, err := q.
		Order(ent.Asc(problemcomment.FieldCreatedAt), ent.Asc(problemcomment.FieldID)).
		Offset((page - 1) * size).
		Limit(size).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	out := make([]*ProblemComment, 0, len(rows))
	ids := make([]int, 0, len(rows))
	for _, e := range rows {
		out = append(out, r.commentToDomain(e))
		ids = append(ids, e.UserID)
	}
	// 批量投影评论人显示名：查询失败不阻断列表（前端回退 `用户 #id`）。
	if names, nerr := r.LoadUserNames(ctx, tenantID, ids); nerr == nil {
		for _, c := range out {
			if name, ok := names[c.UserID]; ok {
				c.UserName = name
			}
		}
	}
	return out, total, nil
}
