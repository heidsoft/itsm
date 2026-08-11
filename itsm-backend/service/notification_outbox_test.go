package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"itsm-backend/connector"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/notification"
	"itsm-backend/ent/notificationdelivery"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func notificationOutboxFixture(t *testing.T) (*ent.Client, context.Context, int, int, int) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	client := enttest.Open(t, dialect.SQLite, dsn)
	ctx := context.Background()
	tenantEntity, err := client.Tenant.Create().SetName("Request Tenant").SetCode("request-tenant").
		SetDomain("request.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	userEntity, err := client.User.Create().SetUsername("request-user").SetEmail("request@example.com").
		SetName("Request User").SetPasswordHash("hash").SetRole("manager").SetActive(true).
		SetTenantID(tenantEntity.ID).Save(ctx)
	require.NoError(t, err)
	request, err := client.ServiceRequest.Create().SetTenantID(tenantEntity.ID).SetCatalogID(1).
		SetRequesterID(userEntity.ID).SetTitle("生产权限申请").Save(ctx)
	require.NoError(t, err)
	return client, ctx, tenantEntity.ID, userEntity.ID, request.ID
}

func TestEnqueueResourceNotificationTxCommitsAndRollsBackWithBusinessTransaction(t *testing.T) {
	client, ctx, tenantID, userID, requestID := notificationOutboxFixture(t)
	build := func(occurrence string) ResourceNotificationCommand {
		return ResourceNotificationCommand{
			TenantID: tenantID, ResourceType: "service_request", ResourceID: requestID,
			RecipientID: userID, NotificationType: "approval_required", Channel: "in_app",
			Content: "您有新的审批待办", OccurrenceKey: occurrence,
		}
	}
	rollbackTx, err := client.Tx(ctx)
	require.NoError(t, err)
	require.NoError(t, EnqueueResourceNotificationTx(ctx, rollbackTx, build("rollback")))
	require.NoError(t, rollbackTx.Rollback())
	count, err := client.OperationalCommand.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)

	commitTx, err := client.Tx(ctx)
	require.NoError(t, err)
	require.NoError(t, EnqueueResourceNotificationTx(ctx, commitTx, build("commit")))
	require.NoError(t, commitTx.Commit())
	command, err := client.OperationalCommand.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, commandbus.CommandDeliverNotification, command.CommandType)
	require.Equal(t, "service_request", command.AggregateType)
	require.Contains(t, command.IdempotencyKey, fmt.Sprintf(":%d:in_app:%d:", requestID, userID))
}

func TestCreateServiceRequestCommitsApprovalNotificationCommand(t *testing.T) {
	client, ctx, tenantID, requesterID, _ := notificationOutboxFixture(t)
	catalog, err := client.ServiceCatalog.Create().SetName("云资源服务").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	item, err := client.ServiceCatalogItem.Create().SetCatalogID(catalog.ID).SetName("生产数据库申请").
		SetTenantID(tenantID).SetRequiresApproval(true).Save(ctx)
	require.NoError(t, err)
	service := NewServiceRequestService(client, zap.NewNop().Sugar(), nil, NewNotificationService(client))
	expiresAt := time.Now().Add(24 * time.Hour)

	created, err := service.CreateServiceRequest(ctx, &dto.CreateServiceRequestRequest{
		CatalogID: item.ID, Title: "申请生产数据库", DataClassification: "internal", ComplianceAck: true,
		ExpireAt: &expiresAt,
	}, requesterID, tenantID)
	require.NoError(t, err)
	require.NotZero(t, created.ID)
	commands, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID),
		operationalcommand.AggregateTypeEQ("service_request"),
		operationalcommand.AggregateIDEQ(created.ID),
	).All(ctx)
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Equal(t, commandbus.CommandDeliverNotification, commands[0].CommandType)
	require.EqualValues(t, requesterID, commands[0].Payload["recipientId"])
}

func TestServiceRequestNotificationHandlerProducesAuditedInAppDelivery(t *testing.T) {
	client, ctx, tenantID, userID, requestID := notificationOutboxFixture(t)
	command, err := commandbus.Enqueue(ctx, client, commandbus.EnqueueRequest{
		TenantID: tenantID, CommandType: commandbus.CommandDeliverNotification,
		AggregateType: "service_request", AggregateID: requestID, IdempotencyKey: "request-approval-1",
		Payload: map[string]interface{}{"resourceType": "service_request", "resourceId": requestID,
			"recipientId": userID, "type": "approval_required", "channel": "in_app", "content": "您有新的审批待办"},
	})
	require.NoError(t, err)
	command.Attempt = 1
	handler := NewNotificationDeliveryCommandHandler(client, connector.NewManager(connector.Default(), zap.NewNop().Sugar()), zap.NewNop().Sugar())
	require.NoError(t, handler.Handle(ctx, command))

	notificationCount, err := client.Notification.Query().Where(notification.TenantIDEQ(tenantID)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, notificationCount)
	deliveryCount, err := client.NotificationDelivery.Query().Where(
		notificationdelivery.TenantIDEQ(tenantID), notificationdelivery.OperationalCommandIDEQ(command.ID),
	).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, deliveryCount)
}

func TestEnqueueResourceNotificationTxRejectsCrossTenantRecipient(t *testing.T) {
	client, ctx, tenantID, _, requestID := notificationOutboxFixture(t)
	otherTenant, err := client.Tenant.Create().SetName("Other").SetCode("other").SetDomain("other.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	otherUser, err := client.User.Create().SetUsername("other").SetEmail("other@example.com").SetName("Other").
		SetPasswordHash("hash").SetTenantID(otherTenant.ID).Save(ctx)
	require.NoError(t, err)
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	err = EnqueueResourceNotificationTx(ctx, tx, ResourceNotificationCommand{
		TenantID: tenantID, ResourceType: "service_request", ResourceID: requestID,
		RecipientID: otherUser.ID, NotificationType: "approval", Channel: "in_app",
		Content: "content", OccurrenceKey: "cross-tenant",
	})
	require.Error(t, err)
	require.NoError(t, tx.Rollback())
	count, err := client.OperationalCommand.Query().Where(operationalcommand.TenantIDEQ(tenantID)).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}
