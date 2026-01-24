package controller

import (
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// SLAController SLA管理控制器
type SLAController struct {
	slaService      *service.SLAService
	slaAlertService *service.SLAAlertService
}

// NewSLAController 创建SLA控制器实例
func NewSLAController(slaService *service.SLAService, slaAlertService *service.SLAAlertService) *SLAController {
	return &SLAController{
		slaService:      slaService,
		slaAlertService: slaAlertService,
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
	if slaDefIDStr := ctx.Query("sla_definition_id"); slaDefIDStr != "" {
		if slaDefID, err := strconv.Atoi(slaDefIDStr); err == nil && slaDefID > 0 {
			filters["sla_definition_id"] = slaDefID
		}
	}
	if metricType := ctx.Query("metric_type"); metricType != "" {
		filters["metric_type"] = metricType
	}

	metrics, err := c.slaService.GetSLAMetrics(ctx.Request.Context(), tenantID.(int), filters)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取SLA指标失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{
		"metrics": metrics,
		"count":   len(metrics),
	})
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
	if isResolvedStr := ctx.Query("is_resolved"); isResolvedStr != "" {
		if isResolved, err := strconv.ParseBool(isResolvedStr); err == nil {
			filters["is_resolved"] = isResolved
		}
	}
	if severity := ctx.Query("severity"); severity != "" {
		filters["severity"] = severity
	}
	if violationType := ctx.Query("violation_type"); violationType != "" {
		filters["violation_type"] = violationType
	}
	if slaDefIDStr := ctx.Query("sla_definition_id"); slaDefIDStr != "" {
		if slaDefID, err := strconv.Atoi(slaDefIDStr); err == nil && slaDefID > 0 {
			filters["sla_definition_id"] = slaDefID
		}
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "20"))

	violations, total, err := c.slaService.GetSLAViolations(ctx.Request.Context(), tenantID.(int), filters, page, size)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取SLA违规列表失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{
		"items":  violations,
		"total":  total,
		"page":   page,
		"size":   size,
	})
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
		IsResolved bool   `json:"is_resolved"`
		Notes      string `json:"notes"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	response, err := c.slaService.UpdateSLAViolationStatus(ctx.Request.Context(), id, req.IsResolved, req.Notes, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "更新违规状态失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// GetSLAMonitoring 获取SLA监控数据
func (c *SLAController) GetSLAMonitoring(ctx *gin.Context) {
	var req dto.SLAMonitoringRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	// 如果没有提供时间范围，使用默认值（最近30天）
	if req.StartTime == "" {
		req.StartTime = time.Now().AddDate(0, 0, -30).Format(time.RFC3339)
	}
	if req.EndTime == "" {
		req.EndTime = time.Now().Format(time.RFC3339)
	}

	response, err := c.slaService.GetSLAMonitoring(ctx.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取SLA监控数据失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// CreateAlertRule 创建SLA预警规则
func (c *SLAController) CreateAlertRule(ctx *gin.Context) {
	var req dto.CreateSLAAlertRuleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	response, err := c.slaAlertService.CreateAlertRule(ctx.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "创建预警规则失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// UpdateAlertRule 更新SLA预警规则
func (c *SLAController) UpdateAlertRule(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的预警规则ID: "+err.Error())
		return
	}

	var req dto.UpdateSLAAlertRuleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	response, err := c.slaAlertService.UpdateAlertRule(ctx.Request.Context(), id, &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "更新预警规则失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// DeleteAlertRule 删除SLA预警规则
func (c *SLAController) DeleteAlertRule(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的预警规则ID: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	err = c.slaAlertService.DeleteAlertRule(ctx.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "删除预警规则失败: "+err.Error())
		return
	}

	common.Success(ctx, map[string]string{"message": "预警规则已删除"})
}

// ListAlertRules 获取SLA预警规则列表
func (c *SLAController) ListAlertRules(ctx *gin.Context) {
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	filters := make(map[string]interface{})
	if slaDefinitionIDStr := ctx.Query("sla_definition_id"); slaDefinitionIDStr != "" {
		if slaDefinitionID, err := strconv.Atoi(slaDefinitionIDStr); err == nil {
			filters["sla_definition_id"] = slaDefinitionID
		}
	}
	if isActiveStr := ctx.Query("is_active"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			filters["is_active"] = isActive
		}
	}
	if alertLevel := ctx.Query("alert_level"); alertLevel != "" {
		filters["alert_level"] = alertLevel
	}

	rules, err := c.slaAlertService.ListAlertRules(ctx.Request.Context(), filters, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取预警规则列表失败: "+err.Error())
		return
	}

	common.Success(ctx, rules)
}

// GetAlertRule 获取SLA预警规则详情
func (c *SLAController) GetAlertRule(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的预警规则ID: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	rule, err := c.slaAlertService.GetAlertRule(ctx.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.NotFoundCode, "预警规则不存在: "+err.Error())
		return
	}

	common.Success(ctx, rule)
}

// GetAlertHistory 获取SLA预警历史
func (c *SLAController) GetAlertHistory(ctx *gin.Context) {
	var req dto.GetSLAAlertHistoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// 尝试从查询参数获取
		if slaDefinitionIDStr := ctx.Query("sla_definition_id"); slaDefinitionIDStr != "" {
			if id, err := strconv.Atoi(slaDefinitionIDStr); err == nil {
				req.SLADefinitionID = &id
			}
		}
		if alertRuleIDStr := ctx.Query("alert_rule_id"); alertRuleIDStr != "" {
			if id, err := strconv.Atoi(alertRuleIDStr); err == nil {
				req.AlertRuleID = &id
			}
		}
		if ticketIDStr := ctx.Query("ticket_id"); ticketIDStr != "" {
			if id, err := strconv.Atoi(ticketIDStr); err == nil {
				req.TicketID = &id
			}
		}
		if alertLevel := ctx.Query("alert_level"); alertLevel != "" {
			req.AlertLevel = &alertLevel
		}
		req.StartTime = ctx.DefaultQuery("start_time", "")
		req.EndTime = ctx.DefaultQuery("end_time", "")
		req.Page = 1
		if pageStr := ctx.Query("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				req.Page = p
			}
		}
		req.PageSize = 20
		if pageSizeStr := ctx.Query("page_size"); pageSizeStr != "" {
			if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
				req.PageSize = ps
			}
		}
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	histories, total, err := c.slaAlertService.GetAlertHistory(ctx.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取预警历史失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{
		"items":     histories,
		"total":     total,
		"page":      req.Page,
		"page_size": req.PageSize,
	})
}
