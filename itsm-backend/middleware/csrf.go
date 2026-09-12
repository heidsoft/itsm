package middleware

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
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
	TokenLength    int      // Token 字节长度，默认 32
	CookieName     string   // Cookie 名称
	HeaderName     string   // Header 名称
	FormName       string   // Form 字段名称
	CookieMaxAge   int      // Cookie 最大年龄（秒），默认 86400 (24小时)
	Secure         bool     // Cookie 是否仅 HTTPS
	Domain         string   // Cookie Domain
	SkipPaths      []string // 跳过 CSRF 验证的路径
	AllowedMethods []string // 需要验证的 HTTP 方法
}

// DefaultCSRFConfig 默认 CSRF 配置
//
// 注意：Secure 字段不再基于 gin.Mode() 静态判断。CSRF cookie 的 Secure 标志
// 在 GenerateCSRFToken 内通过 shouldUseSecureCSRFCookie(c) 动态判断，与
// handlers/common.shouldUseSecureCookies 的认证 cookie 逻辑保持一致。
// 这样可避免 release 模式 + 后端走 HTTP（反代前）时 CSRF cookie 强制 Secure
// 被浏览器拒绝写入，以及 debug 模式 + HTTPS 反代时 CSRF cookie 缺失 Secure。
func DefaultCSRFConfig() *CSRFConfig {
	return &CSRFConfig{
		TokenLength:  32,
		CookieName:   CSRFTokenCookieName,
		HeaderName:   CSRFTokenHeaderName,
		FormName:     CSRFTokenFormName,
		CookieMaxAge: 86400,
		// Secure 运行时由 GenerateCSRFToken 基于 c.Request.TLS / X-Forwarded-Proto 覆盖
		Secure: false,
		// SkipPaths 语义：
		//  - 以 '/' 结尾视为前缀匹配（匹配 prefix+"/..."）；
		//  - 其余视为精确匹配。
		// 安全考虑：前缀匹配仅用于无 Double Submit Cookie 语义的端点
		//（如 SSO 回调：跨域 POST 不带浏览器 cookie）。
		// 注意："/api/v1/auth/sso/" 为前缀匹配，仅供 SSO initiate/callback 使用。
		// 未来若在 /api/v1/auth/sso/ 下新增需要 CSRF 保护的写操作端点（如解绑 SSO），
		// 必须改用精确匹配或将该端点移出 sso/ 前缀，否则会被误跳过 CSRF 验证。
		SkipPaths: []string{
			"/api/v1/auth/login",
			"/api/v1/auth/refresh",
			"/api/v1/auth/refresh-token",
			"/api/v1/auth/logout",
			"/api/v1/auth/webauthn/",
			"/api/v1/auth/sso/",
			"/api/v1/refresh-token",
			"/metrics",
			"/version",
			"/api/v1/health",
		},
		AllowedMethods: []string{"POST", "PUT", "DELETE", "PATCH"},
	}
}

// shouldUseSecureCSRFCookie 判断 CSRF cookie 是否应带 Secure 标志。
// 与 handlers/common.shouldUseSecureCookies 保持一致的动态判断逻辑：
//   - 直接 HTTPS（c.Request.TLS != nil）→ Secure
//   - 反向代理 HTTPS（X-Forwarded-Proto=https）→ Secure
//   - 其他（dev 模式、HTTP 健康检查等）→ 不带 Secure
//
// 不依赖 gin.Mode()，避免运行模式与实际传输协议不一致导致 cookie 写入失败或泄露。
func shouldUseSecureCSRFCookie(c *gin.Context) bool {
	if c.Request == nil {
		return false
	}
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

// CSRFTokenGenerator 生成 CSRF token
func CSRFTokenGenerator(length int) string {
	uid, _ := uuid.NewRandom()
	// uuid is 16 bytes, base64 encode gives ~24 chars
	return base64.URLEncoding.EncodeToString(uid[:])
}

// GenerateCSRFToken 生成 CSRF token 并设置到 cookie
//
// Secure 标志使用 shouldUseSecureCSRFCookie(c) 动态判断，与认证 cookie 逻辑一致，
// 忽略 config.Secure 字段（保留字段仅为兼容既有调用方签名）。
func GenerateCSRFToken(c *gin.Context, config *CSRFConfig) string {
	token := CSRFTokenGenerator(config.TokenLength)

	c.SetCookie(
		config.CookieName,
		token,
		config.CookieMaxAge,
		"/",
		config.Domain,
		shouldUseSecureCSRFCookie(c), // 动态判断，与认证 cookie 一致
		true,                         // HttpOnly - 前端 JS 无法读取
	)

	return token
}

// CSRFProtectionMiddleware CSRF 防护中间件
// 使用 Double Submit Cookie 模式：cookie 中的 token 与 header/form 中的 token 进行比对
func CSRFProtectionMiddleware(config *CSRFConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCSRFConfig()
	}

	// 构建 skip paths：精确匹配 map + 前缀匹配 slice。
	// SkipPaths 中以 '/' 结尾的视为前缀匹配，其余视为精确匹配。
	exactSkipPaths := make(map[string]bool)
	prefixSkipPaths := make([]string, 0)
	for _, path := range config.SkipPaths {
		if strings.HasSuffix(path, "/") {
			prefixSkipPaths = append(prefixSkipPaths, strings.TrimSuffix(path, "/"))
		} else {
			exactSkipPaths[path] = true
		}
	}

	return func(c *gin.Context) {
		// 1. 检查是否跳过 CSRF 验证（精确匹配）
		if exactSkipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}
		// 1'. 前缀匹配：以 prefix + '/' 开头，避免误命中兄弟路径
		for _, prefix := range prefixSkipPaths {
			if strings.HasPrefix(c.Request.URL.Path, prefix+"/") {
				c.Next()
				return
			}
		}

		// 1a. Bearer token 认证：无浏览器 cookie 语义（跨站不会自动附带 Authorization header），
		//     CSRF 攻击面为 0，跳过。仅 Bearer + Authorization header 存在时才生效。
		if hdr := c.GetHeader("Authorization"); hdr != "" && len(hdr) > 7 && hdr[:7] == "Bearer " {
			c.Next()
			return
		}

		// 注：此前的 "cookie access_token 存在即跳过 CSRF" 分支已移除。
		// 该分支使得 cookie-based 会话完全绕过 CSRF 校验，等价于 CSRF 保护失效。
		// SPA/前端集成方式：登录后调 GET /api/v1/csrf-token 拿 token，写入 X-CSRF-Token header。

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
			method := strings.ToUpper(c.Request.Method)
			// 安全方法（GET/HEAD/OPTIONS）可以生成新 token；写操作必须已有 token，否则拒绝（防止首次 POST 绕过）
			if method == "GET" || method == "HEAD" || method == "OPTIONS" {
				GenerateCSRFToken(c, config)
				c.Next()
				return
			}
			zap.S().Warnw("CSRF: missing csrf cookie for write operation, rejected",
				"method", method,
				"path", c.Request.URL.Path,
				"remote_ip", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 4031, "message": "CSRF token missing"})
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

// 已删除以下未接线的 dead code（全仓 grep 确认无外部调用方）：
//   - SameSiteCSRFCookieConfig()：硬编码 Secure=true，与动态判断逻辑冲突
//   - ValidateCSRFOrigin / CSRFOriginValidationMiddleware：从未在 router 注册
//
// 如未来需要 Origin 白名单校验，应作为独立中间件在 router 显式 Use，
// 并补 tenant scope + RBAC 测试，避免再次以 dead code 形式潜伏。

// CSRFTokenTTL token 过期时间
const CSRFTokenTTL = time.Hour * 24
