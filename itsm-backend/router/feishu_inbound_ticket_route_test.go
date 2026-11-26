package router

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/connector"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	entTicket "itsm-backend/ent/ticket"
	feishuHandler "itsm-backend/handlers/feishu"

	_ "itsm-backend/connector/builtin/feishu"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func encryptFeishuEventForTest(t *testing.T, key, plain string) []byte {
	t.Helper()
	hash := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(hash[:])
	require.NoError(t, err)
	payload := append(make([]byte, 16), make([]byte, 4)...)
	binary.BigEndian.PutUint32(payload[16:20], uint32(len(plain)))
	payload = append(payload, plain...)
	pad := aes.BlockSize - len(payload)%aes.BlockSize
	for i := 0; i < pad; i++ {
		payload = append(payload, byte(pad))
	}
	out := make([]byte, aes.BlockSize+len(payload))
	iv := out[:aes.BlockSize]
	_, err = rand.Read(iv)
	require.NoError(t, err)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out[aes.BlockSize:], payload)
	envelope, err := json.Marshal(map[string]string{"encrypt": base64.StdEncoding.EncodeToString(out)})
	require.NoError(t, err)
	return envelope
}

type feishuInboundEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Action       string `json:"action"`
		TicketID     int    `json:"ticketId"`
		TicketNumber string `json:"ticketNumber"`
	} `json:"data"`
}

func newFeishuInboundTicketRoute(t *testing.T, seed int) (*gin.Engine, *ent.Client, int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:feishu_inbound_route_%d_%d?mode=memory&cache=shared&_fk=1", seed, time.Now().UnixNano()))
	t.Cleanup(func() { _ = client.Close() })
	ctx := contextBackground()

	tenant, err := client.Tenant.Create().
		SetName(fmt.Sprintf("inbound-route-%d", seed)).
		SetCode(fmt.Sprintf("inbound-route-%d", seed)).
		SetDomain(fmt.Sprintf("inbound-route-%d.example.com", seed)).
		SetStatus("active").Save(ctx)
	require.NoError(t, err)
	u, err := client.User.Create().
		SetTenantID(tenant.ID).
		SetUsername(fmt.Sprintf("inbound-route-user-%d", seed)).
		SetEmail(fmt.Sprintf("inbound-route-%d@example.com", seed)).
		SetName("Inbound Route User").
		SetPasswordHash("hash").
		SetFeishuOpenID(fmt.Sprintf("ou-route-%d", seed)).
		SetActive(true).Save(ctx)
	require.NoError(t, err)
	_, err = client.SystemConfig.Create().
		SetTenantID(tenant.ID).
		SetKey("feishu_inbound.mode").
		SetValue("auto_create").Save(ctx)
	require.NoError(t, err)

	manager := connector.NewManager(connector.Default(), zap.NewNop().Sugar())
	instanceID := fmt.Sprintf("inbound-instance-%d", seed)
	require.NoError(t, manager.Provision(ctx, connector.Config{
		TenantID: tenant.ID, Name: "feishu", Provider: "feishu", Enabled: true,
		Credentials: map[string]string{
			"app_id": "app", "app_secret": "secret",
			"verification_token": "verify-token", "encrypt_key": feishuEncryptKey,
		},
		Settings: map[string]interface{}{"callbackInstanceId": instanceID},
	}))

	svc := feishuHandler.NewFeishuSyncService(client, zap.NewNop().Sugar())
	h := feishuHandler.NewHandler(manager, svc, nil, zap.NewNop().Sugar())
	h.SetInboundDedup(connector.NewInboundDeduper(client, 5*time.Minute))
	r := gin.New()
	authGroup := r.Group("/api/v1")
	publicGroup := r.Group("/api/v1")
	SetupFeishuRoutes(authGroup, publicGroup, h)
	return r, client, u.ID
}

func postFeishuMessageEvent(t *testing.T, r *gin.Engine, instanceID, messageID, openID, text, nonce string) *httptest.ResponseRecorder {
	t.Helper()
	plain := fmt.Sprintf(`{"uuid":%q,"token":"verify-token","type":"event_callback","ts":"%d","header":{"app_id":"cli_x","tenant_key":"t","event_type":"im.message.receive_v1"},"event":{"sender":{"sender_id":{"open_id":%q}},"message":{"message_id":%q,"chat_id":"oc_1","chat_type":"p2p","content":{"text":%q}}}}`,
		messageID, time.Now().Unix(), openID, messageID, text)
	body := encryptFeishuEventForTest(t, feishuEncryptKey, plain)
	ts := fmt.Sprintf("%d", time.Now().Unix())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feishu/webhook/"+instanceID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Lark-Request-Timestamp", ts)
	req.Header.Set("X-Lark-Request-Nonce", nonce)
	req.Header.Set("X-Lark-Signature", feishuWebhookSignature(ts, nonce, body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestFeishuInboundMessageCreatesTicketThroughRealRoute(t *testing.T) {
	r, client, requesterID := newFeishuInboundTicketRoute(t, 31)
	openID := fmt.Sprintf("ou-route-%d", 31)

	rec := postFeishuMessageEvent(t, r, "inbound-instance-31", "om-route-31", openID, "核心交换机离线", "nonce-31-1")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp feishuInboundEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "ticket_created", resp.Data.Action)
	require.Positive(t, resp.Data.TicketID)
	require.NotEmpty(t, resp.Data.TicketNumber)
	require.NotContains(t, rec.Body.String(), "app_secret")
	require.NotContains(t, rec.Body.String(), "encrypt_key")

	tickets, err := client.Ticket.Query().All(contextBackground())
	require.NoError(t, err)
	require.Len(t, tickets, 1)
	require.Equal(t, requesterID, tickets[0].RequesterID)
	require.Equal(t, "核心交换机离线", tickets[0].Title)

	// 同一 nonce 重放由既有 nonce 去重拒绝。
	replay := postFeishuMessageEvent(t, r, "inbound-instance-31", "om-route-31", openID, "核心交换机离线", "nonce-31-1")
	require.Equal(t, http.StatusForbidden, replay.Code)

	// 新 nonce + 同一 provider message_id 由消息级去重拒绝，不得产生第二张工单。
	duplicate := postFeishuMessageEvent(t, r, "inbound-instance-31", "om-route-31", openID, "核心交换机离线", "nonce-31-2")
	require.Equal(t, http.StatusConflict, duplicate.Code, duplicate.Body.String())
	tickets, err = client.Ticket.Query().Where(entTicket.TenantIDEQ(tickets[0].TenantID)).All(contextBackground())
	require.NoError(t, err)
	require.Len(t, tickets, 1)

	// 未知 instance 不得解析出租户。
	unknown := postFeishuMessageEvent(t, r, "unknown-instance", "om-route-x", openID, "未知实例", "nonce-31-3")
	require.Equal(t, http.StatusBadRequest, unknown.Code)
}

// 未映射发送者 / 未配置自动化账号时，HTTP 层必须分别返回 403/2003 与 503/5003，
// 而不是把两类失败压成 500。这里走真实路由与真实错误映射。
func TestFeishuInboundMessageErrorSemantics(t *testing.T) {
	r, client, _ := newFeishuInboundTicketRouteWithoutRequester(t, 32)

	forbidden := postFeishuMessageEvent(t, r, "inbound-instance-32", "om-403-1", "ou-not-mapped", "未映射发送者", "nonce-403-1")
	require.Equal(t, http.StatusForbidden, forbidden.Code, forbidden.Body.String())
	var forbiddenResp feishuInboundEnvelope
	require.NoError(t, json.Unmarshal(forbidden.Body.Bytes(), &forbiddenResp))
	require.Equal(t, 2003, forbiddenResp.Code)
	require.NotContains(t, forbidden.Body.String(), "encrypt_key")

	unconfigured := postFeishuMessageEvent(t, r, "inbound-instance-32", "om-503-1", "", "无申请人", "nonce-503-1")
	require.Equal(t, http.StatusServiceUnavailable, unconfigured.Code, unconfigured.Body.String())
	var unconfiguredResp feishuInboundEnvelope
	require.NoError(t, json.Unmarshal(unconfigured.Body.Bytes(), &unconfiguredResp))
	require.Equal(t, 5003, unconfiguredResp.Code)

	tickets, err := client.Ticket.Query().All(contextBackground())
	require.NoError(t, err)
	require.Empty(t, tickets, "拒绝路径不得建单")
}

func newFeishuInboundTicketRouteWithoutRequester(t *testing.T, seed int) (*gin.Engine, *ent.Client, int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:feishu_inbound_route_nouser_%d_%d?mode=memory&cache=shared&_fk=1", seed, time.Now().UnixNano()))
	t.Cleanup(func() { _ = client.Close() })
	ctx := contextBackground()

	tenant, err := client.Tenant.Create().
		SetName(fmt.Sprintf("inbound-nouser-%d", seed)).
		SetCode(fmt.Sprintf("inbound-nouser-%d", seed)).
		SetDomain(fmt.Sprintf("inbound-nouser-%d.example.com", seed)).
		SetStatus("active").Save(ctx)
	require.NoError(t, err)
	_, err = client.SystemConfig.Create().
		SetTenantID(tenant.ID).SetKey("feishu_inbound.mode").SetValue("auto_create").Save(ctx)
	require.NoError(t, err)

	manager := connector.NewManager(connector.Default(), zap.NewNop().Sugar())
	instanceID := fmt.Sprintf("inbound-instance-%d", seed)
	require.NoError(t, manager.Provision(ctx, connector.Config{
		TenantID: tenant.ID, Name: "feishu", Provider: "feishu", Enabled: true,
		Credentials: map[string]string{
			"app_id": "app", "app_secret": "secret",
			"verification_token": "verify-token", "encrypt_key": feishuEncryptKey,
		},
		Settings: map[string]interface{}{"callbackInstanceId": instanceID},
	}))

	svc := feishuHandler.NewFeishuSyncService(client, zap.NewNop().Sugar())
	h := feishuHandler.NewHandler(manager, svc, nil, zap.NewNop().Sugar())
	h.SetInboundDedup(connector.NewInboundDeduper(client, 5*time.Minute))
	r := gin.New()
	SetupFeishuRoutes(r.Group("/api/v1"), r.Group("/api/v1"), h)
	return r, client, 0
}

func contextBackground() context.Context { return context.Background() }
