package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"itsm-backend/common"
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
	client       *ent.Client
	jwtSecret    string
	logger       *zap.SugaredLogger
	emailService *service.EmailService
	baseURL      string
}

func NewService(client *ent.Client, jwtSecret string, logger *zap.SugaredLogger) *Service {
	return &Service{client: client, jwtSecret: jwtSecret, logger: logger, baseURL: "http://localhost:3000"}
}

func (s *Service) SetEmailService(emailService *service.EmailService) { s.emailService = emailService }
func (s *Service) SetBaseURL(baseURL string)                          { s.baseURL = baseURL }

func (s *Service) permissions(userEntity *ent.User) []string {
	return userPermissions(userEntity)
}

// userPermissions 按角色解析权限码列表。登录与 SSO 会话签发共用同一口径，
// 避免两条认证链路给出不同权限集。
func userPermissions(userEntity *ent.User) []string {
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
		for _, permission := range middleware.RoleDefaultPermissions(role) {
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
			return nil, fmt.Errorf("User does not exist")
		}
		s.logger.Errorw("Failed to load user for tenant switch", "user_id", userID, "error", err)
		return nil, fmt.Errorf("No permission to access this tenant")
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
		return nil, fmt.Errorf("No permission to access this tenant")
	}
	tenantEntity, err := s.client.Tenant.Get(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("Tenant does not exist")
	}
	if tenantEntity.Status != "active" {
		return nil, fmt.Errorf("Tenant has been suspended")
	}
	if !tenantEntity.ExpiresAt.IsZero() && tenantEntity.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("Tenant has expired")
	}
	accessToken, err := middleware.GenerateAccessToken(userEntity.ID, userEntity.Username, string(userEntity.Role), tenantID, s.jwtSecret, middleware.AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate token")
	}
	refreshToken, err := middleware.GenerateRefreshToken(userEntity.ID, userEntity.Username, string(userEntity.Role), tenantID, s.jwtSecret, middleware.RefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate refresh token")
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
		return nil, fmt.Errorf("Failed to check username")
	} else if exists {
		return nil, common.NewBusinessError(common.ConflictCode, "Username has been registered", "")
	}
	if exists, err := s.client.User.Query().Where(user.EmailEQ(req.Email)).Exist(ctx); err != nil {
		return nil, fmt.Errorf("Failed to check email")
	} else if exists {
		return nil, common.NewBusinessError(common.ConflictCode, "Email has been registered", "")
	}
	tenants, err := s.client.Tenant.Query().Where(tenant.StatusEQ("active")).Order(ent.Asc(tenant.FieldID)).Limit(2).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to query tenant")
	}
	if len(tenants) != 1 {
		// fail-closed：多活跃租户无法从请求推断归属，零活跃租户说明系统未初始化；
		// 目标租户绝不接受请求体指定。两者都是部署状态而非用户输入错误，
		// 报 401 会让新用户以为是自己的凭据有问题。
		if len(tenants) > 1 {
			return nil, common.NewBusinessError(common.ServiceUnavailableCode, "There are multiple active tenants in the system. Self-service registration is not available yet. Please contact the administrator to open an account.", "")
		}
		return nil, common.NewBusinessError(common.ServiceUnavailableCode, "The system has not initialized the active tenant and cannot register it yet.", "")
	}
	tenantID := tenants[0].ID
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("Password encryption failed")
	}
	userEntity, err := s.client.User.Create().SetUsername(req.Username).SetEmail(req.Email).SetName(req.ResolvedDisplayName()).SetPasswordHash(string(hashedPassword)).SetPhone(req.Phone).SetDepartment(req.Company).SetRole(user.RoleEndUser).SetTenantID(tenantID).SetActive(true).Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to create user")
	}
	return &dto.RegisterResponse{ID: userEntity.ID, Username: userEntity.Username, Email: userEntity.Email, Message: "注册成功"}, nil
}

func (s *Service) ForgotPassword(ctx context.Context, req *dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
	if s.emailService == nil {
		return nil, common.NewBusinessError(common.ServiceUnavailableCode, "Password reset function is not enabled: Email service is not configured", "")
	}
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
		return nil, fmt.Errorf("Failed to generate reset token: %w", err)
	}
	if _, err = s.client.PasswordResetToken.Create().SetUserID(userEntity.ID).SetEmail(req.Email).SetToken(token).SetExpiresAt(time.Now().Add(time.Hour)).Save(ctx); err != nil {
		return nil, fmt.Errorf("Failed to generate reset token")
	}
	if err := s.emailService.SendPasswordResetEmail(ctx, []string{req.Email}, token, s.baseURL); err != nil {
		s.logger.Errorw("Failed to send password reset email", "user_id", userEntity.ID, "error", err)
	}
	return genericOK, nil
}

func (s *Service) ResetPassword(ctx context.Context, req *dto.PasswordResetRequest) (*dto.PasswordResetResponse, error) {
	if req.Password != req.PasswordConfirm {
		return nil, fmt.Errorf("Passwords do not match")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("Password encryption failed")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to initiate password reset transaction")
	}
	defer func() { _ = tx.Rollback() }()
	affected, err := tx.PasswordResetToken.Update().Where(passwordresettoken.TokenEQ(req.Token), passwordresettoken.EmailEQ(req.Email), passwordresettoken.Used(false), passwordresettoken.ExpiresAtGT(time.Now())).SetUsed(true).Save(ctx)
	if err != nil || affected != 1 {
		return nil, fmt.Errorf("Token is invalid or already used")
	}
	tokenEntity, err := tx.PasswordResetToken.Query().Where(passwordresettoken.TokenEQ(req.Token), passwordresettoken.EmailEQ(req.Email)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("Token is invalid or already used")
	}
	if _, err = tx.User.UpdateOneID(tokenEntity.UserID).SetPasswordHash(string(hashedPassword)).Save(ctx); err != nil {
		return nil, fmt.Errorf("Failed to update password")
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("Failed to submit password reset")
	}
	if err := middleware.InvalidateUserTokens(ctx, tokenEntity.UserID, time.Now()); err != nil {
		// 改密后旧会话应立即失效。密码已写入，回滚更糟，所以只记录：
		// 吊销存储故障时旧 token 仍可用，属于必须告警的安全延迟。
		s.logger.Errorw("failed to invalidate tokens after password reset",
			"user_id", tokenEntity.UserID, "error", err)
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
