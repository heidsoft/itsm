package rbac

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"itsm-backend/common"
	"itsm-backend/middleware"
)

func TestRoleBuiltinGuardHTTP(t *testing.T) {
	h, client := setupRBACTest(t)
	ctx := context.Background()
	tenant := client.Tenant.Create().SetName("A").SetCode("A").SetDomain("a.test").SaveX(ctx)
	normal := client.Role.Create().SetName("Custom").SetCode("custom").SetTenantID(tenant.ID).SaveX(ctx)
	legacy := client.Role.Create().SetName("Legacy").SetCode("admin").SetTenantID(tenant.ID).SaveX(ctx)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID})
		c.Next()
	})
	// Same production handlers as router/common_system_routes.go.
	router.POST("/api/v1/roles", h.CreateRole)
	router.PUT("/api/v1/roles/:id", h.UpdateRole)
	request := func(t *testing.T, method, path, body string, status, code int) rbacEnvelope {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		require.Equal(t, status, w.Code, w.Body.String())
		var result rbacEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
		require.Equal(t, code, result.Code)
		return result
	}
	for _, code := range []string{"super_admin", "admin", "it_admin"} {
		t.Run("create_"+code, func(t *testing.T) {
			request(t, "POST", "/api/v1/roles", fmt.Sprintf(`{"name":"Injected","code":%q,"isSystem":true}`, code), 409, common.ConflictCode)
		})
		t.Run("update_"+code, func(t *testing.T) {
			request(t, "PUT", fmt.Sprintf("/api/v1/roles/%d", normal.ID), fmt.Sprintf(`{"code":%q}`, code), 409, common.ConflictCode)
			require.Equal(t, "custom", client.Role.GetX(ctx, normal.ID).Code)
		})
	}
	t.Run("generated_builtin_code", func(t *testing.T) {
		request(t, "POST", "/api/v1/roles", `{"name":"SUPER ADMIN"}`, 409, common.ConflictCode)
	})
	for _, r := range []struct {
		id   int
		code string
	}{{normal.ID, "custom"}, {legacy.ID, "admin"}} {
		request(t, "PUT", fmt.Sprintf("/api/v1/roles/%d", r.id), fmt.Sprintf(`{"code":%q}`, r.code), 200, 0)
	}
	result := request(t, "POST", "/api/v1/roles", `{"name":"Safe","code":"safe","isSystem":true}`, 200, 0)
	var data map[string]any
	require.NoError(t, json.Unmarshal(result.Data, &data))
	id := int(data["id"].(float64))
	require.False(t, client.Role.GetX(ctx, id).IsSystem)
	require.Equal(t, "safe", data["code"])
	require.Contains(t, data, "createdAt")
	require.NotContains(t, data, "tenant_id")
}
