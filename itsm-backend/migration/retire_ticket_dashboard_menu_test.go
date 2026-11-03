package migration

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/menu"
)

// 工单仪表盘菜单退役回归。
//
// /tickets/dashboard 与 /tickets/analytics 功能重复，页面已删除、基线已摘除条目。
// 但基线补齐（service/menu_service.go InitMenusFromBaseline）与 seedMenus 都是 add-only，
// 不会删改存量行，于是存量库里这行菜单仍然指向一个 404 入口。
// 本文件锁三件事：迁移被「目录即真相」发现器拾取、SQL 不踩硬编码租户/主键，
// 以及 UPDATE 在真实 menus 表上确实跨租户停用且不删行。
const retireTicketDashboardMenuVersion = "20261101_retire_ticket_dashboard_menu"

func mustRetireDashboardMigration(t *testing.T) Migration {
	t.Helper()
	migs, err := FilesystemMigrations("../migrations")
	require.NoError(t, err)
	for _, m := range migs {
		if m.Version == retireTicketDashboardMenuVersion {
			return m
		}
	}
	t.Fatalf("退役迁移 %s 必须被目录发现器拾取，否则存量库无人收口", retireTicketDashboardMenuVersion)
	return Migration{}
}

func TestRetireTicketDashboardMenu_DiscoveredAndTenantAgnostic(t *testing.T) {
	found := mustRetireDashboardMigration(t)
	require.NotEmpty(t, found.SQLContent, "up script must carry SQL")
	require.NotEmpty(t, found.RollbackSQL, "down script must be paired via the _down.sql suffix")

	// 20260830_ticket_types_menu_reparenting.sql 因硬编码 tenant_id 与自增主键被退役成
	// SELECT 1；同类错误不得再进第二遍。只看可执行语句，注释里的字段名不算。
	stmts := executableStatements(found.SQLContent)
	require.NotContains(t, stmts, "tenant_id",
		"迁移不得按租户过滤：它必须一次覆盖全部租户")
	require.NotContains(t, stmts, "id =")
	require.NotContains(t, stmts, "parent_id")
	require.Contains(t, stmts, "'/tickets/dashboard'")
	require.Equal(t, 1, strings.Count(stmts, "UPDATE menus"), "只允许一条 UPDATE，避免顺带改到别的菜单")
}

func TestRetireTicketDashboardMenu_DisablesAcrossTenantsWithoutDeleting(t *testing.T) {
	const dsn = "file:retire_dashboard_menu?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()
	ctx := context.Background()

	tenantA := client.Tenant.Create().SetName("a").SetCode("retire-a").SetDomain("a.test").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("b").SetCode("retire-b").SetDomain("b.test").SaveX(ctx)

	dashboardA := client.Menu.Create().SetName("工单仪表盘").SetPath("/tickets/dashboard").SetTenantID(tenantA.ID).SaveX(ctx)
	analyticsA := client.Menu.Create().SetName("工单统计").SetPath("/tickets/analytics").SetTenantID(tenantA.ID).SaveX(ctx)
	client.Menu.Create().SetName("工单仪表盘").SetPath("/tickets/dashboard").SetTenantID(tenantB.ID).SaveX(ctx)
	// 运维此前已自行停用的同路径行：重复执行必须无错，且行数不变。
	client.Menu.Create().SetName("工单仪表盘").SetPath("/tickets/dashboard").
		SetTenantID(tenantB.ID).SetIsEnabled(false).SaveX(ctx)
	// 清单外的客户自建菜单必须完全不受影响。
	custom := client.Menu.Create().SetName("自定义看板").SetPath("/tickets/my-board").SetTenantID(tenantB.ID).SaveX(ctx)

	found := mustRetireDashboardMigration(t)
	db := openSharedSQLite(t, dsn)
	defer db.Close()
	_, err := db.ExecContext(ctx, found.SQLContent)
	require.NoError(t, err)

	require.False(t, client.Menu.GetX(ctx, dashboardA.ID).IsEnabled,
		"租户 A 的仪表盘菜单必须停用，否则侧边栏仍给使用者留一个 404 入口")
	enabled, err := client.Menu.Query().
		Where(menu.PathEQ("/tickets/dashboard"), menu.IsEnabledEQ(true)).
		Count(ctx)
	require.NoError(t, err)
	require.Zero(t, enabled, "租户 B 同样要收口：退役不能只覆盖基线租户")

	require.True(t, client.Menu.GetX(ctx, analyticsA.ID).IsEnabled, "同工单统计入口不能被顺带停用")
	require.True(t, client.Menu.GetX(ctx, custom.ID).IsEnabled, "清单外的客户自建菜单不得被动")

	total, err := client.Menu.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 5, total, "退役是停用，不是删行")

	// 幂等：重复执行不得改变形态，也不得报错。
	_, err = db.ExecContext(ctx, found.SQLContent)
	require.NoError(t, err)
	total, err = client.Menu.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 5, total)
	dashboards, err := client.Menu.Query().Where(menu.PathEQ("/tickets/dashboard")).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, dashboards)
}

// TestRetireTicketDashboardMenu_RollbackReEnables 锁回滚方向。
// 已知边界：无法区分「本迁移停用」与「运维此前停用」，同名清单行会被全部激活。
func TestRetireTicketDashboardMenu_RollbackReEnables(t *testing.T) {
	const dsn = "file:retire_dashboard_menu_down?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()
	ctx := context.Background()

	tenantA := client.Tenant.Create().SetName("a").SetCode("retire-dn-a").SetDomain("dna.test").SaveX(ctx)
	dashboard := client.Menu.Create().SetName("工单仪表盘").SetPath("/tickets/dashboard").
		SetTenantID(tenantA.ID).SetIsEnabled(false).SaveX(ctx)

	found := mustRetireDashboardMigration(t)
	db := openSharedSQLite(t, dsn)
	defer db.Close()
	_, err := db.ExecContext(ctx, found.RollbackSQL)
	require.NoError(t, err)
	require.True(t, client.Menu.GetX(ctx, dashboard.ID).IsEnabled)
}

// openSharedSQLite 打开与 enttest 相同的 shared-cache 内存库，用于执行迁移原文 SQL：
// 生成的 ent 客户端未开启 sql/execquery 特性，没有 ExecContext 可用。
func openSharedSQLite(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	return db
}

// executableStatements 去掉 `--` 注释行，只留真正会执行的语句。
// 退役说明里必须写清「不按 tenant_id/主键匹配」，因此不能直接对整份文件断言。
func executableStatements(script string) string {
	lines := strings.Split(script, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
