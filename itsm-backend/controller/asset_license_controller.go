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

// AssetLicenseController 许可证管理控制器
type AssetLicenseController struct {
	logger            *zap.SugaredLogger
	licenseService    *service.AssetLicenseService
}

// NewAssetLicenseController 创建许可证管理控制器
func NewAssetLicenseController(logger *zap.SugaredLogger, licenseService *service.AssetLicenseService) *AssetLicenseController {
	return &AssetLicenseController{
		logger:         logger,
		licenseService: licenseService,
	}
}

// ListLicenses 获取许可证列表
func (lc *AssetLicenseController) ListLicenses(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	licenseType := c.Query("type")
	status := c.Query("status")

	licenses, err := lc.licenseService.ListLicenses(c.Request.Context(), tenantID, page, pageSize, licenseType, status)
	if err != nil {
		lc.logger.Errorw("List licenses failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取许可证列表失败: "+err.Error())
		return
	}

	common.Success(c, licenses)
}

// CreateLicense 创建许可证
func (lc *AssetLicenseController) CreateLicense(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreateLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	license, err := lc.licenseService.CreateLicense(c.Request.Context(), &req, tenantID)
	if err != nil {
		lc.logger.Errorw("Create license failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "创建许可证失败: "+err.Error())
		return
	}

	common.Success(c, license)
}

// GetLicense 获取许可证详情
func (lc *AssetLicenseController) GetLicense(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	licenseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的许可证ID")
		return
	}

	license, err := lc.licenseService.GetLicenseByID(c.Request.Context(), licenseID, tenantID)
	if err != nil {
		lc.logger.Errorw("Get license failed", "error", err, "license_id", licenseID)
		common.Fail(c, common.InternalErrorCode, "获取许可证详情失败: "+err.Error())
		return
	}

	if license == nil {
		common.Fail(c, common.NotFoundCode, "许可证不存在")
		return
	}

	common.Success(c, license)
}

// UpdateLicense 更新许可证
func (lc *AssetLicenseController) UpdateLicense(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	licenseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的许可证ID")
		return
	}

	var req dto.UpdateLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	license, err := lc.licenseService.UpdateLicense(c.Request.Context(), licenseID, tenantID, &req)
	if err != nil {
		lc.logger.Errorw("Update license failed", "error", err, "license_id", licenseID)
		common.Fail(c, common.InternalErrorCode, "更新许可证失败: "+err.Error())
		return
	}

	if license == nil {
		common.Fail(c, common.NotFoundCode, "许可证不存在")
		return
	}

	common.Success(c, license)
}

// DeleteLicense 删除许可证
func (lc *AssetLicenseController) DeleteLicense(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	licenseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的许可证ID")
		return
	}

	err = lc.licenseService.DeleteLicense(c.Request.Context(), licenseID, tenantID)
	if err != nil {
		lc.logger.Errorw("Delete license failed", "error", err, "license_id", licenseID)
		common.Fail(c, common.InternalErrorCode, "删除许可证失败: "+err.Error())
		return
	}

	common.Success(c, nil)
}

// GetLicenseStats 获取许可证统计
func (lc *AssetLicenseController) GetLicenseStats(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	stats, err := lc.licenseService.GetLicenseStats(c.Request.Context(), tenantID)
	if err != nil {
		lc.logger.Errorw("Get license stats failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取许可证统计失败: "+err.Error())
		return
	}

	common.Success(c, stats)
}

// AssignUsers 分配许可证给用户
func (lc *AssetLicenseController) AssignUsers(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	licenseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的许可证ID")
		return
	}

	var req dto.LicenseAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	license, err := lc.licenseService.AssignUsers(c.Request.Context(), licenseID, tenantID, req.UserIDs)
	if err != nil {
		lc.logger.Errorw("Assign users to license failed", "error", err, "license_id", licenseID)
		common.Fail(c, common.InternalErrorCode, "分配许可证失败: "+err.Error())
		return
	}

	common.Success(c, license)
}
