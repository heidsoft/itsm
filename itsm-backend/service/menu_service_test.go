package service

import (
	"context"
	"errors"
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
