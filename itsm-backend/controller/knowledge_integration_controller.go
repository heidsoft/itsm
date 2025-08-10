package controller

import (
	"itsm-backend/common"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// KnowledgeIntegrationController 知识库集成控制器
type KnowledgeIntegrationController struct {
	knowledgeIntegrationService *service.KnowledgeIntegrationService
	logger                      *zap.SugaredLogger
}

// NewKnowledgeIntegrationController 创建知识库集成控制器实例
func NewKnowledgeIntegrationController(
	knowledgeIntegrationService *service.KnowledgeIntegrationService,
	logger *zap.SugaredLogger,
) *KnowledgeIntegrationController {
	return &KnowledgeIntegrationController{
		knowledgeIntegrationService: knowledgeIntegrationService,
		logger:                      logger,
	}
}

// RecommendSolutions 推荐解决方案
// @Summary 推荐解决方案
// @Description 基于工单内容推荐相关的知识库解决方案
// @Tags 知识库集成
// @Accept json
// @Produce json
// @Param ticket_id path int true "工单ID"
// @Param limit query int false "推荐数量限制" default(5)
// @Success 200 {object} common.Response{data=[]service.SolutionRecommendation}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/tickets/{ticket_id}/knowledge/recommendations [get]
func (kic *KnowledgeIntegrationController) RecommendSolutions(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("ticket_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	limitStr := c.DefaultQuery("limit", "5")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 5
	}

	recommendations, err := kic.knowledgeIntegrationService.RecommendSolutions(
		c.Request.Context(),
		ticketID,
		limit,
	)
	if err != nil {
		kic.logger.Errorw("推荐解决方案失败", "error", err, "ticket_id", ticketID)
		common.Fail(c, common.InternalErrorCode, "推荐解决方案失败: "+err.Error())
		return
	}

	common.Success(c, recommendations)
}

// AssociateWithKnowledge 关联知识库文章
// @Summary 关联知识库文章
// @Description 将工单与知识库文章建立关联关系
// @Tags 知识库集成
// @Accept json
// @Produce json
// @Param ticket_id path int true "工单ID"
// @Param request body service.AssociateWithKnowledgeRequest true "关联请求"
// @Success 200 {object} common.Response{data=service.KnowledgeAssociation}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/tickets/{ticket_id}/knowledge/associate [post]
func (kic *KnowledgeIntegrationController) AssociateWithKnowledge(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("ticket_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	var req service.AssociateWithKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	association, err := kic.knowledgeIntegrationService.AssociateWithKnowledge(
		c.Request.Context(),
		ticketID,
		req.ArticleID,
		req.AssociationType,
	)
	if err != nil {
		kic.logger.Errorw("关联知识库文章失败", "error", err, "ticket_id", ticketID, "article_id", req.ArticleID)
		common.Fail(c, common.InternalErrorCode, "关联知识库文章失败: "+err.Error())
		return
	}

	common.Success(c, association)
}

// GetAIRecommendations 获取AI辅助建议
// @Summary 获取AI辅助建议
// @Description 获取工单的AI辅助建议，包括分类、优先级、标签等
// @Tags 知识库集成
// @Accept json
// @Produce json
// @Param ticket_id path int true "工单ID"
// @Success 200 {object} common.Response{data=service.AIRecommendation}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/tickets/{ticket_id}/knowledge/ai-recommendations [get]
func (kic *KnowledgeIntegrationController) GetAIRecommendations(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("ticket_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	recommendation, err := kic.knowledgeIntegrationService.GetAIRecommendations(
		c.Request.Context(),
		ticketID,
	)
	if err != nil {
		kic.logger.Errorw("获取AI建议失败", "error", err, "ticket_id", ticketID)
		common.Fail(c, common.InternalErrorCode, "获取AI建议失败: "+err.Error())
		return
	}

	common.Success(c, recommendation)
}

// GetRelatedArticles 获取相关文章
// @Summary 获取相关文章
// @Description 获取与工单相关的知识库文章
// @Tags 知识库集成
// @Accept json
// @Produce json
// @Param ticket_id path int true "工单ID"
// @Param limit query int false "文章数量限制" default(10)
// @Success 200 {object} common.Response{data=[]ent.KnowledgeArticle}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/tickets/{ticket_id}/knowledge/related-articles [get]
func (kic *KnowledgeIntegrationController) GetRelatedArticles(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("ticket_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	articles, err := kic.knowledgeIntegrationService.GetRelatedArticles(
		c.Request.Context(),
		ticketID,
		limit,
	)
	if err != nil {
		kic.logger.Errorw("获取相关文章失败", "error", err, "ticket_id", ticketID)
		common.Fail(c, common.InternalErrorCode, "获取相关文章失败: "+err.Error())
		return
	}

	common.Success(c, articles)
}

// SearchKnowledgeBase 搜索知识库
// @Summary 搜索知识库
// @Description 基于关键词搜索知识库文章
// @Tags 知识库集成
// @Accept json
// @Produce json
// @Param request body service.SearchKnowledgeRequest true "搜索请求"
// @Success 200 {object} common.Response{data=[]ent.KnowledgeArticle}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/knowledge/search [post]
func (kic *KnowledgeIntegrationController) SearchKnowledgeBase(c *gin.Context) {
	var req service.SearchKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	articles, err := kic.knowledgeIntegrationService.SearchKnowledgeArticles(
		c.Request.Context(),
		req.Keywords,
		tenantID,
		req.Limit,
	)
	if err != nil {
		kic.logger.Errorw("搜索知识库失败", "error", err, "keywords", req.Keywords, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "搜索知识库失败: "+err.Error())
		return
	}

	common.Success(c, articles)
}

// GetKnowledgeAssociations 获取知识库关联
// @Summary 获取知识库关联
// @Description 获取工单的所有知识库关联
// @Tags 知识库集成
// @Accept json
// @Produce json
// @Param ticket_id path int true "工单ID"
// @Success 200 {object} common.Response{data=[]service.KnowledgeAssociation}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/tickets/{ticket_id}/knowledge/associations [get]
func (kic *KnowledgeIntegrationController) GetKnowledgeAssociations(c *gin.Context) {
	ticketID, err := strconv.Atoi(c.Param("ticket_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工单ID")
		return
	}

	associations, err := kic.knowledgeIntegrationService.GetKnowledgeAssociations(
		c.Request.Context(),
		ticketID,
	)
	if err != nil {
		kic.logger.Errorw("获取知识库关联失败", "error", err, "ticket_id", ticketID)
		common.Fail(c, common.InternalErrorCode, "获取知识库关联失败: "+err.Error())
		return
	}

	common.Success(c, associations)
}
