package marketplace

import (
	"context"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/itemversion"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/ent/tenantinstallation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newSkillService(t *testing.T, dsn string, runtime SkillRuntime) (*Service, *ent.Client) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:"+dsn+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	svc := NewService(client, zaptest.NewLogger(t).Sugar())
	if runtime != nil {
		svc.SetSkillRuntime(runtime)
	}
	return svc, client
}

// TestService_InstallItem_SkillActivatesWhenRuntimeHasCode 技能交付的正路径：
// 版本行定位符命中运行时注册码 ⇒ 安装记录 active，且出站记录带上商品边。
func TestService_InstallItem_SkillActivatesWhenRuntimeHasCode(t *testing.T) {
	svc, client := newSkillService(t, "skill_install_ok", fakeSkillRuntime{codes: defaultSkillCodes()})
	ctx := context.Background()
	item := createSkillCatalogItem(t, ctx, client, "ai-triage-skill", "1.0.0", "skill://ai.triage")

	installation, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.NoError(t, err)
	require.Equal(t, tenantinstallation.StatusActive, installation.Status)
	require.Equal(t, "1.0.0", installation.InstalledVersion)
	require.Empty(t, installation.ErrorMessage)
	// 商品边必须带上：deliverable/latestVersion 都来自它，缺失会让界面把「不知道」显示成「不可交付」。
	edge, err := installation.Edges.ItemOrErr()
	require.NoError(t, err)
	require.Equal(t, marketplaceitem.TypeSkill, edge.Type)

	// 卸载撤销的是调用权：安装记录转 uninstalled，运行时不受影响（多租户共享同一份实现）。
	require.NoError(t, svc.UninstallItem(ctx, 1, item.ID))
	requireInstallationState(t, ctx, client, 1, item.ID, tenantinstallation.StatusUninstalled, "")
	require.True(t, fakeSkillRuntime{codes: defaultSkillCodes()}.Has("ai.triage"))
}

// TestService_InstallItem_SkillFailsClosedWithoutRuntime 技能运行时未装配是部署问题：
// 不能跳过校验把商品标成 active，否则界面显示已装好而调用侧永远找不到实现。
func TestService_InstallItem_SkillFailsClosedWithoutRuntime(t *testing.T) {
	svc, client := newSkillService(t, "skill_no_runtime", nil)
	ctx := context.Background()
	item := createSkillCatalogItem(t, ctx, client, "ai-summarize-skill", "1.0.0", "skill://ai.summarize")

	installation, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMarketplaceRuntimeUnavailable), "got %v", err)
	require.NotNil(t, installation)
	requireInstallationState(t, ctx, client, 1, item.ID, tenantinstallation.StatusFailed, "skill runtime is not available")
	assert.True(t, IsDeliveryNotReady(err))
}

// TestService_InstallItem_SkillRejectsUnregisteredCode 定位符写了运行时里没有的码：
// 安装停在 failed，且原因里带上可用码清单，管理员才知道该改定位符还是先发技能。
func TestService_InstallItem_SkillRejectsUnregisteredCode(t *testing.T) {
	svc, client := newSkillService(t, "skill_bad_code", fakeSkillRuntime{codes: defaultSkillCodes()})
	ctx := context.Background()
	item := createSkillCatalogItem(t, ctx, client, "ghost-skill", "1.0.0", "skill://ai.does_not_exist")

	_, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMarketplaceSkillRuntimeMissing), "got %v", err)
	assert.Contains(t, err.Error(), "ai.triage")
	requireInstallationState(t, ctx, client, 1, item.ID, tenantinstallation.StatusFailed, "ai.does_not_exist")
}

// TestService_InstallItem_SkillRequiresVersionLocator 历史种子的版本行没有定位符时，
// 必须显式失败而不是「按最新版本猜一个实现」。
func TestService_InstallItem_SkillRequiresVersionLocator(t *testing.T) {
	svc, client := newSkillService(t, "skill_locator_missing", fakeSkillRuntime{codes: defaultSkillCodes()})
	ctx := context.Background()
	item := createSkillCatalogItem(t, ctx, client, "excel-export-skill", "1.0.0", "")

	_, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMarketplaceSkillArtifactMissing), "got %v", err)
	requireInstallationState(t, ctx, client, 1, item.ID, tenantinstallation.StatusFailed, "runtime locator")
}

// TestService_InstallItem_SkillRequiresPublishedVersionRow latest_version 指向
// 没有已发布版本行的版本号时不能退化成功：版本行才是「这个发布物存在」的证据。
func TestService_InstallItem_SkillRequiresPublishedVersionRow(t *testing.T) {
	svc, client := newSkillService(t, "skill_no_version_row", fakeSkillRuntime{codes: defaultSkillCodes()})
	ctx := context.Background()
	item := createSkillCatalogItem(t, ctx, client, "unpublished-skill", "0.9.0", "skill://ai.triage")
	// 把版本行撤成 draft：条目仍 published，但目标版本没有已发布发布物。
	_, err := client.ItemVersion.Update().
		Where(itemversion.ItemID(item.ID)).
		SetStatus(itemversion.StatusDraft).
		Save(ctx)
	require.NoError(t, err)

	_, err = svc.InstallItem(ctx, 1, item.ID, "1001")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMarketplaceVersionUnavailable), "got %v", err)
	requireInstallationState(t, ctx, client, 1, item.ID, tenantinstallation.StatusFailed, "没有已发布版本行")
}

// TestService_UpdateInstallationConfig_SkillHasNoConfigSurface 技能没有租户级配置面：
// 拒收（参数层不可受理）而不是收下并假装保存成功。
func TestService_UpdateInstallationConfig_SkillHasNoConfigSurface(t *testing.T) {
	svc, client := newSkillService(t, "skill_config_unsupported", fakeSkillRuntime{codes: defaultSkillCodes()})
	ctx := context.Background()
	item := createSkillCatalogItem(t, ctx, client, "ai-chat-skill", "1.0.0", "skill://ai.triage")
	_, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.NoError(t, err)

	installation, err := svc.UpdateInstallationConfig(ctx, 1, item.ID, map[string]interface{}{"prompt": "changed"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMarketplaceConfigUnsupported), "got %v", err)
	assert.False(t, IsDeliveryNotReady(err), "配置面缺失不是运行态未就绪，不应映射成 200/409 那一类")
	require.Nil(t, installation)

	current := requireInstallationState(t, ctx, client, 1, item.ID, tenantinstallation.StatusActive, "")
	assert.NotContains(t, current.Config, "prompt", "拒收的配置不得落库")
}

// TestService_SkillInvocationEntitlement 调用权判定：目录交付了什么、租户装了什么。
func TestService_SkillInvocationEntitlement(t *testing.T) {
	svc, client := newSkillService(t, "skill_entitlement", fakeSkillRuntime{codes: defaultSkillCodes()})
	ctx := context.Background()

	delivered := createSkillCatalogItem(t, ctx, client, "entitled-triage", "1.0.0", "skill://ai.triage")

	// 未被目录交付的技能码：不受市场约束，内置能力照旧可用（开门即断流是更糟的默认）。
	notDelivered, err := svc.SkillInvocationEntitlement(ctx, 1, "ai.summarize")
	require.NoError(t, err)
	assert.False(t, notDelivered.DeliveredByCatalog)
	assert.True(t, notDelivered.Allowed)

	// 目录已交付但本租户没装：拒绝，并指出该装哪个商品。
	denied, err := svc.SkillInvocationEntitlement(ctx, 2, "ai.triage")
	require.NoError(t, err)
	assert.True(t, denied.DeliveredByCatalog)
	assert.False(t, denied.Allowed)
	assert.Equal(t, delivered.Name, denied.ItemName)
	assert.Contains(t, denied.Reason, "尚未安装")

	// 租户 1 装上后生效；租户 2 仍然被拒——判定按租户隔离，不能因为「有租户装了」就放行。
	_, err = svc.InstallItem(ctx, 1, delivered.ID, "1001")
	require.NoError(t, err)
	allowed, err := svc.SkillInvocationEntitlement(ctx, 1, "ai.triage")
	require.NoError(t, err)
	assert.True(t, allowed.DeliveredByCatalog)
	assert.True(t, allowed.Allowed)
	assert.Equal(t, "1.0.0", allowed.InstalledVersion)

	crossTenant, err := svc.SkillInvocationEntitlement(ctx, 2, "ai.triage")
	require.NoError(t, err)
	assert.False(t, crossTenant.Allowed, "租户 1 的安装不能给租户 2 开调用权")

	// 卸载即撤权。
	require.NoError(t, svc.UninstallItem(ctx, 1, delivered.ID))
	afterUninstall, err := svc.SkillInvocationEntitlement(ctx, 1, "ai.triage")
	require.NoError(t, err)
	assert.False(t, afterUninstall.Allowed)
}

// TestService_SkillInvocationEntitlement_RejectsEmptyCode 空技能码是调用方错误，
// 不能退化成「查不到交付声明 ⇒ 放行」，那会把市场开通变成装饰。
func TestService_SkillInvocationEntitlement_RejectsEmptyCode(t *testing.T) {
	svc, _ := newSkillService(t, "skill_entitlement_empty", fakeSkillRuntime{codes: defaultSkillCodes()})

	_, err := svc.SkillInvocationEntitlement(context.Background(), 1, "  ")
	require.Error(t, err)
}

// TestIsDeliverableType 交付词表是单一事实源：界面、目录写、安装、升级都读它。
func TestIsDeliverableType(t *testing.T) {
	assert.True(t, IsDeliverableType(marketplaceitem.TypeConnector))
	assert.True(t, IsDeliverableType(marketplaceitem.TypeSkill))
	assert.False(t, IsDeliverableType(marketplaceitem.TypePlugin))
}
