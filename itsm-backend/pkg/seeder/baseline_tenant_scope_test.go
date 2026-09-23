package seeder

import (
	"context"
	"testing"

	"itsm-backend/ent/department"
	"itsm-backend/ent/group"
	"itsm-backend/ent/menu"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/role"
	"itsm-backend/ent/tenant"
	"itsm-backend/pkg/tenantmode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The baseline helpers must be able to install into an arbitrary tenant, which
// is what lets provisioning reuse them instead of copying from the default
// tenant. Writing into the wrong tenant is the failure mode this pins down.
func TestWithBaselineTenantInstallsIntoTargetTenant(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	root, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)

	target, err := seeder.client.Tenant.Create().
		SetCode("tenant-b").SetName("Tenant B").
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)
	require.NotEqual(t, root.ID, target.ID)

	before, err := countTenantBaseline(ctx, seeder, root.ID)
	require.NoError(t, err)

	view := seeder.withBaselineTenant(target.ID)
	require.NoError(t, view.seedDepartments(ctx))
	view.seedRoles(ctx)
	view.seedPermissions(ctx)
	view.seedGroups(ctx)
	view.seedMenus(ctx)
	view.seedRolePermissions(ctx)

	after, err := countTenantBaseline(ctx, seeder, target.ID)
	require.NoError(t, err)
	assert.NotZero(t, after.roles)
	assert.NotZero(t, after.permissions)
	assert.NotZero(t, after.menus)
	assert.NotZero(t, after.groups)
	assert.NotZero(t, after.departments)

	untouched, err := countTenantBaseline(ctx, seeder, root.ID)
	require.NoError(t, err)
	assert.Equal(t, before, untouched, "installing for tenant B must not add rows to the default tenant")

	require.NoError(t, view.verifyIdentityRBAC(ctx))
	other := seeder.withBaselineTenant(root.ID)
	require.Error(t, other.verifyIdentityRBAC(ctx),
		"the default tenant has no baseline yet and must not pass by reading tenant B")
}

type baselineCounts struct {
	roles, permissions, menus, groups, departments int
}

func countTenantBaseline(ctx context.Context, s *Seeder, tenantID int) (baselineCounts, error) {
	counts := baselineCounts{}
	var err error
	if counts.roles, err = s.client.Role.Query().Where(role.TenantIDEQ(tenantID)).Count(ctx); err != nil {
		return counts, err
	}
	if counts.permissions, err = s.client.Permission.Query().Where(permission.TenantIDEQ(tenantID)).Count(ctx); err != nil {
		return counts, err
	}
	if counts.menus, err = s.client.Menu.Query().Where(menu.TenantIDEQ(tenantID)).Count(ctx); err != nil {
		return counts, err
	}
	if counts.groups, err = s.client.Group.Query().Where(group.TenantIDEQ(tenantID)).Count(ctx); err != nil {
		return counts, err
	}
	counts.departments, err = s.client.Department.Query().Where(department.TenantIDEQ(tenantID)).Count(ctx)
	return counts, err
}
