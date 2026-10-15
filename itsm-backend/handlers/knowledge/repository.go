package knowledge

import (
	"context"
)

// Repository interface for Knowledge domain
type Repository interface {
	Create(ctx context.Context, a *Article) (*Article, error)
	Get(ctx context.Context, id int, tenantID int) (*Article, error)
	List(ctx context.Context, tenantID int, page, size int, category, search, status string) ([]*Article, int, error)
	Update(ctx context.Context, a *Article) (*Article, error)
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
	GetCategories(ctx context.Context, tenantID int) ([]string, error)
	GetStats(ctx context.Context, tenantID int) (*Stats, error)
	// GetByIDs returns articles by a list of IDs, preserving the order of IDs.
	// It filters by tenant and only returns published, non-deleted articles.
	GetByIDs(ctx context.Context, tenantID int, ids []int) ([]*Article, error)
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
