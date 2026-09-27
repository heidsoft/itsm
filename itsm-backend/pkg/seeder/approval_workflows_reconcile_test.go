package seeder

import (
	"testing"

	"itsm-backend/ent/approvalworkflow"
	"itsm-backend/ent/tenant"
	"itsm-backend/pkg/tenantmode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedApprovalWorkflowsReconcilesPartialSet(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	target, err := seeder.client.Tenant.Create().SetCode("aw-partial").SetName("AW Partial").
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)
	view := seeder.withBaselineTenant(target.ID)

	require.NotEmpty(t, seeder.config.ApprovalWorkflows)
	first := seeder.config.ApprovalWorkflows[0]
	_, err = seeder.client.ApprovalWorkflow.Create().
		SetName(first.Name).SetDescription("customer owned").
		SetTicketType(first.TicketType).SetPriority(first.Priority).
		SetNodes(first.Nodes).SetIsActive(true).SetTenantID(target.ID).Save(ctx)
	require.NoError(t, err)

	require.NoError(t, view.seedApprovalWorkflows(ctx))
	require.NoError(t, view.seedApprovalWorkflows(ctx), "重跑必须幂等")

	all, err := seeder.client.ApprovalWorkflow.Query().Where(approvalworkflow.TenantIDEQ(target.ID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, all, len(seeder.config.ApprovalWorkflows), "部分集合必须被补齐")
	for _, item := range all {
		if item.Name == first.Name {
			assert.Equal(t, "customer owned", item.Description, "已有条目不得被覆盖")
		}
	}
}
