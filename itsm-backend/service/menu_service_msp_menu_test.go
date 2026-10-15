package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	enttenant "itsm-backend/ent/tenant"
	entuser "itsm-backend/ent/user"
	"itsm-backend/middleware"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 菜单入口必须与接口授权同口径：msp:read 是 12 个内置角色共有的读码，而
// /api/v1/msp/* 逐请求要求「服务商租户 + users.msp_role + allocation」
// （见 middleware.MSPMiddleware 与 347d44511）。此前 GetUserMenus 只看权限码，
// end_user 也会拿到一个点进去必然是权限拒绝页的「客户管理 (MSP)」菜单。

func seedRBACMenus(t *testing.T, client *ent.Client, tenantID int, roleCode string) {
	t.Helper()
	ctx := context.Background()

	r, err := client.Role.Create().
		SetName(roleCode).
		SetCode(roleCode).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	// 两个角色码一致的调用者拿到同一份授权，唯一变量只剩 MSP 成员身份。
	for _, p := range []struct{ resource, action string }{
		{"msp", "read"},
		{"ticket", "read"},
	} {
		perm, err := client.Permission.Create().
			SetCode(p.resource + ":" + p.action).
			SetName(p.resource + " " + p.action).
			SetResource(p.resource).
			SetAction(p.action).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		_, err = client.RolePermission.Create().
			SetRoleID(r.ID).
			SetPermissionID(perm.ID).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
	}

	for _, m := range []struct{ name, path, code string }{
		{"工单管理", "/tickets", "ticket:read"},
		{"客户管理 (MSP)", "/msp", "msp:read"},
	} {
		_, err = client.Menu.Create().
			SetName(m.name).
			SetPath(m.path).
			SetPermissionCode(m.code).
			SetTenantID(tenantID).
			SetIsVisible(true).
			SetIsEnabled(true).
			Save(ctx)
		require.NoError(t, err)
	}
}

func seedMSPMenuUser(t *testing.T, client *ent.Client, username, tenantType, mspRole string) (int, int) {
	t.Helper()
	ctx := context.Background()

	tenantRow, err := client.Tenant.Create().
		SetName("tenant-" + username).
		SetCode("code-" + username).
		SetType(enttenant.Type(tenantType)).
		Save(ctx)
	require.NoError(t, err)

	u, err := client.User.Create().
		SetUsername(username).
		SetEmail(username + "@example.com").
		SetName(username).
		SetPasswordHash("hash").
		SetRole("agent").
		SetTenantID(tenantRow.ID).
		Save(ctx)
	require.NoError(t, err)

	if mspRole != "" {
		_, err = client.User.UpdateOne(u).SetMspRole(entuser.MspRole(mspRole)).Save(ctx)
		require.NoError(t, err)
	}
	return u.ID, tenantRow.ID
}

func menuPaths(resp *dto.MenuTreeResponse) map[string]bool {
	paths := make(map[string]bool)
	var walk func(items []dto.MenuDTO)
	walk = func(items []dto.MenuDTO) {
		for _, m := range items {
			paths[m.Path] = true
			walk(m.Children)
		}
	}
	walk(resp.Main)
	walk(resp.Admin)
	return paths
}

func TestGetUserMenus_MSPEntryRequiresProviderMembership(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:menu_msp_scope?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewMenuService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	staffID, providerTenantID := seedMSPMenuUser(t, client, "msp_staff", "msp_provider", "provider_admin")
	plainID, customerTenantID := seedMSPMenuUser(t, client, "plain_agent", "standard", "")

	// 两个租户授予同一角色码、同一权限集，菜单基线也一致。
	seedRBACMenus(t, client, providerTenantID, "agent")
	seedRBACMenus(t, client, customerTenantID, "agent")

	t.Run("服务商员工看到 MSP 菜单", func(t *testing.T) {
		resp, err := svc.GetUserMenus(ctx, staffID, providerTenantID)
		require.NoError(t, err)
		require.Contains(t, menuPaths(resp), "/msp")
		require.Contains(t, menuPaths(resp), "/tickets")
	})

	t.Run("同权限码的非成员看不到必然 403 的入口", func(t *testing.T) {
		resp, err := svc.GetUserMenus(ctx, plainID, customerTenantID)
		require.NoError(t, err)
		require.NotContains(t, menuPaths(resp), "/msp",
			"msp:read 不能单独决定 MSP 菜单可见性，必须同时是服务商成员")
		require.Contains(t, menuPaths(resp), "/tickets")
	})

	t.Run("部署模式关闭 MSP 时对成员也隐藏", func(t *testing.T) {
		middleware.SetMSPEnabled(false)
		t.Cleanup(func() { middleware.SetMSPEnabled(true) })

		resp, err := svc.GetUserMenus(ctx, staffID, providerTenantID)
		require.NoError(t, err)
		require.NotContains(t, menuPaths(resp), "/msp")
	})
}
