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
	"strconv"
	"sync/atomic"

	"itsm-backend/common/tenantctx"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
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
			if err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL %s = %d", tenantVarName, tid), nil, nil); err != nil {
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
			if err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL %s = %d", tenantVarName, tid), nil, nil); err != nil {
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
	if d.mode == ModeEnforce {
		ec, err := d.enforceContext(ctx, "Exec", firstToken(query))
		if err != nil {
			return err
		}
		return d.inner.Exec(ec, query, args, v)
	}
	d.observe(ctx, "Exec", firstToken(query))
	return d.inner.Exec(ctx, query, args, v)
}

// Query implements dialect.ExecQuerier.
func (d *Driver) Query(ctx context.Context, query string, args, v any) error {
	if d.mode == ModeEnforce {
		ec, err := d.enforceContext(ctx, "Query", firstToken(query))
		if err != nil {
			return err
		}
		return d.inner.Query(ec, query, args, v)
	}
	d.observe(ctx, "Query", firstToken(query))
	return d.inner.Query(ctx, query, args, v)
}

// enforceContext 为 enforce 模式下的单条语句准备上下文。
//
// 2026-08-29 回归修复（生产就绪评估 P0-3）：旧实现把每条 autocommit 语句包进
// 短事务（BEGIN + SET LOCAL + 语句 + COMMIT）。但 ent 的 Query 把 *sql.Rows 句柄
// 原样交回调用方、由调用方稍后迭代；短事务在调用方扫描前就已提交，行句柄随之
// 失效，启动就绪探针等读路径报 "sql: Rows are closed"。
//
// 现改用 ent 原生会话变量机制（entsql.WithVar）：执行语句前在专用池连接上
// `SET app.current_tenant = '<tid>'`，结果集关闭时自动 `RESET` 并归还连接。
// 行句柄全程有效，租户状态不会泄漏回连接池；显式事务路径仍由 Tx/BeginTx 中
// 的 SET LOCAL 覆盖。
//
// 计数器语义：
//   - SystemBypass：计数后原样透传（由 BYPASSRLS 角色负责跨租户语义）。
//   - 缺失租户：fail-close 拒绝，避免 NULL 租户使策略要么全漏要么全拒。
//   - 其余：注入变量并计 EnforceApplied。
func (d *Driver) enforceContext(ctx context.Context, op, firstTok string) (context.Context, error) {
	if tenantctx.IsSystemBypass(ctx) {
		d.nSystemBypass.Add(1)
		return ctx, nil
	}
	tid, ok := tenantctx.TenantID(ctx)
	if !ok {
		d.nMissingTenant.Add(1)
		d.log.Warnw("rls: statement without tenant scope", "op", op, "stmt", firstTok, "mode", string(d.mode))
		return nil, fmt.Errorf("rls: enforce mode requires tenant_id in context")
	}
	d.nEnforceApplied.Add(1)
	return entsql.WithVar(ctx, tenantVarName, strconv.Itoa(tid)), nil
}

// tenantVarName 是 RLS 策略读取的会话变量名，与 002_pilot_policies.sql 保持一致。
const tenantVarName = "app.current_tenant"

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
