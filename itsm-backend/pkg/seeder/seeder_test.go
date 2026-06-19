package seeder

import (
	"context"
	"testing"

	"itsm-backend/config"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/mspallocation"
	"itsm-backend/ent/servicecatalog"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/standardchange"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	"itsm-backend/pkg/tenantmode"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestSeeder(t *testing.T, mode string) (*Seeder, context.Context) {
	t.Helper()
	t.Setenv("ADMIN_PASSWORD", "test-admin-password")
	t.Setenv("SEED_USER1_PASSWORD", "user123")
	t.Setenv("SEED_SECURITY1_PASSWORD", "sec123")

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		client.Close()
	})

	cfg := &config.Config{
		Deployment: config.DeploymentConfig{
			Mode: mode,
		},
	}

	return NewSeeder(client, zap.NewNop().Sugar(), cfg), context.Background()
}

func TestSeedDefaultTenantPrivateMode(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)

	rootTenant := seeder.seedDefaultTenant(ctx)
	require.NotNil(t, rootTenant)

	assert.Equal(t, "default", rootTenant.Code)
	assert.Equal(t, tenantmode.TenantTypeInternal, string(rootTenant.Type))
	assert.Equal(t, "Default Tenant", rootTenant.Name)
	assert.True(t, rootTenant.BillingEnabled)
	assert.Equal(t, "CNY", rootTenant.Currency)
	assert.Equal(t, "enterprise", rootTenant.ServiceTier)

	seeder.seedModeTenants(ctx, rootTenant)

	hqTenant, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("hq")).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, tenantmode.TenantTypeInternal, string(hqTenant.Type))
	assert.Equal(t, rootTenant.ID, hqTenant.ParentTenantID)
	assert.Equal(t, "HQ-001", hqTenant.CostCenterCode)
}

func TestSeedModeTenantsSaaSMSPCreatesCustomersAndAllocation(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModeSaaSMSP)

	rootTenant := seeder.seedDefaultTenant(ctx)
	require.NotNil(t, rootTenant)
	assert.Equal(t, tenantmode.TenantTypeMSPProvider, string(rootTenant.Type))

	seeder.seedAdmin(ctx)
	seeder.seedModeTenants(ctx, rootTenant)

	customerA, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("customer-a")).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, tenantmode.TenantTypeMSPCustomer, string(customerA.Type))
	assert.Equal(t, rootTenant.ID, customerA.MspProviderID)
	assert.Equal(t, rootTenant.ID, customerA.ParentTenantID)
	assert.Equal(t, "msp-enterprise", customerA.PlanCode)

	customerB, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("customer-b")).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, tenantmode.TenantTypeMSPCustomer, string(customerB.Type))
	assert.Equal(t, rootTenant.ID, customerB.MspProviderID)

	adminUser, err := seeder.client.User.Query().
		Where(user.UsernameEQ("admin"), user.TenantIDEQ(rootTenant.ID)).
		Only(ctx)
	require.NoError(t, err)

	exists, err := seeder.client.MSPAllocation.Query().
		Where(
			mspallocation.MspUserIDEQ(adminUser.ID),
			mspallocation.CustomerTenantIDEQ(customerA.ID),
			mspallocation.DeassignedAtIsNil(),
		).
		Exist(ctx)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSeedAllSaaSModeCreatesPlatformTenantAndAdmin(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModeSaaS)

	seeder.SeedAll(ctx)

	rootTenant, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, "SaaS Platform Tenant", rootTenant.Name)
	assert.Equal(t, tenantmode.TenantTypeInternal, string(rootTenant.Type))

	adminUser, err := seeder.client.User.Query().
		Where(user.UsernameEQ("admin"), user.TenantIDEQ(rootTenant.ID)).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, "super_admin", string(adminUser.Role))
}

func TestSeedAllSaaSMSPModeCreatesAdminAllocation(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModeSaaSMSP)

	seeder.SeedAll(ctx)

	rootTenant, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, tenantmode.TenantTypeMSPProvider, string(rootTenant.Type))

	adminUser, err := seeder.client.User.Query().
		Where(user.UsernameEQ("admin"), user.TenantIDEQ(rootTenant.ID)).
		Only(ctx)
	require.NoError(t, err)

	customers, err := seeder.client.Tenant.Query().
		Where(tenant.TypeEQ(tenant.TypeMspCustomer)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, customers, 2)

	for _, customer := range customers {
		exists, err := seeder.client.MSPAllocation.Query().
			Where(
				mspallocation.MspUserIDEQ(adminUser.ID),
				mspallocation.CustomerTenantIDEQ(customer.ID),
				mspallocation.DeassignedAtIsNil(),
			).
			Exist(ctx)
		require.NoError(t, err)
		assert.True(t, exists, "expected MSP allocation for customer %s", customer.Code)
	}
}

func TestSeedAllProductDefaultsDoNotCreateBusinessSamples(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)

	seeder.SeedAll(ctx)

	incidentCount, err := seeder.client.Incident.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, incidentCount)

	problemCount, err := seeder.client.Problem.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, problemCount)

	changeCount, err := seeder.client.Change.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, changeCount)

	knowledgeCount, err := seeder.client.KnowledgeArticle.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, knowledgeCount)

	serviceCatalogExists, err := seeder.client.ServiceCatalog.Query().Where(servicecatalog.NameNEQ("")).Exist(ctx)
	require.NoError(t, err)
	assert.True(t, serviceCatalogExists)

	slaExists, err := seeder.client.SLADefinition.Query().Where(sladefinition.NameNEQ("")).Exist(ctx)
	require.NoError(t, err)
	assert.True(t, slaExists)

	standardChangeExists, err := seeder.client.StandardChange.Query().Where(standardchange.TitleNEQ("")).Exist(ctx)
	require.NoError(t, err)
	assert.True(t, standardChangeExists)
}

// T012: verify 6 test accounts exist after seeding with correct roles (T005 verification)
func TestSeedRoleTestAccountsAllSixPresent(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)

	seeder.SeedAll(ctx)

	// admin / user1 / security1 live in the default tenant
	defaultTenant, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)

	// engineer1 / manager1 / tenant1admin live in tenant_test
	testTenant, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("tenant_test")).Only(ctx)
	require.NoError(t, err)

	// Define the 6 expected test accounts and their tenants/roles
	expectedAccounts := []struct {
		username string
		tenantID int
		role     user.Role
	}{
		{"admin", defaultTenant.ID, user.RoleSuperAdmin},
		{"user1", defaultTenant.ID, user.RoleEndUser},
		{"security1", defaultTenant.ID, user.RoleSecurity},
		{"engineer1", testTenant.ID, user.RoleTechnician},
		{"manager1", testTenant.ID, user.RoleManager},
		{"tenant1admin", testTenant.ID, user.RoleAdmin},
	}

	for _, acc := range expectedAccounts {
		u, err := seeder.client.User.Query().
			Where(user.UsernameEQ(acc.username), user.TenantIDEQ(acc.tenantID)).
			Only(ctx)
		require.NoError(t, err, "account %s should exist", acc.username)
		assert.Equal(t, acc.role, u.Role, "account %s should have role %s", acc.username, acc.role)
	}
}
