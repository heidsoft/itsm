package ticket

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"
	repoTicket "itsm-backend/repository/ticket"
	"itsm-backend/service"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 指派规则唯一所有者是 productionSvc——跨租户 assignee 必须被拒绝，
// handler 层不得再提供绕过同租户校验的直通 repo 路径。
func TestAssignTicket_RejectsCrossTenantAssignee(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket-assign-tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zap.NewNop().Sugar()

	tenant1, err := client.Tenant.Create().SetName("T1").SetCode("t1-assign").Save(ctx)
	require.NoError(t, err)
	tenant2, err := client.Tenant.Create().SetName("T2").SetCode("t2-assign").Save(ctx)
	require.NoError(t, err)
	requester, err := client.User.Create().SetUsername("req1").SetEmail("req1@example.com").SetName("Req").SetPasswordHash("h").SetTenantID(tenant1.ID).Save(ctx)
	require.NoError(t, err)
	assigneeT1, err := client.User.Create().SetUsername("a1").SetEmail("a1@example.com").SetName("A1").SetPasswordHash("h").SetActive(true).SetTenantID(tenant1.ID).Save(ctx)
	require.NoError(t, err)
	assigneeT2, err := client.User.Create().SetUsername("a2").SetEmail("a2@example.com").SetName("A2").SetPasswordHash("h").SetActive(true).SetTenantID(tenant2.ID).Save(ctx)
	require.NoError(t, err)

	tk := client.Ticket.Create().
		SetTitle("跨租户指派回归").SetTicketNumber("TK-ASSIGN-1").SetStatus("open").
		SetRequesterID(requester.ID).SetTenantID(tenant1.ID).SaveX(ctx)

	svc := NewService(
		NewEntRepository(repoTicket.NewEntRepository(client, logger), client),
		service.NewTicketServiceForTest(client, logger),
		nil, logger,
	)

	_, err = svc.AssignTicket(ctx, tk.ID, assigneeT2.ID, tenant1.ID)
	require.ErrorContains(t, err, "处理人不存在或不可用")

	// 跨租户 assignee 不得落库
	reloaded, err := client.Ticket.Get(ctx, tk.ID)
	require.NoError(t, err)
	require.Zero(t, reloaded.AssigneeID)

	// 同租户激活用户指派成功
	updated, err := svc.AssignTicket(ctx, tk.ID, assigneeT1.ID, tenant1.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.AssigneeID)
	require.Equal(t, assigneeT1.ID, *updated.AssigneeID)
}

// Update 部分更新 params.AssigneeID 同样受同租户校验约束。
func TestUpdate_RejectsCrossTenantAssignee(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket-update-tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zap.NewNop().Sugar()

	tenant1, err := client.Tenant.Create().SetName("T1").SetCode("t1-upd").Save(ctx)
	require.NoError(t, err)
	tenant2, err := client.Tenant.Create().SetName("T2").SetCode("t2-upd").Save(ctx)
	require.NoError(t, err)
	requester, err := client.User.Create().SetUsername("req1").SetEmail("req1u@example.com").SetName("Req").SetPasswordHash("h").SetTenantID(tenant1.ID).Save(ctx)
	require.NoError(t, err)
	assigneeT2, err := client.User.Create().SetUsername("a2").SetEmail("a2u@example.com").SetName("A2").SetPasswordHash("h").SetActive(true).SetTenantID(tenant2.ID).Save(ctx)
	require.NoError(t, err)

	tk := client.Ticket.Create().
		SetTitle("Update 跨租户指派回归").SetTicketNumber("TK-ASSIGN-2").SetStatus("open").
		SetRequesterID(requester.ID).SetTenantID(tenant1.ID).SaveX(ctx)

	svc := NewService(
		NewEntRepository(repoTicket.NewEntRepository(client, logger), client),
		service.NewTicketServiceForTest(client, logger),
		nil, logger,
	)

	assigneeID := assigneeT2.ID
	_, err = svc.Update(ctx, tenant1.ID, tk.ID, &UpdateParams{AssigneeID: &assigneeID}, requester.ID, "super_admin")
	require.ErrorContains(t, err, "处理人不存在或不可用")

	reloaded, err := client.Ticket.Get(ctx, tk.ID)
	require.NoError(t, err)
	require.Zero(t, reloaded.AssigneeID)
}
