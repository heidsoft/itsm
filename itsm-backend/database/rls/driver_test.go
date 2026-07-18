package rls

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"itsm-backend/common/tenantctx"

	"entgo.io/ent/dialect"
	"go.uber.org/zap"
)

// fakeDriver stubs dialect.Driver so we can test the decorator in isolation.
type fakeDriver struct {
	execCount   int
	queryCount  int
	txCount     int
	lastCtxTID  int
	lastCtxSys  bool
	execErr     error
}

func (f *fakeDriver) Dialect() string { return "postgres" }
func (f *fakeDriver) Close() error    { return nil }
func (f *fakeDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	f.txCount++
	return nil, errors.New("tx not used in test")
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
	return nil
}

var _ dialect.Driver = (*fakeDriver)(nil)

func TestParseMode(t *testing.T) {
	cases := map[string]Mode{
		"":             ModeOff,
		"off":          ModeOff,
		"OFF":          ModeOff, // unknown values fall back to off
		"shadow":       ModeShadow,
		"enforce":      ModeEnforce,
		"gibberish":    ModeOff,
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
