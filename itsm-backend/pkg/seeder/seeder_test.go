package seeder

import (
	"context"
	"testing"

	"itsm-backend/config"
	"itsm-backend/ent/enttest"
	_ "itsm-backend/ent/runtime"
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
	"golang.org/x/crypto/bcrypt"
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

	hqExists, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("hq")).Exist(ctx)
	require.NoError(t, err)
	assert.False(t, hqExists)
}

func TestProductSeedDoesNotCreateSampleMSPCustomers(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModeSaaSMSP)

	seeder.SeedAll(ctx)

	for _, code := range []string{"customer-a", "customer-b"} {
		exists, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ(code)).Exist(ctx)
		require.NoError(t, err)
		assert.False(t, exists)
	}
	allocationCount, err := seeder.client.MSPAllocation.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, allocationCount)
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

func TestSeedAllSaaSMSPModeCreatesOnlyProviderTenant(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModeSaaSMSP)

	seeder.SeedAll(ctx)

	rootTenant, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, tenantmode.TenantTypeMSPProvider, string(rootTenant.Type))

	adminExists, err := seeder.client.User.Query().
		Where(user.UsernameEQ("admin"), user.TenantIDEQ(rootTenant.ID)).
		Exist(ctx)
	require.NoError(t, err)
	assert.True(t, adminExists)

	customers, err := seeder.client.Tenant.Query().
		Where(tenant.TypeEQ(tenant.TypeMspCustomer)).
		All(ctx)
	require.NoError(t, err)
	assert.Empty(t, customers)

	allocationCount, err := seeder.client.MSPAllocation.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, allocationCount)
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

	releaseCount, err := seeder.client.Release.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, releaseCount)

	assetLicenseCount, err := seeder.client.AssetLicense.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, assetLicenseCount)

	knownErrorCount, err := seeder.client.KnownError.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, knownErrorCount)

	ticketCount, err := seeder.client.Ticket.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, ticketCount)

	assetCount, err := seeder.client.Asset.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, assetCount)

	ciCount, err := seeder.client.ConfigurationItem.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, ciCount)

	processInstanceCount, err := seeder.client.ProcessInstance.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, processInstanceCount)

	workflowInstanceCount, err := seeder.client.WorkflowInstance.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, workflowInstanceCount)

	notificationCount, err := seeder.client.Notification.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, notificationCount)

	auditCount, err := seeder.client.AuditLog.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, auditCount)

	toolInvocationCount, err := seeder.client.ToolInvocation.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, toolInvocationCount)

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

func TestSeedAllDoesNotCreateTestTenantOrFixedPasswordAccounts(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)

	seeder.SeedAll(ctx)

	exists, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("tenant_test")).Exist(ctx)
	require.NoError(t, err)
	assert.False(t, exists)

	for _, username := range []string{"user1", "security1", "engineer1", "manager1", "tenant1admin"} {
		exists, err := seeder.client.User.Query().Where(user.UsernameEQ(username)).Exist(ctx)
		require.NoError(t, err)
		assert.False(t, exists, "fixed-password account %s must not be seeded", username)
	}
}

func TestSeedAdminPreservesExistingCredentials(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	rootTenant := seeder.seedDefaultTenant(ctx)
	require.NotNil(t, rootTenant)

	originalHash, err := bcrypt.GenerateFromPassword([]byte("operator-rotated-password"), bcrypt.DefaultCost)
	require.NoError(t, err)
	created, err := seeder.client.User.Create().
		SetUsername("admin").
		SetEmail("admin@example.com").
		SetName("Existing Admin").
		SetRole("admin").
		SetPasswordHash(string(originalHash)).
		SetActive(true).
		SetTenantID(rootTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	t.Setenv("ADMIN_PASSWORD", "different-bootstrap-password")
	seeder.seedAdmin(ctx)

	after, err := seeder.client.User.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, string(originalHash), after.PasswordHash)
	assert.Equal(t, user.RoleAdmin, after.Role)
}
