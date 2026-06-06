package common

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldUseSecureCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("localhost http does not force secure", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		req := httptest.NewRequest("POST", "http://localhost:8090/api/v1/auth/login", nil)
		req.Host = "localhost:8090"
		c.Request = req

		require.False(t, shouldUseSecureCookies(c))
		require.Equal(t, "", cookieDomain(c))
	})

	t.Run("forwarded https enables secure", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		req := httptest.NewRequest("POST", "http://example.com/api/v1/auth/login", nil)
		req.Host = "example.com"
		req.Header.Set("X-Forwarded-Proto", "https")
		c.Request = req

		require.True(t, shouldUseSecureCookies(c))
		require.Equal(t, "", cookieDomain(c))
	})
}
