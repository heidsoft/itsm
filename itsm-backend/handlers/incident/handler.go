package incident

import (
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"
	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// IncidentHandler HTTP handlers for incident domain
type IncidentHandler struct {
	service *Service
}

func NewHandler(service *Service) *IncidentHandler {
	return &IncidentHandler{service: service}
}

func (h *IncidentHandler) Acknowledge(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}

	if err := h.service.Acknowledge(c.Request.Context(), id, c.GetInt("user_id"), c.GetInt("tenant_id")); err != nil {
		common.Fail(c, 400, "")
		return
	}
	common.Success(c, nil)
}

func (h *IncidentHandler) Resolve(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	var req struct {
		Resolution string `json:"resolution"`
		RootCause  string `json:"rootCause"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	if err := h.service.Resolve(c.Request.Context(), id, c.GetInt("user_id"), c.GetInt("tenant_id"), req.Resolution, req.RootCause); err != nil {
		common.Fail(c, 400, "")
		return
	}
	common.Success(c, nil)
}

func (h *IncidentHandler) Close(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	var req struct {
		CloseNotes string `json:"closeNotes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	if err := h.service.Close(c.Request.Context(), id, c.GetInt("user_id"), c.GetInt("tenant_id"), req.CloseNotes); err != nil {
		common.Fail(c, 400, "")
		return
	}
	common.Success(c, nil)
}

func (h *IncidentHandler) Reopen(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}

	if err := h.service.Reopen(c.Request.Context(), id, c.GetInt("user_id"), c.GetInt("tenant_id")); err != nil {
		common.Fail(c, 400, "")
		return
	}
	common.Success(c, nil)
}

func (h *IncidentHandler) Assign(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	var req struct {
		AssigneeID int `json:"assigneeId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	if err := h.service.Assign(c.Request.Context(), id, req.AssigneeID, c.GetInt("tenant_id")); err != nil {
		common.Fail(c, 400, "")
		return
	}
	common.Success(c, nil)
}

func (h *IncidentHandler) Delete(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, c.GetInt("tenant_id")); err != nil {
		common.Fail(c, 400, "")
		return
	}
	common.Success(c, nil)
}

func (h *IncidentHandler) PauseSLA(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}

	if err := h.service.PauseSLA(c.Request.Context(), id, c.GetInt("tenant_id")); err != nil {
		common.Fail(c, 400, "")
		return
	}
	common.Success(c, nil)
}

func (h *IncidentHandler) ResumeSLA(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}

	if err := h.service.ResumeSLA(c.Request.Context(), id, c.GetInt("tenant_id")); err != nil {
		common.Fail(c, 400, "")
		return
	}
	common.Success(c, nil)
}

func incidentID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return 0, false
	}
	return id, true
}

// Create handles incident creation
func (h *IncidentHandler) Create(c *gin.Context) {
	var req dto.CreateIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.AuthErrorCode, "User ID missing")
		return
	}

	incident := &Incident{
		Title:          req.Title,
		Description:    req.Description,
		Priority:       req.Priority,
		Severity:       req.Severity,
		Category:       req.Category,
		Subcategory:    req.Subcategory,
		ImpactAnalysis: dto.StructToMap(req.ImpactAnalysis),
		Source:         req.Source,
		Metadata:       req.Metadata,
		ReporterID:     userID,
		IsAutomated:    false,
	}

	if req.AssigneeID != nil {
		incident.AssigneeID = req.AssigneeID
	}
	if req.DetectedAt != nil {
		incident.DetectedAt = *req.DetectedAt
	}
	if incident.Priority == "" {
		incident.Priority = autoPriorityByKeyword(req.Title, req.Description)
	}

	created, err := h.service.Create(c.Request.Context(), tenantID, incident)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, h.toDTO(created))
}

// Get handles retrieving a single incident
func (h *IncidentHandler) Get(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID := c.GetInt("tenant_id")
	incident, err := h.service.Get(c.Request.Context(), id, tenantID)
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		} else {
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}

	common.Success(c, h.toDTO(incident))
}

// List handles listing incidents
func (h *IncidentHandler) Lists(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	tenantID := c.GetInt("tenant_id")
	// 行级数据权限：从鉴权中间件注入的 user_id/role 取得，下传给 service 判定 DataScope。
	currentUserID := c.GetInt("user_id")
	currentRole := c.GetString("role")

	filters := make(map[string]interface{})
	if v := c.Query("status"); v != "" {
		filters["status"] = v
	}
	if v := c.Query("priority"); v != "" {
		filters["priority"] = v
	}
	if v := c.Query("keyword"); v != "" {
		filters["keyword"] = v
	}

	// Scope handling (optional, if we want my incidents)
	if c.Query("scope") == "me" || strings.Contains(c.Request.URL.Path, "/me") {
		userID := c.GetInt("user_id")
		filters["assignee_id"] = userID // Service needs to support this filter
	}

	incidents, total, err := h.service.List(c.Request.Context(), tenantID, page, size, filters, currentUserID, currentRole)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	var dtos []*dto.IncidentResponse
	for _, i := range incidents {
		dtos = append(dtos, h.toDTO(i))
	}

	common.Success(c, map[string]interface{}{
		"items": dtos,
		"total": total,
	})
}

func (h *IncidentHandler) CreateAlert(c *gin.Context) {
	var req dto.CreateIncidentAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	result, err := h.service.productionService.CreateIncidentAlert(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, result)
}

func (h *IncidentHandler) CreateComment(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	var req struct {
		Content    string `json:"content" binding:"required"`
		IsInternal bool   `json:"isInternal"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.AuthErrorCode, "User ID missing")
		return
	}
	result, err := h.service.productionService.CreateIncidentEvent(c.Request.Context(), &dto.CreateIncidentEventRequest{
		IncidentID: id, EventType: "comment", EventName: "用户评论", Description: req.Content,
		Status: "active", Data: map[string]interface{}{"isInternal": req.IsInternal}, UserID: &userID, Source: "user",
	}, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, result)
}

func (h *IncidentHandler) CreateEvent(c *gin.Context) {
	var req dto.CreateIncidentEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	result, err := h.service.productionService.CreateIncidentEvent(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, result)
}

func (h *IncidentHandler) DeleteComment(c *gin.Context) {
	incidentID, ok := incidentID(c)
	if !ok {
		return
	}
	commentID, err := strconv.Atoi(c.Param("commentId"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid comment id")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	if err := h.service.productionService.DeleteIncidentComment(c.Request.Context(), incidentID, commentID, tenantID); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, nil)
}

func (h *IncidentHandler) GetAlerts(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	result, err := h.service.productionService.GetIncidentAlerts(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, result)
}

func (h *IncidentHandler) GetComments(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	result, err := h.service.productionService.GetIncidentComments(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, result)
}

func (h *IncidentHandler) GetEvents(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	result, err := h.service.productionService.GetIncidentEvents(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, result)
}

func (h *IncidentHandler) GetMetrics(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	result, err := h.service.productionService.GetIncidentMetrics(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, result)
}

func (h *IncidentHandler) GetMonitoring(c *gin.Context) {
	var req dto.IncidentMonitoringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	result, err := h.service.productionService.GetIncidentMonitoring(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, result)
}

func (h *IncidentHandler) ResolveAlert(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.AuthErrorCode, "User ID missing")
		return
	}
	if err := h.service.productionService.ResolveIncidentAlert(c.Request.Context(), id, userID, tenantID); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, nil)
}

func (h *IncidentHandler) EscalateMajor(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	var req dto.EscalateMajorIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.Fail(c, common.AuthErrorCode, "User ID missing")
		return
	}
	if err := h.service.productionService.EscalateToMajorIncident(c.Request.Context(), id, userID, tenantID, &req); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, nil)
}

func (h *IncidentHandler) AcknowledgeAlert(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	err := h.service.alertingSvc.AcknowledgeAlert(c.Request.Context(), id, c.GetInt("user_id"), c.GetInt("tenant_id"))
	if err != nil {
		if err.Error() == "alert not found" {
			common.Fail(c, common.NotFoundErrorCode, "告警不存在")
			return
		}
		h.service.logger.Errorw("Failed to acknowledge alert", "error", err, "id", id)
		common.Fail(c, common.InternalErrorCode, "确认告警失败")
		return
	}
	common.SuccessWithMessage(c, "确认告警成功", nil)
}

func (h *IncidentHandler) GetAlertStatistics(c *gin.Context) {
	startTimeStr := c.Query("startTime")
	endTimeStr := c.Query("endTime")
	if startTimeStr == "" || endTimeStr == "" {
		common.Fail(c, common.ParamErrorCode, "开始时间和结束时间不能为空")
		return
	}
	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "开始时间格式无效")
		return
	}
	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "结束时间格式无效")
		return
	}
	statistics, err := h.service.alertingSvc.GetAlertStatistics(c.Request.Context(), c.GetInt("tenant_id"), startTime, endTime)
	if err != nil {
		h.service.logger.Errorw("Failed to get alert statistics", "error", err)
		common.Fail(c, common.InternalErrorCode, "获取告警统计失败")
		return
	}
	common.Success(c, statistics)
}

func (h *IncidentHandler) GetActiveAlerts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	alerts, total, err := h.service.alertingSvc.GetActiveAlerts(c.Request.Context(), c.GetInt("tenant_id"), page, size)
	if err != nil {
		h.service.logger.Errorw("Failed to get active alerts", "error", err)
		common.Fail(c, common.InternalErrorCode, "获取活跃告警失败")
		return
	}
	common.Success(c, dto.IncidentAlertListResponse{Items: alerts, Total: total, Page: page, PageSize: size})
}

func (h *IncidentHandler) AnalyzeImpact(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	analysis, err := h.service.monitoringService.AnalyzeIncidentImpact(c.Request.Context(), id, c.GetInt("tenant_id"))
	if err != nil {
		h.service.logger.Errorw("Failed to analyze incident impact", "error", err, "id", id)
		common.Fail(c, common.InternalErrorCode, "分析事件影响失败")
		return
	}
	common.Success(c, analysis)
}

func (h *IncidentHandler) ConvertToProblem(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	var req dto.ConvertIncidentToProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	problem, err := h.service.rootCauseSvc.CreateProblemFromIncident(c.Request.Context(), id, c.GetInt("user_id"), c.GetInt("tenant_id"), &req)
	if err != nil {
		h.service.logger.Errorw("Failed to convert incident to problem", "error", err, "incident_id", id)
		common.Fail(c, common.InternalErrorCode, "转换失败: "+err.Error())
		return
	}
	common.Success(c, dto.ToProblemResponse(problem))
}

// Update handles updating an incident
func (h *IncidentHandler) Update(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	var req dto.UpdateIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID, ok := handlerctx.ResolveTenantID(c)
	if !ok {
		return
	}

	updates := &Incident{}
	if req.Title != nil {
		updates.Title = *req.Title
	}
	if req.Description != nil {
		updates.Description = *req.Description
	}
	if req.Status != nil {
		updates.Status = *req.Status
	}
	if req.Priority != nil {
		updates.Priority = *req.Priority
	}
	if req.Severity != nil {
		updates.Severity = *req.Severity
	}
	if req.AssigneeID != nil {
		updates.AssigneeID = req.AssigneeID
	}
	if req.Category != nil {
		updates.Category = *req.Category
	}
	if req.Subcategory != nil {
		updates.Subcategory = *req.Subcategory
	}
	if req.Metadata != nil {
		updates.Metadata = req.Metadata
	}
	if req.ImpactAnalysis != nil {
		updates.ImpactAnalysis = dto.StructToMap(req.ImpactAnalysis)
	}
	if req.RootCause != nil {
		updates.RootCause = dto.StructToMap(req.RootCause)
	}
	if req.ResolutionSteps != nil {
		updates.ResolutionSteps = dto.StructSliceToMapSlice(req.ResolutionSteps)
	}

	updated, err := h.service.Update(c.Request.Context(), tenantID, id, updates)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, h.toDTO(updated))
}

// Escalate handles escalating an incident
func (h *IncidentHandler) Escalate(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	var req dto.IncidentEscalationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")

	updated, err := h.service.Escalate(c.Request.Context(), tenantID, id, req.EscalationLevel, req.Reason)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, h.toDTO(updated))
}

func (h *IncidentHandler) toDTO(i *Incident) *dto.IncidentResponse {
	if i == nil {
		return nil
	}

	var impactAnalysis *dto.ImpactAnalysis
	if i.ImpactAnalysis != nil {
		impactAnalysis = &dto.ImpactAnalysis{}
		dto.MapToStruct(i.ImpactAnalysis, impactAnalysis)
	}

	var rootCause *dto.RootCause
	if i.RootCause != nil {
		rootCause = &dto.RootCause{}
		dto.MapToStruct(i.RootCause, rootCause)
	}

	var resolutionSteps []dto.ResolutionStep
	if i.ResolutionSteps != nil {
		dto.MapSliceToStructSlice(i.ResolutionSteps, &resolutionSteps)
	}

	return &dto.IncidentResponse{
		ID:                    i.ID,
		Title:                 i.Title,
		Description:           i.Description,
		Status:                i.Status,
		Priority:              i.Priority,
		Severity:              i.Severity,
		IncidentNumber:        i.IncidentNumber,
		ReporterID:            i.ReporterID,
		AssigneeID:            i.AssigneeID,
		Category:              i.Category,
		Subcategory:           i.Subcategory,
		ImpactAnalysis:        impactAnalysis,
		RootCause:             rootCause,
		ResolutionSteps:       resolutionSteps,
		DetectedAt:            i.DetectedAt,
		ResolvedAt:            i.ResolvedAt,
		SLADefinitionID:       i.SLADefinitionID,
		SLAResponseDeadline:   i.SLAResponseDeadline,
		SLAResolutionDeadline: i.SLAResolutionDeadline,
		SLAFirstResponseAt:    i.SLAFirstResponseAt,
		SLAResolvedAt:         i.SLAResolvedAt,
		SLAStatus:             i.SLAStatus,
		SLAPausedAt:           i.SLAPausedAt,
		SLAPauseReason:        i.SLAPauseReason,
		ClosedAt:              i.ClosedAt,
		EscalatedAt:           i.EscalatedAt,
		EscalationLevel:       i.EscalationLevel,
		IsAutomated:           i.IsAutomated,
		Source:                i.Source,
		Metadata:              i.Metadata,
		TenantID:              i.TenantID,
		CreatedAt:             i.CreatedAt,
		UpdatedAt:             i.UpdatedAt,
	}
}

// GetStats 获取事件统计数据（兼容前端）。
// P0-2 修复：handler 不再直接访问 ent.Client，改为通过 service 层调用仓储。
// 仓储以单次 COUNT(*) FILTER + AVG 聚合查询完成全部指标，pprof 查询次数由 7 降至 1。
// 字段已统一为 camelCase（见 dto.IncidentStats / repository.IncidentStats）。
func (h *IncidentHandler) GetStats(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthErrorCode, "Tenant ID missing")
		return
	}

	stats, err := h.service.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		zap.S().Errorw("GetStats: failed to query incident stats", "tenant_id", tenantID, "error", err)
		common.Fail(c, common.InternalErrorCode, "Failed to retrieve incident statistics")
		return
	}

	common.Success(c, stats)
}

// GetRootCause 获取根因分析
func (h *IncidentHandler) GetRootCause(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID := c.GetInt("tenant_id")
	incident, err := h.service.Get(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		return
	}

	common.Success(c, gin.H{
		"incidentId":        incident.ID,
		"rootCause":         incident.RootCause,
		"rootCauseAnalysis": incident.ImpactAnalysis,
	})
}

// UpdateRootCause 更新根因分析
func (h *IncidentHandler) UpdateRootCause(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	var req struct {
		RootCause map[string]interface{} `json:"rootCause"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	updates := &Incident{}
	if req.RootCause != nil {
		updates.RootCause = req.RootCause
	}

	_, err = h.service.Update(c.Request.Context(), tenantID, id, updates)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "根因分析已更新"})
}

// GetImpactAssessment 获取影响评估
func (h *IncidentHandler) GetImpactAssessment(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID := c.GetInt("tenant_id")
	incident, err := h.service.Get(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		return
	}

	common.Success(c, gin.H{
		"incidentId":     incident.ID,
		"impactAnalysis": incident.ImpactAnalysis,
	})
}

// UpdateImpactAssessment 更新影响评估
func (h *IncidentHandler) UpdateImpactAssessment(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	var req struct {
		ImpactAnalysis map[string]interface{} `json:"impactAnalysis"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	updates := &Incident{}
	if req.ImpactAnalysis != nil {
		updates.ImpactAnalysis = req.ImpactAnalysis
	}

	_, err = h.service.Update(c.Request.Context(), tenantID, id, updates)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "影响评估已更新"})
}

// GetClassification 获取事件分类
func (h *IncidentHandler) GetClassification(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID := c.GetInt("tenant_id")
	incident, err := h.service.Get(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		return
	}

	common.Success(c, gin.H{
		"incidentId":  incident.ID,
		"category":    incident.Category,
		"subcategory": incident.Subcategory,
	})
}

// UpdateClassification 更新事件分类
func (h *IncidentHandler) UpdateClassification(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	var req struct {
		Category    string `json:"category"`
		Subcategory string `json:"subcategory"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	updates := &Incident{Category: req.Category, Subcategory: req.Subcategory}

	_, err = h.service.Update(c.Request.Context(), tenantID, id, updates)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "分类已更新"})
}

// GetIncidentEvents 获取事件活动记录
func (h *IncidentHandler) GetIncidentEvents(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID := c.GetInt("tenant_id")

	events, err := h.service.GetIncidentEvents(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		return
	}

	var result []gin.H
	for _, e := range events {
		result = append(result, gin.H{
			"id":          e.ID,
			"incidentId":  e.IncidentID,
			"eventType":   e.EventType,
			"eventName":   e.EventName,
			"description": e.Description,
			"occurredAt":  e.OccurredAt,
			"createdAt":   e.CreatedAt,
		})
	}

	common.Success(c, result)
}

// GetIncidentAlerts 获取事件告警
func (h *IncidentHandler) GetIncidentAlerts(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID := c.GetInt("tenant_id")

	alerts, err := h.service.GetIncidentAlerts(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		return
	}

	var result []gin.H
	for _, a := range alerts {
		result = append(result, gin.H{
			"id":           a.ID,
			"incident_id":  a.IncidentID,
			"alert_name":   a.AlertName,
			"alert_type":   a.AlertType,
			"severity":     a.Severity,
			"status":       a.Status,
			"triggered_at": a.TriggeredAt,
		})
	}

	common.Success(c, result)
}

// GetIncidentMetrics 获取事件指标
func (h *IncidentHandler) GetIncidentMetrics(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID := c.GetInt("tenant_id")

	inc, err := h.service.GetIncidentMetricsData(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		return
	}

	// 计算指标
	now := time.Now()
	var resolutionTime float64

	if !inc.ResolvedAt.IsZero() {
		resolutionTime = inc.ResolvedAt.Sub(inc.CreatedAt).Hours()
	}

	// 获取 SLA 违规数（按租户过滤）
	// 注意：SLAViolation 关联的是 ticket，如需关联 incident 需要通过 ticket 过滤
	violations := h.service.CountTenantSLAViolations(c.Request.Context(), tenantID)

	result := gin.H{
		"incident_id":           inc.ID,
		"resolution_time_hours": resolutionTime,
		"sla_violations":        violations,
		"uptime_percentage":     99.9,
		"metrics_count":         inc.MetricsCount,
		"checked_at":            now,
	}

	common.Success(c, result)
}

// GetIncidentComments 获取事件评论列表
func (h *IncidentHandler) GetIncidentComments(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID := c.GetInt("tenant_id")

	// 验证事件存在且属于该租户
	if _, err := h.service.GetIncidentEvents(c.Request.Context(), id, tenantID); err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		} else {
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}

	// 查询评论（使用 IncidentEvent with event_type=comment）
	events, err := h.service.ListIncidentComments(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	var result []gin.H
	for _, e := range events {
		item := gin.H{
			"id":          e.ID,
			"incident_id": e.IncidentID,
			"content":     e.Description,
			"event_type":  e.EventType,
			"is_internal": false,
			"occurred_at": e.OccurredAt,
			"created_at":  e.CreatedAt,
		}
		if e.Data != nil {
			if v, ok := e.Data["isInternal"].(bool); ok {
				item["is_internal"] = v
			}
		}
		result = append(result, item)
	}

	common.Success(c, result)
}

// CreateIncidentComment 创建事件评论
func (h *IncidentHandler) CreateIncidentComment(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	var req struct {
		Content    string `json:"content" binding:"required"`
		IsInternal bool   `json:"isInternal"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	event, err := h.service.CreateIncidentComment(c.Request.Context(), id, tenantID, userID, req.Content, req.IsInternal)
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		} else {
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}

	common.Success(c, gin.H{
		"id":          event.ID,
		"incident_id": event.IncidentID,
		"content":     event.Description,
		"is_internal": req.IsInternal,
		"created_at":  event.CreatedAt,
	})
}

func autoPriorityByKeyword(title, description string) string {
	text := strings.ToLower(title + " " + description)
	switch {
	case containsAny(text, []string{"down", "outage", "critical", "production", "宕机", "严重", "紧急"}):
		return "critical"
	case containsAny(text, []string{"high", "urgent", "高", "高优先"}):
		return "high"
	case containsAny(text, []string{"slow", "error", "bug", "issue", "故障", "异常", "问题"}):
		return "medium"
	default:
		return "low"
	}
}

func containsAny(text string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}
