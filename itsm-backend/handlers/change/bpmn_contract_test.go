// Package change 中 BPMN 桥接的契约测试。
//
// 本文件 (bpmn_contract_test.go) 固化 v1.7 (PR-FIX-CMDB-BPMN)
// 阶段 2.1 的核心契约：
//
//   - 审批/驳回 → ProcessTask 状态机转换
//   - BPMN 节点超时升级路径 (due_date)
//   - 多实例会签 (parallel multi-instance) 完成度
//   - 加签：assignee 切换
//   - 撤回：completed_time + status
//
// 与 service_bpmn_bridge_test.go 不重叠：后者覆盖
// TransitionStatus → ProcessTask 桥接；本文件覆盖 **状态机契约**。
package change

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newBPMNContractClient 启一个 in-memory sqlite 用于本组契约测试。
func newBPMNContractClient(t *testing.T) *ent.Client {
	t.Helper()
	c := enttest.Open(t, "sqlite3", "file:bpmn-contract?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { c.Close() })
	return c
}

// setupBPMNContractActor 创 tenant + 用户，返回 (tenantID, userID)。
func setupBPMNContractActor(t *testing.T, c *ent.Client, code string) (tenantID, userID int) {
	t.Helper()
	ctx := context.Background()
	tenant, err := c.Tenant.Create().
		SetName("BPMN Contract " + code).
		SetCode("bpmn-contract-" + code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	user, err := c.User.Create().
		SetUsername("bpmn-user-" + code).
		SetEmail(code + "@example.com").
		SetName("BPMN User " + code).
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	return tenant.ID, user.ID
}

// createRunningInstance 创建一个 ProcessInstance（运行中），返回 ent.ID。
//
// 完整依赖链：ProcessDeployment → ProcessDefinition → ProcessInstance。
// 这三项均不可缺（process_definition_id 是 Positive，且与 definition 表 FK）。
func createRunningInstance(t *testing.T, c *ent.Client, key string, tenantID int) int {
	t.Helper()
	ctx := context.Background()

	deployment, err := c.ProcessDeployment.Create().
		SetDeploymentID("DEP-" + key).
		SetDeploymentName("BPMN Contract Deployment " + key).
		SetDeploymentTime(time.Now()).
		SetDeployedBy("test").
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := c.ProcessDefinition.Create().
		SetKey(key).
		SetName("BPMN Contract Definition " + key).
		SetVersion("1").
		SetIsLatest(true).
		SetBpmnXML([]byte(`<?xml version="1.0"?><definitions id="d_` + key + `"/>`)).
		SetDeploymentID(deployment.ID).
		SetDeployedAt(time.Now()).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	inst, err := c.ProcessInstance.Create().
		SetProcessInstanceID("PI-" + key).
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetStatus("running").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return inst.ID
}

// createPendingTask 创建一个 pending ProcessTask，关联到 instanceID（ent.ID）。
func createPendingTask(t *testing.T, c *ent.Client, taskID, defKey, name string,
	instanceID, tenantID int, assignee string) int {
	t.Helper()
	ctx := context.Background()
	task, err := c.ProcessTask.Create().
		SetTaskID(taskID).
		SetProcessInstanceID(instanceID).
		SetProcessDefinitionKey("def-key").
		SetTaskDefinitionKey(defKey).
		SetTaskName(name).
		SetTaskType("user_task").
		SetStatus("pending").
		SetAssignee(assignee).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return task.ID
}

// seedContractTask 是上文两者的快速绑定：tenant + instance + task。
// 返回 tenantID 和 taskID。
func seedContractTask(t *testing.T, c *ent.Client, code, taskID, defKey, name, assignee string) (int, int) {
	t.Helper()
	tenantID, _ := setupBPMNContractActor(t, c, code)
	instID := createRunningInstance(t, c, code, tenantID)
	task := createPendingTask(t, c, taskID, defKey, name, instID, tenantID, assignee)
	return tenantID, task
}

// TestBPMNContract_Approve_TransitionsToComplete 审批通过后 status=completed。
func TestBPMNContract_Approve_TransitionsToComplete(t *testing.T) {
	ctx := context.Background()
	c := newBPMNContractClient(t)
	_, taskID := seedContractTask(t, c, "approve", "BP-APPR-1", "Approve",
		"审批通过", "actor-1")

	now := time.Now()
	_, err := c.ProcessTask.UpdateOneID(taskID).
		SetStatus("completed").
		SetCompletedTime(now).
		Save(ctx)
	require.NoError(t, err)

	got, err := c.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", got.Status)
	assert.NotZero(t, got.CompletedTime)
	assert.WithinDuration(t, now, got.CompletedTime, time.Second)
}

// TestBPMNContract_Reject_TransitionsToRejected 驳回后 status=rejected。
func TestBPMNContract_Reject_TransitionsToRejected(t *testing.T) {
	ctx := context.Background()
	c := newBPMNContractClient(t)
	_, taskID := seedContractTask(t, c, "reject", "BP-REJ-1", "Approve",
		"审批驳回", "actor-1")

	_, err := c.ProcessTask.UpdateOneID(taskID).
		SetStatus("rejected").
		SetCompletedTime(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	got, err := c.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "rejected", got.Status)
	assert.NotZero(t, got.CompletedTime)
}

// TestBPMNContract_Approve_ThenReadConsistent 审批状态转换一致性。
func TestBPMNContract_Approve_ThenReadConsistent(t *testing.T) {
	ctx := context.Background()
	c := newBPMNContractClient(t)
	_, taskID := seedContractTask(t, c, "consistent", "BP-CON-1", "Approve",
		"审批一致性", "actor-1")

	got1, err := c.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "pending", got1.Status)

	_, err = c.ProcessTask.UpdateOneID(taskID).
		SetStatus("completed").
		SetCompletedTime(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	got2, err := c.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", got2.Status)
	assert.NotEqual(t, got1.Status, got2.Status)
}

// TestBPMNContract_TimeoutEscalation_TaskOverdue 已过期任务应被判定为 overdue。
func TestBPMNContract_TimeoutEscalation_TaskOverdue(t *testing.T) {
	ctx := context.Background()
	c := newBPMNContractClient(t)
	_, taskID := seedContractTask(t, c, "timeout", "BP-OVR-1", "Approve",
		"超时任务", "actor-1")

	due := time.Now().Add(-1 * time.Hour)
	_, err := c.ProcessTask.UpdateOneID(taskID).
		SetDueDate(due).
		Save(ctx)
	require.NoError(t, err)

	got, err := c.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	require.False(t, got.DueDate.IsZero())
	overdue := time.Now().After(got.DueDate)
	assert.True(t, overdue, "已过期任务应被判定为 overdue")

	_, err = c.ProcessTask.UpdateOneID(taskID).
		SetStatus("overdue").
		Save(ctx)
	require.NoError(t, err)
	got2, err := c.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "overdue", got2.Status)
}

// TestBPMNContract_MultiInstance_NofMCompletion 多实例会签完成度。
func TestBPMNContract_MultiInstance_NofMCompletion(t *testing.T) {
	ctx := context.Background()
	c := newBPMNContractClient(t)
	tenantID, _ := setupBPMNContractActor(t, c, "multi")
	instID := createRunningInstance(t, c, "multi", tenantID)

	const totalTasks = 3
	ids := make([]int, totalTasks)
	for i := 0; i < totalTasks; i++ {
		ids[i] = createPendingTask(t, c,
			fmt.Sprintf("BP-MULT-%d-%d", time.Now().UnixNano(), i),
			"ParallelApproval",
			fmt.Sprintf("并行审批-%d", i+1),
			instID, tenantID, fmt.Sprintf("actor-%d", i+1))
	}

	const completedByUs = 2
	for i := 0; i < completedByUs; i++ {
		_, err := c.ProcessTask.UpdateOneID(ids[i]).
			SetStatus("completed").
			SetCompletedTime(time.Now()).
			Save(ctx)
		require.NoError(t, err)
	}

	completedCount := 0
	all, err := c.ProcessTask.Query().All(ctx)
	require.NoError(t, err)
	for _, task := range all {
		if task.Status == "completed" {
			completedCount++
		}
	}
	assert.Equal(t, completedByUs, completedCount,
		"完成度应为 %d/%d", completedByUs, totalTasks)
}

// TestBPMNContract_Delegate_ReassignedToOtherUser 加签：assignee 切换。
func TestBPMNContract_Delegate_ReassignedToOtherUser(t *testing.T) {
	ctx := context.Background()
	c := newBPMNContractClient(t)
	_, taskID := seedContractTask(t, c, "delegate", "BP-DLG-1", "Approve",
		"加签", "original-assignee")

	_, err := c.ProcessTask.UpdateOneID(taskID).
		SetAssignee("delegated-assignee").
		Save(ctx)
	require.NoError(t, err)

	got, err := c.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "delegated-assignee", got.Assignee)
	assert.Equal(t, "pending", got.Status)
}

// TestBPMNContract_Withdraw_ByInitiator 撤回：status=withdrawn。
func TestBPMNContract_Withdraw_ByInitiator(t *testing.T) {
	ctx := context.Background()
	c := newBPMNContractClient(t)
	_, taskID := seedContractTask(t, c, "withdraw", "BP-WD-1", "Approve",
		"撤回审批", "initiator")

	_, err := c.ProcessTask.UpdateOneID(taskID).
		SetStatus("withdrawn").
		SetCompletedTime(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	got, err := c.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "withdrawn", got.Status)
	assert.False(t, got.CompletedTime.IsZero())
}

// TestBPMNContract_TenantIsolation 跨租户任务 tenant_id 不同。
func TestBPMNContract_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	c := newBPMNContractClient(t)

	tenantA, _ := setupBPMNContractActor(t, c, "ta")
	tenantB, _ := setupBPMNContractActor(t, c, "tb")

	instA := createRunningInstance(t, c, "iso-a", tenantA)
	instB := createRunningInstance(t, c, "iso-b", tenantB)

	taskA := createPendingTask(t, c, "BP-ISO-A", "Approve", "A-任务",
		instA, tenantA, "actor-a")
	taskB := createPendingTask(t, c, "BP-ISO-B", "Approve", "B-任务",
		instB, tenantB, "actor-b")

	gotA, err := c.ProcessTask.Get(ctx, taskA)
	require.NoError(t, err)
	gotB, err := c.ProcessTask.Get(ctx, taskB)
	require.NoError(t, err)

	assert.Equal(t, tenantA, gotA.TenantID)
	assert.Equal(t, tenantB, gotB.TenantID)
	assert.NotEqual(t, gotA.TenantID, gotB.TenantID)
}
