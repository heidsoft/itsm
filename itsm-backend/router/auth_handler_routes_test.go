package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
