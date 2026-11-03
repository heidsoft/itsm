package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

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

func TestService_InstallItem_IsIdempotentForExistingInstallation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:marketplace_idempotent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	item := createPublishedMarketplaceItem(t, ctx, client, "feishu")

	first, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.NoError(t, err)
	require.NotNil(t, first)

	second, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, tenantinstallation.StatusActive, second.Status)

	refreshedItem, err := client.MarketplaceItem.Get(ctx, item.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, refreshedItem.InstallCount)
}

func TestService_UninstallItem_ReturnsNotFoundForMissingInstallation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:marketplace_uninstall_missing?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	item := createPublishedMarketplaceItem(t, ctx, client, "wecom")

	err := svc.UninstallItem(ctx, 1, item.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMarketplaceInstallationAbsent))
}

func TestService_InstallItem_ReactivatesUninstalledHistory(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:marketplace_reactivate?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	item := createPublishedMarketplaceItem(t, ctx, client, "dingtalk")

	first, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.NoError(t, err)

	err = svc.UninstallItem(ctx, 1, item.ID)
	require.NoError(t, err)

	reinstalled, err := svc.InstallItem(ctx, 1, item.ID, "2002")
	require.NoError(t, err)
	assert.Equal(t, first.ID, reinstalled.ID)
	assert.Equal(t, tenantinstallation.StatusActive, reinstalled.Status)
	assert.Equal(t, "2002", reinstalled.InstalledBy)

	refreshedItem, err := client.MarketplaceItem.Get(ctx, item.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, refreshedItem.InstallCount)
}

func TestService_UnsupportedDeliveryDoesNotMutateInstallations(t *testing.T) {
	// 只覆盖 plugin：平台没有插件宿主，交付词表里不含它，因此安装与配置都拒收。
	// skill 已有真实交付语义（见 skill_delivery_test.go），不再属于「不支持的类型」。
	for _, itemType := range []marketplaceitem.Type{marketplaceitem.TypePlugin} {
		for _, status := range []tenantinstallation.Status{"", tenantinstallation.StatusActive, tenantinstallation.StatusDisabled, tenantinstallation.StatusInstalling, tenantinstallation.StatusFailed, tenantinstallation.StatusUninstalled} {
			t.Run(string(itemType)+"/"+string(status), func(t *testing.T) {
				client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared&_fk=1")
				t.Cleanup(func() { _ = client.Close() })
				ctx := context.Background()
				svc := NewService(client, zaptest.NewLogger(t).Sugar())
				item := createPublishedMarketplaceItem(t, ctx, client, "unsupported")
				item, err := item.Update().SetType(itemType).SetInstallCount(7).Save(ctx)
				require.NoError(t, err)

				var existing *ent.TenantInstallation
				if status != "" {
					existing, err = client.TenantInstallation.Create().
						SetTenantID(1).SetItemID(item.ID).SetInstalledVersion("0.9.0").
						SetStatus(status).SetInstalledBy("1001").
						SetConfig(map[string]interface{}{"mode": "original"}).Save(ctx)
					require.NoError(t, err)
					existing, err = client.TenantInstallation.Get(ctx, existing.ID)
					require.NoError(t, err)
				}
				other, err := client.TenantInstallation.Create().
					SetTenantID(2).SetItemID(item.ID).SetInstalledVersion("0.8.0").
					SetStatus(tenantinstallation.StatusActive).SetInstalledBy("2001").Save(ctx)
				require.NoError(t, err)
				other, err = client.TenantInstallation.Get(ctx, other.ID)
				require.NoError(t, err)

				result, installErr := svc.InstallItem(ctx, 1, item.ID, "1002")
				require.ErrorIs(t, installErr, ErrMarketplaceDeliveryUnsupported)
				require.Nil(t, result)
				if status != "" && status != tenantinstallation.StatusUninstalled {
					result, configErr := svc.UpdateInstallationConfig(ctx, 1, item.ID, map[string]interface{}{"mode": "changed"})
					assert.ErrorIs(t, configErr, ErrMarketplaceDeliveryUnsupported)
					assert.Nil(t, result)
				}

				count, err := client.TenantInstallation.Query().
					Where(tenantinstallation.TenantID(1), tenantinstallation.ItemID(item.ID)).Count(ctx)
				require.NoError(t, err)
				if existing == nil {
					assert.Zero(t, count)
				} else {
					assert.Equal(t, 1, count)
					current, err := client.TenantInstallation.Get(ctx, existing.ID)
					require.NoError(t, err)
					before, err := json.Marshal(existing)
					require.NoError(t, err)
					after, err := json.Marshal(current)
					require.NoError(t, err)
					assert.JSONEq(t, string(before), string(after))
				}
				currentOther, err := client.TenantInstallation.Get(ctx, other.ID)
				require.NoError(t, err)
				before, err := json.Marshal(other)
				require.NoError(t, err)
				after, err := json.Marshal(currentOther)
				require.NoError(t, err)
				assert.JSONEq(t, string(before), string(after))
				currentItem, err := client.MarketplaceItem.Get(ctx, item.ID)
				require.NoError(t, err)
				assert.Equal(t, 7, currentItem.InstallCount)
			})
		}
	}
}

func createPublishedMarketplaceItem(t *testing.T, ctx context.Context, client *ent.Client, name string) *ent.MarketplaceItem {
	t.Helper()

	item, err := client.MarketplaceItem.Create().
		SetName(name).
		SetType(marketplaceitem.TypeConnector).
		SetTitle(name + " connector").
		SetProvider("Open Source").
		SetLatestVersion("1.0.0").
		SetStatus(marketplaceitem.StatusPublished).
		Save(ctx)
	require.NoError(t, err)
	return item
}

// fakeSkillRuntime 是技能运行时的测试替身，只声明「本进程里有哪些技能码」。
// 码取自 handlers/ai 实际注册的技能，避免测试用运行时里根本不存在的名字自证通过。
type fakeSkillRuntime struct {
	codes []string
}

func (f fakeSkillRuntime) Has(code string) bool {
	for _, c := range f.codes {
		if c == code {
			return true
		}
	}
	return false
}

func (f fakeSkillRuntime) Codes() []string {
	return append([]string(nil), f.codes...)
}

// defaultSkillCodes 覆盖技能测试用到的运行时码。
func defaultSkillCodes() []string {
	return []string{"ai.triage", "ai.create_ticket", "ai.summarize"}
}

// createSkillCatalogItem 建一个「目录已上架 + 有已发布版本行 + 版本行带定位符」的技能商品，
// 即技能交付链判定为可用所需的最小完整形态。版本行是必须的：缺它就没有定位符可读，
// 安装会停在 failed（这正是 skill_delivery_test.go 里另一组用例要证明的）。
func createSkillCatalogItem(
	t *testing.T,
	ctx context.Context,
	client *ent.Client,
	name string,
	version string,
	locator string,
) *ent.MarketplaceItem {
	t.Helper()

	item, err := client.MarketplaceItem.Create().
		SetName(name).
		SetType(marketplaceitem.TypeSkill).
		SetTitle(name).
		SetProvider("internal").
		SetLatestVersion(version).
		SetStatus(marketplaceitem.StatusPublished).
		Save(ctx)
	require.NoError(t, err)

	create := client.ItemVersion.Create().
		SetItemID(item.ID).
		SetVersion(version).
		SetStatus(itemversion.StatusPublished).
		SetReleasedAt(time.Now())
	if locator != "" {
		create = create.SetManifestPath(locator)
	}
	_, err = create.Save(ctx)
	require.NoError(t, err)
	return item
}

// requireInstallationState 读取某租户对某条目的安装记录并核对状态与错误原因。
func requireInstallationState(
	t *testing.T,
	ctx context.Context,
	client *ent.Client,
	tenantID int,
	itemID int,
	wantStatus tenantinstallation.Status,
	wantErrContains string,
) *ent.TenantInstallation {
	t.Helper()

	installation, err := client.TenantInstallation.Query().
		Where(tenantinstallation.TenantID(tenantID), tenantinstallation.ItemID(itemID)).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, string(wantStatus), string(installation.Status))
	if wantErrContains != "" {
		require.Contains(t, installation.ErrorMessage, wantErrContains)
	}
	return installation
}
