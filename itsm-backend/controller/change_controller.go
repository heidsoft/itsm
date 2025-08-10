package controller

import (
	"net/http"
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ChangeController 变更管理控制器
type ChangeController struct {
	logger        *zap.SugaredLogger
	changeService *service.ChangeService
}

// NewChangeController 创建变更管理控制器
func NewChangeController(logger *zap.SugaredLogger, changeService *service.ChangeService) *ChangeController {
	return &ChangeController{
		logger:        logger,
		changeService: changeService,
	}
}

// ListChanges 获取变更列表
func (cc *ChangeController) ListChanges(c *gin.Context) {
	// 获取租户信息
	tenantID := middleware.GetTenantID(c)
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	status := c.Query("status")
	search := c.Query("search")

	// 调用服务
	changes, err := cc.changeService.ListChanges(c.Request.Context(), tenantID, page, pageSize, status, search)
	if err != nil {
		cc.logger.Errorw("List changes failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取变更列表失败: "+err.Error())
		return
	}

	common.Success(c, changes)
}

// CreateChange 创建变更
func (cc *ChangeController) CreateChange(c *gin.Context) {
	// 获取租户信息
	tenantID := middleware.GetTenantID(c)
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	// 获取用户信息
	userID := middleware.GetUserID(c)
	if userID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	// 解析请求体
	var req dto.CreateChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	// 调用服务
	change, err := cc.changeService.CreateChange(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		cc.logger.Errorw("Create change failed", "error", err, "tenant_id", tenantID, "user_id", userID)
		common.Fail(c, common.InternalErrorCode, "创建变更失败: "+err.Error())
		return
	}

	common.Success(c, change)
}

// GetChange 获取变更详情
func (cc *ChangeController) GetChange(c *gin.Context) {
	// 获取租户信息
	tenantID := middleware.GetTenantID(c)
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	// 获取变更ID
	changeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的变更ID")
		return
	}

	// 调用服务
	change, err := cc.changeService.GetChange(c.Request.Context(), changeID, tenantID)
	if err != nil {
		if err.Error() == "change not found" {
			common.Fail(c, common.NotFoundCode, "变更不存在")
			return
		}
		cc.logger.Errorw("Get change failed", "error", err, "change_id", changeID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取变更详情失败: "+err.Error())
		return
	}

	common.Success(c, change)
}

// UpdateChange 更新变更
func (cc *ChangeController) UpdateChange(c *gin.Context) {
	// 获取租户信息
	tenantID := middleware.GetTenantID(c)
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	// 获取变更ID
	changeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的变更ID")
		return
	}

	// 解析请求体
	var req dto.UpdateChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	// 调用服务
	change, err := cc.changeService.UpdateChange(c.Request.Context(), changeID, &req, tenantID)
	if err != nil {
		if err.Error() == "change not found" {
			common.Fail(c, common.NotFoundCode, "变更不存在")
			return
		}
		cc.logger.Errorw("Update change failed", "error", err, "change_id", changeID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "更新变更失败: "+err.Error())
		return
	}

	common.Success(c, change)
}

// DeleteChange 删除变更
func (cc *ChangeController) DeleteChange(c *gin.Context) {
	// 获取租户信息
	tenantID := middleware.GetTenantID(c)
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	// 获取变更ID
	changeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的变更ID")
		return
	}

	// 调用服务
	err = cc.changeService.DeleteChange(c.Request.Context(), changeID, tenantID)
	if err != nil {
		if err.Error() == "change not found" {
			common.Fail(c, common.NotFoundCode, "变更不存在")
			return
		}
		cc.logger.Errorw("Delete change failed", "error", err, "change_id", changeID, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "删除变更失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{"message": "变更删除成功"})
}

// GetChangeStats 获取变更统计
func (cc *ChangeController) GetChangeStats(c *gin.Context) {
	// 获取租户信息
	tenantID := middleware.GetTenantID(c)
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	// 调用服务
	stats, err := cc.changeService.GetChangeStats(c.Request.Context(), tenantID)
	if err != nil {
		cc.logger.Errorw("Get change stats failed", "error", err, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "获取变更统计失败: "+err.Error())
		return
	}

	common.Success(c, stats)
}

// UpdateChangeStatus 更新变更状态
func (cc *ChangeController) UpdateChangeStatus(c *gin.Context) {
	// 获取租户信息
	tenantID := middleware.GetTenantID(c)
	if tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	// 获取变更ID
	changeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.BadRequestCode, "无效的变更ID")
		return
	}

	// 解析请求体
	var req struct {
		Status dto.ChangeStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.BadRequestCode, "请求参数错误: "+err.Error())
		return
	}

	// 调用服务
	err = cc.changeService.UpdateChangeStatus(c.Request.Context(), changeID, req.Status, tenantID)
	if err != nil {
		if err.Error() == "change not found" {
			common.Fail(c, common.NotFoundCode, "变更不存在")
			return
		}
		cc.logger.Errorw("Update change status failed", "error", err, "change_id", changeID, "status", req.Status, "tenant_id", tenantID)
		common.Fail(c, common.InternalErrorCode, "更新变更状态失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{"message": "变更状态更新成功"})
}
