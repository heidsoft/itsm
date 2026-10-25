package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
)

func newToolExecutorFixture(t *testing.T) (*ent.Client, *ent.Tenant, *ent.User, *ToolExecutor) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", testDSN())
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("ToolExecutor").SetCode("TOOL_EXEC").SetDomain("tool-exec.test").SetStatus("active").
		SaveX(ctx)
	user := client.User.Create().
		SetUsername("tool-operator").SetEmail("tool-operator@example.com").SetName("Operator").
		SetPasswordHash("hash").SetActive(true).SetTenantID(tenant.ID).
		SaveX(ctx)

	executor := NewToolExecutor(client, NewToolRegistry(nil, nil, nil, client), zap.NewNop().Sugar())
	return client, tenant, user, executor
}

func createInvocation(t *testing.T, client *ent.Client, tenantID, userID int, tool, args, status, state string) *ent.ToolInvocation {
	t.Helper()
	inv, err := client.ToolInvocation.Create().
		SetTenantID(tenantID).
		SetUserID(userID).
		SetToolName(tool).
		SetArguments(args).
		SetStatus(status).
		SetNeedsApproval(true).
		SetApprovalState(state).
		Save(context.Background())
	require.NoError(t, err)
	return inv
}

func TestToolExecutorExecutesApprovedInvocation(t *testing.T) {
	client, tenant, user, executor := newToolExecutorFixture(t)
	ctx := context.Background()
	inv := createInvocation(t, client, tenant.ID, user.ID, "list_tickets", `{}`, "pending", "approved")

	require.NoError(t, executor.ExecuteApprovedInvocation(ctx, inv.ID, tenant.ID))

	stored, err := client.ToolInvocation.Get(ctx, inv.ID)
	require.NoError(t, err)
	require.Equal(t, "done", stored.Status)
	require.NotNil(t, stored.Result)
	require.Nil(t, stored.Error)
}

func TestToolExecutorRefusesUnapprovedInvocation(t *testing.T) {
	client, tenant, user, executor := newToolExecutorFixture(t)
	ctx := context.Background()
	inv := createInvocation(t, client, tenant.ID, user.ID, "list_tickets", `{}`, "pending", "pending")

	err := executor.ExecuteApprovedInvocation(ctx, inv.ID, tenant.ID)
	require.ErrorContains(t, err, "not approved")

	stored, err := client.ToolInvocation.Get(ctx, inv.ID)
	require.NoError(t, err)
	require.Equal(t, "pending", stored.Status, "拒绝执行不能推进调用状态")
	require.Nil(t, stored.Result)
}

func TestToolExecutorIsTenantScoped(t *testing.T) {
	client, tenant, user, executor := newToolExecutorFixture(t)
	ctx := context.Background()
	otherTenant := client.Tenant.Create().
		SetName("Other").SetCode("OTHER_EXEC").SetDomain("other-exec.test").SetStatus("active").
		SaveX(ctx)
	inv := createInvocation(t, client, otherTenant.ID, user.ID, "list_tickets", `{}`, "pending", "approved")

	err := executor.ExecuteApprovedInvocation(ctx, inv.ID, tenant.ID)
	require.Error(t, err, "跨租户执行必须按资源不存在拒绝")

	stored, err := client.ToolInvocation.Get(ctx, inv.ID)
	require.NoError(t, err)
	require.Equal(t, "pending", stored.Status)
	require.Nil(t, stored.Result)
}

func TestToolExecutorDoesNotReclaimRunningInvocation(t *testing.T) {
	client, tenant, user, executor := newToolExecutorFixture(t)
	ctx := context.Background()
	inv := createInvocation(t, client, tenant.ID, user.ID, "list_tickets", `{}`, "running", "approved")

	err := executor.ExecuteApprovedInvocation(ctx, inv.ID, tenant.ID)
	require.ErrorContains(t, err, "not claimable")

	stored, err := client.ToolInvocation.Get(ctx, inv.ID)
	require.NoError(t, err)
	require.Equal(t, "running", stored.Status)
	require.Nil(t, stored.Result)
}

func TestToolExecutorTreatsDoneAsIdempotent(t *testing.T) {
	client, tenant, user, executor := newToolExecutorFixture(t)
	ctx := context.Background()
	result := `{"ok":true}`
	inv, err := client.ToolInvocation.Create().
		SetTenantID(tenant.ID).
		SetUserID(user.ID).
		SetToolName("list_tickets").
		SetArguments(`{}`).
		SetStatus("done").
		SetResult(result).
		SetApprovalState("approved").
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, executor.ExecuteApprovedInvocation(ctx, inv.ID, tenant.ID), "已完成的调用重放不应报错")

	stored, err := client.ToolInvocation.Get(ctx, inv.ID)
	require.NoError(t, err)
	require.Equal(t, result, *stored.Result)
}

func TestToolExecutorFinalizesFailures(t *testing.T) {
	client, tenant, user, executor := newToolExecutorFixture(t)
	ctx := context.Background()

	t.Run("非法参数", func(t *testing.T) {
		inv := createInvocation(t, client, tenant.ID, user.ID, "list_tickets", `{not-json`, "pending", "approved")
		require.Error(t, executor.ExecuteApprovedInvocation(ctx, inv.ID, tenant.ID))

		stored, err := client.ToolInvocation.Get(ctx, inv.ID)
		require.NoError(t, err)
		require.Equal(t, "failed", stored.Status)
		require.NotNil(t, stored.Error)
	})

	t.Run("工具执行失败", func(t *testing.T) {
		// update_ticket 依赖未注入的 TicketService：执行失败必须落 failed，而不是停在 running。
		inv := createInvocation(t, client, tenant.ID, user.ID, "update_ticket", `{"ticketId":1}`, "pending", "approved")
		require.Error(t, executor.ExecuteApprovedInvocation(ctx, inv.ID, tenant.ID))

		stored, err := client.ToolInvocation.Get(ctx, inv.ID)
		require.NoError(t, err)
		require.Equal(t, "failed", stored.Status)
		require.NotNil(t, stored.Error)
	})
}

// 结果没写回（行被删除/租户不匹配）时不能把命令标成功，否则审计丢失。
func TestToolExecutorFinalizeRequiresPersistedRow(t *testing.T) {
	client, tenant, user, executor := newToolExecutorFixture(t)
	ctx := context.Background()
	inv := createInvocation(t, client, tenant.ID, user.ID, "list_tickets", `{}`, "running", "approved")
	require.NoError(t, client.ToolInvocation.DeleteOneID(inv.ID).Exec(ctx))

	err := executor.finalize(ctx, inv.ID, tenant.ID, map[string]interface{}{"ok": true}, nil)
	require.ErrorContains(t, err, "not persisted")
}

func TestWithInvocationUserUsesCamelCaseRequesterID(t *testing.T) {
	merged := withInvocationUser(map[string]interface{}{"title": "x"}, 7)
	require.Equal(t, float64(7), merged["requesterId"])
	_, snake := merged["user_id"]
	require.False(t, snake, "工具参数必须使用 camelCase，与 schema 保持一致")

	provided := withInvocationUser(map[string]interface{}{"requesterId": float64(9)}, 7)
	require.Equal(t, float64(9), provided["requesterId"], "调用方显式指定的请求人不应被覆盖")
}
