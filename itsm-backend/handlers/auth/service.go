package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/mspallocation"
	"itsm-backend/ent/passwordresettoken"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	"itsm-backend/middleware"
	"itsm-backend/pkg/tenantmode"
	"itsm-backend/service"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	client         *ent.Client
	jwtSecret      string
	logger         *zap.SugaredLogger
	tokenBlacklist *service.TokenBlacklistService
	emailService   *service.EmailService
	baseURL        string
}

func NewService(client *ent.Client, jwtSecret string, logger *zap.SugaredLogger, tokenBlacklist *service.TokenBlacklistService) *Service {
	return &Service{client: client, jwtSecret: jwtSecret, logger: logger, tokenBlacklist: tokenBlacklist, baseURL: "http://localhost:3000"}
}

func (s *Service) SetEmailService(emailService *service.EmailService) { s.emailService = emailService }
func (s *Service) SetBaseURL(baseURL string)                          { s.baseURL = baseURL }

func (s *Service) permissions(userEntity *ent.User) []string {
	if userEntity.Role == user.RoleSuperAdmin {
		return []string{"*"}
	}
	permissions := make([]string, 0)
	seen := make(map[string]bool)
	roles := []string{string(userEntity.Role)}
	if userEntity.MspRole != "" {
		if role := middleware.GetMSPRBACRole(string(userEntity.MspRole)); role != "" {
			roles = append(roles, role)
		}
	}
	for _, role := range roles {
		for _, permission := range middleware.RolePermissions[role] {
			key := permission.Resource + ":" + permission.Action
			if !seen[key] {
				seen[key] = true
				permissions = append(permissions, key)
			}
		}
	}
	return permissions
}

func (s *Service) SwitchTenant(ctx context.Context, userID, tenantID int) (*dto.LoginResponse, error) {
	userEntity, err := s.client.User.Get(ctx, userID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("用户不存在")
		}
		s.logger.Errorw("Failed to load user for tenant switch", "user_id", userID, "error", err)
		return nil, fmt.Errorf("无权限访问该租户")
	}
	nativeSwitch := userEntity.TenantID == tenantID
	superAdmin := userEntity.Role == user.RoleSuperAdmin
	mspAllowed := false
	if !nativeSwitch && !superAdmin && string(userEntity.MspRole) != "" {
		origin, originErr := s.client.Tenant.Get(ctx, userEntity.TenantID)
		if originErr == nil && tenantmode.IsMSPProviderTenantType(string(origin.Type)) {
			target, targetErr := s.client.Tenant.Get(ctx, tenantID)
			if targetErr == nil && tenantmode.IsCustomerTenantType(string(target.Type)) {
				count, queryErr := s.client.MSPAllocation.Query().Where(mspallocation.MspUserIDEQ(userID), mspallocation.CustomerTenantIDEQ(tenantID), mspallocation.DeassignedAtIsNil()).Count(ctx)
				mspAllowed = queryErr == nil && count > 0
			}
		}
	}
	if !nativeSwitch && !superAdmin && !mspAllowed {
		s.logger.Warnw("Switch tenant denied", "user_id", userID, "tenant_id", tenantID, "native_switch", nativeSwitch, "super_admin", superAdmin, "msp_role", string(userEntity.MspRole))
		return nil, fmt.Errorf("无权限访问该租户")
	}
	tenantEntity, err := s.client.Tenant.Get(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("租户不存在")
	}
	if tenantEntity.Status != "active" {
		return nil, fmt.Errorf("租户已被暂停")
	}
	if !tenantEntity.ExpiresAt.IsZero() && tenantEntity.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("租户已过期")
	}
	accessToken, err := middleware.GenerateAccessToken(userEntity.ID, userEntity.Username, string(userEntity.Role), tenantID, s.jwtSecret, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("生成token失败")
	}
	refreshToken, err := middleware.GenerateRefreshToken(userEntity.ID, s.jwtSecret, 7*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败")
	}
	mspRole := string(userEntity.MspRole)
	var mspRolePtr *string
	if mspRole != "" {
		mspRolePtr = &mspRole
	}
	return &dto.LoginResponse{AccessToken: accessToken, RefreshToken: refreshToken, User: &dto.LoginUserResponse{ID: userEntity.ID, Username: userEntity.Username, Email: userEntity.Email, Name: userEntity.Name, Role: string(userEntity.Role), MSPRole: mspRolePtr, Department: userEntity.Department, DepartmentID: userEntity.DepartmentID, Phone: userEntity.Phone, Active: userEntity.Active, TenantID: userEntity.TenantID, CreatedAt: userEntity.CreatedAt, UpdatedAt: userEntity.UpdatedAt, Permissions: s.permissions(userEntity)}, Tenant: tenantEntity}, nil
}

func (s *Service) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	if exists, err := s.client.User.Query().Where(user.UsernameEQ(req.Username)).Exist(ctx); err != nil {
		return nil, fmt.Errorf("检查用户名失败")
	} else if exists {
		return nil, fmt.Errorf("用户名已被注册")
	}
	if exists, err := s.client.User.Query().Where(user.EmailEQ(req.Email)).Exist(ctx); err != nil {
		return nil, fmt.Errorf("检查邮箱失败")
	} else if exists {
		return nil, fmt.Errorf("邮箱已被注册")
	}
	var tenantID int
	if req.TenantCode != "" {
		tenantEntity, err := s.client.Tenant.Query().Where(tenant.CodeEQ(req.TenantCode)).First(ctx)
		if err != nil {
			return nil, fmt.Errorf("租户不存在")
		}
		tenantID = tenantEntity.ID
	} else {
		tenants, err := s.client.Tenant.Query().Where(tenant.StatusEQ("active")).Order(ent.Asc(tenant.FieldID)).Limit(2).All(ctx)
		if err != nil {
			return nil, fmt.Errorf("查询租户失败")
		}
		if len(tenants) != 1 {
			return nil, fmt.Errorf("请指定要加入的租户(tenantCode)")
		}
		tenantID = tenants[0].ID
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败")
	}
	role := user.Role(req.Role)
	if role.String() == "" {
		role = user.RoleEndUser
	}
	userEntity, err := s.client.User.Create().SetUsername(req.Username).SetEmail(req.Email).SetName(req.DisplayName).SetPasswordHash(string(hashedPassword)).SetPhone(req.Phone).SetDepartment(req.Company).SetRole(role).SetTenantID(tenantID).SetActive(true).Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建用户失败")
	}
	return &dto.RegisterResponse{ID: userEntity.ID, Username: userEntity.Username, Email: userEntity.Email, Message: "注册成功"}, nil
}

func (s *Service) ForgotPassword(ctx context.Context, req *dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
	genericOK := &dto.ForgotPasswordResponse{Message: "如果该邮箱已注册，我们将发送密码重置链接"}
	query := s.client.User.Query().Where(user.EmailEQ(req.Email))
	if req.TenantCode != "" {
		tenantEntity, err := s.client.Tenant.Query().Where(tenant.CodeEQ(req.TenantCode)).First(ctx)
		if err != nil {
			return genericOK, nil
		}
		query = query.Where(user.TenantIDEQ(tenantEntity.ID))
	}
	userEntity, err := query.First(ctx)
	if err != nil {
		return genericOK, nil
	}
	token, err := generateResetToken()
	if err != nil {
		return nil, fmt.Errorf("生成重置令牌失败: %w", err)
	}
	if _, err = s.client.PasswordResetToken.Create().SetUserID(userEntity.ID).SetEmail(req.Email).SetToken(token).SetExpiresAt(time.Now().Add(time.Hour)).Save(ctx); err != nil {
		return nil, fmt.Errorf("生成重置令牌失败")
	}
	if s.emailService != nil {
		if err := s.emailService.SendPasswordResetEmail(ctx, []string{req.Email}, token, s.baseURL); err != nil {
			s.logger.Errorw("Failed to send password reset email", "user_id", userEntity.ID, "error", err)
		}
	}
	return genericOK, nil
}

func (s *Service) ResetPassword(ctx context.Context, req *dto.PasswordResetRequest) (*dto.PasswordResetResponse, error) {
	if req.Password != req.PasswordConfirm {
		return nil, fmt.Errorf("两次输入的密码不一致")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("启动密码重置事务失败")
	}
	defer func() { _ = tx.Rollback() }()
	affected, err := tx.PasswordResetToken.Update().Where(passwordresettoken.TokenEQ(req.Token), passwordresettoken.EmailEQ(req.Email), passwordresettoken.Used(false), passwordresettoken.ExpiresAtGT(time.Now())).SetUsed(true).Save(ctx)
	if err != nil || affected != 1 {
		return nil, fmt.Errorf("令牌无效或已使用")
	}
	tokenEntity, err := tx.PasswordResetToken.Query().Where(passwordresettoken.TokenEQ(req.Token), passwordresettoken.EmailEQ(req.Email)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("令牌无效或已使用")
	}
	if _, err = tx.User.UpdateOneID(tokenEntity.UserID).SetPasswordHash(string(hashedPassword)).Save(ctx); err != nil {
		return nil, fmt.Errorf("更新密码失败")
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交密码重置失败")
	}
	if s.tokenBlacklist != nil {
		_ = s.tokenBlacklist.RevokeUserTokens(ctx, tokenEntity.UserID)
	}
	return &dto.PasswordResetResponse{Message: "密码重置成功，请使用新密码登录"}, nil
}

func (s *Service) ValidateResetToken(ctx context.Context, req *dto.ValidateResetTokenRequest) (*dto.ValidateResetTokenResponse, error) {
	tokenEntity, err := s.client.PasswordResetToken.Query().Where(passwordresettoken.TokenEQ(req.Token), passwordresettoken.EmailEQ(req.Email), passwordresettoken.Used(false)).First(ctx)
	valid := err == nil && !time.Now().After(tokenEntity.ExpiresAt)
	return &dto.ValidateResetTokenResponse{Valid: valid, Email: req.Email}, nil
}

func generateResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("read cryptographic random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
