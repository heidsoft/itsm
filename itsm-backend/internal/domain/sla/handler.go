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
