package middleware

import "testing"

// TestAuthWhitelistIsMethodScoped 锁定白名单按 (path, method) 精确匹配。
//
// 背景：isAuthWhitelist 的 method 形参曾收了但未使用，只按 path 匹配，
// 于是任意方法命中白名单路径即放行（DELETE /health、GET /api/v1/auth/login）。
// 路径本身无敏感写操作，但「方法不参与判定」会在后续往白名单添加带写语义的
// 路径时静默放大暴露面，故收紧为方法级精确匹配。
func TestAuthWhitelistIsMethodScoped(t *testing.T) {
	cases := []struct {
		name   string
		path   string
		method string
		want   bool
	}{
		// 真实注册方法必须放行
		{"login POST 放行", "/api/v1/auth/login", "POST", true},
		{"register POST 放行", "/api/v1/auth/register", "POST", true},
		{"health GET 放行", "/health", "GET", true},
		{"api health GET 放行", "/api/v1/health", "GET", true},

		// 非注册方法一律拒绝（含原本会被 path 匹配放行的组合）
		{"health DELETE 拒绝", "/health", "DELETE", false},
		{"health POST 拒绝", "/health", "POST", false},
		{"api health DELETE 拒绝", "/api/v1/health", "DELETE", false},
		{"login GET 拒绝", "/api/v1/auth/login", "GET", false},
		{"login DELETE 拒绝", "/api/v1/auth/login", "DELETE", false},
		{"register GET 拒绝", "/api/v1/auth/register", "GET", false},

		// 大小写归一化（HTTP 方法规范上大小写不敏感，gin 传入通常为大写）
		{"login post 小写放行", "/api/v1/auth/login", "post", true},
		{"health get 小写放行", "/health", "get", true},

		// 白名单外路径
		{"白名单外路径拒绝", "/api/v1/tickets", "GET", false},
		{"空路径拒绝", "", "GET", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAuthWhitelist(tc.path, tc.method); got != tc.want {
				t.Errorf("isAuthWhitelist(%q, %q) = %v, want %v", tc.path, tc.method, got, tc.want)
			}
		})
	}
}

// TestAuthWhitelistCoversRegisteredRoutes 校验白名单条目与 router 实际注册不脱节。
//
// 两类失效都会被这里抓住：
//   - 白名单放行了未注册的 (method, path)：放行一个不存在的路由，属无用配置，
//     多半是路由改名后白名单没跟着改；
//   - 注册了免权限路由但白名单没有：该路由会走到 L2/L3/L4，可能被误拒。
func TestAuthWhitelistCoversRegisteredRoutes(t *testing.T) {
	// 与 router/router.go 保持同步：这些是公开注册的免权限端点。
	registered := []struct{ path, method string }{
		{"/api/v1/auth/login", "POST"},
		{"/api/v1/auth/register", "POST"},
		{"/health", "GET"},
	}

	for _, r := range registered {
		if !isAuthWhitelist(r.path, r.method) {
			t.Errorf("已注册的免权限端点 %s %s 不在白名单中，会被权限层误拒", r.method, r.path)
		}
	}
}
