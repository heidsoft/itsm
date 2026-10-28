package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/connector"
	_ "itsm-backend/connector/builtin/feishu"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestOAuthState_SingleUseAndBinding(t *testing.T) {
	h := NewHandler(nil, nil, nil, zap.NewNop().Sugar())

	state, err := h.issueOAuthState(1, "inst-1", 42)
	require.NoError(t, err)
	assert.Len(t, state, 64, "state 必须是 32 字节随机串（hex 64 字符），不可猜测")
	assert.NotContains(t, state, "tenant_id", "不得回退到可枚举的 tenant_id/user_id 拼接")

	// 绑定不符：另一实例或另一租户即便拿到 state 也拒绝。
	assert.False(t, h.consumeOAuthState(state, "inst-other", 1))
	// 上一次校验已单次消费（命中即删），正确绑定也不能再复用。
	assert.False(t, h.consumeOAuthState(state, "inst-1", 1))

	// 重新签发后正确绑定可消费一次，第二次重放拒绝。
	state2, err := h.issueOAuthState(1, "inst-1", 42)
	require.NoError(t, err)
	assert.True(t, h.consumeOAuthState(state2, "inst-1", 1))
	assert.False(t, h.consumeOAuthState(state2, "inst-1", 1), "state 必须单次消费")

	// 租户隔离：同 state 在另一租户拒绝。
	state3, err := h.issueOAuthState(1, "inst-1", 42)
	require.NoError(t, err)
	assert.False(t, h.consumeOAuthState(state3, "inst-1", 2))
}

func TestOAuthState_ExpiryAndEmpty(t *testing.T) {
	h := NewHandler(nil, nil, nil, zap.NewNop().Sugar())
	assert.False(t, h.consumeOAuthState("", "inst-1", 1), "空 state 必须拒绝")
	assert.False(t, h.consumeOAuthState("never-issued", "inst-1", 1), "未知 state 必须拒绝")

	state, err := h.issueOAuthState(1, "inst-1", 42)
	require.NoError(t, err)
	h.oauthStateMu.Lock()
	entry := h.oauthStates[state]
	entry.expiresAt = time.Now().Add(-time.Minute)
	h.oauthStates[state] = entry
	h.oauthStateMu.Unlock()
	assert.False(t, h.consumeOAuthState(state, "inst-1", 1), "过期 state 必须拒绝")
}

// newOAuthFixture  provision 一个飞书连接器并建好 handler，返回管理器与实例 ID。
func newOAuthFixture(t *testing.T) (*Handler, string) {
	t.Helper()
	manager := connector.NewManager(connector.Default(), zap.NewNop().Sugar())
	require.NoError(t, manager.Provision(context.Background(), connector.Config{
		TenantID: 1, Name: "feishu", Provider: "feishu", Enabled: true,
		Credentials: map[string]string{"app_id": "app", "app_secret": "secret", "encrypt_key": "key"},
		Settings:    map[string]interface{}{"callbackInstanceId": "inst-1"},
	}))
	return NewHandler(manager, nil, nil, zap.NewNop().Sugar()), "inst-1"
}

// authURLState 驱动真实 GetOAuthAuthURL handler，取回服务端签发的 state。
func authURLState(t *testing.T, h *Handler) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/feishu/oauth/auth-url", nil)
	c.Set("tenant_id", 1)
	c.Set("user_id", 42)
	h.GetOAuthAuthURL(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			State   string `json:"state"`
			AuthURL string `json:"authUrl"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.NotEmpty(t, envelope.Data.State)
	assert.Contains(t, envelope.Data.AuthURL, "state="+envelope.Data.State, "授权 URL 必须携带服务端 state")
	return envelope.Data.State
}

// callbackStatus 驱动真实 OAuthCallback handler（公开回调），返回 HTTP 状态与业务码。
// 这些用例都在换取 token 之前被 state 闸门拒绝，因此不触外网。
func callbackStatus(t *testing.T, h *Handler, instanceID, query string) (int, int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/feishu/oauth/callback/"+instanceID+"?"+query, nil)
	c.Params = gin.Params{{Key: "instance_id", Value: instanceID}}
	h.OAuthCallback(c)

	var envelope struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	return rec.Code, envelope.Code
}

// 关键回归（预览能力收口）：回调必须校验并单次消费 state。此前 OAuthCallback 完全
// 不读 state，且发起阶段回退到可枚举的 tenant_id/user_id 拼接——攻击者可用猜得到的
// state 把自己的授权码塞进受害者连接器（CSRF）。
func TestOAuthCallback_RejectsMissingState(t *testing.T) {
	h, instanceID := newOAuthFixture(t)
	_ = authURLState(t, h) // 发起一次，但回调不带 state
	httpStatus, code := callbackStatus(t, h, instanceID, "code=attacker-code")
	assert.Equal(t, http.StatusForbidden, httpStatus)
	assert.Equal(t, common.ForbiddenCode, code)
}

func TestOAuthCallback_RejectsForgedGuessableState(t *testing.T) {
	h, instanceID := newOAuthFixture(t)
	_ = authURLState(t, h)
	// 旧的可猜测 state 格式：修复前会被无视并继续兑换授权码。
	httpStatus, code := callbackStatus(t, h, instanceID, "code=attacker-code&state=tenant_id%3D1%26user_id%3D42")
	assert.Equal(t, http.StatusForbidden, httpStatus)
	assert.Equal(t, common.ForbiddenCode, code)
}
