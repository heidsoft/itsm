package migration

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// 本文件锁住 20261118_mcp_read_permission 的空卷首装行为。
//
// 故障（2026-11-21 fresh-install-gate run 44871855241 实证）：
//
//	itsm-init 在空卷上报
//	pq: mcp:read 权限码插入失败（tenant_id=1） (P0001)
//	整条初始化链中断，backend/worker/frontend 全部起不来。
//
// 根因是执行顺序，不是 SQL 写错：
//
//	internal/bootstrap/app.go 的 InitializeStorage 先跑
//	runPostSchemaMigrations，再在 AutoSeed 分支里跑 seeder。
//	空卷上迁移执行时 tenants 表为空，Step 1 的 INSERT ... FROM tenants
//	插入 0 行，Step 2 的 DO 块据此 RAISE EXCEPTION。
//	该迁移原本是为「既有部署补码」写的，天然假设租户已存在。
//
// 修复后必须同时满足三个不变量，本文件逐条验证：
//  1. 空租户（全新安装）—— 迁移成功且不写入任何数据，seeder 负责播种；
//  2. 已有租户（存量部署）—— 仍按最小授权补齐 agent / audit_admin，且幂等；
//  3. 已有租户但 Step 1 实际没插入 —— 必须响亮失败，不允许静默留 403。
const mcpReadPermissionVersion = "20261118_mcp_read_permission"

// freshInstallDDL 复刻空卷首装时迁移可见的表形状：ent Schema.Create 已建表，
// seeder 尚未运行，因此四张表全为空。列只保留该迁移实际读写的部分。
const freshInstallDDL = `
CREATE TABLE tenants (
	id serial PRIMARY KEY,
	name text NOT NULL DEFAULT ''
);
CREATE TABLE permissions (
	id serial PRIMARY KEY,
	code text NOT NULL,
	name text NOT NULL DEFAULT '',
	description text NOT NULL DEFAULT '',
	resource text NOT NULL DEFAULT '',
	action text NOT NULL DEFAULT '',
	tenant_id integer NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE roles (
	id serial PRIMARY KEY,
	code text NOT NULL,
	tenant_id integer NOT NULL
);
CREATE TABLE role_permissions (
	id serial PRIMARY KEY,
	role_id integer NOT NULL,
	permission_id integer NOT NULL,
	tenant_id integer NOT NULL
)`

// newFreshInstallSchema 复用 user_username_tenant_unique_postgres_test.go 的
// 隔离连接器：每条连接都校验库名白名单并钉住独立 search_path，
// 因此测试不可能写进真实 schema。未提供 DSN 时整组跳过。
func newFreshInstallSchema(t *testing.T) (context.Context, *sql.DB) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("ITSM_INITIALIZATION_TEST_DSN"))
	if dsn == "" {
		t.Skip("set ITSM_INITIALIZATION_TEST_DSN to an isolated itsm_init_test_<name> database to run the fresh-install migration regressions")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)

	connector, err := pq.NewConnector(dsn)
	require.NoError(t, err, "parse explicit regression DSN")
	admin := sql.OpenDB(&userUniqueTestConnector{Connector: connector, schema: "pg_catalog"})
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	require.NoError(t, admin.PingContext(ctx), "validate regression database identity before any writes")

	schema := "freshinstall_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.ExecContext(ctx, "CREATE SCHEMA "+pq.QuoteIdentifier(schema))
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, err := admin.ExecContext(cleanupCtx, "DROP SCHEMA "+pq.QuoteIdentifier(schema)+" CASCADE")
		require.NoError(t, err, "remove only this test's isolated schema")
	})

	db := sql.OpenDB(&userUniqueTestConnector{Connector: connector, schema: schema})
	db.SetMaxOpenConns(2)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, db.PingContext(ctx))

	_, err = db.ExecContext(ctx, freshInstallDDL)
	require.NoError(t, err, "create the empty-volume table shape")
	return ctx, db
}

// applyMCPReadMigration 复刻 migrator 的事务路径：剥离脚本内嵌的 BEGIN/COMMIT，
// 在托管事务里执行磁盘上的原始字节。用真实文件而不是测试内联 SQL，
// 否则改坏迁移文本仍会绿。
func applyMCPReadMigration(ctx context.Context, db *sql.DB) error {
	raw := migrationSQLForTest(mcpReadPermissionVersion)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, stripEmbeddedTxControl(raw)); err != nil {
		return err
	}
	return tx.Commit()
}

// migrationSQLForTest 与 migrationSQL(t, version) 同义，但可在非 *testing.T 的
// 调用链里使用（错误路径测试需要拿到 error 而不是直接 FailNow）。
func migrationSQLForTest(version string) string {
	migs, err := FilesystemMigrations("../migrations")
	if err != nil {
		panic("discover migrations: " + err.Error())
	}
	for _, m := range migs {
		if m.Version == version {
			if m.SQLContent == "" {
				panic(version + " carries no SQL")
			}
			return m.SQLContent
		}
	}
	panic("migration " + version + " not discovered on disk")
}

func countRows(t *testing.T, ctx context.Context, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRowContext(ctx, query, args...).Scan(&n))
	return n
}

// 不变量 1：空卷首装时 tenants 为空，迁移必须成功且不写入任何数据。
func TestFreshInstall_MCPReadPermissionMigrationToleratesEmptyTenants(t *testing.T) {
	ctx, db := newFreshInstallSchema(t)

	require.NoError(t, applyMCPReadMigration(ctx, db),
		"空卷首装：tenants 为空时迁移必须跳过播种，而不是中断整条初始化链")

	require.Equal(t, 0, countRows(t, ctx, db, `SELECT count(*) FROM permissions`),
		"空卷上没有租户可播种，迁移不得凭空造租户或权限码")
	require.Equal(t, 0, countRows(t, ctx, db, `SELECT count(*) FROM role_permissions`))

	// 账本可能因重跑/重放再次执行同一版本，迁移必须可重复执行。
	require.NoError(t, applyMCPReadMigration(ctx, db), "迁移必须可重复执行")
}

// 不变量 2：既有部署（租户已存在）仍按最小授权补码，并且幂等。
func TestFreshInstall_MCPReadPermissionMigrationSeedsExistingTenants(t *testing.T) {
	ctx, db := newFreshInstallSchema(t)

	for _, tenantID := range []int{1, 7} {
		_, err := db.ExecContext(ctx, `INSERT INTO tenants (id, name) VALUES ($1, $2)`, tenantID, "t")
		require.NoError(t, err)
		for _, role := range []string{"agent", "audit_admin", "end_user"} {
			_, err := db.ExecContext(ctx, `INSERT INTO roles (code, tenant_id) VALUES ($1, $2)`, role, tenantID)
			require.NoError(t, err)
		}
	}

	require.NoError(t, applyMCPReadMigration(ctx, db))

	require.Equal(t, 2, countRows(t, ctx, db,
		`SELECT count(*) FROM permissions WHERE code = 'mcp:read'`),
		"每个既有租户都应补一条 mcp:read，不能只补默认租户")

	granted := func(role string) int {
		return countRows(t, ctx, db, `
			SELECT count(*)
			FROM role_permissions rp
			JOIN roles r ON r.id = rp.role_id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE r.code = $1 AND p.code = 'mcp:read' AND rp.tenant_id = r.tenant_id`, role)
	}
	require.Equal(t, 2, granted("agent"), "agent 是 MCP 典型客户端，两个租户都必须授权")
	require.Equal(t, 2, granted("audit_admin"), "service 主体载体角色必须能发现工具面")
	require.Equal(t, 0, granted("end_user"), "自助旅程身份不得进入对外协议编排面")

	require.NoError(t, applyMCPReadMigration(ctx, db), "迁移必须可重复执行")
	require.Equal(t, 2, countRows(t, ctx, db,
		`SELECT count(*) FROM permissions WHERE code = 'mcp:read'`), "重跑不得产生重复权限码")
	require.Equal(t, 4, countRows(t, ctx, db, `SELECT count(*) FROM role_permissions`),
		"重跑不得产生重复角色关联")
}

// 不变量 3：租户已存在但权限码没落库时必须响亮失败。
// 若把 Step 2 改成「查不到就跳过」，MCP 通道会在存量部署上永久 403 且无人知晓。
func TestFreshInstall_MCPReadPermissionMigrationFailsLoudlyWhenCodeCannotBeSeeded(t *testing.T) {
	ctx, db := newFreshInstallSchema(t)

	_, err := db.ExecContext(ctx, `INSERT INTO tenants (id, name) VALUES (1, 'default')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO roles (code, tenant_id) VALUES ('agent', 1)`)
	require.NoError(t, err)
	// 让 Step 1 的 INSERT ... SELECT 静默插入 0 行：这正是「码没补上」的形态。
	_, err = db.ExecContext(ctx, `
		CREATE FUNCTION block_permission_insert() RETURNS trigger AS $$
		BEGIN
			RETURN NULL;
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER block_permission_insert
			BEFORE INSERT ON permissions
			FOR EACH ROW EXECUTE FUNCTION block_permission_insert();`)
	require.NoError(t, err)

	err = applyMCPReadMigration(ctx, db)
	require.Error(t, err, "租户已存在但权限码没落库时必须失败，而不是静默跳过")
	require.Contains(t, err.Error(), "mcp:read",
		"错误信息必须指明缺失的权限码，便于运维定位")
}
