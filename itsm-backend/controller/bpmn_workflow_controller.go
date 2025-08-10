package controller

import (
	"net/http"
	"strconv"

	"itsm-backend/ent"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// BPMNWorkflowController BPMN工作流控制器
type BPMNWorkflowController struct {
	processEngine service.ProcessEngine
}

// NewBPMNWorkflowController 创建BPMN工作流控制器
func NewBPMNWorkflowController(processEngine service.ProcessEngine) *BPMNWorkflowController {
	return &BPMNWorkflowController{
		processEngine: processEngine,
	}
}

// RegisterRoutes 注册路由
func (c *BPMNWorkflowController) RegisterRoutes(r *gin.RouterGroup) {
	bpmn := r.Group("/bpmn")
	{
		// 流程定义管理
		bpmn.POST("/process-definitions", c.CreateProcessDefinition)
		bpmn.GET("/process-definitions", c.ListProcessDefinitions)
		bpmn.GET("/process-definitions/:key", c.GetProcessDefinition)
		bpmn.PUT("/process-definitions/:key", c.UpdateProcessDefinition)
		bpmn.DELETE("/process-definitions/:key", c.DeleteProcessDefinition)
		bpmn.PUT("/process-definitions/:key/active", c.SetProcessDefinitionActive)

		// 流程实例管理
		bpmn.POST("/process-instances", c.StartProcess)
		bpmn.GET("/process-instances", c.ListProcessInstances)
		bpmn.GET("/process-instances/:id", c.GetProcessInstance)
		bpmn.PUT("/process-instances/:id/variables", c.SetProcessInstanceVariables)
		bpmn.PUT("/process-instances/:id/suspend", c.SuspendProcess)
		bpmn.PUT("/process-instances/:id/resume", c.ResumeProcess)
		bpmn.PUT("/process-instances/:id/terminate", c.TerminateProcess)

		// 任务管理
		bpmn.GET("/tasks", c.ListUserTasks)
		bpmn.GET("/tasks/:id", c.GetTask)
		bpmn.PUT("/tasks/:id/assign", c.AssignTask)
		bpmn.PUT("/tasks/:id/complete", c.CompleteTask)
		bpmn.PUT("/tasks/:id/cancel", c.CancelTask)
		bpmn.PUT("/tasks/:id/variables", c.SetTaskVariables)
	}
}

// CreateProcessDefinition 创建流程定义
func (c *BPMNWorkflowController) CreateProcessDefinition(ctx *gin.Context) {
	var req service.CreateProcessDefinitionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	req.TenantID = tenantID.(int)

	definition, err := c.processEngine.ProcessDefinitionService().CreateProcessDefinition(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "创建流程定义失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "流程定义创建成功",
		"data":    definition,
	})
}

// ListProcessDefinitions 获取流程定义列表
func (c *BPMNWorkflowController) ListProcessDefinitions(ctx *gin.Context) {
	var req service.ListProcessDefinitionsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	req.TenantID = tenantID.(int)

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	definitions, total, err := c.processEngine.ProcessDefinitionService().ListProcessDefinitions(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取流程定义列表失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": definitions,
		"pagination": gin.H{
			"page":      req.Page,
			"page_size": req.PageSize,
			"total":     total,
		},
	})
}

// GetProcessDefinition 获取流程定义
func (c *BPMNWorkflowController) GetProcessDefinition(ctx *gin.Context) {
	key := ctx.Param("key")
	version := ctx.Query("version")

	var definition *ent.ProcessDefinition
	var err error

	if version != "" {
		definition, err = c.processEngine.ProcessDefinitionService().GetProcessDefinition(ctx, key, version)
	} else {
		definition, err = c.processEngine.ProcessDefinitionService().GetLatestProcessDefinition(ctx, key)
	}

	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "流程定义不存在: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": definition,
	})
}

// UpdateProcessDefinition 更新流程定义
func (c *BPMNWorkflowController) UpdateProcessDefinition(ctx *gin.Context) {
	key := ctx.Param("key")
	version := ctx.Query("version")
	if version == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "版本参数不能为空"})
		return
	}

	var req service.UpdateProcessDefinitionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	definition, err := c.processEngine.ProcessDefinitionService().UpdateProcessDefinition(ctx, key, version, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "更新流程定义失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "流程定义更新成功",
		"data":    definition,
	})
}

// DeleteProcessDefinition 删除流程定义
func (c *BPMNWorkflowController) DeleteProcessDefinition(ctx *gin.Context) {
	key := ctx.Param("key")
	version := ctx.Query("version")
	if version == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "版本参数不能为空"})
		return
	}

	err := c.processEngine.ProcessDefinitionService().DeleteProcessDefinition(ctx, key, version)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "删除流程定义失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "流程定义删除成功",
	})
}

// SetProcessDefinitionActive 激活/停用流程定义
func (c *BPMNWorkflowController) SetProcessDefinitionActive(ctx *gin.Context) {
	key := ctx.Param("key")
	version := ctx.Query("version")
	if version == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "版本参数不能为空"})
		return
	}

	var req struct {
		Active bool `json:"active" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	err := c.processEngine.ProcessDefinitionService().SetProcessDefinitionActive(ctx, key, version, req.Active)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "设置流程定义状态失败: " + err.Error()})
		return
	}

	status := "激活"
	if !req.Active {
		status = "停用"
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "流程定义" + status + "成功",
	})
}

// StartProcess 启动流程实例
func (c *BPMNWorkflowController) StartProcess(ctx *gin.Context) {
	var req struct {
		ProcessDefinitionKey string                 `json:"process_definition_key" binding:"required"`
		BusinessKey          string                 `json:"business_key" binding:"required"`
		Variables            map[string]interface{} `json:"variables"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	instance, err := c.processEngine.StartProcess(ctx, req.ProcessDefinitionKey, req.BusinessKey, req.Variables)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "启动流程实例失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "流程实例启动成功",
		"data":    instance,
	})
}

// ListProcessInstances 获取流程实例列表
func (c *BPMNWorkflowController) ListProcessInstances(ctx *gin.Context) {
	var req service.ListProcessInstancesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	req.TenantID = tenantID.(int)

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	instances, total, err := c.processEngine.ProcessInstanceService().ListProcessInstances(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取流程实例列表失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": instances,
		"pagination": gin.H{
			"page":      req.Page,
			"page_size": req.PageSize,
			"total":     total,
		},
	})
}

// GetProcessInstance 获取流程实例
func (c *BPMNWorkflowController) GetProcessInstance(ctx *gin.Context) {
	processInstanceID := ctx.Param("id")

	instance, err := c.processEngine.ProcessInstanceService().GetProcessInstance(ctx, processInstanceID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "流程实例不存在: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": instance,
	})
}

// SetProcessInstanceVariables 设置流程实例变量
func (c *BPMNWorkflowController) SetProcessInstanceVariables(ctx *gin.Context) {
	processInstanceID := ctx.Param("id")

	var req struct {
		Variables map[string]interface{} `json:"variables" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	err := c.processEngine.ProcessInstanceService().SetProcessInstanceVariables(ctx, processInstanceID, req.Variables)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "设置流程实例变量失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "流程实例变量设置成功",
	})
}

// SuspendProcess 暂停流程实例
func (c *BPMNWorkflowController) SuspendProcess(ctx *gin.Context) {
	processInstanceID := ctx.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	err := c.processEngine.SuspendProcess(ctx, processInstanceID, req.Reason)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "暂停流程实例失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "流程实例暂停成功",
	})
}

// ResumeProcess 恢复流程实例
func (c *BPMNWorkflowController) ResumeProcess(ctx *gin.Context) {
	processInstanceID := ctx.Param("id")

	err := c.processEngine.ResumeProcess(ctx, processInstanceID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "恢复流程实例失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "流程实例恢复成功",
	})
}

// TerminateProcess 终止流程实例
func (c *BPMNWorkflowController) TerminateProcess(ctx *gin.Context) {
	processInstanceID := ctx.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	err := c.processEngine.TerminateProcess(ctx, processInstanceID, req.Reason)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "终止流程实例失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "流程实例终止成功",
	})
}

// ListUserTasks 获取用户任务列表
func (c *BPMNWorkflowController) ListUserTasks(ctx *gin.Context) {
	var req service.ListUserTasksRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 从JWT获取用户信息
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	// 如果没有指定assignee，则使用当前用户
	if req.Assignee == "" {
		req.Assignee = strconv.Itoa(userID.(int))
	}

	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}
	req.TenantID = tenantID.(int)

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	tasks, total, err := c.processEngine.TaskService().ListUserTasks(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户任务列表失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": tasks,
		"pagination": gin.H{
			"page":      req.Page,
			"page_size": req.PageSize,
			"total":     total,
		},
	})
}

// GetTask 获取任务
func (c *BPMNWorkflowController) GetTask(ctx *gin.Context) {
	taskID := ctx.Param("id")

	task, err := c.processEngine.TaskService().GetTask(ctx, taskID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "任务不存在: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": task,
	})
}

// AssignTask 分配任务
func (c *BPMNWorkflowController) AssignTask(ctx *gin.Context) {
	taskID := ctx.Param("id")

	var req struct {
		Assignee string `json:"assignee" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	err := c.processEngine.TaskService().AssignTask(ctx, taskID, req.Assignee)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "分配任务失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "任务分配成功",
	})
}

// CompleteTask 完成任务
func (c *BPMNWorkflowController) CompleteTask(ctx *gin.Context) {
	taskID := ctx.Param("id")

	var req struct {
		Variables map[string]interface{} `json:"variables"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	err := c.processEngine.TaskService().CompleteTask(ctx, taskID, req.Variables)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "完成任务失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "任务完成成功",
	})
}

// CancelTask 取消任务
func (c *BPMNWorkflowController) CancelTask(ctx *gin.Context) {
	taskID := ctx.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	err := c.processEngine.TaskService().CancelTask(ctx, taskID, req.Reason)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "取消任务失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "任务取消成功",
	})
}

// SetTaskVariables 设置任务变量
func (c *BPMNWorkflowController) SetTaskVariables(ctx *gin.Context) {
	taskID := ctx.Param("id")

	var req struct {
		Variables map[string]interface{} `json:"variables" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	err := c.processEngine.TaskService().SetTaskVariables(ctx, taskID, req.Variables)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "设置任务变量失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "任务变量设置成功",
	})
}
