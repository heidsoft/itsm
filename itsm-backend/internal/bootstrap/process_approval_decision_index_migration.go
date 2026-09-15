package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"
)

// prepareProcessApprovalDecisionIndexMigration drops the legacy UNIQUE index
// on process_approval_decisions(tenant_id, process_task_id) before Ent's
// auto-migration runs.
//
// The table is an append-only audit fact: one approval task may legitimately
// produce MULTIPLE decision rows (delegate → delegatee completes, add_approver
// co-signs, repeated delegation chains, withdraw/re-entry). The old unique
// index made "delegate then complete" always fail with a constraint violation
// on the second insert, silently swallowing the delegate audit row or breaking
// task completion depending on write order.
//
// Ent's schema diff cannot be trusted for unique→non-unique downgrades on
// existing tables (some drivers keep the old unique index when the new name
// differs), so we drop the legacy index explicitly here. Fresh installations
// never had the unique index and the hook is a no-op.
func prepareProcessApprovalDecisionIndexMigration(
	ctx context.Context,
	db *sql.DB,
	logger *zap.SugaredLogger,
) error {
	if db == nil {
		return nil
	}

	const legacyIndex = "processapprovaldecision_tenant_id_process_task_id"

	var tableExists bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = 'process_approval_decisions'
		)
	`).Scan(&tableExists); err != nil {
		return fmt.Errorf("inspect process_approval_decisions table: %w", err)
	}
	if !tableExists {
		return nil
	}

	var indexExists bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_indexes
			WHERE schemaname = current_schema()
			  AND tablename = 'process_approval_decisions'
			  AND indexname = $1
		)
	`, legacyIndex).Scan(&indexExists); err != nil {
		return fmt.Errorf("inspect process_approval_decisions legacy index: %w", err)
	}
	if !indexExists {
		return nil
	}

	if _, err := db.ExecContext(ctx,
		fmt.Sprintf(`DROP INDEX IF EXISTS %q`, legacyIndex)); err != nil {
		return fmt.Errorf("drop process_approval_decisions legacy unique index: %w", err)
	}

	logger.Infow("process_approval_decisions migration: legacy unique index (tenant_id, process_task_id) dropped",
		"index", legacyIndex)
	return nil
}
