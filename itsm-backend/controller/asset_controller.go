package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AssetController 资产管理控制器
type AssetController struct {
	logger       *zap.SugaredLogger
	assetService *service.AssetService
}

// NewAssetController 创建资产管理控制器
func NewAssetController(logger *zap.SugaredLogger, assetService *service.AssetService) *AssetController {
	return &AssetController{
		logger:       logger,
		assetService: assetService,
	}
}

// ListAssets 获取资产列表
// @Summary 获取资产列表
// @Description 获取所有资产的列表，支持分页和筛选
// @Tags 资产管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param type query string false "资产类型"
// @Param status query string false "资产状态"
// @Param category query string false "资产分类"
// @Success 200 {object} common.Response
// @Router /api/v1/assets [get]
func (ac *AssetController) ListAssets(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	assetType := c.Query("type")
	status := c.Query("status")
	category := c.Query("category")

	assets, err := ac.assetService.ListAssets(c.Request.Context(), tenantID, page, pageSize, assetType, status, category)
	if err != nil {
		ac.logger.Errorw("List assets failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取资产列表失败: "+err.Error())
		return
	}

	common.Success(c, assets)
}

// CreateAsset 创建资产
// @Summary 创建资产
// @Description 创建新的资产记录
// @Tags 资产管理
// @Accept json
// @Produce json
// @Param request body dto.CreateAssetRequest true "创建资产请求"
// @Success 200 {object} common.Response
// @Router /api/v1/assets [post]
func (ac *AssetController) CreateAsset(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	asset, err := ac.assetService.CreateAsset(c.Request.Context(), &req, tenantID)
	if err != nil {
		ac.logger.Errorw("Create asset failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "创建资产失败: "+err.Error())
		return
	}

	common.Success(c, asset)
}

// GetAsset 获取资产详情
// @Summary 获取资产详情
// @Description 根据ID获取资产的详细信息
// @Tags 资产管理
// @Accept json
// @Produce json
// @Param id path int true "资产ID"
// @Success 200 {object} common.Response
// @Router /api/v1/assets/{id} [get]
func (ac *AssetController) GetAsset(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的资产ID")
		return
	}

	asset, err := ac.assetService.GetAssetByID(c.Request.Context(), assetID, tenantID)
	if err != nil {
		ac.logger.Errorw("Get asset failed", "error", err, "asset_id", assetID)
		common.Fail(c, common.InternalErrorCode, "获取资产详情失败: "+err.Error())
		return
	}

	if asset == nil {
		common.Fail(c, common.NotFoundCode, "资产不存在")
		return
	}

	common.Success(c, asset)
}

// UpdateAsset 更新资产
func (ac *AssetController) UpdateAsset(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的资产ID")
		return
	}

	var req dto.UpdateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	asset, err := ac.assetService.UpdateAsset(c.Request.Context(), assetID, tenantID, &req)
	if err != nil {
		ac.logger.Errorw("Update asset failed", "error", err, "asset_id", assetID)
		common.Fail(c, common.InternalErrorCode, "更新资产失败: "+err.Error())
		return
	}

	if asset == nil {
		common.Fail(c, common.NotFoundCode, "资产不存在")
		return
	}

	common.Success(c, asset)
}

// UpdateAssetStatus 更新资产状态
func (ac *AssetController) UpdateAssetStatus(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的资产ID")
		return
	}

	var req dto.AssetStatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	asset, err := ac.assetService.UpdateAssetStatus(c.Request.Context(), assetID, tenantID, string(req.Status), req.AssignedTo)
	if err != nil {
		ac.logger.Errorw("Update asset status failed", "error", err, "asset_id", assetID)
		common.Fail(c, common.InternalErrorCode, "更新资产状态失败: "+err.Error())
		return
	}

	if asset == nil {
		common.Fail(c, common.NotFoundCode, "资产不存在")
		return
	}

	common.Success(c, asset)
}

// DeleteAsset 删除资产
func (ac *AssetController) DeleteAsset(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的资产ID")
		return
	}

	err = ac.assetService.DeleteAsset(c.Request.Context(), assetID, tenantID)
	if err != nil {
		ac.logger.Errorw("Delete asset failed", "error", err, "asset_id", assetID)
		common.Fail(c, common.InternalErrorCode, "删除资产失败: "+err.Error())
		return
	}

	common.Success(c, nil)
}

// GetAssetStats 获取资产统计
func (ac *AssetController) GetAssetStats(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	stats, err := ac.assetService.GetAssetStats(c.Request.Context(), tenantID)
	if err != nil {
		ac.logger.Errorw("Get asset stats failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取资产统计失败: "+err.Error())
		return
	}

	common.Success(c, stats)
}

// AssignAsset 分配资产
func (ac *AssetController) AssignAsset(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的资产ID")
		return
	}

	var req dto.AssetAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	asset, err := ac.assetService.AssignAsset(c.Request.Context(), assetID, tenantID, req.AssignedTo)
	if err != nil {
		ac.logger.Errorw("Assign asset failed", "error", err, "asset_id", assetID)
		common.Fail(c, common.InternalErrorCode, "分配资产失败: "+err.Error())
		return
	}

	common.Success(c, asset)
}

// RetireAsset 退役资产
func (ac *AssetController) RetireAsset(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的资产ID")
		return
	}

	var req dto.AssetRetireRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	asset, err := ac.assetService.RetireAsset(c.Request.Context(), assetID, tenantID, req.RetireReason)
	if err != nil {
		ac.logger.Errorw("Retire asset failed", "error", err, "asset_id", assetID)
		common.Fail(c, common.InternalErrorCode, "退役资产失败: "+err.Error())
		return
	}

	common.Success(c, asset)
}
