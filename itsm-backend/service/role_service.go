package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/role"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/user"

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

// CreateRole 创建角色
func (s *RoleService) CreateRole(ctx context.Context, req *dto.CreateRoleRequest, tenantID int) (*dto.RoleResponse, error) {
	s.logger.Infow("Creating role", "name", req.Name, "tenant_id", tenantID)

	roleEntity, err := s.client.Role.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetCode(req.Code).
		SetTenantID(tenantID).
		SetIsSystem(req.IsSystem).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建角色失败: %w", err)
	}

	return s.toRoleResponse(roleEntity), nil
}

// GetRole 获取角色详情
func (s *RoleService) GetRole(ctx context.Context, id int, tenantID int) (*dto.RoleResponse, error) {
	roleEntity, err := s.client.Role.Query().
		Where(
			role.IDEQ(id),
			role.TenantID(tenantID),
		).
		WithPermissions().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("角色不存在")
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}

	return s.toRoleResponse(roleEntity), nil
}

// ListRoles 获取角色列表
func (s *RoleService) ListRoles(ctx context.Context, tenantID int, page, pageSize int) ([]*dto.RoleResponse, int, error) {
	query := s.client.Role.Query().
		Where(role.TenantID(tenantID))

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("统计角色数量失败: %w", err)
	}

	roles, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		WithPermissions().
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询角色列表失败: %w", err)
	}

	responses := make([]*dto.RoleResponse, len(roles))
	for i, r := range roles {
		responses[i] = s.toRoleResponse(r)
	}

	return responses, total, nil
}

// UpdateRole 更新角色
func (s *RoleService) UpdateRole(ctx context.Context, id int, req *dto.UpdateRoleRequest, tenantID int) (*dto.RoleResponse, error) {
	s.logger.Infow("Updating role", "id", id, "tenant_id", tenantID)

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

	_, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新角色失败: %w", err)
	}

	return s.GetRole(ctx, id, tenantID)
}

// DeleteRole 删除角色
func (s *RoleService) DeleteRole(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting role", "id", id, "tenant_id", tenantID)

	// 检查是否为系统角色
	roleEntity, err := s.client.Role.Query().
		Where(role.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("角色不存在: %w", err)
	}

	if roleEntity.IsSystem {
		return fmt.Errorf("系统角色不能删除")
	}

	// 检查是否有用户使用此角色
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

	// 删除角色的权限关联
	_, err = s.client.Role.Delete().Where(role.IDEQ(id)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除角色失败: %w", err)
	}

	return nil
}

// AssignPermissions 给角色分配权限
func (s *RoleService) AssignPermissions(ctx context.Context, roleID int, permissionIDs []int, tenantID int) error {
	s.logger.Infow("Assigning permissions to role", "role_id", roleID, "permission_count", len(permissionIDs))

	roleEntity, err := s.client.Role.Query().
		Where(
			role.IDEQ(roleID),
			role.TenantID(tenantID),
		).
		WithPermissions().
		Only(ctx)
	if err != nil {
		return fmt.Errorf("角色不存在: %w", err)
	}

	// 验证权限存在
	perms, err := s.client.Permission.Query().
		Where(permission.IDIn(permissionIDs...)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("查询权限失败: %w", err)
	}

	if len(perms) != len(permissionIDs) {
		return fmt.Errorf("部分权限不存在")
	}

	// 清除现有权限
	builder := roleEntity.Update().ClearPermissions()
	_, err = builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("清除权限失败: %w", err)
	}

	// 添加新权限
	for _, p := range perms {
		_, err = roleEntity.Update().AddPermissions(p).Save(ctx)
		if err != nil {
			s.logger.Warnw("Failed to add permission to role", "permission_id", p.ID, "error", err)
		}
	}

	return nil
}

func (s *RoleService) toRoleResponse(roleEntity *ent.Role) *dto.RoleResponse {
	permissions := make([]dto.PermissionInfo, 0)
	if roleEntity.Edges.Permissions != nil {
		for _, p := range roleEntity.Edges.Permissions {
			permissions = append(permissions, dto.PermissionInfo{
				ID:   p.ID,
				Code: p.Code,
				Name: p.Name,
			})
		}
	}

	return &dto.RoleResponse{
		ID:          roleEntity.ID,
		Name:        roleEntity.Name,
		Code:        roleEntity.Code,
		Description: roleEntity.Description,
		IsSystem:    roleEntity.IsSystem,
		Permissions: permissions,
		TenantID:    roleEntity.TenantID,
		CreatedAt:   roleEntity.CreatedAt,
		UpdatedAt:   roleEntity.UpdatedAt,
	}
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
	defaultPermissions := []dto.CreatePermissionRequest{
		{Code: "ticket:create", Name: "创建工单", Resource: "ticket", Action: "create"},
		{Code: "ticket:read", Name: "查看工单", Resource: "ticket", Action: "read"},
		{Code: "ticket:update", Name: "更新工单", Resource: "ticket", Action: "update"},
		{Code: "ticket:delete", Name: "删除工单", Resource: "ticket", Action: "delete"},
		{Code: "ticket:assign", Name: "分配工单", Resource: "ticket", Action: "assign"},
		{Code: "incident:create", Name: "创建事件", Resource: "incident", Action: "create"},
		{Code: "incident:read", Name: "查看事件", Resource: "incident", Action: "read"},
		{Code: "incident:update", Name: "更新事件", Resource: "incident", Action: "update"},
		{Code: "change:create", Name: "创建变更", Resource: "change", Action: "create"},
		{Code: "change:approve", Name: "审批变更", Resource: "change", Action: "approve"},
		{Code: "user:manage", Name: "管理用户", Resource: "user", Action: "manage"},
		{Code: "role:manage", Name: "管理角色", Resource: "role", Action: "manage"},
		{Code: "report:view", Name: "查看报表", Resource: "report", Action: "view"},
		{Code: "admin:all", Name: "完全管理", Resource: "*", Action: "*"},
	}

	for _, req := range defaultPermissions {
		// 检查是否已存在
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
