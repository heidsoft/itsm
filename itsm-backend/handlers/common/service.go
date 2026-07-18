package common

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	enttenant "itsm-backend/ent/tenant"
	entuser "itsm-backend/ent/user"
	"itsm-backend/middleware"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      Repository
	jwtSecret string
	logger    *zap.SugaredLogger
	client    *ent.Client // For legacy integrations if needed
	redis     *redis.Client
}

func NewService(repo Repository, jwtSecret string, logger *zap.SugaredLogger, client *ent.Client) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		logger:    logger,
		client:    client,
	}
}

// SetRedis 注入 Redis 客户端；启用 refresh token 黑名单（token rotation 后旧值失效）
func (s *Service) SetRedis(r *redis.Client) {
	s.redis = r
}

// refreshBlacklistKey Redis key for refresh token blacklist
func refreshBlacklistKey(token string) string { return "refresh:blacklist:" + token }

// blacklistRefreshToken 将 refresh token 加入黑名单，TTL = 剩余有效期
func (s *Service) blacklistRefreshToken(ctx context.Context, token string, expiresAt time.Time) error {
	if s.redis == nil {
		return nil // 未注入 redis：降级为无黑名单（记 warn 由调用方处理）
	}
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}
	return s.redis.Set(ctx, refreshBlacklistKey(token), "1", ttl).Err()
}

// isRefreshBlacklisted 检查 refresh token 是否已拉黑
func (s *Service) isRefreshBlacklisted(ctx context.Context, token string) (bool, error) {
	if s.redis == nil {
		return false, nil
	}
	n, err := s.redis.Exists(ctx, refreshBlacklistKey(token)).Result()
	return n > 0, err
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
			return nil, fmt.Errorf("invalid credentials")
		}
		u = toUserDomain(entUser)
	} else {
		entUser, err = s.client.User.Query().
			Where(entuser.UsernameEQ(username), entuser.TenantID(tenantID)).
			Only(ctx)
		if err != nil {
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
		return nil, fmt.Errorf("invalid credentials")
	}

	if !u.Active {
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
	accessToken, err := middleware.GenerateAccessToken(u.ID, u.Username, u.Role, u.TenantID, s.jwtSecret, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	refreshToken, err := middleware.GenerateRefreshToken(u.ID, s.jwtSecret, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	// 获取用户权限
	u.Permissions = s.getUserPermissions(u.Role)

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         u,
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error) {
	claims, err := middleware.ValidateRefreshToken(refreshToken, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// 黑名单检查：token rotation 后旧值不允许再使用
	if blacklisted, chkErr := s.isRefreshBlacklisted(ctx, refreshToken); chkErr != nil {
		s.logger.Warnw("refresh blacklist check failed, deny by default", "error", chkErr)
		return nil, fmt.Errorf("refresh token validation failed")
	} else if blacklisted {
		s.logger.Warnw("refresh token replay detected", "user_id", claims.UserID)
		return nil, fmt.Errorf("refresh token has been revoked")
	}

	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// regenerate tokens
	accessToken, err := middleware.GenerateAccessToken(user.ID, user.Username, user.Role, user.TenantID, s.jwtSecret, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	newRefresh, err := middleware.GenerateRefreshToken(user.ID, s.jwtSecret, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	// 拉黑旧 refresh token，TTL = 剩余有效期
	if claims.ExpiresAt != nil {
		if bErr := s.blacklistRefreshToken(ctx, refreshToken, claims.ExpiresAt.Time); bErr != nil {
			s.logger.Warnw("failed to blacklist old refresh token", "user_id", user.ID, "error", bErr)
		}
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		User:         user,
	}, nil
}

// User Management

func (s *Service) GetUser(ctx context.Context, id int) (*User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *Service) ListUsers(ctx context.Context, tenantID int) ([]*User, error) {
	return s.repo.ListUsers(ctx, tenantID)
}

// Organization Management

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

// GetUserTenants 获取用户所属的租户列表
func (s *Service) GetUserTenants(ctx context.Context, userID int) ([]interface{}, error) {
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

	if tenant == nil {
		return []interface{}{}, nil
	}

	return []interface{}{
		map[string]interface{}{
			"id":     tenant.ID,
			"name":   tenant.Name,
			"code":   tenant.Code,
			"type":   tenant.Type,
			"status": tenant.Status,
		},
	}, nil
}
