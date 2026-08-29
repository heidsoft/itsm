package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"
)

// prepareTicketFormFieldsMigration upgrades installations that were created
// before the tickets.form_fields column was added with NOT NULL + default.
// Ent cannot add a required column to a populated table directly, so the
// compatibility step adds the column as nullable, backfills existing rows
// with an empty JSON object, and lets Ent apply the final NOT NULL contract
// in Schema.Create.
func prepareTicketFormFieldsMigration(
	ctx context.Context,
	db *sql.DB,
	logger *zap.SugaredLogger,
) error {
	if db == nil {
		return nil
	}

	var tableExists bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = 'tickets'
		)
	`).Scan(&tableExists); err != nil {
		return fmt.Errorf("inspect tickets table: %w", err)
	}
	if !tableExists {
		return nil
	}

	var columnExists bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'tickets'
			  AND column_name = 'form_fields'
		)
	`).Scan(&columnExists); err != nil {
		return fmt.Errorf("inspect tickets.form_fields: %w", err)
	}

	if columnExists {
		// Column already present (e.g. created by a previous partial migration).
		// Make sure existing rows are backfilled with a non-NULL default so
		// Ent's later ALTER ... SET NOT NULL succeeds.
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin ticket form_fields backfill: %w", err)
		}
		defer tx.Rollback()

		if _, err := tx.ExecContext(ctx,
			`UPDATE tickets SET form_fields = '{}'::jsonb WHERE form_fields IS NULL`); err != nil {
			return fmt.Errorf("backfill tickets.form_fields: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit ticket form_fields backfill: %w", err)
		}
		logger.Infow("ticket form_fields migration: backfilled existing column")
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin ticket form_fields migration: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`ALTER TABLE tickets ADD COLUMN form_fields JSONB NOT NULL DEFAULT '{}'::jsonb`); err != nil {
		return fmt.Errorf("add tickets.form_fields: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit ticket form_fields migration: %w", err)
	}
	logger.Infow("ticket form_fields migration prepared", "column_added", true)
	return nil
}
