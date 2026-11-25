package migration

// B5 真实执行验证：把迁移 SQL 字节读出来，在内存 sqlite 上跑一遍，证明
// 「存量 sysadmin 补上 mcp:read、幂等、不跨租户、不动其它角色、down 精确回退」。
// 静态文本断言不够——SQL 里的 join / NOT EXISTS 写错时文本仍看起来正确。

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSysadminMCPReadReconcile_ExecutesOnSqlite(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", "file:sysadmin_mcp_reconcile?mode=memory&cache=shared")
	require.NoError(t, err)
	defer db.Close()

	ddl := `
	CREATE TABLE permissions (id INTEGER PRIMARY KEY, code TEXT, tenant_id INTEGER);
	CREATE TABLE roles (id INTEGER PRIMARY KEY, code TEXT, tenant_id INTEGER);
	CREATE TABLE role_permissions (id INTEGER PRIMARY KEY, role_id INTEGER, permission_id INTEGER, tenant_id INTEGER);
	`
	_, err = db.ExecContext(ctx, ddl)
	require.NoError(t, err)

	// 两个租户各有 sysadmin 与 mcp:read 码；tenant 2 的 sysadmin 已有该关联。
	db.ExecContext(ctx, `INSERT INTO permissions (id, code, tenant_id) VALUES (1,'mcp:read',1),(2,'mcp:read',2),(3,'ticket:read',1)`)
	db.ExecContext(ctx, `INSERT INTO roles (id, code, tenant_id) VALUES (1,'sysadmin',1),(2,'sysadmin',2),(3,'agent',1)`)
	db.ExecContext(ctx, `INSERT INTO role_permissions (id, role_id, permission_id, tenant_id) VALUES (1,2,2,2),(2,3,1,1)`)

	readSQL := func(name string) string {
		b, e := os.ReadFile(filepath.Join("..", "migrations", name))
		require.NoError(t, e)
		return string(b)
	}
	up := readSQL("20261125_reconcile_sysadmin_mcp_read.sql")
	// sqlite 不支持 BEGIN/COMMIT 嵌套在 Exec 多语句里？go-sqlite3 支持，直接执行。
	_, err = db.ExecContext(ctx, up)
	require.NoError(t, err)

	count := func(q string) int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, q).Scan(&n))
		return n
	}
	require.Equal(t, 3, count(`SELECT COUNT(*) FROM role_permissions`), "tenant 1 补 1 条；tenant 2 已存在不重复；其它关联不动")
	require.Equal(t, 1, count(`SELECT COUNT(*) FROM role_permissions WHERE role_id=1 AND permission_id=1`), "tenant 1 sysadmin 补上 mcp:read")
	require.Equal(t, 1, count(`SELECT COUNT(*) FROM role_permissions WHERE role_id=2 AND permission_id=2`), "tenant 2 既有保持")

	// 重复执行必须幂等。
	_, err = db.ExecContext(ctx, up)
	require.NoError(t, err)
	require.Equal(t, 3, count(`SELECT COUNT(*) FROM role_permissions`), "重跑不得新增关联")

	// down 精确回退：只删 sysadmin 的 mcp:read，保留 agent 的关联与 permissions 行。
	_, err = db.ExecContext(ctx, readSQL("20261125_reconcile_sysadmin_mcp_read_down.sql"))
	require.NoError(t, err)
	require.Equal(t, 1, count(`SELECT COUNT(*) FROM role_permissions`), "down 后只剩 agent 的关联")
	require.Equal(t, 3, count(`SELECT COUNT(*) FROM permissions`), "down 不得删权限码")
}
