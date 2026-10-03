package seeder

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/common/tenantctx"
	"itsm-backend/config"
	"itsm-backend/ent"
	"itsm-backend/ent/role"
	_ "itsm-backend/ent/runtime"
)

func validateRolesSequenceTestDatabase(name string) error {
	const prefix = "itsm_init_test_"
	if !strings.HasPrefix(name, prefix) || len(name) == len(prefix) {
		return errors.New("roles sequence regression requires a database named itsm_init_test_<name>")
	}
	return nil
}

// rolesSequenceTestConnector mirrors initializationTestConnector: every physical
// connection is validated against the isolated test database and pinned to this
// test's schema before any statement runs. Only the explicit test DSN is used;
// application config and .env are never read.
type rolesSequenceTestConnector struct {
	driver.Connector
	schema string
}

func (c *rolesSequenceTestConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	ready := false
	defer func() {
		if !ready {
			_ = conn.Close()
		}
	}()
	querier, ok := conn.(driver.QueryerContext)
	if !ok {
		return nil, errors.New("PostgreSQL driver does not support database identity validation")
	}
	rows, err := querier.QueryContext(ctx, "SELECT current_database()::text", nil)
	if err != nil {
		return nil, err
	}
	values := make([]driver.Value, 1)
	err = rows.Next(values)
	closeErr := rows.Close()
	if err != nil || closeErr != nil {
		return nil, errors.Join(err, closeErr)
	}
	name, ok := values[0].(string)
	if !ok {
		return nil, errors.New("unexpected PostgreSQL database identity type")
	}
	if err := validateRolesSequenceTestDatabase(name); err != nil {
		return nil, err
	}
	execer, ok := conn.(driver.ExecerContext)
	if !ok {
		return nil, errors.New("PostgreSQL driver does not support isolated test sessions")
	}
	_, err = execer.ExecContext(ctx, `SELECT set_config('search_path', $1, false),
		set_config('statement_timeout', '120000', false), set_config('lock_timeout', '3000', false)`,
		[]driver.NamedValue{{Ordinal: 1, Value: c.schema}})
	if err != nil {
		return nil, err
	}
	ready = true
	return conn, nil
}

// newRolesSequencePostgresFixture opens the explicit regression DSN and builds a
// full ent schema inside a throwaway schema of the isolated database.
func newRolesSequencePostgresFixture(t *testing.T) (*sql.DB, *ent.Client) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("ITSM_INITIALIZATION_TEST_DSN"))
	if dsn == "" {
		t.Skip("set ITSM_INITIALIZATION_TEST_DSN to an isolated itsm_init_test_<name> database to run the roles sequence regression")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	t.Cleanup(cancel)
	connector, err := pq.NewConnector(dsn)
	require.NoError(t, err, "parse roles sequence regression DSN")
	admin := sql.OpenDB(&rolesSequenceTestConnector{Connector: connector, schema: "pg_catalog"})
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	require.NoError(t, admin.PingContext(ctx), "validate roles sequence regression database before any writes")
	schema := "rolesseq_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.ExecContext(ctx, "CREATE SCHEMA "+pq.QuoteIdentifier(schema))
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, err := admin.ExecContext(cleanupCtx, "DROP SCHEMA "+pq.QuoteIdentifier(schema)+" CASCADE")
		require.NoError(t, err, "remove only this test's isolated schema")
	})

	db := sql.OpenDB(&rolesSequenceTestConnector{Connector: connector, schema: schema})
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, db.PingContext(ctx))

	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	require.NoError(t, client.Schema.Create(ctx))
	return db, client
}

// TestSeedRolesReconcilesRolesIDSequenceAfterExplicitIDMigration 复刻生产缺陷：
// migrations/20260501_enable_rbac_from_db.sql 以显式 id 100-112 插入平台内置角色
// 但不推进 roles_id_seq；租户引导的串行分配一旦追上 100 即撞 roles_pkey，且
// PostgreSQL 序列不随事务回滚，重试永远撞在同一区间上，租户引导死信。回归证明
// seedRoles 必须先把序列对齐到 max(roles.id) 再插入。
func TestSeedRolesReconcilesRolesIDSequenceAfterExplicitIDMigration(t *testing.T) {
	db, client := newRolesSequencePostgresFixture(t)
	ctx := tenantctx.SystemContext(context.Background(), "initialization:test", "roles sequence regression")

	platform, err := client.Tenant.Create().
		SetName("Platform Tenant").
		SetCode("platform-seq").
		Save(ctx)
	require.NoError(t, err)
	target, err := client.Tenant.Create().
		SetName("Bootstrapped Tenant").
		SetCode("bootstrap-seq").
		Save(ctx)
	require.NoError(t, err)

	// 迁移的显式 id 行：13 个平台内置角色，id 100-112（与线上 tenant 1 对齐）。
	const legacyRowCount = 13
	placeholders := make([]string, 0, legacyRowCount)
	args := make([]any, 0, legacyRowCount*4)
	for i := 0; i < legacyRowCount; i++ {
		id := 100 + i
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d, true, NOW(), NOW())", len(args)+1, len(args)+2, len(args)+3, len(args)+4))
		args = append(args, id, fmt.Sprintf("legacy_migration_role_%d", id), fmt.Sprintf("迁移内置角色 %d", id), platform.ID)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO roles (id, code, name, tenant_id, is_system, created_at, updated_at) VALUES `+strings.Join(placeholders, ", "),
		args...)
	require.NoError(t, err)

	// 复刻消耗后的序列状态：next 从 96 开始，第 5 次分配即追上显式 id 100。
	_, err = db.ExecContext(ctx, "SELECT setval(pg_get_serial_sequence('roles', 'id'), 95, true)")
	require.NoError(t, err)

	seeder := NewSeeder(client, zap.NewNop().Sugar(), &config.Config{})
	seeder.sqlDriver = entsql.OpenDB(dialect.Postgres, db)
	seeder.withBaselineTenant(target.ID).seedRoles(ctx)

	for _, builtin := range BuiltinRoles() {
		count, err := client.Role.Query().
			Where(role.CodeEQ(builtin.Code), role.TenantIDEQ(target.ID)).
			Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, count, "builtin role %q must exist in the bootstrapped tenant", builtin.Code)
	}

	// 迁移写入的平台内置角色不得被改动或删除。
	for i := 0; i < legacyRowCount; i++ {
		id := 100 + i
		count, err := client.Role.Query().
			Where(role.IDEQ(id), role.TenantIDEQ(platform.ID)).
			Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, count, "legacy migration role id=%d must remain untouched", id)
	}
}
