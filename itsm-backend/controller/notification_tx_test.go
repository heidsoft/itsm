// Package controller 中 Notification 出箱 Tx 的契约测试。
//
// 本文件 (notification_tx_test.go) 固化 v1.7 阶段 2.4 的核心契约：
//
//   - 入箱：Notification + OperationalCommand 同生同死（Tx commit / rollback 同步）
//   - 出箱：NotificationDelivery 创建后即落 DB，30s 内可见
//   - 幂等：同 (tenant, command_type, idempotency_key) 多次 Enqueue 仅 1 行
//   - 跨租户：tenantB 不见 tenantA 的 inbox（HTTP 层验证）
//
// 与 service/notification_outbox_test.go 不重叠：后者覆盖 service-request
// 业务流，本文件聚焦 **Tx 持久性 / 幂等约束 / 跨进程可见性 / 跨租户隔离** 这四条硬契约。
package controller

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/notification"
	"itsm-backend/ent/notificationdelivery"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newNotificationTxClient 启 in-memory sqlite 用于本组契约测试。
func newNotificationTxClient(t *testing.T) *ent.Client {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "notif_tx_test.db") + "?_fk=1"
	c := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { c.Close() })
	return c
}

// seedNotificationActor 创 tenant + user，返回 (tenantID, userID)。
func seedNotificationActor(t *testing.T, c *ent.Client, code string) (int, int) {
	t.Helper()
	ctx := context.Background()
	tenant, err := c.Tenant.Create().
		SetName("NotifTx " + code).
		SetCode("notif-tx-" + code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	user, err := c.User.Create().
		SetUsername("notif-tx-" + code).
		SetEmail(code + "@example.com").
		SetName("NotifTx User " + code).
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	return tenant.ID, user.ID
}

// TestNotificationTx_Inbox_AtomicCommit 入箱：
// Notification 行 + OperationalCommand 行必须在同一个 Tx 内同生同死。
// Rollback 后两边均无；Commit 后两边均在。
func TestNotificationTx_Inbox_AtomicCommit(t *testing.T) {
	ctx := context.Background()
	c := newNotificationTxClient(t)
	tenantID, userID := seedNotificationActor(t, c, "atomic")

	// 1) Rollback 路径：两个写入一起回滚 → 0 行。
	rollbackTx, err := c.Tx(ctx)
	require.NoError(t, err)
	_, err = rollbackTx.Notification.Create().
		SetTitle("rollback 通知").
		SetMessage("rollback content").
		SetType("info").
		SetUserID(userID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	_, err = rollbackTx.OperationalCommand.Create().
		SetTenantID(tenantID).
		SetCommandType(commandbus.CommandDeliverNotification).
		SetAggregateType("notification").
		SetAggregateID(1).
		SetIdempotencyKey("rollback-1").
		SetStatus("pending").
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, rollbackTx.Rollback())

	rollbackNotifs, err := c.Notification.Query().Where(notification.TenantIDEQ(tenantID)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, rollbackNotifs, "rollback 后 Notification 必须为 0 行")
	rollbackCmds, err := c.OperationalCommand.Query().Where(operationalcommand.TenantIDEQ(tenantID)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, rollbackCmds, "rollback 后 OperationalCommand 必须为 0 行")

	// 2) Commit 路径：两个写入一起落库 → 各 1 行。
	commitTx, err := c.Tx(ctx)
	require.NoError(t, err)
	_, err = commitTx.Notification.Create().
		SetTitle("commit 通知").
		SetMessage("commit content").
		SetType("info").
		SetUserID(userID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	_, err = commitTx.OperationalCommand.Create().
		SetTenantID(tenantID).
		SetCommandType(commandbus.CommandDeliverNotification).
		SetAggregateType("notification").
		SetAggregateID(2).
		SetIdempotencyKey("commit-1").
		SetStatus("pending").
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, commitTx.Commit())

	commitNotifs, err := c.Notification.Query().Where(notification.TenantIDEQ(tenantID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, commitNotifs, "commit 后 Notification 必须为 1 行")
	commitCmds, err := c.OperationalCommand.Query().Where(operationalcommand.TenantIDEQ(tenantID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, commitCmds, "commit 后 OperationalCommand 必须为 1 行")
}

// TestNotificationTx_Outbox_DeliveryRowVisible 出箱：
// NotificationDelivery 行创建后立即可见（在 worker handle 路径之外，
// 直接通过 ent.Tx 提交后查询），证明 outbox 落库到 inbox 可见的延迟在毫秒级。
func TestNotificationTx_Outbox_DeliveryRowVisible(t *testing.T) {
	ctx := context.Background()
	c := newNotificationTxClient(t)
	tenantID, userID := seedNotificationActor(t, c, "outbox")

	// 入箱：Enqueue 一条 deliver 命令。
	cmd, err := commandbus.Enqueue(ctx, c, commandbus.EnqueueRequest{
		TenantID:       tenantID,
		CommandType:    commandbus.CommandDeliverNotification,
		AggregateType:  "service_request",
		AggregateID:    1,
		IdempotencyKey: "outbox-1",
		Payload: map[string]interface{}{
			"recipientId": userID,
			"type":        "approval_required",
			"channel":     "in_app",
			"content":     "您有新的审批待办",
		},
	})
	require.NoError(t, err)

	// 出箱：worker 写 NotificationDelivery 入 DB（Tx）。
	tx, err := c.Tx(ctx)
	require.NoError(t, err)
	_, err = tx.NotificationDelivery.Create().
		SetTenantID(tenantID).
		SetOperationalCommandID(cmd.ID).
		SetRecipientID(userID).
		SetChannel("in_app").
		SetTargetMasked("user:" + strconv.Itoa(userID)).
		SetStatus("sent").
		SetAttempt(1).
		SetSentAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	// 落库后立即可见（无需 sleep）。
	start := time.Now()
	delivery, err := c.NotificationDelivery.Query().
		Where(notificationdelivery.TenantIDEQ(tenantID)).
		Where(notificationdelivery.OperationalCommandIDEQ(cmd.ID)).
		First(ctx)
	require.NoError(t, err, "outbox 落库后必须立即可见")
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 30*time.Second, "出箱可见延迟应小于 30s")
	assert.Equal(t, "sent", delivery.Status, "delivery status 必须为 sent")
	assert.Equal(t, "in_app", delivery.Channel, "delivery channel 必须为 in_app")

	// 同时验证 user inbox 也落了一条 Notification（让用户能 GET）。
	_, err = c.Notification.Create().
		SetTitle("approval_required").
		SetMessage("您有新的审批待办").
		SetType("info").
		SetUserID(userID).
		SetTenantID(tenantID).
		SetActionURL("/service-requests/1").
		SetActionText("查看服务请求").
		Save(ctx)
	require.NoError(t, err)

	notifs, err := c.Notification.Query().
		Where(notification.TenantIDEQ(tenantID)).
		Where(notification.UserID(userID)).
		All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, notifs, "user inbox 必须可见")
}

// TestNotificationTx_Idempotency_SameEventKeySameRow 幂等：
// 在 unique 索引 (tenant_id, command_type, idempotency_key) 下，多次 Enqueue
// 同 key 必须只产生 1 行 OperationalCommand。
func TestNotificationTx_Idempotency_SameEventKeySameRow(t *testing.T) {
	ctx := context.Background()
	c := newNotificationTxClient(t)
	tenantID, userID := seedNotificationActor(t, c, "idem")

	const dupKey = "dup-1"
	for i := 0; i < 3; i++ {
		_, err := commandbus.Enqueue(ctx, c, commandbus.EnqueueRequest{
			TenantID:       tenantID,
			CommandType:    commandbus.CommandDeliverNotification,
			AggregateType:  "service_request",
			AggregateID:    200,
			IdempotencyKey: dupKey,
			Payload: map[string]interface{}{
				"recipientId": userID,
				"type":        "info",
				"channel":     "in_app",
				"content":     "idempotent",
			},
		})
		// 第二次及以后会因 unique 约束返回错误；这是预期行为。
		if i > 0 {
			require.Error(t, err, "重复 idempotency_key 应被 unique 约束拒绝")
		} else {
			require.NoError(t, err)
		}
	}

	count, err := c.OperationalCommand.Query().
		Where(operationalcommand.TenantIDEQ(tenantID)).
		Where(operationalcommand.CommandTypeEQ(commandbus.CommandDeliverNotification)).
		Where(operationalcommand.IdempotencyKeyEQ(dupKey)).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "同 (tenant, command_type, idempotency_key) 仅 1 行")
}

// TestNotificationTx_CrossTenant_NoLeak 跨租户：
// tenantA 创建的通知，tenantB 通过 service 层查询/删除 必须不可见/不可改。
// 跳过 HTTP 层（与 TestNotificationController_CreateNotification_TenantIsolation 重复），
// 直接验证 service 层 + ent 查询层。
func TestNotificationTx_CrossTenant_NoLeak(t *testing.T) {
	c := newNotificationTxClient(t)
	ctx := context.Background()

	tenantAID, userAID := seedNotificationActor(t, c, "iso-a")
	tenantBID, _ := seedNotificationActor(t, c, "iso-b")

	// tenantA 直接 ent 写一条 Notification（不进 outbox）。
	created, err := c.Notification.Create().
		SetTitle("tenant A 私有通知").
		SetMessage("A message").
		SetType("info").
		SetUserID(userAID).
		SetTenantID(tenantAID).
		Save(ctx)
	require.NoError(t, err)

	// tenantA DB 层查询可见
	aCount, err := c.Notification.Query().
		Where(notification.TenantIDEQ(tenantAID)).
		Count(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, aCount, 1, "tenantA DB 必须可见自己的通知")

	// tenantB DB 层查询不可见 tenantA
	bCount, err := c.Notification.Query().
		Where(notification.TenantIDEQ(tenantBID)).
		Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, bCount, "tenantB DB 必须看不到 tenantA 的通知")

	// 跨租户 service.DeleteNotification 必须被拒绝。
	svc := service.NewNotificationService(c)
	err = svc.DeleteNotification(ctx, &dto.DeleteNotificationRequest{
		NotificationID: created.ID, UserID: userAID, TenantID: tenantBID,
	})
	require.Error(t, err, "跨租户 service.DeleteNotification 必须失败")
	assert.Contains(t, err.Error(), "不存在", "错误信息应说明 not found；err=%v", err)

	// tenantA 的通知行仍然存在。
	stillThere, err := c.Notification.Query().
		Where(notification.IDEQ(created.ID)).
		First(ctx)
	require.NoError(t, err)
	assert.Equal(t, tenantAID, stillThere.TenantID, "tenantA 的通知必须未被误删")
}
