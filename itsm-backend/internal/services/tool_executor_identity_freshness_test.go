package service

// B3：审批通过后，执行前必须按发起人当前账号状态/角色重判。
// RoleSnapshot 只是历史证据，不再作为授权依据。
//
// 反向验证：把 handlers/ai.Service.ResolveCurrentInitiator 改成直接返回
// snapshotRole（旧行为），本测试的 user_deactivated / role_downgraded
// 子用例必须红在 spy.calls（变成 1），且不是编译错误。

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"itsm-backend/ent"
	"itsm-backend/ent/user"
	"itsm-backend/internal/authz"
)

func TestToolExecutorRevalidatesCurrentInitiator(t *testing.T) {
	for _, scenario := range []string{"user_deactivated", "role_downgraded", "still_valid"} {
		t.Run(scenario, func(t *testing.T) {
			client, tenant, initiator, executor := newToolExecutorFixture(t)
			ctx := context.Background()

			// 造一个持 ticket:write 的角色并绑定给发起人（当前角色 = 高权限）。
			policy := authz.DerivePolicy("ticket", "write")
			policy.AllowedPrincipals = []string{authz.PrincipalHuman, authz.PrincipalAgent}
			encoded, err := authz.EncodeDefinition("ticket write", policy)
			require.NoError(t, err)
			permission := client.Permission.Create().SetCode("ticket:write").SetName("ticket write").
				SetDescription(encoded).SetResource("ticket").SetAction("write").SetTenantID(tenant.ID).SaveX(ctx)
			// 用内置角色 code（与 user.role enum 一致）建权限绑定，确保
			// 当前角色查表能命中这条绑定。
			role := client.Role.Create().SetName("IT 管理员").SetCode(string(user.RoleItAdmin)).SetTenantID(tenant.ID).SaveX(ctx)
			client.RolePermission.Create().SetRoleID(role.ID).SetPermissionID(permission.ID).SetTenantID(tenant.ID).SaveX(ctx)

			// 发起人当前角色设为持权角色 it_admin。
			client.User.UpdateOneID(initiator.ID).SetRole(user.RoleItAdmin).SaveX(ctx)

			inv := createInvocation(t, client, tenant.ID, initiator.ID, "create_ticket", `{}`, "pending", "approved")
			def, err := authz.NewRegistry(client).Lookup(ctx, tenant.ID, "ticket:write")
			require.NoError(t, err)
			client.ToolInvocation.UpdateOneID(inv.ID).SetPolicyVersion(def.PolicyVersion()).SaveX(ctx)

			switch scenario {
			case "user_deactivated":
				// 只停用账号，角色权限绑定保持不变。
				client.User.UpdateOneID(initiator.ID).SetActive(false).SaveX(ctx)
			case "role_downgraded":
				// 只把当前角色降为 end_user，旧角色的权限绑定仍在。
				client.User.UpdateOneID(initiator.ID).SetRole(user.RoleEndUser).SaveX(ctx)
			}

			spy := &executionCounter{}
			executor.tools = spy
			err = executor.HandleCommand(ctx, &ent.OperationalCommand{AggregateID: inv.ID, TenantID: tenant.ID})

			if scenario == "still_valid" {
				require.NoError(t, err)
				require.Equal(t, 1, spy.calls, "身份仍有效时应正常执行")
				return
			}
			require.Error(t, err, "身份失效后不得执行已排队的调用")
			require.Zero(t, spy.calls, "身份失效后工具副作用必须为 0")
			stored := client.ToolInvocation.GetX(ctx, inv.ID)
			require.Equal(t, "failed", stored.Status)
			require.Equal(t, "denied", stored.PermissionCheck)
		})
	}
}
