package middleware

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"itsm-backend/common"

	"github.com/gin-gonic/gin"
)

// RateLimiter 请求限流器
type RateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int           // 每分钟最大请求数
	window   time.Duration // 时间窗口
}

// NewRateLimiter 创建限流器
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// 获取客户端请求历史
	requests, exists := rl.requests[clientIP]
	if !exists {
		requests = make([]time.Time, 0)
	}

	// 清理过期请求
	validRequests := make([]time.Time, 0)
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// 检查是否超过限制
	if len(validRequests) >= rl.limit {
		return false
	}

	// 添加当前请求
	validRequests = append(validRequests, now)
	rl.requests[clientIP] = validRequests

	return true
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		if !limiter.Allow(clientIP) {
			common.Fail(c, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPWhitelistMiddleware IP白名单中间件
func IPWhitelistMiddleware(allowedIPs []string) gin.HandlerFunc {
	// 预处理IP列表，支持CIDR格式
	var allowedNetworks []*net.IPNet
	var allowedSingleIPs []net.IP

	for _, ipStr := range allowedIPs {
		if strings.Contains(ipStr, "/") {
			// CIDR格式
			_, network, err := net.ParseCIDR(ipStr)
			if err == nil {
				allowedNetworks = append(allowedNetworks, network)
			}
		} else {
			// 单个IP
			ip := net.ParseIP(ipStr)
			if ip != nil {
				allowedSingleIPs = append(allowedSingleIPs, ip)
			}
		}
	}

	return func(c *gin.Context) {
		// 如果没有配置白名单，则允许所有IP
		if len(allowedIPs) == 0 {
			c.Next()
			return
		}

		clientIP := net.ParseIP(c.ClientIP())
		if clientIP == nil {
			common.Fail(c, common.ForbiddenCode, "无效的客户端IP")
			c.Abort()
			return
		}

		// 检查单个IP
		for _, allowedIP := range allowedSingleIPs {
			if clientIP.Equal(allowedIP) {
				c.Next()
				return
			}
		}

		// 检查网络段
		for _, network := range allowedNetworks {
			if network.Contains(clientIP) {
				c.Next()
				return
			}
		}

		common.Fail(c, common.ForbiddenCode, "IP地址不在白名单中")
		c.Abort()
	}
}

// SQLInjectionProtectionMiddleware SQL注入防护中间件
func SQLInjectionProtectionMiddleware() gin.HandlerFunc {
	// SQL注入检测模式
	sqlInjectionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union\s+select|select\s+.*\s+from|insert\s+into|update\s+.*\s+set|delete\s+from)`),
		regexp.MustCompile(`(?i)(drop\s+table|drop\s+database|truncate\s+table)`),
		regexp.MustCompile(`(?i)(exec\s*\(|execute\s*\(|sp_executesql)`),
		regexp.MustCompile(`(?i)(script\s*>|javascript:|vbscript:)`),
		regexp.MustCompile(`(?i)(\'\s*or\s*\'|\'\s*and\s*\'|--\s*|\/\*|\*\/)`),
		regexp.MustCompile(`(?i)(0x[0-9a-f]+|char\s*\(|ascii\s*\()`),
	}

	return func(c *gin.Context) {
		// 检查查询参数
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if containsSQLInjection(value, sqlInjectionPatterns) {
					common.Fail(c, common.ParamErrorCode, fmt.Sprintf("检测到潜在的SQL注入攻击: 参数 %s", key))
					c.Abort()
					return
				}
			}
		}

		// 检查POST表单数据
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			if err := c.Request.ParseForm(); err == nil {
				for key, values := range c.Request.PostForm {
					for _, value := range values {
						if containsSQLInjection(value, sqlInjectionPatterns) {
							common.Fail(c, common.ParamErrorCode, fmt.Sprintf("检测到潜在的SQL注入攻击: 表单字段 %s", key))
							c.Abort()
							return
						}
					}
				}
			}
		}

		c.Next()
	}
}

// XSSProtectionMiddleware XSS防护中间件
func XSSProtectionMiddleware() gin.HandlerFunc {
	// XSS检测模式
	xssPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
		regexp.MustCompile(`(?i)<object[^>]*>.*?</object>`),
		regexp.MustCompile(`(?i)<embed[^>]*>.*?</embed>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)vbscript:`),
		regexp.MustCompile(`(?i)onload\s*=|onclick\s*=|onerror\s*=|onmouseover\s*=`),
		regexp.MustCompile(`(?i)expression\s*\(|url\s*\(|@import`),
	}

	return func(c *gin.Context) {
		// 检查查询参数
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if containsXSS(value, xssPatterns) {
					common.Fail(c, common.ParamErrorCode, fmt.Sprintf("检测到潜在的XSS攻击: 参数 %s", key))
					c.Abort()
					return
				}
			}
		}

		// 设置安全响应头
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self';")

		c.Next()
	}
}

// containsSQLInjection 检查字符串是否包含SQL注入模式
func containsSQLInjection(input string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// containsXSS 检查字符串是否包含XSS模式
func containsXSS(input string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// SecurityHeadersMiddleware 安全响应头中间件
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止点击劫持
		c.Header("X-Frame-Options", "DENY")
		
		// 防止MIME类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")
		
		// XSS保护
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// 强制HTTPS
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		
		// 内容安全策略
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data: https:; font-src 'self' https:")
		
		// 引用策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// 权限策略
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// RequestSizeMiddleware 请求大小限制中间件
func RequestSizeMiddleware(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			common.Fail(c, common.ParamErrorCode, "请求体过大")
			c.Abort()
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}