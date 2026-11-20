package migration

import (
	"context"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/menu"
)

// 「AI 中心」一级菜单退役回归（2026-11-21 方案 A）。
//
// 一级导航按技术而非业务任务组织，且与顶栏全局「AI 助手」指向同一 /ai/chat 页面。
// 页面与路由零改动，存量库里那行一级菜单仍会给出重复入口，必须靠迁移停用。
// 本文件锁三件事：迁移被「目录即真相」发现器拾取、SQL 不踩硬编码租户/主键，
// 以及 UPDATE 在真实 menus 表上确实跨租户停用且不删行。
const retireAICenterMenuVersion = "20261121_retire_ai_center_menu"

func mustRetireAICenterMigration(t *testing.T) Migration {
	t.Helper()
	migs, err := FilesystemMigrations("../migrations")
	require.NoError(t, err)
	for _, m := range migs {
		if m.Version == retireAICenterMenuVersion {
			return m
		}
	}
	t.Fatalf("退役迁移 %s 必须被目录发现器拾取，否则存量库仍挂着已下线的一级入口", retireAICenterMenuVersion)
	return Migration{}
}

func TestRetireAICenterMenu_DiscoveredAndTenantAgnostic(t *testing.T) {
	found := mustRetireAICenterMigration(t)
	require.NotEmpty(t, found.SQLContent, "up script must carry SQL")
	require.NotEmpty(t, found.RollbackSQL, "down script must be paired via the _down.sql suffix")

	// 只看可执行语句，注释里的字段名不算。
	stmts := executableStatements(found.SQLContent)
	require.NotContains(t, stmts, "tenant_id", "迁移不得按租户过滤：它必须一次覆盖全部租户")
	require.NotContains(t, stmts, "id =")
	require.NotContains(t, stmts, "parent_id", "子菜单重挂由基线 upsert 完成，迁移不写父级字段")
	require.Contains(t, stmts, "'/ai/chat'")
	require.Equal(t, 1, strings.Count(stmts, "UPDATE menus"), "只允许一条 UPDATE，避免顺带改到别的菜单")
}

func TestRetireAICenterMenu_DisablesAcrossTenantsWithoutDeleting(t *testing.T) {
	const dsn = "file:retire_ai_center_menu?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()
	ctx := context.Background()

	tenantA := client.Tenant.Create().SetName("a").SetCode("retire-ai-a").SetDomain("a.test").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("b").SetCode("retire-ai-b").SetDomain("b.test").SaveX(ctx)

	aiCenterA := client.Menu.Create().SetName("AI 中心").SetPath("/ai/chat").SetTenantID(tenantA.ID).SaveX(ctx)
	client.Menu.Create().SetName("AI 中心").SetPath("/ai/chat").SetTenantID(tenantB.ID).SaveX(ctx)
	// 归位后的业务能力子菜单：迁移只针对一级行，不得顺带停用。
	chatChild := client.Menu.Create().SetName("AI 创建工单").SetPath("/tickets/ai-create").SetTenantID(tenantA.ID).SaveX(ctx)
	// 清单外的客户自建菜单必须完全不受影响。
	custom := client.Menu.Create().SetName("自定义 AI 页").SetPath("/ai/custom").SetTenantID(tenantB.ID).SaveX(ctx)

	found := mustRetireAICenterMigration(t)
	db := openSharedSQLite(t, dsn)
	defer db.Close()
	_, err := db.ExecContext(ctx, found.SQLContent)
	require.NoError(t, err)

	require.False(t, client.Menu.GetX(ctx, aiCenterA.ID).IsEnabled,
		"租户 A 的「AI 中心」必须停用，否则侧边栏仍留着与顶栏重复的入口")
	enabled, err := client.Menu.Query().
		Where(menu.PathEQ("/ai/chat"), menu.IsEnabledEQ(true)).
		Count(ctx)
	require.NoError(t, err)
	require.Zero(t, enabled, "租户 B 同样要收口：退役不能只覆盖基线租户")

	require.True(t, client.Menu.GetX(ctx, chatChild.ID).IsEnabled, "归位后的 AI 能力子菜单不能被顺带停用")
	require.True(t, client.Menu.GetX(ctx, custom.ID).IsEnabled, "清单外的客户自建菜单不得被动")

	total, err := client.Menu.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, total, "退役是停用，不是删行")

	// 幂等：重复执行不得改变形态，也不得报错。
	_, err = db.ExecContext(ctx, found.SQLContent)
	require.NoError(t, err)
	total, err = client.Menu.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, total)
}

func TestRetireAICenterMenu_RollbackReEnables(t *testing.T) {
	const dsn = "file:retire_ai_center_menu_down?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()
	ctx := context.Background()

	tenantA := client.Tenant.Create().SetName("a").SetCode("retire-ai-dn-a").SetDomain("dna.test").SaveX(ctx)
	aiCenter := client.Menu.Create().SetName("AI 中心").SetPath("/ai/chat").
		SetTenantID(tenantA.ID).SetIsEnabled(false).SaveX(ctx)

	found := mustRetireAICenterMigration(t)
	db := openSharedSQLite(t, dsn)
	defer db.Close()
	_, err := db.ExecContext(ctx, found.RollbackSQL)
	require.NoError(t, err)
	require.True(t, client.Menu.GetX(ctx, aiCenter.ID).IsEnabled)
}
