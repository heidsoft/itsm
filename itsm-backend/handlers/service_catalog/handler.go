package service_catalog

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for Service Catalog
type Handler struct {
	service *Service
}

func failServiceCatalog(c *gin.Context, err error) {
	if appErr, ok := common.AsAppError(err); ok {
		switch appErr.Code {
		case common.ErrCodeBadRequest, common.ErrCodeValidation:
			common.Fail(c, common.ParamErrorCode, appErr.Message)
		case common.ErrCodeConflict:
			common.Fail(c, common.ConflictCode, appErr.Error())
		case common.ErrCodeNotFound:
			common.Fail(c, common.NotFoundErrorCode, appErr.Message)
		default:
			common.Fail(c, common.InternalErrorCode, appErr.Message)
		}
		return
	}
	if ent.IsNotFound(err) {
		common.Fail(c, common.NotFoundErrorCode, "服务目录不存在")
		return
	}
	common.FailWithErr(c, err, "操作失败")
}

// NewHandler creates a new Handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles partial match for GetServiceCatalogs
func (h *Handler) List(c *gin.Context) {
	var req dto.GetServiceCatalogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	filters := ListFilters{
		Category: req.Category,
		Status:   req.Status,
		Page:     req.Page,
		Size:     req.Size,
	}
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Size < 1 {
		filters.Size = 10
	}
	if filters.Size > 100 {
		filters.Size = 100
	}

	catalogs, total, err := h.service.List(c.Request.Context(), tenantID, filters)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
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

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	catalog, err := h.service.Get(c.Request.Context(), tenantID, id)
	if err != nil {
		failServiceCatalog(c, err)
		return
	}

	common.Success(c, h.toDTO(catalog))
}

// Create handles CreateServiceCatalog
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateServiceCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	normalizeServiceCatalogRequest(&req)
	if req.CloudServiceID > 0 && req.CITypeID == 0 {
		common.Fail(c, 1001, "关联云服务时必须选择CI类型")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	deliveryTime := 0
	if req.DeliveryTime != "" {
		val, parseErr := strconv.Atoi(req.DeliveryTime)
		if parseErr != nil || val <= 0 {
			common.Fail(c, common.ParamErrorCode, "deliveryTime 必须为正整数天数")
			return
		}
		deliveryTime = val
	}

	requiresApproval := true
	if req.RequiresApproval != nil {
		requiresApproval = *req.RequiresApproval
	}

	catalog := &ServiceCatalog{
		Name:              req.Name,
		Category:          req.Category,
		Description:       req.Description,
		Icon:              req.Icon,
		ServiceType:       req.ServiceType,
		Price:             req.Price,
		DeliveryTime:      deliveryTime,
		Unit:              req.Unit,
		RequiresApproval:  requiresApproval,
		ApprovalLevel:     req.ApprovalLevel,
		Approvers:         req.Approvers,
		SLAResponseTime:   req.SLAResponseTime,
		SLAResolutionTime: req.SLAResolutionTime,
		CITypeID:          req.CITypeID,
		CloudServiceID:    req.CloudServiceID,
		FormSchema:        req.FormSchema,
		AvailableRegions:  req.AvailableRegions,
		AvailableSpecs:    req.AvailableSpecs,
		Status:            req.Status,
		SortOrder:         req.SortOrder,
		TenantID:          tenantID,
	}

	result, err := h.service.Create(c.Request.Context(), catalog)
	if err != nil {
		failServiceCatalog(c, err)
		return
	}

	common.Success(c, h.toDTO(result))
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
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	normalizeUpdateServiceCatalogRequest(&req)

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	deliveryTime := 0
	if req.DeliveryTime != nil && *req.DeliveryTime != "" {
		val, parseErr := strconv.Atoi(*req.DeliveryTime)
		if parseErr != nil || val < 0 {
			common.Fail(c, common.ParamErrorCode, "deliveryTime 必须为非负整数天数")
			return
		}
		deliveryTime = val
	}

	catalog := &ServiceCatalog{
		ID:                id,
		Name:              getStringValue(req.Name),
		Category:          getStringValue(req.Category),
		Description:       getStringValue(req.Description),
		Icon:              getStringValue(req.Icon),
		ServiceType:       getStringValue(req.ServiceType),
		Price:             getFloat64Value(req.Price),
		DeliveryTime:      deliveryTime,
		Unit:              getStringValue(req.Unit),
		RequiresApproval:  getBoolValue(req.RequiresApproval),
		ApprovalLevel:     getIntValue(req.ApprovalLevel),
		Approvers:         req.Approvers,
		SLAResponseTime:   getIntValue(req.SLAResponseTime),
		SLAResolutionTime: getIntValue(req.SLAResolutionTime),
		CITypeID:          getIntValue(req.CITypeID),
		CloudServiceID:    getIntValue(req.CloudServiceID),
		FormSchema:        getMapValue(req.FormSchema),
		AvailableRegions:  req.AvailableRegions,
		AvailableSpecs:    req.AvailableSpecs,
		Status:            getStringValue(req.Status),
		SortOrder:         getIntValue(req.SortOrder),
	}

	_, err = h.service.Update(c.Request.Context(), tenantID, catalog)
	if err != nil {
		failServiceCatalog(c, err)
		return
	}

	// Fetch updated to return full object
	updated, err := h.service.Get(c.Request.Context(), tenantID, id)
	if err != nil {
		failServiceCatalog(c, err)
		return
	}

	common.Success(c, h.toDTO(updated))
}

func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getIntValue(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func getFloat64Value(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func getBoolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func getMapValue(m *map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	return *m
}

func normalizeServiceCatalogRequest(req *dto.CreateServiceCatalogRequest) {
	if req.DeliveryTime == "" {
		req.DeliveryTime = "1"
	}
	if req.Status == "" {
		req.Status = "enabled"
	}
}

func normalizeUpdateServiceCatalogRequest(req *dto.UpdateServiceCatalogRequest) {
}

// Delete handles DeleteServiceCatalog
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, 1001, "无效的服务目录ID")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	if err := h.service.Delete(c.Request.Context(), tenantID, id); err != nil {
		failServiceCatalog(c, err)
		return
	}

	common.Success(c, nil)
}

// Search handles GET /api/v1/service-catalogs/search?q=xxx
func (h *Handler) Search(c *gin.Context) {
	keyword := c.Query("q")
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	filters := ListFilters{
		Category: c.Query("category"),
		Status:   "enabled",
		Page:     1,
		Size:     20,
	}

	catalogs, total, err := h.service.Search(c.Request.Context(), tenantID, keyword, filters)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
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

// Stats handles GET /api/v1/service-catalogs/stats
func (h *Handler) Stats(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	stats, err := h.service.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, stats)
}

func (h *Handler) toDTO(c *ServiceCatalog) dto.ServiceCatalogResponse {
	return dto.ServiceCatalogResponse{
		ID:                c.ID,
		Name:              c.Name,
		Category:          c.Category,
		Description:       c.Description,
		Icon:              c.Icon,
		ServiceType:       c.ServiceType,
		Price:             c.Price,
		DeliveryTime:      strconv.Itoa(c.DeliveryTime),
		Unit:              c.Unit,
		RequiresApproval:  c.RequiresApproval,
		ApprovalLevel:     c.ApprovalLevel,
		Approvers:         c.Approvers,
		SLAResponseTime:   c.SLAResponseTime,
		SLAResolutionTime: c.SLAResolutionTime,
		CITypeID:          c.CITypeID,
		CloudServiceID:    c.CloudServiceID,
		FormSchema:        c.FormSchema,
		AvailableRegions:  c.AvailableRegions,
		AvailableSpecs:    c.AvailableSpecs,
		Status:            c.Status,
		IsActive:          c.IsActive,
		SortOrder:         c.SortOrder,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}
