package sla

import (
	"net/http"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Map domain definition to DTO
func toSLADefinitionDTO(s *SLADefinition) *dto.SLADefinitionResponse {
	if s == nil {
		return nil
	}
	return &dto.SLADefinitionResponse{
		ID:              s.ID,
		Name:            s.Name,
		Description:     s.Description,
		ServiceType:     s.ServiceType,
		Priority:        s.Priority,
		ResponseTime:    s.ResponseTime,
		ResolutionTime:  s.ResolutionTime,
		BusinessHours:   s.BusinessHours,
		EscalationRules: s.EscalationRules,
		Conditions:      s.Conditions,
		IsActive:        s.IsActive,
		TenantID:        s.TenantID,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}

// CreateSLADefinition handles POST /api/v1/sla/definitions
func (h *Handler) CreateSLADefinition(c *gin.Context) {
	var req dto.CreateSLADefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	def := &SLADefinition{
		Name:            req.Name,
		Description:     req.Description,
		ServiceType:     req.ServiceType,
		Priority:        req.Priority,
		ResponseTime:    req.ResponseTime,
		ResolutionTime:  req.ResolutionTime,
		BusinessHours:   req.BusinessHours,
		EscalationRules: req.EscalationRules,
		Conditions:      req.Conditions,
		IsActive:        req.IsActive,
		TenantID:        tenantIDVal.(int),
	}

	res, err := h.svc.CreateDefinition(c.Request.Context(), def)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, toSLADefinitionDTO(res))
}

// GetSLADefinition handles GET /api/v1/sla/definitions/:id
func (h *Handler) GetSLADefinition(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")

	res, err := h.svc.GetDefinition(c.Request.Context(), id, tenantIDVal.(int))
	if err != nil {
		common.Fail(c, http.StatusNotFound, "SLA Definition not found")
		return
	}

	common.Success(c, toSLADefinitionDTO(res))
}

// ListSLADefinitions handles GET /api/v1/sla/definitions
func (h *Handler) ListSLADefinitions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	tenantIDVal, _ := c.Get("tenant_id")

	list, total, err := h.svc.ListDefinitions(c.Request.Context(), tenantIDVal.(int), page, size)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	var dtos []*dto.SLADefinitionResponse
	for _, item := range list {
		dtos = append(dtos, toSLADefinitionDTO(item))
	}

	common.Success(c, gin.H{
		"items": dtos,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// UpdateSLADefinition handles PUT /api/v1/sla/definitions/:id
func (h *Handler) UpdateSLADefinition(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")

	var req dto.UpdateSLADefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	existing, err := h.svc.GetDefinition(c.Request.Context(), id, tenantIDVal.(int))
	if err != nil {
		common.Fail(c, http.StatusNotFound, "SLA Definition not found")
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	// Map other fields as needed

	res, err := h.svc.UpdateDefinition(c.Request.Context(), existing)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, toSLADefinitionDTO(res))
}

// DeleteSLADefinition handles DELETE /api/v1/sla/definitions/:id
func (h *Handler) DeleteSLADefinition(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")

	if err := h.svc.DeleteDefinition(c.Request.Context(), id, tenantIDVal.(int)); err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, nil)
}

// CreateAlertRule handles POST /api/v1/sla/alert-rules
func (h *Handler) CreateAlertRule(c *gin.Context) {
	var req dto.CreateSLAAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")
	rule := &SLAAlertRule{
		SLADefinitionID:      req.SLADefinitionID,
		Name:                 req.Name,
		ThresholdPercentage:  req.ThresholdPercentage,
		AlertLevel:           req.AlertLevel,
		NotificationChannels: req.NotificationChannels,
		IsActive:             req.IsActive,
		TenantID:             tenantIDVal.(int),
	}
	res, err := h.svc.CreateAlertRule(c.Request.Context(), rule)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, res)
}

// ListAlertRules handles GET /api/v1/sla/alert-rules
func (h *Handler) ListAlertRules(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	filters := make(map[string]interface{})
	if slaID := c.Query("sla_definition_id"); slaID != "" {
		id, _ := strconv.Atoi(slaID)
		filters["sla_definition_id"] = id
	}
	res, err := h.svc.ListAlertRules(c.Request.Context(), tenantIDVal.(int), filters)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, res)
}

// GetAlertRule handles GET /api/v1/sla/alert-rules/:id
func (h *Handler) GetAlertRule(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")

	res, err := h.svc.GetAlertRule(c.Request.Context(), id, tenantIDVal.(int))
	if err != nil {
		common.Fail(c, http.StatusNotFound, "SLA Alert Rule not found")
		return
	}
	common.Success(c, res)
}

// UpdateAlertRule handles PUT /api/v1/sla/alert-rules/:id
func (h *Handler) UpdateAlertRule(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")

	var req dto.UpdateSLAAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	existing, err := h.svc.GetAlertRule(c.Request.Context(), id, tenantIDVal.(int))
	if err != nil {
		common.Fail(c, http.StatusNotFound, "SLA Alert Rule not found")
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.ThresholdPercentage != nil {
		existing.ThresholdPercentage = *req.ThresholdPercentage
	}
	if req.AlertLevel != nil {
		existing.AlertLevel = *req.AlertLevel
	}
	if req.NotificationChannels != nil {
		existing.NotificationChannels = *req.NotificationChannels
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	res, err := h.svc.UpdateAlertRule(c.Request.Context(), existing)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, res)
}

// DeleteAlertRule handles DELETE /api/v1/sla/alert-rules/:id
func (h *Handler) DeleteAlertRule(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")

	if err := h.svc.DeleteAlertRule(c.Request.Context(), id, tenantIDVal.(int)); err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, nil)
}

// GetSLAMetrics handles GET /api/v1/sla/metrics
func (h *Handler) GetSLAMetrics(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	filters := make(map[string]interface{})
	if slaID := c.Query("sla_definition_id"); slaID != "" {
		id, _ := strconv.Atoi(slaID)
		filters["sla_definition_id"] = id
	}
	if metricType := c.Query("metric_type"); metricType != "" {
		filters["metric_type"] = metricType
	}

	res, err := h.svc.GetSLAMetrics(c.Request.Context(), tenantIDVal.(int), filters)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, gin.H{
		"metrics": res,
		"count":   len(res),
	})
}

// GetSLAViolations handles GET /api/v1/sla/violations
func (h *Handler) GetSLAViolations(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	filters := make(map[string]interface{})
	if isResolved := c.Query("is_resolved"); isResolved != "" {
		if val, err := strconv.ParseBool(isResolved); err == nil {
			filters["is_resolved"] = val
		}
	}
	if severity := c.Query("severity"); severity != "" {
		filters["severity"] = severity
	}
	if violationType := c.Query("violation_type"); violationType != "" {
		filters["violation_type"] = violationType
	}
	if slaID := c.Query("sla_definition_id"); slaID != "" {
		if id, _ := strconv.Atoi(slaID); id > 0 {
			filters["sla_definition_id"] = id
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	res, total, err := h.svc.GetSLAViolations(c.Request.Context(), tenantIDVal.(int), page, size, filters)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, gin.H{
		"items": res,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// UpdateViolationStatus handles PUT /api/v1/sla/violations/:id
func (h *Handler) UpdateViolationStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")

	var req struct {
		IsResolved bool   `json:"is_resolved"`
		Notes      string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.svc.UpdateSLAViolationStatus(c.Request.Context(), id, req.IsResolved, req.Notes, tenantIDVal.(int))
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, res)
}

// GetSLAMonitoring handles POST /api/v1/sla/monitoring
func (h *Handler) GetSLAMonitoring(c *gin.Context) {
	var req dto.SLAMonitoringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	tenantIDVal, _ := c.Get("tenant_id")

	// Use request times or default to last 30 days
	startTime := req.StartTime
	endTime := req.EndTime
	if startTime == "" {
		startTime = "30d"
	}
	if endTime == "" {
		endTime = "now"
	}

	res, err := h.svc.GetSLAMonitoring(c.Request.Context(), tenantIDVal.(int), startTime, endTime)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, res)
}

// CheckSLACompliance handles POST /api/v1/sla/check-compliance/:ticketId
func (h *Handler) CheckSLACompliance(c *gin.Context) {
	ticketIDStr := c.Param("ticketId")
	ticketID, _ := strconv.Atoi(ticketIDStr)
	tenantIDVal, _ := c.Get("tenant_id")

	err := h.svc.CheckSLACompliance(c.Request.Context(), ticketID, tenantIDVal.(int))
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, gin.H{"message": "SLA合规性检查完成"})
}

// GetAlertHistory handles GET /api/v1/sla/alert-history
func (h *Handler) GetAlertHistory(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	filters := make(map[string]interface{})
	if slaID := c.Query("sla_definition_id"); slaID != "" {
		if id, _ := strconv.Atoi(slaID); id > 0 {
			filters["sla_definition_id"] = id
		}
	}
	if alertRuleID := c.Query("alert_rule_id"); alertRuleID != "" {
		if id, _ := strconv.Atoi(alertRuleID); id > 0 {
			filters["alert_rule_id"] = id
		}
	}
	if alertLevel := c.Query("alert_level"); alertLevel != "" {
		filters["alert_level"] = alertLevel
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	res, total, err := h.svc.GetAlertHistory(c.Request.Context(), tenantIDVal.(int), page, size, filters)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, gin.H{
		"items":     res,
		"total":     total,
		"page":      page,
		"page_size": size,
	})
}
