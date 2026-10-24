package service

import (
	"context"
	"fmt"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/internal/commandbus"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// workflow.start 消费侧的工单选路契约：payload 里的流程 Key 来自
// 「请求 > 工单类型 > 工单分类」，分类绑定现在会真的出现在 payload 中；
// 一旦它指向本租户不存在或已停用的流程定义，命令必须回落到绑定表/内置兜底并继续，
// 而不是反复重试把工单流程卡死。

func newTicketRouteFixture(t *testing.T) (*ent.Client, context.Context, int, int) {
	t.Helper()
	client, ctx := setupChangeWorkflowTest(t)
	tenant, err := createRouteTestTenant(ctx, client, fmt.Sprintf("ticket-%d", nextRouteTestSeq()))
	require.NoError(t, err)
	// 生产基线：内置模板由 BPMNTemplateService 在每个 active 租户部署，
	// ticket_general_flow 必然存在。fixture 若不造它，测的就不是真实环境。
	deployTicketDefinition(t, client, ctx, tenant.ID, "ticket_general_flow")
	return client, ctx, tenant.ID, createRouteUser(t, client, ctx, tenant.ID)
}

// createRouteUser tickets.requester_id 有外键约束，测试必须落一个真实用户。
func createRouteUser(t *testing.T, client *ent.Client, ctx context.Context, tenantID int) int {
	t.Helper()
	seq := nextRouteTestSeq()
	u, err := client.User.Create().
		SetUsername(fmt.Sprintf("route_req_%d", seq)).
		SetEmail(fmt.Sprintf("route-%d@example.com", seq)).
		SetName("Route Requester").SetPasswordHash("hash").
		SetRole("end_user").SetActive(true).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	return u.ID
}

func createTicketForRouteTest(t *testing.T, client *ent.Client, ctx context.Context, tenantID, requesterID int, priority string) *ent.Ticket {
	t.Helper()
	seq := nextRouteTestSeq()
	tkt, err := client.Ticket.Create().
		SetTicketNumber(fmt.Sprintf("TKT-ROUTE-%d", seq)).
		SetTitle(fmt.Sprintf("路由契约测试工单 %d", seq)).
		SetDescription("ticket routing contract").
		SetType("incident").
		SetPriority(priority).
		SetStatus("new").
		SetRequesterID(requesterID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return tkt
}

func createTicketRouteCommand(t *testing.T, client *ent.Client, ctx context.Context, ticketID, tenantID int, payload map[string]interface{}) *ent.OperationalCommand {
	t.Helper()
	cmd, err := client.OperationalCommand.Create().
		SetTenantID(tenantID).
		SetCommandType(commandbus.CommandStartBPMN).
		SetAggregateType("ticket").
		SetAggregateID(ticketID).
		SetIdempotencyKey(fmt.Sprintf("ticket:%d:workflow:start:test", ticketID)).
		SetPayload(payload).
		SetStatus(commandbus.StatusPending).
		Save(ctx)
	require.NoError(t, err)
	return cmd
}

// deployTicketDefinition 部署流程定义并返回它，便于用例再把 is_active 关掉。
func deployTicketDefinition(t *testing.T, client *ent.Client, ctx context.Context, tenantID int, key string) *ent.ProcessDefinition {
	t.Helper()
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID(fmt.Sprintf("dep-ticket-%s-%d", key, tenantID)).
		SetDeploymentName("deploy " + key).
		SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey(key).SetName("流程 " + key).SetBpmnXML([]byte("<definitions/>")).
		SetDeploymentID(dep.ID).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	return def
}

func addTicketBinding(t *testing.T, client *ent.Client, ctx context.Context, tenantID int, key string) {
	t.Helper()
	_, err := client.ProcessBinding.Create().
		SetBusinessType("ticket").
		SetBusinessSubType("incident").
		SetProcessDefinitionKey(key).
		SetPriority(10).
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
}

func TestWorkflowStart_TicketRouting(t *testing.T) {
	tests := []struct {
		name        string
		priority    string
		payloadKey  string
		deployKey   bool
		disableKey  bool
		routeKey    string
		deployRoute bool
		extraDeploy []string
		wantKey     string
	}{
		{
			name: "分类绑定的流程已部署_按绑定启动", priority: "medium",
			payloadKey: "cat_bound_flow", deployKey: true,
			wantKey: "cat_bound_flow",
		},
		{
			name: "绑定的流程未部署_回退内置并继续", priority: "medium",
			payloadKey: "ghost_ticket_flow",
			wantKey:    "ticket_general_flow",
		},
		{
			name: "绑定的流程已停用_回退内置", priority: "medium",
			payloadKey: "inactive_ticket_flow", deployKey: true, disableKey: true,
			wantKey: "ticket_general_flow",
		},
		{
			name: "绑定失效但路由表命中已部署流程_用路由表", priority: "medium",
			payloadKey: "ghost_ticket_flow", routeKey: "routed_ticket_flow", deployRoute: true,
			wantKey: "routed_ticket_flow",
		},
		{
			name: "无绑定_高优先级走内置加急流程", priority: "high",
			extraDeploy: []string{"ticket_urgent_flow"},
			wantKey:     "ticket_urgent_flow",
		},
		{
			// 2026-10-24 生产实测：ProcessResolver.ResolveWithPriority 对 high/urgent
			// 兜底返回 ticket_urgent_flow，而内置模板里从来没有这个 key。若不回退，
			// workflow.start 重试 8 次进 dead_letter，工单永远没有流程实例。
			name: "无绑定_高优先级但加急流程未部署_回退通用流程", priority: "high",
			wantKey: "ticket_general_flow",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, ctx, tenantID, requesterID := newTicketRouteFixture(t)

			for _, key := range tc.extraDeploy {
				deployTicketDefinition(t, client, ctx, tenantID, key)
			}
			if tc.deployKey {
				def := deployTicketDefinition(t, client, ctx, tenantID, tc.payloadKey)
				if tc.disableKey {
					_, err := client.ProcessDefinition.UpdateOneID(def.ID).SetIsActive(false).Save(ctx)
					require.NoError(t, err)
				}
			}
			if tc.deployRoute {
				deployTicketDefinition(t, client, ctx, tenantID, tc.routeKey)
			}
			if tc.routeKey != "" {
				addTicketBinding(t, client, ctx, tenantID, tc.routeKey)
			}

			tkt := createTicketForRouteTest(t, client, ctx, tenantID, requesterID, tc.priority)
			payload := map[string]interface{}{"businessType": "ticket", "businessId": tkt.ID}
			if tc.payloadKey != "" {
				payload["workflowDefinitionKey"] = tc.payloadKey
			}
			cmd := createTicketRouteCommand(t, client, ctx, tkt.ID, tenantID, payload)

			trigger := &fakeWorkflowTrigger{}
			handler := NewWorkflowStartCommandHandler(client, trigger, NewProcessResolver(client, nil), zaptest.NewLogger(t).Sugar())

			require.NoError(t, handler.Handle(ctx, cmd))
			require.Len(t, trigger.reqs, 1)
			assert.Equal(t, tc.wantKey, trigger.reqs[0].ProcessDefinitionKey)
			assert.Equal(t, dto.BusinessTypeTicket, trigger.reqs[0].BusinessType)
			assert.Equal(t, tenantID, trigger.reqs[0].TenantID)
		})
	}
}

// TestWorkflowStart_TicketRouting_TenantIsolation 他租户部署了同名流程时，
// 本租户必须视为不存在并回落，避免跨租户流程定义被间接引用。
func TestWorkflowStart_TicketRouting_TenantIsolation(t *testing.T) {
	client, ctx := setupChangeWorkflowTest(t)
	tenantA, err := createRouteTestTenant(ctx, client, fmt.Sprintf("iso-a-%d", nextRouteTestSeq()))
	require.NoError(t, err)
	tenantB, err := createRouteTestTenant(ctx, client, fmt.Sprintf("iso-b-%d", nextRouteTestSeq()))
	require.NoError(t, err)

	deployTicketDefinition(t, client, ctx, tenantB.ID, "shared_name_flow")
	// 租户 A 的内置流程是生产基线；隔离语义是「不命中 B 的同名流程」。
	deployTicketDefinition(t, client, ctx, tenantA.ID, "ticket_general_flow")

	tkt := createTicketForRouteTest(t, client, ctx, tenantA.ID, createRouteUser(t, client, ctx, tenantA.ID), "medium")
	cmd := createTicketRouteCommand(t, client, ctx, tkt.ID, tenantA.ID, map[string]interface{}{
		"businessType": "ticket", "businessId": tkt.ID, "workflowDefinitionKey": "shared_name_flow",
	})

	trigger := &fakeWorkflowTrigger{}
	handler := NewWorkflowStartCommandHandler(client, trigger, NewProcessResolver(client, nil), zaptest.NewLogger(t).Sugar())
	require.NoError(t, handler.Handle(ctx, cmd))

	require.Len(t, trigger.reqs, 1)
	assert.Equal(t, "ticket_general_flow", trigger.reqs[0].ProcessDefinitionKey)
}

// TestWorkflowStart_NoDefinitionAnywhere_FailsActionably 锁死「一个可用定义都没有」
// 时的失败语义：必须报错（而不是把不存在的 key 交给触发器后反复重试），
// 且错误里要能直接看出租户与候选 key——死信是运维唯一能看到的东西。
func TestWorkflowStart_NoDefinitionAnywhere_FailsActionably(t *testing.T) {
	client, ctx := setupChangeWorkflowTest(t)
	tenant, err := createRouteTestTenant(ctx, client, fmt.Sprintf("empty-%d", nextRouteTestSeq()))
	require.NoError(t, err)

	tkt := createTicketForRouteTest(t, client, ctx, tenant.ID, createRouteUser(t, client, ctx, tenant.ID), "high")
	cmd := createTicketRouteCommand(t, client, ctx, tkt.ID, tenant.ID, map[string]interface{}{
		"businessType": "ticket", "businessId": tkt.ID,
	})

	trigger := &fakeWorkflowTrigger{}
	handler := NewWorkflowStartCommandHandler(client, trigger, NewProcessResolver(client, nil), zaptest.NewLogger(t).Sugar())

	err = handler.Handle(ctx, cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("tenant %d", tenant.ID))
	assert.Contains(t, err.Error(), "ticket_general_flow")
	assert.Contains(t, err.Error(), "process_bindings")
	assert.Empty(t, trigger.reqs, "定义缺失时不得把不存在的 key 交给触发器")
}
