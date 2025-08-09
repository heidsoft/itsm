package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware 处理跨域请求
func CORSMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 允许来源：支持通过环境变量 ITSM_CORS_ALLOWED_ORIGINS 配置，逗号分隔
		allowedOrigins := os.Getenv("ITSM_CORS_ALLOWED_ORIGINS")
		originHeader := "*"
		if allowedOrigins != "" {
			// 简单收敛：如配置了白名单，则仅当请求 Origin 命中时回显该 Origin，否则不设置
			reqOrigin := c.GetHeader("Origin")
			for _, o := range strings.Split(allowedOrigins, ",") {
				if strings.TrimSpace(o) == reqOrigin && reqOrigin != "" {
					originHeader = reqOrigin
					break
				}
			}
		}
		c.Header("Access-Control-Allow-Origin", originHeader)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Tenant-Code, X-Tenant-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		// 暴露后端附带的请求ID头，便于前端与日志对齐
		c.Header("Access-Control-Expose-Headers", "X-Request-Id")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}
