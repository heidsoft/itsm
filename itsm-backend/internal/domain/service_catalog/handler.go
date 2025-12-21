package service_catalog

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for Service Catalog
type Handler struct {
	service *Service
}

// NewHandler creates a new Handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles partial match for GetServiceCatalogs
func (h *Handler) List(c *gin.Context) {
	var req dto.GetServiceCatalogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	filters := ListFilters{
		Category: req.Category,
		Status:   req.Status,
		Page:     req.Page,
		Size:     req.Size,
	}

	catalogs, total, err := h.service.List(c.Request.Context(), tenantID.(int), filters)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	var responses []dto.ServiceCatalogResponse
	for _, cat := range catalogs {
		responses = append(responses, h.toDTO(cat))
	}

	common.Success(c, dto.ServiceCatalogListResponse{
		Catalogs: responses,
		Total:    total,
		Page:     filters.Page,
		Size:     filters.Size,
	})
}

// Get handles GetServiceCatalogByID
func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, 1001, "无效的服务目录ID")
		return
	}

	catalog, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, h.toDTO(catalog))
}

// Create handles CreateServiceCatalog
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateServiceCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, 2001, "租户信息缺失")
		return
	}

	deliveryTime := 0
	if req.DeliveryTime != "" {
		if val, err := strconv.Atoi(req.DeliveryTime); err == nil {
			deliveryTime = val
		}
	}

	catalog, err := h.service.Create(
		c.Request.Context(),
		req.Name,
		req.Category,
		req.Description,
		deliveryTime,
		tenantID.(int),
		req.Status,
	)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, h.toDTO(catalog))
}

// Update handles UpdateServiceCatalog
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, 1001, "无效的服务目录ID")
		return
	}

	var req dto.UpdateServiceCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// tenantID unused for delete currently, maybe needed for permission check refinement
	// tenantID, exists := c.Get("tenant_id")
	// if !exists {
	// 	common.Fail(c, 2001, "租户信息缺失")
	// 	return
	// }

	deliveryTime := 0
	if req.DeliveryTime != "" {
		if val, err := strconv.Atoi(req.DeliveryTime); err == nil {
			deliveryTime = val
		}
	}

	_, err = h.service.Update(
		c.Request.Context(),
		id,
		req.Name,
		req.Category,
		req.Description,
		deliveryTime,
		req.Status,
	)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	// Fetch updated to return full object
	updated, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, h.toDTO(updated))
}

// Delete handles DeleteServiceCatalog
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, 1001, "无效的服务目录ID")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		common.Fail(c, 5001, err.Error())
		return
	}

	common.Success(c, nil)
}

func (h *Handler) toDTO(c *ServiceCatalog) dto.ServiceCatalogResponse {
	return dto.ServiceCatalogResponse{
		ID:           c.ID,
		Name:         c.Name,
		Category:     c.Category,
		Description:  c.Description,
		DeliveryTime: strconv.Itoa(c.DeliveryTime),
		Status:       c.Status,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}
