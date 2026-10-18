package webhook

// P1 幂等透传回归：delivery command 的 idempotency key（msg.ID）必须以
// X-Idempotency-Key 头透传给出站 HTTP 端点，接收方可据此做重试去重，
// 覆盖「外部已处理但本地 audit 写入失败后重试」场景。空 ID 不得发送该头
//（避免把空串当成有效幂等身份）。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"itsm-backend/connector"
)

func TestWebhookSend_PassesIdempotencyKeyHeader(t *testing.T) {
	var gotKey, gotEvent, gotSignature string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Idempotency-Key")
		gotEvent = r.Header.Get("X-ITSM-Event")
		gotSignature = r.Header.Get("X-ITSM-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New()
	if err := c.Init(context.Background(), connector.Config{
		Settings:    map[string]interface{}{"url": srv.URL},
		Credentials: map[string]string{"secret": "s3cret"},
	}); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := c.Send(context.Background(), &connector.Message{
		ID: "cmd-xyz-1", Channel: "hook", Type: "ticket_created", Content: "hello",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotKey != "cmd-xyz-1" {
		t.Fatalf("X-Idempotency-Key = %q, want cmd-xyz-1", gotKey)
	}
	if gotEvent != "ticket_created" {
		t.Fatalf("X-ITSM-Event = %q", gotEvent)
	}
	if !strings.HasPrefix(gotSignature, "sha256=") {
		t.Fatalf("signature missing: %q", gotSignature)
	}
}

func TestWebhookSend_EmptyID_NoIdempotencyHeader(t *testing.T) {
	sawHeader := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Idempotency-Key") != "" {
			sawHeader = true
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New()
	if err := c.Init(context.Background(), connector.Config{
		Settings: map[string]interface{}{"url": srv.URL},
	}); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := c.Send(context.Background(), &connector.Message{
		Channel: "hook", Type: "text", Content: "hi",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if sawHeader {
		t.Fatal("empty msg.ID must not emit X-Idempotency-Key")
	}
}
