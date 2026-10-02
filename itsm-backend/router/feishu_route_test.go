package router

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/connector"
	feishuHandler "itsm-backend/handlers/feishu"

	_ "itsm-backend/connector/builtin/feishu"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// feishuWebhookSignature 复刻飞书事件订阅的签名算法（timestamp + nonce + body，HMAC-SHA256）。
const feishuEncryptKey = "webhook-secret"

func feishuWebhookSignature(timestamp, nonce string, body []byte) string {
	h := hmac.New(sha256.New, []byte(feishuEncryptKey))
	h.Write([]byte(timestamp))
	h.Write([]byte(nonce))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

// newFeishuRouteFixture 用真实注册函数 SetupFeishuRoutes 建栈。
// auth 分组挂一个哨兵中间件，用来证明「哪些端点在鉴权之后」这一契约本身。
// 哨兵用 418 而不是 401：401 也是 handler 自己缺租户上下文时的 fail-closed 返回值，
// 用独立状态码才能把「分组归属」和「handler 鉴权」两件事分开断言。
const authGuardBlockedStatus = http.StatusTeapot

func newFeishuRouteFixture(t *testing.T) (*gin.Engine, *connector.Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	manager := connector.NewManager(connector.Default(), zap.NewNop().Sugar())
	require.NoError(t, manager.Provision(context.Background(), connector.Config{
		TenantID: 7,
		Name:     "feishu",
		Provider: "feishu",
		Enabled:  true,
		Credentials: map[string]string{
			"app_id":             "app",
			"app_secret":         "secret",
			"verification_token": "verify-token",
			"encrypt_key":        feishuEncryptKey,
		},
		Settings: map[string]interface{}{"callbackInstanceId": "instance-7"},
	}))

	r := gin.New()
	authGroup := r.Group("/api/v1")
	authGroup.Use(func(c *gin.Context) {
		if c.GetHeader("X-Test-Auth") != "granted" {
			c.AbortWithStatus(authGuardBlockedStatus)
			return
		}
		c.Next()
	})
	publicGroup := r.Group("/api/v1")

	SetupFeishuRoutes(authGroup, publicGroup, feishuHandler.NewHandler(manager, nil, nil, zap.NewNop().Sugar()))
	return r, manager
}

// 回归（2026-10-02 边缘功能收口 E3-3）：handlers/feishu/handler.go 曾有一份从未被调用的
// RegisterRoutes 副本，与 router.SetupFeishuRoutes 逐字重复 4 条注册，两处同时生效 gin 会因
// 重复路由直接 panic。那份副本的分组参数由调用方决定，旧测试正是以 RegisterRoutes(api, api)
// （同一个分组既当 auth 又当 public）调它的，按该形态接线会把两条外部回调挪进鉴权组。
// 本域的路由契约只能由 SetupFeishuRoutes 拥有，因此形状与分组归属都锁在这里。
func TestSetupFeishuRoutes_RegisteredShape(t *testing.T) {
	r, _ := newFeishuRouteFixture(t)

	registered := map[string]bool{}
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	require.Equal(t, map[string]bool{
		http.MethodGet + " /api/v1/feishu/oauth/auth-url":              true,
		http.MethodGet + " /api/v1/feishu/oauth/callback/:instance_id": true,
		http.MethodPost + " /api/v1/feishu/sync/ticket/:ticket_id":     true,
		http.MethodPost + " /api/v1/feishu/webhook/:instance_id":       true,
	}, registered, "飞书域只能有这 4 条路由，且完整前缀必须与前端一致")
}

// 公开回调不得经过鉴权组，管理端点必须经过——这正是被删除的那份副本会破坏的不变量。
func TestSetupFeishuRoutes_PublicCallbacksBypassAuthGroup(t *testing.T) {
	r, _ := newFeishuRouteFixture(t)

	// dispatch 返回请求是否被鉴权哨兵拦下（即该端点注册在 auth 分组上）。
	blockedByAuthGuard := func(method, path string, granted bool) bool {
		req := httptest.NewRequest(method, path, nil)
		if granted {
			req.Header.Set("X-Test-Auth", "granted")
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code == authGuardBlockedStatus
	}

	require.True(t, blockedByAuthGuard(http.MethodGet, "/api/v1/feishu/oauth/auth-url", false),
		"OAuth 授权地址必须由调用者身份驱动，不能匿名可取")
	require.False(t, blockedByAuthGuard(http.MethodGet, "/api/v1/feishu/oauth/auth-url", true),
		"已鉴权的 OAuth 授权地址应穿过哨兵进入 handler")
	require.True(t, blockedByAuthGuard(http.MethodPost, "/api/v1/feishu/sync/ticket/123", false),
		"工单同步是管理动作，必须在鉴权组之后")

	// 两个外部回调必须能在完全没有本系统鉴权的情况下抵达 handler
	// （下面用 GET/POST 打真实 method，回调拿到的应是 handler 自己的 4xx，而不是哨兵的 418）。
	require.False(t, blockedByAuthGuard(http.MethodGet, "/api/v1/feishu/oauth/callback/instance-7", false),
		"OAuth 回调由飞书跳转回来，必须留在 public 分组")
	require.False(t, blockedByAuthGuard(http.MethodPost, "/api/v1/feishu/webhook/instance-7", false),
		"事件订阅 webhook 由飞书服务端调用，必须留在 public 分组")
}

// 飞书事件订阅的四个安全分支：首次放行、nonce 重放拒绝、时间戳过期拒绝、
// 未知 instance 不得解析出租户。全部走真实注册的路由。
func TestSetupFeishuRoutes_WebhookSecurityContract(t *testing.T) {
	r, _ := newFeishuRouteFixture(t)

	body := []byte(`{"type":"url_verification","token":"verify-token","challenge":"ok"}`)
	request := func(instanceID, timestamp, nonce string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/feishu/webhook/"+instanceID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Lark-Request-Timestamp", timestamp)
		req.Header.Set("X-Lark-Request-Nonce", nonce)
		req.Header.Set("X-Lark-Signature", feishuWebhookSignature(timestamp, nonce, body))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	now := fmt.Sprintf("%d", time.Now().Unix())
	require.Equal(t, http.StatusOK, request("instance-7", now, "nonce-1").Code, "合法签名首次必须放行")
	require.Equal(t, http.StatusForbidden, request("instance-7", now, "nonce-1").Code, "nonce 重放必须拒绝")
	require.Equal(t, http.StatusForbidden,
		request("instance-7", fmt.Sprintf("%d", time.Now().Add(-6*time.Minute).Unix()), "nonce-2").Code,
		"过期时间戳必须拒绝")
	require.Equal(t, http.StatusBadRequest, request("unknown-instance", now, "nonce-3").Code,
		"未知 instance 不得解析出任何租户")
}
