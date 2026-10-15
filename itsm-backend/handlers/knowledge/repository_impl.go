package knowledge

import (
	"context"
	"sort"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/knowledgearticlereview"
	"itsm-backend/ent/knowledgearticleversion"
	"itsm-backend/ent/user"
)

// defaultCategories provides baseline knowledge-base categories so the category
// dropdown is never empty for a freshly provisioned tenant. The list is always
// merged with whatever distinct categories already exist in the article table,
// so adding more categories later does not lose them.
var defaultCategories = []string{
	"故障排查",
	"操作指南",
	"最佳实践",
	"变更发布",
	"安全合规",
	"常见问题",
	"工单模板",
	"流程说明",
}

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

// toDomain maps ent KnowledgeArticle to domain Article
func toDomain(e *ent.KnowledgeArticle) *Article {
	if e == nil {
		return nil
	}
	tags := []string{}
	if e.Tags != "" {
		tags = strings.Split(e.Tags, ",")
	}
	return &Article{
		ID:          e.ID,
		Title:       e.Title,
		Content:     e.Content,
		Category:    e.Category,
		Tags:        tags,
		AuthorID:    e.AuthorID,
		TenantID:    e.TenantID,
		IsPublished: e.IsPublished,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,

		// 评审状态机字段。ReviewStatus 经 NormalizeReviewStatus 归一，
		// 保证前端拿到的永远是四个合法值之一（存量未回填时为 draft）。
		ReviewStatus:  NormalizeReviewStatus(string(e.ReviewStatus)),
		ReviewerID:    e.ReviewerID,
		ReviewedAt:    e.ReviewedAt,
		ReviewComment: e.ReviewComment,

		ValidFrom:          e.ValidFrom,
		ValidUntil:         e.ValidUntil,
		LastReviewedAt:     e.LastReviewedAt,
		ReviewIntervalDays: e.ReviewIntervalDays,
		AuthorityLevel:     e.AuthorityLevel,
	}
}

func (r *EntRepository) Create(ctx context.Context, a *Article) (*Article, error) {
	tagsStr := strings.Join(a.Tags, ",")
	e, err := r.client.KnowledgeArticle.Create().
		SetTitle(a.Title).
		SetContent(a.Content).
		SetCategory(a.Category).
		SetTags(tagsStr).
		SetAuthorID(a.AuthorID).
		SetTenantID(a.TenantID).
		SetIsPublished(a.IsPublished).
		// 时效性（L1）与权威性（L2）：三个时间字段用 Nillable setter，
		// nil 表示不设时效/不设复核，与数据库默认值语义一致。
		SetNillableValidFrom(a.ValidFrom).
		SetNillableValidUntil(a.ValidUntil).
		SetNillableLastReviewedAt(a.LastReviewedAt).
		SetReviewIntervalDays(a.ReviewIntervalDays).
		SetAuthorityLevel(a.AuthorityLevel).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(e), nil
}

func (r *EntRepository) Get(ctx context.Context, id int, tenantID int) (*Article, error) {
	e, err := r.client.KnowledgeArticle.Query().
		Where(knowledgearticle.ID(id), knowledgearticle.TenantID(tenantID), knowledgearticle.DeletedAtIsNil()).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(e), nil
}

func (r *EntRepository) List(ctx context.Context, tenantID int, page, size int, category, search, status string) ([]*Article, int, error) {
	q := r.client.KnowledgeArticle.Query().Where(knowledgearticle.TenantID(tenantID), knowledgearticle.DeletedAtIsNil())

	if category != "" {
		q = q.Where(knowledgearticle.Category(category))
	}
	if search != "" {
		q = q.Where(
			knowledgearticle.Or(
				knowledgearticle.TitleContains(search),
				knowledgearticle.ContentContains(search),
			),
		)
	}
	if status != "" {
		if strings.ToLower(status) == "published" {
			q = q.Where(knowledgearticle.IsPublished(true))
		} else if strings.ToLower(status) == "draft" {
			q = q.Where(knowledgearticle.IsPublished(false))
		}
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	es, err := q.Limit(size).Offset((page - 1) * size).Order(ent.Desc(knowledgearticle.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var results []*Article
	for _, e := range es {
		results = append(results, toDomain(e))
	}
	return results, total, nil
}

func (r *EntRepository) Update(ctx context.Context, a *Article) (*Article, error) {
	tagsStr := strings.Join(a.Tags, ",")
	u := r.client.KnowledgeArticle.UpdateOneID(a.ID).
		Where(knowledgearticle.TenantID(a.TenantID), knowledgearticle.DeletedAtIsNil()).
		SetTitle(a.Title).
		SetContent(a.Content).
		SetCategory(a.Category).
		SetTags(tagsStr).
		SetIsPublished(a.IsPublished).
		SetReviewIntervalDays(a.ReviewIntervalDays).
		SetAuthorityLevel(a.AuthorityLevel)

	// 时效字段传 nil 表示「解除时效设置」，必须显式 Clear。
	// 注意 SetNillableXxx(nil) 在 ent 里是 no-op（不清除既有值），
	// 若这里偷懒用它，管理员将无法撤销误设的失效时间——
	// 知识会一直被 L1 过滤掉，且界面上看不出原因。
	if a.ValidFrom != nil {
		u = u.SetValidFrom(*a.ValidFrom)
	} else {
		u = u.ClearValidFrom()
	}
	if a.ValidUntil != nil {
		u = u.SetValidUntil(*a.ValidUntil)
	} else {
		u = u.ClearValidUntil()
	}
	if a.LastReviewedAt != nil {
		u = u.SetLastReviewedAt(*a.LastReviewedAt)
	} else {
		u = u.ClearLastReviewedAt()
	}

	e, err := u.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(e), nil
}

// MarkReviewed 记录一次内容复核。
// 只更新 last_reviewed_at，不触碰正文——复核是对内容「仍然适用」的确认，
// 若与内容更新混在一个入口，复核动作就可能夹带未审校的正文改动。
func (r *EntRepository) MarkReviewed(ctx context.Context, id int, tenantID int) (*Article, error) {
	e, err := r.client.KnowledgeArticle.UpdateOneID(id).
		Where(knowledgearticle.TenantID(tenantID), knowledgearticle.DeletedAtIsNil()).
		SetLastReviewedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(e), nil
}

// TransitionReview 执行一次评审状态流转，并**在同一事务**内落评审流水。
//
// 事务必要性：状态与审计必须同生共死。分开写会出现「状态已改但审计没记」
// （事后无法回答"谁批的"）或「审计已记但状态没改」（审计与事实矛盾），
// 这两种不一致在合规审计里都是不可接受的。
//
// toStatus 传空串表示只记流水、不改状态。
func (r *EntRepository) TransitionReview(
	ctx context.Context, id, tenantID, operatorID int,
	action, toStatus, comment string,
) (*Article, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}

	// 注意：事务内一律用 tx.* 做读写，不要用 r.client —— 那样会走事务外的
	// 连接池，既看不到本事务的中间态，还可能在提交前被别的连接读走。
	current, err := tx.KnowledgeArticle.Query().
		Where(
			knowledgearticle.ID(id),
			knowledgearticle.TenantID(tenantID),
			knowledgearticle.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	fromStatus := NormalizeReviewStatus(string(current.ReviewStatus))
	actionToStatus := toStatus
	if actionToStatus == "" {
		actionToStatus = fromStatus
	}

	if _, err := tx.KnowledgeArticleReview.Create().
		SetArticleID(id).
		SetTenantID(tenantID).
		SetFromStatus(fromStatus).
		SetToStatus(actionToStatus).
		SetAction(action).
		SetOperatorID(operatorID).
		SetComment(comment).
		Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if toStatus != "" && toStatus != fromStatus {
		upd := tx.KnowledgeArticle.UpdateOneID(id).
			Where(knowledgearticle.TenantID(tenantID), knowledgearticle.DeletedAtIsNil()).
			SetReviewStatus(knowledgearticle.ReviewStatus(toStatus)).
			SetReviewerID(operatorID).
			SetReviewedAt(time.Now()).
			SetReviewComment(comment)

		// 通过评审 = 对外可见；驳回 = 撤回可见性。
		// 必须同时改 is_published，否则会出现 review_status=approved
		// 但 is_published=false 的矛盾状态——审核通过却搜不到。
		if toStatus == ReviewStatusApproved {
			upd = upd.SetIsPublished(true)
		} else if toStatus == ReviewStatusRejected || toStatus == ReviewStatusDraft {
			upd = upd.SetIsPublished(false)
		}

		if _, err := upd.Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}

		// 可见性变更必须与状态变更同事务进出向量索引：
		// 评审通过后若索引不同步，RAG 检索不到这篇刚批准的知识；
		// 驳回后若不同步，已被否定的内容仍会被 AI 引用 —— 后者比"搜不到"更糟。
		if toStatus == ReviewStatusApproved || toStatus == ReviewStatusRejected {
			action := vectorIndexSync
			if toStatus == ReviewStatusRejected {
				action = vectorIndexDelete
			}
			articleForIndex := &Article{ID: id, TenantID: tenantID, UpdatedAt: time.Now()}
			if err := enqueueVectorSyncTx(ctx, tx, articleForIndex, action); err != nil {
				_ = tx.Rollback()
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	updated, err := r.client.KnowledgeArticle.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return toDomain(updated), nil
}

// ListReviews 返回文章的评审历史，时间倒序。
func (r *EntRepository) ListReviews(
	ctx context.Context, articleID, tenantID, page, pageSize int,
) ([]*ReviewRecord, int, error) {
	// 评审流水以 tenant_id 独立承载隔离，但仍先校验文章归属，
	// 避免拿到不存在的文章时返回一个空列表（调用方会误判为"没有评审记录"）。
	if err := r.articleExistsForTenant(ctx, articleID, tenantID); err != nil {
		return nil, 0, err
	}

	q := r.client.KnowledgeArticleReview.Query().
		Where(
			knowledgearticlereview.ArticleID(articleID),
			knowledgearticlereview.TenantID(tenantID),
		)

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}

	rows, err := q.Order(ent.Desc(knowledgearticlereview.FieldCreatedAt), ent.Desc(knowledgearticlereview.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	out := make([]*ReviewRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, &ReviewRecord{
			ID:         row.ID,
			ArticleID:  row.ArticleID,
			TenantID:   row.TenantID,
			FromStatus: row.FromStatus,
			ToStatus:   row.ToStatus,
			Action:     row.Action,
			OperatorID: row.OperatorID,
			Comment:    row.Comment,
			CreatedAt:  row.CreatedAt,
		})
	}
	return out, total, nil
}

// articleExistsForTenant 校验文章存在且属于该租户（版本表无 tenant_id，
// 所有版本操作的租户隔离都经这道文章归属校验实现）。
func (r *EntRepository) articleExistsForTenant(ctx context.Context, articleID, tenantID int) error {
	exist, err := r.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.ID(articleID),
			knowledgearticle.TenantID(tenantID),
			knowledgearticle.DeletedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exist {
		return ErrArticleNotFound
	}
	return nil
}

// userDisplayNames 批量查询用户展示名，供版本列表补 createdByName。
func (r *EntRepository) userDisplayNames(ctx context.Context, ids []int) map[int]string {
	names := make(map[int]string, len(ids))
	if len(ids) == 0 {
		return names
	}
	users, err := r.client.User.Query().Where(user.IDIn(ids...)).All(ctx)
	if err != nil {
		// 展示名缺失不阻断版本历史读取，退化为空串由前端兜底
		return names
	}
	for _, u := range users {
		if u.Name != "" {
			names[u.ID] = u.Name
		} else {
			names[u.ID] = u.Username
		}
	}
	return names
}

func toVersionDomain(e *ent.KnowledgeArticleVersion, names map[int]string) *ArticleVersion {
	if e == nil {
		return nil
	}
	return &ArticleVersion{
		Version:       e.Version,
		Title:         e.Title,
		Content:       e.Content,
		CreatedBy:     e.AuthorID,
		CreatedByName: names[e.AuthorID],
		ChangeLog:     e.ChangeSummary,
		CreatedAt:     e.CreatedAt,
	}
}

func (r *EntRepository) ListVersions(ctx context.Context, articleID, tenantID, page, pageSize int) ([]*ArticleVersion, int, error) {
	if err := r.articleExistsForTenant(ctx, articleID, tenantID); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	total, err := r.client.KnowledgeArticleVersion.Query().
		Where(knowledgearticleversion.ArticleID(articleID)).
		Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.client.KnowledgeArticleVersion.Query().
		Where(knowledgearticleversion.ArticleID(articleID)).
		Order(ent.Desc(knowledgearticleversion.FieldVersion)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	authorIDs := make([]int, 0, len(rows))
	for _, row := range rows {
		authorIDs = append(authorIDs, row.AuthorID)
	}
	names := r.userDisplayNames(ctx, authorIDs)

	out := make([]*ArticleVersion, 0, len(rows))
	for _, row := range rows {
		out = append(out, toVersionDomain(row, names))
	}
	return out, total, nil
}

func (r *EntRepository) GetVersion(ctx context.Context, articleID, version, tenantID int) (*ArticleVersion, error) {
	if err := r.articleExistsForTenant(ctx, articleID, tenantID); err != nil {
		return nil, err
	}
	row, err := r.client.KnowledgeArticleVersion.Query().
		Where(
			knowledgearticleversion.ArticleID(articleID),
			knowledgearticleversion.Version(version),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return toVersionDomain(row, r.userDisplayNames(ctx, []int{row.AuthorID})), nil
}

func (r *EntRepository) RestoreVersion(ctx context.Context, articleID, version, tenantID, userID int) (*Article, error) {
	if err := r.articleExistsForTenant(ctx, articleID, tenantID); err != nil {
		return nil, err
	}
	versionEntity, err := r.client.KnowledgeArticleVersion.Query().
		Where(
			knowledgearticleversion.ArticleID(articleID),
			knowledgearticleversion.Version(version),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	article, err := r.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.ID(articleID),
			knowledgearticle.TenantID(tenantID),
			knowledgearticle.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	// 恢复前为当前正文建快照，保证历史链不断（与共享 service 的语义一致）。
	// 快照失败不阻断恢复：恢复是用户显式动作，正文覆写优先。
	if snapshotErr := r.createVersionSnapshot(ctx, article, "恢复前版本"); snapshotErr != nil {
		_ = snapshotErr
	}

	updated, err := r.client.KnowledgeArticle.UpdateOne(article).
		SetTitle(versionEntity.Title).
		SetContent(versionEntity.Content).
		SetCategory(versionEntity.Category).
		SetTags(versionEntity.Tags).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(updated), nil
}

// createVersionSnapshot 为文章当前正文创建一个新版本快照。
func (r *EntRepository) createVersionSnapshot(ctx context.Context, article *ent.KnowledgeArticle, changeSummary string) error {
	latest, _ := r.client.KnowledgeArticleVersion.Query().
		Where(knowledgearticleversion.ArticleID(article.ID)).
		Order(ent.Desc(knowledgearticleversion.FieldVersion)).
		First(ctx)
	nextVersion := 1
	if latest != nil {
		nextVersion = latest.Version + 1
	}
	_, err := r.client.KnowledgeArticleVersion.Create().
		SetArticleID(article.ID).
		SetVersion(nextVersion).
		SetTitle(article.Title).
		SetContent(article.Content).
		SetCategory(article.Category).
		SetTags(article.Tags).
		SetAuthorID(article.AuthorID).
		SetChangeSummary(changeSummary).
		Save(ctx)
	return err
}

func (r *EntRepository) Delete(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.KnowledgeArticle.Update().
		Where(knowledgearticle.ID(id), knowledgearticle.TenantID(tenantID), knowledgearticle.DeletedAtIsNil()).
		SetDeletedAt(time.Now()).
		Save(ctx)
	return err
}

func (r *EntRepository) GetCategories(ctx context.Context, tenantID int) ([]string, error) {
	existing, err := r.client.KnowledgeArticle.Query().
		Where(knowledgearticle.TenantID(tenantID), knowledgearticle.DeletedAtIsNil()).
		GroupBy(knowledgearticle.FieldCategory).
		Strings(ctx)
	if err != nil {
		return nil, err
	}

	// Merge defaults with whatever already exists, then dedupe and sort.
	seen := make(map[string]struct{}, len(existing)+len(defaultCategories))
	merged := make([]string, 0, len(existing)+len(defaultCategories))
	for _, c := range defaultCategories {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		merged = append(merged, c)
	}
	for _, c := range existing {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		merged = append(merged, c)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i] < merged[j] })
	return merged, nil
}

func (r *EntRepository) GetStats(ctx context.Context, tenantID int) (*Stats, error) {
	// Query all articles for this tenant
	query := r.client.KnowledgeArticle.Query().Where(knowledgearticle.TenantID(tenantID), knowledgearticle.DeletedAtIsNil())

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Get published count
	published, err := query.Clone().Where(knowledgearticle.IsPublished(true)).Count(ctx)
	if err != nil {
		return nil, err
	}

	// Draft count
	draft, err := query.Clone().Where(knowledgearticle.IsPublished(false)).Count(ctx)
	if err != nil {
		return nil, err
	}

	// Sum of view counts
	type ViewSum struct {
		TotalViews int `json:"total_views"` // matches ent aggregate alias for Scan()
	}
	var viewResult []ViewSum
	err = r.client.KnowledgeArticle.Query().
		Where(knowledgearticle.TenantID(tenantID), knowledgearticle.DeletedAtIsNil()).
		Aggregate(ent.As(ent.Sum(knowledgearticle.FieldViewCount), "total_views")).
		Scan(ctx, &viewResult)
	if err != nil {
		return nil, err
	}
	totalViews := int64(0)
	if len(viewResult) > 0 {
		totalViews = int64(viewResult[0].TotalViews)
	}

	// Sum of like counts
	type LikeSum struct {
		TotalLikes int `json:"total_likes"` // matches ent aggregate alias for Scan()
	}
	var likeResult []LikeSum
	err = r.client.KnowledgeArticle.Query().
		Where(knowledgearticle.TenantID(tenantID), knowledgearticle.DeletedAtIsNil()).
		Aggregate(ent.As(ent.Sum(knowledgearticle.FieldLikeCount), "total_likes")).
		Scan(ctx, &likeResult)
	if err != nil {
		return nil, err
	}
	totalLikes := int64(0)
	if len(likeResult) > 0 {
		totalLikes = int64(likeResult[0].TotalLikes)
	}

	// Get category distribution
	type CategoryGroup struct {
		Category string `json:"category"`
		Count    int    `json:"count"`
	}
	var categoryResults []CategoryGroup
	err = r.client.KnowledgeArticle.Query().
		Where(knowledgearticle.TenantID(tenantID), knowledgearticle.DeletedAtIsNil()).
		GroupBy(knowledgearticle.FieldCategory).
		Aggregate(ent.Count()).
		Scan(ctx, &categoryResults)
	if err != nil {
		return nil, err
	}

	categories := make([]CategoryStat, 0, len(categoryResults))
	for _, cr := range categoryResults {
		categories = append(categories, CategoryStat{
			Name:  cr.Category,
			Count: int64(cr.Count),
		})
	}

	return &Stats{
		Total:      int64(total),
		Published:  int64(published),
		Draft:      int64(draft),
		TotalViews: totalViews,
		TotalLikes: totalLikes,
		Categories: categories,
	}, nil
}

// GetByIDs returns published, non-deleted articles by ID for a given tenant.
// The returned slice preserves the order of the input ids slice.
func (r *EntRepository) GetByIDs(ctx context.Context, tenantID int, ids []int) ([]*Article, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	es, err := r.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.IDIn(ids...),
			knowledgearticle.TenantID(tenantID),
			knowledgearticle.DeletedAtIsNil(),
			knowledgearticle.IsPublished(true),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// Build a map for quick lookup
	domainMap := make(map[int]*Article, len(es))
	for _, e := range es {
		domainMap[e.ID] = toDomain(e)
	}

	// Preserve input order
	result := make([]*Article, 0, len(ids))
	for _, id := range ids {
		if a, ok := domainMap[id]; ok {
			result = append(result, a)
		}
	}
	return result, nil
}
