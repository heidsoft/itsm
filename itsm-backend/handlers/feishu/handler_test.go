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
	payload := []byte(`{"type":"event_callback"}`)
	ctx := context.Background()

	require.NoError(t, handler.consumeWebhookNonce(ctx, 1, timestamp, "same-nonce", payload))
	require.ErrorIs(t, handler.consumeWebhookNonce(ctx, 1, timestamp, "same-nonce", payload), errWebhookReplay)
	// 租户隔离：同 nonce 在另一租户是首次出现
	require.NoError(t, handler.consumeWebhookNonce(ctx, 2, timestamp, "same-nonce", payload))
	require.ErrorIs(t, handler.consumeWebhookNonce(ctx, 0, timestamp, "nonce", payload), errWebhookReplay)
	require.ErrorIs(t, handler.consumeWebhookNonce(ctx, 1, "invalid", "nonce", payload), errWebhookReplay)
	// 超出 ±5 分钟窗口按重放处理（不是服务故障）
	stale := fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).Unix())
	require.ErrorIs(t, handler.consumeWebhookNonce(ctx, 1, stale, "fresh-nonce", payload), errWebhookReplay)
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
