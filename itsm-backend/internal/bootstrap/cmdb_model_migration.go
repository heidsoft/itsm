package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"
)

// prepareCMDBModelMigration validates data before Ent adds tenant-scoped
// uniqueness constraints and removes the superseded global serial-number index.
func prepareCMDBModelMigration(ctx context.Context, db *sql.DB, logger *zap.SugaredLogger) error {
	if db == nil {
		return nil
	}

	checks := []struct {
		name  string
		query string
	}{
		{
			name: "duplicate CI type names within a tenant",
			query: `SELECT 1 FROM ci_types
				GROUP BY tenant_id, name
				HAVING COUNT(*) > 1
				LIMIT 1`,
		},
		{
			name: "duplicate non-empty CI serial numbers within a tenant",
			query: `SELECT 1 FROM configuration_items
				WHERE serial_number IS NOT NULL AND serial_number <> ''
				GROUP BY tenant_id, serial_number
				HAVING COUNT(*) > 1
				LIMIT 1`,
		},
		{
			name: "duplicate CI attribute names within a tenant and type",
			query: `SELECT 1 FROM ci_attribute_definitions
				GROUP BY tenant_id, ci_type_id, name
				HAVING COUNT(*) > 1
				LIMIT 1`,
		},
		{
			name: "orphan CI attribute definitions",
			query: `SELECT 1 FROM ci_attribute_definitions a
				LEFT JOIN ci_types t
					ON t.id = a.ci_type_id AND t.tenant_id = a.tenant_id
				WHERE t.id IS NULL
				LIMIT 1`,
		},
	}
	for _, check := range checks {
		var found int
		err := db.QueryRowContext(ctx, check.query).Scan(&found)
		if err == nil {
			return fmt.Errorf("%s must be resolved before CMDB model migration", check.name)
		}
		if err != sql.ErrNoRows {
			// Fresh databases do not have these tables yet; Ent creates them below.
			logger.Debugw("CMDB model preflight skipped", "check", check.name, "error", err)
		}
	}

	if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS configurationitem_serial_number`); err != nil {
		return fmt.Errorf("drop superseded global CI serial-number index: %w", err)
	}
	return nil
}
