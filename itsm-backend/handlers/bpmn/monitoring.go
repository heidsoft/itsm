package bpmn

import (
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// MonitoringHandler BPMN监控控制器
type MonitoringHandler struct {
	monitoringService *service.BPMNMonitoringService
}

// NewMonitoringHandler 创建BPMN监控控制器
func NewMonitoringHandler(monitoringService *service.BPMNMonitoringService) *MonitoringHandler {
	return &MonitoringHandler{
		monitoringService: monitoringService,
	}
}

// SetMonitoringService 设置监控服务（用于延迟注入）
func (c *MonitoringHandler) SetMonitoringService(s *service.BPMNMonitoringService) {
	c.monitoringService = s
}

// RegisterRoutes 注册路由
func (c *MonitoringHandler) RegisterRoutes(r *gin.RouterGroup) {
	monitoring := r.Group("/bpmn/monitoring")
	{
		// 流程指标监控
		monitoring.GET("/metrics", c.GetProcessMetrics)
		monitoring.GET("/metrics/:processKey", c.GetProcessMetricsByKey)

		// 流程实例状态监控
		monitoring.GET("/instances/:instanceId/status", c.GetProcessInstanceStatus)
		monitoring.GET("/instances/status", c.ListProcessInstancesStatus)
		// 新增：完整执行轨迹时间线
		monitoring.GET("/instances/:instanceId/timeline", c.GetProcessTimeline)

		// 性能监控
		monitoring.GET("/performance", c.GetPerformanceMetrics)
		monitoring.GET("/performance/alerts", c.GetPerformanceAlerts)

		// 系统健康检查
		monitoring.GET("/health", c.GetSystemHealth)

		// 审计日志
		monitoring.GET("/audit-logs", c.GetAuditLogs)
	}
}

// tenantIDFromCtx 从 gin.Context 解析 tenantID，缺失或类型错误时调用 Fail 并返回 false
func tenantIDFromCtx(ctx *gin.Context) (int, bool) {
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.AuthFailedCode, "未授权访问")
		return 0, false
	}
	id, ok := tenantID.(int)
	if !ok {
		common.Fail(ctx, common.InternalErrorCode, "租户ID类型错误")
		return 0, false
	}
	return id, true
}

// GetProcessMetrics 获取流程指标
func (c *MonitoringHandler) GetProcessMetrics(ctx *gin.Context) {
	tenantID, ok := tenantIDFromCtx(ctx)
	if !ok {
		return
	}

	timeRange := ctx.Query("timeRange")
	if timeRange == "" {
		timeRange = "24h"
	}

	req := &service.ProcessMetricsRequest{
		TenantID:  tenantID,
		TimeRange: timeRange,
	}

	if startTimeStr := ctx.Query("startTime"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &startTime
		}
	}

	if endTimeStr := ctx.Query("endTime"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &endTime
		}
	}

	metrics, err := c.monitoringService.GetProcessMetrics(ctx, req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取流程指标失败: "+err.Error())
		return
	}

	common.Success(ctx, metrics)
}

// GetProcessMetricsByKey 根据流程定义键获取指标
func (c *MonitoringHandler) GetProcessMetricsByKey(ctx *gin.Context) {
	processKey := ctx.Param("processKey")
	if processKey == "" {
		common.Fail(ctx, common.ParamErrorCode, "流程定义键不能为空")
		return
	}

	tenantID, ok := tenantIDFromCtx(ctx)
	if !ok {
		return
	}

	timeRange := ctx.Query("timeRange")
	if timeRange == "" {
		timeRange = "24h"
	}

	req := &service.ProcessMetricsRequest{
		ProcessDefinitionKey: processKey,
		TenantID:             tenantID,
		TimeRange:            timeRange,
	}

	if startTimeStr := ctx.Query("startTime"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &startTime
		}
	}

	if endTimeStr := ctx.Query("endTime"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &endTime
		}
	}

	metrics, err := c.monitoringService.GetProcessMetrics(ctx, req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取流程指标失败: "+err.Error())
		return
	}

	common.Success(ctx, metrics)
}

// GetProcessInstanceStatus 获取流程实例状态
func (c *MonitoringHandler) GetProcessInstanceStatus(ctx *gin.Context) {
	instanceIDStr := ctx.Param("instanceId")
	instanceID, err := strconv.Atoi(instanceIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的流程实例ID")
		return
	}

	tenantID, ok := tenantIDFromCtx(ctx)
	if !ok {
		return
	}

	status, err := c.monitoringService.GetProcessInstanceStatus(ctx, instanceID, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取流程实例状态失败: "+err.Error())
		return
	}

	common.Success(ctx, status)
}

// ListProcessInstancesStatus 获取流程实例状态列表
func (c *MonitoringHandler) ListProcessInstancesStatus(ctx *gin.Context) {
	tenantID, ok := tenantIDFromCtx(ctx)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))

	processKey := ctx.Query("processKey")
	status := ctx.Query("status")
	assignee := ctx.Query("assignee")

	query := &service.ListProcessInstanceStatusQuery{
		TenantID:   tenantID,
		Page:       page,
		PageSize:   pageSize,
		ProcessKey: processKey,
		Status:     status,
		Assignee:   assignee,
	}

	if startTimeStr := ctx.Query("startTime"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			query.StartTime = &startTime
		}
	}
	if endTimeStr := ctx.Query("endTime"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			query.EndTime = &endTime
		}
	}

	statuses, total, err := c.monitoringService.ListProcessInstancesStatus(ctx, query)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取流程实例状态失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{
		"instances": statuses,
		"total":     total,
		"page":      page,
		"pageSize":  pageSize,
	})
}

// GetProcessTimeline 获取流程实例完整时间线
func (c *MonitoringHandler) GetProcessTimeline(ctx *gin.Context) {
	processInstanceKey := ctx.Param("instanceId")
	if processInstanceKey == "" {
		common.Fail(ctx, common.ParamErrorCode, "流程实例Key不能为空")
		return
	}
	tenantID, ok := tenantIDFromCtx(ctx)
	if !ok {
		return
	}

	entries, err := c.monitoringService.GetProcessTimeline(ctx, processInstanceKey, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取流程时间线失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{
		"processInstanceId": processInstanceKey,
		"entries":           entries,
		"total":             len(entries),
	})
}

// GetPerformanceMetrics 获取性能指标
func (c *MonitoringHandler) GetPerformanceMetrics(ctx *gin.Context) {
	tenantID, ok := tenantIDFromCtx(ctx)
	if !ok {
		return
	}

	timeRange := ctx.DefaultQuery("timeRange", "24h")

	req := &service.ProcessMetricsRequest{
		TenantID:  tenantID,
		TimeRange: timeRange,
	}

	metrics, err := c.monitoringService.GetProcessMetrics(ctx, req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取性能指标失败: "+err.Error())
		return
	}

	common.Success(ctx, metrics.PerformanceMetrics)
}

// GetPerformanceAlerts 获取性能告警
func (c *MonitoringHandler) GetPerformanceAlerts(ctx *gin.Context) {
	tenantID, ok := tenantIDFromCtx(ctx)
	if !ok {
		return
	}

	alerts, err := c.monitoringService.GetPerformanceAlerts(ctx, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取性能告警失败: "+err.Error())
		return
	}

	common.Success(ctx, alerts)
}

// GetSystemHealth 获取系统健康状态
func (c *MonitoringHandler) GetSystemHealth(ctx *gin.Context) {
	tenantID, ok := tenantIDFromCtx(ctx)
	if !ok {
		return
	}

	health, err := c.monitoringService.GetSystemHealth(ctx, tenantID)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取系统健康状态失败: "+err.Error())
		return
	}

	common.Success(ctx, health)
}

// GetAuditLogs 获取审计日志
func (c *MonitoringHandler) GetAuditLogs(ctx *gin.Context) {
	tenantID, ok := tenantIDFromCtx(ctx)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	userID := ctx.Query("userId")
	action := ctx.Query("action")
	resourceType := ctx.Query("resourceType")
	resourceID := ctx.Query("resourceId")

	req := &service.AuditLogRequest{
		TenantID:     tenantID,
		Page:         page,
		PageSize:     pageSize,
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}

	if startTimeStr := ctx.Query("startTime"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &startTime
		}
	}

	if endTimeStr := ctx.Query("endTime"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &endTime
		}
	}

	logs, total, err := c.monitoringService.GetAuditLogs(ctx, req)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取审计日志失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{
		"logs":     logs,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
