package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"itsm-backend/common"
	domainrole "itsm-backend/domain/role"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/role"
	"itsm-backend/ent/rolepermission"
	"itsm-backend/ent/user"
	"itsm-backend/internal/authz"
	"itsm-backend/middleware"

	"go.uber.org/zap"
)

// roleServiceImpl 是 RoleService 接口的 Ent 实现（域内自持，不再放共享 service 包）。
type roleServiceImpl struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewRoleService 创建角色服务
func NewRoleService(client *ent.Client, logger *zap.SugaredLogger) RoleService {
	return &roleServiceImpl{
		client: client,
		logger: logger,
	}
}

// getRolePermissionsByIDs 批量查询多个角色的权限（按roleID分组，tenantID隔离）
func (s *roleServiceImpl) getRolePermissionsByIDs(ctx context.Context, roleIDs []int, tenantID int) (map[int][]dto.PermissionInfo, error) {
	result := make(map[int][]dto.PermissionInfo, len(roleIDs))
	for _, rid := range roleIDs {
		result[rid] = []dto.PermissionInfo{}
	}

	if len(roleIDs) == 0 {
		return result, nil
	}

	// 查询 role_permissions 联表（必须带 TenantID 过滤防止跨租户泄露）
	rps, err := s.client.RolePermission.Query().
		Where(
			rolepermission.RoleIDIn(roleIDs...),
			rolepermission.TenantID(tenantID),
		).
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

	// 批量查询权限详情（带 TenantID 过滤）
	perms, err := s.client.Permission.Query().
		Where(
			permission.IDIn(permIDs...),
			permission.TenantID(tenantID),
		).
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
func (s *roleServiceImpl) buildRoleResponse(roleEntity *ent.Role) *dto.RoleResponse {
	return &dto.RoleResponse{
		ID:          roleEntity.ID,
		Name:        roleEntity.Name,
		Code:        roleEntity.Code,
		Description: roleEntity.Description,
		IsSystem:    roleEntity.IsSystem,
		Status:      dto.RoleStatusFromActive(roleEntity.IsActive),
		IsActive:    roleEntity.IsActive,
		DataScope:   string(roleEntity.DataScope),
		Permissions: []dto.PermissionInfo{},
		TenantID:    roleEntity.TenantID,
		CreatedAt:   roleEntity.CreatedAt,
		UpdatedAt:   roleEntity.UpdatedAt,
	}
}

// CreateRole 创建角色
func (s *roleServiceImpl) CreateRole(ctx context.Context, req *dto.CreateRoleRequest, tenantID int) (*dto.RoleResponse, error) {
	s.logger.Infow("Creating role", "name", req.Name, "tenant_id", tenantID)

	code := req.Code
	if code == "" {
		code = generateCodeFromName(req.Name)
	}

	if domainrole.IsBuiltinCode(code) {
		return nil, common.NewStateConflictError("内置角色码只能由系统播种", code)
	}

	roleEntity, err := s.client.Role.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetCode(code).
		SetTenantID(tenantID).
		SetIsSystem(false).
		SetIsActive(req.Status != dto.RoleStatusInactive). // 默认启用，仅显式 inactive 时禁用
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
func (s *roleServiceImpl) GetRole(ctx context.Context, id int, tenantID int) (*dto.RoleResponse, error) {
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

	perms, err := s.getRolePermissionsByIDs(ctx, []int{id}, tenantID)
	if err != nil {
		s.logger.Warnw("Failed to load role permissions", "role_id", id, "error", err)
	} else {
		resp.Permissions = perms[id]
	}

	return resp, nil
}

// ListRoles 获取角色列表
func (s *roleServiceImpl) ListRoles(ctx context.Context, tenantID int, params *dto.GetRolesParams) ([]*dto.RoleResponse, int, error) {
	page, pageSize := common.ValidatePagination(params.Page, params.PageSize)

	query := s.client.Role.Query().
		Where(role.TenantID(tenantID))

	if strings.TrimSpace(params.Search) != "" {
		keyword := strings.TrimSpace(params.Search)
		query = query.Where(role.Or(
			role.NameContainsFold(keyword),
			role.CodeContainsFold(keyword),
			role.DescriptionContainsFold(keyword),
		))
	}

	// status 取值已由 handler 用 dto.RoleActiveFromStatus 校验，非法值不会到这里
	if active, ok := dto.RoleActiveFromStatus(params.Status); ok {
		query = query.Where(role.IsActive(active))
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
		permMap, err = s.getRolePermissionsByIDs(ctx, roleIDs, tenantID)
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
func (s *roleServiceImpl) UpdateRole(ctx context.Context, id int, req *dto.UpdateRoleRequest, tenantID int) (*dto.RoleResponse, error) {
	s.logger.Infow("Updating role", "id", id, "tenant_id", tenantID)

	roleEntity, err := s.client.Role.Query().
		Where(
			role.IDEQ(id),
			role.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("角色不存在或无权访问")
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
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
		code := strings.TrimSpace(*req.Code)
		if code != roleEntity.Code && domainrole.IsBuiltinCode(code) {
			return nil, common.NewStateConflictError("内置角色码只能由系统播种", code)
		}
		update = update.SetCode(code)
	}
	// 启用状态：线上契约 status(active/inactive) → 持久层 is_active(bool)
	// handler 已用 dto.RoleActiveFromStatus 校验取值，此处只需按契约映射
	if req.Status != nil {
		update = update.SetIsActive(*req.Status == dto.RoleStatusActive)
	}

	// 权限码必须先校验再落库。历史实现把 Save 放在校验之前：只要有一个码
	// 不在本租户目录里，名称/状态已经写入而权限未变，接口返回失败但数据库
	// 留下半成品，前端提示「保存失败」也无法解释实际发生了什么。
	var permissionIDs []int
	if req.Permissions != nil {
		permissionIDs, err = s.permissionIDsByCodes(ctx, req.Permissions, tenantID)
		if err != nil {
			return nil, err
		}
	}

	_, err = update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新角色失败: %w", err)
	}

	// 非 nil 表示请求明确提交了权限字段；空切片是合法的「清空全部权限」。
	if permissionIDs != nil {
		if err := s.AssignPermissions(ctx, id, permissionIDs, tenantID); err != nil {
			return nil, err
		}
	}

	middleware.InvalidateRolePermissionCache(roleEntity.Code, tenantID)

	return s.GetRole(ctx, id, tenantID)
}

func (s *roleServiceImpl) permissionIDsByCodes(ctx context.Context, permissionCodes []string, tenantID int) ([]int, error) {
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
func (s *roleServiceImpl) DeleteRole(ctx context.Context, id int, tenantID int) error {
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
		return common.NewBusinessError(common.ConflictCode, "系统角色不能删除", roleEntity.Code)
	}

	// P0 双重保护：即使 seed 时 IsSystem 字段漏设，
	// 只要 code 命中 domain/role 内置词表，仍视为系统角色禁止删除。
	if domainrole.IsBuiltinCode(roleEntity.Code) {
		s.logger.Warnw("Blocked deletion of builtin role (is_system=false but code matches)",
			"role_id", roleEntity.ID, "role_code", roleEntity.Code, "tenant_id", tenantID)
		return common.NewBusinessError(common.ConflictCode, "内置系统角色不能删除（请升级或联系管理员）", roleEntity.Code)
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
// 使用事务包裹 Delete+CreateBulk，循环内 err 必须 return 触发 Rollback，
// 防止"旧权限已清空、新权限只写入部分"的越权/失权静默故障。
// ErrPermissionNotInTenant 表示请求绑定的权限中存在不属于当前租户（或不存在）的项。
// 属于调用方输入错误（应映射为 400），而非服务端内部错误（500）。
var ErrPermissionNotInTenant = errors.New("部分权限不存在或不属于当前租户")

func (s *roleServiceImpl) AssignPermissions(ctx context.Context, roleID int, permissionIDs []int, tenantID int) error {
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

	// 权限变更审计： 事务前快照旧权限（code 列表），用于 after 提交后写 diff。
	beforeCodes, err := s.rolePermissionCodes(ctx, roleID, tenantID)
	if err != nil {
		// 快照失败不阻断授权，但记录（审计降级而非业务失败）
		s.logger.Warnw("权限变更前快照失败，审计diff将缺失before", "role_id", roleID, "error", err)
		beforeCodes = nil
	}

	if len(permissionIDs) > 0 {
		// 权限是租户级实体，绑定角色时必须按本租户过滤。
		// 否则可传入他租户 permission ID 通过 len 校验，造成跨租户提权。
		perms, err := s.client.Permission.Query().
			Where(
				permission.IDIn(permissionIDs...),
				permission.TenantID(tenantID),
			).
			All(ctx)
		if err != nil {
			return fmt.Errorf("查询权限失败: %w", err)
		}
		if len(perms) != len(permissionIDs) {
			return ErrPermissionNotInTenant
		}
	}

	// 开启事务：DELETE + CREATE BULK 必须原子化
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	// 清除现有权限关联
	_, err = tx.RolePermission.Delete().
		Where(
			rolepermission.RoleID(roleID),
			rolepermission.TenantID(tenantID),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("清除权限关联失败: %w", err)
	}

	// 批量添加新权限关联（使用 CreateBulk 避免循环）
	if len(permissionIDs) > 0 {
		bulk := make([]*ent.RolePermissionCreate, 0, len(permissionIDs))
		for _, pid := range permissionIDs {
			bulk = append(bulk, tx.RolePermission.Create().
				SetRoleID(roleID).
				SetPermissionID(pid).
				SetTenantID(tenantID),
			)
		}
		if err := tx.RolePermission.CreateBulk(bulk...).Exec(ctx); err != nil {
			// CreateBulk 任何一条失败全部回滚，避免部分写入
			return fmt.Errorf("创建权限关联失败: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	middleware.InvalidateRolePermissionCache(roleEntity.Code, tenantID)

	// 权限变更审计： 提交后快照新权限并写 diff（before/after code 列表）。
	s.writePermissionChangeAudit(ctx, roleEntity, tenantID, beforeCodes, permissionIDs)

	return nil
}

// rolePermissionCodes 返回角色当前权限的 code 列表（排序后，供审计 diff）。
func (s *roleServiceImpl) rolePermissionCodes(ctx context.Context, roleID, tenantID int) ([]string, error) {
	// RolePermission 无 ent 边（schema Edges 为 nil），按既有模式两步查询：
	// 先取 role_permissions 的 permission_id，再按租户过滤查 permission.code。
	rolePerms, err := s.client.RolePermission.Query().
		Where(rolepermission.RoleIDEQ(roleID), rolepermission.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(rolePerms) == 0 {
		return []string{}, nil
	}
	permIDs := make([]int, 0, len(rolePerms))
	for _, rp := range rolePerms {
		permIDs = append(permIDs, rp.PermissionID)
	}
	permEntities, err := s.client.Permission.Query().
		Where(permission.IDIn(permIDs...), permission.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(permEntities))
	for _, p := range permEntities {
		codes = append(codes, p.Code)
	}
	sort.Strings(codes)
	return codes, nil
}

// writePermissionChangeAudit 写权限变更 diff 审计。
// after 由提交后的重新快照获得（而非信任入参），确保审计与库内终态一致。
func (s *roleServiceImpl) writePermissionChangeAudit(ctx context.Context, roleEntity *ent.Role, tenantID int, before []string, _ []int) {
	after, err := s.rolePermissionCodes(ctx, roleEntity.ID, tenantID)
	if err != nil {
		s.logger.Warnw("权限变更后快照失败，审计diff缺失after", "role_id", roleEntity.ID, "error", err)
		return
	}
	sort.Strings(before)

	diff := map[string]any{
		"role_id":   roleEntity.ID,
		"role_code": roleEntity.Code,
		"before":    before,
		"after":     after,
	}
	payload, err := json.Marshal(diff)
	if err != nil {
		s.logger.Warnw("权限变更diff序列化失败", "role_id", roleEntity.ID, "error", err)
		return
	}

	if _, err := s.client.AuditLog.Create().
		SetTenantID(tenantID).
		SetResource("role_permissions").
		SetAction("assign").
		SetPath(fmt.Sprintf("/api/v1/roles/%d/permissions", roleEntity.ID)).
		SetMethod("POST").
		SetStatusCode(200).
		SetRequestBody(string(payload)).
		Save(ctx); err != nil {
		// 审计写失败不回滚业务（权限已生效），但必须告警。
		s.logger.Errorw("权限变更审计写入失败", "role_id", roleEntity.ID, "error", err)
		return
	}
	s.logger.Infow("权限变更审计已记录", "role_id", roleEntity.ID, "before_count", len(before), "after_count", len(after))
}

// permissionServiceImpl 是 PermissionService 接口的 Ent 实现。
//
// 权限码的运行期真源是 DB permissions 行；本实现只负责读写那一行，
// 不再内联任何 resource/action 字典。策略以版本化信封存在 description 里，
// 变更后通过 middleware.InvalidateAuthzCaches 让所有鉴权消费方（含其它副本）立即看到。
type permissionServiceImpl struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewPermissionService 创建权限服务
func NewPermissionService(client *ent.Client, logger *zap.SugaredLogger) PermissionService {
	return &permissionServiceImpl{
		client: client,
		logger: logger,
	}
}

// policyFromPayload 把 API 载荷转成策略，isSystem 永远来自库中现有行。
func policyFromPayload(p *dto.PermissionPolicyPayload, isSystem bool) authz.PermissionPolicy {
	return authz.PermissionPolicy{
		RiskLevel:         p.RiskLevel,
		AllowedPrincipals: append([]string(nil), p.AllowedPrincipals...),
		RequiresApproval:  p.RequiresApproval,
		MinApprovals:      p.MinApprovals,
		Category:          p.Category,
		DisplayName:       p.DisplayName,
		Icon:              p.Icon,
		Color:             p.Color,
		I18nKey:           p.I18nKey,
		SortOrder:         p.SortOrder,
		IsActive:          p.IsActive,
		IsSystem:          isSystem,
	}
}

// validateCodeTriple 拒绝 code 与 resource:action 不一致、以及非法字符，
// 避免 DB 里出现「路由声明永远匹配不上」的死码。
func validateCodeTriple(code, resource, action string) error {
	code = strings.TrimSpace(code)
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	if code == "" || resource == "" || action == "" {
		return fmt.Errorf("code/resource/action 均不能为空")
	}
	if code != resource+":"+action {
		return fmt.Errorf("权限码 %q 必须等于 resource:action（%s:%s）", code, resource, action)
	}
	if strings.ContainsAny(code, " \t\n\r") {
		return fmt.Errorf("权限码 %q 不得包含空白字符", code)
	}
	return nil
}

// CreatePermission 创建权限。
//
// 与旧实现的差别：写入前校验码三元组、同租户唯一性、策略闭集；
// 描述统一编码成策略信封，并让共享注册表缓存失效。
func (s *permissionServiceImpl) CreatePermission(ctx context.Context, req *dto.CreatePermissionRequest, tenantID int) (*dto.PermissionResponse, error) {
	return s.createPermission(ctx, req, tenantID, false)
}

// createPermission 是唯一写入口。isSystem 由调用方决定：引导清单补齐的码受保护，
// 管理员在运行期新建的码允许退役，客户端无法伪造这个标记。
func (s *permissionServiceImpl) createPermission(ctx context.Context, req *dto.CreatePermissionRequest, tenantID int, isSystem bool) (*dto.PermissionResponse, error) {
	s.logger.Infow("Creating permission", "code", req.Code, "tenant_id", tenantID)

	if err := validateCodeTriple(req.Code, req.Resource, req.Action); err != nil {
		return nil, fmt.Errorf("参数错误: %w", err)
	}
	exists, err := s.client.Permission.Query().
		Where(permission.CodeEQ(strings.TrimSpace(req.Code)), permission.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询权限重复失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("权限码 %q 已存在", req.Code)
	}

	policy := authz.DerivePolicy(req.Resource, req.Action)
	policy.IsSystem = isSystem
	if req.Policy != nil {
		policy = policyFromPayload(req.Policy, isSystem)
		if err := authz.ValidatePolicy(policy); err != nil {
			return nil, fmt.Errorf("参数错误: %w", err)
		}
	}
	encoded, err := authz.EncodeDefinition(req.Description, policy)
	if err != nil {
		return nil, fmt.Errorf("参数错误: %w", err)
	}

	permEntity, err := s.client.Permission.Create().
		SetCode(strings.TrimSpace(req.Code)).
		SetName(req.Name).
		SetDescription(encoded).
		SetResource(strings.TrimSpace(req.Resource)).
		SetAction(strings.TrimSpace(req.Action)).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建权限失败: %w", err)
	}
	middleware.InvalidateAuthzCaches(tenantID)
	return s.toPermissionResponse(permEntity), nil
}

// ListPermissions 获取权限列表。
//
// 读面用宽松解码（坏信封降级为派生默认策略而不是 500），但派生默认策略
// 恒为 AI 零权限，所以降级方向是收紧而不是放宽。
func (s *permissionServiceImpl) ListPermissions(ctx context.Context, tenantID int, resource string) ([]*dto.PermissionResponse, error) {
	query := s.client.Permission.Query().
		Where(permission.TenantIDEQ(tenantID))

	if resource != "" {
		query = query.Where(permission.ResourceEQ(resource))
	}

	perms, err := query.Order(ent.Asc(permission.FieldCode)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询权限列表失败: %w", err)
	}

	responses := make([]*dto.PermissionResponse, len(perms))
	for i, p := range perms {
		responses[i] = s.toPermissionResponse(p)
	}
	return responses, nil
}

// GetPermission 按 ID 读取单条定义（租户隔离）。
func (s *permissionServiceImpl) GetPermission(ctx context.Context, id, tenantID int) (*dto.PermissionResponse, error) {
	p, err := s.client.Permission.Query().
		Where(permission.IDEQ(id), permission.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("权限不存在: %w", err)
	}
	return s.toPermissionResponse(p), nil
}

// UpdatePermission 整体替换名称/描述/策略，不改码身份。
//
// 写入前跑 ValidatePolicy：例如「允许 AI 执行高风险操作却关掉审批」会被拒绝，
// 安全不变量不会因为管理员误操作而在 DB 里落地。
func (s *permissionServiceImpl) UpdatePermission(ctx context.Context, id, tenantID int, req *dto.UpdatePermissionRequest) (*dto.PermissionResponse, error) {
	existing, err := s.client.Permission.Query().
		Where(permission.IDEQ(id), permission.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("权限不存在: %w", err)
	}
	_, current, decErr := authz.DecodeDefinition(existing.Description, existing.Resource, existing.Action)
	if decErr != nil {
		// 坏信封不能就地覆盖：先报错让管理员走显式修复，避免静默丢数据。
		return nil, fmt.Errorf("该权限的策略信封损坏，无法更新: %w", decErr)
	}
	policy := policyFromPayload(req.Policy, current.IsSystem)
	if err := authz.ValidatePolicy(policy); err != nil {
		return nil, fmt.Errorf("参数错误: %w", err)
	}
	description := req.Description
	if description == "" {
		description = currentDescription(existing.Description)
	}
	encoded, err := authz.EncodeDefinition(description, policy)
	if err != nil {
		return nil, fmt.Errorf("参数错误: %w", err)
	}

	update := s.client.Permission.UpdateOneID(id).
		Where(permission.TenantIDEQ(tenantID)).
		SetDescription(encoded)
	if strings.TrimSpace(req.Name) != "" {
		update.SetName(strings.TrimSpace(req.Name))
	}
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新权限失败: %w", err)
	}
	// InvalidateAuthzCaches 现在统一失效三类缓存（角色权限 / Registry / ACL）
	// 并向其它副本广播；不要再单独调 InvalidateAllPermissionCachesEx，
	// 后者只清本进程且不广播，会让停用状态在其它副本残留到 TTL 到期。
	middleware.InvalidateAuthzCaches(tenantID)
	s.logger.Infow("permission policy updated",
		"tenant_id", tenantID, "code", existing.Code,
		"risk", policy.RiskLevel, "principals", policy.AllowedPrincipals,
		"min_approvals", policy.MinApprovals, "active", policy.IsActive)
	return s.toPermissionResponse(updated), nil
}

// DeletePermission 只做「停用」而不物理删除：role_permissions 与路由声明
// 可能仍引用这个码，物理删除会让历史绑定变成悬空 ID。
func (s *permissionServiceImpl) DeletePermission(ctx context.Context, id, tenantID int) error {
	existing, err := s.client.Permission.Query().
		Where(permission.IDEQ(id), permission.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("权限不存在: %w", err)
	}
	_, policy, err := authz.DecodeDefinition(existing.Description, existing.Resource, existing.Action)
	if err != nil {
		return fmt.Errorf("该权限的策略信封损坏，无法停用: %w", err)
	}
	if policy.IsSystem {
		return fmt.Errorf("引导清单权限码 %q 不允许删除，可将其停用", existing.Code)
	}
	policy.IsActive = false
	encoded, err := authz.EncodeDefinition(currentDescription(existing.Description), policy)
	if err != nil {
		return err
	}
	if _, err := s.client.Permission.UpdateOneID(id).
		Where(permission.TenantIDEQ(tenantID)).
		SetDescription(encoded).Save(ctx); err != nil {
		return fmt.Errorf("停用权限失败: %w", err)
	}
	// 与 UpdatePermission 走同一失效原语（三类缓存 + 跨副本广播）。
	middleware.InvalidateAuthzCaches(tenantID)
	return nil
}

// CatalogMeta 返回展示口径闭集，前端零硬编码。
func (s *permissionServiceImpl) CatalogMeta() *dto.PermissionCatalogMetaResponse {
	labels := make(map[string]string, len(authz.AllCategories))
	for _, key := range authz.AllCategories {
		labels[key] = authz.CategoryLabel(key)
	}
	return &dto.PermissionCatalogMetaResponse{
		Principals:      append([]string(nil), authz.AllPrincipals...),
		RiskLevels:      append([]string(nil), authz.AllRiskLevels...),
		Categories:      append([]string(nil), authz.AllCategories...),
		CategoryLabels:  labels,
		MinApprovalsMax: 2,
	}
}

// InitDefaultPermissions 把引导清单补齐到当前租户。
//
// 这里曾经维护第二张手写 resourceActions 字典（含 dashboard:view / admin:all /
// groups:* 等码空间里根本不存在的码），是「权限码到处散落」的主要来源。
// 现在只从 authz.Definitions() 读——那是**唯一**的引导清单，且仅用于初始化。
//
// 幂等且尊重运行期配置：已存在的行一律不覆盖（管理员改过的策略/名称不会被
// 重启冲掉），只有缺失的码才补齐。
func (s *permissionServiceImpl) InitDefaultPermissions(ctx context.Context, tenantID int) error {
	created := 0
	for _, def := range authz.Definitions() {
		exists, err := s.client.Permission.Query().
			Where(permission.CodeEQ(def.Code), permission.TenantIDEQ(tenantID)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("查询权限 %s 失败: %w", def.Code, err)
		}
		if exists {
			continue
		}
		req := &dto.CreatePermissionRequest{
			Code:        def.Code,
			Name:        def.Name,
			Description: def.Description,
			Resource:    def.Resource,
			Action:      def.Action,
		}
		if _, err := s.createPermission(ctx, req, tenantID, true); err != nil {
			return fmt.Errorf("补齐引导权限 %s 失败: %w", def.Code, err)
		}
		created++
	}
	middleware.InvalidateAuthzCaches(tenantID)
	s.logger.Infow("permission bootstrap reconciled", "tenant_id", tenantID, "created", created, "total", len(authz.Definitions()))
	return nil
}

// currentDescription 从信封里取出人类描述；纯文本/损坏信封原样返回。
func currentDescription(raw string) string {
	desc, policy, err := authz.DecodeDefinition(raw, "", "")
	if err != nil {
		return raw
	}
	_ = policy
	return desc
}

func (s *permissionServiceImpl) toPermissionResponse(permEntity *ent.Permission) *dto.PermissionResponse {
	raw := permEntity.Description
	desc, policy := authz.DecodeDefinitionLenient(raw, permEntity.Resource, permEntity.Action)
	trimmed := strings.TrimSpace(raw)
	configured := strings.HasPrefix(trimmed, "{")
	return &dto.PermissionResponse{
		ID:          permEntity.ID,
		Code:        permEntity.Code,
		Name:        permEntity.Name,
		Description: desc,
		Resource:    permEntity.Resource,
		Action:      permEntity.Action,
		TenantID:    permEntity.TenantID,
		CreatedAt:   permEntity.CreatedAt,
		UpdatedAt:   permEntity.UpdatedAt,

		RiskLevel:         policy.RiskLevel,
		AllowedPrincipals: append([]string(nil), policy.AllowedPrincipals...),
		RequiresApproval:  policy.RequiresApproval,
		MinApprovals:      policy.MinApprovals,
		IsActive:          policy.IsActive,
		IsSystem:          policy.IsSystem,
		Configured:        configured,

		Category:      policy.Category,
		CategoryLabel: authz.CategoryLabel(policy.Category),
		DisplayName:   policy.DisplayName,
		Icon:          policy.Icon,
		Color:         policy.Color,
		I18nKey:       policy.I18nKey,
		SortOrder:     policy.SortOrder,
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
