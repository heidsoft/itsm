package service

import (
	"context"
	"fmt"
	"strings"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/knowledgearticlelike"

	"go.uber.org/zap"
)

type KnowledgeService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewKnowledgeService(client *ent.Client, logger *zap.SugaredLogger) *KnowledgeService {
	return &KnowledgeService{
		client: client,
		logger: logger,
	}
}

// CreateArticle 创建知识库文章
func (ks *KnowledgeService) CreateArticle(ctx context.Context, req *dto.CreateKnowledgeArticleRequest, tenantID int, authorID int) (*ent.KnowledgeArticle, error) {
	ks.logger.Infow("Creating knowledge article", "title", req.Title, "tenant_id", tenantID, "author_id", authorID)

	if strings.TrimSpace(req.Content) == "" {
		return nil, fmt.Errorf("内容不能为空")
	}
	// 将Tags数组转换为字符串
	tagsStr := ""
	if len(req.Tags) > 0 {
		tagsStr = strings.Join(req.Tags, ",")
	}

	// 创建文章
	article, err := ks.client.KnowledgeArticle.Create().
		SetTitle(req.Title).
		SetContent(req.Content).
		SetCategory(req.Category).
		SetAuthorID(authorID).
		SetTags(tagsStr).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		ks.logger.Errorw("Failed to create knowledge article", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("创建文章失败: %w", err)
	}

	ks.logger.Infow("Knowledge article created successfully", "id", article.ID, "tenant_id", tenantID)
	return article, nil
}

// GetArticle 获取知识库文章详情
func (ks *KnowledgeService) GetArticle(ctx context.Context, id, tenantID int) (*ent.KnowledgeArticle, error) {
	article, err := ks.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.ID(id),
			knowledgearticle.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("文章不存在")
		}
		return nil, fmt.Errorf("获取文章失败: %w", err)
	}

	return article, nil
}

// ListArticles 获取知识库文章列表
func (ks *KnowledgeService) ListArticles(ctx context.Context, req *dto.ListKnowledgeArticlesRequest, tenantID int) ([]*ent.KnowledgeArticle, int, error) {
	query := ks.client.KnowledgeArticle.Query().
		Where(knowledgearticle.TenantID(tenantID))

	// 添加过滤条件
	if req.Category != "" {
		query = query.Where(knowledgearticle.Category(req.Category))
	}
	if req.Search != "" {
		query = query.Where(
			knowledgearticle.Or(
				knowledgearticle.TitleContains(req.Search),
				knowledgearticle.ContentContains(req.Search),
			),
		)
	}
	// 状态过滤：published/draft（兼容 DTO 中的 Status）
	if strings.TrimSpace(req.Status) != "" {
		if strings.ToLower(strings.TrimSpace(req.Status)) == "published" {
			query = query.Where(knowledgearticle.IsPublished(true))
		} else if strings.ToLower(strings.TrimSpace(req.Status)) == "draft" {
			query = query.Where(knowledgearticle.IsPublished(false))
		}
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取文章总数失败: %w", err)
	}

	// 分页查询
	articles, err := query.
		Order(ent.Desc(knowledgearticle.FieldCreatedAt)).
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取文章列表失败: %w", err)
	}

	return articles, total, nil
}

// UpdateArticle 更新知识库文章
func (ks *KnowledgeService) UpdateArticle(ctx context.Context, id int, req *dto.UpdateKnowledgeArticleRequest, tenantID int) (*ent.KnowledgeArticle, error) {
	update := ks.client.KnowledgeArticle.UpdateOneID(id)

	if req.Title != nil {
		update = update.SetTitle(*req.Title)
	}
	if req.Content != nil {
		update = update.SetContent(*req.Content)
	}
	if req.Category != nil {
		update = update.SetCategory(*req.Category)
	}
	if req.Tags != nil {
		if len(req.Tags) > 0 {
			// 将Tags数组转换为字符串
			tagsStr := strings.Join(req.Tags, ",")
			update = update.SetTags(tagsStr)
		}
	}

	article, err := update.Save(ctx)
	if err != nil {
		ks.logger.Errorw("Failed to update knowledge article", "error", err, "id", id, "tenant_id", tenantID)
		return nil, fmt.Errorf("更新文章失败: %w", err)
	}

	ks.logger.Infow("Knowledge article updated successfully", "id", id, "tenant_id", tenantID)
	return article, nil
}

// DeleteArticle 删除知识库文章
func (ks *KnowledgeService) DeleteArticle(ctx context.Context, id, tenantID int) error {
	ks.logger.Infow("Deleting knowledge article", "id", id, "tenant_id", tenantID)

	err := ks.client.KnowledgeArticle.DeleteOneID(id).
		Where(knowledgearticle.TenantID(tenantID)).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			ks.logger.Warnw("Knowledge article not found", "id", id, "tenant_id", tenantID)
			return fmt.Errorf("文章不存在")
		}
		ks.logger.Errorw("Failed to delete knowledge article", "error", err, "id", id)
		return fmt.Errorf("删除文章失败: %w", err)
	}

	ks.logger.Infow("Knowledge article deleted successfully", "id", id)
	return nil
}

// GetCategories 获取知识库分类列表
func (ks *KnowledgeService) GetCategories(ctx context.Context, tenantID int) ([]string, error) {
	categories, err := ks.client.KnowledgeArticle.Query().
		Where(knowledgearticle.TenantID(tenantID)).
		GroupBy(knowledgearticle.FieldCategory).
		Strings(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取分类列表失败: %w", err)
	}

	return categories, nil
}

// LikeArticle 点赞/取消点赞知识库文章
func (ks *KnowledgeService) LikeArticle(ctx context.Context, id, userID, tenantID int) error {
	// 检查文章是否存在
	exist, err := ks.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.ID(id),
			knowledgearticle.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("检查文章失败: %w", err)
	}
	if !exist {
		return fmt.Errorf("文章不存在")
	}

	// 检查用户是否已经点赞
	existingLike, err := ks.client.KnowledgeArticleLike.Query().
		Where(
			knowledgearticlelike.ArticleID(id),
			knowledgearticlelike.UserID(userID),
			knowledgearticlelike.TenantID(tenantID),
		).
		Only(ctx)

	if err != nil && !ent.IsNotFound(err) {
		return fmt.Errorf("检查点赞状态失败: %w", err)
	}

	// 如果已经点赞，则取消点赞
	if existingLike != nil {
		// 删除点赞记录
		err = ks.client.KnowledgeArticleLike.DeleteOne(existingLike).Exec(ctx)
		if err != nil {
			return fmt.Errorf("取消点赞失败: %w", err)
		}

		// 减少点赞数
		_, err = ks.client.KnowledgeArticle.UpdateOneID(id).
			AddLikeCount(-1).
			Save(ctx)
		if err != nil {
			ks.logger.Warnw("Failed to decrement like count", "article_id", id, "error", err)
		}

		ks.logger.Infow("User unliked article", "user_id", userID, "article_id", id)
		return nil
	}

	// 如果未点赞，则添加点赞
	_, err = ks.client.KnowledgeArticleLike.Create().
		SetArticleID(id).
		SetUserID(userID).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("创建点赞记录失败: %w", err)
	}

	// 增加点赞数
	_, err = ks.client.KnowledgeArticle.UpdateOneID(id).
		AddLikeCount(1).
		Save(ctx)
	if err != nil {
		ks.logger.Warnw("Failed to increment like count", "article_id", id, "error", err)
	}

	ks.logger.Infow("User liked article", "user_id", userID, "article_id", id)
	return nil
}

// GetLikeStatus 获取用户对文章的点赞状态
func (ks *KnowledgeService) GetLikeStatus(ctx context.Context, articleID, userID, tenantID int) (bool, error) {
	exist, err := ks.client.KnowledgeArticleLike.Query().
		Where(
			knowledgearticlelike.ArticleID(articleID),
			knowledgearticlelike.UserID(userID),
			knowledgearticlelike.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return false, fmt.Errorf("检查点赞状态失败: %w", err)
	}
	return exist, nil
}

// GetLikeCount 获取文章的点赞数
func (ks *KnowledgeService) GetLikeCount(ctx context.Context, articleID, tenantID int) (int, error) {
	article, err := ks.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.ID(articleID),
			knowledgearticle.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return 0, fmt.Errorf("获取文章失败: %w", err)
	}
	return article.LikeCount, nil
}
