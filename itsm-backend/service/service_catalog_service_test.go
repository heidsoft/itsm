package service

import (
	"context"
	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	"testing"
	
	"github.com/stretchr/testify/assert"
	_ "github.com/mattn/go-sqlite3"
)

func TestServiceCatalogService_GetServiceCatalogs(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	
	service := NewServiceCatalogService(client)
	ctx := context.Background()
	
	// 创建测试数据
	_, err := client.ServiceCatalog.Create().
		SetName("测试服务").
		SetCategory("IT服务").
		SetDescription("测试服务描述").
		SetDeliveryTime("1-3个工作日").
		SetStatus("enabled").
		Save(ctx)
	
	assert.NoError(t, err)
	
	// 测试获取服务目录列表
	req := &dto.GetServiceCatalogsRequest{
		Page: 1,
		Size: 10,
	}
	
	result, err := service.GetServiceCatalogs(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Catalogs, 1)
	assert.Equal(t, "测试服务", result.Catalogs[0].Name)
}