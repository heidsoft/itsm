package middleware

import (
	"strconv"
	"time"

	"itsm-backend/metrics"

	"github.com/gin-gonic/gin"
)

// PrometheusMetricsMiddleware 记录 HTTP 请求指标的中间件
func PrometheusMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		// 记录请求总数
		metrics.HTTPRequestTotal.WithLabelValues(method, path, status).Inc()

		// 记录请求时长
		metrics.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration)
	}
}
