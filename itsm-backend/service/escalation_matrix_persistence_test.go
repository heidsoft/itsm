package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
)

// fakeEscalationStore 是 EscalationMatrixStore 的内存实现，用于验证
// 「先落库再更缓存」的语义而不依赖真实 DB。
type fakeEscalationStore struct {
	mu      sync.Mutex
	data    map[int]EscalationMatrix
	saved   []int // 记录 Save 被调用的租户
	saveErr error
	loadErr error
	deletes int
}

func newFakeEscalationStore() *fakeEscalationStore {
	return &fakeEscalationStore{data: map[int]EscalationMatrix{}}
}

func (f *fakeEscalationStore) Load(_ context.Context, tenantID int) (EscalationMatrix, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	m, ok := f.data[tenantID]
	if !ok {
		return nil, nil // 未配置
	}
	return cloneMatrix(m), nil
}

func (f *fakeEscalationStore) Save(_ context.Context, tenantID int, matrix EscalationMatrix) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	f.data[tenantID] = cloneMatrix(matrix)
	f.saved = append(f.saved, tenantID)
	return nil
}

func (f *fakeEscalationStore) Delete(_ context.Context, tenantID int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, tenantID)
	f.deletes++
	return nil
}

// TestEscalationMatrixService_PersistsAcrossInstances 验证核心修复目标：
// 写入后**新进程/新实例**（空缓存）能读到同一份配置。
//
// 修复前 SetMatrix 只写内存 map，重启即丢；本用例是新实例读旧数据的回归锁。
func TestEscalationMatrixService_PersistsAcrossInstances(t *testing.T) {
	store := newFakeEscalationStore()
	custom := EscalationMatrix{
		"critical": {{Level: 1, AfterMinutes: 5, NotifyRoles: []string{"it_admin"}, Description: "5分钟升级"}},
	}

	// 第一个实例：写入
	w := NewEscalationMatrixServiceWithStore(zap.NewNop().Sugar(), store)
	if err := w.SetMatrix(7, custom); err != nil {
		t.Fatalf("SetMatrix 返回错误: %v", err)
	}

	// 第二个实例：全新空缓存，模拟进程重启
	r := NewEscalationMatrixServiceWithStore(zap.NewNop().Sugar(), store)
	got := r.GetMatrix(7)

	levels, ok := got["critical"]
	if !ok {
		t.Fatalf("重启后未读到 custom critical 链，实际 keys=%v", keysOf(got))
	}
	if len(levels) != 1 || levels[0].AfterMinutes != 5 {
		t.Errorf("重启后读到错误的级别链: %+v", levels)
	}
}

// TestEscalationMatrixService_SaveFailureKeepsCache 验证落库失败时**不改缓存**。
//
// 这是防"接口返回成功但重启丢配置"的关键：Save 失败必须返回 error，
// 且缓存保持原值（此处是默认矩阵），而不是把未持久化的值写进缓存。
func TestEscalationMatrixService_SaveFailureKeepsCache(t *testing.T) {
	store := newFakeEscalationStore()
	store.saveErr = errors.New("db down")

	svc := NewEscalationMatrixServiceWithStore(zap.NewNop().Sugar(), store)
	before := svc.GetMatrix(1) // 预热缓存为默认矩阵

	err := svc.SetMatrix(1, EscalationMatrix{"low": {{Level: 1, AfterMinutes: 999}}})
	if err == nil {
		t.Fatal("落库失败时 SetMatrix 应返回 error，而不是静默成功")
	}

	after := svc.GetMatrix(1)
	// 注意：断言"值未被污染"而不是"键不存在"——默认矩阵本来就含 low，
	// 键存在是正常的，要看的是它仍是默认的 480 而不是落库失败的 999。
	if len(after["low"]) != len(before["low"]) || after["low"][0].AfterMinutes != before["low"][0].AfterMinutes {
		t.Errorf("缓存应保持落库前状态: before=%v after=%v", before["low"], after["low"])
	}
	if after["low"][0].AfterMinutes == 999 {
		t.Error("缓存被污染：未持久化的自定义值进入了缓存")
	}
}

// TestEscalationMatrixService_LoadErrorFallsBackToDefault 验证 DB 读失败时
// 降级到默认矩阵而不是返回空矩阵。
//
// 升级是 SLA 兜底路径：因 DB 抖动让升级判定停摆，比用默认矩阵更糟。
func TestEscalationMatrixService_LoadErrorFallsBackToDefault(t *testing.T) {
	store := newFakeEscalationStore()
	store.loadErr = errors.New("db timeout")

	svc := NewEscalationMatrixServiceWithStore(zap.NewNop().Sugar(), store)
	got := svc.GetMatrix(42)

	if len(got) == 0 {
		t.Fatal("读失败时不应返回空矩阵")
	}
	for _, p := range []string{"critical", "high", "medium", "low"} {
		if _, ok := got[p]; !ok {
			t.Errorf("读失败降级时缺少默认优先级 %q，实际 keys=%v", p, keysOf(got))
		}
	}
}

// TestEscalationMatrixService_UnconfiguredFallsBackToDefault 验证未配置租户
// （store 返回 nil,nil）回落默认矩阵。
func TestEscalationMatrixService_UnconfiguredFallsBackToDefault(t *testing.T) {
	store := newFakeEscalationStore() // 空 data → Load 返回 (nil, nil)
	svc := NewEscalationMatrixServiceWithStore(zap.NewNop().Sugar(), store)

	got := svc.GetMatrix(99)
	if _, ok := got["critical"]; !ok {
		t.Errorf("未配置租户应回落默认矩阵，实际 keys=%v", keysOf(got))
	}
}

// TestEscalationMatrixService_InvalidateCacheReloadsFromStore 验证缓存失效后
// 从 store 重新加载，而不是回落到默认。
func TestEscalationMatrixService_InvalidateCacheReloadsFromStore(t *testing.T) {
	store := newFakeEscalationStore()
	svc := NewEscalationMatrixServiceWithStore(zap.NewNop().Sugar(), store)

	if err := svc.SetMatrix(3, EscalationMatrix{"high": {{Level: 1, AfterMinutes: 7}}}); err != nil {
		t.Fatalf("SetMatrix: %v", err)
	}
	svc.InvalidateCache(3)

	got := svc.GetMatrix(3)
	if len(got["high"]) != 1 || got["high"][0].AfterMinutes != 7 {
		t.Errorf("失效后应从 store 重载自定义值，实际=%+v", got["high"])
	}
}

// TestEscalationMatrixService_NilStoreKeepsMemoryBehavior 验证未注入 store 时
// （旧构造函数路径、测试与降级）仍保持纯内存语义，不 panic。
func TestEscalationMatrixService_NilStoreKeepsMemoryBehavior(t *testing.T) {
	svc := NewEscalationMatrixService(zap.NewNop().Sugar())

	if err := svc.SetMatrix(5, EscalationMatrix{"low": {{Level: 1, AfterMinutes: 11}}}); err != nil {
		t.Fatalf("nil store 下 SetMatrix 应成功: %v", err)
	}
	if got := svc.GetMatrix(5); got["low"][0].AfterMinutes != 11 {
		t.Errorf("nil store 下应内存生效，实际=%+v", got["low"])
	}
}

// TestEscalationMatrixService_GetMatrixReturnsCopy 验证返回值与缓存隔离：
// 调用方改动返回值不得污染服务内部缓存。
func TestEscalationMatrixService_GetMatrixReturnsCopy(t *testing.T) {
	store := newFakeEscalationStore()
	svc := NewEscalationMatrixServiceWithStore(zap.NewNop().Sugar(), store)

	first := svc.GetMatrix(1)
	first["critical"] = []EscalationLevel{{Level: 99, AfterMinutes: 999}}

	second := svc.GetMatrix(1)
	if second["critical"][0].AfterMinutes == 999 {
		t.Error("调用方改动了返回值，缓存被污染")
	}
}

// keysOf 取矩阵的优先级 key 列表，用于失败信息。
func keysOf(m EscalationMatrix) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestEscalationMatrixService_NotifyTargetsSerializeAsArrays 锁定 HTTP 响应契约：
// 未配置通知目标的级别返回空数组而不是 null。handler 直接序列化矩阵，
// null 会让前端每一处取值都得再判一次空。
func TestEscalationMatrixService_NotifyTargetsSerializeAsArrays(t *testing.T) {
	store := newFakeEscalationStore()
	store.data[4242] = EscalationMatrix{
		"critical": {{Level: 1, AfterMinutes: 15, Description: "只配了时长"}},
	}
	svc := NewEscalationMatrixServiceWithStore(zap.NewNop().Sugar(), store)

	raw, err := json.Marshal(svc.GetMatrix(4242))
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	got := string(raw)
	for _, want := range []string{`"notifyRoles":[]`, `"notifyUserIDs":[]`} {
		if !strings.Contains(got, want) {
			t.Errorf("响应缺少 %s: %s", want, got)
		}
	}
	if strings.Contains(got, "null") {
		t.Errorf("通知目标不得序列化为 null: %s", got)
	}

	// 未配置自定义矩阵时回落默认，同样不得出现 null。
	defaultRaw, err := json.Marshal(NewEscalationMatrixService(zap.NewNop().Sugar()).GetMatrix(7))
	if err != nil {
		t.Fatalf("序列化默认矩阵失败: %v", err)
	}
	if strings.Contains(string(defaultRaw), "null") {
		t.Errorf("默认矩阵序列化为 null: %s", defaultRaw)
	}
}
