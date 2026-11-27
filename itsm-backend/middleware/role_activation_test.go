package middleware

// 组织与权限域审计 P1-5：角色停用对主角色不生效。
//
// 现象：附加角色（user_roles m2m）在 RBACMiddleware 物化时按 r.IsActive 过滤，
// 停用即退出授权集合；但主角色（users.role）没有任何 IsActive 判定——
// 停用角色行后，持有该主角色的用户权限全部照旧生效。停用只撤销了一半。
//
// 本文件锁住四件事：
//  1. 停用的主角色在真实授权链路上必须被拒（走 AuthorizeResource，
//     即 RequirePermission 调用的同一判定函数，不是自造判据）；
//  2. 角色行缺失的三态语义不变（未初始化租户走默认基线、已配置租户空集
//     fail-closed）——停用不能被误判成「未配置」而回退默认权限；
//  3. 附加角色的既有行为不回退；
//  4. 平台超管 super_admin 的直通语义保留（见下方 super_admin 用例注释）。

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"itsm-backend/ent/role"
)

func TestRoleActivation_PrimaryRoleDeactivationRevokes(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	client, _ := openSharedCacheSQLite(t, "role_activation_primary")
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("Activation").SetCode("activation").SetDomain("activation.test").SetStatus("active").
		SaveX(ctx)
	// 主角色码取词表内的 agent（users.role 枚举值），与生产一致。
	r := client.Role.Create().
		SetCode("agent").SetName("服务台坐席").SetTenantID(tenant.ID).SaveX(ctx)
	perm := client.Permission.Create().
		SetCode("ticket:read").SetName("读取工单").
		SetResource("ticket").SetAction("read").
		SetDescription("查看工单列表").
		SetTenantID(tenant.ID).SaveX(ctx)
	client.RolePermission.Create().
		SetRoleID(r.ID).SetPermissionID(perm.ID).SetTenantID(tenant.ID).SaveX(ctx)

	allow := func() bool {
		InvalidateAllPermissionCaches()
		return AuthorizeResource(ctx, client, []string{r.Code}, "ticket", "read", tenant.ID)
	}

	// 反向验证的前提：修复前这里必须是 true，否则「停用后拒绝」会假绿。
	require.True(t, allow(), "启用状态的主角色必须正常放行")

	// 管理员在角色管理里停用该角色（roles.is_active=false）。
	require.NoError(t, client.Role.UpdateOneID(r.ID).SetIsActive(false).Exec(ctx))

	require.False(t, allow(), "停用的主角色必须立即失去授权（停用只撤销附加角色是原缺陷）")
}

// 停用不能被误判成「角色行不存在」——后者在 DBOnly 下会回退编译期默认权限基线，
// 把「显式撤销」变成「重新发放默认权限」。
func TestRoleActivation_DeactivatedRoleDoesNotRestoreDefaults(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	client, _ := openSharedCacheSQLite(t, "role_activation_defaults")
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("Defaults").SetCode("defaults").SetDomain("defaults.test").SetStatus("active").
		SaveX(ctx)
	r := client.Role.Create().
		SetCode("agent").SetName("服务台坐席").SetTenantID(tenant.ID).SetIsActive(false).SaveX(ctx)

	InvalidateAllPermissionCaches()
	require.False(t,
		AuthorizeResource(ctx, client, []string{r.Code}, "ticket", "read", tenant.ID),
		"停用角色不得经默认权限基线复活（agent 默认持有 ticket:read）")

	// 同一租户里再放一个启用角色：租户已配置，缺失角色必须空集 fail-closed，
	// 不得因「租户有角色行」而回退默认基线。
	other := client.Role.Create().
		SetCode("technician").SetName("技术员").SetTenantID(tenant.ID).SaveX(ctx)
	_ = other

	InvalidateAllPermissionCaches()
	require.False(t,
		AuthorizeResource(ctx, client, []string{"manager"}, "ticket", "read", tenant.ID),
		"租户已配置时缺失角色必须空集 fail-closed")
}

// 角色行缺失的三态语义必须与本次修复前完全一致：
// 未初始化租户（无任何角色行）走默认基线，已配置租户（缺该角色行）空集 fail-closed。
func TestRoleActivation_MissingRoleKeepsDBOnlyTriState(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	client, _ := openSharedCacheSQLite(t, "role_activation_tristate")
	ctx := context.Background()

	// 未初始化租户：没有任何角色行 → 走默认基线（agent 默认持 ticket:read）。
	fresh := client.Tenant.Create().
		SetName("Fresh").SetCode("fresh").SetDomain("fresh.test").SetStatus("active").
		SaveX(ctx)
	InvalidateAllPermissionCaches()
	require.True(t,
		AuthorizeResource(ctx, client, []string{"agent"}, "ticket", "read", fresh.ID),
		"未初始化租户仍走默认权限基线，既有契约不得改变")

	// 已配置租户：缺失该角色行 → 空集 fail-closed。
	configured := client.Tenant.Create().
		SetName("Configured").SetCode("configured").SetDomain("configured.test").SetStatus("active").
		SaveX(ctx)
	client.Role.Create().SetCode("technician").SetName("技术员").SetTenantID(configured.ID).SaveX(ctx)
	InvalidateAllPermissionCaches()
	require.False(t,
		AuthorizeResource(ctx, client, []string{"agent"}, "ticket", "read", configured.ID),
		"已配置租户缺失角色必须空集 fail-closed")
}

// 附加角色的既有行为不得回退：停用的 m2m 角色仍被 RBACMiddleware 排除。
func TestRoleActivation_AdditionalRoleDeactivationStillExcluded(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	client, _ := openSharedCacheSQLite(t, "role_activation_additional")
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("Additional").SetCode("additional").SetDomain("additional.test").SetStatus("active").
		SaveX(ctx)
	extra := client.Role.Create().
		SetCode("change_manager").SetName("变更经理").SetTenantID(tenant.ID).SaveX(ctx)
	perm := client.Permission.Create().
		SetCode("change:read").SetName("读取变更").
		SetResource("change").SetAction("read").
		SetDescription("查看变更单").
		SetTenantID(tenant.ID).SaveX(ctx)
	client.RolePermission.Create().
		SetRoleID(extra.ID).SetPermissionID(perm.ID).SetTenantID(tenant.ID).SaveX(ctx)

	InvalidateAllPermissionCaches()
	require.True(t,
		AuthorizeResource(ctx, client, []string{"change_manager"}, "change", "read", tenant.ID),
		"启用的附加角色必须放行")

	// 停用附加角色：授权集合必须收缩。
	require.NoError(t, client.Role.UpdateOneID(extra.ID).SetIsActive(false).Exec(ctx))
	InvalidateAllPermissionCaches()
	require.False(t,
		AuthorizeResource(ctx, client, []string{"change_manager"}, "change", "read", tenant.ID),
		"停用的附加角色必须失去授权")
}

// 平台超管 super_admin 的直通语义保留。
//
// 依据：super_admin 的授权不来自 roles 表的 role_permission 行（播种器对它不
// 建权限绑定，走 Login ["*"] 旁路），停用其角色行本就不改变任何授权面；
// 而它是平台级兜底身份（内置词表 + 删除双防线），一旦停用生效就会把
// 唯一的管理员恢复通道一并锁死。因此 super_admin 的停用不改变直通判定，
// 与既有 DeleteRole 的双防线（is_system + IsBuiltinCode）保持同一立场。
func TestRoleActivation_SuperAdminBypassPreserved(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	client, _ := openSharedCacheSQLite(t, "role_activation_super")
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("Super").SetCode("super").SetDomain("super.test").SetStatus("active").
		SaveX(ctx)
	super := client.Role.Create().
		SetCode("super_admin").SetName("超级管理员").SetTenantID(tenant.ID).SaveX(ctx)

	InvalidateAllPermissionCaches()
	require.True(t,
		AuthorizeResource(ctx, client, []string{super.Code}, "anything", "write", tenant.ID),
		"super_admin 直通必须保留")

	require.NoError(t, client.Role.UpdateOneID(super.ID).SetIsActive(false).Exec(ctx))
	InvalidateAllPermissionCaches()
	require.True(t,
		AuthorizeResource(ctx, client, []string{super.Code}, "anything", "write", tenant.ID),
		"super_admin 是平台兜底身份，停用其角色行不改变直通（与删除双防线同一立场）")
}

// 缓存与停用的交互：权限缓存里不能留下可继续放行的权限集。
// 停用是管理动作，缓存失效后重载必须读到停用态。
func TestRoleActivation_DeactivationSurvivesCacheWarmup(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	client, _ := openSharedCacheSQLite(t, "role_activation_cache")
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("Cache").SetCode("cache").SetDomain("cache.test").SetStatus("active").
		SaveX(ctx)
	r := client.Role.Create().
		SetCode("agent").SetName("服务台坐席").SetTenantID(tenant.ID).SaveX(ctx)
	perm := client.Permission.Create().
		SetCode("ticket:read").SetName("读取工单").
		SetResource("ticket").SetAction("read").
		SetDescription("查看工单列表").
		SetTenantID(tenant.ID).SaveX(ctx)
	client.RolePermission.Create().
		SetRoleID(r.ID).SetPermissionID(perm.ID).SetTenantID(tenant.ID).SaveX(ctx)

	// 先暖缓存（走真实写路径的失效逻辑，而不是手工清空）。
	require.True(t,
		AuthorizeResource(ctx, client, []string{r.Code}, "ticket", "read", tenant.ID),
		"暖缓存路径必须放行")

	// 停用并按 UpdateRole 的既有做法失效该角色缓存。
	require.NoError(t, client.Role.UpdateOneID(r.ID).SetIsActive(false).Exec(ctx))
	InvalidateRolePermissionCache(r.Code, tenant.ID)

	require.False(t,
		AuthorizeResource(ctx, client, []string{r.Code}, "ticket", "read", tenant.ID),
		"停用后即使缓存曾经是热的也必须拒绝")
}

// 非 DBOnly 模式（开发/测试环境）没有「角色行不存在即未配置」的分支，
// 停用角色会经由 RoleDefaultPermissions 重新拿到编译期默认权限——
// 停用必须真的停用，四种模式都要一致。
func TestRoleActivation_DeactivationHoldsInEveryPermissionMode(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})

	for _, mode := range []PermissionConfigMode{
		PermissionConfigModeDBOnly,
		PermissionConfigModeHardcodeOnly,
		PermissionConfigModeMerge,
		PermissionConfigModeFallback,
	} {
		t.Run(modeName(mode), func(t *testing.T) {
			PermissionConfig.Mode = mode
			InvalidateAllPermissionCaches()

			client, _ := openSharedCacheSQLite(t, "role_activation_modes_"+modeName(mode))
			ctx := context.Background()

			tenant := client.Tenant.Create().
				SetName("Modes").SetCode("modes").SetDomain("modes.test").SetStatus("active").
				SaveX(ctx)
			r := client.Role.Create().
				SetCode("agent").SetName("服务台坐席").SetTenantID(tenant.ID).SaveX(ctx)
			perm := client.Permission.Create().
				SetCode("ticket:read").SetName("读取工单").
				SetResource("ticket").SetAction("read").
				SetDescription("查看工单列表").
				SetTenantID(tenant.ID).SaveX(ctx)
			client.RolePermission.Create().
				SetRoleID(r.ID).SetPermissionID(perm.ID).SetTenantID(tenant.ID).SaveX(ctx)

			InvalidateAllPermissionCaches()
			require.True(t,
				AuthorizeResource(ctx, client, []string{r.Code}, "ticket", "read", tenant.ID),
				"启用状态必须放行（否则停用用例会假绿）")

			require.NoError(t, client.Role.UpdateOneID(r.ID).SetIsActive(false).Exec(ctx))
			InvalidateAllPermissionCaches()
			require.False(t,
				AuthorizeResource(ctx, client, []string{r.Code}, "ticket", "read", tenant.ID),
				"停用的主角色在 %s 模式下必须失去授权", modeName(mode))
		})
	}
}

func modeName(mode PermissionConfigMode) string {
	switch mode {
	case PermissionConfigModeDBOnly:
		return "db_only"
	case PermissionConfigModeHardcodeOnly:
		return "hardcode_only"
	case PermissionConfigModeMerge:
		return "merge"
	case PermissionConfigModeFallback:
		return "fallback"
	default:
		return "unknown"
	}
}

// 角色停用判定必须落在「角色是否生效」这一处语义上：停用与角色行缺失
// 是两种不同状态，SQL 层不得用 IsActive 过滤把两者合并。
func TestRoleActivation_DeactivatedIsNotTreatedAsMissing(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	client, _ := openSharedCacheSQLite(t, "role_activation_distinct")
	ctx := context.Background()

	tenant := client.Tenant.Create().
		SetName("Distinct").SetCode("distinct").SetDomain("distinct.test").SetStatus("active").
		SaveX(ctx)
	r := client.Role.Create().
		SetCode("agent").SetName("服务台坐席").SetTenantID(tenant.ID).SetIsActive(false).SaveX(ctx)
	_ = r

	// 直接断言底层判定源：停用行必须返回 errRoleInactive，
	// 而不是 ent.IsNotFound（那会让调用方误判成「角色行不存在」）。
	_, err := lookupEffectiveRole(ctx, client, "agent", tenant.ID)
	require.ErrorIs(t, err, errRoleInactive)

	// 角色行真的不存在时返回 NotFound，与停用严格可区分。
	_, missingErr := lookupEffectiveRole(ctx, client, "no_such_role", tenant.ID)
	require.NotErrorIs(t, missingErr, errRoleInactive)

	// 停用行确实存在于 roles 表（避免"其实是被 SQL 过滤掉了"的假验证）。
	count := client.Role.Query().
		Where(role.Code("agent"), role.TenantID(tenant.ID)).
		CountX(ctx)
	require.Equal(t, 1, count, "停用行仍必须存在于 roles 表")
}
