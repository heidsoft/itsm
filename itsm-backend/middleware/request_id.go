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
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDCtxKey{}, reqID))

		c.Next()
	}
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
