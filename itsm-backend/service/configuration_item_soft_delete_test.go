package service

import (
	"context"
	"errors"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/configurationitemhistory"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/hook"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupCISoftDeleteFixture(t *testing.T) (*ent.Client, *ConfigurationItemService, context.Context, int, int, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", testDSN())
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	tenant, err := client.Tenant.Create().SetName("CMDB soft delete").SetCode("cmdb-soft-delete").SetDomain("cmdb-soft-delete.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	source, err := client.ConfigurationItem.Create().SetName("source").SetCiTypeID(ciType.ID).SetCiType("Server").SetStatus("active").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	target, err := client.ConfigurationItem.Create().SetName("target").SetCiTypeID(ciType.ID).SetCiType("Server").SetStatus("active").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	relation, err := client.CIRelationship.Create().SetTenantID(tenant.ID).SetRelationshipType("depends_on").SetSourceCiID(source.ID).SetTargetCiID(target.ID).Save(ctx)
	require.NoError(t, err)
	history := NewCIHistoryService(client, logger)
	require.NoError(t, history.RecordCIHistory(ctx, source.ID, tenant.ID, 0, "system", "create", "", nil, source))
	return client, NewConfigurationItemService(client, logger, history, NewCITagService(client, logger)), ctx, tenant.ID, source.ID, relation.ID
}

func TestConfigurationItemService_DeleteCIRetiresAndPreservesAudit(t *testing.T) {
	client, svc, ctx, tenantID, ciID, relationID := setupCISoftDeleteFixture(t)
	defer client.Close()

	require.NoError(t, svc.DeleteCI(ctx, ciID, tenantID))
	visible, err := svc.GetCIByID(ctx, ciID, tenantID, false)
	require.NoError(t, err)
	require.Nil(t, visible)
	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusScrapped, stored.LifecycleStatus)
	require.Equal(t, common.CIStatusRetired, stored.Status)
	require.False(t, stored.ExpireAt.IsZero())
	relation, err := client.CIRelationship.Query().Where(cirelationship.IDEQ(relationID)).Only(ctx)
	require.NoError(t, err)
	require.False(t, relation.IsActive)
	deleteHistory, err := client.ConfigurationItemHistory.Query().Where(
		configurationitemhistory.CiIDEQ(ciID), configurationitemhistory.TenantIDEQ(tenantID),
		configurationitemhistory.OperationEQ("delete"),
	).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusScrapped, deleteHistory.After["lifecycle_status"])
}

func TestConfigurationItemService_DeleteCIRollsBackWhenAuditFails(t *testing.T) {
	client, svc, ctx, tenantID, ciID, relationID := setupCISoftDeleteFixture(t)
	defer client.Close()
	client.ConfigurationItemHistory.Use(func(next ent.Mutator) ent.Mutator {
		return hook.ConfigurationItemHistoryFunc(func(context.Context, *ent.ConfigurationItemHistoryMutation) (ent.Value, error) {
			return nil, errors.New("injected history failure")
		})
	})

	err := svc.DeleteCI(ctx, ciID, tenantID)
	require.ErrorContains(t, err, "injected history failure")
	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	require.NotEqual(t, common.CILifecycleStatusScrapped, stored.LifecycleStatus)
	relation, err := client.CIRelationship.Get(ctx, relationID)
	require.NoError(t, err)
	require.True(t, relation.IsActive)
}

func TestConfigurationItemService_DeleteCIRejectsCrossTenant(t *testing.T) {
	client, svc, ctx, _, ciID, _ := setupCISoftDeleteFixture(t)
	defer client.Close()
	other, err := client.Tenant.Create().SetName("Other").SetCode("cmdb-soft-delete-other").SetDomain("other.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	require.Error(t, svc.DeleteCI(ctx, ciID, other.ID))
	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	require.NotEqual(t, common.CILifecycleStatusScrapped, stored.LifecycleStatus)
}
