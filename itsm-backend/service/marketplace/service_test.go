package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
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
	for _, itemType := range []marketplaceitem.Type{marketplaceitem.TypeSkill, marketplaceitem.TypePlugin} {
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
