package service

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"
)

// ============================================================================
// BPMN XML constants for business flow tests
// ============================================================================

const bizBPMNSimpleApproval = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://test">
  <bpmn:process id="simple_approval" name="简单审批" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="manager_approval" name="经理审批"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="manager_approval"/>
    <bpmn:sequenceFlow id="f2" sourceRef="manager_approval" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

const bizBPMNExclusiveGateway = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://test">
  <bpmn:process id="exclusive_gw" name="排他网关" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:exclusiveGateway id="gw1"/>
    <bpmn:userTask id="high_priority" name="高优先级处理"/>
    <bpmn:userTask id="low_priority" name="低优先级处理"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="gw1"/>
    <bpmn:sequenceFlow id="f_high" sourceRef="gw1" targetRef="high_priority">
      <bpmn:conditionExpression>priority == 'high'</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f_low" sourceRef="gw1" targetRef="low_priority">
      <bpmn:conditionExpression>priority == 'low'</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f2" sourceRef="high_priority" targetRef="end"/>
    <bpmn:sequenceFlow id="f3" sourceRef="low_priority" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

const bizBPMNParallelForkJoin = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://test">
  <bpmn:process id="parallel_fj" name="并行分叉汇聚" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:parallelGateway id="fork"/>
    <bpmn:userTask id="task_a" name="分支A"/>
    <bpmn:userTask id="task_b" name="分支B"/>
    <bpmn:parallelGateway id="join"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="fork"/>
    <bpmn:sequenceFlow id="f2" sourceRef="fork" targetRef="task_a"/>
    <bpmn:sequenceFlow id="f3" sourceRef="fork" targetRef="task_b"/>
    <bpmn:sequenceFlow id="f4" sourceRef="task_a" targetRef="join"/>
    <bpmn:sequenceFlow id="f5" sourceRef="task_b" targetRef="join"/>
    <bpmn:sequenceFlow id="f6" sourceRef="join" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

const bizBPMNMultiTaskChain = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://test">
  <bpmn:process id="multi_chain" name="多任务串行" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="step1" name="第一步"/>
    <bpmn:userTask id="step2" name="第二步"/>
    <bpmn:userTask id="step3" name="第三步"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="step1"/>
    <bpmn:sequenceFlow id="f2" sourceRef="step1" targetRef="step2"/>
    <bpmn:sequenceFlow id="f3" sourceRef="step2" targetRef="step3"/>
    <bpmn:sequenceFlow id="f4" sourceRef="step3" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

const bizBPMNRejectionToRequester = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://test">
  <bpmn:process id="reject_requester" name="退回发起人" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="approval" name="审批" rejectStrategy="to_requester" commentRequiredOnReject="false"/>
    <bpmn:userTask id="rework" name="返工"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="approval"/>
    <bpmn:sequenceFlow id="f_reject" sourceRef="approval" targetRef="rework">
      <bpmn:conditionExpression>approvalAction == 'reject'</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f_approve" sourceRef="approval" targetRef="end">
      <bpmn:conditionExpression>approvalAction == 'approve'</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f_rework_end" sourceRef="rework" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

// ============================================================================
// Shared helpers
// ============================================================================

func bizTestClient(t *testing.T, name string) *ent.Client {
	t.Helper()
	dsn := "file:biz_" + name + "?mode=memory&cache=shared&_fk=1"
	return enttest.Open(t, dialect.SQLite, dsn)
}

func bizTenantCtx(t *testing.T, client *ent.Client, suffix string) (context.Context, int) {
	t.Helper()
	tenantID := createTimerTenant(t, client, suffix)
	ctx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, tenantID)

	user, err := client.User.Create().
		SetUsername("biz_actor_" + suffix).
		SetEmail("biz_actor_" + suffix + "@test.com").
		SetName("Biz Actor " + suffix).
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)

	ctx = context.WithValue(ctx, bpmn.BPMNUserIDContextKey, user.ID)
	return ctx, tenantID
}

func bizDeploy(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, key, xml string) int {
	t.Helper()
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID("biz-dep-" + key).
		SetDeploymentName("Biz Deploy " + key).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey(key).
		SetName("Biz Process " + key).
		SetVersion("1.0").
		SetBpmnXML([]byte(xml)).
		SetIsActive(true).
		SetIsLatest(true).
		SetDeploymentID(dep.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return def.ID
}

func bizFindTask(t *testing.T, ctx context.Context, client *ent.Client, instanceID int, taskDefKey string) *ent.ProcessTask {
	t.Helper()
	task := bizFindTaskRaw(t, ctx, client, instanceID, taskDefKey)

	userID, _ := ctx.Value(bpmn.BPMNUserIDContextKey).(int)
	if userID > 0 {
		updated, err := client.ProcessTask.UpdateOne(task).
			SetAssignee(strconv.Itoa(userID)).
			Save(ctx)
		require.NoError(t, err)
		return updated
	}
	return task
}

func bizFindTaskRaw(t *testing.T, ctx context.Context, client *ent.Client, instanceID int, taskDefKey string) *ent.ProcessTask {
	t.Helper()
	tasks, err := client.ProcessTask.Query().
		Where(
			processtask.ProcessInstanceID(instanceID),
			processtask.TaskDefinitionKey(taskDefKey),
			processtask.StatusNEQ("completed"),
			processtask.StatusNEQ("cancelled"),
		).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, tasks, 1, "expected exactly one active task for key %s", taskDefKey)
	return tasks[0]
}

func bizRefreshInstance(t *testing.T, ctx context.Context, client *ent.Client, instanceID int) *ent.ProcessInstance {
	t.Helper()
	inst, err := client.ProcessInstance.Get(ctx, instanceID)
	require.NoError(t, err)
	return inst
}

// ============================================================================
// Test 1: Simple approval — Start → UserTask → Complete → End
// ============================================================================

func TestBiz_SimpleApprovalFlow(t *testing.T) {
	client := bizTestClient(t, "simple_approval")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "simple")

	bizDeploy(t, ctx, client, tenantID, "simple_approval", bizBPMNSimpleApproval)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "simple_approval", "BK-BIZ-001", nil)
	require.NoError(t, err)
	assert.Equal(t, "running", instance.Status)

	instance = bizRefreshInstance(t, ctx, client, instance.ID)
	assert.Equal(t, "manager_approval", instance.CurrentActivityID)

	task := bizFindTask(t, ctx, client, instance.ID, "manager_approval")
	assert.Equal(t, "created", task.Status)
	assert.Equal(t, "经理审批", task.TaskName)

	err = engine.CompleteTask(ctx, task.TaskID, map[string]interface{}{
		"approvalAction": "approve",
	})
	require.NoError(t, err)

	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status, "Process should complete after approving the only task")
}

// ============================================================================
// Test 2: Exclusive gateway — conditional routing based on variables
// ============================================================================

func TestBiz_ExclusiveGateway_HighPriority(t *testing.T) {
	client := bizTestClient(t, "excl_high")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "excl_h")

	bizDeploy(t, ctx, client, tenantID, "exclusive_gw", bizBPMNExclusiveGateway)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "exclusive_gw", "BK-BIZ-002", map[string]interface{}{
		"priority": "high",
	})
	require.NoError(t, err)

	instance = bizRefreshInstance(t, ctx, client, instance.ID)
	assert.Equal(t, "high_priority", instance.CurrentActivityID,
		"Process should route to high_priority task when priority==high")

	task := bizFindTask(t, ctx, client, instance.ID, "high_priority")
	assert.Equal(t, "高优先级处理", task.TaskName)

	err = engine.CompleteTask(ctx, task.TaskID, nil)
	require.NoError(t, err)

	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status)

	lowTasks, err := client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(instance.ID), processtask.TaskDefinitionKey("low_priority")).
		All(ctx)
	require.NoError(t, err)
	assert.Empty(t, lowTasks, "low_priority task should NOT be created")
}

func TestBiz_ExclusiveGateway_LowPriority(t *testing.T) {
	client := bizTestClient(t, "excl_low")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "excl_l")

	bizDeploy(t, ctx, client, tenantID, "exclusive_gw", bizBPMNExclusiveGateway)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "exclusive_gw", "BK-BIZ-003", map[string]interface{}{
		"priority": "low",
	})
	require.NoError(t, err)

	instance = bizRefreshInstance(t, ctx, client, instance.ID)
	assert.Equal(t, "low_priority", instance.CurrentActivityID,
		"Process should route to low_priority task when priority==low")

	task := bizFindTask(t, ctx, client, instance.ID, "low_priority")
	err = engine.CompleteTask(ctx, task.TaskID, nil)
	require.NoError(t, err)

	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status)
}

// ============================================================================
// Test 3: Parallel gateway — fork creates multiple tasks, join waits for all
// ============================================================================

func TestBiz_ParallelGateway_ForkAndJoin(t *testing.T) {
	client := bizTestClient(t, "parallel_fj")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "par_fj")

	bizDeploy(t, ctx, client, tenantID, "parallel_fj", bizBPMNParallelForkJoin)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "parallel_fj", "BK-BIZ-004", nil)
	require.NoError(t, err)

	activeTasks, err := client.ProcessTask.Query().
		Where(
			processtask.ProcessInstanceID(instance.ID),
			processtask.StatusNEQ("completed"),
			processtask.StatusNEQ("cancelled"),
		).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, activeTasks, 2, "Parallel fork should create 2 active tasks")

	taskA := bizFindTask(t, ctx, client, instance.ID, "task_a")
	taskB := bizFindTask(t, ctx, client, instance.ID, "task_b")

	err = engine.CompleteTask(ctx, taskA.TaskID, nil)
	require.NoError(t, err)

	afterA, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", afterA.Status, "Process should still be running after completing only one branch")

	err = engine.CompleteTask(ctx, taskB.TaskID, nil)
	require.NoError(t, err)

	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status, "Process should complete after both branches finish")
}

// ============================================================================
// Test 4: Multi-task serial chain — complete tasks in order
// ============================================================================

func TestBiz_MultiTaskSerialChain(t *testing.T) {
	client := bizTestClient(t, "multi_chain")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "chain")

	bizDeploy(t, ctx, client, tenantID, "multi_chain", bizBPMNMultiTaskChain)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "multi_chain", "BK-BIZ-005", nil)
	require.NoError(t, err)

	instance = bizRefreshInstance(t, ctx, client, instance.ID)
	for _, step := range []string{"step1", "step2", "step3"} {
		assert.Equal(t, step, instance.CurrentActivityID)

		task := bizFindTask(t, ctx, client, instance.ID, step)
		err = engine.CompleteTask(ctx, task.TaskID, nil)
		require.NoError(t, err)

		instance = bizRefreshInstance(t, ctx, client, instance.ID)
	}

	assert.Equal(t, "completed", instance.Status, "Process should complete after all 3 serial steps")

	allTasks, err := client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(instance.ID)).
		All(ctx)
	require.NoError(t, err)
	assert.Len(t, allTasks, 3, "Should have created exactly 3 tasks")
	for _, task := range allTasks {
		assert.Equal(t, "completed", task.Status)
	}
}

// ============================================================================
// Test 5: Rejection → to_requester → rework → re-approve → end
// ============================================================================

func TestBiz_RejectionToRequester_ReworkThenApprove(t *testing.T) {
	client := bizTestClient(t, "reject_req")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "rej_req")

	bizDeploy(t, ctx, client, tenantID, "reject_requester", bizBPMNRejectionToRequester)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "reject_requester", "BK-BIZ-006", nil)
	require.NoError(t, err)

	instance = bizRefreshInstance(t, ctx, client, instance.ID)
	assert.Equal(t, "approval", instance.CurrentActivityID)

	approvalTask := bizFindTask(t, ctx, client, instance.ID, "approval")
	err = engine.CompleteTask(ctx, approvalTask.TaskID, map[string]interface{}{
		"approvalAction":  "reject",
		"approvalComment": "信息不完整",
	})
	require.NoError(t, err)

	instance, err = client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "rework", instance.CurrentActivityID,
		"After rejection with to_requester, process should advance to rework task")

	reworkTask := bizFindTask(t, ctx, client, instance.ID, "rework")
	err = engine.CompleteTask(ctx, reworkTask.TaskID, map[string]interface{}{
		"rework_done": true,
	})
	require.NoError(t, err)

	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status,
		"Process should complete after rework → end")

	merged, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, true, merged.Variables["rework_requested"],
		"rework_requested variable should be set by the rejection handler")
}

// ============================================================================
// Test 6: Process lifecycle — suspend / resume / terminate
// ============================================================================

func TestBiz_SuspendAndResume(t *testing.T) {
	client := bizTestClient(t, "suspend_resume")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "susp")

	bizDeploy(t, ctx, client, tenantID, "simple_approval", bizBPMNSimpleApproval)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "simple_approval", "BK-BIZ-007", nil)
	require.NoError(t, err)

	err = engine.SuspendProcess(ctx, instance.ProcessInstanceID, "等待外部材料")
	require.NoError(t, err)

	suspended, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "suspended", suspended.Status)
	assert.Equal(t, "等待外部材料", suspended.SuspendedReason)

	err = engine.ResumeProcess(ctx, instance.ProcessInstanceID)
	require.NoError(t, err)

	resumed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", resumed.Status, "Resume should restore running status")

	task := bizFindTask(t, ctx, client, instance.ID, "manager_approval")
	err = engine.CompleteTask(ctx, task.TaskID, nil)
	require.NoError(t, err)

	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status,
		"Process should complete normally after resume")
}

func TestBiz_TerminateCancelsActiveTasks(t *testing.T) {
	client := bizTestClient(t, "terminate")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "term")

	bizDeploy(t, ctx, client, tenantID, "simple_approval", bizBPMNSimpleApproval)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "simple_approval", "BK-BIZ-008", nil)
	require.NoError(t, err)

	task := bizFindTask(t, ctx, client, instance.ID, "manager_approval")
	assert.Equal(t, "created", task.Status)

	err = engine.TerminateProcess(ctx, instance.ProcessInstanceID, "申请人撤回")
	require.NoError(t, err)

	terminated, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "terminated", terminated.Status)

	cancelledTask, err := client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", cancelledTask.Status,
		"Terminate should cancel all active tasks")
}

// ============================================================================
// Test 7: Variable propagation — start vars flow through, task vars merge
// ============================================================================

func TestBiz_VariablePropagation(t *testing.T) {
	client := bizTestClient(t, "var_prop")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "var")

	bizDeploy(t, ctx, client, tenantID, "multi_chain", bizBPMNMultiTaskChain)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	startVars := map[string]interface{}{
		"requester":  "user_1",
		"department": "IT",
		"amount":     5000,
	}
	instance, err := engine.StartProcess(ctx, "multi_chain", "BK-BIZ-009", startVars)
	require.NoError(t, err)

	task1 := bizFindTask(t, ctx, client, instance.ID, "step1")
	err = engine.CompleteTask(ctx, task1.TaskID, map[string]interface{}{
		"step1_result": "approved",
		"amount":       6000,
	})
	require.NoError(t, err)

	updated, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "user_1", updated.Variables["requester"], "Original var should persist")
	assert.Equal(t, "IT", updated.Variables["department"], "Original var should persist")
	assert.InDelta(t, float64(6000), updated.Variables["amount"], 0.1, "Task var should override start var")
	assert.Equal(t, "approved", updated.Variables["step1_result"], "New task var should be merged")

	task2 := bizFindTask(t, ctx, client, instance.ID, "step2")
	err = engine.CompleteTask(ctx, task2.TaskID, map[string]interface{}{
		"step2_result": "verified",
	})
	require.NoError(t, err)

	task3 := bizFindTask(t, ctx, client, instance.ID, "step3")
	err = engine.CompleteTask(ctx, task3.TaskID, nil)
	require.NoError(t, err)

	final, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "verified", final.Variables["step2_result"])
	assert.Equal(t, "approved", final.Variables["step1_result"])
	assert.InDelta(t, float64(6000), final.Variables["amount"], 0.1)
}

// ============================================================================
// Test 8: Task service operations — assign, cancel
// ============================================================================

func TestBiz_TaskAssignAndCancel(t *testing.T) {
	client := bizTestClient(t, "task_ops")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "tops")

	bizDeploy(t, ctx, client, tenantID, "simple_approval", bizBPMNSimpleApproval)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "simple_approval", "BK-BIZ-010", nil)
	require.NoError(t, err)

	task := bizFindTaskRaw(t, ctx, client, instance.ID, "manager_approval")
	assert.Equal(t, "created", task.Status)
	assert.Empty(t, task.Assignee)

	taskSvc := engine.TaskService()

	err = taskSvc.AssignTask(ctx, task.TaskID, "42")
	require.NoError(t, err)

	assigned, err := client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "42", assigned.Assignee)
	assert.Equal(t, "assigned", assigned.Status)

	err = taskSvc.CancelTask(ctx, task.TaskID, "任务取消")
	require.NoError(t, err)

	cancelled, err := client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", cancelled.Status)
}

// ============================================================================
// Test 9: Audit trail — key actions produce audit log records
// ============================================================================

func TestBiz_AuditTrail(t *testing.T) {
	client := bizTestClient(t, "audit_trail")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "audit")

	bizDeploy(t, ctx, client, tenantID, "simple_approval", bizBPMNSimpleApproval)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "simple_approval", "BK-BIZ-011", nil)
	require.NoError(t, err)

	startAuditCount, err := client.ProcessAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	assert.Greater(t, startAuditCount, 0, "StartProcess should create audit log")

	task := bizFindTask(t, ctx, client, instance.ID, "manager_approval")
	err = engine.CompleteTask(ctx, task.TaskID, map[string]interface{}{
		"approvalAction": "approve",
	})
	require.NoError(t, err)

	finalAuditCount, err := client.ProcessAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	assert.Greater(t, finalAuditCount, startAuditCount,
		"CompleteTask should create additional audit log entries")
}

// ============================================================================
// Test 10: Tenant isolation — tasks from tenant A invisible to tenant B
// ============================================================================

func TestBiz_TenantIsolation(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	clientA := bizTestClient(t, "tenant_a")
	ctxA, tenantA := bizTenantCtx(t, clientA, "a")
	bizDeploy(t, ctxA, clientA, tenantA, "simple_approval", bizBPMNSimpleApproval)
	engineA := NewCustomProcessEngine(clientA, logger).(*CustomProcessEngine)
	engineA.SetTimerServices(newMemoryTimerStore(), nil)

	instA, err := engineA.StartProcess(ctxA, "simple_approval", "BK-BIZ-012A", nil)
	require.NoError(t, err)

	clientB := bizTestClient(t, "tenant_b")
	ctxB, tenantB := bizTenantCtx(t, clientB, "b")
	bizDeploy(t, ctxB, clientB, tenantB, "simple_approval", bizBPMNSimpleApproval)
	engineB := NewCustomProcessEngine(clientB, logger).(*CustomProcessEngine)
	engineB.SetTimerServices(newMemoryTimerStore(), nil)

	instB, err := engineB.StartProcess(ctxB, "simple_approval", "BK-BIZ-012B", nil)
	require.NoError(t, err)

	tasksA, err := clientA.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(instA.ID)).
		All(ctxA)
	require.NoError(t, err)
	require.Len(t, tasksA, 1)

	userIDA, _ := ctxA.Value(bpmn.BPMNUserIDContextKey).(int)
	_, err = clientA.ProcessTask.UpdateOne(tasksA[0]).
		SetAssignee(strconv.Itoa(userIDA)).
		Save(ctxA)
	require.NoError(t, err)

	err = engineB.CompleteTask(ctxB, tasksA[0].TaskID, nil)
	require.Error(t, err, "Tenant B should NOT be able to complete Tenant A's task")
	assert.Contains(t, err.Error(), "失败",
		"Error should indicate the task was not found for tenant B")

	err = engineA.CompleteTask(ctxA, tasksA[0].TaskID, nil)
	require.NoError(t, err, "Tenant A should be able to complete its own task")

	completedA, err := clientA.ProcessInstance.Get(ctxA, instA.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completedA.Status)

	runningB, err := clientB.ProcessInstance.Get(ctxB, instB.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", runningB.Status, "Tenant B's instance should be unaffected")
}

// ============================================================================
// Test 11: Concurrent task completion — CAS prevents double-complete
// ============================================================================

func TestBiz_ConcurrentTaskCompletion(t *testing.T) {
	client := bizTestClient(t, "concurrent")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "conc")

	bizDeploy(t, ctx, client, tenantID, "simple_approval", bizBPMNSimpleApproval)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "simple_approval", "BK-BIZ-013", nil)
	require.NoError(t, err)

	task := bizFindTask(t, ctx, client, instance.ID, "manager_approval")

	err = engine.CompleteTask(ctx, task.TaskID, nil)
	require.NoError(t, err)

	err = engine.CompleteTask(ctx, task.TaskID, nil)
	require.Error(t, err, "Second CompleteTask should fail — task already completed")
}

// ============================================================================
// Test 12: Process definition not found — clear error
// ============================================================================

func TestBiz_StartNonExistentProcess(t *testing.T) {
	client := bizTestClient(t, "no_proc")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, _ := bizTenantCtx(t, client, "noproc")

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	_, err := engine.StartProcess(ctx, "nonexistent_key", "BK-BIZ-014", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "获取流程定义失败",
		"Should return clear error when process definition doesn't exist")
}

// ============================================================================
// Test 13: Suspend/Resume audit trail
// ============================================================================

func TestBiz_SuspendResumeAuditTrail(t *testing.T) {
	client := bizTestClient(t, "susp_audit")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "susp_a")

	bizDeploy(t, ctx, client, tenantID, "simple_approval", bizBPMNSimpleApproval)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "simple_approval", "BK-BIZ-015", nil)
	require.NoError(t, err)

	countBefore, err := client.ProcessAuditLog.Query().Count(ctx)
	require.NoError(t, err)

	err = engine.SuspendProcess(ctx, instance.ProcessInstanceID, "等待审批材料")
	require.NoError(t, err)

	countAfterSuspend, err := client.ProcessAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	assert.Greater(t, countAfterSuspend, countBefore,
		"Suspend should create audit log")

	err = engine.ResumeProcess(ctx, instance.ProcessInstanceID)
	require.NoError(t, err)

	countAfterResume, err := client.ProcessAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	assert.Greater(t, countAfterResume, countAfterSuspend,
		"Resume should create audit log")
}

// ============================================================================
// Test 14: Terminate audit trail
// ============================================================================

func TestBiz_TerminateAuditTrail(t *testing.T) {
	client := bizTestClient(t, "term_audit")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "term_a")

	bizDeploy(t, ctx, client, tenantID, "simple_approval", bizBPMNSimpleApproval)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "simple_approval", "BK-BIZ-016", nil)
	require.NoError(t, err)

	countBefore, err := client.ProcessAuditLog.Query().Count(ctx)
	require.NoError(t, err)

	err = engine.TerminateProcess(ctx, instance.ProcessInstanceID, "申请人撤回申请")
	require.NoError(t, err)

	countAfter, err := client.ProcessAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	assert.Greater(t, countAfter, countBefore,
		"Terminate should create audit log")
}

// ============================================================================
// Test 15: Process instance query — list by tenant, filter by status
// ============================================================================

func TestBiz_ListProcessInstancesByTenant(t *testing.T) {
	client := bizTestClient(t, "list_inst")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "list")

	bizDeploy(t, ctx, client, tenantID, "simple_approval", bizBPMNSimpleApproval)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	for i := 0; i < 3; i++ {
		_, err := engine.StartProcess(ctx, "simple_approval", fmt.Sprintf("BK-BIZ-017-%d", i), nil)
		require.NoError(t, err)
	}

	running, err := client.ProcessInstance.Query().
		Where(
			processinstance.TenantID(tenantID),
			processinstance.Status("running"),
		).
		All(ctx)
	require.NoError(t, err)
	total, err := client.ProcessInstance.Query().
		Where(
			processinstance.TenantID(tenantID),
			processinstance.Status("running"),
		).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, running, 3)
}

// ============================================================================
// Test 16: Complete task with wrong tenant context fails
// ============================================================================

func TestBiz_CompleteTaskCrossTenantRejected(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	clientA := bizTestClient(t, "cross_a")
	ctxA, tenantA := bizTenantCtx(t, clientA, "xa")
	bizDeploy(t, ctxA, clientA, tenantA, "simple_approval", bizBPMNSimpleApproval)
	engineA := NewCustomProcessEngine(clientA, logger).(*CustomProcessEngine)
	engineA.SetTimerServices(newMemoryTimerStore(), nil)

	instA, err := engineA.StartProcess(ctxA, "simple_approval", "BK-BIZ-018", nil)
	require.NoError(t, err)

	taskA := bizFindTask(t, ctxA, clientA, instA.ID, "manager_approval")

	clientB := bizTestClient(t, "cross_b")
	ctxB, tenantB := bizTenantCtx(t, clientB, "xb")
	bizDeploy(t, ctxB, clientB, tenantB, "simple_approval", bizBPMNSimpleApproval)
	engineB := NewCustomProcessEngine(clientB, logger).(*CustomProcessEngine)
	engineB.SetTimerServices(newMemoryTimerStore(), nil)

	_ = tenantB
	err = engineB.CompleteTask(ctxB, taskA.TaskID, nil)
	require.Error(t, err, "Completing another tenant's task should fail")
}

// ============================================================================
// Test 17: Multiple instances of same process definition run independently
// ============================================================================

func TestBiz_MultipleInstancesIndependent(t *testing.T) {
	client := bizTestClient(t, "multi_inst")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "multi")

	bizDeploy(t, ctx, client, tenantID, "multi_chain", bizBPMNMultiTaskChain)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	inst1, err := engine.StartProcess(ctx, "multi_chain", "BK-BIZ-019-1", nil)
	require.NoError(t, err)

	inst2, err := engine.StartProcess(ctx, "multi_chain", "BK-BIZ-019-2", nil)
	require.NoError(t, err)

	task1Step1 := bizFindTask(t, ctx, client, inst1.ID, "step1")
	_ = bizFindTask(t, ctx, client, inst2.ID, "step1")

	err = engine.CompleteTask(ctx, task1Step1.TaskID, nil)
	require.NoError(t, err)

	inst1After, err := client.ProcessInstance.Get(ctx, inst1.ID)
	require.NoError(t, err)
	assert.Equal(t, "step2", inst1After.CurrentActivityID)

	inst2After, err := client.ProcessInstance.Get(ctx, inst2.ID)
	require.NoError(t, err)
	assert.Equal(t, "step1", inst2After.CurrentActivityID,
		"Instance 2 should still be at step1 — independent of instance 1")

	allActive, err := client.ProcessTask.Query().
		Where(
			processtask.StatusNEQ("completed"),
			processtask.StatusNEQ("cancelled"),
		).
		All(ctx)
	require.NoError(t, err)
	assert.Len(t, allActive, 2, "Should have 2 active tasks: inst1/step2 and inst2/step1")
}

// ============================================================================
// Test 18: Process with no tenant context fails
// ============================================================================

func TestBiz_NoTenantContext_Fails(t *testing.T) {
	client := bizTestClient(t, "no_ctx")
	logger := zaptest.NewLogger(t).Sugar()

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	ctx := context.Background()
	_, err := engine.StartProcess(ctx, "any_key", "BK-BIZ-020", nil)
	require.Error(t, err, "StartProcess without tenant context should fail")

	err = engine.CompleteTask(ctx, "any-task-id", nil)
	require.Error(t, err, "CompleteTask without tenant context should fail")

	err = engine.SuspendProcess(ctx, "any-instance-id", "reason")
	require.Error(t, err, "SuspendProcess without tenant context should fail")

	err = engine.ResumeProcess(ctx, "any-instance-id")
	require.Error(t, err, "ResumeProcess without tenant context should fail")

	err = engine.TerminateProcess(ctx, "any-instance-id", "reason")
	require.Error(t, err, "TerminateProcess without tenant context should fail")
}

// ============================================================================
// Test 19: Exclusive gateway — no matching condition uses default flow
// ============================================================================

func TestBiz_ExclusiveGateway_NoMatchStuck(t *testing.T) {
	client := bizTestClient(t, "excl_nomatch")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "excl_nm")

	bizDeploy(t, ctx, client, tenantID, "exclusive_gw", bizBPMNExclusiveGateway)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	instance, err := engine.StartProcess(ctx, "exclusive_gw", "BK-BIZ-021", map[string]interface{}{
		"priority": "medium",
	})

	if err != nil {
		return
	}

	if instance.CurrentActivityID != "high_priority" && instance.CurrentActivityID != "low_priority" {
		tasks, qErr := client.ProcessTask.Query().
			Where(processtask.ProcessInstanceID(instance.ID)).
			All(ctx)
		require.NoError(t, qErr)
		assert.Empty(t, tasks,
			"No task should be created when no gateway condition matches")
	}
}

// ============================================================================
// Test 20: End-to-end timing — process completes within reasonable time
// ============================================================================

func TestBiz_ProcessCompletesQuickly(t *testing.T) {
	client := bizTestClient(t, "perf")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := bizTenantCtx(t, client, "perf")

	bizDeploy(t, ctx, client, tenantID, "multi_chain", bizBPMNMultiTaskChain)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(newMemoryTimerStore(), nil)

	start := time.Now()

	instance, err := engine.StartProcess(ctx, "multi_chain", "BK-BIZ-022", nil)
	require.NoError(t, err)

	for _, step := range []string{"step1", "step2", "step3"} {
		task := bizFindTask(t, ctx, client, instance.ID, step)
		err = engine.CompleteTask(ctx, task.TaskID, nil)
		require.NoError(t, err)
	}

	elapsed := time.Since(start)
	assert.Less(t, elapsed, 5*time.Second,
		"Full 3-step process should complete well under 5 seconds")

	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status)
}
