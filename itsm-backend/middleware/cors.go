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
		allowedOriginsEnv := os.Getenv("ITSM_CORS_ALLOWED_ORIGINS")
		originHeader := ""

		// 安全策略：如果没有配置白名单，不允许任何跨域请求（除非明确设置允许所有）
		if allowedOriginsEnv != "" {
			// 白名单模式：只允许配置中明确列出的 origin
			reqOrigin := c.GetHeader("Origin")
			for _, o := range strings.Split(allowedOriginsEnv, ",") {
				trimmed := strings.TrimSpace(o)
				if trimmed == reqOrigin && reqOrigin != "" {
					originHeader = reqOrigin
					break
				}
			}
			// 如果请求 origin 不在白名单中，不设置 Allow-Origin（安全默认）
		} else if os.Getenv("ITSM_ALLOW_ALL_ORIGINS") == "true" {
			// 仅在明确设置时允许所有 origin（仅限开发环境）
			originHeader = "*"
		}
		// else: 默认不设置 Allow-Origin，拒绝跨域请求

		// 仅在有有效 origin 时设置 CORS 头
		if originHeader != "" {
			c.Header("Access-Control-Allow-Origin", originHeader)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Tenant-Code, X-Tenant-ID")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Expose-Headers", "X-Request-Id")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}
