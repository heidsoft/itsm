package seeder

import (
	"testing"

	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/tenant"
	"itsm-backend/pkg/tenantmode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 存量库的 SLA 定义常常只有部分集合（清单命名演进后新旧两代混杂）：
// "任一行存在即跳过"会让 verify 要求的新增条目永远补不上，前滚被永久卡死。
func TestSeedSLADefinitionsReconcilesPartialSet(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	target, err := seeder.client.Tenant.Create().SetCode("sla-partial").SetName("SLA Partial").
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)
	view := seeder.withBaselineTenant(target.ID)

	require.NotEmpty(t, seeder.config.SLADefinitions)
	first := seeder.config.SLADefinitions[0]
	_, err = seeder.client.SLADefinition.Create().
		SetName(first.Name).SetDescription("customer owned").SetServiceType(first.ServiceType).
		SetPriority(first.Priority).SetResponseTime(first.ResponseTime).
		SetResolutionTime(first.ResolutionTime).SetIsActive(true).SetTenantID(target.ID).Save(ctx)
	require.NoError(t, err)

	require.NoError(t, view.seedSLADefinitions(ctx))
	require.NoError(t, view.seedSLADefinitions(ctx), "重跑必须幂等")

	all, err := seeder.client.SLADefinition.Query().Where(sladefinition.TenantIDEQ(target.ID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, all, len(seeder.config.SLADefinitions), "部分集合必须被补齐")
	for _, item := range all {
		if item.Name == first.Name {
			assert.Equal(t, "customer owned", item.Description, "已有条目不得被覆盖")
		}
	}
}
