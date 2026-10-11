package service

import (
	"context"
	"strings"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/configurationitemhistory"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupCILifecycleFixture(t *testing.T) (*ent.Client, *ConfigurationItemService, context.Context, int, int, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", testDSN())
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	tenant, err := client.Tenant.Create().SetName("CMDB lifecycle").SetCode("cmdb-lifecycle").SetDomain("cmdb-lifecycle.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	// schema 默认 lifecycle_status=online、version=1
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

func TestConfigurationItemService_LifecycleDraftToOnline(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	_, err := client.ConfigurationItem.UpdateOneID(ciID).SetLifecycleStatus(common.CILifecycleStatusDraft).Save(ctx)
	require.NoError(t, err)

	ci, err := svc.UpdateLifecycleStatus(ctx, ciID, tenantID, LifecycleChangeParams{
		Status:          common.CILifecycleStatusOnline,
		Remark:          "publish",
		ExpectedVersion: 1,
	}, 1, "tester")
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusOnline, ci.LifecycleStatus)

	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusOnline, stored.LifecycleStatus)
	require.Equal(t, 2, stored.Version)

	hist, err := client.ConfigurationItemHistory.Query().Where(
		configurationitemhistory.CiIDEQ(ciID),
		configurationitemhistory.TenantIDEQ(tenantID),
		configurationitemhistory.OperationEQ("lifecycle_update"),
	).First(ctx)
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusDraft, hist.Before["lifecycle_status"])
	require.Equal(t, common.CILifecycleStatusOnline, hist.After["lifecycle_status"])
}

func TestConfigurationItemService_LifecycleSameStatusIdempotent(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	ci, err := svc.UpdateLifecycleStatus(ctx, ciID, tenantID, LifecycleChangeParams{
		Status: common.CILifecycleStatusOnline,
	}, 1, "tester")
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusOnline, ci.LifecycleStatus)

	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	require.Equal(t, 1, stored.Version, "idempotent no-op must not bump version")
}

func TestConfigurationItemService_LifecycleRejectsIllegalTransition(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	_, err := svc.UpdateLifecycleStatus(ctx, ciID, tenantID, LifecycleChangeParams{
		Status: common.CILifecycleStatusDraft,
	}, 1, "tester")
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr)
	require.Equal(t, common.ConflictCode, bizErr.Code)

	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusOnline, stored.LifecycleStatus, "rejected transition must not mutate")
}

func TestConfigurationItemService_LifecycleVersionConflict(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	_, err := svc.UpdateLifecycleStatus(ctx, ciID, tenantID, LifecycleChangeParams{
		Status:          common.CILifecycleStatusMaintenance,
		ExpectedVersion: 99,
	}, 1, "tester")
	require.Error(t, err)
	var conflictErr *common.VersionConflictError
	require.ErrorAs(t, err, &conflictErr)
	require.Equal(t, ciID, conflictErr.ResourceID)
	require.Equal(t, 99, conflictErr.CurrentVersion)
	require.Equal(t, 1, conflictErr.ServerVersion)

	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusOnline, stored.LifecycleStatus)
	require.Equal(t, 1, stored.Version)
}

func TestConfigurationItemService_LifecycleNotFound(t *testing.T) {
	client, svc, ctx, tenantID, _, _ := setupCILifecycleFixture(t)
	defer client.Close()

	_, err := svc.UpdateLifecycleStatus(ctx, 999999, tenantID, LifecycleChangeParams{
		Status:          common.CILifecycleStatusOnline,
		ExpectedVersion: 1,
	}, 1, "tester")
	require.Error(t, err)
	require.True(t, ent.IsNotFound(err), "wrapped not-found must stay classifiable as 404, got %v", err)
}

func TestConfigurationItemService_LifecycleCrossTenantFails(t *testing.T) {
	client, svc, ctx, _, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	other, err := client.Tenant.Create().SetName("Other").SetCode("cmdb-lifecycle-other").SetDomain("other-lifecycle.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	_, err = svc.UpdateLifecycleStatus(ctx, ciID, other.ID, LifecycleChangeParams{
		Status: common.CILifecycleStatusMaintenance,
	}, 1, "tester")
	require.Error(t, err)
	require.True(t, ent.IsNotFound(err), "cross-tenant access must fail closed as not-found, got %v", err)

	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusOnline, stored.LifecycleStatus)
}

func TestConfigurationItemService_LifecycleRetireFullPath(t *testing.T) {
	client, svc, ctx, tenantID, ciID, relationID := setupCILifecycleFixture(t)
	defer client.Close()

	ci, err := svc.UpdateLifecycleStatus(ctx, ciID, tenantID, LifecycleChangeParams{
		Status:           common.CILifecycleStatusScrapped,
		RetirementReason: "manual",
		Remark:           "asset disposal",
		ExpectedVersion:  1,
	}, 1, "tester")
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusScrapped, ci.LifecycleStatus)

	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusScrapped, stored.LifecycleStatus)
	require.Equal(t, common.CIStatusRetired, stored.Status)
	require.False(t, stored.ExpireAt.IsZero())
	require.Equal(t, 2, stored.Version)

	relation, err := client.CIRelationship.Query().Where(cirelationship.IDEQ(relationID)).Only(ctx)
	require.NoError(t, err)
	require.False(t, relation.IsActive, "retirement must deactivate relationships")

	hist, err := client.ConfigurationItemHistory.Query().Where(
		configurationitemhistory.CiIDEQ(ciID),
		configurationitemhistory.TenantIDEQ(tenantID),
		configurationitemhistory.OperationEQ("lifecycle_update"),
	).First(ctx)
	require.NoError(t, err)
	require.Contains(t, hist.Remark, "[retire:manual]")
	require.Contains(t, hist.Remark, "asset disposal")
	require.Equal(t, common.CILifecycleStatusScrapped, hist.After["lifecycle_status"])
}

func TestConfigurationItemService_LifecycleRetireRequiresControlledReason(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	_, err := svc.UpdateLifecycleStatus(ctx, ciID, tenantID, LifecycleChangeParams{
		Status: common.CILifecycleStatusScrapped,
	}, 1, "tester")
	require.Error(t, err)
	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, 400, appErr.HTTPStatus)

	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusOnline, stored.LifecycleStatus)
}

func TestConfigurationItemService_LifecycleRetireRejectsDraft(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	_, err := client.ConfigurationItem.UpdateOneID(ciID).SetLifecycleStatus(common.CILifecycleStatusDraft).Save(ctx)
	require.NoError(t, err)

	_, err = svc.UpdateLifecycleStatus(ctx, ciID, tenantID, LifecycleChangeParams{
		Status:           common.CILifecycleStatusScrapped,
		RetirementReason: "manual",
	}, 1, "tester")
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr)
	require.Equal(t, common.ConflictCode, bizErr.Code)
}

func TestConfigurationItemService_BatchLifecycleReportsPerItemFailures(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	missingID := 999999
	resp, err := svc.BatchUpdateLifecycleStatus(ctx, []int{ciID, missingID}, tenantID, LifecycleChangeParams{
		Status: common.CILifecycleStatusMaintenance,
	}, 1, "tester")
	require.NoError(t, err)
	require.Equal(t, 1, resp.SuccessCount)
	require.Equal(t, 1, resp.FailedCount)
	require.Equal(t, []int{missingID}, resp.FailedIDs)
	require.Len(t, resp.Errors, 1)
	require.True(t, strings.Contains(resp.Errors[0], "999999"))
}

func TestConfigurationItemService_UpdateCIBlocksScrappedAndIllegalLifecycle(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	_, err := svc.UpdateCI(ctx, ciID, tenantID, UpdateCIRequestForTest("scrapped"))
	require.Error(t, err)
	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, 400, appErr.HTTPStatus)

	_, err = svc.UpdateCI(ctx, ciID, tenantID, UpdateCIRequestForTest(common.CILifecycleStatusDraft))
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr)
	require.Equal(t, common.ConflictCode, bizErr.Code)

	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusOnline, stored.LifecycleStatus)
}

func TestConfigurationItemService_UpdateCISameLifecycleValueIdempotent(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	// 导入/发现重跑：lifecycle 与库内同值按未传处理，不因状态机无自环误报 409
	resp, err := svc.UpdateCI(ctx, ciID, tenantID, UpdateCIRequestForTest(common.CILifecycleStatusOnline))
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusOnline, resp.LifecycleStatus)
}

func TestConfigurationItemService_UpdateCIVersionConflictTyped(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	req := &dto.UpdateCIRequest{Version: 99, Description: "stale write"}
	_, err := svc.UpdateCI(ctx, ciID, tenantID, req)
	require.Error(t, err)
	var conflictErr *common.VersionConflictError
	require.ErrorAs(t, err, &conflictErr)
	require.Equal(t, 99, conflictErr.CurrentVersion)
	require.Equal(t, 1, conflictErr.ServerVersion)
}

func UpdateCIRequestForTest(lifecycle string) *dto.UpdateCIRequest {
	return &dto.UpdateCIRequest{Version: 1, LifecycleStatus: lifecycle}
}
