package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ChangeApprovalController 变更审批控制器
type ChangeApprovalController struct {
	changeApprovalService *service.ChangeApprovalService
	logger                *zap.SugaredLogger
}

// NewChangeApprovalController 创建变更审批控制器
func NewChangeApprovalController(changeApprovalService *service.ChangeApprovalService, logger *zap.SugaredLogger) *ChangeApprovalController {
	return &ChangeApprovalController{
		changeApprovalService: changeApprovalService,
		logger:                logger,
	}
}

// CreateChangeApproval 创建变更审批
// @Summary 创建变更审批
// @Description 为指定变更创建审批记录
// @Tags 变更审批
// @Accept json
// @Produce json
// @Param request body dto.CreateChangeApprovalRequest true "创建审批请求"
// @Success 200 {object} dto.ChangeApprovalResponse
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/changes/approvals [post]
func (c *ChangeApprovalController) CreateChangeApproval(ctx *gin.Context) {
	var req dto.CreateChangeApprovalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Failed to bind JSON", "error", err)
		common.Fail(ctx, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	// 从上下文获取租户ID和用户ID
	tenantID := ctx.GetInt("tenant_id")
	_ = ctx.GetInt("user_id") // 暂时不使用，但保留获取逻辑

	response, err := c.changeApprovalService.CreateChangeApproval(ctx, &req, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to create change approval", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "创建变更审批失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// UpdateChangeApproval 更新变更审批状态
// @Summary 更新变更审批状态
// @Description 更新指定审批记录的状态
// @Tags 变更审批
// @Accept json
// @Produce json
// @Param id path int true "审批ID"
// @Param request body dto.UpdateChangeApprovalRequest true "更新审批请求"
// @Success 200 {object} dto.ChangeApprovalResponse
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/changes/approvals/{id} [put]
func (c *ChangeApprovalController) UpdateChangeApproval(ctx *gin.Context) {
	approvalID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.BadRequestCode, "无效的审批ID")
		return
	}

	var req dto.UpdateChangeApprovalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Failed to bind JSON", "error", err)
		common.Fail(ctx, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	// 从上下文获取租户ID
	tenantID := ctx.GetInt("tenant_id")

	response, err := c.changeApprovalService.UpdateChangeApproval(ctx, approvalID, &req, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to update change approval", "error", err, "approval_id", approvalID)
		common.Fail(ctx, common.InternalErrorCode, "更新变更审批失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// CreateChangeApprovalWorkflow 创建变更审批工作流
// @Summary 创建变更审批工作流
// @Description 为指定变更创建审批工作流
// @Tags 变更审批
// @Accept json
// @Produce json
// @Param request body dto.ChangeApprovalWorkflowRequest true "创建审批工作流请求"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/changes/approval-workflows [post]
func (c *ChangeApprovalController) CreateChangeApprovalWorkflow(ctx *gin.Context) {
	var req dto.ChangeApprovalWorkflowRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Failed to bind JSON", "error", err)
		common.Fail(ctx, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	// 从上下文获取租户ID
	tenantID := ctx.GetInt("tenant_id")

	err := c.changeApprovalService.CreateChangeApprovalWorkflow(ctx, &req, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to create change approval workflow", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "创建变更审批工作流失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{"message": "Change approval workflow created successfully"})
}

// GetChangeApprovalSummary 获取变更审批摘要
// @Summary 获取变更审批摘要
// @Description 获取指定变更的审批摘要信息
// @Tags 变更审批
// @Produce json
// @Param id path int true "变更ID"
// @Success 200 {object} dto.ChangeApprovalSummary
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/changes/{id}/approval-summary [get]
func (c *ChangeApprovalController) GetChangeApprovalSummary(ctx *gin.Context) {
	changeID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.BadRequestCode, "无效的变更ID")
		return
	}

	// 从上下文获取租户ID
	tenantID := ctx.GetInt("tenant_id")

	summary, err := c.changeApprovalService.GetChangeApprovalSummary(ctx, changeID, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to get change approval summary", "error", err, "change_id", changeID)
		common.Fail(ctx, common.InternalErrorCode, "获取变更审批摘要失败: "+err.Error())
		return
	}

	common.Success(ctx, summary)
}

// CreateChangeRiskAssessment 创建变更风险评估
// @Summary 创建变更风险评估
// @Description 为指定变更创建风险评估
// @Tags 变更风险评估
// @Accept json
// @Produce json
// @Param request body dto.CreateChangeRiskAssessmentRequest true "创建风险评估请求"
// @Success 200 {object} dto.ChangeRiskAssessmentResponse
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/changes/risk-assessments [post]
func (c *ChangeApprovalController) CreateChangeRiskAssessment(ctx *gin.Context) {
	var req dto.CreateChangeRiskAssessmentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Failed to bind JSON", "error", err)
		common.Fail(ctx, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	// 从上下文获取租户ID
	tenantID := ctx.GetInt("tenant_id")

	response, err := c.changeApprovalService.CreateChangeRiskAssessment(ctx, &req, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to create risk assessment", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "创建变更风险评估失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// GetChangeRiskAssessment 获取变更风险评估
// @Summary 获取变更风险评估
// @Description 获取指定变更的风险评估信息
// @Tags 变更风险评估
// @Produce json
// @Param id path int true "变更ID"
// @Success 200 {object} dto.ChangeRiskAssessmentResponse
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/changes/{id}/risk-assessment [get]
func (c *ChangeApprovalController) GetChangeRiskAssessment(ctx *gin.Context) {
	changeID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.BadRequestCode, "无效的变更ID")
		return
	}

	// 从上下文获取租户ID
	tenantID := ctx.GetInt("tenant_id")

	assessment, err := c.changeApprovalService.GetChangeRiskAssessment(ctx, changeID, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to get risk assessment", "error", err, "change_id", changeID)
		common.Fail(ctx, common.InternalErrorCode, "获取变更风险评估失败: "+err.Error())
		return
	}

	common.Success(ctx, assessment)
}

// CreateChangeImplementationPlan 创建变更实施计划
// @Summary 创建变更实施计划
// @Description 为指定变更创建实施计划
// @Tags 变更实施计划
// @Accept json
// @Produce json
// @Param request body dto.CreateChangeImplementationPlanRequest true "创建实施计划请求"
// @Success 200 {object} dto.ChangeImplementationPlanResponse
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/changes/implementation-plans [post]
func (c *ChangeApprovalController) CreateChangeImplementationPlan(ctx *gin.Context) {
	var req dto.CreateChangeImplementationPlanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Failed to bind JSON", "error", err)
		common.Fail(ctx, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	// 从上下文获取租户ID
	tenantID := ctx.GetInt("tenant_id")

	response, err := c.changeApprovalService.CreateChangeImplementationPlan(ctx, &req, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to create implementation plan", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "创建变更实施计划失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// CreateChangeRollbackPlan 创建变更回滚计划
// @Summary 创建变更回滚计划
// @Description 为指定变更创建回滚计划
// @Tags 变更回滚计划
// @Accept json
// @Produce json
// @Param request body dto.CreateChangeRollbackPlanRequest true "创建回滚计划请求"
// @Success 200 {object} dto.ChangeRollbackPlanResponse
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/changes/rollback-plans [post]
func (c *ChangeApprovalController) CreateChangeRollbackPlan(ctx *gin.Context) {
	var req dto.CreateChangeRollbackPlanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Failed to bind JSON", "error", err)
		common.Fail(ctx, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	// 从上下文获取租户ID
	tenantID := ctx.GetInt("tenant_id")

	response, err := c.changeApprovalService.CreateChangeRollbackPlan(ctx, &req, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to create rollback plan", "error", err, "tenant_id", tenantID)
		common.Fail(ctx, common.InternalErrorCode, "创建变更回滚计划失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// ExecuteChangeRollback 执行变更回滚
// @Summary 执行变更回滚
// @Description 执行指定变更的回滚操作
// @Tags 变更回滚计划
// @Accept json
// @Produce json
// @Param id path int true "变更ID"
// @Param rollbackPlanId path int true "回滚计划ID"
// @Param request body map[string]interface{} true "回滚执行请求"
// @Success 200 {object} dto.ChangeRollbackExecutionResponse
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/changes/{id}/rollback-plans/{rollbackPlanId}/execute [post]
func (c *ChangeApprovalController) ExecuteChangeRollback(ctx *gin.Context) {
	changeID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.BadRequestCode, "无效的变更ID")
		return
	}

	rollbackPlanID, err := strconv.Atoi(ctx.Param("rollbackPlanId"))
	if err != nil {
		common.Fail(ctx, common.BadRequestCode, "无效的回滚计划ID")
		return
	}

	var req map[string]interface{}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Failed to bind JSON", "error", err)
		common.Fail(ctx, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	// 从上下文获取租户ID和用户ID
	tenantID := ctx.GetInt("tenant_id")
	userID := ctx.GetInt("user_id")

	// 获取触发原因
	triggerReason, ok := req["trigger_reason"].(string)
	if !ok {
		common.Fail(ctx, common.BadRequestCode, "缺少触发原因")
		return
	}

	response, err := c.changeApprovalService.ExecuteChangeRollback(ctx, changeID, rollbackPlanID, userID, triggerReason, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to execute rollback", "error", err, "change_id", changeID)
		common.Fail(ctx, common.InternalErrorCode, "执行变更回滚失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}
