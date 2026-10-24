package middleware

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
)

// 回归：权限加载失败不得被当成「DB 显式配置了空权限集」。
// 线上表现：manager 角色在若干分钟内对 /tickets、/service-requests、/bpmn/tasks 全量 403/2003
// 后自愈，而 roles/permissions 数据完好。根因是联表查询报错后仍把空集合写入进程内缓存，
// DBOnly 把空集合解释为显式撤销，于是在整个 TTL 窗口内对该角色 fail-closed。
func TestPermissionCache_TransientLoadErrorMustNotLockOutRole(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	client, raw := openSharedCacheSQLite(t, "perm_poison")
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("Poison").SetCode("poison").SetDomain("poison.test").SetStatus("active").
		SaveX(ctx)
	role := client.Role.Create().SetCode("poison_manager").SetName("经理").SetTenantID(tenant.ID).SaveX(ctx)
	perm := client.Permission.Create().
		SetCode("ticket:read").SetName("读取工单").SetResource("ticket").SetAction("read").
		SetTenantID(tenant.ID).SaveX(ctx)
	client.RolePermission.Create().
		SetRoleID(role.ID).SetPermissionID(perm.ID).SetTenantID(tenant.ID).SaveX(ctx)

	require.True(t, AuthorizeResource(ctx, client, []string{role.Code}, "ticket", "read", tenant.ID),
		"baseline：DB 已授权的权限必须命中")

	cacheKey := role.Code + "_" + strconv.Itoa(tenant.ID)
	invalidatePermissionCache(role.Code, tenant.ID)

	// 角色行仍可查、联表查询报错：等价于线上一次瞬时 DB 故障或请求 ctx 取消。
	renameTable(t, raw, "role_permissions", "role_permissions_offline")

	require.False(t, AuthorizeResource(ctx, client, []string{role.Code}, "ticket", "read", tenant.ID),
		"加载失败当次必须 fail-closed")

	permissionCacheLock.RLock()
	_, poisoned := permissionCache[cacheKey]
	permissionCacheLock.RUnlock()
	require.False(t, poisoned, "加载失败不得写入权限缓存")

	renameTable(t, raw, "role_permissions_offline", "role_permissions")

	require.True(t, AuthorizeResource(ctx, client, []string{role.Code}, "ticket", "read", tenant.ID),
		"故障恢复后必须立刻重新授权，不得在 TTL 窗口内锁死整个角色")
}

// 回归：DBOnly 下角色查询报错（非 NotFound）不得被当成「租户未配置 RBAC」，
// 否则瞬时故障会回退硬编码默认权限，即 fail-open。
func TestAuthorizeResource_DBOnly_RoleQueryErrorFailsClosed(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	client, raw := openSharedCacheSQLite(t, "perm_failopen")
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("FailOpen").SetCode("failopen").SetDomain("failopen.test").SetStatus("active").
		SaveX(ctx)

	require.True(t, hasPermissionInSlice(RoleDefaultPermissions("agent"), "ticket", "read"),
		"用例前提：agent 硬编码默认权限含 ticket:read，命中兜底即证明 fail-open")

	renameTable(t, raw, "roles", "roles_offline")
	t.Cleanup(func() { renameTable(t, raw, "roles_offline", "roles") })

	if AuthorizeResource(ctx, client, []string{"agent"}, "ticket", "read", tenant.ID) {
		t.Fatal("roles 查询报错必须按 unavailable fail-closed，不得回退硬编码默认权限")
	}
}

// 守护既有语义：角色行确实不存在（NotFound）仍走硬编码兜底，新装租户不该全量 403。
func TestAuthorizeResource_DBOnly_MissingRoleRowUsesHardcodedDefaults(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	client, _ := openSharedCacheSQLite(t, "perm_unconfigured")
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("Fresh").SetCode("fresh").SetDomain("fresh.test").SetStatus("active").
		SaveX(ctx)

	require.True(t, AuthorizeResource(ctx, client, []string{"agent"}, "ticket", "read", tenant.ID),
		"未跑 RBAC 初始化的租户应走 authz 默认权限兜底")
}

func TestAuthorizeResource_DBOnly_DeletedRoleDoesNotRestoreDefaults(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	client, _ := openSharedCacheSQLite(t, "perm_deleted_role")
	ctx := context.Background()
	tenant := client.Tenant.Create().
		SetName("Deleted").SetCode("deleted-role").SetDomain("deleted-role.test").SetStatus("active").
		SaveX(ctx)
	client.Role.Create().SetCode("other_role").SetName("Other").SetTenantID(tenant.ID).SaveX(ctx)

	require.False(t, AuthorizeResource(ctx, client, []string{"agent"}, "ticket", "read", tenant.ID),
		"租户已有 RBAC 角色时，缺失的 agent 角色必须按撤销处理，不得恢复默认权限")
}

func openSharedCacheSQLite(t *testing.T, name string) (*ent.Client, *sql.DB) {
	t.Helper()
	dsn := "file:" + name + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	raw, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	require.NoError(t, raw.Ping())
	t.Cleanup(func() {
		raw.Close()
		client.Close()
	})
	return client, raw
}

func renameTable(t *testing.T, raw *sql.DB, from, to string) {
	t.Helper()
	_, err := raw.Exec("ALTER TABLE " + from + " RENAME TO " + to)
	require.NoError(t, err)
}
