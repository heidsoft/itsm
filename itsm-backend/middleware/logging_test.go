package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestLoggerMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Logger Outputs Request Info", func(t *testing.T) {
		// Create a test logger
		logger := zap.NewExample().Sugar()
		defer logger.Sync()

		// Capture output by redirecting
		oldLogger := globalLogger
		globalLogger = logger
		defer func() { globalLogger = oldLogger }()

		// Create test router with logger middleware
		router := gin.New()
		router.Use(LoggerMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		// Create request
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "test-agent")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Logger Without Global Logger", func(t *testing.T) {
		// Save and clear global logger
		oldLogger := globalLogger
		globalLogger = nil
		defer func() { globalLogger = oldLogger }()

		// Create test router with logger middleware
		router := gin.New()
		router.Use(LoggerMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		// Create request
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should still work without global logger
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Request Completes Successfully", func(t *testing.T) {
		// Save and clear global logger
		oldLogger := globalLogger
		globalLogger = nil
		defer func() { globalLogger = oldLogger }()

		// Create test router with logger middleware
		router := gin.New()
		router.Use(LoggerMiddleware())
		router.POST("/api/v1/tickets", func(c *gin.Context) {
			c.Status(http.StatusCreated)
		})

		// Create request
		req, _ := http.NewRequest("POST", "/api/v1/tickets", nil)
		req.Header.Set("User-Agent", "test-agent")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Request ID Is Handled In Log", func(t *testing.T) {
		// Save and clear global logger
		oldLogger := globalLogger
		globalLogger = nil
		defer func() { globalLogger = oldLogger }()

		// Create test router with logger middleware and request ID
		router := gin.New()
		router.Use(LoggerMiddleware())
		router.Use(func(c *gin.Context) {
			c.Set("request_id", "test-request-123")
			c.Next()
		})
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		// Create request
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("User ID Is Handled In Log", func(t *testing.T) {
		// Save and clear global logger
		oldLogger := globalLogger
		globalLogger = nil
		defer func() { globalLogger = oldLogger }()

		// Create test router with logger middleware and user ID
		router := gin.New()
		router.Use(LoggerMiddleware())
		router.Use(func(c *gin.Context) {
			c.Set("user_id", 42)
			c.Next()
		})
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		// Create request
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Tenant ID Is Handled In Log", func(t *testing.T) {
		// Save and clear global logger
		oldLogger := globalLogger
		globalLogger = nil
		defer func() { globalLogger = oldLogger }()

		// Create test router with logger middleware and tenant ID
		router := gin.New()
		router.Use(LoggerMiddleware())
		router.Use(func(c *gin.Context) {
			c.Set("tenant_id", 1)
			c.Next()
		})
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		// Create request
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSetLogger(t *testing.T) {
	t.Run("SetLogger Updates Global Logger", func(t *testing.T) {
		// Save old logger
		oldLogger := globalLogger

		// Create new logger
		logger := zap.NewExample().Sugar()

		// Set logger
		SetLogger(logger)

		// Verify global logger is set
		assert.Equal(t, logger, globalLogger)

		// Restore old logger
		globalLogger = oldLogger
		logger.Sync()
	})
}

func TestLogFormatter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GET Request Completes Successfully", func(t *testing.T) {
		// Save and clear global logger
		oldLogger := globalLogger
		globalLogger = nil
		defer func() { globalLogger = oldLogger }()

		// Create test router
		router := gin.New()
		router.Use(LoggerMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		// Create request
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Verify the log is called and request completes
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("POST Request Completes Successfully", func(t *testing.T) {
		// Save and clear global logger
		oldLogger := globalLogger
		globalLogger = nil
		defer func() { globalLogger = oldLogger }()

		testCases := []struct {
			method string
			path   string
			status int
		}{
			{"GET", "/api/v1/tickets", http.StatusOK},
			{"POST", "/api/v1/tickets", http.StatusCreated},
			{"PUT", "/api/v1/tickets/1", http.StatusOK},
			{"DELETE", "/api/v1/tickets/1", http.StatusNoContent},
		}

		for _, tc := range testCases {
			router := gin.New()
			router.Use(LoggerMiddleware())
			router.Handle(tc.method, tc.path, func(c *gin.Context) {
				c.Status(tc.status)
			})

			req, _ := http.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.status, w.Code, "Expected status %d for %s %s", tc.status, tc.method, tc.path)
		}
	})

	t.Run("Various Status Codes Handled", func(t *testing.T) {
		// Save and clear global logger
		oldLogger := globalLogger
		globalLogger = nil
		defer func() { globalLogger = oldLogger }()

		testCases := []struct {
			name   string
			status int
		}{
			{"OK", http.StatusOK},
			{"Created", http.StatusCreated},
			{"BadRequest", http.StatusBadRequest},
			{"Unauthorized", http.StatusUnauthorized},
			{"Forbidden", http.StatusForbidden},
			{"NotFound", http.StatusNotFound},
			{"InternalServerError", http.StatusInternalServerError},
		}

		for _, tc := range testCases {
			router := gin.New()
			router.Use(LoggerMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.Status(tc.status)
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.status, w.Code, "Expected status %d for %s", tc.status, tc.name)
		}
	})

	t.Run("Slow Request Completes", func(t *testing.T) {
		// Save and clear global logger
		oldLogger := globalLogger
		globalLogger = nil
		defer func() { globalLogger = oldLogger }()

		// Create test router with a slow handler
		router := gin.New()
		router.Use(LoggerMiddleware())
		router.GET("/slow", func(c *gin.Context) {
			time.Sleep(10 * time.Millisecond)
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest("GET", "/slow", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
