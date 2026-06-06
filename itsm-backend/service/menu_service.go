package service

import (
	"context"
	"fmt"
	"strings"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/menu"
	"itsm-backend/ent/role"
	"itsm-backend/ent/user"
	"itsm-backend/middleware"

	"go.uber.org/zap"
)

// MenuService 菜单服务
type MenuService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewMenuService 创建菜单服务
func NewMenuService(client *ent.Client, logger *zap.SugaredLogger) *MenuService {
	return &MenuService{
		client: client,
		logger: logger,
	}
}

// CreateMenu 创建菜单
func (s *MenuService) CreateMenu(ctx context.Context, req *dto.CreateMenuRequest, tenantID int) (*dto.MenuDTO, error) {
	s.logger.Infow("Creating menu", "name", req.Name, "tenant_id", tenantID)

	// 验证父菜单是否存在（如果指定了父菜单）
	if req.ParentID != nil {
		parent, err := s.client.Menu.Query().
			Where(menu.ID(*req.ParentID), menu.TenantID(tenantID)).
			Only(ctx)
		if err != nil {
			return nil, fmt.Errorf("父菜单不存在: %w", err)
		}
		if parent == nil {
			return nil, fmt.Errorf("父菜单不存在")
		}
	}

	// 设置默认值
	sortOrder := req.SortOrder
	if sortOrder == 0 {
		sortOrder = 100
	}
	isVisible := true
	if req.IsVisible != nil {
		isVisible = *req.IsVisible
	}
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	menuEntity, err := s.client.Menu.Create().
		SetName(req.Name).
		SetPath(req.Path).
		SetTenantID(tenantID).
		SetSortOrder(sortOrder).
		SetIsVisible(isVisible).
		SetIsEnabled(isEnabled).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建菜单失败: %w", err)
	}

	// 设置可选字段
	if req.Icon != "" {
		menuEntity, err = menuEntity.Update().SetIcon(req.Icon).Save(ctx)
		if err != nil {
			s.logger.Warnw("Failed to set icon", "error", err)
		}
	}
	if req.ParentID != nil {
		menuEntity, err = menuEntity.Update().SetParentID(*req.ParentID).Save(ctx)
		if err != nil {
			s.logger.Warnw("Failed to set parent_id", "error", err)
		}
	}
	if req.PermissionCode != nil && *req.PermissionCode != "" {
		menuEntity, err = menuEntity.Update().SetPermissionCode(*req.PermissionCode).Save(ctx)
		if err != nil {
			s.logger.Warnw("Failed to set permission_code", "error", err)
		}
	}
	if req.Description != "" {
		menuEntity, err = menuEntity.Update().SetDescription(req.Description).Save(ctx)
		if err != nil {
			s.logger.Warnw("Failed to set description", "error", err)
		}
	}

	return s.toMenuDTO(menuEntity), nil
}

// GetMenu 获取菜单
func (s *MenuService) GetMenu(ctx context.Context, id int, tenantID int) (*dto.MenuDTO, error) {
	menuEntity, err := s.client.Menu.Query().
		Where(menu.ID(id), menu.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("菜单不存在: %w", err)
	}

	return s.toMenuDTO(menuEntity), nil
}

// ListMenus 获取菜单列表
func (s *MenuService) ListMenus(ctx context.Context, tenantID int) ([]*dto.MenuDTO, error) {
	menus, err := s.client.Menu.Query().
		Where(menu.TenantID(tenantID)).
		Order(ent.Asc(menu.FieldSortOrder)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取菜单列表失败: %w", err)
	}

	dtos := make([]*dto.MenuDTO, 0, len(menus))
	for _, m := range menus {
		dtos = append(dtos, s.toMenuDTO(m))
	}

	return dtos, nil
}

// UpdateMenu 更新菜单
func (s *MenuService) UpdateMenu(ctx context.Context, id int, req *dto.UpdateMenuRequest, tenantID int) (*dto.MenuDTO, error) {
	s.logger.Infow("Updating menu", "id", id, "tenant_id", tenantID)

	menuEntity, err := s.client.Menu.Query().
		Where(menu.ID(id), menu.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("菜单不存在: %w", err)
	}

	update := menuEntity.Update()

	if req.Name != nil && *req.Name != "" {
		update = update.SetName(*req.Name)
	}
	if req.Path != nil && *req.Path != "" {
		update = update.SetPath(*req.Path)
	}
	if req.Icon != nil {
		update = update.SetIcon(*req.Icon)
	}
	if req.ParentID != nil {
		update = update.SetParentID(*req.ParentID)
	}
	if req.PermissionCode != nil {
		update = update.SetPermissionCode(*req.PermissionCode)
	}
	if req.SortOrder != nil {
		update = update.SetSortOrder(*req.SortOrder)
	}
	if req.IsVisible != nil {
		update = update.SetIsVisible(*req.IsVisible)
	}
	if req.IsEnabled != nil {
		update = update.SetIsEnabled(*req.IsEnabled)
	}
	if req.Description != nil {
		update = update.SetDescription(*req.Description)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新菜单失败: %w", err)
	}

	return s.toMenuDTO(updated), nil
}

// DeleteMenu 删除菜单
func (s *MenuService) DeleteMenu(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting menu", "id", id, "tenant_id", tenantID)

	// 检查是否有子菜单
	children, err := s.client.Menu.Query().
		Where(menu.ParentID(id)).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("检查子菜单失败: %w", err)
	}
	if children > 0 {
		return fmt.Errorf("请先删除子菜单")
	}

	_, err = s.client.Menu.Delete().
		Where(menu.ID(id), menu.TenantID(tenantID)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除菜单失败: %w", err)
	}

	return nil
}

// GetUserMenus 获取用户可见菜单（根据用户角色和权限）
func (s *MenuService) GetUserMenus(ctx context.Context, userID int, tenantID int) (*dto.MenuTreeResponse, error) {
	s.logger.Infow("Getting user menus", "user_id", userID, "tenant_id", tenantID)

	// 获取用户角色
	userEntity, err := s.client.User.Query().
		Where(user.ID(userID)).
		WithRoles(func(q *ent.RoleQuery) {
			q.Select(role.FieldCode)
		}).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 获取用户的权限列表
	permissions := s.getUserPermissions(ctx, userEntity, tenantID)
	roleCodes := collectUserRoleCodes(userEntity)

	// 获取所有启用的菜单
	allMenus, err := s.client.Menu.Query().
		Where(
			menu.TenantID(tenantID),
			menu.IsEnabled(true),
			menu.IsVisible(true),
		).
		Order(ent.Asc(menu.FieldSortOrder)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取菜单失败: %w", err)
	}

	// 根据权限过滤菜单
	filteredMenus := s.filterMenusByPermission(allMenus, permissions, roleCodes)

	// 构建菜单树
	mainMenus, adminMenus := s.buildMenuTree(filteredMenus)

	return &dto.MenuTreeResponse{
		Main:  mainMenus,
		Admin: adminMenus,
	}, nil
}

// getUserPermissions 获取用户的权限列表
func (s *MenuService) getUserPermissions(ctx context.Context, userEntity *ent.User, tenantID int) map[string]bool {
	permissions := make(map[string]bool)

	// 超级管理员拥有所有权限
	if userEntity.Role == "super_admin" {
		permissions["*"] = true
		return permissions
	}

	// 从用户直接角色获取权限
	if userEntity.Role != "" {
		rolePerms := middleware.RolePermissions[string(userEntity.Role)]
		for _, p := range rolePerms {
			key := p.Resource + ":" + p.Action
			permissions[key] = true
			// 也添加通配符权限
			if p.Action == "*" {
				permissions[p.Resource+":*"] = true
			}
		}
	}

	// 从数据库获取角色的权限（多对多关系）
	roleCodes := make([]string, 0)
	if userEntity.Edges.Roles != nil {
		for _, r := range userEntity.Edges.Roles {
			roleCodes = append(roleCodes, r.Code)
		}
	}

	// 如果用户有额外的角色，获取角色权限
	if len(roleCodes) > 0 {
		for _, roleCode := range roleCodes {
			rolePerms := middleware.RolePermissions[roleCode]
			for _, p := range rolePerms {
				key := p.Resource + ":" + p.Action
				permissions[key] = true
				// 也添加通配符权限
				if p.Action == "*" {
					permissions[p.Resource+":*"] = true
				}
			}
		}
	}

	return permissions
}

// filterMenusByPermission 根据权限过滤菜单
func (s *MenuService) filterMenusByPermission(menus []*ent.Menu, permissions map[string]bool, roleCodes map[string]bool) []*ent.Menu {
	filtered := make([]*ent.Menu, 0)

	for _, m := range menus {
		if shouldRestrictMenuForRole(m.Path, roleCodes) {
			continue
		}

		// 如果没有权限要求，或者用户有权限，则包含
		if m.PermissionCode == "" {
			filtered = append(filtered, m)
			continue
		}

		permCode := m.PermissionCode
		// 检查精确权限
		if permissions[permCode] {
			filtered = append(filtered, m)
			continue
		}

		// 检查通配符权限（如 ticket:* 可以访问 ticket:read）
		parts := splitPermission(permCode)
		if len(parts) == 2 {
			wildcard := parts[0] + ":*"
			if permissions[wildcard] || permissions["*"] {
				filtered = append(filtered, m)
				continue
			}

			// 尝试 Action 别名映射（如 dashboard:view → dashboard:read）
			resolvedAction := resolveActionAliases(parts[1])
			if resolvedAction != parts[1] {
				// 使用别名后的权限代码进行精确匹配
				aliasPermCode := parts[0] + ":" + resolvedAction
				if permissions[aliasPermCode] {
					filtered = append(filtered, m)
					continue
				}
				// 尝试别名后的通配符权限
				aliasWildcard := parts[0] + ":*"
				if permissions[aliasWildcard] {
					filtered = append(filtered, m)
					continue
				}
			}

			// 尝试 Resource 别名映射（如 service:read → service_catalog:read）
			resolvedResource := resolveResourceAliases(parts[0])
			resolvedActionForResource := resolveActionAliases(parts[1])
			if resolvedResource != parts[0] || resolvedActionForResource != parts[1] {
				// 使用别名后的权限代码进行精确匹配
				finalPermCode := resolvedResource + ":" + resolvedActionForResource
				if permissions[finalPermCode] {
					filtered = append(filtered, m)
					continue
				}
				// 尝试别名后的通配符权限
				finalWildcard := resolvedResource + ":*"
				if permissions[finalWildcard] || permissions["*"] {
					filtered = append(filtered, m)
					continue
				}
			}
		}
	}

	return filtered
}

func collectUserRoleCodes(userEntity *ent.User) map[string]bool {
	roleCodes := make(map[string]bool)

	if userEntity.Role != "" {
		roleCodes[strings.ToLower(string(userEntity.Role))] = true
	}

	if userEntity.Edges.Roles != nil {
		for _, r := range userEntity.Edges.Roles {
			if r.Code != "" {
				roleCodes[strings.ToLower(r.Code)] = true
			}
		}
	}

	return roleCodes
}

func shouldRestrictMenuForRole(path string, roleCodes map[string]bool) bool {
	if path == "" || isElevatedMenuRole(roleCodes) {
		return false
	}

	restrictedPrefixes := []string{
		"/admin",
		"/enterprise",
		"/system",
		"/workflow",
	}

	for _, prefix := range restrictedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

func isElevatedMenuRole(roleCodes map[string]bool) bool {
	elevatedRoles := []string{
		"super_admin",
		"admin",
		"manager",
	}

	for _, roleCode := range elevatedRoles {
		if roleCodes[roleCode] {
			return true
		}
	}

	return false
}

// actionAliasMap 权限 Action 别名映射：前端使用的 action → 后端定义的 action
var actionAliasMap = map[string]string{
	// 基础别名
	"view":   "read",  // view 是 read 的别名
	"use":    "read",  // use 是 read 的别名
	"manage": "admin", // manage 是 admin 的别名
	"create": "write", // create 是 write 的别名
	"update": "write", // update 是 write 的别名
	// 扩展别名（数据库中使用的权限action）
	"approve": "admin", // approve 审批权限等同于 admin
	"analyze": "read",  // analyze 分析权限等同于 read
	"audit":   "read",  // audit 审计权限等同于 read
	"config":  "write", // config 配置权限等同于 write
	"request": "read",  // request 请求权限等同于 read
	"access":  "read",  // access 访问权限等同于 read
}

// resourceAliasMap 权限 Resource 别名映射：前端使用的 resource → 后端定义的 resource
var resourceAliasMap = map[string]string{
	// 基础别名
	"service":  "service_catalog", // service 是 service_catalog 的别名
	"workflow": "bpmn",            // workflow 是 bpmn 的别名
	"report":   "report",          // report 没有对应资源，保持不变
	// 扩展别名（数据库中使用的权限resource）
	"helpdesk":   "ticket",        // helpdesk 等同于 ticket
	"approval":   "permission",    // approval 等同于 permission
	"department": "org",           // department 等同于 org
	"team":       "org",           // team 等同于 org
	"system":     "system_config", // system 等同于 system_config
	"tenant":     "org",           // tenant 等同于 org
}

// resolveActionAliases 解析 Action 别名，返回后端定义的 Action
func resolveActionAliases(action string) string {
	if resolved, ok := actionAliasMap[action]; ok {
		return resolved
	}
	return action
}

// resolveResourceAliases 解析 Resource 别名，返回后端定义的 Resource
func resolveResourceAliases(resource string) string {
	if resolved, ok := resourceAliasMap[resource]; ok {
		return resolved
	}
	return resource
}

// splitPermission 拆分权限代码
func splitPermission(permCode string) []string {
	for i := 0; i < len(permCode); i++ {
		if permCode[i] == ':' {
			return []string{permCode[:i], permCode[i+1:]}
		}
	}
	return []string{permCode}
}

// buildMenuTree 构建菜单树
func (s *MenuService) buildMenuTree(menus []*ent.Menu) (mainMenus, adminMenus []dto.MenuDTO) {
	// 构建子菜单映射
	childrenMap := make(map[int][]*ent.Menu)
	for _, m := range menus {
		if m.ParentID != nil {
			childrenMap[*m.ParentID] = append(childrenMap[*m.ParentID], m)
		}
	}

	// 递归构建菜单树
	var buildChildren func(parentID int) []dto.MenuDTO
	buildChildren = func(parentID int) []dto.MenuDTO {
		result := make([]dto.MenuDTO, 0)
		children, ok := childrenMap[parentID]
		if !ok {
			return result
		}

		for _, child := range children {
			childDTO := *s.toMenuDTO(child)
			childDTO.Children = buildChildren(child.ID)
			result = append(result, childDTO)
		}

		return result
	}

	// 找出顶级菜单（parent_id 为 nil）
	rootMenus := make([]*ent.Menu, 0)
	for _, m := range menus {
		if m.ParentID == nil {
			rootMenus = append(rootMenus, m)
		}
	}

	// 分离主菜单和管理菜单
	for _, m := range rootMenus {
		dto := *s.toMenuDTO(m)
		dto.Children = buildChildren(m.ID)

		// 根据路径判断是管理菜单还是主菜单
		if isAdminMenu(m.Path) {
			adminMenus = append(adminMenus, dto)
		} else {
			mainMenus = append(mainMenus, dto)
		}
	}

	return mainMenus, adminMenus
}

// isAdminMenu 判断是否为管理菜单
func isAdminMenu(path string) bool {
	adminPaths := []string{"/admin", "/system", "/workflow"}
	for _, p := range adminPaths {
		if len(path) >= len(p) && path[:len(p)] == p {
			return true
		}
	}
	return false
}

// toMenuDTO 转换为 DTO
func (s *MenuService) toMenuDTO(menuEntity *ent.Menu) *dto.MenuDTO {
	dto := &dto.MenuDTO{
		ID:        menuEntity.ID,
		Name:      menuEntity.Name,
		Path:      menuEntity.Path,
		SortOrder: menuEntity.SortOrder,
		TenantID:  menuEntity.TenantID,
		IsVisible: menuEntity.IsVisible,
		IsEnabled: menuEntity.IsEnabled,
	}

	if menuEntity.Icon != "" {
		dto.Icon = menuEntity.Icon
	}
	if menuEntity.ParentID != nil {
		dto.ParentID = menuEntity.ParentID
	}
	if menuEntity.PermissionCode != "" {
		dto.PermissionCode = &menuEntity.PermissionCode
	}
	if menuEntity.Description != "" {
		dto.Description = menuEntity.Description
	}

	return dto
}
