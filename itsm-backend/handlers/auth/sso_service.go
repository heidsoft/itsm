package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ssoidentity"
	"itsm-backend/ent/ssoprovider"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	"itsm-backend/middleware"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// allowedJITRoles 是 JIT 自动开通允许的角色白名单。
// admin 及以上的等保角色绝不由 IdP 声明直接授予：提供方配置一旦被写错，
// 就等于把租户最高权限交给外部系统的某条属性。提权走管理员显式授权。
var allowedJITRoles = map[string]bool{
	"end_user":   true,
	"agent":      true,
	"technician": true,
}

// SSOService 承载 SSO（OIDC）全部业务逻辑：发起、回调、绑定、解绑、
// 提供方管理。会话签发复用 middleware 的 token 工厂，与账号密码登录同源。
type SSOService struct {
	client     *ent.Client
	jwtSecret  string
	logger     *zap.SugaredLogger
	encryption *middleware.EncryptionService
	states     SSOStateStore
	oidc       *oidcClient
}

func NewSSOService(
	client *ent.Client,
	jwtSecret string,
	logger *zap.SugaredLogger,
	encryption *middleware.EncryptionService,
	states SSOStateStore,
) *SSOService {
	if states == nil {
		states = NewSSOStateStore(nil)
	}
	return &SSOService{
		client:     client,
		jwtSecret:  jwtSecret,
		logger:     logger,
		encryption: encryption,
		states:     states,
		oidc:       newOIDCClient(),
	}
}

// Initiate 生成授权 URL 并落 state。
//
// bindUserID > 0 表示「已登录用户绑定外部身份」：state 记录该用户，
// 回调时校验当前会话仍是同一人，避免把绑定流程变成登录流程。
func (s *SSOService) Initiate(ctx context.Context, providerCode, tenantCode string, bindUserID int) (*dto.SSOInitiateResponse, error) {
	provider, err := s.resolveProviderForLogin(ctx, providerCode, tenantCode)
	if err != nil {
		return nil, err
	}
	if provider.ClientID == "" {
		return nil, common.NewBusinessError(common.ServiceUnavailableCode, fmt.Sprintf("Provider %s does not configure client_id, please contact the administrator", provider.Code), "")
	}
	if strings.TrimSpace(provider.RedirectURI) == "" {
		return nil, common.NewBusinessError(common.ServiceUnavailableCode, fmt.Sprintf("Provider %s has not configured a callback address, please contact the administrator", provider.Code), "")
	}

	endpoints, err := s.oidc.resolveEndpoints(ctx, provider.ID, provider.Issuer, provider.AuthorizationEndpoint, provider.TokenEndpoint, provider.UserinfoEndpoint, provider.JwksURI)
	if err != nil {
		s.markProviderError(ctx, provider.ID, err)
		return nil, common.NewBusinessError(common.ServiceUnavailableCode, fmt.Sprintf("Provider %s endpoint resolution failed, please try again later or contact the administrator", provider.Code), "")
	}

	state, err := randomTokenHex(32)
	if err != nil {
		return nil, common.NewBusinessError(common.InternalErrorCode, "Failed to generate login status", "")
	}
	nonce, err := randomTokenHex(32)
	if err != nil {
		return nil, common.NewBusinessError(common.InternalErrorCode, "Failed to generate login status", "")
	}
	verifier, err := randomTokenBase64URL(32)
	if err != nil {
		return nil, common.NewBusinessError(common.InternalErrorCode, "Failed to generate login status", "")
	}
	challengeSum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeSum[:])

	mode := "login"
	if bindUserID > 0 {
		mode = "bind"
	}
	record := &SSOStateRecord{
		ProviderID: provider.ID,
		TenantID:   provider.TenantID,
		Mode:       mode,
		BindUserID: bindUserID,
		Nonce:      nonce,
		Verifier:   verifier,
		CreatedAt:  time.Now(),
	}
	if err := s.states.Save(ctx, state, record, SSOStateTTL); err != nil {
		return nil, common.NewBusinessError(common.ServiceUnavailableCode, "The login status service is temporarily unavailable, please try again later.", "")
	}

	scopes := strings.TrimSpace(provider.Scopes)
	if scopes == "" {
		scopes = "openid profile email"
	}
	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", provider.ClientID)
	query.Set("redirect_uri", provider.RedirectURI)
	query.Set("scope", scopes)
	query.Set("state", state)
	query.Set("nonce", nonce)
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")
	redirectURL := endpoints.AuthorizationEndpoint + "?" + query.Encode()

	return &dto.SSOInitiateResponse{
		RedirectURL: redirectURL,
		State:       state,
		Provider:    provider.Code,
	}, nil
}

// Callback 处理授权码回调：换 token、验签、定位/开通用户、签发会话。
// state 单次消费；任何一步失败都不建立会话。
func (s *SSOService) Callback(ctx context.Context, providerCode, code, state string) (*dto.SSOSessionResponse, error) {
	record, err := s.states.Consume(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("The login status is invalid or expired, please re-initiate SSO login.")
	}
	provider, err := s.client.SSOProvider.Get(ctx, record.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("Provider does not exist")
	}
	if providerCode != "" && !strings.EqualFold(provider.Code, providerCode) {
		return nil, fmt.Errorf("The callback provider is inconsistent with the one when it was initiated.")
	}
	if !provider.Enabled {
		return nil, fmt.Errorf("Provider %s is deactivated", provider.Code)
	}
	if record.Mode == "bind" {
		return nil, fmt.Errorf("This state belongs to the account binding process. Please complete it through the binding entrance.")
	}

	identity, err := s.exchangeAndIdentify(ctx, provider, code, record)
	if err != nil {
		s.markProviderError(ctx, provider.ID, err)
		return nil, err
	}

	userEntity, provisioned, err := s.resolveUser(ctx, provider, identity)
	if err != nil {
		return nil, err
	}
	if !userEntity.Active {
		return nil, fmt.Errorf("Account has been deactivated, please contact the administrator")
	}
	tenantEntity, err := s.client.Tenant.Get(ctx, userEntity.TenantID)
	if err != nil {
		return nil, fmt.Errorf("Tenant does not exist")
	}
	if tenantEntity.Status != "active" {
		return nil, fmt.Errorf("Tenant has been suspended")
	}
	if !tenantEntity.ExpiresAt.IsZero() && tenantEntity.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("Tenant has expired")
	}

	accessToken, err := middleware.GenerateAccessToken(userEntity.ID, userEntity.Username, string(userEntity.Role), userEntity.TenantID, s.jwtSecret, middleware.AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate access token")
	}
	refreshToken, err := middleware.GenerateRefreshToken(userEntity.ID, userEntity.Username, string(userEntity.Role), userEntity.TenantID, s.jwtSecret, middleware.RefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate refresh token")
	}

	if err := s.touchIdentity(ctx, provider.ID, userEntity.ID, identity); err != nil {
		// 会话已签发，登录元数据写失败只记录：回滚会话比丢一条 last_login_at 更糟。
		s.logger.Errorw("failed to record sso identity login", "user_id", userEntity.ID, "provider_id", provider.ID, "error", err)
	}

	return &dto.SSOSessionResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(middleware.AccessTokenTTL.Seconds()),
		User:         s.loginUserResponse(userEntity),
		Tenant:       tenantEntity,
		Provisioned:  provisioned,
	}, nil
}

// Bind 把已认证用户的会话绑定到外部身份。
// sessionUserID 必须来自本请求的有效会话，且与 state 记录的一致。
func (s *SSOService) Bind(ctx context.Context, sessionUserID int, providerCode, code, state string) error {
	record, err := s.states.Consume(ctx, state)
	if err != nil {
		return fmt.Errorf("The login status is invalid or expired, please re-initiate binding.")
	}
	if record.Mode != "bind" || record.BindUserID <= 0 {
		return fmt.Errorf("This state does not belong to the binding process")
	}
	if record.BindUserID != sessionUserID {
		return fmt.Errorf("The session does not match the binding request, please reinitiate binding.")
	}
	provider, err := s.client.SSOProvider.Get(ctx, record.ProviderID)
	if err != nil {
		return fmt.Errorf("Provider does not exist")
	}
	if providerCode != "" && !strings.EqualFold(provider.Code, providerCode) {
		return fmt.Errorf("The binding provider is inconsistent with the one initiated")
	}
	if !provider.Enabled {
		return fmt.Errorf("Provider %s is deactivated", provider.Code)
	}

	identity, err := s.exchangeAndIdentify(ctx, provider, code, record)
	if err != nil {
		s.markProviderError(ctx, provider.ID, err)
		return err
	}

	sessionUser, err := s.client.User.Get(ctx, sessionUserID)
	if err != nil {
		return fmt.Errorf("User does not exist")
	}
	if sessionUser.TenantID != provider.TenantID {
		return fmt.Errorf("Cannot bind providers of other tenants")
	}

	// 该外部身份已被同租户其他用户占用时拒绝：静默改绑会把原主人的登录路径偷走。
	existing, err := s.client.SSOIdentity.Query().
		Where(ssoidentity.TenantIDEQ(provider.TenantID), ssoidentity.ProviderIDEQ(provider.ID), ssoidentity.SubjectEQ(identity.Subject)).
		Only(ctx)
	if err == nil && existing.UserID != sessionUserID {
		return fmt.Errorf("The external identity is already bound to another account")
	}
	if err != nil && !ent.IsNotFound(err) {
		return fmt.Errorf("Failed to query binding relationship")
	}

	if _, err := s.upsertIdentity(ctx, provider.ID, sessionUserID, identity); err != nil {
		return fmt.Errorf("Failed to save binding relationship: %w", err)
	}
	s.logger.Infow("sso identity bound", "user_id", sessionUserID, "provider_id", provider.ID, "subject", maskSubject(identity.Subject))
	return nil
}

// Unbind 删除当前用户与指定提供方的绑定。
func (s *SSOService) Unbind(ctx context.Context, sessionUserID int, providerCode string) error {
	provider, err := s.resolveProviderByCode(ctx, providerCode)
	if err != nil {
		return err
	}
	affected, err := s.client.SSOIdentity.Delete().
		Where(ssoidentity.TenantIDEQ(mustUserTenant(ctx, s, sessionUserID)), ssoidentity.ProviderIDEQ(provider.ID), ssoidentity.UserIDEQ(sessionUserID)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("Unbinding failed")
	}
	if affected == 0 {
		return fmt.Errorf("The provider has not been bound yet")
	}
	s.logger.Infow("sso identity unbound", "user_id", sessionUserID, "provider_id", provider.ID)
	return nil
}

// ListIdentities 返回当前用户的全部身份绑定。
func (s *SSOService) ListIdentities(ctx context.Context, sessionUserID int) ([]dto.SSOIdentityResponse, error) {
	sessionUser, err := s.client.User.Get(ctx, sessionUserID)
	if err != nil {
		return nil, fmt.Errorf("User does not exist")
	}
	identities, err := s.client.SSOIdentity.Query().
		Where(ssoidentity.TenantIDEQ(sessionUser.TenantID), ssoidentity.UserIDEQ(sessionUserID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to query binding relationship")
	}
	if len(identities) == 0 {
		return []dto.SSOIdentityResponse{}, nil
	}
	providerIDs := make([]int, 0, len(identities))
	for _, identity := range identities {
		providerIDs = append(providerIDs, identity.ProviderID)
	}
	providers, err := s.client.SSOProvider.Query().Where(ssoprovider.IDIn(providerIDs...)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to query provider")
	}
	providerByID := make(map[int]*ent.SSOProvider, len(providers))
	for _, provider := range providers {
		providerByID[provider.ID] = provider
	}
	out := make([]dto.SSOIdentityResponse, 0, len(identities))
	for _, identity := range identities {
		item := dto.SSOIdentityResponse{
			Subject:          identity.Subject,
			ExternalUsername: identity.ExternalUsername,
			ExternalEmail:    identity.ExternalEmail,
			CreatedAt:        identity.CreatedAt,
		}
		if identity.LastLoginAt != nil {
			at := *identity.LastLoginAt
			item.LastLoginAt = &at
		}
		if provider, ok := providerByID[identity.ProviderID]; ok {
			item.ProviderCode = provider.Code
			item.ProviderName = provider.Name
		}
		out = append(out, item)
	}
	return out, nil
}

// ListEnabledProviders 供登录页展示可用的 SSO 入口（不含端点与凭证）。
func (s *SSOService) ListEnabledProviders(ctx context.Context, tenantCode string) ([]dto.SSOBrief, error) {
	query := s.client.SSOProvider.Query().Where(ssoprovider.EnabledEQ(true))
	if strings.TrimSpace(tenantCode) != "" {
		tenantEntity, err := s.client.Tenant.Query().Where(tenant.CodeEQ(tenantCode)).Only(ctx)
		if err != nil {
			// 租户编码不存在时返回空列表而非报错：登录页不该因一个错误租户码变红。
			return []dto.SSOBrief{}, nil
		}
		query = query.Where(ssoprovider.TenantIDEQ(tenantEntity.ID))
	}
	providers, err := query.Order(ent.Asc(ssoprovider.FieldID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to query provider")
	}
	out := make([]dto.SSOBrief, 0, len(providers))
	for _, provider := range providers {
		out = append(out, dto.SSOBrief{Code: provider.Code, Name: provider.Name, Protocol: provider.Protocol})
	}
	return out, nil
}

// exchangeAndIdentify 完成授权码交换与身份解析（id_token 优先，userinfo 兜底）。
func (s *SSOService) exchangeAndIdentify(ctx context.Context, provider *ent.SSOProvider, code string, record *SSOStateRecord) (*oidcIdentity, error) {
	if code == "" {
		return nil, fmt.Errorf("Missing authorization code")
	}
	endpoints, err := s.oidc.resolveEndpoints(ctx, provider.ID, provider.Issuer, provider.AuthorizationEndpoint, provider.TokenEndpoint, provider.UserinfoEndpoint, provider.JwksURI)
	if err != nil {
		return nil, fmt.Errorf("Provider endpoint resolution failed: %w", err)
	}
	clientSecret := ""
	if provider.ClientSecretEncrypted != "" {
		decrypted, decErr := s.encryption.Decrypt(provider.ClientSecretEncrypted)
		if decErr != nil {
			return nil, fmt.Errorf("client_secret decryption failed, please reconfigure the provider")
		}
		clientSecret = decrypted
	}
	tokenResp, err := s.oidc.exchangeCode(ctx, endpoints, provider.ClientID, clientSecret, code, provider.RedirectURI, record.Verifier)
	if err != nil {
		return nil, err
	}

	if tokenResp.IDToken != "" {
		identity, verifyErr := s.oidc.verifyIDToken(ctx, endpoints, provider.Issuer, provider.ClientID, tokenResp.IDToken, record.Nonce, s.logger.Warnf)
		if verifyErr != nil {
			return nil, verifyErr
		}
		return identity, nil
	}
	identity, err := s.oidc.fetchUserinfo(ctx, endpoints, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("Failed to obtain identity: %w", err)
	}
	return identity, nil
}

// resolveUser 定位本地用户：先查绑定，再按已验证邮箱链接，最后按策略 JIT 开通。
func (s *SSOService) resolveUser(ctx context.Context, provider *ent.SSOProvider, identity *oidcIdentity) (*ent.User, bool, error) {
	binding, err := s.client.SSOIdentity.Query().
		Where(ssoidentity.TenantIDEQ(provider.TenantID), ssoidentity.ProviderIDEQ(provider.ID), ssoidentity.SubjectEQ(identity.Subject)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, false, fmt.Errorf("Failed to query identity binding")
	}
	if err == nil {
		userEntity, userErr := s.client.User.Get(ctx, binding.UserID)
		if userErr != nil {
			return nil, false, fmt.Errorf("The bound local user does not exist, please contact the administrator")
		}
		return userEntity, false, nil
	}

	if err := s.checkEmailDomain(provider, identity.Email); err != nil {
		return nil, false, err
	}

	// 邮箱已验证时按邮箱链接既有账号（企业常见：先有本地账号，后开 SSO）。
	// 未验证的邮箱绝不用于链接：否则 IdP 侧任意声明一个邮箱即可登录他人账号。
	if identity.Email != "" && identity.EmailVerified {
		existing, emailErr := s.client.User.Query().
			Where(user.EmailEQ(identity.Email), user.TenantIDEQ(provider.TenantID)).
			Only(ctx)
		if emailErr == nil {
			if _, upsertErr := s.upsertIdentity(ctx, provider.ID, existing.ID, identity); upsertErr != nil {
				s.logger.Errorw("failed to auto-link sso identity by verified email", "user_id", existing.ID, "error", upsertErr)
			}
			return existing, false, nil
		}
		if !ent.IsNotFound(emailErr) {
			return nil, false, fmt.Errorf("Failed to match user by email")
		}
	}

	if !provider.AutoProvision {
		return nil, false, fmt.Errorf("The account does not exist and the provider has not enabled automatic activation. Please contact the administrator.")
	}
	if identity.Email == "" {
		return nil, false, fmt.Errorf("The IdP did not return the email address and the account cannot be automatically opened.")
	}
	userEntity, provisionErr := s.provisionUser(ctx, provider, identity)
	if provisionErr != nil {
		return nil, false, provisionErr
	}
	if _, upsertErr := s.upsertIdentity(ctx, provider.ID, userEntity.ID, identity); upsertErr != nil {
		s.logger.Errorw("failed to persist sso identity after provisioning", "user_id", userEntity.ID, "error", upsertErr)
	}
	return userEntity, true, nil
}

// provisionUser JIT 建号。username 在租户内去重；密码为随机值（SSO 用户不走密码登录）。
func (s *SSOService) provisionUser(ctx context.Context, provider *ent.SSOProvider, identity *oidcIdentity) (*ent.User, error) {
	if !allowedJITRoles[provider.DefaultRole] {
		return nil, fmt.Errorf("Provider default role %q does not allow automatic provisioning (end_user/agent/technician only), please contact the administrator", provider.DefaultRole)
	}
	username, err := s.uniqueUsername(ctx, provider.TenantID, identity)
	if err != nil {
		return nil, err
	}
	randomPassword := make([]byte, 32)
	if _, err := rand.Read(randomPassword); err != nil {
		return nil, fmt.Errorf("Failed to generate random credentials")
	}
	passwordHash, err := bcrypt.GenerateFromPassword(randomPassword, bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("Password processing failed")
	}
	displayName := strings.TrimSpace(identity.Name)
	if displayName == "" {
		displayName = username
	}
	userEntity, err := s.client.User.Create().
		SetUsername(username).
		SetEmail(identity.Email).
		SetName(displayName).
		SetPasswordHash(string(passwordHash)).
		SetRole(user.Role(provider.DefaultRole)).
		SetTenantID(provider.TenantID).
		SetActive(true).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, fmt.Errorf("There is already an account in this mailbox and cannot be automatically opened. Please contact the administrator.")
		}
		return nil, fmt.Errorf("Failed to create user")
	}
	s.logger.Infow("sso jit user provisioned", "user_id", userEntity.ID, "provider_id", provider.ID, "tenant_id", provider.TenantID)
	return userEntity, nil
}

// uniqueUsername 由外部用户名/邮箱本地部分派生租户内唯一用户名。
func (s *SSOService) uniqueUsername(ctx context.Context, tenantID int, identity *oidcIdentity) (string, error) {
	base := sanitizeUsername(identity.PreferredUsername)
	if base == "" && identity.Email != "" {
		local := strings.SplitN(identity.Email, "@", 2)[0]
		base = sanitizeUsername(local)
	}
	if base == "" {
		base = "sso_user"
	}
	if len(base) > 47 {
		base = base[:47]
	}
	for attempt := 0; attempt < 50; attempt++ {
		candidate := base
		if attempt > 0 {
			candidate = fmt.Sprintf("%s_%d", base, attempt+1)
		}
		exists, err := s.client.User.Query().Where(user.UsernameEQ(candidate), user.TenantIDEQ(tenantID)).Exist(ctx)
		if err != nil {
			return "", fmt.Errorf("Failed to check username")
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("Unable to generate unique username, please create an account manually")
}

func sanitizeUsername(raw string) string {
	lowered := strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	for _, r := range lowered {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "._-")
}

// checkEmailDomain 执行提供方的邮箱域白名单（空表示不限制）。
func (s *SSOService) checkEmailDomain(provider *ent.SSOProvider, email string) error {
	allowed := strings.TrimSpace(provider.AllowedEmailDomains)
	if allowed == "" {
		return nil
	}
	if email == "" {
		return fmt.Errorf("The provider restricted the mailbox domain, but the IdP did not return the mailbox")
	}
	domain := strings.ToLower(strings.SplitN(email, "@", 2)[1])
	for _, candidate := range strings.Split(allowed, ",") {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		candidate = strings.TrimPrefix(candidate, "@")
		if candidate != "" && candidate == domain {
			return nil
		}
	}
	return fmt.Errorf("Email domain is not allowed.")
}

// upsertIdentity 写入或刷新绑定（同一用户+提供方只保留一条）。
func (s *SSOService) upsertIdentity(ctx context.Context, providerID, userID int, identity *oidcIdentity) (*ent.SSOIdentity, error) {
	provider, err := s.client.SSOProvider.Get(ctx, providerID)
	if err != nil {
		return nil, err
	}
	existing, err := s.client.SSOIdentity.Query().
		Where(ssoidentity.TenantIDEQ(provider.TenantID), ssoidentity.UserIDEQ(userID), ssoidentity.ProviderIDEQ(providerID)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if err == nil {
		return s.client.SSOIdentity.UpdateOneID(existing.ID).
			SetSubject(identity.Subject).
			SetExternalUsername(identity.PreferredUsername).
			SetExternalEmail(identity.Email).
			Save(ctx)
	}
	return s.client.SSOIdentity.Create().
		SetTenantID(provider.TenantID).
		SetProviderID(providerID).
		SetUserID(userID).
		SetSubject(identity.Subject).
		SetExternalUsername(identity.PreferredUsername).
		SetExternalEmail(identity.Email).
		Save(ctx)
}

func (s *SSOService) touchIdentity(ctx context.Context, providerID, userID int, identity *oidcIdentity) error {
	provider, err := s.client.SSOProvider.Get(ctx, providerID)
	if err != nil {
		return err
	}
	_, err = s.client.SSOIdentity.Update().
		Where(ssoidentity.TenantIDEQ(provider.TenantID), ssoidentity.ProviderIDEQ(providerID), ssoidentity.SubjectEQ(identity.Subject)).
		SetLastLoginAt(time.Now()).
		Save(ctx)
	return err
}

func (s *SSOService) markProviderError(ctx context.Context, providerID int, cause error) {
	if _, err := s.client.SSOProvider.UpdateOneID(providerID).SetStatus("error").SetLastError(truncateForLog(cause.Error(), 500)).Save(ctx); err != nil {
		s.logger.Errorw("failed to mark sso provider error", "provider_id", providerID, "error", err)
	}
}

func (s *SSOService) loginUserResponse(userEntity *ent.User) *dto.LoginUserResponse {
	mspRole := string(userEntity.MspRole)
	var mspRolePtr *string
	if mspRole != "" {
		mspRolePtr = &mspRole
	}
	return &dto.LoginUserResponse{
		ID:           userEntity.ID,
		Username:     userEntity.Username,
		Email:        userEntity.Email,
		Name:         userEntity.Name,
		Role:         string(userEntity.Role),
		MSPRole:      mspRolePtr,
		Department:   userEntity.Department,
		DepartmentID: userEntity.DepartmentID,
		Phone:        userEntity.Phone,
		Active:       userEntity.Active,
		TenantID:     userEntity.TenantID,
		CreatedAt:    userEntity.CreatedAt,
		UpdatedAt:    userEntity.UpdatedAt,
		Permissions:  userPermissions(userEntity),
	}
}

// resolveProviderForLogin 按编码解析启用中的提供方。
// 未指定租户时全局匹配，命中多个启用中的同编码提供方则拒绝（fail-closed），
// 与「用户名跨租户重名必须消歧」同一口径。
func (s *SSOService) resolveProviderForLogin(ctx context.Context, providerCode, tenantCode string) (*ent.SSOProvider, error) {
	code := strings.TrimSpace(providerCode)
	if code == "" {
		code = "default"
	}
	query := s.client.SSOProvider.Query().Where(ssoprovider.CodeEQ(code), ssoprovider.EnabledEQ(true))
	if strings.TrimSpace(tenantCode) != "" {
		tenantEntity, err := s.client.Tenant.Query().Where(tenant.CodeEQ(tenantCode)).Only(ctx)
		if err != nil {
			return nil, common.NewBusinessError(common.NotFoundCode, "Tenant does not exist", "")
		}
		query = query.Where(ssoprovider.TenantIDEQ(tenantEntity.ID))
	}
	providers, err := query.Order(ent.Asc(ssoprovider.FieldID)).All(ctx)
	if err != nil {
		return nil, common.NewBusinessError(common.InternalErrorCode, "Failed to query provider", "")
	}
	switch len(providers) {
	case 0:
		return nil, common.NewBusinessError(common.NotFoundCode, fmt.Sprintf("SSO provider %s does not exist or is not enabled", code), "")
	case 1:
		return providers[0], nil
	default:
		return nil, common.NewBusinessError(common.BadRequestCode, fmt.Sprintf("Provider code %s is enabled under multiple tenants, please bring the tenant code and try again", code), "")
	}
}

func (s *SSOService) resolveProviderByCode(ctx context.Context, providerCode string) (*ent.SSOProvider, error) {
	code := strings.TrimSpace(providerCode)
	if code == "" {
		return nil, fmt.Errorf("Missing provider code")
	}
	providers, err := s.client.SSOProvider.Query().Where(ssoprovider.CodeEQ(code)).Order(ent.Asc(ssoprovider.FieldID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to query provider")
	}
	switch len(providers) {
	case 0:
		return nil, fmt.Errorf("SSO provider %s does not exist", code)
	case 1:
		return providers[0], nil
	default:
		return nil, fmt.Errorf("Provider code %s exists under multiple tenants, please contact the administrator", code)
	}
}

// mustUserTenant 读取用户租户；用户已由调用方会话保证存在。
func mustUserTenant(ctx context.Context, s *SSOService, userID int) int {
	userEntity, err := s.client.User.Get(ctx, userID)
	if err != nil {
		return 0
	}
	return userEntity.TenantID
}

func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func maskSubject(subject string) string {
	if len(subject) <= 8 {
		return "***"
	}
	return subject[:4] + "***" + subject[len(subject)-4:]
}

func randomTokenHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func randomTokenBase64URL(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
