package change

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

// Map domain to DTO
func toDTO(c *Change) *dto.ChangeResponse {
	if c == nil {
		return nil
	}
	res := &dto.ChangeResponse{
		ID:                 c.ID,
		Title:              c.Title,
		Description:        c.Description,
		Justification:      c.Justification,
		Type:               dto.ChangeType(c.Type),
		Status:             dto.ChangeStatus(c.Status),
		Priority:           dto.ChangePriority(c.Priority),
		ImpactScope:        dto.ChangeImpact(c.ImpactScope),
		RiskLevel:          dto.ChangeRisk(c.RiskLevel),
		AssigneeID:         c.AssigneeID,
		CreatedBy:          c.CreatedBy,
		TenantID:           c.TenantID,
		PlannedStartDate:   c.PlannedStartDate,
		PlannedEndDate:     c.PlannedEndDate,
		ActualStartDate:    c.ActualStartDate,
		ActualEndDate:      c.ActualEndDate,
		ImplementationPlan: c.ImplementationPlan,
		RollbackPlan:       c.RollbackPlan,
		AffectedCIs:        c.AffectedCIs,
		RelatedTickets:     c.RelatedTickets,
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
	return res
}

// CreateChange handles POST /api/v1/changes
func (h *Handler) CreateChange(c *gin.Context) {
	var req dto.CreateChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	userIDVal, _ := c.Get("user_id")
	userID := userIDVal.(int)

	changeEntity := &Change{
		Title:              req.Title,
		Description:        req.Description,
		Justification:      req.Justification,
		Type:               string(req.Type),
		Status:             "draft",
		Priority:           string(req.Priority),
		ImpactScope:        string(req.ImpactScope),
		RiskLevel:          string(req.RiskLevel),
		CreatedBy:          userID,
		TenantID:           tenantID,
		PlannedStartDate:   req.PlannedStartDate,
		PlannedEndDate:     req.PlannedEndDate,
		ImplementationPlan: req.ImplementationPlan,
		RollbackPlan:       req.RollbackPlan,
		AffectedCIs:        req.AffectedCIs,
		RelatedTickets:     req.RelatedTickets,
	}

	res, err := h.svc.CreateChange(c.Request.Context(), changeEntity)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, toDTO(res))
}

// GetChange handles GET /api/v1/changes/:id
func (h *Handler) GetChange(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	res, err := h.svc.GetChange(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, http.StatusNotFound, "Change not found")
		return
	}

	common.Success(c, toDTO(res))
}

// ListChanges handles GET /api/v1/changes
func (h *Handler) ListChanges(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status := c.Query("status")
	search := c.Query("search")
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	list, total, err := h.svc.ListChanges(c.Request.Context(), tenantID, page, pageSize, status, search)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	var dtos []dto.ChangeResponse
	for _, item := range list {
		dtos = append(dtos, *toDTO(item))
	}

	common.Success(c, gin.H{
		"changes": dtos,
		"total":   total,
		"page":    page,
		"size":    pageSize,
	})
}

// UpdateChange handles PUT /api/v1/changes/:id
func (h *Handler) UpdateChange(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	var req dto.UpdateChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// First get existing
	existing, err := h.svc.GetChange(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, http.StatusNotFound, "Change not found")
		return
	}

	// Update fields if present in request
	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Type != nil {
		existing.Type = string(*req.Type)
	}
	if req.Priority != nil {
		existing.Priority = string(*req.Priority)
	}
	if req.ImpactScope != nil {
		existing.ImpactScope = string(*req.ImpactScope)
	}
	if req.RiskLevel != nil {
		existing.RiskLevel = string(*req.RiskLevel)
	}
	if req.PlannedStartDate != nil {
		existing.PlannedStartDate = req.PlannedStartDate
	}
	if req.PlannedEndDate != nil {
		existing.PlannedEndDate = req.PlannedEndDate
	}
	if req.ImplementationPlan != nil {
		existing.ImplementationPlan = *req.ImplementationPlan
	}
	if req.RollbackPlan != nil {
		existing.RollbackPlan = *req.RollbackPlan
	}
	if req.AffectedCIs != nil {
		existing.AffectedCIs = req.AffectedCIs
	}
	if req.RelatedTickets != nil {
		existing.RelatedTickets = req.RelatedTickets
	}

	res, err := h.svc.UpdateChange(c.Request.Context(), existing)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, toDTO(res))
}

// SubmitApproval handles POST /api/v1/changes/:id/approvals
func (h *Handler) SubmitApproval(c *gin.Context) {
	idStr := c.Param("id")
	changeID, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	var req dto.CreateChangeApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.ChangeID = changeID

	record := &ApprovalRecord{
		ChangeID:   req.ChangeID,
		ApproverID: req.ApproverID,
		Comment:    req.Comment,
	}

	res, err := h.svc.SubmitApproval(c.Request.Context(), record, tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, res)
}

// GetStats handles GET /api/v1/changes/stats
func (h *Handler) GetStats(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)
	res, err := h.svc.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, res)
}
