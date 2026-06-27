package service

import (
	"context"
	"sync"
	"testing"
)

// TestEscalationMatrixService_DefaultMatrix 验证默认矩阵覆盖 4 个优先级
func TestEscalationMatrixService_DefaultMatrix(t *testing.T) {
	svc := NewEscalationMatrixService(nil)

	cases := []struct {
		priority  string
		expLevels int
		expFirst  int // AfterMinutes
	}{
		{"critical", 3, 15},
		{"high", 2, 60},
		{"medium", 1, 240},
		{"low", 1, 480},
	}

	for _, tc := range cases {
		t.Run(tc.priority, func(t *testing.T) {
			matrix := svc.GetMatrix(1)
			levels, ok := matrix[tc.priority]
			if !ok {
				t.Fatalf("priority %q not in matrix", tc.priority)
			}
			if len(levels) != tc.expLevels {
				t.Errorf("priority %q: expected %d levels, got %d",
					tc.priority, tc.expLevels, len(levels))
			}
			if levels[0].AfterMinutes != tc.expFirst {
				t.Errorf("priority %q: first level expected %d min, got %d",
					tc.priority, tc.expFirst, levels[0].AfterMinutes)
			}
		})
	}
}

// TestEscalationMatrixService_FindNextEscalationLevel_P1 验证 P1 三级连续升级
func TestEscalationMatrixService_FindNextEscalationLevel_P1(t *testing.T) {
	svc := NewEscalationMatrixService(nil)

	cases := []struct {
		name         string
		elapsedMin   int
		currentLevel int
		expectNil    bool
		expectLevel  int
		expectRole   string
	}{
		{"elapsed=0", 0, 0, true, 0, ""},
		{"elapsed=10", 10, 0, true, 0, ""},
		{"elapsed=14", 14, 0, true, 0, ""},
		{"elapsed=15 -> L1", 15, 0, false, 1, "team_lead"},
		{"elapsed=29 current=L1", 29, 1, true, 0, ""},
		{"elapsed=30 current=L1 -> L2", 30, 1, false, 2, "manager"},
		{"elapsed=59 current=L2", 59, 2, true, 0, ""},
		{"elapsed=60 current=L2 -> L3", 60, 2, false, 3, "director"},
		{"elapsed=120 current=L3 -> nil", 120, 3, true, 0, ""},
	}

	for _, tc := range cases {
		got := svc.FindNextEscalationLevel(1, "critical", tc.elapsedMin, tc.currentLevel)
		if tc.expectNil {
			if got != nil {
				t.Errorf("%s: expected nil, got L%d", tc.name, got.Level)
			}
			continue
		}
		if got == nil {
			t.Errorf("%s: expected non-nil", tc.name)
			continue
		}
		if got.Level != tc.expectLevel {
			t.Errorf("%s: expected L%d, got L%d", tc.name, tc.expectLevel, got.Level)
		}
		if tc.expectRole != "" {
			if len(got.NotifyRoles) == 0 || got.NotifyRoles[0] != tc.expectRole {
				t.Errorf("%s: expected role %q, got %v", tc.name, tc.expectRole, got.NotifyRoles)
			}
		}
	}
}

// TestEscalationMatrixService_FindNextEscalationLevel_MultiLevelSkip 验证跨级升级
// elapsed=60min 已跨 L1(15) L2(30) L3(60)，单次调用应返回 L1（多级由调用方 while 循环）
func TestEscalationMatrixService_FindNextEscalationLevel_MultiLevelSkip(t *testing.T) {
	svc := NewEscalationMatrixService(nil)

	// 第一次调用 elapsed=60, currentMax=0 → 返回 L1
	lvl := svc.FindNextEscalationLevel(1, "critical", 60, 0)
	if lvl == nil || lvl.Level != 1 {
		t.Errorf("expected L1, got %v", lvl)
	}

	// 第二次调用 elapsed=60, currentMax=1 → 返回 L2
	lvl = svc.FindNextEscalationLevel(1, "critical", 60, 1)
	if lvl == nil || lvl.Level != 2 {
		t.Errorf("expected L2, got %v", lvl)
	}

	// 第三次调用 elapsed=60, currentMax=2 → 返回 L3
	lvl = svc.FindNextEscalationLevel(1, "critical", 60, 2)
	if lvl == nil || lvl.Level != 3 {
		t.Errorf("expected L3, got %v", lvl)
	}

	// 第四次调用 elapsed=60, currentMax=3 → 返回 nil
	lvl = svc.FindNextEscalationLevel(1, "critical", 60, 3)
	if lvl != nil {
		t.Errorf("expected nil (已到顶), got L%d", lvl.Level)
	}
}

// TestEscalationMatrixService_FindNextEscalationLevel_UnknownPriority 验证未知 priority 降级到 medium
func TestEscalationMatrixService_FindNextEscalationLevel_UnknownPriority(t *testing.T) {
	svc := NewEscalationMatrixService(nil)

	// 未知 priority + elapsed=240 → 走 medium 兜底，应返回 L1
	lvl := svc.FindNextEscalationLevel(1, "unknown_priority", 240, 0)
	if lvl == nil || lvl.Level != 1 {
		t.Errorf("expected L1 from medium fallback, got %v", lvl)
	}
	if len(lvl.NotifyRoles) == 0 || lvl.NotifyRoles[0] != "team_lead" {
		t.Errorf("expected team_lead role, got %v", lvl.NotifyRoles)
	}
}

// TestEscalationMatrixService_SetMatrix 验证自定义矩阵
func TestEscalationMatrixService_SetMatrix(t *testing.T) {
	svc := NewEscalationMatrixService(nil)

	custom := EscalationMatrix{
		"critical": {
			{Level: 1, AfterMinutes: 5, NotifyRoles: []string{"oncall"}, Description: "on-call"},
			{Level: 2, AfterMinutes: 10, NotifyRoles: []string{"vp"}},
		},
	}
	svc.SetMatrix(42, custom)

	matrix := svc.GetMatrix(42)
	if len(matrix["critical"]) != 2 {
		t.Errorf("custom critical levels: expected 2, got %d", len(matrix["critical"]))
	}
	if matrix["critical"][0].AfterMinutes != 5 {
		t.Errorf("expected AfterMinutes=5, got %d", matrix["critical"][0].AfterMinutes)
	}

	// 其他租户不受影响
	matrix2 := svc.GetMatrix(99)
	if matrix2["critical"][0].AfterMinutes != 15 {
		t.Errorf("other tenant matrix should be default, got L1=%d min",
			matrix2["critical"][0].AfterMinutes)
	}
}

// TestEscalationMatrixService_InvalidateCache 验证缓存清除
func TestEscalationMatrixService_InvalidateCache(t *testing.T) {
	svc := NewEscalationMatrixService(nil)

	svc.SetMatrix(1, EscalationMatrix{
		"critical": {{Level: 1, AfterMinutes: 999}},
	})

	// 第一次取到自定义
	if got := svc.GetMatrix(1)["critical"][0].AfterMinutes; got != 999 {
		t.Errorf("expected 999, got %d", got)
	}

	// 清除缓存
	svc.InvalidateCache(1)

	// 第二次取到默认（15）
	if got := svc.GetMatrix(1)["critical"][0].AfterMinutes; got != 15 {
		t.Errorf("after invalidate expected 15, got %d", got)
	}
}

// TestEscalationMatrixService_CachedMatrixSnapshot 验证快照接口
func TestEscalationMatrixService_CachedMatrixSnapshot(t *testing.T) {
	svc := NewEscalationMatrixService(nil)

	svc.GetMatrix(1)
	svc.GetMatrix(2)
	svc.GetMatrix(2) // 第二次不增加

	snap := svc.CachedMatrixSnapshot()
	if len(snap) != 2 {
		t.Errorf("expected 2 cached tenants, got %d", len(snap))
	}
}

// TestEscalationMatrixService_ConcurrentAccess 验证并发安全
func TestEscalationMatrixService_ConcurrentAccess(t *testing.T) {
	svc := NewEscalationMatrixService(nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = svc.GetMatrix(i % 5)
			_ = svc.FindNextEscalationLevel(i%5, "critical", i, 0)
		}(i)
	}
	wg.Wait()
}

// TestInMemoryEscalationHistoryRecorder 验证内存历史记录器
func TestInMemoryEscalationHistoryRecorder(t *testing.T) {
	rec := NewInMemoryEscalationHistoryRecorder()

	if err := rec.RecordEscalation(context.Background(), EscalationHistoryEntry{
		TicketID: 1, Level: 1, Priority: "critical",
		NotifyRoles: []string{"team_lead"},
	}); err != nil {
		t.Fatalf("RecordEscalation failed: %v", err)
	}

	if err := rec.RecordEscalation(context.Background(), EscalationHistoryEntry{
		TicketID: 1, Level: 2, Priority: "critical",
		NotifyRoles: []string{"manager"},
	}); err != nil {
		t.Fatalf("RecordEscalation 2nd failed: %v", err)
	}

	entries := rec.Entries()
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Level != 1 || entries[1].Level != 2 {
		t.Errorf("expected L1, L2 sequence, got %d, %d", entries[0].Level, entries[1].Level)
	}
}

// TestEscalationMatrix_String 验证 String 格式化
func TestEscalationMatrix_String(t *testing.T) {
	m := EscalationMatrix{
		"critical": {
			{Level: 1, AfterMinutes: 15, NotifyRoles: []string{"team_lead"}},
			{Level: 2, AfterMinutes: 30, NotifyRoles: []string{"manager"}},
		},
	}
	out := m.String("critical")
	if out == "" {
		t.Error("String returned empty")
	}
	// 应包含 critical 关键字
	if !stringHasSubstr(out, "critical") {
		t.Errorf("output missing 'critical': %s", out)
	}

	out2 := m.String("unknown")
	if !stringHasSubstr(out2, "undefined") {
		t.Errorf("unknown priority should be marked undefined: %s", out2)
	}
}

func stringHasSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestNewEscalationService 验证创建时自动初始化 matrix service
func TestNewEscalationService(t *testing.T) {
	es := NewEscalationService(nil, nil)
	if es.MatrixService() == nil {
		t.Error("MatrixService should be auto-initialized")
	}
}

// TestEscalationService_SetMatrixService 验证依赖注入
func TestEscalationService_SetMatrixService(t *testing.T) {
	es := NewEscalationService(nil, nil)
	custom := NewEscalationMatrixService(nil)
	custom.SetMatrix(1, EscalationMatrix{
		"critical": {{Level: 1, AfterMinutes: 999}},
	})
	es.SetMatrixService(custom)

	if got := es.MatrixService().GetMatrix(1)["critical"][0].AfterMinutes; got != 999 {
		t.Errorf("expected custom 999, got %d", got)
	}
}

// TestEscalationService_SetMatrixService_NilSafe 验证 nil 输入安全
func TestEscalationService_SetMatrixService_NilSafe(t *testing.T) {
	es := NewEscalationService(nil, nil)
	es.SetMatrixService(nil) // 不应改变 matrix service
	if es.MatrixService() == nil {
		t.Error("matrix service should not become nil after SetMatrixService(nil)")
	}
}
