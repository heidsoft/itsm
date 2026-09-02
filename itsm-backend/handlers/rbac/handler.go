package rbac

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler HTTP handler for RBAC domain (role, permission, menu)
type Handler struct {
	roleService       *service.RoleService
	permissionService *service.PermissionService
	menuService      *service.MenuService
	logger           *zap.SugaredLogger
}

// NewHandler creates a new RBAC handler
func NewHandler(roleService *service.RoleService, permissionService *service.PermissionService, menuService *service.MenuService, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		roleService:       roleService,
		permissionService: permissionService,
		menuService:      menuService,
		logger:           logger,
	}
}

// =============================================================================
// Role Handlers
// =============================================================================

// CreateRole creates a new role
func (h *Handler) CreateRole(c *gin.Context) {
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

	role, err := h.roleService.CreateRole(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, role)
}

// GetRole gets a role by ID
func (h *Handler) GetRole(c *gin.Context) {
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

	role, err := h.roleService.GetRole(c.Request.Context(), id, tenantID)
	if err != nil {
		common.NotFoundWithErr(c, err, "resource not found")
		return
	}

	common.Success(c, role)
}

// ListRoles lists roles with pagination
func (h *Handler) ListRoles(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")

	roles, total, err := h.roleService.ListRoles(c.Request.Context(), tenantID, page, pageSize, search)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
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

// UpdateRole updates a role
func (h *Handler) UpdateRole(c *gin.Context) {
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

	role, err := h.roleService.UpdateRole(c.Request.Context(), id, &req, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, role)
}

// DeleteRole deletes a role
func (h *Handler) DeleteRole(c *gin.Context) {
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

	err = h.roleService.DeleteRole(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// AssignPermissions assigns permissions to a role
func (h *Handler) AssignPermissions(c *gin.Context) {
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

	err = h.roleService.AssignPermissions(c.Request.Context(), id, req.PermissionIDs, tenantID)
	if err != nil {
		// R1: cross-tenant/invalid permission IDs return 400 not 500
		if errors.Is(err, service.ErrPermissionNotInTenant) {
			h.logger.Warnw("Cross-tenant permission assignment rejected",
				"role_id", id, "tenant_id", tenantID, "error", err)
			common.ParamErrorWithErr(c, err, "请求参数错误")
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, nil)
}

// =============================================================================
// Permission Handlers
// =============================================================================

// CreatePermission creates a new permission
func (h *Handler) CreatePermission(c *gin.Context) {
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

	perm, err := h.permissionService.CreatePermission(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, perm)
}

// ListPermissions lists permissions
func (h *Handler) ListPermissions(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	resource := c.Query("resource")

	perms, err := h.permissionService.ListPermissions(c.Request.Context(), tenantID, resource)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	// Keep the string array for legacy callers and expose catalog details for real RBAC assignment
	permStrings := make([]string, len(perms))
	permItems := make([]*dto.PermissionResponse, len(perms))
	for i, p := range perms {
		permStrings[i] = fmt.Sprintf("%s:%s", p.Resource, p.Action)
		permItems[i] = p
	}

	common.Success(c, gin.H{"permissions": permStrings, "items": permItems})
}

// InitDefaultPermissions initializes default permissions
func (h *Handler) InitDefaultPermissions(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	err = h.permissionService.InitDefaultPermissions(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{"message": "权限初始化成功"})
}

// =============================================================================
// Menu Handlers
// =============================================================================

// ListMenus lists menus for a tenant
func (h *Handler) ListMenus(c *gin.Context) {
	tid, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}
	tenantID, ok2 := tid.(int)
	if !ok2 {
		common.Fail(c, common.UnauthorizedCode, "租户上下文类型错误")
		return
	}

	menus, err := h.menuService.ListMenus(c.Request.Context(), tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, dto.MenuListResponse{
		Menus: menus,
		Total: len(menus),
	})
}

// GetMenu gets a menu by ID
func (h *Handler) GetMenu(c *gin.Context) {
	tid, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}
	tenantID, ok2 := tid.(int)
	if !ok2 {
		common.Fail(c, common.UnauthorizedCode, "租户上下文类型错误")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的菜单ID")
		return
	}

	menu, err := h.menuService.GetMenu(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, menu)
}

// CreateMenu creates a new menu
func (h *Handler) CreateMenu(c *gin.Context) {
	tid, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}
	tenantID, ok2 := tid.(int)
	if !ok2 {
		common.Fail(c, common.UnauthorizedCode, "租户上下文类型错误")
		return
	}

	var req dto.CreateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	menu, err := h.menuService.CreateMenu(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, menu)
}

// UpdateMenu updates a menu
func (h *Handler) UpdateMenu(c *gin.Context) {
	tid, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}
	tenantID, ok2 := tid.(int)
	if !ok2 {
		common.Fail(c, common.UnauthorizedCode, "租户上下文类型错误")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的菜单ID")
		return
	}

	var req dto.UpdateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	menu, err := h.menuService.UpdateMenu(c.Request.Context(), id, &req, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, menu)
}

// DeleteMenu deletes a menu
func (h *Handler) DeleteMenu(c *gin.Context) {
	tid, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}
	tenantID, ok2 := tid.(int)
	if !ok2 {
		common.Fail(c, common.UnauthorizedCode, "租户上下文类型错误")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "无效的菜单ID")
		return
	}

	err = h.menuService.DeleteMenu(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, nil)
}

// GetUserMenus gets menus visible to the current user
func (h *Handler) GetUserMenus(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, common.AuthFailedCode, "用户未认证")
		return
	}
	userID := userIDVal.(int)

	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	tenantID := tenantIDVal.(int)

	menus, err := h.menuService.GetUserMenus(c.Request.Context(), userID, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, menus)
}

// InitDefaultMenus initializes default menus
func (h *Handler) InitDefaultMenus(c *gin.Context) {
	tid, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}
	tenantID, ok2 := tid.(int)
	if !ok2 {
		common.Fail(c, common.UnauthorizedCode, "租户上下文类型错误")
		return
	}

	// Check if menus already exist
	existing, err := h.menuService.ListMenus(c.Request.Context(), tenantID)
	if err == nil && len(existing) > 0 {
		common.Success(c, dto.MenuInitResponse{
			Message: "菜单已初始化",
			Count:   len(existing),
		})
		return
	}

	common.Success(c, dto.MenuInitResponse{
		Message: "请通过种子数据初始化菜单",
	})
}
