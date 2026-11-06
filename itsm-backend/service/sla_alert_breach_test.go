package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/slaalerthistory"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// SLA 超时后告警缺失的回归锁。
//
// 缺陷（2026-11-05 白盒审查 P0-2）：CheckAndTriggerAlerts 只在
// `timeRemaining > 0` 时评估规则，TriggerSLAWarning 在 `percentage <= 0` 时直接
// 返回。而 EscalationService 唯一的升级数据源就是 sla_alert_history 里
// `resolved_at IS NULL` 的行——于是**已经超时的工单永远没有告警行，也就永远
// 不升级**，升级矩阵对最该生效的场景完全失效。
//
// 这里锁住三件事：
//  1. 超时后仍会落一条告警（升级链路有数据源）；
//  2. 超时告警的 actual_percentage 是 0，不是负数；
//  3. 多档阈值（80/50/20）在超时后只补发最紧急的一档，不会一次刷三条。

type breachFixture struct {
	client     *ent.Client
	svc        *SLAAlertService
	tenantID   int
	ticketID   int
	stateID    int
	definition *ent.SLADefinition
}

func newBreachedTicketFixture(t *testing.T, suffix string) breachFixture {
	t.Helper()
	client, _, ctx := setupSLAMonitorTest(t)
	t.Cleanup(func() { client.Close() })

	tn, err := createSLATestTenant(ctx, client, suffix)
	require.NoError(t, err)
	user, err := createSLATestUser(ctx, client, tn.ID, suffix)
	require.NoError(t, err)
	def, err := createSLATestDefinition(ctx, client, tn.ID, "SLA "+suffix)
	require.NoError(t, err)

	// 4 小时前创建、deadline 已过 2 小时：总时长为正、剩余为负，正是超时形态。
	ticketEntity, err := client.Ticket.Create().
		SetTitle("超时工单 " + suffix).
		SetTicketNumber("TKT-BREACH-" + suffix).
		SetStatus("open").
		SetPriority("high").
		SetType("incident").
		SetTenantID(tn.ID).
		SetRequesterID(user.ID).
		SetCreatedAt(time.Now().Add(-4 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	state, err := client.SLAState.Create().
		SetTenantID(tn.ID).
		SetAggregateType("ticket").
		SetAggregateID(ticketEntity.ID).
		SetSLADefinitionID(def.ID).
		SetStatus("active").
		SetResponseDeadline(time.Now().Add(-2 * time.Hour)).
		SetResolutionDeadline(time.Now().Add(-1 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	return breachFixture{
		client:     client,
		svc:        NewSLAAlertService(client, zaptest.NewLogger(t).Sugar()),
		tenantID:   tn.ID,
		ticketID:   ticketEntity.ID,
		stateID:    state.ID,
		definition: def,
	}
}

func (f breachFixture) createRule(t *testing.T, threshold int) *ent.SLAAlertRule {
	t.Helper()
	rule, err := f.client.SLAAlertRule.Create().
		SetName("超时规则-" + strconv.Itoa(threshold)).
		SetSLADefinitionID(f.definition.ID).
		SetAlertLevel("warning").
		SetThresholdPercentage(threshold).
		SetEscalationEnabled(true).
		SetIsActive(true).
		SetTenantID(f.tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return rule
}

func TestCheckAndTriggerAlerts_FiresAfterDeadlinePassed(t *testing.T) {
	ctx := context.Background()
	f := newBreachedTicketFixture(t, "breach-basic")
	rule := f.createRule(t, 80)

	triggered, err := f.svc.CheckAndTriggerAlerts(ctx, f.ticketID, f.tenantID)
	require.NoError(t, err)
	require.True(t, triggered, "超时后必须建告警，否则升级链路没有数据源")

	alerts, err := f.client.SLAAlertHistory.Query().
		Where(slaalerthistory.TicketIDEQ(f.ticketID)).
		All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, alerts, "sla_alert_history 必须有记录")

	for _, alert := range alerts {
		require.Equal(t, 0.0, alert.ActualPercentage, "超时按 0%% 记录，不写负数")
	}

	// 升级服务的数据源口径：同一规则下 resolved_at 为空的行。
	pending, err := f.client.SLAAlertHistory.Query().
		Where(
			slaalerthistory.AlertRuleIDEQ(rule.ID),
			slaalerthistory.ResolvedAtIsNil(),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, pending, "升级服务必须能查到这条待升级告警")
}

func TestCheckAndTriggerAlerts_BreachUsesOnlyMostUrgentRule(t *testing.T) {
	ctx := context.Background()
	f := newBreachedTicketFixture(t, "breach-multi")
	f.createRule(t, 80)
	f.createRule(t, 50)
	f.createRule(t, 20)

	_, err := f.svc.CheckAndTriggerAlerts(ctx, f.ticketID, f.tenantID)
	require.NoError(t, err)

	alerts, err := f.client.SLAAlertHistory.Query().
		Where(slaalerthistory.TicketIDEQ(f.ticketID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, alerts, 1, "超时后只补发最紧急（阈值最低）的一档，避免一次刷三条升级通知")
	require.Equal(t, 20, alerts[0].ThresholdPercentage)
}

func TestCheckAndTriggerAlerts_StillSilentWhenTimeRemains(t *testing.T) {
	ctx := context.Background()
	f := newBreachedTicketFixture(t, "breach-future")
	rule := f.createRule(t, 20)

	// deadline 推到未来 6 小时：总时长 10h、剩余 6h → 60%，远高于 20% 阈值。
	_, err := f.client.SLAState.UpdateOneID(f.stateID).
		SetResponseDeadline(time.Now().Add(6 * time.Hour)).
		SetResolutionDeadline(time.Now().Add(6 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	triggered, err := f.svc.CheckAndTriggerAlerts(ctx, f.ticketID, f.tenantID)
	require.NoError(t, err)
	require.False(t, triggered, "剩余时间充足时不应告警（确认修复没有放宽成误报）")

	count, err := f.client.SLAAlertHistory.Query().
		Where(slaalerthistory.AlertRuleIDEQ(rule.ID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
