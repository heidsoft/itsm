package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"itsm-backend/ent/user"
)

func TestRBACBuiltinGuardAdditionalRole(t *testing.T) {
	client, _ := openSharedCacheSQLite(t, "rbac_builtin_guard")
	ctx := context.Background()
	tenant := client.Tenant.Create().SetName("A").SetCode("A").SetDomain("a.test").SaveX(ctx)
	extra := client.Role.Create().SetName("Forged").SetCode("super_admin").SetTenantID(tenant.ID).SetIsActive(true).SaveX(ctx)
	safe := client.Role.Create().SetName("Safe").SetCode("custom").SetTenantID(tenant.ID).SetIsActive(true).SaveX(ctx)
	actor := client.User.Create().SetUsername("actor").SetEmail("actor@a.test").SetPasswordHash("hash").SetName("Actor").SetRole(user.RoleAdmin).SetTenantID(tenant.ID).SetActive(true).AddRoles(extra, safe).SaveX(ctx)
	for _, primary := range []string{"admin", "super_admin"} {
		t.Run(primary, func(t *testing.T) {
			client.User.UpdateOneID(actor.ID).SetRole(user.Role(primary)).SaveX(ctx)
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("user_id", actor.ID)
				c.Set("tenant_id", tenant.ID)
				c.Set("role", primary)
				c.Next()
			}, RBACMiddleware(client))
			router.GET("/api/v1/auth/menus", func(c *gin.Context) {
				codes, exists := c.Get("roles")
				require.True(t, exists)
				require.Equal(t, []string{"custom"}, codes)
				roles := GetContextRoles(c)
				require.Contains(t, roles, "custom")
				if primary == "admin" {
					require.NotContains(t, roles, "super_admin")
				} else {
					require.Contains(t, roles, "super_admin")
					require.True(t, AuthorizeResource(ctx, client, roles, "unknown", "write", tenant.ID))
				}
				c.Status(204)
			})
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/auth/menus", nil))
			require.Equal(t, 204, w.Code, w.Body.String())
		})
	}
}
