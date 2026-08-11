package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"itsm-backend/connector"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/notificationdelivery"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/ent/ticket"
	"itsm-backend/internal/commandbus"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type notificationTestConnector struct {
	sent atomic.Int32
	fail bool
}

func (*notificationTestConnector) Manifest() connector.Manifest {
	return connector.Manifest{Name: "feishu", Version: "1.0.0", Title: "Test Feishu", Type: connector.TypeIM,
		Capabilities: []connector.Capability{connector.CapSendMessage}, RequiredPermissions: []string{"connector:send"}}
}
func (*notificationTestConnector) Init(context.Context, connector.Config) error { return nil }
func (c *notificationTestConnector) Send(context.Context, *connector.Message) error {
	c.sent.Add(1)
	if c.fail {
		return errors.New("provider-secret-shaped-error-must-not-persist")
	}
	return nil
}
func (*notificationTestConnector) HealthCheck(context.Context) connector.HealthStatus {
	return connector.HealthStatus{OK: true}
}
func (*notificationTestConnector) Close() error { return nil }

func notificationDeliveryFixture(t *testing.T) (*ent.Client, context.Context, int, int, int) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	client := enttest.Open(t, dialect.SQLite, dsn)
	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("Notify Tenant").SetCode("notify-tenant").SetDomain("notify.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().SetUsername("notify-user").SetEmail("notify@example.com").SetName("Notify User").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	ticket, err := client.Ticket.Create().SetTitle("通知工单").SetTicketNumber("NOTIFY-1").SetRequesterID(user.ID).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	return client, ctx, tenant.ID, user.ID, ticket.ID
}

func TestTicketNotificationServiceEnqueuesDurableDelivery(t *testing.T) {
	client, ctx, tenantID, userID, ticketID := notificationDeliveryFixture(t)
	svc := NewTicketNotificationService(client, zap.NewNop().Sugar())
	svc.EnableOutbox()
	err := svc.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: []int{userID, userID}, Type: "assigned", Channel: "in_app", Content: "你有一个新工单", IdempotencyKey: "assignment-event-1",
	}, tenantID)
	require.NoError(t, err)
	err = svc.SendNotification(ctx, ticketID, &dto.SendTicketNotificationRequest{
		UserIDs: []int{userID}, Type: "assigned", Channel: "in_app", Content: "你有一个新工单", IdempotencyKey: "assignment-event-1",
	}, tenantID)
	require.NoError(t, err)
	commands, err := client.OperationalCommand.Query().Where(operationalcommand.TenantIDEQ(tenantID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Equal(t, commandbus.CommandDeliverNotification, commands[0].CommandType)
}

func TestNotificationDeliveryHandlerCreatesInAppRecordsIdempotently(t *testing.T) {
	client, ctx, tenantID, userID, ticketID := notificationDeliveryFixture(t)
	cmd, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
		TenantID: tenantID, CommandType: commandbus.CommandDeliverNotification,
		AggregateType: "ticket", AggregateID: ticketID, IdempotencyKey: "notify-once",
		Payload: map[string]interface{}{"ticketId": ticketID, "recipientId": userID, "type": "assigned", "channel": "in_app", "content": "你有一个新工单"},
	})
	require.NoError(t, err)
	cmd.Attempt = 1
	handler := NewNotificationDeliveryCommandHandler(client, nil, zap.NewNop().Sugar())
	require.NoError(t, handler.Handle(ctx, cmd))
	require.NoError(t, handler.Handle(ctx, cmd))

	ticketNotifications, err := client.TicketNotification.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, ticketNotifications, 1)
	notifications, err := client.Notification.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	deliveries, err := client.NotificationDelivery.Query().Where(notificationdelivery.OperationalCommandIDEQ(cmd.ID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, deliveries, 1)
	require.Equal(t, "sent", deliveries[0].Status)
}

func TestNotificationDeliveryHandlerRejectsCrossTenantRecipient(t *testing.T) {
	client, ctx, tenantID, _, ticketID := notificationDeliveryFixture(t)
	otherTenant, err := client.Tenant.Create().SetName("Other").SetCode("other-notify").SetDomain("other-notify.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	foreign, err := client.User.Create().SetUsername("foreign-notify").SetEmail("foreign@example.com").SetName("Foreign").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(otherTenant.ID).Save(ctx)
	require.NoError(t, err)
	cmd, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
		TenantID: tenantID, CommandType: commandbus.CommandDeliverNotification,
		AggregateType: "ticket", AggregateID: ticketID, IdempotencyKey: "cross-tenant",
		Payload: map[string]interface{}{"ticketId": ticketID, "recipientId": foreign.ID, "type": "assigned", "channel": "in_app", "content": "forbidden"},
	})
	require.NoError(t, err)
	err = NewNotificationDeliveryCommandHandler(client, nil, zap.NewNop().Sugar()).Handle(ctx, cmd)
	require.Error(t, err)
	require.Zero(t, client.Notification.Query().CountX(ctx))
}

func TestNotificationDeliveryHandlerSendsConnectorAndAuditsMaskedTarget(t *testing.T) {
	client, ctx, tenantID, userID, ticketID := notificationDeliveryFixture(t)
	_, err := client.User.UpdateOneID(userID).SetFeishuOpenID("ou_sensitive_target_1234").Save(ctx)
	require.NoError(t, err)
	fake := &notificationTestConnector{}
	registry := connector.NewRegistry()
	registry.Register(func() connector.Connector { return fake })
	manager := connector.NewManager(registry, zap.NewNop().Sugar())
	require.NoError(t, manager.Provision(ctx, connector.Config{TenantID: tenantID, Name: "feishu", Provider: "test", Enabled: true}))
	cmd, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
		TenantID: tenantID, CommandType: commandbus.CommandDeliverNotification,
		AggregateType: "ticket", AggregateID: ticketID, IdempotencyKey: "feishu-once",
		Payload: map[string]interface{}{"ticketId": ticketID, "recipientId": userID, "type": "assigned", "channel": "feishu", "content": "你有一个新工单"},
	})
	require.NoError(t, err)
	cmd.Attempt = 1
	handler := NewNotificationDeliveryCommandHandler(client, manager, zap.NewNop().Sugar())
	require.NoError(t, handler.Handle(ctx, cmd))
	require.NoError(t, handler.Handle(ctx, cmd))
	require.Equal(t, int32(1), fake.sent.Load())
	delivery, err := client.NotificationDelivery.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "sent", delivery.Status)
	require.NotContains(t, delivery.TargetMasked, "sensitive")
	require.NotEqual(t, "ou_sensitive_target_1234", delivery.TargetMasked)
}

func TestNotificationDeliveryFailureRetriesThenDeadLettersWithSafeAudit(t *testing.T) {
	client, ctx, tenantID, userID, ticketID := notificationDeliveryFixture(t)
	_, err := client.User.UpdateOneID(userID).SetFeishuOpenID("ou_failure_target").Save(ctx)
	require.NoError(t, err)
	fake := &notificationTestConnector{fail: true}
	registry := connector.NewRegistry()
	registry.Register(func() connector.Connector { return fake })
	manager := connector.NewManager(registry, zap.NewNop().Sugar())
	require.NoError(t, manager.Provision(ctx, connector.Config{TenantID: tenantID, Name: "feishu", Provider: "test", Enabled: true}))
	cmd, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
		TenantID: tenantID, CommandType: commandbus.CommandDeliverNotification, AggregateType: "ticket", AggregateID: ticketID,
		IdempotencyKey: "feishu-dead-letter", MaxAttempts: 2,
		Payload: map[string]interface{}{"ticketId": ticketID, "recipientId": userID, "type": "assigned", "channel": "feishu", "content": "失败重试"},
	})
	require.NoError(t, err)
	commandRegistry := commandbus.NewRegistry()
	handler := NewNotificationDeliveryCommandHandler(client, manager, zap.NewNop().Sugar())
	require.NoError(t, commandRegistry.Register(commandbus.CommandDeliverNotification, handler.Handle))
	worker := commandbus.NewWorker(client, commandRegistry, zap.NewNop().Sugar(), "notification-worker")
	processed, err := worker.RunOnce(ctx)
	require.NoError(t, err)
	require.True(t, processed)
	_, err = client.OperationalCommand.UpdateOneID(cmd.ID).SetAvailableAt(time.Now().Add(-time.Second)).Save(ctx)
	require.NoError(t, err)
	processed, err = worker.RunOnce(ctx)
	require.NoError(t, err)
	require.True(t, processed)
	stored, err := client.OperationalCommand.Get(ctx, cmd.ID)
	require.NoError(t, err)
	require.Equal(t, commandbus.StatusDeadLetter, stored.Status)
	require.NotContains(t, stored.LastError, "provider-secret")
	delivery, err := client.NotificationDelivery.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "failed", delivery.Status)
	require.Equal(t, 2, delivery.Attempt)
	require.NotContains(t, delivery.ErrorMessage, "provider-secret")
}

// TestNotifyTicketCreatedTxRollsBackWithTicket 验证事务性入箱与主表变更「同生同死」：
// 在 tx 内调用 NotifyTicketCreatedTx 后 rollback，operational_command 必须为零。
// 这是阶段 B（工单创建）下沉的核心契约：业务事务失败时不得有「孤儿」通知入箱。
func TestNotifyTicketCreatedTxRollsBackWithTicket(t *testing.T) {
	client, ctx, tenantID, userID, ticketID := notificationDeliveryFixture(t)
	svc := NewTicketNotificationService(client, zap.NewNop().Sugar())
	svc.EnableTxOutbox()

	tx, err := client.Tx(ctx)
	require.NoError(t, err)

	createdTicket, err := tx.Ticket.Create().
		SetTitle("rollback-ticket").
		SetTicketNumber("NOTIFY-RB").
		SetRequesterID(userID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	err = svc.NotifyTicketCreatedTx(ctx, tx, createdTicket)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	commands, err := client.OperationalCommand.Query().
		Where(operationalcommand.TenantIDEQ(tenantID)).
		All(ctx)
	require.NoError(t, err)
	require.Empty(t, commands, "rollback 之后 operational_command 必须随业务事务一起消失")
	_, err = client.Ticket.Get(ctx, ticketID)
	require.NoError(t, err)
	_, err = client.Ticket.Query().Where(ticket.TicketNumberEQ("NOTIFY-RB")).Only(ctx)
	require.Error(t, err, "rollback 后刚创建的 ticket 也必须消失")
}

// TestNotifyTicketCreatedTxCommitsWithTicket 验证事务提交后通知与工单都可见，且收件人覆盖 assignee/requester。
func TestNotifyTicketCreatedTxCommitsWithTicket(t *testing.T) {
	client, ctx, tenantID, userID, _ := notificationDeliveryFixture(t)
	assignee, err := client.User.Create().SetUsername("notify-assignee").SetEmail("assignee@example.com").SetName("Assignee").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)

	svc := NewTicketNotificationService(client, zap.NewNop().Sugar())
	svc.EnableTxOutbox()

	tx, err := client.Tx(ctx)
	require.NoError(t, err)

	createdTicket, err := tx.Ticket.Create().
		SetTitle("commit-ticket").
		SetTicketNumber("NOTIFY-OK").
		SetRequesterID(userID).
		SetAssigneeID(assignee.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, svc.NotifyTicketCreatedTx(ctx, tx, createdTicket))
	require.NoError(t, tx.Commit())

	commands, err := client.OperationalCommand.Query().
		Where(operationalcommand.TenantIDEQ(tenantID), operationalcommand.AggregateIDEQ(createdTicket.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, commands, 2, "assignee + requester 两个收件人应各产生一条入箱行")

	recipients := map[int]struct{}{}
	for _, cmd := range commands {
		require.Equal(t, commandbus.CommandDeliverNotification, cmd.CommandType)
		require.Equal(t, "ticket", cmd.AggregateType)
		require.NotNil(t, cmd.Payload)
		recipients[asInt(cmd.Payload["recipientId"])] = struct{}{}
		require.Equal(t, "created", cmd.Payload["type"])
		require.Equal(t, "in_app", cmd.Payload["channel"])
	}
	require.Contains(t, recipients, userID)
	require.Contains(t, recipients, assignee.ID)
	// 域下沉的语义保证：recipientId 必须是 user.id，而不是 ticket.id 或其它主键。
	// fixture 中 ticketID=1 与 userID=1 恰好相同是巧合，所以这里改用「恰好 2 条收件人」
	// 且 Payload 中 ticketId 字段等于 createdTicket.ID 的硬约束来证明这一点。
	seenTicketID := 0
	for _, cmd := range commands {
		seenTicketID++
		require.Equal(t, createdTicket.ID, asInt(cmd.Payload["ticketId"]))
	}
	require.Equal(t, 2, seenTicketID)
}

func TestNotificationDeliveryHandlerDeliversChangeInApp(t *testing.T) {
	client, ctx, tenantID, userID, _ := notificationDeliveryFixture(t)
	creator, err := client.User.Get(ctx, userID)
	require.NoError(t, err)
	changeEntity, err := client.Change.Create().SetTitle("approval delivery").SetDescription("change notification").
		SetCreatedBy(creator.ID).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	cmd, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
		TenantID: tenantID, CommandType: commandbus.CommandDeliverNotification,
		AggregateType: "change", AggregateID: changeEntity.ID, IdempotencyKey: "change-delivery-test",
		Payload: map[string]interface{}{
			"resourceType": "change", "resourceId": changeEntity.ID, "recipientId": userID,
			"type": "change_approval_required", "channel": "in_app", "content": "请审批变更",
		},
	})
	require.NoError(t, err)

	handler := NewNotificationDeliveryCommandHandler(client, nil, zap.NewNop().Sugar())
	require.NoError(t, handler.Handle(ctx, cmd))

	deliveries, err := client.NotificationDelivery.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, deliveries, 1)
	require.Nil(t, deliveries[0].TicketID)
	require.Nil(t, deliveries[0].TicketNotificationID)
	notifications, err := client.Notification.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	require.Equal(t, fmt.Sprintf("/changes/%d", changeEntity.ID), notifications[0].ActionURL)
}

// asInt 兼容 SQLite JSON 反序列化后数字统一为 float64 的现实，阶段 A 测试专用。
func asInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return -1
	}
}

// TestNotifySLABreachedTxRollsBack 验证 SLA 违规通知同样随事务回滚消失。
func TestNotifySLABreachedTxRollsBack(t *testing.T) {
	client, ctx, tenantID, _, ticketID := notificationDeliveryFixture(t)
	svc := NewTicketNotificationService(client, zap.NewNop().Sugar())
	svc.EnableTxOutbox()

	tx, err := client.Tx(ctx)
	require.NoError(t, err)

	err = svc.NotifySLABreachedTx(ctx, tx, ticketID, "response_time", 30, tenantID)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	commands, err := client.OperationalCommand.Query().
		Where(operationalcommand.TenantIDEQ(tenantID), operationalcommand.AggregateIDEQ(ticketID)).
		All(ctx)
	require.NoError(t, err)
	require.Empty(t, commands, "SLA breach 通知随 tx rollback 必须消失")
}

// TestNotifySLABreachedTxIdempotent 验证 SLA breach 入箱对同一参数幂等：
// 重复 enqueue 同一 occurrenceKey 只产生一行 operational_command。
func TestNotifySLABreachedTxIdempotent(t *testing.T) {
	client, ctx, tenantID, _, ticketID := notificationDeliveryFixture(t)
	svc := NewTicketNotificationService(client, zap.NewNop().Sugar())
	svc.EnableTxOutbox()

	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	require.NoError(t, svc.NotifySLABreachedTx(ctx, tx, ticketID, "response_time", 30, tenantID))
	require.NoError(t, svc.NotifySLABreachedTx(ctx, tx, ticketID, "response_time", 30, tenantID))
	require.NoError(t, tx.Commit())

	commands, err := client.OperationalCommand.Query().
		Where(operationalcommand.TenantIDEQ(tenantID), operationalcommand.AggregateIDEQ(ticketID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, commands, 1, "同一 occurrenceKey 重复 enqueue 必须被唯一索引合并为一行")
}

// TestNotifySLABreachedTxRejectsCrossTenant 验证调用方传入与 ticket 不一致的 tenantID 直接 fail-closed。
func TestNotifySLABreachedTxRejectsCrossTenant(t *testing.T) {
	client, ctx, tenantID, _, ticketID := notificationDeliveryFixture(t)
	svc := NewTicketNotificationService(client, zap.NewNop().Sugar())
	svc.EnableTxOutbox()

	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	err = svc.NotifySLABreachedTx(ctx, tx, ticketID, "response_time", 30, tenantID+999)
	require.Error(t, err, "tenantID 与 ticket.TenantID 不一致必须被拒绝")
	commands, err := client.OperationalCommand.Query().Where(operationalcommand.AggregateIDEQ(ticketID)).All(ctx)
	require.NoError(t, err)
	require.Empty(t, commands)
}

// TestNotifyTicketCreatedTxFailsClosedWhenFlagDisabled 验证未调用 EnableTxOutbox 时 Tx API 直接报错，
// 避免静默回退到 client 路径产生主表/通知分离提交。
func TestNotifyTicketCreatedTxFailsClosedWhenFlagDisabled(t *testing.T) {
	client, ctx, tenantID, userID, _ := notificationDeliveryFixture(t)
	svc := NewTicketNotificationService(client, zap.NewNop().Sugar())

	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	createdTicket, err := tx.Ticket.Create().
		SetTitle("disabled-flag-ticket").
		SetTicketNumber("NOTIFY-DIS").
		SetRequesterID(userID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	err = svc.NotifyTicketCreatedTx(ctx, tx, createdTicket)
	require.Error(t, err)
	require.Contains(t, err.Error(), "transactional notification outbox disabled")
}

// TestNotifySLAAlertLevelChangedTxCommits 验证 SLA 预警级别变更通知随 tx 提交而持久化。
func TestNotifySLAAlertLevelChangedTxCommits(t *testing.T) {
	client, ctx, tenantID, _, ticketID := notificationDeliveryFixture(t)
	svc := NewTicketNotificationService(client, zap.NewNop().Sugar())
	svc.EnableTxOutbox()

	tx, err := client.Tx(ctx)
	require.NoError(t, err)

	require.NoError(t, svc.NotifySLAAlertLevelChangedTx(ctx, tx, ticketID, "warning", 80.0, tenantID))
	require.NoError(t, tx.Commit())

	commands, err := client.OperationalCommand.Query().
		Where(operationalcommand.AggregateIDEQ(ticketID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Equal(t, commandbus.CommandDeliverNotification, commands[0].CommandType)
	require.NotNil(t, commands[0].Payload)
	require.Equal(t, "sla_alert", commands[0].Payload["type"])
}
