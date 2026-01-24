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

// RoleController 角色管理控制器
type RoleController struct {
	roleService *service.RoleService
	logger      *zap.SugaredLogger
}

// NewRoleController 创建角色控制器
func NewRoleController(roleService *service.RoleService, logger *zap.SugaredLogger) *RoleController {
	return &RoleController{
		roleService: roleService,
		logger:      logger,
	}
}

// CreateRole 创建角色
func (rc *RoleController) CreateRole(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	role, err := rc.roleService.CreateRole(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "创建角色失败: "+err.Error())
		return
	}

	common.Success(c, role)
}

// GetRole 获取角色详情
func (rc *RoleController) GetRole(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的角色ID")
		return
	}

	role, err := rc.roleService.GetRole(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundCode, err.Error())
		return
	}

	common.Success(c, role)
}

// ListRoles 获取角色列表
func (rc *RoleController) ListRoles(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	roles, total, err := rc.roleService.ListRoles(c.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "查询角色列表失败: "+err.Error())
		return
	}

	common.Success(c, gin.H{
		"data":  roles,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// UpdateRole 更新角色
func (rc *RoleController) UpdateRole(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的角色ID")
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	role, err := rc.roleService.UpdateRole(c.Request.Context(), id, &req, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "更新角色失败: "+err.Error())
		return
	}

	common.Success(c, role)
}

// DeleteRole 删除角色
func (rc *RoleController) DeleteRole(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的角色ID")
		return
	}

	err = rc.roleService.DeleteRole(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

// AssignPermissions 分配权限
func (rc *RoleController) AssignPermissions(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "无效的角色ID")
		return
	}

	var req dto.AssignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	err = rc.roleService.AssignPermissions(c.Request.Context(), id, req.PermissionIDs, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "分配权限失败: "+err.Error())
		return
	}

	common.Success(c, nil)
}

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

	common.Success(c, perms)
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
