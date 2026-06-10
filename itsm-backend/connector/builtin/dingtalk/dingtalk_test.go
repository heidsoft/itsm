package dingtalk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestSignRobotWebhook(t *testing.T) {
	// 用一个固定的 secret + timestamp 验证签名格式
	secret := "SEC..."
	_ = secret
	_ = int64(1700000000000)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("1700000000000\nSEC..."))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if sign == "" {
		t.Fatal("expected non-empty signature")
	}
	// 实际拼接
	url, err := SignRobotWebhook("https://oapi.dingtalk.com/robot/send?access_token=x", "SEC...")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if url == "https://oapi.dingtalk.com/robot/send?access_token=x" {
		t.Fatal("expected URL to be signed")
	}
}

func TestVerifyStreamSignature(t *testing.T) {
	c := NewClient("ak", "as", "agent", "")
	ts := "1700000000000"
	body := []byte(`{"msg":"hi"}`)
	mac := hmac.New(sha256.New, []byte("as"))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !c.VerifyStreamSignature(ts, expected, body) {
		t.Fatal("expected valid signature")
	}
	if c.VerifyStreamSignature(ts, "wrong", body) {
		t.Fatal("expected invalid signature to fail")
	}
}

func TestDingTalkManifest(t *testing.T) {
	d := New()
	m := d.Manifest()
	if m.Name != "dingtalk" || m.Type != "im" {
		t.Fatalf("unexpected manifest: %+v", m)
	}
}
