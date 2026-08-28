package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"

	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 一条「UserTask -> 排他网关」的 BPMN；网关条件恒不成立且无 default 流，
// 用于验证 CompleteTask 推进失败时能整体回滚（P2 事务原子化）。
const atomicBrokenBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
                  id="Definitions_atomic" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_atomic" name="Atomic" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Start"/>
    <bpmn:userTask id="UserTask_1" name="Do Work"/>
    <bpmn:exclusiveGateway id="Gateway_1" name="Decision"/>
    <bpmn:endEvent id="EndEvent_1" name="End"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="UserTask_1"/>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="UserTask_1" targetRef="Gateway_1"/>
    <bpmn:sequenceFlow id="Flow_3" sourceRef="Gateway_1" targetRef="EndEvent_1">
      <bpmn:conditionExpression xsi:type="bpmn:tFormalExpression">approved == true</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
  </bpmn:process>
</bpmn:definitions>`

// 一条「UserTask -> 排他网关(default 流) -> End」的 BPMN，用于验证正常完成仍可用（回归）。
const atomicOKBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
                  id="Definitions_ok" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_ok" name="OK" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Start"/>
    <bpmn:userTask id="UserTask_1" name="Do Work"/>
    <bpmn:exclusiveGateway id="Gateway_1" name="Decision" default="Flow_3"/>
    <bpmn:endEvent id="EndEvent_1" name="End"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="UserTask_1"/>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="UserTask_1" targetRef="Gateway_1"/>
    <bpmn:sequenceFlow id="Flow_3" sourceRef="Gateway_1" targetRef="EndEvent_1"/>
  </bpmn:process>
</bpmn:definitions>`

func seedEngineFixture(t *testing.T, bpmnXML string) (*CustomProcessEngine, *ent.Client, context.Context, int, int, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", testDSN())
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	tenantID := 7
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-ENGINE").SetDeploymentName("engine").SetTenantID(tenantID).Save(context.Background())
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey("engineDemo").SetName("engine").SetBpmnXML([]byte(bpmnXML)).
		SetDeploymentID(dep.ID).SetTenantID(tenantID).SetIsActive(true).SetIsLatest(true).Save(context.Background())
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, tenantID)

	inst, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-ENGINE-1").SetProcessDefinitionKey("engineDemo").
		SetProcessDefinitionID(def.ID).SetTenantID(tenantID).SetStatus("running").
		SetVariables(map[string]interface{}{}).SetStartTime(time.Now()).Save(context.Background())
	require.NoError(t, err)

	task, err := client.ProcessTask.Create().
		SetTaskID("TASK-ENGINE-1").SetProcessInstanceID(inst.ID).
		SetProcessDefinitionKey("engineDemo").SetTaskDefinitionKey("UserTask_1").
		SetTaskName("Do Work").SetTaskType("user_task").SetStatus("created").SetTenantID(tenantID).Save(context.Background())
	require.NoError(t, err)

	return engine, client, ctx, tenantID, inst.ID, task.ID
}

// TestCompleteTask_AtomicRollbackOnFailure 验证推进失败时任务状态整体回滚，
// 不残留「已完成」标记、也不残留新创建的任务（P2 事务原子化）。
func TestCompleteTask_AtomicRollbackOnFailure(t *testing.T) {
	engine, client, ctx, _, instID, taskID := seedEngineFixture(t, atomicBrokenBPMN)

	err := engine.CompleteTask(ctx, "TASK-ENGINE-1", map[string]interface{}{})
	require.Error(t, err, "网关无匹配路径时应返回错误")

	// 原任务应回滚为未完成的 created（而非 completed）
	tk, err := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "created", tk.Status, "失败时任务不应被标记为 completed")

	// 不应残留新创建的任务（推进中 createUserTask 的产物应随事务回滚）
	all, err := client.ProcessTask.Query().Where(processtask.ProcessInstanceID(instID)).All(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1, "失败时不应创建额外任务")

	inst, err := client.ProcessInstance.Get(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, "running", inst.Status, "失败时流程实例不应结束")
}

// TestCompleteTask_SuccessCommits 验证正常完成路径仍按预期提交并结束流程（回归）。
func TestCompleteTask_SuccessCommits(t *testing.T) {
	engine, client, ctx, _, instID, taskID := seedEngineFixture(t, atomicOKBPMN)

	err := engine.CompleteTask(ctx, "TASK-ENGINE-1", map[string]interface{}{})
	require.NoError(t, err)

	tk, err := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", tk.Status, "成功时任务应标记为 completed")

	inst, err := client.ProcessInstance.Get(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, "completed", inst.Status, "成功时应到达 End 并结束流程")
}

// TestCompleteTaskByID_RejectsCrossTenantRead 验证按数据库 ID 完成任务时，
// 首次读取就按租户过滤，不会读取或完成其他租户的任务。
func TestCompleteTaskByID_RejectsCrossTenantRead(t *testing.T) {
	_, client, _, _, _, taskID := seedEngineFixture(t, atomicOKBPMN)
	service := &bpmnTaskService{client: client, logger: zaptest.NewLogger(t).Sugar()}
	otherTenantCtx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, 8)

	err := service.CompleteTaskByID(otherTenantCtx, taskID, map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "获取任务失败")

	task, getErr := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, getErr)
	assert.Equal(t, "created", task.Status)
}

// TestDetectStuckInstances 验证卡死检测（P3 可观测性）。
func TestDetectStuckInstances(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	tenantID := 9

	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-STUCK").SetDeploymentName("stuck").SetTenantID(tenantID).Save(context.Background())
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey("stuckDemo").SetName("stuck").SetBpmnXML([]byte("<def/>")).
		SetDeploymentID(dep.ID).SetTenantID(tenantID).SetIsActive(true).SetIsLatest(true).Save(context.Background())
	require.NoError(t, err)

	old := time.Now().Add(-72 * time.Hour)
	_, err = client.ProcessInstance.Create().
		SetProcessInstanceID("PI-STUCK-1").SetProcessDefinitionKey("stuckDemo").SetProcessDefinitionID(def.ID).
		SetTenantID(tenantID).SetStatus("running").SetVariables(map[string]interface{}{}).
		SetStartTime(old).Save(context.Background())
	require.NoError(t, err)

	_, err = client.ProcessInstance.Create().
		SetProcessInstanceID("PI-DONE-1").SetProcessDefinitionKey("stuckDemo").SetProcessDefinitionID(def.ID).
		SetTenantID(tenantID).SetStatus("completed").SetVariables(map[string]interface{}{}).
		SetStartTime(old).Save(context.Background())
	require.NoError(t, err)

	_, err = client.ProcessInstance.Create().
		SetProcessInstanceID("PI-RECENT-1").SetProcessDefinitionKey("stuckDemo").SetProcessDefinitionID(def.ID).
		SetTenantID(tenantID).SetStatus("running").SetVariables(map[string]interface{}{}).
		SetStartTime(time.Now()).Save(context.Background())
	require.NoError(t, err)

	stuck, err := engine.DetectStuckInstances(context.Background(), tenantID, time.Now().Add(-24*time.Hour))
	require.NoError(t, err)
	require.Len(t, stuck, 1, "仅应返回 72h 前启动且仍 running 的实例")
	assert.Equal(t, "PI-STUCK-1", stuck[0].ProcessInstanceID)
}
