package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent/enttest"
	authHandler "itsm-backend/handlers/auth"
	domainCommon "itsm-backend/handlers/common"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/crypto/bcrypt"
)

func TestSetupRoutes_AuthHandlerProductionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:auth-handler-routes?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	const jwtSecret = "auth-handler-route-test-secret"
	handler := authHandler.NewHandler(authHandler.NewService(client, jwtSecret, logger))
	commonHandler := domainCommon.NewHandler(domainCommon.NewService(domainCommon.NewEntRepository(client), jwtSecret, logger, client))

	tenantA, err := client.Tenant.Create().SetName("Tenant A").SetCode("TENANT-A").SetDomain("a.example.com").SetStatus("active").Save(context.Background())
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().SetName("Tenant B").SetCode("TENANT-B").SetDomain("b.example.com").SetStatus("active").Save(context.Background())
	require.NoError(t, err)
	userA, err := client.User.Create().SetUsername("route-user").SetEmail("route@example.com").SetName("Route User").SetPasswordHash("unused").SetTenantID(tenantA.ID).SetActive(true).Save(context.Background())
	require.NoError(t, err)

	router := gin.New()
	SetupRoutes(router, &RouterConfig{JWTSecret: jwtSecret, Logger: logger, Client: client, AuthHandler: handler, CommonHandler: commonHandler})

	t.Run("register retains request and response contract", func(t *testing.T) {
		body := []byte(`{"username":"newuser","email":"new@example.com","password":"password123","fullName":"New User","tenantCode":"TENANT-A"}`)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var envelope common.Response
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
		require.Equal(t, common.SuccessCode, envelope.Code)
	})

	t.Run("switch tenant rejects cross-tenant access", func(t *testing.T) {
		token, err := middleware.GenerateAccessToken(userA.ID, userA.Username, string(userA.Role), tenantA.ID, jwtSecret, 15*time.Minute)
		require.NoError(t, err)
		body := []byte(`{"tenantId":` + strconv.Itoa(tenantB.ID) + `}`)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/switch-tenant", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
	})
}

func TestSetupRoutes_AuthCookieOnlyResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:auth-cookie-responses?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	const jwtSecret = "auth-cookie-response-test-secret"
	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("Cookie Tenant").SetCode("COOKIE").SetDomain("cookie.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	require.NoError(t, err)
	user, err := client.User.Create().SetUsername("cookie-user").SetEmail("cookie@example.com").SetName("Cookie User").SetPasswordHash(string(passwordHash)).SetTenantID(tenant.ID).SetActive(true).Save(ctx)
	require.NoError(t, err)
	handler := domainCommon.NewHandler(domainCommon.NewService(domainCommon.NewEntRepository(client), jwtSecret, logger, client))
	router := gin.New()
	SetupRoutes(router, &RouterConfig{JWTSecret: jwtSecret, Logger: logger, Client: client, CommonHandler: handler})

	// refresh token 是单次使用凭证：rotation 成功即吊销旧值。每个续签用例必须自带一枚新鲜
	// token，共享 fixture 会让第二个用例合理地失败。TTL 偏移保证 token 内容互不相同，
	// 而不是依赖签发时间的偶然差异。
	var minted int
	seen := map[string]bool{}
	freshRefreshToken := func(t *testing.T) string {
		t.Helper()
		minted++
		token, err := middleware.GenerateRefreshToken(user.ID, user.Username, string(user.Role), tenant.ID, jwtSecret, time.Hour+time.Duration(minted)*time.Second)
		require.NoError(t, err)
		require.False(t, seen[token], "refresh fixture must be unique per case")
		seen[token] = true
		return token
	}

	assertSession := func(t *testing.T, response *httptest.ResponseRecorder, secure bool, presentedRefresh string) {
		t.Helper()
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var envelope struct {
			Code    int                        `json:"code"`
			Message string                     `json:"message"`
			Data    map[string]json.RawMessage `json:"data"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
		require.Equal(t, common.SuccessCode, envelope.Code)
		require.NotEmpty(t, envelope.Message)
		for _, field := range []string{"accessToken", "refreshToken", "access_token", "refresh_token"} {
			_, exists := envelope.Data[field]
			require.False(t, exists, "JSON response must omit %s", field)
		}
		require.Len(t, envelope.Data, 2)
		var expiresIn int
		require.NoError(t, json.Unmarshal(envelope.Data["expiresIn"], &expiresIn))
		require.Equal(t, int(middleware.AccessTokenTTL.Seconds()), expiresIn, "expiresIn must come from the server-side TTL")
		var returnedUser map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(envelope.Data["user"], &returnedUser))
		for _, field := range []string{"password", "passwordHash", "tenant_id"} {
			_, exists := returnedUser[field]
			require.False(t, exists, "user response must omit %s", field)
		}
		var identity struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
			TenantID int    `json:"tenantId"`
		}
		require.NoError(t, json.Unmarshal(envelope.Data["user"], &identity))
		require.Equal(t, user.ID, identity.ID)
		require.Equal(t, user.Username, identity.Username)
		require.Equal(t, tenant.ID, identity.TenantID)
		cookies := response.Result().Cookies()
		require.Len(t, cookies, 2)
		for _, cookie := range cookies {
			require.Contains(t, []string{middleware.AccessTokenCookie, middleware.RefreshTokenCookie}, cookie.Name)
			require.NotEmpty(t, cookie.Value)
			require.True(t, cookie.HttpOnly)
			require.Equal(t, secure, cookie.Secure)
			require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
			require.Equal(t, "/", cookie.Path)
			require.Empty(t, cookie.Domain)
			require.NotContains(t, response.Body.String(), cookie.Value, "JSON must not contain cookie credentials")
			var claims *middleware.Claims
			var tokenErr error
			if cookie.Name == middleware.AccessTokenCookie {
				require.Equal(t, int(middleware.AccessTokenTTL.Seconds()), cookie.MaxAge)
				claims, tokenErr = middleware.ValidateAccessToken(cookie.Value, jwtSecret)
			} else {
				require.Equal(t, int(middleware.RefreshTokenTTL.Seconds()), cookie.MaxAge)
				if presentedRefresh != "" {
					require.NotEqual(t, presentedRefresh, cookie.Value, "refresh credential must be renewed")
				}
				claims, tokenErr = middleware.ValidateRefreshToken(cookie.Value, jwtSecret)
			}
			require.NoError(t, tokenErr)
			require.NotNil(t, claims)
			require.Equal(t, user.ID, claims.UserID)
			require.Equal(t, tenant.ID, claims.TenantID)
		}
	}

	for _, transport := range []struct {
		name      string
		origin    string
		forwarded string
		secure    bool
	}{
		{name: "http", origin: "http://localhost"},
		{name: "https", origin: "https://cookie.example.com", secure: true},
		{name: "forwarded https", origin: "http://cookie.example.com", forwarded: "https", secure: true},
	} {
		t.Run(transport.name, func(t *testing.T) {
			t.Run("login", func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPost, transport.origin+"/api/v1/auth/login", strings.NewReader(`{"username":"cookie-user","password":"password123","tenantCode":"COOKIE"}`))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("X-Forwarded-Proto", transport.forwarded)
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				assertSession(t, response, transport.secure, "")
			})
			for _, path := range []string{"/api/v1/auth/refresh", "/api/v1/refresh-token"} {
				t.Run(path, func(t *testing.T) {
					refreshToken := freshRefreshToken(t)
					request := httptest.NewRequest(http.MethodPost, transport.origin+path, nil)
					request.Header.Set("X-Forwarded-Proto", transport.forwarded)
					request.AddCookie(&http.Cookie{Name: middleware.RefreshTokenCookie, Value: refreshToken})
					response := httptest.NewRecorder()
					router.ServeHTTP(response, request)
					assertSession(t, response, transport.secure, refreshToken)
				})
			}
		})
	}

	for _, path := range []string{"/api/v1/auth/refresh", "/api/v1/refresh-token"} {
		t.Run("JSON refresh request "+path, func(t *testing.T) {
			refreshToken := freshRefreshToken(t)
			body, err := json.Marshal(map[string]string{"refreshToken": refreshToken})
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			assertSession(t, response, false, refreshToken)
		})
	}

	for _, test := range []struct {
		name string
		path string
		body string
	}{
		{name: "invalid password", path: "/api/v1/auth/login", body: `{"username":"cookie-user","password":"wrong-password","tenantCode":"COOKIE"}`},
		{name: "invalid refresh", path: "/api/v1/auth/refresh", body: `{"refreshToken":"invalid-token"}`},
		{name: "invalid legacy refresh", path: "/api/v1/refresh-token", body: `{"refreshToken":"invalid-token"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, http.StatusUnauthorized, response.Code)
			var envelope common.Response
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
			require.Equal(t, common.AuthFailedCode, envelope.Code)
			require.Nil(t, envelope.Data)
			require.Empty(t, response.Result().Cookies())
		})
	}
}

// TestSetupRoutes_SessionTruthAndLogout 锁死会话真相的后端语义：登出在 access token
// 过期时仍必须清 cookie 并吊销 refresh token；refresh token 单次使用；
// /auth/session 是前端唯一的会话来源。
func TestSetupRoutes_SessionTruthAndLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:auth-session-truth?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	const jwtSecret = "auth-session-truth-test-secret"
	ctx := context.Background()

	tenant, err := client.Tenant.Create().SetName("Session Tenant").SetCode("SESSION").SetDomain("session.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().SetUsername("session-user").SetEmail("session@example.com").SetName("Session User").SetPasswordHash("unused").SetTenantID(tenant.ID).SetActive(true).Save(ctx)
	require.NoError(t, err)

	handler := domainCommon.NewHandler(domainCommon.NewService(domainCommon.NewEntRepository(client), jwtSecret, logger, client))
	router := gin.New()
	SetupRoutes(router, &RouterConfig{JWTSecret: jwtSecret, Logger: logger, Client: client, CommonHandler: handler})

	mintSession := func(t *testing.T, accessTTL time.Duration) (string, string) {
		t.Helper()
		accessToken, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), tenant.ID, jwtSecret, accessTTL)
		require.NoError(t, err)
		refreshToken, err := middleware.GenerateRefreshToken(user.ID, user.Username, string(user.Role), tenant.ID, jwtSecret, time.Hour)
		require.NoError(t, err)
		return accessToken, refreshToken
	}

	assertCleared := func(t *testing.T, response *httptest.ResponseRecorder) {
		t.Helper()
		cleared := map[string]bool{}
		for _, cookie := range response.Result().Cookies() {
			require.Empty(t, cookie.Value, "cleared cookie must not carry a value")
			require.Negative(t, cookie.MaxAge, "cleared cookie must expire immediately")
			cleared[cookie.Name] = true
		}
		require.True(t, cleared[middleware.AccessTokenCookie], "logout must clear the access cookie")
		require.True(t, cleared[middleware.RefreshTokenCookie], "logout must clear the refresh cookie")
	}

	t.Run("logout succeeds when the access token already expired", func(t *testing.T) {
		accessToken, refreshToken := mintSession(t, -time.Minute)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		request.AddCookie(&http.Cookie{Name: middleware.AccessTokenCookie, Value: accessToken})
		request.AddCookie(&http.Cookie{Name: middleware.RefreshTokenCookie, Value: refreshToken})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		// 过期凭证过去会在这里 401，浏览器留下 7 天的 refresh cookie，
		// 下一次请求又被自动续签成「登不掉」的会话。
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		assertCleared(t, response)

		refresh := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		refresh.AddCookie(&http.Cookie{Name: middleware.RefreshTokenCookie, Value: refreshToken})
		refreshResponse := httptest.NewRecorder()
		router.ServeHTTP(refreshResponse, refresh)
		require.Equal(t, http.StatusUnauthorized, refreshResponse.Code, refreshResponse.Body.String())
		var envelope common.Response
		require.NoError(t, json.Unmarshal(refreshResponse.Body.Bytes(), &envelope))
		require.Equal(t, common.AuthFailedCode, envelope.Code)
	})

	t.Run("logout without any credential is idempotent", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	})

	t.Run("refresh token is single use", func(t *testing.T) {
		_, refreshToken := mintSession(t, middleware.AccessTokenTTL)

		first := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		first.AddCookie(&http.Cookie{Name: middleware.RefreshTokenCookie, Value: refreshToken})
		firstResponse := httptest.NewRecorder()
		router.ServeHTTP(firstResponse, first)
		require.Equal(t, http.StatusOK, firstResponse.Code, firstResponse.Body.String())

		// 重放旧凭证（并发续签的第二个请求、被截获的旧 cookie）必须被拒绝。
		replay := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		replay.AddCookie(&http.Cookie{Name: middleware.RefreshTokenCookie, Value: refreshToken})
		replayResponse := httptest.NewRecorder()
		router.ServeHTTP(replayResponse, replay)
		require.Equal(t, http.StatusUnauthorized, replayResponse.Code, replayResponse.Body.String())
	})

	t.Run("session endpoint is the only session truth", func(t *testing.T) {
		accessToken, _ := mintSession(t, middleware.AccessTokenTTL)
		request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
		request.AddCookie(&http.Cookie{Name: middleware.AccessTokenCookie, Value: accessToken})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())

		var envelope struct {
			Code    int                        `json:"code"`
			Message string                     `json:"message"`
			Data    map[string]json.RawMessage `json:"data"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
		require.Equal(t, common.SuccessCode, envelope.Code)
		require.Len(t, envelope.Data, 3)
		for _, field := range []string{"user", "tenants", "expiresIn"} {
			_, exists := envelope.Data[field]
			require.True(t, exists, "session response must carry %s", field)
		}
		var identity struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
			TenantID int    `json:"tenantId"`
		}
		require.NoError(t, json.Unmarshal(envelope.Data["user"], &identity))
		require.Equal(t, user.ID, identity.ID)
		require.Equal(t, tenant.ID, identity.TenantID)
		var tenants []struct {
			ID     int    `json:"id"`
			Code   string `json:"code"`
			Status string `json:"status"`
		}
		require.NoError(t, json.Unmarshal(envelope.Data["tenants"], &tenants))
		require.Len(t, tenants, 1)
		require.Equal(t, tenant.Code, tenants[0].Code)
		require.NotContains(t, response.Body.String(), accessToken, "session response must not echo credentials")

		unauthenticated := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
		unauthenticatedResponse := httptest.NewRecorder()
		router.ServeHTTP(unauthenticatedResponse, unauthenticated)
		require.Equal(t, http.StatusUnauthorized, unauthenticatedResponse.Code)
	})
}
