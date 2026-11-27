package migration

// 存量客户租户 MSP 服务商归属回填迁移的守卫与真实执行验证。
//
// 背景（组织与权限域审计 P0-3 修复的数据前置）：分配写路径新增「服务商↔客户
// 委托关系必须真实存在」的校验，判据是 tenants.msp_provider_id 必须指向目标
// MSP 员工所属的服务商租户。该字段此前只有租户管理 API 一条写入路径且前端无
// 设置入口，存量客户租户普遍为 NULL——修复落地后存量 MSP 租户会因委托关系
// 缺失被全线拒绝（fail-closed 是正确的安全默认，但需要数据兜底）。
//
// 这里锁两件事：
//  1. 磁盘发现机制能发现该迁移及其 down；
//  2. 回填判据是「只从既有活跃分配反推、无歧义才写」：无歧义的单服务商归属
//     被回填；多服务商歧义、有历史分配但已解除、无任何分配的行一律不动。

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

const (
	mspProviderUpMigration   = "20261126_backfill_msp_provider_ownership.sql"
	mspProviderDownMigration = "20261126_backfill_msp_provider_ownership_down.sql"
)

func TestMSPProviderOwnershipBackfill_MigrationShape(t *testing.T) {
	dir := filepath.Join("..", "migrations")
	upBytes, err := os.ReadFile(filepath.Join(dir, mspProviderUpMigration))
	require.NoError(t, err, "回填迁移必须存在")
	downBytes, err := os.ReadFile(filepath.Join(dir, mspProviderDownMigration))
	require.NoError(t, err, "必须提供 down，否则无法回退回填")

	upSQL := string(upBytes)
	// 上行只更新归属字段，不得删除任何行。
	require.NotContains(t, strings.ToUpper(upSQL), "DELETE FROM", "上行不得删除任何数据")
	require.Contains(t, upSQL, "msp_provider_id IS NULL", "只回填空值行，重复执行不改已回填的行")
	require.Contains(t, upSQL, "COUNT(DISTINCT u.tenant_id) = 1", "多服务商歧义必须不回填")
	require.Contains(t, upSQL, "deassigned_at IS NULL", "只用活跃分配反推归属")
	require.Contains(t, upSQL, "p.type IN ('msp_provider', 'msp')", "provider 侧必须校验租户类型")

	downSQL := string(downBytes)
	require.Contains(t, downSQL, "SET msp_provider_id = NULL")
	require.NotContains(t, strings.ToUpper(downSQL), "DELETE FROM", "down 不得删除租户行")

	found, err := FilesystemMigrations(dir)
	require.NoError(t, err)
	versions := make([]string, 0, len(found))
	for _, m := range found {
		versions = append(versions, m.Version)
	}
	require.Contains(t, versions, "20261126_backfill_msp_provider_ownership")
}

func TestMSPProviderOwnershipBackfill_ExecutesOnSqlite(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", "file:msp_provider_backfill?mode=memory&cache=shared")
	require.NoError(t, err)
	defer db.Close()

	ddl := `
	CREATE TABLE tenants (
		id INTEGER PRIMARY KEY,
		type TEXT,
		msp_provider_id INTEGER
	);
	CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		tenant_id INTEGER
	);
	CREATE TABLE msp_allocations (
		id INTEGER PRIMARY KEY,
		msp_user_id INTEGER,
		customer_tenant_id INTEGER,
		deassigned_at TIMESTAMP
	);
	`
	_, err = db.ExecContext(ctx, ddl)
	require.NoError(t, err)

	// provider 租户 1/2；客户租户 10（唯一服务商→可回填）、11（两个服务商→歧义不回填）、
	// 12（只有已解除分配→不回填）、13（无任何分配→不回填）、14（已显式设定→不得覆盖）。
	db.ExecContext(ctx, `INSERT INTO tenants (id, type, msp_provider_id) VALUES
		(1,'msp_provider',NULL),(2,'msp_provider',NULL),
		(10,'customer',NULL),(11,'customer',NULL),(12,'customer',NULL),(13,'customer',NULL),(14,'customer',2)`)
	db.ExecContext(ctx, `INSERT INTO users (id, tenant_id) VALUES (100,1),(101,1),(102,2)`)
	db.ExecContext(ctx, `INSERT INTO msp_allocations (id, msp_user_id, customer_tenant_id, deassigned_at) VALUES
		(1,100,10,NULL),(2,100,11,NULL),(3,102,11,NULL),
		(4,100,12,'2026-01-01 00:00:00'),
		(5,100,14,NULL)`)

	readSQL := func(name string) string {
		b, e := os.ReadFile(filepath.Join("..", "migrations", name))
		require.NoError(t, e)
		return string(b)
	}
	up := readSQL(mspProviderUpMigration)
	_, err = db.ExecContext(ctx, up)
	require.NoError(t, err)

	providerOf := func(id int) any {
		var v any
		require.NoError(t, db.QueryRowContext(ctx, `SELECT msp_provider_id FROM tenants WHERE id = ?`, id).Scan(&v))
		return v
	}
	require.EqualValues(t, 1, providerOf(10), "唯一服务商的活跃分配应回填归属")
	require.Nil(t, providerOf(11), "两个服务商同时有活跃分配→歧义，不回填")
	require.Nil(t, providerOf(12), "只有已解除分配→不回填")
	require.Nil(t, providerOf(13), "无任何分配→不回填")
	require.EqualValues(t, 2, providerOf(14), "已显式设定的归属不得被覆盖")

	// 重复执行必须幂等。
	_, err = db.ExecContext(ctx, up)
	require.NoError(t, err)
	require.EqualValues(t, 1, providerOf(10), "重跑不得改变已回填的行")

	// down 回退：清掉本次回填写出的归属，保留租户行本身。
	_, err = db.ExecContext(ctx, readSQL(mspProviderDownMigration))
	require.NoError(t, err)
	require.Nil(t, providerOf(10), "down 应清空回填写出的归属")

	var tenants int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tenants`).Scan(&tenants))
	require.Equal(t, 7, tenants, "down 不得删除租户行")
}
