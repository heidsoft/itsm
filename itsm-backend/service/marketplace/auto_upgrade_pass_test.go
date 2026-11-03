package marketplace

import (
	"context"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/itemversion"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/ent/tenantinstallation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 自动升级后台 pass 的行为测试。
//
// auto_upgrade 开关此前只有写入路径：租户打开了它，却没有任何执行者把
// 「目录发布了新版本」变成「这个租户真的升上去」。这里锁定 pass 的四种结果，
// 尤其是一条失败不能冻结其余租户——一个商品的坏版本行会命中所有装了它的租户。

func addPublishedSkillVersion(t *testing.T, ctx context.Context, client *ent.Client, itemID int, version, locator string) {
	t.Helper()
	_, err := client.ItemVersion.Create().
		SetItemID(itemID).
		SetVersion(version).
		SetStatus(itemversion.StatusPublished).
		SetManifestPath(locator).
		SetReleasedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
}

func requireInstalledVersion(t *testing.T, ctx context.Context, client *ent.Client, tenantID, itemID int) (string, tenantinstallation.Status) {
	t.Helper()
	installation, err := client.TenantInstallation.Query().
		Where(tenantinstallation.TenantID(tenantID), tenantinstallation.ItemID(itemID)).
		Only(ctx)
	require.NoError(t, err)
	return installation.InstalledVersion, installation.Status
}

// TestRunAutoUpgradePass_UpgradesEnabledInstallations 开关打开且目录有更新版本时真的换版本号。
func TestRunAutoUpgradePass_UpgradesEnabledInstallations(t *testing.T) {
	svc, client := newSkillService(t, "auto_upgrade_pass_ok", fakeSkillRuntime{codes: defaultSkillCodes()})
	ctx := context.Background()

	item := createSkillCatalogItem(t, ctx, client, "triage-skill", "1.0.0", "skill://ai.triage")
	_, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.NoError(t, err)

	// 目录发布 1.1.0 并把 latest_version 推进到它：这就是「运营上架了新版本」。
	addPublishedSkillVersion(t, ctx, client, item.ID, "1.1.0", "skill://ai.triage")
	_, err = svc.PublishItemVersion(ctx, item.ID, "1.1.0")
	require.NoError(t, err)

	summary, err := svc.RunAutoUpgradePass(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Scanned)
	assert.Equal(t, 1, summary.Upgraded)
	assert.Zero(t, summary.Failed)
	assert.Empty(t, summary.Failures)

	version, status := requireInstalledVersion(t, ctx, client, 1, item.ID)
	assert.Equal(t, "1.1.0", version, "开关打开就必须真的升上去，否则 auto_upgrade 只是界面字段")
	assert.Equal(t, tenantinstallation.StatusActive, status)
}

// TestRunAutoUpgradePass_SkipsDisabledAndUpToDate 关闭开关的租户与已在最新版本的安装只统计不改动。
func TestRunAutoUpgradePass_SkipsDisabledAndUpToDate(t *testing.T) {
	svc, client := newSkillService(t, "auto_upgrade_pass_skip", fakeSkillRuntime{codes: defaultSkillCodes()})
	ctx := context.Background()

	item := createSkillCatalogItem(t, ctx, client, "triage-skill", "1.0.0", "skill://ai.triage")
	_, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.NoError(t, err)
	_, err = svc.InstallItem(ctx, 2, item.ID, "1002")
	require.NoError(t, err)
	// 租户 2 主动关闭自动升级：后台 pass 不得替它决定。
	_, err = svc.SetAutoUpgrade(ctx, 2, item.ID, false, "1002")
	require.NoError(t, err)

	addPublishedSkillVersion(t, ctx, client, item.ID, "1.1.0", "skill://ai.triage")
	_, err = svc.PublishItemVersion(ctx, item.ID, "1.1.0")
	require.NoError(t, err)

	summary, err := svc.RunAutoUpgradePass(ctx)
	require.NoError(t, err)
	// 关闭开关的安装连扫描都不进：扫描数就是「有资格被升级的安装数」。
	assert.Equal(t, 1, summary.Scanned, "auto_upgrade=false 的安装不应进入扫描")
	assert.Equal(t, 1, summary.Upgraded)
	version, _ := requireInstalledVersion(t, ctx, client, 2, item.ID)
	assert.Equal(t, "1.0.0", version, "未开自动升级的租户版本不能被后台改动")

	// 已在最新版本的安装进入扫描后计为 skipped，而不是失败。
	summary, err = svc.RunAutoUpgradePass(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Scanned)
	assert.Zero(t, summary.Upgraded)
	assert.Equal(t, 1, summary.Skipped)
}

// TestRunAutoUpgradePass_FailureIsolation 一个商品升失败不得冻结其它商品的升级。
func TestRunAutoUpgradePass_FailureIsolation(t *testing.T) {
	svc, client := newSkillService(t, "auto_upgrade_pass_failure", fakeSkillRuntime{codes: defaultSkillCodes()})
	ctx := context.Background()

	broken := createSkillCatalogItem(t, ctx, client, "broken-skill", "1.0.0", "skill://ai.triage")
	healthy := createSkillCatalogItem(t, ctx, client, "healthy-skill", "1.0.0", "skill://ai.summarize")
	_, err := svc.InstallItem(ctx, 1, broken.ID, "1001")
	require.NoError(t, err)
	_, err = svc.InstallItem(ctx, 1, healthy.ID, "1001")
	require.NoError(t, err)

	// broken 的新版本指向运行时不存在的码：provision 失败，版本必须保持不变。
	addPublishedSkillVersion(t, ctx, client, broken.ID, "1.1.0", "skill://ai.removed-in-runtime")
	_, err = svc.PublishItemVersion(ctx, broken.ID, "1.1.0")
	require.NoError(t, err)
	addPublishedSkillVersion(t, ctx, client, healthy.ID, "1.1.0", "skill://ai.summarize")
	_, err = svc.PublishItemVersion(ctx, healthy.ID, "1.1.0")
	require.NoError(t, err)

	summary, err := svc.RunAutoUpgradePass(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Scanned)
	assert.Equal(t, 1, summary.Upgraded, "同批健康商品照常升级")
	assert.Equal(t, 1, summary.Failed)
	require.Len(t, summary.Failures, 1)
	assert.Contains(t, summary.Failures[0], "ai.removed-in-runtime", "逐条归因要能被运维直接定位")

	brokenVersion, brokenStatus := requireInstalledVersion(t, ctx, client, 1, broken.ID)
	assert.Equal(t, "1.0.0", brokenVersion, "失败不得推进版本号")
	assert.Equal(t, tenantinstallation.StatusActive, brokenStatus, "旧发布物仍在运行，不能谎报成失败停机")

	healthyVersion, _ := requireInstalledVersion(t, ctx, client, 1, healthy.ID)
	assert.Equal(t, "1.1.0", healthyVersion)
}

// TestRunAutoUpgradePass_IgnoresUndeliverableAndUnpublished 扫描范围只含可交付且已上架的商品：
// 已下架商品与新租户安装不受影响，插件根本没有执行者。
func TestRunAutoUpgradePass_IgnoresUndeliverableAndUnpublished(t *testing.T) {
	svc, client := newSkillService(t, "auto_upgrade_pass_scope", fakeSkillRuntime{codes: defaultSkillCodes()})
	ctx := context.Background()

	unpublished := createSkillCatalogItem(t, ctx, client, "retired-skill", "1.0.0", "skill://ai.triage")
	installed, err := svc.InstallItem(ctx, 1, unpublished.ID, "1001")
	require.NoError(t, err)
	require.True(t, installed.AutoUpgrade, "默认开启自动升级；否则本用例测不到扫描范围")

	plugin, err := client.MarketplaceItem.Create().
		SetName("legacy-plugin").
		SetType(marketplaceitem.TypePlugin).
		SetTitle("遗留插件").
		SetProvider("internal").
		SetLatestVersion("1.0.0").
		SetStatus(marketplaceitem.StatusPublished).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.TenantInstallation.Create().
		SetTenantID(1).SetItemID(plugin.ID).SetInstalledVersion("1.0.0").
		SetStatus(tenantinstallation.StatusActive).SetAutoUpgrade(true).
		SetInstalledBy("1001").Save(ctx)
	require.NoError(t, err)

	// 下架该技能商品：目录不再对租户负责，后台也不该继续推版本。
	_, err = svc.UnpublishCatalogItem(ctx, unpublished.ID)
	require.NoError(t, err)
	addPublishedSkillVersion(t, ctx, client, unpublished.ID, "1.1.0", "skill://ai.triage")

	summary, err := svc.RunAutoUpgradePass(ctx)
	require.NoError(t, err)
	assert.Zero(t, summary.Scanned, "已下架商品与不可交付类型都不该进入升级扫描")
	assert.Zero(t, summary.Upgraded)
	assert.Zero(t, summary.Failed)

	version, _ := requireInstalledVersion(t, ctx, client, 1, plugin.ID)
	assert.Equal(t, "1.0.0", version)
}
