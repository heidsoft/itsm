package service_catalog

import (
	"strconv"
	"strings"

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
			common.Fail(c, common.ConflictCode, appErr.Message)
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
		Category:       req.Category,
		Status:         req.Status,
		Keyword:        req.Keyword,
		CiTypeID:       req.CiTypeID,
		CloudServiceID: req.CloudServiceID,
	}
	// 分页在 HTTP 入口只夹紧一次：common.GetPaginationFromQuery（缺省 1/20，只采纳
	// (0,100]，越界回落默认页长）。handler、Service.List、Service.Search 和 DTO 上的
	// binding:"max=1000" 都不再各定一套页长。
	pagination := common.GetPaginationFromQuery(c)
	filters.Page, filters.PageSize = pagination.Page, pagination.PageSize

	catalogs, total, err := h.service.List(c.Request.Context(), tenantID, filters)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	// 显式 make 而不是 var：nil slice 序列化成 null，前端契约要求空集合是 []。
	responses := make([]dto.ServiceCatalogResponse, 0, len(catalogs))
	for _, cat := range catalogs {
		responses = append(responses, h.toDTO(cat))
	}

	common.Success(c, serviceCatalogListResponse(responses, total, filters.Page, filters.PageSize))
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

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	patch := &CatalogPatch{
		Name:              req.Name,
		Category:          req.Category,
		Description:       req.Description,
		Icon:              req.Icon,
		ServiceType:       req.ServiceType,
		Price:             req.Price,
		Unit:              req.Unit,
		RequiresApproval:  req.RequiresApproval,
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
	}
	// deliveryTime 在线上是字符串、在领域里是天数：只有真正提交过键才参与更新，
	// 空串和非数字都是明确的输入错误，不能静默当成「不改」。
	if req.DeliveryTime != nil {
		days, parseErr := strconv.Atoi(strings.TrimSpace(*req.DeliveryTime))
		if parseErr != nil {
			common.Fail(c, common.ParamErrorCode, "deliveryTime 必须是正整数天数")
			return
		}
		patch.DeliveryTime = &days
	}

	updated, err := h.service.Update(c.Request.Context(), tenantID, id, patch)
	if err != nil {
		failServiceCatalog(c, err)
		return
	}

	common.Success(c, h.toDTO(updated))
}

func normalizeServiceCatalogRequest(req *dto.CreateServiceCatalogRequest) {
	if req.DeliveryTime == "" {
		req.DeliveryTime = "1"
	}
	if req.Status == "" {
		req.Status = "enabled"
	}
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
	}
	// 原实现把 Page/Size 写死成 1/20，因此 /search 永远只能拿到第一页；
	// 现在与 List 共用同一个 HTTP 入口所有者，翻页参数真正生效。
	pagination := common.GetPaginationFromQuery(c)
	filters.Page, filters.PageSize = pagination.Page, pagination.PageSize

	catalogs, total, err := h.service.Search(c.Request.Context(), tenantID, keyword, filters)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	// 显式 make 而不是 var：nil slice 序列化成 null，前端契约要求空集合是 []。
	responses := make([]dto.ServiceCatalogResponse, 0, len(catalogs))
	for _, cat := range catalogs {
		responses = append(responses, h.toDTO(cat))
	}

	common.Success(c, serviceCatalogListResponse(responses, total, filters.Page, filters.PageSize))
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

// serviceCatalogListResponse 用同一份 common.NewPaginationResponse 计算五键信封，
// 避免 List 与 Search 各自用本地除法算 totalPages。
func serviceCatalogListResponse(items []dto.ServiceCatalogResponse, total, page, pageSize int) dto.ServiceCatalogListResponse {
	pagination := common.NewPaginationResponse(page, pageSize, int64(total))
	return dto.ServiceCatalogListResponse{
		Items:      items,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: pagination.TotalPages,
	}
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
