package bootstrap

import (
	"context"
	"errors"
	"testing"

	"itsm-backend/migration"

	"github.com/stretchr/testify/require"
)

type recordingPostSchemaMigrator struct {
	failStep      string
	failure       error
	calls         []string
	contexts      []context.Context
	streams       [][]migration.Migration
	filesystem    []migration.Migration
	adoptionInput []migration.Migration
	adopted       int
}

func (m *recordingPostSchemaMigrator) record(step string) error {
	m.calls = append(m.calls, step)
	if step == m.failStep {
		return m.failure
	}
	return nil
}

func (m *recordingPostSchemaMigrator) EnsureMigrationsTable(ctx context.Context) error {
	m.contexts = append(m.contexts, ctx)
	return m.record("ledger")
}

func (m *recordingPostSchemaMigrator) RunMigrations(ctx context.Context, migrations []migration.Migration) (int, error) {
	m.contexts = append(m.contexts, ctx)
	step := "registered"
	if len(m.streams) > 0 {
		step = "filesystem"
	}
	m.streams = append(m.streams, migrations)
	if err := m.record(step); err != nil {
		return 0, err
	}
	return len(migrations), nil
}

func (m *recordingPostSchemaMigrator) FilesystemMigrations() ([]migration.Migration, error) {
	return m.filesystem, m.record("discovery")
}

func (m *recordingPostSchemaMigrator) RecordLegacyMigrationsApplied(ctx context.Context, _ migrationLogger) error {
	m.contexts = append(m.contexts, ctx)
	return m.record("backfill")
}

func (m *recordingPostSchemaMigrator) AdoptUnrecordedFilesystemMigrations(ctx context.Context, fs []migration.Migration, _ migrationLogger) (int, error) {
	m.contexts = append(m.contexts, ctx)
	m.adoptionInput = fs
	return m.adopted, m.record("adoption")
}

func TestRunPostSchemaMigrationsAppliesRegisteredStream(t *testing.T) {
	for _, tc := range []struct {
		name    string
		adopted int
	}{
		{name: "fresh install"},
		{name: "existing install", adopted: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// All I/O is mocked: no repository directory, environment, or database.
			fs := []migration.Migration{{Version: "add_missing_indexes", SQLContent: "SELECT 1;"}}
			runner := &recordingPostSchemaMigrator{filesystem: fs, adopted: tc.adopted}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err := runPostSchemaMigrations(ctx, runner, nil)

			require.NoError(t, err)
			require.Equal(t, []string{"ledger", "registered", "discovery", "backfill", "adoption", "filesystem"}, runner.calls)
			require.Len(t, runner.streams, 2)
			require.Equal(t, migration.PostSchemaMigrations(), runner.streams[0])
			require.Equal(t, migration.MergeWithRegistered(fs), runner.streams[1])
			require.Contains(t, runner.streams[1], fs[0], "filesystem migration must reach the second run")
			require.Equal(t, fs, runner.adoptionInput)
			require.Len(t, runner.contexts, 5)
			for _, got := range runner.contexts {
				require.Same(t, ctx, got, "bootstrap context must reach every database step")
			}
		})
	}
}

func TestRunPostSchemaMigrationsFailsClosed(t *testing.T) {
	steps := []string{"ledger", "registered", "discovery", "backfill", "adoption", "filesystem"}
	for i, tc := range []struct {
		step    string
		message string
		runs    int
	}{
		{step: "ledger", message: "ensure migration ledger", runs: 0},
		{step: "registered", message: "run post-schema migrations", runs: 1},
		{step: "discovery", message: "discover filesystem migrations", runs: 1},
		{step: "backfill", message: "backfill legacy migration ledger", runs: 1},
		{step: "adoption", message: "adopt filesystem migrations", runs: 1},
		{step: "filesystem", message: "run filesystem migrations", runs: 2},
	} {
		t.Run(tc.step, func(t *testing.T) {
			failure := errors.New("injected " + tc.step + " failure")
			runner := &recordingPostSchemaMigrator{
				failStep:   tc.step,
				failure:    failure,
				filesystem: []migration.Migration{{Version: "add_missing_indexes", SQLContent: "SELECT 1;"}},
			}

			err := runPostSchemaMigrations(context.Background(), runner, noopMigrationLogger{})

			require.ErrorIs(t, err, failure, "the caller must receive the original failure")
			require.ErrorContains(t, err, tc.message)
			require.Equal(t, steps[:i+1], runner.calls, "no later initialization step may run")
			require.Len(t, runner.streams, tc.runs)
		})
	}
}

func TestRunPostSchemaMigrationsRequiresRunner(t *testing.T) {
	err := runPostSchemaMigrations(context.Background(), nil, nil)
	require.EqualError(t, err, "migration runner is required")
}
