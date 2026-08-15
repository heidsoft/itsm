package controller

import (
	"errors"
	"strconv"
	"time"

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
// @Summary 创建角色
// @Description 创建新的角色
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param request body dto.CreateRoleRequest true "角色信息"
// @Success 200 {object} common.Response
// @Router /api/v1/roles [post]
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
	search := c.Query("search")

	roles, total, err := rc.roleService.ListRoles(c.Request.Context(), tenantID, page, pageSize, search)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "查询角色列表失败: "+err.Error())
		return
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	roleItems := make([]dto.RoleDTO, 0, len(roles))
	for _, role := range roles {
		permissionCodes := make([]string, 0, len(role.Permissions))
		for _, permission := range role.Permissions {
			permissionCodes = append(permissionCodes, permission.Code)
		}

		status := "inactive"
		if role.IsActive {
			status = "active"
		}

		roleItems = append(roleItems, dto.RoleDTO{
			ID:          role.ID,
			Name:        role.Name,
			Code:        role.Code,
			Description: role.Description,
			Permissions: permissionCodes,
			Status:      status,
			IsSystem:    role.IsSystem,
			CreatedAt:   role.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   role.UpdatedAt.Format(time.RFC3339),
			TenantID:    role.TenantID,
			DataScope:   string(role.DataScope),
		})
	}

	common.Success(c, dto.RoleListResponse{
		Roles:      roleItems,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
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
		// R1：跨租户/不存在的权限 ID 属于调用方输入错误，返回 400 而非 500。
		if errors.Is(err, service.ErrPermissionNotInTenant) {
			rc.logger.Warnw("Cross-tenant permission assignment rejected",
				"role_id", id, "tenant_id", tenantID, "error", err)
			common.Fail(c, common.BadRequestCode, err.Error())
			return
		}
		common.Fail(c, common.InternalErrorCode, "分配权限失败: "+err.Error())
		return
	}

	common.Success(c, nil)
}
