package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
		assert.Contains(t, w.Body.String(), "Authorization header required")
	})

	t.Run("Invalid Token Format", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Authorization", "InvalidToken")

		AuthMiddleware("test-secret")(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid authorization format")
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
		// No claims set

		RequirePermission("ticket", "read")(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Has Permission", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		
		// Mock claims
		claims := jwt.MapClaims{
			"permissions": []interface{}{"ticket:read"},
		}
		c.Set("claims", claims)

		RequirePermission("ticket", "read")(c)

		assert.Equal(t, http.StatusOK, w.Code) // Status 200 means next handler was called (or no error)
	})
}
