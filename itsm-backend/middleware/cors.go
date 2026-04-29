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
		reqOrigin := c.GetHeader("Origin")

		// 安全策略：优先使用白名单；开发环境默认回显请求 Origin 以支持本地多端口调试
		if allowedOriginsEnv != "" {
			// 白名单模式：只允许配置中明确列出的 origin
			for _, o := range strings.Split(allowedOriginsEnv, ",") {
				trimmed := strings.TrimSpace(o)
				if trimmed == reqOrigin && reqOrigin != "" {
					originHeader = reqOrigin
					break
				}
			}
			// 如果请求 origin 不在白名单中，不设置 Allow-Origin（安全默认）
		} else if gin.Mode() != gin.ReleaseMode {
			if reqOrigin != "" {
				originHeader = reqOrigin
			}
		} else if os.Getenv("ITSM_ALLOW_ALL_ORIGINS") == "true" {
			if reqOrigin != "" {
				originHeader = reqOrigin
			} else {
				originHeader = "*"
			}
		}
		// else: 默认不设置 Allow-Origin，拒绝跨域请求

		// 仅在有有效 origin 时设置 CORS 头
		if originHeader != "" {
			c.Header("Access-Control-Allow-Origin", originHeader)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Tenant-Code, X-Tenant-ID")
			if originHeader != "*" {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			c.Header("Access-Control-Expose-Headers", "X-Request-Id")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}
