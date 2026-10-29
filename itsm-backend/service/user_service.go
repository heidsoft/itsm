package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
	domainrole "itsm-backend/domain/role"
	"itsm-backend/dto"
	"itsm-backend/ent"
	entrole "itsm-backend/ent/role"
	"itsm-backend/ent/user"
	"itsm-backend/middleware"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewUserService(client *ent.Client, logger *zap.SugaredLogger) *UserService {
	return &UserService{
		client: client,
		logger: logger,
	}
}

// CreateUser 创建用户
func (s *UserService) CreateUser(ctx context.Context, req *dto.CreateUserRequest, tenantID int) (*ent.User, error) {
	s.logger.Infof("创建用户: %s", req.Username)

	// 用户名按租户查重：唯一键是 (tenant_id, username)，每个租户都可以有自己的 admin。
	// 邮箱仍按全局查重，因为找回密码只按 email 定位账号（handlers/auth/service.go）。
	exists, err := s.client.User.Query().
		Where(user.UsernameEQ(req.Username), user.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("检查用户名失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("用户名已存在: %s", req.Username)
	}

	// 检查邮箱是否已存在
	exists, err = s.client.User.Query().
		Where(user.EmailEQ(req.Email)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("检查邮箱失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("邮箱已存在: %s", req.Email)
	}

	// 加密密码（如果不提供则生成随机密码）
	password := req.Password
	if password == "" {
		password = fmt.Sprintf("P@ssw0rd%08d", time.Now().UnixNano()%100000000)
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 创建用户
	uc := s.client.User.Create().
		SetUsername(req.Username).
		SetEmail(req.Email).
		SetName(req.Name).
		SetDepartment(req.Department).
		SetPhone(req.Phone).
		SetPasswordHash(string(hashedPassword)).
		SetActive(true).
		SetTenantID(tenantID)
	// 如果请求中提供了角色，则设置角色；否则使用Schema默认值（end_user）。
	// service 层第二道防线：IsBuiltinCode 白名单拒绝未知/退役码，防 handler 校验被绕过
	// 后未知 code 落库（ent 枚举会拒但错误不友好，且未来枚举放宽时这里就是提权面）。
	primaryRole := ""
	if strings.TrimSpace(req.Role) != "" {
		primaryRole = strings.ToLower(strings.TrimSpace(req.Role))
		// 兼容前端传的"user"角色，自动转换为"end_user"
		if primaryRole == "user" {
			primaryRole = "end_user"
		}
		if !domainrole.IsBuiltinCode(primaryRole) {
			return nil, fmt.Errorf("无效的用户角色: %s", primaryRole)
		}
		uc = uc.SetRole(user.Role(primaryRole))

	}
	// 如果请求中提供了MSP角色，则设置MSP角色
	if strings.TrimSpace(req.MSPRole) != "" {
		uc = uc.SetMspRole(user.MspRole(strings.ToLower(strings.TrimSpace(req.MSPRole))))
	}
	userEntity, err := uc.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	// RBAC 多角色：显式 roleIds 优先；否则把主角色同步到 user_roles 边（存量回填同语义），
	// 使按角色解析审批人/M2M 权限链路能命中新用户。
	roleIDs := req.RoleIDs
	if len(roleIDs) == 0 && primaryRole != "" {
		if id, err := s.resolveRoleID(ctx, tenantID, primaryRole); err == nil {
			roleIDs = []int{id}
		} else {
			// 主角色无对应 roles 表实体（如 legacy "security"），仅告警不阻断建用户。
			s.logger.Warnw("主角色无对应 roles 表实体，user_roles 边未写入",
				"username", req.Username, "role", primaryRole, "error", err)
		}
	}
	if len(roleIDs) > 0 {
		if err := s.SyncUserRoles(ctx, userEntity.ID, tenantID, roleIDs); err != nil {
			return nil, fmt.Errorf("写入用户角色边失败: %w", err)
		}
	}

	s.logger.Infof("用户创建成功: ID=%d, Username=%s", userEntity.ID, userEntity.Username)
	return userEntity, nil
}

// resolveRoleID 按租户与 code 解析 roles 表实体 ID。
func (s *UserService) resolveRoleID(ctx context.Context, tenantID int, code string) (int, error) {
	r, err := s.client.Role.Query().
		Where(entrole.CodeEQ(code), entrole.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return 0, err
	}
	return r.ID, nil
}

// SyncUserRoles 整体替换用户的 user_roles M2M 边（仅接受本租户内的角色 ID，防跨租户提权）。
func (s *UserService) SyncUserRoles(ctx context.Context, userID, tenantID int, roleIDs []int) error {
	if len(roleIDs) == 0 {
		_, err := s.client.User.UpdateOneID(userID).ClearRoles().Save(ctx)
		return err
	}
	// 校验全部角色属于同一租户，避免恶意 roleIds 跨租户挂角色。
	n, err := s.client.Role.Query().
		Where(entrole.IDIn(roleIDs...), entrole.TenantIDEQ(tenantID)).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("校验角色失败: %w", err)
	}
	if n != len(roleIDs) {
		return fmt.Errorf("存在不属于当前租户的角色: 期望 %d 个，匹配 %d 个", len(roleIDs), n)
	}
	_, err = s.client.User.UpdateOneID(userID).ClearRoles().AddRoleIDs(roleIDs...).Save(ctx)
	return err
}

// CanGrantRoles 校验调用者是否有权授予指定角色集合（防 roleIds 越权提权）。
// 空 roleIDs 视为无操作，直接返回 nil。
func (s *UserService) CanGrantRoles(ctx context.Context, tenantID int, roleIDs []int, callerRole string) error {
	if len(roleIDs) == 0 {
		return nil
	}
	roles, err := s.client.Role.Query().
		Where(entrole.IDIn(roleIDs...), entrole.TenantIDEQ(tenantID)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("查询角色失败: %w", err)
	}
	if len(roles) != len(roleIDs) {
		return fmt.Errorf("存在不属于当前租户的角色: 期望 %d 个，匹配 %d 个", len(roleIDs), len(roles))
	}
	callerRank := serviceRoleRank(callerRole)
	for _, r := range roles {
		if serviceRoleRank(r.Code) > callerRank {
			return fmt.Errorf("无权限分配高于自身角色的用户角色: %s", r.Code)
		}
	}
	return nil
}

// serviceRoleRank 返回角色权限层级（与 handler 层同一词表，单一事实源在 domain/role.Rank）。
func serviceRoleRank(role string) int {
	return domainrole.Rank(strings.ToLower(strings.TrimSpace(role)))
}

// ListUsers 获取用户列表
func (s *UserService) ListUsers(ctx context.Context, req *dto.ListUsersRequest, tenantID int) (*dto.PagedUsersResponse, error) {
	s.logger.Infof("获取用户列表: page=%d, pageSize=%d", req.Page, req.PageSize)

	query := s.client.User.Query().
		Where(user.TenantIDEQ(tenantID))

	// 按状态过滤
	if req.Status != "" {
		active := req.Status == "active"
		query = query.Where(user.ActiveEQ(active))
	}

	// 按部门过滤
	if req.Department != "" {
		query = query.Where(user.DepartmentContainsFold(req.Department))
	}

	// 搜索过滤
	if req.Search != "" {
		search := strings.TrimSpace(req.Search)
		query = query.Where(
			user.Or(
				user.UsernameContainsFold(search),
				user.NameContainsFold(search),
				user.EmailContainsFold(search),
			),
		)
	}

	// 计算总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("统计用户总数失败: %w", err)
	}

	// 分页查询（预加载 roles 边供前端编辑表单回填）
	users, err := query.
		WithRoles().
		Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize).
		Order(ent.Desc(user.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	// 转换为响应格式
	userResponses := make([]*dto.UserDetailResponse, 0, len(users))
	for _, u := range users {
		roleIDs := make([]int, 0, len(u.Edges.Roles))
		roleNames := make([]string, 0, len(u.Edges.Roles))
		for _, r := range u.Edges.Roles {
			roleIDs = append(roleIDs, r.ID)
			roleNames = append(roleNames, r.Name)
		}
		userResponses = append(userResponses, &dto.UserDetailResponse{
			ID:         u.ID,
			Username:   u.Username,
			Email:      u.Email,
			Name:       u.Name,
			Department: u.Department,
			Phone:      u.Phone,
			Active:     u.Active,
			TenantID:   u.TenantID,
			Role:       string(u.Role),
			RoleIDs:    roleIDs,
			RoleNames:  roleNames,
			CreatedAt:  u.CreatedAt,
			UpdatedAt:  u.UpdatedAt,
		})
	}

	pagination := common.NewPaginationResponse(req.Page, req.PageSize, int64(total))
	response := &dto.PagedUsersResponse{
		Items:      userResponses,
		Total:      pagination.Total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: pagination.TotalPages,
	}

	s.logger.Infof("用户列表查询成功: total=%d, returned=%d", total, len(users))
	return response, nil
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(ctx context.Context, id int, tenantID int) (*ent.User, error) {
	s.logger.Infof("获取用户详情: ID=%d", id)

	userEntity, err := s.client.User.Query().
		Where(
			user.IDEQ(id),
			user.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("用户不存在: ID=%d", id)
		}
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	return userEntity, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, id int, req *dto.UpdateUserRequest, tenantID int) (*ent.User, error) {
	s.logger.Infof("更新用户: ID=%d", id)

	// 验证用户属于当前租户，防止跨租户访问
	existingUser, err := s.client.User.Query().
		Where(user.IDEQ(id), user.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("用户不存在: ID=%d", id)
		}
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	update := s.client.User.UpdateOneID(id).Where(user.TenantIDEQ(tenantID))

	// 检查用户名是否已被本租户其他用户使用（唯一键按租户组合，跨租户同名合法）
	if req.Username != "" && req.Username != existingUser.Username {
		exists, err := s.client.User.Query().
			Where(
				user.And(
					user.UsernameEQ(req.Username),
					user.TenantIDEQ(tenantID),
					user.IDNEQ(id),
				),
			).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("检查用户名失败: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("用户名已存在: %s", req.Username)
		}
		update = update.SetUsername(req.Username)
	}

	// 检查邮箱是否已被其他用户使用
	if req.Email != "" && req.Email != existingUser.Email {
		exists, err := s.client.User.Query().
			Where(
				user.And(
					user.EmailEQ(req.Email),
					user.IDNEQ(id),
				),
			).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("检查邮箱失败: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("邮箱已存在: %s", req.Email)
		}
		update = update.SetEmail(req.Email)
	}

	// 更新其他字段
	if req.Name != "" {
		update = update.SetName(req.Name)
	}
	// 指针字段：非 nil 即提交，空串即清空（区别于「未传」）
	if req.Department != nil {
		update = update.SetDepartment(*req.Department)
	}
	if req.Phone != nil {
		update = update.SetPhone(*req.Phone)
	}
	// 角色更新（仅在提供时设置），管理员权限由RBAC控制。
	// service 层第二道防线：白名单拒绝未知/退役码（同 CreateUser）。
	if strings.TrimSpace(req.Role) != "" {
		role := strings.ToLower(strings.TrimSpace(req.Role))
		// 兼容前端传的"user"角色，自动转换为"end_user"
		if role == "user" {
			role = "end_user"
		}
		if !domainrole.IsBuiltinCode(role) {
			return nil, fmt.Errorf("无效的用户角色: %s", role)
		}
		if role != string(existingUser.Role) {
			update = update.SetRole(user.Role(role))
			// 角色变更后立即吊销该用户全部存量 access token，
			// 防止旧角色权限在 token 有效期内（15 分钟）继续生效。
			if err := middleware.InvalidateUserTokens(ctx, id, time.Now()); err != nil {
				// 吊销失败不阻断角色变更本身，但必须记录：降权延迟窗口存在安全影响。
				s.logger.Errorw("用户角色变更后吊销存量token失败（降权延迟风险）",
					"user_id", id, "old_role", existingUser.Role, "new_role", role, "error", err)
			} else {
				s.logger.Infow("用户角色变更，已吊销存量access token", "user_id", id, "new_role", role)
			}
		}
	}

	userEntity, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	// RBAC 多角色替换：仅在显式提供 roleIds 时整体替换 user_roles 边
	// （nil=不动；空数组=清空），避免部分更新语义破坏多角色并集。
	if req.RoleIDs != nil {
		if err := s.SyncUserRoles(ctx, id, tenantID, req.RoleIDs); err != nil {
			return nil, fmt.Errorf("替换用户角色边失败: %w", err)
		}
		// 角色集合变更影响权限，与主角色变更同语义：吊销存量 token。
		if err := middleware.InvalidateUserTokens(ctx, id, time.Now()); err != nil {
			s.logger.Errorw("用户多角色变更后吊销存量token失败（降权延迟风险）",
				"user_id", id, "role_ids", req.RoleIDs, "error", err)
		}
	}

	s.logger.Infof("用户更新成功: ID=%d", id)
	return userEntity, nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(ctx context.Context, id int, tenantID int) error {
	s.logger.Infof("删除用户: ID=%d", id)

	// 检查用户是否存在
	_, err := s.GetUserByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	// 软删语义（2026-10-29）：deleted_at 与「停用」分离。
	//   删除 = active=false + deleted_at=now —— 读路径经 RegisterSoftDeleteInterceptors
	//          屏蔽（管理列表/搜索/WithXxx 关联），且启用 toggle 不可见、无法复活；
	//   停用 = 仅 active=false（ChangeUserStatus），仍可见可重新启用。
	// active=false 保留为纵深：登录复核 u.Active（handlers/common/service.go），
	// 即使任何路径绕过拦截器读到行，被删用户也无法登录。
	now := time.Now()
	err = s.client.User.UpdateOneID(id).
		Where(user.TenantIDEQ(tenantID)).
		SetActive(false).
		SetDeletedAt(now).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	// 立即吊销全部存量 access token：软删只是标记 + SetActive(false)，AuthMiddleware
	// 纯 JWT 验签不查 Active，不吊销的话被删用户在 token 有效期内仍可调用仅
	// AuthMiddleware 保护的路由（/users/profile、/auth/tenants、WS 等）。
	if revokeErr := middleware.InvalidateUserTokens(ctx, id, now); revokeErr != nil {
		s.logger.Errorw("删除用户后吊销存量token失败", "user_id", id, "error", revokeErr)
	} else {
		s.logger.Infow("用户已删除，存量access token已吊销", "user_id", id)
	}

	s.logger.Infof("用户删除成功: ID=%d", id)
	return nil
}

// ChangeUserStatus 更改用户状态
// currentUserID 是当前操作的用户ID，用于防止用户停用自己
func (s *UserService) ChangeUserStatus(ctx context.Context, id int, active bool, currentUserID int, tenantID int) error {
	s.logger.Infof("更改用户状态: ID=%d, active=%t, currentUserID=%d", id, active, currentUserID)

	// 检查用户是否存在
	_, err := s.GetUserByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	// 防止用户停用自己
	if id == currentUserID && !active {
		return fmt.Errorf("不能停用当前登录用户")
	}

	err = s.client.User.UpdateOneID(id).
		Where(user.TenantIDEQ(tenantID)).
		SetActive(active).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("更改用户状态失败: %w", err)
	}

	// 停用账户时立即吊销其全部存量 access token。
	if !active {
		if revokeErr := middleware.InvalidateUserTokens(ctx, id, time.Now()); revokeErr != nil {
			s.logger.Errorw("停用用户后吊销存量token失败", "user_id", id, "error", revokeErr)
		} else {
			s.logger.Infow("用户已停用，存量access token已吊销", "user_id", id)
		}
	}

	s.logger.Infof("用户状态更改成功: ID=%d, active=%t", id, active)
	return nil
}

// ResetPassword 重置用户密码
func (s *UserService) ResetPassword(ctx context.Context, id int, newPassword string, tenantID int) error {
	s.logger.Infof("重置用户密码: ID=%d", id)

	// 检查用户是否存在
	_, err := s.GetUserByID(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if err := validatePassword(newPassword); err != nil {
		return err
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	err = s.client.User.UpdateOneID(id).
		Where(user.TenantIDEQ(tenantID)).
		SetPasswordHash(string(hashedPassword)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("重置密码失败: %w", err)
	}

	// 密码重置后立即吊销该用户全部存量 access token（防旧会话存活）。
	if revokeErr := middleware.InvalidateUserTokens(ctx, id, time.Now()); revokeErr != nil {
		s.logger.Errorw("密码重置后吊销存量token失败", "user_id", id, "error", revokeErr)
	}

	s.logger.Infof("用户密码重置成功: ID=%d", id)
	return nil
}

// GetUserStats 获取用户统计信息
func (s *UserService) GetUserStats(ctx context.Context, tenantID int) (*dto.UserStatsResponse, error) {
	s.logger.Infof("获取用户统计: tenantID=%d", tenantID)

	if tenantID <= 0 {
		return nil, fmt.Errorf("租户信息无效")
	}
	query := s.client.User.Query().Where(user.TenantIDEQ(tenantID))

	// 总用户数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("统计总用户数失败: %w", err)
	}

	// 活跃用户数
	active, err := query.Clone().Where(user.ActiveEQ(true)).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("统计活跃用户数失败: %w", err)
	}

	monthStart := time.Now().In(time.Local)
	monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, monthStart.Location())
	newThisMonth, err := query.Clone().Where(user.CreatedAtGTE(monthStart)).Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("统计本月新增用户失败: %w", err)
	}

	response := &dto.UserStatsResponse{
		Total:  total,
		Active: active,
		// 当前模型没有登录会话/心跳数据，返回0，避免把“启用”误报为“在线”。
		Online:       0,
		ByRole:       map[string]int{},
		ByDepartment: map[string]int{},
		NewThisMonth: newThisMonth,
	}

	s.logger.Infow("用户统计获取成功", "total", total, "active", active, "online", 0, "new_this_month", newThisMonth)
	return response, nil
}

// BatchUpdateUsers 批量更新用户
func (s *UserService) BatchUpdateUsers(ctx context.Context, req *dto.BatchUpdateUsersRequest, tenantID int) error {
	s.logger.Infof("批量更新用户: count=%d", len(req.UserIDs))

	if len(req.UserIDs) == 0 {
		return fmt.Errorf("用户ID列表不能为空")
	}

	ids := uniquePositiveIDs(req.UserIDs)
	if len(ids) == 0 {
		return fmt.Errorf("用户ID列表不能为空")
	}
	if req.Action == "deactivate" && req.OperatorID > 0 {
		for _, id := range ids {
			if id == req.OperatorID {
				return fmt.Errorf("批量停用不能包含当前登录用户")
			}
		}
	}
	matched, err := s.client.User.Query().Where(user.IDIn(ids...), user.TenantIDEQ(tenantID)).Count(ctx)
	if err != nil {
		return fmt.Errorf("校验批量用户失败: %w", err)
	}
	if matched != len(ids) {
		return fmt.Errorf("部分用户不存在或不属于当前租户，请刷新列表后重试")
	}

	update := s.client.User.Update().
		Where(
			user.IDIn(ids...),
			user.TenantIDEQ(tenantID),
		)

	// 根据操作类型更新
	switch req.Action {
	case "activate":
		update = update.SetActive(true)
	case "deactivate":
		update = update.SetActive(false)
	case "department":
		if req.Department == "" {
			return fmt.Errorf("部门不能为空")
		}
		update = update.SetDepartment(req.Department)
	default:
		return fmt.Errorf("不支持的操作类型: %s", req.Action)
	}

	count, err := update.Save(ctx)
	if err != nil {
		return fmt.Errorf("批量更新用户失败: %w", err)
	}

	s.logger.Infof("批量更新用户成功: updated=%d", count)
	return nil
}

// validatePassword 返回 common.ErrCodeValidation 分类错误，让 handler 能映射成 400；
// 用普通 error 会被 handler 归为内部错误（500），把输入问题伪装成服务端故障。
// ChangeOwnPassword 用户自助修改密码。
//
// 与 ResetPassword 的区别（这是两个不同权限面，不能互相顶替）：
//   - ResetPassword：管理员重置他人密码，不校验旧密码
//   - ChangeOwnPassword：用户改自己的密码，**必须校验旧密码**，
//     并拒绝新旧相同
//
// 返回的错误语义：
//   - 旧密码错误 → ForbiddenCode（不暴露"用户存在"这一信息）
//   - 新密码不合规 / 与旧密码相同 → ValidationError
func (s *UserService) ChangeOwnPassword(ctx context.Context, userID int, oldPassword, newPassword string, tenantID int) error {
	if userID <= 0 {
		return common.NewBusinessError(common.UnauthorizedCode, "未授权访问", "")
	}

	// 变量名避开 user —— 该包已 import ent/user，遮蔽后
	// user.TenantIDEQ 会解析成 *ent.User 上的方法（不存在）导致编译失败。
	target, err := s.GetUserByID(ctx, userID, tenantID)
	if err != nil {
		return err
	}

	// 先校验旧密码。这里刻意统一返回 Forbidden 而不是区分"用户不存在"，
	// 避免探测接口被用来枚举账号。
	if bcrypt.CompareHashAndPassword([]byte(target.PasswordHash), []byte(oldPassword)) != nil {
		s.logger.Warnf("自助改密失败（旧密码错误）: user_id=%d", userID)
		return common.NewBusinessError(common.ForbiddenCode, "当前密码不正确", "")
	}

	if err := validatePassword(newPassword); err != nil {
		return err
	}

	if oldPassword == newPassword {
		return common.NewValidationError("新密码不能与当前密码相同", nil)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	if err := s.client.User.UpdateOneID(userID).
		Where(user.TenantIDEQ(tenantID)).
		SetPasswordHash(string(hashedPassword)).
		Exec(ctx); err != nil {
		return fmt.Errorf("修改密码失败: %w", err)
	}

	// 与 ResetPassword 一致：改密后吊销全部存量 access token，
	// 否则旧会话会在改密后继续存活 —— 改密最常见的动机恰恰是"可能泄露了"。
	if revokeErr := middleware.InvalidateUserTokens(ctx, userID, time.Now()); revokeErr != nil {
		s.logger.Errorw("改密后吊销存量token失败", "user_id", userID, "error", revokeErr)
	}

	s.logger.Infof("用户自助改密成功: ID=%d", userID)
	return nil
}

// UpdateOwnProfile 更新调用者本人的资料（自助面，对应 PUT /api/v1/account/profile）。
//
// 与 UpdateUser（管理面 PUT /users/:id，需 user:write）分属两个权限面：自助面只挂认证，
// 字段白名单由 DTO 结构限定，这里再显式只写 name/email/phone/department ——
// username/role/active/tenant 永不接受，端点因此不具备提权能力。
//
// 邮箱唯一性沿用管理面口径（跨租户全局唯一），避免自助面绕开约束建出重复登录邮箱。
func (s *UserService) UpdateOwnProfile(ctx context.Context, userID, tenantID int, req *dto.UpdateOwnProfileRequest) (*ent.User, error) {
	if userID <= 0 {
		return nil, common.NewBusinessError(common.UnauthorizedCode, "未授权访问", "")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, common.NewValidationError("姓名不能为空", nil)
	}
	// 邮箱是登录身份，不属于自助面可清空字段：留空必须拒绝而不是静默跳过，
	// 否则任何绕过 DTO binding 的调用方都能把账号改成无法登录。
	if strings.TrimSpace(req.Email) == "" {
		return nil, common.NewValidationError("邮箱不能为空", nil)
	}

	target, err := s.client.User.Query().
		Where(user.IDEQ(userID), user.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// 自助面的 id 来自鉴权上下文，正常永远命中；走到这里说明 JWT 里的租户与库内
			// 归属已不一致（切租户后的旧会话）。按 not-found 掩码，不回显租户细节。
			return nil, common.NewBusinessError(common.NotFoundCode, "用户资料不存在", "")
		}
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	update := s.client.User.UpdateOneID(userID).
		Where(user.TenantIDEQ(tenantID)).
		SetName(name)

	if req.Email != target.Email {
		exists, err := s.client.User.Query().
			Where(user.And(user.EmailEQ(req.Email), user.IDNEQ(userID))).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("检查邮箱失败: %w", err)
		}
		if exists {
			// 自助表单唯一可复现的冲突，必须带可展示文案：返回未分类错误会让
			// 前端只能显示「更新个人资料失败」，用户不知道要换一个邮箱。
			return nil, common.NewBusinessError(common.ConflictCode, "该邮箱已被其他账号使用", "")
		}
		update = update.SetEmail(req.Email)
	}

	if req.Phone != nil {
		update = update.SetPhone(strings.TrimSpace(*req.Phone))
	}
	if req.Department != nil {
		update = update.SetDepartment(strings.TrimSpace(*req.Department))
	}

	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新个人资料失败: %w", err)
	}

	s.logger.Infof("用户自助更新资料成功: ID=%d", userID)
	return updated, nil
}

func validatePassword(password string) error {
	if len(password) < 12 || len(password) > 128 {
		return common.NewValidationError("密码长度必须为12到128位", nil)
	}
	var upper, lower, digit, special bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= '0' && r <= '9':
			digit = true
		default:
			special = true
		}
	}
	if !upper || !lower || !digit || !special {
		return common.NewValidationError("密码必须同时包含大写字母、小写字母、数字和特殊字符", nil)
	}
	return nil
}

// SearchUsers 搜索用户
func (s *UserService) SearchUsers(ctx context.Context, req *dto.SearchUsersRequest, tenantID int) ([]*dto.UserDetailResponse, error) {
	s.logger.Infof("搜索用户: keyword=%s", req.Keyword)

	if req.Keyword == "" {
		return []*dto.UserDetailResponse{}, nil
	}

	query := s.client.User.Query().
		Where(
			user.Or(
				user.UsernameContainsFold(req.Keyword),
				user.NameContainsFold(req.Keyword),
				user.EmailContainsFold(req.Keyword),
			),
		)

	query = query.Where(user.TenantIDEQ(tenantID))

	// 只返回活跃用户
	query = query.Where(user.ActiveEQ(true))

	users, err := query.
		Limit(req.PageSize).
		Order(ent.Asc(user.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("搜索用户失败: %w", err)
	}

	// 转换为响应格式
	userResponses := make([]*dto.UserDetailResponse, 0, len(users))
	for _, u := range users {
		userResponses = append(userResponses, &dto.UserDetailResponse{
			ID:         u.ID,
			Username:   u.Username,
			Email:      u.Email,
			Name:       u.Name,
			Department: u.Department,
			Phone:      u.Phone,
			Active:     u.Active,
			TenantID:   u.TenantID,
			Role:       string(u.Role),
			CreatedAt:  u.CreatedAt,
			UpdatedAt:  u.UpdatedAt,
		})
	}

	s.logger.Infof("用户搜索成功: found=%d", len(users))
	return userResponses, nil
}
