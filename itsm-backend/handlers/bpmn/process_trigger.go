package bpmn

import (
	"errors"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// ProcessTriggerHandler 流程触发控制器
type ProcessTriggerHandler struct {
	triggerService *service.ProcessTriggerService
	bindingService *service.ProcessBindingService
	configService  *service.ConfigInheritanceService
}

// NewProcessTriggerHandler 创建流程触发控制器
func NewProcessTriggerHandler(triggerService *service.ProcessTriggerService, bindingService *service.ProcessBindingService, configService *service.ConfigInheritanceService) *ProcessTriggerHandler {
	return &ProcessTriggerHandler{
		triggerService: triggerService,
		bindingService: bindingService,
		configService:  configService,
	}
}

// RegisterRoutes 注册路由
//
// 授权契约（2026-09-17 P0「越权写收口」）：流程触发与流程绑定决定「哪类业务走哪条流程」，
// 属流程配置面，统一收归 bpmn:write（读为 bpmn:read）。此前整组零校验，
// 仅靠 ResourceActionMap 的粗粒度路径预检兜底，而 bpmn:write 曾同时授予 technician
// → 二线技术员可触发任意流程、改写流程绑定（prod 实测 400 而非 403）。
func (c *ProcessTriggerHandler) RegisterRoutes(r *gin.RouterGroup) {
	// 流程触发
	trigger := r.Group("/process-trigger")
	{
		trigger.POST("", middleware.RequirePermission("bpmn", "write"), c.TriggerProcess)
		trigger.GET("/status/:instance_id", middleware.RequirePermission("bpmn", "read"), c.GetProcessStatus)
		trigger.POST("/cancel/:instance_id", middleware.RequirePermission("bpmn", "write"), c.CancelProcess)
		trigger.POST("/suspend/:instance_id", middleware.RequirePermission("bpmn", "write"), c.SuspendProcess)
		trigger.POST("/resume/:instance_id", middleware.RequirePermission("bpmn", "write"), c.ResumeProcess)
	}

	// 流程绑定管理
	bindings := r.Group("/process-bindings")
	{
		bindings.POST("", middleware.RequirePermission("bpmn", "write"), c.CreateBinding)
		bindings.GET("", middleware.RequirePermission("bpmn", "read"), c.QueryBindings)
		bindings.GET("/by-type/:business_type", middleware.RequirePermission("bpmn", "read"), c.GetBindingsByBusinessType)
		bindings.GET("/:id", middleware.RequirePermission("bpmn", "read"), c.GetBinding)
		bindings.PUT("/:id", middleware.RequirePermission("bpmn", "write"), c.UpdateBinding)
		bindings.DELETE("/:id", middleware.RequirePermission("bpmn", "write"), c.DeleteBinding)
	}

	// 部门流程配置（读写均落在 department 资源上，与路径预检 /api/v1/departments 一致）
	departments := r.Group("/departments")
	{
		departments.GET("/:id/processes", middleware.RequirePermission("department", "read"), c.GetDepartmentProcesses)
		departments.POST("/:id/init-processes", middleware.RequirePermission("department", "write"), c.InitDepartmentProcesses)
	}

	// 域流程配置
	domainConfigs := r.Group("/domain-configs")
	{
		domainConfigs.GET("", middleware.RequirePermission("bpmn", "read"), c.ListDomainConfigs)
		domainConfigs.POST("", middleware.RequirePermission("bpmn", "write"), c.SetDomainConfig)
		domainConfigs.GET("/effective", middleware.RequirePermission("bpmn", "read"), c.GetEffectiveDomainConfig)
	}
}

// TriggerProcess 触发流程
// @Summary 触发流程
// @Description 根据业务类型和业务ID触发对应的流程
// @Tags BPMN-ProcessTrigger
// @Accept json
// @Produce json
// @Param request body dto.ProcessTriggerRequest true "流程触发请求"
// @Success 200 {object} common.Response{dto.ProcessTriggerResponse}
// @Failure 400 {object} common.Response
// @Router /api/v1/process-trigger [post]
func (c *ProcessTriggerHandler) TriggerProcess(ctx *gin.Context) {
	var req dto.ProcessTriggerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, err.Error())
		return
	}

	// 租户上下文取自中间件：历史实现为「缺失时沿用请求体」，存在跨租户触发风险，
	// 且 tenantID 在上下文缺失时会 panic（nil 断言）。改为 fail-closed 401。
	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	req.TenantID = tenantID

	result, err := c.triggerService.TriggerProcess(ctx.Request.Context(), &req)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// GetProcessStatus 获取流程状态
func (c *ProcessTriggerHandler) GetProcessStatus(ctx *gin.Context) {
	instanceID, err := strconv.Atoi(ctx.Param("instance_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的流程实例ID")
		return
	}

	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}

	result, err := c.triggerService.GetProcessStatus(ctx.Request.Context(), instanceID, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// CancelProcess 取消流程
func (c *ProcessTriggerHandler) CancelProcess(ctx *gin.Context) {
	instanceID, err := strconv.Atoi(ctx.Param("instance_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的流程实例ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	ctx.ShouldBindJSON(&req)

	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}

	err = c.triggerService.CancelProcess(ctx.Request.Context(), instanceID, req.Reason, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程已取消", nil)
}

// SuspendProcess 暂停流程
func (c *ProcessTriggerHandler) SuspendProcess(ctx *gin.Context) {
	instanceID, err := strconv.Atoi(ctx.Param("instance_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的流程实例ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	ctx.ShouldBindJSON(&req)

	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}

	err = c.triggerService.SuspendProcess(ctx.Request.Context(), instanceID, req.Reason, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程已暂停", nil)
}

// ResumeProcess 恢复流程
func (c *ProcessTriggerHandler) ResumeProcess(ctx *gin.Context) {
	instanceID, err := strconv.Atoi(ctx.Param("instance_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的流程实例ID")
		return
	}

	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}

	err = c.triggerService.ResumeProcess(ctx.Request.Context(), instanceID, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程已恢复", nil)
}

// CreateBinding 创建流程绑定
func (c *ProcessTriggerHandler) CreateBinding(ctx *gin.Context) {
	var binding dto.ProcessBinding
	if err := ctx.ShouldBindJSON(&binding); err != nil {
		common.Fail(ctx, 1001, err.Error())
		return
	}

	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	binding.TenantID = tenantID

	result, err := c.bindingService.CreateBinding(ctx.Request.Context(), &binding)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// GetBinding 获取流程绑定
func (c *ProcessTriggerHandler) GetBinding(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的绑定ID")
		return
	}

	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}

	result, err := c.bindingService.GetBinding(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// QueryBindings 查询流程绑定列表
func (c *ProcessTriggerHandler) QueryBindings(ctx *gin.Context) {
	var req dto.ProcessBindingQueryRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		common.Fail(ctx, 1001, err.Error())
		return
	}

	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	req.TenantID = tenantID

	result, err := c.bindingService.QueryBindings(ctx.Request.Context(), &req)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// UpdateBinding 更新流程绑定
func (c *ProcessTriggerHandler) UpdateBinding(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的绑定ID")
		return
	}

	var binding dto.ProcessBinding
	if err := ctx.ShouldBindJSON(&binding); err != nil {
		common.Fail(ctx, 1001, err.Error())
		return
	}

	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	binding.TenantID = tenantID

	result, err := c.bindingService.UpdateBinding(ctx.Request.Context(), id, &binding)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// DeleteBinding 删除流程绑定
func (c *ProcessTriggerHandler) DeleteBinding(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的绑定ID")
		return
	}

	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}

	err = c.bindingService.DeleteBinding(ctx.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "绑定已删除", nil)
}

// GetBindingsByBusinessType 根据业务类型获取绑定列表
func (c *ProcessTriggerHandler) GetBindingsByBusinessType(ctx *gin.Context) {
	businessType := dto.BusinessType(ctx.Param("business_type"))

	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}

	result, err := c.bindingService.GetBindingsByBusinessType(ctx.Request.Context(), businessType, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// GetDepartmentProcesses 获取部门专属流程绑定
func (c *ProcessTriggerHandler) GetDepartmentProcesses(ctx *gin.Context) {
	departmentID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的部门ID")
		return
	}

	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}

	result, err := c.bindingService.GetDepartmentBindings(ctx.Request.Context(), tenantID, departmentID)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// InitDepartmentProcesses 初始化部门默认流程模板
func (c *ProcessTriggerHandler) InitDepartmentProcesses(ctx *gin.Context) {
	departmentID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的部门ID")
		return
	}

	var req struct {
		DepartmentType string `json:"departmentType" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, err.Error())
		return
	}

	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}

	if err := c.bindingService.InitDepartmentDefaultBindings(ctx.Request.Context(), tenantID, departmentID, req.DepartmentType); err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "部门流程模板已初始化", nil)
}

// ListDomainConfigs 查询当前租户配置
func (c *ProcessTriggerHandler) ListDomainConfigs(ctx *gin.Context) {
	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	configType := ctx.Query("configType")

	result, err := c.configService.ListConfigs(ctx.Request.Context(), tenantID, configType)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, dto.ToDomainConfigResponseList(result))
}

// SetDomainConfig 创建或更新层级配置
func (c *ProcessTriggerHandler) SetDomainConfig(ctx *gin.Context) {
	var req struct {
		ConfigType   string                 `json:"configType" binding:"required"`
		ConfigKey    string                 `json:"configKey" binding:"required"`
		ConfigValue  map[string]interface{} `json:"configValue" binding:"required"`
		InheritMode  string                 `json:"inheritMode"`
		DepartmentID int                    `json:"departmentId"`
		TeamID       int                    `json:"teamId"`
		Description  string                 `json:"description"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, err.Error())
		return
	}
	if req.InheritMode == "" {
		req.InheritMode = "inherit"
	}

	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	err := c.configService.SetConfig(
		ctx.Request.Context(),
		tenantID,
		req.DepartmentID,
		req.TeamID,
		req.ConfigType,
		req.ConfigKey,
		req.ConfigValue,
		req.InheritMode,
		req.Description,
	)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "配置已保存", nil)
}

// GetEffectiveDomainConfig 获取继承解析后的有效配置
func (c *ProcessTriggerHandler) GetEffectiveDomainConfig(ctx *gin.Context) {
	tenantID, tenantOK := middleware.TenantIDOrUnauthorized(ctx)
	if !tenantOK {
		return
	}
	departmentID, err := parseOptionalIntQuery(ctx, "departmentId")
	if err != nil {
		common.Fail(ctx, 1001, "无效的部门ID")
		return
	}
	teamID, err := parseOptionalIntQuery(ctx, "teamId")
	if err != nil {
		common.Fail(ctx, 1001, "无效的团队ID")
		return
	}
	configType := ctx.Query("configType")
	configKey := ctx.Query("configKey")
	if configType == "" || configKey == "" {
		common.Fail(ctx, 1001, "configType 和 configKey 不能为空")
		return
	}

	result, err := c.configService.GetEffectiveConfig(ctx.Request.Context(), tenantID, departmentID, teamID, configType, configKey)
	if err != nil {
		// 各继承层级均未命中配置是合法空结果，返回 data=null 而非 500。
		if errors.Is(err, service.ErrConfigNotFound) {
			common.Success(ctx, nil)
			return
		}
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

func parseOptionalIntQuery(ctx *gin.Context, key string) (int, error) {
	value := ctx.Query(key)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}
