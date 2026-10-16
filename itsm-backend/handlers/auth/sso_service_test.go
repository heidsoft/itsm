package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/ssoidentity"
	"itsm-backend/ent/ssoprovider"
	"itsm-backend/middleware"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type ssoFixture struct {
	client  *ent.Client
	service *SSOService
	states  SSOStateStore
	tenant  *ent.Tenant
	ctx     context.Context
}

func newSSOFixture(t *testing.T) *ssoFixture {
	t.Helper()
	ctx := context.Background()
	// 每个测试独立内存库：cache=shared 的库名在同一进程内跨测试复用会撞
	// tenants.code 唯一键（fixture 每次新建同编码租户）。
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:sso_%s?mode=memory&cache=shared&_fk=1", t.Name()))
	logger := zaptest.NewLogger(t).Sugar()

	states := NewSSOStateStore(nil)
	encryption := middleware.NewEncryptionService("test-encryption-key-16bytes")
	service := NewSSOService(client, "sso-test-secret", logger, encryption, states)

	tenant, err := client.Tenant.Create().
		SetName("SSO Tenant").
		SetCode("sso-tenant").
		SetDomain("sso.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	return &ssoFixture{client: client, service: service, states: states, tenant: tenant, ctx: ctx}
}

func (f *ssoFixture) createProvider(t *testing.T, code string, autoProvision bool, defaultRole string) *ent.SSOProvider {
	t.Helper()
	provider, err := f.client.SSOProvider.Create().
		SetTenantID(f.tenant.ID).
		SetCode(code).
		SetName("Provider " + code).
		SetProtocol("oidc").
		SetClientID("client-" + code).
		SetClientSecretEncrypted("encrypted-secret").
		SetIssuer("https://idp.example.com/" + code).
		SetAuthorizationEndpoint("https://idp.example.com/" + code + "/authorize").
		SetTokenEndpoint("https://idp.example.com/" + code + "/token").
		SetUserinfoEndpoint("https://idp.example.com/" + code + "/userinfo").
		SetJwksURI("https://idp.example.com/" + code + "/jwks").
		SetRedirectURI("https://itsm.example.com/sso/callback").
		SetAutoProvision(autoProvision).
		SetDefaultRole(defaultRole).
		SetEnabled(true).
		Save(f.ctx)
	require.NoError(t, err)
	return provider
}

func TestSSOInitiate_GeneratesPKCEChallengeAndSingleUseState(t *testing.T) {
	f := newSSOFixture(t)
	f.createProvider(t, "okta", true, "end_user")

	response, err := f.service.Initiate(f.ctx, "okta", "", 0)
	require.NoError(t, err)
	require.NotEmpty(t, response.RedirectURL)
	require.NotEmpty(t, response.State)
	assert.Contains(t, response.RedirectURL, "code_challenge_method=S256")
	assert.Contains(t, response.RedirectURL, "response_type=code")
	assert.Contains(t, response.RedirectURL, "state="+response.State)
	assert.Contains(t, response.RedirectURL, "nonce=")

	// state 单次消费：第二次 Consume 必须失败（重放防护）。
	record, err := f.states.Consume(f.ctx, response.State)
	require.NoError(t, err)
	assert.Equal(t, "login", record.Mode)
	assert.NotEmpty(t, record.Verifier)
	assert.NotEmpty(t, record.Nonce)

	_, err = f.states.Consume(f.ctx, response.State)
	assert.ErrorIs(t, err, ErrSSOStateNotFound, "state 必须单次消费")
}

func TestSSOInitiate_BindModeRecordsUser(t *testing.T) {
	f := newSSOFixture(t)
	f.createProvider(t, "bind-idp", true, "end_user")

	user, err := f.client.User.Create().
		SetUsername("bind-user").SetEmail("bind@example.com").SetName("Bind User").
		SetPasswordHash("unused").SetTenantID(f.tenant.ID).SetActive(true).Save(f.ctx)
	require.NoError(t, err)

	response, err := f.service.Initiate(f.ctx, "bind-idp", "", user.ID)
	require.NoError(t, err)
	record, err := f.states.Consume(f.ctx, response.State)
	require.NoError(t, err)
	assert.Equal(t, "bind", record.Mode)
	assert.Equal(t, user.ID, record.BindUserID)
}

func TestSSOInitiate_UnknownOrDisabledProviderRejected(t *testing.T) {
	f := newSSOFixture(t)
	provider := f.createProvider(t, "disabled-idp", true, "end_user")
	_, err := f.client.SSOProvider.UpdateOneID(provider.ID).SetEnabled(false).Save(f.ctx)
	require.NoError(t, err)

	_, err = f.service.Initiate(f.ctx, "disabled-idp", "", 0)
	assert.Error(t, err, "停用的提供方不得发起登录")

	_, err = f.service.Initiate(f.ctx, "missing-idp", "", 0)
	assert.Error(t, err, "不存在的提供方必须失败")
}

func TestSSOInitiate_AmbiguousCodeFailsClosed(t *testing.T) {
	f := newSSOFixture(t)
	// 第二个租户启用同编码提供方：不带租户编码时必须拒绝，绝不任选一个。
	tenantB, err := f.client.Tenant.Create().
		SetName("Tenant B").SetCode("tenant-b").SetDomain("b.example.com").SetStatus("active").Save(f.ctx)
	require.NoError(t, err)
	_, err = f.client.SSOProvider.Create().
		SetTenantID(tenantB.ID).SetCode("shared").SetName("Shared B").SetProtocol("oidc").
		SetClientID("client-b").SetIssuer("https://idp.example.com/b").
		SetAuthorizationEndpoint("https://idp.example.com/b/authorize").
		SetTokenEndpoint("https://idp.example.com/b/token").
		SetJwksURI("https://idp.example.com/b/jwks").
		SetRedirectURI("https://itsm.example.com/sso/callback").
		SetAutoProvision(true).SetDefaultRole("end_user").SetEnabled(true).Save(f.ctx)
	require.NoError(t, err)
	f.createProvider(t, "shared", true, "end_user")

	_, err = f.service.Initiate(f.ctx, "shared", "", 0)
	assert.Error(t, err, "跨租户同编码必须 fail-closed")

	response, err := f.service.Initiate(f.ctx, "shared", "sso-tenant", 0)
	require.NoError(t, err)
	assert.Equal(t, "shared", response.Provider)
}

func TestSSOResolveUser_BindingBeatsEmail(t *testing.T) {
	f := newSSOFixture(t)
	provider := f.createProvider(t, "binding-idp", true, "end_user")

	boundUser, err := f.client.User.Create().
		SetUsername("bound").SetEmail("bound@example.com").SetName("Bound").
		SetPasswordHash("unused").SetTenantID(f.tenant.ID).SetActive(true).Save(f.ctx)
	require.NoError(t, err)
	_, err = f.client.SSOIdentity.Create().
		SetTenantID(f.tenant.ID).SetProviderID(provider.ID).SetUserID(boundUser.ID).
		SetSubject("sub-bound").Save(f.ctx)
	require.NoError(t, err)

	// 同邮箱的另一个已存在账号不得抢戏：绑定优先。
	_, err = f.client.User.Create().
		SetUsername("other").SetEmail("bound@example.com").SetName("Other").
		SetPasswordHash("unused").SetTenantID(f.tenant.ID).SetActive(true).Save(f.ctx)
	require.Error(t, err, "fixture 预期邮箱全局唯一，绑定用例不需要第二个同邮箱用户")

	user, provisioned, err := f.service.resolveUser(f.ctx, provider, &oidcIdentity{Subject: "sub-bound", Email: "bound@example.com"})
	require.NoError(t, err)
	assert.False(t, provisioned)
	assert.Equal(t, boundUser.ID, user.ID)
}

func TestSSOResolveUser_VerifiedEmailLinksExistingAccount(t *testing.T) {
	f := newSSOFixture(t)
	provider := f.createProvider(t, "link-idp", true, "end_user")

	existing, err := f.client.User.Create().
		SetUsername("legacy").SetEmail("legacy@example.com").SetName("Legacy").
		SetPasswordHash("unused").SetTenantID(f.tenant.ID).SetActive(true).Save(f.ctx)
	require.NoError(t, err)

	user, provisioned, err := f.service.resolveUser(f.ctx, provider, &oidcIdentity{
		Subject: "sub-legacy", Email: "legacy@example.com", EmailVerified: true,
	})
	require.NoError(t, err)
	assert.False(t, provisioned)
	assert.Equal(t, existing.ID, user.ID)

	binding, err := f.client.SSOIdentity.Query().
		Where(ssoidentity.ProviderIDEQ(provider.ID), ssoidentity.SubjectEQ("sub-legacy")).Only(f.ctx)
	require.NoError(t, err)
	assert.Equal(t, existing.ID, binding.UserID, "已验证邮箱必须落绑定关系")
}

func TestSSOResolveUser_UnverifiedEmailNeverLinks(t *testing.T) {
	f := newSSOFixture(t)
	provider := f.createProvider(t, "unverified-idp", false, "end_user")

	_, err := f.client.User.Create().
		SetUsername("victim").SetEmail("victim@example.com").SetName("Victim").
		SetPasswordHash("unused").SetTenantID(f.tenant.ID).SetActive(true).Save(f.ctx)
	require.NoError(t, err)

	_, _, err = f.service.resolveUser(f.ctx, provider, &oidcIdentity{
		Subject: "sub-attacker", Email: "victim@example.com", EmailVerified: false,
	})
	assert.Error(t, err, "未验证邮箱不得链接他人账号；auto_provision 关闭时必须失败")
}

func TestSSOResolveUser_JITProvision(t *testing.T) {
	f := newSSOFixture(t)
	provider := f.createProvider(t, "jit-idp", true, "agent")

	user, provisioned, err := f.service.resolveUser(f.ctx, provider, &oidcIdentity{
		Subject: "sub-new", Email: "new.user@example.com", PreferredUsername: "New.User", Name: "New User",
	})
	require.NoError(t, err)
	assert.True(t, provisioned)
	assert.Equal(t, "new.user", user.Username, "用户名应来自 preferred_username 并小写化")
	assert.Equal(t, "agent", string(user.Role))
	assert.Equal(t, f.tenant.ID, user.TenantID)
	assert.True(t, user.Active)
	assert.NotEmpty(t, user.PasswordHash, "JIT 账号密码为随机值，不得为空")

	binding, err := f.client.SSOIdentity.Query().
		Where(ssoidentity.ProviderIDEQ(provider.ID), ssoidentity.SubjectEQ("sub-new")).Only(f.ctx)
	require.NoError(t, err)
	assert.Equal(t, user.ID, binding.UserID)
}

func TestSSOResolveUser_JITRejectsPrivilegedDefaultRole(t *testing.T) {
	f := newSSOFixture(t)
	// default_role=admin 的提供方（模拟历史脏配置）绝不允许 JIT 提权。
	provider := f.createProvider(t, "priv-idp", true, "admin")

	_, _, err := f.service.resolveUser(f.ctx, provider, &oidcIdentity{
		Subject: "sub-priv", Email: "priv@example.com",
	})
	assert.Error(t, err, "JIT 不得授予 admin 及以上角色")
}

func TestSSOResolveUser_EmailDomainWhitelist(t *testing.T) {
	f := newSSOFixture(t)
	provider := f.createProvider(t, "domain-idp", true, "end_user")
	_, err := f.client.SSOProvider.UpdateOneID(provider.ID).SetAllowedEmailDomains("corp.example.com, partner.com").Save(f.ctx)
	require.NoError(t, err)
	// 重新读取：resolveUser 消费的是传入的结构体，更新后的字段必须重新加载。
	provider, err = f.client.SSOProvider.Get(f.ctx, provider.ID)
	require.NoError(t, err)

	_, _, err = f.service.resolveUser(f.ctx, provider, &oidcIdentity{
		Subject: "sub-domain", Email: "user@gmail.com", EmailVerified: true,
	})
	assert.Error(t, err, "白名单之外的邮箱域必须拒绝")

	user, _, err := f.service.resolveUser(f.ctx, provider, &oidcIdentity{
		Subject: "sub-domain-ok", Email: "user@corp.example.com", EmailVerified: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "user@corp.example.com", user.Email)
}

func TestSSOCallback_InvalidStateRejected(t *testing.T) {
	f := newSSOFixture(t)
	f.createProvider(t, "cb-idp", true, "end_user")

	_, err := f.service.Callback(f.ctx, "cb-idp", "some-code", "bogus-state")
	assert.Error(t, err, "未知 state 必须失败，不得建立会话")
}

func TestSSOBind_RequiresMatchingSession(t *testing.T) {
	f := newSSOFixture(t)
	f.createProvider(t, "bind-flow", true, "end_user")

	user, err := f.client.User.Create().
		SetUsername("binder").SetEmail("binder@example.com").SetName("Binder").
		SetPasswordHash("unused").SetTenantID(f.tenant.ID).SetActive(true).Save(f.ctx)
	require.NoError(t, err)

	response, err := f.service.Initiate(f.ctx, "bind-flow", "", user.ID)
	require.NoError(t, err)

	// state 是 bind 模式，但 Callback（登录语义）必须拒绝，防止绑定流程被当成登录。
	_, err = f.service.Callback(f.ctx, "bind-flow", "code", response.State)
	assert.Error(t, err)

	// 会话用户与 state 记录不一致必须拒绝。
	other, err := f.client.User.Create().
		SetUsername("other-binder").SetEmail("other-binder@example.com").SetName("Other").
		SetPasswordHash("unused").SetTenantID(f.tenant.ID).SetActive(true).Save(f.ctx)
	require.NoError(t, err)
	response2, err := f.service.Initiate(f.ctx, "bind-flow", "", user.ID)
	require.NoError(t, err)
	err = f.service.Bind(f.ctx, other.ID, "bind-flow", "code", response2.State)
	assert.Error(t, err, "会话与绑定请求不匹配必须拒绝")
}

func TestSSOProviderAdmin_ValidationAndLifecycle(t *testing.T) {
	f := newSSOFixture(t)

	// http 回调（非 localhost）拒绝
	_, err := f.service.CreateProvider(f.ctx, f.tenant.ID, ssoCreateRequest("okta", "https://itsm.example.com/sso/callback"))
	require.NoError(t, err)
	_, err = f.service.CreateProvider(f.ctx, f.tenant.ID, ssoCreateRequest("insecure", "http://itsm.example.com/sso/callback"))
	assert.Error(t, err, "非 localhost 的 http 回调必须拒绝")

	// localhost http 放行（本地开发）
	_, err = f.service.CreateProvider(f.ctx, f.tenant.ID, ssoCreateRequest("local", "http://localhost:3000/sso/callback"))
	require.NoError(t, err)

	// admin 默认角色拒绝
	_, err = f.service.CreateProvider(f.ctx, f.tenant.ID, ssoCreateRequestWithRole("admin-idp", "https://itsm.example.com/sso/callback", "admin"))
	assert.Error(t, err, "默认角色不允许 admin")

	// 重复编码拒绝
	_, err = f.service.CreateProvider(f.ctx, f.tenant.ID, ssoCreateRequest("okta", "https://itsm.example.com/sso/callback"))
	assert.Error(t, err, "同租户重复编码必须拒绝")

	// 更新：secret 留空保持原值，传新值即轮换
	created, err := f.service.GetProvider(f.ctx, f.tenant.ID, mustProviderID(t, f, "okta"))
	require.NoError(t, err)
	assert.True(t, created.HasClientSecret)
	assert.NotContains(t, mustMarshalJSON(t, created), "secret-value", "响应不得回显 secret 原文或密文")

	newName := "Renamed"
	updated, err := f.service.UpdateProvider(f.ctx, f.tenant.ID, mustProviderID(t, f, "okta"), &dto.SSOUpdateProviderRequest{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", updated.Name)
	assert.True(t, updated.HasClientSecret, "secret 留空时必须保持原值")

	rotated := "rotated-secret"
	_, err = f.service.UpdateProvider(f.ctx, f.tenant.ID, mustProviderID(t, f, "okta"), &dto.SSOUpdateProviderRequest{ClientSecret: &rotated})
	require.NoError(t, err)
	stored, err := f.client.SSOProvider.Query().Where(ssoprovider.TenantIDEQ(f.tenant.ID), ssoprovider.CodeEQ("okta")).Only(f.ctx)
	require.NoError(t, err)
	assert.NotEqual(t, "encrypted-secret", stored.ClientSecretEncrypted, "轮换后密文必须变化")
	decrypted, decErr := f.service.encryption.Decrypt(stored.ClientSecretEncrypted)
	require.NoError(t, decErr)
	assert.Equal(t, "rotated-secret", decrypted)

	// 删除：提供方与绑定一并清除
	user, err := f.client.User.Create().
		SetUsername("del-user").SetEmail("del@example.com").SetName("Del").
		SetPasswordHash("unused").SetTenantID(f.tenant.ID).SetActive(true).Save(f.ctx)
	require.NoError(t, err)
	_, err = f.client.SSOIdentity.Create().
		SetTenantID(f.tenant.ID).SetProviderID(stored.ID).SetUserID(user.ID).SetSubject("sub-del").Save(f.ctx)
	require.NoError(t, err)

	require.NoError(t, f.service.DeleteProvider(f.ctx, f.tenant.ID, stored.ID))
	count, err := f.client.SSOIdentity.Query().Where(ssoidentity.ProviderIDEQ(stored.ID)).Count(f.ctx)
	require.NoError(t, err)
	assert.Zero(t, count, "删除提供方必须连带清理身份绑定")
}

func TestSSOProviderAdmin_TenantIsolation(t *testing.T) {
	f := newSSOFixture(t)
	provider := f.createProvider(t, "iso-idp", true, "end_user")

	otherTenant, err := f.client.Tenant.Create().
		SetName("Other").SetCode("other").SetDomain("other.example.com").SetStatus("active").Save(f.ctx)
	require.NoError(t, err)

	_, err = f.service.GetProvider(f.ctx, otherTenant.ID, provider.ID)
	assert.Error(t, err, "跨租户读取必须 404")
	_, err = f.service.UpdateProvider(f.ctx, otherTenant.ID, provider.ID, &dto.SSOUpdateProviderRequest{})
	assert.Error(t, err, "跨租户更新必须拒绝")
	assert.Error(t, f.service.DeleteProvider(f.ctx, otherTenant.ID, provider.ID), "跨租户删除必须拒绝")
}

func TestSSOStateStore_MemorySingleUse(t *testing.T) {
	store := newMemorySSOStateStore()
	record := &SSOStateRecord{ProviderID: 1, Mode: "login", Nonce: "n", Verifier: "v"}
	require.NoError(t, store.Save(context.Background(), "state-1", record, SSOStateTTL))

	consumed, err := store.Consume(context.Background(), "state-1")
	require.NoError(t, err)
	assert.Equal(t, "login", consumed.Mode)

	_, err = store.Consume(context.Background(), "state-1")
	assert.ErrorIs(t, err, ErrSSOStateNotFound)
}

func TestSSOStateStore_MemoryCollisionRejected(t *testing.T) {
	store := newMemorySSOStateStore()
	record := &SSOStateRecord{ProviderID: 1, Mode: "login"}
	require.NoError(t, store.Save(context.Background(), "state-x", record, SSOStateTTL))
	err := store.Save(context.Background(), "state-x", record, SSOStateTTL)
	assert.Error(t, err, "state 碰撞不得覆盖已有记录")
}

// ---------------------------------------------------------------------
// 测试辅助
// ---------------------------------------------------------------------

func ssoCreateRequest(code, redirectURI string) *dto.SSOCreateProviderRequest {
	autoProvision := true
	enabled := true
	return &dto.SSOCreateProviderRequest{
		Code:                  code,
		Name:                  "Provider " + code,
		ClientID:              "client-" + code,
		ClientSecret:          "secret-value",
		Issuer:                "https://idp.example.com/" + code,
		AuthorizationEndpoint: "https://idp.example.com/" + code + "/authorize",
		TokenEndpoint:         "https://idp.example.com/" + code + "/token",
		UserinfoEndpoint:      "https://idp.example.com/" + code + "/userinfo",
		JWKSURI:               "https://idp.example.com/" + code + "/jwks",
		RedirectURI:           redirectURI,
		AutoProvision:         &autoProvision,
		Enabled:               &enabled,
	}
}

func ssoCreateRequestWithRole(code, redirectURI, role string) *dto.SSOCreateProviderRequest {
	req := ssoCreateRequest(code, redirectURI)
	req.DefaultRole = role
	return req
}

func mustProviderID(t *testing.T, f *ssoFixture, code string) int {
	t.Helper()
	provider, err := f.client.SSOProvider.Query().
		Where(ssoprovider.TenantIDEQ(f.tenant.ID), ssoprovider.CodeEQ(code)).Only(f.ctx)
	require.NoError(t, err)
	return provider.ID
}

func mustMarshalJSON(t *testing.T, v interface{}) string {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return string(raw)
}
