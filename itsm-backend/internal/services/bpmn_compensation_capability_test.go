package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service/bpmn"
)

// 审计 P1-09：补偿调用原 handler 会把正向副作用再做一遍。
// 补偿能力必须 opt-in（bpmn.CompensationCapableHandler），未声明的一律不登记、
// 补偿阶段 fail-closed。

// forwardOnlyHandler 只会执行正向动作、**不实现** SupportsCompensation 的 handler，
// 等价于生产里的 webhook/ticket/notification handler。
type forwardOnlyHandler struct {
	id              string
	execCalls       int
	compensateCalls int
}

func (h *forwardOnlyHandler) GetTaskType() string  { return "ServiceTask" }
func (h *forwardOnlyHandler) GetHandlerID() string { return h.id }
func (h *forwardOnlyHandler) Validate(context.Context, map[string]interface{}) error {
	return nil
}

func (h *forwardOnlyHandler) Execute(_ context.Context, _ *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	if compensate, _ := variables["_compensate"].(bool); compensate {
		h.compensateCalls++
	} else {
		h.execCalls++
	}
	return &dto.ServiceTaskResult{Success: true, OutputVars: map[string]interface{}{h.id + "_done": true}}, nil
}

// TestHandlerSupportsCompensation_FailClosed 未实现能力接口 / 显式返回 false /
// nil handler 都必须判为不支持（fail-closed），只有显式 opt-in 才为 true。
func TestHandlerSupportsCompensation_FailClosed(t *testing.T) {
	assert.False(t, bpmn.HandlerSupportsCompensation(nil), "nil handler 必须不支持")
	assert.False(t, bpmn.HandlerSupportsCompensation(&forwardOnlyHandler{id: "fwd"}),
		"未实现 CompensationCapableHandler 的 handler 必须不支持")
	assert.True(t, bpmn.HandlerSupportsCompensation(&compensationHandler{id: "cap"}),
		"显式 opt-in 的 handler 应支持")
}

// setupForwardOnlyEngine 搭一个「handler 不支持补偿」的引擎（与
// setupCompensationEngine 同构，只换 handler 类型）。
func setupForwardOnlyEngine(t *testing.T, client *ent.Client) (*CustomProcessEngine, *forwardOnlyHandler, *forwardOnlyHandler, int) {
	t.Helper()
	tenantID := 92
	ctx := context.Background()
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-FWD").SetDeploymentName("forward-only").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey("compensationFlow").SetName("forward only").SetBpmnXML([]byte(compensationFlowBPMN)).
		SetDeploymentID(deployment.ID).SetTenantID(tenantID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)

	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	handlerA := &forwardOnlyHandler{id: "test_comp_handler_a"}
	handlerB := &forwardOnlyHandler{id: "test_comp_handler_b"}
	engine.callbackRegistry.RegisterHandler(handlerA)
	engine.callbackRegistry.RegisterHandler(handlerB)
	return engine, handlerA, handlerB, tenantID
}

// TestRegisterCompensation_SkipsHandlerWithoutCapability 正向 handler 执行成功后
// **不得**被登记为可补偿：登记即承诺可回滚，而不支持补偿的 handler 只会把正向
// 动作再做一遍。流程本身必须照常完成（能力门禁不能误伤正常推进）。
func TestRegisterCompensation_SkipsHandlerWithoutCapability(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	engine, handlerA, handlerB, tenantID := setupForwardOnlyEngine(t, client)
	workflowCtx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, tenantID)

	instance, err := engine.StartProcess(workflowCtx, "compensationFlow", "change:5", map[string]interface{}{})
	require.NoError(t, err)
	runServiceTaskCommands(t, workflowCtx, client, engine, tenantID)

	assert.Equal(t, 1, handlerA.execCalls, "正向执行不受能力门禁影响")
	assert.Equal(t, 1, handlerB.execCalls, "正向执行不受能力门禁影响")
	assert.Equal(t, 0, handlerA.compensateCalls)
	assert.Equal(t, 0, handlerB.compensateCalls)

	completed, err := client.ProcessInstance.Get(context.Background(), instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status, "能力门禁只跳过补偿登记，不阻断流程推进")
	assert.Empty(t, completed.Variables["_compensationRegistered"],
		"不支持补偿的活动不得被登记（登记即承诺可回滚）")
	assert.Empty(t, pendingCompensationTargets(completed.Variables),
		"不支持补偿的活动不得出现在待补偿清单里")
}

// TestRunCompensation_RejectsHandlerWithoutCapability 覆盖**防御性复检**：
// 补偿范围可能来自门禁加入之前的老实例（持久化在实例变量里），此时必须 fail-closed
// 拒绝，而不是把正向动作当补偿重做一遍。
func TestRunCompensation_RejectsHandlerWithoutCapability(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	tenantID := 93

	engine, handlerA, _, _ := setupForwardOnlyEngine(t, client)

	// 模拟门禁加入之前登记过的老实例：范围里已有一个不支持补偿的活动。
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-LEGACY").SetDeploymentName("legacy").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey("legacyCompFlow").SetName("legacy").SetBpmnXML([]byte(compensationFlowBPMN)).
		SetDeploymentID(deployment.ID).SetTenantID(tenantID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("proc-inst-legacy-comp").
		SetProcessDefinitionKey("legacyCompFlow").
		SetProcessDefinitionID(def.ID).
		SetStatus("running").
		SetCurrentActivityID("ServiceTask_1").
		SetTenantID(tenantID).
		SetVariables(map[string]interface{}{
			"_compensationOrder": []string{"ServiceTask_1"},
			"_compensationRegistered": map[string]interface{}{
				"ServiceTask_1": map[string]interface{}{
					"occurrence": 1,
					"serviceRef": "test_comp_handler_a",
					"variables":  map[string]interface{}{},
				},
			},
		}).
		Save(ctx)
	require.NoError(t, err)

	fresh, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)

	err = engine.runCompensation(ctx, client, fresh, &BPMNCompensateEventDefinition{})
	require.Error(t, err, "不支持补偿的活动必须 fail-closed 拒绝")
	assert.Contains(t, err.Error(), "未声明可补偿能力")
	assert.Equal(t, 0, handlerA.compensateCalls, "拒绝路径不得执行任何补偿（正向动作不得被重做）")
}
