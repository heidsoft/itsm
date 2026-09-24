package seeder

import (
	"testing"
	"time"

	"itsm-backend/ent/team"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/tickettype"
	"itsm-backend/pkg/tenantmode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 已安装环境里 teams / 工单类型可能只有部分集合：旧实现"任一行存在即跳过"
// 会让缺口永远补不上。必须按条目补齐，且不覆盖客户改名。
func TestSeedTeamsReconcilesPartialSet(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	target, err := seeder.client.Tenant.Create().SetCode("team-partial").SetName("Team Partial").
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)
	view := seeder.withBaselineTenant(target.ID)

	require.NotEmpty(t, seeder.config.Teams)
	first := seeder.config.Teams[0]
	firstCode := teamCode(first)
	_, err = seeder.client.Team.Create().
		SetName("客户改名团队").SetCode(firstCode).SetDescription("customer owned").
		SetStatus("active").SetTenantID(target.ID).Save(ctx)
	require.NoError(t, err)

	require.NoError(t, view.seedTeams(ctx))
	require.NoError(t, view.seedTeams(ctx), "重跑必须幂等")

	all, err := seeder.client.Team.Query().Where(team.TenantIDEQ(target.ID), team.DeletedAtIsNil()).All(ctx)
	require.NoError(t, err)
	require.Len(t, all, len(seeder.config.Teams), "部分集合必须被补齐")
	for _, item := range all {
		if item.Code == firstCode {
			assert.Equal(t, "客户改名团队", item.Name, "客户改名必须保留")
		}
	}
}

func TestSeedTicketTypesReconcilesPartialSet(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	target, err := seeder.client.Tenant.Create().SetCode("types-partial").SetName("Types Partial").
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)
	view := seeder.withBaselineTenant(target.ID)
	creator, err := view.ensureTenantSystemAccount(ctx, target.ID)
	require.NoError(t, err)

	definitions := ticketTypeDefinitions()
	require.NotEmpty(t, definitions)
	_, err = seeder.client.TicketType.Create().
		SetCode(definitions[0].Code).SetName("客户改名类型").SetDescription("customer owned").
		SetIcon(definitions[0].Icon).SetColor(definitions[0].Color).SetStatus("active").
		SetCustomFields(map[string]interface{}{}).SetApprovalChain([]interface{}{}).
		SetAssignmentRules([]interface{}{}).SetNotificationConfig(map[string]interface{}{}).
		SetPermissionConfig(map[string]interface{}{}).
		SetCreatedAt(time.Now()).SetUpdatedAt(time.Now()).
		SetCreatedBy(int64(creator.ID)).SetTenantID(int64(target.ID)).Save(ctx)
	require.NoError(t, err)

	require.NoError(t, view.seedTicketTypes(ctx))
	require.NoError(t, view.seedTicketTypes(ctx), "重跑必须幂等")

	all, err := seeder.client.TicketType.Query().Where(tickettype.TenantIDEQ(int64(target.ID))).All(ctx)
	require.NoError(t, err)
	require.Len(t, all, len(definitions), "部分集合必须被补齐")
	for _, item := range all {
		if item.Code == definitions[0].Code {
			assert.Equal(t, "客户改名类型", item.Name, "客户改名必须保留")
		}
	}
}
