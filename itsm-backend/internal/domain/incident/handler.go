package incident

import (
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"

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
