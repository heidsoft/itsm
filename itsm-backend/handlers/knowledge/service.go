package knowledge

import (
	"context"

	"go.uber.org/zap"
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
)

type Service struct {
	repo   Repository
	logger *zap.SugaredLogger
	// 向量索引同步（可选注入）：发布→索引，取消发布/软删除→移除向量。
	// nil 时跳过同步，RAG 仍可退化为关键字搜索。
	rag *service.RAGService
}

func NewService(repo Repository, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// SetRAG wires the optional RAG service for vector index synchronization.
func (s *Service) SetRAG(rag *service.RAGService) {
	s.rag = rag
}

// syncVectorIndex keeps the vectors table in sync with article publish state.
// 失败仅告警，不阻断文章主流程；向量库始终只反映“已发布且未删除”的文章。
func (s *Service) syncVectorIndex(ctx context.Context, tenantID, articleID int, title, content string, published bool) {
	if s.rag == nil {
		return
	}
	var err error
	if published {
		err = s.rag.IndexArticle(ctx, tenantID, articleID, title, content)
	} else {
		err = s.rag.RemoveArticle(ctx, tenantID, articleID)
	}
	if err != nil {
		s.logger.Warnw("KnowledgeService: vector index sync failed", "article_id", articleID, "tenant_id", tenantID, "published", published, "error", err)
	}
}

func (s *Service) CreateArticle(ctx context.Context, a *Article) (*Article, error) {
	// XSS 消毒：Title 走 strict（纯文本），Content 走 UGC（保留富文本白名单，剥离 script/on*/javascript:）
	a.Title = common.SanitizeText(a.Title)
	a.Content = common.SanitizeHTML(a.Content)
	s.logger.Infow("Creating Knowledge Article", "title", a.Title, "category", a.Category)
	created, err := s.repo.Create(ctx, a)
	if err != nil {
		return nil, err
	}
	// 创建即发布（如导入/模板初始化）时同步向量索引；草稿无需入向量库。
	if created.IsPublished {
		s.syncVectorIndex(ctx, created.TenantID, created.ID, created.Title, created.Content, true)
	}
	return created, nil
}

func (s *Service) GetArticle(ctx context.Context, id int, tenantID int) (*Article, error) {
	return s.repo.Get(ctx, id, tenantID)
}

func (s *Service) ListArticles(ctx context.Context, tenantID int, page, size int, category, search, status string) ([]*Article, int, error) {
	return s.repo.List(ctx, tenantID, page, size, category, search, status)
}

func (s *Service) UpdateArticle(ctx context.Context, a *Article) (*Article, error) {
	// XSS 消毒
	a.Title = common.SanitizeText(a.Title)
	a.Content = common.SanitizeHTML(a.Content)
	s.logger.Infow("Updating Knowledge Article", "id", a.ID, "title", a.Title)
	updated, err := s.repo.Update(ctx, a)
	if err != nil {
		return nil, err
	}
	// 发布→重新索引（内容可能已变）；取消发布→移除向量，草稿不得进入 RAG 结果。
	s.syncVectorIndex(ctx, updated.TenantID, updated.ID, updated.Title, updated.Content, updated.IsPublished)
	return updated, nil
}

func (s *Service) DeleteArticle(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting Knowledge Article", "id", id)
	if err := s.repo.Delete(ctx, id, tenantID); err != nil {
		return err
	}
	// 软删除后同步移除向量，避免检索侧残留空标题条目（RemoveArticle 幂等）。
	s.syncVectorIndex(ctx, tenantID, id, "", "", false)
	return nil
}

func (s *Service) GetCategories(ctx context.Context, tenantID int) ([]string, error) {
	return s.repo.GetCategories(ctx, tenantID)
}

// SearchArticles performs a RAG-powered search over published knowledge articles.
// It first retrieves articles via the RAG service (vector + keyword hybrid search),
// then fetches the full article records from the database to return complete content.
// Results are sorted by relevance score descending.
func (s *Service) SearchArticles(ctx context.Context, tenantID int, query string, category string, limit int) ([]*Article, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	// If RAG service is wired, use it for real vector + keyword retrieval.
	if s.rag != nil {
		rawResults, err := s.rag.Ask(ctx, tenantID, query, limit)
		if err != nil {
			s.logger.Warnw("KnowledgeService: RAG Ask failed, falling back to keyword", "query", query, "error", err)
			// Fall through to keyword-only path via repo
		} else {
			if len(rawResults) == 0 {
				return []*Article{}, nil
			}

			// Collect article IDs from RAG results, preserving order (by relevance).
			type scoredID struct {
				id    int
				score float64
			}
			scored := make([]struct {
				id    int
				score float64
			}, 0, len(rawResults))
			for _, r := range rawResults {
				if id, ok := r["id"].(int); ok {
					score, _ := r["score"].(float64)
					scored = append(scored, struct {
						id    int
						score float64
					}{id: id, score: score})
				}
			}

			if len(scored) == 0 {
				return []*Article{}, nil
			}

			// Fetch full articles from DB in a single call.
			articles, err := s.repo.GetByIDs(ctx, tenantID, scoredIDsToIDs(scored))
			if err != nil {
				s.logger.Warnw("KnowledgeService: failed to fetch articles by IDs", "error", err)
				return nil, err
			}

			// Re-order by RAG score descending and attach scores.
			scoreMap := make(map[int]float64, len(scored))
			for _, s := range scored {
				scoreMap[s.id] = s.score
			}
			sorted := make([]*Article, 0, len(articles))
			for _, a := range articles {
				if score, ok := scoreMap[a.ID]; ok {
					a.RelevanceScore = score
					sorted = append(sorted, a)
				}
			}
			// Sort by score descending
			for i := 0; i < len(sorted)-1; i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[j].RelevanceScore > sorted[i].RelevanceScore {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
			return sorted, nil
		}
	}

	// No RAG service: fall back to plain ListArticles with keyword search.
	articles, _, err := s.repo.List(ctx, tenantID, 1, limit, category, query, "published")
	return articles, err
}

func scoredIDsToIDs(scored []struct {
	id    int
	score float64
},
) []int {
	ids := make([]int, len(scored))
	for i, s := range scored {
		ids[i] = s.id
	}
	return ids
}

func (s *Service) GetStats(ctx context.Context, tenantID int) (*dto.KnowledgeStatsResponse, error) {
	stats, err := s.repo.GetStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Calculate average rating based on total likes / total articles
	// Since we only have likes (not a 1-5 star rating), we'll use likes as a proxy for rating
	var avgRating float64
	if stats.Total > 0 {
		avgRating = float64(stats.TotalLikes) / float64(stats.Total)
	}

	// Convert categories to DTO format
	categoryStats := make([]dto.CategoryStats, 0, len(stats.Categories))
	for _, cat := range stats.Categories {
		categoryStats = append(categoryStats, dto.CategoryStats{
			Name:  cat.Name,
			Count: int(cat.Count),
		})
	}

	return &dto.KnowledgeStatsResponse{
		Total:      int(stats.Total),
		Published:  int(stats.Published),
		Draft:      int(stats.Draft),
		Views:      stats.TotalViews,
		Rating:     avgRating,
		Categories: categoryStats,
	}, nil
}
