package seeder

import (
	"testing"

	"itsm-backend/ent/menu"
	"itsm-backend/ent/tenant"
	"itsm-backend/internal/initialization"
	"itsm-backend/pkg/tenantmode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Re-running initialization must repair menu structure without undoing the
// operator's decision to hide or disable a managed menu.
func TestSeedMenusPreservesOperatorVisibilityAcrossReconcile(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	components, err := ProductionInitializers(seeder)
	require.NoError(t, err)
	scope := initialization.Scope{Type: "platform", ID: 0}
	identity := components[0]

	plan, err := identity.Plan(ctx, scope)
	require.NoError(t, err)
	_, err = applyTestComponent(ctx, seeder, identity, scope, plan)
	require.NoError(t, err)

	root, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)
	_, err = seeder.client.Menu.Update().Where(
		menu.TenantIDEQ(root.ID), menu.PathEQ("/dashboard"),
	).SetIsVisible(false).SetIsEnabled(false).Save(ctx)
	require.NoError(t, err)

	_, err = applyTestComponent(ctx, seeder, identity, scope, plan)
	require.NoError(t, err, "second reconcile must succeed")

	hidden, err := seeder.client.Menu.Query().Where(
		menu.TenantIDEQ(root.ID), menu.PathEQ("/dashboard"),
	).Only(ctx)
	require.NoError(t, err)
	assert.False(t, hidden.IsVisible, "operator-hidden menu must stay hidden")
	assert.False(t, hidden.IsEnabled, "operator-disabled menu must stay disabled")

	// Structural fields remain manifest-owned.
	manifest := menuDefinitions()
	var expectedSort int
	var expectedIcon string
	for _, spec := range manifest {
		if spec.Path == "/dashboard" {
			expectedSort, expectedIcon = spec.SortOrder, spec.Icon
		}
	}
	assert.Equal(t, expectedSort, hidden.SortOrder)
	assert.Equal(t, expectedIcon, hidden.Icon)

	require.NoError(t, identity.Verify(ctx, scope, plan),
		"verification must not treat operator visibility choices as missing menus")
}
