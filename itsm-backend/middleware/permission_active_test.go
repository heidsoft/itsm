package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"itsm-backend/internal/authz"
)

// AUDIT-020：权限停用（IsActive=false）此前只对 AI 主体判定链生效。
// 人工 RBAC 路径直接读 Permission.Resource/Action，导致管理员在界面上
// 「停用/删除」一个权限后，人类用户仍能通过 RequirePermission。
//
// 本测试走真实 ent 库 + 真实 AuthorizeResource（RequirePermission 调用的
// 同一判定函数），锁住停用即失效、存量纯文本兼容、损坏信封 fail-closed。
func TestAuthorizeResource_DeactivatedPermissionDeniesHumanPath(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	client, _ := openSharedCacheSQLite(t, "perm_active")
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("Active").SetCode("active").SetDomain("active.test").SetStatus("active").
		SaveX(ctx)
	role := client.Role.Create().SetCode("active_manager").SetName("经理").SetTenantID(tenant.ID).SaveX(ctx)
	// 存量纯文本描述：兼容旧语义，仍应放行。
	perm := client.Permission.Create().
		SetCode("ticket:read").SetName("读取工单").
		SetResource("ticket").SetAction("read").
		SetDescription("查看工单列表").
		SetTenantID(tenant.ID).SaveX(ctx)
	client.RolePermission.Create().
		SetRoleID(role.ID).SetPermissionID(perm.ID).SetTenantID(tenant.ID).SaveX(ctx)

	allow := func() bool {
		InvalidateAllPermissionCaches()
		return AuthorizeResource(ctx, client, []string{role.Code}, "ticket", "read", tenant.ID)
	}

	require.True(t, allow(), "baseline：存量纯文本权限必须继续放行")

	// 管理员停用该码：与 UpdatePermissionPolicy / DeletePermission 写入格式一致。
	_, policy, err := authz.DecodeDefinition(perm.Description, perm.Resource, perm.Action)
	require.NoError(t, err)
	policy.IsActive = false
	encoded, err := authz.EncodeDefinition("查看工单列表", policy)
	require.NoError(t, err)
	_, err = client.Permission.UpdateOneID(perm.ID).SetDescription(encoded).Save(ctx)
	require.NoError(t, err)

	require.False(t, allow(), "已停用的权限码在人工 RBAC 路径上必须不可见")

	// 重新启用后应立即恢复（缓存失效由策略写入侧负责）。
	policy.IsActive = true
	encoded, err = authz.EncodeDefinition("查看工单列表", policy)
	require.NoError(t, err)
	_, err = client.Permission.UpdateOneID(perm.ID).SetDescription(encoded).Save(ctx)
	require.NoError(t, err)
	require.True(t, allow(), "重新启用后必须恢复授权")

	// 以 "{" 开头但解不开的信封：与 authz.Registry 一致 fail-closed。
	_, err = client.Permission.UpdateOneID(perm.ID).
		SetDescription(`{"v":9,"policy":broken`).Save(ctx)
	require.NoError(t, err)
	require.False(t, allow(), "损坏的策略信封不得按旧语义放行")
}
