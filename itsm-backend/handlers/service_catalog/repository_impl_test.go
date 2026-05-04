package service_catalog

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestEntRepository_Search_IncludesLegacyActiveStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()

	repo := NewEntRepository(client)

	_, err := client.ServiceCatalog.Create().
		SetName("VM Service").
		SetCategory("it_service").
		SetDescription("virtual machine").
		SetDeliveryTime(1).
		SetStatus("active").
		SetTenantID(1).
		Save(ctx)
	require.NoError(t, err)

	list, total, err := repo.Search(ctx, 1, "vm", ListFilters{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, list, 1)
	require.Equal(t, "active", list[0].Status)
}
