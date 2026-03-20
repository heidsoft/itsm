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
	GetCategories(ctx context.Context, tenantID int) ([]string, error)
	GetStats(ctx context.Context, tenantID int) (*Stats, error)
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
