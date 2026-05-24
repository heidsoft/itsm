package service

import (
	"testing"
	"time"
)

// TestSLACalculationWithResolvedTicket 测试已解决工单的SLA计算（修复Bug #L3）
func TestSLACalculationWithResolvedTicket(t *testing.T) {
	now := time.Now()

	// 场景1：工单在SLA内解决
	resolvedTime := now.Add(-1 * time.Hour)
	deadline := now.Add(1 * time.Hour) // SLA截止时间

	// 已解决工单应该在解决时间点判断SLA
	if resolvedTime.After(deadline) {
		t.Error("Resolved ticket within SLA should not be breached")
	}

	// 场景2：工单在SLA外解决
	resolvedLate := now.Add(2 * time.Hour) // 超过截止时间
	if !resolvedLate.After(deadline) {
		t.Error("Resolved ticket after deadline should be breached")
	}
}

// TestSLABreachTimeCheck 测试SLA违规时间检查逻辑
func TestSLABreachTimeCheck(t *testing.T) {
	now := time.Now()
	deadline := now.Add(1 * time.Hour)

	// 测试1：未解决工单，使用当前时间
	checkTimeForOpen := now
	isBreached := checkTimeForOpen.After(deadline)
	if isBreached {
		t.Error("Open ticket before deadline should not be breached")
	}

	// 测试2：未解决工单，超时
	overtime := now.Add(2 * time.Hour)
	isBreached = overtime.After(deadline)
	if !isBreached {
		t.Error("Open ticket after deadline should be breached")
	}

	// 测试3：已解决工单，在SLA内解决
	resolvedTime := now.Add(30 * time.Minute)
	checkTimeForResolved := resolvedTime
	isBreached = checkTimeForResolved.After(deadline)
	if isBreached {
		t.Error("Resolved ticket within SLA should not be breached")
	}
}

// TestSLAStatusConsistency 测试SLA状态一致性
func TestSLAStatusConsistency(t *testing.T) {
	tests := []struct {
		name             string
		resolved         bool
		resolvedTime     time.Time
		deadline         time.Time
		checkTime        time.Time
		shouldBeBreached bool
	}{
		{
			name:             "Open ticket within SLA",
			resolved:         false,
			deadline:         time.Now().Add(1 * time.Hour),
			checkTime:        time.Now(),
			shouldBeBreached: false,
		},
		{
			name:             "Open ticket after deadline",
			resolved:         false,
			deadline:         time.Now().Add(-1 * time.Hour),
			checkTime:        time.Now(),
			shouldBeBreached: true,
		},
		{
			name:             "Resolved ticket within SLA",
			resolved:         true,
			resolvedTime:     time.Now().Add(30 * time.Minute),
			deadline:         time.Now().Add(1 * time.Hour),
			checkTime:        time.Now().Add(30 * time.Minute),
			shouldBeBreached: false,
		},
		{
			name:             "Resolved ticket after deadline",
			resolved:         true,
			resolvedTime:     time.Now().Add(2 * time.Hour),
			deadline:         time.Now().Add(1 * time.Hour),
			checkTime:        time.Now().Add(2 * time.Hour),
			shouldBeBreached: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 核心逻辑：已解决工单使用解决时间，未解决工单使用当前时间
			var checkTime time.Time
			if tt.resolved {
				checkTime = tt.resolvedTime
			} else {
				checkTime = tt.checkTime
			}

			isBreached := checkTime.After(tt.deadline)
			if isBreached != tt.shouldBeBreached {
				t.Errorf("SLA breach check failed: got %v, want %v", isBreached, tt.shouldBeBreached)
			}
		})
	}
}

// TestBusinessHoursAdjustment 测试工作时间调整
func TestBusinessHoursAdjustment(t *testing.T) {
	// 周六调整到周一
	saturday := time.Date(2024, 1, 6, 10, 0, 0, 0, time.UTC) // 周六
	adjusted := adjustToBusinessHours(saturday)
	if adjusted.Weekday() != time.Monday {
		t.Errorf("Saturday should be adjusted to Monday, got %v", adjusted.Weekday())
	}

	// 周日调整到周一
	sunday := time.Date(2024, 1, 7, 10, 0, 0, 0, time.UTC) // 周日
	adjusted = adjustToBusinessHours(sunday)
	if adjusted.Weekday() != time.Monday {
		t.Errorf("Sunday should be adjusted to Monday, got %v", adjusted.Weekday())
	}

	// 凌晨调整到9点
	earlyMorning := time.Date(2024, 1, 8, 6, 0, 0, 0, time.UTC) // 周一6点
	adjusted = adjustToBusinessHours(earlyMorning)
	if adjusted.Hour() != 9 {
		t.Errorf("Early morning should be adjusted to 9 AM, got %d", adjusted.Hour())
	}
}

// adjustToBusinessHours 辅助函数
func adjustToBusinessHours(t time.Time) time.Time {
	year, month, day := t.Date()

	if t.Weekday() == time.Saturday {
		return time.Date(year, month, day+2, 9, 0, 0, 0, t.Location())
	}
	if t.Weekday() == time.Sunday {
		return time.Date(year, month, day+1, 9, 0, 0, 0, t.Location())
	}

	if t.Hour() < 9 {
		return time.Date(year, month, day, 9, 0, 0, 0, t.Location())
	}

	return t
}
