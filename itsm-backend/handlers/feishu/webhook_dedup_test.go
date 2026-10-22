package feishu

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/connector"
	"itsm-backend/ent"
	"itsm-backend/ent/connectorinbounddedup"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newDedupTestClient 建一个独立 sqlite，供「跨进程」类断言共享。
func newDedupTestClient(t *testing.T, name string) *ent.Client {
	t.Helper()
	return enttest.Open(t, "sqlite3", "file:feishu_dedup_"+name+"?mode=memory&cache=shared&_fk=1")
}

// TestWebhookDedupSurvivesProcessRestart 是本批的核心证据。
//
// 旧的去重是 Handler 私有 map：进程重启后清零，平台重投的同一事件会被当成首次
// 处理（task_event 会重复建/改飞书任务），多副本部署时更是各存一份。
// 这里用「新 Handler + 新建去重器、共享同一 DB」模拟重启：第二个实例必须仍判重复。
func TestWebhookDedupSurvivesProcessRestart(t *testing.T) {
	client := newDedupTestClient(t, "restart")
	defer client.Close()
	ctx := context.Background()
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	payload := []byte(`{"header":{"event_type":"task.task.updated"}}`)

	before := NewHandler(nil, nil, nil, zap.NewNop().Sugar())
	before.SetInboundDedup(connector.NewInboundDeduper(client, 5*time.Minute))
	require.NoError(t, before.consumeWebhookNonce(ctx, 1, timestamp, "nonce-restart", payload))

	// 重启：全新 Handler、全新去重器实例，只共享数据库
	after := NewHandler(nil, nil, nil, zap.NewNop().Sugar())
	after.SetInboundDedup(connector.NewInboundDeduper(client, 5*time.Minute))
	require.ErrorIs(t, after.consumeWebhookNonce(ctx, 1, timestamp, "nonce-restart", payload), errWebhookReplay,
		"persistent dedup must survive a process restart; the in-memory map cannot")

	// 同一 nonce 在另一个租户仍是首次（租户隔离由表中 tenant_id 列保证）
	require.NoError(t, after.consumeWebhookNonce(ctx, 2, timestamp, "nonce-restart", payload))

	// 去重行只应存在两条（租户 1、租户 2 各一）
	rows, err := client.ConnectorInboundDedup.Query().
		Where(connectorinbounddedup.ConnectorNameEQ("feishu")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

// TestWebhookDedupReleaseAllowsPlatformRetry 锁住「失败可重投」：
// 业务处理失败时 handler 会释放去重行；平台重投同一事件必须能重新接管，
// 否则一次瞬时失败就让事件永久丢失（平台看到的是重复而不是错误，不会再来第三次）。
func TestWebhookDedupReleaseAllowsPlatformRetry(t *testing.T) {
	client := newDedupTestClient(t, "release")
	defer client.Close()
	ctx := context.Background()
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	payload := []byte(`{"header":{"event_type":"task.task.updated"}}`)

	handler := NewHandler(nil, nil, nil, zap.NewNop().Sugar())
	handler.SetInboundDedup(connector.NewInboundDeduper(client, 5*time.Minute))

	require.NoError(t, handler.consumeWebhookNonce(ctx, 1, timestamp, "nonce-retry", payload))
	require.ErrorIs(t, handler.consumeWebhookNonce(ctx, 1, timestamp, "nonce-retry", payload), errWebhookReplay)

	// 模拟业务处理失败后释放
	handler.forgetWebhookEvent(ctx, 1, timestamp, "nonce-retry")

	require.NoError(t, handler.consumeWebhookNonce(ctx, 1, timestamp, "nonce-retry", payload),
		"after releasing the dedup row the platform retry must be processed again")
}

// TestWebhookDedupMarksProcessingStatus 保证运维能看到「这次到底处理成功没有」。
func TestWebhookDedupMarksProcessingStatus(t *testing.T) {
	client := newDedupTestClient(t, "status")
	defer client.Close()
	ctx := context.Background()
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	payload := []byte(`{"header":{"event_type":"task.task.updated"}}`)

	dedup := connector.NewInboundDeduper(client, 5*time.Minute)
	handler := NewHandler(nil, nil, nil, zap.NewNop().Sugar())
	handler.SetInboundDedup(dedup)

	require.NoError(t, handler.consumeWebhookNonce(ctx, 1, timestamp, "nonce-status", payload))

	status, found, err := dedup.LookupStatus(ctx, 1, "feishu", fmt.Sprintf("%s:%s", timestamp, "nonce-status"))
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "processing", status, "new events start as processing")

	handler.markWebhookProcessed(ctx, 1, timestamp, "nonce-status", "ok")

	status, found, err = dedup.LookupStatus(ctx, 1, "feishu", fmt.Sprintf("%s:%s", timestamp, "nonce-status"))
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "ok", status)
}
