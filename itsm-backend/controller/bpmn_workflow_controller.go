package controller

import (
	"net/http"
	"strconv"
	"time"

	"itsm-backend/ent"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// BPMNWorkflowController BPMN工作流控制器
type BPMNWorkflowController struct {
	processEngine  service.ProcessEngine
	versionService *service.BPMNVersionService
}

// NewBPMNWorkflowController 创建BPMN工作流控制器
func NewBPMNWorkflowController(processEngine service.ProcessEngine, versionService *service.BPMNVersionService) *BPMNWorkflowController {
	return &BPMNWorkflowController{
		processEngine:  processEngine,
		versionService: versionService,
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
		bpmn.GET("/process-definitions/:key/export", c.ExportProcessDefinition)
		bpmn.POST("/process-definitions/:key/clone", c.CloneProcessDefinition)
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

		// 会签管理
		bpmn.POST("/tasks/:id/counter-sign", c.CreateCounterSignTasks)
		bpmn.GET("/tasks/:id/counter-sign-status", c.GetCounterSignStatus)
		bpmn.PUT("/tasks/:id/vote", c.Vote)

		// 统计
		bpmn.GET("/stats/instances", c.GetInstanceStats)
		bpmn.GET("/stats/tasks", c.GetTaskStats)

		// 版本管理
		bpmn.GET("/versions", c.ListVersions)
		bpmn.GET("/versions/:key/:version", c.GetVersion)
		bpmn.POST("/versions", c.CreateVersion)
		bpmn.PUT("/versions/:key/:version/activate", c.ActivateVersion)
		bpmn.PUT("/versions/:key/:version/rollback", c.RollbackVersion)
		bpmn.GET("/versions/:key/compare", c.CompareVersions)

		// 版本变更日志
		bpmn.GET("/process-definitions/:key/changelogs", c.GetVersionChangeLogs)
		bpmn.GET("/process-definitions/:id/changelogs", c.GetVersionChangeLogsByID)
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

// ExportProcessDefinition 导出流程定义
func (c *BPMNWorkflowController) ExportProcessDefinition(ctx *gin.Context) {
	key := ctx.Param("key")
	version := ctx.Query("version")
	if version == "" {
		version = "1.0.0"
	}

	definition, err := c.processEngine.ProcessDefinitionService().GetProcessDefinition(ctx, key, version)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "获取流程定义失败: " + err.Error()})
		return
	}

	// 导出格式
	exportData := map[string]interface{}{
		"workflow": map[string]interface{}{
			"key":         definition.Key,
			"name":        definition.Name,
			"description": definition.Description,
			"category":    definition.Category,
			"version":     definition.Version,
			"bpmn_xml":    string(definition.BpmnXML),
		},
		"exportTime":    time.Now().Format(time.RFC3339),
		"exportVersion": "1.0",
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    exportData,
	})
}

// CloneProcessDefinition 复制流程定义
func (c *BPMNWorkflowController) CloneProcessDefinition(ctx *gin.Context) {
	key := ctx.Param("key")
	version := ctx.Query("version")
	if version == "" {
		version = "1.0.0"
	}

	var req struct {
		NewKey  string `json:"new_key" binding:"required"`
		NewName string `json:"new_name" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 获取原流程定义
	definition, err := c.processEngine.ProcessDefinitionService().GetProcessDefinition(ctx, key, version)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "获取流程定义失败: " + err.Error()})
		return
	}

	// 创建新流程定义
	bpmnXML := string(definition.BpmnXML)
	newDefinition := &service.CreateProcessDefinitionRequest{
		Key:         req.NewKey,
		Name:        req.NewName,
		Description: definition.Description,
		Category:    definition.Category,
		BPMNXML:     bpmnXML,
		TenantID:    definition.TenantID,
	}

	created, err := c.processEngine.ProcessDefinitionService().CreateProcessDefinition(ctx, newDefinition)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "复制流程定义失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    created,
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

	// 如果没有指定assignee，则查询当前用户有权限查看的任务
	// 包括：分配给自己的任务 + 未分配的任务
	if req.Assignee == "" {
		// 不再强制设置assignee，让查询返回所有任务（按租户过滤）
		// 后续可以在前端根据用户角色进行过滤
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

// ListVersions 获取版本列表
func (c *BPMNWorkflowController) ListVersions(ctx *gin.Context) {
	processKey := ctx.Query("process_key")
	tenantID := ctx.GetInt("tenant_id")

	if processKey == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "缺少process_key参数"})
		return
	}

	// 如果 process_key 是数字ID，尝试查找对应的流程定义key
	if id, err := strconv.Atoi(processKey); err == nil {
		def, err := c.processEngine.ProcessDefinitionService().GetProcessDefinitionByID(ctx, id)
		if err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "流程定义不存在"})
			return
		}
		processKey = def.Key
	}

	versions, err := c.versionService.ListVersions(ctx, processKey, tenantID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取版本列表失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    versions,
	})
}

// GetVersion 获取指定版本详情
func (c *BPMNWorkflowController) GetVersion(ctx *gin.Context) {
	processKey := ctx.Param("key")
	versionStr := ctx.Param("version")
	tenantID := ctx.GetInt("tenant_id")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的版本号"})
		return
	}

	versionInfo, err := c.versionService.GetVersion(ctx, processKey, version, tenantID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "版本不存在"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    versionInfo,
	})
}

// CreateVersion 创建新版本
func (c *BPMNWorkflowController) CreateVersion(ctx *gin.Context) {
	var req service.CreateVersionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	req.TenantID = ctx.GetInt("tenant_id")
	req.CreatedBy = ctx.GetString("user_id")

	version, err := c.versionService.CreateVersion(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "创建版本失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    version,
	})
}

// ActivateVersion 激活指定版本
func (c *BPMNWorkflowController) ActivateVersion(ctx *gin.Context) {
	processKey := ctx.Param("key")
	versionStr := ctx.Param("version")
	tenantID := ctx.GetInt("tenant_id")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的版本号"})
		return
	}

	err = c.versionService.ActivateVersion(ctx, processKey, version, tenantID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "激活版本失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

// RollbackVersion 回滚到指定版本
func (c *BPMNWorkflowController) RollbackVersion(ctx *gin.Context) {
	processKey := ctx.Param("key")
	versionStr := ctx.Param("version")
	tenantID := ctx.GetInt("tenant_id")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的版本号"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	ctx.ShouldBindJSON(&req)

	err = c.versionService.RollbackToVersion(ctx, processKey, version, tenantID, req.Reason)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "回滚版本失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

// CompareVersions 比较两个版本
func (c *BPMNWorkflowController) CompareVersions(ctx *gin.Context) {
	processKey := ctx.Param("key")
	baseVersion := ctx.Query("base_version")
	targetVersion := ctx.Query("target_version")
	tenantID := ctx.GetInt("tenant_id")

	// 如果没有提供版本参数，返回友好错误
	if baseVersion == "" || targetVersion == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请提供 base_version 和 target_version 参数"})
		return
	}

	base, err := strconv.Atoi(baseVersion)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的基础版本号"})
		return
	}

	target, err := strconv.Atoi(targetVersion)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的目标版本号"})
		return
	}

	// 相同版本无需比较
	if base == target {
		ctx.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "版本相同，无需比较",
			"data":    nil,
		})
		return
	}

	comparison, err := c.versionService.CompareVersions(ctx, processKey, base, target, tenantID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "版本比较失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    comparison,
	})
}

// GetInstanceStats 获取实例统计
func (c *BPMNWorkflowController) GetInstanceStats(ctx *gin.Context) {
	var req service.InstanceStatisticsRequest
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

	stats, err := c.processEngine.ProcessInstanceService().GetInstanceStatistics(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取实例统计失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

// GetTaskStats 获取任务统计
func (c *BPMNWorkflowController) GetTaskStats(ctx *gin.Context) {
	var req service.TaskStatisticsRequest
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

	stats, err := c.processEngine.TaskService().GetTaskStatistics(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务统计失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

// CreateCounterSignTasks 创建会签任务
func (c *BPMNWorkflowController) CreateCounterSignTasks(ctx *gin.Context) {
	taskID := ctx.Param("id")

	var req service.CounterSignRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	tasks, err := c.processEngine.TaskService().CreateCounterSignTasks(ctx, taskID, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "创建会签任务失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data":    tasks,
	})
}

// GetCounterSignStatus 获取会签状态
func (c *BPMNWorkflowController) GetCounterSignStatus(ctx *gin.Context) {
	taskID := ctx.Param("id")

	status, err := c.processEngine.TaskService().GetCounterSignStatus(ctx, taskID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取会签状态失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    status,
	})
}

// Vote 投票
func (c *BPMNWorkflowController) Vote(ctx *gin.Context) {
	taskID := ctx.Param("id")

	var req service.VoteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	err := c.processEngine.TaskService().Vote(ctx, taskID, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "投票失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}
