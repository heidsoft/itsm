package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type KnowledgeController struct {
	knowledgeService *service.KnowledgeService
	logger           *zap.SugaredLogger
}

func NewKnowledgeController(knowledgeService *service.KnowledgeService, logger *zap.SugaredLogger) *KnowledgeController {
	return &KnowledgeController{
		knowledgeService: knowledgeService,
		logger:           logger,
	}
}

// CreateArticle 创建知识库文章
// @Summary 创建知识库文章
// @Description 创建新的知识库文章
// @Tags 知识库
// @Accept json
// @Produce json
// @Param article body dto.CreateKnowledgeArticleRequest true "文章信息"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 {object} common.Response
// @Router /api/v1/knowledge-articles [post]
func (kc *KnowledgeController) CreateArticle(c *gin.Context) {
	var req dto.CreateKnowledgeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 从上下文获取租户ID和用户信息（与中间件键名保持一致）
	tenantID, _ := c.Get("tenant_id")
	userId, _ := c.Get("user_id")

	article, err := kc.knowledgeService.CreateArticle(c.Request.Context(), &req, tenantID.(int), userId.(int))
	if err != nil {
		kc.logger.Errorf("创建知识库文章失败: %v", err)
		common.Fail(c, 5001, "创建文章失败: "+err.Error())
		return
	}

	// tags为string转[]string
	tags := []string{}
	if article.Tags != "" {
		tags = strings.Split(article.Tags, ",")
	}

	response := &dto.KnowledgeArticleResponse{
		ID:        article.ID,
		Title:     article.Title,
		Content:   article.Content,
		Category:  article.Category,
		Tags:      tags,
		TenantID:  article.TenantID,
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}

	common.Success(c, response)
}

// GetArticle 获取知识库文章详情
// @Summary 获取知识库文章详情
// @Description 根据ID获取知识库文章详情
// @Tags 知识库
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 {object} common.Response
// @Router /api/v1/knowledge-articles/{id} [get]
func (kc *KnowledgeController) GetArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的文章ID")
		return
	}

	tenantID, _ := c.Get("tenant_id")

	article, err := kc.knowledgeService.GetArticle(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		kc.logger.Errorf("获取知识库文章失败: %v", err)
		common.Fail(c, 5001, "获取文章失败: "+err.Error())
		return
	}

	// tags为string转[]string
	tags := []string{}
	if article.Tags != "" {
		tags = strings.Split(article.Tags, ",")
	}
	response := &dto.KnowledgeArticleResponse{
		ID:        article.ID,
		Title:     article.Title,
		Content:   article.Content,
		Category:  article.Category,
		Tags:      tags,
		TenantID:  article.TenantID,
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}

	common.Success(c, response)
}

// ListArticles 获取知识库文章列表
// @Summary 获取知识库文章列表
// @Description 分页获取知识库文章列表
// @Tags 知识库
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param category query string false "分类过滤"
// @Param status query string false "状态过滤"
// @Param search query string false "搜索关键词"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleListResponse}
// @Failure 400 {object} common.Response
// @Router /api/v1/knowledge-articles [get]
func (kc *KnowledgeController) ListArticles(c *gin.Context) {
	var req dto.ListKnowledgeArticlesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		kc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	tenantID, _ := c.Get("tenant_id")

	articles, total, err := kc.knowledgeService.ListArticles(c.Request.Context(), &req, tenantID.(int))
	if err != nil {
		kc.logger.Errorf("获取知识库文章列表失败: %v", err)
		common.Fail(c, 5001, "获取文章列表失败: "+err.Error())
		return
	}

	// 转换响应格式
	articleResponses := make([]dto.KnowledgeArticleResponse, len(articles))
	for i, article := range articles {
		tags := []string{}
		if article.Tags != "" {
			tags = strings.Split(article.Tags, ",")
		}
		articleResponses[i] = dto.KnowledgeArticleResponse{
			ID:        article.ID,
			Title:     article.Title,
			Content:   article.Content,
			Category:  article.Category,
			Tags:      tags,
			TenantID:  article.TenantID,
			CreatedAt: article.CreatedAt,
			UpdatedAt: article.UpdatedAt,
		}
	}

	response := &dto.KnowledgeArticleListResponse{
		Articles: articleResponses,
		Total:    total,
		Page:     req.Page,
		Size:     req.PageSize,
	}

	common.Success(c, response)
}

// UpdateArticle 更新知识库文章
// @Summary 更新知识库文章
// @Description 更新指定的知识库文章
// @Tags 知识库
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Param article body dto.UpdateKnowledgeArticleRequest true "文章信息"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Failure 400 {object} common.Response
// @Router /api/v1/knowledge-articles/{id} [put]
func (kc *KnowledgeController) UpdateArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的文章ID")
		return
	}

	var req dto.UpdateKnowledgeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kc.logger.Errorf("参数绑定失败: %v", err)
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")

	article, err := kc.knowledgeService.UpdateArticle(c.Request.Context(), id, &req, tenantID.(int))
	if err != nil {
		kc.logger.Errorf("更新知识库文章失败: %v", err)
		common.Fail(c, 5001, "更新文章失败: "+err.Error())
		return
	}

	// tags为string转[]string
	tags := []string{}
	if article.Tags != "" {
		tags = strings.Split(article.Tags, ",")
	}
	response := &dto.KnowledgeArticleResponse{
		ID:        article.ID,
		Title:     article.Title,
		Content:   article.Content,
		Category:  article.Category,
		Tags:      tags,
		TenantID:  article.TenantID,
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}

	common.Success(c, response)
}

// DeleteArticle 删除知识库文章
// @Summary 删除知识库文章
// @Description 删除指定的知识库文章
// @Tags 知识库
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Router /api/v1/knowledge-articles/{id} [delete]
func (kc *KnowledgeController) DeleteArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的文章ID")
		return
	}

	tenantID, _ := c.Get("tenant_id")

	err = kc.knowledgeService.DeleteArticle(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		kc.logger.Errorf("删除知识库文章失败: %v", err)
		common.Fail(c, 5001, "删除文章失败: "+err.Error())
		return
	}

	common.Success(c, nil)
}

// GetCategories 获取知识库分类列表
// @Summary 获取知识库分类列表
// @Description 获取当前租户的知识库分类列表
// @Tags 知识库
// @Accept json
// @Produce json
// @Success 200 {object} common.Response{data=[]string}
// @Failure 400 {object} common.Response
// @Router /api/v1/knowledge-articles/categories [get]
func (kc *KnowledgeController) GetCategories(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	categories, err := kc.knowledgeService.GetCategories(c.Request.Context(), tenantID.(int))
	if err != nil {
		kc.logger.Errorf("获取知识库分类失败: %v", err)
		common.Fail(c, 5001, "获取分类失败: "+err.Error())
		return
	}

	common.Success(c, categories)
}
