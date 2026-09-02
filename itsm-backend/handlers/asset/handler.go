package asset

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 资产与许可证 HTTP 层
type Handler struct {
	svc            *service.AssetService
	licenseSvc     *service.AssetLicenseService
	logger         *zap.SugaredLogger
}

// NewHandler 创建 asset handler
func NewHandler(svc *service.AssetService, licenseSvc *service.AssetLicenseService, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		svc:        svc,
		licenseSvc: licenseSvc,
		logger:     logger,
	}
}

// tenantIDFromCtx 从 gin 上下文提取租户ID（鉴权中间件已注入）。
func tenantIDFromCtx(c *gin.Context) (int, bool) {
	v, ok := c.Get("tenant_id")
	if !ok {
		return 0, false
	}
	tid, ok := v.(int)
	return tid, ok
}

// ==================== Asset CRUD ====================

// ListAssets GET /api/v1/assets
func (h *Handler) ListAssets(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		common.Fail(c, common.BadRequestCode, "page 必须是正整数")
		return
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if err != nil || pageSize < 1 || pageSize > 200 {
		common.Fail(c, common.BadRequestCode, "pageSize 必须在 1 到 200 之间")
		return
	}
	assetType := c.Query("type")
	status := c.Query("status")
	category := c.Query("category")

	assets, err := h.svc.ListAssets(c.Request.Context(), tenantID, page, pageSize, assetType, status, category)
	if err != nil {
		h.logger.Errorw("List assets failed", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, assets)
}

// CreateAsset POST /api/v1/assets
func (h *Handler) CreateAsset(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	asset, err := h.svc.CreateAsset(c.Request.Context(), &req, tenantID)
	if err != nil {
		h.logger.Errorw("Create asset failed", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, asset)
}

// GetAsset GET /api/v1/assets/:id
func (h *Handler) GetAsset(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的资产ID")
		return
	}

	asset, err := h.svc.GetAssetByID(c.Request.Context(), assetID, tenantID)
	if err != nil {
		h.logger.Errorw("Get asset failed", "error", err, "asset_id", assetID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	if asset == nil {
		common.Fail(c, common.NotFoundCode, "资产不存在")
		return
	}
	common.Success(c, asset)
}

// UpdateAsset PUT /api/v1/assets/:id
func (h *Handler) UpdateAsset(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的资产ID")
		return
	}

	var req dto.UpdateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	asset, err := h.svc.UpdateAsset(c.Request.Context(), assetID, tenantID, &req)
	if err != nil {
		h.logger.Errorw("Update asset failed", "error", err, "asset_id", assetID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	if asset == nil {
		common.Fail(c, common.NotFoundCode, "资产不存在")
		return
	}
	common.Success(c, asset)
}

// UpdateAssetStatus PUT /api/v1/assets/:id/status
func (h *Handler) UpdateAssetStatus(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的资产ID")
		return
	}

	var req dto.AssetStatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	asset, err := h.svc.UpdateAssetStatus(c.Request.Context(), assetID, tenantID, string(req.Status), req.AssignedTo)
	if err != nil {
		h.logger.Errorw("Update asset status failed", "error", err, "asset_id", assetID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	if asset == nil {
		common.Fail(c, common.NotFoundCode, "资产不存在")
		return
	}
	common.Success(c, asset)
}

// DeleteAsset DELETE /api/v1/assets/:id
func (h *Handler) DeleteAsset(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的资产ID")
		return
	}

	err = h.svc.DeleteAsset(c.Request.Context(), assetID, tenantID)
	if err != nil {
		h.logger.Errorw("Delete asset failed", "error", err, "asset_id", assetID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, nil)
}

// GetAssetStats GET /api/v1/assets/stats
func (h *Handler) GetAssetStats(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	stats, err := h.svc.GetAssetStats(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Get asset stats failed", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, stats)
}

// AssignAsset PUT /api/v1/assets/:id/assign
func (h *Handler) AssignAsset(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的资产ID")
		return
	}

	var req dto.AssetAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	asset, err := h.svc.AssignAsset(c.Request.Context(), assetID, tenantID, req.AssignedTo)
	if err != nil {
		h.logger.Errorw("Assign asset failed", "error", err, "asset_id", assetID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, asset)
}

// RetireAsset PUT /api/v1/assets/:id/retire
func (h *Handler) RetireAsset(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的资产ID")
		return
	}

	var req dto.AssetRetireRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	asset, err := h.svc.RetireAsset(c.Request.Context(), assetID, tenantID, req.RetireReason)
	if err != nil {
		h.logger.Errorw("Retire asset failed", "error", err, "asset_id", assetID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, asset)
}

// ==================== License CRUD ====================

// ListLicenses GET /api/v1/licenses
func (h *Handler) ListLicenses(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	licenseType := c.Query("type")
	status := c.Query("status")

	licenses, err := h.licenseSvc.ListLicenses(c.Request.Context(), tenantID, page, pageSize, licenseType, status)
	if err != nil {
		h.logger.Errorw("List licenses failed", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, licenses)
}

// CreateLicense POST /api/v1/licenses
func (h *Handler) CreateLicense(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreateLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	license, err := h.licenseSvc.CreateLicense(c.Request.Context(), &req, tenantID)
	if err != nil {
		h.logger.Errorw("Create license failed", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, license)
}

// GetLicense GET /api/v1/licenses/:id
func (h *Handler) GetLicense(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	licenseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的许可证ID")
		return
	}

	license, err := h.licenseSvc.GetLicenseByID(c.Request.Context(), licenseID, tenantID)
	if err != nil {
		h.logger.Errorw("Get license failed", "error", err, "license_id", licenseID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	if license == nil {
		common.Fail(c, common.NotFoundCode, "许可证不存在")
		return
	}
	common.Success(c, license)
}

// UpdateLicense PUT /api/v1/licenses/:id
func (h *Handler) UpdateLicense(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	licenseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的许可证ID")
		return
	}

	var req dto.UpdateLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	license, err := h.licenseSvc.UpdateLicense(c.Request.Context(), licenseID, tenantID, &req)
	if err != nil {
		h.logger.Errorw("Update license failed", "error", err, "license_id", licenseID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	if license == nil {
		common.Fail(c, common.NotFoundCode, "许可证不存在")
		return
	}
	common.Success(c, license)
}

// DeleteLicense DELETE /api/v1/licenses/:id
func (h *Handler) DeleteLicense(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	licenseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的许可证ID")
		return
	}

	err = h.licenseSvc.DeleteLicense(c.Request.Context(), licenseID, tenantID)
	if err != nil {
		h.logger.Errorw("Delete license failed", "error", err, "license_id", licenseID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, nil)
}

// GetLicenseStats GET /api/v1/licenses/stats
func (h *Handler) GetLicenseStats(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	stats, err := h.licenseSvc.GetLicenseStats(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Errorw("Get license stats failed", "error", err, "tenant_id", tenantID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, stats)
}

// AssignUsers PUT /api/v1/licenses/:id/assign
func (h *Handler) AssignUsers(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	licenseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的许可证ID")
		return
	}

	var req dto.LicenseAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	license, err := h.licenseSvc.AssignUsers(c.Request.Context(), licenseID, tenantID, req.UserIDs)
	if err != nil {
		h.logger.Errorw("Assign users to license failed", "error", err, "license_id", licenseID)
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, license)
}
