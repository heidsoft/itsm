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
	handler := authHandler.NewHandler(authHandler.NewService(client, jwtSecret, logger, nil))
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
	refreshToken, err := middleware.GenerateRefreshToken(user.ID, user.Username, string(user.Role), tenant.ID, jwtSecret, time.Hour)
	require.NoError(t, err)
	handler := domainCommon.NewHandler(domainCommon.NewService(domainCommon.NewEntRepository(client), jwtSecret, logger, client))
	router := gin.New()
	SetupRoutes(router, &RouterConfig{JWTSecret: jwtSecret, Logger: logger, Client: client, CommonHandler: handler})

	assertSession := func(t *testing.T, response *httptest.ResponseRecorder, secure bool) {
		t.Helper()
		require.Equal(t, http.StatusOK, response.Code)
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
		require.Len(t, envelope.Data, 1)
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
			require.Contains(t, []string{"access_token", "refresh_token"}, cookie.Name)
			require.NotEmpty(t, cookie.Value)
			require.True(t, cookie.HttpOnly)
			require.Equal(t, secure, cookie.Secure)
			require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
			require.Equal(t, "/", cookie.Path)
			require.Empty(t, cookie.Domain)
			require.False(t, strings.Contains(response.Body.String(), cookie.Value), "JSON must not contain cookie credentials")
			var claims *middleware.Claims
			var tokenErr error
			if cookie.Name == "access_token" {
				require.Equal(t, 900, cookie.MaxAge)
				claims, tokenErr = middleware.ValidateAccessToken(cookie.Value, jwtSecret)
			} else {
				require.Equal(t, 604800, cookie.MaxAge)
				require.False(t, cookie.Value == refreshToken, "refresh credential must be renewed")
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
				assertSession(t, response, transport.secure)
			})
			for _, path := range []string{"/api/v1/auth/refresh", "/api/v1/refresh-token"} {
				t.Run(path, func(t *testing.T) {
					request := httptest.NewRequest(http.MethodPost, transport.origin+path, nil)
					request.Header.Set("X-Forwarded-Proto", transport.forwarded)
					request.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
					response := httptest.NewRecorder()
					router.ServeHTTP(response, request)
					assertSession(t, response, transport.secure)
				})
			}
		})
	}

	for _, path := range []string{"/api/v1/auth/refresh", "/api/v1/refresh-token"} {
		t.Run("JSON refresh request "+path, func(t *testing.T) {
			body, err := json.Marshal(map[string]string{"refreshToken": refreshToken})
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			assertSession(t, response, false)
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
