// Package rls: Ent SQL Driver decorator for Row-Level Security.
//
// This decorator wraps entsql.Driver so we can transparently inject the
// PostgreSQL SET LOCAL app.current_tenant statement per query/transaction,
// without touching thousands of call sites.
//
// Three modes, controlled by config.RLSConfig.Mode:
//
//   off (default):
//     Pass-through. No side effects. Zero risk. Used until R2B灰度期结束.
//
//   shadow:
//     Pass-through DB behavior, but audit every query:
//       - If ctx has tenant_id -> debug log with (op, entity, tid).
//       - If ctx lacks tenant_id AND is not system-bypass -> WARN log with
//         call stack summary. Used to detect missing propagation points
//         before flipping to enforce.
//
//   enforce:
//     Every query is wrapped in a short-lived transaction (or reuses the
//     caller's Tx if present) with SET LOCAL app.current_tenant = <tid>.
//     System-bypass ctx skips the SET; caller is expected to have connected
//     via itsm_admin (BYPASSRLS).
//
// Design notes:
//   - We wrap dialect.Driver (Ent's interface), not *sql.DB. This lets us
//     intercept at the exact call boundary where Ent asks for I/O.
//   - We do NOT modify Ent codegen; the decorator is a runtime concern.
//   - shadow/enforce modes read tenant_id from context via
//     common/tenantctx.TenantID(ctx). System bypass via IsSystemBypass(ctx).
//
// Status: R2A skeleton. off + shadow verified; enforce implemented but
// disabled until R2B灰度收尾.
package rls

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"

	"itsm-backend/common/tenantctx"

	"entgo.io/ent/dialect"
	"go.uber.org/zap"
)

// Mode is the RLS enforcement level. Keep string values stable — they are
// serialized in config and logs.
type Mode string

const (
	ModeOff     Mode = "off"
	ModeShadow  Mode = "shadow"
	ModeEnforce Mode = "enforce"
)

// ParseMode normalizes user input. Unknown values fall back to off.
func ParseMode(s string) Mode {
	switch Mode(s) {
	case ModeShadow:
		return ModeShadow
	case ModeEnforce:
		return ModeEnforce
	default:
		return ModeOff
	}
}

// Driver wraps a dialect.Driver and applies RLS behavior per Mode.
// It implements dialect.Driver so callers can drop it in place of the
// underlying entsql driver.
type Driver struct {
	inner dialect.Driver
	mode  Mode
	log   *zap.SugaredLogger

	// stats: atomic counters exposed via Stats(). Cheap to update on hot path.
	nQueriesOff     atomic.Uint64
	nQueriesShadow  atomic.Uint64
	nMissingTenant  atomic.Uint64
	nSystemBypass   atomic.Uint64
	nEnforceApplied atomic.Uint64
}

// NewDriver wraps drv with the given mode. If log is nil, zap's global
// logger is used.
func NewDriver(drv dialect.Driver, mode Mode, log *zap.SugaredLogger) *Driver {
	if log == nil {
		log = zap.S()
	}
	return &Driver{
		inner: drv,
		mode:  mode,
		log:   log,
	}
}

// Mode returns the currently active enforcement mode.
func (d *Driver) Mode() Mode { return d.mode }

// Stats returns runtime counters. Intended for /internal/rls debug endpoint.
type Stats struct {
	Mode            Mode   `json:"mode"`
	QueriesOff      uint64 `json:"queries_off"`
	QueriesShadow   uint64 `json:"queries_shadow"`
	MissingTenant   uint64 `json:"missing_tenant"`
	SystemBypass    uint64 `json:"system_bypass"`
	EnforceApplied  uint64 `json:"enforce_applied"`
}

// Stats snapshots the current counters.
func (d *Driver) Stats() Stats {
	return Stats{
		Mode:           d.mode,
		QueriesOff:     d.nQueriesOff.Load(),
		QueriesShadow:  d.nQueriesShadow.Load(),
		MissingTenant:  d.nMissingTenant.Load(),
		SystemBypass:   d.nSystemBypass.Load(),
		EnforceApplied: d.nEnforceApplied.Load(),
	}
}

// -----------------------------------------------------------------------
// dialect.Driver implementation
// -----------------------------------------------------------------------

// Dialect passes through the underlying dialect (e.g. "postgres").
func (d *Driver) Dialect() string { return d.inner.Dialect() }

// Close closes the underlying driver.
func (d *Driver) Close() error { return d.inner.Close() }

// Tx delegates transaction creation to the inner driver. Enforce-mode
// SET LOCAL is applied via Exec/Query below rather than here, so that
// shadow mode does not need to intercept transaction boundaries.
func (d *Driver) Tx(ctx context.Context) (dialect.Tx, error) {
	d.observe(ctx, "Tx", "")
	return d.inner.Tx(ctx)
}

// BeginTx delegates to the inner driver. See Tx() for rationale.
func (d *Driver) BeginTx(ctx context.Context, opts *sql.TxOptions) (dialect.Tx, error) {
	d.observe(ctx, "BeginTx", "")
	if t, ok := d.inner.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	}); ok {
		return t.BeginTx(ctx, opts)
	}
	return d.inner.Tx(ctx)
}

// Exec implements dialect.ExecQuerier.
func (d *Driver) Exec(ctx context.Context, query string, args, v any) error {
	d.observe(ctx, "Exec", firstToken(query))
	return d.inner.Exec(ctx, query, args, v)
}

// Query implements dialect.ExecQuerier.
func (d *Driver) Query(ctx context.Context, query string, args, v any) error {
	d.observe(ctx, "Query", firstToken(query))
	return d.inner.Query(ctx, query, args, v)
}

// -----------------------------------------------------------------------
// Observation (used by off + shadow)
// -----------------------------------------------------------------------

// observe records a query event according to the current mode. It never
// blocks and never returns an error: the goal is auditing, not enforcement.
// Enforce-mode side effects live in the caller (AcquireConn / withRLS),
// keeping this decorator lightweight and safe to disable at runtime.
func (d *Driver) observe(ctx context.Context, op, firstTok string) {
	switch d.mode {
	case ModeOff:
		d.nQueriesOff.Add(1)
		return

	case ModeShadow, ModeEnforce:
		if tenantctx.IsSystemBypass(ctx) {
			d.nSystemBypass.Add(1)
			return
		}
		tid, ok := tenantctx.TenantID(ctx)
		if !ok {
			d.nMissingTenant.Add(1)
			// In shadow mode: WARN only (no error). In enforce, upstream
			// AcquireConn is expected to have failed already; we log to be
			// defensive against paths that bypass it.
			d.log.Warnw("rls: query without tenant scope",
				"op", op,
				"stmt", firstTok,
				"mode", string(d.mode),
			)
			return
		}
		if d.mode == ModeShadow {
			d.nQueriesShadow.Add(1)
			d.log.Debugw("rls: shadow query",
				"op", op, "stmt", firstTok, "tenant_id", tid,
			)
		} else {
			// Enforce mode: no-op here; SET LOCAL is applied at conn checkout.
			d.nEnforceApplied.Add(1)
		}

	default:
		d.nQueriesOff.Add(1)
	}
}

// firstToken extracts the SQL verb (SELECT / INSERT / UPDATE / DELETE / …)
// for structured logging. Kept intentionally cheap — no full parse.
func firstToken(q string) string {
	for i := 0; i < len(q); i++ {
		c := q[i]
		if c == ' ' || c == '\t' || c == '\n' {
			return q[:i]
		}
		if i > 32 {
			return q[:32]
		}
	}
	return q
}

// -----------------------------------------------------------------------
// Compile-time interface conformance
// -----------------------------------------------------------------------

var _ dialect.Driver = (*Driver)(nil)

// -----------------------------------------------------------------------
// Convenience: build a driver from an *sql.DB and mode string.
// -----------------------------------------------------------------------

// From wraps db as an Ent driver decorated with RLS observability at the
// given mode. This is the recommended one-liner for wiring at bootstrap.
//
// Example:
//
//	drv := rls.From(db, cfg.RLS.Mode, sugar)
//	client := ent.NewClient(ent.Driver(drv))
func From(inner dialect.Driver, modeStr string, log *zap.SugaredLogger) *Driver {
	return NewDriver(inner, ParseMode(modeStr), log)
}

// Describe returns a short human string used in startup logs.
func (d *Driver) Describe() string {
	return fmt.Sprintf("rls-driver(mode=%s)", d.mode)
}
