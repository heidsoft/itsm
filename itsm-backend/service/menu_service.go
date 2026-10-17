package service

import (
	"context"
	"fmt"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/menu"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/role"
	"itsm-backend/ent/rolepermission"
	"itsm-backend/ent/user"
	"itsm-backend/middleware"
	"itsm-backend/pkg/menubaseline"

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

	// 验证父菜单是否存在（如果指定了父菜单）。parentId 0 与未传同义，都表示顶级菜单。
	if req.ParentID != nil && *req.ParentID != 0 {
		exists, err := s.client.Menu.Query().
			Where(menu.ID(*req.ParentID), menu.TenantID(tenantID)).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("查询父菜单失败: %w", err)
		}
		if !exists {
			return nil, common.NewValidationError("上级菜单不存在或不属于当前租户", nil)
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

	// 同租户 (tenant, path) 查重给出友好 conflict；唯一索引 menu_tenant_id_path 兜底并发窗口
	exists, err := s.client.Menu.Query().
		Where(menu.PathEQ(req.Path), menu.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询菜单失败: %w", err)
	}
	if exists {
		return nil, common.NewBusinessError(common.ConflictCode, "同级路径菜单已存在", req.Path)
	}

	// 单次 INSERT 写入全部字段；权限码写失败必须向上抛错，否则菜单 PermissionCode 落空后
	// filterMenusByPermission 视为「无权限要求」→ 对全员可见（菜单越权）。
	create := s.client.Menu.Create().
		SetName(req.Name).
		SetPath(req.Path).
		SetTenantID(tenantID).
		SetSortOrder(sortOrder).
		SetIsVisible(isVisible).
		SetIsEnabled(isEnabled)
	if req.Icon != "" {
		create = create.SetIcon(req.Icon)
	}
	if req.ParentID != nil && *req.ParentID != 0 {
		create = create.SetParentID(*req.ParentID)
	}
	if req.PermissionCode != nil && *req.PermissionCode != "" {
		create = create.SetPermissionCode(*req.PermissionCode)
	}
	if req.Description != "" {
		create = create.SetDescription(req.Description)
	}
	menuEntity, err := create.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, common.NewBusinessError(common.ConflictCode, "同级路径菜单已存在", req.Path)
		}
		return nil, fmt.Errorf("创建菜单失败: %w", err)
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

// assertParentAssignable 校验上级菜单可指向：必须同租户、非自身、且不是自身的后代，
// 否则菜单树会成环，页面按 parentId 组装时会出现整棵子树消失。
// 菜单量级有限，一次读回全租户再做遍历，避免递归查库。
func (s *MenuService) assertParentAssignable(ctx context.Context, menuID int, parentID int, tenantID int) error {
	if parentID == menuID {
		return common.NewValidationError("上级菜单不能是当前菜单自身", nil)
	}

	menus, err := s.client.Menu.Query().Where(menu.TenantID(tenantID)).All(ctx)
	if err != nil {
		return fmt.Errorf("查询菜单失败: %w", err)
	}

	children := make(map[int][]int, len(menus))
	parentFound := false
	for _, m := range menus {
		if m.ID == parentID {
			parentFound = true
		}
		if m.ParentID != nil {
			children[*m.ParentID] = append(children[*m.ParentID], m.ID)
		}
	}
	if !parentFound {
		return common.NewValidationError("上级菜单不存在或不属于当前租户", nil)
	}

	visited := map[int]bool{menuID: true}
	stack := []int{menuID}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, childID := range children[current] {
			if childID == parentID {
				return common.NewValidationError("上级菜单不能是当前菜单的子菜单", nil)
			}
			if !visited[childID] {
				visited[childID] = true
				stack = append(stack, childID)
			}
		}
	}
	return nil
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
	// parentId 用 0 表示「设为顶级菜单」：JSON null 与未传都无法区分，
	// 只按指针判断会让页面上移除上级菜单的操作静默失效。
	if req.ParentID != nil {
		if *req.ParentID == 0 {
			update = update.ClearParentID()
		} else {
			if err := s.assertParentAssignable(ctx, id, *req.ParentID, tenantID); err != nil {
				return nil, err
			}
			update = update.SetParentID(*req.ParentID)
		}
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

	// 与 GetMenu/UpdateMenu 同一口径：未命中按 ent 的 not-found 上抛，由响应层统一映射成
	// 404/4004「资源不存在或已被删除」。删除不存在的菜单不能返回成功。
	_, err := s.client.Menu.Query().
		Where(menu.ID(id), menu.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("菜单不存在: %w", err)
	}

	// 检查是否有子菜单
	children, err := s.client.Menu.Query().
		Where(menu.ParentID(id), menu.TenantID(tenantID)).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("检查子菜单失败: %w", err)
	}
	if children > 0 {
		return common.NewBusinessError(common.ConflictCode, "请先删除子菜单", fmt.Sprintf("menu_id=%d children=%d", id, children))
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
		WithTenant().
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

	// 回填父项：子菜单通过权限过滤但父菜单未通过时，保留父菜单作为导航容器，
	// 否则 buildMenuTree 会因为找不到父节点而丢弃整棵子树。
	filteredMenus = s.backfillParentMenus(filteredMenus, allMenus)

	// 构建菜单树
	mainMenus, adminMenus := s.buildMenuTree(filteredMenus)

	// MSP 菜单要求服务商成员身份，与 MSPMiddleware 同口径：msp:read 是 12 个内置角色
	// 共有的读码，只按权限码过滤会把非服务商成员导航进一个必然 403 的入口。
	// 部署模式关闭 MSP 时同样隐藏。隐藏菜单不是授权，接口仍逐请求把关。
	tenantType := ""
	if userEntity.Edges.Tenant != nil {
		tenantType = string(userEntity.Edges.Tenant.Type)
	}
	if !middleware.IsMSPEnabled() || !middleware.IsMSPStaff(tenantType, string(userEntity.MspRole)) {
		mainMenus = filterMSPMenus(mainMenus)
		adminMenus = filterMSPMenus(adminMenus)
	}

	return &dto.MenuTreeResponse{
		Main:  mainMenus,
		Admin: adminMenus,
	}, nil
}

// filterMSPMenus 递归剔除 MSP 模块相关菜单项。
// 判定标准：菜单路径以 "/msp" 开头，或权限码以 "msp" 开头（覆盖 msp:*、msp_allocation:*、msp_customer:*、msp_ticket:*、msp_report:*）。
func filterMSPMenus(menus []dto.MenuDTO) []dto.MenuDTO {
	if len(menus) == 0 {
		return menus
	}
	result := make([]dto.MenuDTO, 0, len(menus))
	for _, m := range menus {
		if isMSPMenu(m) {
			continue
		}
		// 保留非 MSP 父级，但递归过滤其子菜单中的 MSP 项
		if len(m.Children) > 0 {
			m.Children = filterMSPMenus(m.Children)
		}
		result = append(result, m)
	}
	return result
}

// isMSPMenu 判定单个菜单项是否属于 MSP 模块
func isMSPMenu(m dto.MenuDTO) bool {
	if m.Path != "" && strings.HasPrefix(m.Path, "/msp") {
		return true
	}
	if m.PermissionCode != nil && *m.PermissionCode != "" {
		if strings.HasPrefix(*m.PermissionCode, "msp") {
			return true
		}
	}
	return false
}

// getUserPermissions 获取用户的权限列表
func (s *MenuService) getUserPermissions(ctx context.Context, userEntity *ent.User, tenantID int) map[string]bool {
	permissions := make(map[string]bool)

	// 超级管理员拥有所有权限
	if userEntity.Role == "super_admin" {
		permissions["*"] = true
		return permissions
	}

	roleCodes := make([]string, 0, 1+len(userEntity.Edges.Roles))

	if userEntity.Role != "" {
		roleCodes = append(roleCodes, string(userEntity.Role))
	}

	if userEntity.Edges.Roles != nil {
		for _, r := range userEntity.Edges.Roles {
			roleCodes = append(roleCodes, r.Code)
		}
	}

	if middleware.PermissionConfig.Mode != middleware.PermissionConfigModeDBOnly {
		for _, roleCode := range roleCodes {
			rolePerms := middleware.RoleDefaultPermissions(roleCode)
			for _, p := range rolePerms {
				key := p.Resource + ":" + p.Action
				permissions[key] = true
				if p.Action == "*" {
					permissions[p.Resource+":*"] = true
				}
			}
		}
	}

	s.addDatabaseRolePermissions(ctx, permissions, tenantID, roleCodes)

	return permissions
}

func (s *MenuService) addDatabaseRolePermissions(ctx context.Context, permissions map[string]bool, tenantID int, roleCodes []string) {
	if len(roleCodes) == 0 {
		return
	}

	uniqueRoleCodes := make([]string, 0, len(roleCodes))
	seenRoleCodes := make(map[string]bool, len(roleCodes))
	for _, code := range roleCodes {
		code = strings.TrimSpace(code)
		if code == "" || seenRoleCodes[code] {
			continue
		}
		seenRoleCodes[code] = true
		uniqueRoleCodes = append(uniqueRoleCodes, code)
	}
	if len(uniqueRoleCodes) == 0 {
		return
	}

	roles, err := s.client.Role.Query().
		Where(role.TenantIDEQ(tenantID), role.CodeIn(uniqueRoleCodes...)).
		All(ctx)
	if err != nil {
		s.logger.Warnw("Failed to query database roles for menu permissions", "error", err, "tenant_id", tenantID)
		return
	}
	if len(roles) == 0 {
		return
	}

	roleIDs := make([]int, 0, len(roles))
	for _, r := range roles {
		roleIDs = append(roleIDs, r.ID)
	}

	rolePerms, err := s.client.RolePermission.Query().
		Where(rolepermission.RoleIDIn(roleIDs...)).
		All(ctx)
	if err != nil {
		s.logger.Warnw("Failed to query role permissions for menus", "error", err, "tenant_id", tenantID)
		return
	}
	if len(rolePerms) == 0 {
		return
	}

	permissionIDs := make([]int, 0, len(rolePerms))
	seenPermissionIDs := make(map[int]bool, len(rolePerms))
	for _, rp := range rolePerms {
		if rp.PermissionID == 0 || seenPermissionIDs[rp.PermissionID] {
			continue
		}
		seenPermissionIDs[rp.PermissionID] = true
		permissionIDs = append(permissionIDs, rp.PermissionID)
	}
	if len(permissionIDs) == 0 {
		return
	}

	dbPermissions, err := s.client.Permission.Query().
		Where(permission.IDIn(permissionIDs...), permission.TenantIDEQ(tenantID)).
		All(ctx)
	if err != nil {
		s.logger.Warnw("Failed to query permissions for menus", "error", err, "tenant_id", tenantID)
		return
	}

	for _, p := range dbPermissions {
		if p.Resource == "" || p.Action == "" {
			continue
		}
		key := p.Resource + ":" + p.Action
		permissions[key] = true
		if p.Action == "*" {
			permissions[p.Resource+":*"] = true
		}
	}
}

// filterMenusByPermission 根据权限过滤菜单
func (s *MenuService) filterMenusByPermission(menus []*ent.Menu, permissions map[string]bool, roleCodes map[string]bool) []*ent.Menu {
	filtered := make([]*ent.Menu, 0)

	// debugLog nil 守卫：单元测试直接构造 MenuService{}，logger 可能为 nil；
	// 调试日志不得成为 nil 指针 panic 源。
	debugLog := func(msg string, kv ...interface{}) {
		if s.logger != nil {
			s.logger.Infow(msg, kv...)
		}
	}

	debugLog("DEBUG filterMenusByPermission entry",
		"total_menus", len(menus),
		"permissions_count", len(permissions),
		"permissions_keys", keysOfMap(permissions),
		"roleCodes", keysOfMap(roleCodes))

	for _, m := range menus {
		// 菜单过滤纯按权限码判定；不再按角色名/路径前缀硬编码截断
		// （AGENTS.md 强制：隐藏菜单不是授权，前端不得根据角色名称推导超出后端返回权限的能力）。

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
			debugLog("DEBUG menu excluded",
				"id", m.ID, "name", m.Name, "path", m.Path, "perm", m.PermissionCode, "reason", "no-permission-match")
		} else {
			debugLog("DEBUG menu excluded",
				"id", m.ID, "name", m.Name, "path", m.Path, "perm", m.PermissionCode, "reason", "split-not-2")
		}
	}

	return filtered
}

func keysOfMap(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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

// backfillParentMenus 确保所有子菜单的父节点存在于过滤结果中。
// 当父菜单因权限码不满足被 filterMenusByPermission 剔除，但其至少一个子菜单满足时，
// 将父菜单作为纯导航容器回填到结果集，以保证 buildMenuTree 能正确构建层级——
// 否则 buildMenuTree 只从存活的根节点出发建树，存活子菜单会因父节点缺失被整棵丢弃。
// 回填只是展示层导航容器：被回填的父菜单不代表授权，接口仍逐请求把关。
func (s *MenuService) backfillParentMenus(filtered, all []*ent.Menu) []*ent.Menu {
	kept := make(map[int]bool, len(filtered))
	for _, m := range filtered {
		kept[m.ID] = true
	}

	allByID := make(map[int]*ent.Menu, len(all))
	for _, m := range all {
		allByID[m.ID] = m
	}

	changed := true
	for changed {
		changed = false
		for _, m := range filtered {
			if m.ParentID == nil {
				continue
			}
			pid := *m.ParentID
			if kept[pid] {
				continue
			}
			if parent, ok := allByID[pid]; ok {
				filtered = append(filtered, parent)
				kept[pid] = true
				changed = true
			}
		}
	}

	return filtered
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

// ExportMenus 导出当前租户菜单为 seeder 兼容格式（parentPath 替代 parentId）。
func (s *MenuService) ExportMenus(ctx context.Context, tenantID int) ([]dto.MenuExportItem, error) {
	menus, err := s.client.Menu.Query().
		Where(menu.TenantID(tenantID)).
		Order(ent.Asc(menu.FieldSortOrder)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("导出菜单失败: %w", err)
	}

	idToPath := make(map[int]string, len(menus))
	for _, m := range menus {
		idToPath[m.ID] = m.Path
	}

	items := make([]dto.MenuExportItem, 0, len(menus))
	for _, m := range menus {
		item := dto.MenuExportItem{
			Name:           m.Name,
			Path:           m.Path,
			Icon:           m.Icon,
			PermissionCode: m.PermissionCode,
			SortOrder:      m.SortOrder,
			Description:    m.Description,
		}
		if m.ParentID != nil {
			if parentPath, ok := idToPath[*m.ParentID]; ok {
				item.ParentPath = parentPath
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// InitMenusFromBaseline 从 pkg/menubaseline 基线补齐缺失菜单，返回 diff 报告。
// 已存在的菜单不覆盖 isVisible/isEnabled 等运营字段。
func (s *MenuService) InitMenusFromBaseline(ctx context.Context, tenantID int) (*dto.MenuInitDiffResponse, error) {
	specs := menubaseline.MenuDefinitions()

	existing, err := s.client.Menu.Query().
		Where(menu.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询现有菜单失败: %w", err)
	}

	pathSet := make(map[string]*ent.Menu, len(existing))
	for _, m := range existing {
		pathSet[m.Path] = m
	}

	// 第一遍：创建/检查顶级菜单
	for i := range specs {
		if specs[i].ParentPath != "" {
			continue
		}
		s.upsertBaselineMenu(ctx, tenantID, &specs[i], pathSet)
	}

	// 刷新 pathSet 以包含刚创建的顶级菜单
	refreshed, err := s.client.Menu.Query().
		Where(menu.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("刷新菜单失败: %w", err)
	}
	pathSet = make(map[string]*ent.Menu, len(refreshed))
	for _, m := range refreshed {
		pathSet[m.Path] = m
	}

	// 第二遍：创建/检查子菜单
	for i := range specs {
		if specs[i].ParentPath == "" {
			continue
		}
		s.upsertBaselineMenu(ctx, tenantID, &specs[i], pathSet)
	}

	// 构建 diff：必须用初始化前的快照比对 pathSet，否则初始化过程中被 upsertBaselineMenu
	// 创建的菜单都会进 pathSet，added 恒空（diff 永远报告 0 新增）。
	existedBefore := make(map[string]struct{}, len(existing))
	for _, m := range existing {
		existedBefore[m.Path] = struct{}{}
	}
	added := make([]dto.MenuExportItem, 0)
	unchanged := make([]dto.MenuExportItem, 0)
	for _, spec := range specs {
		item := dto.MenuExportItem{
			Name:           spec.Name,
			Path:           spec.Path,
			Icon:           spec.Icon,
			ParentPath:     spec.ParentPath,
			PermissionCode: spec.PermissionCode,
			SortOrder:      spec.SortOrder,
			Description:    spec.Description,
		}
		if _, existed := existedBefore[spec.Path]; existed {
			unchanged = append(unchanged, item)
		} else {
			added = append(added, item)
		}
	}

	return &dto.MenuInitDiffResponse{
		Added:      added,
		Unchanged:  unchanged,
		TotalAdded: len(added),
	}, nil
}

// upsertBaselineMenu 按 (tenantID, path) 创建缺失菜单或更新已有菜单的基线字段。
func (s *MenuService) upsertBaselineMenu(ctx context.Context, tenantID int, spec *menubaseline.MenuSpec, pathSet map[string]*ent.Menu) {
	if m, exists := pathSet[spec.Path]; exists {
		var parentID *int
		if spec.ParentPath != "" {
			if parent, ok := pathSet[spec.ParentPath]; ok {
				parentID = &parent.ID
			}
		}
		updater := m.Update().
			SetName(spec.Name).
			SetIcon(spec.Icon).
			SetSortOrder(spec.SortOrder).
			SetPermissionCode(spec.PermissionCode)
		if parentID != nil {
			updater = updater.SetParentID(*parentID)
		} else {
			updater = updater.ClearParentID()
		}
		if spec.Description != "" {
			updater = updater.SetDescription(spec.Description)
		}
		if _, err := updater.Save(ctx); err != nil {
			s.logger.Warnw("update baseline menu failed", "path", spec.Path, "error", err)
		}
		return
	}

	creator := s.client.Menu.Create().
		SetName(spec.Name).
		SetPath(spec.Path).
		SetIcon(spec.Icon).
		SetTenantID(tenantID).
		SetSortOrder(spec.SortOrder).
		SetIsVisible(true).
		SetIsEnabled(true).
		SetPermissionCode(spec.PermissionCode)
	if spec.ParentPath != "" {
		if parent, ok := pathSet[spec.ParentPath]; ok {
			creator = creator.SetParentID(parent.ID)
		}
	}
	if spec.Description != "" {
		creator = creator.SetDescription(spec.Description)
	}
	created, err := creator.Save(ctx)
	if err != nil {
		s.logger.Warnw("create baseline menu failed", "path", spec.Path, "error", err)
		return
	}
	pathSet[spec.Path] = created
}
