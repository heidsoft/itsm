package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProblemController 问题管理控制器
type ProblemController struct {
	logger         *zap.SugaredLogger
	problemService *service.ProblemService
}

// NewProblemController 创建问题管理控制器
func NewProblemController(logger *zap.SugaredLogger, problemService *service.ProblemService) *ProblemController {
	return &ProblemController{
		logger:         logger,
		problemService: problemService,
	}
}

// CreateProblem 创建问题
func (pc *ProblemController) CreateProblem(c *gin.Context) {
	var req dto.CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	// 设置创建者ID
	req.CreatedBy = userID

	problem, err := pc.problemService.CreateProblem(c.Request.Context(), &req, tenantID)
	if err != nil {
		pc.logger.Errorw("Create problem failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "创建问题失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"message":    "问题创建成功",
		"problem_id": problem.ID,
	})
}

// GetProblem 获取问题详情
func (pc *ProblemController) GetProblem(c *gin.Context) {
	problemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的问题ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	problem, err := pc.problemService.GetProblem(c.Request.Context(), problemID, tenantID)
	if err != nil {
		pc.logger.Errorw("Get problem failed", "error", err, "problem_id", problemID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取问题失败: "+err.Error())
		return
	}

	common.Success(c, problem)
}

// ListProblems 获取问题列表
func (pc *ProblemController) ListProblems(c *gin.Context) {
	var req dto.ListProblemsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	response, err := pc.problemService.ListProblems(c.Request.Context(), &req, tenantID)
	if err != nil {
		pc.logger.Errorw("List problems failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取问题列表失败: "+err.Error())
		return
	}

	common.Success(c, response)
}

// UpdateProblem 更新问题
func (pc *ProblemController) UpdateProblem(c *gin.Context) {
	problemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的问题ID")
		return
	}

	var req dto.UpdateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")

	problem, err := pc.problemService.UpdateProblem(c.Request.Context(), problemID, &req, tenantID)
	if err != nil {
		pc.logger.Errorw("Update problem failed", "error", err, "problem_id", problemID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "更新问题失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"message":    "问题更新成功",
		"problem_id": problem.ID,
	})
}

// DeleteProblem 删除问题
func (pc *ProblemController) DeleteProblem(c *gin.Context) {
	problemID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的问题ID")
		return
	}

	tenantID := c.GetInt("tenant_id")

	err = pc.problemService.DeleteProblem(c.Request.Context(), problemID, tenantID)
	if err != nil {
		pc.logger.Errorw("Delete problem failed", "error", err, "problem_id", problemID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "删除问题失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"message":    "问题删除成功",
		"problem_id": problemID,
	})
}

// GetProblemStats 获取问题统计
func (pc *ProblemController) GetProblemStats(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	stats, err := pc.problemService.GetProblemStats(c.Request.Context(), tenantID)
	if err != nil {
		pc.logger.Errorw("Get problem stats failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取问题统计失败: "+err.Error())
		return
	}

	common.Success(c, stats)
}
