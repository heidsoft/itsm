package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTicketTemplateService_CreateTemplate_WritesFieldDefinitions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_template_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewTicketTemplateService(client)

	fieldSvc := NewFieldDefinitionService(client)

	template, err := svc.CreateTemplate(ctx, &CreateTemplateRequest{
		Name:     "网络接入申请",
		Category: "网络",
		TenantID: 1,
		Fields: []FieldDefinitionInput{
			{Name: "office_location", Label: "办公地点", FieldType: "text", Required: true, SortOrder: 0},
			{Name: "device_count", Label: "设备数量", FieldType: "number", SortOrder: 1},
		},
	})
	require.NoError(t, err)

	defs, err := fieldSvc.ListDefinitions(ctx, 1, "ticket_template", template.ID)
	require.NoError(t, err)
	require.Len(t, defs, 2)
	assert.Equal(t, "office_location", defs[0].Name)
	assert.Equal(t, "device_count", defs[1].Name)
}

func TestTicketTemplateService_UpdateTemplate_ReplacesFieldDefinitions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_template_update_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewTicketTemplateService(client)
	fieldSvc := NewFieldDefinitionService(client)

	template, err := svc.CreateTemplate(ctx, &CreateTemplateRequest{
		Name: "模板", Category: "cat", TenantID: 1,
		Fields: []FieldDefinitionInput{{Name: "a", Label: "A", FieldType: "text"}},
	})
	require.NoError(t, err)

	_, err = svc.UpdateTemplate(ctx, template.ID, &UpdateTemplateRequest{
		Fields: []FieldDefinitionInput{{Name: "b", Label: "B", FieldType: "text"}},
	}, 1)
	require.NoError(t, err)

	defs, err := fieldSvc.ListDefinitions(ctx, 1, "ticket_template", template.ID)
	require.NoError(t, err)
	require.Len(t, defs, 1)
	assert.Equal(t, "b", defs[0].Name)
}

func TestTicketTemplateService_UpdateTemplate_NilFieldsPreservesExisting(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_template_nil_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewTicketTemplateService(client)
	fieldSvc := NewFieldDefinitionService(client)

	template, err := svc.CreateTemplate(ctx, &CreateTemplateRequest{
		Name: "模板", Category: "cat", TenantID: 1,
		Fields: []FieldDefinitionInput{{Name: "a", Label: "A", FieldType: "text"}},
	})
	require.NoError(t, err)

	// 只改状态，不碰 Fields——Fields 是 nil，应该被当成"不修改"，不是"清空"。
	isActive := false
	_, err = svc.UpdateTemplate(ctx, template.ID, &UpdateTemplateRequest{
		IsActive: &isActive,
	}, 1)
	require.NoError(t, err)

	defs, err := fieldSvc.ListDefinitions(ctx, 1, "ticket_template", template.ID)
	require.NoError(t, err)
	require.Len(t, defs, 1)
	assert.Equal(t, "a", defs[0].Name)
}

func TestTicketTemplateService_DeleteTemplate_DeletesFieldDefinitions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_template_delete_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewTicketTemplateService(client)
	fieldSvc := NewFieldDefinitionService(client)

	template, err := svc.CreateTemplate(ctx, &CreateTemplateRequest{
		Name: "模板", Category: "cat", TenantID: 1,
		Fields: []FieldDefinitionInput{{Name: "a", Label: "A", FieldType: "text"}},
	})
	require.NoError(t, err)
	require.NoError(t, svc.DeleteTemplate(ctx, template.ID, 1))

	defs, err := fieldSvc.ListDefinitions(ctx, 1, "ticket_template", template.ID)
	require.NoError(t, err)
	assert.Empty(t, defs)
}
