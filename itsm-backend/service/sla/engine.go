// Package sla provides a unified SLA computation engine.
//
// Phase 3 Step 3.4（2026-09-27）：收敛分散在 ticket_sla_service / sla_policy_service /
// bpmn_sla_service 的三套截止时间计算逻辑。
//
// 设计原则：
//   - 单源计算：business hours 解析与截止时间推进算法只在此处实现
//   - 可注入：TenantLocationResolver 允许调用方注入租户时区解析逻辑，避免循环依赖
//   - 持久化无关：引擎只负责计算，由调用方决定写入哪张表
//
// 三阶段迁移（Task #18）：
//   阶段 1 — 双写：调用方在旧逻辑基础上额外调用 ComputeDeadlines 写入 sla_state
//   阶段 2 — 存量迁移：从旧内嵌字段回填 sla_states
//   阶段 3 — 切读：读取切到 sla_states，移除旧内嵌字段
package sla

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// TenantLocationResolver 解析租户时区。
// 由调用方注入（通常包装 service.LoadTenantLocation），避免 service/sla → service 循环依赖。
// 返回 nil 时引擎回退到 Asia/Shanghai。
type TenantLocationResolver func(ctx context.Context, tenantID int) *time.Location

// Engine 统一 SLA 计算引擎。
type Engine struct {
	nowFunc           func() time.Time
	tenantLocResolver TenantLocationResolver
}

// NewEngine 创建 SLA 引擎。
func NewEngine(resolver TenantLocationResolver) *Engine {
	if resolver == nil {
		resolver = func(_ context.Context, _ int) *time.Location {
			return nil
		}
	}
	return &Engine{
		nowFunc:           time.Now,
		tenantLocResolver: resolver,
	}
}

// WithNowFunc 覆盖时钟（测试用）。
func (e *Engine) WithNowFunc(f func() time.Time) *Engine {
	e.nowFunc = f
	return e
}

// ComputeInput 计算截止时间所需的输入参数。
type ComputeInput struct {
	TenantID       int
	ResponseTime   int // 响应时间（分钟），0 表示不计算响应截止
	ResolutionTime int // 解决时间（分钟），0 表示不计算解决截止
	BusinessHours  map[string]interface{}
	StartTime      time.Time
}

// ComputeResult 截止时间计算结果。
type ComputeResult struct {
	ResponseDeadline   *time.Time
	ResolutionDeadline *time.Time
	Location           *time.Location
}

// ComputeDeadlines 计算响应/解决截止时间。
// 纯计算，不写数据库。
// 当 BusinessHours 为空时，直接按自然时间累加（24x7 无限制）。
func (e *Engine) ComputeDeadlines(ctx context.Context, input ComputeInput) ComputeResult {
	var result ComputeResult

	if len(input.BusinessHours) == 0 {
		if input.ResponseTime > 0 {
			d := input.StartTime.Add(time.Duration(input.ResponseTime) * time.Minute)
			result.ResponseDeadline = &d
		}
		if input.ResolutionTime > 0 {
			d := input.StartTime.Add(time.Duration(input.ResolutionTime) * time.Minute)
			result.ResolutionDeadline = &d
		}
		return result
	}

	loc := e.tenantLocResolver(ctx, input.TenantID)
	cfg := parseBusinessHoursConfig(input.BusinessHours)
	if loc != nil {
		cfg.loc = loc
	}
	if cfg.loc == nil {
		cfg.loc = defaultLocation()
	}
	result.Location = cfg.loc

	if input.ResponseTime > 0 {
		d := addBusinessMinutes(input.StartTime, input.ResponseTime, cfg)
		result.ResponseDeadline = &d
	}
	if input.ResolutionTime > 0 {
		d := addBusinessMinutes(input.StartTime, input.ResolutionTime, cfg)
		result.ResolutionDeadline = &d
	}
	return result
}

// SLAStatus 计算 SLA 当前状态。
func SLAStatus(now time.Time, responseDeadline, resolutionDeadline *time.Time, firstResponseAt, resolvedAt time.Time) string {
	if !resolvedAt.IsZero() && resolutionDeadline != nil && now.After(*resolutionDeadline) {
		return "breached"
	}
	if responseDeadline != nil && now.After(*responseDeadline) && firstResponseAt.IsZero() {
		return "breached"
	}
	if !firstResponseAt.IsZero() && responseDeadline != nil && firstResponseAt.After(*responseDeadline) {
		return "breached"
	}
	if resolutionDeadline != nil && now.After(*resolutionDeadline) && resolvedAt.IsZero() {
		return "breached"
	}
	if responseDeadline != nil && now.Before(*responseDeadline) && firstResponseAt.IsZero() && now.Add(30*time.Minute).After(*responseDeadline) {
		return "warning"
	}
	return "ok"
}

// ─── Business Hours Helpers ─────────────────────────────────────────────────────
//
// 以下函数从 service/ticket_sla_service.go 迁移而来，是 SLA 截止时间的权威实现。
// 旧代码（ticket_sla_service.go / sla_policy_service.go / bpmn_sla_service.go）应逐步委托到此。

// businessHoursConfig 业务时间配置。
type businessHoursConfig struct {
	workDays  map[time.Weekday]bool
	startHour int
	startMin  int
	endHour   int
	endMin    int
	holidays  map[string]bool
	is24x7    bool
	loc       *time.Location
}

func defaultLocation() *time.Location {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return loc
}

func defaultBusinessHoursConfig() businessHoursConfig {
	return businessHoursConfig{
		workDays: map[time.Weekday]bool{
			time.Monday: true, time.Tuesday: true, time.Wednesday: true,
			time.Thursday: true, time.Friday: true,
		},
		startHour: 9,
		endHour:   18,
		holidays:  map[string]bool{},
	}
}

// parseBusinessHoursConfig 从 JSON 配置解析业务时间。
// 兼容多种键名：start_time/start_hour/work_hour_start、end_time/end_hour/work_hour_end、
// time_zone/timezone、work_days/workdays、is_24_7。
func parseBusinessHoursConfig(raw map[string]interface{}) businessHoursConfig {
	cfg := defaultBusinessHoursConfig()
	if len(raw) == 0 {
		return cfg
	}

	if v, ok := raw["is_24_7"].(bool); ok {
		cfg.is24x7 = v
	} else if v, ok := raw["is_24_7"].(string); ok {
		cfg.is24x7 = v == "true" || v == "1"
	}

	parseDays := func() {
		if days, ok := raw["work_days"].([]interface{}); ok && len(days) > 0 {
			cfg.workDays = map[time.Weekday]bool{}
			dayMap := map[int]time.Weekday{
				1: time.Monday, 2: time.Tuesday, 3: time.Wednesday,
				4: time.Thursday, 5: time.Friday, 6: time.Saturday, 7: time.Sunday,
			}
			for _, d := range days {
				if dv, ok := d.(float64); ok {
					if wd, ok := dayMap[int(dv)]; ok {
						cfg.workDays[wd] = true
					}
				}
			}
		} else if days, ok := raw["workdays"].([]interface{}); ok && len(days) > 0 {
			cfg.workDays = map[time.Weekday]bool{}
			dayMap := map[int]time.Weekday{
				1: time.Monday, 2: time.Tuesday, 3: time.Wednesday,
				4: time.Thursday, 5: time.Friday, 6: time.Saturday, 7: time.Sunday,
			}
			for _, d := range days {
				if dv, ok := d.(float64); ok {
					if wd, ok := dayMap[int(dv)]; ok {
						cfg.workDays[wd] = true
					}
				}
			}
		}
	}
	parseDays()

	parseHM := func(s string) (int, int) {
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return -1, -1
		}
		h, e1 := strconv.Atoi(parts[0])
		m, e2 := strconv.Atoi(parts[1])
		if e1 != nil || e2 != nil {
			return -1, -1
		}
		return h, m
	}

	if st, ok := raw["start_time"].(string); ok {
		if h, m := parseHM(st); h >= 0 {
			cfg.startHour, cfg.startMin = h, m
		}
	} else if h, ok := raw["start_hour"].(float64); ok {
		cfg.startHour = int(h)
	} else if h, ok := raw["work_hour_start"].(float64); ok {
		cfg.startHour = int(h)
	}

	if et, ok := raw["end_time"].(string); ok {
		if h, m := parseHM(et); h >= 0 {
			cfg.endHour, cfg.endMin = h, m
		}
	} else if h, ok := raw["end_hour"].(float64); ok {
		cfg.endHour = int(h)
	} else if h, ok := raw["work_hour_end"].(float64); ok {
		cfg.endHour = int(h)
	}

	if holidays, ok := raw["holiday_list"].([]interface{}); ok {
		for _, h := range holidays {
			if hs, ok := h.(string); ok {
				cfg.holidays[hs] = true
			}
		}
	}

	if tz, ok := raw["time_zone"].(string); ok && tz != "" {
		if loc, e := time.LoadLocation(tz); e == nil {
			cfg.loc = loc
		}
	} else if tz, ok := raw["timezone"].(string); ok && tz != "" {
		if loc, e := time.LoadLocation(tz); e == nil {
			cfg.loc = loc
		}
	}

	if cfg.is24x7 {
		cfg.workDays = map[time.Weekday]bool{
			time.Monday: true, time.Tuesday: true, time.Wednesday: true,
			time.Thursday: true, time.Friday: true, time.Saturday: true, time.Sunday: true,
		}
		cfg.startHour, cfg.startMin = 0, 0
		cfg.endHour, cfg.endMin = 23, 59
	}

	if cfg.loc == nil {
		cfg.loc = defaultLocation()
	}
	return cfg
}

func (c businessHoursConfig) isHoliday(t time.Time) bool {
	return c.holidays[t.Format("2006-01-02")]
}

func (c businessHoursConfig) isWorkDay(t time.Time) bool {
	if c.isHoliday(t) {
		return false
	}
	return c.workDays[t.Weekday()]
}

func (c businessHoursConfig) workDayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, c.startHour, c.startMin, 0, 0, t.Location())
}

func (c businessHoursConfig) workDayEnd(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, c.endHour, c.endMin, 0, 0, t.Location())
}

func (c businessHoursConfig) nextWorkDayStart(t time.Time) time.Time {
	next := t.AddDate(0, 0, 1)
	for !c.isWorkDay(next) {
		next = next.AddDate(0, 0, 1)
	}
	return c.workDayStart(next)
}

func adjustToBusinessHoursStart(t time.Time, cfg businessHoursConfig) time.Time {
	for !cfg.isWorkDay(t) {
		t = cfg.nextWorkDayStart(t)
	}
	dayStart := cfg.workDayStart(t)
	dayEnd := cfg.workDayEnd(t)
	if t.Before(dayStart) {
		return dayStart
	}
	if !t.Before(dayEnd) {
		return cfg.nextWorkDayStart(t)
	}
	return t
}

func addBusinessMinutes(start time.Time, minutes int, cfg businessHoursConfig) time.Time {
	if cfg.is24x7 {
		loc := cfg.loc
		if loc == nil {
			loc = defaultLocation()
		}
		return start.In(loc).Add(time.Duration(minutes) * time.Minute)
	}
	remaining := time.Duration(minutes) * time.Minute
	loc := cfg.loc
	if loc == nil {
		loc = defaultLocation()
	}
	cursor := adjustToBusinessHoursStart(start.In(loc), cfg)

	for i := 0; i < 366 && remaining > 0; i++ {
		dayEnd := cfg.workDayEnd(cursor)
		available := dayEnd.Sub(cursor)
		if available <= 0 {
			cursor = cfg.nextWorkDayStart(cursor)
			continue
		}
		if remaining <= available {
			return cursor.Add(remaining)
		}
		remaining -= available
		cursor = cfg.nextWorkDayStart(cursor)
	}
	return cursor
}
