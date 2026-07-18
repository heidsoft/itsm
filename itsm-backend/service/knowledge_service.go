package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/knowledgearticlelike"
	"itsm-backend/ent/knowledgearticlesession"
	"itsm-backend/ent/knowledgearticleversion"

	"go.uber.org/zap"
)

type KnowledgeService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

var allowedKnowledgeCategories = map[string]struct{}{
	"accountManagement": {},
	"troubleshooting":   {},
	"network":           {},
	"processGuide":      {},
	"systemConfig":      {},
	"账号管理":              {},
	"故障排除":              {},
	"网络连接":              {},
	"流程指南":              {},
	"系统配置":              {},
	"用户指南":              {},
	"技术文档":              {},
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
	// XSS 消毒：Title 走 strict（纯文本），Content 走 UGC（保留常见富文本，剥离 script/on*/javascript:）
	req.Title = common.SanitizeText(req.Title)
	req.Content = common.SanitizeHTML(req.Content)
	req.Category = strings.TrimSpace(req.Category)
	if err := validateKnowledgeCategory(req.Category); err != nil {
		return nil, err
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
	update := ks.client.KnowledgeArticle.UpdateOneID(id).
		Where(knowledgearticle.TenantID(tenantID))

	if req.Title != nil {
		cleaned := common.SanitizeText(*req.Title)
		update = update.SetTitle(cleaned)
	}
	if req.Content != nil {
		cleaned := common.SanitizeHTML(*req.Content)
		update = update.SetContent(cleaned)
	}
	if req.Category != nil {
		category := strings.TrimSpace(*req.Category)
		if err := validateKnowledgeCategory(category); err != nil {
			return nil, err
		}
		update = update.SetCategory(category)
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
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("文章不存在")
		}
		ks.logger.Errorw("Failed to update knowledge article", "error", err, "id", id, "tenant_id", tenantID)
		return nil, fmt.Errorf("更新文章失败: %w", err)
	}

	ks.logger.Infow("Knowledge article updated successfully", "id", id, "tenant_id", tenantID)
	return article, nil
}

func validateKnowledgeCategory(category string) error {
	if category == "" {
		return fmt.Errorf("分类不能为空")
	}
	if _, ok := allowedKnowledgeCategories[category]; !ok {
		return fmt.Errorf("无效的知识库分类: %s", category)
	}
	return nil
}

// DeleteArticle 删除知识库文章（级联删除关联数据）
func (ks *KnowledgeService) DeleteArticle(ctx context.Context, id, tenantID int) error {
	ks.logger.Infow("Deleting knowledge article", "id", id, "tenant_id", tenantID)

	// 先删除关联的 user_likes
	_, err := ks.client.KnowledgeArticleLike.Delete().
		Where(knowledgearticlelike.ArticleIDEQ(id)).
		Exec(ctx)
	if err != nil && !ent.IsNotFound(err) {
		ks.logger.Errorw("Failed to delete article likes", "error", err, "id", id)
		return fmt.Errorf("删除文章点赞记录失败: %w", err)
	}

	err = ks.client.KnowledgeArticle.DeleteOneID(id).
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

// ============ 版本历史相关方法 ============

// CreateVersion 创建文章版本快照
func (ks *KnowledgeService) CreateVersion(ctx context.Context, article *ent.KnowledgeArticle, changeSummary string) (*ent.KnowledgeArticleVersion, error) {
	// 获取当前最新版本号
	latestVersion, _ := ks.client.KnowledgeArticleVersion.Query().
		Where(knowledgearticleversion.ArticleID(article.ID)).
		Order(ent.Desc(knowledgearticleversion.FieldVersion)).
		Only(ctx)

	newVersion := 1
	if latestVersion != nil {
		newVersion = latestVersion.Version + 1
	}

	version, err := ks.client.KnowledgeArticleVersion.Create().
		SetArticleID(article.ID).
		SetVersion(newVersion).
		SetTitle(article.Title).
		SetContent(article.Content).
		SetCategory(article.Category).
		SetTags(article.Tags).
		SetAuthorID(article.AuthorID).
		SetChangeSummary(changeSummary).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建版本失败: %w", err)
	}

	ks.logger.Infow("Article version created", "article_id", article.ID, "version", newVersion)
	return version, nil
}

// ListVersions 获取文章版本列表
func (ks *KnowledgeService) ListVersions(ctx context.Context, articleID, tenantID, page, pageSize int) ([]*ent.KnowledgeArticleVersion, int, error) {
	// 验证文章存在且属于该租户
	exist, err := ks.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.ID(articleID),
			knowledgearticle.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("检查文章失败: %w", err)
	}
	if !exist {
		return nil, 0, fmt.Errorf("文章不存在")
	}

	// 查询总数
	total, err := ks.client.KnowledgeArticleVersion.Query().
		Where(knowledgearticleversion.ArticleID(articleID)).
		Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取版本总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	versions, err := ks.client.KnowledgeArticleVersion.Query().
		Where(knowledgearticleversion.ArticleID(articleID)).
		Order(ent.Desc(knowledgearticleversion.FieldVersion)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取版本列表失败: %w", err)
	}

	return versions, total, nil
}

// GetVersion 获取指定版本
func (ks *KnowledgeService) GetVersion(ctx context.Context, articleID, version, tenantID int) (*ent.KnowledgeArticleVersion, error) {
	// 验证文章存在且属于该租户
	exist, err := ks.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.ID(articleID),
			knowledgearticle.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("检查文章失败: %w", err)
	}
	if !exist {
		return nil, fmt.Errorf("文章不存在")
	}

	versionEntity, err := ks.client.KnowledgeArticleVersion.Query().
		Where(
			knowledgearticleversion.ArticleID(articleID),
			knowledgearticleversion.Version(version),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("版本不存在")
		}
		return nil, fmt.Errorf("获取版本失败: %w", err)
	}

	return versionEntity, nil
}

// RestoreVersion 恢复文章到指定版本
func (ks *KnowledgeService) RestoreVersion(ctx context.Context, articleID, version, tenantID, userID int) (*ent.KnowledgeArticle, error) {
	// 获取版本
	versionEntity, err := ks.GetVersion(ctx, articleID, version, tenantID)
	if err != nil {
		return nil, err
	}

	// 获取当前文章
	article, err := ks.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.ID(articleID),
			knowledgearticle.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取文章失败: %w", err)
	}

	// 创建当前版本快照
	_, err = ks.CreateVersion(ctx, article, "恢复前版本")
	if err != nil {
		ks.logger.Warnw("Failed to create version before restore", "error", err)
	}

	// 更新文章内容
	article, err = ks.client.KnowledgeArticle.UpdateOne(article).
		SetTitle(versionEntity.Title).
		SetContent(versionEntity.Content).
		SetCategory(versionEntity.Category).
		SetTags(versionEntity.Tags).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("恢复版本失败: %w", err)
	}

	ks.logger.Infow("Article restored to version", "article_id", articleID, "version", version)
	return article, nil
}

// ============ 实时协作Session相关方法 ============

// CreateSession 创建或加入协作会话
func (ks *KnowledgeService) CreateSession(ctx context.Context, articleID, userID, tenantID int) (*ent.KnowledgeArticleSession, error) {
	// 验证文章存在
	exist, err := ks.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.ID(articleID),
			knowledgearticle.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("检查文章失败: %w", err)
	}
	if !exist {
		return nil, fmt.Errorf("文章不存在")
	}

	// 检查是否已有活跃会话
	existingSession, err := ks.client.KnowledgeArticleSession.Query().
		Where(
			knowledgearticlesession.ArticleID(articleID),
			knowledgearticlesession.StatusEQ(knowledgearticlesession.StatusActive),
		).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("检查会话失败: %w", err)
	}

	// 生成会话Token
	sessionToken := fmt.Sprintf("session_%d_%d_%d", articleID, userID, time.Now().UnixNano())

	if existingSession != nil {
		// 用户加入现有会话
		existingSession, err = ks.client.KnowledgeArticleSession.UpdateOne(existingSession).
			AddUserIDs(userID).
			SetLastHeartbeat(time.Now()).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("加入会话失败: %w", err)
		}
		ks.logger.Infow("User joined existing session", "user_id", userID, "session_id", existingSession.ID)
		return existingSession, nil
	}

	// 创建新会话
	session, err := ks.client.KnowledgeArticleSession.Create().
		SetArticleID(articleID).
		SetUserID(userID).
		SetSessionToken(sessionToken).
		SetStatus(knowledgearticlesession.StatusActive).
		SetLastHeartbeat(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}

	ks.logger.Infow("Session created", "session_id", session.ID, "article_id", articleID)
	return session, nil
}

// Heartbeat 更新会话心跳
func (ks *KnowledgeService) Heartbeat(ctx context.Context, sessionToken string, cursorPos *int) error {
	if cursorPos != nil {
		// 更新光标位置 - 需要通过 Participant 更新
	}

	_, err := ks.client.KnowledgeArticleSession.Update().
		Where(knowledgearticlesession.SessionTokenEQ(sessionToken)).
		SetLastHeartbeat(time.Now()).
		SetStatus(knowledgearticlesession.StatusActive).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("更新心跳失败: %w", err)
	}

	return nil
}

// GetSession 获取当前会话
func (ks *KnowledgeService) GetSession(ctx context.Context, articleID, userID, tenantID int) (*ent.KnowledgeArticleSession, error) {
	// 验证文章存在
	exist, err := ks.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.ID(articleID),
			knowledgearticle.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil || !exist {
		return nil, fmt.Errorf("文章不存在")
	}

	// 查询活跃会话
	session, err := ks.client.KnowledgeArticleSession.Query().
		Where(
			knowledgearticlesession.ArticleID(articleID),
			knowledgearticlesession.StatusEQ(knowledgearticlesession.StatusActive),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil // 没有活跃会话
		}
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}

	return session, nil
}

// ListParticipants 获取参与者列表
func (ks *KnowledgeService) ListParticipants(ctx context.Context, articleID, tenantID int) ([]*ent.KnowledgeArticleSession, error) {
	// 验证文章存在
	exist, err := ks.client.KnowledgeArticle.Query().
		Where(
			knowledgearticle.ID(articleID),
			knowledgearticle.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil || !exist {
		return nil, fmt.Errorf("文章不存在")
	}

	// 查询活跃会话
	sessions, err := ks.client.KnowledgeArticleSession.Query().
		Where(
			knowledgearticlesession.ArticleID(articleID),
			knowledgearticlesession.StatusIn(knowledgearticlesession.StatusActive, knowledgearticlesession.StatusIdle),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取参与者失败: %w", err)
	}

	return sessions, nil
}

// LeaveSession 离开会话
func (ks *KnowledgeService) LeaveSession(ctx context.Context, articleID, userID int) error {
	_, err := ks.client.KnowledgeArticleSession.Update().
		Where(
			knowledgearticlesession.ArticleID(articleID),
			knowledgearticlesession.UserID(userID),
			knowledgearticlesession.StatusEQ(knowledgearticlesession.StatusActive),
		).
		SetStatus(knowledgearticlesession.StatusInactive).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("离开会话失败: %w", err)
	}

	ks.logger.Infow("User left session", "user_id", userID, "article_id", articleID)
	return nil
}
