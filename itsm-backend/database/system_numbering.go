package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"itsm-backend/common/tenantctx"
	"itsm-backend/database/rls"

	"go.uber.org/zap"
)

// SystemNumberingExecutor is the narrow, auditable escape hatch for business
// identifiers that are intentionally global (incident and CI numbers). It is
// not a generic cross-tenant query API.
type SystemNumberingExecutor struct {
	db     *sql.DB
	logger *zap.SugaredLogger
}

func NewSystemNumberingExecutor(db *sql.DB, logger *zap.SugaredLogger) *SystemNumberingExecutor {
	return &SystemNumberingExecutor{db: db, logger: logger}
}

// WithTx executes a global-number allocation in one dedicated transaction.
// The caller must explicitly construct a SystemContext with a documented
// reason. Every allocation is logged as a security audit event without
// exposing user content or credentials.
func (e *SystemNumberingExecutor) WithTx(ctx context.Context, resource string, fn func(*sql.Tx) (string, error)) (string, error) {
	if e == nil || e.db == nil {
		return "", fmt.Errorf("system numbering database is not configured")
	}
	if !tenantctx.IsSystemBypass(ctx) {
		return "", fmt.Errorf("system numbering requires an explicit system context")
	}
	if resource == "" {
		return "", fmt.Errorf("system numbering resource is required")
	}

	var tx *sql.Tx
	var conn *sql.Conn
	var err error
	driver := GetRLSDriver()
	if driver == nil || driver.Mode() == rls.ModeOff {
		tx, err = e.db.BeginTx(ctx, nil)
	} else {
		conn, err = rls.AcquireConn(ctx, e.db)
		if err == nil {
			tx, err = conn.BeginTx(ctx, nil)
		}
	}
	if err != nil {
		return "", fmt.Errorf("begin system numbering transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
		if conn != nil {
			_ = rls.ReleaseConn(ctx, conn)
		}
	}()

	number, err := fn(tx)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit system numbering transaction: %w", err)
	}
	if e.logger != nil {
		e.logger.Infow("system_numbering_allocate", "resource", resource, "number", number, "at", time.Now().UTC())
	}
	return number, nil
}
