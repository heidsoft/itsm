package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent/enttest"
)

type variableFixture struct {
	svc      *BPMNVariableService
	client   interface{ Close() error }
	tenantID int
	instID   int
	defID    int
}

func setupVariableFixture(t *testing.T) variableFixture {
	t.Helper()
	client := enttest.Open(t, dialect.SQLite, testDSN())
	logger := zaptest.NewLogger(t).Sugar()
	svc := &BPMNVariableService{client: client, logger: logger}
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Var Tenant").SetCode("var-test").SetDomain("var.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-VAR").SetDeploymentName("Var Deploy").
		SetIsActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey("var_test").SetName("变量测试").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).SetBpmnXML([]byte("<bpmn/>")).
		SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	inst, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-VAR-1").
		SetProcessDefinitionKey("var_test").SetProcessDefinitionID(def.ID).
		SetStatus("running").SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return variableFixture{svc: svc, client: client, tenantID: tenant.ID, instID: inst.ID, defID: def.ID}
}

func TestBPMNVariable_CreateAndGet(t *testing.T) {
	f := setupVariableFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	got, err := f.svc.CreateVariable(ctx, &CreateVariableRequest{
		Name:              "priority",
		Value:             "high",
		Type:              TypeString,
		Scope:             ScopeInstance,
		ProcessInstanceID: "1",
		TenantID:          f.tenantID,
	})
	require.NoError(t, err)
	assert.Equal(t, "priority", got.VariableName)
	assert.Equal(t, `"high"`, got.VariableValue)

	loaded, err := f.svc.GetVariable(ctx, got.VariableID, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, got.VariableID, loaded.VariableID)
}

func TestBPMNVariable_CreateValidationReject(t *testing.T) {
	f := setupVariableFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	_, err := f.svc.CreateVariable(ctx, &CreateVariableRequest{
		Name:              "bad",
		Value:             "not-a-number",
		Type:              TypeInteger,
		Scope:             ScopeInstance,
		ProcessInstanceID: "1",
		TenantID:          f.tenantID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "变量值验证")

	_, err = f.svc.CreateVariable(ctx, &CreateVariableRequest{
		Name:              "nil_val",
		Value:             nil,
		Type:              TypeString,
		Scope:             ScopeInstance,
		ProcessInstanceID: "1",
		TenantID:          f.tenantID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "变量值不能为空")
}

func TestBPMNVariable_Update(t *testing.T) {
	f := setupVariableFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	created, err := f.svc.CreateVariable(ctx, &CreateVariableRequest{
		Name:              "count",
		Value:             1,
		Type:              TypeInteger,
		Scope:             ScopeInstance,
		ProcessInstanceID: "1",
		TenantID:          f.tenantID,
	})
	require.NoError(t, err)

	newVal := true
	updated, err := f.svc.UpdateVariable(ctx, created.VariableID, &UpdateVariableRequest{
		Value:       42,
		IsTransient: &newVal,
	}, f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, "42", updated.VariableValue)
	assert.True(t, updated.IsTransient)
}

func TestBPMNVariable_Delete(t *testing.T) {
	f := setupVariableFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	created, err := f.svc.CreateVariable(ctx, &CreateVariableRequest{
		Name:              "temp",
		Value:             "x",
		Type:              TypeString,
		Scope:             ScopeInstance,
		ProcessInstanceID: "1",
		TenantID:          f.tenantID,
	})
	require.NoError(t, err)

	err = f.svc.DeleteVariable(ctx, created.VariableID, f.tenantID)
	require.NoError(t, err)

	_, err = f.svc.GetVariable(ctx, created.VariableID, f.tenantID)
	require.Error(t, err)
}

func TestBPMNVariable_TenantIsolation(t *testing.T) {
	f := setupVariableFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	created, err := f.svc.CreateVariable(ctx, &CreateVariableRequest{
		Name:              "secret",
		Value:             "tenant-data",
		Type:              TypeString,
		Scope:             ScopeInstance,
		ProcessInstanceID: "1",
		TenantID:          f.tenantID,
	})
	require.NoError(t, err)

	otherTenant := f.tenantID + 999
	_, err = f.svc.GetVariable(ctx, created.VariableID, otherTenant)
	require.Error(t, err, "跨租户不应能读取变量")

	vars, total, err := f.svc.ListVariables(ctx, &ListVariablesRequest{TenantID: otherTenant})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, vars)
}

func TestBPMNVariable_ListWithFiltersAndPagination(t *testing.T) {
	f := setupVariableFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := f.svc.CreateVariable(ctx, &CreateVariableRequest{
			Name:              "var",
			Value:             "v",
			Type:              TypeString,
			Scope:             ScopeInstance,
			ProcessInstanceID: "1",
			TenantID:          f.tenantID,
		})
		require.NoError(t, err)
	}

	vars, total, err := f.svc.ListVariables(ctx, &ListVariablesRequest{
		TenantID:  f.tenantID,
		Page:      1,
		PageSize:  3,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, vars, 3)

	vars2, total2, err := f.svc.ListVariables(ctx, &ListVariablesRequest{
		TenantID:  f.tenantID,
		Page:      2,
		PageSize:  3,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, total2)
	assert.Len(t, vars2, 2)
}

func TestBPMNVariable_SetAndGetVariableValue(t *testing.T) {
	f := setupVariableFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	err := f.svc.SetVariable(ctx, "color", "blue", ScopeInstance, "1", f.tenantID)
	require.NoError(t, err)

	val, err := f.svc.GetVariableValue(ctx, "color", ScopeInstance, "1", f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, "blue", val)

	err = f.svc.SetVariable(ctx, "color", "red", ScopeInstance, "1", f.tenantID)
	require.NoError(t, err)

	val2, err := f.svc.GetVariableValue(ctx, "color", ScopeInstance, "1", f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, "red", val2, "SetVariable 应覆盖已有值")
}

func TestBPMNVariable_GetVariableValue_NotFound(t *testing.T) {
	f := setupVariableFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	_, err := f.svc.GetVariableValue(ctx, "nonexistent", ScopeInstance, "1", f.tenantID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestBPMNVariable_DeleteVariablesByScope(t *testing.T) {
	f := setupVariableFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	err := f.svc.SetVariable(ctx, "x1", "v1", ScopeInstance, "1", f.tenantID)
	require.NoError(t, err)
	err = f.svc.SetVariable(ctx, "x2", "v2", ScopeInstance, "1", f.tenantID)
	require.NoError(t, err)

	err = f.svc.DeleteVariablesByScope(ctx, ScopeInstance, "1", f.tenantID)
	require.NoError(t, err)

	vars, err := f.svc.GetVariablesByScope(ctx, ScopeInstance, "1", f.tenantID)
	require.NoError(t, err)
	assert.Empty(t, vars)
}

func TestBPMNVariable_CopyVariables(t *testing.T) {
	f := setupVariableFixture(t)
	defer f.client.Close()
	ctx := context.Background()

	err := f.svc.SetVariable(ctx, "a", "alpha", ScopeInstance, "1", f.tenantID)
	require.NoError(t, err)
	err = f.svc.SetVariable(ctx, "b", "beta", ScopeInstance, "1", f.tenantID)
	require.NoError(t, err)

	inst2, err := f.svc.client.ProcessInstance.Create().
		SetProcessInstanceID("PI-VAR-2").
		SetProcessDefinitionKey("var_test").
		SetProcessDefinitionID(f.defID).
		SetStatus("running").SetTenantID(f.tenantID).
		Save(ctx)
	require.NoError(t, err)

	err = f.svc.CopyVariables(ctx, ScopeInstance, "1", ScopeInstance, strconv.Itoa(inst2.ID), f.tenantID)
	require.NoError(t, err)

	copied, err := f.svc.GetVariablesByScope(ctx, ScopeInstance, strconv.Itoa(inst2.ID), f.tenantID)
	require.NoError(t, err)
	assert.Len(t, copied, 2)
}

func TestBPMNVariable_ValidateVariableValue_AllTypes(t *testing.T) {
	f := setupVariableFixture(t)
	defer f.client.Close()

	tests := []struct {
		name    string
		value   interface{}
		typ     VariableType
		wantErr bool
	}{
		{"string ok", "hello", TypeString, false},
		{"string reject int", 42, TypeString, true},
		{"integer ok", 42, TypeInteger, false},
		{"integer reject string", "42", TypeInteger, true},
		{"float ok", 3.14, TypeFloat, false},
		{"float reject int", 42, TypeFloat, true},
		{"boolean ok", true, TypeBoolean, false},
		{"boolean reject string", "true", TypeBoolean, true},
		{"datetime ok", time.Now(), TypeDateTime, false},
		{"datetime reject string", "2026-01-01", TypeDateTime, true},
		{"json ok", map[string]any{"k": "v"}, TypeJSON, false},
		{"binary ok", []byte{0x01}, TypeBinary, false},
		{"binary reject string", "bytes", TypeBinary, true},
		{"unknown type", "x", VariableType("unknown"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := f.svc.validateVariableValue(tt.value, tt.typ)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBPMNVariable_InferVariableType(t *testing.T) {
	f := setupVariableFixture(t)
	defer f.client.Close()

	tests := []struct {
		value interface{}
		want  VariableType
	}{
		{"hello", TypeString},
		{42, TypeInteger},
		{int64(100), TypeInteger},
		{3.14, TypeFloat},
		{float32(1.0), TypeFloat},
		{true, TypeBoolean},
		{time.Now(), TypeDateTime},
		{[]byte{0x01}, TypeBinary},
		{map[string]any{"k": "v"}, TypeJSON},
		{[]string{"a", "b"}, TypeJSON},
	}

	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			got := f.svc.inferVariableType(tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}
