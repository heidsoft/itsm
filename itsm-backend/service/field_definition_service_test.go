package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldDefinitionSchema_RoundTrip(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_definition_schema?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	created, err := client.FieldDefinition.Create().
		SetTenantID(1).
		SetEntityType("ticket_template").
		SetEntityID(4).
		SetName("office_location").
		SetLabel("办公地点").
		SetFieldType("text").
		SetRequired(true).
		SetSortOrder(0).
		Save(ctx)
	require.NoError(t, err)
	assert.Equal(t, "office_location", created.Name)

	fetched, err := client.FieldDefinition.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "办公地点", fetched.Label)
}

func TestFieldDefinitionService_ReplaceDefinitions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_definition_replace?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewFieldDefinitionService(client)

	defs, err := svc.ReplaceDefinitions(ctx, 1, "ticket_template", 4, []FieldDefinitionInput{
		{Name: "office_location", Label: "办公地点", FieldType: "text", Required: true, SortOrder: 0},
		{Name: "device_count", Label: "设备数量", FieldType: "number", Required: false, SortOrder: 1},
	})
	require.NoError(t, err)
	require.Len(t, defs, 2)
	assert.Equal(t, "office_location", defs[0].Name)
	assert.Equal(t, "device_count", defs[1].Name)

	// 再次 Replace 应该完全替换掉旧的，而不是追加
	defs2, err := svc.ReplaceDefinitions(ctx, 1, "ticket_template", 4, []FieldDefinitionInput{
		{Name: "device_count", Label: "设备数量", FieldType: "number", SortOrder: 0},
	})
	require.NoError(t, err)
	require.Len(t, defs2, 1)

	listed, err := svc.ListDefinitions(ctx, 1, "ticket_template", 4)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "device_count", listed[0].Name)
}

func TestFieldDefinitionService_TenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_definition_tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewFieldDefinitionService(client)

	_, err := svc.ReplaceDefinitions(ctx, 1, "ticket_template", 4, []FieldDefinitionInput{
		{Name: "office_location", Label: "办公地点", FieldType: "text"},
	})
	require.NoError(t, err)

	// 租户 2 查租户 1 的模板定义，必须查不到
	listed, err := svc.ListDefinitions(ctx, 2, "ticket_template", 4)
	require.NoError(t, err)
	assert.Empty(t, listed)
}

func TestFieldDefinitionService_ReplaceDefinitions_TransactionRollback(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_definition_rollback?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewFieldDefinitionService(client)

	// 初始化一个字段定义
	initial, err := svc.ReplaceDefinitions(ctx, 1, "ticket_template", 5, []FieldDefinitionInput{
		{Name: "original_field", Label: "原始字段", FieldType: "text", SortOrder: 0},
	})
	require.NoError(t, err)
	require.Len(t, initial, 1)
	assert.Equal(t, "original_field", initial[0].Name)

	// 尝试替换为包含重复名字的定义，应该失败
	_, err = svc.ReplaceDefinitions(ctx, 1, "ticket_template", 5, []FieldDefinitionInput{
		{Name: "new_field_a", Label: "新字段A", FieldType: "text", SortOrder: 0},
		{Name: "new_field_a", Label: "新字段A重复", FieldType: "text", SortOrder: 1}, // 重复的名字
	})
	require.Error(t, err)

	// 验证原始定义仍然存在（事务已回滚）
	listed, err := svc.ListDefinitions(ctx, 1, "ticket_template", 5)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "original_field", listed[0].Name)
	assert.Equal(t, "原始字段", listed[0].Label)
}
