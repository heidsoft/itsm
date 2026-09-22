package seeder

import (
	"testing"

	"itsm-backend/ent/slaalertrule"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/tenant"
	"itsm-backend/pkg/tenantmode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The old implementation hardcoded SLA definition names that only exist in
// the embedded config, so a JSON SLA section silently produced zero rules
// while the log claimed eight.
func TestSeedSLAAlertRulesFollowsConfiguredDefinitions(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	root, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	require.NoError(t, err)

	require.Error(t, seeder.seedSLAAlertRules(ctx), "must fail instead of skipping when no definitions exist")

	seeder.config.SLADefinitions = []SLADefinitionSeed{
		{Name: "Incident-P0", Description: "d", ServiceType: "incident", Priority: "urgent", ResponseTime: 15, ResolutionTime: 120},
		{Name: "Change-Standard", Description: "d", ServiceType: "change", Priority: "medium", ResponseTime: 60, ResolutionTime: 480},
	}
	seeder.seedSLADefinitions(ctx)

	require.NoError(t, seeder.seedSLAAlertRules(ctx))
	require.NoError(t, seeder.seedSLAAlertRules(ctx), "repeat run must be idempotent per definition")

	rules, err := seeder.client.SLAAlertRule.Query().Where(slaalertrule.TenantIDEQ(root.ID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, rules, 4)

	definitions, err := seeder.client.SLADefinition.Query().Where(sladefinition.TenantIDEQ(root.ID)).All(ctx)
	require.NoError(t, err)
	byID := make(map[int]string, len(definitions))
	for _, d := range definitions {
		byID[d.ID] = d.Name
	}
	for _, rule := range rules {
		assert.Contains(t, rule.Name, byID[rule.SLADefinitionID], "rule must reference its own definition")
		if byID[rule.SLADefinitionID] == "Incident-P0" {
			assert.Equal(t, []string{"email", "sms"}, rule.NotificationChannels)
		} else {
			assert.Equal(t, []string{"email"}, rule.NotificationChannels)
		}
	}
}
