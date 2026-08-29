package rls

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"itsm-backend/common/tenantctx"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"go.uber.org/zap"
)

// fakeDriver stubs dialect.Driver so we can test the decorator in isolation.
type fakeDriver struct {
	execCount    int
	queryCount   int
	txCount      int
	lastCtxTID   int
	lastCtxSys   bool
	lastQueryCtx context.Context
	execErr      error
	txErr        error // Tx 返回的 error；nil 表示返回可用的 fakeTx
}

// fakeTx 实现 dialect.Tx：测试中仅需可调用 Exec/Query/Rollback/Commit 即可。
type fakeTx struct{}

func (fakeTx) Exec(_ context.Context, _ string, _, _ any) error  { return nil }
func (fakeTx) Query(_ context.Context, _ string, _, _ any) error { return nil }
func (fakeTx) Commit() error                                     { return nil }
func (fakeTx) Rollback() error                                   { return nil }

func (f *fakeDriver) Dialect() string { return "postgres" }
func (f *fakeDriver) Close() error    { return nil }
func (f *fakeDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	f.txCount++
	if f.txErr != nil {
		return nil, f.txErr
	}
	return fakeTx{}, nil
}

func (f *fakeDriver) Exec(ctx context.Context, query string, args, v any) error {
	f.execCount++
	if tid, ok := tenantctx.TenantID(ctx); ok {
		f.lastCtxTID = tid
	}
	f.lastCtxSys = tenantctx.IsSystemBypass(ctx)
	return f.execErr
}

func (f *fakeDriver) Query(ctx context.Context, query string, args, v any) error {
	f.queryCount++
	f.lastQueryCtx = ctx
	return nil
}

var _ dialect.Driver = (*fakeDriver)(nil)

func TestParseMode(t *testing.T) {
	cases := map[string]Mode{
		"":          ModeOff,
		"off":       ModeOff,
		"OFF":       ModeOff, // unknown values fall back to off
		"shadow":    ModeShadow,
		"enforce":   ModeEnforce,
		"gibberish": ModeOff,
	}
	for in, want := range cases {
		if got := ParseMode(in); got != want {
			t.Errorf("ParseMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDriverOffModeIsPassthrough(t *testing.T) {
	fake := &fakeDriver{}
	d := NewDriver(fake, ModeOff, zap.NewNop().Sugar())
	if d.Dialect() != "postgres" {
		t.Fatalf("Dialect mismatch")
	}
	// No tenant in ctx, no bypass — should NOT warn (mode=off is silent).
	if err := d.Exec(context.Background(), "SELECT 1", nil, nil); err != nil {
		t.Fatalf("Exec err: %v", err)
	}
	if fake.execCount != 1 {
		t.Fatalf("inner Exec not called")
	}
	s := d.Stats()
	if s.QueriesOff != 1 {
		t.Errorf("QueriesOff = %d, want 1", s.QueriesOff)
	}
	if s.MissingTenant != 0 {
		t.Errorf("MissingTenant should stay 0 in off mode, got %d", s.MissingTenant)
	}
}

func TestDriverShadowModeCountsMissingTenant(t *testing.T) {
	fake := &fakeDriver{}
	d := NewDriver(fake, ModeShadow, zap.NewNop().Sugar())

	// 1) query without tenant -> counted as MissingTenant
	_ = d.Query(context.Background(), "SELECT 1", nil, nil)
	// 2) query with tenant -> counted as QueriesShadow
	ctx := tenantctx.WithTenantID(context.Background(), 42)
	_ = d.Query(ctx, "SELECT 2", nil, nil)
	// 3) query with system bypass -> counted as SystemBypass
	sysCtx := tenantctx.WithSystemBypass(context.Background())
	_ = d.Query(sysCtx, "SELECT 3", nil, nil)

	s := d.Stats()
	if s.MissingTenant != 1 {
		t.Errorf("MissingTenant = %d, want 1", s.MissingTenant)
	}
	if s.QueriesShadow != 1 {
		t.Errorf("QueriesShadow = %d, want 1", s.QueriesShadow)
	}
	if s.SystemBypass != 1 {
		t.Errorf("SystemBypass = %d, want 1", s.SystemBypass)
	}
	// Inner driver saw all three queries — shadow mode is pass-through.
	if fake.queryCount != 3 {
		t.Errorf("inner queryCount = %d, want 3", fake.queryCount)
	}
}

func TestDriverEnforceCountsAppliedAndBypass(t *testing.T) {
	fake := &fakeDriver{}
	d := NewDriver(fake, ModeEnforce, zap.NewNop().Sugar())
	ctx := tenantctx.WithTenantID(context.Background(), 7)
	_ = d.Exec(ctx, "UPDATE changes SET title='x'", nil, nil)
	sysCtx := tenantctx.WithSystemBypass(context.Background())
	_ = d.Exec(sysCtx, "SELECT COUNT(*) FROM changes", nil, nil)

	s := d.Stats()
	if s.EnforceApplied != 1 {
		t.Errorf("EnforceApplied = %d, want 1", s.EnforceApplied)
	}
	if s.SystemBypass != 1 {
		t.Errorf("SystemBypass = %d, want 1", s.SystemBypass)
	}
}

func TestDriverExecPropagatesError(t *testing.T) {
	fake := &fakeDriver{execErr: sql.ErrConnDone}
	d := NewDriver(fake, ModeOff, zap.NewNop().Sugar())
	if err := d.Exec(context.Background(), "X", nil, nil); !errors.Is(err, sql.ErrConnDone) {
		t.Fatalf("expected ErrConnDone, got %v", err)
	}
}

func TestFirstToken(t *testing.T) {
	cases := map[string]string{
		"SELECT * FROM x":      "SELECT",
		"INSERT INTO y VALUES": "INSERT",
		"UPDATE\ta SET b=1":    "UPDATE",
		"":                     "",
	}
	for in, want := range cases {
		if got := firstToken(in); got != want {
			t.Errorf("firstToken(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestDriverEnforceQueryInjectsSessionVar 回归用例（2026-08-29 生产就绪评估 P0-3）：
// enforce 模式旧实现把 Query 包进短事务，事务在调用方扫描行之前提交，
// 行句柄失效报 "sql: Rows are closed"，启动就绪探针因此崩溃，RLS 被迫关闭。
// 修复后 Query/Exec 不再开事务，而是通过上下文（entsql.WithVar）把租户变量
// 交给内层驱动，由 ent 在专用池连接上于语句前后 SET/RESET。
func TestDriverEnforceQueryInjectsSessionVar(t *testing.T) {
	fake := &fakeDriver{}
	d := NewDriver(fake, ModeEnforce, zap.NewNop().Sugar())

	ctx := tenantctx.WithTenantID(context.Background(), 42)
	if err := d.Query(ctx, "SELECT 1", nil, nil); err != nil {
		t.Fatalf("Query: %v", err)
	}
	if fake.txCount != 0 {
		t.Errorf("enforce Query must not open short tx, txCount = %d", fake.txCount)
	}
	got, ok := entsql.VarFromContext(fake.lastQueryCtx, tenantVarName)
	if !ok || got != "42" {
		t.Errorf("session var = %q,%v; want \"42\",true", got, ok)
	}

	// bypass 透传且不注入变量
	sysCtx := tenantctx.WithSystemBypass(context.Background())
	if err := d.Query(sysCtx, "SELECT 2", nil, nil); err != nil {
		t.Fatalf("bypass Query: %v", err)
	}
	if _, ok := entsql.VarFromContext(fake.lastQueryCtx, tenantVarName); ok {
		t.Error("bypass query must not carry tenant session var")
	}

	// 缺失租户时 fail-close
	if err := d.Query(context.Background(), "SELECT 3", nil, nil); err == nil {
		t.Error("expected enforce failure for missing tenant")
	}
}
