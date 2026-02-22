package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("No Token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)

		AuthMiddleware("test-secret")(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "缺少认证token")
	})

	t.Run("Invalid Token Format", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Authorization", "InvalidToken")

		AuthMiddleware("test-secret")(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "token格式错误")
	})

	// Note: Valid token test would require a shared secret/mocking key generation logic
	// which depends on the actual implementation details in auth.go
}

func TestRequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Missing Permissions in Context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		// No role/tenant_id set

		RequirePermission("ticket", "read")(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "用户角色信息缺失")
	})

	t.Run("Missing Tenant in Context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		// Set role but no tenant_id
		c.Set("role", "admin")

		RequirePermission("ticket", "read")(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "租户信息缺失")
	})

	t.Run("Has Permission", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)

		// Mock role and tenant_id
		c.Set("role", "admin")
		c.Set("tenant_id", 1)

		RequirePermission("ticket", "read")(c)

		// Status 0 or 200 means next handler was called (or no error)
		// Since we don't have a real handler, the response body will be empty or have success response
		assert.NotEqual(t, http.StatusUnauthorized, w.Code)
	})
}
