package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDashboardOverview_DBFailureReturnsErrorNotFakeZeros 锁定「数据库不可用时
// 仪表盘总览必须报错，而不是降级成全 0 的假成功」。
//
// SLA 合规块此前在 GetSLAComplianceData 出错时吞错并填 &SLAComplianceData{}：
// 0% 合规率会被误读成「全部超时」，与「无工单」「采集失败」无法区分。现已改为
// 与工单趋势/事件分布/满意度同口径上抛。
//
// 说明：SLA 块唯一的传播性错误是工单计数查询失败，而这与更早的 KPI 计数共用同一
// 健康客户端——健康库下该分支不会被独立触发（属纵深防御）。本用例用关闭的客户端
// 制造数据库级故障，锁定总览「故障即报错、不伪造零值」的整体契约。
func TestDashboardOverview_DBFailureReturnsErrorNotFakeZeros(t *testing.T) {
	client, svc, ctx := setupDashboardTest(t)
	require.NoError(t, client.Close())

	viewer := DashboardViewer{UserID: 1, Role: "manager", ScopeAll: true}
	overview, err := svc.GetDashboardOverview(ctx, 1, viewer)
	require.Error(t, err, "数据库不可用时必须返回错误，不能降级成全 0 的假成功")
	require.Nil(t, overview)
}
