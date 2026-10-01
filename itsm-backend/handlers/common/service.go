package common

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	enttenant "itsm-backend/ent/tenant"
	entuser "itsm-backend/ent/user"
	"itsm-backend/middleware"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      Repository
	jwtSecret string
	logger    *zap.SugaredLogger
	client    *ent.Client // For legacy integrations if needed
}

func NewService(repo Repository, jwtSecret string, logger *zap.SugaredLogger, client *ent.Client) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		logger:    logger,
		client:    client,
	}
}

// Auth

// getUserPermissions 获取用户的权限列表
func (s *Service) getUserPermissions(role string) []string {
	permissions := make([]string, 0)

	// 超级管理员拥有所有权限
	if role == "super_admin" {
		return []string{"*"}
	}

	// 从 middleware.RolePermissions 获取角色权限
	rolePerms, ok := middleware.RolePermissions[role]
	if !ok {
		return permissions
	}

	seen := make(map[string]bool)
	for _, p := range rolePerms {
		key := p.Resource + ":" + p.Action
		if !seen[key] {
			seen[key] = true
			permissions = append(permissions, key)
		}
	}

	return permissions
}

func (s *Service) Login(ctx context.Context, username, password string, tenantID int, tenantCode string) (*AuthResult, error) {
	// Resolve tenant
	if tenantID == 0 && tenantCode != "" {
		t, err := s.client.Tenant.Query().Where(enttenant.CodeEQ(tenantCode)).First(ctx)
		if err == nil {
			tenantID = t.ID
		}
	}
	// When no tenant is specified, find user by username alone (matches across tenants)
	var u *User
	var entUser *ent.User
	var err error
	if tenantID == 0 {
		// Look for user by username without tenant filter
		entUser, err = s.client.User.Query().Where(entuser.UsernameEQ(username)).Only(ctx)
		if err != nil {
			middleware.RecordLoginAudit(ctx, s.client, 0, tenantID, username, "LOGIN_FAILED", "用户不存在")
			return nil, fmt.Errorf("invalid credentials")
		}
		u = toUserDomain(entUser)
	} else {
		entUser, err = s.client.User.Query().
			Where(entuser.UsernameEQ(username), entuser.TenantID(tenantID)).
			Only(ctx)
		if err != nil {
			middleware.RecordLoginAudit(ctx, s.client, 0, tenantID, username, "LOGIN_FAILED", "用户不存在")
			return nil, fmt.Errorf("invalid credentials")
		}
		u = toUserDomain(entUser)
	}

	// Set msp_role from ent user
	mspRoleStr := string(entUser.MspRole)
	if mspRoleStr != "" {
		u.MSPRole = &mspRoleStr
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(entUser.PasswordHash), []byte(password)); err != nil {
		middleware.RecordLoginAudit(ctx, s.client, 0, entUser.TenantID, username, "LOGIN_FAILED", "密码错误")
		return nil, fmt.Errorf("invalid credentials")
	}

	if !u.Active {
		middleware.RecordLoginAudit(ctx, s.client, 0, entUser.TenantID, username, "LOGIN_FAILED", "账户锁定")
		return nil, fmt.Errorf("user account is inactive")
	}

	// 对于 MSP 用户，需要将 MSP 角色转换为 RBAC 角色
	// u.Role 是数据库中存储的 RBAC 角色（MSP 用户的 Role 是 admin）
	// 如果用户有 MSP 角色，则从 MSP 角色映射到正确的 RBAC 角色
	if mspRoleStr != "" {
		if mappedRole := middleware.GetMSPRBACRole(mspRoleStr); mappedRole != "" {
			u.Role = mappedRole
		}
	}

	// Generate tokens
	accessToken, err := middleware.GenerateAccessToken(u.ID, u.Username, u.Role, u.TenantID, s.jwtSecret, middleware.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	refreshToken, err := middleware.GenerateRefreshToken(u.ID, u.Username, u.Role, u.TenantID, s.jwtSecret, middleware.RefreshTokenTTL)
	if err != nil {
		return nil, err
	}

	// 获取用户权限
	u.Permissions = s.getUserPermissions(u.Role)
	middleware.RecordLoginAudit(ctx, s.client, entUser.ID, entUser.TenantID, username, "LOGIN_SUCCESS", "")

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(middleware.AccessTokenTTL.Seconds()),
		User:         u,
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error) {
	claims, err := middleware.ValidateRefreshToken(refreshToken, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}
	if claims.ExpiresAt == nil || claims.IssuedAt == nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// 吊销状态只有一份真相（middleware 的共享存储）：无 Redis 时退化为进程内存储，
	// 但绝不再静默跳过检查——那会让单副本部署的旧 refresh token 可以无限重放。
	revoked, err := middleware.IsRefreshTokenRevoked(ctx, refreshToken)
	if err != nil {
		s.logger.Errorw("refresh token revocation check failed, deny by default", "user_id", claims.UserID, "error", err)
		return nil, fmt.Errorf("refresh token validation failed")
	}
	if revoked {
		s.logger.Warnw("refresh token replay detected", "user_id", claims.UserID)
		return nil, fmt.Errorf("refresh token has been revoked")
	}

	// 改密/停用/降权后的签发约束同样适用于 refresh token，否则 7 天凭证可以绕过重新登录。
	minIAT, err := middleware.UserTokenMinIssuedAt(ctx, claims.UserID)
	if err != nil {
		s.logger.Errorw("refresh token min issued-at check failed, deny by default", "user_id", claims.UserID, "error", err)
		return nil, fmt.Errorf("refresh token validation failed")
	}
	if !minIAT.IsZero() && claims.IssuedAt.Time.Before(minIAT) {
		s.logger.Warnw("refresh token issued before account change, rejected",
			"user_id", claims.UserID, "issued_at", claims.IssuedAt.Time, "min_issued_at", minIAT)
		return nil, fmt.Errorf("账号信息已变更，请重新登录")
	}

	user, err := s.repo.GetUserByID(ctx, claims.UserID, claims.TenantID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	if !user.Active {
		return nil, fmt.Errorf("user account is inactive")
	}

	// 原子认领旧 token：并发续签里只有第一个请求能完成 rotation，
	// 第二个即使拿着同一个 token 也只能失败。
	claimed, err := middleware.RevokeRefreshToken(ctx, refreshToken, claims.ExpiresAt.Time)
	if err != nil {
		s.logger.Errorw("failed to revoke old refresh token during rotation", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("refresh token revocation unavailable")
	}
	if !claimed {
		s.logger.Warnw("refresh token rotation race detected", "user_id", user.ID)
		return nil, fmt.Errorf("refresh token has been revoked")
	}

	// regenerate tokens
	accessToken, err := middleware.GenerateAccessToken(user.ID, user.Username, user.Role, user.TenantID, s.jwtSecret, middleware.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	newRefresh, err := middleware.GenerateRefreshToken(user.ID, user.Username, user.Role, user.TenantID, s.jwtSecret, middleware.RefreshTokenTTL)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		ExpiresIn:    int(middleware.AccessTokenTTL.Seconds()),
		User:         user,
	}, nil
}

// User Management

// GetUser 获取用户信息（/auth/me 数据源）。
// 前端刷新页面后由 AuthGuard 重建 user，若此处不带 permissions，
// hasPermission 会全部返回 false，导致 Sidebar 管理功能区等权限驱动 UI 消失。
// 因此与 Login 相同，按角色填充权限列表（super_admin → ["*"]）。
func (s *Service) GetUser(ctx context.Context, id int, tenantID int) (*User, error) {
	u, err := s.repo.GetUserByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if len(u.Permissions) == 0 {
		u.Permissions = s.getUserPermissions(u.Role)
	}
	return u, nil
}

func (s *Service) ListUsers(ctx context.Context, tenantID int) ([]*User, error) {
	return s.repo.ListUsers(ctx, tenantID)
}

// Organization Management

func (s *Service) GetDepartment(ctx context.Context, id int, tenantID int) (*Department, error) {
	return s.repo.GetDepartment(ctx, id, tenantID)
}

func (s *Service) GetDepartmentTree(ctx context.Context, tenantID int) ([]*Department, error) {
	return s.repo.GetDepartmentTree(ctx, tenantID)
}

func (s *Service) ListDepartments(ctx context.Context, tenantID int) ([]*Department, error) {
	return s.repo.ListDepartments(ctx, tenantID)
}

func (s *Service) CreateDepartment(ctx context.Context, d *Department) (*Department, error) {
	return s.repo.CreateDepartment(ctx, d)
}

func (s *Service) UpdateDepartment(ctx context.Context, d *Department) (*Department, error) {
	return s.repo.UpdateDepartment(ctx, d)
}

func (s *Service) DeleteDepartment(ctx context.Context, id int, tenantID int) error {
	return s.repo.DeleteDepartment(ctx, id, tenantID)
}

func (s *Service) ListTeams(ctx context.Context, tenantID int) ([]*Team, error) {
	return s.repo.ListTeams(ctx, tenantID)
}

func (s *Service) GetTeam(ctx context.Context, id int, tenantID int) (*Team, error) {
	return s.repo.GetTeam(ctx, id, tenantID)
}

func (s *Service) CreateTeam(ctx context.Context, t *Team) (*Team, error) {
	return s.repo.CreateTeam(ctx, t)
}

func (s *Service) UpdateTeam(ctx context.Context, t *Team) (*Team, error) {
	return s.repo.UpdateTeam(ctx, t)
}

func (s *Service) DeleteTeam(ctx context.Context, id int, tenantID int) error {
	return s.repo.DeleteTeam(ctx, id, tenantID)
}

func (s *Service) AddTeamMember(ctx context.Context, teamID int, userID int) error {
	return s.repo.AddTeamMember(ctx, teamID, userID)
}

// Tags

func (s *Service) ListTags(ctx context.Context, tenantID int) ([]*Tag, error) {
	return s.repo.ListTags(ctx, tenantID)
}

func (s *Service) CreateTag(ctx context.Context, t *Tag) (*Tag, error) {
	return s.repo.CreateTag(ctx, t)
}

// Auditing

func (s *Service) LogActivity(ctx context.Context, log *AuditLog) error {
	return s.repo.CreateAuditLog(ctx, log)
}

func (s *Service) GetAuditLogs(ctx context.Context, tenantID int, userID int) ([]*AuditLog, error) {
	return s.repo.ListAuditLogs(ctx, tenantID, userID, 100)
}

// GetUserTenants 获取用户可访问的租户列表。
func (s *Service) GetUserTenants(ctx context.Context, userID int) ([]TenantBrief, error) {
	// 直接使用 ent client 查询用户关联的租户
	user, err := s.client.User.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 通过 tenant_id 直接查询租户
	tenant, err := s.client.Tenant.Get(ctx, user.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return []TenantBrief{{
		ID:     tenant.ID,
		Name:   tenant.Name,
		Code:   tenant.Code,
		Type:   string(tenant.Type),
		Status: tenant.Status,
	}}, nil
}

// GetSession 组装「当前会话」的唯一后端真相。
// 身份与权限复用 GetUser，避免前端再拼第二个探活请求；accessExpiresIn 由调用方
// 从已认证 token 的服务端签发时间给出，禁止前端用浏览器时钟推算。
func (s *Service) GetSession(ctx context.Context, userID int, tenantID int, accessExpiresIn int) (*SessionResponse, error) {
	u, err := s.GetUser(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}
	tenants, err := s.GetUserTenants(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &SessionResponse{
		User:      u,
		Tenants:   tenants,
		ExpiresIn: accessExpiresIn,
	}, nil
}
