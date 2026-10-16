package router

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/user"
	authHandler "itsm-backend/handlers/auth"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// mockIdP 是一个最小 OIDC 提供方：JWKS + 令牌交换。
// id_token 故意不填 nonce（部分企业 IdP 行为）：验证「缺失只告警不阻断」，
// 而 state 单次消费 + PKCE 仍提供重放防护。nonce 不匹配的拒绝由
// handlers/auth 包内测试覆盖。
type mockIdP struct {
	server        *httptest.Server
	key           *rsa.PrivateKey
	clientID      string
	issuer        string
	subject       string
	email         string
	nonceOverride string // 非空时 id_token 填该 nonce（用于不匹配场景）
}

func newMockIdP(t *testing.T, clientID, subject, email string) *mockIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	idp := &mockIdP{key: key, clientID: clientID, subject: subject, email: email}

	mux := http.NewServeMux()
	idp.server = httptest.NewServer(mux)
	idp.issuer = idp.server.URL
	t.Cleanup(idp.server.Close)

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"keys": []map[string]string{{
				"kty": "RSA",
				"kid": "mock-key",
				"alg": "RS256",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString([]byte{0x01, 0x00, 0x01}),
			}},
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		// PKCE verifier 必须随令牌交换到达，否则说明发起链路被改坏。
		assert.NotEmpty(t, r.Form.Get("code_verifier"), "令牌交换必须携带 PKCE code_verifier")
		assert.Equal(t, idp.clientID, r.Form.Get("client_id"))
		now := time.Now()
		claims := jwt.MapClaims{
			"iss":                idp.issuer,
			"aud":                idp.clientID,
			"sub":                idp.subject,
			"email":              idp.email,
			"email_verified":     true,
			"preferred_username": "idp.user",
			"name":               "IdP User",
			"iat":                now.Unix(),
			"exp":                now.Add(5 * time.Minute).Unix(),
		}
		if idp.nonceOverride != "" {
			claims["nonce"] = idp.nonceOverride
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = "mock-key"
		signed, signErr := token.SignedString(key)
		require.NoError(t, signErr)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     signed,
		})
	})
	return idp
}

func (idp *mockIdP) createProviderRequest(code string) *dto.SSOCreateProviderRequest {
	enabled := true
	return &dto.SSOCreateProviderRequest{
		Code:                  code,
		Name:                  "Mock IdP",
		ClientID:              idp.clientID,
		ClientSecret:          "mock-secret",
		Issuer:                idp.issuer,
		AuthorizationEndpoint: idp.issuer + "/authorize",
		TokenEndpoint:         idp.issuer + "/token",
		UserinfoEndpoint:      idp.issuer + "/userinfo",
		JWKSURI:               idp.issuer + "/jwks",
		RedirectURI:           "https://itsm.example.com/sso/callback",
		AllowedEmailDomains:   "",
		Enabled:               &enabled,
	}
}

func setupSSORouter(t *testing.T, dsn string) (*gin.Engine, *ent.Client, *authHandler.Handler, *authHandler.SSOService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", dsn)
	logger := zaptest.NewLogger(t).Sugar()

	authService := authHandler.NewService(client, "sso-router-secret", logger)
	httpHandler := authHandler.NewHandler(authService)
	ssoService := authHandler.NewSSOService(
		client,
		"sso-router-secret",
		logger,
		middleware.NewEncryptionService("sso-router-encryption-key"),
		authHandler.NewSSOStateStore(nil),
	)
	httpHandler.SetSSOService(ssoService)

	router := gin.New()
	SetupRoutes(router, &RouterConfig{
		JWTSecret:   "sso-router-secret",
		Logger:      logger,
		Client:      client,
		AuthHandler: httpHandler,
	})
	_ = ctx
	return router, client, httpHandler, ssoService
}

func postJSON(t *testing.T, router *gin.Engine, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	return postJSONWithToken(t, router, path, body, "")
}

// postJSONWithToken 在 postJSON 基础上附带 access token cookie。
//
// SSO 自助面（bind/unbind/identities）挂 AuthMiddleware，必须带凭据；
// 公开面（initiate/callback）传空 token 即可。原先只有一个不带 token 的
// postJSON，导致自助面用例恒 401 —— 测试自己把正确的 401 当成了实现缺陷。
func postJSONWithToken(t *testing.T, router *gin.Engine, path string, body interface{}, token string) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.AddCookie(&http.Cookie{Name: middleware.AccessTokenCookie, Value: token})
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestSetupRoutes_SSOEndToEndLoginWithMockIdP(t *testing.T) {
	router, client, _, ssoService := setupSSORouter(t, "file:sso_router_e2e?mode=memory&cache=shared&_fk=1")
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("SSO Router Tenant").SetCode("sso-router").SetDomain("sso.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	idp := newMockIdP(t, "router-client", "router-subject", "router.user@example.com")
	_, err = ssoService.CreateProvider(ctx, tenant.ID, idp.createProviderRequest("mock"))
	require.NoError(t, err)

	// 登录页数据源：公开列出启用中的提供方。
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/v1/auth/sso/providers", nil))
	require.Equal(t, http.StatusOK, listResponse.Code, listResponse.Body.String())
	var listEnvelope struct {
		Code int `json:"code"`
		Data []struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listResponse.Body.Bytes(), &listEnvelope))
	require.Len(t, listEnvelope.Data, 1)
	assert.Equal(t, "mock", listEnvelope.Data[0].Code)

	// 发起：拿到授权 URL 与 state。
	initiateResponse := postJSON(t, router, "/api/v1/auth/sso/initiate", map[string]string{"provider": "mock"})
	require.Equal(t, http.StatusOK, initiateResponse.Code, initiateResponse.Body.String())
	var initiateEnvelope struct {
		Code int `json:"code"`
		Data struct {
			RedirectURL string `json:"redirectUrl"`
			State       string `json:"state"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(initiateResponse.Body.Bytes(), &initiateEnvelope))
	require.Equal(t, common.SuccessCode, initiateEnvelope.Code)
	assert.Contains(t, initiateEnvelope.Data.RedirectURL, idp.issuer+"/authorize")
	require.NotEmpty(t, initiateEnvelope.Data.State)

	// 回调：mock IdP 换 token + 验签 + JIT 建号 + 签发会话。
	callbackResponse := postJSON(t, router, "/api/v1/auth/sso/callback?provider=mock", map[string]string{
		"code":  "mock-auth-code",
		"state": initiateEnvelope.Data.State,
	})
	require.Equal(t, http.StatusOK, callbackResponse.Code, callbackResponse.Body.String())

	var sessionEnvelope struct {
		Code int `json:"code"`
		Data struct {
			ExpiresIn   int  `json:"expiresIn"`
			Provisioned bool `json:"provisioned"`
			User        struct {
				ID       int    `json:"id"`
				Username string `json:"username"`
				Email    string `json:"email"`
				TenantID int    `json:"tenantId"`
				Role     string `json:"role"`
			} `json:"user"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(callbackResponse.Body.Bytes(), &sessionEnvelope))
	require.Equal(t, common.SuccessCode, sessionEnvelope.Code)
	assert.True(t, sessionEnvelope.Data.Provisioned, "首次 SSO 登录必须 JIT 建号")
	assert.Equal(t, int(middleware.AccessTokenTTL.Seconds()), sessionEnvelope.Data.ExpiresIn)
	assert.Equal(t, tenant.ID, sessionEnvelope.Data.User.TenantID)
	assert.Equal(t, "router.user@example.com", sessionEnvelope.Data.User.Email)

	// 凭证只进 httpOnly cookie，响应体不得回显。
	cookies := map[string]*http.Cookie{}
	for _, cookie := range callbackResponse.Result().Cookies() {
		cookies[cookie.Name] = cookie
	}
	require.Contains(t, cookies, middleware.AccessTokenCookie)
	require.Contains(t, cookies, middleware.RefreshTokenCookie)
	for name, cookie := range cookies {
		assert.True(t, cookie.HttpOnly, "%s 必须 HttpOnly", name)
		assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		assert.NotContains(t, callbackResponse.Body.String(), cookie.Value, "响应体不得包含 %s", name)
	}
	assert.NotContains(t, callbackResponse.Body.String(), "accessToken")
	assert.NotContains(t, callbackResponse.Body.String(), "refreshToken")

	// JIT 用户真实落库。
	created, err := client.User.Query().Where(user.EmailEQ("router.user@example.com")).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, sessionEnvelope.Data.User.ID, created.ID)
	assert.Equal(t, "end_user", string(created.Role))

	// state 单次消费：重放同一 state 必须 401 且不建立会话。
	replay := postJSON(t, router, "/api/v1/auth/sso/callback?provider=mock", map[string]string{
		"code":  "mock-auth-code",
		"state": initiateEnvelope.Data.State,
	})
	assert.Equal(t, http.StatusUnauthorized, replay.Code, replay.Body.String())
	assert.Empty(t, replay.Result().Cookies(), "重放不得写入任何会话凭证")
}

func TestSetupRoutes_SSOCallbackInvalidStateRejected(t *testing.T) {
	router, client, _, _ := setupSSORouter(t, "file:sso_router_badstate?mode=memory&cache=shared&_fk=1")
	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("T").SetCode("t").SetDomain("t.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	idp := newMockIdP(t, "bad-state-client", "sub", "user@example.com")
	_ = tenant
	_ = idp

	response := postJSON(t, router, "/api/v1/auth/sso/callback?provider=mock", map[string]string{
		"code":  "code",
		"state": "never-issued",
	})
	assert.Equal(t, http.StatusUnauthorized, response.Code)
	assert.Empty(t, response.Result().Cookies())
}

func TestSetupRoutes_SSOProviderAdminPermissionGating(t *testing.T) {
	router, client, _, ssoService := setupSSORouter(t, "file:sso_router_admin?mode=memory&cache=shared&_fk=1")
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Admin Tenant").SetCode("sso-admin").SetDomain("admin.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	superAdmin, err := client.User.Create().
		SetUsername("sso-super").SetEmail("sso-super@example.com").SetName("Super").
		SetPasswordHash("unused").SetRole("super_admin").SetTenantID(tenant.ID).SetActive(true).Save(ctx)
	require.NoError(t, err)
	endUser, err := client.User.Create().
		SetUsername("sso-end").SetEmail("sso-end@example.com").SetName("End").
		SetPasswordHash("unused").SetRole("end_user").SetTenantID(tenant.ID).SetActive(true).Save(ctx)
	require.NoError(t, err)

	idp := newMockIdP(t, "admin-client", "sub", "user@example.com")
	_, err = ssoService.CreateProvider(ctx, tenant.ID, idp.createProviderRequest("mock"))
	require.NoError(t, err)

	getWithToken := func(path string, u *ent.User) *httptest.ResponseRecorder {
		token, tokenErr := middleware.GenerateAccessToken(u.ID, u.Username, string(u.Role), tenant.ID, "sso-router-secret", 15*time.Minute)
		require.NoError(t, tokenErr)
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(&http.Cookie{Name: middleware.AccessTokenCookie, Value: token})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}

	unauthenticated := httptest.NewRecorder()
	router.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/api/v1/admin/sso/providers", nil))
	assert.Equal(t, http.StatusUnauthorized, unauthenticated.Code, "管理面必须要求认证")

	forbidden := getWithToken("/api/v1/admin/sso/providers", endUser)
	assert.Equal(t, http.StatusForbidden, forbidden.Code, "end_user 不得读取 SSO 配置")

	allowed := getWithToken("/api/v1/admin/sso/providers", superAdmin)
	require.Equal(t, http.StatusOK, allowed.Code, allowed.Body.String())
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				Code            string `json:"code"`
				HasClientSecret bool   `json:"hasClientSecret"`
			} `json:"items"`
			Total int `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(allowed.Body.Bytes(), &envelope))
	require.Equal(t, 1, envelope.Data.Total)
	assert.Equal(t, "mock", envelope.Data.Items[0].Code)
	assert.True(t, envelope.Data.Items[0].HasClientSecret)
	assert.NotContains(t, allowed.Body.String(), "mock-secret", "响应不得回显 client_secret")
}

func TestSetupRoutes_SSOBindAndUnbindRoundTrip(t *testing.T) {
	router, client, _, ssoService := setupSSORouter(t, "file:sso_router_bind?mode=memory&cache=shared&_fk=1")
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Bind Tenant").SetCode("sso-bind").SetDomain("bind.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	sessionUser, err := client.User.Create().
		SetUsername("bind-session").SetEmail("bind-session@example.com").SetName("Bind Session").
		SetPasswordHash("unused").SetRole("agent").SetTenantID(tenant.ID).SetActive(true).Save(ctx)
	require.NoError(t, err)

	idp := newMockIdP(t, "bind-client", "bind-subject", "bind-session@example.com")
	_, err = ssoService.CreateProvider(ctx, tenant.ID, idp.createProviderRequest("mock"))
	require.NoError(t, err)

	token, err := middleware.GenerateAccessToken(sessionUser.ID, sessionUser.Username, string(sessionUser.Role), tenant.ID, "sso-router-secret", 15*time.Minute)
	require.NoError(t, err)

	// 携带会话发起 → 服务端应记录 bind 模式。
	initiateRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sso/initiate", bytes.NewReader([]byte(`{"provider":"mock"}`)))
	initiateRequest.Header.Set("Content-Type", "application/json")
	initiateRequest.AddCookie(&http.Cookie{Name: middleware.AccessTokenCookie, Value: token})
	initiateResponse := httptest.NewRecorder()
	router.ServeHTTP(initiateResponse, initiateRequest)
	require.Equal(t, http.StatusOK, initiateResponse.Code, initiateResponse.Body.String())
	var initiateEnvelope struct {
		Data struct {
			State string `json:"state"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(initiateResponse.Body.Bytes(), &initiateEnvelope))

	// 绑定：复用回调链路完成身份交换（bind 模式由 state 决定）。
	// 必须携带会话 cookie：bind 属于已认证端点，且会话用户须与 state 记录一致。
	bindPayload, marshalErr := json.Marshal(map[string]string{
		"provider": "mock",
		"code":     "bind-code",
		"state":    initiateEnvelope.Data.State,
	})
	require.NoError(t, marshalErr)
	bindRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sso/bind", bytes.NewReader(bindPayload))
	bindRequest.Header.Set("Content-Type", "application/json")
	bindRequest.AddCookie(&http.Cookie{Name: middleware.AccessTokenCookie, Value: token})
	bindResponse := httptest.NewRecorder()
	router.ServeHTTP(bindResponse, bindRequest)
	require.Equal(t, http.StatusOK, bindResponse.Code, bindResponse.Body.String())

	// 绑定关系可查。
	identitiesRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sso/identities", nil)
	identitiesRequest.AddCookie(&http.Cookie{Name: middleware.AccessTokenCookie, Value: token})
	identitiesResponse := httptest.NewRecorder()
	router.ServeHTTP(identitiesResponse, identitiesRequest)
	require.Equal(t, http.StatusOK, identitiesResponse.Code, identitiesResponse.Body.String())
	var identitiesEnvelope struct {
		Data []struct {
			ProviderCode string `json:"providerCode"`
			Subject      string `json:"subject"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(identitiesResponse.Body.Bytes(), &identitiesEnvelope))
	require.Len(t, identitiesEnvelope.Data, 1)
	assert.Equal(t, "mock", identitiesEnvelope.Data[0].ProviderCode)

	// 解绑后再查为空。unbind 挂 AuthMiddleware，必须带会话凭据 ——
	// 上面 GET identities 带了 cookie，此处不带就是测试自身的疏漏。
	unbindResponse := postJSONWithToken(t, router, "/api/v1/auth/sso/unbind", map[string]string{"provider": "mock"}, token)
	require.Equal(t, http.StatusOK, unbindResponse.Code, unbindResponse.Body.String())

	identitiesResponse2 := httptest.NewRecorder()
	router.ServeHTTP(identitiesResponse2, identitiesRequest)
	require.Equal(t, http.StatusOK, identitiesResponse2.Code, identitiesResponse2.Body.String())
	require.NoError(t, json.Unmarshal(identitiesResponse2.Body.Bytes(), &identitiesEnvelope))
	assert.Empty(t, identitiesEnvelope.Data, "解绑后不得残留绑定关系")
}

func TestSetupRoutes_SSOCallbackNonceMismatchRejected(t *testing.T) {
	router, client, _, ssoService := setupSSORouter(t, "file:sso_router_nonce?mode=memory&cache=shared&_fk=1")
	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Nonce Tenant").SetCode("sso-nonce").SetDomain("nonce.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	idp := newMockIdP(t, "nonce-client", "nonce-subject", "nonce.user@example.com")
	idp.nonceOverride = "attacker-chosen-nonce"
	_, err = ssoService.CreateProvider(ctx, tenant.ID, idp.createProviderRequest("mock"))
	require.NoError(t, err)

	initiateResponse := postJSON(t, router, "/api/v1/auth/sso/initiate", map[string]string{"provider": "mock"})
	require.Equal(t, http.StatusOK, initiateResponse.Code, initiateResponse.Body.String())
	var initiateEnvelope struct {
		Data struct {
			State string `json:"state"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(initiateResponse.Body.Bytes(), &initiateEnvelope))

	callbackResponse := postJSON(t, router, "/api/v1/auth/sso/callback?provider=mock", map[string]string{
		"code":  "nonce-code",
		"state": initiateEnvelope.Data.State,
	})
	assert.Equal(t, http.StatusUnauthorized, callbackResponse.Code, "nonce 不匹配必须拒绝")
	assert.Empty(t, callbackResponse.Result().Cookies(), "拒绝路径不得写入会话凭证")

	count, err := client.User.Query().Where(user.EmailEQ("nonce.user@example.com")).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count, "验证失败不得 JIT 建号")
}
