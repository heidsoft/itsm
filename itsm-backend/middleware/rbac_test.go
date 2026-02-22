package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRBACMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("No User ID in Context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)

		RBACMiddleware(nil)(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "用户未认证")
	})

	t.Run("No Tenant ID in Context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Set("user_id", 1)

		RBACMiddleware(nil)(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "租户信息缺失")
	})

	// Note: "No Role in Context" test requires a valid client to query the database
	// The RBACMiddleware will attempt to fetch user from DB which panics with nil client
	// This is expected behavior - in production, client should never be nil
}

func TestRequirePermissionForRBAC(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Missing Role Returns Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		// No role set

		RequirePermission("ticket", "read")(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "用户角色信息缺失")
	})

	t.Run("Missing Tenant ID Returns Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Set("role", "admin")
		// No tenant_id

		RequirePermission("ticket", "read")(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "租户信息缺失")
	})
}

func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Missing Role Returns Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)

		RequireRole("admin", "manager")(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "缺少角色信息")
	})

	t.Run("Role Not Allowed Returns Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Set("role", "end_user")

		RequireRole("admin", "manager")(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "无权限执行该操作")
	})

	t.Run("Role Allowed Succeeds", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Set("role", "admin")

		RequireRole("admin", "manager")(c)

		// Should not abort
		assert.False(t, c.IsAborted())
	})

	t.Run("Role Matching Is Case Insensitive", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Set("role", "ADMIN")

		RequireRole("admin", "manager")(c)

		// Should not abort
		assert.False(t, c.IsAborted())
	})
}

func TestHasResourcePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Super Admin Has All Permissions", func(t *testing.T) {
		result := hasResourcePermission(nil, "super_admin", "any_resource", "any_action", 1)
		assert.True(t, result)
	})

	t.Run("Admin Has Ticket Write Permission", func(t *testing.T) {
		result := hasResourcePermission(nil, "admin", "ticket", "write", 1)
		assert.True(t, result)
	})

	t.Run("End User Has Ticket Read Permission", func(t *testing.T) {
		result := hasResourcePermission(nil, "end_user", "ticket", "read", 1)
		assert.True(t, result)
	})

	t.Run("End User Does Not Have Ticket Delete Permission", func(t *testing.T) {
		result := hasResourcePermission(nil, "end_user", "ticket", "delete", 1)
		assert.False(t, result)
	})

	t.Run("Agent Has Dashboard Read Permission", func(t *testing.T) {
		result := hasResourcePermission(nil, "agent", "dashboard", "read", 1)
		assert.True(t, result)
	})

	t.Run("Technician Has CMDB Read Permission", func(t *testing.T) {
		result := hasResourcePermission(nil, "technician", "cmdb", "read", 1)
		assert.True(t, result)
	})

	t.Run("Manager Has User Read Permission", func(t *testing.T) {
		result := hasResourcePermission(nil, "manager", "user", "read", 1)
		assert.True(t, result)
	})

	t.Run("Unknown Role Has No Permissions", func(t *testing.T) {
		result := hasResourcePermission(nil, "unknown_role", "ticket", "read", 1)
		assert.False(t, result)
	})

	t.Run("Wildcard Permission", func(t *testing.T) {
		// Super admin has wildcard "*" permission
		result := hasResourcePermission(nil, "super_admin", "ticket", "delete", 1)
		assert.True(t, result)

		result = hasResourcePermission(nil, "super_admin", "anything", "anything", 1)
		assert.True(t, result)
	})
}

func TestMatchPath(t *testing.T) {
	t.Run("Exact Match", func(t *testing.T) {
		assert.True(t, matchPath("/api/v1/tickets", "/api/v1/tickets"))
		assert.False(t, matchPath("/api/v1/tickets", "/api/v1/users"))
	})

	t.Run("Wildcard Match", func(t *testing.T) {
		assert.True(t, matchPath("/api/v1/tickets/*", "/api/v1/tickets/123"))
		assert.True(t, matchPath("/api/v1/tickets/*", "/api/v1/tickets/abc/edit"))
		assert.False(t, matchPath("/api/v1/tickets/*", "/api/v1/users/123"))
	})

	t.Run("No Match", func(t *testing.T) {
		assert.False(t, matchPath("/api/v1/tickets", "/api/v1/users"))
		assert.False(t, matchPath("/api/v1/tickets/*", "/api/v1/tickets"))
	})
}

func TestGetPermissionFromPath(t *testing.T) {
	t.Run("GET Tickets Returns Read Permission", func(t *testing.T) {
		perm := getPermissionFromPath("GET", "/api/v1/tickets")
		assert.NotNil(t, perm)
		assert.Equal(t, "ticket", perm.Resource)
		assert.Equal(t, "read", perm.Action)
	})

	t.Run("POST Tickets Returns Write Permission", func(t *testing.T) {
		perm := getPermissionFromPath("POST", "/api/v1/tickets")
		assert.NotNil(t, perm)
		assert.Equal(t, "ticket", perm.Resource)
		assert.Equal(t, "write", perm.Action)
	})

	t.Run("DELETE Tickets Returns Delete Permission", func(t *testing.T) {
		perm := getPermissionFromPath("DELETE", "/api/v1/tickets/123")
		assert.NotNil(t, perm)
		assert.Equal(t, "ticket", perm.Resource)
		assert.Equal(t, "delete", perm.Action)
	})

	t.Run("Unknown Path Returns Nil", func(t *testing.T) {
		perm := getPermissionFromPath("GET", "/api/v1/unknown/path")
		assert.Nil(t, perm)
	})
}
