package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldValueService_CreateAndListValues(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_value_create?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	defSvc := NewFieldDefinitionService(client)
	valSvc := NewFieldValueService(client)

	_, err := defSvc.ReplaceDefinitions(ctx, 1, "ticket_template", 4, []FieldDefinitionInput{
		{Name: "office_location", Label: "办公地点", FieldType: "text", SortOrder: 0},
		{Name: "device_count", Label: "设备数量", FieldType: "number", SortOrder: 1},
	})
	require.NoError(t, err)

	err = valSvc.CreateValues(ctx, 1, "ticket_template", 4, "ticket", 100, map[string]interface{}{
		"office_location": "北京",
		"device_count":    float64(2),
		"unknown_field":   "should be ignored",
	})
	require.NoError(t, err)

	values, err := valSvc.ListValues(ctx, 1, "ticket", 100)
	require.NoError(t, err)
	require.Len(t, values, 2) // unknown_field 被忽略，不匹配任何定义
	assert.Equal(t, "office_location", values[0].Name)
	assert.Equal(t, "办公地点", values[0].Label)
	assert.Equal(t, "北京", values[0].Value)
	assert.Equal(t, "device_count", values[1].Name)
	assert.Equal(t, float64(2), values[1].Value)
}

func TestFieldValueService_ListValues_EmptyWhenNoValues(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_value_empty?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	valSvc := NewFieldValueService(client)

	values, err := valSvc.ListValues(ctx, 1, "ticket", 999)
	require.NoError(t, err)
	assert.Empty(t, values)
}

func TestFieldValueService_CreateAdHocValues_NoFieldDefinitionRequired(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_value_adhoc?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	valSvc := NewFieldValueService(client)

	// 没有任何 field_definitions 行——静态预设场景。
	err := valSvc.CreateAdHocValues(ctx, 1, "ticket", 200, []AdHocFieldValue{
		{Name: "replicas", Label: "副本数", SortOrder: 0, Value: float64(3)},
	})
	require.NoError(t, err)

	values, err := valSvc.ListValues(ctx, 1, "ticket", 200)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, "replicas", values[0].Name)
	assert.Equal(t, "副本数", values[0].Label)
	assert.Equal(t, float64(3), values[0].Value)
}

func TestFieldValueService_CreateValues_SurvivesDefinitionDeletion(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_value_survive?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	defSvc := NewFieldDefinitionService(client)
	valSvc := NewFieldValueService(client)

	_, err := defSvc.ReplaceDefinitions(ctx, 1, "ticket_template", 4, []FieldDefinitionInput{
		{Name: "office_location", Label: "办公地点", FieldType: "text"},
	})
	require.NoError(t, err)
	require.NoError(t, valSvc.CreateValues(ctx, 1, "ticket_template", 4, "ticket", 100, map[string]interface{}{
		"office_location": "北京",
	}))

	// 模板字段定义被删除/改名后（这里模拟改名：Replace 成一个新 name）
	_, err = defSvc.ReplaceDefinitions(ctx, 1, "ticket_template", 4, []FieldDefinitionInput{
		{Name: "office_location_v2", Label: "办公地点(新)", FieldType: "text"},
	})
	require.NoError(t, err)

	// 老工单的历史值展示不受影响
	values, err := valSvc.ListValues(ctx, 1, "ticket", 100)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, "office_location", values[0].Name)
	assert.Equal(t, "办公地点", values[0].Label)
	assert.Equal(t, "北京", values[0].Value)
}
