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

// ReleaseController 发布管理控制器
type ReleaseController struct {
	logger        *zap.SugaredLogger
	releaseService *service.ReleaseService
}

// NewReleaseController 创建发布管理控制器
func NewReleaseController(logger *zap.SugaredLogger, releaseService *service.ReleaseService) *ReleaseController {
	return &ReleaseController{
		logger:         logger,
		releaseService: releaseService,
	}
}

// ListReleases 获取发布列表
func (rc *ReleaseController) ListReleases(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	status := c.Query("status")
	releaseType := c.Query("type")

	releases, err := rc.releaseService.ListReleases(c.Request.Context(), tenantID, page, pageSize, status, releaseType)
	if err != nil {
		rc.logger.Errorw("List releases failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取发布列表失败: "+err.Error())
		return
	}

	common.Success(c, releases)
}

// CreateRelease 创建发布
func (rc *ReleaseController) CreateRelease(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	userID, err := middleware.GetUserID(c)
	if err != nil || userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreateReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	release, err := rc.releaseService.CreateRelease(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		rc.logger.Errorw("Create release failed", "error", err, "tenant_id", tenantID, "user_id", userID)
		common.Fail(c, common.InternalErrorCode, "创建发布失败: "+err.Error())
		return
	}

	common.Success(c, release)
}

// GetRelease 获取发布详情
func (rc *ReleaseController) GetRelease(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	releaseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的发布ID")
		return
	}

	release, err := rc.releaseService.GetReleaseByID(c.Request.Context(), releaseID, tenantID)
	if err != nil {
		rc.logger.Errorw("Get release failed", "error", err, "release_id", releaseID)
		common.Fail(c, common.InternalErrorCode, "获取发布详情失败: "+err.Error())
		return
	}

	if release == nil {
		common.Fail(c, common.NotFoundCode, "发布不存在")
		return
	}

	common.Success(c, release)
}

// UpdateRelease 更新发布
func (rc *ReleaseController) UpdateRelease(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	releaseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的发布ID")
		return
	}

	var req dto.UpdateReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	release, err := rc.releaseService.UpdateRelease(c.Request.Context(), releaseID, tenantID, &req)
	if err != nil {
		rc.logger.Errorw("Update release failed", "error", err, "release_id", releaseID)
		common.Fail(c, common.InternalErrorCode, "更新发布失败: "+err.Error())
		return
	}

	if release == nil {
		common.Fail(c, common.NotFoundCode, "发布不存在")
		return
	}

	common.Success(c, release)
}

// UpdateReleaseStatus 更新发布状态
func (rc *ReleaseController) UpdateReleaseStatus(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	releaseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的发布ID")
		return
	}

	var req dto.ReleaseStatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	release, err := rc.releaseService.UpdateReleaseStatus(c.Request.Context(), releaseID, tenantID, string(req.Status))
	if err != nil {
		rc.logger.Errorw("Update release status failed", "error", err, "release_id", releaseID)
		common.Fail(c, common.InternalErrorCode, "更新发布状态失败: "+err.Error())
		return
	}

	if release == nil {
		common.Fail(c, common.NotFoundCode, "发布不存在")
		return
	}

	common.Success(c, release)
}

// DeleteRelease 删除发布
func (rc *ReleaseController) DeleteRelease(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	releaseID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的发布ID")
		return
	}

	err = rc.releaseService.DeleteRelease(c.Request.Context(), releaseID, tenantID)
	if err != nil {
		rc.logger.Errorw("Delete release failed", "error", err, "release_id", releaseID)
		common.Fail(c, common.InternalErrorCode, "删除发布失败: "+err.Error())
		return
	}

	common.Success(c, nil)
}

// GetReleaseStats 获取发布统计
func (rc *ReleaseController) GetReleaseStats(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	stats, err := rc.releaseService.GetReleaseStats(c.Request.Context(), tenantID)
	if err != nil {
		rc.logger.Errorw("Get release stats failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取发布统计失败: "+err.Error())
		return
	}

	common.Success(c, stats)
}
