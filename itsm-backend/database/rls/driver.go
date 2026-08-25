// Package rls: Ent SQL Driver decorator for Row-Level Security.
//
// This decorator wraps entsql.Driver so we can transparently inject the
// PostgreSQL SET LOCAL app.current_tenant statement per query/transaction,
// without touching thousands of call sites.
//
// Three modes, controlled by config.RLSConfig.Mode:
//
//	off (default):
//	  Pass-through. No side effects. Zero risk. Used until R2B灰度期结束.
//
//	shadow:
//	  Pass-through DB behavior, but audit every query:
//	    - If ctx has tenant_id -> debug log with (op, entity, tid).
//	    - If ctx lacks tenant_id AND is not system-bypass -> WARN log with
//	      call stack summary. Used to detect missing propagation points
//	      before flipping to enforce.
//
//	enforce:
//	  Every query is wrapped in a short-lived transaction (or reuses the
//	  caller's Tx if present) with SET LOCAL app.current_tenant = <tid>.
//	  System-bypass ctx skips the SET; caller is expected to have connected
//	  via itsm_admin (BYPASSRLS).
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
	Mode           Mode   `json:"mode"`
	QueriesOff     uint64 `json:"queries_off"`
	QueriesShadow  uint64 `json:"queries_shadow"`
	MissingTenant  uint64 `json:"missing_tenant"`
	SystemBypass   uint64 `json:"system_bypass"`
	EnforceApplied uint64 `json:"enforce_applied"`
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

// Tx delegates transaction creation to the inner driver. In enforce mode,
// we execute SET LOCAL immediately after transaction creation to set the
// tenant context for all subsequent queries in this transaction.
func (d *Driver) Tx(ctx context.Context) (dialect.Tx, error) {
	d.observe(ctx, "Tx", "")
	tx, err := d.inner.Tx(ctx)
	if err != nil {
		return nil, err
	}

	// In enforce mode, set tenant context immediately after transaction start
	if d.mode == ModeEnforce && !tenantctx.IsSystemBypass(ctx) {
		if tid, ok := tenantctx.TenantID(ctx); ok {
			// Execute SET LOCAL to set tenant context for this transaction
			if err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant = %d", tid), nil, nil); err != nil {
				// Rollback on failure - don't return a broken tx
				_ = tx.Rollback()
				return nil, fmt.Errorf("rls: failed to set tenant in transaction: %w", err)
			}
			d.nEnforceApplied.Add(1)
			d.log.Debugw("rls: SET LOCAL applied in transaction", "tenant_id", tid)
		} else {
			// No tenant in context - fail closed in enforce mode
			_ = tx.Rollback()
			return nil, fmt.Errorf("rls: enforce mode requires tenant_id in context")
		}
	}

	return tx, nil
}

// BeginTx delegates to the inner driver. In enforce mode, we execute
// SET LOCAL immediately after transaction creation (same as Tx).
func (d *Driver) BeginTx(ctx context.Context, opts *sql.TxOptions) (dialect.Tx, error) {
	d.observe(ctx, "BeginTx", "")

	var tx dialect.Tx
	var err error

	if t, ok := d.inner.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	}); ok {
		tx, err = t.BeginTx(ctx, opts)
	} else {
		tx, err = d.inner.Tx(ctx)
	}
	if err != nil {
		return nil, err
	}

	// In enforce mode, set tenant context immediately after transaction start
	if d.mode == ModeEnforce && !tenantctx.IsSystemBypass(ctx) {
		if tid, ok := tenantctx.TenantID(ctx); ok {
			if err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant = %d", tid), nil, nil); err != nil {
				_ = tx.Rollback()
				return nil, fmt.Errorf("rls: failed to set tenant in transaction: %w", err)
			}
			d.nEnforceApplied.Add(1)
			d.log.Debugw("rls: SET LOCAL applied in BeginTx", "tenant_id", tid)
		} else {
			_ = tx.Rollback()
			return nil, fmt.Errorf("rls: enforce mode requires tenant_id in context")
		}
	}

	return tx, nil
}

// Exec implements dialect.ExecQuerier.
func (d *Driver) Exec(ctx context.Context, query string, args, v any) error {
	// 全路径接线：enforce 模式下，每个非事务（autocommit）语句都包裹进一个
	// 带 SET LOCAL app.current_tenant 的短事务，使 RLS 策略对读/写路径一致生效。
	// 此前 SET LOCAL 仅在 Tx/BeginTx 注入，导致绝大多数 autocommit 查询隔离失效。
	if d.mode == ModeEnforce {
		return d.withTenantTx(ctx, func(tx dialect.Tx) error {
			return tx.Exec(ctx, query, args, v)
		})
	}
	d.observe(ctx, "Exec", firstToken(query))
	return d.inner.Exec(ctx, query, args, v)
}

// Query implements dialect.ExecQuerier.
func (d *Driver) Query(ctx context.Context, query string, args, v any) error {
	if d.mode == ModeEnforce {
		return d.withTenantTx(ctx, func(tx dialect.Tx) error {
			return tx.Query(ctx, query, args, v)
		})
	}
	d.observe(ctx, "Query", firstToken(query))
	return d.inner.Query(ctx, query, args, v)
}

// withTenantTx 在 enforce 模式下为单条语句建立短事务并注入租户上下文。
//
// 设计要点：
//   - SystemBypass 上下文或缺失租户：直接内层事务执行，不注入（由 BYPASSRLS
//     角色或调用方负责跨租户语义）。缺失租户时 fail-close 拒绝，避免 NULL 租户
//     使 RLS 策略要么全漏要么全拒。
//   - SET LOCAL 仅在事务内有效，提交后即失效，绝不会泄漏到后续连接/查询。
//   - 每条 autocommit 语句多一次 BEGIN/SET LOCAL/COMMIT 往返，换取读/写隔离
//     一致性；仅在 enforce 模式启用，off/shadow 零开销。
//
// 计数器语义：
//   - EnforceApplied 仅在非 SystemBypass 且成功执行 SET LOCAL 后 +1。
//   - SystemBypass 在 SystemBypass 路径上 +1，避免 enforce 模式下 bypass 路径
//     不被监控计数，与 shadow 模式保持可观测语义一致。
func (d *Driver) withTenantTx(ctx context.Context, fn func(dialect.Tx) error) error {
	tx, err := d.inner.Tx(ctx)
	if err != nil {
		return err
	}
	// 任何失败路径都回滚，避免悬挂事务。
	defer func() { _ = tx.Rollback() }()

	if tenantctx.IsSystemBypass(ctx) {
		d.nSystemBypass.Add(1)
	} else {
		tid, ok := tenantctx.TenantID(ctx)
		if !ok {
			return fmt.Errorf("rls: enforce mode requires tenant_id in context")
		}
		if err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant = %d", tid), nil, nil); err != nil {
			return fmt.Errorf("rls: failed to set tenant in statement: %w", err)
		}
		d.nEnforceApplied.Add(1)
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
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
			d.log.Warnw(
				"rls: query without tenant scope",
				"op", op,
				"stmt", firstTok,
				"mode", string(d.mode),
			)
			return
		}
		if d.mode == ModeShadow {
			d.nQueriesShadow.Add(1)
			d.log.Debugw(
				"rls: shadow query",
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
