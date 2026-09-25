package service

import (
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// 回归（2026-09-25 假成功收口）：事件/变更指标必须来自真实领域数据。
// 此前 AvgResolutionTime=240 分钟、SuccessRate=95.5% 是写死的模拟值，
// 且两个函数数的是工单而非事件/变更域。
func TestDashboardDomainMetricsAreReal(t *testing.T) {
	client, svc, ctx := setupDashboardTest(t)
	defer client.Close()
	tenant, err := createDashboardTestTenant(ctx, client, "metrics")
	require.NoError(t, err)
	user, err := createDashboardTestUser(ctx, client, tenant.ID, "metrics")
	require.NoError(t, err)
	other, err := createDashboardTestTenant(ctx, client, "empty")
	require.NoError(t, err)

	base := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	// 两个已解决事件：分别用时 60 / 120 分钟 → 平均 90
	client.Incident.Create().SetTitle("resolved-1").SetIncidentNumber("INC-M1").SetStatus("resolved").
		SetReporterID(user.ID).SetTenantID(tenant.ID).SetCreatedAt(base).SetResolvedAt(base.Add(60 * time.Minute)).SaveX(ctx)
	client.Incident.Create().SetTitle("resolved-2").SetIncidentNumber("INC-M2").SetStatus("resolved").
		SetReporterID(user.ID).SetTenantID(tenant.ID).SetCreatedAt(base).SetResolvedAt(base.Add(120 * time.Minute)).SaveX(ctx)
	// 一个未关闭高优先级事件计入高优先级；一个已关闭 critical 不计
	client.Incident.Create().SetTitle("open-high").SetIncidentNumber("INC-M3").SetStatus("open").SetPriority("high").
		SetReporterID(user.ID).SetTenantID(tenant.ID).SaveX(ctx)
	client.Incident.Create().SetTitle("closed-critical").SetIncidentNumber("INC-M4").SetStatus("closed").SetPriority("critical").
		SetReporterID(user.ID).SetTenantID(tenant.ID).SaveX(ctx)

	client.Change.Create().SetTitle("c1").SetChangeNumber("CHG-M1").SetStatus("completed").SetCreatedBy(user.ID).SetTenantID(tenant.ID).SaveX(ctx)
	client.Change.Create().SetTitle("c2").SetChangeNumber("CHG-M2").SetStatus("completed").SetCreatedBy(user.ID).SetTenantID(tenant.ID).SaveX(ctx)
	client.Change.Create().SetTitle("c3").SetChangeNumber("CHG-M3").SetStatus("failed").SetCreatedBy(user.ID).SetTenantID(tenant.ID).SaveX(ctx)
	client.Change.Create().SetTitle("c4").SetChangeNumber("CHG-M4").SetStatus("submitted").SetCreatedBy(user.ID).SetTenantID(tenant.ID).SaveX(ctx)
	client.Change.Create().SetTitle("c5").SetChangeNumber("CHG-M5").SetStatus("draft").SetCreatedBy(user.ID).SetTenantID(tenant.ID).SaveX(ctx)

	incidents, err := svc.getIncidentMetrics(ctx, tenant.ID)
	require.NoError(t, err)
	require.Equal(t, 4, incidents.TotalIncidents)
	require.Equal(t, 1, incidents.HighPriorityCount)
	require.Equal(t, 90, incidents.AvgResolutionTime, "平均解决时长必须按真实数据计算")
	require.NotEqual(t, 240, incidents.AvgResolutionTime, "禁止模拟值")

	changes, err := svc.getChangeMetrics(ctx, tenant.ID)
	require.NoError(t, err)
	require.Equal(t, 5, changes.TotalChanges)
	require.Equal(t, 1, changes.PendingApproval)
	require.InDelta(t, 66.67, changes.SuccessRate, 0.01, "成功率 = completed/(completed+failed)")
	require.NotEqual(t, 95.5, changes.SuccessRate, "禁止模拟值")

	// 空租户：如实为 0，而不是拿模拟值充数
	emptyIncidents, err := svc.getIncidentMetrics(ctx, other.ID)
	require.NoError(t, err)
	require.Equal(t, 0, emptyIncidents.AvgResolutionTime)
	emptyChanges, err := svc.getChangeMetrics(ctx, other.ID)
	require.NoError(t, err)
	require.Equal(t, 0.0, emptyChanges.SuccessRate)
}
