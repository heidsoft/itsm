package seeder

import (
	"context"
	"os"
	"testing"
	"time"

	"itsm-backend/common/tenantctx"
	"itsm-backend/ent"
	"itsm-backend/ent/servicecatalog"
	"itsm-backend/ent/servicecatalogitem"
	"itsm-backend/ent/tenant"
	"itsm-backend/internal/initialization"
	"itsm-backend/pkg/tenantmode"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// This optional PostgreSQL regression uses the installed schema but rolls back
// every fixture and seed write. It never reconciles an existing tenant.
func TestEnglishCatalogSeedsPreserveCustomizedLegacyIdentityPostgres(t *testing.T) {
	dsn := os.Getenv("ITSM_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set ITSM_TEST_POSTGRES_DSN to run the transactional PostgreSQL regression")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ctx = tenantctx.SystemContext(ctx, "initialization:test", "transactional legacy seed identity regression")
	client, err := ent.Open("postgres", dsn)
	require.NoError(t, err)
	defer client.Close()
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer tx.Rollback()
	isolated := tx.Client()
	createTenant := func(suffix string) int {
		tenantRow, err := isolated.Tenant.Create().SetName("Legacy catalog regression " + suffix).
			SetCode("english-seed-test-" + uuid.NewString()).Save(ctx)
		require.NoError(t, err)
		return tenantRow.ID
	}
	tenantA, tenantB := createTenant("A"), createTenant("B")
	createLegacy := func(tenantID int, description string) *ent.ServiceCatalog {
		row, err := isolated.ServiceCatalog.Create().SetTenantID(tenantID).
			SetName("云服务器 ECS").SetDescription(description).SetCategory("custom-category").
			SetServiceType("vm").SetRequiresApproval(false).SetDeliveryTime(7).
			SetStatus("disabled").SetIsActive(false).Save(ctx)
		require.NoError(t, err)
		return row
	}
	originalA := createLegacy(tenantA, "Tenant A custom request policy")
	originalB := createLegacy(tenantB, "Tenant B separate request policy")
	item, err := isolated.ServiceCatalogItem.Create().SetTenantID(tenantA).SetCatalogID(originalA.ID).
		SetName("ECS 实例申请").SetDescription("Tenant A custom fulfillment instructions").
		SetBusinessSubType("service_request").SetProcessDefinitionKey("custom-request-flow").
		SetRequiresApproval(false).SetEstimatedDays(9).SetIsActive(false).Save(ctx)
	require.NoError(t, err)
	s := NewSeeder(isolated, zap.NewNop().Sugar(), nil).withBaselineTenant(tenantA)
	s.config = &SeedConfig{
		ServiceCatalog: []ServiceCatalogSeed{{Name: "ECS Cloud Server", Description: "Elastic cloud server", Category: "云计算", ServiceType: "vm", RequiresApproval: true, DeliveryTime: 1}},
		ServiceCatalogItems: []ServiceCatalogItemSeed{
			{CatalogName: "ECS Cloud Server", Name: "ECS Instance Request", Description: "Request a new elastic cloud server instance", BusinessSubType: "service_request", ProcessDefinitionKey: "service_request_flow", RequiresApproval: true, EstimatedDays: 1},
			{CatalogName: "ECS Cloud Server", Name: "ECS Configuration Change", BusinessSubType: "change", ProcessDefinitionKey: "change_normal_flow", RequiresApproval: true, EstimatedDays: 1},
		},
	}
	for attempt := 0; attempt < 2; attempt++ {
		require.NoError(t, s.seedServiceCatalog(ctx))
		require.NoError(t, s.seedServiceCatalogItems(ctx))
		require.NoError(t, s.verifyExtensionTemplates(ctx))
		catalogs, err := isolated.ServiceCatalog.Query().Where(servicecatalog.TenantIDEQ(tenantA)).All(ctx)
		require.NoError(t, err)
		require.Len(t, catalogs, 1, "English seeding must not recreate a customized legacy catalog")
		assert.Equal(t, originalA.ID, catalogs[0].ID)
		assert.Equal(t, originalA.Name, catalogs[0].Name)
		assert.Equal(t, originalA.Description, catalogs[0].Description)
		assert.Equal(t, originalA.Category, catalogs[0].Category)
		assert.Equal(t, originalA.Status, catalogs[0].Status)
		assert.Equal(t, originalA.RequiresApproval, catalogs[0].RequiresApproval)
		assert.Equal(t, originalA.DeliveryTime, catalogs[0].DeliveryTime)
		assert.False(t, catalogs[0].IsActive)
		items, err := isolated.ServiceCatalogItem.Query().Where(servicecatalogitem.TenantIDEQ(tenantA)).All(ctx)
		require.NoError(t, err)
		require.Len(t, items, 2)
		for _, row := range items {
			assert.Equal(t, originalA.ID, row.CatalogID)
			if row.ID == item.ID {
				assert.Equal(t, item.Name, row.Name)
				assert.Equal(t, item.Description, row.Description)
				assert.Equal(t, item.ProcessDefinitionKey, row.ProcessDefinitionKey)
				assert.Equal(t, item.EstimatedDays, row.EstimatedDays)
				assert.False(t, row.IsActive)
			}
		}
	}
	unchangedB, err := isolated.ServiceCatalog.Get(ctx, originalB.ID)
	require.NoError(t, err)
	assert.Equal(t, originalB.Description, unchangedB.Description)
	countB, err := isolated.ServiceCatalogItem.Query().Where(servicecatalogitem.TenantIDEQ(tenantB)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, countB)

	t.Run("catalog name collision prefers exact English parent", func(t *testing.T) {
		english, err := isolated.ServiceCatalog.Create().SetTenantID(tenantA).
			SetName("ECS Cloud Server").SetDescription("Separate English catalog").Save(ctx)
		require.NoError(t, err)
		require.NoError(t, s.seedServiceCatalogItems(ctx))
		require.NoError(t, s.verifyExtensionTemplates(ctx))
		count, err := isolated.ServiceCatalogItem.Query().Where(servicecatalogitem.TenantIDEQ(tenantA), servicecatalogitem.CatalogIDEQ(english.ID)).Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
		_, err = isolated.ServiceCatalog.Create().SetTenantID(tenantA).SetName("ECS Cloud Server").Save(ctx)
		require.NoError(t, err)
		require.ErrorContains(t, s.verifyExtensionTemplates(ctx), "not singular", "duplicate exact names must still fail verification")
	})

	t.Run("legacy SLA policy customizations survive", func(t *testing.T) {
		legacy, err := isolated.SLAPolicy.Create().SetTenantID(tenantA).SetName("默认P1事件SLA").
			SetDescription("Custom policy").SetPriority("low").SetResponseTimeMinutes(777).
			SetResolutionTimeMinutes(999).SetExcludeWeekends(true).SetExcludeHolidays(true).
			SetIsActive(false).SetPriorityScore(42).Save(ctx)
		require.NoError(t, err)
		s.config.SLAPolicies = []SLAPolicySeed{{Name: "Default P1 Incident SLA", Description: "Managed default", Priority: "high", ResponseTimeMinutes: 15, ResolutionTimeMinutes: 120, IsActive: true, PriorityScore: 1}}
		s.seedSLAPolicies(ctx)
		actual, err := isolated.SLAPolicy.Get(ctx, legacy.ID)
		require.NoError(t, err)
		assert.Equal(t, legacy.Description, actual.Description)
		assert.Equal(t, legacy.Priority, actual.Priority)
		assert.Equal(t, legacy.ResponseTimeMinutes, actual.ResponseTimeMinutes)
		assert.Equal(t, legacy.ResolutionTimeMinutes, actual.ResolutionTimeMinutes)
		assert.Equal(t, legacy.ExcludeWeekends, actual.ExcludeWeekends)
		assert.Equal(t, legacy.ExcludeHolidays, actual.ExcludeHolidays)
		assert.Equal(t, legacy.IsActive, actual.IsActive)
		assert.Equal(t, legacy.PriorityScore, actual.PriorityScore)
		english, err := isolated.SLAPolicy.Create().SetTenantID(tenantA).SetName("Default P1 Incident SLA").
			SetDescription("Old English settings").SetResponseTimeMinutes(30).SetResolutionTimeMinutes(240).Save(ctx)
		require.NoError(t, err)
		s.seedSLAPolicies(ctx)
		exact, err := isolated.SLAPolicy.Get(ctx, english.ID)
		require.NoError(t, err)
		assert.Equal(t, "Managed default", exact.Description)
		assert.Equal(t, 15, exact.ResponseTimeMinutes)
	})

	t.Run("SLA definition collision prefers exact English", func(t *testing.T) {
		_, err := isolated.SLADefinition.Create().SetTenantID(tenantA).SetName("SLA-P0-紧急").Save(ctx)
		require.NoError(t, err)
		english, err := isolated.SLADefinition.Create().SetTenantID(tenantA).SetName("SLA-P0-Urgent").Save(ctx)
		require.NoError(t, err)
		_, err = isolated.SLAAlertRule.Create().SetTenantID(tenantA).SetSLADefinitionID(english.ID).SetName("English rule").Save(ctx)
		require.NoError(t, err)
		s.config.SLADefinitions = []SLADefinitionSeed{{Name: "SLA-P0-Urgent"}}
		require.NoError(t, s.verifySLATemplates(ctx))
	})
}

// The catalog manifest lives in two sections joined by name, so a section
// replacement can silently orphan every item. This must be caught statically.
func TestMergedCatalogManifestItemReferencesResolve(t *testing.T) {
	merged := loadSeedConfig(zap.NewNop().Sugar())
	require.NotEmpty(t, merged.ServiceCatalog)
	require.NotEmpty(t, merged.ServiceCatalogItems)

	names := make(map[string]struct{}, len(merged.ServiceCatalog))
	for _, catalog := range merged.ServiceCatalog {
		require.NotContains(t, names, catalog.Name, "duplicate catalog %s", catalog.Name)
		names[catalog.Name] = struct{}{}
	}
	for _, item := range merged.ServiceCatalogItems {
		_, ok := names[item.CatalogName]
		assert.True(t, ok, "item %q references unknown catalog %q", item.Name, item.CatalogName)
	}
}

func TestSeedServiceCatalogItemsRejectsDanglingParent(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	seeder.config.ServiceCatalog = []ServiceCatalogSeed{
		{Name: "账号与权限", Description: "d", Category: "c", ServiceType: "access"},
	}
	seeder.config.ServiceCatalogItems = []ServiceCatalogItemSeed{
		{CatalogName: "并不存在的目录", Name: "孤儿子项"},
	}

	err := seeder.seedServiceCatalogItems(ctx)
	require.ErrorContains(t, err, "references unknown catalog 并不存在的目录")
}

func TestProductionInitializerSeedsEveryCatalogItem(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	components, err := ProductionInitializers(seeder)
	require.NoError(t, err)
	scope := initialization.Scope{Type: "platform", ID: 0}

	for _, component := range components {
		plan, planErr := component.Plan(ctx, scope)
		require.NoError(t, planErr)
		_, applyErr := applyTestComponent(ctx, seeder, component, scope, plan)
		require.NoError(t, applyErr, component.Name())
	}

	root, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)
	catalogs, err := seeder.client.ServiceCatalog.Query().Where(servicecatalog.TenantIDEQ(root.ID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, catalogs, len(seeder.config.ServiceCatalog))

	parentIDs := make(map[string]int, len(catalogs))
	for _, catalog := range catalogs {
		parentIDs[catalog.Name] = catalog.ID
	}
	for _, item := range seeder.config.ServiceCatalogItems {
		exists, err := seeder.client.ServiceCatalogItem.Query().Where(
			servicecatalogitem.TenantIDEQ(root.ID),
			servicecatalogitem.CatalogIDEQ(parentIDs[item.CatalogName]),
			servicecatalogitem.NameEQ(item.Name),
		).Exist(ctx)
		require.NoError(t, err)
		assert.True(t, exists, "item %s must exist under %s", item.Name, item.CatalogName)
	}

	extension := findProductionComponent(t, components, "extension-core")
	plan, err := extension.Plan(ctx, scope)
	require.NoError(t, err)

	// Dropping a managed item leaves an empty storefront that only an
	// item-level check can detect.
	first := seeder.config.ServiceCatalogItems[0]
	deleted, err := seeder.client.ServiceCatalogItem.Delete().Where(
		servicecatalogitem.TenantIDEQ(root.ID),
		servicecatalogitem.CatalogIDEQ(parentIDs[first.CatalogName]),
		servicecatalogitem.NameEQ(first.Name),
	).Exec(ctx)
	require.NoError(t, err)
	require.Positive(t, deleted)
	require.ErrorContains(t, extension.Verify(ctx, scope, plan), "missing under "+first.CatalogName)
}
