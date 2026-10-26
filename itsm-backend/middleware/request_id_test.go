package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddleware_InjectsContextAndHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestIDMiddleware())

	var fromCtx string
	var ok bool
	var fromGin string
	r.GET("/x", func(c *gin.Context) {
		fromCtx, ok = RequestIDFromContext(c.Request.Context())
		fromGin = c.GetString("request_id")
	})

	req := httptest.NewRequest("GET", "/x", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.True(t, ok, "request_id 必须注入 request.Context，供 service/observer 层归因（c.Set 只对 gin 链可见）")
	assert.Len(t, fromCtx, 32, "未带头时应生成 32 位 hex request id")
	assert.Equal(t, fromGin, fromCtx, "gin 与 request context 必须是同一个 ID")
	assert.Equal(t, fromCtx, rec.Header().Get("X-Request-Id"))
}

func TestRequestIDMiddleware_PrefersIncomingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestIDMiddleware())

	var fromCtx string
	var ok bool
	r.GET("/x", func(c *gin.Context) {
		fromCtx, ok = RequestIDFromContext(c.Request.Context())
	})

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Request-Id", "caller-supplied-id")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.True(t, ok)
	assert.Equal(t, "caller-supplied-id", fromCtx)
	assert.Equal(t, "caller-supplied-id", rec.Header().Get("X-Request-Id"))
}
