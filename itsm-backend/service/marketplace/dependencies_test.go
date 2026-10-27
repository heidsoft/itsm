package marketplace

import (
	"context"
	"errors"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/itemversion"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/ent/tenantinstallation"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func createItemWithVersion(t *testing.T, ctx context.Context, client *ent.Client, name, version string, deps map[string]string) *ent.MarketplaceItem {
	t.Helper()
	item, err := client.MarketplaceItem.Create().
		SetName(name).
		SetType(marketplaceitem.TypeConnector).
		SetTitle(name + " connector").
		SetProvider("Open Source").
		SetLatestVersion(version).
		SetStatus(marketplaceitem.StatusPublished).
		Save(ctx)
	require.NoError(t, err)

	create := client.ItemVersion.Create().
		SetItemID(item.ID).
		SetVersion(version).
		SetStatus(itemversion.StatusPublished)
	if len(deps) > 0 {
		create = create.SetDependencies(deps)
	}
	_, err = create.Save(ctx)
	require.NoError(t, err)
	return item
}

func TestService_InstallItem_RejectsMissingDependency(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:marketplace_dep_missing?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewService(client, zaptest.NewLogger(t).Sugar())

	// 依赖组件存在于市场，但当前租户没装。
	createItemWithVersion(t, ctx, client, "base", "1.0.0", nil)
	item := createItemWithVersion(t, ctx, client, "dep", "1.0.0", map[string]string{"base": ">=1.0.0"})

	_, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMarketplaceDependencyUnsatisfied))
	assert.Contains(t, err.Error(), "base")
	assert.Contains(t, err.Error(), "未安装")

	// 关键不变量：依赖不满足时不得留下 installing/failed 安装记录。
	count, err := client.TenantInstallation.Query().
		Where(tenantinstallation.TenantID(1), tenantinstallation.ItemID(item.ID)).
		Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count)
}

func TestService_InstallItem_SatisfiesInstalledDependency(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:marketplace_dep_ok?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewService(client, zaptest.NewLogger(t).Sugar())

	base := createItemWithVersion(t, ctx, client, "base", "1.5.0", nil)
	_, err := svc.InstallItem(ctx, 1, base.ID, "1001")
	require.NoError(t, err)

	item := createItemWithVersion(t, ctx, client, "dep", "1.0.0", map[string]string{"base": ">=1.0.0"})
	installed, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.NoError(t, err)
	assert.Equal(t, tenantinstallation.StatusActive, installed.Status)
}

func TestService_InstallItem_RejectsVersionMismatch(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:marketplace_dep_ver?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewService(client, zaptest.NewLogger(t).Sugar())

	base := createItemWithVersion(t, ctx, client, "base", "1.0.0", nil)
	baseInstall, err := svc.InstallItem(ctx, 1, base.ID, "1001")
	require.NoError(t, err)
	// 已安装版本低于依赖要求。
	_, err = baseInstall.Update().SetInstalledVersion("0.9.0").Save(ctx)
	require.NoError(t, err)

	item := createItemWithVersion(t, ctx, client, "dep", "1.0.0", map[string]string{"base": ">=1.0.0"})
	_, err = svc.InstallItem(ctx, 1, item.ID, "1001")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMarketplaceDependencyUnsatisfied))
	assert.Contains(t, err.Error(), "已安装 0.9.0")
	assert.Contains(t, err.Error(), ">=1.0.0")
}

func TestService_InstallItem_DisabledDependencyCountsAsInstalled(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:marketplace_dep_disabled?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewService(client, zaptest.NewLogger(t).Sugar())

	base := createItemWithVersion(t, ctx, client, "base", "1.0.0", nil)
	baseInstall, err := svc.InstallItem(ctx, 1, base.ID, "1001")
	require.NoError(t, err)
	_, err = baseInstall.Update().SetStatus(tenantinstallation.StatusDisabled).Save(ctx)
	require.NoError(t, err)

	item := createItemWithVersion(t, ctx, client, "dep", "1.0.0", map[string]string{"base": "1.0.0"})
	installed, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.NoError(t, err)
	assert.Equal(t, tenantinstallation.StatusActive, installed.Status)
}

func TestService_InstallItem_WithoutVersionRowKeepsWorking(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:marketplace_dep_noversion?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewService(client, zaptest.NewLogger(t).Sugar())

	// 只有 MarketplaceItem、没有版本行：视为未声明依赖，不得误拦截。
	item := createPublishedMarketplaceItem(t, ctx, client, "no-version-row")
	installed, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.NoError(t, err)
	assert.Equal(t, tenantinstallation.StatusActive, installed.Status)
}

func TestService_InstallItem_RejectsSelfDependency(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:marketplace_dep_self?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewService(client, zaptest.NewLogger(t).Sugar())

	item := createItemWithVersion(t, ctx, client, "selfish", "1.0.0", map[string]string{"selfish": ">=1.0.0"})
	_, err := svc.InstallItem(ctx, 1, item.ID, "1001")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMarketplaceDependencyUnsatisfied))
	assert.Contains(t, err.Error(), "依赖自身")
}

func TestService_UninstallItem_BlocksWhenDependentInstalled(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:marketplace_uninstall_dependent?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewService(client, zaptest.NewLogger(t).Sugar())

	base := createItemWithVersion(t, ctx, client, "base", "1.0.0", nil)
	_, err := svc.InstallItem(ctx, 1, base.ID, "1001")
	require.NoError(t, err)
	dependent := createItemWithVersion(t, ctx, client, "dependent", "2.0.0", map[string]string{"base": ">=1.0.0"})
	_, err = svc.InstallItem(ctx, 1, dependent.ID, "1001")
	require.NoError(t, err)

	err = svc.UninstallItem(ctx, 1, base.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMarketplaceItemRequiredByDependents))
	assert.Contains(t, err.Error(), "dependent")
	assert.Contains(t, err.Error(), "2.0.0")

	// 不变量：卸载被拒时安装记录保持 active（不能被拆成半个状态）。
	installation, err := client.TenantInstallation.Query().
		Where(tenantinstallation.TenantID(1), tenantinstallation.ItemID(base.ID)).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, tenantinstallation.StatusActive, installation.Status)

	// 先卸依赖方，前置即可卸载。
	require.NoError(t, svc.UninstallItem(ctx, 1, dependent.ID))
	require.NoError(t, svc.UninstallItem(ctx, 1, base.ID))
}

func TestService_UninstallItem_DisabledDependentStillBlocks(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:marketplace_uninstall_disabled?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewService(client, zaptest.NewLogger(t).Sugar())

	base := createItemWithVersion(t, ctx, client, "base", "1.0.0", nil)
	_, err := svc.InstallItem(ctx, 1, base.ID, "1001")
	require.NoError(t, err)
	dependent := createItemWithVersion(t, ctx, client, "dependent", "1.0.0", map[string]string{"base": "1.0.0"})
	dependentInstall, err := svc.InstallItem(ctx, 1, dependent.ID, "1001")
	require.NoError(t, err)
	_, err = dependentInstall.Update().SetStatus(tenantinstallation.StatusDisabled).Save(ctx)
	require.NoError(t, err)

	// disabled 仍是租户内的已装组件，与安装期闸门口径一致。
	err = svc.UninstallItem(ctx, 1, base.ID)
	assert.True(t, errors.Is(err, ErrMarketplaceItemRequiredByDependents))
}

func TestService_UninstallItem_IgnoresFailedDependent(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:marketplace_uninstall_failed?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewService(client, zaptest.NewLogger(t).Sugar())

	base := createItemWithVersion(t, ctx, client, "base", "1.0.0", nil)
	_, err := svc.InstallItem(ctx, 1, base.ID, "1001")
	require.NoError(t, err)
	dependent := createItemWithVersion(t, ctx, client, "dependent", "1.0.0", map[string]string{"base": "1.0.0"})
	dependentInstall, err := svc.InstallItem(ctx, 1, dependent.ID, "1001")
	require.NoError(t, err)
	_, err = dependentInstall.Update().SetStatus(tenantinstallation.StatusFailed).Save(ctx)
	require.NoError(t, err)

	// failed 的依赖方没有生效，它的重试会重新过安装闸门。
	require.NoError(t, svc.UninstallItem(ctx, 1, base.ID))
}

func TestService_UninstallItem_UsesInstalledVersionDependencies(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:marketplace_uninstall_latest?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewService(client, zaptest.NewLogger(t).Sugar())

	base := createItemWithVersion(t, ctx, client, "base", "1.0.0", nil)
	_, err := svc.InstallItem(ctx, 1, base.ID, "1001")
	require.NoError(t, err)
	dependent := createItemWithVersion(t, ctx, client, "dependent", "1.0.0", nil)
	_, err = svc.InstallItem(ctx, 1, dependent.ID, "1001")
	require.NoError(t, err)

	// v2.0.0 新增了对 base 的依赖，但存量安装仍是 1.0.0（无依赖）。
	_, err = client.ItemVersion.Create().
		SetItemID(dependent.ID).
		SetVersion("2.0.0").
		SetStatus(itemversion.StatusPublished).
		SetDependencies(map[string]string{"base": ">=1.0.0"}).
		Save(ctx)
	require.NoError(t, err)
	_, err = dependent.Update().SetLatestVersion("2.0.0").Save(ctx)
	require.NoError(t, err)

	require.NoError(t, svc.UninstallItem(ctx, 1, base.ID))
}

func TestVersionSatisfies(t *testing.T) {
	cases := []struct {
		name        string
		installed   string
		requirement string
		want        bool
		wantErr     bool
	}{
		{name: "empty requirement is always satisfied", installed: "0.1.0", requirement: "", want: true},
		{name: "greater than", installed: "1.2.0", requirement: ">=1.0.0", want: true},
		{name: "equal boundary", installed: "1.0.0", requirement: ">=1.0.0", want: true},
		{name: "lower than", installed: "0.9.0", requirement: ">=1.0.0", want: false},
		{name: "strict greater fails on equal", installed: "1.0.0", requirement: ">1.0.0", want: false},
		{name: "less than", installed: "0.9.0", requirement: "<1.0.0", want: true},
		{name: "exact with v prefix", installed: "v1.2.3", requirement: "1.2.3", want: true},
		{name: "exact mismatch", installed: "1.2.3", requirement: "1.2.4", want: false},
		{name: "missing patch treated as zero", installed: "1.2", requirement: ">=1.2.0", want: true},
		{name: "prerelease suffix ignored", installed: "1.0.0-rc.1", requirement: "=1.0.0", want: true},
		{name: "unparseable installed", installed: "latest", requirement: ">=1.0.0", wantErr: true},
		{name: "unparseable requirement", installed: "1.0.0", requirement: ">=x.y", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := versionSatisfies(tc.installed, tc.requirement)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
