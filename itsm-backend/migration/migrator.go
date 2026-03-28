package migration

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Migration represents a single database migration
type Migration struct {
	Version     string
	Description string
	AppliedAt   *time.Time
	RollbackSQL string
}

// Migrator handles database migrations
type Migrator struct {
	db     *sql.DB
	logger *zap.SugaredLogger
}

// NewMigrator creates a new Migrator instance
func NewMigrator(db *sql.DB, logger *zap.SugaredLogger) *Migrator {
	return &Migrator{db: db, logger: logger}
}

// EnsureMigrationsTable creates the migrations tracking table if it doesn't exist
func (m *Migrator) EnsureMigrationsTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		description TEXT NOT NULL,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		rollback_sql TEXT
	)`
	_, err := m.db.ExecContext(ctx, query)
	return err
}

// GetAppliedMigrations returns all applied migrations sorted by version
func (m *Migrator) GetAppliedMigrations(ctx context.Context) ([]Migration, error) {
	query := `SELECT version, description, applied_at, rollback_sql FROM schema_migrations ORDER BY version`
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var mig Migration
		var rollback sql.NullString
		if err := rows.Scan(&mig.Version, &mig.Description, &mig.AppliedAt, &rollback); err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		if rollback.Valid {
			mig.RollbackSQL = rollback.String
		}
		migrations = append(migrations, mig)
	}
	return migrations, rows.Err()
}

// GetPendingMigrations returns migrations that haven't been applied yet
func (m *Migrator) GetPendingMigrations(ctx context.Context, available []Migration) ([]Migration, error) {
	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	appliedVersions := make(map[string]bool)
	for _, mig := range applied {
		appliedVersions[mig.Version] = true
	}

	var pending []Migration
	for _, mig := range available {
		if !appliedVersions[mig.Version] {
			pending = append(pending, mig)
		}
	}
	return pending, nil
}

// ApplyMigration applies a single migration
func (m *Migrator) ApplyMigration(ctx context.Context, mig Migration) error {
	// Skip migrations without SQL (like initial schema handled by Ent)
	if mig.Version == "001_initial_schema" {
		m.logger.Infow("Skipping initial schema migration (handled by Ent)", "version", mig.Version)
		return nil
	}

	// Get the SQL to execute
	sql := GetMigrationSQL(mig.Version)
	if sql == "" {
		m.logger.Infow("No SQL to execute for migration", "version", mig.Version)
		return nil
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	m.logger.Infow("Applying migration", "version", mig.Version, "description", mig.Description)

	// Execute migration SQL
	if _, err := tx.ExecContext(ctx, sql); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration
	_, err = tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, description, applied_at, rollback_sql) VALUES ($1, $2, $3, $4)`,
		mig.Version, mig.Description, time.Now(), mig.RollbackSQL)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	m.logger.Infow("Migration applied successfully", "version", mig.Version)
	return nil
}

// RollbackMigration rolls back a single migration
func (m *Migrator) RollbackMigration(ctx context.Context, mig Migration) error {
	if mig.RollbackSQL == "" {
		return fmt.Errorf("no rollback SQL defined for migration %s", mig.Version)
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	m.logger.Infow("Rolling back migration", "version", mig.Version)

	// Execute rollback SQL
	if _, err := tx.ExecContext(ctx, mig.RollbackSQL); err != nil {
		return fmt.Errorf("failed to execute rollback: %w", err)
	}

	// Remove migration record
	if _, err := tx.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version = $1`, mig.Version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	m.logger.Infow("Migration rolled back successfully", "version", mig.Version)
	return nil
}

// Status returns the current migration status
func (m *Migrator) Status(ctx context.Context, available []Migration) ([]Migration, []Migration, error) {
	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, nil, err
	}

	pending, err := m.GetPendingMigrations(ctx, available)
	if err != nil {
		return nil, nil, err
	}

	return applied, pending, nil
}

// RunMigrations runs all pending migrations
func (m *Migrator) RunMigrations(ctx context.Context, available []Migration) (int, error) {
	pending, err := m.GetPendingMigrations(ctx, available)
	if err != nil {
		return 0, err
	}

	if len(pending) == 0 {
		m.logger.Info("No pending migrations")
		return 0, nil
	}

	appliedCount := 0
	for _, mig := range pending {
		if err := m.ApplyMigration(ctx, mig); err != nil {
			return appliedCount, fmt.Errorf("failed to apply migration %s: %w", mig.Version, err)
		}
		appliedCount++
	}

	return appliedCount, nil
}

// DryRun returns the SQL that would be executed without actually running it
func (m *Migrator) DryRun(ctx context.Context, mig Migration) (string, error) {
	if mig.Version == "001_initial_schema" {
		return "-- Initial schema handled by Ent", nil
	}

	sql := GetMigrationSQL(mig.Version)
	if sql == "" {
		return "-- No SQL to execute", nil
	}

	return sql, nil
}
