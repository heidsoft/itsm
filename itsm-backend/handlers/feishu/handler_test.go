package feishu

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/connector"
	_ "itsm-backend/connector/builtin/feishu"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

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
