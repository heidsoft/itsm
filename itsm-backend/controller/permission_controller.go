package controller

import (
	"fmt"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PermissionController 权限管理控制器
type PermissionController struct {
	permissionService *service.PermissionService
	logger            *zap.SugaredLogger
}

// NewPermissionController 创建权限控制器
func NewPermissionController(permissionService *service.PermissionService, logger *zap.SugaredLogger) *PermissionController {
	return &PermissionController{
		permissionService: permissionService,
		logger:            logger,
	}
}

// CreatePermission 创建权限
func (pc *PermissionController) CreatePermission(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	perm, err := pc.permissionService.CreatePermission(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "创建权限失败: "+err.Error())
		return
	}

	common.Success(c, perm)
}

// ListPermissions 获取权限列表
func (pc *PermissionController) ListPermissions(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	resource := c.Query("resource")

	perms, err := pc.permissionService.ListPermissions(c.Request.Context(), tenantID, resource)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "查询权限列表失败: "+err.Error())
		return
	}

	// Keep the string array for legacy callers and expose catalog details for real RBAC assignment.
	permStrings := make([]string, len(perms))
	permItems := make([]*dto.PermissionResponse, len(perms))
	for i, p := range perms {
		permStrings[i] = fmt.Sprintf("%s:%s", p.Resource, p.Action)
		permItems[i] = p
	}

	common.Success(c, gin.H{"permissions": permStrings, "items": permItems})
}

// InitDefaultPermissions 初始化默认权限
func (pc *PermissionController) InitDefaultPermissions(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	err = pc.permissionService.InitDefaultPermissions(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "初始化权限失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{"message": "权限初始化成功"})
}
