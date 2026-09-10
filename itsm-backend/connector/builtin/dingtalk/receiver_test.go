package dingtalk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"itsm-backend/connector"

	"github.com/stretchr/testify/require"
)

// sign 模拟钉钉 server 用 app_secret 算签名：base64(HMAC-SHA256(secret, ts + "\n" + body))
func sign(secret, ts string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "\n" + string(body)))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func newTestDingTalk(t *testing.T, secret string) *DingTalk {
	t.Helper()
	d := &DingTalk{}
	d.client = NewClient("appkey", secret, "agentid", BaseURL)
	require.NoError(t, d.init(secret))
	return d
}

// 避免在测试中直接调用 unexported Init
func (d *DingTalk) init(secret string) error {
	d.cfg = connector.Config{TenantID: 1, Name: "dingtalk"}
	d.startedAt = time.Now()
	return nil
}

func TestDingTalkVerifySignatureAcceptsWithinWindow(t *testing.T) {
	d := newTestDingTalk(t, "secret-1")
	body := []byte(`{"EventType":"event_callback","EventId":"e-1"}`)
	ts := time.Now().UnixMilli()
	headers := map[string]string{"Timestamp": mustSprint(ts), "Signature": sign("secret-1", mustSprint(ts), body)}
	require.NoError(t, d.VerifySignature(headers, body))
}

func TestDingTalkVerifySignatureRejectsOutOfWindow(t *testing.T) {
	d := newTestDingTalk(t, "secret-1")
	body := []byte(`{"EventType":"event_callback"}`)
	ts := time.Now().Add(-10 * time.Minute).UnixMilli()
	headers := map[string]string{"Timestamp": mustSprint(ts), "Signature": sign("secret-1", mustSprint(ts), body)}
	err := d.VerifySignature(headers, body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timestamp")
}

func TestDingTalkVerifySignatureRejectsBadSecret(t *testing.T) {
	d := newTestDingTalk(t, "secret-1")
	body := []byte(`{"EventType":"event_callback"}`)
	ts := time.Now().UnixMilli()
	headers := map[string]string{"Timestamp": mustSprint(ts), "Signature": sign("wrong-secret", mustSprint(ts), body)}
	err := d.VerifySignature(headers, body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "signature")
}

func TestDingTalkParseInboundURLVerification(t *testing.T) {
	d := newTestDingTalk(t, "secret-1")
	body := []byte(`{"EventType":"url_verification","echostr":"abc123"}`)
	msg, err := d.ParseInbound(body)
	require.NoError(t, err)
	require.Equal(t, "url_verification", msg.Type)
	require.Equal(t, "abc123", msg.Content)
}

func TestDingTalkParseInboundEventCallback(t *testing.T) {
	d := newTestDingTalk(t, "secret-1")
	body := []byte(`{"EventType":"user_add_org","EventId":"e-2","Event":{"sender":{"sender_id":"u-1"},"text":{"content":"hi"},"chatId":"c-1"}}`)
	msg, err := d.ParseInbound(body)
	require.NoError(t, err)
	require.Equal(t, "user_add_org", msg.Type)
	require.Equal(t, "u-1", msg.UserID)
	require.Equal(t, "hi", msg.Content)
	require.Equal(t, "c-1", msg.ChatID)
}

func mustSprint(v int64) string {
	b := make([]byte, 0, 20)
	neg := v < 0
	if neg {
		v = -v
	}
	if v == 0 {
		b = append(b, '0')
	}
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return strings.TrimSpace(string(b))
}