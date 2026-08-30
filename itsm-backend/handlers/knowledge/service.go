package knowledge

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/permission"
	"itsm-backend/handlers/common/knowledgeaccess"
	"itsm-backend/service"
)

type Service struct {
	repo   Repository
	logger *zap.SugaredLogger
	// 向量索引同步（可选注入）：发布→索引，取消发布/软删除→移除向量。
	// nil 时跳过同步，RAG 仍可退化为关键字搜索。
	rag *service.RAGService
	// client 用于访问权限表（knowledgeaccess 的纳管记录存在 permission 表）。
	// 经 SetEntClient 注入；未注入时纳管接口返回明确错误而非静默失败。
	client *ent.Client
	// knowledgeGuard 知识分类可见性守卫（L0 权限边界）。
	knowledgeGuard *knowledgeaccess.Guard
	// freshness 时效性判定器（L1 正确性边界）。
	// 与 RAG 服务内的判定器共用同一套策略，此处用于 RAG 未装配时的检索兜底路径。
	freshness *knowledgeaccess.FreshnessJudger
}

func NewService(repo Repository, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
		// 默认装配时效判定器，保证即便调用方忘记注入，检索也不会引用过期知识。
		freshness: knowledgeaccess.NewFreshnessJudger(knowledgeaccess.DefaultFreshnessPolicy(), nil),
	}
}

// SetEntClient 注入 ent 客户端，启用知识分类纳管能力。
func (s *Service) SetEntClient(client *ent.Client) { s.client = client }

// SetKnowledgeGuard 注入知识分类可见性守卫，用于纳管后失效缓存。
func (s *Service) SetKnowledgeGuard(g *knowledgeaccess.Guard) { s.knowledgeGuard = g }

// SetFreshnessJudger 注入时效性判定器（L1）。传 nil 表示关闭时效过滤。
func (s *Service) SetFreshnessJudger(j *knowledgeaccess.FreshnessJudger) { s.freshness = j }

// MarkArticleReviewed 记录一次内容复核。
//
// L1 时效管控的闭环动作：声明了复核周期的知识一旦逾期，会从 RAG 检索结果中消失，
// 复核是把它恢复回来的唯一途径。只更新复核时间，不改正文，
// 因此复核动作不可能夹带未经审校的内容改动。
func (s *Service) MarkArticleReviewed(ctx context.Context, id int, tenantID int) (*Article, error) {
	a, err := s.repo.MarkReviewed(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	s.logger.Infow("知识内容复核完成",
		"article_id", id, "tenant_id", tenantID, "last_reviewed_at", a.LastReviewedAt)
	return a, nil
}

// filterCitable 按时效性过滤检索结果（L1）。
//
// 注意这里必须与管理端列表区分开：
// ListArticles 要看到已失效、待复核的知识，否则管理员无从治理；
// 而检索是给 LLM 引用的入口，逾期内容一旦进来就会变成带来源的错误答案。
// 本方法只服务「检索」语义，不要把它接到管理端列表上。
func (s *Service) filterCitable(articles []*Article) []*Article {
	if s.freshness == nil || len(articles) == 0 {
		return articles
	}
	kept := make([]*Article, 0, len(articles))
	for _, a := range articles {
		if a == nil {
			continue
		}
		if s.freshness.Citable(knowledgeaccess.FreshnessFields{
			ValidFrom:          a.ValidFrom,
			ValidUntil:         a.ValidUntil,
			LastReviewedAt:     a.LastReviewedAt,
			ReviewIntervalDays: a.ReviewIntervalDays,
		}) {
			kept = append(kept, a)
		}
	}
	return kept
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
	if err != nil {
		return nil, err
	}
	// RAG 未装配时，这条路径就是知识检索的唯一入口，时效性必须在这里拦住，
	// 否则未部署向量库的环境会整体绕过 L1。
	return s.filterCitable(articles), nil
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

// ===== 知识分类可见性纳管（L0 权限边界） =====
//
// 纳管语义：为某分类在 RBAC permission 表写入一条
// resource="knowledge_category" / action="read:<分类>" 记录。
// 写入即纳管——此后该分类下的知识仅对持有该权限的角色可见。
// 这是「企业内部知识可引用性」的地基：没有它，上层任何检索优化
// 都是在给越权内容做更好的曝光。

// ListRestrictedCategories 返回当前租户已被纳管的知识分类。
func (s *Service) ListRestrictedCategories(ctx context.Context, tenantID int) ([]string, error) {
	if s.client == nil {
		return nil, fmt.Errorf("知识分类纳管能力未装配（ent client 缺失）")
	}
	perms, err := s.client.Permission.Query().
		Where(permission.TenantIDEQ(tenantID), permission.ResourceEQ(knowledgeaccess.CategoryResource)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询受限分类失败: %w", err)
	}
	cats := make([]string, 0, len(perms))
	for _, p := range perms {
		if !strings.HasPrefix(p.Action, knowledgeaccess.ActionPrefix) {
			continue
		}
		if c := strings.TrimSpace(p.Action[len(knowledgeaccess.ActionPrefix):]); c != "" {
			cats = append(cats, c)
		}
	}
	sort.Strings(cats)
	return cats, nil
}

// SetCategoryRestriction 纳管（restricted=true）或解除纳管（false）某个知识分类。
//
// 纳管时若该分类无任何文章，仍然允许——管理员可提前规划分类权限模型。
// 解除纳管即删除对应 permission 记录，该分类恢复为全租户可读。
func (s *Service) SetCategoryRestriction(ctx context.Context, tenantID int, category string, restricted bool) error {
	if s.client == nil {
		return fmt.Errorf("知识分类纳管能力未装配（ent client 缺失）")
	}
	category = strings.TrimSpace(category)
	if category == "" {
		return fmt.Errorf("分类名不能为空")
	}

	action := knowledgeaccess.ActionForCategory(category)

	if !restricted {
		n, err := s.client.Permission.Delete().
			Where(
				permission.TenantIDEQ(tenantID),
				permission.ResourceEQ(knowledgeaccess.CategoryResource),
				permission.ActionEQ(action),
			).Exec(ctx)
		if err != nil {
			return fmt.Errorf("解除纳管失败: %w", err)
		}
		s.logger.Infow("知识分类已解除纳管", "tenant_id", tenantID, "category", category, "deleted", n)
		if s.knowledgeGuard != nil {
			s.knowledgeGuard.Invalidate(tenantID)
		}
		return nil
	}

	// 纳管：已存在则幂等跳过，避免重复记录干扰权限判定
	exists, err := s.client.Permission.Query().
		Where(
			permission.TenantIDEQ(tenantID),
			permission.ResourceEQ(knowledgeaccess.CategoryResource),
			permission.ActionEQ(action),
		).Exist(ctx)
	if err != nil {
		return fmt.Errorf("校验纳管状态失败: %w", err)
	}
	if !exists {
		_, err = s.client.Permission.Create().
			SetCode(knowledgeaccess.CategoryResource + "_" + category).
			SetName("知识分类-" + category).
			SetDescription("知识分类可见性：仅授权角色可读该分类下的知识（AI 检索同样受控）").
			SetResource(knowledgeaccess.CategoryResource).
			SetAction(action).
			SetTenantID(tenantID).Save(ctx)
		if err != nil {
			return fmt.Errorf("纳管分类失败: %w", err)
		}
		s.logger.Infow("知识分类已纳管", "tenant_id", tenantID, "category", category)
	}

	if s.knowledgeGuard != nil {
		s.knowledgeGuard.Invalidate(tenantID)
	}
	return nil
}
