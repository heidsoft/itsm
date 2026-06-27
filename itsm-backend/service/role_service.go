package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/role"
	"itsm-backend/ent/rolepermission"
	"itsm-backend/ent/user"
	"itsm-backend/middleware"

	"go.uber.org/zap"
)

// RoleService 角色服务
type RoleService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewRoleService 创建角色服务
func NewRoleService(client *ent.Client, logger *zap.SugaredLogger) *RoleService {
	return &RoleService{
		client: client,
		logger: logger,
	}
}

// getRolePermissionsByIDs 批量查询多个角色的权限（按roleID分组）
func (s *RoleService) getRolePermissionsByIDs(ctx context.Context, roleIDs []int) (map[int][]dto.PermissionInfo, error) {
	result := make(map[int][]dto.PermissionInfo, len(roleIDs))
	for _, rid := range roleIDs {
		result[rid] = []dto.PermissionInfo{}
	}

	if len(roleIDs) == 0 {
		return result, nil
	}

	// 查询 role_permissions 联表
	rps, err := s.client.RolePermission.Query().
		Where(rolepermission.RoleIDIn(roleIDs...)).
		All(ctx)
	if err != nil {
		return result, fmt.Errorf("查询角色权限关联失败: %w", err)
	}

	// 收集所有 permission IDs
	permIDs := make([]int, 0, len(rps))
	for _, rp := range rps {
		permIDs = append(permIDs, rp.PermissionID)
	}
	if len(permIDs) == 0 {
		return result, nil
	}

	// 批量查询权限详情
	perms, err := s.client.Permission.Query().
		Where(permission.IDIn(permIDs...)).
		All(ctx)
	if err != nil {
		return result, fmt.Errorf("查询权限详情失败: %w", err)
	}

	permMap := make(map[int]*ent.Permission, len(perms))
	for _, p := range perms {
		permMap[p.ID] = p
	}

	// 按 roleID 分组
	for _, rp := range rps {
		if p, ok := permMap[rp.PermissionID]; ok {
			result[rp.RoleID] = append(result[rp.RoleID], dto.PermissionInfo{
				ID:   p.ID,
				Code: p.Code,
				Name: p.Name,
			})
		}
	}
	return result, nil
}

// buildRoleResponse 构建角色基础响应（不加载权限）
func (s *RoleService) buildRoleResponse(roleEntity *ent.Role) *dto.RoleResponse {
	return &dto.RoleResponse{
		ID:          roleEntity.ID,
		Name:        roleEntity.Name,
		Code:        roleEntity.Code,
		Description: roleEntity.Description,
		IsSystem:    roleEntity.IsSystem,
		IsActive:    roleEntity.IsActive,
		Permissions: []dto.PermissionInfo{},
		TenantID:    roleEntity.TenantID,
		CreatedAt:   roleEntity.CreatedAt,
		UpdatedAt:   roleEntity.UpdatedAt,
	}
}

// CreateRole 创建角色
func (s *RoleService) CreateRole(ctx context.Context, req *dto.CreateRoleRequest, tenantID int) (*dto.RoleResponse, error) {
	s.logger.Infow("Creating role", "name", req.Name, "tenant_id", tenantID)

	code := req.Code
	if code == "" {
		code = generateCodeFromName(req.Name)
	}

	roleEntity, err := s.client.Role.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetCode(code).
		SetTenantID(tenantID).
		SetIsSystem(req.IsSystem).
		SetIsActive(true). // 新建角色默认启用
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建角色失败: %w", err)
	}

	if len(req.Permissions) > 0 {
		permissionIDs, err := s.permissionIDsByCodes(ctx, req.Permissions, tenantID)
		if err != nil {
			return nil, err
		}
		if err := s.AssignPermissions(ctx, roleEntity.ID, permissionIDs, tenantID); err != nil {
			return nil, err
		}
	}

	return s.GetRole(ctx, roleEntity.ID, tenantID)
}

// GetRole 获取角色详情
func (s *RoleService) GetRole(ctx context.Context, id int, tenantID int) (*dto.RoleResponse, error) {
	roleEntity, err := s.client.Role.Query().
		Where(
			role.IDEQ(id),
			role.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("角色不存在")
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}

	resp := s.buildRoleResponse(roleEntity)

	perms, err := s.getRolePermissionsByIDs(ctx, []int{id})
	if err != nil {
		s.logger.Warnw("Failed to load role permissions", "role_id", id, "error", err)
	} else {
		resp.Permissions = perms[id]
	}

	return resp, nil
}

// ListRoles 获取角色列表
func (s *RoleService) ListRoles(ctx context.Context, tenantID int, page, pageSize int, search string) ([]*dto.RoleResponse, int, error) {
	query := s.client.Role.Query().
		Where(role.TenantID(tenantID))

	if strings.TrimSpace(search) != "" {
		keyword := strings.TrimSpace(search)
		query = query.Where(role.Or(
			role.NameContainsFold(keyword),
			role.CodeContainsFold(keyword),
			role.DescriptionContainsFold(keyword),
		))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("统计角色数量失败: %w", err)
	}

	roles, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询角色列表失败: %w", err)
	}

	roleIDs := make([]int, len(roles))
	for i, r := range roles {
		roleIDs[i] = r.ID
	}

	permMap := make(map[int][]dto.PermissionInfo)
	if len(roleIDs) > 0 {
		permMap, err = s.getRolePermissionsByIDs(ctx, roleIDs)
		if err != nil {
			s.logger.Warnw("Failed to batch load role permissions", "error", err)
		}
	}

	responses := make([]*dto.RoleResponse, len(roles))
	for i, r := range roles {
		resp := s.buildRoleResponse(r)
		resp.Permissions = permMap[r.ID]
		responses[i] = resp
	}

	return responses, total, nil
}

// UpdateRole 更新角色
func (s *RoleService) UpdateRole(ctx context.Context, id int, req *dto.UpdateRoleRequest, tenantID int) (*dto.RoleResponse, error) {
	s.logger.Infow("Updating role", "id", id, "tenant_id", tenantID)

	roleEntity, err := s.client.Role.Query().
		Where(role.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("角色不存在: %w", err)
	}

	update := s.client.Role.Update().
		Where(
			role.IDEQ(id),
			role.TenantID(tenantID),
		)

	if req.Name != nil {
		update = update.SetName(*req.Name)
	}
	if req.Description != nil {
		update = update.SetDescription(*req.Description)
	}
	if req.Code != nil && strings.TrimSpace(*req.Code) != "" {
		update = update.SetCode(strings.TrimSpace(*req.Code))
	}
	// 支持更新角色启用状态
	if req.IsActive != nil {
		update = update.SetIsActive(*req.IsActive)
	}

	_, err = update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新角色失败: %w", err)
	}

	if req.Permissions != nil {
		permissionIDs, err := s.permissionIDsByCodes(ctx, req.Permissions, tenantID)
		if err != nil {
			return nil, err
		}
		if err := s.AssignPermissions(ctx, id, permissionIDs, tenantID); err != nil {
			return nil, err
		}
	}

	middleware.InvalidateRolePermissionCache(roleEntity.Code, tenantID)

	return s.GetRole(ctx, id, tenantID)
}

func (s *RoleService) permissionIDsByCodes(ctx context.Context, permissionCodes []string, tenantID int) ([]int, error) {
	normalizedCodes := make([]string, 0, len(permissionCodes))
	seen := make(map[string]struct{}, len(permissionCodes))
	for _, code := range permissionCodes {
		trimmed := strings.TrimSpace(code)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalizedCodes = append(normalizedCodes, trimmed)
	}
	if len(normalizedCodes) == 0 {
		return []int{}, nil
	}

	perms, err := s.client.Permission.Query().
		Where(
			permission.TenantID(tenantID),
			permission.CodeIn(normalizedCodes...),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询权限失败: %w", err)
	}
	if len(perms) != len(normalizedCodes) {
		found := make(map[string]struct{}, len(perms))
		for _, permEntity := range perms {
			found[permEntity.Code] = struct{}{}
		}
		missing := make([]string, 0)
		for _, code := range normalizedCodes {
			if _, ok := found[code]; !ok {
				missing = append(missing, code)
			}
		}
		return nil, fmt.Errorf("权限不存在，请先初始化权限字典: %s", strings.Join(missing, ", "))
	}

	ids := make([]int, 0, len(perms))
	for _, permEntity := range perms {
		ids = append(ids, permEntity.ID)
	}
	return ids, nil
}

// DeleteRole 删除角色
func (s *RoleService) DeleteRole(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting role", "id", id, "tenant_id", tenantID)

	roleEntity, err := s.client.Role.Query().
		Where(
			role.IDEQ(id),
			role.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("角色不存在: %w", err)
	}

	if roleEntity.IsSystem {
		return fmt.Errorf("系统角色不能删除")
	}

	count, err := s.client.User.Query().
		Where(
			user.HasRolesWith(role.IDEQ(roleEntity.ID)),
			user.TenantID(tenantID),
		).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("检查用户失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("有%d个用户使用此角色，无法删除", count)
	}

	roleCode := roleEntity.Code

	// 删除角色权限关联
	_, err = s.client.RolePermission.Delete().
		Where(rolepermission.RoleID(id)).
		Exec(ctx)
	if err != nil {
		s.logger.Warnw("Failed to delete role permissions", "role_id", id, "error", err)
	}

	_, err = s.client.Role.Delete().
		Where(
			role.IDEQ(id),
			role.TenantID(tenantID),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除角色失败: %w", err)
	}

	middleware.InvalidateRolePermissionCache(roleCode, tenantID)

	return nil
}

// AssignPermissions 给角色分配权限（直接操作 role_permissions 联表）
func (s *RoleService) AssignPermissions(ctx context.Context, roleID int, permissionIDs []int, tenantID int) error {
	s.logger.Infow("Assigning permissions to role", "role_id", roleID, "permission_count", len(permissionIDs))

	roleEntity, err := s.client.Role.Query().
		Where(
			role.IDEQ(roleID),
			role.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("角色不存在: %w", err)
	}

	if len(permissionIDs) > 0 {
		perms, err := s.client.Permission.Query().
			Where(permission.IDIn(permissionIDs...)).
			All(ctx)
		if err != nil {
			return fmt.Errorf("查询权限失败: %w", err)
		}
		if len(perms) != len(permissionIDs) {
			return fmt.Errorf("部分权限不存在")
		}
	}

	// 清除现有权限关联
	_, err = s.client.RolePermission.Delete().
		Where(rolepermission.RoleID(roleID)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("清除权限关联失败: %w", err)
	}

	// 添加新权限关联
	for _, pid := range permissionIDs {
		_, err = s.client.RolePermission.Create().
			SetRoleID(roleID).
			SetPermissionID(pid).
			Save(ctx)
		if err != nil {
			s.logger.Warnw("Failed to create role-permission association", "role_id", roleID, "permission_id", pid, "error", err)
		}
	}

	middleware.InvalidateRolePermissionCache(roleEntity.Code, tenantID)

	return nil
}

// PermissionService 权限服务
type PermissionService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewPermissionService 创建权限服务
func NewPermissionService(client *ent.Client, logger *zap.SugaredLogger) *PermissionService {
	return &PermissionService{
		client: client,
		logger: logger,
	}
}

// CreatePermission 创建权限
func (s *PermissionService) CreatePermission(ctx context.Context, req *dto.CreatePermissionRequest, tenantID int) (*dto.PermissionResponse, error) {
	s.logger.Infow("Creating permission", "code", req.Code, "tenant_id", tenantID)

	permEntity, err := s.client.Permission.Create().
		SetCode(req.Code).
		SetName(req.Name).
		SetDescription(req.Description).
		SetResource(req.Resource).
		SetAction(req.Action).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建权限失败: %w", err)
	}

	return s.toPermissionResponse(permEntity), nil
}

// ListPermissions 获取权限列表
func (s *PermissionService) ListPermissions(ctx context.Context, tenantID int, resource string) ([]*dto.PermissionResponse, error) {
	query := s.client.Permission.Query().
		Where(permission.TenantID(tenantID))

	if resource != "" {
		query = query.Where(permission.Resource(resource))
	}

	perms, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询权限列表失败: %w", err)
	}

	responses := make([]*dto.PermissionResponse, len(perms))
	for i, p := range perms {
		responses[i] = s.toPermissionResponse(p)
	}

	return responses, nil
}

// InitDefaultPermissions 初始化默认权限
func (s *PermissionService) InitDefaultPermissions(ctx context.Context, tenantID int) error {
	defaultPermissions := []dto.CreatePermissionRequest{}
	resourceActions := map[string][]string{
		"dashboard":       {"read", "view"},
		"ticket":          {"create", "read", "write", "update", "delete", "assign", "export"},
		"ticket_category": {"create", "read", "write", "update", "delete"},
		"incident":        {"create", "read", "write", "update", "delete", "assign", "export"},
		"problem":         {"create", "read", "write", "update", "delete", "export"},
		"change":          {"create", "read", "write", "update", "delete", "approve", "export"},
		"service_catalog": {"create", "read", "write", "update", "delete"},
		"service_request": {"create", "read", "write", "update", "delete", "approve"},
		"knowledge":       {"create", "read", "write", "update", "delete"},
		"cmdb":            {"create", "read", "write", "update", "delete", "export"},
		"asset":           {"create", "read", "write", "update", "delete", "assign"},
		"license":         {"create", "read", "write", "update", "delete", "assign"},
		"release":         {"create", "read", "write", "update", "delete"},
		"report":          {"read", "view", "export"},
		"user":            {"read", "write", "delete", "manage"},
		"role":            {"read", "write", "delete", "manage"},
		"groups":          {"read", "write", "delete"},
		"org":             {"read", "write"},
		"bpmn":            {"read", "write", "delete"},
		"workflow":        {"read", "write", "delete"},
		"system_config":   {"read", "write", "update"},
		"admin":           {"read", "write", "all"},
		"ai":              {"read", "write"},
	}

	for resource, actions := range resourceActions {
		for _, action := range actions {
			defaultPermissions = append(defaultPermissions, dto.CreatePermissionRequest{
				Code:        fmt.Sprintf("%s:%s", resource, action),
				Name:        fmt.Sprintf("%s %s", resource, action),
				Description: fmt.Sprintf("允许对 %s 执行 %s 操作", resource, action),
				Resource:    resource,
				Action:      action,
			})
		}
	}
	defaultPermissions = append(defaultPermissions, dto.CreatePermissionRequest{
		Code:        "admin:all",
		Name:        "完全管理",
		Description: "平台级全部管理权限",
		Resource:    "admin",
		Action:      "all",
	})

	for _, req := range defaultPermissions {
		exists, _ := s.client.Permission.Query().
			Where(
				permission.Code(req.Code),
				permission.TenantID(tenantID),
			).
			Exist(ctx)
		if exists {
			continue
		}

		_, err := s.CreatePermission(ctx, &req, tenantID)
		if err != nil {
			s.logger.Warnw("Failed to create default permission", "code", req.Code, "error", err)
		}
	}

	return nil
}

func (s *PermissionService) toPermissionResponse(permEntity *ent.Permission) *dto.PermissionResponse {
	return &dto.PermissionResponse{
		ID:          permEntity.ID,
		Code:        permEntity.Code,
		Name:        permEntity.Name,
		Description: permEntity.Description,
		Resource:    permEntity.Resource,
		Action:      permEntity.Action,
		TenantID:    permEntity.TenantID,
		CreatedAt:   permEntity.CreatedAt,
		UpdatedAt:   permEntity.UpdatedAt,
	}
}

// generateCodeFromName 从名称生成代码
func generateCodeFromName(name string) string {
	code := strings.ToLower(name)
	code = regexp.MustCompile(`[^a-z0-9\x{4e00}-\x{9fa5}]`).ReplaceAllString(code, "_")
	code = regexp.MustCompile(`_+`).ReplaceAllString(code, "_")
	code = strings.Trim(code, "_")
	if len(code) > 20 {
		code = code[:20]
	}
	return code
}
