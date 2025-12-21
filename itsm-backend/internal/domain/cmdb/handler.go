package cmdb

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

// toCIDTO maps domain CI to DTO
func toCIDTO(ci *ConfigurationItem) *dto.CIResponse {
	if ci == nil {
		return nil
	}
	return &dto.CIResponse{
		ID:          ci.ID,
		Name:        ci.Name,
		CITypeID:    ci.CITypeID,
		Description: ci.Description,
		Status:      ci.Status,
		TenantID:    ci.TenantID,
		CreatedAt:   ci.CreatedAt,
		UpdatedAt:   ci.UpdatedAt,
	}
}

// CreateCI handles POST /api/v1/cmdb/cis
func (h *Handler) CreateCI(c *gin.Context) {
	var req dto.CreateCIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	ci := &ConfigurationItem{
		Name:        req.Name,
		CITypeID:    req.CITypeID,
		Description: req.Description,
		Status:      req.Status,
		TenantID:    tenantID,
	}

	res, err := h.svc.CreateCI(c.Request.Context(), ci)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, toCIDTO(res))
}

// GetCI handles GET /api/v1/cmdb/cis/:id
func (h *Handler) GetCI(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	res, err := h.svc.GetCI(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, http.StatusNotFound, "Configuration Item not found")
		return
	}

	common.Success(c, toCIDTO(res))
}

// ListCIs handles GET /api/v1/cmdb/cis
func (h *Handler) ListCIs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status := c.Query("status")
	typeID, _ := strconv.Atoi(c.Query("ci_type_id"))
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	list, total, err := h.svc.ListCIs(c.Request.Context(), tenantID, page, pageSize, typeID, status)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	var dtos []*dto.CIResponse
	for _, item := range list {
		dtos = append(dtos, toCIDTO(item))
	}

	common.Success(c, gin.H{
		"items": dtos,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// UpdateCI handles PUT /api/v1/cmdb/cis/:id
func (h *Handler) UpdateCI(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	var req dto.UpdateCIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	existing, err := h.svc.GetCI(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, http.StatusNotFound, "Configuration Item not found")
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Status != "" {
		existing.Status = req.Status
	}

	res, err := h.svc.UpdateCI(c.Request.Context(), existing)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, toCIDTO(res))
}

// DeleteCI handles DELETE /api/v1/cmdb/cis/:id
func (h *Handler) DeleteCI(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	if err := h.svc.DeleteCI(c.Request.Context(), id, tenantID); err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, nil)
}

// GetStats handles GET /api/v1/cmdb/stats
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

// ListTypes handles GET /api/v1/cmdb/types
func (h *Handler) ListTypes(c *gin.Context) {
	tenantIDVal, _ := c.Get("tenant_id")
	tenantID := tenantIDVal.(int)

	list, err := h.svc.ListTypes(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(c, list)
}
