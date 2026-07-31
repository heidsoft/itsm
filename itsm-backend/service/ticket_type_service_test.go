package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestTicketTypeService_CreateAndGet(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	svc := NewTicketTypeService(client, zap.NewNop().Sugar())
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	req := &dto.CreateTicketTypeRequest{
		Code:        "incident",
		Name:        "事件工单",
		Description: "IT 基础设施事件",
		Icon:        "alert",
		Color:       "#ff4d4f",
	}

	created, err := svc.CreateTicketType(ctx, req, tenant.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "incident", created.Code)
	assert.Equal(t, "事件工单", created.Name)

	// 同租户重复编码必须失败
	_, err = svc.CreateTicketType(ctx, req, tenant.ID, 1)
	assert.Error(t, err)

	got, err := svc.GetTicketType(ctx, created.ID, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)

	// 跨租户读取必须失败（租户隔离）
	_, err = svc.GetTicketType(ctx, created.ID, tenant.ID+1)
	assert.Error(t, err)
}

func TestIntPtrHelpers(t *testing.T) {
	assert.Nil(t, intPtr(0))
	if v := intPtr(7); assert.NotNil(t, v) {
		assert.Equal(t, 7, *v)
	}

	assert.Nil(t, intPtrFromInt64(0))
	if v := intPtrFromInt64(42); assert.NotNil(t, v) {
		assert.Equal(t, 42, *v)
	}

	assert.Nil(t, strPtrFromInt64(0))
	if s := strPtrFromInt64(9); assert.NotNil(t, s) {
		assert.Equal(t, "9", *s)
	}
}

func TestConvertApprovalChain(t *testing.T) {
	assert.Nil(t, convertApprovalChain(nil))

	items := []interface{}{
		map[string]interface{}{"level": 1, "approverType": "manager"},
	}
	chain := convertApprovalChain(items)
	require.Len(t, chain, 1)
	assert.Equal(t, 1, chain[0].Level)
}

func TestStructToMap(t *testing.T) {
	assert.Empty(t, structToMap(nil))

	m := structToMap(struct {
		Name string `json:"name"`
	}{Name: "incident"})
	assert.Equal(t, "incident", m["name"])
}

func TestToInterfaceSlice(t *testing.T) {
	out := toInterfaceSlice([]string{"a", "b"})
	require.Len(t, out, 2)
	assert.Equal(t, "a", out[0])
	assert.Equal(t, "b", out[1])

	assert.Empty(t, toInterfaceSlice([]int{}))
}
