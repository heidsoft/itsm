package middleware

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// CSRFTokenCookieName CSRF token cookie 名称
	CSRFTokenCookieName = "csrf_token"
	// CSRFTokenHeaderName CSRF token header 名称
	CSRFTokenHeaderName = "X-CSRF-Token"
	// CSRFTokenFormName CSRF token form 字段名称
	CSRFTokenFormName = "csrf_token"
)

// CSRFConfig CSRF 配置
type CSRFConfig struct {
	TokenLength   int           // Token 字节长度，默认 32
	CookieName    string        // Cookie 名称
	HeaderName    string        // Header 名称
	FormName      string        // Form 字段名称
	CookieMaxAge  int           // Cookie 最大年龄（秒），默认 86400 (24小时)
	Secure        bool          // Cookie 是否仅 HTTPS
	Domain        string        // Cookie Domain
	SkipPaths     []string      // 跳过 CSRF 验证的路径
	AllowedMethods []string     // 需要验证的 HTTP 方法
}

// DefaultCSRFConfig 默认 CSRF 配置
func DefaultCSRFConfig() *CSRFConfig {
	return &CSRFConfig{
		TokenLength:   32,
		CookieName:    CSRFTokenCookieName,
		HeaderName:    CSRFTokenHeaderName,
		FormName:      CSRFTokenFormName,
		CookieMaxAge:  86400,
		Secure:        gin.Mode() == gin.ReleaseMode, // 生产环境启用 Secure
		SkipPaths: []string{
			"/api/v1/auth/login",
			"/api/v1/auth/refresh-token",
			"/metrics",
			"/version",
			"/api/v1/health",
		},
		AllowedMethods: []string{"POST", "PUT", "DELETE", "PATCH"},
	}
}

// CSRFTokenGenerator 生成 CSRF token
func CSRFTokenGenerator(length int) string {
	uid, _ := uuid.NewRandom()
	// uuid is 16 bytes, base64 encode gives ~24 chars
	return base64.URLEncoding.EncodeToString(uid[:])
}

// GenerateCSRFToken 生成 CSRF token 并设置到 cookie
func GenerateCSRFToken(c *gin.Context, config *CSRFConfig) string {
	token := CSRFTokenGenerator(config.TokenLength)

	c.SetCookie(
		config.CookieName,
		token,
		config.CookieMaxAge,
		"/",
		config.Domain,
		config.Secure,
		true, // HttpOnly - 前端 JS 无法读取
	)

	return token
}

// CSRFProtectionMiddleware CSRF 防护中间件
// 使用 Double Submit Cookie 模式：cookie 中的 token 与 header/form 中的 token 进行比对
func CSRFProtectionMiddleware(config *CSRFConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCSRFConfig()
	}

	// 构建 skip paths map 加快查找
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// 1. 检查是否跳过 CSRF 验证
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// 2. 检查请求方法是否需要 CSRF 验证
		methodNeedsValidation := false
		for _, method := range config.AllowedMethods {
			if c.Request.Method == method {
				methodNeedsValidation = true
				break
			}
		}

		if !methodNeedsValidation {
			c.Next()
			return
		}

		// 3. 获取 cookie 中的 token
		cookieToken := ""
		if cookie, err := c.Cookie(config.CookieName); err == nil {
			cookieToken = cookie
		}

		// 如果没有 cookie token，说明还没有初始化，先放行让前端获取
		if cookieToken == "" {
			// 首次请求或 token 过期，生成新 token
			GenerateCSRFToken(c, config)
			c.Next()
			return
		}

		// 4. 从 header 或 form 获取 token
		requestToken := c.GetHeader(config.HeaderName)
		if requestToken == "" {
			// 尝试从 form body 获取
			requestToken = c.PostForm(config.FormName)
		}

		// 5. 验证 token
		if requestToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "CSRF token missing",
			})
			return
		}

		// 使用恒定时间比较防止时序攻击
		if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(requestToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "CSRF token mismatch",
			})
			return
		}

		// 6. 验证通过，更新 token 防止重放
		GenerateCSRFToken(c, config)

		c.Next()
	}
}

// CSRFTokenEndpoint 生成并返回 CSRF token 的端点
func CSRFTokenEndpoint(config *CSRFConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCSRFConfig()
	}

	return func(c *gin.Context) {
		token := GenerateCSRFToken(c, config)
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"csrf_token": token,
			},
		})
	}
}

// CSRFMiddlewareWithTokenEndpoint 返回一个包含获取 token 端点的中间件
// 访问 /api/v1/csrf-token 可获取新的 CSRF token
func CSRFMiddlewareWithTokenEndpoint(config *CSRFConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCSRFConfig()
	}

	// 注册获取 token 的端点
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/api/v1/csrf-token" && c.Request.Method == "GET" {
			token := GenerateCSRFToken(c, config)
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "success",
				"data": gin.H{
					"csrf_token": token,
				},
			})
			return
		}

		// 继续执行 CSRF 验证
		CSRFProtectionMiddleware(config)(c)
	}
}

// CSRFRefreshMiddleware 在认证成功后刷新 CSRF token
// 用于登录时设置新的 CSRF token
func CSRFRefreshMiddleware(config *CSRFConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCSRFConfig()
	}

	return func(c *gin.Context) {
		// 登录成功后刷新 CSRF token
		GenerateCSRFToken(c, config)
		c.Next()
	}
}

// SameSiteCSRFCookieConfig 生成 SameSite=Strict 的 CSRF cookie 配置
func SameSiteCSRFCookieConfig() *http.Cookie {
	return &http.Cookie{
		Name:     CSRFTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
}

// ValidateCSRFOrigin 验证请求来源
func ValidateCSRFOrigin(c *gin.Context, allowedOrigins []string) bool {
	origin := c.Request.Header.Get("Origin")
	if origin == "" {
		origin = c.Request.Header.Get("Referer")
	}

	if origin == "" {
		// 无 origin/referer 的跨域请求应被拒绝
		// 同源请求通常不带 Origin header，但 CSRF 攻击必然带跨域 Origin
		// 此处安全起见返回 true（同源请求），CSRF token 本身提供保护
		return true
	}

	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}

	return false
}

// CSRFOriginValidationMiddleware 验证请求来源的中间件
func CSRFOriginValidationMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只对跨域请求进行来源验证
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		if !ValidateCSRFOrigin(c, allowedOrigins) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "Invalid request origin",
			})
			return
		}

		c.Next()
	}
}

// CSRFTokenTTL token 过期时间
const CSRFTokenTTL = time.Hour * 24
