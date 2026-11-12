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

func setupAddApproverFixture(t *testing.T, allowAdd bool) (*CustomProcessEngine, context.Context, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:addapprover_test_"+t.Name()+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("AddApprover Tenant").SetCode("addapprover").SetDomain("addapprover.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	approver, err := client.User.Create().
		SetUsername("approver").SetEmail("approver@test.com").SetName("审批人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	newApprover, err := client.User.Create().
		SetUsername("newapprover").SetEmail("newapprover@test.com").SetName("加签人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-ADDAPPROVER").SetDeploymentName("AddApprover Deploy").
		SetIsActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey("addapprover_test").SetName("加签测试").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).SetBpmnXML([]byte("<bpmn/>")).
		SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-ADDAPPROVER-1").
		SetProcessDefinitionKey(def.Key).SetProcessDefinitionID(def.ID).
		SetStatus("running").
		SetVariables(map[string]interface{}{"business_type": "ticket", "business_id": "1"}).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	taskVars := map[string]interface{}{
		"taskPurpose":      "approval",
		"allowAddApprover": allowAdd,
	}
	_, err = client.ProcessTask.Create().
		SetTaskID("TASK-ADDAPPROVER-1").
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

	_ = newApprover
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, approver.ID)

	return engine, workflowCtx, tenant.ID
}

func TestAddApproverTask_Allowed_CreatesChildTaskAndAudit(t *testing.T) {
	engine, workflowCtx, tenantID := setupAddApproverFixture(t, true)

	err := engine.TaskService().AddApproverTask(workflowCtx, "TASK-ADDAPPROVER-1", "newapprover")
	require.NoError(t, err)

	client := engine.client
	children, err := client.ProcessTask.Query().All(workflowCtx)
	require.NoError(t, err)
	require.Len(t, children, 2)

	var added *struct {
		taskID   string
		assignee string
	}
	for _, c := range children {
		if c.TaskID != "TASK-ADDAPPROVER-1" {
			added = &struct {
				taskID   string
				assignee string
			}{c.TaskID, c.Assignee}
			assert.Equal(t, "newapprover", c.Assignee)
			assert.Equal(t, "assigned", c.Status)
			assert.Equal(t, "approver", c.TaskVariables["added_by"])
			assert.Equal(t, "TASK-ADDAPPROVER-1", c.ParentTaskID)
		}
	}
	require.NotNil(t, added, "应创建加签子任务")

	decisions, err := client.ProcessApprovalDecision.Query().
		Where(processapprovaldecision.TenantID(tenantID)).
		All(workflowCtx)
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "add_approver", decisions[0].Action)
	assert.Equal(t, "added", decisions[0].Decision)
}

func TestAddApproverTask_NotAllowed_ReturnsError(t *testing.T) {
	engine, workflowCtx, _ := setupAddApproverFixture(t, false)

	err := engine.TaskService().AddApproverTask(workflowCtx, "TASK-ADDAPPROVER-1", "newapprover")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不允许加签")
}

func TestAddApproverTask_EmptyTarget_ReturnsError(t *testing.T) {
	engine, workflowCtx, _ := setupAddApproverFixture(t, true)

	err := engine.TaskService().AddApproverTask(workflowCtx, "TASK-ADDAPPROVER-1", "  ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "加签目标不能为空")
}

func TestAddApproverTaskByID_CreatesCorrectly(t *testing.T) {
	engine, workflowCtx, tenantID := setupAddApproverFixture(t, true)

	client := engine.client
	task, err := client.ProcessTask.Query().All(workflowCtx)
	require.NoError(t, err)
	require.Len(t, task, 1)
	pk := task[0].ID

	err = engine.TaskService().AddApproverTaskByID(workflowCtx, pk, "newapprover")
	require.NoError(t, err)

	allTasks, err := client.ProcessTask.Query().All(workflowCtx)
	require.NoError(t, err)
	assert.Len(t, allTasks, 2)

	decisions, err := client.ProcessApprovalDecision.Query().
		Where(processapprovaldecision.TenantID(tenantID)).
		All(workflowCtx)
	require.NoError(t, err)
	assert.Len(t, decisions, 1)
}
