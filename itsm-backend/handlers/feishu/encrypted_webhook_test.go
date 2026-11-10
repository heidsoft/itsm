package feishu

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
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
	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// encryptForTest 按官方算法造密文（与 connector 层 crypto_test 同构，
// 此处是另一包不能复用私有 helper）。
func encryptForTest(t *testing.T, encryptKey, msg string) string {
	t.Helper()
	key := sha256.Sum256([]byte(encryptKey))
	block, err := aes.NewCipher(key[:])
	require.NoError(t, err)
	plain := append(make([]byte, 16), make([]byte, 4)...)
	binary.BigEndian.PutUint32(plain[16:20], uint32(len(msg)))
	plain = append(plain, msg...)
	pad := aes.BlockSize - len(plain)%aes.BlockSize
	for i := 0; i < pad; i++ {
		plain = append(plain, byte(pad))
	}
	out := make([]byte, aes.BlockSize+len(plain))
	iv := out[:aes.BlockSize]
	_, err = rand.Read(iv)
	require.NoError(t, err)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out[aes.BlockSize:], plain)
	return base64.StdEncoding.EncodeToString(out)
}

func signOfficial(ts, nonce, encryptKey string, body []byte) string {
	h := sha256.New()
	h.Write([]byte(ts + nonce + encryptKey))
	h.Write(body)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func newWebhookTestRouter(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	manager := connector.NewManager(connector.Default(), zap.NewNop().Sugar())
	require.NoError(t, manager.Provision(t.Context(), connector.Config{
		TenantID: 1, Name: "feishu", Provider: "feishu", Enabled: true,
		Credentials: map[string]string{
			"app_id": "app", "app_secret": "secret",
			"verification_token": "v-token", "encrypt_key": "ek-e2e",
		},
		Settings: map[string]interface{}{"callbackInstanceId": "cb-e2e"},
	}))
	client := enttest.Open(t, "sqlite3", "file:feishu_enc_e2e?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	h := NewHandler(manager, nil, nil, zap.NewNop().Sugar())
	h.SetInboundDedup(connector.NewInboundDeduper(client, 5*time.Minute))
	r := gin.New()
	r.POST("/webhook/:instance_id", h.Webhook)
	return r, "cb-e2e"
}

// TestWebhookEncryptedChallengeEndToEnd 是生产入口的加密回调闭环：
// 官方算法签名 + Encrypt 信封 → 真实 Webhook handler → 明文 challenge 回包。
func TestWebhookEncryptedChallengeEndToEnd(t *testing.T) {
	r, instanceID := newWebhookTestRouter(t)
	plain := `{"type":"url_verification","challenge":"chal-e2e","token":"v-token"}`
	raw, _ := json.Marshal(map[string]string{"encrypt": encryptForTest(t, "ek-e2e", plain)})
	ts := fmt.Sprintf("%d", time.Now().Unix())

	req := httptest.NewRequest(http.MethodPost, "/webhook/"+instanceID, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Lark-Request-Timestamp", ts)
	req.Header.Set("X-Lark-Request-Nonce", "nonce-e2e")
	req.Header.Set("X-Lark-Signature", signOfficial(ts, "nonce-e2e", "ek-e2e", raw))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	// challenge 回包按官方 SDK 语义保持明文
	require.JSONEq(t, `{"challenge":"chal-e2e"}`, w.Body.String())
}

// TestWebhookRejectsHMACSignature 锁定官方签名算法：HMAC 形式的签名在
// 生产入口必须被拒绝（此前 HMAC 实现与真实飞书算法不兼容）。
func TestWebhookRejectsHMACSignature(t *testing.T) {
	r, instanceID := newWebhookTestRouter(t)
	raw := []byte(`{"type":"url_verification","challenge":"c","token":"v-token"}`)
	ts := fmt.Sprintf("%d", time.Now().Unix())
	hm := hmac.New(sha256.New, []byte("ek-e2e"))
	hm.Write([]byte(ts + "nonce-hmac"))
	hm.Write(raw)

	req := httptest.NewRequest(http.MethodPost, "/webhook/"+instanceID, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Lark-Request-Timestamp", ts)
	req.Header.Set("X-Lark-Request-Nonce", "nonce-hmac")
	req.Header.Set("X-Lark-Signature", fmt.Sprintf("%x", hm.Sum(nil)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code, "HMAC-signed request must be rejected: %s", w.Body.String())
}
