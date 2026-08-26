package incident

import (
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/incidentevent"
	"itsm-backend/ent/slaviolation"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create handles incident creation
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthErrorCode, "Tenant ID missing")
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
func (h *Handler) Get(c *gin.Context) {
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
func (h *Handler) List(c *gin.Context) {
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

// Update handles updating an incident
func (h *Handler) Update(c *gin.Context) {
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

	tenantID := c.GetInt("tenant_id")

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
func (h *Handler) Escalate(c *gin.Context) {
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

func (h *Handler) toDTO(i *Incident) *dto.IncidentResponse {
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
		ID:              i.ID,
		Title:           i.Title,
		Description:     i.Description,
		Status:          i.Status,
		Priority:        i.Priority,
		Severity:        i.Severity,
		IncidentNumber:  i.IncidentNumber,
		ReporterID:      i.ReporterID,
		AssigneeID:      i.AssigneeID,
		Category:        i.Category,
		Subcategory:     i.Subcategory,
		ImpactAnalysis:  impactAnalysis,
		RootCause:       rootCause,
		ResolutionSteps: resolutionSteps,
		DetectedAt:      i.DetectedAt,
		ResolvedAt:      i.ResolvedAt,
		ClosedAt:        i.ClosedAt,
		EscalatedAt:     i.EscalatedAt,
		EscalationLevel: i.EscalationLevel,
		IsAutomated:     i.IsAutomated,
		Source:          i.Source,
		Metadata:        i.Metadata,
		TenantID:        i.TenantID,
		CreatedAt:       i.CreatedAt,
		UpdatedAt:       i.UpdatedAt,
	}
}

// GetStats 获取事件统计数据（兼容前端）。
// P0-2 修复：handler 不再直接访问 ent.Client，改为通过 service 层调用仓储。
// 仓储以单次 COUNT(*) FILTER + AVG 聚合查询完成全部指标，pprof 查询次数由 7 降至 1。
// 字段已统一为 camelCase（见 dto.IncidentStats / repository.IncidentStats）。
func (h *Handler) GetStats(c *gin.Context) {
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
func (h *Handler) GetRootCause(c *gin.Context) {
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
		"incidentId":         incident.ID,
		"rootCause":          incident.RootCause,
		"rootCauseAnalysis":  incident.ImpactAnalysis,
	})
}

// UpdateRootCause 更新根因分析
func (h *Handler) UpdateRootCause(c *gin.Context) {
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
func (h *Handler) GetImpactAssessment(c *gin.Context) {
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
func (h *Handler) UpdateImpactAssessment(c *gin.Context) {
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
func (h *Handler) GetClassification(c *gin.Context) {
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
func (h *Handler) UpdateClassification(c *gin.Context) {
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
func (h *Handler) GetIncidentEvents(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID := c.GetInt("tenant_id")
	client, exists := c.Get("client")
	if !exists {
		common.Fail(c, common.InternalErrorCode, "Database client not found")
		return
	}
	entClient := client.(*ent.Client)

	// 使用 Edge 查询事件活动记录
	inc, err := entClient.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID)).
		WithIncidentEvents().
		Only(c.Request.Context())
	if err != nil {
		common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		return
	}

	var result []gin.H
	for _, e := range inc.Edges.IncidentEvents {
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
func (h *Handler) GetIncidentAlerts(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID := c.GetInt("tenant_id")
	client, exists := c.Get("client")
	if !exists {
		common.Fail(c, common.InternalErrorCode, "Database client not found")
		return
	}
	entClient := client.(*ent.Client)

	// 使用 Edge 查询事件告警
	inc, err := entClient.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID)).
		WithIncidentAlerts().
		Only(c.Request.Context())
	if err != nil {
		common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		return
	}

	var result []gin.H
	for _, a := range inc.Edges.IncidentAlerts {
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
func (h *Handler) GetIncidentMetrics(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID := c.GetInt("tenant_id")
	client, exists := c.Get("client")
	if !exists {
		common.Fail(c, common.InternalErrorCode, "Database client not found")
		return
	}
	entClient := client.(*ent.Client)

	// 获取事件信息及指标
	inc, err := entClient.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID)).
		WithIncidentMetrics().
		Only(c.Request.Context())
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
	violations, err := entClient.SLAViolation.Query().
		Where(slaviolation.TenantIDEQ(tenantID)).
		Count(c.Request.Context())
	if err != nil {
		zap.S().Errorw("GetIncidentMetrics: failed to count SLA violations", "error", err)
		violations = 0
	}

	result := gin.H{
		"incident_id":           inc.ID,
		"resolution_time_hours": resolutionTime,
		"sla_violations":        violations,
		"uptime_percentage":     99.9,
		"metrics_count":         len(inc.Edges.IncidentMetrics),
		"checked_at":            now,
	}

	common.Success(c, result)
}

// GetIncidentComments 获取事件评论列表
func (h *Handler) GetIncidentComments(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID := c.GetInt("tenant_id")
	client, exists := c.Get("client")
	if !exists {
		common.Fail(c, common.InternalErrorCode, "Database client not found")
		return
	}
	entClient := client.(*ent.Client)

	// 验证事件存在且属于该租户
	_, err = entClient.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID)).
		Only(c.Request.Context())
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		} else {
			common.FailWithErr(c, err, "操作失败")
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
		WithIncident().
		All(c.Request.Context())
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
func (h *Handler) CreateIncidentComment(c *gin.Context) {
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
	client, exists := c.Get("client")
	if !exists {
		common.Fail(c, common.InternalErrorCode, "Database client not found")
		return
	}
	entClient := client.(*ent.Client)

	// 验证事件存在且属于该租户
	_, err = entClient.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID)).
		Only(c.Request.Context())
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		} else {
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}

	// 创建评论（使用 IncidentEvent with event_type=comment）
	data := map[string]interface{}{"isInternal": req.IsInternal}
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
		Save(c.Request.Context())
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
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
