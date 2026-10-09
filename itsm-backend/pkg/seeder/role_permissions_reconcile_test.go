package seeder

import (
	"context"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/role"
	"itsm-backend/ent/rolepermission"
	"itsm-backend/ent/tenant"
	"itsm-backend/pkg/tenantmode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rolePermFixture 为 target 租户装好角色、权限基线，并持有 agent 角色
// （内置码集含 sla:read 不含 sla:write）与绑定了该租户的播种视图。
type rolePermFixture struct {
	seeder   *Seeder
	ctx      context.Context
	tenantID int
	view     *Seeder
	ops      *ent.Role
}

func newRolePermFixture(t *testing.T, seeder *Seeder, ctx context.Context, code string) *rolePermFixture {
	t.Helper()
	target, err := seeder.client.Tenant.Create().SetCode(code).SetName(code).
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)
	view := seeder.withBaselineTenant(target.ID)
	view.seedRoles(ctx)
	view.seedPermissions(ctx)
	ops, err := seeder.client.Role.Query().
		Where(role.CodeEQ("agent"), role.TenantIDEQ(target.ID)).
		Only(ctx)
	require.NoError(t, err)
	return &rolePermFixture{seeder: seeder, ctx: ctx, tenantID: target.ID, view: view, ops: ops}
}

// grantExists 查询 (role, permission, tenant) 授权行是否存在。
func (f *rolePermFixture) grantExists(t *testing.T, code string) bool {
	t.Helper()
	p, err := f.seeder.client.Permission.Query().
		Where(permission.CodeEQ(code), permission.TenantIDEQ(f.tenantID)).
		Only(f.ctx)
	require.NoError(t, err)
	exists, err := f.seeder.client.RolePermission.Query().
		Where(rolepermission.RoleIDEQ(f.ops.ID), rolepermission.PermissionID(p.ID), rolepermission.TenantIDEQ(f.tenantID)).
		Exist(f.ctx)
	require.NoError(t, err)
	return exists
}

// hotFixGrant 模拟运维在数据库里手工补的授权行。
func (f *rolePermFixture) hotFixGrant(t *testing.T, code string) {
	t.Helper()
	p, err := f.seeder.client.Permission.Query().
		Where(permission.CodeEQ(code), permission.TenantIDEQ(f.tenantID)).
		Only(f.ctx)
	require.NoError(t, err)
	_, err = f.seeder.client.RolePermission.Create().
		SetRoleID(f.ops.ID).SetPermissionID(p.ID).SetTenantID(f.tenantID).
		Save(f.ctx)
	require.NoError(t, err)
}

// TestSeedRolePermissionsPreservesManualGrants 只增不减：内置码集之外的既有授权
// （运维手工热修）在下一次播种后必须仍然存活。回归场景（R2-d 修复前）：
// seedRolePermissions 每次按内置码集反向 Delete 不在清单内的授权行，
// 手工补的授权会被静默撤销。
func TestSeedRolePermissionsPreservesManualGrants(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	f := newRolePermFixture(t, seeder, ctx, "rp-additive")

	f.view.seedRolePermissions(ctx)
	require.True(t, f.grantExists(t, "sla:read"), "内置码集授权必须在播种后存在")

	f.hotFixGrant(t, "sla:write")

	f.view.seedRolePermissions(ctx)
	f.view.seedRolePermissions(ctx)

	assert.True(t, f.grantExists(t, "sla:write"), "手工热修授权在下一次播种后必须仍然存活（只增不减契约）")
	assert.True(t, f.grantExists(t, "sla:read"), "内置码集授权不受影响")
}

// TestSeedRolePermissionsAppliesExplicitRetirement 显式退役：只增不减契约下
// 唯一的收缩通道。退役清单内的授权被移除且重跑不得复活；清单外的授权不受影响。
// 场景模拟旧安装遗留行：内置码集演化后 agent 不再包含 sla:write，
// 但存量库中该授权行仍在——登记退役清单后由播种器移除。
func TestSeedRolePermissionsAppliesExplicitRetirement(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	f := newRolePermFixture(t, seeder, ctx, "rp-retire")

	f.view.seedRolePermissions(ctx)
	f.hotFixGrant(t, "sla:write")
	require.True(t, f.grantExists(t, "sla:write"))

	f.view.retiredRolePermissions = map[string][]string{"agent": {"sla:write"}}
	f.view.seedRolePermissions(ctx)

	assert.False(t, f.grantExists(t, "sla:write"), "退役清单内的授权必须被移除")
	assert.True(t, f.grantExists(t, "ticket:read"), "退役清单外的内置授权不受影响")

	f.view.seedRolePermissions(ctx)
	assert.False(t, f.grantExists(t, "sla:write"), "重跑不得复活已退役的授权")
}
