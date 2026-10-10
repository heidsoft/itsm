package cmdb

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// helper: 构造一个 CI 用于测试
func makeCI(id int, status, lifecycle, source, owner string, lastSync time.Time) *ConfigurationItem {
	var syncPtr *time.Time
	if !lastSync.IsZero() {
		syncPtr = &lastSync
	}
	return &ConfigurationItem{
		ID:                 id,
		CINumber:           "CI-TEST-" + strings.Repeat("0", 6-len(itoa(id))) + itoa(id),
		Name:               "ci-" + itoa(id),
		Status:             status,
		LifecycleStatus:    lifecycle,
		DiscoverySource:    source,
		Source:             source,
		OwnedBy:            owner,
		AssignedTo:         owner,
		CloudResourceRefID: id * 10,
		CloudSyncTime:      syncPtr,
		TenantID:           1,
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	digits := []byte{}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

// ----- CanRetireCI 校验 -----

func TestCanRetireCI_Online_Manual_OK(t *testing.T) {
	ci := makeCI(1, "active", "online", "manual", "alice", time.Time{})
	if err := CanRetireCI(ci, "manual"); err != nil {
		t.Fatalf("online + manual reason should retire, got %v", err)
	}
}

func TestCanRetireCI_Retired_Conflict(t *testing.T) {
	ci := makeCI(2, "retired", "retired", "manual", "alice", time.Time{})
	err := CanRetireCI(ci, "manual")
	if !errors.Is(err, ErrInvalidCIRetirementTransition) {
		t.Fatalf("expected ErrInvalidCIRetirementTransition, got %v", err)
	}
}

func TestCanRetireCI_UnknownReason(t *testing.T) {
	ci := makeCI(3, "active", "online", "manual", "alice", time.Time{})
	err := CanRetireCI(ci, "free_text_reason")
	if !errors.Is(err, ErrUnknownRetirementReason) {
		t.Fatalf("expected ErrUnknownRetirementReason, got %v", err)
	}
}

func TestCanRetireCI_Nil(t *testing.T) {
	if err := CanRetireCI(nil, "manual"); err == nil {
		t.Fatal("nil CI should fail")
	}
}

// ----- ApplyCIRetirement 状态机 -----

func TestApplyCIRetirement_HappyPath(t *testing.T) {
	now := time.Now().UTC()
	ci := makeCI(10, "active", "online", "manual", "alice", time.Time{})
	if err := ApplyCIRetirement(ci, CILifecycleRetiring, "manual", now); err != nil {
		t.Fatalf("online->retiring should pass, got %v", err)
	}
	if ci.LifecycleStatus != CILifecycleRetiring {
		t.Fatalf("expected retiring, got %q", ci.LifecycleStatus)
	}
	if err := ApplyCIRetirement(ci, CILifecycleRetired, "manual", now); err != nil {
		t.Fatalf("retiring->retired should pass, got %v", err)
	}
	if ci.LifecycleStatus != CILifecycleRetired {
		t.Fatalf("expected retired, got %q", ci.LifecycleStatus)
	}
	if ci.Status != CILifecycleRetired {
		t.Fatalf("expected status=retired, got %q", ci.Status)
	}
	if !ci.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt not advanced to now")
	}
}

func TestApplyCIRetirement_RestoreFromRetired(t *testing.T) {
	now := time.Now().UTC()
	ci := makeCI(11, "retired", "retired", "manual", "alice", time.Time{})
	if err := ApplyCIRetirement(ci, CILifecycleOnline, "manual", now); err != nil {
		t.Fatalf("retired->online restore should pass, got %v", err)
	}
	if ci.LifecycleStatus != CILifecycleOnline {
		t.Fatalf("expected online, got %q", ci.LifecycleStatus)
	}
}

func TestApplyCIRetirement_RejectsSkipOnlineToRetired(t *testing.T) {
	ci := makeCI(12, "active", "online", "manual", "alice", time.Time{})
	err := ApplyCIRetirement(ci, CILifecycleRetired, "manual", time.Now())
	if !errors.Is(err, ErrInvalidCIRetirementTransition) {
		t.Fatalf("online->retired (skip retiring) must reject, got %v", err)
	}
}

func TestApplyCIRetirement_RejectsUnknownSourceState(t *testing.T) {
	ci := makeCI(13, "active", "ghost", "manual", "alice", time.Time{})
	err := ApplyCIRetirement(ci, CILifecycleRetiring, "manual", time.Now())
	if !errors.Is(err, ErrInvalidCIRetirementTransition) {
		t.Fatalf("unknown source state must reject, got %v", err)
	}
}

// ----- IsStaleCI / IsOrphanCI 判定 -----

func TestIsStaleCI_NilSyncIsNotStale(t *testing.T) {
	ci := makeCI(20, "active", "online", "manual", "alice", time.Time{})
	if IsStaleCI(ci, 30*24*time.Hour, time.Now()) {
		t.Fatal("CI without CloudSyncTime must not be stale")
	}
}

func TestIsStaleCI_RecentSyncNotStale(t *testing.T) {
	now := time.Now().UTC()
	ci := makeCI(21, "active", "online", "manual", "alice", now.Add(-1*time.Hour))
	if IsStaleCI(ci, 24*time.Hour, now) {
		t.Fatal("recently synced CI should not be stale")
	}
}

func TestIsStaleCI_OldSyncIsStale(t *testing.T) {
	now := time.Now().UTC()
	ci := makeCI(22, "active", "online", "aliyun", "alice", now.Add(-100*24*time.Hour))
	if !IsStaleCI(ci, 30*24*time.Hour, now) {
		t.Fatal("CI synced 100d ago with 30d threshold should be stale")
	}
}

func TestIsOrphanCI_AllFieldsEmpty(t *testing.T) {
	ci := &ConfigurationItem{ID: 1}
	if !IsOrphanCI(ci) {
		t.Fatal("fully empty CI should be orphan")
	}
}

func TestIsOrphanCI_AnyOwnerBreaks(t *testing.T) {
	cases := []struct {
		name       string
		mut        func(*ConfigurationItem)
		wantOrphan bool
	}{
		{"owner", func(c *ConfigurationItem) { c.OwnedBy = "alice" }, false},
		{"assignee", func(c *ConfigurationItem) { c.AssignedTo = "alice" }, false},
		{"discovery", func(c *ConfigurationItem) { c.DiscoverySource = "aliyun" }, false},
		{"source", func(c *ConfigurationItem) { c.Source = "manual" }, false},
	}
	for _, tc := range cases {
		ci := &ConfigurationItem{}
		tc.mut(ci)
		if IsOrphanCI(ci) != tc.wantOrphan {
			t.Fatalf("%s: got orphan=%v, want %v", tc.name, IsOrphanCI(ci), tc.wantOrphan)
		}
	}
}

// ----- ComputeQualityMetrics 汇总 -----

func TestComputeQualityMetrics_EmptySet(t *testing.T) {
	m := ComputeQualityMetrics(nil, 30*24*time.Hour, time.Now())
	if m.Total != 0 || m.Active != 0 || m.Retired != 0 || m.CompletenessPct != 0 {
		t.Fatalf("empty set should be zeroed, got %+v", m)
	}
}

func TestComputeQualityMetrics_MixedPopulation(t *testing.T) {
	now := time.Now().UTC()
	cis := []*ConfigurationItem{
		// 完全属性：发现 + owner + 最近同步 → complete
		makeCI(1, "active", "online", "aliyun", "alice", now.Add(-2*time.Hour)),
		// 孤儿：所有元数据为空 → orphan + incomplete
		{ID: 2, LifecycleStatus: "online"},
		// 退役：跳过其它判定，仅计入 retired
		makeCI(3, "retired", "retired", "manual", "bob", now.Add(-2*time.Hour)),
		// 陈旧：超阈值未同步 → stale
		makeCI(4, "active", "online", "manual", "carol", now.Add(-200*24*time.Hour)),
	}
	m := ComputeQualityMetrics(cis, 30*24*time.Hour, now)
	if m.Total != 4 {
		t.Fatalf("Total: got %d want 4", m.Total)
	}
	if m.Active != 3 {
		t.Fatalf("Active: got %d want 3", m.Active)
	}
	if m.Retired != 1 {
		t.Fatalf("Retired: got %d want 1", m.Retired)
	}
	if m.Stale != 1 {
		t.Fatalf("Stale: got %d want 1", m.Stale)
	}
	if m.Orphan != 1 {
		t.Fatalf("Orphan: got %d want 1", m.Orphan)
	}
	if m.Incomplete != 1 {
		t.Fatalf("Incomplete: got %d want 1", m.Incomplete)
	}
	// 4 中 3 完全 → 75.00%
	if m.CompletenessPct != 75.0 {
		t.Fatalf("CompletenessPct: got %v want 75.0", m.CompletenessPct)
	}
}

func TestComputeQualityMetrics_RoundingToTwoDecimals(t *testing.T) {
	now := time.Now().UTC()
	// 3 完全 + 1 incomplete → 75.00%
	cis := []*ConfigurationItem{
		makeCI(1, "active", "online", "aliyun", "alice", now),
		makeCI(2, "active", "online", "aliyun", "bob", now),
		makeCI(3, "active", "online", "aliyun", "carol", now),
		{ID: 4, LifecycleStatus: "online"},
	}
	m := ComputeQualityMetrics(cis, 30*24*time.Hour, now)
	if m.CompletenessPct != 75.0 {
		t.Fatalf("75%% expected, got %v", m.CompletenessPct)
	}
}

// ----- DiffReconciliation 分类 -----

func TestDiffReconciliation_ClassifiesAddAndNoOp(t *testing.T) {
	cis := []*ConfigurationItem{
		{ID: 1, CloudResourceRefID: 100},
	}
	resources := []*CloudResource{
		{ID: 100}, // 已链接
		{ID: 200}, // 新发现
	}
	diffs := DiffReconciliation(cis, resources)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(diffs))
	}
	counts := CountByAction(diffs)
	if counts[ReconcileActionNoOp] != 1 {
		t.Fatalf("noop: got %d want 1", counts[ReconcileActionNoOp])
	}
	if counts[ReconcileActionAdd] != 1 {
		t.Fatalf("add: got %d want 1", counts[ReconcileActionAdd])
	}
}

func TestDiffReconciliation_RetireConfirmForDisappeared(t *testing.T) {
	cis := []*ConfigurationItem{
		{ID: 1, CloudResourceRefID: 100, CINumber: "CI-001"},
		{ID: 2, CloudResourceRefID: 200, CINumber: "CI-002"},
	}
	// 本批次只有 100，200 消失了
	diffs := DiffReconciliation(cis, []*CloudResource{{ID: 100}})
	counts := CountByAction(diffs)
	if counts[ReconcileActionNoOp] != 1 {
		t.Fatalf("noop: got %d want 1", counts[ReconcileActionNoOp])
	}
	if counts[ReconcileActionRetireConfirm] != 1 {
		t.Fatalf("retire_confirm: got %d want 1", counts[ReconcileActionRetireConfirm])
	}
	// 验证 retire_confirm 条目带上了 CI 编号
	found := false
	for _, d := range diffs {
		if d.Action == ReconcileActionRetireConfirm && d.CINumber == "CI-002" {
			found = true
		}
	}
	if !found {
		t.Fatal("retire_confirm entry missing CI number attribution")
	}
}

func TestDiffReconciliation_DetectsDuplicateInBatch(t *testing.T) {
	cis := []*ConfigurationItem{}
	resources := []*CloudResource{
		{ID: 100},
		{ID: 100}, // 重复
	}
	diffs := DiffReconciliation(cis, resources)
	counts := CountByAction(diffs)
	if counts[ReconcileActionAdd] != 1 {
		t.Fatalf("add: got %d want 1", counts[ReconcileActionAdd])
	}
	if counts[ReconcileActionDuplicate] != 1 {
		t.Fatalf("duplicate: got %d want 1", counts[ReconcileActionDuplicate])
	}
}

func TestCountByAction_IgnoresUnknownAction(t *testing.T) {
	entries := []ReconciliationEntry{
		{Action: ReconcileActionAdd},
		{Action: ReconciliationAction("made_up")},
		{Action: ReconcileActionNoOp},
	}
	got := CountByAction(entries)
	if got[ReconcileActionAdd] != 1 {
		t.Fatalf("add: got %d want 1", got[ReconcileActionAdd])
	}
	if got[ReconcileActionNoOp] != 1 {
		t.Fatalf("noop: got %d want 1", got[ReconcileActionNoOp])
	}
	if got[ReconcileActionRetireConfirm] != 0 {
		t.Fatalf("retire_confirm should stay 0, got %d", got[ReconcileActionRetireConfirm])
	}
}

func TestDiffReconciliation_NilEntriesAreSkipped(t *testing.T) {
	cis := []*ConfigurationItem{nil}
	resources := []*CloudResource{nil, {ID: 50}}
	diffs := DiffReconciliation(cis, resources)
	if len(diffs) != 1 {
		t.Fatalf("nil entries should be skipped, got %d", len(diffs))
	}
	if diffs[0].Action != ReconcileActionAdd {
		t.Fatalf("expected add, got %s", diffs[0].Action)
	}
}
