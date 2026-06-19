package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"itsm-backend/common"
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

func getBPMNTenantContext(ctx *gin.Context) (context.Context, int, bool) {
	tenantID := ctx.GetInt("tenant_id")
	if tenantID <= 0 {
		common.AuthFailed(ctx, "未授权访问")
		return nil, 0, false
	}
	return context.WithValue(ctx.Request.Context(), "bpmn_tenant_id", tenantID), tenantID, true
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
		bpmn.PUT("/tasks/:id/claim", c.ClaimTask)
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

		// 版本变更日志 (注意：:id 路由必须在 :key 之前定义)
		bpmn.GET("/process-definitions/changelogs/:id", c.GetVersionChangeLogsByID)
		bpmn.GET("/process-definitions/:key/changelogs", c.GetVersionChangeLogs)
	}
}

// CreateProcessDefinition 创建流程定义
func (c *BPMNWorkflowController) CreateProcessDefinition(ctx *gin.Context) {
	var req service.CreateProcessDefinitionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.AuthFailed(ctx, "未授权访问")
		return
	}
	req.TenantID = tenantID.(int)

	definition, err := c.processEngine.ProcessDefinitionService().CreateProcessDefinition(ctx, &req)
	if err != nil {
		common.InternalError(ctx, "创建流程定义失败: "+err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程定义创建成功", definition)
}

// ListProcessDefinitions 获取流程定义列表
func (c *BPMNWorkflowController) ListProcessDefinitions(ctx *gin.Context) {
	var req service.ListProcessDefinitionsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.AuthFailed(ctx, "未授权访问")
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
		common.InternalError(ctx, "获取流程定义列表失败: "+err.Error())
		return
	}

	// 使用统一响应格式
	listResponse := common.NewListResponse(definitions, common.NewPaginationResponse(int(req.Page), int(req.PageSize), int64(total)))
	common.Success(ctx, listResponse)
}

// GetProcessDefinition 获取流程定义
func (c *BPMNWorkflowController) GetProcessDefinition(ctx *gin.Context) {
	key := ctx.Param("key")
	version := ctx.Query("version")
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	var definition *ent.ProcessDefinition
	var err error

	if version != "" {
		definition, err = c.processEngine.ProcessDefinitionService().GetProcessDefinition(workflowCtx, key, version)
	} else {
		definition, err = c.processEngine.ProcessDefinitionService().GetLatestProcessDefinition(workflowCtx, key)
	}

	if err != nil {
		common.NotFound(ctx, "流程定义不存在: "+err.Error())
		return
	}

	common.Success(ctx, definition)
}

// UpdateProcessDefinition 更新流程定义
func (c *BPMNWorkflowController) UpdateProcessDefinition(ctx *gin.Context) {
	key := ctx.Param("key")
	version := ctx.Query("version")
	if version == "" {
		common.Fail(ctx, common.BadRequestCode, "版本参数不能为空")
		return
	}

	var req service.UpdateProcessDefinitionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	definition, err := c.processEngine.ProcessDefinitionService().UpdateProcessDefinition(workflowCtx, key, version, &req)
	if err != nil {
		common.InternalError(ctx, "更新流程定义失败: "+err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程定义更新成功", definition)
}

// DeleteProcessDefinition 删除流程定义
func (c *BPMNWorkflowController) DeleteProcessDefinition(ctx *gin.Context) {
	key := ctx.Param("key")
	version := ctx.Query("version")
	if version == "" {
		common.Fail(ctx, common.BadRequestCode, "版本参数不能为空")
		return
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	err := c.processEngine.ProcessDefinitionService().DeleteProcessDefinition(workflowCtx, key, version)
	if err != nil {
		common.InternalError(ctx, "删除流程定义失败: "+err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程定义删除成功", nil)
}

// ExportProcessDefinition 导出流程定义
func (c *BPMNWorkflowController) ExportProcessDefinition(ctx *gin.Context) {
	key := ctx.Param("key")
	version := ctx.Query("version")
	if version == "" {
		version = "1.0.0"
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	definition, err := c.processEngine.ProcessDefinitionService().GetProcessDefinition(workflowCtx, key, version)
	if err != nil {
		common.NotFound(ctx, "获取流程定义失败: "+err.Error())
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

	common.Success(ctx, exportData)
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
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	workflowCtx, tenantID, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	// 获取原流程定义
	definition, err := c.processEngine.ProcessDefinitionService().GetProcessDefinition(workflowCtx, key, version)
	if err != nil {
		common.NotFound(ctx, "获取流程定义失败: "+err.Error())
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
		TenantID:    tenantID,
	}

	created, err := c.processEngine.ProcessDefinitionService().CreateProcessDefinition(workflowCtx, newDefinition)
	if err != nil {
		common.InternalError(ctx, "复制流程定义失败: "+err.Error())
		return
	}

	common.Success(ctx, created)
}

// SetProcessDefinitionActive 激活/停用流程定义
func (c *BPMNWorkflowController) SetProcessDefinitionActive(ctx *gin.Context) {
	key := ctx.Param("key")
	version := ctx.Query("version")
	if version == "" {
		common.Fail(ctx, common.BadRequestCode, "版本参数不能为空")
		return
	}

	var req struct {
		Active bool `json:"active" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	err := c.processEngine.ProcessDefinitionService().SetProcessDefinitionActive(workflowCtx, key, version, req.Active)
	if err != nil {
		common.InternalError(ctx, "设置流程定义状态失败: "+err.Error())
		return
	}

	status := "激活"
	if !req.Active {
		status = "停用"
	}

	common.SuccessWithMessage(ctx, "流程定义"+status+"成功", nil)
}

// StartProcess 启动流程实例
func (c *BPMNWorkflowController) StartProcess(ctx *gin.Context) {
	var req struct {
		ProcessDefinitionKey string                 `json:"process_definition_key" binding:"required"`
		BusinessKey          string                 `json:"business_key" binding:"required"`
		Variables            map[string]interface{} `json:"variables"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	instance, err := c.processEngine.StartProcess(workflowCtx, req.ProcessDefinitionKey, req.BusinessKey, req.Variables)
	if err != nil {
		common.InternalError(ctx, "启动流程实例失败: "+err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程实例启动成功", instance)
}

// ListProcessInstances 获取流程实例列表
func (c *BPMNWorkflowController) ListProcessInstances(ctx *gin.Context) {
	var req service.ListProcessInstancesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.AuthFailed(ctx, "未授权访问")
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
		common.InternalError(ctx, "获取流程实例列表失败: "+err.Error())
		return
	}

	// 使用统一响应格式
	listResponse := common.NewListResponse(instances, common.NewPaginationResponse(int(req.Page), int(req.PageSize), int64(total)))
	common.Success(ctx, listResponse)
}

// GetProcessInstance 获取流程实例
func (c *BPMNWorkflowController) GetProcessInstance(ctx *gin.Context) {
	processInstanceID := ctx.Param("id")
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	instance, err := c.processEngine.ProcessInstanceService().GetProcessInstance(workflowCtx, processInstanceID)
	if err != nil {
		common.NotFound(ctx, "流程实例不存在: "+err.Error())
		return
	}

	common.Success(ctx, instance)
}

// SetProcessInstanceVariables 设置流程实例变量
func (c *BPMNWorkflowController) SetProcessInstanceVariables(ctx *gin.Context) {
	processInstanceID := ctx.Param("id")

	var req struct {
		Variables map[string]interface{} `json:"variables" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	err := c.processEngine.ProcessInstanceService().SetProcessInstanceVariables(workflowCtx, processInstanceID, req.Variables)
	if err != nil {
		common.InternalError(ctx, "设置流程实例变量失败: "+err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程实例变量设置成功", nil)
}

// SuspendProcess 暂停流程实例
func (c *BPMNWorkflowController) SuspendProcess(ctx *gin.Context) {
	processInstanceID := ctx.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	err := c.processEngine.SuspendProcess(workflowCtx, processInstanceID, req.Reason)
	if err != nil {
		common.InternalError(ctx, "暂停流程实例失败: "+err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程实例暂停成功", nil)
}

// ResumeProcess 恢复流程实例
func (c *BPMNWorkflowController) ResumeProcess(ctx *gin.Context) {
	processInstanceID := ctx.Param("id")
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	err := c.processEngine.ResumeProcess(workflowCtx, processInstanceID)
	if err != nil {
		common.InternalError(ctx, "恢复流程实例失败: "+err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程实例恢复成功", nil)
}

// TerminateProcess 终止流程实例
func (c *BPMNWorkflowController) TerminateProcess(ctx *gin.Context) {
	processInstanceID := ctx.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	err := c.processEngine.TerminateProcess(workflowCtx, processInstanceID, req.Reason)
	if err != nil {
		common.InternalError(ctx, "终止流程实例失败: "+err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程实例终止成功", nil)
}

// ListUserTasks 获取用户任务列表
func (c *BPMNWorkflowController) ListUserTasks(ctx *gin.Context) {
	var req service.ListUserTasksRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
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
		common.AuthFailed(ctx, "未授权访问")
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
		common.InternalError(ctx, "获取用户任务列表失败: "+err.Error())
		return
	}

	// 使用统一响应格式
	listResponse := common.NewListResponse(tasks, common.NewPaginationResponse(int(req.Page), int(req.PageSize), int64(total)))
	common.Success(ctx, listResponse)
}

// GetTask 获取任务
func (c *BPMNWorkflowController) GetTask(ctx *gin.Context) {
	taskID := ctx.Param("id")
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	// 先尝试解析为数字ID（数据库自增ID）
	id, err := strconv.Atoi(taskID)
	var task interface{}
	if err == nil {
		// 数字ID，使用GetTaskByID
		task, err = c.processEngine.TaskService().GetTaskByID(workflowCtx, id)
	} else {
		// 字符串ID（BPMN标准task_id），使用GetTask
		task, err = c.processEngine.TaskService().GetTask(workflowCtx, taskID)
	}
	if err != nil {
		common.NotFound(ctx, "任务不存在: "+err.Error())
		return
	}

	common.Success(ctx, task)
}

// AssignTask 分配任务
func (c *BPMNWorkflowController) AssignTask(ctx *gin.Context) {
	taskID := ctx.Param("id")

	var req struct {
		Assignee string `json:"assignee" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	err := c.processEngine.TaskService().AssignTask(workflowCtx, taskID, req.Assignee)
	if err != nil {
		common.InternalError(ctx, "分配任务失败: "+err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "任务分配成功", nil)
}

// ClaimTask 认领任务
func (c *BPMNWorkflowController) ClaimTask(ctx *gin.Context) {
	taskID := ctx.Param("id")
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	// 从上下文获取用户ID
	userID, exists := ctx.Get("user_id")
	if !exists {
		common.AuthFailed(ctx, "未授权访问")
		return
	}

	var req struct {
		UserID int `json:"user_id"`
	}
	// 如果请求中提供了user_id则使用，否则使用当前登录用户
	if err := ctx.ShouldBindJSON(&req); err == nil && req.UserID > 0 {
		userID = req.UserID
	}

	// 尝试解析为数字ID
	id, err := strconv.Atoi(taskID)
	var claimErr error
	if err == nil {
		// 数字ID
		claimErr = c.processEngine.TaskService().ClaimTaskByID(workflowCtx, id, userID.(int))
	} else {
		// 字符串ID，使用ClaimTask
		claimErr = c.processEngine.TaskService().ClaimTask(workflowCtx, taskID, fmt.Sprintf("%d", userID))
	}
	if claimErr != nil {
		common.InternalError(ctx, "认领任务失败: "+claimErr.Error())
		return
	}

	common.SuccessWithMessage(ctx, "任务认领成功", nil)
}

// CompleteTask 完成任务
func (c *BPMNWorkflowController) CompleteTask(ctx *gin.Context) {
	taskID := ctx.Param("id")

	var req struct {
		Variables map[string]interface{} `json:"variables"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	// 尝试解析为数字ID
	id, err := strconv.Atoi(taskID)
	if err == nil {
		// 数字ID，使用CompleteTaskByID
		err = c.processEngine.TaskService().CompleteTaskByID(workflowCtx, id, req.Variables)
	} else {
		// 字符串ID，使用原来的CompleteTask
		err = c.processEngine.TaskService().CompleteTask(workflowCtx, taskID, req.Variables)
	}
	if err != nil {
		common.InternalError(ctx, "完成任务失败: "+err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "任务完成成功", nil)
}

// CancelTask 取消任务
func (c *BPMNWorkflowController) CancelTask(ctx *gin.Context) {
	taskID := ctx.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	err := c.processEngine.TaskService().CancelTask(workflowCtx, taskID, req.Reason)
	if err != nil {
		common.InternalError(ctx, "取消任务失败: "+err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "任务取消成功", nil)
}

// SetTaskVariables 设置任务变量
func (c *BPMNWorkflowController) SetTaskVariables(ctx *gin.Context) {
	taskID := ctx.Param("id")

	var req struct {
		Variables map[string]interface{} `json:"variables" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	err := c.processEngine.TaskService().SetTaskVariables(workflowCtx, taskID, req.Variables)
	if err != nil {
		common.InternalError(ctx, "设置任务变量失败: "+err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "任务变量设置成功", nil)
}

// ListVersions 获取版本列表
func (c *BPMNWorkflowController) ListVersions(ctx *gin.Context) {
	processKey := ctx.Query("process_key")
	tenantID := ctx.GetInt("tenant_id")

	if processKey == "" {
		common.Fail(ctx, common.BadRequestCode, "缺少process_key参数")
		return
	}

	// 如果 process_key 是数字ID，尝试查找对应的流程定义key
	if id, err := strconv.Atoi(processKey); err == nil {
		workflowCtx, _, ok := getBPMNTenantContext(ctx)
		if !ok {
			return
		}
		def, err := c.processEngine.ProcessDefinitionService().GetProcessDefinitionByID(workflowCtx, id)
		if err != nil {
			common.NotFound(ctx, "流程定义不存在")
			return
		}
		processKey = def.Key
	}

	versions, err := c.versionService.ListVersions(ctx, processKey, tenantID)
	if err != nil {
		common.InternalError(ctx, "获取版本列表失败: "+err.Error())
		return
	}

	common.Success(ctx, versions)
}

// GetVersion 获取指定版本详情
func (c *BPMNWorkflowController) GetVersion(ctx *gin.Context) {
	processKey := ctx.Param("key")
	versionStr := ctx.Param("version")
	tenantID := ctx.GetInt("tenant_id")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		common.Fail(ctx, common.BadRequestCode, "无效的版本号")
		return
	}

	versionInfo, err := c.versionService.GetVersion(ctx, processKey, version, tenantID)
	if err != nil {
		common.NotFound(ctx, "版本不存在")
		return
	}

	common.Success(ctx, versionInfo)
}

// CreateVersion 创建新版本
func (c *BPMNWorkflowController) CreateVersion(ctx *gin.Context) {
	var req service.CreateVersionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	req.TenantID = ctx.GetInt("tenant_id")
	req.CreatedBy = ctx.GetString("user_id")

	version, err := c.versionService.CreateVersion(ctx, &req)
	if err != nil {
		common.InternalError(ctx, "创建版本失败: "+err.Error())
		return
	}

	common.Success(ctx, version)
}

// ActivateVersion 激活指定版本
func (c *BPMNWorkflowController) ActivateVersion(ctx *gin.Context) {
	processKey := ctx.Param("key")
	versionStr := ctx.Param("version")
	tenantID := ctx.GetInt("tenant_id")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		common.Fail(ctx, common.BadRequestCode, "无效的版本号")
		return
	}

	err = c.versionService.ActivateVersion(ctx, processKey, version, tenantID)
	if err != nil {
		common.InternalError(ctx, "激活版本失败: "+err.Error())
		return
	}

	common.Success(ctx, nil)
}

// RollbackVersion 回滚到指定版本
func (c *BPMNWorkflowController) RollbackVersion(ctx *gin.Context) {
	processKey := ctx.Param("key")
	versionStr := ctx.Param("version")
	tenantID := ctx.GetInt("tenant_id")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		common.Fail(ctx, common.BadRequestCode, "无效的版本号")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	ctx.ShouldBindJSON(&req)

	err = c.versionService.RollbackToVersion(ctx, processKey, version, tenantID, req.Reason)
	if err != nil {
		common.InternalError(ctx, "回滚版本失败: "+err.Error())
		return
	}

	common.Success(ctx, nil)
}

// CompareVersions 比较两个版本
func (c *BPMNWorkflowController) CompareVersions(ctx *gin.Context) {
	processKey := ctx.Param("key")
	baseVersion := ctx.Query("base_version")
	targetVersion := ctx.Query("target_version")
	tenantID := ctx.GetInt("tenant_id")

	// 如果没有提供版本参数，返回友好错误
	if baseVersion == "" || targetVersion == "" {
		common.Fail(ctx, common.BadRequestCode, "请提供 base_version 和 target_version 参数")
		return
	}

	base, err := strconv.Atoi(baseVersion)
	if err != nil {
		common.Fail(ctx, common.BadRequestCode, "无效的基础版本号")
		return
	}

	target, err := strconv.Atoi(targetVersion)
	if err != nil {
		common.Fail(ctx, common.BadRequestCode, "无效的目标版本号")
		return
	}

	// 相同版本无需比较
	if base == target {
		common.SuccessWithMessage(ctx, "版本相同，无需比较", nil)
		return
	}

	comparison, err := c.versionService.CompareVersions(ctx, processKey, base, target, tenantID)
	if err != nil {
		common.InternalError(ctx, "版本比较失败: "+err.Error())
		return
	}

	common.Success(ctx, comparison)
}

// GetInstanceStats 获取实例统计
func (c *BPMNWorkflowController) GetInstanceStats(ctx *gin.Context) {
	var req service.InstanceStatisticsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.AuthFailed(ctx, "未授权访问")
		return
	}
	req.TenantID = tenantID.(int)

	stats, err := c.processEngine.ProcessInstanceService().GetInstanceStatistics(ctx, &req)
	if err != nil {
		common.InternalError(ctx, "获取实例统计失败: "+err.Error())
		return
	}

	common.Success(ctx, stats)
}

// GetTaskStats 获取任务统计
func (c *BPMNWorkflowController) GetTaskStats(ctx *gin.Context) {
	var req service.TaskStatisticsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.AuthFailed(ctx, "未授权访问")
		return
	}
	req.TenantID = tenantID.(int)

	stats, err := c.processEngine.TaskService().GetTaskStatistics(ctx, &req)
	if err != nil {
		common.InternalError(ctx, "获取任务统计失败: "+err.Error())
		return
	}

	common.Success(ctx, stats)
}

// CreateCounterSignTasks 创建会签任务
func (c *BPMNWorkflowController) CreateCounterSignTasks(ctx *gin.Context) {
	taskID := ctx.Param("id")

	var req service.CounterSignRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	tasks, err := c.processEngine.TaskService().CreateCounterSignTasks(workflowCtx, taskID, &req)
	if err != nil {
		common.InternalError(ctx, "创建会签任务失败: "+err.Error())
		return
	}

	common.Success(ctx, tasks)
}

// GetCounterSignStatus 获取会签状态
func (c *BPMNWorkflowController) GetCounterSignStatus(ctx *gin.Context) {
	taskID := ctx.Param("id")
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	status, err := c.processEngine.TaskService().GetCounterSignStatus(workflowCtx, taskID)
	if err != nil {
		common.InternalError(ctx, "获取会签状态失败: "+err.Error())
		return
	}

	common.Success(ctx, status)
}

// Vote 投票
func (c *BPMNWorkflowController) Vote(ctx *gin.Context) {
	taskID := ctx.Param("id")

	var req service.VoteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	workflowCtx, _, ok := getBPMNTenantContext(ctx)
	if !ok {
		return
	}

	err := c.processEngine.TaskService().Vote(workflowCtx, taskID, &req)
	if err != nil {
		common.InternalError(ctx, "投票失败: "+err.Error())
		return
	}

	common.Success(ctx, nil)
}

// GetVersionChangeLogs 获取流程定义的版本变更日志列表
func (c *BPMNWorkflowController) GetVersionChangeLogs(ctx *gin.Context) {
	processKey := ctx.Param("key")
	tenantID := ctx.GetInt("tenant_id")

	changelogs, err := c.versionService.GetChangeLogsByProcessKey(ctx, processKey, tenantID)
	if err != nil {
		common.InternalError(ctx, "获取变更日志失败: "+err.Error())
		return
	}

	common.Success(ctx, changelogs)
}

// GetVersionChangeLogsByID 根据流程定义ID获取版本变更日志
func (c *BPMNWorkflowController) GetVersionChangeLogsByID(ctx *gin.Context) {
	processDefIDStr := ctx.Param("id")
	processDefID, err := strconv.Atoi(processDefIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的流程定义ID")
		return
	}

	changelogs, err := c.versionService.GetChangeLogsByProcessDefinitionID(ctx, processDefID)
	if err != nil {
		common.InternalError(ctx, "获取变更日志失败: "+err.Error())
		return
	}

	common.Success(ctx, changelogs)
}
