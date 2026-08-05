package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestServiceCatalogItemService_CreateAndGet_PersistsFieldDefinitions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:catalog_item_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("t").SetCode("catalog-item-fields").SetDomain("d.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	catalog, err := client.ServiceCatalog.Create().SetName("云资源").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	svc := NewServiceCatalogItemService(client, zaptest.NewLogger(t).Sugar())
	created, err := svc.CreateServiceCatalogItem(ctx, &dto.CreateServiceCatalogItemRequest{
		CatalogID: catalog.ID,
		Name:      "云主机申请",
		Fields: []map[string]interface{}{
			{"name": "cpu_cores", "label": "CPU核数", "type": "number"},
		},
	}, tenant.ID)
	require.NoError(t, err)
	require.Len(t, created.Fields, 1)
	assert.Equal(t, "cpu_cores", created.Fields[0]["name"])

	fetched, err := svc.GetServiceCatalogItem(ctx, created.ID, tenant.ID)
	require.NoError(t, err)
	require.Len(t, fetched.Fields, 1)
	assert.Equal(t, "CPU核数", fetched.Fields[0]["label"])
}
