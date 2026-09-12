package wecom

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"
	"testing"
	"time"

	"itsm-backend/connector"

	"github.com/stretchr/testify/require"
)

// wcomSign 计算企业微信回调签名：SHA1(sort([token, timestamp, nonce, msg_encrypt]))。
// 注意：token、timestamp、nonce、encrypt 四项按字典序排序后再拼接。
func wcomSign(token, ts, nonce, encrypt string) string {
	parts := []string{token, ts, nonce, encrypt}
	sort.Strings(parts)
	h := sha1.New()
	h.Write([]byte(strings.Join(parts, "")))
	return hex.EncodeToString(h.Sum(nil))
}

func newTestWeCom(t *testing.T, token string) *WeCom {
	t.Helper()
	w := &WeCom{}
	w.token = token
	w.client = NewClient("corp-id", "corp-secret", "agent-id", BaseURL)
	w.cfg = connector.Config{TenantID: 1, Name: "wecom"}
	w.startedAt = time.Now()
	return w
}

func TestWeComVerifySignatureAcceptsFreshTimestamp(t *testing.T) {
	w := newTestWeCom(t, "token-1")
	body := []byte(`{"Encrypt":"abc"}`)
	ts := time.Now().Unix()
	nonce := "n-1"
	sig := wcomSign("token-1", intToStr(ts), nonce, "abc")
	headers := map[string]string{
		"timestamp":     intToStr(ts),
		"nonce":         nonce,
		"msg_signature": sig,
	}
	require.NoError(t, w.VerifySignature(headers, body))
}

func TestWeComVerifySignatureRejectsStaleTimestamp(t *testing.T) {
	w := newTestWeCom(t, "token-1")
	body := []byte(`{"Encrypt":"abc"}`)
	ts := time.Now().Add(-10 * time.Minute).Unix()
	sig := wcomSign("token-1", intToStr(ts), "n-1", "abc")
	headers := map[string]string{
		"timestamp":     intToStr(ts),
		"nonce":         "n-1",
		"msg_signature": sig,
	}
	err := w.VerifySignature(headers, body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timestamp")
}

func TestWeComVerifySignatureRejectsWrongSignature(t *testing.T) {
	w := newTestWeCom(t, "token-1")
	body := []byte(`{"Encrypt":"abc"}`)
	ts := time.Now().Unix()
	headers := map[string]string{
		"timestamp":     intToStr(ts),
		"nonce":         "n-1",
		"msg_signature": "deadbeef",
	}
	err := w.VerifySignature(headers, body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "signature")
}

func TestWeComVerifySignatureRejectsMissingFields(t *testing.T) {
	w := newTestWeCom(t, "token-1")
	err := w.VerifySignature(map[string]string{}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing")
}

func TestWeComVerifySignatureRejectsMissingCallbackToken(t *testing.T) {
	w := newTestWeCom(t, "")
	ts := time.Now().Unix()
	headers := map[string]string{
		"timestamp":     intToStr(ts),
		"nonce":         "n-1",
		"msg_signature": "x",
	}
	err := w.VerifySignature(headers, []byte(`{"Encrypt":"abc"}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "callback_token")
}

func TestWeComParseInboundURLVerification(t *testing.T) {
	w := newTestWeCom(t, "token-1")
	body := []byte(`{"echostr":"echo-1"}`)
	msg, err := w.ParseInbound(body)
	require.NoError(t, err)
	require.Equal(t, "url_verification", msg.Type)
	require.Equal(t, "echo-1", msg.Content)
}

func intToStr(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		buf = append([]byte{byte('0' + v%10)}, buf...)
		v /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
