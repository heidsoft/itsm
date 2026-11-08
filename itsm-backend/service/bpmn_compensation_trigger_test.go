package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// triggerFlowBPMN 故意不携带补偿结束事件：两个 ServiceTask 完成后流程正常
// 结束，但补偿范围已登记。这样显式触发入口才有可验证的待补偿目标。
const triggerFlowBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="Process_trigger_compensation" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1"/>
    <bpmn:serviceTask id="ServiceTask_1" implementation="test_comp_handler_a"/>
    <bpmn:serviceTask id="ServiceTask_2" implementation="test_comp_handler_b"/>
    <bpmn:endEvent id="EndEvent_1"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="ServiceTask_1"/>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="ServiceTask_1" targetRef="ServiceTask_2"/>
    <bpmn:sequenceFlow id="Flow_3" sourceRef="ServiceTask_2" targetRef="EndEvent_1"/>
  </bpmn:process>
</bpmn:definitions>`

func setupTriggerEngine(t *testing.T, client *ent.Client) (*CustomProcessEngine, *compensationHandler, *compensationHandler, int, context.Context) {
	t.Helper()
	tenantID := 91
	ctx := context.Background()
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-TRIGGER").SetDeploymentName("trigger").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey("triggerCompensationFlow").SetName("trigger compensation").SetBpmnXML([]byte(triggerFlowBPMN)).
		SetDeploymentID(deployment.ID).SetTenantID(tenantID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	handlerA := &compensationHandler{id: "test_comp_handler_a"}
	handlerB := &compensationHandler{id: "test_comp_handler_b"}
	engine.callbackRegistry.RegisterHandler(handlerA)
	engine.callbackRegistry.RegisterHandler(handlerB)
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	return engine, handlerA, handlerB, tenantID, workflowCtx
}

func startTriggerInstance(t *testing.T, engine *CustomProcessEngine, client *ent.Client, workflowCtx context.Context, tenantID int) *ent.ProcessInstance {
	t.Helper()
	instance, err := engine.StartProcess(workflowCtx, "triggerCompensationFlow", "change:710", map[string]interface{}{})
	require.NoError(t, err)
	runServiceTaskCommands(t, workflowCtx, client, engine, tenantID)
	return instance
}

// TestTriggerCompensationDryRunDoesNotWrite dry-run 只给计划，不产生任何补偿
// 调用、执行状态或触发留痕。
func TestTriggerCompensationDryRunDoesNotWrite(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	engine, handlerA, handlerB, tenantID, workflowCtx := setupTriggerEngine(t, client)
	instance := startTriggerInstance(t, engine, client, workflowCtx, tenantID)

	result, err := engine.TriggerCompensation(workflowCtx, CompensationTriggerRequest{
		TenantID: tenantID, ActorUserID: 501, InstanceID: instance.ID,
		Reason: "复核发现前置动作需要回滚", DryRun: true,
	})
	require.NoError(t, err)
	assert.True(t, result.DryRun)
	assert.False(t, result.Applied)
	assert.Equal(t, []string{"ServiceTask_2", "ServiceTask_1"}, result.Targets, "必须按完成顺序逆序补偿")
	assert.Equal(t, 0, handlerA.compensations)
	assert.Equal(t, 0, handlerB.compensations)

	inst, err := client.ProcessInstance.Get(context.Background(), instance.ID)
	require.NoError(t, err)
	assert.Empty(t, intMapVariable(inst.Variables, "_compensationExecuted"))
	assert.NotContains(t, inst.Variables, "_compensationTriggerHistory")
}

// TestTriggerCompensationAppliesAndIsIdempotent 显式触发在同一个事务里完成
// 补偿、执行状态与触发留痕；重复触发不得二次补偿。
func TestTriggerCompensationAppliesAndIsIdempotent(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	engine, handlerA, handlerB, tenantID, workflowCtx := setupTriggerEngine(t, client)
	instance := startTriggerInstance(t, engine, client, workflowCtx, tenantID)

	result, err := engine.TriggerCompensation(workflowCtx, CompensationTriggerRequest{
		TenantID: tenantID, ActorUserID: 501, InstanceID: instance.ID,
		Reason: "复核发现前置动作需要回滚",
	})
	require.NoError(t, err)
	assert.True(t, result.Applied)
	assert.Equal(t, []string{"ServiceTask_2", "ServiceTask_1"}, result.Targets)
	assert.Equal(t, 1, handlerA.compensations)
	assert.Equal(t, 1, handlerB.compensations)

	inst, err := client.ProcessInstance.Get(context.Background(), instance.ID)
	require.NoError(t, err)
	executed := intMapVariable(inst.Variables, "_compensationExecuted")
	assert.Equal(t, 1, executed["ServiceTask_1"])
	assert.Equal(t, 1, executed["ServiceTask_2"])
	assert.Contains(t, inst.Variables, "_compensationTriggerHistory", "触发原因与操作人必须同事务留痕")

	again, err := engine.TriggerCompensation(workflowCtx, CompensationTriggerRequest{
		TenantID: tenantID, ActorUserID: 501, InstanceID: instance.ID,
		Reason: "重复触发",
	})
	require.NoError(t, err)
	assert.False(t, again.Applied)
	assert.Empty(t, again.Targets, "重复触发不得再产生补偿目标")
	assert.Equal(t, 1, handlerA.compensations, "重复触发不得二次补偿")
	assert.Equal(t, 1, handlerB.compensations, "重复触发不得二次补偿")
}

// TestTriggerCompensationFailsClosedForCrossTenantAndUnknownActivity 跨租户
// 实例与未登记的 activityRef 都必须 fail-closed，且不得产生任何补偿调用。
func TestTriggerCompensationFailsClosedForCrossTenantAndUnknownActivity(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	engine, handlerA, handlerB, tenantID, workflowCtx := setupTriggerEngine(t, client)
	instance := startTriggerInstance(t, engine, client, workflowCtx, tenantID)

	_, err := engine.TriggerCompensation(workflowCtx, CompensationTriggerRequest{
		TenantID: tenantID + 1, ActorUserID: 501, InstanceID: instance.ID, Reason: "跨租户探测",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不存在或不属于租户")

	_, err = engine.TriggerCompensation(workflowCtx, CompensationTriggerRequest{
		TenantID: tenantID, ActorUserID: 501, InstanceID: instance.ID,
		ActivityRef: "never_registered", Reason: "指定不存在的补偿目标",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未登记")
	assert.Equal(t, 0, handlerA.compensations)
	assert.Equal(t, 0, handlerB.compensations)
}

// TestBuiltinCompensationTemplateHasTriggerNodes 内置模板必须真的带补偿节点，
// 防止「补偿模板」变成一个只有名字的空壳文件。
func TestBuiltinCompensationTemplateHasTriggerNodes(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("bpmn", "compensation_recovery_flow.bpmn"))
	require.NoError(t, err)
	defs, err := NewBPMNParser().ParseXML(data)
	require.NoError(t, err)
	require.Len(t, defs.Processes, 1)
	process := defs.Processes[0]

	compensationEnd := false
	for _, end := range process.EndEvents {
		if end.CompensateEventDefinition != nil {
			compensationEnd = true
		}
	}
	assert.True(t, compensationEnd, "模板必须包含补偿结束事件")

	boundaryCompensation := false
	for _, boundary := range process.BoundaryEvents {
		if boundary.CompensateEventDefinition != nil && boundary.AttachedToRef != "" {
			boundaryCompensation = true
		}
	}
	assert.True(t, boundaryCompensation, "模板必须声明挂到具体活动上的补偿边界事件")

	compensationHandlerNode := false
	for _, task := range process.ServiceTasks {
		if task.IsForCompensation {
			compensationHandlerNode = true
		}
	}
	assert.True(t, compensationHandlerNode, "模板必须包含 isForCompensation 补偿处理器节点")
}
