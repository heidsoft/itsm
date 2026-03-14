package controller

import (
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// BPMNDashboardController BPMN监控仪表盘控制器
type BPMNDashboardController struct {
	metricsService *service.BPMNMetricsService
	auditService   *service.BPMNAuditService
	tenantService  *service.BPMNTenantService
	slaService     *service.BPMNSLAService
}

// NewBPMNDashboardController 创建BPMN监控仪表盘控制器
func NewBPMNDashboardController(
	metricsService *service.BPMNMetricsService,
	auditService *service.BPMNAuditService,
	tenantService *service.BPMNTenantService,
	slaService *service.BPMNSLAService,
) *BPMNDashboardController {
	return &BPMNDashboardController{
		metricsService: metricsService,
		auditService:   auditService,
		tenantService:  tenantService,
		slaService:    slaService,
	}
}

// RegisterRoutes 注册路由
func (c *BPMNDashboardController) RegisterRoutes(r *gin.RouterGroup) {
	dashboard := r.Group("/bpmn/dashboard")
	{
		// 仪表盘
		dashboard.GET("/metrics", c.GetDashboardMetrics)
		dashboard.GET("/process/:key/metrics", c.GetProcessMetrics)

		// 审计日志
		dashboard.GET("/audit-logs", c.GetAuditLogs)
		dashboard.GET("/audit-logs/timeline", c.GetProcessTimeline)
		dashboard.GET("/audit-logs/user/:userId", c.GetUserActivity)

		// SLA
		dashboard.GET("/sla/violations", c.GetSLAViolations)
		dashboard.GET("/sla/compliance", c.GetSLACompliance)

		// 租户统计
		dashboard.GET("/tenant/stats", c.GetTenantStats)

		// 瓶颈分析
		dashboard.GET("/bottlenecks", c.GetBottleneckAnalysis)
	}
}

// GetDashboardMetrics 获取仪表盘指标
// @Summary 获取仪表盘指标
// @Tags BPMN仪表盘
// @Produce json
// @Param tenant_id query int true "租户ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} common.Response
func (c *BPMNDashboardController) GetDashboardMetrics(ctx *gin.Context) {
	tenantID, err := strconv.Atoi(ctx.Query("tenant_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的租户ID")
		return
	}

	// 默认查询最近7天
	startTime := time.Now().AddDate(0, 0, -7)
	endTime := time.Now()

	if startTimeStr := ctx.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse("2006-01-02", startTimeStr); err == nil {
			startTime = t
		}
	}

	if endTimeStr := ctx.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse("2006-01-02", endTimeStr); err == nil {
			endTime = t
		}
	}

	metrics, err := c.metricsService.GetDashboardMetrics(ctx.Request.Context(), tenantID, startTime, endTime)
	if err != nil {
		common.Fail(ctx, 5001, "获取仪表盘指标失败: "+err.Error())
		return
	}

	common.Success(ctx, metrics)
}

// GetProcessMetrics 获取流程指标
// @Summary 获取流程指标
// @Tags BPMN仪表盘
// @Produce json
// @Param key path string true "流程定义Key"
// @Param tenant_id query int true "租户ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} common.Response
func (c *BPMNDashboardController) GetProcessMetrics(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		common.Fail(ctx, 1001, "流程定义Key不能为空")
		return
	}

	tenantID, err := strconv.Atoi(ctx.Query("tenant_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的租户ID")
		return
	}

	// 默认查询最近7天
	startTime := time.Now().AddDate(0, 0, -7)
	endTime := time.Now()

	if startTimeStr := ctx.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse("2006-01-02", startTimeStr); err == nil {
			startTime = t
		}
	}

	if endTimeStr := ctx.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse("2006-01-02", endTimeStr); err == nil {
			endTime = t
		}
	}

	metrics, err := c.metricsService.GetProcessMetrics(ctx.Request.Context(), key, tenantID, startTime, endTime)
	if err != nil {
		common.Fail(ctx, 5001, "获取流程指标失败: "+err.Error())
		return
	}

	common.Success(ctx, metrics)
}

// GetAuditLogs 获取审计日志
// @Summary 获取审计日志
// @Tags BPMN仪表盘
// @Produce json
// @Param tenant_id query int true "租户ID"
// @Param process_instance_id query int false "流程实例ID"
// @Param process_definition_key query string false "流程定义Key"
// @Param action query string false "操作类型"
// @Param user_id query int false "用户ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} common.Response
func (c *BPMNDashboardController) GetAuditLogs(ctx *gin.Context) {
	tenantID, err := strconv.Atoi(ctx.Query("tenant_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的租户ID")
		return
	}

	req := &service.QueryAuditLogsRequest{
		TenantID: tenantID,
	}

	if v := ctx.Query("process_instance_id"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			req.ProcessInstanceID = id
		}
	}

	if v := ctx.Query("process_definition_key"); v != "" {
		req.ProcessDefinitionKey = v
	}

	if v := ctx.Query("action"); v != "" {
		req.Action = v
	}

	if v := ctx.Query("user_id"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			req.UserID = id
		}
	}

	if v := ctx.Query("activity_type"); v != "" {
		req.ActivityType = v
	}

	if startTimeStr := ctx.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse("2006-01-02", startTimeStr); err == nil {
			req.StartTime = t
		}
	}

	if endTimeStr := ctx.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse("2006-01-02", endTimeStr); err == nil {
			req.EndTime = t
		}
	}

	if v := ctx.Query("page"); v != "" {
		if page, err := strconv.Atoi(v); err == nil {
			req.Page = page
		}
	}

	if v := ctx.Query("page_size"); v != "" {
		if pageSize, err := strconv.Atoi(v); err == nil {
			req.PageSize = pageSize
		}
	}

	if req.PageSize == 0 {
		req.PageSize = 20
	}

	logs, total, err := c.auditService.QueryAuditLogs(ctx.Request.Context(), req)
	if err != nil {
		common.Fail(ctx, 5001, "查询审计日志失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{
		"list":  logs,
		"total": total,
		"page":  req.Page,
	})
}

// GetProcessTimeline 获取流程时间线
// @Summary 获取流程时间线
// @Tags BPMN仪表盘
// @Produce json
// @Param process_instance_key path string true "流程实例Key"
// @Success 200 {object} common.Response
func (c *BPMNDashboardController) GetProcessTimeline(ctx *gin.Context) {
	processInstanceKey := ctx.Param("process_instance_key")
	if processInstanceKey == "" {
		common.Fail(ctx, 1001, "流程实例Key不能为空")
		return
	}

	timeline, err := c.auditService.GetProcessTimeline(ctx.Request.Context(), processInstanceKey)
	if err != nil {
		common.Fail(ctx, 5001, "获取流程时间线失败: "+err.Error())
		return
	}

	common.Success(ctx, timeline)
}

// GetUserActivity 获取用户活动
// @Summary 获取用户活动
// @Tags BPMN仪表盘
// @Produce json
// @Param userId path int true "用户ID"
// @Param tenant_id query int true "租户ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} common.Response
func (c *BPMNDashboardController) GetUserActivity(ctx *gin.Context) {
	userID, err := strconv.Atoi(ctx.Param("userId"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的用户ID")
		return
	}

	tenantID, err := strconv.Atoi(ctx.Query("tenant_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的租户ID")
		return
	}

	// 默认查询最近7天
	startTime := time.Now().AddDate(0, 0, -7)
	endTime := time.Now()

	if startTimeStr := ctx.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse("2006-01-02", startTimeStr); err == nil {
			startTime = t
		}
	}

	if endTimeStr := ctx.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse("2006-01-02", endTimeStr); err == nil {
			endTime = t
		}
	}

	activity, err := c.auditService.GetUserActivity(ctx.Request.Context(), userID, tenantID, startTime, endTime)
	if err != nil {
		common.Fail(ctx, 5001, "获取用户活动失败: "+err.Error())
		return
	}

	common.Success(ctx, activity)
}

// GetSLAViolations 获取SLA违规
// @Summary 获取SLA违规
// @Tags BPMN仪表盘
// @Produce json
// @Param tenant_id query int true "租户ID"
// @Success 200 {object} common.Response
func (c *BPMNDashboardController) GetSLAViolations(ctx *gin.Context) {
	tenantID, err := strconv.Atoi(ctx.Query("tenant_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的租户ID")
		return
	}

	violations, err := c.slaService.CheckSLAViolations(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, 5001, "获取SLA违规失败: "+err.Error())
		return
	}

	common.Success(ctx, violations)
}

// GetSLACompliance 获取SLA合规率
// @Summary 获取SLA合规率
// @Tags BPMN仪表盘
// @Produce json
// @Param key query string true "流程定义Key"
// @Param tenant_id query int true "租户ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} common.Response
func (c *BPMNDashboardController) GetSLACompliance(ctx *gin.Context) {
	key := ctx.Query("key")
	if key == "" {
		common.Fail(ctx, 1001, "流程定义Key不能为空")
		return
	}

	tenantID, err := strconv.Atoi(ctx.Query("tenant_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的租户ID")
		return
	}

	// 默认查询最近7天
	startTime := time.Now().AddDate(0, 0, -7)
	endTime := time.Now()

	if startTimeStr := ctx.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse("2006-01-02", startTimeStr); err == nil {
			startTime = t
		}
	}

	if endTimeStr := ctx.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse("2006-01-02", endTimeStr); err == nil {
			endTime = t
		}
	}

	rate, compliant, total, err := c.slaService.GetSLAComplianceRate(ctx.Request.Context(), key, startTime, endTime, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, "获取SLA合规率失败: "+err.Error())
		return
	}

	common.Success(ctx, gin.H{
		"compliance_rate": rate,
		"compliant":      compliant,
		"total":          total,
	})
}

// GetTenantStats 获取租户统计
// @Summary 获取租户统计
// @Tags BPMN仪表盘
// @Produce json
// @Param tenant_id query int true "租户ID"
// @Success 200 {object} common.Response
func (c *BPMNDashboardController) GetTenantStats(ctx *gin.Context) {
	tenantID, err := strconv.Atoi(ctx.Query("tenant_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的租户ID")
		return
	}

	stats, err := c.tenantService.GetTenantStatistics(ctx.Request.Context(), tenantID)
	if err != nil {
		common.Fail(ctx, 5001, "获取租户统计失败: "+err.Error())
		return
	}

	common.Success(ctx, stats)
}

// GetBottleneckAnalysis 获取瓶颈分析
// @Summary 获取瓶颈分析
// @Tags BPMN仪表盘
// @Produce json
// @Param key query string true "流程定义Key"
// @Param tenant_id query int true "租户ID"
// @Success 200 {object} common.Response
func (c *BPMNDashboardController) GetBottleneckAnalysis(ctx *gin.Context) {
	key := ctx.Query("key")
	if key == "" {
		common.Fail(ctx, 1001, "流程定义Key不能为空")
		return
	}

	tenantID, err := strconv.Atoi(ctx.Query("tenant_id"))
	if err != nil {
		common.Fail(ctx, 1001, "无效的租户ID")
		return
	}

	bottlenecks, err := c.metricsService.GetBottleneckAnalysis(ctx.Request.Context(), key, tenantID)
	if err != nil {
		common.Fail(ctx, 5001, "获取瓶颈分析失败: "+err.Error())
		return
	}

	common.Success(ctx, bottlenecks)
}
