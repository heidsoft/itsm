package rbac

import (
	"errors"
	"strconv"
	"strings"
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
	roleService       RoleService
	permissionService PermissionService
	menuService       MenuService
	logger            *zap.SugaredLogger
}

// NewHandler creates a new RBAC handler
func NewHandler(roleService RoleService, permissionService PermissionService, menuService MenuService, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		roleService:       roleService,
		permissionService: permissionService,
		menuService:       menuService,
		logger:            logger,
	}
}

// =============================================================================
// Role Handlers
// =============================================================================

// normalizeRoleStatus 校验并归一化启用状态取值（绑定层职责）。
// 空字符串表示调用方未提交该字段，原样返回让 service 走默认值；
// 非法取值必须拒绝为参数错误（1001），禁止静默落库为默认值造成「假成功」。
func normalizeRoleStatus(c *gin.Context, status string) (string, bool) {
	if strings.TrimSpace(status) == "" {
		return "", true
	}
	active, ok := dto.RoleActiveFromStatus(status)
	if !ok {
		common.ParamError(c, "status value is illegal, only active/inactive is supported")
		return "", false
	}
	return dto.RoleStatusFromActive(active), true
}

// CreateRole creates a new role
func (h *Handler) CreateRole(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access")
		return
	}

	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "Parameter error")
		return
	}

	normalizedStatus, ok := normalizeRoleStatus(c, req.Status)
	if !ok {
		return
	}
	req.Status = normalizedStatus

	role, err := h.roleService.CreateRole(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "Operation failed")
		return
	}

	common.Success(c, role)
}

// GetRole gets a role by ID
func (h *Handler) GetRole(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "Invalid role ID")
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
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access")
		return
	}

	pagination := common.GetPaginationFromQuery(c)

	status, ok := normalizeRoleStatus(c, c.Query("status"))
	if !ok {
		return
	}

	params := &dto.GetRolesParams{
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
		Status:   status,
		Search:   c.Query("search"),
	}

	roles, total, err := h.roleService.ListRoles(c.Request.Context(), tenantID, params)
	if err != nil {
		common.FailWithErr(c, err, "Operation failed")
		return
	}

	totalPages := 0
	if params.PageSize > 0 {
		totalPages = (total + params.PageSize - 1) / params.PageSize
	}

	roleItems := make([]dto.RoleDTO, 0, len(roles))
	for _, role := range roles {
		permissionCodes := make([]string, 0, len(role.Permissions))
		for _, permission := range role.Permissions {
			permissionCodes = append(permissionCodes, permission.Code)
		}

		roleItems = append(roleItems, dto.RoleDTO{
			ID:          role.ID,
			Name:        role.Name,
			Code:        role.Code,
			Description: role.Description,
			Permissions: permissionCodes,
			Status:      role.Status,
			IsSystem:    role.IsSystem,
			CreatedAt:   role.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   role.UpdatedAt.Format(time.RFC3339),
			TenantID:    role.TenantID,
			DataScope:   role.DataScope,
		})
	}

	common.Success(c, dto.RoleListResponse{
		Items:      roleItems,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	})
}

// UpdateRole updates a role
func (h *Handler) UpdateRole(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "Invalid role ID")
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "Parameter error")
		return
	}

	if req.Status != nil {
		normalized, ok := normalizeRoleStatus(c, *req.Status)
		if !ok {
			return
		}
		// 空串等同「未提交该字段」，必须清空指针，
		// 否则 service 会把 "" 映射成 is_active=false，静默禁用角色
		if normalized == "" {
			req.Status = nil
		} else {
			req.Status = &normalized
		}
	}

	role, err := h.roleService.UpdateRole(c.Request.Context(), id, &req, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "Operation failed")
		return
	}

	common.Success(c, role)
}

// DeleteRole deletes a role
func (h *Handler) DeleteRole(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "Invalid role ID")
		return
	}

	err = h.roleService.DeleteRole(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "Operation failed")
		return
	}

	common.Success(c, nil)
}

// AssignPermissions assigns permissions to a role
func (h *Handler) AssignPermissions(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ParamError(c, "Invalid role ID")
		return
	}

	var req dto.AssignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "Parameter error")
		return
	}

	err = h.roleService.AssignPermissions(c.Request.Context(), id, req.PermissionIDs, tenantID)
	if err != nil {
		// R1: cross-tenant/invalid permission IDs return 400 not 500
		if errors.Is(err, service.ErrPermissionNotInTenant) {
			h.logger.Warnw("Cross-tenant permission assignment rejected",
				"role_id", id, "tenant_id", tenantID, "error", err)
			common.ParamErrorWithErr(c, err, "Wrong request parameters")
			return
		}
		common.FailWithErr(c, err, "Operation failed")
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
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access")
		return
	}

	var req dto.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "Parameter error")
		return
	}

	perm, err := h.permissionService.CreatePermission(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "Operation failed")
		return
	}

	common.Success(c, perm)
}

// ListPermissions lists permissions
func (h *Handler) ListPermissions(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access")
		return
	}

	resource := c.Query("resource")

	perms, err := h.permissionService.ListPermissions(c.Request.Context(), tenantID, resource)
	if err != nil {
		common.FailWithErr(c, err, "Operation failed")
		return
	}

	permItems := make([]*dto.PermissionResponse, len(perms))
	copy(permItems, perms)

	// 权限字典是不分页的完整集合：只返回 {items,total}，不伪造 page/pageSize/totalPages；
	// 也不再同时返回 permissions 字符串别名（码值在 items[].code）。
	common.Success(c, &dto.PermissionListResponse{Items: permItems, Total: len(permItems)})
}

// InitDefaultPermissions initializes default permissions
func (h *Handler) InitDefaultPermissions(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access")
		return
	}

	err = h.permissionService.InitDefaultPermissions(c.Request.Context(), tenantID)
	if err != nil {
		common.FailWithErr(c, err, "Operation failed")
		return
	}

	common.Success(c, gin.H{"message": "Permission initialization successful"})
}

// =============================================================================
// Menu Handlers
// =============================================================================

// ListMenus lists menus for a tenant.
// Super-admin may pass ?tenantId= to inspect another tenant's menu configuration.
func (h *Handler) ListMenus(c *gin.Context) {
	tid, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access: Tenant information is missing")
		return
	}
	tenantID, ok2 := tid.(int)
	if !ok2 {
		common.Fail(c, common.UnauthorizedCode, "Wrong tenant context type")
		return
	}

	if override := c.Query("tenantId"); override != "" {
		role, _ := c.Get("role")
		if role != "super_admin" {
			common.Fail(c, common.ForbiddenCode, "Only super administrators can view other tenant menus")
			return
		}
		overrideID, err := strconv.Atoi(override)
		if err != nil || overrideID <= 0 {
			common.Fail(c, common.ParamErrorCode, "Invalid tenant ID")
			return
		}
		tenantID = overrideID
	}

	menus, err := h.menuService.ListMenus(c.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(c, err, "Failed to query menu")
		return
	}

	common.Success(c, dto.MenuListResponse{
		Items:      menus,
		Total:      len(menus),
		Page:       1,
		PageSize:   len(menus),
		TotalPages: 1,
	})
}

// GetMenu gets a menu by ID
func (h *Handler) GetMenu(c *gin.Context) {
	tid, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access: Tenant information is missing")
		return
	}
	tenantID, ok2 := tid.(int)
	if !ok2 {
		common.Fail(c, common.UnauthorizedCode, "Wrong tenant context type")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "Invalid menu ID")
		return
	}

	menu, err := h.menuService.GetMenu(c.Request.Context(), id, tenantID)
	if err != nil {
		common.RespondError(c, err, "Failed to query menu")
		return
	}

	common.Success(c, menu)
}

// CreateMenu creates a new menu
func (h *Handler) CreateMenu(c *gin.Context) {
	tid, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access: Tenant information is missing")
		return
	}
	tenantID, ok2 := tid.(int)
	if !ok2 {
		common.Fail(c, common.UnauthorizedCode, "Wrong tenant context type")
		return
	}

	var req dto.CreateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "Wrong request parameters")
		return
	}

	menu, err := h.menuService.CreateMenu(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.RespondError(c, err, "Failed to create menu")
		return
	}

	common.Success(c, menu)
}

// UpdateMenu updates a menu
func (h *Handler) UpdateMenu(c *gin.Context) {
	tid, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access: Tenant information is missing")
		return
	}
	tenantID, ok2 := tid.(int)
	if !ok2 {
		common.Fail(c, common.UnauthorizedCode, "Wrong tenant context type")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "Invalid menu ID")
		return
	}

	var req dto.UpdateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "Wrong request parameters")
		return
	}

	menu, err := h.menuService.UpdateMenu(c.Request.Context(), id, &req, tenantID)
	if err != nil {
		common.RespondError(c, err, "Failed to update menu")
		return
	}

	common.Success(c, menu)
}

// DeleteMenu deletes a menu
func (h *Handler) DeleteMenu(c *gin.Context) {
	tid, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access: Tenant information is missing")
		return
	}
	tenantID, ok2 := tid.(int)
	if !ok2 {
		common.Fail(c, common.UnauthorizedCode, "Wrong tenant context type")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "Invalid menu ID")
		return
	}

	err = h.menuService.DeleteMenu(c.Request.Context(), id, tenantID)
	if err != nil {
		common.RespondError(c, err, "Failed to delete menu")
		return
	}

	common.Success(c, nil)
}

// GetUserMenus gets menus visible to the current user
func (h *Handler) GetUserMenus(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, common.AuthFailedCode, "User not authenticated")
		return
	}
	userID := userIDVal.(int)

	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		common.Fail(c, common.AuthFailedCode, "Tenant information is missing")
		return
	}
	tenantID := tenantIDVal.(int)

	menus, err := h.menuService.GetUserMenus(c.Request.Context(), userID, tenantID)
	if err != nil {
		common.RespondError(c, err, "Failed to query user menu")
		return
	}

	common.Success(c, menus)
}

// ExportMenus exports menus in seeder-compatible format (parentPath instead of parentId).
func (h *Handler) ExportMenus(c *gin.Context) {
	tid, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access: Tenant information is missing")
		return
	}
	tenantID, ok2 := tid.(int)
	if !ok2 {
		common.Fail(c, common.UnauthorizedCode, "Wrong tenant context type")
		return
	}

	items, err := h.menuService.ExportMenus(c.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(c, err, "Failed to export menu")
		return
	}

	common.Success(c, items)
}

// InitDefaultMenus seeds missing menus from the canonical baseline and returns a diff report.
func (h *Handler) InitDefaultMenus(c *gin.Context) {
	tid, ok := c.Get("tenant_id")
	if !ok {
		common.Fail(c, common.UnauthorizedCode, "Unauthorized access: Tenant information is missing")
		return
	}
	tenantID, ok2 := tid.(int)
	if !ok2 {
		common.Fail(c, common.UnauthorizedCode, "Wrong tenant context type")
		return
	}

	diff, err := h.menuService.InitMenusFromBaseline(c.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(c, err, "Failed to initialize menu")
		return
	}

	common.Success(c, diff)
}
