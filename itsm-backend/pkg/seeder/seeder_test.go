package seeder

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"itsm-backend/config"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/menu"
	"itsm-backend/ent/permission"
	"itsm-backend/ent/role"
	"itsm-backend/ent/rolepermission"
	_ "itsm-backend/ent/runtime"
	"itsm-backend/ent/servicecatalog"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/standardchange"
	"itsm-backend/ent/systemconfig"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	"itsm-backend/internal/initialization"
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

func TestSeedProductionValidatesAndIsIdempotent(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)

	require.NoError(t, seeder.SeedProduction(ctx))
	first := productionSeedCounts{
		tenants:         mustCount(t, func() (int, error) { return seeder.client.Tenant.Query().Count(ctx) }),
		users:           mustCount(t, func() (int, error) { return seeder.client.User.Query().Count(ctx) }),
		roles:           mustCount(t, func() (int, error) { return seeder.client.Role.Query().Count(ctx) }),
		permissions:     mustCount(t, func() (int, error) { return seeder.client.Permission.Query().Count(ctx) }),
		rolePermissions: mustCount(t, func() (int, error) { return seeder.client.RolePermission.Query().Count(ctx) }),
		menus:           mustCount(t, func() (int, error) { return seeder.client.Menu.Query().Count(ctx) }),
	}

	require.NoError(t, seeder.SeedProduction(ctx))
	second := productionSeedCounts{
		tenants:         mustCount(t, func() (int, error) { return seeder.client.Tenant.Query().Count(ctx) }),
		users:           mustCount(t, func() (int, error) { return seeder.client.User.Query().Count(ctx) }),
		roles:           mustCount(t, func() (int, error) { return seeder.client.Role.Query().Count(ctx) }),
		permissions:     mustCount(t, func() (int, error) { return seeder.client.Permission.Query().Count(ctx) }),
		rolePermissions: mustCount(t, func() (int, error) { return seeder.client.RolePermission.Query().Count(ctx) }),
		menus:           mustCount(t, func() (int, error) { return seeder.client.Menu.Query().Count(ctx) }),
	}

	assert.Equal(t, first, second)
}

func TestProductionInitializersExposeCompleteComponentDAG(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	components, err := ProductionInitializers(seeder)
	require.NoError(t, err)
	require.Len(t, components, len(ProductionComponentNames))

	actualNames := make([]string, 0, len(components))
	for _, component := range components {
		actualNames = append(actualNames, component.Name())
		plan, err := component.Plan(ctx, initialization.Scope{Type: "platform", ID: 0})
		require.NoError(t, err)
		assert.Equal(t, CurrentTenantTemplateVersion, plan.TargetVersion)
		assert.NotEmpty(t, plan.SourceChecksum)
	}
	assert.Equal(t, ProductionComponentNames, actualNames)
}

func TestProductionInitializersApplyAndVerifyCompleteDAG(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	components, err := ProductionInitializers(seeder)
	require.NoError(t, err)
	scope := initialization.Scope{Type: "platform", ID: 0}

	for index, component := range components {
		plan, err := component.Plan(ctx, scope)
		require.NoError(t, err)
		_, err = component.Apply(ctx, scope, plan, int64(index+1))
		require.NoError(t, err, component.Name())
		require.NoError(t, component.Verify(ctx, scope, plan), component.Name())
	}
}

func TestProductionInitializersRepairMissingServiceCatalogWithoutOverwritingTenantCustomization(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	components, err := ProductionInitializers(seeder)
	require.NoError(t, err)
	scope := initialization.Scope{Type: "platform", ID: 0}

	for index, component := range components {
		plan, planErr := component.Plan(ctx, scope)
		require.NoError(t, planErr)
		_, applyErr := component.Apply(ctx, scope, plan, int64(index+1))
		require.NoError(t, applyErr, component.Name())
	}

	rootTenant, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)
	customized := seeder.config.ServiceCatalog[0]
	managed, err := seeder.client.ServiceCatalog.Query().Where(
		servicecatalog.TenantIDEQ(rootTenant.ID),
		servicecatalog.NameEQ(customized.Name),
	).Only(ctx)
	require.NoError(t, err)
	require.NoError(t, managed.Update().SetDescription("tenant customized description").Exec(ctx))

	missing := seeder.config.ServiceCatalog[1]
	_, err = seeder.client.ServiceCatalog.Delete().Where(
		servicecatalog.TenantIDEQ(rootTenant.ID),
		servicecatalog.NameEQ(missing.Name),
	).Exec(ctx)
	require.NoError(t, err)

	extension := components[len(components)-1]
	plan, err := extension.Plan(ctx, scope)
	require.NoError(t, err)
	_, err = extension.Apply(ctx, scope, plan, 100)
	require.NoError(t, err)
	require.NoError(t, extension.Verify(ctx, scope, plan))

	repaired, err := seeder.client.ServiceCatalog.Query().Where(
		servicecatalog.TenantIDEQ(rootTenant.ID),
		servicecatalog.NameEQ(missing.Name),
	).Exist(ctx)
	require.NoError(t, err)
	assert.True(t, repaired)

	preserved, err := seeder.client.ServiceCatalog.Query().Where(
		servicecatalog.TenantIDEQ(rootTenant.ID),
		servicecatalog.NameEQ(customized.Name),
	).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, "tenant customized description", preserved.Description)
}

func TestProductionComponentRollsBackWhenTransactionalVerificationFails(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	components, err := ProductionInitializers(seeder)
	require.NoError(t, err)
	identity := components[0].(*productionComponentInitializer)
	identity.verify = func(context.Context, *Seeder) error {
		return errors.New("injected verification failure")
	}

	plan, err := identity.Plan(ctx, initialization.Scope{Type: "platform", ID: 0})
	require.NoError(t, err)
	_, err = identity.Apply(
		ctx,
		initialization.Scope{Type: "platform", ID: 0},
		plan,
		1,
	)
	require.ErrorContains(t, err, "injected verification failure")

	tenantCount, err := seeder.client.Tenant.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, tenantCount)
	userCount, err := seeder.client.User.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, userCount)
}

func TestSeedProductionRepairsPartialRBACAndMenuInitialization(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	require.NoError(t, seeder.SeedProduction(ctx))
	rootTenant, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)

	_, err = seeder.client.Menu.Delete().
		Where(menu.PathEQ(seeder.expectedMenus[0]), menu.TenantIDEQ(rootTenant.ID)).
		Exec(ctx)
	require.NoError(t, err)
	deletedPermission, err := seeder.client.Permission.Query().
		Where(permission.CodeEQ(seeder.expectedPermissions[0]), permission.TenantIDEQ(rootTenant.ID)).
		Only(ctx)
	require.NoError(t, err)
	_, err = seeder.client.RolePermission.Delete().
		Where(rolepermission.PermissionIDEQ(deletedPermission.ID)).
		Exec(ctx)
	require.NoError(t, err)
	err = seeder.client.Permission.DeleteOne(deletedPermission).Exec(ctx)
	require.NoError(t, err)
	_, err = seeder.client.Role.Delete().
		Where(role.CodeEQ(seeder.config.Roles[0].Code), role.TenantIDEQ(rootTenant.ID)).
		Exec(ctx)
	require.NoError(t, err)

	require.NoError(t, seeder.SeedProduction(ctx))

	menuExists, err := seeder.client.Menu.Query().
		Where(menu.PathEQ(seeder.expectedMenus[0]), menu.TenantIDEQ(rootTenant.ID)).
		Exist(ctx)
	require.NoError(t, err)
	assert.True(t, menuExists)
	permissionExists, err := seeder.client.Permission.Query().
		Where(permission.CodeEQ(seeder.expectedPermissions[0]), permission.TenantIDEQ(rootTenant.ID)).
		Exist(ctx)
	require.NoError(t, err)
	assert.True(t, permissionExists)
	roleExists, err := seeder.client.Role.Query().
		Where(role.CodeEQ(seeder.config.Roles[0].Code), role.TenantIDEQ(rootTenant.ID)).
		Exist(ctx)
	require.NoError(t, err)
	assert.True(t, roleExists)
}

func TestSeedProductionPreservesTenantOwnedGrantOnManagedRole(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	require.NoError(t, seeder.SeedProduction(ctx))
	rootTenant, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)
	adminRole, err := seeder.client.Role.Query().
		Where(role.CodeEQ("sysadmin"), role.TenantIDEQ(rootTenant.ID)).
		Only(ctx)
	require.NoError(t, err)
	customPermission, err := seeder.client.Permission.Create().
		SetCode("customer:custom").
		SetName("Customer Custom").
		SetResource("customer").
		SetAction("custom").
		SetTenantID(rootTenant.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = seeder.client.RolePermission.Create().
		SetRoleID(adminRole.ID).
		SetPermissionID(customPermission.ID).
		SetTenantID(rootTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, seeder.SeedProduction(ctx))

	exists, err := seeder.client.RolePermission.Query().
		Where(
			rolepermission.RoleIDEQ(adminRole.ID),
			rolepermission.PermissionIDEQ(customPermission.ID),
			rolepermission.TenantIDEQ(rootTenant.ID),
		).
		Exist(ctx)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestProvisionTenantReadinessAcrossDeploymentModes(t *testing.T) {
	for _, mode := range []string{
		tenantmode.DeploymentModePrivate,
		tenantmode.DeploymentModeSaaS,
		tenantmode.DeploymentModeSaaSMSP,
	} {
		t.Run(mode, func(t *testing.T) {
			seeder, ctx := newTestSeeder(t, mode)
			require.NoError(t, seeder.SeedProduction(ctx))
			target, err := seeder.client.Tenant.Create().
				SetName("Production Customer").
				SetCode("production-customer").
				SetType(tenant.TypeSaasCustomer).
				Save(ctx)
			require.NoError(t, err)

			require.NoError(t, seeder.ProvisionTenant(ctx, target.ID, CurrentTenantTemplateVersion))
			firstRoles, err := seeder.client.Role.Query().Where(role.TenantIDEQ(target.ID)).Count(ctx)
			require.NoError(t, err)
			require.Positive(t, firstRoles)
			require.NoError(t, seeder.validateTenantReadiness(ctx, target.ID))

			_, err = seeder.client.Role.Create().
				SetName("Customer Custom Role").SetCode("customer_custom").
				SetDescription("tenant-owned").SetTenantID(target.ID).Save(ctx)
			require.NoError(t, err)
			_, err = seeder.client.Menu.Create().
				SetName("Customer Custom Menu").SetPath("/customer/custom").
				SetTenantID(target.ID).Save(ctx)
			require.NoError(t, err)

			require.NoError(t, seeder.ProvisionTenant(ctx, target.ID, CurrentTenantTemplateVersion))
			secondRoles, err := seeder.client.Role.Query().Where(role.TenantIDEQ(target.ID)).Count(ctx)
			require.NoError(t, err)
			assert.Equal(t, firstRoles+1, secondRoles)

			foreignPermission, err := seeder.client.Permission.Query().
				Where(permission.TenantIDEQ(target.ID)).
				First(ctx)
			require.NoError(t, err)
			sourceHasForeignID, err := seeder.client.Permission.Query().
				Where(permission.IDEQ(foreignPermission.ID), permission.TenantIDNEQ(target.ID)).
				Exist(ctx)
			require.NoError(t, err)
			assert.False(t, sourceHasForeignID)
		})
	}
}

func TestProvisionTenantRollsBackWhenSourceTemplateIsIncomplete(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModeSaaS)
	require.NoError(t, seeder.SeedProduction(ctx))
	root, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)
	_, err = seeder.client.CIType.Delete().Where(citype.TenantIDEQ(root.ID)).Exec(ctx)
	require.NoError(t, err)
	target, err := seeder.client.Tenant.Create().
		SetName("Incomplete Target").SetCode("incomplete-target").
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)

	err = seeder.ProvisionTenant(ctx, target.ID, CurrentTenantTemplateVersion)
	require.ErrorContains(t, err, "validate tenant template before commit")

	roleCount, err := seeder.client.Role.Query().Where(role.TenantIDEQ(target.ID)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, roleCount)
	versionExists, err := seeder.client.SystemConfig.Query().
		Where(systemconfig.KeyEQ("tenant.bootstrap.version." + strconv.Itoa(target.ID))).
		Exist(ctx)
	require.NoError(t, err)
	assert.False(t, versionExists)
}

type productionSeedCounts struct {
	tenants         int
	users           int
	roles           int
	permissions     int
	rolePermissions int
	menus           int
}

func mustCount(t *testing.T, query func() (int, error)) int {
	t.Helper()
	count, err := query()
	require.NoError(t, err)
	return count
}

// rolePermissionCodes loads the distinct permission codes bound to a role in the default tenant.
func rolePermissionCodes(t *testing.T, seeder *Seeder, ctx context.Context, tenantID int, roleCode string) map[string]struct{} {
	t.Helper()
	r, err := seeder.client.Role.Query().
		Where(role.CodeEQ(roleCode), role.TenantIDEQ(tenantID)).
		Only(ctx)
	require.NoError(t, err)
	bindings, err := seeder.client.RolePermission.Query().
		Where(rolepermission.RoleIDEQ(r.ID), rolepermission.TenantIDEQ(tenantID)).
		All(ctx)
	require.NoError(t, err)
	codes := make(map[string]struct{}, len(bindings))
	for _, b := range bindings {
		p, err := seeder.client.Permission.Get(ctx, b.PermissionID)
		require.NoError(t, err)
		codes[p.Code] = struct{}{}
	}
	return codes
}

// TestSeedRolePermissionsCoverTicketLifecycle is a regression guard: the engineer and
// end-user roles must be seeded with the fine-grained ticket permissions that the
// production Gin routes require (create/update/assign/escalate). Without these, every
// non-admin user receives 403 on core ticket workflows.
func TestSeedRolePermissionsCoverTicketLifecycle(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	require.NoError(t, seeder.SeedProduction(ctx))

	rootTenant, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)

	endUser := rolePermissionCodes(t, seeder, ctx, rootTenant.ID, "end_user")
	require.Contains(t, endUser, "ticket:create")
	require.Contains(t, endUser, "ticket:update")

	for _, roleCode := range []string{"l1_support", "l2_support", "ops_manager"} {
		codes := rolePermissionCodes(t, seeder, ctx, rootTenant.ID, roleCode)
		for _, want := range []string{"ticket:create", "ticket:update", "ticket:assign", "ticket:escalate"} {
			assert.Contains(t, codes, want, roleCode+" missing "+want)
		}
	}
}
