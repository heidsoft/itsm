package feishu

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"itsm-backend/connector"
	feishuConn "itsm-backend/connector/builtin/feishu"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	entTicket "itsm-backend/ent/ticket"
	"itsm-backend/internal/commandbus"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newInboundFixture(t *testing.T, name string) (*ent.Client, *FeishuSyncService) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:feishu_inbound_%s_%d?mode=memory&cache=shared&_fk=1", name, time.Now().UnixNano()))
	return client, NewFeishuSyncService(client, zap.NewNop().Sugar())
}

func seedInboundTenant(t *testing.T, client *ent.Client, seed int) int {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName(fmt.Sprintf("inbound-tenant-%d", seed)).
		SetCode(fmt.Sprintf("inbound-%d", seed)).
		SetDomain(fmt.Sprintf("inbound-%d.example.com", seed)).
		SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)
	return tenant.ID
}

func seedInboundUser(t *testing.T, client *ent.Client, tenantID int, openID string) int {
	t.Helper()
	builder := client.User.Create().
		SetTenantID(tenantID).
		SetUsername(fmt.Sprintf("inbound-user-%d-%s", tenantID, openID)).
		SetEmail(fmt.Sprintf("inbound-%d-%s@example.com", tenantID, openID)).
		SetName("Inbound User").
		SetPasswordHash("hash").
		SetActive(true)
	if openID != "" {
		builder.SetFeishuOpenID(openID)
	}
	u, err := builder.Save(context.Background())
	require.NoError(t, err)
	return u.ID
}

func seedInboundConfig(t *testing.T, client *ent.Client, tenantID int, key, value string) {
	t.Helper()
	_, err := client.SystemConfig.Create().
		SetTenantID(tenantID).
		SetKey(key).
		SetValue(value).
		Save(context.Background())
	require.NoError(t, err)
}

func inboundMessage(messageID, openID, text string) *connector.InboundMessage {
	return &connector.InboundMessage{
		ConnectorName: "feishu",
		ConnectorType: connector.TypeIM,
		MessageID:     messageID,
		UserID:        openID,
		Content:       text,
		Type:          "im.message.receive_v1",
		Raw:           []byte(fmt.Sprintf(`{"uuid":%q}`, messageID)),
	}
}

func TestInboundTicket_AutoCreateConsistentTenantAndDurableOutbox(t *testing.T) {
	client, svc := newInboundFixture(t, "auto")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 11)
	requester := seedInboundUser(t, client, tenantID, "ou-auto")
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)

	resp, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-auto-1", "ou-auto", "仓库温湿度告警\n请处理"))
	require.NoError(t, err)
	require.Equal(t, "ticket_created", resp.Action)
	require.Equal(t, "im.message.receive_v1", resp.EventType)
	require.Positive(t, resp.TicketID)
	require.NotEmpty(t, resp.TicketNumber)

	ticket, err := client.Ticket.Query().Where(entTicket.ID(resp.TicketID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, tenantID, ticket.TenantID)
	require.Equal(t, requester, ticket.RequesterID)
	require.Equal(t, "仓库温湿度告警", ticket.Title)
	require.Contains(t, ticket.Description, "请处理")

	audit, err := client.AuditLog.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, tenantID, audit.TenantID)
	require.Equal(t, "feishu_inbound", audit.Resource)
	require.Equal(t, "ticket.create", audit.Action)
	require.NotNil(t, audit.RequestBody)
	require.NotContains(t, *audit.RequestBody, "请处理", "消息正文不得进入审计")

	cmd, err := client.OperationalCommand.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, tenantID, cmd.TenantID)
	require.Equal(t, commandbus.CommandDeliverNotification, cmd.CommandType)
	require.Contains(t, cmd.IdempotencyKey, "om-auto-1")
	require.Equal(t, ticket.ID, cmd.AggregateID)
}

func TestInboundTicket_DuplicateMessageDoesNotCreateSecondTicket(t *testing.T) {
	client, svc := newInboundFixture(t, "dup")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 12)
	seedInboundUser(t, client, tenantID, "ou-dup")
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)

	msg := inboundMessage("om-dup-1", "ou-dup", "重复的故障消息")
	_, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, msg)
	require.NoError(t, err)

	_, err = svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, msg)
	require.True(t, errors.Is(err, errInboundDuplicate), "重复消息必须返回稳定冲突错误，得到 %v", err)

	tickets, err := client.Ticket.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, tickets, 1)
	commands, err := client.OperationalCommand.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, commands, 1)
}

func TestInboundTicket_ObserveOnlyCreatesNothing(t *testing.T) {
	client, svc := newInboundFixture(t, "observe")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 13)
	seedInboundUser(t, client, tenantID, "ou-observe")

	resp, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-obs-1", "ou-observe", "只看不建"))
	require.NoError(t, err)
	require.Equal(t, "observed", resp.Action)
	require.Zero(t, resp.TicketID)

	tickets, err := client.Ticket.Query().All(ctx)
	require.NoError(t, err)
	require.Empty(t, tickets)
}

func TestInboundTicket_UnknownSenderIsForbidden(t *testing.T) {
	client, svc := newInboundFixture(t, "forbidden")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 14)
	seedInboundUser(t, client, tenantID, "ou-known")
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)

	_, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-fb-1", "ou-unknown", "未映射发送者"))
	require.True(t, errors.Is(err, errInboundRequesterForbidden), "未映射发送者必须 forbidden，得到 %v", err)
	tickets, err := client.Ticket.Query().All(ctx)
	require.NoError(t, err)
	require.Empty(t, tickets)
}

func TestInboundTicket_UnconfiguredReporterFailsClosed(t *testing.T) {
	client, svc := newInboundFixture(t, "unconfigured")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 15)
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)

	_, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-uc-1", "", "无申请人"))
	require.True(t, errors.Is(err, errInboundUnconfigured), "无申请人且未配置自动化账号必须 unavailable，得到 %v", err)
}

func TestInboundTicket_AutomationReporterFallback(t *testing.T) {
	client, svc := newInboundFixture(t, "reporter")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 16)
	reporter := seedInboundUser(t, client, tenantID, "")
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)
	seedInboundConfig(t, client, tenantID, inboundReporterConfigKey, fmt.Sprintf("%d", reporter))

	resp, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-rp-1", "", "自动化账号建单"))
	require.NoError(t, err)
	require.Equal(t, "ticket_created", resp.Action)
	ticket, err := client.Ticket.Query().Where(entTicket.ID(resp.TicketID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, reporter, ticket.RequesterID)
}

func TestInboundTicket_TenantIsolation(t *testing.T) {
	client, svc := newInboundFixture(t, "tenant")
	ctx := context.Background()
	tenantA := seedInboundTenant(t, client, 17)
	tenantB := seedInboundTenant(t, client, 18)
	seedInboundUser(t, client, tenantA, "ou-a")
	seedInboundUser(t, client, tenantB, "ou-b")
	seedInboundConfig(t, client, tenantA, inboundModeConfigKey, inboundModeAutoCreate)
	seedInboundConfig(t, client, tenantB, inboundModeConfigKey, inboundModeAutoCreate)

	_, err := svc.CreateTicketFromInboundMessage(ctx, tenantA, &feishuConn.Feishu{}, inboundMessage("om-t-1", "ou-a", "A 租户消息"))
	require.NoError(t, err)

	// 用 B 租户调用同一 open_id 必须拒绝：身份映射是租户本地的，不能跨租户借用。
	_, err = svc.CreateTicketFromInboundMessage(ctx, tenantB, &feishuConn.Feishu{}, inboundMessage("om-t-2", "ou-a", "跨租户注入"))
	require.True(t, errors.Is(err, errInboundRequesterForbidden), "跨租户借用发送者必须拒绝，得到 %v", err)

	owned, err := client.Ticket.Query().Where(entTicket.TenantIDEQ(tenantB)).All(ctx)
	require.NoError(t, err)
	require.Empty(t, owned)
}

func TestInboundTicket_DedupRowKeepsStableErrorClass(t *testing.T) {
	client, svc := newInboundFixture(t, "dedup")
	ctx := context.Background()
	tenantID := seedInboundTenant(t, client, 19)
	seedInboundUser(t, client, tenantID, "ou-dd")
	seedInboundConfig(t, client, tenantID, inboundModeConfigKey, inboundModeAutoCreate)

	_, err := svc.CreateTicketFromInboundMessage(ctx, tenantID, &feishuConn.Feishu{}, inboundMessage("om-dd-1", "ou-dd", "去重状态检查"))
	require.NoError(t, err)
	rows, err := client.ConnectorInboundDedup.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "ok", rows[0].ResponseStatus)
	require.Equal(t, inboundMessageDedupConnector, rows[0].ConnectorName)
}
