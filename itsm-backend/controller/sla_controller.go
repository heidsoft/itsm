package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// SLAController SLA管理控制器
type SLAController struct {
	slaService *service.SLAService
}

// NewSLAController 创建SLA控制器实例
func NewSLAController(slaService *service.SLAService) *SLAController {
	return &SLAController{
		slaService: slaService,
	}
}

// CreateSLADefinition 创建SLA定义
func (c *SLAController) CreateSLADefinition(ctx *gin.Context) {
	var sla dto.CreateSLADefinitionRequest
	if err := ctx.ShouldBindJSON(&sla); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}
	// sla.TenantID will be set by the service

	response, err := c.slaService.CreateSLADefinition(ctx.Request.Context(), &sla, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "创建SLA定义失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// GetSLADefinition 获取SLA定义
func (c *SLAController) GetSLADefinition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的SLA ID: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	sla, err := c.slaService.GetSLADefinition(ctx.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.NotFoundCode, "SLA定义不存在: "+err.Error())
		return
	}

	common.Success(ctx, sla)
}

// UpdateSLADefinition 更新SLA定义
func (c *SLAController) UpdateSLADefinition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的SLA ID: "+err.Error())
		return
	}

	var updates map[string]interface{}
	if err := ctx.ShouldBindJSON(&updates); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	// Convert updates to UpdateSLADefinitionRequest
	req := &dto.UpdateSLADefinitionRequest{}
	// TODO: Map updates to req fields

	response, err := c.slaService.UpdateSLADefinition(ctx.Request.Context(), id, req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "更新SLA定义失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// DeleteSLADefinition 删除SLA定义
func (c *SLAController) DeleteSLADefinition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的SLA ID: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	if err := c.slaService.DeleteSLADefinition(ctx.Request.Context(), id, tenantID.(int)); err != nil {
		common.Fail(ctx, common.InternalErrorCode, "删除SLA定义失败: "+err.Error())
		return
	}

	common.Success(ctx, nil)
}

// ListSLADefinitions 获取SLA定义列表
func (c *SLAController) ListSLADefinitions(ctx *gin.Context) {
	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	// 获取查询参数
	filters := make(map[string]interface{})
	if serviceType := ctx.Query("service_type"); serviceType != "" {
		filters["service_type"] = serviceType
	}
	if priority := ctx.Query("priority"); priority != "" {
		filters["priority"] = priority
	}
	if isActiveStr := ctx.Query("is_active"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			filters["is_active"] = isActive
		}
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))

	slas, total, err := c.slaService.ListSLADefinitions(ctx.Request.Context(), tenantID.(int), page, size)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取SLA定义列表失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{
		"items":    slas,
		"total":    total,
		"page":     page,
		"pageSize": size,
	})
}

// CheckSLACompliance 检查SLA合规性
func (c *SLAController) CheckSLACompliance(ctx *gin.Context) {
	ticketIDStr := ctx.Param("ticketId")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工单ID: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	err = c.slaService.CheckSLACompliance(ctx.Request.Context(), ticketID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "检查SLA合规性失败: "+err.Error())
		return
	}

	common.Success(ctx, map[string]string{"message": "SLA合规性检查完成"})
}

// GetSLAMetrics 获取SLA指标
func (c *SLAController) GetSLAMetrics(ctx *gin.Context) {
	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	// 获取查询参数
	filters := make(map[string]interface{})
	if serviceType := ctx.Query("service_type"); serviceType != "" {
		filters["service_type"] = serviceType
	}
	if priority := ctx.Query("priority"); priority != "" {
		filters["priority"] = priority
	}

	// TODO: 实现GetSLAMetrics方法
	_ = tenantID
	metrics := []interface{}{}

	common.Success(ctx, metrics)
}

// GetSLAViolations 获取SLA违规列表
func (c *SLAController) GetSLAViolations(ctx *gin.Context) {
	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	// 获取查询参数
	filters := make(map[string]interface{})
	if status := ctx.Query("status"); status != "" {
		filters["status"] = status
	}
	if severity := ctx.Query("severity"); severity != "" {
		filters["severity"] = severity
	}
	if violationType := ctx.Query("violation_type"); violationType != "" {
		filters["violation_type"] = violationType
	}

	// TODO: 实现GetSLAViolations方法
	_ = tenantID
	violations := []interface{}{}

	common.Success(ctx, violations)
}

// UpdateViolationStatus 更新违规状态
func (c *SLAController) UpdateViolationStatus(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的违规ID: "+err.Error())
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
		Notes  string `json:"notes"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// TODO: 实现UpdateViolationStatus方法
	_ = id
	common.Success(ctx, map[string]string{"message": "违规状态更新成功"})
}
