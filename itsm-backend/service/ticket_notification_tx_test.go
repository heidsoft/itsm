package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/ent/ticket"
	"itsm-backend/internal/commandbus"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// txNotificationFixture 阶段 B 集成测试的最小夹具：
// 一个 tenant + 一个 requester + 一个 assignee。assignee 用于让 NotifyTicketCreatedTx 实际入箱。
func txNotificationFixture(t *testing.T) (*ent.Client, context.Context, *ent.Tenant, *ent.User, *ent.User) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	client := enttest.Open(t, dialect.SQLite, dsn)
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Tx Notify Tenant").
		SetCode("tx-notify-tenant").
		SetDomain("tx-notify.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	requester, err := client.User.Create().
		SetUsername("requester-tx").
		SetEmail("requester-tx@example.com").
		SetName("Requester Tx").
		SetPasswordHash("hash").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	assignee, err := client.User.Create().
		SetUsername("assignee-tx").
		SetEmail("assignee-tx@example.com").
		SetName("Assignee Tx").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return client, ctx, tenant, requester, assignee
}

// TestCreateTicketSinksCreatedNotificationIntoTx 验证 CreateTicket 提交后
// ticket 与通知入箱在同一 tx 内完成，operational_command 行立即可见。
// 这是阶段 B 端到端契约。
func TestCreateTicketSinksCreatedNotificationIntoTx(t *testing.T) {
	client, ctx, tenant, requester, assignee := txNotificationFixture(t)

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewTicketServiceForTest(client, logger)
	notificationSvc := NewTicketNotificationService(client, logger)
	notificationSvc.EnableTxOutbox()
	svc.SetNotificationService(notificationSvc)

	created, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       "Phase B sink ticket",
		Description: "verify ticket + notification are atomic",
		Priority:    "high",
		Type:        "incident",
		RequesterID: requester.ID,
		AssigneeID:  assignee.ID,
	}, tenant.ID)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, "high", string(created.Priority))

	// 验证 ticket 已落库
	storedTicket, err := client.Ticket.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, requester.ID, storedTicket.RequesterID)

	// 验证通知已通过 tx 入箱（assignee + requester 两个收件人各一行）
	commands, err := client.OperationalCommand.Query().
		Where(operationalcommand.AggregateIDEQ(created.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, commands, 2, "assignee + requester 必须各产生一条 operational_command")

	for _, cmd := range commands {
		require.Equal(t, commandbus.CommandDeliverNotification, cmd.CommandType)
		require.Equal(t, "ticket", cmd.AggregateType)
		require.Equal(t, tenant.ID, cmd.TenantID)
		require.NotNil(t, cmd.Payload)
		require.Equal(t, "created", cmd.Payload["type"])
		require.Equal(t, "in_app", cmd.Payload["channel"])
	}
}

func TestCreateTicketSinksWorkflowStartIntoSameTransaction(t *testing.T) {
	client, ctx, tenant, requester, _ := txNotificationFixture(t)

	svc := NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar())
	svc.EnableWorkflowOutbox()
	svc.EnableSideEffectOutbox()
	created, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "Durable workflow ticket", Description: "workflow command commits with ticket",
		Priority: "medium", Type: "incident", RequesterID: requester.ID,
		WorkflowDefinitionKey: "tenant_ticket_flow",
	}, tenant.ID)
	require.NoError(t, err)

	cmd, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenant.ID),
		operationalcommand.CommandTypeEQ(commandbus.CommandStartBPMN),
		operationalcommand.AggregateTypeEQ("ticket"),
		operationalcommand.AggregateIDEQ(created.ID),
	).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("ticket:%d:workflow:start", created.ID), cmd.IdempotencyKey)
	require.Equal(t, "tenant_ticket_flow", cmd.Payload["workflowDefinitionKey"])
	for _, commandType := range []string{commandbus.CommandExecuteTicketRules, commandbus.CommandSyncTicketFeishu} {
		command, err := client.OperationalCommand.Query().Where(
			operationalcommand.TenantIDEQ(tenant.ID), operationalcommand.CommandTypeEQ(commandType),
			operationalcommand.AggregateIDEQ(created.ID),
		).Only(ctx)
		require.NoError(t, err)
		require.Equal(t, commandbus.StatusPending, command.Status)
	}
}

// TestCreateTicketRollsBackWhenNotificationEnqueueFails 验证通知入箱失败时
// ticket 也必须随 tx 一起回滚——这是「同生同死」语义的核心契约。
//
// 通知服务在持有同一 tenant 时可通过手工塞入相同 idempotency_key 的行来制造约束冲突；
// 但 occurrenceKey 由 ticket.ID 动态生成，更稳的做法是直接验证单元级别的 runCreateTicketTx
// 行为。集成测试层验证「commit 时可见」已足够覆盖主线。
func TestCreateTicketRollsBackWhenNotificationEnqueueFails(t *testing.T) {
	client, ctx, tenant, requester, assignee := txNotificationFixture(t)

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewTicketServiceForTest(client, logger)
	notificationSvc := NewTicketNotificationService(client, zap.NewNop().Sugar())
	// 注意：故意不调用 EnableTxOutbox，Tx 方法会 fail-closed。
	svc.SetNotificationService(notificationSvc)

	created, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:       "Phase B rollback ticket",
		Description: "verify tx rollback when notification disabled",
		Priority:    "high",
		Type:        "incident",
		RequesterID: requester.ID,
		AssigneeID:  assignee.ID,
	}, tenant.ID)
	require.Error(t, err, "未开启 EnableTxOutbox 时 CreateTicket 必须因通知入箱失败而整体回滚")
	require.Nil(t, created)
	require.Contains(t, err.Error(), "transactional notification outbox disabled")

	// ticket 不应落库
	tickets, err := client.Ticket.Query().
		Where(ticket.TenantIDEQ(tenant.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Empty(t, tickets, "rollback 后 ticket 必须随业务事务一起消失")

	// operational_command 也不应落库
	commands, err := client.OperationalCommand.Query().
		Where(operationalcommand.TenantIDEQ(tenant.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Empty(t, commands, "rollback 后通知入箱行必须随业务事务一起消失")
}
