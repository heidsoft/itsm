package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RecoveryMiddleware 恢复中间件，处理panic
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if globalLogger != nil {
			if rid, exists := c.Get("request_id"); exists {
				c.Writer.Header().Set("X-Request-Id", rid.(string))
			}
			globalLogger.Errorw("Panic recovered",
				"error", recovered,
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"client_ip", c.ClientIP(),
				"request_id", c.GetString("request_id"),
			)
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":       5001,
			"message":    "Internal server error",
			"request_id": c.GetString("request_id"),
		})
	})
}
