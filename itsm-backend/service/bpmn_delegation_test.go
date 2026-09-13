package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processapprovaldecision"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"

	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDelegationFixture(t *testing.T, allowDelegate bool) (*CustomProcessEngine, context.Context, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:delegation_test_"+t.Name()+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Delegate Tenant").SetCode("delegate").SetDomain("delegate.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	approver, err := client.User.Create().
		SetUsername("approver").SetEmail("approver@test.com").SetName("审批人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	delegatee, err := client.User.Create().
		SetUsername("delegatee").SetEmail("delegatee@test.com").SetName("被委托人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-DELEGATE").SetDeploymentName("Delegate Deploy").
		SetIsActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey("delegate_test").SetName("委托测试").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).SetBpmnXML([]byte("<bpmn/>")).
		SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-DELEGATE-1").
		SetProcessDefinitionKey(def.Key).SetProcessDefinitionID(def.ID).
		SetStatus("running").
		SetVariables(map[string]interface{}{"business_type": "ticket", "business_id": "1"}).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	taskVars := map[string]interface{}{
		"taskPurpose":   "approval",
		"allowDelegate": allowDelegate,
	}
	_, err = client.ProcessTask.Create().
		SetTaskID("TASK-DELEGATE-1").
		SetTaskDefinitionKey("ApprovalTask").
		SetTaskName("审批").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetStatus("assigned").
		SetAssignee(approver.Username).
		SetTenantID(tenant.ID).
		SetTaskVariables(taskVars).
		Save(ctx)
	require.NoError(t, err)

	_ = delegatee
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, approver.ID)

	return engine, workflowCtx, tenant.ID
}

func TestDelegateTask_Allowed_WritesAuditRecord(t *testing.T) {
	engine, workflowCtx, tenantID := setupDelegationFixture(t, true)

	err := engine.TaskService().DelegateTask(workflowCtx, "TASK-DELEGATE-1", "delegatee")
	require.NoError(t, err)

	task, err := engine.TaskService().GetTask(workflowCtx, "TASK-DELEGATE-1")
	require.NoError(t, err)
	assert.Equal(t, "delegatee", task.Assignee)
	assert.Equal(t, "assigned", task.Status)
	assert.Equal(t, "approver", task.TaskVariables["delegated_from"])

	client := engine.client
	decisions, err := client.ProcessApprovalDecision.Query().
		Where(processapprovaldecision.TenantID(tenantID)).
		All(workflowCtx)
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "delegate", decisions[0].Action)
	assert.Equal(t, "delegated", decisions[0].Decision)
}

func TestDelegateTask_NotAllowed_ReturnsError(t *testing.T) {
	engine, workflowCtx, _ := setupDelegationFixture(t, false)

	err := engine.TaskService().DelegateTask(workflowCtx, "TASK-DELEGATE-1", "delegatee")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不允许委托")
}

func TestDelegateTask_EmptyTarget_ReturnsError(t *testing.T) {
	engine, workflowCtx, _ := setupDelegationFixture(t, true)

	err := engine.TaskService().DelegateTask(workflowCtx, "TASK-DELEGATE-1", "  ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "委托目标不能为空")
}

func TestDelegateTaskByID_DelegatesCorrectly(t *testing.T) {
	engine, workflowCtx, tenantID := setupDelegationFixture(t, true)

	client := engine.client
	task, err := client.ProcessTask.Query().All(workflowCtx)
	require.NoError(t, err)
	require.Len(t, task, 1)
	taskID := task[0].ID

	err = engine.TaskService().DelegateTaskByID(workflowCtx, taskID, "delegatee")
	require.NoError(t, err)

	updated, err := client.ProcessTask.Get(workflowCtx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "delegatee", updated.Assignee)
	assert.Equal(t, "assigned", updated.Status)

	decisions, err := client.ProcessApprovalDecision.Query().
		Where(processapprovaldecision.TenantID(tenantID)).
		All(workflowCtx)
	require.NoError(t, err)
	assert.Len(t, decisions, 1)
}
