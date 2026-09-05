package database

import (
	"context"
	"database/sql"
	"fmt"

	"itsm-backend/common/tenantctx"
	"itsm-backend/database/rls"
)

// SQLExecutor is the deliberately small raw-SQL surface available to domain
// repositories. New code must obtain it through WithTenantSQL rather than
// retaining a process-global *sql.DB.
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// WithTenantSQL runs fn with a connection scoped to tenantID when RLS is
// enabled. In the default/off mode it keeps the existing database/sql
// behavior, but still rejects an explicit context/argument tenant mismatch.
// Rows returned by fn must be fully consumed and closed before fn returns.
func WithTenantSQL[T any](ctx context.Context, db *sql.DB, tenantID int, fn func(SQLExecutor) (T, error)) (T, error) {
	var zero T
	if db == nil {
		return zero, fmt.Errorf("tenant SQL database is not configured")
	}
	if tenantID <= 0 {
		return zero, fmt.Errorf("tenant SQL requires a positive tenant ID")
	}
	if contextTenant, ok := tenantctx.TenantID(ctx); ok && contextTenant != tenantID {
		return zero, fmt.Errorf("tenant SQL context tenant %d does not match query tenant %d", contextTenant, tenantID)
	}

	driver := GetRLSDriver()
	if driver == nil || driver.Mode() == rls.ModeOff {
		return fn(db)
	}
	conn, err := rls.AcquireConn(ctx, db)
	if err != nil {
		return zero, err
	}
	defer rls.ReleaseConn(ctx, conn)
	return fn(conn)
}

// WithTenantTx is the transaction counterpart to WithTenantSQL. The callback
// must only perform database work; this helper commits on success and rolls
// back on every error, so callers cannot accidentally commit a transaction
// whose RLS session belongs to another pooled connection.
func WithTenantTx[T any](ctx context.Context, db *sql.DB, tenantID int, fn func(*sql.Tx) (T, error)) (T, error) {
	var zero T
	if db == nil {
		return zero, fmt.Errorf("tenant SQL database is not configured")
	}
	if tenantID <= 0 {
		return zero, fmt.Errorf("tenant SQL requires a positive tenant ID")
	}
	if contextTenant, ok := tenantctx.TenantID(ctx); ok && contextTenant != tenantID {
		return zero, fmt.Errorf("tenant SQL context tenant %d does not match query tenant %d", contextTenant, tenantID)
	}

	var tx *sql.Tx
	var conn *sql.Conn
	var err error
	driver := GetRLSDriver()
	if driver == nil || driver.Mode() == rls.ModeOff {
		tx, err = db.BeginTx(ctx, nil)
	} else {
		conn, err = rls.AcquireConn(ctx, db)
		if err == nil {
			tx, err = conn.BeginTx(ctx, nil)
		}
	}
	if err != nil {
		if conn != nil {
			_ = rls.ReleaseConn(ctx, conn)
		}
		return zero, err
	}
	defer func() {
		_ = tx.Rollback()
		if conn != nil {
			_ = rls.ReleaseConn(ctx, conn)
		}
	}()

	value, err := fn(tx)
	if err != nil {
		return zero, err
	}
	if err := tx.Commit(); err != nil {
		return zero, err
	}
	return value, nil
}
