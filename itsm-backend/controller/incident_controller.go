package controller

import (
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/incidentevent"
	"itsm-backend/ent/user"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type IncidentController struct {
	incidentService          *service.IncidentService
	ruleEngine               *service.IncidentRuleEngine
	monitoringService        *service.IncidentMonitoringService
	alertingService          *service.IncidentAlertingService
	rootCauseAnalysisService *service.RootCauseAnalysisService
	logger                   *zap.SugaredLogger
}

func NewIncidentController(
	incidentService *service.IncidentService,
	ruleEngine *service.IncidentRuleEngine,
	monitoringService *service.IncidentMonitoringService,
	alertingService *service.IncidentAlertingService,
	rootCauseAnalysisService *service.RootCauseAnalysisService,
	logger *zap.SugaredLogger,
) *IncidentController {
	return &IncidentController{
		incidentService:          incidentService,
		ruleEngine:               ruleEngine,
		monitoringService:        monitoringService,
		alertingService:          alertingService,
		rootCauseAnalysisService: rootCauseAnalysisService,
		logger:                   logger,
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

func (c *IncidentController) resolveTenantID(ctx *gin.Context) (int, bool) {
	tenantID, err := middleware.ResolveRequestTenantID(ctx)
	if err == nil {
		return tenantID, true
	}
	c.logger.Errorw("Failed to resolve tenant ID", "error", err)
	if middleware.AbortIfTenantError(ctx, err) {
		return 0, false
	}
	common.Fail(ctx, common.InternalErrorCode, "获取租户ID失败")
	return 0, false
}

func (c *IncidentController) CreateIncident(ctx *gin.Context) {
	var req dto.CreateIncidentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Invalid request body", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数无效")
		return
	}

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		c.logger.Errorw("Failed to get user ID", "error", err)
		common.Fail(ctx, common.AuthFailedCode, "获取用户ID失败")
		return
	}

	response, err := c.incidentService.CreateIncident(ctx.Request.Context(), &req, tenantID, userID)
	if err != nil {
		c.logger.Errorw("Failed to create incident", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "创建事件失败")
		return
	}

	common.Success(ctx, response)
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

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}
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
// @Param page_size query int false "每页数量" default(10)
// @Param size query int false "每页数量" default(10)
// @Param status query string false "状态筛选"
// @Param priority query string false "优先级筛选"
// @Param severity query string false "严重程度筛选"
// @Param category query string false "分类筛选"
// @Param source query string false "来源筛选"
// @Param keyword query string false "关键词搜索（标题、描述、事件编号）"
// @Param assignee_id query int false "处理人ID筛选"
// @Success 200 {object} common.Response{data=object{incidents=[]dto.IncidentResponse,total=int,page=int,page_size=int,total_pages=int}}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents [get]
func (c *IncidentController) ListIncidents(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	// 支持 page_size 和 size 两种参数名
	pageSizeStr := ctx.Query("page_size")
	if pageSizeStr == "" {
		pageSizeStr = ctx.DefaultQuery("size", "10")
	}
	size, _ := strconv.Atoi(pageSizeStr)
	if size <= 0 {
		size = 10
	}

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
	if source := ctx.Query("source"); source != "" {
		filters["source"] = source
	}
	if keyword := ctx.Query("keyword"); keyword != "" {
		filters["keyword"] = keyword
	}
	if assigneeIDStr := ctx.Query("assignee_id"); assigneeIDStr != "" {
		if assigneeID, err := strconv.Atoi(assigneeIDStr); err == nil {
			filters["assignee_id"] = assigneeID
		}
	}

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	incidents, total, err := c.incidentService.ListIncidents(ctx.Request.Context(), tenantID, page, size, filters)
	if err != nil {
		c.logger.Errorw("Failed to list incidents", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取事件列表失败")
		return
	}

	// 计算总页数
	totalPages := (total + size - 1) / size
	if totalPages == 0 {
		totalPages = 1
	}

	// 返回前端期望的格式
	common.Success(ctx, dto.IncidentListResponse{
		Incidents:  incidents,
		Total:      total,
		Page:       page,
		PageSize:   size,
		TotalPages: totalPages,
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

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}
	response, err := c.incidentService.UpdateIncident(ctx.Request.Context(), id, &req, tenantID)
	if err != nil {
		// 处理版本冲突错误
		if common.IsVersionConflictError(err) {
			conflictErr := err.(*common.VersionConflictError)
			c.logger.Warnw("Version conflict", "error", err, "incident_id", id)
			common.Conflict(ctx, conflictErr.Error(), gin.H{
				"incidentId":     conflictErr.ResourceID,
				"currentVersion": conflictErr.CurrentVersion,
				"serverVersion":  conflictErr.ServerVersion,
			})
			return
		}
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

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}
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

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}
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

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}
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

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}
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

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	events, err := c.incidentService.GetIncidentEvents(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to get incident events", "error", err, "incident_id", id)
		common.Fail(ctx, common.InternalErrorCode, "获取事件活动记录失败")
		return
	}

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

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	alerts, err := c.incidentService.GetIncidentAlerts(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to get incident alerts", "error", err, "incident_id", id)
		common.Fail(ctx, common.InternalErrorCode, "获取事件告警失败")
		return
	}

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

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}
	metrics, err := c.incidentService.GetIncidentMetrics(ctx.Request.Context(), id, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to get incident metrics", "error", err, "incident_id", id)
		common.Fail(ctx, common.InternalErrorCode, "获取事件指标失败")
		return
	}

	common.SuccessWithMessage(ctx, "获取事件指标成功", metrics)
}

// GetIncidentStats 获取事件统计
// @Summary 获取事件统计
// @Description 获取事件统计数据
// @Tags 事件管理
// @Produce json
// @Success 200 {object} common.Response{data=dto.IncidentStatsResponse}
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/stats [get]
func (c *IncidentController) GetIncidentStats(ctx *gin.Context) {
	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	stats, err := c.incidentService.GetIncidentStats(ctx.Request.Context(), tenantID)
	if err != nil {
		c.logger.Errorw("Failed to get incident stats", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取事件统计失败")
		return
	}

	common.Success(ctx, stats)
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

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}
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

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}
	response, err := c.alertingService.CreateIncidentAlert(ctx.Request.Context(), &req, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to create incident alert", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "创建事件告警失败")
		return
	}

	common.SuccessWithMessage(ctx, "创建事件告警成功", response)
}

// AcknowledgeIncident 确认事件（业务流转）
// @Summary 确认事件
// @Description 将事件状态从 new 流转到 acknowledged
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} common.Response
// @Router /api/v1/incidents/:id/acknowledge [post]
func (c *IncidentController) AcknowledgeIncident(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}
	userID := ctx.GetInt("user_id")
	tenantID := ctx.GetInt("tenant_id")
	if err := c.incidentService.AcknowledgeIncident(ctx.Request.Context(), id, userID, tenantID); err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, gin.H{"message": "事件已确认"})
}

// ResolveIncident 解决事件
// @Summary 解决事件
// @Description 将事件状态流转到 resolved
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param id path int true "事件ID"
// @Param body body object true "解决信息"
// @Success 200 {object} common.Response
// @Router /api/v1/incidents/:id/resolve [post]
func (c *IncidentController) ResolveIncident(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}
	var body struct {
		Resolution string `json:"resolution"`
		RootCause  string `json:"rootCause"`
	}
	_ = ctx.ShouldBindJSON(&body)
	userID := ctx.GetInt("user_id")
	tenantID := ctx.GetInt("tenant_id")
	if err := c.incidentService.ResolveIncident(ctx.Request.Context(), id, userID, tenantID, body.Resolution, body.RootCause); err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, gin.H{"message": "事件已解决"})
}

// CloseIncident 关闭事件
// @Summary 关闭事件
// @Description 将已解决的事件关闭
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param id path int true "事件ID"
// @Param body body object true "关闭信息"
// @Success 200 {object} common.Response
// @Router /api/v1/incidents/:id/close [post]
func (c *IncidentController) CloseIncident(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}
	var body struct {
		CloseNotes string `json:"closeNotes"`
	}
	_ = ctx.ShouldBindJSON(&body)
	userID := ctx.GetInt("user_id")
	tenantID := ctx.GetInt("tenant_id")
	if err := c.incidentService.CloseIncident(ctx.Request.Context(), id, userID, tenantID, body.CloseNotes); err != nil {
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, gin.H{"message": "事件已关闭"})
}

// AssignIncident 分配事件
// @Summary 分配事件
// @Description 将事件分配给指定处理人
// @Tags 事件管理
// @Produce json
// @Param id path int true "事件ID"
// @Param request body dto.AssignIncidentRequest true "分配请求"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/assign [post]
func (c *IncidentController) AssignIncident(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	var req dto.AssignIncidentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}
	assigneeID := req.AssigneeID
	if assigneeID <= 0 {
		common.Fail(ctx, common.ParamErrorCode, "assigneeId 必填")
		return
	}

	tenantID := ctx.GetInt("tenant_id")
	incident, err := c.incidentService.AssignIncident(ctx.Request.Context(), id, assigneeID, tenantID)
	if err != nil {
		c.logger.Errorw("Failed to assign incident", "error", err, "id", id)
		common.Fail(ctx, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(ctx, incident)
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

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	err = c.alertingService.AcknowledgeAlert(ctx.Request.Context(), id, userID, tenantID)
	if err != nil {
		if err.Error() == "alert not found" {
			common.Fail(ctx, common.NotFoundErrorCode, "告警不存在")
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

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}
	err = c.alertingService.ResolveAlert(ctx.Request.Context(), id, userID, tenantID)
	if err != nil {
		if err.Error() == "alert not found" {
			common.Fail(ctx, common.NotFoundErrorCode, "告警不存在")
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
// @Success 200 {object} common.Response{data=object{items=[]dto.IncidentAlertResponse,total=int,page=int,pageSize=int}}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/alerts/active [get]
func (c *IncidentController) GetActiveAlerts(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}
	alerts, total, err := c.alertingService.GetActiveAlerts(ctx.Request.Context(), tenantID, page, size)
	if err != nil {
		c.logger.Errorw("Failed to get active alerts", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取活跃告警失败")
		return
	}

	common.Success(ctx, dto.IncidentAlertListResponse{
		Items:    alerts,
		Total:    total,
		Page:     page,
		PageSize: size,
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
	if endTime.Before(startTime) {
		common.Fail(ctx, common.ParamErrorCode, "结束时间不能早于开始时间")
		return
	}

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}
	statistics, err := c.alertingService.GetAlertStatistics(ctx.Request.Context(), tenantID, startTime, endTime)
	if err != nil {
		c.logger.Errorw("Failed to get alert statistics", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取告警统计失败")
		return
	}

	common.SuccessWithMessage(ctx, "获取告警统计成功", statistics)
}

// ConvertToProblem 将事件转换为问题
// @Summary 将事件转换为问题
// @Description 将指定的事件转换为问题记录
// @Tags incidents
// @Accept json
// @Produce json
// @Param id path int true "事件ID"
// @Param request body dto.ConvertIncidentToProblemRequest true "转换请求"
// @Success 200 {object} common.Response{data=ent.Problem}
// @Failure 400 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/convert-to-problem [post]
func (c *IncidentController) ConvertToProblem(ctx *gin.Context) {
	incidentID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	var req dto.ConvertIncidentToProblemRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取用户ID失败")
		return
	}

	tenantID := ctx.GetInt("tenant_id")
	problem, err := c.rootCauseAnalysisService.CreateProblemFromIncident(
		ctx.Request.Context(), incidentID, userID, tenantID, &req,
	)
	if err != nil {
		c.logger.Errorw("Failed to convert incident to problem", "error", err, "incident_id", incidentID)
		common.Fail(ctx, common.InternalErrorCode, "转换失败: "+err.Error())
		return
	}

	common.Success(ctx, dto.ToProblemResponse(problem))
}

// GetRootCause 获取根因分析
// @Summary 获取根因分析
// @Description 获取事件的根因分析
// @Tags 事件管理
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/root-cause [get]
func (c *IncidentController) GetRootCause(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	incident, err := c.incidentService.GetIncident(ctx.Request.Context(), id, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.Fail(ctx, common.NotFoundErrorCode, "事件不存在")
			return
		}
		c.logger.Errorw("Failed to get incident", "error", err, "id", id)
		common.Fail(ctx, common.InternalErrorCode, "获取事件失败")
		return
	}

	common.Success(ctx, gin.H{
		"incident_id": incident.ID,
		"root_cause":  incident.RootCause,
	})
}

// UpdateRootCause 更新根因分析
// @Summary 更新根因分析
// @Description 更新事件的根因分析
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param id path int true "事件ID"
// @Param request body dto.RootCause true "根因分析"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/root-cause [put]
func (c *IncidentController) UpdateRootCause(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	var req dto.RootCause
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Invalid request body", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数无效")
		return
	}

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	_, err = c.incidentService.UpdateIncident(ctx.Request.Context(), id, &dto.UpdateIncidentRequest{
		RootCause: &req,
	}, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.Fail(ctx, common.NotFoundErrorCode, "事件不存在")
			return
		}
		c.logger.Errorw("Failed to update root cause", "error", err, "id", id)
		common.Fail(ctx, common.InternalErrorCode, "更新根因分析失败")
		return
	}

	common.SuccessWithMessage(ctx, "根因分析已更新", nil)
}

// GetImpactAssessment 获取影响评估
// @Summary 获取影响评估
// @Description 获取事件的影响评估
// @Tags 事件管理
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/impact-assessment [get]
func (c *IncidentController) GetImpactAssessment(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	incident, err := c.incidentService.GetIncident(ctx.Request.Context(), id, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.Fail(ctx, common.NotFoundErrorCode, "事件不存在")
			return
		}
		c.logger.Errorw("Failed to get incident", "error", err, "id", id)
		common.Fail(ctx, common.InternalErrorCode, "获取事件失败")
		return
	}

	common.Success(ctx, gin.H{
		"incident_id":       incident.ID,
		"impact_assessment": incident.ImpactAnalysis,
	})
}

// UpdateImpactAssessment 更新影响评估
// @Summary 更新影响评估
// @Description 更新事件的影响评估
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param id path int true "事件ID"
// @Param request body dto.ImpactAnalysis true "影响评估"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/impact-assessment [put]
func (c *IncidentController) UpdateImpactAssessment(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	var req dto.ImpactAnalysis
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Invalid request body", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数无效")
		return
	}

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	_, err = c.incidentService.UpdateIncident(ctx.Request.Context(), id, &dto.UpdateIncidentRequest{
		ImpactAnalysis: &req,
	}, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.Fail(ctx, common.NotFoundErrorCode, "事件不存在")
			return
		}
		c.logger.Errorw("Failed to update impact assessment", "error", err, "id", id)
		common.Fail(ctx, common.InternalErrorCode, "更新影响评估失败")
		return
	}

	common.SuccessWithMessage(ctx, "影响评估已更新", nil)
}

// GetClassification 获取事件分类
// @Summary 获取事件分类
// @Description 获取事件的分类信息
// @Tags 事件管理
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/classification [get]
func (c *IncidentController) GetClassification(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	incident, err := c.incidentService.GetIncident(ctx.Request.Context(), id, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.Fail(ctx, common.NotFoundErrorCode, "事件不存在")
			return
		}
		c.logger.Errorw("Failed to get incident", "error", err, "id", id)
		common.Fail(ctx, common.InternalErrorCode, "获取事件失败")
		return
	}

	common.Success(ctx, gin.H{
		"incident_id": incident.ID,
		"category":    incident.Category,
		"subcategory": incident.Subcategory,
	})
}

// UpdateClassification 更新事件分类
// @Summary 更新事件分类
// @Description 更新事件的分类信息
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param id path int true "事件ID"
// @Param request body object{category=string,subcategory=string} true "分类信息"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/classification [put]
func (c *IncidentController) UpdateClassification(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	var req struct {
		Category    string `json:"category"`
		Subcategory string `json:"subcategory"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Invalid request body", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数无效")
		return
	}

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	cat := req.Category
	sub := req.Subcategory
	_, err = c.incidentService.UpdateIncident(ctx.Request.Context(), id, &dto.UpdateIncidentRequest{
		Category:    &cat,
		Subcategory: &sub,
	}, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			common.Fail(ctx, common.NotFoundErrorCode, "事件不存在")
			return
		}
		c.logger.Errorw("Failed to update classification", "error", err, "id", id)
		common.Fail(ctx, common.InternalErrorCode, "更新分类失败")
		return
	}

	common.SuccessWithMessage(ctx, "分类已更新", nil)
}

// GetIncidentComments 获取事件评论列表
// @Summary 获取事件评论列表
// @Description 获取事件的评论列表
// @Tags 事件管理
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} common.Response{data=[]gin.H}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/comments [get]
func (c *IncidentController) GetIncidentComments(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	client, exists := ctx.Get("client")
	if !exists {
		common.Fail(ctx, common.InternalErrorCode, "数据库客户端未找到")
		return
	}
	entClient := client.(*ent.Client)

	// 验证事件存在且属于该租户
	_, err = entClient.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID)).
		Only(ctx.Request.Context())
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(ctx, common.NotFoundErrorCode, "事件不存在")
		} else {
			c.logger.Errorw("Failed to query incident", "error", err, "id", id)
			common.Fail(ctx, common.InternalErrorCode, "查询事件失败")
		}
		return
	}

	// 查询评论（使用 IncidentEvent with event_type=comment）
	events, err := entClient.IncidentEvent.Query().
		Where(
			incidentevent.IncidentIDEQ(id),
			incidentevent.TenantIDEQ(tenantID),
			incidentevent.EventType("comment"),
		).
		All(ctx.Request.Context())
	if err != nil {
		c.logger.Errorw("Failed to query comments", "error", err, "incident_id", id)
		common.Fail(ctx, common.InternalErrorCode, "查询评论失败")
		return
	}

	userIDs := make([]int, 0, len(events))
	for _, e := range events {
		if e.UserID > 0 {
			userIDs = append(userIDs, e.UserID)
		}
	}

	usersByID := map[int]*ent.User{}
	if len(userIDs) > 0 {
		users, err := entClient.User.Query().
			Where(user.IDIn(userIDs...), user.TenantIDEQ(tenantID)).
			All(ctx.Request.Context())
		if err != nil {
			c.logger.Warnw("Failed to query comment users", "error", err, "incident_id", id)
		} else {
			for _, u := range users {
				usersByID[u.ID] = u
			}
		}
	}

	result := make([]*dto.IncidentCommentResponse, 0, len(events))
	for _, e := range events {
		result = append(result, dto.ToIncidentCommentResponse(e, usersByID[e.UserID]))
	}

	common.Success(ctx, result)
}

// CreateIncidentComment 创建事件评论
// @Summary 创建事件评论
// @Description 为事件创建评论
// @Tags 事件管理
// @Accept json
// @Produce json
// @Param id path int true "事件ID"
// @Param request body dto.CreateIncidentCommentRequest true "创建评论请求"
// @Success 200 {object} common.Response{data=gin.H}
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 500 {object} common.Response
// @Router /api/v1/incidents/{id}/comments [post]
func (c *IncidentController) CreateIncidentComment(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的事件ID")
		return
	}

	var req dto.CreateIncidentCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Errorw("Invalid request body", "error", err)
		common.Fail(ctx, common.ParamErrorCode, "请求参数无效")
		return
	}

	tenantID, ok := c.resolveTenantID(ctx)
	if !ok {
		return
	}

	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		c.logger.Errorw("Failed to get user ID", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "获取用户ID失败")
		return
	}

	client, exists := ctx.Get("client")
	if !exists {
		common.Fail(ctx, common.InternalErrorCode, "数据库客户端未找到")
		return
	}
	entClient := client.(*ent.Client)

	// 验证事件存在且属于该租户
	_, err = entClient.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID)).
		Only(ctx.Request.Context())
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(ctx, common.NotFoundErrorCode, "事件不存在")
		} else {
			c.logger.Errorw("Failed to query incident", "error", err, "id", id)
			common.Fail(ctx, common.InternalErrorCode, "查询事件失败")
		}
		return
	}

	// 创建评论（使用 IncidentEvent with event_type=comment）
	data := map[string]interface{}{
		"isInternal":  req.IsInternal,
		"mentions":    req.Mentions,
		"attachments": req.Attachments,
	}
	event, err := entClient.IncidentEvent.Create().
		SetIncidentID(id).
		SetEventType("comment").
		SetEventName("用户评论").
		SetDescription(req.Content).
		SetStatus("active").
		SetUserID(userID).
		SetSource("user").
		SetData(data).
		SetTenantID(tenantID).
		SetOccurredAt(time.Now()).
		Save(ctx.Request.Context())
	if err != nil {
		c.logger.Errorw("Failed to create comment", "error", err, "incident_id", id)
		common.Fail(ctx, common.InternalErrorCode, "创建评论失败")
		return
	}

	userEntity, err := entClient.User.Query().
		Where(user.IDEQ(userID), user.TenantIDEQ(tenantID)).
		Only(ctx.Request.Context())
	if err != nil {
		c.logger.Warnw("Failed to query comment user", "error", err, "user_id", userID)
	}

	common.Success(ctx, dto.ToIncidentCommentResponse(event, userEntity))
}
