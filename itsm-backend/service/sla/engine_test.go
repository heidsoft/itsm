package sla

import (
	"context"
	"testing"
	"time"
)

func TestComputeDeadlines_NoBusinessHours(t *testing.T) {
	engine := NewEngine(nil)
	start := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)

	result := engine.ComputeDeadlines(context.Background(), ComputeInput{
		TenantID:       1,
		ResponseTime:   60,
		ResolutionTime: 480,
		StartTime:      start,
	})

	if result.ResponseDeadline == nil {
		t.Fatal("expected response deadline to be set")
	}
	if result.ResolutionDeadline == nil {
		t.Fatal("expected resolution deadline to be set")
	}

	expectedResponse := start.Add(60 * time.Minute)
	if !result.ResponseDeadline.Equal(expectedResponse) {
		t.Errorf("response deadline: got %v, want %v", result.ResponseDeadline, expectedResponse)
	}

	expectedResolution := start.Add(480 * time.Minute)
	if !result.ResolutionDeadline.Equal(expectedResolution) {
		t.Errorf("resolution deadline: got %v, want %v", result.ResolutionDeadline, expectedResolution)
	}
}

func TestComputeDeadlines_BusinessHours_WithinDay(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	engine := NewEngine(func(_ context.Context, _ int) *time.Location {
		return loc
	})

	// Wednesday 10:00 CST
	start := time.Date(2026, 9, 23, 10, 0, 0, 0, loc)
	businessHours := map[string]interface{}{
		"start_time": "09:00",
		"end_time":   "18:00",
		"work_days":  []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)},
	}

	result := engine.ComputeDeadlines(context.Background(), ComputeInput{
		TenantID:      1,
		ResponseTime:  60,
		BusinessHours: businessHours,
		StartTime:     start,
	})

	if result.ResponseDeadline == nil {
		t.Fatal("expected response deadline")
	}

	// 10:00 + 60min = 11:00 same day
	expected := time.Date(2026, 9, 23, 11, 0, 0, 0, loc)
	if !result.ResponseDeadline.Equal(expected) {
		t.Errorf("got %v, want %v", result.ResponseDeadline, expected)
	}
}

func TestComputeDeadlines_BusinessHours_CrossDay(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	engine := NewEngine(func(_ context.Context, _ int) *time.Location {
		return loc
	})

	// Friday 17:30 CST — only 30 min left in the workday
	start := time.Date(2026, 9, 25, 17, 30, 0, 0, loc)
	businessHours := map[string]interface{}{
		"start_time": "09:00",
		"end_time":   "18:00",
		"work_days":  []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)},
	}

	result := engine.ComputeDeadlines(context.Background(), ComputeInput{
		TenantID:      1,
		ResponseTime:  60,
		BusinessHours: businessHours,
		StartTime:     start,
	})

	if result.ResponseDeadline == nil {
		t.Fatal("expected response deadline")
	}

	// 30 min on Friday + 30 min on Monday = Monday 09:30
	expected := time.Date(2026, 9, 28, 9, 30, 0, 0, loc)
	if !result.ResponseDeadline.Equal(expected) {
		t.Errorf("got %v, want %v", result.ResponseDeadline, expected)
	}
}

func TestComputeDeadlines_24x7(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	engine := NewEngine(func(_ context.Context, _ int) *time.Location {
		return loc
	})

	// Saturday 23:00
	start := time.Date(2026, 9, 26, 23, 0, 0, 0, loc)
	businessHours := map[string]interface{}{
		"is_24_7": true,
	}

	result := engine.ComputeDeadlines(context.Background(), ComputeInput{
		TenantID:      1,
		ResponseTime:  120,
		BusinessHours: businessHours,
		StartTime:     start,
	})

	if result.ResponseDeadline == nil {
		t.Fatal("expected response deadline")
	}

	// 23:00 + 120min = Sunday 01:00
	expected := start.Add(120 * time.Minute)
	if !result.ResponseDeadline.Equal(expected) {
		t.Errorf("got %v, want %v", result.ResponseDeadline, expected)
	}
}

func TestComputeDeadlines_AfterHours(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	engine := NewEngine(func(_ context.Context, _ int) *time.Location {
		return loc
	})

	// Wednesday 20:00 — after work hours
	start := time.Date(2026, 9, 23, 20, 0, 0, 0, loc)
	businessHours := map[string]interface{}{
		"start_time": "09:00",
		"end_time":   "18:00",
		"work_days":  []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)},
	}

	result := engine.ComputeDeadlines(context.Background(), ComputeInput{
		TenantID:      1,
		ResponseTime:  60,
		BusinessHours: businessHours,
		StartTime:     start,
	})

	if result.ResponseDeadline == nil {
		t.Fatal("expected response deadline")
	}

	// Should start from next day 09:00, then add 60 min = Thursday 10:00
	expected := time.Date(2026, 9, 24, 10, 0, 0, 0, loc)
	if !result.ResponseDeadline.Equal(expected) {
		t.Errorf("got %v, want %v", result.ResponseDeadline, expected)
	}
}

func TestComputeDeadlines_Holiday(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	engine := NewEngine(func(_ context.Context, _ int) *time.Location {
		return loc
	})

	// Monday is a holiday
	start := time.Date(2026, 9, 21, 10, 0, 0, 0, loc)
	businessHours := map[string]interface{}{
		"start_time":   "09:00",
		"end_time":     "18:00",
		"work_days":    []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)},
		"holiday_list": []interface{}{"2026-09-21"},
	}

	result := engine.ComputeDeadlines(context.Background(), ComputeInput{
		TenantID:      1,
		ResponseTime:  60,
		BusinessHours: businessHours,
		StartTime:     start,
	})

	if result.ResponseDeadline == nil {
		t.Fatal("expected response deadline")
	}

	// Monday is holiday, should skip to Tuesday 09:00 + 60min = 10:00
	expected := time.Date(2026, 9, 22, 10, 0, 0, 0, loc)
	if !result.ResponseDeadline.Equal(expected) {
		t.Errorf("got %v, want %v", result.ResponseDeadline, expected)
	}
}

func TestComputeDeadlines_ZeroTime(t *testing.T) {
	engine := NewEngine(nil)
	start := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)

	result := engine.ComputeDeadlines(context.Background(), ComputeInput{
		TenantID:       1,
		ResponseTime:   0,
		ResolutionTime: 0,
		StartTime:      start,
	})

	if result.ResponseDeadline != nil {
		t.Errorf("expected nil response deadline for zero time, got %v", result.ResponseDeadline)
	}
	if result.ResolutionDeadline != nil {
		t.Errorf("expected nil resolution deadline for zero time, got %v", result.ResolutionDeadline)
	}
}

func TestSLAStatus(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name              string
		responseDeadline  *time.Time
		resolutionDeadline *time.Time
		firstResponseAt   time.Time
		resolvedAt        time.Time
		want              string
	}{
		{
			name:     "ok - within deadline",
			responseDeadline:  timePtr(now.Add(1 * time.Hour)),
			resolutionDeadline: timePtr(now.Add(8 * time.Hour)),
			want: "ok",
		},
		{
			name:     "breached - response deadline passed",
			responseDeadline: timePtr(now.Add(-1 * time.Hour)),
			want:             "breached",
		},
		{
			name:     "warning - response deadline within 30 min",
			responseDeadline: timePtr(now.Add(20 * time.Minute)),
			want:             "warning",
		},
		{
			name:            "ok - responded on time",
			responseDeadline: timePtr(now.Add(1 * time.Hour)),
			firstResponseAt: now.Add(-30 * time.Minute),
			resolutionDeadline: timePtr(now.Add(7 * time.Hour)),
			want: "ok",
		},
		{
			name:              "breached - resolution deadline passed",
			responseDeadline:  timePtr(now.Add(-2 * time.Hour)),
			resolutionDeadline: timePtr(now.Add(-1 * time.Hour)),
			firstResponseAt:   now.Add(-90 * time.Minute),
			want:              "breached",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SLAStatus(now, tt.responseDeadline, tt.resolutionDeadline, tt.firstResponseAt, tt.resolvedAt)
			if got != tt.want {
				t.Errorf("SLAStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseBusinessHoursConfig_LegacyKeys(t *testing.T) {
	raw := map[string]interface{}{
		"workdays":         []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)},
		"work_hour_start":  float64(8),
		"work_hour_end":    float64(17),
		"timezone":         "America/New_York",
	}

	cfg := parseBusinessHoursConfig(raw)

	if cfg.startHour != 8 {
		t.Errorf("startHour: got %d, want 8", cfg.startHour)
	}
	if cfg.endHour != 17 {
		t.Errorf("endHour: got %d, want 17", cfg.endHour)
	}
	if cfg.loc.String() != "America/New_York" {
		t.Errorf("loc: got %v, want America/New_York", cfg.loc)
	}
	if len(cfg.workDays) != 5 {
		t.Errorf("workDays: got %d days, want 5", len(cfg.workDays))
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
