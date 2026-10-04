package incident

import (
	"fmt"
	"sort"
	"time"

	"itsm-backend/common"
)

// 报表窗口口径：
//   - 未传 dateFrom/dateTo 时使用 defaultReportDays 天（含当天往前）；
//   - 窗口跨度上限 maxReportDays 天，避免无界时间范围把趋势桶撑爆。
const (
	defaultReportDays = 30
	maxReportDays     = 366
)

// incidentStatusOrder 与 common.IncidentStatus* 常量一一对应，是状态分布的展示基准顺序。
// 事件 status 列没有 Ent 枚举约束，词表外的历史取值按字典序追加（见 orderedIncidentCounts）。
var incidentStatusOrder = []string{
	common.IncidentStatusNew,
	common.IncidentStatusAcknowledged,
	common.IncidentStatusAssigned,
	common.IncidentStatusTriaged,
	common.IncidentStatusInProgress,
	common.IncidentStatusEscalated,
	common.IncidentStatusOnHold,
	common.IncidentStatusResolved,
	common.IncidentStatusClosed,
	common.IncidentStatusCancelled,
}

// incidentPriorityOrder 与 ent/schema/incident.go 对 priority 的校验取值一致：
// 事件域的优先级是 low/medium/high/critical，没有 urgent（urgent 属于工单优先级词表）。
var incidentPriorityOrder = []string{"low", "medium", "high", "critical"}

// orderedIncidentCounts 把分组计数折叠成顺序稳定的分布：
// 先按词表基准顺序输出真实存在的取值（count>0），词表外的取值按字典序追加。
// 既丢弃零计数也保留脏值，因此调用方可以用「分布之和 == 总数」做对账断言。
func orderedIncidentCounts(counts map[string]int, order []string) []IncidentStatCount {
	out := make([]IncidentStatCount, 0, len(counts))
	seen := make(map[string]bool, len(counts))
	for _, value := range order {
		seen[value] = true
		if count := counts[value]; count > 0 {
			out = append(out, IncidentStatCount{Value: value, Count: count})
		}
	}
	unknown := make([]string, 0, len(counts))
	for value, count := range counts {
		if count > 0 && !seen[value] {
			unknown = append(unknown, value)
		}
	}
	sort.Strings(unknown)
	for _, value := range unknown {
		out = append(out, IncidentStatCount{Value: value, Count: counts[value]})
	}
	return out
}

// ReportPeriod 是仓储查询使用的窗口边界，半开区间 [Start, End)，本地时区整日对齐。
type ReportPeriod struct {
	Start time.Time
	End   time.Time
}

// DayLabels 返回窗口内每一天的 YYYY-MM-DD 标签，趋势按此补零，
// 使「没有事件的日子」在图上表现为 0 而不是断点。
func (p ReportPeriod) DayLabels() []string {
	if !p.End.After(p.Start) {
		return nil
	}
	labels := []string{}
	for day := p.Start; day.Before(p.End); day = day.AddDate(0, 0, 1) {
		labels = append(labels, day.Format("2006-01-02"))
	}
	return labels
}

// Window 把查询边界回显为对外契约（dateTo 回到闭区间日）。
func (p ReportPeriod) Window() IncidentReportWindow {
	labels := p.DayLabels()
	if len(labels) == 0 {
		return IncidentReportWindow{}
	}
	return IncidentReportWindow{
		DateFrom: labels[0],
		DateTo:   labels[len(labels)-1],
		Days:     len(labels),
	}
}

// ResolveReportPeriod 把 dateFrom/dateTo 解析成本地时区整日窗口。
//
// 规则（刻意严格，不做静默纠正）：
//   - 两者必须成对出现，只传一端返回参数错误；都不传取最近 defaultReportDays 天（含 now 当天）。
//   - 接受 YYYY-MM-DD 或 RFC3339；dateTo 为闭区间日，内部转成次日 00:00 的排他上界。
//   - 区间倒置或跨度超过 maxReportDays 天返回参数错误，不得回落到默认窗口。
func ResolveReportPeriod(dateFrom, dateTo string, now time.Time) (ReportPeriod, error) {
	startOfDay := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location())
	}

	if dateFrom == "" && dateTo == "" {
		end := startOfDay(now).AddDate(0, 0, 1)
		return ReportPeriod{Start: end.AddDate(0, 0, -defaultReportDays), End: end}, nil
	}
	if dateFrom == "" || dateTo == "" {
		return ReportPeriod{}, common.NewBusinessError(common.ParamErrorCode,
			"dateFrom 与 dateTo 必须同时提供", "事件报表窗口需要成对的日期边界")
	}

	from, err := parseReportDate(dateFrom, "dateFrom")
	if err != nil {
		return ReportPeriod{}, err
	}
	to, err := parseReportDate(dateTo, "dateTo")
	if err != nil {
		return ReportPeriod{}, err
	}

	start := startOfDay(from)
	end := startOfDay(to).AddDate(0, 0, 1)
	days := 0
	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		days++
	}
	if days <= 0 {
		return ReportPeriod{}, common.NewBusinessError(common.ParamErrorCode,
			"dateFrom 不能晚于 dateTo", "事件报表窗口区间倒置")
	}
	if days > maxReportDays {
		return ReportPeriod{}, common.NewBusinessError(common.ParamErrorCode,
			fmt.Sprintf("统计窗口最长 %d 天", maxReportDays), "事件报表窗口跨度过大")
	}
	return ReportPeriod{Start: start, End: end}, nil
}

// parseReportDate 兼容 YYYY-MM-DD 与 RFC3339 两种写法，与 Lists 的 dateFrom/dateTo 口径一致。
func parseReportDate(value, name string) (time.Time, error) {
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse("2006-01-02", value); err == nil {
		return ts, nil
	}
	return time.Time{}, common.NewBusinessError(common.ParamErrorCode,
		name+" 格式错误（RFC3339 或 YYYY-MM-DD）", "事件报表窗口日期无法解析")
}

// dayLabel 把时间归到本地时区的日历日；窗口边界本身是本地整日，
// 因此落在窗口内的记录必然映射到 DayLabels 中的某个标签。
func dayLabel(t time.Time) string {
	return t.Local().Format("2006-01-02")
}
