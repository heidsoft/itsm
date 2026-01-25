package service

import (
    "context"
    "fmt"
    "itsm-backend/dto"
    "itsm-backend/ent"
    "itsm-backend/ent/tenant"
    "itsm-backend/ent/user"
    "itsm-backend/middleware"
    "time"

    "golang.org/x/crypto/bcrypt"
    "go.uber.org/zap"
)

type AuthService struct {
	client          *ent.Client
	jwtSecret       string
	logger          *zap.SugaredLogger
	tokenBlacklist  *TokenBlacklistService
}

func NewAuthService(client *ent.Client, jwtSecret string, logger *zap.SugaredLogger, blacklistService *TokenBlacklistService) *AuthService {
	return &AuthService{
		client:         client,
		jwtSecret:      jwtSecret,
		logger:         logger,
		tokenBlacklist: blacklistService,
	}
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
	// 查询用户
	userQuery := s.client.User.Query().Where(user.UsernameEQ(req.Username))

	// 如果提供了租户代码，则按租户过滤
	if req.TenantCode != "" {
		// 先查询租户
		tenantEntity, err := s.client.Tenant.Query().
			Where(tenant.CodeEQ(req.TenantCode)).
			First(ctx)
		if err != nil {
			s.logger.Warnw("Tenant not found", "tenant_code", req.TenantCode, "error", err)
			return nil, fmt.Errorf("用户名或密码错误")
		}
		userQuery = userQuery.Where(user.TenantIDEQ(tenantEntity.ID))
	}

	userEntity, err := userQuery.First(ctx)
	if err != nil {
		s.logger.Warnw("User not found", "username", req.Username, "error", err)
		return nil, fmt.Errorf("用户名或密码错误")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(userEntity.PasswordHash), []byte(req.Password)); err != nil {
		s.logger.Warnw("Password verification failed", "user_id", userEntity.ID, "username", req.Username)
		return nil, fmt.Errorf("用户名或密码错误")
	}

	// 检查用户是否激活
	if !userEntity.Active {
		s.logger.Warnw("User account is inactive", "user_id", userEntity.ID, "username", req.Username)
		return nil, fmt.Errorf("用户已被禁用")
	}

	// 检查租户状态
	tenantEntity, err := s.client.Tenant.Get(ctx, userEntity.TenantID)
	if err != nil {
		s.logger.Errorw("Failed to get tenant", "tenant_id", userEntity.TenantID, "error", err)
		return nil, fmt.Errorf("租户不存在")
	}

	if tenantEntity.Status != "active" {
		s.logger.Warnw("Tenant is not active", "tenant_id", tenantEntity.ID, "status", tenantEntity.Status)
		return nil, fmt.Errorf("租户已被暂停或过期")
	}

    // 使用数据库中的角色字段，确保JWT与RBAC一致
    role := userEntity.Role

    // 生成access token（15分钟）
    accessToken, err := middleware.GenerateAccessToken(
        userEntity.ID,
        userEntity.Username,
        string(role),
        userEntity.TenantID,
        s.jwtSecret,
        time.Duration(15)*time.Minute,
    )
	if err != nil {
		s.logger.Errorw("Failed to generate access token", "user_id", userEntity.ID, "error", err)
		return nil, fmt.Errorf("生成访问令牌失败")
	}

	// 生成refresh token（7天）
	refreshToken, err := middleware.GenerateRefreshToken(
		userEntity.ID,
		s.jwtSecret,
		time.Duration(7*24)*time.Hour,
	)
	if err != nil {
		s.logger.Errorw("Failed to generate refresh token", "user_id", userEntity.ID, "error", err)
		return nil, fmt.Errorf("生成刷新令牌失败")
	}

	s.logger.Infow("User login successful", "user_id", userEntity.ID, "username", userEntity.Username, "tenant_id", userEntity.TenantID)

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         userEntity,
		Tenant:       tenantEntity,
	}, nil
}

// RefreshToken 刷新token
func (s *AuthService) RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error) {
	// 验证refresh token
	claims, err := middleware.ValidateRefreshToken(req.RefreshToken, s.jwtSecret)
	if err != nil {
		s.logger.Warnw("Invalid refresh token", "error", err)
		return nil, fmt.Errorf("刷新令牌无效")
	}

	// 获取用户信息
	userEntity, err := s.client.User.Get(ctx, claims.UserID)
	if err != nil {
		s.logger.Warnw("User not found for refresh token", "user_id", claims.UserID, "error", err)
		return nil, fmt.Errorf("用户不存在")
	}

	// 检查用户是否激活
	if !userEntity.Active {
		s.logger.Warnw("Inactive user trying to refresh token", "user_id", userEntity.ID)
		return nil, fmt.Errorf("用户已被禁用")
	}

    // 生成新的access token（使用数据库角色）
    newAccessToken, err := middleware.GenerateAccessToken(
        userEntity.ID,
        userEntity.Username,
        string(userEntity.Role),
        userEntity.TenantID,
        s.jwtSecret,
        time.Duration(15)*time.Minute,
    )
	if err != nil {
		s.logger.Errorw("Failed to generate new access token", "user_id", userEntity.ID, "error", err)
		return nil, fmt.Errorf("生成新令牌失败")
	}

	s.logger.Infow("Token refreshed successfully", "user_id", userEntity.ID)

	return &dto.RefreshTokenResponse{
		AccessToken: newAccessToken,
	}, nil
}

// GetUserTenants 获取用户可访问的租户列表
func (s *AuthService) GetUserTenants(ctx context.Context, userID int) (*dto.UserTenantsResponse, error) {
	// 查询用户的租户（这里简化为用户只属于一个租户）
	userEntity, err := s.client.User.Query().
		Where(user.IDEQ(userID)).
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get user", "user_id", userID, "error", err)
		return nil, fmt.Errorf("查询用户信息失败")
	}

	// 查询租户信息
	tenantEntity, err := s.client.Tenant.Get(ctx, userEntity.TenantID)
	if err != nil {
		s.logger.Errorw("Failed to get tenant", "tenant_id", userEntity.TenantID, "error", err)
		return nil, fmt.Errorf("查询租户信息失败")
	}

	tenants := []dto.TenantInfo{
		{
			ID:     tenantEntity.ID,
			Name:   tenantEntity.Name,
			Code:   tenantEntity.Code,
			Domain: tenantEntity.Domain,
			Type:   tenantEntity.Type,
			Status: tenantEntity.Status,
		},
	}

	return &dto.UserTenantsResponse{
		Tenants: tenants,
	}, nil
}

// SwitchTenant 切换租户
func (s *AuthService) SwitchTenant(ctx context.Context, userID, tenantID int) (*dto.LoginResponse, error) {
	// 验证用户是否有权限访问该租户
	userEntity, err := s.client.User.Query().
		Where(
			user.IDEQ(userID),
			user.TenantIDEQ(tenantID),
		).
		First(ctx)
	if err != nil {
		s.logger.Warnw("User has no access to tenant", "user_id", userID, "tenant_id", tenantID, "error", err)
		return nil, fmt.Errorf("无权限访问该租户")
	}

	// 查询租户信息
	tenantEntity, err := s.client.Tenant.Get(ctx, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to get tenant", "tenant_id", tenantID, "error", err)
		return nil, fmt.Errorf("租户不存在")
	}

    // 生成新的access token（使用数据库角色）
    accessToken, err := middleware.GenerateAccessToken(
        userEntity.ID,
        userEntity.Username,
        string(userEntity.Role),
        tenantID,
        s.jwtSecret,
        time.Duration(15)*time.Minute,
    )
	if err != nil {
		s.logger.Errorw("Failed to generate access token for tenant switch", "user_id", userEntity.ID, "tenant_id", tenantID, "error", err)
		return nil, fmt.Errorf("生成token失败")
	}

	// 生成新的refresh token
	refreshToken, err := middleware.GenerateRefreshToken(
		userEntity.ID,
		s.jwtSecret,
		time.Duration(7*24)*time.Hour,
	)
	if err != nil {
		s.logger.Errorw("Failed to generate refresh token for tenant switch", "user_id", userEntity.ID, "error", err)
		return nil, fmt.Errorf("生成刷新令牌失败")
	}

	s.logger.Infow("Tenant switched successfully", "user_id", userEntity.ID, "tenant_id", tenantID)

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         userEntity,
		Tenant:       tenantEntity,
	}, nil
}

// ValidateUser 验证用户是否存在且激活
func (s *AuthService) ValidateUser(ctx context.Context, userID int) (*ent.User, error) {
	userEntity, err := s.client.User.Get(ctx, userID)
	if err != nil {
		s.logger.Warnw("User not found", "user_id", userID, "error", err)
		return nil, fmt.Errorf("用户不存在")
	}

	if !userEntity.Active {
		s.logger.Warnw("User is inactive", "user_id", userID)
		return nil, fmt.Errorf("用户账号已被禁用")
	}

	return userEntity, nil
}

// GetUserInfo 获取用户信息
func (s *AuthService) GetUserInfo(ctx context.Context, userID int) (*dto.UserInfo, error) {
    userEntity, err := s.ValidateUser(ctx, userID)
    if err != nil {
        return nil, err
    }

    // 使用数据库中的角色
    role := userEntity.Role

    return &dto.UserInfo{
        ID:         userEntity.ID,
        Username:   userEntity.Username,
        Role:       string(role),
        Email:      userEntity.Email,
        Name:       userEntity.Name,
        Department: userEntity.Department,
        TenantID:   userEntity.TenantID,
    }, nil
}

// Logout 用户登出
func (s *AuthService) Logout(ctx context.Context, userID int) error {
	s.logger.Infow("User logged out", "user_id", userID)

	// 如果启用了黑名单服务，撤销用户的所有token
	if s.tokenBlacklist != nil {
		if err := s.tokenBlacklist.RevokeUserTokens(ctx, userID); err != nil {
			s.logger.Warnw("Failed to revoke user tokens", "user_id", userID, "error", err)
			// 不影响登出流程
		}
	}

	return nil
}

// RevokeUserTokens 撤销用户的所有Token（登出所有设备）
func (s *AuthService) RevokeUserTokens(ctx context.Context, userID int) error {
	if s.tokenBlacklist == nil {
		return fmt.Errorf("token blacklist service not configured")
	}

	return s.tokenBlacklist.RevokeUserTokens(ctx, userID)
}

// AddTokenToBlacklist 将特定Token加入黑名单
func (s *AuthService) AddTokenToBlacklist(tokenString string, expiresAt time.Time) error {
	if s.tokenBlacklist == nil {
		return fmt.Errorf("token blacklist service not configured")
	}

	return s.tokenBlacklist.AddToBlacklist(tokenString, expiresAt)
}