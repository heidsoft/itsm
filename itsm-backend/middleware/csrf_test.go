package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// parseSetCookie 提取 Set-Cookie 头并解析出 Secure 属性
func parseSetCookie(t *testing.T, w *httptest.ResponseRecorder, cookieName string) (secure bool, raw string) {
	t.Helper()
	resp := w.Result()
	cookies := resp.Cookies()
	for _, c := range cookies {
		if c.Name == cookieName {
			secure = c.Secure
			raw = resp.Header.Get("Set-Cookie")
			return
		}
	}
	return false, ""
}

// TestShouldUseSecureCSRFCookie 验证动态 Secure 判断逻辑
func TestShouldUseSecureCSRFCookie(t *testing.T) {
	t.Run("plain HTTP request -> not secure", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/csrf-token", nil)
		// 无 TLS、无 X-Forwarded-Proto
		assert.False(t, shouldUseSecureCSRFCookie(c))
	})

	t.Run("nil request -> not secure (defensive)", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = nil
		assert.False(t, shouldUseSecureCSRFCookie(c))
	})

	t.Run("X-Forwarded-Proto=https -> secure", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/csrf-token", nil)
		c.Request.Header.Set("X-Forwarded-Proto", "https")
		assert.True(t, shouldUseSecureCSRFCookie(c))
	})

	t.Run("X-Forwarded-Proto=HTTPS (case insensitive) -> secure", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/csrf-token", nil)
		c.Request.Header.Set("X-Forwarded-Proto", "HTTPS")
		assert.True(t, shouldUseSecureCSRFCookie(c))
	})

	t.Run("X-Forwarded-Proto=http -> not secure", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/csrf-token", nil)
		c.Request.Header.Set("X-Forwarded-Proto", "http")
		assert.False(t, shouldUseSecureCSRFCookie(c))
	})

	t.Run("direct TLS -> secure", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/csrf-token", nil)
		c.Request.TLS = &tls.ConnectionState{}
		assert.True(t, shouldUseSecureCSRFCookie(c))
	})
}

// TestGenerateCSRFToken_SecureFlagDynamic 验证 GenerateCSRFToken 的 Secure 标志随请求协议动态变化
// 这是修复的核心回归测试：release 模式 + HTTP 不应强制 Secure（之前 gin.Mode()==Release 会导致强制 Secure）
func TestGenerateCSRFToken_SecureFlagDynamic(t *testing.T) {
	// 在 release 模式下运行，确保不依赖 gin.Mode()
	originalMode := gin.Mode()
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(originalMode)

	config := DefaultCSRFConfig()

	t.Run("HTTP request in release mode -> no Secure", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/csrf-token", nil)
		// 无 TLS、无 X-Forwarded-Proto

		GenerateCSRFToken(c, config)

		secure, _ := parseSetCookie(t, w, config.CookieName)
		assert.False(t, secure, "HTTP 请求不应设置 Secure 标志，即使运行在 release 模式")
	})

	t.Run("HTTPS via X-Forwarded-Proto in release mode -> Secure", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/csrf-token", nil)
		c.Request.Header.Set("X-Forwarded-Proto", "https")

		GenerateCSRFToken(c, config)

		secure, _ := parseSetCookie(t, w, config.CookieName)
		assert.True(t, secure, "HTTPS 反代请求应设置 Secure 标志")
	})

	t.Run("direct TLS in release mode -> Secure", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/csrf-token", nil)
		c.Request.TLS = &tls.ConnectionState{}

		GenerateCSRFToken(c, config)

		secure, _ := parseSetCookie(t, w, config.CookieName)
		assert.True(t, secure, "直接 TLS 请求应设置 Secure 标志")
	})
}

// TestGenerateCSRFToken_HttpOnlyAndToken 验证 cookie 的 HttpOnly 和 token 值
func TestGenerateCSRFToken_HttpOnlyAndToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/csrf-token", nil)

	config := DefaultCSRFConfig()
	token := GenerateCSRFToken(c, config)

	assert.NotEmpty(t, token, "token 不应为空")

	// gin.SetCookie 会对 value 做 URL 编码（== -> %3D%3D）；
	// httptest.Result().Cookies() 解析时不会自动解码，所以比较原始 Set-Cookie 头
	rawCookie := w.Header().Get("Set-Cookie")
	assert.NotEmpty(t, rawCookie, "应设置 Set-Cookie 头")
	assert.Contains(t, rawCookie, config.CookieName+"=", "Set-Cookie 应包含 cookie 名称")
	assert.Contains(t, rawCookie, "HttpOnly", "CSRF cookie 必须 HttpOnly")
	assert.Equal(t, "/", extractCookiePath(rawCookie), "cookie Path 应为 /")

	// 验证 URL 编码后的 token 出现在 Set-Cookie 头
	encodedToken := url.QueryEscape(token)
	assert.Contains(t, rawCookie, encodedToken, "Set-Cookie 应包含 URL 编码后的 token")
}

// extractCookiePath 从 Set-Cookie 头解析 Path 属性
func extractCookiePath(setCookie string) string {
	for _, part := range strings.Split(setCookie, ";") {
		p := strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(p), "path=") {
			return strings.TrimPrefix(p, "Path=")
		}
	}
	return ""
}

// TestCSRFProtectionMiddleware_SkipPaths 验证 skip path 语义
func TestCSRFProtectionMiddleware_SkipPaths(t *testing.T) {
	config := DefaultCSRFConfig()

	t.Run("exact match skip path (login)", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/v1/auth/login", nil)

		CSRFProtectionMiddleware(config)(c)
		assert.False(t, c.IsAborted(), "login 路径应跳过 CSRF")
	})

	t.Run("prefix match skip path (sso/)", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/v1/auth/sso/callback", nil)

		CSRFProtectionMiddleware(config)(c)
		assert.False(t, c.IsAborted(), "sso/ 前缀路径应跳过 CSRF")
	})

	t.Run("prefix match skip path (sso/initiate)", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/v1/auth/sso/initiate", nil)

		CSRFProtectionMiddleware(config)(c)
		assert.False(t, c.IsAborted(), "sso/ 前缀路径应跳过 CSRF")
	})

	t.Run("non-skip write path without token -> rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/v1/tickets", nil)

		CSRFProtectionMiddleware(config)(c)
		assert.True(t, c.IsAborted(), "未 skip 的写操作无 token 应被拒绝")
	})

	t.Run("GET non-skip path -> pass through (not in AllowedMethods)", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)

		CSRFProtectionMiddleware(config)(c)
		// GET 不在 AllowedMethods (POST/PUT/DELETE/PATCH) 中，直接放行
		// token 生成由 GET /api/v1/csrf-token 端点显式处理
		assert.False(t, c.IsAborted(), "GET 请求应直接放行")
	})
}

// TestCSRFProtectionMiddleware_BearerTokenSkipped 验证 Bearer token 认证跳过 CSRF
func TestCSRFProtectionMiddleware_BearerTokenSkipped(t *testing.T) {
	config := DefaultCSRFConfig()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/tickets", nil)
	c.Request.Header.Set("Authorization", "Bearer some-jwt-token")

	CSRFProtectionMiddleware(config)(c)
	assert.False(t, c.IsAborted(), "Bearer token 请求应跳过 CSRF")
}

// TestCSRFProtectionMiddleware_TokenMismatch 验证 token 不匹配时拒绝
func TestCSRFProtectionMiddleware_TokenMismatch(t *testing.T) {
	config := DefaultCSRFConfig()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/tickets", nil)
	// 设置 cookie token 和 header token 不一致
	c.Request.AddCookie(&http.Cookie{Name: config.CookieName, Value: "cookie-token"})
	c.Request.Header.Set(config.HeaderName, "different-header-token")

	CSRFProtectionMiddleware(config)(c)
	assert.True(t, c.IsAborted(), "token 不匹配应被拒绝")
}

// TestCSRFProtectionMiddleware_TokenMatch 验证 token 匹配时通过并刷新 token
func TestCSRFProtectionMiddleware_TokenMatch(t *testing.T) {
	config := DefaultCSRFConfig()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/tickets", nil)
	// 设置匹配的 token
	c.Request.AddCookie(&http.Cookie{Name: config.CookieName, Value: "same-token"})
	c.Request.Header.Set(config.HeaderName, "same-token")

	CSRFProtectionMiddleware(config)(c)
	assert.False(t, c.IsAborted(), "token 匹配应通过")
	// 验证刷新了新 token
	cookies := w.Result().Cookies()
	var newToken string
	for _, ck := range cookies {
		if ck.Name == config.CookieName {
			newToken = ck.Value
			break
		}
	}
	assert.NotEqual(t, "same-token", newToken, "通过后应刷新新 token 防止重放")
	assert.NotEmpty(t, newToken)
}

// TestCSRFProtectionMiddleware_SiblingPathNotSkipped 验证前缀匹配不会误命中兄弟路径
// 例如 /api/v1/auth/sso/ 前缀不应匹配 /api/v1/auth/ssother
func TestCSRFProtectionMiddleware_SiblingPathNotSkipped(t *testing.T) {
	config := DefaultCSRFConfig()

	t.Run("sibling path /api/v1/auth/ssother not skipped", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/v1/auth/ssother", nil)

		CSRFProtectionMiddleware(config)(c)
		assert.True(t, c.IsAborted(), "兄弟路径不应被 sso/ 前缀误匹配")
	})
}

// TestCSRFTokenEndpoint 验证 CSRF token 端点
func TestCSRFTokenEndpoint(t *testing.T) {
	config := DefaultCSRFConfig()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api/v1/csrf-token", nil)

	CSRFTokenEndpoint(config)(c)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.True(t, strings.Contains(body, "csrf_token"), "响应应包含 csrf_token")
	assert.True(t, strings.Contains(body, `"code":0`), "响应 code 应为 0")
}
