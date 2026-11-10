package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
)

// 报表/导出读面回归锁（2026-11-05 扩展白盒审查 R1 / R7 / R9）。
//
// 缺陷：AnalyticsService.GetDeepAnalytics、PredictionService.GetTrendPrediction
// 与 SLAForecastSkill.Execute 的工单查询只有 TenantIDEQ —— 非管理角色可以读到
// 并导出全租户工单量/分布/满意度/时长，与工作台「本人口径」直接矛盾；分母还
// 含已软删工单；且 `.All(ctx)` 无窗口/行数上限，一次 100 年 timeRange 就能把
// 整张 tickets 表拉进内存（handler 的 60s 超时回收不了已分配内存）。
//
// 本文件锁住四件事：①行级范围；②软删不入分母；③窗口上限显式 400；
// ④导出与查询共用同一个 viewer（不存在「页面看本人、导出看全租户」）。

type reportScopeFixture struct {
	client   *ent.Client
	ctx      context.Context
	tenantID int
	viewerID int
	otherID  int
	created  time.Time
}

func newReportScopeFixture(t *testing.T) *reportScopeFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:report_scope_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	tenant := client.Tenant.Create().SetName("报表口径").SetCode("report-scope").
		SetDomain("report-scope.example.com").SetStatus("active").SaveX(ctx)
	viewer := client.User.Create().SetUsername("report-scope-viewer").
		SetEmail("viewer@report-scope.example.com").SetName("Viewer").
		SetPasswordHash("h").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).SaveX(ctx)
	other := client.User.Create().SetUsername("report-scope-other").
		SetEmail("other@report-scope.example.com").SetName("Other").
		SetPasswordHash("h").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).SaveX(ctx)

	return &reportScopeFixture{
		client: client, ctx: ctx, tenantID: tenant.ID,
		viewerID: viewer.ID, otherID: other.ID,
		created: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}
}

func (f *reportScopeFixture) seed(t *testing.T, number string, requesterID, assigneeID int, deleted bool) {
	t.Helper()
	b := f.client.Ticket.Create().
		SetTicketNumber(number).
		SetTitle("报表-" + number).
		SetDescription("正文-" + number).
		SetType("incident").
		SetPriority("high").
		SetStatus("open").
		SetRequesterID(requesterID).
		SetTenantID(f.tenantID).
		SetCreatedAt(f.created)
	if assigneeID > 0 {
		b = b.SetAssigneeID(assigneeID)
	}
	if deleted {
		b = b.SetDeletedAt(time.Now())
	}
	b.SaveX(f.ctx)
}

// 只看得到「本人创建 + 分配给本人」的 2 张；管理员可见 3 张（软删的永远不算）。
func (f *reportScopeFixture) seedAll(t *testing.T) {
	t.Helper()
	f.seed(t, "RPT-OWN-001", f.viewerID, 0, false)
	f.seed(t, "RPT-ASSIGNED-002", f.otherID, f.viewerID, false)
	f.seed(t, "RPT-OTHER-003", f.otherID, 0, false)
	f.seed(t, "RPT-DELETED-004", f.viewerID, 0, true)
}

func analyticsReportRequest() *dto.DeepAnalyticsRequest {
	return &dto.DeepAnalyticsRequest{
		TimeRange:  []string{"2024-01-01", "2024-01-31"},
		Dimensions: []string{"status"},
		Metrics:    []string{"count"},
	}
}

func TestAnalyticsReport_ViewerScopeAndSoftDelete(t *testing.T) {
	f := newReportScopeFixture(t)
	f.seedAll(t)
	svc := NewAnalyticsService(f.client, zaptest.NewLogger(t).Sugar())

	own, err := svc.GetDeepAnalytics(f.ctx, analyticsReportRequest(), f.tenantID,
		DashboardViewer{UserID: f.viewerID, Role: "agent"})
	require.NoError(t, err)
	require.Equal(t, 2, own.Summary.Total, "普通角色只能统计本人创建或受理的工单")
	require.Equal(t, "own", own.Scope)

	all, err := svc.GetDeepAnalytics(f.ctx, analyticsReportRequest(), f.tenantID,
		DashboardViewer{ScopeAll: true})
	require.NoError(t, err)
	require.Equal(t, 3, all.Summary.Total, "管理角色可见全租户，但软删工单仍不得入分母")
	require.Equal(t, "tenant", all.Scope)

	// 身份缺失是 fail-closed：UID=0 + 非管理角色 => 空集，不是全租户兜底。
	anon, err := svc.GetDeepAnalytics(f.ctx, analyticsReportRequest(), f.tenantID, DashboardViewer{})
	require.NoError(t, err)
	require.Equal(t, 0, anon.Summary.Total)
}

func TestAnalyticsReport_WindowBoundsAreRejectedWith400(t *testing.T) {
	f := newReportScopeFixture(t)
	svc := NewAnalyticsService(f.client, zaptest.NewLogger(t).Sugar())

	for name, tr := range map[string][]string{
		"超长窗口":   {"2024-01-01", "2025-06-01"},
		"结束早于起始": {"2024-02-01", "2024-01-01"},
	} {
		t.Run(name, func(t *testing.T) {
			req := analyticsReportRequest()
			req.TimeRange = tr
			_, err := svc.GetDeepAnalytics(f.ctx, req, f.tenantID, DashboardViewer{ScopeAll: true})
			require.Error(t, err)
			var appErr *common.AppError
			require.True(t, errors.As(err, &appErr), "参数错误必须是 AppError，不能是裸 error 被兜底成 500")
			require.Equal(t, 400, appErr.HTTPStatus)
		})
	}
}

func TestPredictionReport_ViewerScopeAndSoftDelete(t *testing.T) {
	f := newReportScopeFixture(t)
	f.seedAll(t)
	svc := NewPredictionService(f.client, zaptest.NewLogger(t).Sugar())
	req := &dto.TrendPredictionRequest{
		PredictionType: "volume",
		TimeRange:      []string{"2024-01-01", "2024-01-31"},
	}

	own, err := svc.GetTrendPrediction(f.ctx, req, f.tenantID,
		DashboardViewer{UserID: f.viewerID, Role: "agent"})
	require.NoError(t, err)
	require.Equal(t, "own", own.Scope)
	require.NotEmpty(t, own.Predictions)
	// 2 张可见工单同在 2024-01 → 月均 2；3 张（含他人工单）→ 月均 3。
	require.Equal(t, float64(2), own.Predictions[0].PredictedValue)

	all, err := svc.GetTrendPrediction(f.ctx, req, f.tenantID, DashboardViewer{ScopeAll: true})
	require.NoError(t, err)
	require.Equal(t, "tenant", all.Scope)
	require.Equal(t, float64(3), all.Predictions[0].PredictedValue,
		"软删工单不得把月均从 3 抬到 4")
}

func TestReportExports_InheritViewerScope(t *testing.T) {
	f := newReportScopeFixture(t)
	f.seedAll(t)

	analyticsSvc := NewAnalyticsService(f.client, zaptest.NewLogger(t).Sugar())
	csvOwn, _, err := analyticsSvc.ExportAnalytics(f.ctx, analyticsReportRequest(), "csv", f.tenantID,
		DashboardViewer{UserID: f.viewerID, Role: "agent"})
	require.NoError(t, err)
	require.Contains(t, string(csvOwn), "Total,2", "导出必须与页面同一行级口径")

	csvAll, _, err := analyticsSvc.ExportAnalytics(f.ctx, analyticsReportRequest(), "csv", f.tenantID,
		DashboardViewer{ScopeAll: true})
	require.NoError(t, err)
	require.Contains(t, string(csvAll), "Total,3")

	predSvc := NewPredictionService(f.client, zaptest.NewLogger(t).Sugar())
	predReq := &dto.TrendPredictionRequest{PredictionType: "volume", TimeRange: []string{"2024-01-01", "2024-01-31"}}
	_, _, err = predSvc.ExportPredictionReport(f.ctx, predReq, "csv", f.tenantID,
		DashboardViewer{UserID: f.viewerID, Role: "agent"})
	require.NoError(t, err)
}
