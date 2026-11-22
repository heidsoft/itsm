package initialization

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/require"
)

// managedInitializer 是带显式托管稳定键的测试组件。testInitializer 的 Plan
// 不带 Actions，无法驱动托管账本写入，故单独定义。
type managedInitializer struct {
	name     string
	actions  []Action
	checksum string
	version  string
	apply    func(context.Context, Scope, Plan, dialect.Driver) (Result, error)
}

func (i *managedInitializer) Name() string           { return i.name }
func (i *managedInitializer) Dependencies() []string { return nil }
func (i *managedInitializer) Plan(context.Context, Scope) (Plan, error) {
	version, checksum := i.version, i.checksum
	if version == "" {
		version = "1.0.0"
	}
	if checksum == "" {
		checksum = i.name + "-checksum"
	}
	return Plan{
		Component:      i.name,
		TargetVersion:  version,
		SourceChecksum: checksum,
		Actions:        i.actions,
	}, nil
}
func (i *managedInitializer) Apply(ctx context.Context, scope Scope, plan Plan, d dialect.Driver) (Result, error) {
	if i.apply != nil {
		return i.apply(ctx, scope, plan, d)
	}
	return Result{Summary: map[string]any{"applied": i.name}}, nil
}
func (i *managedInitializer) Verify(context.Context, Scope, Plan) error { return nil }

func managedActions(keys ...string) []Action {
	out := make([]Action, 0, len(keys))
	for _, k := range keys {
		out = append(out, Action{Type: "reconcile", SourceKey: k, Summary: "managed " + k})
	}
	return out
}

func newManagedEngine(t *testing.T, store Store, comps ...Initializer) *Engine {
	t.Helper()
	engine, err := NewEngine(store, comps, time.Hour)
	require.NoError(t, err)
	return engine
}

func requireManagedRecord(t *testing.T, records []ManagedRecord, component, sourceKey string) ManagedRecord {
	t.Helper()
	for _, rec := range records {
		if rec.Component == component && rec.SourceKey == sourceKey {
			return rec
		}
	}
	t.Fatalf("managed record %s/%s not found in %+v", component, sourceKey, records)
	return ManagedRecord{}
}

// V1/V6：组件成功后托管记录按 scope 可读、字段与 plan/result 一致；空 Actions 不写行。
func TestPostgresManagedRecordsRegisteredOnSuccess(t *testing.T) {
	f := newInitializationPostgresFixture(t)
	request := initializationRequest("executor-managed")
	const component = "rbac"
	engine := newManagedEngine(t, f.store, &managedInitializer{
		name:    component,
		actions: managedActions("role:admin", "permission:ticket:read"),
		version: "1.2.3",
	})
	_, err := engine.Apply(f.ctx, request)
	require.NoError(t, err)

	records, err := f.second.ListManagedRecords(f.ctx, request.Scope)
	require.NoError(t, err)
	require.Len(t, records, 2, "V1: both stable keys registered")

	admin := requireManagedRecord(t, records, component, "role:admin")
	require.Equal(t, request.Scope.Type, admin.ScopeType)
	require.Equal(t, request.Scope.ID, admin.ScopeID)
	require.Equal(t, "1.2.3", admin.SourceVersion)
	require.Equal(t, component+"-checksum", admin.ManifestChecksum)
	require.Equal(t, "managed", admin.OwnershipMode)
	require.JSONEq(t, "[]", string(admin.ManagedFields), "V1: fields-level ownership not fabricated in batch 1")
	require.JSONEq(t, `{"applied":"rbac"}`, string(admin.LastAppliedValues), "V1: last applied values are the real apply summary")
	require.False(t, admin.Deprecated)
	require.False(t, admin.LocalModifiedAt.Valid, "V1: local-modification detection is out of scope, stays NULL")

	_ = requireManagedRecord(t, records, component, "permission:ticket:read")
}

func TestPostgresManagedRecordsEmptyActionsWriteNothing(t *testing.T) {
	f := newInitializationPostgresFixture(t)
	engine := newManagedEngine(t, f.store, &managedInitializer{name: "rbac"})
	_, err := engine.Apply(f.ctx, initializationRequest("executor-empty"))
	require.NoError(t, err)
	records, err := f.second.ListManagedRecords(f.ctx, Scope{Type: "tenant", ID: 42})
	require.NoError(t, err)
	require.Empty(t, records, "V6: no Actions -> no ledger noise rows")
}

// V2/V3：重放幂等（行数不变、checksum/version 更新）；新 source_key 累加而不误删旧键。
func TestPostgresManagedRecordsReplayIsIdempotentAndAccumulates(t *testing.T) {
	f := newInitializationPostgresFixture(t)
	request := initializationRequest("executor-replay")
	comp := &managedInitializer{
		name:    "rbac",
		actions: managedActions("role:admin"),
		version: "1.0.0",
	}
	_, err := newManagedEngine(t, f.store, comp).Apply(f.ctx, request)
	require.NoError(t, err)

	// Same key, new version/checksum -> update in place, no duplicate row.
	comp.version = "2.0.0"
	comp.actions = managedActions("role:admin")
	comp.apply = func(context.Context, Scope, Plan, dialect.Driver) (Result, error) {
		return Result{Summary: map[string]any{"applied": "rbac", "v": 2}}, nil
	}
	_, err = newManagedEngine(t, f.store, comp).Apply(f.ctx, request)
	require.NoError(t, err)

	records, err := f.second.ListManagedRecords(f.ctx, request.Scope)
	require.NoError(t, err)
	require.Len(t, records, 1, "V2: replay must not duplicate the row")
	admin := requireManagedRecord(t, records, "rbac", "role:admin")
	require.Equal(t, "2.0.0", admin.SourceVersion, "V2: version updated on replay")
	require.JSONEq(t, `{"applied":"rbac","v":2}`, string(admin.LastAppliedValues), "V2: applied values updated")

	// Different source key -> accumulate (old key preserved).
	comp.actions = managedActions("role:manager")
	_, err = newManagedEngine(t, f.store, comp).Apply(f.ctx, request)
	require.NoError(t, err)
	records, err = f.second.ListManagedRecords(f.ctx, request.Scope)
	require.NoError(t, err)
	require.Len(t, records, 2, "V3: new source key accumulates, old one kept")
	_ = requireManagedRecord(t, records, "rbac", "role:admin")
	_ = requireManagedRecord(t, records, "rbac", "role:manager")
}

// V4：跨 scope 不可见——租户与平台、两个不同租户互不串数据。
func TestPostgresManagedRecordsAreScopeIsolated(t *testing.T) {
	f := newInitializationPostgresFixture(t)
	tenantA := Scope{Type: "tenant", ID: 11}
	tenantB := Scope{Type: "tenant", ID: 22}
	platform := Scope{Type: "platform", ID: 0}

	for _, scope := range []Scope{tenantA, tenantB, platform} {
		comp := &managedInitializer{name: "rbac", actions: managedActions("role:admin")}
		engine, err := NewEngine(f.store, []Initializer{comp}, time.Hour)
		require.NoError(t, err)
		_, err = engine.Apply(f.ctx, Request{
			Scope: scope, TargetVersion: "1", ReleaseVersion: "v1",
			RequestedBy: "test", ExecutorID: "executor-" + fmt.Sprint(scope.ID),
		})
		require.NoError(t, err)
	}

	for _, scope := range []Scope{tenantA, tenantB, platform} {
		records, err := f.second.ListManagedRecords(f.ctx, scope)
		require.NoError(t, err)
		require.Len(t, records, 1, "V4: exactly one record per scope")
		require.Equal(t, scope.Type, records[0].ScopeType)
		require.Equal(t, scope.ID, records[0].ScopeID)
	}
}

// V5：账本写入失败 → Complete 失败 → 整笔回滚，安装行不置 succeeded。
// 注入方式：source_key 超过 varchar(255) 触发真实 DB 约束错误。
func TestPostgresManagedRecordWriteFailureRollsBackInstallation(t *testing.T) {
	f := newInitializationPostgresFixture(t)
	request := initializationRequest("executor-fail")
	tooLong := strings.Repeat("k", 256)
	comp := &managedInitializer{name: "rbac", actions: managedActions(tooLong)}
	_, err := newManagedEngine(t, f.store, comp).Apply(f.ctx, request)
	require.Error(t, err, "V5: oversized source key must fail the component")
	require.Contains(t, err.Error(), "record managed source")

	statuses, err := f.second.Status(f.ctx, request.Scope)
	require.NoError(t, err)
	require.Len(t, statuses, 1)
	require.NotEqual(t, "succeeded", statuses[0].Status, "V5: ledger failure must not mark installation succeeded")
	require.Empty(t, statuses[0].InstalledVersion, "V5: no partial success version")

	records, err := f.second.ListManagedRecords(f.ctx, request.Scope)
	require.NoError(t, err)
	require.Empty(t, records, "V5: failed transaction leaves no ledger row")
}

// V7：scope 校验——拒绝不合法组合，绝不退化为全表扫。
func TestListManagedRecordsRejectsInvalidScope(t *testing.T) {
	store, err := NewSQLStore(nil)
	require.Error(t, err, "nil db rejected")
	require.Nil(t, store)

	for _, scope := range []Scope{
		{Type: "platform", ID: 7}, // platform 必须 id=0
		{Type: "tenant", ID: 0},   // tenant 必须 id>0
		{Type: "tenant", ID: -1},  // 负 id
		{Type: "batch", ID: 0},    // 不支持的 scope 类型
		{Type: "", ID: 0},
	} {
		// NewSQLStore 需要非 nil db；这里用最小占位驱动只为走 ValidateScope 前置检查，
		// 不会执行任何查询——非法 scope 必须在触碰数据库前被拒绝。
		s := &SQLStore{}
		_, err := s.ListManagedRecords(context.Background(), scope)
		require.Error(t, err, "V7: invalid scope %+v must be rejected", scope)
	}
}

// 反向验证锚点：读契约对 JSON 列不解释、原样投影（字段级语义演进不需改读面）。
func TestManagedRecordJSONColumnsAreOpaque(t *testing.T) {
	f := newInitializationPostgresFixture(t)
	request := initializationRequest("executor-json")
	comp := &managedInitializer{
		name:    "rbac",
		actions: managedActions("role:admin"),
		apply: func(context.Context, Scope, Plan, dialect.Driver) (Result, error) {
			return Result{Summary: map[string]any{
				"nested": map[string]any{"k": 1},
				"flag":   true,
			}}, nil
		},
	}
	_, err := newManagedEngine(t, f.store, comp).Apply(f.ctx, request)
	require.NoError(t, err)
	records, err := f.second.ListManagedRecords(f.ctx, request.Scope)
	require.NoError(t, err)
	require.Len(t, records, 1)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(records[0].LastAppliedValues, &decoded))
	require.Equal(t, true, decoded["flag"])
	require.NotNil(t, decoded["nested"], "JSON columns are raw-projected, not flattened")
}
