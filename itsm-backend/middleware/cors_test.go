package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Save original env
	originalEnv := os.Getenv("ITSM_CORS_ALLOWED_ORIGINS")
	defer os.Setenv("ITSM_CORS_ALLOWED_ORIGINS", originalEnv)

	// Clear the env to use default behavior (echo origin)
	os.Unsetenv("ITSM_CORS_ALLOWED_ORIGINS")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("OPTIONS", "/", nil)
	c.Request.Header.Set("Origin", "http://localhost:3000")

	CORSMiddleware()(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	// In development mode (no ITSM_CORS_ALLOWED_ORIGINS set), it echoes the origin
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORSMiddleware_WithAllowedOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set allowed origins
	os.Setenv("ITSM_CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8080")
	defer os.Unsetenv("ITSM_CORS_ALLOWED_ORIGINS")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("OPTIONS", "/", nil)
	c.Request.Header.Set("Origin", "http://localhost:3000")

	CORSMiddleware()(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_WithoutOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Clear the env
	os.Unsetenv("ITSM_CORS_ALLOWED_ORIGINS")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("OPTIONS", "/", nil)
	// No Origin header

	CORSMiddleware()(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	// Without origin header, defaults to *
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}
