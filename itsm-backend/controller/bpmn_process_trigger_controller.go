package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// BPMNProcessTriggerController 流程触发控制器
type BPMNProcessTriggerController struct {
	triggerService *service.ProcessTriggerService
	bindingService *service.ProcessBindingService
}

// NewBPMNProcessTriggerController 创建流程触发控制器
func NewBPMNProcessTriggerController(triggerService *service.ProcessTriggerService, bindingService *service.ProcessBindingService) *BPMNProcessTriggerController {
	return &BPMNProcessTriggerController{
		triggerService: triggerService,
		bindingService: bindingService,
	}
}

// RegisterRoutes 注册路由
func (c *BPMNProcessTriggerController) RegisterRoutes(r *gin.RouterGroup) {
	// 流程触发
	trigger := r.Group("/process-trigger")
	{
		trigger.POST("", c.TriggerProcess)
		trigger.GET("/status/:instance_id", c.GetProcessStatus)
		trigger.POST("/cancel/:instance_id", c.CancelProcess)
		trigger.POST("/suspend/:instance_id", c.SuspendProcess)
		trigger.POST("/resume/:instance_id", c.ResumeProcess)
	}

	// 流程绑定管理
	bindings := r.Group("/process-bindings")
	{
		bindings.POST("", c.CreateBinding)
		bindings.GET("", c.QueryBindings)
		bindings.GET("/:id", c.GetBinding)
		bindings.PUT("/:id", c.UpdateBinding)
		bindings.DELETE("/:id", c.DeleteBinding)
		bindings.GET("/by-type/:business_type", c.GetBindingsByBusinessType)
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
func (c *BPMNProcessTriggerController) TriggerProcess(ctx *gin.Context) {
	var req dto.ProcessTriggerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, 1001, err.Error())
		return
	}

	// 从上下文获取租户ID
	tenantID, _ := ctx.Get("tenant_id")
	if tenantID != nil {
		req.TenantID = tenantID.(int)
	}

	result, err := c.triggerService.TriggerProcess(ctx.Request.Context(), &req)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// GetProcessStatus 获取流程状态
func (c *BPMNProcessTriggerController) GetProcessStatus(ctx *gin.Context) {
	instanceID, err := strconv.Atoi(ctx.Param("instance_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的流程实例ID")
		return
	}

	tenantID, _ := ctx.Get("tenant_id")

	result, err := c.triggerService.GetProcessStatus(ctx.Request.Context(), instanceID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// CancelProcess 取消流程
func (c *BPMNProcessTriggerController) CancelProcess(ctx *gin.Context) {
	instanceID, err := strconv.Atoi(ctx.Param("instance_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的流程实例ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	ctx.ShouldBindJSON(&req)

	tenantID, _ := ctx.Get("tenant_id")

	err = c.triggerService.CancelProcess(ctx.Request.Context(), instanceID, req.Reason, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程已取消", nil)
}

// SuspendProcess 暂停流程
func (c *BPMNProcessTriggerController) SuspendProcess(ctx *gin.Context) {
	instanceID, err := strconv.Atoi(ctx.Param("instance_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的流程实例ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	ctx.ShouldBindJSON(&req)

	tenantID, _ := ctx.Get("tenant_id")

	err = c.triggerService.SuspendProcess(ctx.Request.Context(), instanceID, req.Reason, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程已暂停", nil)
}

// ResumeProcess 恢复流程
func (c *BPMNProcessTriggerController) ResumeProcess(ctx *gin.Context) {
	instanceID, err := strconv.Atoi(ctx.Param("instance_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的流程实例ID")
		return
	}

	tenantID, _ := ctx.Get("tenant_id")

	err = c.triggerService.ResumeProcess(ctx.Request.Context(), instanceID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "流程已恢复", nil)
}

// CreateBinding 创建流程绑定
func (c *BPMNProcessTriggerController) CreateBinding(ctx *gin.Context) {
	var binding dto.ProcessBinding
	if err := ctx.ShouldBindJSON(&binding); err != nil {
		common.Fail(ctx, 1001, err.Error())
		return
	}

	tenantID, _ := ctx.Get("tenant_id")
	binding.TenantID = tenantID.(int)

	result, err := c.bindingService.CreateBinding(ctx.Request.Context(), &binding)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// GetBinding 获取流程绑定
func (c *BPMNProcessTriggerController) GetBinding(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的绑定ID")
		return
	}

	tenantID, _ := ctx.Get("tenant_id")

	result, err := c.bindingService.GetBinding(ctx.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// QueryBindings 查询流程绑定列表
func (c *BPMNProcessTriggerController) QueryBindings(ctx *gin.Context) {
	var req dto.ProcessBindingQueryRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		common.Fail(ctx, 1001, err.Error())
		return
	}

	tenantID, _ := ctx.Get("tenant_id")
	req.TenantID = tenantID.(int)

	result, err := c.bindingService.QueryBindings(ctx.Request.Context(), &req)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// UpdateBinding 更新流程绑定
func (c *BPMNProcessTriggerController) UpdateBinding(ctx *gin.Context) {
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

	result, err := c.bindingService.UpdateBinding(ctx.Request.Context(), id, &binding)
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}

// DeleteBinding 删除流程绑定
func (c *BPMNProcessTriggerController) DeleteBinding(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的绑定ID")
		return
	}

	tenantID, _ := ctx.Get("tenant_id")

	err = c.bindingService.DeleteBinding(ctx.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.SuccessWithMessage(ctx, "绑定已删除", nil)
}

// GetBindingsByBusinessType 根据业务类型获取绑定列表
func (c *BPMNProcessTriggerController) GetBindingsByBusinessType(ctx *gin.Context) {
	businessType := dto.BusinessType(ctx.Param("business_type"))

	tenantID, _ := ctx.Get("tenant_id")

	result, err := c.bindingService.GetBindingsByBusinessType(ctx.Request.Context(), businessType, tenantID.(int))
	if err != nil {
		common.Fail(ctx, 5001, err.Error())
		return
	}

	common.Success(ctx, result)
}
