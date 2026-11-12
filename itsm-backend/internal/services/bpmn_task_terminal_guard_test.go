package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processapprovaldecision"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"

	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTerminalGuardFixture 构造指定状态的任务委托/加签夹具。
// caseName 进入 DSN，保证同一测试函数内多次取夹具互不串库。
func setupTerminalGuardFixture(t *testing.T, caseName, status string, taskVars map[string]interface{}) (*CustomProcessEngine, context.Context, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:terminal_guard_"+caseName+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("TerminalGuard Tenant").SetCode("termguard" + caseName).SetDomain("termguard.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	approver, err := client.User.Create().
		SetUsername("approver").SetEmail("approver@test.com").SetName("审批人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.User.Create().
		SetUsername("delegatee").SetEmail("delegatee@test.com").SetName("被委托人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.User.Create().
		SetUsername("newapprover").SetEmail("newapprover@test.com").SetName("加签人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-TERM-" + caseName).SetDeploymentName("TerminalGuard Deploy").
		SetIsActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey("terminal_guard_test").SetName("终态守卫测试").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).SetBpmnXML([]byte("<bpmn/>")).
		SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-TERM-" + caseName).
		SetProcessDefinitionKey(def.Key).SetProcessDefinitionID(def.ID).
		SetStatus("running").
		SetVariables(map[string]interface{}{"business_type": "ticket", "business_id": "1"}).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ProcessTask.Create().
		SetTaskID("TASK-TERM-" + caseName).
		SetTaskDefinitionKey("ApprovalTask").
		SetTaskName("审批").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetStatus(status).
		SetAssignee(approver.Username).
		SetTenantID(tenant.ID).
		SetTaskVariables(taskVars).
		Save(ctx)
	require.NoError(t, err)

	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, approver.ID)

	return engine, workflowCtx, tenant.ID
}

// TestDelegateTask_RejectsTerminalStatuses 已结束（completed/cancelled/timeout）的
// 任务不得委托，也不得留下委托审计；任务状态与处理人必须保持原样。
func TestDelegateTask_RejectsTerminalStatuses(t *testing.T) {
	for _, status := range []string{"completed", "cancelled", "timeout"} {
		t.Run(status, func(t *testing.T) {
			engine, workflowCtx, tenantID := setupTerminalGuardFixture(t, status, status, map[string]interface{}{
				"taskPurpose":   "approval",
				"allowDelegate": true,
			})
			client := engine.client

			err := engine.TaskService().DelegateTask(workflowCtx, "TASK-TERM-"+status, "delegatee")
			require.ErrorContains(t, err, "任务已结束，不能委托")

			task, err := client.ProcessTask.Query().
				Where(processtask.TaskID("TASK-TERM-"+status), processtask.TenantID(tenantID)).
				First(workflowCtx)
			require.NoError(t, err)
			assert.Equal(t, status, task.Status, "终态不得被委托覆盖")
			assert.Equal(t, "approver", task.Assignee, "终态任务的负责人不得被改写")

			decisions, err := client.ProcessApprovalDecision.Query().
				Where(processapprovaldecision.TenantIDEQ(tenantID)).
				Count(workflowCtx)
			require.NoError(t, err)
			assert.Equal(t, 0, decisions, "拒绝的委托不得留下审计记录")
		})
	}
}

// TestDelegateTask_AllowsCreatedStatus 待办态（created）委托必须照常成功，
// 守卫不得误伤合法流转。
func TestDelegateTask_AllowsCreatedStatus(t *testing.T) {
	engine, workflowCtx, tenantID := setupTerminalGuardFixture(t, "created", "created", map[string]interface{}{
		"taskPurpose":   "approval",
		"allowDelegate": true,
	})
	client := engine.client

	require.NoError(t, engine.TaskService().DelegateTask(workflowCtx, "TASK-TERM-created", "delegatee"))

	task, err := client.ProcessTask.Query().
		Where(processtask.TaskID("TASK-TERM-created"), processtask.TenantID(tenantID)).
		First(workflowCtx)
	require.NoError(t, err)
	assert.Equal(t, "delegatee", task.Assignee)
	assert.Equal(t, "assigned", task.Status)
}

// TestAddApproverTask_RejectsTerminalStatuses 已结束的审批节点不得加签：
// 不得产生子任务，也不得留下加签审计。
func TestAddApproverTask_RejectsTerminalStatuses(t *testing.T) {
	for _, status := range []string{"completed", "cancelled", "timeout"} {
		t.Run(status, func(t *testing.T) {
			engine, workflowCtx, tenantID := setupTerminalGuardFixture(t, "add"+status, status, map[string]interface{}{
				"taskPurpose":      "approval",
				"allowAddApprover": true,
			})
			client := engine.client

			err := engine.TaskService().AddApproverTask(workflowCtx, "TASK-TERM-add"+status, "newapprover")
			require.ErrorContains(t, err, "任务已结束，不能加签")

			tasks, err := client.ProcessTask.Query().
				Where(processtask.TenantID(tenantID)).
				All(workflowCtx)
			require.NoError(t, err)
			require.Len(t, tasks, 1, "终态任务不得产生加签子任务")
			assert.Equal(t, status, tasks[0].Status)

			decisions, err := client.ProcessApprovalDecision.Query().
				Where(processapprovaldecision.TenantIDEQ(tenantID)).
				Count(workflowCtx)
			require.NoError(t, err)
			assert.Equal(t, 0, decisions, "拒绝的加签不得留下审计记录")
		})
	}
}
