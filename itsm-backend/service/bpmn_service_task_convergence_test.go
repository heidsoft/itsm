package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const convergenceServiceTaskBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="Process_convergence_service" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1"/>
    <bpmn:serviceTask id="ServiceTask_1" implementation="test_convergence_handler"/>
    <bpmn:endEvent id="EndEvent_1"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="ServiceTask_1"/>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="ServiceTask_1" targetRef="EndEvent_1"/>
  </bpmn:process>
</bpmn:definitions>`

// countingHandler 记录执行次数的 ServiceTask 处理器（外部副作用计数器）。
// crashFlag 非 nil 时，Execute 成功返回前置位——模拟「副作用已生效、
// 完成事务尚未提交」的崩溃窗口（测试用注入点）。
type countingHandler struct {
	calls     int
	crashFlag *bool
}

func (h *countingHandler) GetTaskType() string  { return "ServiceTask" }
func (h *countingHandler) GetHandlerID() string { return "test_convergence_handler" }
func (h *countingHandler) Validate(context.Context, map[string]interface{}) error {
	return nil
}
func (h *countingHandler) Execute(_ context.Context, _ *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	h.calls++
	if h.crashFlag != nil {
		*h.crashFlag = true
	}
	return &dto.ServiceTaskResult{Success: true, OutputVars: map[string]interface{}{"serviceResult": "ok"}}, nil
}

// setupConvergenceEngine 搭建：单 ServiceTask 流程 + 计数处理器 + 返回
// (engine, handler, tenantID, workflowCtx)。流程：Start → ServiceTask → End。
func setupConvergenceEngine(t *testing.T, client *ent.Client) (*CustomProcessEngine, *countingHandler, int, context.Context) {
	t.Helper()
	tenantID := 77
	ctx := context.Background()
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-CONVERGENCE").SetDeploymentName("convergence").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey("convergenceService").SetName("convergence").SetBpmnXML([]byte(convergenceServiceTaskBPMN)).
		SetDeploymentID(deployment.ID).SetTenantID(tenantID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	handler := &countingHandler{}
	engine.callbackRegistry.RegisterHandler(handler)
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	return engine, handler, tenantID, workflowCtx
}

// claimServiceTaskLease 模拟 worker 认领命令（生产由 commandbus.Worker 完成）。
func claimServiceTaskLease(t *testing.T, ctx context.Context, client *ent.Client, cmd *ent.OperationalCommand, owner string, token int64) *ent.OperationalCommand {
	t.Helper()
	expiry := time.Now().Add(5 * time.Minute)
	updated, err := client.OperationalCommand.UpdateOneID(cmd.ID).
		SetStatus(commandbus.StatusProcessing).
		SetLeaseOwner(owner).
		SetFencingToken(token).
		SetLeaseExpiresAt(expiry).
		Save(ctx)
	require.NoError(t, err)
	return updated
}

// TestServiceTaskSideEffectExactlyOnceAcrossCrash 事务收敛核心用例：
// 副作用（handler.Execute）在事务外执行，「完成事务提交前崩溃」时，
// 重试不得重复执行副作用——引擎按 _serviceTaskExecuting 意图标记恢复。
//
// 崩溃窗口用 ent hook 模拟：handler 执行成功后，完成事务内的
// ProcessInstance 写入被注入失败（等价于 worker 在提交前被杀）。
func TestServiceTaskSideEffectExactlyOnceAcrossCrash(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	engine, handler, tenantID, workflowCtx := setupConvergenceEngine(t, client)
	ctx := context.Background()

	instance, err := engine.StartProcess(workflowCtx, "convergenceService", "change:42", map[string]interface{}{})
	require.NoError(t, err)
	cmd, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID),
		operationalcommand.CommandTypeEQ(commandbus.CommandExecuteBPMNServiceTask),
		operationalcommand.AggregateIDEQ(instance.ID),
	).Only(ctx)
	require.NoError(t, err)

	// 注入崩溃：只炸「完成事务」的实例变量写入（带 _serviceTaskCompletedOccurrences
	// 标记的那次），执行权登记与输出变量持久化放行——精确模拟 worker 在
	// 副作用生效后、完成事务提交前被杀的窗口。
	var crashAfterExecute bool
	handler.crashFlag = &crashAfterExecute
	client.ProcessInstance.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if crashAfterExecute {
				if vars, ok := m.Field("variables"); ok {
					if varsMap, isMap := vars.(map[string]interface{}); isMap {
						if _, done := varsMap["_serviceTaskCompletedOccurrences"]; done {
							return nil, assert.AnError
						}
					}
				}
			}
			return next.Mutate(ctx, m)
		})
	})

	// 第一次尝试：认领 → 执行权登记 → handler 副作用生效 → 完成事务崩溃。
	claimed := claimServiceTaskLease(t, ctx, client, cmd, "worker-crash", 1)
	err = engine.HandleBPMNServiceTaskCommand(workflowCtx, claimed)
	require.Error(t, err, "完成事务必须因注入失败而回滚")
	require.Equal(t, 1, handler.calls, "副作用在第一次尝试中已生效")
	crashAfterExecute = false

	// 崩溃恢复：另一 worker 重新认领（fencing token 递增），重放同一命令。
	reclaimed := claimServiceTaskLease(t, ctx, client, cmd, "worker-retry", 2)
	require.NoError(t, engine.HandleBPMNServiceTaskCommand(workflowCtx, reclaimed))

	// 副作用 exactly-once：恢复路径跳过重复执行。
	require.Equal(t, 1, handler.calls, "崩溃恢复不得重复执行 ServiceTask 副作用")

	// 流程在恢复尝试中完成，且崩溃尝试中 handler 产出的变量已持久化。
	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status)
	assert.Equal(t, "ok", completed.Variables["serviceResult"], "崩溃恢复不得丢失 handler 输出变量")
	attempts := intMapVariable(completed.Variables, "_serviceTaskAttempts")
	assert.Equal(t, 1, attempts["ServiceTask_1"], "执行权只登记一次（崩溃恢复不重复计数）")
}

// TestServiceTaskHandlerFailureClearsExecutionIntent 处理器明确失败时，
// 执行权登记必须被清除，使后续重试重新执行副作用（不能把失败当成功跳过）。
func TestServiceTaskHandlerFailureClearsExecutionIntent(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	engine, _, tenantID, workflowCtx := setupConvergenceEngine(t, client)
	ctx := context.Background()

	instance, err := engine.StartProcess(workflowCtx, "convergenceService", "change:43", map[string]interface{}{})
	require.NoError(t, err)
	cmd, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID),
		operationalcommand.CommandTypeEQ(commandbus.CommandExecuteBPMNServiceTask),
		operationalcommand.AggregateIDEQ(instance.ID),
	).Only(ctx)
	require.NoError(t, err)

	// 注册一个必失败的处理器替换成功处理器。
	failing := &failingHandler{}
	engine.callbackRegistry.RegisterHandler(failing)

	claimed := claimServiceTaskLease(t, ctx, client, cmd, "worker-fail", 1)
	err = engine.HandleBPMNServiceTaskCommand(workflowCtx, claimed)
	require.Error(t, err)
	require.Equal(t, 1, failing.calls)

	// 执行权登记已清除：实例变量中不存在该 element 的 executing 标记。
	inst, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	executing := intMapVariable(inst.Variables, "_serviceTaskExecuting")
	_, stillMarked := executing["ServiceTask_1"]
	assert.False(t, stillMarked, "handler 失败后必须清除执行权登记，否则重试会跳过副作用")

	// 换回成功处理器重试：副作用应重新执行并推进流程。
	engine.callbackRegistry.RegisterHandler(&countingHandler{})
	reclaimed := claimServiceTaskLease(t, ctx, client, cmd, "worker-retry", 2)
	require.NoError(t, engine.HandleBPMNServiceTaskCommand(workflowCtx, reclaimed))

	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status)
}

// failingHandler 永远返回错误的处理器。
type failingHandler struct {
	calls int
}

func (h *failingHandler) GetTaskType() string  { return "ServiceTask" }
func (h *failingHandler) GetHandlerID() string { return "test_convergence_handler" }
func (h *failingHandler) Validate(context.Context, map[string]interface{}) error {
	return nil
}
func (h *failingHandler) Execute(context.Context, *ent.ProcessTask, map[string]interface{}) (*dto.ServiceTaskResult, error) {
	h.calls++
	return nil, assert.AnError
}
