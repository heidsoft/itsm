package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/tenant"

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
func (ks *KnowledgeService) CreateArticle(ctx context.Context, req *dto.CreateKnowledgeArticleRequest, tenantID int, author string) (*ent.KnowledgeArticle, error) {
	// 验证租户存在
	exists, err := ks.client.Tenant.Query().Where(tenant.ID(tenantID)).Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("验证租户失败: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("租户不存在")
	}

	// 创建文章
	article, err := ks.client.KnowledgeArticle.Create().
		SetTitle(req.Title).
		SetContent(req.Content).
		SetCategory(req.Category).
		SetAuthor(author).
		SetTags(req.Tags).
		SetTenantID(tenantID).
		Save(ctx)

	if err != nil {
		ks.logger.Errorf("创建知识库文章失败: %v", err)
		return nil, fmt.Errorf("创建文章失败: %w", err)
	}

	ks.logger.Infof("创建知识库文章成功: %s", article.Title)
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

	// 增加浏览次数
	_, err = ks.client.KnowledgeArticle.UpdateOneID(id).
		SetViews(article.Views + 1).
		Save(ctx)
	if err != nil {
		ks.logger.Warnf("更新文章浏览次数失败: %v", err)
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
	if req.Status != "" {
		query = query.Where(knowledgearticle.Status(req.Status))
	}
	if req.Search != "" {
		query = query.Where(
			knowledgearticle.Or(
				knowledgearticle.TitleContains(req.Search),
				knowledgearticle.ContentContains(req.Search),
			),
		)
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
	if req.Status != nil {
		update = update.SetStatus(*req.Status)
	}
	if req.Tags != nil {
		update = update.SetTags(req.Tags)
	}

	article, err := update.
		Where(knowledgearticle.TenantID(tenantID)).
		Save(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("文章不存在")
		}
		return nil, fmt.Errorf("更新文章失败: %w", err)
	}

	return article, nil
}

// DeleteArticle 删除知识库文章
func (ks *KnowledgeService) DeleteArticle(ctx context.Context, id, tenantID int) error {
	err := ks.client.KnowledgeArticle.DeleteOneID(id).
		Where(knowledgearticle.TenantID(tenantID)).
		Exec(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("文章不存在")
		}
		return fmt.Errorf("删除文章失败: %w", err)
	}

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
