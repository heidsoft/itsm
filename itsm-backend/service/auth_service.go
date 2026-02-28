package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/passwordresettoken"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	"itsm-backend/middleware"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	client         *ent.Client
	jwtSecret      string
	logger         *zap.SugaredLogger
	tokenBlacklist *TokenBlacklistService
	emailService   *EmailService
	baseURL        string // 前端基础URL，用于生成重置链接
}

func NewAuthService(client *ent.Client, jwtSecret string, logger *zap.SugaredLogger, blacklistService *TokenBlacklistService) *AuthService {
	return &AuthService{
		client:         client,
		jwtSecret:      jwtSecret,
		logger:         logger,
		tokenBlacklist: blacklistService,
		baseURL:        "http://localhost:3000", // 默认值，可在生产环境通过配置覆盖
	}
}

// SetEmailService 设置邮件服务
func (s *AuthService) SetEmailService(emailService *EmailService) {
	s.emailService = emailService
}

// SetBaseURL 设置前端基础URL
func (s *AuthService) SetBaseURL(baseURL string) {
	s.baseURL = baseURL
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

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// 检查用户名是否已存在
	exists, err := s.client.User.Query().
		Where(user.UsernameEQ(req.Username)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check username existence", "username", req.Username, "error", err)
		return nil, fmt.Errorf("检查用户名失败")
	}
	if exists {
		return nil, fmt.Errorf("用户名已被注册")
	}

	// 检查邮箱是否已存在
	exists, err = s.client.User.Query().
		Where(user.EmailEQ(req.Email)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check email existence", "email", req.Email, "error", err)
		return nil, fmt.Errorf("检查邮箱失败")
	}
	if exists {
		return nil, fmt.Errorf("邮箱已被注册")
	}

	// 获取租户ID
	tenantID := 1 // 默认租户
	if req.TenantCode != "" {
		tenantEntity, err := s.client.Tenant.Query().
			Where(tenant.CodeEQ(req.TenantCode)).
			First(ctx)
		if err != nil {
			s.logger.Warnw("Tenant not found", "tenant_code", req.TenantCode, "error", err)
			return nil, fmt.Errorf("租户不存在")
		}
		tenantID = tenantEntity.ID
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Errorw("Failed to hash password", "error", err)
		return nil, fmt.Errorf("密码加密失败")
	}

	// 确定角色
	role := user.Role(req.Role)
	if role.String() == "" {
		role = "end_user"
	}

	// 创建用户
	userEntity, err := s.client.User.Create().
		SetUsername(req.Username).
		SetEmail(req.Email).
		SetName(req.FullName).
		SetPasswordHash(string(hashedPassword)).
		SetPhone(req.Phone).
		SetDepartment(req.Company).
		SetRole(role).
		SetTenantID(tenantID).
		SetActive(true).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create user", "username", req.Username, "error", err)
		return nil, fmt.Errorf("创建用户失败")
	}

	s.logger.Infow("User registered successfully", "user_id", userEntity.ID, "username", req.Username)

	return &dto.RegisterResponse{
		ID:       userEntity.ID,
		Username: userEntity.Username,
		Email:    userEntity.Email,
		Message:  "注册成功",
	}, nil
}

// ForgotPassword 发送密码重置邮件
func (s *AuthService) ForgotPassword(ctx context.Context, req *dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
	// 查找用户
	userQuery := s.client.User.Query().Where(user.EmailEQ(req.Email))

	// 如果提供了租户代码，按租户过滤
	if req.TenantCode != "" {
		tenantEntity, err := s.client.Tenant.Query().
			Where(tenant.CodeEQ(req.TenantCode)).
			First(ctx)
		if err != nil {
			s.logger.Warnw("Tenant not found", "tenant_code", req.TenantCode, "error", err)
			return nil, fmt.Errorf("用户不存在")
		}
		userQuery = userQuery.Where(user.TenantIDEQ(tenantEntity.ID))
	}

	userEntity, err := userQuery.First(ctx)
	if err != nil {
		s.logger.Warnw("User not found for password reset", "email", req.Email, "error", err)
		// 为了安全，不提示用户不存在
		return &dto.ForgotPasswordResponse{
			Message: "如果该邮箱已注册，我们将发送密码重置链接",
		}, nil
	}

	// 生成重置令牌
	token := generateResetToken()
	expiresAt := time.Now().Add(1 * time.Hour) // 1小时后过期

	// 保存重置令牌
	_, err = s.client.PasswordResetToken.Create().
		SetUserID(userEntity.ID).
		SetEmail(req.Email).
		SetToken(token).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create password reset token", "user_id", userEntity.ID, "error", err)
		return nil, fmt.Errorf("生成重置令牌失败")
	}

	// 发送重置邮件
	if s.emailService != nil {
		err = s.emailService.SendPasswordResetEmail(ctx, []string{req.Email}, token, s.baseURL)
		if err != nil {
			s.logger.Errorw("Failed to send password reset email", "user_id", userEntity.ID, "email", req.Email, "error", err)
			// 邮件发送失败不影响流程，只记录日志
		} else {
			s.logger.Infow("Password reset email sent", "user_id", userEntity.ID, "email", req.Email)
		}
	} else {
		s.logger.Warnw("Email service not configured, skipping email send", "user_id", userEntity.ID, "email", req.Email)
	}

	return &dto.ForgotPasswordResponse{
		Message: "如果该邮箱已注册，我们将发送密码重置链接",
	}, nil
}

// ResetPassword 重置密码
func (s *AuthService) ResetPassword(ctx context.Context, req *dto.PasswordResetRequest) (*dto.PasswordResetResponse, error) {
	// 检查两次密码是否一致
	if req.Password != req.PasswordConfirm {
		return nil, fmt.Errorf("两次输入的密码不一致")
	}

	// 查找重置令牌
	tokenEntity, err := s.client.PasswordResetToken.Query().
		Where(
			passwordresettoken.TokenEQ(req.Token),
			passwordresettoken.EmailEQ(req.Email),
			passwordresettoken.Used(false),
		).
		First(ctx)
	if err != nil {
		s.logger.Warnw("Invalid or used reset token", "email", req.Email, "error", err)
		return nil, fmt.Errorf("重置令牌无效或已过期")
	}

	// 检查令牌是否过期
	if time.Now().After(tokenEntity.ExpiresAt) {
		return nil, fmt.Errorf("重置令牌已过期")
	}

	// 获取用户
	userEntity, err := s.client.User.Get(ctx, tokenEntity.UserID)
	if err != nil {
		s.logger.Errorw("User not found for password reset", "user_id", tokenEntity.UserID, "error", err)
		return nil, fmt.Errorf("用户不存在")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Errorw("Failed to hash new password", "user_id", userEntity.ID, "error", err)
		return nil, fmt.Errorf("密码加密失败")
	}

	// 更新用户密码
	_, err = s.client.User.UpdateOneID(userEntity.ID).
		SetPasswordHash(string(hashedPassword)).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update password", "user_id", userEntity.ID, "error", err)
		return nil, fmt.Errorf("更新密码失败")
	}

	// 标记令牌为已使用
	_, err = s.client.PasswordResetToken.UpdateOneID(tokenEntity.ID).
		SetUsed(true).
		Save(ctx)
	if err != nil {
		s.logger.Warnw("Failed to mark token as used", "token_id", tokenEntity.ID, "error", err)
	}

	// 撤销用户的所有token
	if s.tokenBlacklist != nil {
		if err := s.tokenBlacklist.RevokeUserTokens(ctx, userEntity.ID); err != nil {
			s.logger.Warnw("Failed to revoke user tokens", "user_id", userEntity.ID, "error", err)
		}
	}

	s.logger.Infow("Password reset successfully", "user_id", userEntity.ID)

	return &dto.PasswordResetResponse{
		Message: "密码重置成功，请使用新密码登录",
	}, nil
}

// ValidateResetToken 验证重置令牌是否有效
func (s *AuthService) ValidateResetToken(ctx context.Context, req *dto.ValidateResetTokenRequest) (*dto.ValidateResetTokenResponse, error) {
	tokenEntity, err := s.client.PasswordResetToken.Query().
		Where(
			passwordresettoken.TokenEQ(req.Token),
			passwordresettoken.EmailEQ(req.Email),
			passwordresettoken.Used(false),
		).
		First(ctx)
	if err != nil {
		return &dto.ValidateResetTokenResponse{
			Valid: false,
			Email: req.Email,
		}, nil
	}

	// 检查是否过期
	if time.Now().After(tokenEntity.ExpiresAt) {
		return &dto.ValidateResetTokenResponse{
			Valid: false,
			Email: req.Email,
		}, nil
	}

	return &dto.ValidateResetTokenResponse{
		Valid: true,
		Email: req.Email,
	}, nil
}

// generateResetToken 生成密码重置令牌
func generateResetToken() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	token := make([]byte, 32)
	for i := range token {
		token[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(token)
}

// CleanupExpiredTokens 清理过期的重置令牌
func (s *AuthService) CleanupExpiredTokens(ctx context.Context) error {
	_, err := s.client.PasswordResetToken.Delete().
		Where(passwordresettoken.ExpiresAtLT(time.Now())).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to cleanup expired tokens", "error", err)
		return err
	}
	return nil
}
