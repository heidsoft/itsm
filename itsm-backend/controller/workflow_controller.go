package controller

import (
	"itsm-backend/common"
	"itsm-backend/service"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WorkflowController 工作流控制器
type WorkflowController struct {
	workflowService *service.WorkflowService
	workflowEngine  *service.WorkflowEngine
	approvalService *service.WorkflowApprovalService
	taskService     *service.WorkflowTaskService
	monitorService  *service.WorkflowMonitorService
	logger          *zap.Logger
}

// NewWorkflowController 创建工作流控制器
func NewWorkflowController(
	workflowService *service.WorkflowService,
	workflowEngine *service.WorkflowEngine,
	approvalService *service.WorkflowApprovalService,
	taskService *service.WorkflowTaskService,
	monitorService *service.WorkflowMonitorService,
	logger *zap.Logger,
) *WorkflowController {
	return &WorkflowController{
		workflowService: workflowService,
		workflowEngine:  workflowEngine,
		approvalService: approvalService,
		taskService:     taskService,
		monitorService:  monitorService,
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
		Page:     page,
		PageSize: pageSize,
		Type:     workflowType,
		IsActive: active,
		TenantID: tenantID,
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

// CompleteWorkflowStep 完成工作流步骤
func (wc *WorkflowController) CompleteWorkflowStep(c *gin.Context) {
	var req service.CompleteWorkflowStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	err := wc.workflowEngine.CompleteWorkflowStep(c.Request.Context(), &req)
	if err != nil {
		wc.logger.Error("Failed to complete workflow step", zap.Error(err), zap.Int("instance_id", req.InstanceID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "工作流步骤完成成功"})
}

// GetApprovalTasks 获取审批任务列表
func (wc *WorkflowController) GetApprovalTasks(c *gin.Context) {
	assigneeID := c.GetInt("user_id")
	status := c.Query("status")
	priority := c.Query("priority")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	req := &service.GetApprovalTasksRequest{
		AssigneeID: assigneeID,
		Status:     status,
		Priority:   priority,
		Page:       page,
		PageSize:   pageSize,
	}

	tasks, total, err := wc.approvalService.GetApprovalTasks(c.Request.Context(), req)
	if err != nil {
		wc.logger.Error("Failed to get approval tasks", zap.Error(err), zap.Int("assignee_id", assigneeID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{
		"tasks": tasks,
		"total": total,
	})
}

// ApproveTask 审批通过
func (wc *WorkflowController) ApproveTask(c *gin.Context) {
	var req service.ApproveTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	req.UserID = c.GetInt("user_id")

	err := wc.approvalService.ApproveTask(c.Request.Context(), &req)
	if err != nil {
		wc.logger.Error("Failed to approve task", zap.Error(err), zap.Int("task_id", req.TaskID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "审批通过成功"})
}

// RejectTask 审批拒绝
func (wc *WorkflowController) RejectTask(c *gin.Context) {
	var req service.RejectTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	req.UserID = c.GetInt("user_id")

	err := wc.approvalService.RejectTask(c.Request.Context(), &req)
	if err != nil {
		wc.logger.Error("Failed to reject task", zap.Error(err), zap.Int("task_id", req.TaskID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "审批拒绝成功"})
}

// GetWorkflowTasks 获取工作流任务列表
func (wc *WorkflowController) GetWorkflowTasks(c *gin.Context) {
	assigneeID := c.GetInt("user_id")
	instanceIDStr := c.Query("instance_id")
	status := c.Query("status")
	priority := c.Query("priority")
	taskType := c.Query("task_type")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	var instanceID int
	if instanceIDStr != "" {
		instanceID, _ = strconv.Atoi(instanceIDStr)
	}

	req := &service.GetTasksRequest{
		InstanceID: instanceID,
		AssigneeID: assigneeID,
		Status:     status,
		Priority:   priority,
		TaskType:   taskType,
		Page:       page,
		PageSize:   pageSize,
	}

	tasks, total, err := wc.taskService.GetTasks(c.Request.Context(), req)
	if err != nil {
		wc.logger.Error("Failed to get workflow tasks", zap.Error(err), zap.Int("assignee_id", assigneeID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{
		"tasks": tasks,
		"total": total,
	})
}

// StartTask 开始执行任务
func (wc *WorkflowController) StartTask(c *gin.Context) {
	var req service.StartTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	req.UserID = c.GetInt("user_id")

	err := wc.taskService.StartTask(c.Request.Context(), &req)
	if err != nil {
		wc.logger.Error("Failed to start task", zap.Error(err), zap.Int("task_id", req.TaskID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "任务开始执行成功"})
}

// CompleteTask 完成任务
func (wc *WorkflowController) CompleteTask(c *gin.Context) {
	var req service.CompleteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	req.UserID = c.GetInt("user_id")

	err := wc.taskService.CompleteTask(c.Request.Context(), &req)
	if err != nil {
		wc.logger.Error("Failed to complete task", zap.Error(err), zap.Int("task_id", req.TaskID))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{"message": "任务完成成功"})
}

// GetWorkflowMetrics 获取工作流指标
func (wc *WorkflowController) GetWorkflowMetrics(c *gin.Context) {
	workflowIDStr := c.Query("workflow_id")
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	var req service.GetWorkflowMetricsRequest
	if workflowIDStr != "" {
		if workflowID, err := strconv.Atoi(workflowIDStr); err == nil {
			req.WorkflowID = workflowID
		}
	}

	if startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
			req.StartDate = startDate
		}
	}

	if endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
			req.EndDate = endDate
		}
	}

	metrics, err := wc.monitorService.GetWorkflowMetrics(c.Request.Context(), &req)
	if err != nil {
		wc.logger.Error("Failed to get workflow metrics", zap.Error(err))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, metrics)
}

// GetStepMetrics 获取步骤指标
func (wc *WorkflowController) GetStepMetrics(c *gin.Context) {
	instanceIDStr := c.Query("instance_id")
	stepID := c.Query("step_id")
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	var req service.GetStepMetricsRequest
	if instanceIDStr != "" {
		if instanceID, err := strconv.Atoi(instanceIDStr); err == nil {
			req.InstanceID = instanceID
		}
	}
	req.StepID = stepID

	if startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
			req.StartDate = startDate
		}
	}

	if endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
			req.EndDate = endDate
		}
	}

	metrics, err := wc.monitorService.GetStepMetrics(c.Request.Context(), &req)
	if err != nil {
		wc.logger.Error("Failed to get step metrics", zap.Error(err))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, metrics)
}

// GetAlerts 获取告警列表
func (wc *WorkflowController) GetAlerts(c *gin.Context) {
	status := c.Query("status")
	severity := c.Query("severity")
	workflowIDStr := c.Query("workflow_id")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	var workflowID int
	if workflowIDStr != "" {
		workflowID, _ = strconv.Atoi(workflowIDStr)
	}

	req := &service.GetAlertsRequest{
		Status:     status,
		Severity:   severity,
		WorkflowID: workflowID,
		Page:       page,
		PageSize:   pageSize,
	}

	alerts, total, err := wc.monitorService.GetAlerts(c.Request.Context(), req)
	if err != nil {
		wc.logger.Error("Failed to get alerts", zap.Error(err))
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{
		"alerts": alerts,
		"total":  total,
	})
}
