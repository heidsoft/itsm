package controller

import (
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

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
	tenantIDValue, exists := c.Get("tenant_id")
	if !exists || tenantIDValue == nil {
		common.Fail(c, 1001, "租户上下文不存在")
		return
	}
	tenantID := tenantIDValue.(int)
	userId, _ := c.Get("user_id")

	article, err := kc.knowledgeService.CreateArticle(c.Request.Context(), &req, tenantID, userId.(int))
	if err != nil {
		kc.logger.Errorf("创建知识库文章失败: %v", err)
		common.Fail(c, 5001, "创建文章失败: "+err.Error())
		return
	}

	common.Success(c, dto.ToKnowledgeArticleResponse(article))
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

	tenantIDValue, exists := c.Get("tenant_id")
	if !exists || tenantIDValue == nil {
		common.Fail(c, 1001, "租户上下文不存在")
		return
	}
	tenantID := tenantIDValue.(int)

	article, err := kc.knowledgeService.GetArticle(c.Request.Context(), id, tenantID)
	if err != nil {
		kc.logger.Errorf("获取知识库文章失败: %v", err)
		common.Fail(c, 5001, "获取文章失败: "+err.Error())
		return
	}

	common.Success(c, dto.ToKnowledgeArticleResponse(article))
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

	tenantIDValue, exists := c.Get("tenant_id")
	if !exists || tenantIDValue == nil {
		common.Fail(c, 1001, "租户上下文不存在")
		return
	}
	tenantID := tenantIDValue.(int)

	articles, total, err := kc.knowledgeService.ListArticles(c.Request.Context(), &req, tenantID)
	if err != nil {
		kc.logger.Errorf("获取知识库文章列表失败: %v", err)
		common.Fail(c, 5001, "获取文章列表失败: "+err.Error())
		return
	}

	// 转换响应格式
	articleResponses := make([]dto.KnowledgeArticleResponse, 0, len(articles))
	for _, article := range articles {
		resp := dto.ToKnowledgeArticleResponse(article)
		if resp != nil {
			articleResponses = append(articleResponses, *resp)
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

	tenantIDValue, exists := c.Get("tenant_id")
	if !exists || tenantIDValue == nil {
		common.Fail(c, 1001, "租户上下文不存在")
		return
	}
	tenantID := tenantIDValue.(int)

	article, err := kc.knowledgeService.UpdateArticle(c.Request.Context(), id, &req, tenantID)
	if err != nil {
		kc.logger.Errorf("更新知识库文章失败: %v", err)
		common.Fail(c, 5001, "更新文章失败: "+err.Error())
		return
	}

	common.Success(c, dto.ToKnowledgeArticleResponse(article))
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

	tenantIDValue, exists := c.Get("tenant_id")
	if !exists || tenantIDValue == nil {
		common.Fail(c, 1001, "租户上下文不存在")
		return
	}
	tenantID := tenantIDValue.(int)

	err = kc.knowledgeService.DeleteArticle(c.Request.Context(), id, tenantID)
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
	tenantIDValue, exists := c.Get("tenant_id")
	if !exists || tenantIDValue == nil {
		common.Fail(c, 1001, "租户上下文不存在")
		return
	}
	tenantID := tenantIDValue.(int)

	categories, err := kc.knowledgeService.GetCategories(c.Request.Context(), tenantID)
	if err != nil {
		kc.logger.Errorf("获取知识库分类失败: %v", err)
		common.Fail(c, 5001, "获取分类失败: "+err.Error())
		return
	}

	common.Success(c, categories)
}

// ============ 版本历史API ============

// ListVersions 获取文章版本列表
// @Summary 获取文章版本列表
// @Description 获取指定文章的版本历史
// @Tags 知识库-版本
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(20)
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleVersionListResponse}
// @Router /api/v1/knowledge-articles/{id}/versions [get]
func (kc *KnowledgeController) ListVersions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的文章ID")
		return
	}

	var req dto.ListArticleVersionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		// 使用默认值
		req.Page = 1
		req.PageSize = 20
	}

	tenantIDValue, exists := c.Get("tenant_id")
	if !exists || tenantIDValue == nil {
		common.Fail(c, 1001, "租户上下文不存在")
		return
	}
	tenantID := tenantIDValue.(int)

	versions, total, err := kc.knowledgeService.ListVersions(c.Request.Context(), id, tenantID, req.Page, req.PageSize)
	if err != nil {
		kc.logger.Errorf("获取版本列表失败: %v", err)
		common.Fail(c, 5001, err.Error())
		return
	}

	// 转换响应
	versionResponses := make([]dto.KnowledgeArticleVersionResponse, 0, len(versions))
	for _, v := range versions {
		tags := []string{}
		if v.Tags != "" {
			tags = strings.Split(v.Tags, ",")
		}
		versionResponses = append(versionResponses, dto.KnowledgeArticleVersionResponse{
			ID:            v.ID,
			ArticleID:     v.ArticleID,
			Version:       v.Version,
			Title:         v.Title,
			Content:       v.Content,
			Category:      v.Category,
			Tags:          tags,
			AuthorID:      v.AuthorID,
			ChangeSummary: v.ChangeSummary,
			CreatedAt:     v.CreatedAt,
		})
	}

	common.Success(c, dto.KnowledgeArticleVersionListResponse{
		Versions: versionResponses,
		Total:    total,
		Page:     req.Page,
		Size:     req.PageSize,
	})
}

// GetVersion 获取指定版本
// @Summary 获取指定版本
// @Description 获取文章的指定版本详情
// @Tags 知识库-版本
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Param version path int true "版本号"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleVersionResponse}
// @Router /api/v1/knowledge-articles/{id}/versions/{version} [get]
func (kc *KnowledgeController) GetVersion(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的文章ID")
		return
	}

	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		common.Fail(c, 1001, "无效的版本号")
		return
	}

	tenantIDValue, exists := c.Get("tenant_id")
	if !exists || tenantIDValue == nil {
		common.Fail(c, 1001, "租户上下文不存在")
		return
	}
	tenantID := tenantIDValue.(int)

	versionEntity, err := kc.knowledgeService.GetVersion(c.Request.Context(), id, version, tenantID)
	if err != nil {
		kc.logger.Errorf("获取版本失败: %v", err)
		common.Fail(c, 5001, err.Error())
		return
	}

	tags := []string{}
	if versionEntity.Tags != "" {
		tags = strings.Split(versionEntity.Tags, ",")
	}

	common.Success(c, dto.KnowledgeArticleVersionResponse{
		ID:            versionEntity.ID,
		ArticleID:     versionEntity.ArticleID,
		Version:       versionEntity.Version,
		Title:         versionEntity.Title,
		Content:       versionEntity.Content,
		Category:      versionEntity.Category,
		Tags:          tags,
		AuthorID:      versionEntity.AuthorID,
		ChangeSummary: versionEntity.ChangeSummary,
		CreatedAt:     versionEntity.CreatedAt,
	})
}

// RestoreVersion 恢复文章到指定版本
// @Summary 恢复文章到指定版本
// @Description 将文章恢复到指定的历史版本
// @Tags 知识库-版本
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Param body body dto.RestoreArticleVersionRequest true "版本信息"
// @Success 200 {object} common.Response{data=dto.KnowledgeArticleResponse}
// @Router /api/v1/knowledge-articles/{id}/versions/restore [post]
func (kc *KnowledgeController) RestoreVersion(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的文章ID")
		return
	}

	var req dto.RestoreArticleVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	tenantIDValue, exists := c.Get("tenant_id")
	if !exists || tenantIDValue == nil {
		common.Fail(c, 1001, "租户上下文不存在")
		return
	}
	tenantID := tenantIDValue.(int)

	userId, _ := c.Get("user_id")

	article, err := kc.knowledgeService.RestoreVersion(c.Request.Context(), id, req.Version, tenantID, userId.(int))
	if err != nil {
		kc.logger.Errorf("恢复版本失败: %v", err)
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, dto.ToKnowledgeArticleResponse(article))
}

// ============ 实时协作Session API ============

// GetSession 获取当前会话
// @Summary 获取当前会话
// @Description 获取文章的当前协作会话
// @Tags 知识库-协作
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Success 200 {object} common.Response{data=dto.ArticleSessionResponse}
// @Router /api/v1/knowledge-articles/{id}/session [get]
func (kc *KnowledgeController) GetSession(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的文章ID")
		return
	}

	tenantIDValue, exists := c.Get("tenant_id")
	if !exists || tenantIDValue == nil {
		common.Fail(c, 1001, "租户上下文不存在")
		return
	}
	tenantID := tenantIDValue.(int)

	userId, _ := c.Get("user_id")

	session, err := kc.knowledgeService.GetSession(c.Request.Context(), id, userId.(int), tenantID)
	if err != nil {
		kc.logger.Errorf("获取会话失败: %v", err)
		common.Fail(c, 5001, err.Error())
		return
	}

	if session == nil {
		common.Success(c, nil)
		return
	}

	common.Success(c, dto.ArticleSessionResponse{
		SessionID:     session.ID,
		ArticleID:     session.ArticleID,
		UserID:        session.UserID,
		SessionToken:  session.SessionToken,
		Status:        string(session.Status),
		LastHeartbeat: session.LastHeartbeat,
		CreatedAt:     session.CreatedAt,
	})
}

// CreateOrJoinSession 创建或加入会话
// @Summary 创建或加入会话
// @Description 创建新会话或加入现有会话
// @Tags 知识库-协作
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Success 200 {object} common.Response{data=dto.ArticleSessionResponse}
// @Router /api/v1/knowledge-articles/{id}/session [post]
func (kc *KnowledgeController) CreateOrJoinSession(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的文章ID")
		return
	}

	tenantIDValue, exists := c.Get("tenant_id")
	if !exists || tenantIDValue == nil {
		common.Fail(c, 1001, "租户上下文不存在")
		return
	}
	tenantID := tenantIDValue.(int)

	userId, _ := c.Get("user_id")

	session, err := kc.knowledgeService.CreateSession(c.Request.Context(), id, userId.(int), tenantID)
	if err != nil {
		kc.logger.Errorf("创建会话失败: %v", err)
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, dto.ArticleSessionResponse{
		SessionID:     session.ID,
		ArticleID:     session.ArticleID,
		UserID:        session.UserID,
		SessionToken:  session.SessionToken,
		Status:        string(session.Status),
		LastHeartbeat: session.LastHeartbeat,
		CreatedAt:     session.CreatedAt,
	})
}

// Heartbeat 心跳保活
// @Summary 心跳保活
// @Description 更新会话心跳
// @Tags 知识库-协作
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Param body body dto.SessionHeartbeatRequest true "心跳信息"
// @Success 200 {object} common.Response
// @Router /api/v1/knowledge-articles/{id}/session/heartbeat [post]
func (kc *KnowledgeController) Heartbeat(c *gin.Context) {
	var req dto.SessionHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	err := kc.knowledgeService.Heartbeat(c.Request.Context(), req.SessionToken, req.CursorPos)
	if err != nil {
		kc.logger.Errorf("心跳失败: %v", err)
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, nil)
}

// GetParticipants 获取参与者列表
// @Summary 获取参与者列表
// @Description 获取文章的当前参与者列表
// @Tags 知识库-协作
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Success 200 {object} common.Response{data=dto.ListParticipantsResponse}
// @Router /api/v1/knowledge-articles/{id}/participants [get]
func (kc *KnowledgeController) GetParticipants(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的文章ID")
		return
	}

	tenantIDValue, exists := c.Get("tenant_id")
	if !exists || tenantIDValue == nil {
		common.Fail(c, 1001, "租户上下文不存在")
		return
	}
	tenantID := tenantIDValue.(int)

	sessions, err := kc.knowledgeService.ListParticipants(c.Request.Context(), id, tenantID)
	if err != nil {
		kc.logger.Errorf("获取参与者失败: %v", err)
		common.Fail(c, 5001, err.Error())
		return
	}

	// 转换响应
	participants := make([]dto.ArticleParticipantResponse, 0, len(sessions))
	for _, s := range sessions {
		participants = append(participants, dto.ArticleParticipantResponse{
			UserID:   s.UserID,
			IsActive: s.Status == "active",
			JoinedAt: s.CreatedAt,
		})
	}

	common.Success(c, dto.ListParticipantsResponse{
		Participants: participants,
		Total:        len(participants),
	})
}

// LeaveSession 离开会话
// @Summary 离开会话
// @Description 离开当前协作会话
// @Tags 知识库-协作
// @Accept json
// @Produce json
// @Param id path int true "文章ID"
// @Success 200 {object} common.Response
// @Router /api/v1/knowledge-articles/{id}/session [delete]
func (kc *KnowledgeController) LeaveSession(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, 1001, "无效的文章ID")
		return
	}

	userId, _ := c.Get("user_id")

	err = kc.knowledgeService.LeaveSession(c.Request.Context(), id, userId.(int))
	if err != nil {
		kc.logger.Errorf("离开会话失败: %v", err)
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, nil)
}
