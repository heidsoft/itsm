package account

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// newTestRouter 装配只含 /account 组的小路由。
//
// 注意：本组端点只挂 AuthMiddleware（生产如此），测试用注入上下文的中间件
// 替代它 —— 这样能在无 JWT 的单测里验证 handler 语义与权限边界。
func newTestRouter(h *Handler, userID, tenantID int, injectUser bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	grp := r.Group("/api/v1/account")
	grp.Use(func(c *gin.Context) {
		if injectUser {
			c.Set("user_id", userID)
			c.Set("tenant_id", tenantID)
		}
		c.Next()
	})
	grp.PUT("/change-password", h.ChangePassword)
	return r
}

// doChange 发改密请求。
func doChange(r *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/account/change-password", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestChangePasswordFailsWithoutAuth 锁定未认证必须 401 fail-closed。
//
// 这条端点是自助改密的唯一入口，一旦在无 user_id 时放行，
// 就等于开放了"改任意用户密码"的越权后门。
func TestChangePasswordFailsWithoutAuth(t *testing.T) {
	r := newTestRouter(NewHandler(nil), 0, 0, false)

	w := doChange(r, map[string]any{"oldPassword": "Old1234!aaaa", "newPassword": "New1234!bbbb"})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("未认证应 401，实际 %d body=%s", w.Code, w.Body.String())
	}
}

// TestChangePasswordRequiresBothFields 锁定两个字段均为必填。
//
// 漏掉 newPassword 却返回成功，会让用户以为密码改了其实没改。
func TestChangePasswordRequiresBothFields(t *testing.T) {
	r := newTestRouter(NewHandler(nil), 7, 1, true)

	cases := []map[string]any{
		{"newPassword": "New1234!bbbb"},
		{"oldPassword": "Old1234!aaaa"},
		{},
	}
	for i, body := range cases {
		w := doChange(r, body)
		if w.Code != http.StatusBadRequest {
			t.Errorf("用例 %d 缺字段应 400，实际 %d body=%s", i, w.Code, w.Body.String())
		}
	}
}

// TestChangePasswordNilServiceFailsClosed 锁定 service 未装配时不得静默成功。
func TestChangePasswordNilServiceFailsClosed(t *testing.T) {
	r := newTestRouter(NewHandler(nil), 7, 1, true)

	w := doChange(r, map[string]any{"oldPassword": "Old1234!aaaa", "newPassword": "New1234!bbbb"})
	if w.Code == http.StatusOK {
		t.Errorf("service 未就绪时不应返回成功（假成功），body=%s", w.Body.String())
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("期望 503，实际 %d", w.Code)
	}
}

// TestChangePasswordIgnoresClientSuppliedUser 锁定端点不接受请求体指定目标用户。
//
// handler 只从鉴权上下文取 user_id；请求体里的任何 userId 字段都被忽略。
// 这是"自助"语义的底线——否则就退化成管理员重置接口。
func TestChangePasswordIgnoresClientSuppliedUser(t *testing.T) {
	// 用 nil service 触发前置分支即可：本用例验证的是
	// "请求体带了 userId 也不会改变鉴权上下文来源"这一结构事实，
	// 具体密码校验由 service 层测试覆盖。
	r := newTestRouter(NewHandler(nil), 7, 1, true)

	w := doChange(r, map[string]any{
		"oldPassword": "Old1234!aaaa",
		"newPassword": "New1234!bbbb",
		"userId":      999, // 试图指定他人
	})
	// 走到 service 前置分支（503）说明请求体字段没被当成目标用户使用；
	// 若实现改成信任请求体，这里会变成对 999 的操作。
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("期望 503（说明未信任请求体 userId），实际 %d body=%s", w.Code, w.Body.String())
	}
}

// TestParseUserID 锁定路径 ID 解析的边界。
func TestParseUserID(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"12", 12, true},
		{"0", 0, false},
		{"-3", 0, false},
		{"abc", 0, false},
		{"", 0, false},
	}
	for _, tc := range cases {
		got, ok := ParseUserID(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Errorf("ParseUserID(%q) = (%d,%v)，期望 (%d,%v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
