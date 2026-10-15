package workbench

import (
	"context"
	"database/sql"
	"testing"

	"itsm-backend/internal/datascope"

	"github.com/stretchr/testify/require"
)

// TestWorkbenchScopeFiltersNonAdminRows 验证非管理角色在跨域工作台
// 只能看到本人发起或受理的单据。
//
// 这是本轮修复的核心回归：此前 workbench 只靠路由层 dashboard:read 兜底，
// 不做行级过滤，普通角色能看到全租户的工单/事件/问题/变更。
func TestWorkbenchScopeFiltersNonAdminRows(t *testing.T) {
	f := setupWorkbenchTest(t)
	repo := NewRepository(workbenchTestDB(t, f))
	ctx := context.Background()

	// 全量 scope：管理角色视角，能看到 tenantA 的全部活跃记录
	all, err := repo.Query(ctx, WorkbenchQuery{
		TenantID: f.tenantA, Page: 1, PageSize: 50,
		DataScope: datascope.DataScopeAll,
	})
	require.NoError(t, err)
	require.NotEmpty(t, all.Items, "fixture 应当至少有条记录（前提是全量 scope 未被误伤）")

	// owner scope + 不存在的 viewer → 收窄到 0 条
	narrow, err := repo.Query(ctx, WorkbenchQuery{
		TenantID: f.tenantA, Page: 1, PageSize: 50,
		DataScope: datascope.DataScopeOwnedOrAssigned,
		ViewerID:  999999,
	})
	require.NoError(t, err)
	require.Empty(t, narrow.Items,
		"非相关 viewer 在 owner scope 下不应看到任何记录，实际 %d 条", len(narrow.Items))
	require.Equal(t, 0, narrow.Total)
}

// TestWorkbenchScopeZeroViewerIsSafe 验证 ViewerID<=0 时恒返回空（安全收窄），
// 而不是退化成"不过滤"。
func TestWorkbenchScopeZeroViewerIsSafe(t *testing.T) {
	f := setupWorkbenchTest(t)
	repo := NewRepository(workbenchTestDB(t, f))

	res, err := repo.Query(context.Background(), WorkbenchQuery{
		TenantID: f.tenantA, Page: 1, PageSize: 50,
		DataScope: datascope.DataScopeOwnedOrAssigned,
		ViewerID:  0, // 上下文缺 user_id 的情况
	})
	require.NoError(t, err)
	require.Empty(t, res.Items,
		"ViewerID=0 且非全量 scope 时必须安全收窄为空，实际 %d 条", len(res.Items))
}

// TestWorkbenchScopeOwnerSeesOwnRows 验证 owner scope 下本人相关记录可见
// （防止"收窄过头"把本人单据也过滤掉）。
func TestWorkbenchScopeOwnerSeesOwnRows(t *testing.T) {
	f := setupWorkbenchTest(t)
	repo := NewRepository(workbenchTestDB(t, f))

	res, err := repo.Query(context.Background(), WorkbenchQuery{
		TenantID: f.tenantA, Page: 1, PageSize: 50,
		DataScope: datascope.DataScopeOwnedOrAssigned,
		ViewerID:  f.actorA,
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.Items, "本人发起的单据必须可见，否则会误伤正常工作台")
}

// TestWorkbenchScopeCrossTenantStillBlocked 验证行级过滤不能替代租户隔离：
// 即使 viewer 在 A 租户，也不能看到 B 租户的数据。
func TestWorkbenchScopeCrossTenantStillBlocked(t *testing.T) {
	f := setupWorkbenchTest(t)
	repo := NewRepository(workbenchTestDB(t, f))

	// 全量 scope（管理角色）+ 租户 B：只能看到 B 自己的记录，看不到 A
	res, err := repo.Query(context.Background(), WorkbenchQuery{
		TenantID: f.tenantB, Page: 1, PageSize: 50,
		DataScope: datascope.DataScopeAll,
	})
	require.NoError(t, err)
	for _, item := range res.Items {
		require.Equal(t, f.tenantB, item.TenantID, "工作台泄露了其他租户的数据")
	}
}

// TestWorkbenchDomainRequesterColumnsMapped 锁定四域请求人列名映射。
// 列名写错会让 scope 过滤在 SQL 层直接报错（no such column），
// 而这个断言能在纯 Go 层先挡住。
func TestWorkbenchDomainRequesterColumnsMapped(t *testing.T) {
	want := map[string]string{
		"incident": "reporter_id",
		"change":   "created_by",
		"problem":  "created_by",
		"ticket":   "requester_id",
	}
	require.Len(t, workbenchDomains, len(want), "工作台域数量变化时需同步更新映射断言")
	for _, d := range workbenchDomains {
		require.NotEmpty(t, d.requesterCol, "域 %s 缺 requesterCol，owner scope 会生成非法 SQL", d.name)
		require.Equal(t, want[d.name], d.requesterCol, "域 %s 的请求人列名与 ent/schema 不一致", d.name)
	}
}

// workbenchTestDB 让测试拿到 fixture 使用的 *sql.DB。
func workbenchTestDB(t *testing.T, f *workbenchFixture) *sql.DB {
	t.Helper()
	db, ok := f.dbValue()
	require.True(t, ok, "fixture 未暴露 db，无法在仓储层测试")
	return db
}
