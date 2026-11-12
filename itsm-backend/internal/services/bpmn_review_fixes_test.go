package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"
)

func seedTaskAuthFixture(t *testing.T) (*ent.Client, *CustomProcessEngine, *ent.User, *ent.User, *ent.ProcessTask, context.Context) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:bpmn_task_object_auth?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	tenant, err := client.Tenant.Create().
		SetName("T").SetCode("task-auth").SetDomain("task-auth.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	assignee, err := client.User.Create().
		SetUsername("assignee").SetEmail("assignee@test").SetName("Assignee").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	other, err := client.User.Create().
		SetUsername("other").SetEmail("other@test").SetName("Other").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-TASK-AUTH").SetDeploymentName("Task auth").SetIsActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	definition, err := client.ProcessDefinition.Create().
		SetKey("task_auth").SetName("Task auth").SetVersion("1").SetIsLatest(true).SetIsActive(true).
		SetBpmnXML([]byte("<bpmn/>")).SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-TASK-AUTH").SetProcessDefinitionKey(definition.Key).
		SetProcessDefinitionID(definition.ID).SetStatus("running").
		SetVariables(map[string]interface{}{}).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	task, err := client.ProcessTask.Create().
		SetTaskID("TASK-AUTH-1").SetTaskDefinitionKey("T").SetTaskName("T").
		SetProcessDefinitionKey(definition.Key).SetProcessInstanceID(instance.ID).
		SetStatus("assigned").SetAssignee(assignee.Username).SetTenantID(tenant.ID).
		SetTaskVariables(map[string]interface{}{}).Save(ctx)
	require.NoError(t, err)

	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)
	return client, engine, assignee, other, task, workflowCtx
}

func TestTaskObjectAuthorization_RequiresAssigneeCandidateOrAdmin(t *testing.T) {
	_, engine, assignee, other, task, workflowCtx := seedTaskAuthFixture(t)

	assigneeCtx := context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, assignee.ID)
	require.NoError(t, engine.TaskService().AuthorizeTaskAccess(assigneeCtx, task, false))
	require.NoError(t, engine.TaskService().AuthorizeTaskAccess(assigneeCtx, task, true))

	otherCtx := context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, other.ID)
	require.Error(t, engine.TaskService().AuthorizeTaskAccess(otherCtx, task, false))
	require.Error(t, engine.TaskService().AuthorizeTaskAccess(otherCtx, task, true))
}

func TestTaskObjectAuthorization_DelegatedActorCanReadButNotWrite(t *testing.T) {
	client, engine, assignee, delegatee, task, workflowCtx := seedTaskAuthFixture(t)
	task.Assignee = delegatee.Username
	task.TaskVariables = map[string]interface{}{"delegated_from": assignee.Username}
	_, err := client.ProcessTask.UpdateOneID(task.ID).SetAssignee(task.Assignee).SetTaskVariables(task.TaskVariables).Save(workflowCtx)
	require.NoError(t, err)

	actorCtx := context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, assignee.ID)
	require.NoError(t, engine.TaskService().AuthorizeTaskAccess(actorCtx, task, false))
	require.Error(t, engine.TaskService().AuthorizeTaskAccess(actorCtx, task, true))
}

func TestSetTaskVariables_MergesBusinessFieldsAndProtectsEngineMetadata(t *testing.T) {
	client, engine, assignee, _, task, workflowCtx := seedTaskAuthFixture(t)
	_, err := client.ProcessTask.UpdateOneID(task.ID).
		SetTaskVariables(map[string]interface{}{"taskPurpose": "approval", "custom": "old"}).
		Save(workflowCtx)
	require.NoError(t, err)
	actorCtx := context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, assignee.ID)

	err = engine.TaskService().SetTaskVariables(actorCtx, task.TaskID, map[string]interface{}{
		"taskPurpose": "work",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "不允许通过 API 覆盖")

	require.NoError(t, engine.TaskService().SetTaskVariables(actorCtx, task.TaskID, map[string]interface{}{
		"custom": "new", "businessNote": "ok",
	}))
	updated, err := client.ProcessTask.Get(workflowCtx, task.ID)
	require.NoError(t, err)
	require.Equal(t, "approval", updated.TaskVariables["taskPurpose"])
	require.Equal(t, "new", updated.TaskVariables["custom"])
	require.Equal(t, "ok", updated.TaskVariables["businessNote"])
}

func TestProcessInstanceVariables_PreservesEngineStateAndRejectsInternalKeys(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_instance_vars_cas?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)

	tenant, err := client.Tenant.Create().
		SetName("T").SetCode("instance-vars").SetDomain("instance-vars.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-VARS").SetDeploymentName("Vars").SetIsActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	definition, err := client.ProcessDefinition.Create().
		SetKey("instance_vars").SetName("Vars").SetVersion("1").SetIsLatest(true).SetIsActive(true).
		SetBpmnXML([]byte("<bpmn/>")).SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-VARS").SetProcessDefinitionKey(definition.Key).
		SetProcessDefinitionID(definition.ID).SetStatus("running").
		SetVariables(map[string]interface{}{"_done_": map[string]interface{}{"A": true}, "existing": "keep"}).
		SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)
	err = engine.ProcessInstanceService().SetProcessInstanceVariables(
		workflowCtx, instance.ProcessInstanceID, map[string]interface{}{"new": "value"},
	)
	require.NoError(t, err)

	updated, err := client.ProcessInstance.Get(workflowCtx, instance.ID)
	require.NoError(t, err)
	require.Equal(t, "keep", updated.Variables["existing"])
	require.Equal(t, "value", updated.Variables["new"])
	require.Contains(t, updated.Variables, "_done_")

	err = engine.ProcessInstanceService().SetProcessInstanceVariables(
		workflowCtx, instance.ProcessInstanceID,
		map[string]interface{}{"_done_": map[string]interface{}{"A": false}},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "不允许通过 API 覆盖")
	require.Equal(t, tenant.ID, updated.TenantID)
}
