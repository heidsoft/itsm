package feishu

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/connector"
	_ "itsm-backend/connector/builtin/feishu"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWebhookSecurityContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := connector.NewManager(connector.Default(), zap.NewNop().Sugar())
	require.NoError(t, manager.Provision(context.Background(), connector.Config{
		TenantID: 7,
		Name:     "feishu",
		Provider: "feishu",
		Enabled:  true,
		Credentials: map[string]string{
			"app_id":             "app",
			"app_secret":         "secret",
			"verification_token": "verify-token",
			"encrypt_key":        "webhook-secret",
		},
		Settings: map[string]interface{}{"callbackInstanceId": "instance-7"},
	}))

	handler := NewHandler(manager, nil, nil, zap.NewNop().Sugar())
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api, api)
	body := []byte(`{"type":"url_verification","token":"verify-token","challenge":"ok"}`)

	request := func(instanceID, timestamp, nonce string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/feishu/webhook/"+instanceID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Lark-Request-Timestamp", timestamp)
		req.Header.Set("X-Lark-Request-Nonce", nonce)
		req.Header.Set("X-Lark-Signature", webhookSignature("webhook-secret", timestamp, nonce, body))
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp
	}

	now := fmt.Sprintf("%d", time.Now().Unix())
	require.Equal(t, http.StatusOK, request("instance-7", now, "nonce-1").Code)
	require.Equal(t, http.StatusForbidden, request("instance-7", now, "nonce-1").Code, "nonce replay must fail")
	require.Equal(t, http.StatusForbidden, request("instance-7", fmt.Sprintf("%d", time.Now().Add(-6*time.Minute).Unix()), "nonce-2").Code, "stale timestamp must fail")
	require.Equal(t, http.StatusBadRequest, request("unknown-instance", now, "nonce-3").Code, "unknown instance must not resolve a tenant")
}

func TestConsumeWebhookNonceScopesReplayByTenant(t *testing.T) {
	handler := NewHandler(nil, nil, nil, zap.NewNop().Sugar())
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	require.True(t, handler.consumeWebhookNonce(1, timestamp, "same-nonce"))
	require.False(t, handler.consumeWebhookNonce(1, timestamp, "same-nonce"))
	require.True(t, handler.consumeWebhookNonce(2, timestamp, "same-nonce"))
	require.False(t, handler.consumeWebhookNonce(0, timestamp, "nonce"))
	require.False(t, handler.consumeWebhookNonce(1, "invalid", "nonce"))
}

func TestCallbackInstanceIDMustResolveExactlyOneTenant(t *testing.T) {
	manager := connector.NewManager(connector.Default(), zap.NewNop().Sugar())
	for _, tenantID := range []int{1, 2} {
		require.NoError(t, manager.Provision(context.Background(), connector.Config{
			TenantID: tenantID, Name: "feishu", Provider: "feishu", Enabled: true,
			Credentials: map[string]string{"app_id": "app", "app_secret": "secret", "encrypt_key": "key"},
			Settings:    map[string]interface{}{"callbackInstanceId": "duplicate"},
		}))
	}

	resolved, tenantID, ok := manager.GetByCallbackInstanceID("feishu", "duplicate")
	require.False(t, ok)
	require.Nil(t, resolved)
	require.Zero(t, tenantID)
}

func webhookSignature(secret, timestamp, nonce string, body []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(timestamp))
	h.Write([]byte(nonce))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}
