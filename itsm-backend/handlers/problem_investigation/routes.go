package probleminvestigation

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
)

// tenantID 提取租户上下文；沿用 handlerctx 契约（401 语义），
// 与旧 controller 的 c.GetInt("tenant_id") 行为等价（迁移前无显式零值检查，
// handlerctx.RequireTenantID 的 401 守卫是行为收窄，属于预期的加固）。
func tenantID(c *gin.Context) (int, bool) {
	return handlerctx.RequireTenantID(c)
}

// userID 提取当前用户上下文（迁移前 controller 直接 c.GetInt，无校验；
// 此处保持一致以避免行为变化，调用方自行确保中间件已注入）。
func userID(c *gin.Context) int {
	return c.GetInt("user_id")
}

func pathID(c *gin.Context, field string, label string) (int, bool) {
	id, err := strconv.Atoi(c.Param(field))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的"+label+"ID")
		return 0, false
	}
	return id, true
}

// CreateProblemInvestigation 创建问题调查
func (h *Handler) CreateProblemInvestigation(c *gin.Context) {
	var req dto.CreateProblemInvestigationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}
	uid := userID(c)

	// 设置调查者ID（如果未指定，则使用当前用户）
	if req.InvestigatorID == 0 {
		req.InvestigatorID = uid
	}

	investigation, err := h.invService.CreateProblemInvestigation(c.Request.Context(), &req, tid)
	if err != nil {
		h.logger.Errorw("Create problem investigation failed", "error", err, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"message":         "问题调查创建成功",
		"investigationId": investigation.ID,
		"investigation":   investigation,
	})
}

// GetProblemInvestigation 获取问题调查详情
func (h *Handler) GetProblemInvestigation(c *gin.Context) {
	investigationID, ok := pathID(c, "id", "调查")
	if !ok {
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	investigation, err := h.invService.GetProblemInvestigation(c.Request.Context(), investigationID, tid)
	if err != nil {
		h.logger.Errorw("Get problem investigation failed", "error", err, "investigation_id", investigationID, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, investigation)
}

// UpdateProblemInvestigation 更新问题调查
func (h *Handler) UpdateProblemInvestigation(c *gin.Context) {
	investigationID, ok := pathID(c, "id", "调查")
	if !ok {
		return
	}

	var req dto.UpdateProblemInvestigationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	investigation, err := h.invService.UpdateProblemInvestigation(c.Request.Context(), investigationID, &req, tid)
	if err != nil {
		h.logger.Errorw("Update problem investigation failed", "error", err, "investigation_id", investigationID, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"message":       "问题调查更新成功",
		"investigation": investigation,
	})
}

// CreateInvestigationStep 创建调查步骤
func (h *Handler) CreateInvestigationStep(c *gin.Context) {
	var req dto.CreateInvestigationStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	step, err := h.invService.CreateInvestigationStep(c.Request.Context(), &req, tid)
	if err != nil {
		h.logger.Errorw("Create investigation step failed", "error", err, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"message": "调查步骤创建成功",
		"step":    step,
	})
}

// UpdateInvestigationStep 更新调查步骤
func (h *Handler) UpdateInvestigationStep(c *gin.Context) {
	stepID, ok := pathID(c, "id", "步骤")
	if !ok {
		return
	}

	var req dto.UpdateInvestigationStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	step, err := h.invService.UpdateInvestigationStep(c.Request.Context(), stepID, &req, tid)
	if err != nil {
		h.logger.Errorw("Update investigation step failed", "error", err, "step_id", stepID, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"message": "调查步骤更新成功",
		"step":    step,
	})
}

// CreateRootCauseAnalysis 创建根本原因分析
func (h *Handler) CreateRootCauseAnalysis(c *gin.Context) {
	var req dto.CreateRootCauseAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}
	uid := userID(c)

	// 设置分析师ID（如果未指定，则使用当前用户）
	if req.AnalystID == 0 {
		req.AnalystID = uid
	}

	analysis, err := h.invService.CreateRootCauseAnalysis(c.Request.Context(), &req, tid)
	if err != nil {
		h.logger.Errorw("Create root cause analysis failed", "error", err, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"message":  "根本原因分析创建成功",
		"analysis": analysis,
	})
}

// UpdateRootCauseAnalysis 更新根本原因分析
func (h *Handler) UpdateRootCauseAnalysis(c *gin.Context) {
	analysisID, ok := pathID(c, "id", "分析")
	if !ok {
		return
	}

	var req dto.UpdateRootCauseAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	analysis, err := h.invService.UpdateRootCauseAnalysis(c.Request.Context(), analysisID, &req, tid)
	if err != nil {
		h.logger.Errorw("Update root cause analysis failed", "error", err, "analysis_id", analysisID, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"message":  "根本原因分析更新成功",
		"analysis": analysis,
	})
}

// CreateProblemSolution 创建问题解决方案
func (h *Handler) CreateProblemSolution(c *gin.Context) {
	var req dto.CreateProblemSolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}
	uid := userID(c)

	// 设置提议者ID（如果未指定，则使用当前用户）
	if req.ProposedBy == 0 {
		req.ProposedBy = uid
	}

	solution, err := h.invService.CreateProblemSolution(c.Request.Context(), &req, tid)
	if err != nil {
		h.logger.Errorw("Create problem solution failed", "error", err, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"message":  "问题解决方案创建成功",
		"solution": solution,
	})
}

// UpdateProblemSolution 更新问题解决方案
func (h *Handler) UpdateProblemSolution(c *gin.Context) {
	solutionID, ok := pathID(c, "id", "解决方案")
	if !ok {
		return
	}

	var req dto.UpdateProblemSolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	solution, err := h.invService.UpdateProblemSolution(c.Request.Context(), solutionID, &req, tid)
	if err != nil {
		h.logger.Errorw("Update problem solution failed", "error", err, "solution_id", solutionID, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"message":  "问题解决方案更新成功",
		"solution": solution,
	})
}

// GetProblemSolutions 获取问题解决方案列表
func (h *Handler) GetProblemSolutions(c *gin.Context) {
	problemID, ok := pathID(c, "id", "问题")
	if !ok {
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	summary, err := h.invService.GetProblemInvestigationSummary(c.Request.Context(), problemID, tid)
	if err != nil {
		h.logger.Errorw("Get problem solutions failed", "error", err, "problem_id", problemID, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"problemId": problemID,
		"solutions": summary.Solutions,
	})
}

// GetProblemInvestigationSummary 获取问题调查摘要
func (h *Handler) GetProblemInvestigationSummary(c *gin.Context) {
	problemID, ok := pathID(c, "id", "问题")
	if !ok {
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	summary, err := h.invService.GetProblemInvestigationSummary(c.Request.Context(), problemID, tid)
	if err != nil {
		h.logger.Errorw("Get problem investigation summary failed", "error", err, "problem_id", problemID, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, summary)
}

// GetInvestigationSteps 获取调查步骤列表
func (h *Handler) GetInvestigationSteps(c *gin.Context) {
	investigationID, ok := pathID(c, "id", "调查")
	if !ok {
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	// 检查调查记录是否存在
	investigation, err := h.invService.GetProblemInvestigation(c.Request.Context(), investigationID, tid)
	if err != nil {
		h.logger.Errorw("Get investigation steps failed - investigation not found", "error", err, "investigation_id", investigationID, "tenant_id", tid)
		common.Fail(c, common.NotFoundCode, "调查记录不存在")
		return
	}

	// 获取调查步骤
	summary, err := h.invService.GetProblemInvestigationSummary(c.Request.Context(), investigation.ProblemID, tid)
	if err != nil {
		h.logger.Errorw("Get investigation steps failed", "error", err, "investigation_id", investigationID, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"investigationId": investigationID,
		"steps":           summary.Steps,
	})
}

// CreateProblemRelationship 创建问题关联
func (h *Handler) CreateProblemRelationship(c *gin.Context) {
	var req dto.CreateProblemRelationshipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	// 创建问题关联记录（沿用旧实现的简化处理：记录日志即返回成功，
	// 未落库是迁移前既有行为，保持不变）
	h.logger.Info("Creating problem relationship",
		"problem_id", req.ProblemID,
		"related_type", req.RelatedType,
		"related_id", req.RelatedID,
		"tenant_id", tid)

	common.Success(c, gin.H{
		"message":     "问题关联创建成功",
		"problemId":   req.ProblemID,
		"relatedType": req.RelatedType,
		"relatedId":   req.RelatedID,
	})
}

// GetProblemRelationships 获取问题关联列表
func (h *Handler) GetProblemRelationships(c *gin.Context) {
	problemID, ok := pathID(c, "id", "问题")
	if !ok {
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	// 沿用旧实现的简化处理：返回空列表（迁移前既有行为，保持不变）
	h.logger.Info("Getting problem relationships", "problem_id", problemID, "tenant_id", tid)

	common.Success(c, gin.H{
		"problemId":     problemID,
		"relationships": []interface{}{},
	})
}

// CreateKnowledgeArticle 创建知识库文章
func (h *Handler) CreateKnowledgeArticle(c *gin.Context) {
	var req dto.CreateProblemKnowledgeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}
	uid := userID(c)

	article, err := h.invService.CreateKnowledgeArticle(c.Request.Context(), tid, uid, &req)
	if err != nil {
		h.logger.Errorw("Create knowledge article failed", "error", err, "tenant_id", tid)
		common.FailWithErr(c, err, "操作失败")
		return
	}

	h.logger.Info("Created knowledge article from problem", "article_id", article.ID, "tenant_id", tid)

	common.Success(c, gin.H{
		"message":   "知识库文章创建成功",
		"articleId": article.ID,
		"article": gin.H{
			"id":       article.ID,
			"title":    article.Title,
			"content":  article.Content,
			"category": article.Category,
			"tags":     article.Tags,
			"authorId": article.AuthorID,
		},
	})
}

// GetProblemKnowledgeArticles 获取问题知识库文章列表
func (h *Handler) GetProblemKnowledgeArticles(c *gin.Context) {
	problemID, ok := pathID(c, "id", "问题")
	if !ok {
		return
	}

	tid, ok := tenantID(c)
	if !ok {
		return
	}

	// 获取知识库文章列表（按创建时间倒序；沿用旧实现未按 problem 过滤的行为）
	articles, err := h.invService.ListKnowledgeArticles(c.Request.Context(), tid)
	if err != nil {
		h.logger.Errorw("Get knowledge articles failed", "error", err, "tenant_id", tid)
		common.Fail(c, common.InternalErrorCode, "获取知识库文章列表失败")
		return
	}

	var result []gin.H
	for _, a := range articles {
		result = append(result, gin.H{
			"id":        a.ID,
			"title":     a.Title,
			"content":   a.Content,
			"category":  a.Category,
			"tags":      a.Tags,
			"authorId":  a.AuthorID,
			"viewCount": a.ViewCount,
			"likeCount": a.LikeCount,
			"createdAt": a.CreatedAt,
		})
	}

	common.Success(c, gin.H{
		"problemId":         problemID,
		"knowledgeArticles": result,
	})
}
