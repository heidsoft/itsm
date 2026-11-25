package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/datascope"
	ticketrepo "itsm-backend/repository/ticket"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestUpdateTicketToolEnforcesTrustedActorRowScope 是 AI/MCP update_ticket 写路径的
// 行级权限回归。HTTP 更新工单会把当前用户与角色传给 TicketService.UpdateTicket，
// 由 enforceTicketRowScope 拒绝非创建人/处理人的写入；工具写路径过去传 (0, "")
// 显式跳过该校验，低权限角色只要拿到 ticket:write 与人工审批即可改写同租户任意工单。
//
// 可信 actor 只能由服务端写入 context，模型参数不得参与身份判定；缺失或租户不匹配
// 必须 fail-closed。管理员角色保留原有的租户全量写入语义。
func TestUpdateTicketToolEnforcesTrustedActorRowScope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:update_ticket_row_scope?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("scope-tenant").SetCode("scope-tenant").SetDomain("scope.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	alice, err := client.User.Create().
		SetUsername("alice").SetName("alice").SetEmail("alice@scope.test").
		SetPasswordHash("x").SetRole("end_user").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	bob, err := client.User.Create().
		SetUsername("bob").SetName("bob").SetEmail("bob@scope.test").
		SetPasswordHash("x").SetRole("end_user").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	// 显式指定处理人，避免建单时默认分配把 alice 设为 bob 工单的 assignee，
	// 否则 alice 会合法拥有该工单，负例就失去意义。
	carol, err := client.User.Create().
		SetUsername("carol").SetName("carol").SetEmail("carol@scope.test").
		SetPasswordHash("x").SetRole("end_user").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()
	ticketSvc := NewTicketService(&TicketServiceConfig{
		Repository: ticketrepo.NewEntRepository(client, logger),
		Client:     client,
		Logger:     logger,
	})
	reg := NewToolRegistry(nil, nil, nil, client)
	reg.SetTicketService(ticketSvc)

	create := func(title string, requesterID int) *dto.TicketResponse {
		t.Helper()
		res, err := reg.Execute(ctx, tenant.ID, "create_ticket", map[string]interface{}{
			"title":       title,
			"requesterId": float64(requesterID),
			"assigneeId":  float64(carol.ID),
		})
		require.NoError(t, err)
		resp, ok := res.(*dto.TicketResponse)
		require.True(t, ok, "create_ticket must return *dto.TicketResponse, got %T", res)
		return resp
	}
	aliceTicket := create("alice ticket", alice.ID)
	bobTicket := create("bob ticket", bob.ID)

	// 前置条件：alice 既不是 bob 工单的请求人，也不是处理人；否则负例不成立。
	bobRow, err := client.Ticket.Get(ctx, bobTicket.ID)
	require.NoError(t, err)
	require.Equal(t, bob.ID, bobRow.RequesterID)
	require.Equal(t, carol.ID, bobRow.AssigneeID)
	require.NotEqual(t, alice.ID, bobRow.AssigneeID)

	statusOf := func(id int) string {
		t.Helper()
		got, err := client.Ticket.Get(ctx, id)
		require.NoError(t, err)
		return string(got.Status)
	}

	t.Run("non-owner actor cannot update another user's ticket", func(t *testing.T) {
		actorCtx := datascope.WithActor(ctx, datascope.Actor{
			UserID: alice.ID, Role: "end_user", TenantID: tenant.ID,
		})
		_, err := reg.Execute(actorCtx, tenant.ID, "update_ticket", map[string]interface{}{
			"ticketId":   float64(bobTicket.ID),
			"status":     "in_progress",
			"resolution": "should not be written",
		})
		require.Error(t, err, "row scope must deny a non-owner update")
		assert.Contains(t, err.Error(), "无权限")
		assert.Equal(t, bobTicket.Status, statusOf(bobTicket.ID), "denied update must not mutate the ticket")
	})

	t.Run("owner actor can update own ticket", func(t *testing.T) {
		actorCtx := datascope.WithActor(ctx, datascope.Actor{
			UserID: bob.ID, Role: "end_user", TenantID: tenant.ID,
		})
		res, err := reg.Execute(actorCtx, tenant.ID, "update_ticket", map[string]interface{}{
			"ticketId":   float64(bobTicket.ID),
			"status":     "in_progress",
			"resolution": "owner update",
		})
		require.NoError(t, err)
		resp, ok := res.(*dto.TicketResponse)
		require.True(t, ok)
		assert.Equal(t, "in_progress", resp.Status)
		assert.Equal(t, "in_progress", statusOf(bobTicket.ID))
	})

	t.Run("missing actor fails closed", func(t *testing.T) {
		_, err := reg.Execute(ctx, tenant.ID, "update_ticket", map[string]interface{}{
			"ticketId": float64(aliceTicket.ID),
			"status":   "in_progress",
		})
		require.Error(t, err, "a missing trusted actor must never fall back to system scope")
		assert.Contains(t, err.Error(), "update_ticket: authenticated actor context is missing")
		assert.Equal(t, aliceTicket.Status, statusOf(aliceTicket.ID))
	})

	t.Run("actor from another tenant fails closed", func(t *testing.T) {
		wrongCtx := datascope.WithActor(ctx, datascope.Actor{
			UserID: alice.ID, Role: "end_user", TenantID: tenant.ID + 99,
		})
		_, err := reg.Execute(wrongCtx, tenant.ID, "update_ticket", map[string]interface{}{
			"ticketId": float64(aliceTicket.ID),
			"status":   "in_progress",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant mismatch")
		assert.Equal(t, aliceTicket.Status, statusOf(aliceTicket.ID))
	})

	t.Run("admin actor keeps tenant-wide write scope", func(t *testing.T) {
		actorCtx := datascope.WithActor(ctx, datascope.Actor{
			UserID: alice.ID, Role: "super_admin", TenantID: tenant.ID,
		})
		_, err := reg.Execute(actorCtx, tenant.ID, "update_ticket", map[string]interface{}{
			"ticketId":   float64(aliceTicket.ID),
			"status":     "in_progress",
			"resolution": "admin update",
		})
		require.NoError(t, err)
		assert.Equal(t, "in_progress", statusOf(aliceTicket.ID))
	})
}
