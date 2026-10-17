package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestFilterMenusByPermission_RespectsGrantedPermissionCode 修复回归：
// 删除 isElevatedMenuForRole + restrictedPrefixes 之后，菜单过滤必须纯按权限码判定，
// DB 已授权的低角色用户（如 security_admin 持 user:read）能看到 /admin/users 菜单，
// 不再被路径前缀硬编码遮蔽。
func TestFilterMenusByPermission_RespectsGrantedPermissionCode(t *testing.T) {
	svc := &MenuService{}
	menus := []*ent.Menu{
		{Name: "事件管理", Path: "/incidents", PermissionCode: "incident:read"},
		{Name: "工作流", Path: "/workflow", PermissionCode: "workflow:read"},
		{Name: "用户管理", Path: "/admin/users", PermissionCode: "user:read"},
		{Name: "租户设置", Path: "/admin/tenants", PermissionCode: "tenant:write"},
	}

	// 低角色 end_user 只持 workflow:read，应能看到 /workflow，看不到其他三个
	filtered := svc.filterMenusByPermission(
		menus,
		map[string]bool{"workflow:read": true},
		map[string]bool{"end_user": true},
	)
	if len(filtered) != 1 || filtered[0].Path != "/workflow" {
		t.Fatalf("end_user 仅持 workflow:read 应只看到 /workflow，实际: %+v", pathsOf(filtered))
	}

	// security_admin 持 user:read 应能看到 /admin/users（修复前会被路径前缀黑名单拦截）
	filtered = svc.filterMenusByPermission(
		menus,
		map[string]bool{"user:read": true},
		map[string]bool{"security_admin": true},
	)
	if len(filtered) != 1 || filtered[0].Path != "/admin/users" {
		t.Fatalf("security_admin 持 user:read 应看到 /admin/users，实际: %+v", pathsOf(filtered))
	}
}

func pathsOf(menus []*ent.Menu) []string {
	out := make([]string, 0, len(menus))
	for _, m := range menus {
		out = append(out, m.Path)
	}
	return out
}

// TestBackfillParentMenus_OrphanedSubtreePreserved 修复回归（源自 2026-10-06 未落地的
// wip-pre-sprint1 stash，2026-10-17 正式落地）：filterMenusByPermission 逐条独立过滤，
// 父菜单权限码不满足被剔除、子菜单权限码满足存活时，buildMenuTree 只从存活的根节点
// 出发建树，存活子菜单会因父节点缺失被整棵丢弃。backfillParentMenus 必须把父菜单
// 作为纯导航容器回填，让子树在树中可达。
func TestBackfillParentMenus_OrphanedSubtreePreserved(t *testing.T) {
	svc := &MenuService{}
	parentID := 10
	grandparentID := 20
	menus := []*ent.Menu{
		{ID: grandparentID, Name: "工作台", Path: "/workflow", PermissionCode: "approval:read"},
		{ID: parentID, Name: "审批中心", Path: "/approvals", PermissionCode: "approval:read", ParentID: &grandparentID},
		{Name: "待我审批", Path: "/approvals/pending", PermissionCode: "task:read", ParentID: &parentID},
	}

	// 用户只持 task:read：父/祖父两个 approval:read 节点都被过滤，只留下「待我审批」
	filtered := svc.filterMenusByPermission(
		menus,
		map[string]bool{"task:read": true},
		map[string]bool{"agent": true},
	)
	if len(filtered) != 1 || filtered[0].Path != "/approvals/pending" {
		t.Fatalf("过滤后应只剩 /approvals/pending，实际: %+v", pathsOf(filtered))
	}

	backfilled := svc.backfillParentMenus(filtered, menus)
	// 修复点：两级祖先都必须被回填，否则 buildMenuTree 找不到根节点仍丢子树
	got := map[string]bool{}
	for _, m := range backfilled {
		got[m.Path] = true
	}
	for _, want := range []string{"/workflow", "/approvals", "/approvals/pending"} {
		if !got[want] {
			t.Fatalf("回填后 %s 必须存在（导航容器），实际: %+v", want, pathsOf(backfilled))
		}
	}

	// 行为断言：经回填后的建树必须让「待我审批」挂在树上可达
	// （/workflow 按 isAdminMenu 归管理菜单桶，主/管理两桶都查）
	main, admin := svc.buildMenuTree(backfilled)
	if !menuTreeHasPath(main, "/approvals/pending") && !menuTreeHasPath(admin, "/approvals/pending") {
		t.Fatalf("改前缺陷：父节点被过滤导致 /approvals/pending 整棵丢失；回填后树中必须可达")
	}
}

// TestBackfillParentMenus_NoParentNoBackfill 全部子菜单都被过滤时，
// 父菜单不得被回填成空导航容器；已保留的父菜单也不得重复插入。
func TestBackfillParentMenus_NoParentNoBackfill(t *testing.T) {
	svc := &MenuService{}
	parentID := 30
	menus := []*ent.Menu{
		{ID: parentID, Name: "报表", Path: "/reports", PermissionCode: "report:read"},
		{Name: "服务使用", Path: "/reports/usage", PermissionCode: "report:read", ParentID: &parentID},
		{Name: "独立页", Path: "/standalone", PermissionCode: ""},
	}

	// 用户一个 report:read 都没有：只剩无码的「独立页」，父菜单不得回填
	filtered := svc.filterMenusByPermission(
		menus,
		map[string]bool{"irrelevant:read": true},
		map[string]bool{"end_user": true},
	)
	backfilled := svc.backfillParentMenus(filtered, menus)
	if len(backfilled) != 1 || backfilled[0].Path != "/standalone" {
		t.Fatalf("子菜单全被过滤时不得回填父菜单，实际: %+v", pathsOf(backfilled))
	}

	// 父菜单本来就存活：回填必须幂等，不得产生重复项
	// （过滤结果 = /reports + /reports/usage + 无码恒保留的 /standalone，共 3 项）
	filtered = svc.filterMenusByPermission(
		menus,
		map[string]bool{"report:read": true},
		map[string]bool{"manager": true},
	)
	backfilled = svc.backfillParentMenus(filtered, menus)
	if len(backfilled) != 3 {
		t.Fatalf("已存活的父菜单回填后不得重复，实际: %+v", pathsOf(backfilled))
	}
	reportsCount := 0
	for _, m := range backfilled {
		if m.Path == "/reports" {
			reportsCount++
		}
	}
	if reportsCount != 1 {
		t.Fatalf("/reports 在回填结果中必须恰好出现一次，实际 %d 次", reportsCount)
	}
}

// menuTreeHasPath 深度优先判断菜单树中是否存在目标路径。
func menuTreeHasPath(trees []dto.MenuDTO, target string) bool {
	for _, node := range trees {
		if node.Path == target {
			return true
		}
		if menuTreeHasPath(node.Children, target) {
			return true
		}
	}
	return false
}

// TestCreateMenu_DuplicatePathReturnsConflict 修复回归：menus 按 (tenant, path) 业务唯一
// （唯一索引由 20261011 迁移承担，测试库无索引），CreateMenu 侧 Exist 查重必须把重复
// 创建挡成 conflict——原先不查重，重复行会让 seedMenus 的 Only 报错、reconcile 失效。
func TestCreateMenu_DuplicatePathReturnsConflict(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	svc := NewMenuService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Menu Dup Tenant").
		SetCode("menu-dup").
		SetDomain("menu-dup.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	perm := "ticket:read"
	req := &dto.CreateMenuRequest{Name: "工单管理", Path: "/tickets", PermissionCode: &perm}

	created, err := svc.CreateMenu(ctx, req, tenant.ID)
	require.NoError(t, err)
	require.Equal(t, perm, *created.PermissionCode, "权限码必须一次性写入成功")

	_, err = svc.CreateMenu(ctx, req, tenant.ID)
	require.Error(t, err, "重复 path 必须被拒绝")
	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr), "应返回业务 conflict 错误，实际: %v", err)
	require.Equal(t, common.ConflictCode, bizErr.Code)

	// 同 path 不同租户互不影响（唯一键是组合键，不是全局 path）
	tenantB, err := client.Tenant.Create().
		SetName("Menu Dup Tenant B").
		SetCode("menu-dup-b").
		SetDomain("menu-dup-b.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	_, err = svc.CreateMenu(ctx, req, tenantB.ID)
	require.NoError(t, err, "另一租户同 path 应创建成功")
}

func intRef(v int) *int { return &v }

// TestUpdateMenu_ParentAssignmentSemantics 锁住 PUT /menus/{id} 的上级语义：
// 传 0 必须真的清成顶级菜单（此前只按指针判断，页面上「移除上级菜单」静默失效），
// 未传保持原值，指向其他租户 / 自身 / 自己的后代时拒绝且不落脏数据。
func TestUpdateMenu_ParentAssignmentSemantics(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	svc := NewMenuService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Menu Parent Tenant").SetCode("menu-parent").
		SetDomain("menu-parent.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().
		SetName("Menu Parent Tenant B").SetCode("menu-parent-b").
		SetDomain("menu-parent-b.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	parent, err := svc.CreateMenu(ctx, &dto.CreateMenuRequest{Name: "工单管理", Path: "/tickets"}, tenant.ID)
	require.NoError(t, err)
	require.Nil(t, parent.ParentID, "未传上级时不得写入 0 这类悬空引用")
	child, err := svc.CreateMenu(ctx, &dto.CreateMenuRequest{
		Name: "我的工单", Path: "/tickets/mine", ParentID: intRef(parent.ID),
	}, tenant.ID)
	require.NoError(t, err)
	otherTenantMenu, err := svc.CreateMenu(ctx, &dto.CreateMenuRequest{Name: "他租户工单", Path: "/tickets"}, tenantB.ID)
	require.NoError(t, err)

	// 只改名称时上级保持不动
	renamed, err := svc.UpdateMenu(ctx, child.ID, &dto.UpdateMenuRequest{Name: ptr("我的工单 V2")}, tenant.ID)
	require.NoError(t, err)
	require.NotNil(t, renamed.ParentID)
	require.Equal(t, parent.ID, *renamed.ParentID)

	// 显式 0 清空上级，数据库写 NULL 而不是残留 0
	cleared, err := svc.UpdateMenu(ctx, child.ID, &dto.UpdateMenuRequest{ParentID: intRef(0)}, tenant.ID)
	require.NoError(t, err)
	require.Nil(t, cleared.ParentID)
	dbRow, err := client.Menu.Get(ctx, child.ID)
	require.NoError(t, err)
	require.Nil(t, dbRow.ParentID, "清空上级必须写 NULL")

	// 跨租户上级：拒绝且不改数据
	_, err = svc.UpdateMenu(ctx, child.ID, &dto.UpdateMenuRequest{ParentID: intRef(otherTenantMenu.ID)}, tenant.ID)
	require.Error(t, err, "上级菜单必须属于当前租户")
	var invalid *common.AppError
	require.True(t, errors.As(err, &invalid), "应返回参数错误，实际: %v", err)
	require.Equal(t, http.StatusBadRequest, invalid.HTTPStatus)
	dbRow, err = client.Menu.Get(ctx, child.ID)
	require.NoError(t, err)
	require.Nil(t, dbRow.ParentID, "被拒绝的写入不得留下脏数据")

	// 恢复合法上级后拒绝成环：自身与自己的后代都不能当上级
	_, err = svc.UpdateMenu(ctx, child.ID, &dto.UpdateMenuRequest{ParentID: intRef(parent.ID)}, tenant.ID)
	require.NoError(t, err)

	_, err = svc.UpdateMenu(ctx, parent.ID, &dto.UpdateMenuRequest{ParentID: intRef(parent.ID)}, tenant.ID)
	require.Error(t, err, "上级不能是自身")
	_, err = svc.UpdateMenu(ctx, parent.ID, &dto.UpdateMenuRequest{ParentID: intRef(child.ID)}, tenant.ID)
	require.Error(t, err, "上级不能是自己的后代")
	dbRow, err = client.Menu.Get(ctx, parent.ID)
	require.NoError(t, err)
	require.Nil(t, dbRow.ParentID, "成环写入必须被挡在落库之前")
}

// TestDeleteMenu_StateAndTenantSemantics 锁住 DELETE /menus/{id} 的三条失败语义：
// 还有子菜单是 conflict（原先是裸 fmt.Errorf，响应兜底成 500/5001），
// 未知 ID 与跨租户 ID 都按未命中拒绝（原先按 0 行影响静默返回成功）。
func TestDeleteMenu_StateAndTenantSemantics(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	svc := NewMenuService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Menu Delete Tenant").SetCode("menu-del").
		SetDomain("menu-del.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().
		SetName("Menu Delete Tenant B").SetCode("menu-del-b").
		SetDomain("menu-del-b.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	parent, err := svc.CreateMenu(ctx, &dto.CreateMenuRequest{Name: "工单管理", Path: "/tickets"}, tenant.ID)
	require.NoError(t, err)
	child, err := svc.CreateMenu(ctx, &dto.CreateMenuRequest{
		Name: "我的工单", Path: "/tickets/mine", ParentID: intRef(parent.ID),
	}, tenant.ID)
	require.NoError(t, err)
	foreign, err := svc.CreateMenu(ctx, &dto.CreateMenuRequest{Name: "他租户菜单", Path: "/other"}, tenantB.ID)
	require.NoError(t, err)

	err = svc.DeleteMenu(ctx, parent.ID, tenant.ID)
	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr), "有子菜单应返回业务 conflict，实际: %v", err)
	require.Equal(t, common.ConflictCode, bizErr.Code)
	require.Equal(t, "请先删除子菜单", bizErr.Message)
	_, err = client.Menu.Get(ctx, parent.ID)
	require.NoError(t, err, "冲突拒绝不得删掉父菜单")

	var notFound *ent.NotFoundError
	err = svc.DeleteMenu(ctx, 999999, tenant.ID)
	require.True(t, errors.As(err, &notFound), "未知菜单应返回 not found，实际: %v", err)

	err = svc.DeleteMenu(ctx, foreign.ID, tenant.ID)
	require.True(t, errors.As(err, &notFound), "跨租户菜单应按未命中拒绝，实际: %v", err)
	_, err = client.Menu.Get(ctx, foreign.ID)
	require.NoError(t, err, "他租户菜单不得被删除")

	require.NoError(t, svc.DeleteMenu(ctx, child.ID, tenant.ID))
	require.NoError(t, svc.DeleteMenu(ctx, parent.ID, tenant.ID))
	_, err = client.Menu.Get(ctx, child.ID)
	require.True(t, ent.IsNotFound(err), "叶子菜单删除后不应存在")
	_, err = client.Menu.Get(ctx, parent.ID)
	require.True(t, ent.IsNotFound(err), "子菜单清空后父菜单应可删除")
	_, err = client.Menu.Get(ctx, foreign.ID)
	require.NoError(t, err, "删除操作不得波及他租户")
}
