package service

import (
	"testing"

	"itsm-backend/ent/ticketnotification"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestEscalateToLevelByMatrix_ClaimOnce 两副本读同一告警快照并发推进同一级别时，
// 条件更新只有一个副本成功；落败方返回 ErrEscalationLevelClaimed，
// 且矩阵升级通知没有 outbox 幂等键兜底，重复通知只能靠这条 CAS 拦截。
func TestEscalateToLevelByMatrix_ClaimOnce(t *testing.T) {
	client, svc, ctx := setupEscalationTest(t)
	defer client.Close()

	svc.SetNotificationService(NewTicketNotificationService(client, zaptest.NewLogger(t).Sugar()))

	tenant, err := createEscalationTestTenant(ctx, client, "claim")
	require.NoError(t, err)
	requester, err := createEscalationTestUser(ctx, client, tenant.ID, "req")
	require.NoError(t, err)
	admin, err := client.User.Create().
		SetUsername("admin-claim").
		SetEmail("admin-claim@example.com").
		SetName("Admin Claim").
		SetPasswordHash("hashedpassword").
		SetRole("admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tk, err := client.Ticket.Create().
		SetTitle("Claim Once Ticket").
		SetDescription("escalation claim-once regression").
		SetPriority("high").
		SetType("incident").
		SetStatus("open").
		SetTicketNumber("TKT-CLAIM-001").
		SetTenantID(tenant.ID).
		SetRequesterID(requester.ID).
		Save(ctx)
	require.NoError(t, err)

	slaDef, err := client.SLADefinition.Create().
		SetName("SLA Claim").
		SetPriority("high").
		SetResponseTime(60).
		SetResolutionTime(240).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	rule, err := client.SLAAlertRule.Create().
		SetName("Claim Rule").
		SetSLADefinitionID(slaDef.ID).
		SetAlertLevel("warning").
		SetThresholdPercentage(70).
		SetIsActive(true).
		SetEscalationEnabled(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 两副本共同读到的告警快照：escalation_level 仍为 0
	alert, err := client.SLAAlertHistory.Create().
		SetTicketID(tk.ID).
		SetTicketNumber(tk.TicketNumber).
		SetTicketTitle(tk.Title).
		SetAlertRuleID(rule.ID).
		SetAlertRuleName(rule.Name).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	level := &EscalationLevel{
		Level:         1,
		AfterMinutes:  15,
		NotifyUserIDs: []int{admin.ID},
		Description:   "L1 通知管理员",
	}

	// 副本 A 抢到推进权，负责通知
	require.NoError(t, svc.escalateToLevelByMatrix(ctx, alert, rule, level, tenant.ID, "high"))
	stored, err := client.SLAAlertHistory.Get(ctx, alert.ID)
	require.NoError(t, err)
	require.Equal(t, 1, stored.EscalationLevel)

	// 副本 B 拿同一快照抢锁：条件更新 0 行命中，必须落败且不产生第二条通知
	err = svc.escalateToLevelByMatrix(ctx, alert, rule, level, tenant.ID, "high")
	require.ErrorIs(t, err, ErrEscalationLevelClaimed)

	stored, err = client.SLAAlertHistory.Get(ctx, alert.ID)
	require.NoError(t, err)
	require.Equal(t, 1, stored.EscalationLevel, "落败副本不得改动级别")

	sent, err := client.TicketNotification.Query().
		Where(
			ticketnotification.TenantID(tenant.ID),
			ticketnotification.TypeEQ("sla_escalated"),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, sent, "同一级别只允许一条升级通知，双副本不得重复投递")
}

// TestEscalationService_ProcessEscalations_AggregatesErrors:
// 三个子步骤的失败必须聚合为非 nil 错误返回；只打日志、整体返回 nil 时，
// ticker 无法感知扫描失败。
func TestEscalationService_ProcessEscalations_AggregatesErrors(t *testing.T) {
	client, svc, ctx := setupEscalationTest(t)

	tenant, err := createEscalationTestTenant(ctx, client, "agg")
	require.NoError(t, err)

	client.Close() // 模拟 DB 不可用：三个子步骤全部失败

	err = svc.ProcessEscalations(ctx, tenant.ID)
	require.Error(t, err, "子步骤失败必须聚合返回，不得吞成 nil")
	require.Contains(t, err.Error(), "failed to query alert rules")
	require.Contains(t, err.Error(), "failed to query long pending tickets")
	require.Contains(t, err.Error(), "failed to query unassigned tickets")
}
