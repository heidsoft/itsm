package incident

import (
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"

	"github.com/gin-gonic/gin"
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
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, common.AuthErrorCode, "Tenant ID missing")
		return
	}
	userID, exists := c.Get("user_id")
	if !exists {
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
		ImpactAnalysis: req.ImpactAnalysis, // Now map
		Source:         req.Source,
		Metadata:       req.Metadata,
		ReporterID:     userID.(int),
		IsAutomated:    false,
	}

	if req.AssigneeID != nil {
		incident.AssigneeID = req.AssigneeID
	}
	if req.ConfigurationItemID != nil {
		incident.ConfigurationItemID = req.ConfigurationItemID
	}
	if req.DetectedAt != nil {
		incident.DetectedAt = *req.DetectedAt
	}

	created, err := h.service.Create(c.Request.Context(), tenantID.(int), incident)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
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

	tenantID, _ := c.Get("tenant_id")
	incident, err := h.service.Get(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		if ent.IsNotFound(err) {
			common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		} else {
			common.Fail(c, common.InternalErrorCode, err.Error())
		}
		return
	}

	common.Success(c, h.toDTO(incident))
}

// List handles listing incidents
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	tenantID, _ := c.Get("tenant_id")

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
		userID, _ := c.Get("user_id")
		filters["assignee_id"] = userID.(int) // Service needs to support this filter
	}

	incidents, total, err := h.service.List(c.Request.Context(), tenantID.(int), page, size, filters)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
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
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")

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
		updates.ImpactAnalysis = req.ImpactAnalysis
	}
	if req.RootCause != nil {
		updates.RootCause = req.RootCause
	}
	if req.ResolutionSteps != nil {
		updates.ResolutionSteps = req.ResolutionSteps
	}

	updated, err := h.service.Update(c.Request.Context(), tenantID.(int), id, updates)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
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
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")

	updated, err := h.service.Escalate(c.Request.Context(), tenantID.(int), id, req.EscalationLevel, req.Reason)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, h.toDTO(updated))
}

func (h *Handler) toDTO(i *Incident) *dto.IncidentResponse {
	if i == nil {
		return nil
	}
	return &dto.IncidentResponse{
		ID:                  i.ID,
		Title:               i.Title,
		Description:         i.Description,
		Status:              i.Status,
		Priority:            i.Priority,
		Severity:            i.Severity,
		IncidentNumber:      i.IncidentNumber,
		ReporterID:          i.ReporterID,
		AssigneeID:          i.AssigneeID,
		ConfigurationItemID: i.ConfigurationItemID,
		Category:            i.Category,
		Subcategory:         i.Subcategory,
		ImpactAnalysis:      i.ImpactAnalysis,
		RootCause:           i.RootCause,
		ResolutionSteps:     i.ResolutionSteps,
		DetectedAt:          i.DetectedAt,
		ResolvedAt:          i.ResolvedAt,
		ClosedAt:            i.ClosedAt,
		EscalatedAt:         i.EscalatedAt,
		EscalationLevel:     i.EscalationLevel,
		IsAutomated:         i.IsAutomated,
		Source:              i.Source,
		Metadata:            i.Metadata,
		TenantID:            i.TenantID,
		CreatedAt:           i.CreatedAt,
		UpdatedAt:           i.UpdatedAt,
	}
}

// GetStats 获取事件统计数据（兼容前端）
func (h *Handler) GetStats(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDInt := tenantID.(int)

	// 直接从 ent 客户端查询统计数据
	client, exists := c.Get("client")
	if !exists {
		common.Fail(c, common.InternalErrorCode, "Database client not found")
		return
	}
	entClient := client.(*ent.Client)

	// 获取总数
	total, _ := entClient.Incident.Query().Where(incident.TenantID(tenantIDInt)).Count(c.Request.Context())

	// 获取各状态数量
	open, _ := entClient.Incident.Query().Where(incident.TenantID(tenantIDInt), incident.Status("open")).Count(c.Request.Context())
	inProgress, _ := entClient.Incident.Query().Where(incident.TenantID(tenantIDInt), incident.Status("in_progress")).Count(c.Request.Context())
	resolved, _ := entClient.Incident.Query().Where(incident.TenantID(tenantIDInt), incident.Status("resolved")).Count(c.Request.Context())
	closed, _ := entClient.Incident.Query().Where(incident.TenantID(tenantIDInt), incident.Status("closed")).Count(c.Request.Context())

	common.Success(c, gin.H{
		"total_incidents":    total,
		"open_incidents":     open + inProgress,
		"critical_incidents": 0,
		"major_incidents":    0,
		"resolved_incidents": resolved + closed,
		"avg_resolution_time": 0,
	})
}

// GetRootCause 获取根因分析
func (h *Handler) GetRootCause(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invalid id")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	incident, err := h.service.Get(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		return
	}

	common.Success(c, gin.H{
		"incident_id":   incident.ID,
		"root_cause":    incident.RootCause,
		"root_cause_analysis": incident.ImpactAnalysis,
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
		RootCause map[string]interface{} `json:"root_cause"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")
	updates := &Incident{}
	if req.RootCause != nil {
		updates.RootCause = req.RootCause
	}

	_, err = h.service.Update(c.Request.Context(), tenantID.(int), id, updates)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
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

	tenantID, _ := c.Get("tenant_id")
	incident, err := h.service.Get(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		return
	}

	common.Success(c, gin.H{
		"incident_id":         incident.ID,
		"impact_assessment":   incident.ImpactAnalysis,
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
		ImpactAnalysis map[string]interface{} `json:"impact_analysis"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")
	updates := &Incident{}
	if req.ImpactAnalysis != nil {
		updates.ImpactAnalysis = req.ImpactAnalysis
	}

	_, err = h.service.Update(c.Request.Context(), tenantID.(int), id, updates)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
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

	tenantID, _ := c.Get("tenant_id")
	incident, err := h.service.Get(c.Request.Context(), id, tenantID.(int))
	if err != nil {
		common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		return
	}

	common.Success(c, gin.H{
		"incident_id": incident.ID,
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
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")
	updates := &Incident{Category: req.Category, Subcategory: req.Subcategory}

	_, err = h.service.Update(c.Request.Context(), tenantID.(int), id, updates)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
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

	tenantID, _ := c.Get("tenant_id")
	client, exists := c.Get("client")
	if !exists {
		common.Fail(c, common.InternalErrorCode, "Database client not found")
		return
	}
	entClient := client.(*ent.Client)

	// 使用 Edge 查询事件活动记录
	inc, err := entClient.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID.(int))).
		WithIncidentEvents().
		Only(c.Request.Context())
	if err != nil {
		common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		return
	}

	var result []gin.H
	for _, e := range inc.Edges.IncidentEvents {
		result = append(result, gin.H{
			"id":           e.ID,
			"incident_id":  e.IncidentID,
			"event_type":   e.EventType,
			"event_name":   e.EventName,
			"description":  e.Description,
			"occurred_at":  e.OccurredAt,
			"created_at":   e.CreatedAt,
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

	tenantID, _ := c.Get("tenant_id")
	client, exists := c.Get("client")
	if !exists {
		common.Fail(c, common.InternalErrorCode, "Database client not found")
		return
	}
	entClient := client.(*ent.Client)

	// 使用 Edge 查询事件告警
	inc, err := entClient.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID.(int))).
		WithIncidentAlerts().
		Only(c.Request.Context())
	if err != nil {
		common.Fail(c, common.NotFoundErrorCode, "Incident not found")
		return
	}

	var result []gin.H
	for _, a := range inc.Edges.IncidentAlerts {
		result = append(result, gin.H{
			"id":          a.ID,
			"incident_id": a.IncidentID,
			"alert_name":  a.AlertName,
			"alert_type":  a.AlertType,
			"severity":    a.Severity,
			"status":      a.Status,
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

	tenantID, _ := c.Get("tenant_id")
	client, exists := c.Get("client")
	if !exists {
		common.Fail(c, common.InternalErrorCode, "Database client not found")
		return
	}
	entClient := client.(*ent.Client)

	// 获取事件信息及指标
	inc, err := entClient.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID.(int))).
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

	// 获取 SLA 违规数
	violations, _ := entClient.SLAViolation.Query().
		Count(c.Request.Context())

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
