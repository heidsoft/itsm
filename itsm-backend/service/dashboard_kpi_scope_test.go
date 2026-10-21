package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/ticket"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// kpiFixture 给 KPI 测试准备一个 DashboardService：工单计数走 enttest（内存 sqlite），
// 时长/SLA 聚合走 sqlmock，使断言只反映读取者范围与环比口径。
type kpiFixture struct {
	client   *ent.Client
	service  *DashboardService
	ctx      context.Context
	mock     sqlmock.Sqlmock
	tenantID int
}

func newKPIFixture(t *testing.T) *kpiFixture {
	t.Helper()
	client, service, ctx := setupDashboardTest(t)
	t.Cleanup(func() { client.Close() })

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	service.repo = newDashboardRepository(db)
	return &kpiFixture{client: client, service: service, ctx: ctx, mock: mock}
}

// kpi 调用被测生产路径 service.getKPIMetrics —— dashboard_handler 的各 KPI 端点最终走的就是它。
// 聚合期望必须由调用方预先注册。
func (f *kpiFixture) kpi(t *testing.T, viewer DashboardViewer) []KPIMetricData {
	t.Helper()
	metrics, err := f.service.getKPIMetrics(f.ctx, f.tenantID, viewer)
	require.NoError(t, err)
	require.NoError(t, f.mock.ExpectationsWereMet())
	return metrics
}

// zeroAggregates 把本期/上期聚合样本置 0，让工单计数成为唯一变量。
func (f *kpiFixture) zeroAggregates() {
	f.mock.ExpectQuery(`.+`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c", "d", "e", "f"}).AddRow(0, 0, 0, 0, 0, 0))
	f.mock.ExpectQuery(`.+`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"total", "met"}).AddRow(0, 0))
}

func (f *kpiFixture) kpiWithZeroAggregates(t *testing.T, viewer DashboardViewer) []KPIMetricData {
	t.Helper()
	f.zeroAggregates()
	return f.kpi(t, viewer)
}

func newKPIUser(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, suffix string) *ent.User {
	t.Helper()
	u, err := createDashboardTestUser(ctx, client, tenantID, suffix)
	require.NoError(t, err)
	return u
}

// newKPITicket 建一张指定报障人、状态与创建时间的工单。
func newKPITicket(
	t *testing.T, ctx context.Context, client *ent.Client, tenantID, requesterID int,
	status, number string, createdAt time.Time,
) *ent.Ticket {
	t.Helper()
	tk, err := client.Ticket.Create().
		SetTitle("KPI " + number).
		SetDescription("kpi fixture").
		SetStatus(status).
		SetPriority("medium").
		SetTicketNumber(number).
		SetRequesterID(requesterID).
		SetTenantID(tenantID).
		SetCreatedAt(createdAt).
		Save(ctx)
	require.NoError(t, err)
	return tk
}

func kpiByID(t *testing.T, metrics []KPIMetricData, id string) KPIMetricData {
	t.Helper()
	for _, m := range metrics {
		if m.ID == id {
			return m
		}
	}
	t.Fatalf("KPI %s 缺失: %+v", id, metrics)
	return KPIMetricData{}
}

func kpiWindow() (lastMonthStart, thisMonthStart time.Time) {
	now := time.Now()
	thisMonthStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	lastMonth := now.AddDate(0, -1, 0)
	lastMonthStart = time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, time.Local)
	return lastMonthStart, thisMonthStart
}

// TestDashboardService_KPIUsesViewerTicketScope 工作台 KPI 的工单计数必须与工单列表用同一
// 行级范围：坐席只看到本人创建或受理的工单，管理角色看到租户全量。
// 修复前 KPI 只按租户统计，坐席卡片上的数字混进了别人的工单。
func TestDashboardService_KPIUsesViewerTicketScope(t *testing.T) {
	f := newKPIFixture(t)
	client, ctx := f.client, f.ctx
	tenant, err := createDashboardTestTenant(ctx, client, "kpi-scope")
	require.NoError(t, err)
	otherTenant, err := createDashboardTestTenant(ctx, client, "kpi-scope-b")
	require.NoError(t, err)
	f.tenantID = tenant.ID

	mine := newKPIUser(t, ctx, client, tenant.ID, "mine")
	theirs := newKPIUser(t, ctx, client, tenant.ID, "theirs")
	assignee := newKPIUser(t, ctx, client, tenant.ID, "assignee")

	now := time.Now()
	newKPITicket(t, ctx, client, tenant.ID, mine.ID, "open", "SCOPE-1", now)
	newKPITicket(t, ctx, client, tenant.ID, mine.ID, "new", "SCOPE-2", now)
	newKPITicket(t, ctx, client, tenant.ID, theirs.ID, "open", "SCOPE-3", now)
	assigned := newKPITicket(t, ctx, client, tenant.ID, theirs.ID, "assigned", "SCOPE-4", now)
	_, err = client.Ticket.UpdateOne(assigned).SetAssigneeID(assignee.ID).Save(ctx)
	require.NoError(t, err)
	// 跨租户工单绝不能进入本租户任何口径
	newKPITicket(t, ctx, client, otherTenant.ID, mine.ID, "open", "SCOPE-X", now)

	tenantTotal, err := client.Ticket.Query().Where(ticket.TenantID(tenant.ID)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, tenantTotal, "夹具数量要对得上")

	agent := f.kpiWithZeroAggregates(t, DashboardViewer{UserID: mine.ID, Role: "agent"})
	total := kpiByID(t, agent, "total-tickets")
	assert.Equal(t, 2.0, total.Value, "坐席只应看到本人创建的 2 张")
	assert.Equal(t, "own", total.Scope)
	assert.Equal(t, 2.0, kpiByID(t, agent, "pending-tickets").Value, "待处理同样按读取者范围")

	assigneeView := f.kpiWithZeroAggregates(t, DashboardViewer{UserID: assignee.ID, Role: "agent"})
	assert.Equal(t, 1.0, kpiByID(t, assigneeView, "total-tickets").Value, "受理人应看到分配给自己的那张单")

	manager := f.kpiWithZeroAggregates(t, DashboardViewer{UserID: theirs.ID, Role: "manager"})
	mTotal := kpiByID(t, manager, "total-tickets")
	assert.Equal(t, float64(tenantTotal), mTotal.Value, "管理角色保持租户全量")
	assert.Equal(t, "tenant", mTotal.Scope)

	// 无请求者上下文的后台调用必须显式声明全租户，而不是靠空角色兜底
	background := f.kpiWithZeroAggregates(t, DashboardViewer{ScopeAll: true})
	assert.Equal(t, float64(tenantTotal), kpiByID(t, background, "total-tickets").Value)

	// 身份缺失（user_id<=0）的普通角色收窄为空集，不做租户全量兜底
	anonymous := f.kpiWithZeroAggregates(t, DashboardViewer{UserID: 0, Role: "agent"})
	assert.Equal(t, 0.0, kpiByID(t, anonymous, "total-tickets").Value)
}

// TestDashboardService_KPIHasNoFabricatedGrowth 没有可比基线时必须 comparable=false 且
// change=0；此前"上月 0、本月有值"被折算成 +100%，卡片显示出一个不存在的增长率。
func TestDashboardService_KPIHasNoFabricatedGrowth(t *testing.T) {
	f := newKPIFixture(t)
	client, ctx := f.client, f.ctx
	tenant, err := createDashboardTestTenant(ctx, client, "kpi-growth")
	require.NoError(t, err)
	f.tenantID = tenant.ID
	user := newKPIUser(t, ctx, client, tenant.ID, "growth")

	lastMonthStart, thisMonthStart := kpiWindow()
	now := time.Now()
	for i := 0; i < 3; i++ {
		newKPITicket(t, ctx, client, tenant.ID, user.ID, "open", fmt.Sprintf("GR-%d", i), now)
	}

	viewer := DashboardViewer{UserID: user.ID, Role: "agent"}
	for _, id := range []string{
		"total-tickets", "pending-tickets", "in-progress-tickets", "completed-tickets", "overdue-tickets",
	} {
		m := kpiByID(t, f.kpiWithZeroAggregates(t, viewer), id)
		assert.False(t, m.Comparable, "%s 没有可比基线时不应声明可比", id)
		assert.Zero(t, m.Change, "%s 出现捏造增长率", id)
		assert.Equal(t, "stable", m.Trend, "%s 箭头方向必须由真实变化推导", id)
		assert.Equal(t, "stable", m.ChangeType)
	}

	// 补上上月末之前的存量后，总数环比按真实累计值出现：本月 3 + 上月 2 = 5，基线 2 => +150%
	prevMonth := lastMonthStart.AddDate(0, 0, 1)
	newKPITicket(t, ctx, client, tenant.ID, user.ID, "open", "GR-P1", prevMonth)
	newKPITicket(t, ctx, client, tenant.ID, user.ID, "open", "GR-P2", prevMonth)
	total := kpiByID(t, f.kpiWithZeroAggregates(t, viewer), "total-tickets")
	assert.True(t, total.Comparable)
	assert.InDelta(t, 150.0, total.Change, 0.01)
	assert.Equal(t, "up", total.Trend)
	assert.Equal(t, "increase", total.ChangeType)

	// 已完成是同期队列对比：上月新建并完成 2 张、本月新建并完成 1 张 => -50%
	for i := 0; i < 2; i++ {
		newKPITicket(t, ctx, client, tenant.ID, user.ID, "resolved", fmt.Sprintf("GR-C-P%d", i), prevMonth)
	}
	newKPITicket(t, ctx, client, tenant.ID, user.ID, "resolved", "GR-C-1", thisMonthStart.Add(time.Hour))
	completed := kpiByID(t, f.kpiWithZeroAggregates(t, viewer), "completed-tickets")
	assert.True(t, completed.Comparable)
	assert.Equal(t, 1.0, completed.Value)
	assert.InDelta(t, -50.0, completed.Change, 0.01)
	assert.Equal(t, "down", completed.Trend)
	assert.Equal(t, "decrease", completed.ChangeType)

	// 存量快照指标即使有历史数据也不做环比：库里没有上月末的状态快照。
	pending := kpiByID(t, f.kpiWithZeroAggregates(t, viewer), "pending-tickets")
	assert.False(t, pending.Comparable)
	assert.Zero(t, pending.Change)
	assert.Equal(t, "stable", pending.Trend)
}

// TestDashboardService_KPIAggregatesWithoutPreviousSample 时长均值与 SLA 达成率同理：
// 上期窗口没有样本时不得给出百分比变化。
func TestDashboardService_KPIAggregatesWithoutPreviousSample(t *testing.T) {
	f := newKPIFixture(t)
	tenant, err := createDashboardTestTenant(f.ctx, f.client, "kpi-avg")
	require.NoError(t, err)
	f.tenantID = tenant.ID
	user := newKPIUser(t, f.ctx, f.client, tenant.ID, "avg")
	newKPITicket(t, f.ctx, f.client, tenant.ID, user.ID, "open", "AVG-1", time.Now())

	// 本期有时长样本（均值 2h / 6h、10 单考核 9 单达成），上期全 0
	f.mock.ExpectQuery(`.+`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c", "d", "e", "f"}).AddRow(2.0, 6.0, 0, 0, 10, 9))
	f.mock.ExpectQuery(`.+`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"total", "met"}).AddRow(0, 0))

	metrics := f.kpi(t, DashboardViewer{UserID: user.ID, Role: "agent"})

	resp := kpiByID(t, metrics, "avg-first-response")
	assert.Equal(t, 2.0, resp.Value)
	assert.False(t, resp.Comparable, "上期均值为 0 说明没有样本")
	assert.Zero(t, resp.Change)

	sla := kpiByID(t, metrics, "sla-compliance")
	assert.Equal(t, 90.0, sla.Value)
	assert.False(t, sla.Comparable, "上期没有 SLA 考核样本时不得报环比")
	assert.Zero(t, sla.Change)
}

// TestDashboardService_KPIPassesViewerIDToAggregateSQL 读取者身份必须传到聚合 SQL：
// 管理角色与后台全租户口径传 0（不过滤），普通角色传本人 ID。
func TestDashboardService_KPIPassesViewerIDToAggregateSQL(t *testing.T) {
	for i, tc := range []struct {
		name      string
		viewer    DashboardViewer
		wantArg   int
		wantScope string
	}{
		{"agent 传本人 ID", DashboardViewer{UserID: 42, Role: "agent"}, 42, "own"},
		{"manager 传 0", DashboardViewer{UserID: 7, Role: "manager"}, 0, "tenant"},
		{"后台全租户传 0", DashboardViewer{ScopeAll: true}, 0, "tenant"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newKPIFixture(t)
			tenant, err := createDashboardTestTenant(f.ctx, f.client, fmt.Sprintf("kpi-sql-%d", i))
			require.NoError(t, err)
			f.tenantID = tenant.ID

			f.mock.ExpectQuery(`.+`).WithArgs(tenant.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), tc.wantArg).
				WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c", "d", "e", "f"}).AddRow(1, 1, 1, 1, 1, 1))
			f.mock.ExpectQuery(`.+`).WithArgs(tenant.ID, sqlmock.AnyArg(), sqlmock.AnyArg(), tc.wantArg).
				WillReturnRows(sqlmock.NewRows([]string{"total", "met"}).AddRow(0, 0))

			metrics := f.kpi(t, tc.viewer)
			assert.Equal(t, tc.wantScope, kpiByID(t, metrics, "total-tickets").Scope)
		})
	}
}
