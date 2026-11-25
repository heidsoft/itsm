package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/handlers/ai"
	"itsm-backend/internal/authz"
	"itsm-backend/middleware"
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
	guard := ai.NewService(ai.NewEntRepository(client), zap.NewNop().Sugar(), nil, NewToolRegistry(nil, nil, nil, client), nil, nil, nil, nil, nil, nil)
	guard.SetEntClient(client)
	guard.SetPermissionCatalog(authz.NewRegistry(client))
	executor.SetApprovedToolGuard(guard)
	return client, tenant, user, executor
}

func createInvocation(t *testing.T, client *ent.Client, tenantID, userID int, tool, args, status, state string) *ent.ToolInvocation {
	t.Helper()
	inv, err := client.ToolInvocation.Create().
		SetTenantID(tenantID).
		SetUserID(userID).
		SetToolName(tool).
		SetRoleSnapshot("super_admin").
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

type executionCounter struct{ calls int }

func (s *executionCounter) Execute(context.Context, int, string, map[string]interface{}) (interface{}, error) {
	s.calls++
	return map[string]interface{}{"ok": true}, nil
}

func TestToolExecutorRevalidatesApprovedSnapshot(t *testing.T) {
	for _, scenario := range []string{"revoked_role", "changed_policy", "unknown_policy", "unknown_unreadable", "approved", "cross_tenant", "nil_guard", "guard_error", "missing_tool"} {
		t.Run(scenario, func(t *testing.T) {
			client, tenant, user, executor := newToolExecutorFixture(t)
			ctx := context.Background()
			policy := authz.DerivePolicy("ticket", "write")
			policy.AllowedPrincipals = []string{authz.PrincipalHuman, authz.PrincipalAgent}
			encoded, err := authz.EncodeDefinition("ticket write", policy)
			require.NoError(t, err)
			permission := client.Permission.Create().SetCode("ticket:write").SetName("ticket write").
				SetDescription(encoded).SetResource("ticket").SetAction("write").SetTenantID(tenant.ID).SaveX(ctx)
			def, err := authz.NewRegistry(client).Lookup(ctx, tenant.ID, "ticket:write")
			require.NoError(t, err)
			inv := createInvocation(t, client, tenant.ID, user.ID, "create_ticket", "{}", "pending", "approved")
			update := client.ToolInvocation.UpdateOneID(inv.ID).SetPolicyVersion(def.PolicyVersion())
			switch scenario {
			case "revoked_role":
				role := client.Role.Create().SetName("revoked").SetCode("revoked").SetTenantID(tenant.ID).SaveX(ctx)
				binding := client.RolePermission.Create().SetRoleID(role.ID).SetPermissionID(permission.ID).SetTenantID(tenant.ID).SaveX(ctx)
				allowed, _, checkErr := executor.guard.RevalidateApprovedTool(ctx, tenant.ID, user.ID, "revoked", inv.ToolName, inv.Arguments, def.PolicyVersion())
				require.NoError(t, checkErr)
				require.True(t, allowed)
				require.NoError(t, client.RolePermission.DeleteOneID(binding.ID).Exec(ctx))
				middleware.InvalidateAllPermissionCaches()
				update.SetRoleSnapshot("revoked")
			case "changed_policy":
				policy.MinApprovals++
				changed, encodeErr := authz.EncodeDefinition("ticket write", policy)
				require.NoError(t, encodeErr)
				client.Permission.UpdateOneID(permission.ID).SetDescription(changed).SaveX(ctx)
			case "unknown_policy", "unknown_unreadable":
				update.SetPolicyVersion(authz.PolicyVersionUnknown)
				if scenario == "unknown_unreadable" {
					require.NoError(t, client.Permission.DeleteOneID(permission.ID).Exec(ctx))
				}
			case "nil_guard":
				executor.SetApprovedToolGuard(nil)
			case "guard_error":
				executor.SetApprovedToolGuard(failingApprovedToolGuard{})
			case "missing_tool":
				update.SetToolName("removed_tool")
			}
			update.SaveX(ctx)
			spy := &executionCounter{}
			executor.tools = spy
			commandTenant := tenant.ID
			if scenario == "cross_tenant" {
				otherTenant := client.Tenant.Create().SetName("Tenant B").SetCode("GUARD_B").
					SetDomain("guard-b.test").SetStatus("active").SaveX(ctx)
				commandTenant = otherTenant.ID
			}
			err = executor.HandleCommand(ctx, &ent.OperationalCommand{AggregateID: inv.ID, TenantID: commandTenant})
			expected := 0
			if scenario == "approved" {
				expected = 1
			}
			require.Equal(t, expected, spy.calls, "execution side effects")
			stored := client.ToolInvocation.GetX(ctx, inv.ID)
			if expected == 1 {
				require.NoError(t, err)
				require.Equal(t, "done", stored.Status)
				require.NoError(t, executor.HandleCommand(ctx, &ent.OperationalCommand{AggregateID: inv.ID, TenantID: commandTenant}))
				require.Equal(t, 1, spy.calls)
			} else {
				require.Error(t, err)
				if scenario == "cross_tenant" {
					require.Equal(t, "pending", stored.Status)
				} else {
					require.Equal(t, "failed", stored.Status)
					require.Equal(t, "denied", stored.PermissionCheck)
					require.NotEmpty(t, stored.PermissionReason)
				}
			}
		})
	}
}

type failingApprovedToolGuard struct{}

func (failingApprovedToolGuard) RevalidateApprovedTool(context.Context, int, int, string, string, string, string) (bool, string, error) {
	return true, "unsafe provider detail", errors.New("private provider error")
}
