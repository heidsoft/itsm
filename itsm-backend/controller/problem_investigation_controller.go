package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

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
