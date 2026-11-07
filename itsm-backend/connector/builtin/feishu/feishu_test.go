package feishu

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"itsm-backend/connector"
)

// TestVerifyEventSignature 锁定官方签名算法：对 timestamp + nonce + encrypt_key + body
// 的拼接串做裸 SHA256（hex）。不是 HMAC——真实飞书回调按官方算法（larksuite/
// oapi-sdk-go event.Signature）计算，此前的 HMAC 实现会让全部真实回调验签失败。
func TestVerifyEventSignature(t *testing.T) {
	c := NewClient("", "app", "sec", "", "ek1")
	ts, nonce, body := "1700000000", "abc123", []byte(`{"hello":"world"}`)

	h := sha256.New()
	h.Write([]byte(ts + nonce + "ek1"))
	h.Write(body)
	sig := hex.EncodeToString(h.Sum(nil))
	if !c.VerifyEventSignature(ts, nonce, sig, body) {
		t.Fatal("expected official concat-SHA256 signature to be valid")
	}

	// HMAC-SHA256 形式的签名必须拒绝：同一时刻只有一套算法能通过
	hm := hmac.New(sha256.New, []byte("ek1"))
	hm.Write([]byte(ts))
	hm.Write([]byte(nonce))
	hm.Write(body)
	if c.VerifyEventSignature(ts, nonce, hex.EncodeToString(hm.Sum(nil)), body) {
		t.Fatal("HMAC signature must not verify under the official algorithm")
	}

	if c.VerifyEventSignature(ts, nonce, "deadbeef", body) {
		t.Fatal("expected wrong signature to be invalid")
	}
}

func TestVerifyURLToken(t *testing.T) {
	c := NewClient("", "app", "sec", "tok-xyz", "")
	if !c.VerifyURLToken("tok-xyz") {
		t.Fatal("expected matching token to be valid")
	}
	if c.VerifyURLToken("nope") {
		t.Fatal("expected mismatched token to be invalid")
	}
}

func TestParseInbound_URLVerification(t *testing.T) {
	f := New()
	body := []byte(`{"type":"url_verification","challenge":"chal-1","token":"tok"}`)
	if f.client == nil {
		// 未 init 的情况下用临时 client 解析
		f.client = NewClient("", "app", "sec", "tok", "")
	}
	msg, err := f.ParseInbound(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if msg.Type != "url_verification" || msg.Content != "chal-1" {
		t.Fatalf("unexpected msg: %+v", msg)
	}
}

func TestParseInbound_MessageReceive(t *testing.T) {
	f := New()
	f.client = NewClient("", "app", "sec", "", "")
	body := []byte(`{
		"uuid":"u-1",
		"token":"t",
		"type":"event_callback",
		"ts":"1700000000",
		"header":{"app_id":"a","tenant_key":"tk","event_type":"im.message.receive_v1"},
		"event":{
			"sender":{"sender_id":{"open_id":"ou_123"}},
			"message":{"chat_id":"oc_1","chat_type":"p2p","message_id":"om_1","content":{"text":"hi"}}
		}
	}`)
	msg, err := f.ParseInbound(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if msg.UserID != "ou_123" || msg.Content != "hi" || msg.ChatID != "oc_1" {
		t.Fatalf("unexpected msg: %+v", msg)
	}
}

func TestBuildFeishuMessageBody(t *testing.T) {
	m := connector.Message{Channel: "ou_x", Type: "text", Content: "hello"}
	b := buildFeishuMessageBody(&m)
	if b["msg_type"] != "text" || b["receive_id"] != "ou_x" {
		t.Fatalf("unexpected body: %+v", b)
	}
}

// TestFeishuMessageSendPath_UUIDIdempotency 回归：delivery command 的
// idempotency key（msg.ID）必须透传为飞书 create message 的 uuid 参数，
// 覆盖「外部发送成功但本地 audit 写入失败后重试」场景；空 ID 退回基础路径。
func TestFeishuMessageSendPath_UUIDIdempotency(t *testing.T) {
	if got := feishuMessageSendPath("open_id", "cmd-abc-123"); got != "/open-apis/im/v1/messages?receive_id_type=open_id&uuid=cmd-abc-123" {
		t.Fatalf("unexpected path: %s", got)
	}
	if got := feishuMessageSendPath("open_id", ""); got != "/open-apis/im/v1/messages?receive_id_type=open_id" {
		t.Fatalf("unexpected path: %s", got)
	}
	if got := feishuMessageSendPath("open_id", "  "); got != "/open-apis/im/v1/messages?receive_id_type=open_id" {
		t.Fatalf("unexpected path: %s", got)
	}
}

// TestFeishuMessageReplyPath_UUIDIdempotency reply 端点同样支持 uuid 幂等。
func TestFeishuMessageReplyPath_UUIDIdempotency(t *testing.T) {
	if got := feishuMessageReplyPath("om_123", "cmd-re-1"); got != "/open-apis/im/v1/messages/om_123/reply?uuid=cmd-re-1" {
		t.Fatalf("unexpected path: %s", got)
	}
	if got := feishuMessageReplyPath("om_123", ""); got != "/open-apis/im/v1/messages/om_123/reply" {
		t.Fatalf("unexpected path: %s", got)
	}
}
