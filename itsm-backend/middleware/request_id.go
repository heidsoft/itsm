package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

type requestIDCtxKey struct{}

// RequestIDMiddleware injects a unique X-Request-Id into context and response headers
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-Id")
		if reqID == "" {
			reqID = generateRequestID()
		}
		// Set into context and response header
		c.Set("request_id", reqID)
		c.Writer.Header().Set("X-Request-Id", reqID)
		// 同步注入 request.Context：c.Set 只对 gin 链可见，service/observer 层
		// 只能从 context.Context 取值（与 tenant 注入同口径）。
		c.Request = c.Request.WithContext(WithRequestID(c.Request.Context(), reqID))

		c.Next()
	}
}

// WithRequestID 把 request ID 注入 context.Context。生产链路由
// RequestIDMiddleware 调用；Worker/skill 等非 HTTP 入口也可复用，保证
// 底账、日志、metric 的 request_id 归因同源。
func WithRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, requestIDCtxKey{}, reqID)
}

// RequestIDFromContext 返回请求链路携带的 request ID；未注入时 ok=false。
func RequestIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(requestIDCtxKey{}).(string)
	return v, ok
}

func generateRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// fallback to zero bytes; hex still deterministic
	}
	return hex.EncodeToString(b[:])
}
