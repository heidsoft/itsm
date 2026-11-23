package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// P1-02 回归（codex 工作流引擎审计 2026-11-23）：
// 任务完成边界（PUT /bpmn/tasks/:id/complete → TaskService → engine.CompleteTask
// → completeTaskWithClient）必须拒绝引擎保留键（_ 前缀）、身份键 tenant_id 与
// 不可变业务归属 business_id/business_type/business_key。旧实现把客户端
// variables 全量合并进实例变量，当前任务操作者可写 _done_ 提前满足汇聚、
// 写 _serviceTaskCompletedOccurrences 跳过并行 ServiceTask——不需要实例变量
// 管理权限。被拒绝的请求必须零副作用：任务/实例/命令/审计全无变化。

const completionGuardBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  targetNamespace="http://test">
  <bpmn:process id="completion_guard" name="完成边界守卫" isExecutable="true">
    <bpmn:startEvent id="Start_1" name="开始"/>
    <bpmn:userTask id="ApprovalTask" name="审批" taskPurpose="approval"/>
    <bpmn:endEvent id="End_1" name="结束"/>
    <bpmn:sequenceFlow id="F_Start" sourceRef="Start_1" targetRef="ApprovalTask"/>
    <bpmn:sequenceFlow id="F_Approve" sourceRef="ApprovalTask" targetRef="End_1"/>
  </bpmn:process>
</bpmn:definitions>`

// completionGuardFixture 返回 (engine, workflowCtx, instanceID, taskID)。
// 实例变量模拟后端 TriggerProcess 启动形态（含同值 tenant_id 与业务归属）。
func completionGuardFixture(t *testing.T) (*CustomProcessEngine, context.Context, int, string) {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		"file:completion_guard_"+t.Name()+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("守卫租户").SetCode("guard").SetDomain("guard.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	approver, err := client.User.Create().
		SetUsername("guard-approver").SetEmail("guard-approver@test.com").SetName("审批人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-GUARD").SetDeploymentName("Guard Deploy").SetDeploymentTime(time.Now()).
		SetDeployedBy("test").SetIsActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey("completion_guard").SetName("完成边界守卫").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).SetBpmnXML([]byte(completionGuardBPMN)).
		SetDeploymentID(deployment.ID).SetDeployedAt(time.Now()).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-GUARD-1").
		SetProcessDefinitionKey(def.Key).SetProcessDefinitionID(def.ID).
		SetStatus("running").
		SetVariables(map[string]interface{}{
			"requester_id":  1,
			"business_type": "ticket",
			"business_id":   "1",
			"tenant_id":     tenant.ID,
			"existing":      "keep",
		}).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	task, err := client.ProcessTask.Create().
		SetTaskID("TASK-GUARD-1").
		SetTaskDefinitionKey("ApprovalTask").
		SetTaskName("审批").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetStatus("created").
		SetAssignee(approver.Username).
		SetTenantID(tenant.ID).
		SetTaskVariables(map[string]interface{}{"taskPurpose": "approval"}).
		Save(ctx)
	require.NoError(t, err)

	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, approver.ID)
	return engine, workflowCtx, instance.ID, task.TaskID
}

// completionGuardSnapshot 记录任务/实例/命令/审计基线，供拒绝后零副作用比对。
type completionGuardSnapshot struct {
	taskStatus      string
	taskVariables   map[string]interface{}
	instanceVars    map[string]interface{}
	instanceVersion int
	instanceStatus  string
	commandCount    int
	decisionCount   int
}

func snapshotCompletionGuardState(t *testing.T, engine *CustomProcessEngine, instanceID int, taskID string) completionGuardSnapshot {
	t.Helper()
	ctx := context.Background()
	task, err := engine.client.ProcessTask.Query().
		Where(processtask.TaskID(taskID)).Only(ctx)
	require.NoError(t, err)
	inst, err := engine.client.ProcessInstance.Get(ctx, instanceID)
	require.NoError(t, err)
	commands, err := engine.client.OperationalCommand.Query().Count(ctx)
	require.NoError(t, err)
	decisions, err := engine.client.ProcessApprovalDecision.Query().Count(ctx)
	require.NoError(t, err)
	return completionGuardSnapshot{
		taskStatus:      task.Status,
		taskVariables:   task.TaskVariables,
		instanceVars:    inst.Variables,
		instanceVersion: inst.Version,
		instanceStatus:  inst.Status,
		commandCount:    commands,
		decisionCount:   decisions,
	}
}

// TestCompleteTask_RejectsProtectedVariables 完成边界拒绝内部键/身份键/业务归属，
// 且任务、实例、命令、审批决策全无变化。
func TestCompleteTask_RejectsProtectedVariables(t *testing.T) {
	cases := []struct {
		name    string
		vars    map[string]interface{}
		wantMsg string
	}{
		{
			name:    "伪造 _done_ 汇聚标记",
			vars:    map[string]interface{}{"_done_": map[string]interface{}{"ApprovalTask": true}},
			wantMsg: "流程内部变量",
		},
		{
			name:    "伪造 ServiceTask 完成凭据",
			vars:    map[string]interface{}{"_serviceTaskCompletedOccurrences": map[string]interface{}{"Svc": 9}},
			wantMsg: "流程内部变量",
		},
		{
			name:    "改写租户身份",
			vars:    map[string]interface{}{"tenant_id": 999},
			wantMsg: "tenant_id",
		},
		{
			name:    "改写业务归属",
			vars:    map[string]interface{}{"business_id": "999"},
			wantMsg: "business_id",
		},
		{
			name:    "改写业务类型",
			vars:    map[string]interface{}{"business_type": "change"},
			wantMsg: "business_type",
		},
		{
			name:    "正常审批变量之外夹带保留键",
			vars:    map[string]interface{}{"approvalAction": "approve", "_done_": map[string]interface{}{"X": true}},
			wantMsg: "流程内部变量",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine, ctx, instanceID, taskID := completionGuardFixture(t)
			before := snapshotCompletionGuardState(t, engine, instanceID, taskID)

			err := engine.CompleteTask(ctx, taskID, tc.vars)
			require.Error(t, err, "保护键必须被拒绝")
			assert.Contains(t, err.Error(), tc.wantMsg)

			after := snapshotCompletionGuardState(t, engine, instanceID, taskID)
			assert.Equal(t, before, after, "被拒绝的完成请求必须零副作用（任务/实例/命令/审计全无变化）")
		})
	}
}

// TestCompleteTask_LegitVariablesStillComplete 正向对照：正常审批变量照常完成任务，
// 守卫不得误伤合法完成链路。
func TestCompleteTask_LegitVariablesStillComplete(t *testing.T) {
	engine, ctx, instanceID, taskID := completionGuardFixture(t)

	err := engine.CompleteTask(ctx, taskID, map[string]interface{}{
		"approvalAction":  "approve",
		"approvalResult":  "approved",
		"approvalComment": "同意",
	})
	require.NoError(t, err, "合法完成不得被守卫误伤")

	after := snapshotCompletionGuardState(t, engine, instanceID, taskID)
	assert.Equal(t, "completed", after.taskStatus)
	assert.Equal(t, "completed", after.instanceStatus)
	assert.Equal(t, 1, after.decisionCount, "审批决策必须落库")
	// 同值 tenant_id 副本保留，业务归属不被改写
	assert.Equal(t, "1", after.instanceVars["business_id"])
}

// TestStartProcess_RejectsConflictingTenantVariable 启动边界拒绝冲突租户变量，
// 被拒绝的启动不得留下实例行；同值副本（后端 TriggerProcess 形态）照常放行。
func TestStartProcess_RejectsConflictingTenantVariable(t *testing.T) {
	engine, ctx, _, _ := completionGuardFixture(t)
	tenantID, _ := ctx.Value(bpmn.BPMNTenantIDContextKey).(int)

	_, err := engine.StartProcess(ctx, "completion_guard", "bk-conflict",
		map[string]interface{}{"tenant_id": 999, "requester_id": 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "冲突")

	count, err := engine.client.ProcessInstance.Query().Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, count, "被拒绝的启动不得新增实例行（仅 fixture 基线 1 行）")

	// 同值副本放行
	inst, err := engine.StartProcess(ctx, "completion_guard", "bk-match",
		map[string]interface{}{"tenant_id": tenantID, "requester_id": 1})
	require.NoError(t, err)
	require.NotNil(t, inst)
	assert.Equal(t, "running", inst.Status)
}

// TestSetProcessInstanceVariables_RejectsTenantIdentity 变量 PATCH 拒绝身份键
// tenant_id（P1-01 注入面之一），引擎保留键维持既有拒绝。
func TestSetProcessInstanceVariables_RejectsTenantIdentity(t *testing.T) {
	engine, ctx, instanceID, _ := completionGuardFixture(t)
	inst, err := engine.client.ProcessInstance.Get(context.Background(), instanceID)
	require.NoError(t, err)

	before := snapshotCompletionGuardState(t, engine, instanceID, "TASK-GUARD-1")

	err = engine.ProcessInstanceService().SetProcessInstanceVariables(
		ctx, inst.ProcessInstanceID, map[string]interface{}{"tenant_id": 999})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant_id")

	after := snapshotCompletionGuardState(t, engine, instanceID, "TASK-GUARD-1")
	assert.Equal(t, before, after, "被拒绝的 PATCH 必须零副作用")
}
