package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

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

		c.Next()
	}
}

func generateRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// fallback to zero bytes; hex still deterministic
	}
	return hex.EncodeToString(b[:])
}
