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
