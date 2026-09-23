package seeder

import (
	"context"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/menu"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/role"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/tickettype"
	"itsm-backend/ent/user"
	"itsm-backend/internal/initialization"
	"itsm-backend/pkg/tenantmode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func newTenantScopeFixture(t *testing.T) (*Seeder, context.Context, *ent.Tenant) {
	t.Helper()
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	target, err := seeder.client.Tenant.Create().
		SetCode("tenant-install").SetName("Tenant Install").
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)
	return seeder, ctx, target
}

func installTenantBaseline(t *testing.T, seeder *Seeder, ctx context.Context, scope initialization.Scope) {
	t.Helper()
	components, err := ProductionInitializers(seeder)
	require.NoError(t, err)
	for _, component := range components {
		plan, err := component.Plan(ctx, scope)
		require.NoError(t, err)
		_, err = applyTestComponent(ctx, seeder, component, scope, plan)
		require.NoError(t, err, component.Name())
		require.NoError(t, component.Verify(ctx, scope, plan), component.Name())
	}
}

func TestTenantScopeInstallsBaselineIntoTargetTenant(t *testing.T) {
	seeder, ctx, target := newTenantScopeFixture(t)
	root, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ(platformTenantCode)).Only(ctx)
	require.NoError(t, err)
	scope := initialization.Scope{Type: "tenant", ID: int64(target.ID)}

	installTenantBaseline(t, seeder, ctx, scope)

	roles, err := seeder.client.Role.Query().Where(role.TenantIDEQ(target.ID)).Count(ctx)
	require.NoError(t, err)
	assert.NotZero(t, roles)
	permissions, err := seeder.client.Permission.Query().Where(permission.TenantIDEQ(target.ID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(seeder.expectedPermissions), permissions)
	menuCount, err := seeder.client.Menu.Query().Where(menu.TenantIDEQ(target.ID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(seeder.expectedMenus), menuCount)

	// Menu hierarchy must be rebuilt for the target tenant, not left flat.
	children, err := seeder.client.Menu.Query().Where(
		menu.TenantIDEQ(target.ID), menu.ParentIDNotNil(),
	).Count(ctx)
	require.NoError(t, err)
	assert.NotZero(t, children, "tenant menus must keep their parent links")

	// Template rows need an audit owner; provisioning installs a non-loginable
	// system account instead of borrowing a user from another tenant.
	account, err := seeder.client.User.Query().Where(
		user.UsernameEQ(tenantSystemAccountName(target.ID)), user.TenantIDEQ(target.ID),
	).Only(ctx)
	require.NoError(t, err)
	assert.False(t, account.Active, "the baseline account must stay inactive")
	assert.Error(t, bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte("")),
		"the baseline account must never authenticate")

	types, err := seeder.client.TicketType.Query().Where(tickettype.TenantIDEQ(int64(target.ID))).All(ctx)
	require.NoError(t, err)
	require.Len(t, types, len(ticketTypeDefinitions()))
	for _, record := range types {
		assert.Equal(t, int64(account.ID), record.CreatedBy, "template rows belong to the tenant system account")
	}

	// The platform tenant must be untouched by a tenant-scoped install.
	platformRoles, err := seeder.client.Role.Query().Where(role.TenantIDEQ(root.ID)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, platformRoles, "tenant install must not write into the default tenant")
}

func TestTenantScopeIsIdempotent(t *testing.T) {
	seeder, ctx, target := newTenantScopeFixture(t)
	scope := initialization.Scope{Type: "tenant", ID: int64(target.ID)}

	installTenantBaseline(t, seeder, ctx, scope)
	before, err := seeder.client.Permission.Query().Where(permission.TenantIDEQ(target.ID)).Count(ctx)
	require.NoError(t, err)
	accounts, err := seeder.client.User.Query().Where(user.TenantIDEQ(target.ID)).Count(ctx)
	require.NoError(t, err)

	installTenantBaseline(t, seeder, ctx, scope)

	after, err := seeder.client.Permission.Query().Where(permission.TenantIDEQ(target.ID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, before, after, "re-install must not duplicate baseline rows")
	repeatAccounts, err := seeder.client.User.Query().Where(user.TenantIDEQ(target.ID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, accounts, repeatAccounts, "re-install must not duplicate the baseline account")
}

func TestVerifyRejectsCrossTenantBaseline(t *testing.T) {
	seeder, ctx, target := newTenantScopeFixture(t)
	installTenantBaseline(t, seeder, ctx, initialization.Scope{Type: "tenant", ID: int64(target.ID)})

	components, err := ProductionInitializers(seeder)
	require.NoError(t, err)
	root, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ(platformTenantCode)).Only(ctx)
	require.NoError(t, err)

	// Tenant B is fully installed; tenant A (the empty default tenant) must not
	// pass verification by reading tenant B's rows.
	_, err = applyTestComponent(ctx, seeder, components[0], initialization.Scope{Type: "platform", ID: 0},
		mustPlan(t, ctx, components[0], initialization.Scope{Type: "platform", ID: 0}))
	require.NoError(t, err, "platform install is needed before scope isolation is meaningful")
	deleted, err := seeder.client.User.Delete().Where(
		user.TenantIDEQ(root.ID), user.UsernameEQ("admin"),
	).Exec(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, deleted)
	plan, err := components[0].Plan(ctx, initialization.Scope{Type: "tenant", ID: int64(root.ID)})
	require.NoError(t, err)
	require.Error(t, components[0].Verify(ctx, initialization.Scope{Type: "tenant", ID: int64(root.ID)}, plan),
		"the default tenant cannot pass tenant-scoped verification after losing its administrator")
}

func TestScopeValidationRejectsMismatchedScope(t *testing.T) {
	seeder, ctx, _ := newTenantScopeFixture(t)
	components, err := ProductionInitializers(seeder)
	require.NoError(t, err)
	plan, err := components[0].Plan(ctx, initialization.Scope{})
	require.NoError(t, err)

	for _, scope := range []initialization.Scope{
		{Type: "platform", ID: 7},
		{Type: "tenant", ID: 0},
		{Type: "tenant", ID: -1},
		{Type: "", ID: 0},
		{Type: "enterprise", ID: 3},
	} {
		_, err := components[0].Apply(ctx, scope, plan, seeder.sqlDriver)
		require.Error(t, err, scope)
		require.Error(t, components[0].Verify(ctx, scope, plan), scope)
	}
	require.NoError(t, initialization.ValidateScope(initialization.Scope{Type: "platform", ID: 0}))
}

func mustPlan(t *testing.T, ctx context.Context, component initialization.Initializer, scope initialization.Scope) initialization.Plan {
	t.Helper()
	plan, err := component.Plan(ctx, scope)
	require.NoError(t, err)
	return plan
}
