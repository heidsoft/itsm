package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

// noopTestLogger 满足 migrationLogger 接口，避免测试依赖 zap。
type noopTestLogger struct{}

func (noopTestLogger) Errorw(string, ...interface{}) {}
func (noopTestLogger) Infow(string, ...interface{})  {}

func TestRunPostSchemaMigrationsAppliesRegisteredStream(t *testing.T) {
	// 把文件系统发现的 MIGRATIONS_DIR 重定向到仓库绝对路径。
	// 仓库根目录相对于 package root 是 ../migrations，但 Go 测试 cwd 不固定。
	absMigrations, absErr := filepath.Abs("../migrations")
	if absErr != nil {
		t.Fatalf("resolve migrations dir: %v", absErr)
	}
	t.Setenv("MIGRATIONS_DIR", absMigrations)
	if _, statErr := os.Stat(absMigrations); statErr != nil {
		t.Skipf("filesystem migration dir unavailable in this run: %v", statErr)
	}
	runner := &recordingPostSchemaMigrator{}

	err := runPostSchemaMigrations(context.Background(), runner, noopTestLogger{})

	require.NoError(t, err)
	require.True(t, runner.ensured)
	// 注意：合并流后会多于 14 个；硬编码注册表刚好 14 条（018_backfill_ci_number,
	// 020_add_ai_vector_observability_storage, 021_add_workflow_template_catalog
	// + 007-017）。磁盘会追加 ~27 条 _down 配对的迁移。
	require.GreaterOrEqual(t, len(runner.migrations), 14)
	versions := map[string]bool{}
	for _, m := range runner.migrations {
		versions[m.Version] = true
	}
	for _, expected := range []string{
		"007_add_change_execution_tables",
		"009_enable_rls_tenant_isolation",
		"018_backfill_ci_number",
		"020_add_ai_vector_observability_storage",
	} {
		require.True(t, versions[expected], "missing registered migration %s", expected)
	}
	// 外部审计 P0：add_missing_indexes 必须被发现并入流
	require.True(t, versions["add_missing_indexes"], "filesystem-discovered add_missing_indexes must be merged")
}

func TestRunPostSchemaMigrationsFailsClosed(t *testing.T) {
	t.Setenv("MIGRATIONS_DIR", "")

	t.Run("ledger", func(t *testing.T) {
		runner := &recordingPostSchemaMigrator{ensureErr: errors.New("ledger unavailable")}
		err := runPostSchemaMigrations(context.Background(), runner, noopTestLogger{})
		require.ErrorContains(t, err, "ensure migration ledger")
		require.Empty(t, runner.migrations)
	})

	t.Run("migration", func(t *testing.T) {
		runner := &recordingPostSchemaMigrator{runErr: errors.New("migration failed")}
		err := runPostSchemaMigrations(context.Background(), runner, noopTestLogger{})
		require.ErrorContains(t, err, "run post-schema migrations")
	})
}
