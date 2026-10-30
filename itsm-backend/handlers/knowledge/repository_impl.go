package knowledge

import (
	"context"
	"fmt"
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
		SourceProblemID:    e.SourceProblemID,
		PublishedVersion:   e.PublishedVersion,
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
		SetNillableSourceProblemID(a.SourceProblemID).
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

// Update 更新文章正文与属性。
//
// 事务边界外置（见 updateTx）：向量 outbox 路径要把「正文更新」与「索引
// 任务入箱」放进同一事务，这里自开事务会导致两层独立事务、外层回滚时内层
// 已提交的错位。
func (r *EntRepository) Update(ctx context.Context, a *Article) (*Article, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	updated, err := updateTx(ctx, tx.Client(), a)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return updated, nil
}

// updateTx 在给定客户端上执行更新，并在可见性翻转时同步维护「线上版本」指针。
//
// 指针必须在这里也维护：PUT 的 status=published 与 /publish 是同一可见性投影
// （UpdateArticle 已对两者施加相同的评审门禁），若只维护其中一个入口，另一
// 个入口就会让「线上是哪个版本」的答案与事实脱节。
func updateTx(ctx context.Context, client *ent.Client, a *Article) (*Article, error) {
	tagsStr := strings.Join(a.Tags, ",")

	// 读权威当前可见性，判定是否翻转。不采纳入参 PublishedVersion：
	// 指针是系统事实，任何 PUT 都不该能改写线上版本的指向。
	current, err := client.KnowledgeArticle.Query().
		Where(knowledgearticle.ID(a.ID), knowledgearticle.TenantID(a.TenantID), knowledgearticle.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	// 翻转方向：false→true 落快照并指向它；true→false 清指针。
	// 快照必须在正文被覆写**之前**按「即将上线的内容」落，否则快照存的是旧正文。
	var publishVersion *int
	clearPointer := false
	switch {
	case a.IsPublished && !current.IsPublished:
		snapshot := &ent.KnowledgeArticle{
			ID:       a.ID,
			Title:    a.Title,
			Content:  a.Content,
			Category: a.Category,
			Tags:     tagsStr,
			AuthorID: a.AuthorID,
		}
		version, err := (&EntRepository{client: client}).createVersionSnapshotTx(ctx, client, snapshot, "发布版本")
		if err != nil {
			return nil, fmt.Errorf("create publish snapshot: %w", err)
		}
		publishVersion = &version
	case !a.IsPublished && current.IsPublished:
		clearPointer = true
	}

	u := client.KnowledgeArticle.UpdateOneID(a.ID).
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
	// 指针写入必须与 is_published 同一次 Save：分成两条语句会出现
	// 「线上已换版本但指针还指着上一个」的中间态。
	if publishVersion != nil {
		u = u.SetPublishedVersion(*publishVersion)
	} else if clearPointer {
		u = u.ClearPublishedVersion()
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
		//
		// 可见性翻转时同步维护「线上版本」指针：评审通过是发布的主路径
		// （/publish 与 PUT 是另外两条），漏了这条就等于主路径上的线上内容
		// 没有版本身份。指针与 is_published 同一次 Save，不产生中间态。
		switch {
		case toStatus == ReviewStatusApproved && !current.IsPublished:
			version, err := (&EntRepository{client: tx.Client()}).
				createVersionSnapshotTx(ctx, tx.Client(), current, "发布版本")
			if err != nil {
				_ = tx.Rollback()
				return nil, fmt.Errorf("create publish snapshot: %w", err)
			}
			upd = upd.SetIsPublished(true).SetPublishedVersion(version)
		case (toStatus == ReviewStatusRejected || toStatus == ReviewStatusDraft) && current.IsPublished:
			// 撤下后线上已无内容，指针必须一起清。
			upd = upd.SetIsPublished(false).ClearPublishedVersion()
		case toStatus == ReviewStatusApproved:
			upd = upd.SetIsPublished(true)
		default:
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

	// 恢复前为当前正文建快照，保证历史链不断。快照失败必须阻断恢复：
	// 否则覆写后旧正文无处可寻，历史链出现不可恢复的空洞。
	if err := r.createVersionSnapshot(ctx, article, "恢复前版本"); err != nil {
		return nil, err
	}

	// 恢复是内容变更：旧正文未经重新评审不得继续对外可见。
	// 回到 draft 并撤销发布，由作者重新走 submit→approve；
	// 旧索引的清理在 service 的向量删除 outbox 中同事务完成。
	updated, err := r.client.KnowledgeArticle.UpdateOne(article).
		SetTitle(versionEntity.Title).
		SetContent(versionEntity.Content).
		SetCategory(versionEntity.Category).
		SetTags(versionEntity.Tags).
		SetIsPublished(false).
		SetReviewStatus(knowledgearticle.ReviewStatus(ReviewStatusDraft)).
		// 撤回后线上已无内容：指针必须一起清，否则「线上是哪个版本」
		// 会指向一个已经不可见的版本号。
		ClearPublishedVersion().
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(updated), nil
}

// createVersionSnapshotTx 在指定客户端上为文章当前正文创建版本快照，返回新版本号。
// tx 语义由调用方决定（生产路径传事务客户端，保证快照与指针同生共死）。
func (r *EntRepository) createVersionSnapshotTx(ctx context.Context, client *ent.Client, article *ent.KnowledgeArticle, changeSummary string) (int, error) {
	latest, _ := client.KnowledgeArticleVersion.Query().
		Where(knowledgearticleversion.ArticleID(article.ID)).
		Order(ent.Desc(knowledgearticleversion.FieldVersion)).
		First(ctx)
	nextVersion := 1
	if latest != nil {
		nextVersion = latest.Version + 1
	}
	_, err := client.KnowledgeArticleVersion.Create().
		SetArticleID(article.ID).
		SetVersion(nextVersion).
		SetTitle(article.Title).
		SetContent(article.Content).
		SetCategory(article.Category).
		SetTags(article.Tags).
		SetAuthorID(article.AuthorID).
		SetChangeSummary(changeSummary).
		Save(ctx)
	return nextVersion, err
}

// createVersionSnapshot 为文章当前正文创建一个新版本快照。
func (r *EntRepository) createVersionSnapshot(ctx context.Context, article *ent.KnowledgeArticle, changeSummary string) error {
	_, err := r.createVersionSnapshotTx(ctx, r.client, article, changeSummary)
	return err
}

// publishWithPointerTx 在给定客户端上执行「发布」三步：
//
//  1. 为即将上线的正文落版本快照（发布时刻的内容从此在版本表有对应行）；
//  2. published_version 指向该快照；
//  3. is_published 翻 true。
//
// 此前发布只翻布尔位，版本表存的又是「更新前快照」——线上正文的版本身份
// 无处可查，「线上是哪个版本」这个问题没有唯一事实源。三步必须同一事务：
// 指针指向不存在的快照、或快照落了而指针没更新，都会让答案与事实脱节。
//
// client 由调用方给出而非内部自开事务：ent 的 tx.Client().Tx() 会另开一个
// 独立事务（不是 savepoint），外层回滚时内层已提交。service 的向量 outbox
// 路径要把「指针」与「索引任务」放进同一个事务，所以事务边界必须外置。
func publishWithPointerTx(ctx context.Context, client *ent.Client, id, tenantID int) (*Article, error) {
	article, err := client.KnowledgeArticle.Query().Where(
		knowledgearticle.ID(id),
		knowledgearticle.TenantID(tenantID),
		knowledgearticle.DeletedAtIsNil(),
	).Only(ctx)
	if err != nil {
		return nil, err
	}

	version, err := (&EntRepository{client: client}).createVersionSnapshotTx(ctx, client, article, "发布版本")
	if err != nil {
		return nil, fmt.Errorf("create publish snapshot: %w", err)
	}

	updated, err := client.KnowledgeArticle.UpdateOneID(id).
		SetIsPublished(true).
		SetPublishedVersion(version).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(updated), nil
}

// unpublishWithPointerTx 撤回发布：清指针 + 翻 is_published。
// 指针必须清：撤回后线上已无内容，保留指针会让「线上是哪个版本」
// 得到一个已不可见的版本号。
func unpublishWithPointerTx(ctx context.Context, client *ent.Client, id, tenantID int) (*Article, error) {
	updated, err := client.KnowledgeArticle.UpdateOneID(id).
		Where(knowledgearticle.TenantID(tenantID), knowledgearticle.DeletedAtIsNil()).
		SetIsPublished(false).
		ClearPublishedVersion().
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(updated), nil
}

// PublishWithPointer 发布文章并建立「线上版本」指针。
func (r *EntRepository) PublishWithPointer(ctx context.Context, id, tenantID int) (*Article, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	updated, err := publishWithPointerTx(ctx, tx.Client(), id, tenantID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return updated, nil
}

// UnpublishWithPointer 撤回发布并清除「线上版本」指针。
func (r *EntRepository) UnpublishWithPointer(ctx context.Context, id, tenantID int) (*Article, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	updated, err := unpublishWithPointerTx(ctx, tx.Client(), id, tenantID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return updated, nil
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

// ListBySourceProblem 返回由指定问题沉淀的文章（含草稿，创建时间倒序）。
// 租户过滤在查询里做：跨租户的问题 ID 查不到任何文章，
// 与问题域可见性校验（ErrProblemNotVisible）共同构成双向防线。
func (r *EntRepository) ListBySourceProblem(ctx context.Context, tenantID, problemID int) ([]*Article, error) {
	es, err := r.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.TenantID(tenantID),
			knowledgearticle.SourceProblemID(problemID),
			knowledgearticle.DeletedAtIsNil(),
		).
		Order(ent.Desc(knowledgearticle.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	articles := make([]*Article, 0, len(es))
	for _, e := range es {
		articles = append(articles, toDomain(e))
	}
	return articles, nil
}
