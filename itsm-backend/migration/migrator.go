package migration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Migration represents a single database migration
type Migration struct {
	Version        string
	Description    string
	AppliedAt      *time.Time
	RollbackSQL    string
	Checksum       string
	ExecutionMS    int64
	ReleaseVersion string

	// SQLContent 用于「目录即真相」自发现迁移（2026-09-08 落地）：
	// 文件系统发现的 SQL 文件加载到此处，绕过 GetMigrationSQL 的硬编码注册表，
	// 避免 Go 改动滞后于 SQL 文件导致孤儿脚本。空值时回退到 GetMigrationSQL。
	// 仅作为输入与 ApplyMigration 之间传输 SQL 的载体，不参与序列化。
	SQLContent string
}

// Migrator handles database migrations
type Migrator struct {
	db             *sql.DB
	logger         *zap.SugaredLogger
	releaseVersion string
}

// NewMigrator creates a new Migrator instance
func NewMigrator(db *sql.DB, logger *zap.SugaredLogger) *Migrator {
	releaseVersion := os.Getenv("ITSM_RELEASE_VERSION")
	if releaseVersion == "" {
		releaseVersion = "unversioned"
	}
	return &Migrator{db: db, logger: logger, releaseVersion: releaseVersion}
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
	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return err
	}
	_, err := m.db.ExecContext(ctx, `
		ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum VARCHAR(128) NOT NULL DEFAULT '';
		ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS execution_ms BIGINT NOT NULL DEFAULT 0;
		ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS release_version VARCHAR(64) NOT NULL DEFAULT '';
	`)
	return err
}

// GetAppliedMigrations returns all applied migrations sorted by version
func (m *Migrator) GetAppliedMigrations(ctx context.Context) ([]Migration, error) {
	query := `SELECT version, description, applied_at, rollback_sql, checksum, execution_ms, release_version
		FROM schema_migrations ORDER BY version`
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var mig Migration
		var rollback sql.NullString
		if err := rows.Scan(
			&mig.Version, &mig.Description, &mig.AppliedAt, &rollback,
			&mig.Checksum, &mig.ExecutionMS, &mig.ReleaseVersion,
		); err != nil {
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
		expected := checksumSQL(migrationSQLFor(mig.Version, available))
		if mig.Checksum != "" && expected != "" && mig.Checksum != expected {
			return nil, fmt.Errorf(
				"migration checksum mismatch for %s: applied=%s current=%s",
				mig.Version, mig.Checksum, expected,
			)
		}
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

	// Get the SQL to execute. 优先用磁盘发现的 SQLContent，回退到 Go 硬编码注册表。
	sql := mig.SQLContent
	if sql == "" {
		sql = GetMigrationSQL(mig.Version)
	}
	if sql == "" {
		m.logger.Infow("No SQL to execute for migration", "version", mig.Version)
		return nil
	}

	// CREATE INDEX CONCURRENTLY 不能在事务块内执行（PG 25001）：
	// 此类迁移走非事务路径，逐条语句执行 + 逐条落账（账本 INSERT 独立提交）。
	// 半失败场景：已建索引可幂等重建（IF NOT EXISTS），账本未记则重跑补齐。
	if strings.Contains(strings.ToUpper(sql), "CONCURRENTLY") {
		return m.applyNonTransactional(ctx, mig, sql)
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	m.logger.Infow("Applying migration", "version", mig.Version, "description", mig.Description)

	started := time.Now()
	// Execute migration SQL
	if _, err := tx.ExecContext(ctx, sql); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration
	executionMS := time.Since(started).Milliseconds()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO schema_migrations
			(version, description, applied_at, rollback_sql, checksum, execution_ms, release_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, mig.Version, mig.Description, time.Now(), mig.RollbackSQL,
		checksumSQL(sql), executionMS, m.releaseVersion)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	m.logger.Infow("Migration applied successfully", "version", mig.Version)
	return nil
}

func checksumSQL(sql string) string {
	if sql == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(sql))
	return hex.EncodeToString(sum[:])
}

// applyNonTransactional 执行含 CONCURRENTLY 的迁移：整个文件在事务外逐语句跑，
// 全部成功后单独落账。语句按分号切分（该文件不含函数体/ dollar-quoted 字符串）。
func (m *Migrator) applyNonTransactional(ctx context.Context, mig Migration, sql string) error {
	m.logger.Infow("Applying migration (non-transactional, contains CONCURRENTLY)",
		"version", mig.Version, "description", mig.Description)

	started := time.Now()
	for _, stmt := range splitSQLStatements(sql) {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" || trimmed == ";" {
			continue
		}
		if _, err := m.db.ExecContext(ctx, trimmed); err != nil {
			return fmt.Errorf("failed to execute non-transactional statement: %w", err)
		}
	}

	executionMS := time.Since(started).Milliseconds()
	if _, err := m.db.ExecContext(ctx, `
		INSERT INTO schema_migrations
			(version, description, applied_at, rollback_sql, checksum, execution_ms, release_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, mig.Version, mig.Description, time.Now(), mig.RollbackSQL,
		checksumSQL(sql), executionMS, m.releaseVersion); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	m.logger.Infow("Migration applied successfully (non-transactional)", "version", mig.Version)
	return nil
}

// splitSQLStatements 按分号切分 SQL（不处理函数体；CONCURRENTLY 索引脚本不含）。
func splitSQLStatements(sql string) []string {
	return strings.Split(sql, ";")
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

	sql := mig.SQLContent
	if sql == "" {
		sql = GetMigrationSQL(mig.Version)
	}
	if sql == "" {
		return "-- No SQL to execute", nil
	}

	return sql, nil
}

// migrationSQLFor 在「available 中找对应版本的 SQLContent」与「GetMigrationSQL」之间仲裁。
// 与 ApplyMigration 内的 SQL 解析顺序保持一致，保证 checksum 比对与实际执行同源。
func migrationSQLFor(version string, available []Migration) string {
	for _, m := range available {
		if m.Version == version && m.SQLContent != "" {
			return m.SQLContent
		}
	}
	return GetMigrationSQL(version)
}
