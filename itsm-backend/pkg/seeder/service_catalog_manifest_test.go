package seeder

import (
	"testing"

	"itsm-backend/ent/servicecatalog"
	"itsm-backend/ent/servicecatalogitem"
	"itsm-backend/ent/tenant"
	"itsm-backend/internal/initialization"
	"itsm-backend/pkg/tenantmode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

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
