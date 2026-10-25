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
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/schema"
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

// setupItemTest 组装目录项服务的独立测试库与基础数据。
func setupItemTest(t *testing.T) (ctx context.Context, svc *ServiceCatalogItemService, client *ent.Client, catalogID int) {
	t.Helper()
	client = enttest.Open(t, dialect.SQLite, "file:ent?mode=memory&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger, _ := zap.NewDevelopment()
	svc = NewServiceCatalogItemService(client, logger.Sugar())
	ctx = context.Background()
	catalog := client.ServiceCatalog.Create().
		SetName("绑定校验目录").
		SetDescription("测试").
		SetTenantID(1).
		SaveX(ctx)
	return ctx, svc, client, catalog.ID
}

// TestCreateServiceCatalogItem_SLABindingValidated 锁定 2026-10-23 收口：
// 目录项创建时 sla_id 必须存在且属于本租户——此前 sla_id 无任何校验，
// 可以指向不存在的 SLA，配置面上绑定成功、消费面上必然失效（假绑定）。
func TestCreateServiceCatalogItem_SLABindingValidated(t *testing.T) {
	ctx, svc, client, catalogID := setupItemTest(t)

	t.Run("不存在的 sla_id 拒绝", func(t *testing.T) {
		req := &dto.CreateServiceCatalogItemRequest{
			CatalogID:        catalogID,
			Name:             "坏 SLA 绑定",
			RequiresApproval: false,
			SlaID:            99999,
		}
		_, err := svc.CreateServiceCatalogItem(ctx, req, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sla definition not found")
	})

	t.Run("他租户的 sla_id 拒绝（租户隔离）", func(t *testing.T) {
		foreign := client.SLADefinition.Create().
			SetName("他租户SLA").
			SetTenantID(2).
			SaveX(ctx)
		req := &dto.CreateServiceCatalogItemRequest{
			CatalogID: catalogID,
			Name:      "越权 SLA 绑定",
			SlaID:     foreign.ID,
		}
		_, err := svc.CreateServiceCatalogItem(ctx, req, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sla definition not found")
	})

	t.Run("本租户合法 sla_id 通过", func(t *testing.T) {
		sla := client.SLADefinition.Create().
			SetName("本租户SLA").
			SetTenantID(1).
			SaveX(ctx)
		req := &dto.CreateServiceCatalogItemRequest{
			CatalogID: catalogID,
			Name:      "合法 SLA 绑定",
			SlaID:     sla.ID,
		}
		resp, err := svc.CreateServiceCatalogItem(ctx, req, 1)
		require.NoError(t, err)
		assert.Equal(t, sla.ID, resp.SlaID)
	})
}

// TestUpdateServiceCatalogItem_BindingValidationSymmetric 锁定 Create/Update
// 校验对称化：Update 与 Create 共用同一绑定校验器，且按「请求 ∪ 现值」的
// 生效状态判定——不再出现「创建必填、更新可非法」的口径漂移。
func TestUpdateServiceCatalogItem_BindingValidationSymmetric(t *testing.T) {
	ctx, svc, client, catalogID := setupItemTest(t)

	sla := client.SLADefinition.Create().SetName("SLA-A").SetTenantID(1).SaveX(ctx)
	chain := client.ApprovalChain.Create().
		SetName("链-A").SetEntityType("service_request").SetStatus("active").SetTenantID(1).
		SetChain([]schema.ApprovalChainStep{{Level: 1, Role: "manager", IsRequired: true}}).
		SaveX(ctx)

	newItem := func(t *testing.T, name string) int {
		req := &dto.CreateServiceCatalogItemRequest{
			CatalogID:        catalogID,
			Name:             name,
			RequiresApproval: false,
		}
		resp, err := svc.CreateServiceCatalogItem(ctx, req, 1)
		require.NoError(t, err)
		return resp.ID
	}

	t.Run("不存在的 sla_id 拒绝（此前 Update 可改成任意整数）", func(t *testing.T) {
		id := newItem(t, "改坏 SLA")
		_, err := svc.UpdateServiceCatalogItem(ctx, id, &dto.UpdateServiceCatalogItemRequest{SlaID: itemPtrInt(99999)}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sla definition not found")
	})

	t.Run("不存在的审批链拒绝（此前 Update 可改成任意整数）", func(t *testing.T) {
		id := newItem(t, "改坏审批链")
		_, err := svc.UpdateServiceCatalogItem(ctx, id, &dto.UpdateServiceCatalogItemRequest{ApprovalChainID: itemPtrInt(99999)}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "approval chain not found")
	})

	t.Run("requires_approval 置 true 但生效审批链为空拒绝（对称 Create）", func(t *testing.T) {
		id := newItem(t, "开审批没链")
		_, err := svc.UpdateServiceCatalogItem(ctx, id, &dto.UpdateServiceCatalogItemRequest{RequiresApproval: itemPtrBool(true)}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "approval chain is required")
	})

	t.Run("未携带审批链但现值合法 → 生效值语义通过", func(t *testing.T) {
		id := newItem(t, "沿用现值")
		// 先把合法审批链与 requires_approval 一次性写入
		_, err := svc.UpdateServiceCatalogItem(ctx, id, &dto.UpdateServiceCatalogItemRequest{
			RequiresApproval: itemPtrBool(true),
			ApprovalChainID:  itemPtrInt(chain.ID),
		}, 1)
		require.NoError(t, err)
		// 再只改名字：不带审批链字段，生效值沿用合法现值，不得误报
		_, err = svc.UpdateServiceCatalogItem(ctx, id, &dto.UpdateServiceCatalogItemRequest{Name: itemPtrString("改名后")}, 1)
		require.NoError(t, err)
	})

	t.Run("合法 sla_id 更新通过", func(t *testing.T) {
		id := newItem(t, "绑合法 SLA")
		_, err := svc.UpdateServiceCatalogItem(ctx, id, &dto.UpdateServiceCatalogItemRequest{SlaID: itemPtrInt(sla.ID)}, 1)
		require.NoError(t, err)
	})
}

func itemPtrInt(i int) *int          { return &i }
func itemPtrBool(b bool) *bool       { return &b }
func itemPtrString(s string) *string { return &s }
