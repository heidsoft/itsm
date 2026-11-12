package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processauditlog"
	"itsm-backend/ent/processinstance"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== BPMN 补偿事件测试 ====================

// compensationFlowBPMN：Start → ServiceTask_1（可补偿）→ ServiceTask_2 →
// CompensationEnd（<compensateEventDefinition/>）→ End。
const compensationFlowBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="Process_compensation" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1"/>
    <bpmn:serviceTask id="ServiceTask_1" implementation="test_comp_handler_a"/>
    <bpmn:serviceTask id="ServiceTask_2" implementation="test_comp_handler_b"/>
    <bpmn:endEvent id="CompensationEnd">
      <bpmn:compensateEventDefinition/>
    </bpmn:endEvent>
    <bpmn:endEvent id="EndEvent_2"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="ServiceTask_1"/>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="ServiceTask_1" targetRef="ServiceTask_2"/>
    <bpmn:sequenceFlow id="Flow_3" sourceRef="ServiceTask_2" targetRef="CompensationEnd"/>
    <bpmn:sequenceFlow id="Flow_4" sourceRef="CompensationEnd" targetRef="EndEvent_2"/>
  </bpmn:process>
</bpmn:definitions>`

// compensationHandler 记录调用与补偿调用的处理器。
type compensationHandler struct {
	id            string
	execCalls     int
	compensations int
	lastVars      map[string]interface{}
}

func (h *compensationHandler) GetTaskType() string  { return "ServiceTask" }
func (h *compensationHandler) GetHandlerID() string { return h.id }
func (h *compensationHandler) Validate(context.Context, map[string]interface{}) error {
	return nil
}

func (h *compensationHandler) Execute(_ context.Context, _ *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	if compensate, _ := variables["_compensate"].(bool); compensate {
		h.compensations++
	} else {
		h.execCalls++
	}
	h.lastVars = variables
	return &dto.ServiceTaskResult{Success: true, OutputVars: map[string]interface{}{h.id + "_done": true}}, nil
}

// setupCompensationEngine 搭建补偿流程引擎（两个可补偿 ServiceTask）。
func setupCompensationEngine(t *testing.T, client *ent.Client) (*CustomProcessEngine, *compensationHandler, *compensationHandler, int, context.Context) {
	t.Helper()
	tenantID := 88
	ctx := context.Background()
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-COMP").SetDeploymentName("comp").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey("compensationFlow").SetName("compensation").SetBpmnXML([]byte(compensationFlowBPMN)).
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

// runServiceTaskCommands 驱动命令总线把实例内所有 ServiceTask 命令跑完。
func runServiceTaskCommands(t *testing.T, ctx context.Context, client *ent.Client, engine *CustomProcessEngine, tenantID int) {
	t.Helper()
	registry := commandbus.NewRegistry()
	require.NoError(t, registry.Register(commandbus.CommandExecuteBPMNServiceTask, engine.HandleBPMNServiceTaskCommand))
	worker := commandbus.NewWorker(client, registry, zaptest.NewLogger(t).Sugar(), "comp-test-worker")
	for i := 0; i < 5; i++ {
		processed, err := worker.RunOnce(ctx)
		require.NoError(t, err)
		if !processed {
			return
		}
	}
}

// TestCompensationEndEventCompensatesInReverseOrder 补偿结束事件：
// 两个 ServiceTask 完成后到达补偿结束事件，应按逆序补偿
// （后完成的 ServiceTask_2 先补偿），各补一次，流程随后正常结束。
func TestCompensationEndEventCompensatesInReverseOrder(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	engine, handlerA, handlerB, tenantID, workflowCtx := setupCompensationEngine(t, client)
	ctx := context.Background()

	instance, err := engine.StartProcess(workflowCtx, "compensationFlow", "change:1", map[string]interface{}{})
	require.NoError(t, err)
	runServiceTaskCommands(t, ctx, client, engine, tenantID)

	require.Equal(t, 1, handlerA.execCalls, "ServiceTask_1 正常执行一次")
	require.Equal(t, 1, handlerB.execCalls, "ServiceTask_2 正常执行一次")
	require.Equal(t, 1, handlerA.compensations, "ServiceTask_1 被补偿一次")
	require.Equal(t, 1, handlerB.compensations, "ServiceTask_2 被补偿一次")

	// 补偿顺序：ServiceTask_2 的补偿必须晚于其执行、且先于 ServiceTask_1 的补偿。
	// 通过审计行的 timestamp/ID 顺序验证。
	audits, err := client.ProcessAuditLog.Query().
		Where(processauditlog.ActionEQ("compensated")).
		Order(ent.Asc(processauditlog.FieldID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, audits, 2)
	assert.Equal(t, "ServiceTask_2", audits[0].ActivityID, "后完成的活动先补偿")
	assert.Equal(t, "ServiceTask_1", audits[1].ActivityID)

	// 补偿变量带 _compensate 标记与完成时快照。
	assert.Equal(t, true, handlerB.lastVars["_compensate"])
	assert.Equal(t, "ServiceTask_2", handlerB.lastVars["_compensatingActivity"])

	// 流程正常结束。
	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status)
}

// TestCompensationIsIdempotentPerOccurrence 重复到达补偿结束事件
// （实例变量恢复路径）不得重复补偿：每 occurrence 只补一次。
func TestCompensationIsIdempotentPerOccurrence(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	engine, handlerA, handlerB, tenantID, workflowCtx := setupCompensationEngine(t, client)
	ctx := context.Background()

	instance, err := engine.StartProcess(workflowCtx, "compensationFlow", "change:2", map[string]interface{}{})
	require.NoError(t, err)
	runServiceTaskCommands(t, ctx, client, engine, tenantID)
	require.Equal(t, 1, handlerA.compensations)
	require.Equal(t, 1, handlerB.compensations)

	// 手动重放补偿（模拟崩溃恢复/重复触发）。
	inst, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	engine2 := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	engine2.callbackRegistry.RegisterHandler(handlerA)
	engine2.callbackRegistry.RegisterHandler(handlerB)
	require.NoError(t, engine2.runCompensation(ctx, client, inst, &BPMNCompensateEventDefinition{}))

	assert.Equal(t, 1, handlerA.compensations, "重复补偿必须幂等跳过")
	assert.Equal(t, 1, handlerB.compensations, "重复补偿必须幂等跳过")
}

// TestCompensationUnknownActivityRefFailsClosed activityRef 指向未登记
// 活动时必须失败关闭，不得静默跳过。
func TestCompensationUnknownActivityRefFailsClosed(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	engine, _, _, tenantID, workflowCtx := setupCompensationEngine(t, client)
	ctx := context.Background()

	_, err := engine.StartProcess(workflowCtx, "compensationFlow", "change:3", map[string]interface{}{})
	require.NoError(t, err)
	inst, err := client.ProcessInstance.Query().Where(
		processinstance.TenantIDEQ(tenantID),
	).First(ctx)
	require.NoError(t, err)

	err = engine.runCompensation(ctx, client, inst, &BPMNCompensateEventDefinition{ActivityRef: "ServiceTask_1"})
	require.Error(t, err, "未登记的补偿目标必须 fail-closed")
	assert.Contains(t, err.Error(), "未登记")
}

// TestCompensationOnlyServiceTaskSkippedOnMainFlow isForCompensation 活动
// 在主流程到达时跳过执行，只作为补偿处理器存在。
func TestCompensationOnlyServiceTaskSkippedOnMainFlow(t *testing.T) {
	compOnlyXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="Process_comp_only" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1"/>
    <bpmn:serviceTask id="ServiceTask_1" implementation="test_comp_main"/>
    <bpmn:serviceTask id="CompensateTask_1" implementation="test_comp_only_handler" isForCompensation="true"/>
    <bpmn:endEvent id="EndEvent_1"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="ServiceTask_1"/>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="ServiceTask_1" targetRef="CompensateTask_1"/>
    <bpmn:sequenceFlow id="Flow_3" sourceRef="CompensateTask_1" targetRef="EndEvent_1"/>
  </bpmn:process>
</bpmn:definitions>`
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	tenantID := 89
	ctx := context.Background()
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-COMP-ONLY").SetDeploymentName("comp-only").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey("compOnlyFlow").SetName("comp-only").SetBpmnXML([]byte(compOnlyXML)).
		SetDeploymentID(deployment.ID).SetTenantID(tenantID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	main := &compensationHandler{id: "test_comp_main"}
	compOnly := &compensationHandler{id: "test_comp_only_handler"}
	engine.callbackRegistry.RegisterHandler(main)
	engine.callbackRegistry.RegisterHandler(compOnly)
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)

	instance, err := engine.StartProcess(workflowCtx, "compOnlyFlow", "change:4", map[string]interface{}{})
	require.NoError(t, err)
	runServiceTaskCommands(t, ctx, client, engine, tenantID)

	assert.Equal(t, 1, main.execCalls, "主活动正常执行")
	assert.Equal(t, 0, compOnly.execCalls, "isForCompensation 活动不得在主流程执行")

	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status, "跳过补偿专用活动后流程应正常结束")
}

// TestPendingCompensationTargetsReportsUncompensated 盘点辅助：
// 未补偿的已登记活动应被列出。
func TestPendingCompensationTargetsReportsUncompensated(t *testing.T) {
	variables := map[string]interface{}{
		"_compensationRegistered": map[string]interface{}{
			"A": map[string]interface{}{"occurrence": float64(1), "serviceRef": "h"},
			"B": map[string]interface{}{"occurrence": float64(1), "serviceRef": "h"},
		},
		"_compensationOrder": []interface{}{"A", "B"},
		"_compensationExecuted": map[string]interface{}{
			"A": float64(1),
		},
	}
	pending := pendingCompensationTargets(variables)
	require.Len(t, pending, 1)
	assert.Equal(t, "B", pending[0])
}

// pendingCompensationTargets 返回尚未补偿的已登记活动（盘点/诊断用）。
