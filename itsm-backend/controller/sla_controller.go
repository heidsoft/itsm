package controller

import (
	"strconv"

	"itsm-backend/common"
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
	var sla service.SLADefinition
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
	sla.TenantID = tenantID.(int)

	if err := c.slaService.CreateSLADefinition(ctx, &sla); err != nil {
		common.Fail(ctx, common.InternalErrorCode, "创建SLA定义失败: "+err.Error())
		return
	}

	common.Success(ctx, sla)
}

// GetSLADefinition 获取SLA定义
func (c *SLAController) GetSLADefinition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的SLA ID: "+err.Error())
		return
	}

	sla, err := c.slaService.GetSLADefinition(ctx, id)
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

	if err := c.slaService.UpdateSLADefinition(ctx, id, updates); err != nil {
		common.Fail(ctx, common.InternalErrorCode, "更新SLA定义失败: "+err.Error())
		return
	}

	common.Success(ctx, nil)
}

// DeleteSLADefinition 删除SLA定义
func (c *SLAController) DeleteSLADefinition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的SLA ID: "+err.Error())
		return
	}

	if err := c.slaService.DeleteSLADefinition(ctx, id); err != nil {
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

	slas, err := c.slaService.ListSLADefinitions(ctx, tenantID.(int), filters)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取SLA定义列表失败: "+err.Error())
		return
	}

	common.Success(ctx, slas)
}

// CheckSLACompliance 检查SLA合规性
func (c *SLAController) CheckSLACompliance(ctx *gin.Context) {
	ticketIDStr := ctx.Param("ticketId")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工单ID: "+err.Error())
		return
	}

	violation, err := c.slaService.CheckSLACompliance(ctx, ticketID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "检查SLA合规性失败: "+err.Error())
		return
	}

	if violation == nil {
		common.Success(ctx, map[string]string{"message": "SLA合规性检查完成，无违规"})
		return
	}

	common.Success(ctx, violation)
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

	metrics, err := c.slaService.GetSLAMetrics(ctx, tenantID.(int), filters)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取SLA指标失败: "+err.Error())
		return
	}

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

	violations, err := c.slaService.GetSLAViolations(ctx, tenantID.(int), filters)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取SLA违规列表失败: "+err.Error())
		return
	}

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

	if err := c.slaService.UpdateViolationStatus(ctx, id, req.Status, req.Notes); err != nil {
		common.Fail(ctx, common.InternalErrorCode, "更新违规状态失败: "+err.Error())
		return
	}

	common.Success(ctx, nil)
}
