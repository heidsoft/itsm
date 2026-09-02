package standard_change

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	svc    *Service
	logger *zap.SugaredLogger
}

func NewHandler(svc *Service, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
	}
}

func (h *Handler) toResponse(sc *ent.StandardChange) *dto.StandardChangeResponse {
	if sc == nil {
		return nil
	}
	return &dto.StandardChangeResponse{
		ID:                  sc.ID,
		Title:               sc.Title,
		Description:         sc.Description,
		ImplementationPlan:  sc.ImplementationPlan,
		RollbackPlan:        sc.RollbackPlan,
		Justification:       sc.Justification,
		Category:            sc.Category,
		RiskLevel:           sc.RiskLevel,
		ImpactScope:         sc.ImpactScope,
		ExpectedDuration:    sc.ExpectedDuration,
		ApprovalRequired:    sc.ApprovalRequired,
		AffectedCis:         sc.AffectedCis,
		Prerequisites:       sc.Prerequisites,
		Remarks:             sc.Remarks,
		CreatedBy:           sc.CreatedBy,
		TenantID:            sc.TenantID,
		IsActive:            sc.IsActive,
		CreatedAt:           sc.CreatedAt,
		UpdatedAt:           sc.UpdatedAt,
	}
}

func (h *Handler) ListStandardChanges(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	category := c.Query("category")
	search := c.Query("search")
	activeOnly := c.Query("active_only") == "true"

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	results, total, err := h.svc.ListStandardChanges(c.Request.Context(), tenantID, page, pageSize, category, search, activeOnly)
	if err != nil {
		h.logger.Warnw("Failed to list standard changes", "error", err)
		common.InternalError(c, "Failed to list standard changes")
		return
	}

	templates := make([]dto.StandardChangeResponse, 0, len(results))
	for _, sc := range results {
		templates = append(templates, *h.toResponse(sc))
	}

	common.Success(c, gin.H{
		"templates": templates,
		"total":     total,
		"page":      page,
		"pageSize":  pageSize,
	})
}

func (h *Handler) GetStandardChange(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	sc, err := h.svc.GetStandardChange(c.Request.Context(), tenantID, id)
	if err != nil {
		common.FailWithErr(c, err, "获取标准变更模板失败")
		return
	}

	common.Success(c, h.toResponse(sc))
}

func (h *Handler) CreateStandardChange(c *gin.Context) {
	var req dto.CreateStandardChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, err.Error())
		return
	}

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	createdBy := h.getUserID(c)

	input := &SCCreateInput{
		TenantID:           tenantID,
		Title:              req.Title,
		Description:        req.Description,
		ImplementationPlan: req.ImplementationPlan,
		RollbackPlan:       req.RollbackPlan,
		Justification:      req.Justification,
		Category:           req.Category,
		RiskLevel:          req.RiskLevel,
		ImpactScope:        req.ImpactScope,
		ExpectedDuration:   req.ExpectedDuration,
		ApprovalRequired:   req.ApprovalRequired,
		AFFECTEDCIs:        req.AffectedCis,
		Prerequisites:      req.Prerequisites,
		Remarks:            req.Remarks,
		CreatedBy:          createdBy,
	}

	sc, err := h.svc.CreateStandardChange(c.Request.Context(), input)
	if err != nil {
		common.FailWithErr(c, err, "创建标准变更模板失败")
		return
	}

	common.Success(c, h.toResponse(sc))
}

func (h *Handler) UpdateStandardChange(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateStandardChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, err.Error())
		return
	}

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	input := &SCUpdateInput{}
	if req.Title != nil {
		input.Title = req.Title
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.ImplementationPlan != nil {
		input.ImplementationPlan = req.ImplementationPlan
	}
	if req.RollbackPlan != nil {
		input.RollbackPlan = req.RollbackPlan
	}
	if req.Justification != nil {
		input.Justification = req.Justification
	}
	if req.Category != nil {
		input.Category = req.Category
	}
	if req.RiskLevel != nil {
		input.RiskLevel = req.RiskLevel
	}
	if req.ImpactScope != nil {
		input.ImpactScope = req.ImpactScope
	}
	if req.ExpectedDuration != nil {
		input.ExpectedDuration = req.ExpectedDuration
	}
	if req.ApprovalRequired != nil {
		input.ApprovalRequired = req.ApprovalRequired
	}
	if req.AffectedCis != nil {
		input.AFFECTEDCIs = &req.AffectedCis
	}
	if req.Prerequisites != nil {
		input.Prerequisites = &req.Prerequisites
	}
	if req.Remarks != nil {
		input.Remarks = req.Remarks
	}
	if req.IsActive != nil {
		input.IsActive = req.IsActive
	}

	sc, err := h.svc.UpdateStandardChange(c.Request.Context(), tenantID, id, input)
	if err != nil {
		common.FailWithErr(c, err, "更新标准变更模板失败")
		return
	}

	common.Success(c, h.toResponse(sc))
}

func (h *Handler) DeleteStandardChange(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	if err := h.svc.DeleteStandardChange(c.Request.Context(), tenantID, id); err != nil {
		common.FailWithErr(c, err, "删除标准变更模板失败")
		return
	}

	common.Success(c, nil)
}

func (h *Handler) GetCategories(c *gin.Context) {
	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	cats, err := h.svc.GetCategories(c.Request.Context(), tenantID)
	if err != nil {
		common.InternalError(c, "获取分类失败")
		return
	}

	common.Success(c, gin.H{"items": cats})
}

func (h *Handler) InstantiateStandardChange(c *gin.Context) {
	id, ok := common.ParsePositiveID(c, "id")
	if !ok {
		return
	}

	tenantID, ok := h.getTenantID(c)
	if !ok {
		return
	}

	createdBy := h.getUserID(c)

	change, err := h.svc.Instantiate(c.Request.Context(), tenantID, id, createdBy)
	if err != nil {
		common.FailWithErr(c, err, "实例化标准变更失败")
		return
	}

	common.Success(c, gin.H{
		"change_id": change.ID,
		"change_number": change.ChangeNumber,
	})
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	sc := r.Group("/standard-changes")
	{
		sc.GET("", h.ListStandardChanges)
		sc.POST("", h.CreateStandardChange)
		sc.GET("/categories", h.GetCategories)
		sc.GET("/:id", h.GetStandardChange)
		sc.PUT("/:id", h.UpdateStandardChange)
		sc.DELETE("/:id", h.DeleteStandardChange)
		sc.POST("/:id/instantiate", h.InstantiateStandardChange)
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
