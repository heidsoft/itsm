package ticket

import (
	"fmt"
	"time"

	"itsm-backend/common"
	"itsm-backend/internal/datascope"
	repoTicket "itsm-backend/repository/ticket"
)

// 统计窗口口径与事件报表（handlers/incident/report.go）保持一致：
// 日期成对提供、最长 maxStatsWindowDays 天、只传一端是参数错误而不是静默补全。
const maxStatsWindowDays = 366

// ResolveStatsWindow 把 dateFrom/dateTo 解析成统计窗口（本地时区整日闭区间）。
//
// 两者都不传返回 (nil, nil)：工作台卡片等调用方要保持「可见范围内全量」的旧口径，
// 不能在这里悄悄塞一个默认 30 天，否则 KPI 数字会在没人改筛选条件的情况下变小。
// 带窗口的请求才会填充趋势与 SLA 读数。
func ResolveStatsWindow(dateFrom, dateTo string, now time.Time) (*repoTicket.StatsWindow, error) {
	if dateFrom == "" && dateTo == "" {
		return nil, nil
	}
	if dateFrom == "" || dateTo == "" {
		return nil, common.NewBusinessError(common.ParamErrorCode,
			"dateFrom 与 dateTo 必须同时提供", "工单统计窗口需要成对的日期边界")
	}

	from, err := parseStatsDate(dateFrom, "dateFrom")
	if err != nil {
		return nil, err
	}
	to, err := parseStatsDate(dateTo, "dateTo")
	if err != nil {
		return nil, err
	}

	loc := now.Location()
	start := startOfDay(from, loc)
	// List 的日期过滤是 CreatedAtLTE，因此上界取当日最后一纳秒，
	// 否则「到今天」会把今天 00:00:00 之后创建的工单全部排除。
	end := startOfDay(to, loc).AddDate(0, 0, 1).Add(-time.Nanosecond)
	if end.Before(start) {
		return nil, common.NewBusinessError(common.ParamErrorCode,
			"dateFrom 不能晚于 dateTo", "工单统计窗口区间倒置")
	}
	days := int(end.Sub(start).Hours()/24) + 1
	if days > maxStatsWindowDays {
		return nil, common.NewBusinessError(common.ParamErrorCode,
			fmt.Sprintf("统计窗口最长 %d 天", maxStatsWindowDays), "工单统计窗口跨度过大")
	}
	return &repoTicket.StatsWindow{From: start, To: end}, nil
}

// StatsWindowEcho 把窗口回显为对外契约（YYYY-MM-DD + 天数），供页面标注口径。
func StatsWindowEcho(window *repoTicket.StatsWindow) *TicketStatsWindow {
	if window == nil {
		return nil
	}
	days := int(window.To.Sub(window.From).Hours()/24) + 1
	return &TicketStatsWindow{
		DateFrom: window.From.Format("2006-01-02"),
		DateTo:   window.To.Format("2006-01-02"),
		Days:     days,
	}
}

// StatsScopeLabel 把行级数据范围映射为对外口径词表值，与 datascope 解析
// Role.data_scope 用的取值（all/department/owner）一致，前端据此决定提示文案。
func StatsScopeLabel(scope datascope.DataScope) string {
	switch scope {
	case datascope.DataScopeAll:
		return "all"
	case datascope.DataScopeDepartment:
		return "department"
	default:
		return "owner"
	}
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// parseStatsDate 兼容 YYYY-MM-DD 与 RFC3339，与列表接口的 dateFrom/dateTo 口径一致。
func parseStatsDate(value, name string) (time.Time, error) {
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse("2006-01-02", value); err == nil {
		return ts, nil
	}
	return time.Time{}, common.NewBusinessError(common.ParamErrorCode,
		name+" 格式错误（RFC3339 或 YYYY-MM-DD）", "工单统计窗口日期无法解析")
}
