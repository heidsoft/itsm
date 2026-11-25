package migration

// B5：存量 sysadmin 缺 mcp:read 关联的补偿迁移。
// 这里锁两件事（不依赖数据库）：
//  1. 磁盘发现机制能发现新迁移及其 down；
//  2. 上行是「只增不减」（NOT EXISTS + 同租户 join），down 只精确删本次码的
//     sysadmin 关联，不动其它角色/码/权限行——避免全量补齐被回滚成全量撤权。
//
// 用户拍板：全量自动补齐；UPGRADE.md 明示「会补回管理员此前主动撤权」的取舍。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSysadminMCPReadReconcileMigration_DiscoveredWithRollback(t *testing.T) {
	dir := filepath.Join("..", "migrations")
	up := filepath.Join(dir, "20261125_reconcile_sysadmin_mcp_read.sql")
	down := filepath.Join(dir, "20261125_reconcile_sysadmin_mcp_read_down.sql")

	upBytes, err := os.ReadFile(up)
	require.NoError(t, err, "补偿迁移必须存在")
	downBytes, err := os.ReadFile(down)
	require.NoError(t, err, "必须提供 down，否则无法回退全量补齐")

	upSQL := string(upBytes)
	require.Contains(t, upSQL, "INSERT INTO role_permissions", "上行只新增关联")
	require.Contains(t, upSQL, "r.code = 'sysadmin'")
	require.Contains(t, upSQL, "p.code = 'mcp:read'")
	require.Contains(t, upSQL, "p.tenant_id = r.tenant_id", "必须同租户 join，防止跨租户串码")
	require.Contains(t, upSQL, "NOT EXISTS", "必须幂等，防止重复关联")
	require.NotContains(t, strings.ToUpper(upSQL), "DELETE FROM", "上行不得删除任何既有授权")

	downSQL := string(downBytes)
	require.Contains(t, downSQL, "DELETE FROM role_permissions")
	require.NotContains(t, strings.ToUpper(downSQL), "DELETE FROM permissions", "down 不得删权限码")
	require.Contains(t, downSQL, "sysadmin")
	require.Contains(t, downSQL, "mcp:read")

	// 发现机制：磁盘目录下必须能枚举到该版本。
	found, err := FilesystemMigrations(dir)
	require.NoError(t, err)
	versions := make([]string, 0, len(found))
	for _, m := range found {
		versions = append(versions, m.Version)
	}
	require.Contains(t, versions, "20261125_reconcile_sysadmin_mcp_read")
}
