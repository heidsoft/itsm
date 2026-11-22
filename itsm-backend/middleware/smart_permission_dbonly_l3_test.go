package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent/permission"
	"itsm-backend/internal/authz"
)

// AUDIT-021：DBOnly 下 L3 URL 推断此前在显式映射未命中时回退到编译期
// RoleDefaultPermissions，绕过 L4 的模式 gate。管理员撤销的权限仍可通过
// 「/api/v1/{resource}/...」这类未登记路径拿到授权。
//
// 本测试用真实 ent 库 + 真实 SmartCheckPermission，锁住：
//   - configured（角色行存在但无授权）：L3 不得发放默认权限
//   - unconfigured（租户没有任何角色）：保留既有默认兜底语义
func TestSmartCheckPermission_DBOnlyL3UsesDatabaseNotHardcodedDefaults(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	client, _ := openSharedCacheSQLite(t, "l3_dbonly")
	ctx := context.Background()
	path := "/api/v1/ticket/anything"

	// 该路径不在生成的预检映射里，L3 只能靠 URL 推断；资源段是单数 ticket，
	// GET 推断为 ticket:read——正是 it_admin 的内置默认权限。
	require.Nil(t, getPermissionFromPath("GET", path),
		"夹具必须走 URL 推断分支，否则本测试无法证明 L3 的行为")

	tenant := client.Tenant.Create().
		SetName("L3").SetCode("l3").SetDomain("l3.test").SetStatus("active").
		SaveX(ctx)

	// unconfigured：租户没有任何角色 → 沿用 authz 默认兜底，保持既有行为。
	require.True(t, SmartCheckPermission(newL3Context(path), client, "it_admin", "GET", path, tenant.ID),
		"DBOnly 未配置租户仍应使用默认权限兜底")

	// configured：角色行存在但显式无授权 → DBOnly 必须以 DB 为准（空集=fail-closed）。
	role := client.Role.Create().SetCode("it_admin").SetName("IT管理员").SetTenantID(tenant.ID).SaveX(ctx)
	require.False(t, SmartCheckPermission(newL3Context(path), client, "it_admin", "GET", path, tenant.ID),
		"DBOnly 已配置角色的 L3 不得回退到编译期默认权限")

	// 显式授予后应恢复。
	perm := client.Permission.Create().
		SetCode("ticket:read").SetName("读取工单").SetResource("ticket").SetAction("read").
		SetDescription("查看工单").SetTenantID(tenant.ID).SaveX(ctx)
	client.RolePermission.Create().
		SetRoleID(role.ID).SetPermissionID(perm.ID).SetTenantID(tenant.ID).SaveX(ctx)
	InvalidateAllPermissionCaches()

	require.True(t, SmartCheckPermission(newL3Context(path), client, "it_admin", "GET", path, tenant.ID),
		"DB 已授权时 L3 必须放行")

	// 停用权限后（复用 AUDIT-020 的过滤逻辑）应再次拒绝。
	_, policy, err := authz.DecodeDefinition(perm.Description, perm.Resource, perm.Action)
	require.NoError(t, err)
	policy.IsActive = false
	encoded, err := authz.EncodeDefinition(perm.Description, policy)
	require.NoError(t, err)
	_, err = client.Permission.Update().
		Where(permission.IDEQ(perm.ID), permission.TenantIDEQ(tenant.ID)).
		SetDescription(encoded).Save(ctx)
	require.NoError(t, err)
	InvalidateAllPermissionCaches()

	require.False(t, SmartCheckPermission(newL3Context(path), client, "it_admin", "GET", path, tenant.ID),
		"停用权限不得经 L3 复活")
}

// newL3Context 返回带请求的 gin 上下文（SmartCheckPermission 从 Request.Context 取 ctx）。
func newL3Context(path string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", path, nil)
	return c
}
