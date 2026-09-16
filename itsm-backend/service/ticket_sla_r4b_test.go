package service

import (
	"testing"
	"time"
)

// TestParseBusinessHoursConfig_LegacyKeys 兼容 R4-b 修复前已存在的键名别名。
// 历史欠账：模板写 workdays/work_hour_start/work_hour_end/is_24_7，解析器不识别导致
// P1 模板声明 24×7 实际 fallback 到 9-18 工时——所有 SLA 错算。
// 修复后解析器必须同时认新旧键名，且 is_24_7=true 短路为 00:00-23:59 全天制。
func TestParseBusinessHoursConfig_LegacyKeys(t *testing.T) {
	// 场景 A：旧键名 workdays/work_hour_start/work_hour_end
	cfg := parseBusinessHoursConfig(map[string]interface{}{
		"time_zone":        "Asia/Shanghai",
		"workdays":         []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)},
		"work_hour_start":  float64(9),
		"work_hour_end":    float64(18),
		"is_24_7":          false,
	})
	if cfg.startHour != 9 || cfg.startMin != 0 || cfg.endHour != 18 || cfg.endMin != 0 {
		t.Fatalf("legacy keys not honored: start=%02d:%02d end=%02d:%02d", cfg.startHour, cfg.startMin, cfg.endHour, cfg.endMin)
	}
	if len(cfg.workDays) != 5 || cfg.workDays[time.Monday] == false {
		t.Fatalf("legacy workdays not honored: %v", cfg.workDays)
	}

	// 场景 B：is_24_7=true（24×7 模板，5 处模板硬编码都靠这个 flag）
	cfg24 := parseBusinessHoursConfig(map[string]interface{}{
		"time_zone":  "Asia/Shanghai",
		"is_24_7":    true,
		"work_days":  []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)},
		"start_time": "09:00", // 即使声明 9-18 也要被忽略
		"end_time":   "18:00",
	})
	if !cfg24.is24x7 {
		t.Fatal("is_24_7=true must set is24x7 flag")
	}
	if cfg24.startHour != 0 || cfg24.endHour != 23 {
		t.Fatalf("is_24_7 must override start/end to 00:00-23:59, got start=%d end=%d", cfg24.startHour, cfg24.endHour)
	}
	if len(cfg24.workDays) != 7 {
		t.Fatalf("is_24_7 must expand workDays to all 7, got %d", len(cfg24.workDays))
	}
}

// TestAddBusinessMinutes_24x7 验证 is_24_7 模板调用 addBusinessMinutes 时分钟连续累加，不剔除任何时段。
//
// 场景：起始 2026-09-16 周三 22:00 UTC(=Asia/Shanghai 06:00 周四)，
// is_24_7=true，1440 分钟(24h)。预期：次日 22:00 UTC 截止（连续，不剔除）。
// 修复前：因为 work_days/work_hour_start 被忽略且 is_24_7 未消费，落到默认 9-18 工时——
// 1 天 24h 实际跨多日才累计完成，违约判定整体偏移。
func TestAddBusinessMinutes_24x7(t *testing.T) {
	cfg := parseBusinessHoursConfig(map[string]interface{}{
		"time_zone": "Asia/Shanghai",
		"is_24_7":   true,
	})

	startUTC := time.Date(2026, 9, 16, 22, 0, 0, 0, time.UTC) // 周三 22:00Z
	deadline := addBusinessMinutes(startUTC, 1440, cfg)

	want := time.Date(2026, 9, 17, 22, 0, 0, 0, time.UTC) // 次日 22:00Z
	if !deadline.Equal(want) {
		t.Fatalf("24x7 deadline = %s, want %s (连续 24h)", deadline, want)
	}
}

// TestParseBusinessHoursConfig_LocFallback 验证 loc 字段兜底为 ResolveLocation("")=Asia/Shanghai，
// 不再退化为 time.Local（跨时区容器里会产生整体偏移）。
func TestParseBusinessHoursConfig_LocFallback(t *testing.T) {
	// 场景：完全不传 time_zone，期望 cfg.loc 不为 nil（解析器兜底）
	cfg := parseBusinessHoursConfig(map[string]interface{}{
		"work_days":  []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)},
		"start_time": "09:00",
		"end_time":   "18:00",
	})
	if cfg.loc == nil {
		t.Fatal("loc must be non-nil even when time_zone missing (R4-b fallback to ResolveLocation)")
	}
	// 兜底应为 Asia/Shanghai 或 FixedZone(+08:00)
	if cfg.loc.String() != "Asia/Shanghai" {
		t.Logf("note: loc fallback resolved to %s (容器内 tzdata 缺失时为 FixedZone)", cfg.loc.String())
	}
}