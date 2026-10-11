package service

import (
	"testing"

	"itsm-backend/ent"
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