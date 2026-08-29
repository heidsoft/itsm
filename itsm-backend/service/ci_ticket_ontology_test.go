package service

import (
	"context"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/enttest"
	entticket "itsm-backend/ent/ticket"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// setupCITicketOntologyFixture 构造两个互不相关的租户，各自持有 CI 与工单，
// 用于验证 ITSM↔CMDB 本体绑定的租户隔离。
func setupCITicketOntologyFixture(t *testing.T) (*ent.Client, *ConfigurationItemService, context.Context) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", testDSN())
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewConfigurationItemService(client, logger, NewCIHistoryService(client, logger), NewCITagService(client, logger))
	return client, svc, ctx
}

// newTenantWithCIAndTicket 建一个租户，并在其中建一个 CI 和一个工单。
func newTenantWithCIAndTicket(t *testing.T, ctx context.Context, client *ent.Client, name string) (tenantID, ciID, ticketID int) {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName(name).SetCode(name).SetDomain(name + ".test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	ci, err := client.ConfigurationItem.Create().
		SetName(name + "-db-01").
		SetCiTypeID(ciType.ID).SetCiType("database").
		SetStatus("active").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// tickets.requester_id 有外键约束，必须指向真实用户
	user, err := client.User.Create().
		SetUsername(name + "-user").
		SetName(name + "-user").
		SetEmail(name + "@example.test").
		SetPasswordHash("x").
		SetRole("end_user").
		SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	tkt, err := client.Ticket.Create().
		SetTitle(name + " DB connection timeout").
		SetDescription("connection timeout observed").
		SetTicketNumber("TKT-TEST-" + name).
		SetStatus("open").SetPriority("high").
		SetRequesterID(user.ID).
		SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	return tenant.ID, ci.ID, tkt.ID
}

func TestLinkTicketToCI_BindsForeignKey(t *testing.T) {
	client, svc, ctx := setupCITicketOntologyFixture(t)
	defer client.Close()

	tenantID, ciID, ticketID := newTenantWithCIAndTicket(t, ctx, client, "tenant-a")

	require.NoError(t, svc.LinkTicketToCI(ctx, tenantID, ciID, ticketID))

	// 通过外键反查：工单应挂到该 CI 上
	tickets, err := svc.ListCITickets(ctx, tenantID, ciID, 10)
	require.NoError(t, err)
	require.Len(t, tickets, 1)
	require.Equal(t, ticketID, tickets[0]["id"])

	// 从 CI 侧用外键谓词反查，确认关系落在 tickets.configuration_item_tickets 真实外键列上
	// （而非塞进 form_fields JSON —— 后者会被工单类型字段校验拒绝）
	exists, err := client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(ciID), configurationitem.HasTicketsWith(entticket.IDEQ(ticketID))).
		Exist(ctx)
	require.NoError(t, err)
	require.True(t, exists, "CI 与工单之间应存在真实外键关联")
}

func TestLinkTicketToCI_RejectsCrossTenantCI(t *testing.T) {
	client, svc, ctx := setupCITicketOntologyFixture(t)
	defer client.Close()

	_, ciID, _ := newTenantWithCIAndTicket(t, ctx, client, "tenant-a")
	tenantB, _, ticketB := newTenantWithCIAndTicket(t, ctx, client, "tenant-b")

	// 用租户 B 的身份，去绑定租户 A 的 CI —— 必须拒绝
	err := svc.LinkTicketToCI(ctx, tenantB, ciID, ticketB)
	require.Error(t, err)
	require.Contains(t, err.Error(), "配置项不存在或无权访问")
}

func TestLinkTicketToCI_RejectsCrossTenantTicket(t *testing.T) {
	client, svc, ctx := setupCITicketOntologyFixture(t)
	defer client.Close()

	tenantA, ciA, _ := newTenantWithCIAndTicket(t, ctx, client, "tenant-a")
	_, _, ticketB := newTenantWithCIAndTicket(t, ctx, client, "tenant-b")

	// 用租户 A 的身份，去绑定租户 B 的工单 —— 必须拒绝
	err := svc.LinkTicketToCI(ctx, tenantA, ciA, ticketB)
	require.Error(t, err)
	require.Contains(t, err.Error(), "工单不存在或无权访问")
}

func TestLinkTicketToCI_RejectsNonPositiveIDs(t *testing.T) {
	client, svc, ctx := setupCITicketOntologyFixture(t)
	defer client.Close()

	err := svc.LinkTicketToCI(ctx, 1, 0, 1)
	require.Error(t, err)
	err = svc.LinkTicketToCI(ctx, 1, 1, 0)
	require.Error(t, err)
	err = svc.LinkTicketToCI(ctx, 1, -1, 1)
	require.Error(t, err)
}

func TestListCITickets_IsolatesTenant(t *testing.T) {
	client, svc, ctx := setupCITicketOntologyFixture(t)
	defer client.Close()

	tenantA, ciA, ticketA := newTenantWithCIAndTicket(t, ctx, client, "tenant-a")
	tenantB, _, _ := newTenantWithCIAndTicket(t, ctx, client, "tenant-b")

	require.NoError(t, svc.LinkTicketToCI(ctx, tenantA, ciA, ticketA))

	// 租户 B 用自己的身份去查租户 A 的 CI —— 必须查不到
	_, err := svc.ListCITickets(ctx, tenantB, ciA, 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "配置项不存在或无权访问")
}

func TestUnlinkTicketFromCI(t *testing.T) {
	client, svc, ctx := setupCITicketOntologyFixture(t)
	defer client.Close()

	tenantID, ciID, ticketID := newTenantWithCIAndTicket(t, ctx, client, "tenant-a")

	require.NoError(t, svc.LinkTicketToCI(ctx, tenantID, ciID, ticketID))
	require.NoError(t, svc.UnlinkTicketFromCI(ctx, tenantID, ciID, ticketID))

	tickets, err := svc.ListCITickets(ctx, tenantID, ciID, 10)
	require.NoError(t, err)
	require.Empty(t, tickets)
}

func TestUnlinkTicketFromCI_RejectsCrossTenantCI(t *testing.T) {
	client, svc, ctx := setupCITicketOntologyFixture(t)
	defer client.Close()

	_, ciA, _ := newTenantWithCIAndTicket(t, ctx, client, "tenant-a")
	tenantB, _, ticketB := newTenantWithCIAndTicket(t, ctx, client, "tenant-b")

	require.Error(t, svc.UnlinkTicketFromCI(ctx, tenantB, ciA, ticketB))
}

func TestToolRegistry_NewOntologyToolsDeclared(t *testing.T) {
	reg := NewToolRegistry(nil, nil, nil, nil)

	ciTickets := reg.GetTool("get_ci_tickets")
	require.NotNil(t, ciTickets)
	require.True(t, ciTickets.ReadOnly, "get_ci_tickets 必须是只读工具")
	require.Equal(t, "cmdb", ciTickets.Resource)

	link := reg.GetTool("link_ticket_ci")
	require.NotNil(t, link)
	require.False(t, link.ReadOnly, "link_ticket_ci 必须走审批流")
	require.Equal(t, "write", link.Action)
}

func TestToolRegistry_CanExecuteWriteToolRequiresDependency(t *testing.T) {
	reg := NewToolRegistry(nil, nil, nil, nil)

	// 领域服务未注入时，不得宣称自己能执行 —— 否则 ToolQueue 委派过来会在运行时 panic
	require.False(t, reg.canExecuteWriteTool("create_ticket"))
	require.False(t, reg.canExecuteWriteTool("link_ticket_ci"))
	require.False(t, reg.canExecuteWriteTool("create_ticket_type"))

	// 只读工具不走审批委派路径
	regNoop := NewToolRegistry(nil, nil, nil, nil)
	require.False(t, regNoop.canExecuteWriteTool("get_ci_tickets"))

	// 未知工具一律不允许
	require.False(t, reg.canExecuteWriteTool("not_a_tool"))
}
