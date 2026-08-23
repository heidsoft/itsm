package migration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigration_Struct(t *testing.T) {
	// Test that Migration struct can be created correctly
	mig := Migration{
		Version:     "002_add_notification_preferences",
		Description: "Test migration",
		RollbackSQL: "DROP TABLE user_notification_preferences",
	}

	assert.Equal(t, "002_add_notification_preferences", mig.Version)
	assert.Equal(t, "Test migration", mig.Description)
	assert.NotEmpty(t, mig.RollbackSQL)
}

func TestMigration_NoRollbackSQL(t *testing.T) {
	// Test migration without rollback SQL
	mig := Migration{
		Version:     "001_initial_schema",
		Description: "Initial schema",
		RollbackSQL: "",
	}

	assert.Equal(t, "001_initial_schema", mig.Version)
	assert.Empty(t, mig.RollbackSQL)
}

func TestMigrationSlice_Versions(t *testing.T) {
	// Test that migrations can be sorted by version
	migrations := []Migration{
		{Version: "003_add_audit_indexes"},
		{Version: "001_initial_schema"},
		{Version: "002_add_notification_preferences"},
	}

	assert.Equal(t, 3, len(migrations))
	assert.Equal(t, "003_add_audit_indexes", migrations[0].Version)
}

func TestRegisteredMigrations(t *testing.T) {
	// Test that RegisteredMigrations contains expected migrations
	assert.NotEmpty(t, RegisteredMigrations)

	assert.Equal(t, "007_add_change_execution_tables", RegisteredMigrations[0].Version)
	assert.Equal(t, "001_initial_schema", LegacyMigrations[0].Version)
}

func TestGetMigrationSQL(t *testing.T) {
	// Test GetMigrationSQL returns SQL for known migrations
	sql := GetMigrationSQL("002_add_notification_preferences")
	assert.NotEmpty(t, sql)
	assert.Contains(t, sql, "CREATE TABLE")

	// Test GetMigrationSQL returns empty for unknown migrations
	sql = GetMigrationSQL("999_unknown")
	assert.Empty(t, sql)
}

func TestGetMigrationSQL_InitialSchema(t *testing.T) {
	// Test that initial schema returns empty (handled by Ent)
	sql := GetMigrationSQL("001_initial_schema")
	assert.Empty(t, sql)
}

func TestChangeExecutionTablesAreVersioned(t *testing.T) {
	sql := GetMigrationSQL("007_add_change_execution_tables")
	assert.NotEmpty(t, sql)

	for _, table := range []string{
		"change_approval_chains",
		"change_risk_assessments",
		"change_rollback_plans",
		"change_rollback_executions",
		"change_implementation_plans",
	} {
		assert.Contains(t, sql, "CREATE TABLE IF NOT EXISTS "+table)
	}
	assert.Contains(t, sql, "tenant_id BIGINT NOT NULL")
	assert.Contains(t, sql, "ALTER COLUMN tenant_id DROP DEFAULT")
	assert.Contains(t, sql, "ALTER TABLE change_approvals ALTER COLUMN tenant_id SET NOT NULL")
	assert.Contains(t, sql, "rows without a resolvable tenant")
	for _, table := range []string{
		"change_approval_chains",
		"change_risk_assessments",
		"change_rollback_plans",
		"change_rollback_executions",
		"change_implementation_plans",
	} {
		assert.Contains(t, sql, "ALTER TABLE "+table+" ALTER COLUMN tenant_id DROP DEFAULT")
	}
}

func TestPostSchemaMigrationsStartsAtUnifiedVersion(t *testing.T) {
	migrations := PostSchemaMigrations()
	assert.NotEmpty(t, migrations)
	assert.Equal(t, "007_add_change_execution_tables", migrations[0].Version)
	for _, migration := range migrations {
		assert.GreaterOrEqual(t, migration.Version, "007_")
		assert.NotEmpty(t, GetMigrationSQL(migration.Version))
	}
}

func TestInitializationLedgerIsVersioned(t *testing.T) {
	sql := GetMigrationSQL("008_add_initialization_ledger")
	assert.NotEmpty(t, sql)
	for _, table := range []string{
		"initialization_installations",
		"initialization_runs",
		"initialization_component_attempts",
		"initialization_managed_records",
	} {
		assert.Contains(t, sql, "CREATE TABLE IF NOT EXISTS "+table)
	}
	assert.Contains(t, sql, "fencing_token")
	assert.Contains(t, sql, "lease_expires_at")
	assert.Contains(t, sql, "UNIQUE(scope_type, scope_id, component)")
}

func TestMigrationSQLChecksumIsDeterministic(t *testing.T) {
	sql := GetMigrationSQL("008_add_initialization_ledger")
	first := checksumSQL(sql)
	second := checksumSQL(sql)
	assert.NotEmpty(t, first)
	assert.Equal(t, first, second)
	assert.NotEqual(t, first, checksumSQL(sql+" -- changed"))
}

func TestProcessBindingRouteKeyMigrationsAreSeparated(t *testing.T) {
	deduplicateSQL := GetMigrationSQL("012_deduplicate_active_process_bindings")
	constraintSQL := GetMigrationSQL("013_enforce_active_process_binding_route_key")

	assert.Contains(t, deduplicateSQL, "ROW_NUMBER() OVER")
	assert.Contains(t, deduplicateSQL, "WHERE is_active = TRUE")
	assert.Contains(t, deduplicateSQL, "duplicate.route_rank > 1")
	assert.NotContains(t, deduplicateSQL, "CREATE UNIQUE INDEX")

	assert.Contains(t, constraintSQL, "CREATE UNIQUE INDEX uq_process_bindings_active_route_key")
	assert.Contains(t, constraintSQL, "COALESCE(business_sub_type, '')")
	assert.Contains(t, constraintSQL, "WHERE is_active = TRUE")
	assert.NotContains(t, constraintSQL, "DELETE FROM")
}

func TestTicketTypePermissionMigrationBackfillsAllTenantsIdempotently(t *testing.T) {
	sql := GetMigrationSQL("016_add_ticket_type_permissions")

	for _, code := range []string{"ticket_type:manage", "ticket_type:install_preset", "ticket_type:archive"} {
		assert.Contains(t, sql, code)
	}
	assert.Contains(t, sql, "FROM tenants tenant")
	assert.Contains(t, sql, "existing.tenant_id = tenant.id")
	assert.Contains(t, sql, "existing.role_id = role.id")
	assert.Contains(t, sql, "existing.permission_id = permission.id")
	assert.Contains(t, sql, "existing.tenant_id = role.tenant_id")
}
