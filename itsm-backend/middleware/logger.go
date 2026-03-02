package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var globalLogger *zap.SugaredLogger

// LoggerConfig 日志中间件配置
type LoggerConfig struct {
	// SlowRequestThreshold 慢请求阈值，默认 1 秒
	SlowRequestThreshold time.Duration
	// SkipPaths 跳过日志的路径（用于健康检查等高频请求）
	SkipPaths map[string]bool
	// EnableStackTrace 是否对 5xx 错误启用堆栈跟踪
	EnableStackTrace bool
}

// SetLogger 注入全局日志器（在 main 中设置）
func SetLogger(logger *zap.SugaredLogger) {
	globalLogger = logger
}

// LoggerMiddleware 日志中间件（复用全局 logger，避免每请求创建实例）
// 默认配置：慢请求阈值 1 秒，启用 5xx 堆栈跟踪
func LoggerMiddleware() gin.HandlerFunc {
	return LoggerMiddlewareWithConfig(LoggerConfig{
		SlowRequestThreshold: 1 * time.Second,
		SkipPaths: map[string]bool{
			"/health":  true,
			"/ready":   true,
			"/metrics": true,
		},
		EnableStackTrace: true,
	})
}

// LoggerMiddlewareWithConfig 带配置的日志中间件
func LoggerMiddlewareWithConfig(cfg LoggerConfig) gin.HandlerFunc {
	if cfg.SlowRequestThreshold == 0 {
		cfg.SlowRequestThreshold = 1 * time.Second
	}
	if cfg.SkipPaths == nil {
		cfg.SkipPaths = make(map[string]bool)
	}

	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 检查是否跳过此路径
		if cfg.SkipPaths[param.Path] {
			return ""
		}

		// 结构化输出到全局 logger（若可用）
		if globalLogger != nil {
			// 基础字段
			fields := []interface{}{
				"timestamp", param.TimeStamp.Format(time.RFC3339),
				"method", param.Method,
				"path", param.Path,
				"status_code", param.StatusCode,
				"latency", param.Latency.String(),
				"client_ip", param.ClientIP,
			}

			// 添加请求 ID（如果存在）
			if rid, ok := param.Keys["request_id"]; ok && rid != nil {
				fields = append(fields, "request_id", rid)
			}

			// 添加用户代理
			if ua := param.Request.UserAgent(); ua != "" {
				fields = append(fields, "user_agent", ua)
			}

			// 添加用户 ID 和租户 ID（如果存在）
			if userID, exists := param.Keys["user_id"]; exists && userID != nil {
				fields = append(fields, "user_id", userID)
			}
			if tenantID, exists := param.Keys["tenant_id"]; exists && tenantID != nil {
				fields = append(fields, "tenant_id", tenantID)
			}

			// 添加请求体大小
			if param.Request.ContentLength > 0 {
				fields = append(fields, "body_size", param.Request.ContentLength)
			}

			// 根据状态码和延迟决定日志级别
			statusCode := param.StatusCode
			latency := param.Latency

			// 根据状态码和延迟决定日志级别
			// 5xx 错误：记录为 ERROR，可选堆栈跟踪
			if statusCode >= http.StatusInternalServerError {
				if cfg.EnableStackTrace {
					fields = append(fields, "stack_trace", true)
					globalLogger.Errorw("HTTP Server Error", fields...)
					// 额外记录详细堆栈（通过 zap 的堆栈跟踪）
					globalLogger.Desugar().Error("HTTP 5xx error",
						zap.String("path", param.Path),
						zap.String("method", param.Method),
						zap.Int("status", statusCode),
						zap.String("latency", latency.String()),
						zap.String("request_id", fmt.Sprintf("%v", param.Keys["request_id"])),
						zap.Stack("stack"),
					)
				} else {
					globalLogger.Errorw("HTTP Server Error", fields...)
				}
			} else if statusCode >= http.StatusBadRequest {
				// 4xx 错误：记录为 WARN
				fields = append(fields, "error_category", "client_error")
				globalLogger.Warnw("HTTP Client Error", fields...)
			} else if latency > cfg.SlowRequestThreshold {
				// 慢请求：记录为 WARN
				fields = append(fields,
					"slow_request", true,
					"threshold", cfg.SlowRequestThreshold.String(),
				)
				globalLogger.Warnw("HTTP Slow Request", fields...)
			} else {
				// 正常请求：记录为 INFO
				globalLogger.Infow("HTTP Request", fields...)
			}
		}

		// 同时返回一行简单文本给 gin 默认输出（跳过健康检查等）
		if cfg.SkipPaths[param.Path] {
			return ""
		}
		return fmt.Sprintf("%s - %s %s %d %s\n",
			param.TimeStamp.Format(time.RFC3339),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
		)
	})
}

// IsHealthCheckPath 检查是否为健康检查路径
func IsHealthCheckPath(path string) bool {
	healthPaths := []string{"/health", "/ready", "/live", "/metrics", "/favicon.ico"}
	for _, hp := range healthPaths {
		if strings.HasPrefix(path, hp) {
			return true
		}
	}
	return false
}
