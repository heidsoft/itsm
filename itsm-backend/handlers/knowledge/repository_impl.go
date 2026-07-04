package knowledge

import (
	"context"
	"strings"

	"itsm-backend/ent"
	"itsm-backend/ent/knowledgearticle"
)

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
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(e), nil
}

func (r *EntRepository) Get(ctx context.Context, id int, tenantID int) (*Article, error) {
	e, err := r.client.KnowledgeArticle.Query().
		Where(knowledgearticle.ID(id), knowledgearticle.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(e), nil
}

func (r *EntRepository) List(ctx context.Context, tenantID int, page, size int, category, search, status string) ([]*Article, int, error) {
	q := r.client.KnowledgeArticle.Query().Where(knowledgearticle.TenantID(tenantID))

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
	e, err := r.client.KnowledgeArticle.UpdateOneID(a.ID).
		SetTitle(a.Title).
		SetContent(a.Content).
		SetCategory(a.Category).
		SetTags(tagsStr).
		SetIsPublished(a.IsPublished).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toDomain(e), nil
}

func (r *EntRepository) Delete(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.KnowledgeArticle.Delete().
		Where(knowledgearticle.ID(id), knowledgearticle.TenantID(tenantID)).
		Exec(ctx)
	return err
}

func (r *EntRepository) GetCategories(ctx context.Context, tenantID int) ([]string, error) {
	return r.client.KnowledgeArticle.Query().
		Where(knowledgearticle.TenantID(tenantID)).
		GroupBy(knowledgearticle.FieldCategory).
		Strings(ctx)
}

func (r *EntRepository) GetStats(ctx context.Context, tenantID int) (*Stats, error) {
	// Query all articles for this tenant
	query := r.client.KnowledgeArticle.Query().Where(knowledgearticle.TenantID(tenantID))

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
		Where(knowledgearticle.TenantID(tenantID)).
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
		Where(knowledgearticle.TenantID(tenantID)).
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
		Where(knowledgearticle.TenantID(tenantID)).
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
