package marketplace

import (
	"context"
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
