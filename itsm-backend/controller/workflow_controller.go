package controller

import (
	"itsm-backend/common"
	"itsm-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WorkflowController 工作流控制器
type WorkflowController struct {
	workflowService *service.WorkflowService
	logger          *zap.Logger
}

// NewWorkflowController 创建工作流控制器
func NewWorkflowController(workflowService *service.WorkflowService, logger *zap.Logger) *WorkflowController {
	return &WorkflowController{
		workflowService: workflowService,
		logger:          logger,
	}
}

// CreateWorkflow 创建工作流
func (wc *WorkflowController) CreateWorkflow(c *gin.Context) {
	var req service.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	req.TenantID = tenantID

	workflow, err := wc.workflowService.CreateWorkflow(c.Request.Context(), &req)
	if err != nil {
		wc.logger.Error("Failed to create workflow", zap.Error(err), zap.Int("tenant_id", tenantID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, workflow)
}

// UpdateWorkflow 更新工作流
func (wc *WorkflowController) UpdateWorkflow(c *gin.Context) {
	workflowID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工作流ID")
		return
	}

	var req service.UpdateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	workflow, err := wc.workflowService.UpdateWorkflow(c.Request.Context(), workflowID, &req)
	if err != nil {
		wc.logger.Error("Failed to update workflow", zap.Error(err), zap.Int("workflow_id", workflowID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, workflow)
}

// DeleteWorkflow 删除工作流
func (wc *WorkflowController) DeleteWorkflow(c *gin.Context) {
	workflowID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工作流ID")
		return
	}

	err = wc.workflowService.DeleteWorkflow(c.Request.Context(), workflowID)
	if err != nil {
		wc.logger.Error("Failed to delete workflow", zap.Error(err), zap.Int("workflow_id", workflowID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "工作流删除成功"})
}

// GetWorkflow 获取工作流
func (wc *WorkflowController) GetWorkflow(c *gin.Context) {
	workflowID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的工作流ID")
		return
	}

	workflow, err := wc.workflowService.GetWorkflow(c.Request.Context(), workflowID)
	if err != nil {
		wc.logger.Error("Failed to get workflow", zap.Error(err), zap.Int("workflow_id", workflowID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, workflow)
}

// ListWorkflows 获取工作流列表
func (wc *WorkflowController) ListWorkflows(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	workflowType := c.Query("type")
	isActiveStr := c.Query("is_active")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	var active *bool
	if isActiveStr != "" {
		if a, err := strconv.ParseBool(isActiveStr); err == nil {
			active = &a
		}
	}

	req := &service.ListWorkflowsRequest{
		Page:        page,
		PageSize:    pageSize,
		Type:        workflowType,
		IsActive:    active,
		TenantID:    tenantID,
	}

	workflows, total, err := wc.workflowService.ListWorkflows(c.Request.Context(), req)
	if err != nil {
		wc.logger.Error("Failed to list workflows", zap.Error(err), zap.Int("tenant_id", tenantID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{
		"workflows": workflows,
		"total":     total,
	})
}

// StartWorkflow 启动工作流
func (wc *WorkflowController) StartWorkflow(c *gin.Context) {
	var req service.StartWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	req.TenantID = tenantID

	instance, err := wc.workflowService.StartWorkflow(c.Request.Context(), &req)
	if err != nil {
		wc.logger.Error("Failed to start workflow", zap.Error(err), zap.Int("tenant_id", tenantID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, instance)
}

// ExecuteWorkflowStep 执行工作流步骤
func (wc *WorkflowController) ExecuteWorkflowStep(c *gin.Context) {
	var req service.ExecuteWorkflowStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	err := wc.workflowService.ExecuteWorkflowStep(c.Request.Context(), &req)
	if err != nil {
		wc.logger.Error("Failed to execute workflow step", zap.Error(err), zap.Int("instance_id", req.InstanceID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "工作流步骤执行成功"})
}
