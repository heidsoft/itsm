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
