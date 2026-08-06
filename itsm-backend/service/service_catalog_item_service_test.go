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

func TestServiceCatalogItemService_List_BatchLoadsFieldDefinitionsPerItem(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:catalog_item_list_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("t").SetCode("catalog-item-list-fields").SetDomain("d.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	catalog, err := client.ServiceCatalog.Create().SetName("云资源").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	svc := NewServiceCatalogItemService(client, zaptest.NewLogger(t).Sugar())

	// item1 has a field definition, item2 has none — the batch fetch must not
	// leak item1's definitions onto item2, and must still find item1's own.
	item1, err := svc.CreateServiceCatalogItem(ctx, &dto.CreateServiceCatalogItemRequest{
		CatalogID: catalog.ID,
		Name:      "云主机申请",
		Fields: []map[string]interface{}{
			{"name": "cpu_cores", "label": "CPU核数", "type": "number"},
		},
	}, tenant.ID)
	require.NoError(t, err)
	item2, err := svc.CreateServiceCatalogItem(ctx, &dto.CreateServiceCatalogItemRequest{
		CatalogID: catalog.ID,
		Name:      "网络申请",
	}, tenant.ID)
	require.NoError(t, err)

	list, err := svc.ListServiceCatalogItems(ctx, catalog.ID, tenant.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)

	byID := map[int]*dto.ServiceCatalogItemResponse{}
	for _, item := range list {
		byID[item.ID] = item
	}
	require.Len(t, byID[item1.ID].Fields, 1)
	assert.Equal(t, "cpu_cores", byID[item1.ID].Fields[0]["name"])
	assert.Empty(t, byID[item2.ID].Fields)
}
