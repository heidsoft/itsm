package known_error

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	svc   *Service
	logger *zap.SugaredLogger
}

func NewHandler(svc *Service, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		svc:   svc,
		logger: logger,
	}
}

func (h *Handler) toResponse(ke *ent.KnownError) *dto.KEDBResponse {
	if ke == nil {
		return nil
	}
	return &dto.KEDBResponse{
		ID:               ke.ID,
		Title:            ke.Title,
		Description:      ke.Description,
		Symptoms:         ke.Symptoms,
		RootCause:        ke.RootCause,
		Workaround:       ke.Workaround,
		Resolution:       ke.Resolution,
		Status:           ke.Status,
		Category:         ke.Category,
		Severity:         ke.Severity,
		AffectedProducts: ke.AffectedProducts,
		AffectedCIs:      ke.AffectedCis,
		Keywords:         ke.Keywords,
		OccurrenceCount:  ke.OccurrenceCount,
		CreatedBy:        ke.CreatedBy,
		TenantID:         ke.TenantID,
		CreatedAt:        ke.CreatedAt,
		UpdatedAt:        ke.UpdatedAt,
	}
}

func (h *Handler) ListKnownErrors(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	items, total, err := h.svc.ListKnownErrors(c.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		common.InternalError(c, "获取已知错误列表失败")
		return
	}

	dtos := make([]dto.KEDBResponse, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, *h.toResponse(item))
	}

	common.SuccessWithList(c, dtos, total, page, pageSize)
}

func (h *Handler) GetKnownError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的ID")
		return
	}

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	item, err := h.svc.GetKnownError(c.Request.Context(), tenantID, id)
	if err != nil {
		common.FailWithErr(c, err, "获取已知错误失败")
		return
	}

	common.Success(c, h.toResponse(item))
}

func (h *Handler) CreateKnownError(c *gin.Context) {
	var req dto.CreateKnownErrorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, err.Error())
		return
	}

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	createdBy := h.getUserID(c)

	input := &KnownErrorCreateInput{
		TenantID:    tenantID,
		Title:       req.Title,
		Description: req.Description,
		Symptoms:    req.Symptoms,
		RootCause:   req.RootCause,
		Workaround:  req.Workaround,
		Resolution:  req.Resolution,
		Status:      req.Status,
		Category:    req.Category,
		Severity:    req.Severity,
		CreatedBy:   createdBy,
	}

	item, err := h.svc.CreateKnownError(c.Request.Context(), input)
	if err != nil {
		common.FailWithErr(c, err, "创建已知错误失败")
		return
	}

	common.Success(c, h.toResponse(item))
}

func (h *Handler) UpdateKnownError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的ID")
		return
	}

	var req dto.KEDBUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, err.Error())
		return
	}

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	input := &KnownErrorUpdateInput{}
	if req.Title != nil {
		input.Title = req.Title
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.Symptoms != nil {
		input.Symptoms = req.Symptoms
	}
	if req.RootCause != nil {
		input.RootCause = req.RootCause
	}
	if req.Workaround != nil {
		input.Workaround = req.Workaround
	}
	if req.Resolution != nil {
		input.Resolution = req.Resolution
	}
	if req.Status != nil {
		input.Status = req.Status
	}
	if req.Category != nil {
		input.Category = req.Category
	}
	if req.Severity != nil {
		input.Severity = req.Severity
	}
	if req.AffectedProducts != nil {
		input.AffectedProducts = &req.AffectedProducts
	}
	if req.AffectedCIs != nil {
		input.AffectedCIs = &req.AffectedCIs
	}
	if req.Keywords != nil {
		input.Keywords = &req.Keywords
	}

	item, err := h.svc.UpdateKnownError(c.Request.Context(), tenantID, id, input)
	if err != nil {
		common.FailWithErr(c, err, "更新已知错误失败")
		return
	}

	common.Success(c, h.toResponse(item))
}

func (h *Handler) DeleteKnownError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的ID")
		return
	}

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	if err := h.svc.DeleteKnownError(c.Request.Context(), tenantID, id); err != nil {
		common.FailWithErr(c, err, "删除已知错误失败")
		return
	}

	common.Success(c, nil)
}

func (h *Handler) GetStats(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	stats, err := h.svc.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		common.InternalError(c, "获取统计信息失败")
		return
	}

	common.Success(c, gin.H{
		"total":       stats.Total,
		"active":      stats.Active,
		"resolved":    stats.Resolved,
		"deprecated":  stats.Deprecated,
		"critical":    stats.Critical,
		"high":        stats.High,
		"medium":      stats.Medium,
		"low":         stats.Low,
		"totalPages":  1,
		"page":        1,
	})
}

func (h *Handler) SearchKnownErrors(c *gin.Context) {
	keyword := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	items, total, err := h.svc.SearchKnownErrors(c.Request.Context(), tenantID, keyword, page, pageSize)
	if err != nil {
		common.InternalError(c, "搜索已知错误失败")
		return
	}

	dtos := make([]dto.KEDBResponse, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, *h.toResponse(item))
	}

	common.SuccessWithList(c, dtos, total, page, pageSize)
}

func (h *Handler) GetCategories(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	items, _, err := h.svc.ListKnownErrors(c.Request.Context(), tenantID, 1, 1000)
	if err != nil {
		common.InternalError(c, "获取分类失败")
		return
	}

	categorySet := make(map[string]bool)
	categories := make([]string, 0)
	for _, ke := range items {
		if ke.Category != "" {
			if !categorySet[ke.Category] {
				categorySet[ke.Category] = true
				categories = append(categories, ke.Category)
			}
		}
	}

	common.Success(c, gin.H{"items": categories})
}

func (h *Handler) PromoteToKnownError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的ID")
		return
	}

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	item, err := h.svc.GetKnownError(c.Request.Context(), tenantID, id)
	if err != nil {
		common.FailWithErr(c, err, "获取已知错误失败")
		return
	}

	if item.Status == "resolved" {
		common.Fail(c, common.BadRequestCode, "已解决的问题不能再次提升")
		return
	}

	updated, err := h.svc.UpdateKnownError(c.Request.Context(), tenantID, id, &KnownErrorUpdateInput{
		Status: ptrString("resolved"),
	})
	if err != nil {
		common.FailWithErr(c, err, "提升已知错误失败")
		return
	}

	common.Success(c, h.toResponse(updated))
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	errors := r.Group("/known-errors")
	{
		errors.GET("", h.ListKnownErrors)
		errors.POST("", h.CreateKnownError)
		errors.GET("/stats", h.GetStats)
		errors.GET("/search", h.SearchKnownErrors)
		errors.GET("/categories", h.GetCategories)
		errors.GET("/:id", h.GetKnownError)
		errors.PUT("/:id", h.UpdateKnownError)
		errors.DELETE("/:id", h.DeleteKnownError)
		errors.POST("/:id/promote", h.PromoteToKnownError)
	}
}

func (h *Handler) getTenantID(c *gin.Context) (int, bool) {
	tenantIDVal, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.AuthErrorCode, "租户上下文缺失")
		return 0, false
	}
	tenantID, ok := tenantIDVal.(int)
	if !ok || tenantID == 0 {
		common.Fail(c, common.AuthErrorCode, "无效的租户上下文")
		return 0, false
	}
	return tenantID, true
}

func (h *Handler) getUserID(c *gin.Context) int {
	if uid, ok := c.Get("user_id"); ok {
		if id, ok := uid.(int); ok {
			return id
		}
	}
	return 0
}

func ptrString(s string) *string {
	return &s
}
