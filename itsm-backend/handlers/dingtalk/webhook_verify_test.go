package dingtalk

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"itsm-backend/connector"
	_ "itsm-backend/connector/builtin/dingtalk"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func signBody(secret, ts string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "\n" + string(body)))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func newDingTalkRouter(t *testing.T, secret, instanceID string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	manager := connector.NewManager(connector.Default(), zap.NewNop().Sugar())
	require.NoError(t, manager.Provision(context.Background(), connector.Config{
		TenantID: 1, Name: "dingtalk", Provider: "dingtalk", Enabled: true,
		Credentials: map[string]string{
			"app_key": "appkey", "app_secret": secret, "agent_id": "agentid",
		},
		Settings: map[string]interface{}{"callbackInstanceId": instanceID},
	}))
	handler := NewHandler(manager, nil, zap.NewNop().Sugar())
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router
}

// 关键回归（生产入口）：无签名头的钉钉业务回调必须 403。
// 此前 VerifySignature 在 timestamp/signature 都缺失时 return nil（"探活豁免"），
// 任何知道有效 instance_id 的未认证 caller 都能注入事件、污染去重表。
func TestWebhook_UnsignedRequestRejected(t *testing.T) {
	router := newDingTalkRouter(t, "secret-1", "dt-inst-1")
	body := `{"EventType":"user_add_org","EventId":"e-1","Event":{"sender":{"sender_id":"u-1"},"text":{"content":"hi"}}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dingtalk/webhook/dt-inst-1", strings.NewReader(body))
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code, "无签名回调必须 fail-closed")
}

// 无签名的 url_verification 探活同样必须 403，且不得回显 echostr——
// 否则未认证 caller 可借探活区分有效/无效 instance_id 并取回任意 echo。
func TestWebhook_UnsignedURLVerificationRejected(t *testing.T) {
	router := newDingTalkRouter(t, "secret-1", "dt-inst-1")
	body := `{"EventType":"url_verification","echostr":"probe-123"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dingtalk/webhook/dt-inst-1", strings.NewReader(body))
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	assert.NotContains(t, rec.Body.String(), "probe-123", "不得回显未认证探活的 echostr")
}

// 正向：带合法签名的业务事件仍正常接收，修复不得破坏已签名流程。
func TestWebhook_SignedEventAccepted(t *testing.T) {
	router := newDingTalkRouter(t, "secret-1", "dt-inst-1")
	body := `{"EventType":"user_add_org","EventId":"e-2","Event":{"sender":{"sender_id":"u-1"},"text":{"content":"hi"}}}`
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dingtalk/webhook/dt-inst-1", strings.NewReader(body))
	req.Header.Set("Timestamp", ts)
	req.Header.Set("Signature", signBody("secret-1", ts, []byte(body)))
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
