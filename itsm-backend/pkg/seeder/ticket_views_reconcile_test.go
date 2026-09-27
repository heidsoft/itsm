package seeder

import (
	"testing"

	"itsm-backend/ent/tenant"
	"itsm-backend/ent/ticketview"
	"itsm-backend/pkg/tenantmode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestSeedTicketViewsReconcilesPartialSet(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	target, err := seeder.client.Tenant.Create().SetCode("tv-partial").SetName("TV Partial").
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)

	hashed, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.MinCost)
	require.NoError(t, err)
	admin, err := seeder.client.User.Create().
		SetUsername("admin").SetEmail("admin@tv-partial.local").
		SetPasswordHash(string(hashed)).SetName("Admin").SetRole("super_admin").
		SetDepartment("IT").SetActive(true).SetTenantID(target.ID).
		Save(ctx)
	require.NoError(t, err)

	view := seeder.withBaselineTenant(target.ID)

	require.NotEmpty(t, seeder.config.TicketViews)
	first := seeder.config.TicketViews[0]
	_, err = seeder.client.TicketView.Create().
		SetName(first.Name).SetDescription("customer owned").
		SetFilters(map[string]interface{}{}).SetColumns(first.Columns).
		SetIsShared(first.IsShared).SetCreatedBy(admin.ID).SetTenantID(target.ID).Save(ctx)
	require.NoError(t, err)

	require.NoError(t, view.seedTicketViews(ctx))
	require.NoError(t, view.seedTicketViews(ctx), "重跑必须幂等")

	all, err := seeder.client.TicketView.Query().Where(ticketview.TenantIDEQ(target.ID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, all, len(seeder.config.TicketViews), "部分集合必须被补齐")
	for _, item := range all {
		if item.Name == first.Name {
			assert.Equal(t, "customer owned", item.Description, "已有条目不得被覆盖")
		}
	}
}
