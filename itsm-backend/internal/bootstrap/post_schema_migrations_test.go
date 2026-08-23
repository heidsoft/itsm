package bootstrap

import (
	"context"
	"errors"
	"testing"

	"itsm-backend/migration"

	"github.com/stretchr/testify/require"
)

type recordingPostSchemaMigrator struct {
	ensureErr  error
	runErr     error
	ensured    bool
	migrations []migration.Migration
}

func (m *recordingPostSchemaMigrator) EnsureMigrationsTable(context.Context) error {
	m.ensured = true
	return m.ensureErr
}

func (m *recordingPostSchemaMigrator) RunMigrations(_ context.Context, migrations []migration.Migration) (int, error) {
	m.migrations = migrations
	return len(migrations), m.runErr
}

func TestRunPostSchemaMigrationsAppliesRegisteredStream(t *testing.T) {
	runner := &recordingPostSchemaMigrator{}

	err := runPostSchemaMigrations(context.Background(), runner)

	require.NoError(t, err)
	require.True(t, runner.ensured)
	require.Len(t, runner.migrations, 10)
	require.Equal(t, "007_add_change_execution_tables", runner.migrations[0].Version)
	require.Equal(t, "008_add_initialization_ledger", runner.migrations[1].Version)
	require.Equal(t, "009_enable_rls_tenant_isolation", runner.migrations[2].Version)
	require.Equal(t, "010_add_ticket_types", runner.migrations[3].Version)
	require.Equal(t, "011_add_tool_invocation_tenant_id", runner.migrations[4].Version)
	require.Equal(t, "012_deduplicate_active_process_bindings", runner.migrations[5].Version)
	require.Equal(t, "013_enforce_active_process_binding_route_key", runner.migrations[6].Version)
	require.Equal(t, "014_add_change_approval_quorum", runner.migrations[7].Version)
	require.Equal(t, "015_ticket_type_runtime_binding", runner.migrations[8].Version)
	require.Equal(t, "016_add_ticket_type_permissions", runner.migrations[9].Version)
}

func TestRunPostSchemaMigrationsFailsClosed(t *testing.T) {
	t.Run("ledger", func(t *testing.T) {
		runner := &recordingPostSchemaMigrator{ensureErr: errors.New("ledger unavailable")}
		err := runPostSchemaMigrations(context.Background(), runner)
		require.ErrorContains(t, err, "ensure migration ledger")
		require.Empty(t, runner.migrations)
	})

	t.Run("migration", func(t *testing.T) {
		runner := &recordingPostSchemaMigrator{runErr: errors.New("migration failed")}
		err := runPostSchemaMigrations(context.Background(), runner)
		require.ErrorContains(t, err, "run post-schema migrations")
	})
}
