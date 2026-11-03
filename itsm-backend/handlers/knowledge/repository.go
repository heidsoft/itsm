package knowledge

import (
	"context"
)

// Visibility 描述当前访问者的分类级可见性收窄条件。
// DeniedCategories 为空表示不做过滤；ViewerUserID > 0 时用于作者豁免。
type Visibility struct {
	DeniedCategories []string
	ViewerUserID     int
}

// Repository interface for Knowledge domain
type Repository interface {
	Create(ctx context.Context, a *Article) (*Article, error)
	Get(ctx context.Context, id int, tenantID int) (*Article, error)
	// List 的可见性过滤在 SQL 层完成：先过滤再分页，total 才代表
	// 当前访问者真正能看到的条数；先分页后过滤会让分页语义失真。
	List(ctx context.Context, tenantID int, page, size int, category, search, status string, vis Visibility) ([]*Article, int, error)
	Update(ctx context.Context, a *Article) (*Article, error)
	// PublishWithPointer 发布文章并建立「线上版本」指针：同事务内为即将上线的
	// 正文落版本快照、把 published_version 指向它、翻 is_published。
	// 有了它，「线上是哪个版本」才有唯一事实源——此前发布只翻布尔位，
	// 版本表存的又是「更新前快照」，发布时刻的内容在版本表里没有对应行。
	PublishWithPointer(ctx context.Context, id, tenantID int) (*Article, error)
	// UnpublishWithPointer 撤回发布并清指针：撤回后线上已无内容，
	// 保留指针会让「线上是哪个版本」指向一个已不可见的版本号。
	// 撤回不设评审门槛——把内容从可见面撤下永远是安全方向。
	UnpublishWithPointer(ctx context.Context, id, tenantID int) (*Article, error)
	Delete(ctx context.Context, id int, tenantID int) error
	// MarkReviewed 记录一次内容复核：把 last_reviewed_at 置为当前时间。
	// 这是 L1 时效管控的闭环动作——声明了复核周期的知识逾期后会被 RAG 过滤，
	// 复核是把它恢复回检索结果的唯一途径。只改复核时间，不触碰正文。
	MarkReviewed(ctx context.Context, id int, tenantID int) (*Article, error)
	// TransitionReview 执行一次评审状态流转，并落一条评审流水。
	//
	// 为什么不放 service：状态流转必须与流水写入在**同一事务**里，
	// 否则会出现「状态已改但审计没记」或反之的可追溯性缺口。
	// toStatus 传空串表示不动 review_status（仅记流水），用于纯审计场景。
	TransitionReview(ctx context.Context, id, tenantID, operatorID int, action, toStatus, comment string) (*Article, error)
	// ListReviews 返回文章的评审历史，时间倒序。
	ListReviews(ctx context.Context, articleID, tenantID, page, pageSize int) ([]*ReviewRecord, int, error)
	// ListVersions 返回文章的版本快照，按版本号倒序（最新在前）。
	// 版本表无 tenant_id，租户隔离经文章归属校验实现。
	ListVersions(ctx context.Context, articleID, tenantID, page, pageSize int) ([]*ArticleVersion, int, error)
	// GetVersion 返回指定版本号的快照；版本不存在或文章非本租户时返回错误。
	GetVersion(ctx context.Context, articleID, version, tenantID int) (*ArticleVersion, error)
	// RestoreVersion 把文章正文恢复到指定版本：先为当前正文建快照，再覆写。
	// 恢复本身也会留下一个新版本，历史链不断。
	RestoreVersion(ctx context.Context, articleID, version, tenantID, userID int) (*Article, error)
	GetCategories(ctx context.Context, tenantID int, vis Visibility) ([]string, error)
	GetStats(ctx context.Context, tenantID int, vis Visibility) (*Stats, error)
	// GetByIDs returns articles by a list of IDs, preserving the order of IDs.
	// It filters by tenant and only returns published, non-deleted articles.
	GetByIDs(ctx context.Context, tenantID int, ids []int) ([]*Article, error)
	// ListBySourceProblem 返回由指定问题沉淀的文章（含草稿，创建时间倒序）。
	// 问题详情页「沉淀了哪些知识」反查的主路径；含草稿是因为
	// 沉淀动作的当事人需要立刻看到沉淀结果（草稿也要走评审流）。
	ListBySourceProblem(ctx context.Context, tenantID, problemID int) ([]*Article, error)
}

// Stats represents knowledge base statistics
type Stats struct {
	Total      int64
	Published  int64
	Draft      int64
	TotalViews int64
	TotalLikes int64
	Categories []CategoryStat
}

// CategoryStat represents category statistics
type CategoryStat struct {
	Name  string
	Count int64
}
