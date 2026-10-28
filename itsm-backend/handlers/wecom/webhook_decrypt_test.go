package wecom

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"itsm-backend/connector"
	_ "itsm-backend/connector/builtin/wecom"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const handlerTestAESKey = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"

// sealEcho 独立构造企微 url_verification 密文（16 随机 + 4 大端长度 + 明文 + corpid，
// PKCS#7 块 32 填充，AES-256-CBC，IV=密钥前 16 字节）。
func sealEcho(t *testing.T, plaintext, corpID string) string {
	t.Helper()
	key, err := base64.StdEncoding.DecodeString(handlerTestAESKey + "=")
	require.NoError(t, err)
	raw := make([]byte, 0, 64)
	raw = append(raw, []byte("0123456789abcdef")...)
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(plaintext)))
	raw = append(raw, lenBuf...)
	raw = append(raw, []byte(plaintext)...)
	raw = append(raw, []byte(corpID)...)
	pad := 32 - len(raw)%32
	for i := 0; i < pad; i++ {
		raw = append(raw, byte(pad))
	}
	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	out := make([]byte, len(raw))
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(out, raw)
	return base64.StdEncoding.EncodeToString(out)
}

func signEcho(token, ts, nonce, echo string) string {
	parts := []string{token, ts, nonce, echo}
	sort.Strings(parts)
	h := sha1.New()
	h.Write([]byte(strings.Join(parts, "")))
	return hex.EncodeToString(h.Sum(nil))
}

// 关键回归（生产入口）：企微回调 URL 验证必须返回解密后的明文 echo。
// 此前 ParseInbound 把密文 echostr 当明文原样返回，企微后台保存回调配置永远校验失败。
func TestWebhook_URLVerificationReturnsDecryptedEcho(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := connector.NewManager(connector.Default(), zap.NewNop().Sugar())
	require.NoError(t, manager.Provision(context.Background(), connector.Config{
		TenantID: 1, Name: "wecom", Provider: "wecom", Enabled: true,
		Credentials: map[string]string{
			"corp_id": "corp-1", "corp_secret": "secret",
			"callback_token": "token-1", "encoding_aes_key": handlerTestAESKey,
		},
		Settings: map[string]interface{}{"callbackInstanceId": "wecom-inst-1"},
	}))
	handler := NewHandler(manager, nil, zap.NewNop().Sugar())
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))

	echoCipher := sealEcho(t, "echo-ok-123", "corp-1")
	ts := time.Now().Unix()
	nonce := "nonce-1"
	body := `{"echostr":"` + echoCipher + `"}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wecom/webhook/wecom-inst-1", strings.NewReader(body))
	// 直接以小写键写入 header，匹配 VerifySignature 读取的键名（绕过 MIME 规范化）。
	req.Header["timestamp"] = []string{intToStr(ts)}
	req.Header["nonce"] = []string{nonce}
	req.Header["msg_signature"] = []string{signEcho("token-1", intToStr(ts), nonce, echoCipher)}
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "echo-ok-123", rec.Body.String(), "必须返回解密后的明文 echo")
	assert.NotEqual(t, echoCipher, rec.Body.String(), "不得回显密文")
}

func intToStr(v int64) string { return strconv.FormatInt(v, 10) }
