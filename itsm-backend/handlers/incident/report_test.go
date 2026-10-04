package incident

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"itsm-backend/common"
)

// 本文件锁住事件趋势报表的窗口解析与分布折叠规则。
//
// 修复前 /reports/incident-trends 在浏览器里用 ticket 分析接口自己数：
// 事件没有窗口端点，页面只能拿工单趋势冒充，并把「所选周期」留在本地不发请求。
// 窗口口径因此必须由后端唯一决定，且严格到「错就报参数错误」，
// 否则任何静默纠正都会让页面上的读数与用户选择脱钩。

func mustResolvePeriod(t *testing.T, dateFrom, dateTo string, now time.Time) ReportPeriod {
	t.Helper()
	period, err := ResolveReportPeriod(dateFrom, dateTo, now)
	require.NoError(t, err)
	return period
}

func requireParamError(t *testing.T, err error, wantMessage string) {
	t.Helper()
	require.Error(t, err)
	biz, ok := err.(*common.BusinessError)
	require.True(t, ok, "窗口错误必须是业务参数错误，实际 %T: %v", err, err)
	require.Equal(t, common.ParamErrorCode, biz.Code, biz.Message)
	require.Equal(t, wantMessage, biz.Message)
}

// asStrings 把绑定参数切片转成可读文本，用于逐位核对占位符顺序。
func asStrings(t *testing.T, in []any) []string {
	t.Helper()
	out := make([]string, 0, len(in))
	for _, arg := range in {
		value, ok := arg.(string)
		require.True(t, ok, "状态参数必须是字符串，实际 %T", arg)
		out = append(out, value)
	}
	return out
}

func TestResolveReportPeriodDefaultWindow(t *testing.T) {
	now := time.Date(2026, 5, 20, 15, 30, 0, 0, time.Local)

	period := mustResolvePeriod(t, "", "", now)
	require.Equal(t, time.Date(2026, 4, 21, 0, 0, 0, 0, time.Local), period.Start)
	// 缺省窗口包含 now 当天，因此排他上界是次日 00:00。
	require.Equal(t, time.Date(2026, 5, 21, 0, 0, 0, 0, time.Local), period.End)
	require.Len(t, period.DayLabels(), defaultReportDays)
	require.Equal(t, IncidentReportWindow{
		DateFrom: "2026-04-21",
		DateTo:   "2026-05-20",
		Days:     defaultReportDays,
	}, period.Window())
}

func TestResolveReportPeriodExplicitDates(t *testing.T) {
	now := time.Date(2026, 5, 20, 15, 30, 0, 0, time.Local)

	t.Run("单日窗口", func(t *testing.T) {
		period := mustResolvePeriod(t, "2026-05-12", "2026-05-12", now)
		require.Equal(t, []string{"2026-05-12"}, period.DayLabels())
		require.Equal(t, IncidentReportWindow{DateFrom: "2026-05-12", DateTo: "2026-05-12", Days: 1}, period.Window())
	})

	t.Run("dateTo 是闭区间", func(t *testing.T) {
		period := mustResolvePeriod(t, "2026-05-01", "2026-05-03", now)
		require.Equal(t, time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local), period.Start)
		require.Equal(t, time.Date(2026, 5, 4, 0, 0, 0, 0, time.Local), period.End, "结束日整天必须纳入窗口")
		require.Equal(t, []string{"2026-05-01", "2026-05-02", "2026-05-03"}, period.DayLabels())
	})

	t.Run("RFC3339 与日期混用", func(t *testing.T) {
		from := time.Date(2026, 5, 2, 0, 0, 0, 0, time.Local).Format(time.RFC3339)
		period := mustResolvePeriod(t, from, "2026-05-04", now)
		require.Equal(t, 3, period.Window().Days)
		require.Equal(t, "2026-05-02", period.Window().DateFrom)
		require.Equal(t, "2026-05-04", period.Window().DateTo)
	})

	t.Run("跨度上限内可用", func(t *testing.T) {
		from := now.AddDate(0, 0, -(maxReportDays - 1)).Format("2006-01-02")
		period := mustResolvePeriod(t, from, "2026-05-20", now)
		require.Equal(t, maxReportDays, period.Window().Days)
	})
}

func TestResolveReportPeriodRejects(t *testing.T) {
	now := time.Date(2026, 5, 20, 15, 30, 0, 0, time.Local)

	// 只传一端不得静默补另一端：否则页面以为自己按日期过滤，实际读的是缺省窗口。
	_, err := ResolveReportPeriod("2026-05-01", "", now)
	requireParamError(t, err, "dateFrom 与 dateTo 必须同时提供")
	_, err = ResolveReportPeriod("", "2026-05-10", now)
	requireParamError(t, err, "dateFrom 与 dateTo 必须同时提供")

	_, err = ResolveReportPeriod("2026-05-10", "2026-05-01", now)
	requireParamError(t, err, "dateFrom 不能晚于 dateTo")

	_, err = ResolveReportPeriod(now.AddDate(0, 0, -maxReportDays).Format("2006-01-02"), "2026-05-20", now)
	requireParamError(t, err, "统计窗口最长 366 天")

	for _, bad := range []string{"2026/05/01", "2026-13-45", "May 1, 2026"} {
		_, err = ResolveReportPeriod(bad, "2026-05-10", now)
		requireParamError(t, err, "dateFrom 格式错误（RFC3339 或 YYYY-MM-DD）")
	}
	_, err = ResolveReportPeriod("2026-05-01", "not-a-date", now)
	requireParamError(t, err, "dateTo 格式错误（RFC3339 或 YYYY-MM-DD）")
}

func TestReportPeriodDayLabelsOnEmptyWindow(t *testing.T) {
	// 仓储层以此长度为补零基准，退化区间必须显式为空而不是 1 天。
	instant := time.Date(2026, 5, 20, 15, 30, 0, 0, time.Local)
	period := ReportPeriod{Start: instant, End: instant}
	require.Empty(t, period.DayLabels())
	require.Equal(t, IncidentReportWindow{}, period.Window())

	// 区间倒置同样不得产出任何标签。
	require.Empty(t, ReportPeriod{Start: instant, End: instant.AddDate(0, 0, -1)}.DayLabels())
}

func TestOrderedIncidentCounts(t *testing.T) {
	t.Run("词表顺序优先且丢弃零计数", func(t *testing.T) {
		counts := map[string]int{
			"critical": 1, "in_progress": 2, "medium": 3, "new": 4,
			"low": 0, "resolved": 0,
		}
		require.Equal(t,
			[]IncidentStatCount{{"new", 4}, {"in_progress", 2}, {"critical", 1}, {"medium", 3}},
			orderedIncidentCounts(counts, incidentStatusOrder),
			"状态按词表顺序，优先级不在状态词表内，按字典序追加")
		require.Equal(t,
			[]IncidentStatCount{{"medium", 3}, {"critical", 1}},
			orderedIncidentCounts(map[string]int{"medium": 3, "critical": 1, "low": 0, "high": 0}, incidentPriorityOrder),
			"优先级按词表顺序，零计数不进入分布")
	})

	t.Run("词表外取值按字典序追加且不丢计数", func(t *testing.T) {
		got := orderedIncidentCounts(
			map[string]int{"high": 2, "zzz_legacy": 1, "awaiting_vendor": 3, "closed": 0},
			incidentPriorityOrder,
		)
		require.Equal(t,
			[]IncidentStatCount{{"high", 2}, {"awaiting_vendor", 3}, {"zzz_legacy", 1}},
			got, "脏值必须原样返回，既不能并进别的桶也不能被静默丢弃")
	})

	t.Run("空输入返回非 nil 空切片", func(t *testing.T) {
		got := orderedIncidentCounts(map[string]int{"resolved": 0}, incidentStatusOrder)
		require.NotNil(t, got)
		require.Empty(t, got, "零计数不得进入分布，否则之和与总数对不上")
	})
}

// TestIncidentVocabularyHasNoTicketLeftovers 锁住事件域词表本身。
// 'open' 从来不是事件状态（新建落库是 'new'），'urgent' 从来不是事件优先级
// （ent/schema/incident.go 只允许 low/medium/high/critical）：
// 这两个值是 SQL 与前端常量把 ticket 词表抄进 incident 留下的，
// 一旦重新出现，openIncidents 又会静默归零。
func TestIncidentVocabularyHasNoTicketLeftovers(t *testing.T) {
	for _, list := range [][]string{incidentStatusOrder, incidentOpenStatuses, incidentResolvedStatuses} {
		require.NotContains(t, list, "open", "事件状态没有 open")
	}
	require.NotContains(t, incidentPriorityOrder, "urgent", "事件优先级没有 urgent")

	// 状态顺序必须与 common 常量表逐项一致，避免又出现第三套词表。
	require.Equal(t, []string{
		"new", "acknowledged", "assigned", "triaged", "in_progress",
		"escalated", "on_hold", "resolved", "closed", "cancelled",
	}, incidentStatusOrder)
}

// TestIncidentStatusPartition 锁住「未终结 / 已解决」两个集合的边界：
// 它们是标量聚合与报表解决口径的唯一来源，必须互斥且只含合法状态。
func TestIncidentStatusPartition(t *testing.T) {
	require.Equal(t, []string{common.IncidentStatusResolved, common.IncidentStatusClosed}, incidentResolvedStatuses)
	require.Equal(t,
		[]string{
			common.IncidentStatusNew, common.IncidentStatusAcknowledged, common.IncidentStatusAssigned,
			common.IncidentStatusTriaged, common.IncidentStatusInProgress, common.IncidentStatusEscalated,
			common.IncidentStatusOnHold,
		},
		incidentOpenStatuses,
	)

	for _, status := range incidentOpenStatuses {
		require.True(t, IsOpenIncidentStatus(status), status)
		require.False(t, IsResolvedIncidentStatus(status), status)
		require.True(t, slices.Contains(incidentStatusOrder, status), status+" 必须在展示词表内")
	}
	for _, status := range incidentResolvedStatuses {
		require.True(t, IsResolvedIncidentStatus(status), status)
		require.False(t, IsOpenIncidentStatus(status), status)
	}
	// 取消既不算未终结也不算已解决，不得被静默并入任一桶。
	require.False(t, IsOpenIncidentStatus(common.IncidentStatusCancelled))
	require.False(t, IsResolvedIncidentStatus(common.IncidentStatusCancelled))
	require.False(t, IsOpenIncidentStatus("open"), "词表外取值不得被判定为未终结")
}

// TestBuildStatsQueryBindsVocabulary 锁住 PostgreSQL 标量聚合的取值来源：
// 状态一律走占位符绑定，SQL 文本里不允许再出现 'open' 之类硬编码字面量。
func TestBuildStatsQueryBindsVocabulary(t *testing.T) {
	query, args := buildStatsQuery(42)

	require.Equal(t, 42, args[0], "$1 必须是租户")
	require.Len(t, args, 1+len(incidentOpenStatuses)+len(incidentResolvedStatuses))
	require.NotContains(t, query, "'open'", "SQL 文本不得再内联事件状态字面量")
	require.NotContains(t, query, "status IN ('", "状态取值必须全部走绑定参数")
	for _, status := range append(append([]string{}, incidentOpenStatuses...), incidentResolvedStatuses...) {
		require.Contains(t, args, status, status+" 必须作为参数下推")
	}
	// 参数顺序即占位符顺序：$1 租户，$2..$8 未终结集合，$9..$10 已解决集合。
	require.Equal(t, strings.Join(asStrings(t, args[1:1+len(incidentOpenStatuses)]), ","),
		strings.Join(incidentOpenStatuses, ","))
	resolvedStart := 1 + len(incidentOpenStatuses)
	require.Equal(t, strings.Join(asStrings(t, args[resolvedStart:]), ","),
		strings.Join(incidentResolvedStatuses, ","))

	openPlaceholders := "$2, $3, $4, $5, $6, $7, $8"
	resolvedPlaceholders := "$9, $10"
	require.Contains(t, query, "status IN ("+openPlaceholders+")", "未终结集合必须绑定全部 7 个状态")
	require.Equal(t, 2, strings.Count(query, "status IN ("+resolvedPlaceholders+")"),
		"已解决集合在解决数与平均时长两处复用同一组占位符，重复引用而非重复入参")
	require.True(t, strings.Contains(query, "tenant_id = $1"))
	require.True(t, strings.Contains(query, "deleted_at IS NULL"))
}
