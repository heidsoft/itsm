package service

import (
	"context"
	"fmt"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/auditlog"
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
		CustomFields: []dto.CustomFieldDefinition{
			{ID: "node", Name: "pacsNode", Label: "PACS 节点", Type: dto.CustomFieldTypeText, Required: true, Order: 0},
			{ID: "dept", Name: "department", Label: "影响科室", Type: dto.CustomFieldTypeSelect, Required: true, Order: 1, Options: []dto.CustomFieldOption{{Label: "急诊", Value: "er"}}},
		},
	}

	created, err := svc.CreateTicketType(ctx, req, tenant.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "incident", created.Code)
	assert.Equal(t, "事件工单", created.Name)
	require.Len(t, created.CustomFields, 2)
	assert.Equal(t, "pacsNode", created.CustomFields[0].Name)
	assert.Equal(t, "department", created.CustomFields[1].Name)

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

func TestValidateCustomFields(t *testing.T) {
	assert.Error(t, validateCustomFields([]dto.CustomFieldDefinition{{Name: "bad-name", Label: "Bad", Type: dto.CustomFieldTypeText}}))
	assert.Error(t, validateCustomFields([]dto.CustomFieldDefinition{{Name: "choice", Label: "Choice", Type: dto.CustomFieldTypeSelect}}))
	assert.Error(t, validateCustomFields([]dto.CustomFieldDefinition{{Name: "same", Label: "A", Type: dto.CustomFieldTypeText}, {Name: "same", Label: "B", Type: dto.CustomFieldTypeText}}))
	assert.NoError(t, validateCustomFields([]dto.CustomFieldDefinition{{Name: "validField", Label: "Valid", Type: dto.CustomFieldTypeText}}))
}

func TestTicketTypeServiceRejectsCrossTenantBinding(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	svc := NewTicketTypeService(client, zap.NewNop().Sugar())
	a, err := client.Tenant.Create().SetName("A").SetCode("bind-a").SetDomain("a.bind").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	b, err := client.Tenant.Create().SetName("B").SetCode("bind-b").SetDomain("b.bind").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	category, err := client.TicketCategory.Create().SetName("Foreign").SetCode("foreign").SetDescription("foreign").SetIsActive(true).SetTenantID(b.ID).Save(ctx)
	require.NoError(t, err)
	_, err = svc.CreateTicketType(ctx, &dto.CreateTicketTypeRequest{Code: "invalid_binding", Name: "Invalid", CategoryID: &category.ID}, a.ID, 1)
	assert.Error(t, err)
}

func TestTicketTypePresetInstallIsTenantScoped(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	svc := NewTicketTypeService(client, zap.NewNop().Sugar())
	tenant, err := client.Tenant.Create().SetName("Preset").SetCode("preset").SetDomain("preset.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	installed, err := svc.InstallPreset(ctx, "pacs-incident", &dto.InstallTicketTypePresetRequest{}, tenant.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, tenant.ID, installed.TenantID)
	assert.Equal(t, "pacs_incident", installed.Code)
	require.NotEmpty(t, installed.CustomFields)
	logs, err := client.AuditLog.Query().Where(auditlog.TenantIDEQ(tenant.ID), auditlog.ActionEQ("preset.install")).All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, 1, logs[0].UserID)
	assert.Contains(t, *logs[0].RequestBody, `"presetId":"pacs-incident"`)
}

func TestTicketTypeArchiveAndBindingChangesAreAudited(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	svc := NewTicketTypeService(client, zap.NewNop().Sugar())
	tenant, err := client.Tenant.Create().SetName("Audit").SetCode("audit-type").SetDomain("audit.type").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	deployment, err := client.ProcessDeployment.Create().SetDeploymentID("dep-audit-type").SetDeploymentName("Audit deployment").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().SetKey("audit_flow").SetName("Audit Flow").SetBpmnXML([]byte("<definitions/>")).SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)
	created, err := svc.CreateTicketType(ctx, &dto.CreateTicketTypeRequest{Code: "audited", Name: "Audited"}, tenant.ID, 21)
	require.NoError(t, err)
	key := "audit_flow"
	_, err = svc.UpdateTicketType(ctx, created.ID, &dto.UpdateTicketTypeRequest{WorkflowDefinitionKey: &key}, tenant.ID, 22)
	require.NoError(t, err)
	require.NoError(t, svc.DeleteTicketType(ctx, created.ID, tenant.ID, 23))
	logs, err := client.AuditLog.Query().Where(auditlog.TenantIDEQ(tenant.ID), auditlog.ResourceEQ("ticket_type:"+fmt.Sprint(created.ID))).All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 2)
	actions := []string{logs[0].Action, logs[1].Action}
	assert.ElementsMatch(t, []string{"binding.update", "archive"}, actions)
	archived, err := client.TicketType.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(23), archived.ArchivedBy)
}

func TestTicketTypeWorkflowBindingRequiresActiveTenantDefinition(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	svc := NewTicketTypeService(client, zap.NewNop().Sugar())
	tenant, err := client.Tenant.Create().SetName("Workflow").SetCode("workflow-binding").SetDomain("workflow.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	deployment, err := client.ProcessDeployment.Create().SetDeploymentID("dep-ticket-type").SetDeploymentName("Ticket type deployment").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().SetKey("pacs_flow").SetName("PACS Flow").SetBpmnXML([]byte("<definitions/>")).SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)
	created, err := svc.CreateTicketType(ctx, &dto.CreateTicketTypeRequest{Code: "workflow_bound", Name: "Workflow Bound", WorkflowDefinitionKey: "pacs_flow"}, tenant.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "pacs_flow", created.WorkflowDefinitionKey)
	_, err = svc.CreateTicketType(ctx, &dto.CreateTicketTypeRequest{Code: "missing_workflow", Name: "Missing", WorkflowDefinitionKey: "missing"}, tenant.ID, 1)
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
