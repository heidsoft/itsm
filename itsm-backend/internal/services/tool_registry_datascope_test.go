package service

// B4：AI / MCP 工具读面必须与 /tickets、/cmdb 页面同一行级范围。
// 此前 list_tickets 只按 tenant 过滤，低权限角色可经 AI 工具读取同租户他人单据。
//
// 反向验证：把 list_tickets 恢复成 t.client.Ticket.Query().Where(TenantID...) 的
// 全量查询（不接 ApplyTicketFilter），下面「普通角色只看本人」必须红在断言
// （出现他人单据），且不能是编译错误。

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/datascope"
)

func TestToolRegistryListTicketsRespectsDataScope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:tool_registry_datascope?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant := client.Tenant.Create().SetName("Scope").SetCode("SCOPE").
		SetDomain("scope.test").SetStatus("active").SaveX(ctx)
	mkUser := func(name string) *ent.User {
		return client.User.Create().SetUsername(name).SetEmail(name + "@test.com").
			SetName(name).SetPasswordHash("x").SetRole("end_user").SetActive(true).
			SetTenantID(tenant.ID).SaveX(ctx)
	}
	alice, bob := mkUser("alice"), mkUser("bob")
	client.Ticket.Create().SetTitle("alice ticket").SetTicketNumber("T-1").
		SetRequesterID(alice.ID).SetTenantID(tenant.ID).SaveX(ctx)
	client.Ticket.Create().SetTitle("bob ticket").SetTicketNumber("T-2").
		SetRequesterID(bob.ID).SetTenantID(tenant.ID).SaveX(ctx)

	registry := NewToolRegistry(nil, nil, nil, client)

	t.Run("普通角色只看本人创建或受理的单据", func(t *testing.T) {
		actorCtx := datascope.WithActor(ctx, datascope.Actor{
			UserID: alice.ID, Role: "end_user", TenantID: tenant.ID,
		})
		result, err := registry.Execute(actorCtx, tenant.ID, "list_tickets", map[string]interface{}{})
		require.NoError(t, err)
		items, ok := result.([]*dto.TicketResponse)
		require.True(t, ok)
		require.Len(t, items, 1, "普通角色不得读到他人的工单")
		require.Equal(t, "alice ticket", items[0].Title)
	})

	t.Run("管理角色保持全租户可见", func(t *testing.T) {
		actorCtx := datascope.WithActor(ctx, datascope.Actor{
			UserID: alice.ID, Role: "super_admin", TenantID: tenant.ID,
		})
		result, err := registry.Execute(actorCtx, tenant.ID, "list_tickets", map[string]interface{}{})
		require.NoError(t, err)
		require.Len(t, result.([]*dto.TicketResponse), 2)
	})

	t.Run("缺失或错租户 actor 必须 fail-closed", func(t *testing.T) {
		_, err := registry.Execute(ctx, tenant.ID, "list_tickets", map[string]interface{}{})
		require.Error(t, err, "无 actor 时不得退回租户全量")
		wrongCtx := datascope.WithActor(ctx, datascope.Actor{
			UserID: alice.ID, Role: "super_admin", TenantID: tenant.ID + 99,
		})
		_, err = registry.Execute(wrongCtx, tenant.ID, "list_tickets", map[string]interface{}{})
		require.Error(t, err, "actor 租户不匹配时不得放行")
	})

	t.Run("模型参数不能伪造 actor", func(t *testing.T) {
		actorCtx := datascope.WithActor(ctx, datascope.Actor{
			UserID: alice.ID, Role: "end_user", TenantID: tenant.ID,
		})
		result, err := registry.Execute(actorCtx, tenant.ID, "list_tickets", map[string]interface{}{
			"userId": float64(bob.ID), "role": "super_admin", "scopeAll": true,
		})
		require.NoError(t, err)
		require.Len(t, result.([]*dto.TicketResponse), 1, "模型参数不得覆盖服务端 actor")
	})
}

func TestToolRegistryListCIsRespectsDataScope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:tool_registry_ci_scope?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant := client.Tenant.Create().SetName("ScopeCI").SetCode("SCOPECI").
		SetDomain("scopeci.test").SetStatus("active").SaveX(ctx)
	owner := client.User.Create().SetUsername("owner").SetEmail("owner@test.com").
		SetName("Owner").SetPasswordHash("x").SetRole("end_user").SetActive(true).
		SetTenantID(tenant.ID).SaveX(ctx)
	ciType := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).SaveX(ctx)
	client.ConfigurationItem.Create().SetName("owned-ci").SetCiTypeID(ciType.ID).
		SetCiType("server").SetOwnerID(owner.ID).SetTenantID(tenant.ID).SaveX(ctx)
	client.ConfigurationItem.Create().SetName("other-ci").SetCiTypeID(ciType.ID).
		SetCiType("server").SetTenantID(tenant.ID).SaveX(ctx)

	logger := zap.NewNop().Sugar()
	ciSvc := NewConfigurationItemService(client, logger, NewCIHistoryService(client, logger), NewCITagService(client, logger))
	registry := NewToolRegistry(nil, nil, ciSvc, client)

	t.Run("缺少 actor 时 CI 工具 fail-closed", func(t *testing.T) {
		_, err := registry.Execute(ctx, tenant.ID, "list_cis", map[string]interface{}{})
		require.Error(t, err, "CI 工具无 actor 时不得返回全租户资产")
	})

	t.Run("普通角色只看到本人持有或未归属的 CI", func(t *testing.T) {
		actorCtx := datascope.WithActor(ctx, datascope.Actor{
			UserID: owner.ID, Role: "end_user", TenantID: tenant.ID,
		})
		result, err := registry.Execute(actorCtx, tenant.ID, "list_cis", map[string]interface{}{})
		require.NoError(t, err)
		items, ok := result.([]*dto.CIResponse)
		require.True(t, ok)
		// owned-ci（本人）与 other-ci（双 NULL 公共资产）均可见，这是既有产品规则；
		// 关键是不能因缺 actor 而绕过范围过滤。
		require.NotEmpty(t, items)
	})
}
