package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeQualityDashboard 工作台概览源的假实现：不触发 DB，锁定聚合器自身的
// 版本/窗口/降级契约。KPI 块的真实计算语义由 dashboard_service_test 覆盖，
// 这里只验证聚合器的组装与显式降级。
type fakeQualityDashboard struct {
	overview *DashboardOverviewData
	trend    []TicketTrendData
	ovErr    error
}

func (f *fakeQualityDashboard) GetDashboardOverview(ctx context.Context, tenantID int, viewer DashboardViewer) (*DashboardOverviewData, error) {
	if f.ovErr != nil {
		return nil, f.ovErr
	}
	return f.overview, nil
}

func (f *fakeQualityDashboard) GetTicketTrend(ctx context.Context, tenantID, days int) ([]TicketTrendData, error) {
	return f.trend, nil
}

func newFakeQualityDashboard() *fakeQualityDashboard {
	return &fakeQualityDashboard{
		overview: &DashboardOverviewData{
			KPIMetrics:           []KPIMetricData{{ID: "total-tickets", Value: 5, Comparable: true, Scope: "tenant"}},
			SLAData:              &SLAComplianceData{ComplianceRate: 0.95, TotalTickets: 100, CompliantTickets: 95},
			IncidentDistribution: []IncidentDistributionData{{Category: "network", Count: 3}},
			SatisfactionData:     []SatisfactionData{{Month: "2026-10", Rating: 4.5, Responses: 12}},
		},
		trend: []TicketTrendData{{Date: "2026-10-23", Open: 2, Resolved: 1}},
	}
}

func qualityViewer() DashboardViewer {
	return DashboardViewer{UserID: 7, Role: "admin", ScopeAll: true}
}

// TestServiceQualityReport_MetricVersionContract 锁定口径版本契约：
// 总体版本 + 各块版本必须存在——下游（管理层月报、外部对账脚本）以版本号
// 感知「同样的数字现在含义不同了」。这是报表域收口的核心新契约。
func TestServiceQualityReport_MetricVersionContract(t *testing.T) {
	svc := NewServiceQualityReportService(newFakeQualityDashboard(), nil)

	report, err := svc.Build(context.Background(), 1, qualityViewer(), 30)
	require.NoError(t, err)

	assert.Equal(t, ServiceQualityMetricVersion, report.MetricVersion)
	require.NotNil(t, report.MetricVersions)
	for _, block := range []string{"kpi", "sla", "incidentDistribution", "satisfaction", "ticketTrend", "ticketStats"} {
		assert.NotEmpty(t, report.MetricVersions[block], "block %s 缺少口径版本", block)
	}

	// 窗口契约：from/to 覆盖请求天数；scope 显式标注行级口径。
	assert.Equal(t, 30, report.Window.Days)
	assert.True(t, report.Window.To.After(report.Window.From))
	assert.Equal(t, "tenant", report.Scope)

	// 聚合必须透传各块数据，不得静默丢弃。
	require.Len(t, report.KPI, 1)
	assert.Equal(t, 0.95, report.SLA.ComplianceRate)
	assert.Len(t, report.IncidentDistribution, 1)
	assert.Len(t, report.Satisfaction, 1)
	assert.Len(t, report.TicketTrend, 1)

	// analytics 未装配 → ticketStats 块显式缺省（能力降级可见）。
	assert.Nil(t, report.TicketStats)
}

func TestServiceQualityReport_WindowClamp(t *testing.T) {
	svc := NewServiceQualityReportService(newFakeQualityDashboard(), nil)

	report, err := svc.Build(context.Background(), 1, qualityViewer(), 0)
	require.NoError(t, err)
	assert.Equal(t, 30, report.Window.Days, "非法窗口回退默认 30 天")

	report, err = svc.Build(context.Background(), 1, qualityViewer(), 3650)
	require.NoError(t, err)
	assert.Equal(t, MaxQualityWindowDays, report.Window.Days, "超上限窗口钳制到 90 天")

	report, err = svc.Build(context.Background(), 1, qualityViewer(), 7)
	require.NoError(t, err)
	assert.Equal(t, 7, report.Window.Days)
}

func TestServiceQualityReport_DashboardMissingFails(t *testing.T) {
	svc := NewServiceQualityReportService(nil, nil)
	_, err := svc.Build(context.Background(), 1, qualityViewer(), 30)
	require.Error(t, err, "dashboard 未装配必须显式报错，不得返回空报表")
}

// TestServiceQualityReport_OverviewFailurePropagates 工作台块失败必须整体失败：
// 管理视图宁可报错也不许给「半张纸」——管理者无法区分"没有事件"和"查不到事件"。
func TestServiceQualityReport_OverviewFailurePropagates(t *testing.T) {
	dash := newFakeQualityDashboard()
	dash.ovErr = errors.New("kpi query down")
	svc := NewServiceQualityReportService(dash, nil)

	_, err := svc.Build(context.Background(), 1, qualityViewer(), 30)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build overview block")
}

// TestServiceQualityReport_EmptyBlocksAreArraysNotNil 空数据块必须落成空 JSON
// 数组——序列化成 null 会让前端 `.map()` 直接崩。
func TestServiceQualityReport_EmptyBlocksAreArraysNotNil(t *testing.T) {
	dash := &fakeQualityDashboard{overview: &DashboardOverviewData{}}
	svc := NewServiceQualityReportService(dash, nil)

	report, err := svc.Build(context.Background(), 1, qualityViewer(), 30)
	require.NoError(t, err)
	assert.NotNil(t, report.KPI)
	assert.NotNil(t, report.IncidentDistribution)
	assert.NotNil(t, report.Satisfaction)
	assert.NotNil(t, report.TicketTrend)
}
