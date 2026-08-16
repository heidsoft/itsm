package service

import (
	"testing"
	"time"
)

// TestParseBusinessHoursConfig_TimeZone 验证解析 time_zone（兼容 timezone）并设置 loc。
func TestParseBusinessHoursConfig_TimeZone(t *testing.T) {
	// 文档约定键名 time_zone
	cfg := parseBusinessHoursConfig(map[string]interface{}{
		"time_zone":  "Asia/Shanghai",
		"work_days":  []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)},
		"start_time": "09:00",
		"end_time":   "18:00",
	})
	if cfg.loc == nil {
		t.Fatal("expected loc to be set from time_zone")
	}
	sh, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("cannot load Asia/Shanghai: %v", err)
	}
	if cfg.loc.String() != sh.String() {
		t.Fatalf("loc = %s, want %s", cfg.loc, sh)
	}

	// DTO 示例键名 timezone（小写 z）也应被识别
	cfg2 := parseBusinessHoursConfig(map[string]interface{}{
		"timezone": "America/New_York",
	})
	if cfg2.loc == nil || cfg2.loc.String() != "America/New_York" {
		t.Fatalf("expected America/New_York from timezone key, got %v", cfg2.loc)
	}
}

// TestAddBusinessMinutes_HonorsTimeZone 验证跨时区租户的工时窗口按配置时区计算。
//
// 场景：配置 Asia/Shanghai(UTC+8)，工作时段 09:00-18:00，周一至周五。
// 起始时刻 = 2026-08-17(周一) 01:00:00 UTC，即上海 09:00:00。
// 480 分钟(8h) 解决时限，应在上海 17:00 截止 = UTC 09:00 同日。
// 修复前：窗口按宿主机/UTC 计算，01:00Z 被推到 09:00Z 起算，截止 17:00Z（整体偏移 +8h）。
func TestAddBusinessMinutes_HonorsTimeZone(t *testing.T) {
	cfg := parseBusinessHoursConfig(map[string]interface{}{
		"time_zone":  "Asia/Shanghai",
		"work_days":  []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)},
		"start_time": "09:00",
		"end_time":   "18:00",
	})

	startUTC := time.Date(2026, 8, 17, 1, 0, 0, 0, time.UTC) // 上海 09:00
	deadline := addBusinessMinutes(startUTC, 480, cfg)

	want := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC) // 上海 17:00 = UTC 09:00
	if !deadline.Equal(want) {
		t.Fatalf("deadline = %s, want %s (上海 17:00)", deadline, want)
	}
}
