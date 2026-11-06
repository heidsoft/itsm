package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/commandbus"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// recordingWSHub 只记录 SendToUser 调用。刻意不实现 SendToTenant：
// 接线必须走单播，任何「推给租户全员」的实现都过不了这个注入接口。
type recordingWSHub struct {
	mu       sync.Mutex
	delivery []recordedWSDelivery
}

type recordedWSDelivery struct {
	userID  int
	message WebSocketMessage
}

func (h *recordingWSHub) SendToUser(userID int, message WebSocketMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.delivery = append(h.delivery, recordedWSDelivery{userID: userID, message: message})
}

func (h *recordingWSHub) snapshot() []recordedWSDelivery {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]recordedWSDelivery, len(h.delivery))
	copy(out, h.delivery)
	return out
}

func wsDeliveryCommand(t *testing.T, client *ent.Client, ctx context.Context, tenantID, userID, ticketID int) *ent.OperationalCommand {
	t.Helper()
	cmd, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
		TenantID: tenantID, CommandType: commandbus.CommandDeliverNotification,
		AggregateType: "ticket", AggregateID: ticketID, IdempotencyKey: "ws-delivery-once",
		Payload: map[string]interface{}{
			"ticketId": ticketID, "recipientId": userID, "type": "assigned",
			"channel": "in_app", "content": "你有一个新工单",
		},
	})
	require.NoError(t, err)
	cmd.Attempt = 1
	return cmd
}

// N1 主回归锁：in_app 通知落库后必须按 userID 单播推送，
// 且 payload 必须与 GET /notifications 的 dto.Notification 契约一致。
func TestDeliverInAppPushesNotificationOnlyToRecipient(t *testing.T) {
	client, ctx, tenantID, userID, ticketID := notificationDeliveryFixture(t)
	otherUser, err := client.User.Create().SetUsername("notify-other").SetEmail("other@example.com").
		SetName("Other User").SetPasswordHash("hash").SetRole("agent").SetActive(true).
		SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)

	hub := &recordingWSHub{}
	cmd := wsDeliveryCommand(t, client, ctx, tenantID, userID, ticketID)
	handler := NewNotificationDeliveryCommandHandler(client, nil, zap.NewNop().Sugar(), hub)
	require.NoError(t, handler.Handle(ctx, cmd))

	got := hub.snapshot()
	require.Len(t, got, 1, "只能投递一次单播，不得退化为 SendToTenant 广播")
	require.Equal(t, userID, got[0].userID)
	require.NotEqual(t, otherUser.ID, got[0].userID)
	require.Equal(t, "notification", got[0].message.Type)

	payload, ok := got[0].message.Payload.(dto.Notification)
	require.True(t, ok, "payload 必须是前端 UserNotification 对应的 dto.Notification")
	stored, err := client.Notification.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, stored.ID, payload.ID, "推送 id 必须等于落库 id，前端据此去重")
	require.Equal(t, "你有一个新工单", payload.Message)
	require.Equal(t, userID, payload.UserID)
	require.Equal(t, tenantID, payload.TenantID)
	require.False(t, payload.Read)
	require.NotNil(t, payload.ActionURL)
	require.Contains(t, *payload.ActionURL, "/tickets/")
}

// 命令重放（worker at-least-once）不得重复推送，否则前端列表出现重复条目。
func TestDeliverInAppReplayDoesNotPushTwice(t *testing.T) {
	client, ctx, tenantID, userID, ticketID := notificationDeliveryFixture(t)
	hub := &recordingWSHub{}
	cmd := wsDeliveryCommand(t, client, ctx, tenantID, userID, ticketID)
	handler := NewNotificationDeliveryCommandHandler(client, nil, zap.NewNop().Sugar(), hub)

	require.NoError(t, handler.Handle(ctx, cmd))
	require.NoError(t, handler.Handle(ctx, cmd))
	require.Len(t, hub.snapshot(), 1)
	require.Equal(t, 1, client.Notification.Query().CountX(ctx))
}

// 未装配 hub 的夹具/独立 worker 仍须正常落库（WS 只是加速通道）。
func TestDeliverInAppWithoutHubStillPersists(t *testing.T) {
	client, ctx, tenantID, userID, ticketID := notificationDeliveryFixture(t)
	cmd := wsDeliveryCommand(t, client, ctx, tenantID, userID, ticketID)
	handler := NewNotificationDeliveryCommandHandler(client, nil, zap.NewNop().Sugar())

	require.NoError(t, handler.Handle(ctx, cmd))
	require.Equal(t, 1, client.Notification.Query().CountX(ctx))
}

// N5：满缓冲的慢客户端不能把 ReadPump 卡死在 ping 回复上。
func TestWebSocketPingDoesNotBlockOnFullSendBuffer(t *testing.T) {
	hub := NewWebSocketHub(zap.NewNop().Sugar())
	client := &WebSocketClient{
		ID: "slow-client", UserID: 7, Hub: hub,
		Send: make(chan []byte, 1),
	}
	client.Send <- []byte("occupied-by-slow-writer")

	done := make(chan struct{})
	go func() {
		client.handleMessage(WebSocketMessage{Type: "ping"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ping 在 Send 缓冲满时阻塞了 ReadPump，会导致 Hub client 泄漏")
	}
	require.Len(t, client.Send, 1, "pong 应被丢弃而不是挤占/阻塞缓冲")
}

// N6：超时扫描对同一 (task, dueDate) 反复入箱必须被 ON CONFLICT 吞掉，
// 不能让 25P02 连坐回滚同事务的业务写入。
func TestEnqueueResourceNotificationTxIsIdempotentUnderRepeatedScan(t *testing.T) {
	client, ctx, tenantID, userID, requestID := notificationOutboxFixture(t)
	req := ResourceNotificationCommand{
		TenantID: tenantID, ResourceType: "service_request", ResourceID: requestID,
		RecipientID: userID, NotificationType: "timeout_warning", Channel: "in_app",
		Content: "任务即将超时", OccurrenceKey: "task_timeout:T-1:1700000000",
	}

	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	require.NoError(t, EnqueueResourceNotificationTx(ctx, tx, req))
	require.NoError(t, EnqueueResourceNotificationTx(ctx, tx, req),
		"重复扫描必须幂等命中，不得触发唯一约束错误把事务打成 25P02")
	require.NoError(t, tx.Commit())

	count, err := client.OperationalCommand.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

// 独立夹具，避免与 notificationOutboxFixture 的共享内存 DSN 串库。
func TestEnqueueResourceNotificationTxRequiresValidRecipient(t *testing.T) {
	dsn := "file:ws-outbox-invalid?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, dialect.SQLite, dsn)
	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("T").SetCode("ws-outbox-invalid-t").
		SetDomain("invalid.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	err = EnqueueResourceNotificationTx(ctx, tx, ResourceNotificationCommand{
		TenantID: tenant.ID, ResourceType: "process_task", ResourceID: 1,
		RecipientID: 999999, NotificationType: "timeout", Channel: "in_app",
		Content: "x", OccurrenceKey: "task_timeout:missing:1",
	})
	require.Error(t, err)
	_ = tx.Rollback()
}
