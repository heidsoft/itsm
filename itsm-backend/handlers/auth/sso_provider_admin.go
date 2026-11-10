package auth

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ssoidentity"
	"itsm-backend/ent/ssoprovider"
)

// validateProviderInputs 在落库前校验提供方配置。
// 校验口径：
//   - protocol 只支持 oidc（SAML 未实现，不接受配置后静默不可用）；
//   - default_role 必须落在 JIT 白名单，防止配置错误即提权；
//   - 所有端点/issuer/redirect_uri 必须是 http(s) URL；
//   - redirect_uri 必须 https，唯一例外是 localhost/127.0.0.1（本地开发）。
func validateProviderInputs(issuer, authzURL, tokenURL, userinfoURL, jwksURL, redirectURI, defaultRole string) error {
	if defaultRole == "" {
		defaultRole = "end_user"
	}
	if !allowedJITRoles[defaultRole] {
		return fmt.Errorf("Default role only supports end_user/agent/technician, received value: %q", defaultRole)
	}
	if err := validateHTTPURL("issuer", issuer, false); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"authorization_endpoint": authzURL,
		"token_endpoint":         tokenURL,
		"userinfo_endpoint":      userinfoURL,
		"jwks_uri":               jwksURL,
	} {
		if strings.TrimSpace(value) == "" {
			continue // 端点可为空：由 Discovery 派生
		}
		if err := validateHTTPURL(name, value, false); err != nil {
			return err
		}
	}
	return validateHTTPURL("redirect_uri", redirectURI, true)
}

func validateHTTPURL(name, raw string, requireHTTPS bool) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fmt.Errorf("%s cannot be empty", name)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("%s is not a valid URL: %w", name, err)
	}
	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		if !requireHTTPS {
			return nil
		}
		host := strings.ToLower(parsed.Hostname())
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return nil
		}
		return fmt.Errorf("%s must use https (except localhost development)", name)
	default:
		return fmt.Errorf("%s only supports http/https", name)
	}
}

func normalizeOptionalURL(raw string) string {
	return strings.TrimSpace(raw)
}

// CreateProvider 创建提供方。ClientSecret 加密后落库，响应只回 hasClientSecret。
func (s *SSOService) CreateProvider(ctx context.Context, tenantID int, req *dto.SSOCreateProviderRequest) (*dto.SSOProviderResponse, error) {
	protocol := strings.TrimSpace(req.Protocol)
	if protocol == "" {
		protocol = "oidc"
	}
	if protocol != "oidc" {
		return nil, fmt.Errorf("Only oidc protocol supported, value received: %q", req.Protocol)
	}
	defaultRole := strings.TrimSpace(req.DefaultRole)
	if defaultRole == "" {
		defaultRole = "end_user"
	}
	if err := validateProviderInputs(req.Issuer, req.AuthorizationEndpoint, req.TokenEndpoint, req.UserinfoEndpoint, req.JWKSURI, req.RedirectURI, defaultRole); err != nil {
		return nil, err
	}
	exists, err := s.client.SSOProvider.Query().Where(ssoprovider.TenantIDEQ(tenantID), ssoprovider.CodeEQ(req.Code)).Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to check provider encoding")
	}
	if exists {
		return nil, fmt.Errorf("Provider code %s already exists", req.Code)
	}

	encryptedSecret, err := s.encryptSecret(req.ClientSecret)
	if err != nil {
		return nil, err
	}
	autoProvision := true
	if req.AutoProvision != nil {
		autoProvision = *req.AutoProvision
	}
	enabled := false
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	scopes := strings.TrimSpace(req.Scopes)
	if scopes == "" {
		scopes = "openid profile email"
	}

	provider, err := s.client.SSOProvider.Create().
		SetTenantID(tenantID).
		SetCode(req.Code).
		SetName(req.Name).
		SetProtocol(protocol).
		SetClientID(req.ClientID).
		SetNillableClientSecretEncrypted(nilIfEmpty(encryptedSecret)).
		SetIssuer(strings.TrimSpace(req.Issuer)).
		SetAuthorizationEndpoint(normalizeOptionalURL(req.AuthorizationEndpoint)).
		SetTokenEndpoint(normalizeOptionalURL(req.TokenEndpoint)).
		SetUserinfoEndpoint(normalizeOptionalURL(req.UserinfoEndpoint)).
		SetJwksURI(normalizeOptionalURL(req.JWKSURI)).
		SetScopes(scopes).
		SetRedirectURI(strings.TrimSpace(req.RedirectURI)).
		SetAutoProvision(autoProvision).
		SetDefaultRole(defaultRole).
		SetAllowedEmailDomains(strings.TrimSpace(req.AllowedEmailDomains)).
		SetEnabled(enabled).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, fmt.Errorf("Provider code %s already exists", req.Code)
		}
		return nil, fmt.Errorf("Failed to create provider")
	}
	return toProviderResponse(provider), nil
}

// GetProvider 读取单个提供方（租户隔离）。
func (s *SSOService) GetProvider(ctx context.Context, tenantID, providerID int) (*dto.SSOProviderResponse, error) {
	provider, err := s.client.SSOProvider.Query().
		Where(ssoprovider.IDEQ(providerID), ssoprovider.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("Provider does not exist")
		}
		return nil, fmt.Errorf("Failed to query provider")
	}
	return toProviderResponse(provider), nil
}

// ListProviders 分页读取租户下提供方。
func (s *SSOService) ListProviders(ctx context.Context, tenantID, page, pageSize int) ([]*dto.SSOProviderResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := s.client.SSOProvider.Query().Where(ssoprovider.TenantIDEQ(tenantID))
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to query provider")
	}
	providers, err := query.Order(ent.Desc(ssoprovider.FieldID)).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to query provider")
	}
	items := make([]*dto.SSOProviderResponse, 0, len(providers))
	for _, provider := range providers {
		items = append(items, toProviderResponse(provider))
	}
	return items, total, nil
}

// UpdateProvider 部分更新。ClientSecret 传空表示保持原值（原文不可读回，
// 「留空即不改」是唯一自洽语义）；传新值即轮换。
func (s *SSOService) UpdateProvider(ctx context.Context, tenantID, providerID int, req *dto.SSOUpdateProviderRequest) (*dto.SSOProviderResponse, error) {
	provider, err := s.client.SSOProvider.Query().
		Where(ssoprovider.IDEQ(providerID), ssoprovider.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("Provider does not exist")
		}
		return nil, fmt.Errorf("Failed to query provider")
	}

	update := s.client.SSOProvider.UpdateOneID(provider.ID)
	if req.Name != nil {
		update = update.SetName(*req.Name)
	}
	if req.ClientID != nil {
		update = update.SetClientID(*req.ClientID)
	}
	if req.ClientSecret != nil {
		if strings.TrimSpace(*req.ClientSecret) == "" {
			return nil, fmt.Errorf("client_secret cannot be cleared (if public clients are not supported, please delete the provider and rebuild)")
		}
		encrypted, encErr := s.encryptSecret(*req.ClientSecret)
		if encErr != nil {
			return nil, encErr
		}
		update = update.SetClientSecretEncrypted(encrypted)
	}
	if req.Issuer != nil {
		if err := validateHTTPURL("issuer", *req.Issuer, false); err != nil {
			return nil, err
		}
		update = update.SetIssuer(strings.TrimSpace(*req.Issuer))
	}
	if req.AuthorizationEndpoint != nil {
		if err := validateOptionalHTTPURL("authorization_endpoint", *req.AuthorizationEndpoint); err != nil {
			return nil, err
		}
		update = update.SetAuthorizationEndpoint(normalizeOptionalURL(*req.AuthorizationEndpoint))
	}
	if req.TokenEndpoint != nil {
		if err := validateOptionalHTTPURL("token_endpoint", *req.TokenEndpoint); err != nil {
			return nil, err
		}
		update = update.SetTokenEndpoint(normalizeOptionalURL(*req.TokenEndpoint))
	}
	if req.UserinfoEndpoint != nil {
		if err := validateOptionalHTTPURL("userinfo_endpoint", *req.UserinfoEndpoint); err != nil {
			return nil, err
		}
		update = update.SetUserinfoEndpoint(normalizeOptionalURL(*req.UserinfoEndpoint))
	}
	if req.JWKSURI != nil {
		if err := validateOptionalHTTPURL("jwks_uri", *req.JWKSURI); err != nil {
			return nil, err
		}
		update = update.SetJwksURI(normalizeOptionalURL(*req.JWKSURI))
	}
	if req.RedirectURI != nil {
		if err := validateHTTPURL("redirect_uri", *req.RedirectURI, true); err != nil {
			return nil, err
		}
		update = update.SetRedirectURI(strings.TrimSpace(*req.RedirectURI))
	}
	if req.Scopes != nil {
		scopes := strings.TrimSpace(*req.Scopes)
		if scopes == "" {
			scopes = "openid profile email"
		}
		update = update.SetScopes(scopes)
	}
	if req.AutoProvision != nil {
		update = update.SetAutoProvision(*req.AutoProvision)
	}
	if req.DefaultRole != nil {
		role := strings.TrimSpace(*req.DefaultRole)
		if !allowedJITRoles[role] {
			return nil, fmt.Errorf("Default role only supports end_user/agent/technician, received value: %q", role)
		}
		update = update.SetDefaultRole(role)
	}
	if req.AllowedEmailDomains != nil {
		update = update.SetAllowedEmailDomains(strings.TrimSpace(*req.AllowedEmailDomains))
	}
	if req.Enabled != nil {
		update = update.SetEnabled(*req.Enabled)
		if *req.Enabled {
			// 重新启用时清掉历史错误，避免状态栏停留在旧故障上。
			update = update.SetStatus("configured").ClearLastError()
		}
	}

	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to update provider")
	}
	return toProviderResponse(updated), nil
}

// DeleteProvider 删除提供方及其全部身份绑定（事务：半成品绑定会把用户锁在
// 「能登录但解绑不了」的状态）。
func (s *SSOService) DeleteProvider(ctx context.Context, tenantID, providerID int) error {
	provider, err := s.client.SSOProvider.Query().
		Where(ssoprovider.IDEQ(providerID), ssoprovider.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("Provider does not exist")
		}
		return fmt.Errorf("Failed to query provider")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("Failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.SSOIdentity.Delete().Where(ssoidentity.ProviderIDEQ(provider.ID)).Exec(ctx); err != nil {
		return fmt.Errorf("Failed to clean up identity binding")
	}
	if err := tx.SSOProvider.DeleteOneID(provider.ID).Exec(ctx); err != nil {
		return fmt.Errorf("Failed to delete provider")
	}
	return tx.Commit()
}

func (s *SSOService) encryptSecret(plain string) (string, error) {
	if strings.TrimSpace(plain) == "" {
		return "", nil
	}
	if s.encryption == nil {
		return "", fmt.Errorf("Encryption service is not configured and client_secret cannot be saved")
	}
	encrypted, err := s.encryption.Encrypt(plain)
	if err != nil {
		return "", fmt.Errorf("Encryption client_secret failed")
	}
	return encrypted, nil
}

func validateOptionalHTTPURL(name, raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return validateHTTPURL(name, raw, false)
}

func nilIfEmpty(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func toProviderResponse(provider *ent.SSOProvider) *dto.SSOProviderResponse {
	response := &dto.SSOProviderResponse{
		ID:                    provider.ID,
		TenantID:              provider.TenantID,
		Code:                  provider.Code,
		Name:                  provider.Name,
		Protocol:              provider.Protocol,
		ClientID:              provider.ClientID,
		HasClientSecret:       provider.ClientSecretEncrypted != "",
		Issuer:                provider.Issuer,
		AuthorizationEndpoint: provider.AuthorizationEndpoint,
		TokenEndpoint:         provider.TokenEndpoint,
		UserinfoEndpoint:      provider.UserinfoEndpoint,
		JWKSURI:               provider.JwksURI,
		Scopes:                provider.Scopes,
		RedirectURI:           provider.RedirectURI,
		AutoProvision:         provider.AutoProvision,
		DefaultRole:           provider.DefaultRole,
		AllowedEmailDomains:   provider.AllowedEmailDomains,
		Enabled:               provider.Enabled,
		Status:                provider.Status,
		LastError:             provider.LastError,
		CreatedAt:             provider.CreatedAt,
		UpdatedAt:             provider.UpdatedAt,
	}
	if provider.LastHealthCheckAt != nil {
		at := *provider.LastHealthCheckAt
		response.LastHealthCheckAt = &at
	}
	return response
}
