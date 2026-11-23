package rbac

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/internal/authz"
	"itsm-backend/middleware"

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
		common.ParamError(c, "status 取值非法，仅支持 active/inactive")
		return "", false
	}
	return dto.RoleStatusFromActive(active), true
}

// CreateRole creates a new role

// @Summary 创建角色
// @Description 在当前租户新建角色；名称需租户内唯一。权限码 role:write。
// @Tags 角色与权限管理
// @Produce json
// @Param body body dto.CreateRoleRequest true "角色创建请求"
// @Success 200 {object} common.Response{data=dto.RoleResponse}
// @Failure 400 {object} common.Response "参数错误"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 403 {object} common.Response "缺少 role:write 权限"
// @Failure 409 {object} common.Response "角色已存在"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/roles [post]
func (h *Handler) CreateRole(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}

	normalizedStatus, ok := normalizeRoleStatus(c, req.Status)
	if !ok {
		return
	}
	req.Status = normalizedStatus

	role, err := h.roleService.CreateRole(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, role)
}

// GetRole gets a role by ID

// @Summary 获取角色详情
// @Description 按 ID 读取当前租户角色详情。权限码 role:read。
// @Tags 角色与权限管理
// @Produce json
// @Param id path int true "角色ID"
// @Success 200 {object} common.Response{data=dto.RoleResponse}
// @Failure 400 {object} common.Response "无效ID"
// @Failure 403 {object} common.Response "缺少 role:read 权限"
// @Failure 404 {object} common.Response "角色不存在"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/roles/{id} [get]
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

// @Summary 列出角色
// @Description 分页列出当前租户角色，按权限码 role:read 守卫。
// @Tags 角色与权限管理
// @Produce json
// @Success 200 {object} common.Response "data 含 items/total/page/pageSize"
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 403 {object} common.Response "缺少 role:read 权限"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/roles [get]
func (h *Handler) ListRoles(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
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
		common.FailWithErr(c, err, "操作失败")
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

// @Summary 更新角色
// @Description 按 ID 更新当前租户角色。权限码 role:write。
// @Tags 角色与权限管理
// @Produce json
// @Param id path int true "角色ID"
// @Param body body dto.UpdateRoleRequest true "角色更新请求"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "参数错误"
// @Failure 403 {object} common.Response "缺少 role:write 权限"
// @Failure 404 {object} common.Response "角色不存在"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/roles/{id} [put]
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
		common.ParamErrorWithErr(c, err, "参数错误")
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
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, role)
}

// DeleteRole deletes a role

// @Summary 删除角色
// @Description 按 ID 删除当前租户角色。权限码 role:delete，内置角色通常禁止删除。
// @Tags 角色与权限管理
// @Produce json
// @Param id path int true "角色ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "无效ID"
// @Failure 403 {object} common.Response "缺少 role:delete 权限"
// @Failure 404 {object} common.Response "角色不存在"
// @Failure 409 {object} common.Response "内置角色或仍被引用"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/roles/{id} [delete]
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

// @Summary 为角色分配权限
// @Description 整体替换角色的权限码集合；提交的是全量列表而非增量。权限码 role:write。
// @Tags 角色与权限管理
// @Produce json
// @Param id path int true "角色ID"
// @Param body body dto.AssignPermissionsRequest true "权限分配请求"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "参数错误"
// @Failure 403 {object} common.Response "缺少 role:write 权限"
// @Failure 404 {object} common.Response "角色不存在"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/roles/{id}/permissions [post]
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
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}

	err = h.roleService.AssignPermissions(c.Request.Context(), id, req.PermissionIDs, tenantID)
	if err != nil {
		// R1: cross-tenant/invalid permission IDs return 400 not 500
		if errors.Is(err, ErrPermissionNotInTenant) {
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

// @Summary 创建权限定义
// @Description 新建权限码定义；复用 role:write 权限守卫。
// @Tags 角色与权限管理
// @Produce json
// @Param body body dto.CreatePermissionRequest true "权限创建请求"
// @Success 200 {object} common.Response{data=dto.PermissionResponse}
// @Failure 400 {object} common.Response "参数错误"
// @Failure 403 {object} common.Response "缺少 role:write 权限"
// @Failure 409 {object} common.Response "权限码已存在"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/permissions [post]
func (h *Handler) CreatePermission(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}

	var req dto.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
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

// @Summary 列出权限定义
// @Description 列出当前租户可见的权限定义。权限码 permission:read。
// @Tags 角色与权限管理
// @Produce json
// @Success 200 {object} common.Response{data=dto.PermissionListResponse}
// @Failure 401 {object} common.Response "租户上下文缺失"
// @Failure 403 {object} common.Response "缺少 permission:read 权限"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/permissions [get]
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

	permItems := make([]*dto.PermissionResponse, len(perms))
	copy(permItems, perms)

	// 权限字典是不分页的完整集合：只返回 {items,total}，不伪造 page/pageSize/totalPages；
	// 也不再同时返回 permissions 字符串别名（码值在 items[].code）。
	common.Success(c, &dto.PermissionListResponse{Items: permItems, Total: len(permItems)})
}

// parsePermissionID 统一解析路径里的权限 ID，避免每个 handler 各写一套。
func parsePermissionID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ParamError(c, "无效的权限ID")
		return 0, false
	}
	return id, true
}

// GetPermission 返回单条权限定义（含运行期策略）。

// @Summary 获取权限详情
// @Description 按 ID 读取权限定义。权限码 permission:read。
// @Tags 角色与权限管理
// @Produce json
// @Param id path int true "权限ID"
// @Success 200 {object} common.Response{data=dto.PermissionResponse}
// @Failure 400 {object} common.Response "无效ID"
// @Failure 403 {object} common.Response "缺少 permission:read 权限"
// @Failure 404 {object} common.Response "权限不存在"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/permissions/{id} [get]
func (h *Handler) GetPermission(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}
	id, ok := parsePermissionID(c)
	if !ok {
		return
	}
	perm, err := h.permissionService.GetPermission(c.Request.Context(), id, tenantID)
	if err != nil {
		common.NotFoundWithErr(c, err, "resource not found")
		return
	}
	common.Success(c, perm)
}

// UpdatePermission 整体替换一条权限的名称/描述/运行期策略。
// 策略是安全配置：非法组合（例如允许 AI 执行高危操作却关闭审批）在 service 层拒绝。

// @Summary 更新权限定义
// @Description 按 ID 更新权限定义（含运行期策略：风险等级/主体授权/审批强度），改完即时生效。权限码 role:write。
// @Tags 角色与权限管理
// @Produce json
// @Param id path int true "权限ID"
// @Param body body dto.UpdatePermissionRequest true "权限更新请求"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "参数错误"
// @Failure 403 {object} common.Response "缺少 role:write 权限"
// @Failure 404 {object} common.Response "权限不存在"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/permissions/{id} [put]
func (h *Handler) UpdatePermission(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}
	id, ok := parsePermissionID(c)
	if !ok {
		return
	}
	var req dto.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	perm, err := h.permissionService.UpdatePermission(c.Request.Context(), id, tenantID, &req)
	if err != nil {
		// 策略闭集校验失败是管理员输入问题，不是服务故障：
		// 500/5001 会让前端把「填错」显示成「服务异常」。
		switch {
		case errors.Is(err, authz.ErrPolicyInvalid):
			h.logger.Warnw("invalid permission policy rejected",
				"permission_id", id, "tenant_id", tenantID, "error", err)
			common.ParamErrorWithErr(c, err, "权限策略非法")
		case ent.IsNotFound(err):
			common.NotFoundWithErr(c, err, "权限不存在")
		default:
			common.FailWithErr(c, err, "操作失败")
		}
		return
	}
	common.Success(c, perm)
}

// DeletePermission 停用权限（不物理删除，保留历史 role_permissions 引用）。

// @Summary 删除权限定义
// @Description 按 ID 删除权限定义；仍被角色引用的权限可能被拒绝。权限码 role:write。
// @Tags 角色与权限管理
// @Produce json
// @Param id path int true "权限ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "无效ID"
// @Failure 403 {object} common.Response "缺少 role:write 权限"
// @Failure 404 {object} common.Response "权限不存在"
// @Failure 409 {object} common.Response "仍被角色引用"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/permissions/{id} [delete]
func (h *Handler) DeletePermission(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil || tenantID == 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return
	}
	id, ok := parsePermissionID(c)
	if !ok {
		return
	}
	if err := h.permissionService.DeletePermission(c.Request.Context(), id, tenantID); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, gin.H{"message": "权限已停用"})
}

// GetPermissionCatalogMeta 返回展示口径闭集，前端据此渲染筛选/标签，不再内置词表。

// @Summary 获取权限目录元数据
// @Description 返回权限码目录的分组/资源等元数据，供前端渲染权限选择树。权限码 permission:read。
// @Tags 角色与权限管理
// @Produce json
// @Success 200 {object} common.Response
// @Failure 403 {object} common.Response "缺少 permission:read 权限"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/permissions/meta [get]
func (h *Handler) GetPermissionCatalogMeta(c *gin.Context) {
	common.Success(c, h.permissionService.CatalogMeta())
}

// InitDefaultPermissions initializes default permissions

// @Summary 初始化默认权限
// @Description 幂等初始化平台默认权限码目录；已存在的不重复写入。权限码 role:write。
// @Tags 角色与权限管理
// @Produce json
// @Success 200 {object} common.Response
// @Failure 403 {object} common.Response "缺少 role:write 权限"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/permissions/init [post]
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

// ListMenus lists menus for a tenant.
// Super-admin may pass ?tenantId= to inspect another tenant's menu configuration.

// @Summary 列出菜单
// @Description 分页列出当前租户菜单树。权限码 system_config:read。
// @Tags 菜单与导航
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "页长" default(20)
// @Success 200 {object} common.Response{data=dto.MenuListResponse}
// @Failure 403 {object} common.Response "缺少 system_config:read 权限"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/menus [get]
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

	if override := c.Query("tenantId"); override != "" {
		role, _ := c.Get("role")
		if role != "super_admin" {
			common.Fail(c, common.ForbiddenCode, "仅超级管理员可查看其他租户菜单")
			return
		}
		overrideID, err := strconv.Atoi(override)
		if err != nil || overrideID <= 0 {
			common.Fail(c, common.ParamErrorCode, "无效的租户ID")
			return
		}
		tenantID = overrideID
	}

	menus, err := h.menuService.ListMenus(c.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(c, err, "查询菜单失败")
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

// @Summary 获取菜单详情
// @Description 按 ID 读取菜单项。权限码 system_config:read。
// @Tags 菜单与导航
// @Produce json
// @Param id path int true "菜单ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "无效ID"
// @Failure 403 {object} common.Response "缺少 system_config:read 权限"
// @Failure 404 {object} common.Response "菜单不存在"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/menus/{id} [get]
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
		common.RespondError(c, err, "查询菜单失败")
		return
	}

	common.Success(c, menu)
}

// CreateMenu creates a new menu

// @Summary 创建菜单
// @Description 新建菜单项。权限码 system_config:write。
// @Tags 菜单与导航
// @Produce json
// @Param body body dto.CreateMenuRequest true "菜单创建请求"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "参数错误"
// @Failure 403 {object} common.Response "缺少 system_config:write 权限"
// @Failure 409 {object} common.Response "菜单标识已存在"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/menus [post]
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
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	menu, err := h.menuService.CreateMenu(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.RespondError(c, err, "创建菜单失败")
		return
	}

	common.Success(c, menu)
}

// UpdateMenu updates a menu

// @Summary 更新菜单
// @Description 按 ID 更新菜单项。权限码 system_config:write。
// @Tags 菜单与导航
// @Produce json
// @Param id path int true "菜单ID"
// @Param body body dto.UpdateMenuRequest true "菜单更新请求"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "参数错误"
// @Failure 403 {object} common.Response "缺少 system_config:write 权限"
// @Failure 404 {object} common.Response "菜单不存在"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/menus/{id} [put]
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
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	menu, err := h.menuService.UpdateMenu(c.Request.Context(), id, &req, tenantID)
	if err != nil {
		common.RespondError(c, err, "更新菜单失败")
		return
	}

	common.Success(c, menu)
}

// DeleteMenu deletes a menu

// @Summary 删除菜单
// @Description 按 ID 删除菜单项。权限码 system_config:write。
// @Tags 菜单与导航
// @Produce json
// @Param id path int true "菜单ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "无效ID"
// @Failure 403 {object} common.Response "缺少 system_config:write 权限"
// @Failure 404 {object} common.Response "菜单不存在"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/menus/{id} [delete]
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
		common.RespondError(c, err, "删除菜单失败")
		return
	}

	common.Success(c, nil)
}

// GetUserMenus gets menus visible to the current user

// @Summary 获取当前用户菜单
// @Description 按当前用户角色过滤后返回可见菜单（前端 layout 动态路由数据源）。仅需认证，不按菜单细粒度权限守卫。
// @Tags 菜单与导航
// @Produce json
// @Success 200 {object} common.Response
// @Failure 401 {object} common.Response "未认证"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/auth/menus [get]
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
		common.RespondError(c, err, "查询用户菜单失败")
		return
	}

	common.Success(c, menus)
}

// ExportMenus exports menus in seeder-compatible format (parentPath instead of parentId).

// @Summary 导出菜单
// @Description 导出当前租户菜单配置为可导入的结构化 JSON。权限码 system_config:read。
// @Tags 菜单与导航
// @Produce json
// @Success 200 {object} common.Response
// @Failure 403 {object} common.Response "缺少 system_config:read 权限"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/menus/export [get]
func (h *Handler) ExportMenus(c *gin.Context) {
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

	items, err := h.menuService.ExportMenus(c.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(c, err, "导出菜单失败")
		return
	}

	common.Success(c, items)
}

// InitDefaultMenus seeds missing menus from the canonical baseline and returns a diff report.

// @Summary 初始化默认菜单
// @Description 幂等初始化平台默认菜单树；已存在的不重复写入。权限码 system_config:write。
// @Tags 菜单与导航
// @Produce json
// @Success 200 {object} common.Response
// @Failure 403 {object} common.Response "缺少 system_config:write 权限"
// @Failure 500 {object} common.Response "后端故障"
// @Security BearerAuth
// @Router /api/v1/menus/init [post]
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

	diff, err := h.menuService.InitMenusFromBaseline(c.Request.Context(), tenantID)
	if err != nil {
		common.RespondError(c, err, "初始化菜单失败")
		return
	}

	common.Success(c, diff)
}
