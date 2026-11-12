package service

import (
	"context"
	"strconv"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/ticketautomationrule"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupAutomationRuleTest(t *testing.T) (*ent.Client, *TicketAutomationRuleService, context.Context) {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		"file:automation_rule_test_"+t.Name()+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewTicketAutomationRuleService(client, logger)
	ctx := context.Background()
	return client, svc, ctx
}

// TestAutomationSetStatusRespectsStateMachine 自动化 set_status 必须走权威状态机：
// 非法迁移拒绝、resolved 必须带解决方案、同状态幂等跳过、合法迁移 CAS 推进版本并留审计。
func TestAutomationSetStatusRespectsStateMachine(t *testing.T) {
	client, svc, ctx := setupAutomationRuleTest(t)
	defer client.Close()
	tn, err := createIncidentTestTenant(ctx, client, "autost")
	require.NoError(t, err)
	user, err := createIncidentTestUser(ctx, client, tn.ID, "autost")
	require.NoError(t, err)

	mkTicket := func(t *testing.T, status string, resolution string) *ent.Ticket {
		t.Helper()
		create := client.Ticket.Create().
			SetTitle("自动化状态机工单").
			SetTicketNumber("TKT-AUTO-ST-" + status + "-" + t.Name()).
			SetStatus(status).
			SetPriority("medium").
			SetType("incident").
			SetTenantID(tn.ID).
			SetRequesterID(user.ID)
		if resolution != "" {
			create = create.SetResolution(resolution)
		}
		tk, err := create.Save(ctx)
		require.NoError(t, err)
		return tk
	}
	mkRule := func(t *testing.T, name string) *ent.TicketAutomationRule {
		t.Helper()
		rule, err := client.TicketAutomationRule.Create().
			SetName(name).
			SetPriority(10).
			SetConditions([]map[string]interface{}{}).
			SetActions([]map[string]interface{}{
				{"type": "set_status", "status": "in_progress"},
			}).
			SetIsActive(true).
			SetCreatedBy(user.ID).
			SetTenantID(tn.ID).
			Save(ctx)
		require.NoError(t, err)
		return rule
	}
	countStatusAudit := func(t *testing.T, ticketID int) int {
		t.Helper()
		n, err := client.AuditLog.Query().
			Where(
				auditlog.TenantIDEQ(tn.ID),
				auditlog.ActionEQ("automation_set_status"),
				auditlog.ResourceEQ("ticket:"+strconv.Itoa(ticketID)),
			).
			Count(ctx)
		require.NoError(t, err)
		return n
	}
	mkRule(t, "状态机规则")

	cases := []struct {
		name            string
		startStatus     string
		resolution      string
		wantStatus      string
		wantVersionBump bool
		wantAudit       int
	}{
		{"open→in_progress 合法", "open", "", "in_progress", true, 1},
		{"closed→in_progress 拒绝", "closed", "", "closed", false, 0},
		{"cancelled→in_progress 拒绝", "cancelled", "", "cancelled", false, 0},
		{"resolved 缺解决方案 →in_progress 仍合法（门槛只挡进入 resolved）", "resolved", "", "in_progress", true, 1},
		{"resolved 有解决方案 →in_progress 合法", "resolved", "已解决", "in_progress", true, 1},
		{"in_progress→in_progress 幂等跳过", "in_progress", "", "in_progress", false, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tk := mkTicket(t, tc.startStatus, tc.resolution)
			require.NoError(t, svc.ExecuteRulesForTicket(ctx, tk.ID, tn.ID))

			after, err := client.Ticket.Get(ctx, tk.ID)
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, after.Status)
			if tc.wantVersionBump {
				assert.Equal(t, tk.Version+1, after.Version, "合法迁移必须 CAS 推进版本")
			} else {
				assert.Equal(t, tk.Version, after.Version, "拒绝/幂等不得改写版本")
			}
			assert.Equal(t, tc.wantAudit, countStatusAudit(t, tk.ID))
		})
	}
}

// TestAutomationSetStatusCrossRuleReload 规则执行后必须重载工单：
// 规则 A 把状态推进到 in_progress 后，规则 B 的 status 条件必须能基于最新状态匹配。
func TestAutomationSetStatusCrossRuleReload(t *testing.T) {
	client, svc, ctx := setupAutomationRuleTest(t)
	defer client.Close()
	tn, err := createIncidentTestTenant(ctx, client, "autoreload")
	require.NoError(t, err)
	user, err := createIncidentTestUser(ctx, client, tn.ID, "autoreload")
	require.NoError(t, err)

	tk, err := client.Ticket.Create().
		SetTitle("跨规则重载工单").
		SetTicketNumber("TKT-AUTO-RELOAD").
		SetStatus("open").
		SetPriority("low").
		SetType("incident").
		SetTenantID(tn.ID).
		SetRequesterID(user.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.TicketAutomationRule.Create().
		SetName("A-推进状态").
		SetPriority(20).
		SetConditions([]map[string]interface{}{}).
		SetActions([]map[string]interface{}{
			{"type": "set_status", "status": "in_progress"},
		}).
		SetIsActive(true).
		SetCreatedBy(user.ID).
		SetTenantID(tn.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.TicketAutomationRule.Create().
		SetName("B-依赖A结果").
		SetPriority(10).
		SetConditions([]map[string]interface{}{
			{"field": "status", "operator": "equals", "value": "in_progress"},
		}).
		SetActions([]map[string]interface{}{
			{"type": "set_priority", "priority": "high"},
		}).
		SetIsActive(true).
		SetCreatedBy(user.ID).
		SetTenantID(tn.ID).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, svc.ExecuteRulesForTicket(ctx, tk.ID, tn.ID))

	after, err := client.Ticket.Get(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "in_progress", after.Status)
	assert.Equal(t, "high", after.Priority, "规则 B 必须基于规则 A 重载后的状态匹配")
}

// TestAutomationAssignValidatesTargetTenant 自动化 assign 的目标必须同租户且活跃。
func TestAutomationAssignValidatesTargetTenant(t *testing.T) {
	client, svc, ctx := setupAutomationRuleTest(t)
	defer client.Close()
	tnA, err := createIncidentTestTenant(ctx, client, "autoassA")
	require.NoError(t, err)
	tnB, err := createIncidentTestTenant(ctx, client, "autoassB")
	require.NoError(t, err)
	userA, err := createIncidentTestUser(ctx, client, tnA.ID, "autoassA")
	require.NoError(t, err)
	userB, err := createIncidentTestUser(ctx, client, tnB.ID, "autoassB")
	require.NoError(t, err)

	tk, err := client.Ticket.Create().
		SetTitle("assign 租户校验工单").
		SetTicketNumber("TKT-AUTO-ASSIGN").
		SetStatus("open").
		SetPriority("medium").
		SetType("incident").
		SetTenantID(tnA.ID).
		SetRequesterID(userA.ID).
		Save(ctx)
	require.NoError(t, err)

	mkAssignRule := func(targetID int) *ent.TicketAutomationRule {
		rule, err := client.TicketAutomationRule.Create().
			SetName("assign 规则").
			SetPriority(10).
			SetConditions([]map[string]interface{}{}).
			SetActions([]map[string]interface{}{
				{"type": "assign", "user_id": float64(targetID)},
			}).
			SetIsActive(true).
			SetCreatedBy(userA.ID).
			SetTenantID(tnA.ID).
			Save(ctx)
		require.NoError(t, err)
		return rule
	}

	// 跨租户目标：拒绝，处理人不变。
	ruleCross := mkAssignRule(userB.ID)
	require.NoError(t, svc.ExecuteRulesForTicket(ctx, tk.ID, tnA.ID))
	after, err := client.Ticket.Get(ctx, tk.ID)
	require.NoError(t, err)
	assert.False(t, after.AssigneeID > 0 && after.AssigneeID == userB.ID, "跨租户目标不得被指派")
	_, err = client.TicketAutomationRule.Delete().
		Where(ticketautomationrule.ID(ruleCross.ID), ticketautomationrule.TenantIDEQ(tnA.ID)).
		Exec(ctx)
	require.NoError(t, err)

	// 本租户活跃目标：成功。
	mkAssignRule(userA.ID)
	require.NoError(t, svc.ExecuteRulesForTicket(ctx, tk.ID, tnA.ID))
	after, err = client.Ticket.Get(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, userA.ID, after.AssigneeID, "同租户活跃目标必须被指派")
}

// TestAutomationCommandHandlerEntryAppliesStateMachine 命令入口（worker 消费）同样走状态机：
// 终态工单不得被自动化复活。
func TestAutomationCommandHandlerEntryAppliesStateMachine(t *testing.T) {
	client, svc, ctx := setupAutomationRuleTest(t)
	defer client.Close()
	tn, err := createIncidentTestTenant(ctx, client, "autocmd")
	require.NoError(t, err)
	user, err := createIncidentTestUser(ctx, client, tn.ID, "autocmd")
	require.NoError(t, err)

	tk, err := client.Ticket.Create().
		SetTitle("命令入口状态机工单").
		SetTicketNumber("TKT-AUTO-CMD").
		SetStatus("closed").
		SetPriority("medium").
		SetType("incident").
		SetTenantID(tn.ID).
		SetRequesterID(user.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.TicketAutomationRule.Create().
		SetName("复活规则").
		SetPriority(10).
		SetConditions([]map[string]interface{}{}).
		SetActions([]map[string]interface{}{
			{"type": "set_status", "status": "in_progress"},
		}).
		SetIsActive(true).
		SetCreatedBy(user.ID).
		SetTenantID(tn.ID).
		Save(ctx)
	require.NoError(t, err)

	handler := NewTicketAutomationCommandHandler(svc)
	err = handler.Handle(ctx, &ent.OperationalCommand{
		TenantID:      tn.ID,
		CommandType:   "ticket_automation",
		AggregateType: "ticket",
		AggregateID:   tk.ID,
	})
	require.NoError(t, err, "业务拒绝不冒充命令失败")

	after, err := client.Ticket.Get(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "closed", after.Status, "终态工单不得被自动化复活")
}
