package controller

import (
	"net/http"
	"strconv"
	"time"

	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// BPMNMonitoringController BPMN监控控制器
type BPMNMonitoringController struct {
	monitoringService *service.BPMNMonitoringService
}

// NewBPMNMonitoringController 创建BPMN监控控制器
func NewBPMNMonitoringController(monitoringService *service.BPMNMonitoringService) *BPMNMonitoringController {
	return &BPMNMonitoringController{
		monitoringService: monitoringService,
	}
}

// RegisterRoutes 注册路由
func (c *BPMNMonitoringController) RegisterRoutes(r *gin.RouterGroup) {
	monitoring := r.Group("/bpmn/monitoring")
	{
		// 流程指标监控
		monitoring.GET("/metrics", c.GetProcessMetrics)
		monitoring.GET("/metrics/:processKey", c.GetProcessMetricsByKey)
		
		// 流程实例状态监控
		monitoring.GET("/instances/:instanceId/status", c.GetProcessInstanceStatus)
		monitoring.GET("/instances/status", c.ListProcessInstancesStatus)
		
		// 性能监控
		monitoring.GET("/performance", c.GetPerformanceMetrics)
		monitoring.GET("/performance/alerts", c.GetPerformanceAlerts)
		
		// 系统健康检查
		monitoring.GET("/health", c.GetSystemHealth)
		
		// 审计日志
		monitoring.GET("/audit-logs", c.GetAuditLogs)
	}
}

// GetProcessMetrics 获取流程指标
func (c *BPMNMonitoringController) GetProcessMetrics(ctx *gin.Context) {
	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	// 解析查询参数
	timeRange := ctx.Query("time_range")
	if timeRange == "" {
		timeRange = "24h" // 默认24小时
	}

	req := &service.ProcessMetricsRequest{
		TenantID:  tenantID.(int),
		TimeRange: timeRange,
	}

	// 解析时间范围
	if startTimeStr := ctx.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &startTime
		}
	}

	if endTimeStr := ctx.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &endTime
		}
	}

	metrics, err := c.monitoringService.GetProcessMetrics(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取流程指标失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "获取流程指标成功",
		"data":    metrics,
	})
}

// GetProcessMetricsByKey 根据流程定义键获取指标
func (c *BPMNMonitoringController) GetProcessMetricsByKey(ctx *gin.Context) {
	processKey := ctx.Param("processKey")
	if processKey == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "流程定义键不能为空"})
		return
	}

	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	// 解析查询参数
	timeRange := ctx.Query("time_range")
	if timeRange == "" {
		timeRange = "24h"
	}

	req := &service.ProcessMetricsRequest{
		ProcessDefinitionKey: processKey,
		TenantID:            tenantID.(int),
		TimeRange:           timeRange,
	}

	// 解析时间范围
	if startTimeStr := ctx.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &startTime
		}
	}

	if endTimeStr := ctx.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &endTime
		}
	}

	metrics, err := c.monitoringService.GetProcessMetrics(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取流程指标失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "获取流程指标成功",
		"data":    metrics,
	})
}

// GetProcessInstanceStatus 获取流程实例状态
func (c *BPMNMonitoringController) GetProcessInstanceStatus(ctx *gin.Context) {
	instanceID := ctx.Param("instanceId")
	if instanceID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "流程实例ID不能为空"})
		return
	}

	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	status, err := c.monitoringService.GetProcessInstanceStatus(ctx, instanceID, tenantID.(int))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取流程实例状态失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "获取流程实例状态成功",
		"data":    status,
	})
}

// ListProcessInstancesStatus 获取流程实例状态列表
func (c *BPMNMonitoringController) ListProcessInstancesStatus(ctx *gin.Context) {
	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "未授权访问"})
		return
	}

	// 解析分页参数
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	
	// 解析查询参数
	processKey := ctx.Query("process_key")
	status := ctx.Query("status")
	assignee := ctx.Query("assignee")

	// 构建查询条件 - 使用临时结构体
	type ProcessInstanceStatusQuery struct {
		TenantID   int
		Page       int
		PageSize   int
		ProcessKey string
		Status     string
		Assignee   string
		StartTime  *time.Time
		EndTime    *time.Time
	}
	
	query := &ProcessInstanceStatusQuery{
		TenantID:   tenantID.(int),
		Page:       page,
		PageSize:   pageSize,
		ProcessKey: processKey,
		Status:     status,
		Assignee:   assignee,
	}

	// 解析时间范围
	if startTimeStr := ctx.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			query.StartTime = &startTime
		}
	}

	if endTimeStr := ctx.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			query.EndTime = &endTime
		}
	}

	// TODO: 实现批量查询流程实例状态的方法
	// 这里需要先在BPMNMonitoringService中添加相应的方法
	ctx.JSON(http.StatusOK, gin.H{
		"message": "功能开发中",
		"data":    []interface{}{},
	})
}

// GetPerformanceMetrics 获取性能指标
func (c *BPMNMonitoringController) GetPerformanceMetrics(ctx *gin.Context) {
	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	// 解析时间范围
	timeRange := ctx.DefaultQuery("time_range", "24h")
	
	req := &service.ProcessMetricsRequest{
		TenantID:  tenantID.(int),
		TimeRange: timeRange,
	}

	metrics, err := c.monitoringService.GetProcessMetrics(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取性能指标失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "获取性能指标成功",
		"data":    metrics.PerformanceMetrics,
	})
}

// GetPerformanceAlerts 获取性能告警
func (c *BPMNMonitoringController) GetPerformanceAlerts(ctx *gin.Context) {
	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	alerts, err := c.monitoringService.GetPerformanceAlerts(ctx, tenantID.(int))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取性能告警失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "获取性能告警成功",
		"data":    alerts,
	})
}

// GetSystemHealth 获取系统健康状态
func (c *BPMNMonitoringController) GetSystemHealth(ctx *gin.Context) {
	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	health, err := c.monitoringService.GetSystemHealth(ctx, tenantID.(int))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取系统健康状态失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "获取系统健康状态成功",
		"data":    health,
	})
}

// GetAuditLogs 获取审计日志
func (c *BPMNMonitoringController) GetAuditLogs(ctx *gin.Context) {
	// 从JWT获取租户ID
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	// 解析查询参数
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	userID := ctx.Query("user_id")
	action := ctx.Query("action")
	resourceType := ctx.Query("resource_type")
	resourceID := ctx.Query("resource_id")

	req := &service.AuditLogRequest{
		TenantID:   tenantID.(int),
		Page:       page,
		PageSize:   pageSize,
		UserID:     userID,
		Action:     action,
		ResourceType: resourceType,
		ResourceID: resourceID,
	}

	// 解析时间范围
	if startTimeStr := ctx.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &startTime
		}
	}

	if endTimeStr := ctx.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &endTime
		}
	}

	logs, total, err := c.monitoringService.GetAuditLogs(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取审计日志失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "获取审计日志成功",
		"data": gin.H{
			"logs":  logs,
			"total": total,
			"page":  page,
			"page_size": pageSize,
		},
	})
}
