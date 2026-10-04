package incident

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 本文件锁住 GET /api/v1/incidents/stats/report 的读数口径。
//
// 修复前 /reports/incident-trends 读的是 ticket 分析接口：页面标题写着「事件趋势」，
// 数字全部来自工单；「新增/已解决」由浏览器把返回的日趋势再累加一次，超过一页的数据
// 被静默截断后仍当全量展示；平均解决时长把后端的分钟当小时渲染（60 倍误差）；
// 请求失败只 console.error + message.error，卡片继续显示 0。
// 因此这里逐字段断言：窗口回显、分布之和、补零趋势、解决集合口径、软删与租户隔离、
// 以及「出错就没有读数」。

type reportContractFixture struct {
	client   *ent.Client
	handler  *IncidentHandler
	tenantA  int
	tenantB  int
	agentA   int
	today    time.Time
	dateFrom string
	dateTo   string
	labels   []string
}

// newReportContractFixture 建一个 7 天窗口（today-6 .. today，闭区间）。
// 种子时间统一落在正午前后，避免测试跨过午夜边界产生歧义。
func newReportContractFixture(t *testing.T) *reportContractFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := enttest.Open(t, "sqlite3", "file:incident_report_contract?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenantA := client.Tenant.Create().
		SetName("ReportA").SetCode("report-a").SetDomain("report-a.test").SaveX(ctx)
	tenantB := client.Tenant.Create().
		SetName("ReportB").SetCode("report-b").SetDomain("report-b.test").SaveX(ctx)
	agentA := client.User.Create().
		SetUsername("agent-a").SetName("agent-a").SetEmail("agent-a@example.com").
		SetPasswordHash("hash").SetTenantID(tenantA.ID).SaveX(ctx)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	from := today.AddDate(0, 0, -6)

	labels := make([]string, 0, 7)
	for day := from; !day.After(today); day = day.AddDate(0, 0, 1) {
		labels = append(labels, day.Format("2006-01-02"))
	}

	return &reportContractFixture{
		client:   client,
		handler:  NewHandler(NewService(NewEntRepository(client), nil, nil, nil, nil, nil, zap.NewNop().Sugar())),
		tenantA:  tenantA.ID,
		tenantB:  tenantB.ID,
		agentA:   agentA.ID,
		today:    today,
		dateFrom: from.Format("2006-01-02"),
		dateTo:   today.Format("2006-01-02"),
		labels:   labels,
	}
}

// label 返回相对今天偏移 dayOffset 的日历日标签。
func (f *reportContractFixture) label(dayOffset int) string {
	return f.today.AddDate(0, 0, dayOffset).Format("2006-01-02")
}

// at 返回今天偏移 dayOffset 天、当天 hour:minute 的时刻。
func (f *reportContractFixture) at(dayOffset, hour, minute int) time.Time {
	return f.today.AddDate(0, 0, dayOffset).Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute)
}

type reportSeed struct {
	number     string
	status     string
	priority   string
	createdDay int
	createdAt  time.Time // 非零则覆盖 createdDay 的默认正午时刻
	resolvedAt time.Time
	deleted    bool
}

func (f *reportContractFixture) seed(t *testing.T, tenantID int, in reportSeed) {
	t.Helper()
	ctx := context.Background()

	created := in.createdAt
	if created.IsZero() {
		created = f.at(in.createdDay, 12, 0)
	}
	create := f.client.Incident.Create().
		SetTitle("报表契约 " + in.number).
		SetDescription("d").
		SetStatus(in.status).
		SetPriority(in.priority).
		SetSeverity("medium").
		SetIncidentNumber(in.number).
		SetReporterID(f.agentA).
		SetTenantID(tenantID).
		SetCreatedAt(created)
	if !in.resolvedAt.IsZero() {
		create.SetResolvedAt(in.resolvedAt)
	}
	if in.deleted {
		create.SetDeletedAt(time.Now())
	}
	create.SaveX(ctx)
}

// decodeReport 返回 data 的原始字段集合与类型化视图。
// 逐字段核对而不是「包含某字段」，多余字段与缺失字段都会在这里失败。
func decodeReport(t *testing.T, w *httptest.ResponseRecorder) (map[string]json.RawMessage, []string) {
	t.Helper()

	var envelope struct {
		Code int                        `json:"code"`
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), w.Body.String())
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Zero(t, envelope.Code, w.Body.String())
	require.NotEmpty(t, envelope.Data, "data 必须是报表对象")

	keys := make([]string, 0, len(envelope.Data))
	for key := range envelope.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return envelope.Data, keys
}

func (f *reportContractFixture) get(t *testing.T, tenantID int, query string) (map[string]json.RawMessage, []string) {
	t.Helper()

	r := gin.New()
	r.GET("/api/v1/incidents/stats/report", func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		f.handler.GetReport(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/incidents/stats/report?"+query, nil))
	return decodeReport(t, w)
}

func unmarshalReport(t *testing.T, data map[string]json.RawMessage) IncidentReport {
	t.Helper()
	raw, err := json.Marshal(data)
	require.NoError(t, err)
	var report IncidentReport
	require.NoError(t, json.Unmarshal(raw, &report))
	return report
}

func distribution(in []IncidentStatCount) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		out = append(out, item.Value)
	}
	return out
}

func sumCounts(in []IncidentStatCount) int {
	total := 0
	for _, item := range in {
		total += item.Count
	}
	return total
}

func sumCreated(in []IncidentTrendPoint) int {
	total := 0
	for _, point := range in {
		total += point.Created
	}
	return total
}

func sumResolved(in []IncidentTrendPoint) int {
	total := 0
	for _, point := range in {
		total += point.Resolved
	}
	return total
}

// TestIncidentReportReadingsAreAuthoritative 覆盖报表全部读数口径。
func TestIncidentReportReadingsAreAuthoritative(t *testing.T) {
	f := newReportContractFixture(t)

	// 窗口内创建的 cohort：7 条（含词表外状态与已取消）。
	f.seed(t, f.tenantA, reportSeed{number: "INC-R-1", status: "new", priority: "low", createdDay: -5})
	f.seed(t, f.tenantA, reportSeed{number: "INC-R-2", status: "in_progress", priority: "high", createdDay: -5, createdAt: f.at(-5, 13, 0)})
	f.seed(t, f.tenantA, reportSeed{
		number: "INC-R-3", status: "resolved", priority: "medium", createdDay: -2,
		resolvedAt: f.at(-2, 13, 30),
	}) // 90 分钟
	f.seed(t, f.tenantA, reportSeed{number: "INC-R-4", status: "awaiting_vendor", priority: "medium", createdDay: -3})
	f.seed(t, f.tenantA, reportSeed{number: "INC-R-5", status: "cancelled", priority: "low", createdDay: -1})
	f.seed(t, f.tenantA, reportSeed{
		number: "INC-R-6", status: "resolved", priority: "low", createdAt: f.at(-1, 9, 0),
		resolvedAt: f.at(0, 9, 0),
	}) // 1440 分钟
	// created 在窗口内、resolved_at 在窗口外：进 cohort，不进解决集合。
	f.seed(t, f.tenantA, reportSeed{
		number: "INC-R-7", status: "resolved", priority: "medium", createdDay: -2,
		resolvedAt: f.at(-9, 12, 0),
	})

	// created 在窗口外、窗口内解决：必须计入解决集合与平均时长（19 天 = 27360 分钟）。
	f.seed(t, f.tenantA, reportSeed{
		number: "INC-R-8", status: "closed", priority: "critical", createdDay: -20,
		resolvedAt: f.at(-1, 12, 0),
	})
	f.seed(t, f.tenantA, reportSeed{number: "INC-R-9", status: "new", priority: "low", createdDay: -8})
	// 软删除：既不进 cohort 也不进解决集合。
	f.seed(t, f.tenantA, reportSeed{
		number: "INC-R-10", status: "resolved", priority: "high", createdDay: -1,
		resolvedAt: f.at(-1, 14, 0), deleted: true,
	})

	// 租户 B 只应有自己的 1 条。
	f.seed(t, f.tenantB, reportSeed{number: "INC-B-1", status: "new", priority: "low", createdDay: -1})

	data, keys := f.get(t, f.tenantA, "dateFrom="+f.dateFrom+"&dateTo="+f.dateTo)
	require.Equal(t,
		[]string{"avgResolutionMinutes", "byPriority", "byStatus", "createdInWindow", "dailyTrend", "resolvedInWindow", "window"},
		keys, "报表契约不得多字段也不得少字段")

	report := unmarshalReport(t, data)

	require.Equal(t, IncidentReportWindow{DateFrom: f.dateFrom, DateTo: f.dateTo, Days: 7}, report.Window,
		"窗口必须回显请求区间，前端不得自行推断")

	require.Equal(t, 7, report.CreatedInWindow, "cohort 只含窗口内创建且未软删的事件")
	require.Equal(t,
		[]string{"new", "in_progress", "resolved", "cancelled", "awaiting_vendor"},
		distribution(report.ByStatus), "状态按词表顺序输出，词表外取值原样追加")
	require.Equal(t, []string{"low", "medium", "high"}, distribution(report.ByPriority))
	require.Equal(t, report.CreatedInWindow, sumCounts(report.ByStatus), "状态分布之和必须等于 cohort 总数")
	require.Equal(t, report.CreatedInWindow, sumCounts(report.ByPriority), "优先级分布之和必须等于 cohort 总数")

	// 解决口径按 resolved_at：前天 1 条、昨天 1 条（窗口外创建）、今天 1 条。
	require.Equal(t, 3, report.ResolvedInWindow)
	// (90 + 1440 + 27360) / 3 = 9630 分钟，单位写在字段名里。
	require.Equal(t, 9630, report.AvgResolutionMinutes)

	require.Len(t, report.DailyTrend, 7, "趋势必须逐日补零，不得只返回有数据的日子")
	labels := make([]string, 0, len(report.DailyTrend))
	created := make(map[string]int, len(report.DailyTrend))
	resolved := make(map[string]int, len(report.DailyTrend))
	for _, point := range report.DailyTrend {
		labels = append(labels, point.Date)
		created[point.Date] = point.Created
		resolved[point.Date] = point.Resolved
	}
	require.Equal(t, f.labels, labels)
	require.Equal(t, report.CreatedInWindow, sumCreated(report.DailyTrend), "每日新建之和必须等于 createdInWindow")
	require.Equal(t, report.ResolvedInWindow, sumResolved(report.DailyTrend), "每日解决之和必须等于 resolvedInWindow")

	require.Equal(t, map[string]int{
		f.label(-6): 0, f.label(-5): 2, f.label(-4): 0, f.label(-3): 1,
		f.label(-2): 2, f.label(-1): 2, f.label(0): 0,
	}, created)
	// 解决集合与新建集合是两套口径：窗口前几天没有解决事件的日子必须补零。
	require.Equal(t, map[string]int{
		f.label(-6): 0, f.label(-5): 0, f.label(-4): 0, f.label(-3): 0,
		f.label(-2): 1, f.label(-1): 1, f.label(0): 1,
	}, resolved)

	// 租户隔离：B 看不见 A 的任何读数。
	bData, _ := f.get(t, f.tenantB, "dateFrom="+f.dateFrom+"&dateTo="+f.dateTo)
	b := unmarshalReport(t, bData)
	require.Equal(t, 1, b.CreatedInWindow)
	require.Zero(t, b.ResolvedInWindow)
	require.Zero(t, b.AvgResolutionMinutes)
	require.Equal(t, []IncidentStatCount{{Value: "new", Count: 1}}, b.ByStatus)
	require.Equal(t, []IncidentStatCount{{Value: "low", Count: 1}}, b.ByPriority)
	require.Len(t, b.DailyTrend, 7)
	require.Equal(t, 1, sumCreated(b.DailyTrend))
}

// TestIncidentReportDefaultWindow 确认不传日期时走服务端缺省窗口，
// 而不是像修复前那样把「最近 7/30/90 天」的选择留在浏览器里不发请求。
func TestIncidentReportDefaultWindow(t *testing.T) {
	f := newReportContractFixture(t)
	f.seed(t, f.tenantA, reportSeed{number: "INC-D-1", status: "new", priority: "low", createdDay: -1})

	data, _ := f.get(t, f.tenantA, "")
	report := unmarshalReport(t, data)

	require.Equal(t, defaultReportDays, report.Window.Days)
	require.Equal(t, f.label(-(defaultReportDays - 1)), report.Window.DateFrom)
	require.Equal(t, f.label(0), report.Window.DateTo)
	require.Len(t, report.DailyTrend, defaultReportDays)
	require.Equal(t, f.label(0), report.DailyTrend[len(report.DailyTrend)-1].Date)
	require.Equal(t, 1, sumCreated(report.DailyTrend))
}

// TestIncidentReportRejectsBadWindows 锁住参数错误语义：
// 区间倒置、跨度超限、只传一端、格式非法都必须是 400/1001，
// 不得回落到缺省窗口后返回一份「看起来成功」的读数。
func TestIncidentReportRejectsBadWindows(t *testing.T) {
	f := newReportContractFixture(t)

	cases := map[string]string{
		"只传 dateFrom": "dateFrom=" + f.dateFrom,
		"只传 dateTo":   "dateTo=" + f.dateTo,
		"区间倒置":        "dateFrom=" + f.dateTo + "&dateTo=" + f.dateFrom,
		"跨度过大":        "dateFrom=2000-01-01&dateTo=2026-01-01",
		"日期格式非法":      "dateFrom=2026%2F05%2F01&dateTo=" + f.dateTo,
		"结束日无法解析":     "dateFrom=" + f.dateFrom + "&dateTo=hello",
	}

	for name, query := range cases {
		r := gin.New()
		r.GET("/api/v1/incidents/stats/report", func(c *gin.Context) {
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: f.tenantA})
			f.handler.GetReport(c)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/incidents/stats/report?"+query, nil))

		var envelope struct {
			Code int `json:"code"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), "%s: %s", name, w.Body.String())
		require.Equal(t, http.StatusBadRequest, w.Code, "%s: %s", name, w.Body.String())
		require.Equal(t, common.ParamErrorCode, envelope.Code, "%s: %s", name, w.Body.String())
	}
}

// TestIncidentReportFailsClosedWithoutTenantContext 锁住缺租户上下文的语义：401/2001，
// 不得回落默认租户，也不得返回空报表伪装成功。
func TestIncidentReportFailsClosedWithoutTenantContext(t *testing.T) {
	f := newReportContractFixture(t)

	r := gin.New()
	r.GET("/api/v1/incidents/stats/report", func(c *gin.Context) { f.handler.GetReport(c) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/incidents/stats/report", nil))

	var envelope struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), w.Body.String())
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	require.Equal(t, 2001, envelope.Code, w.Body.String())
}
