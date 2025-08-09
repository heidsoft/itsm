package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var globalLogger *zap.SugaredLogger

// SetLogger 注入全局日志器（在 main 中设置）
func SetLogger(logger *zap.SugaredLogger) {
	globalLogger = logger
}

// LoggerMiddleware 日志中间件（复用全局 logger，避免每请求创建实例）
func LoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 结构化输出到全局 logger（若可用）
		if globalLogger != nil {
			fields := []interface{}{
				"timestamp", param.TimeStamp.Format(time.RFC3339),
				"method", param.Method,
				"path", param.Path,
				"status_code", param.StatusCode,
				"latency", param.Latency.String(),
				"client_ip", param.ClientIP,
			}
			if rid, ok := param.Keys["request_id"]; ok && rid != nil {
				fields = append(fields, "request_id", rid)
			}
			if ua := param.Request.UserAgent(); ua != "" {
				fields = append(fields, "user_agent", ua)
			}
			if userID, exists := param.Keys["user_id"]; exists && userID != nil {
				fields = append(fields, "user_id", userID)
			}
			if tenantID, exists := param.Keys["tenant_id"]; exists && tenantID != nil {
				fields = append(fields, "tenant_id", tenantID)
			}
			globalLogger.Infow("HTTP Request", fields...)
		}

		// 同时返回一行简单文本给 gin 默认输出
		return fmt.Sprintf("%s - %s %s %d %s\n",
			param.TimeStamp.Format(time.RFC3339),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
		)
	})
}
