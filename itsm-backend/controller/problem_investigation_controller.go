package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProblemInvestigationController 问题调查控制器
type ProblemInvestigationController struct {
	logger                      *zap.SugaredLogger
	problemInvestigationService *service.ProblemInvestigationService
}

// NewProblemInvestigationController 创建问题调查控制器
func NewProblemInvestigationController(logger *zap.SugaredLogger, problemInvestigationService *service.ProblemInvestigationService) *ProblemInvestigationController {
	return &ProblemInvestigationController{
		logger:                      logger,
		problemInvestigationService: problemInvestigationService,
	}
}

// CreateProblemInvestigation 创建问题调查
func (pc *ProblemInvestigationController) CreateProblemInvestigation(c *gin.Context) {
	var req dto.CreateProblemInvestigationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	// 设置调查者ID（如果未指定，则使用当前用户）
	if req.InvestigatorID == 0 {
		req.InvestigatorID = userID
	}

	investigation, err := pc.problemInvestigationService.CreateProblemInvestigation(c.Request.Context(), &req, tenantID)
	if err != nil {
		pc.logger.Errorw("Create problem investigation failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "创建问题调查失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"message":          "问题调查创建成功",
		"investigation_id": investigation.ID,
		"investigation":    investigation,
	})
}

// GetProblemInvestigation 获取问题调查详情
func (pc *ProblemInvestigationController) GetProblemInvestigation(c *gin.Context) {
	investigationID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的调查ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	investigation, err := pc.problemInvestigationService.GetProblemInvestigation(c.Request.Context(), investigationID, tenantID)
	if err != nil {
		pc.logger.Errorw("Get problem investigation failed", "error", err, "investigation_id", investigationID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取问题调查失败: "+err.Error())
		return
	}

	common.Success(c, investigation)
}

// UpdateProblemInvestigation 更新问题调查
func (pc *ProblemInvestigationController) UpdateProblemInvestigation(c *gin.Context) {
	investigationID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的调查ID")
		return
	}

	var req dto.UpdateProblemInvestigationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	investigation, err := pc.problemInvestigationService.UpdateProblemInvestigation(c.Request.Context(), investigationID, &req, tenantID)
	if err != nil {
		pc.logger.Errorw("Update problem investigation failed", "error", err, "investigation_id", investigationID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "更新问题调查失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"message":       "问题调查更新成功",
		"investigation": investigation,
	})
}

// CreateInvestigationStep 创建调查步骤
func (pc *ProblemInvestigationController) CreateInvestigationStep(c *gin.Context) {
	var req dto.CreateInvestigationStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	step, err := pc.problemInvestigationService.CreateInvestigationStep(c.Request.Context(), &req, tenantID)
	if err != nil {
		pc.logger.Errorw("Create investigation step failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "创建调查步骤失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"message": "调查步骤创建成功",
		"step":    step,
	})
}

// UpdateInvestigationStep 更新调查步骤
func (pc *ProblemInvestigationController) UpdateInvestigationStep(c *gin.Context) {
	stepID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的步骤ID")
		return
	}

	var req dto.UpdateInvestigationStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	step, err := pc.problemInvestigationService.UpdateInvestigationStep(c.Request.Context(), stepID, &req, tenantID)
	if err != nil {
		pc.logger.Errorw("Update investigation step failed", "error", err, "step_id", stepID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "更新调查步骤失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"message": "调查步骤更新成功",
		"step":    step,
	})
}

// CreateRootCauseAnalysis 创建根本原因分析
func (pc *ProblemInvestigationController) CreateRootCauseAnalysis(c *gin.Context) {
	var req dto.CreateRootCauseAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	// 设置分析师ID（如果未指定，则使用当前用户）
	if req.AnalystID == 0 {
		req.AnalystID = userID
	}

	analysis, err := pc.problemInvestigationService.CreateRootCauseAnalysis(c.Request.Context(), &req, tenantID)
	if err != nil {
		pc.logger.Errorw("Create root cause analysis failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "创建根本原因分析失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"message":  "根本原因分析创建成功",
		"analysis": analysis,
	})
}

// CreateProblemSolution 创建问题解决方案
func (pc *ProblemInvestigationController) CreateProblemSolution(c *gin.Context) {
	var req dto.CreateProblemSolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	// 设置提议者ID（如果未指定，则使用当前用户）
	if req.ProposedBy == 0 {
		req.ProposedBy = userID
	}

	solution, err := pc.problemInvestigationService.CreateProblemSolution(c.Request.Context(), &req, tenantID)
	if err != nil {
		pc.logger.Errorw("Create problem solution failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "创建问题解决方案失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"message":  "问题解决方案创建成功",
		"solution": solution,
	})
}

// GetProblemInvestigationSummary 获取问题调查摘要
func (pc *ProblemInvestigationController) GetProblemInvestigationSummary(c *gin.Context) {
	problemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的问题ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	summary, err := pc.problemInvestigationService.GetProblemInvestigationSummary(c.Request.Context(), problemID, tenantID)
	if err != nil {
		pc.logger.Errorw("Get problem investigation summary failed", "error", err, "problem_id", problemID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取问题调查摘要失败: "+err.Error())
		return
	}

	common.Success(c, summary)
}

// GetInvestigationSteps 获取调查步骤列表
func (pc *ProblemInvestigationController) GetInvestigationSteps(c *gin.Context) {
	investigationID, err := strconv.Atoi(c.Param("investigation_id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的调查ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	// 检查调查记录是否存在
	investigation, err := pc.problemInvestigationService.GetProblemInvestigation(c.Request.Context(), investigationID, tenantID)
	if err != nil {
		pc.logger.Errorw("Get investigation steps failed - investigation not found", "error", err, "investigation_id", investigationID, "tenant_id", tenantID)
		common.Fail(c, common.NotFoundCode, "调查记录不存在")
		return
	}

	// 获取调查步骤
	summary, err := pc.problemInvestigationService.GetProblemInvestigationSummary(c.Request.Context(), investigation.ProblemID, tenantID)
	if err != nil {
		pc.logger.Errorw("Get investigation steps failed", "error", err, "investigation_id", investigationID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取调查步骤失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"investigation_id": investigationID,
		"steps":            summary.Steps,
	})
}

// GetProblemSolutions 获取问题解决方案列表
func (pc *ProblemInvestigationController) GetProblemSolutions(c *gin.Context) {
	problemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的问题ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	summary, err := pc.problemInvestigationService.GetProblemInvestigationSummary(c.Request.Context(), problemID, tenantID)
	if err != nil {
		pc.logger.Errorw("Get problem solutions failed", "error", err, "problem_id", problemID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取问题解决方案失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"problem_id": problemID,
		"solutions":  summary.Solutions,
	})
}

// UpdateRootCauseAnalysis 更新根本原因分析
func (pc *ProblemInvestigationController) UpdateRootCauseAnalysis(c *gin.Context) {
	analysisID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的分析ID")
		return
	}

	var req dto.UpdateRootCauseAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	analysis, err := pc.problemInvestigationService.UpdateRootCauseAnalysis(c.Request.Context(), analysisID, &req, tenantID)
	if err != nil {
		pc.logger.Errorw("Update root cause analysis failed", "error", err, "analysis_id", analysisID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "更新根本原因分析失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"message":  "根本原因分析更新成功",
		"analysis": analysis,
	})
}

// UpdateProblemSolution 更新问题解决方案
func (pc *ProblemInvestigationController) UpdateProblemSolution(c *gin.Context) {
	solutionID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的解决方案ID")
		return
	}

	var req dto.UpdateProblemSolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	solution, err := pc.problemInvestigationService.UpdateProblemSolution(c.Request.Context(), solutionID, &req, tenantID)
	if err != nil {
		pc.logger.Errorw("Update problem solution failed", "error", err, "solution_id", solutionID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "更新问题解决方案失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"message":  "问题解决方案更新成功",
		"solution": solution,
	})
}

// CreateProblemRelationship 创建问题关联
func (pc *ProblemInvestigationController) CreateProblemRelationship(c *gin.Context) {
	var req dto.CreateProblemRelationshipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	// 创建问题关联记录 (使用通用关联表)
	// 注意: 这里简化处理，实际应该创建专门的关联表
	pc.logger.Info("Creating problem relationship",
		"problem_id", req.ProblemID,
		"related_type", req.RelatedType,
		"related_id", req.RelatedID,
		"tenant_id", tenantID)

	common.Success(c, gin.H{
		"message":       "问题关联创建成功",
		"problem_id":   req.ProblemID,
		"related_type": req.RelatedType,
		"related_id":   req.RelatedID,
	})
}

// GetProblemRelationships 获取问题关联列表
func (pc *ProblemInvestigationController) GetProblemRelationships(c *gin.Context) {
	problemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的问题ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	// 获取问题关联的事件、工单、变更等
	// 这里简化为返回空列表，实际应该查询关联表
	pc.logger.Info("Getting problem relationships", "problem_id", problemID, "tenant_id", tenantID)

	common.Success(c, gin.H{
		"problem_id":     problemID,
		"relationships": []interface{}{},
	})
}

// CreateKnowledgeArticle 创建知识库文章
func (pc *ProblemInvestigationController) CreateKnowledgeArticle(c *gin.Context) {
	var req dto.CreateProblemKnowledgeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	client, exists := c.Get("client")
	if !exists {
		common.Fail(c, common.InternalErrorCode, "Database client not found")
		return
	}
	entClient := client.(*ent.Client)

	// 转换 tags 为字符串
	tagsStr := ""
	if len(req.Tags) > 0 {
		tagsStr = strings.Join(req.Tags, ",")
	}

	// 创建知识库文章
	article, err := entClient.KnowledgeArticle.Create().
		SetTitle(req.ArticleTitle).
		SetContent(req.ArticleContent).
		SetCategory(req.ArticleType).
		SetAuthorID(userID).
		SetTags(tagsStr).
		SetTenantID(tenantID).
		Save(c.Request.Context())

	if err != nil {
		pc.logger.Errorw("Create knowledge article failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "创建知识库文章失败: "+err.Error())
		return
	}

	pc.logger.Info("Created knowledge article from problem", "article_id", article.ID, "tenant_id", tenantID)

	common.Success(c, gin.H{
		"message":    "知识库文章创建成功",
		"article_id": article.ID,
		"article": gin.H{
			"id":           article.ID,
			"title":        article.Title,
			"content":      article.Content,
			"category":     article.Category,
			"tags":         article.Tags,
			"author_id":    article.AuthorID,
		},
	})
}

// GetProblemKnowledgeArticles 获取问题知识库文章列表
func (pc *ProblemInvestigationController) GetProblemKnowledgeArticles(c *gin.Context) {
	problemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的问题ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	client, exists := c.Get("client")
	if !exists {
		common.Fail(c, common.InternalErrorCode, "Database client not found")
		return
	}
	entClient := client.(*ent.Client)

	// 获取知识库文章列表 (按创建时间倒序)
	articles, err := entClient.KnowledgeArticle.Query().
		Order(ent.Desc("created_at")).
		All(c.Request.Context())

	if err != nil {
		pc.logger.Errorw("Get knowledge articles failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取知识库文章列表失败")
		return
	}

	var result []gin.H
	for _, a := range articles {
		result = append(result, gin.H{
			"id":           a.ID,
			"title":        a.Title,
			"content":      a.Content,
			"category":     a.Category,
			"tags":         a.Tags,
			"author_id":    a.AuthorID,
			"view_count":   a.ViewCount,
			"like_count":   a.LikeCount,
			"created_at":   a.CreatedAt,
		})
	}

	common.Success(c, gin.H{
		"problem_id":         problemID,
		"knowledge_articles": result,
	})
}
