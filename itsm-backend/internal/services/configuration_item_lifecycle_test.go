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
	require.Equal(t, common.CILifecycleStatusDraft, hist.Before["lifecycleStatus"])
	require.Equal(t, common.CILifecycleStatusOnline, hist.After["lifecycleStatus"])
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
	require.Equal(t, common.CILifecycleStatusScrapped, hist.After["lifecycleStatus"])
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

// ---------------------------------------------------------------------------
// P2-5 上线必填属性门禁
//
// 必填校验器（CIAttributeValidator）只在 CreateCI/UpdateCI 两条写路径上生效，
// 而生命周期推进是第三条写路径：它直接改 lifecycle_status，从不碰 attributes。
// 于是下面三类 CI 都可以被推进为 in_service（online），变成「已在服务中」却缺
// 关键信息的记录——影响分析、变更风险评估与权限分配读的都是这张表：
//   a) 创建时填过必填项、之后被清空
//   b) 属性定义在 CI 创建之后才新增了必填项
//   c) 云发现/交付投影落库后未补齐（投影层缺值时本就拒绝写，但历史数据存在）
// ---------------------------------------------------------------------------

// seedRequiredAttribute 给指定 CI 类型挂一个必填属性定义。
//
// displayName 刻意用中文并带括号外的可识别串：错误文案要能被人读懂，
// 同时 name 也要出现（前端按 name 定位表单字段）。
func seedRequiredAttribute(t *testing.T, client *ent.Client, tenantID, ciTypeID int, name, displayName, defaultValue string) {
	t.Helper()
	builder := client.CIAttributeDefinition.Create().
		SetName(name).
		SetDisplayName(displayName).
		SetType("string").
		SetRequired(true).
		SetCiTypeID(ciTypeID).
		SetTenantID(tenantID).
		SetIsActive(true)
	if defaultValue != "" {
		builder.SetDefaultValue(defaultValue)
	}
	_, err := builder.Save(context.Background())
	require.NoError(t, err)
}

func TestConfigurationItemService_OnlineBlockedByMissingRequiredAttribute(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)

	seedRequiredAttribute(t, client, tenantID, stored.CiTypeID, "ip_address", "IP 地址", "")
	seedRequiredAttribute(t, client, tenantID, stored.CiTypeID, "owner_team", "归属团队", "")

	// 回到 draft 才能推进到 online
	_, err = client.ConfigurationItem.UpdateOneID(ciID).
		SetLifecycleStatus(common.CILifecycleStatusDraft).Save(ctx)
	require.NoError(t, err)

	_, err = svc.UpdateLifecycleStatus(ctx, ciID, tenantID, LifecycleChangeParams{
		Status: common.CILifecycleStatusOnline,
		Remark: "publish",
	}, 1, "tester")

	require.Error(t, err, "缺必填属性的 CI 不得上线")
	require.ErrorIs(t, err, dto.ErrCIAttributeValidationRequiredOnline,
		"必须命中专用 sentinel，便于调用方与本包其他属性错误区分")

	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, 400, appErr.HTTPStatus, "参数/前置条件不满足应是 4xx，不是 500")

	// 文案必须点名缺哪几项，并且同时给出展示名与字段名（前端按 name 定位表单）
	require.Contains(t, appErr.Message, "IP 地址")
	require.Contains(t, appErr.Message, "ip_address")
	require.Contains(t, appErr.Message, "归属团队")
	require.Contains(t, appErr.Message, "owner_team")
	require.Contains(t, appErr.Message, "2 个必填属性")

	// 状态不得被推进：门禁必须发生在落库前
	stored, err = client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusDraft, stored.LifecycleStatus,
		"门禁失败时生命周期状态不得变化")
	require.Equal(t, 1, stored.Version, "门禁失败时不应消耗版本号")
}

// 补齐必填属性后必须能正常上线——防止门禁写死成「永远拒绝」。
func TestConfigurationItemService_OnlineAllowedWhenRequiredAttributesPresent(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	seedRequiredAttribute(t, client, tenantID, stored.CiTypeID, "ip_address", "IP 地址", "")

	_, err = client.ConfigurationItem.UpdateOneID(ciID).
		SetAttributes(map[string]interface{}{"ip_address": "10.0.0.1"}).
		SetLifecycleStatus(common.CILifecycleStatusDraft).Save(ctx)
	require.NoError(t, err)

	res, err := svc.UpdateLifecycleStatus(ctx, ciID, tenantID, LifecycleChangeParams{
		Status: common.CILifecycleStatusOnline,
		Remark: "publish",
	}, 1, "tester")
	require.NoError(t, err, "必填属性齐全时应放行")
	require.Equal(t, common.CILifecycleStatusOnline, res.LifecycleStatus)
}

// 空白字符串等同缺失：创建时填了默认值、后来被清空成 "" 的 CI 不能借「定义有默认值」蒙混过去。
func TestConfigurationItemService_OnlineBlockedByBlankStringAttribute(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	seedRequiredAttribute(t, client, tenantID, stored.CiTypeID, "ip_address", "IP 地址", "0.0.0.0")

	_, err = client.ConfigurationItem.UpdateOneID(ciID).
		SetAttributes(map[string]interface{}{"ip_address": "   "}).
		SetLifecycleStatus(common.CILifecycleStatusDraft).Save(ctx)
	require.NoError(t, err)

	_, err = svc.UpdateLifecycleStatus(ctx, ciID, tenantID, LifecycleChangeParams{
		Status: common.CILifecycleStatusOnline,
	}, 1, "tester")
	require.ErrorIs(t, err, dto.ErrCIAttributeValidationRequiredOnline,
		"纯空白值等同于缺失（默认值定义不能豁免已落库的空值）")
}

// 类型没有任何必填定义时不得被门禁误伤：既有 CI 全都不带属性定义，门禁必须放行。
func TestConfigurationItemService_OnlineAllowedWithoutAnyRequiredDefinition(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	_, err := client.ConfigurationItem.UpdateOneID(ciID).
		SetLifecycleStatus(common.CILifecycleStatusDraft).Save(ctx)
	require.NoError(t, err)

	res, err := svc.UpdateLifecycleStatus(ctx, ciID, tenantID, LifecycleChangeParams{
		Status: common.CILifecycleStatusOnline,
	}, 1, "tester")
	require.NoError(t, err, "无必填定义的类型应直接放行")
	require.Equal(t, common.CILifecycleStatusOnline, res.LifecycleStatus)
}

// 非上线迁移不受门禁影响：draft→offline 也是合法迁移，但不能因为缺属性就拦住
// （离线/维护态本就是「不提供服务」的状态，对属性完整度不该有同等要求）。
func TestConfigurationItemService_NonOnlineTransitionIgnoresRequiredAttributes(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	seedRequiredAttribute(t, client, tenantID, stored.CiTypeID, "ip_address", "IP 地址", "")

	// online → maintenance 缺属性也必须放行
	_, err = svc.UpdateLifecycleStatus(ctx, ciID, tenantID, LifecycleChangeParams{
		Status: common.CILifecycleStatusMaintenance,
		Remark: "planned maintenance",
	}, 1, "tester")
	require.NoError(t, err, "维护态迁移不应被必填属性门禁拦截")
}

// 批量路径复用单条逻辑，因此逐项返回失败而不整体失败。
func TestConfigurationItemService_BatchOnlineReportsMissingAttributePerCI(t *testing.T) {
	client, svc, ctx, tenantID, ciID, _ := setupCILifecycleFixture(t)
	defer client.Close()

	stored, err := client.ConfigurationItem.Get(ctx, ciID)
	require.NoError(t, err)
	seedRequiredAttribute(t, client, tenantID, stored.CiTypeID, "ip_address", "IP 地址", "")

	// 再造一条已填属性的 CI，验证批量里「能上的上、不能上的留在原地」
	filled, err := client.ConfigurationItem.Create().
		SetName("filled").SetCiTypeID(stored.CiTypeID).SetCiType("Server").
		SetStatus("active").SetTenantID(tenantID).
		SetAttributes(map[string]interface{}{"ip_address": "10.0.0.2"}).
		SetLifecycleStatus(common.CILifecycleStatusDraft).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ConfigurationItem.UpdateOneID(ciID).
		SetLifecycleStatus(common.CILifecycleStatusDraft).Save(ctx)
	require.NoError(t, err)

	res, err := svc.BatchUpdateLifecycleStatus(ctx, []int{ciID, filled.ID}, tenantID, LifecycleChangeParams{
		Status: common.CILifecycleStatusOnline,
	}, 1, "tester")
	require.NoError(t, err)
	require.Equal(t, 1, res.SuccessCount, "只有属性齐全的那条应成功")
	require.Equal(t, 1, res.FailedCount)
	require.Equal(t, []int{ciID}, res.FailedIDs, "缺属性的那条必须被点名")
	require.NotEmpty(t, res.Errors)
	require.Contains(t, res.Errors[0], "ip_address", "批量错误文案也要能定位缺哪项")

	got, err := client.ConfigurationItem.Get(ctx, filled.ID)
	require.NoError(t, err)
	require.Equal(t, common.CILifecycleStatusOnline, got.LifecycleStatus)
}
