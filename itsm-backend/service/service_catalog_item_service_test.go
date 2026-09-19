package service

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
)

func TestToItemResponse_IncludesNewFields(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, "file:ent?mode=memory&_fk=1")
	defer client.Close()

	logger, _ := zap.NewDevelopment()
	svc := NewServiceCatalogItemService(client, logger.Sugar())

	ctx := context.Background()

	catalog := client.ServiceCatalog.Create().
		SetName("测试目录").
		SetDescription("测试").
		SetTenantID(1).
		SaveX(ctx)

	item := client.ServiceCatalogItem.Create().
		SetCatalogID(catalog.ID).
		SetName("ECS 实例申请").
		SetDescription("申请新建弹性云服务器实例").
		SetBusinessSubType("service_request").
		SetProcessDefinitionKey("service_request_flow").
		SetRequiresApproval(true).
		SetEstimatedDays(1).
		SetTenantID(1).
		SaveX(ctx)

	resp := svc.toItemResponse(item)
	assert.Equal(t, "ECS 实例申请", resp.Name)
	assert.Equal(t, "service_request", resp.BusinessSubType)
	assert.Equal(t, "service_request_flow", resp.ProcessDefinitionKey)
	assert.True(t, resp.RequiresApproval)
	assert.Equal(t, 1, resp.EstimatedDays)
}

func TestCreateServiceCatalogItem_WithBusinessSubType(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, "file:ent?mode=memory&_fk=1")
	defer client.Close()

	logger, _ := zap.NewDevelopment()
	svc := NewServiceCatalogItemService(client, logger.Sugar())

	ctx := context.Background()

	catalog := client.ServiceCatalog.Create().
		SetName("测试目录").
		SetDescription("测试").
		SetTenantID(1).
		SaveX(ctx)

	req := &dto.CreateServiceCatalogItemRequest{
		CatalogID:            catalog.ID,
		Name:                 "密码重置",
		Description:          "账户密码重置服务",
		BusinessSubType:      "service_request",
		ProcessDefinitionKey: "service_request_flow",
		RequiresApproval:     false,
		EstimatedDays:        0,
	}

	resp, err := svc.CreateServiceCatalogItem(ctx, req, 1)
	require.NoError(t, err)
	assert.Equal(t, "密码重置", resp.Name)
	assert.Equal(t, "service_request", resp.BusinessSubType)
	assert.Equal(t, "service_request_flow", resp.ProcessDefinitionKey)
}
