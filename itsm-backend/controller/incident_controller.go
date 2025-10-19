package controller

import (
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type IncidentController struct {
	incidentService   *service.IncidentService
	ruleEngine        *service.IncidentRuleEngine
	monitoringService *service.IncidentMonitoringService
	alertingService   *service.IncidentAlertingService
	logger            *zap.SugaredLogger
}

func NewIncidentController(
	incidentService *service.IncidentService,
	ruleEngine *service.IncidentRuleEngine,
	monitoringService *service.IncidentMonitoringService,
	alertingService *service.IncidentAlertingService,
	logger *zap.SugaredLogger,
) *IncidentController {
	return &IncidentController{
		incidentService:   incidentService,
		ruleEngine:        ruleEngine,
		monitoringService: monitoringService,
		alertingService:   alertingService,
		logger:            logger,
	}
}

// CreateIncident 创建事件
// @Summary 创建事件
// @Description 创建新的事件记录
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param request body dto.CreateIncidentRequest true "创建事件请求"
// @Success 200 {object} common.Response{data=dto.IncidentResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents [post]
func (c *IncidentController) CreateIncident(ctx *gin.Context) {
	var req dto.CreateIncidentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Invalid request body", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数无效")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		c.logger.Errorw("Failed to get tenant ID", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	response, err := c.incidentService.CreateIncident(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to create incident", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "创建事件失败")
		return
	}

	common.SuccessWithMessage(ctx, "创建事件成功", response)
}

// GetIncident 获取事件详情
// @Summary 获取事件详情
// @Description 根据ID获取事件详细信息
// @Tags 事件管理
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} common.Response{data=dto.IncidentResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id} [get]
func (c *IncidentController) GetIncident(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	response, err := c.incidentService.GetIncident(ctx.Request.Context(), id, tenantID)
	if err != nil {
		if err.Error() == "incident not found" {
			common.Fail(ctx, common.ParamErrorCode, "事件不存在")
			return
		}
		c.logger.Errorw("Failed to get incident", "error", err, "id", id)
		common.Fail(ctx, common.InternalErrorCode, "获取事件失败")
		return
	}

	common.SuccessWithMessage(ctx, "获取事件成功", response)
}

// ListIncidents 获取事件列表
// @Summary 获取事件列表
// @Description 分页获取事件列表，支持筛选
// @Tags 事件管理
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param status query string false "状态筛选"
// @Param priority query string false "优先级筛选"
// @Param severity query string false "严重程度筛选"
// @Param category query string false "分类筛选"
// @Param assignee_id query int false "处理人ID筛选"
// @Success 200 {object} common.Response{data=common.PaginatedResponse{items=[]dto.IncidentResponse}}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents [get]
func (c *IncidentController) ListIncidents(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))

	// 构建筛选条件
	filters := make(map[string]interface{})
	if status := ctx.Query("status"); status != "" {
		filters["status"] = status
	}
	if priority := ctx.Query("priority"); priority != "" {
		filters["priority"] = priority
	}
	if severity := ctx.Query("severity"); severity != "" {
		filters["severity"] = severity
	}
	if category := ctx.Query("category"); category != "" {
		filters["category"] = category
	}
	if assigneeIDStr := ctx.Query("assignee_id"); assigneeIDStr != "" {
		if assigneeID, err := strconv.Atoi(assigneeIDStr); err == nil {
			filters["assignee_id"] = assigneeID
		}
	}

	tenantID, err := middleware.GetTenantID(ctx)
	incidents, total, err := c.incidentService.ListIncidents(ctx.Request.Context(), tenantID, page, size, filters)
	if err != nil {
		c.logger.Errorw("Failed to list incidents", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取事件列表失败")
		return
	}

	common.Success(ctx, gin.H{
		"items":    incidents,
		"total":    total,
		"page":     page,
		"pageSize": size,
	})
}

// UpdateIncident 更新事件
// @Summary 更新事件
// @Description 更新事件信息
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param id path int true "事件ID"
// @Param request body dto.UpdateIncidentRequest true "更新事件请求"
// @Success 200 {object} common.Response{data=dto.IncidentResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id} [put]
func (c *IncidentController) UpdateIncident(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	var req dto.UpdateIncidentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Invalid request body", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数无效")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	response, err := c.incidentService.UpdateIncident(ctx.Request.Context(), id, &req, tenantID)
	if err != nil {
		if err.Error() == "incident not found" {
			common.Fail(ctx, common.ParamErrorCode, "事件不存在")
			return
		}
		c.logger.Errorw("Failed to update incident", "error", err, "id", id)
		common.Fail(ctx, common.InternalErrorCode, "更新事件失败")
		return
	}

	common.SuccessWithMessage(ctx, "更新事件成功", response)
}

// DeleteIncident 删除事件
// @Summary 删除事件
// @Description 删除指定的事件
// @Tags 事件管理
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id} [delete]
func (c *IncidentController) DeleteIncident(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	err = c.incidentService.DeleteIncident(ctx.Request.Context(), id, tenantID)
	if err != nil {
		if err.Error() == "incident not found" {
			common.Fail(ctx, common.ParamErrorCode, "事件不存在")
			return
		}
		c.logger.Errorw("Failed to delete incident", "error", err, "id", id)
		common.Fail(ctx, common.InternalErrorCode, "删除事件失败")
		return
	}

	common.SuccessWithMessage(ctx, "删除事件成功", nil)
}

// EscalateIncident 升级事件
// @Summary 升级事件
// @Description 将事件升级到指定级别
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param request body dto.IncidentEscalationRequest true "事件升级请求"
// @Success 200 {object} common.Response{data=dto.IncidentEscalationResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/escalate [post]
func (c *IncidentController) EscalateIncident(ctx *gin.Context) {
	var req dto.IncidentEscalationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Invalid request body", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数无效")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	response, err := c.incidentService.EscalateIncident(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		if err.Error() == "incident not found" {
			common.Fail(ctx, common.ParamErrorCode, "事件不存在")
			return
		}
		c.logger.Errorw("Failed to escalate incident", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "升级事件失败")
		return
	}

	common.SuccessWithMessage(ctx, "升级事件成功", response)
}

// GetIncidentMonitoring 获取事件监控数据
// @Summary 获取事件监控数据
// @Description 获取事件监控统计和趋势数据
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param request body dto.IncidentMonitoringRequest true "监控请求"
// @Success 200 {object} common.Response{data=dto.IncidentMonitoringResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/monitoring [post]
func (c *IncidentController) GetIncidentMonitoring(ctx *gin.Context) {
	var req dto.IncidentMonitoringRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Invalid request body", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数无效")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	response, err := c.monitoringService.GenerateIncidentReport(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to get incident monitoring", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取监控数据失败")
		return
	}

	common.SuccessWithMessage(ctx, "获取监控数据成功", response)
}

// AnalyzeIncidentImpact 分析事件影响
// @Summary 分析事件影响
// @Description 分析事件的影响范围和业务影响
// @Tags 事件管理
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} common.Response{data=map[string]interface{}}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/impact [get]
func (c *IncidentController) AnalyzeIncidentImpact(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	analysis, err := c.monitoringService.AnalyzeIncidentImpact(ctx.Request.Context(), id, tenantID)
	if err != nil {
		if err.Error() == "incident not found" {
			common.Fail(ctx, common.ParamErrorCode, "事件不存在")
			return
		}
		c.logger.Errorw("Failed to analyze incident impact", "error", err, "id", id)
		common.Fail(ctx, common.InternalErrorCode, "分析事件影响失败")
		return
	}

	common.SuccessWithMessage(ctx, "分析事件影响成功", analysis)
}

// GetIncidentEvents 获取事件活动记录
// @Summary 获取事件活动记录
// @Description 获取事件的活动记录列表
// @Tags 事件管理
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} common.Response{data=[]dto.IncidentEventResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/events [get]
func (c *IncidentController) GetIncidentEvents(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		c.logger.Errorw("Failed to get tenant ID", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	// TODO: 实现GetIncidentEvents方法，使用id和tenantID
	_ = id
	_ = tenantID
	events := []interface{}{}

	common.SuccessWithMessage(ctx, "获取事件活动记录成功", events)
}

// GetIncidentAlerts 获取事件告警
// @Summary 获取事件告警
// @Description 获取事件的告警列表
// @Tags 事件管理
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} common.Response{data=[]dto.IncidentAlertResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/alerts [get]
func (c *IncidentController) GetIncidentAlerts(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		c.logger.Errorw("Failed to get tenant ID", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	// TODO: 实现GetIncidentAlerts方法，使用id和tenantID
	_ = id
	_ = tenantID
	alerts := []interface{}{}

	common.SuccessWithMessage(ctx, "获取事件告警成功", alerts)
}

// GetIncidentMetrics 获取事件指标
// @Summary 获取事件指标
// @Description 获取事件的指标数据
// @Tags 事件管理
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} common.Response{data=[]dto.IncidentMetricResponse}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/metrics [get]
func (c *IncidentController) GetIncidentMetrics(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	// TODO: 实现GetIncidentMetrics方法
	_ = id
	_ = tenantID
	metrics := []interface{}{}

	common.SuccessWithMessage(ctx, "获取事件指标成功", metrics)
}

// CreateIncidentEvent 创建事件活动记录
// @Summary 创建事件活动记录
// @Description 为事件创建活动记录
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param request body dto.CreateIncidentEventRequest true "创建事件活动记录请求"
// @Success 200 {object} common.Response{data=dto.IncidentEventResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/events [post]
func (c *IncidentController) CreateIncidentEvent(ctx *gin.Context) {
	var req dto.CreateIncidentEventRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Invalid request body", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数无效")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	response, err := c.incidentService.CreateIncidentEvent(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to create incident event", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "创建事件活动记录失败")
		return
	}

	common.SuccessWithMessage(ctx, "创建事件活动记录成功", response)
}

// CreateIncidentAlert 创建事件告警
// @Summary 创建事件告警
// @Description 为事件创建告警
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param request body dto.CreateIncidentAlertRequest true "创建事件告警请求"
// @Success 200 {object} common.Response{data=dto.IncidentAlertResponse}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/alerts [post]
func (c *IncidentController) CreateIncidentAlert(ctx *gin.Context) {
	var req dto.CreateIncidentAlertRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Invalid request body", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数无效")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	response, err := c.alertingService.CreateIncidentAlert(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to create incident alert", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "创建事件告警失败")
		return
	}

	common.SuccessWithMessage(ctx, "创建事件告警成功", response)
}

// AcknowledgeAlert 确认告警
// @Summary 确认告警
// @Description 确认指定告警
// @Tags 事件管理
// @Produce json
// @Param id path int true "告警ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/alerts/{id}/acknowledge [post]
func (c *IncidentController) AcknowledgeAlert(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的告警ID")
		return
	}

	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		c.logger.Errorw("Failed to get user ID", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取用户ID失败")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		c.logger.Errorw("Failed to get tenant ID", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
		return
	}

	err = c.alertingService.AcknowledgeAlert(ctx.Request.Context(), id, userID, tenantID)
	if err != nil {
		if err.Error() == "alert not found" {
			common.Fail(ctx, common.ParamErrorCode, "告警不存在")
			return
		}
		c.logger.Errorw("Failed to acknowledge alert", "error", err, "id", id)
		common.Fail(ctx, common.InternalErrorCode, "确认告警失败")
		return
	}

	common.SuccessWithMessage(ctx, "确认告警成功", nil)
}

// ResolveAlert 解决告警
// @Summary 解决告警
// @Description 解决指定告警
// @Tags 事件管理
// @Produce json
// @Param id path int true "告警ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/alerts/{id}/resolve [post]
func (c *IncidentController) ResolveAlert(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的告警ID")
		return
	}

	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		c.logger.Errorw("Failed to get user ID", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取用户ID失败")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	err = c.alertingService.ResolveAlert(ctx.Request.Context(), id, userID, tenantID)
	if err != nil {
		if err.Error() == "alert not found" {
			common.Fail(ctx, common.ParamErrorCode, "告警不存在")
			return
		}
		c.logger.Errorw("Failed to resolve alert", "error", err, "id", id)
		common.Fail(ctx, common.InternalErrorCode, "解决告警失败")
		return
	}

	common.SuccessWithMessage(ctx, "解决告警成功", nil)
}

// GetActiveAlerts 获取活跃告警
// @Summary 获取活跃告警
// @Description 获取当前活跃的告警列表
// @Tags 事件管理
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} common.Response{data=common.PaginatedResponse{items=[]dto.IncidentAlertResponse}}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/alerts/active [get]
func (c *IncidentController) GetActiveAlerts(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))

	tenantID, err := middleware.GetTenantID(ctx)
	alerts, total, err := c.alertingService.GetActiveAlerts(ctx.Request.Context(), tenantID, page, size)
	if err != nil {
		c.logger.Errorw("Failed to get active alerts", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取活跃告警失败")
		return
	}

	common.Success(ctx, gin.H{
		"items":    alerts,
		"total":    total,
		"page":     page,
		"pageSize": size,
	})
}

// GetAlertStatistics 获取告警统计
// @Summary 获取告警统计
// @Description 获取告警统计数据
// @Tags 事件管理
// @Produce json
// @Param start_time query string true "开始时间" example(2024-01-01T00:00:00Z)
// @Param end_time query string true "结束时间" example(2024-01-31T23:59:59Z)
// @Success 200 {object} common.Response{data=map[string]interface{}}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/alerts/statistics [get]
func (c *IncidentController) GetAlertStatistics(ctx *gin.Context) {
	startTimeStr := ctx.Query("start_time")
	endTimeStr := ctx.Query("end_time")

	if startTimeStr == "" || endTimeStr == "" {
		common.Fail(ctx, common.ParamErrorCode, "开始时间和结束时间不能为空")
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "开始时间格式无效")
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "结束时间格式无效")
		return
	}

	tenantID, err := middleware.GetTenantID(ctx)
	statistics, err := c.alertingService.GetAlertStatistics(ctx.Request.Context(), tenantID, startTime, endTime)
	if err != nil {
		c.logger.Errorw("Failed to get alert statistics", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取告警统计失败")
		return
	}

	common.SuccessWithMessage(ctx, "获取告警统计成功", statistics)
}
