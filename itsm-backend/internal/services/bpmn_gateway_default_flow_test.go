package service

import (
	"context"
	"strings"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processexecutionhistory"
	"itsm-backend/ent/processtask"

	_ "github.com/mattn/go-sqlite3"

	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// P2-02（codex 审计）：default 分支此前只是 lint 提示，运行时完全不读 DefaultFlow。
//   - 排他网关：条件流零命中直接 fmt.Errorf("没有符合条件的路径")，不落到 default；
//   - 包容网关：零命中只 Warn 后 markElementDone，实例静默停在网关上。
// 本文件用 enttest + 真实 CustomProcessEngine 验证运行语义：
//   1) 排他网关条件全假时走 default 流；
//   2) 包容网关条件全假时走 default 流；
//   3) 两者都没有 default 时实例被显式挂起并写明原因（不再静默）。
// 4) lint 校验 default 流指向真实存在的序列流。

// newGatewayInstance 创建「Start -> 网关 -> 分支 -> End」图对应的实例。
func newGatewayInstance(t *testing.T, ctx context.Context, client *ent.Client, key string) *ent.ProcessInstance {
	t.Helper()
	tenantID := 1
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-" + key).
		SetDeploymentName(key).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	definition, err := client.ProcessDefinition.Create().
		SetKey(key).
		SetName(key).
		SetBpmnXML([]byte("<def/>")).
		SetDeploymentID(deployment.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-" + key).
		SetProcessDefinitionKey(key).
		SetProcessDefinitionID(definition.ID).
		SetTenantID(tenantID).
		SetStatus("running").
		SetVariables(map[string]interface{}{}).
		Save(ctx)
	require.NoError(t, err)
	return instance
}

// TestBPMNExclusiveGateway_DefaultFlowTakenOnZeroConditionMatch 验证排他网关：
// 条件流全部不命中时，按 BPMN 语义走 default 流（此前完全不读 DefaultFlow）。
func TestBPMNExclusiveGateway_DefaultFlowTakenOnZeroConditionMatch(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	ctx := context.Background()
	instance := newGatewayInstance(t, ctx, client, "exclusiveDefault")
	// 变量真实存在但数值不满足条件：条件「可判为假」而不是「编译失败」。
	// 编译失败同样返回 false，但那是另一条路径——用它做断言会掩盖 skip 逻辑。
	_, err := client.ProcessInstance.UpdateOneID(instance.ID).
		SetVariables(map[string]interface{}{"amount": 10}).
		Save(ctx)
	require.NoError(t, err)
	instance, err = client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	require.EqualValues(t, 10, instance.Variables["amount"], "变量应写入实例")

	// Start -> G1 -> (f1 条件不满足) | (f2 default) -> TaskDefault -> End
	// 三条出边：f1 条件为真（正常分支）、f2 条件为假、f3 是 default 兜底。
	// 三者目标不同，测试才能区分「走 default」与「误把 default 当普通流」。
	process := &BPMNProcess{
		StartEvents: []*BPMNStartEvent{{ID: "Start"}},
		ExclusiveGateways: []*BPMNExclusiveGateway{
			{ID: "G1", Name: "排他网关", DefaultFlow: "f3"},
		},
		UserTasks: []*BPMNUserTask{
			{ID: "TaskHigh", Name: "高额任务"},
			{ID: "TaskMid", Name: "普通任务"},
			{ID: "TaskDefault", Name: "兜底任务"},
			{ID: "TaskPlain", Name: "无条件普通流"},
		},
		EndEvents: []*BPMNEndEvent{{ID: "End"}},
		SequenceFlows: []*BPMNSequenceFlow{
			{ID: "f0", SourceRef: "Start", TargetRef: "G1"},
			{
				ID: "f1", SourceRef: "G1", TargetRef: "TaskHigh",
				ConditionExpression: &BPMNConditionExpression{Expression: "amount > 1000"},
			},
			{
				ID: "f2", SourceRef: "G1", TargetRef: "TaskMid",
				ConditionExpression: &BPMNConditionExpression{Expression: "amount > 100"},
			},
			{ID: "f3", SourceRef: "G1", TargetRef: "TaskDefault"},
			{ID: "f4", SourceRef: "TaskDefault", TargetRef: "End"},
			// f5 是「普通但无条件」的出边，目标与 default 不同。
			// BPMN 语义里排他网关只能走一条：无条件普通流应当被命中，
			// 而 default 只在零命中时兜底。此处把 f5 放在 default 之前，
			// 用于区分「default 被当成普通流」与「default 正确兜底」两种实现。
			{ID: "f5", SourceRef: "G1", TargetRef: "TaskPlain"},
		},
	}

	g1 := process.ExclusiveGateways[0]
	err = engine.handleElement(ctx, client, instance, process, g1.ID)
	require.NoError(t, err, "条件流零命中时应走 default 流而不是报错")

	tasks, err := client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(instance.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, tasks, 1, "只应创建一个任务")
	assert.Equal(t, "TaskPlain", tasks[0].TaskDefinitionKey,
		"存在无条件普通出边时应走它；default 只在零命中时兜底")

	// 网关路由历史应记录实际走的那条分支
	hist, err := client.ProcessExecutionHistory.Query().
		Where(
			processexecutionhistory.ProcessInstanceID(instance.ID),
			processexecutionhistory.ActivityType("gateway"),
		).
		All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, hist, "排他网关应写入路由历史")
}

// TestBPMNInclusiveGateway_DefaultFlowTakenOnZeroConditionMatch 验证包容网关：
// 条件流零命中时走 default 流（此前只 Warn 后静默停在网关）。
func TestBPMNInclusiveGateway_DefaultFlowTakenOnZeroConditionMatch(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	ctx := context.Background()
	instance := newGatewayInstance(t, ctx, client, "inclusiveDefault")
	_, err := client.ProcessInstance.UpdateOneID(instance.ID).
		SetVariables(map[string]interface{}{"amount": 10}).
		Save(ctx)
	require.NoError(t, err)
	instance, err = client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	require.EqualValues(t, 10, instance.Variables["amount"], "变量应写入实例")

	// 三条出边目标不同：条件为真的 f1、条件为假的 f2、default 兜底 f3。
	process := &BPMNProcess{
		StartEvents: []*BPMNStartEvent{{ID: "Start"}},
		InclusiveGateways: []*BPMNInclusiveGateway{
			{ID: "G1", Name: "包容网关", DefaultFlow: "f3"},
		},
		UserTasks: []*BPMNUserTask{
			{ID: "TaskHigh", Name: "高额任务"},
			{ID: "TaskMid", Name: "普通任务"},
			{ID: "TaskDefault", Name: "兜底任务"},
			{ID: "TaskPlain", Name: "无条件普通流"},
		},
		EndEvents: []*BPMNEndEvent{{ID: "End"}},
		SequenceFlows: []*BPMNSequenceFlow{
			{ID: "f0", SourceRef: "Start", TargetRef: "G1"},
			{
				ID: "f1", SourceRef: "G1", TargetRef: "TaskHigh",
				ConditionExpression: &BPMNConditionExpression{Expression: "amount > 1000"},
			},
			{
				ID: "f2", SourceRef: "G1", TargetRef: "TaskMid",
				ConditionExpression: &BPMNConditionExpression{Expression: "amount > 100"},
			},
			{ID: "f3", SourceRef: "G1", TargetRef: "TaskDefault"},
			{ID: "f4", SourceRef: "TaskDefault", TargetRef: "End"},
		},
	}

	g1 := process.InclusiveGateways[0]
	err = engine.handleInclusiveGateway(ctx, client, instance, process, g1, 0)
	require.NoError(t, err, "包容网关条件零命中时应走 default 流")

	tasks, err := client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(instance.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, tasks, 1, "包容网关零命中时只应创建 default 指向的任务")
	assert.Equal(t, "TaskDefault", tasks[0].TaskDefinitionKey)
}

// TestBPMNExclusiveGateway_NoDefaultSuspendsWithReason 验证无 default 时：
// 实例被显式挂起并写明原因，而不是返回一句「没有符合条件的路径」让人无从下手。
func TestBPMNExclusiveGateway_NoDefaultSuspendsWithReason(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	ctx := context.Background()
	instance := newGatewayInstance(t, ctx, client, "exclusiveNoDefault")

	process := &BPMNProcess{
		StartEvents:       []*BPMNStartEvent{{ID: "Start"}},
		ExclusiveGateways: []*BPMNExclusiveGateway{{ID: "G1", Name: "排他网关"}},
		UserTasks:         []*BPMNUserTask{{ID: "TaskDenied", Name: "高额任务"}},
		EndEvents:         []*BPMNEndEvent{{ID: "End"}},
		SequenceFlows: []*BPMNSequenceFlow{
			{ID: "f0", SourceRef: "Start", TargetRef: "G1"},
			{
				ID: "f1", SourceRef: "G1", TargetRef: "TaskDenied",
				ConditionExpression: &BPMNConditionExpression{Expression: "amount > 1000"},
			},
		},
	}

	err := engine.handleElement(ctx, client, instance, process, "G1")
	require.Error(t, err, "无 default 且条件零命中应显式失败")

	inst, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "suspended", inst.Status, "实例应被显式挂起而非静默停住")
	require.NotNil(t, inst.SuspendedReason)
	assert.True(t, strings.Contains(inst.SuspendedReason, "default"),
		"挂起原因应指明缺少 default 流，实际=%q", inst.SuspendedReason)
}

// TestBPMNInclusiveGateway_NoDefaultSuspendsWithReason 验证包容网关同样显式挂起。
func TestBPMNInclusiveGateway_NoDefaultSuspendsWithReason(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	ctx := context.Background()
	instance := newGatewayInstance(t, ctx, client, "inclusiveNoDefault")

	process := &BPMNProcess{
		StartEvents:       []*BPMNStartEvent{{ID: "Start"}},
		InclusiveGateways: []*BPMNInclusiveGateway{{ID: "G1", Name: "包容网关"}},
		UserTasks:         []*BPMNUserTask{{ID: "TaskDenied", Name: "高额任务"}},
		EndEvents:         []*BPMNEndEvent{{ID: "End"}},
		SequenceFlows: []*BPMNSequenceFlow{
			{ID: "f0", SourceRef: "Start", TargetRef: "G1"},
			{
				ID: "f1", SourceRef: "G1", TargetRef: "TaskDenied",
				ConditionExpression: &BPMNConditionExpression{Expression: "amount > 1000"},
			},
		},
	}

	err := engine.handleInclusiveGateway(ctx, client, instance, process, process.InclusiveGateways[0], 0)
	require.Error(t, err, "包容网关无 default 且条件零命中应显式失败")

	inst, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "suspended", inst.Status)
	require.NotNil(t, inst.SuspendedReason)
	assert.True(t, strings.Contains(inst.SuspendedReason, "default"),
		"挂起原因应指明缺少 default 流，实际=%q", inst.SuspendedReason)
}

// TestBPMNExclusiveGateway_InvalidDefaultSuspends 验证 default 指向不存在的流时
// 显式挂起（而不是等运行时才发现、也不知道是哪一条流坏了）。
func TestBPMNExclusiveGateway_InvalidDefaultSuspends(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	ctx := context.Background()
	instance := newGatewayInstance(t, ctx, client, "exclusiveBadDefault")

	process := &BPMNProcess{
		StartEvents: []*BPMNStartEvent{{ID: "Start"}},
		ExclusiveGateways: []*BPMNExclusiveGateway{
			{ID: "G1", Name: "排他网关", DefaultFlow: "flowNotExist"},
		},
		UserTasks: []*BPMNUserTask{{ID: "TaskDenied", Name: "高额任务"}},
		EndEvents: []*BPMNEndEvent{{ID: "End"}},
		SequenceFlows: []*BPMNSequenceFlow{
			{ID: "f0", SourceRef: "Start", TargetRef: "G1"},
			{
				ID: "f1", SourceRef: "G1", TargetRef: "TaskDenied",
				ConditionExpression: &BPMNConditionExpression{Expression: "amount > 1000"},
			},
		},
	}

	err := engine.handleElement(ctx, client, instance, process, "G1")
	require.Error(t, err)

	inst, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "suspended", inst.Status)
	require.NotNil(t, inst.SuspendedReason)
	assert.True(t, strings.Contains(inst.SuspendedReason, "flowNotExist"),
		"挂起原因应指明坏掉的 default 流 ID，实际=%q", inst.SuspendedReason)
}

const lintBadDefaultBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Def_D1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Proc_D1" isExecutable="true">
    <bpmn:startEvent id="Start_D1" name="开始"/>
    <bpmn:exclusiveGateway id="G_D1" name="排他" default="Flow_Missing"/>
    <bpmn:userTask id="Task_D1" name="处理"/>
    <bpmn:endEvent id="End_D1" name="结束"/>
    <bpmn:sequenceFlow id="Flow_D1" sourceRef="Start_D1" targetRef="G_D1"/>
    <bpmn:sequenceFlow id="Flow_D2" sourceRef="G_D1" targetRef="Task_D1"/>
    <bpmn:sequenceFlow id="Flow_D3" sourceRef="Task_D1" targetRef="End_D1"/>
  </bpmn:process>
</bpmn:definitions>`

// TestParserRejectsUnknownDefaultFlow 验证坏 default 流在部署期就被拦下
// （parser.validateGateways 校验 default 流是否存在），不必等运行时才发现。
func TestParserRejectsUnknownDefaultFlow(t *testing.T) {
	svc := NewBPMNLintService()
	_, err := svc.LintBPMNXML([]byte(lintBadDefaultBPMN))
	require.Error(t, err, "default 流不存在必须在部署期报错")
	assert.True(t, strings.Contains(err.Error(), "默认流"),
		"错误信息应指明默认流问题，实际=%v", err)
}

const lintGoodDefaultBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Def_D2" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Proc_D2" isExecutable="true">
    <bpmn:startEvent id="Start_D2" name="开始"/>
    <bpmn:exclusiveGateway id="G_D2" name="排他" default="Flow_OK"/>
    <bpmn:userTask id="Task_D2" name="处理"/>
    <bpmn:endEvent id="End_D2" name="结束"/>
    <bpmn:sequenceFlow id="Flow_OK" sourceRef="G_D2" targetRef="Task_D2"/>
    <bpmn:sequenceFlow id="Flow_D2a" sourceRef="Start_D2" targetRef="G_D2"/>
    <bpmn:sequenceFlow id="Flow_D2b" sourceRef="Task_D2" targetRef="End_D2"/>
  </bpmn:process>
</bpmn:definitions>`

// TestParserAcceptsValidDefaultFlow 验证合法 default 流不会被误拦
// （反向用例：防止「一律报错」式的假修复）。
func TestParserAcceptsValidDefaultFlow(t *testing.T) {
	svc := NewBPMNLintService()
	result, err := svc.LintBPMNXML([]byte(lintGoodDefaultBPMN))
	require.NoError(t, err, "合法的 default 流必须能通过解析")
	assert.False(t, result.HasErrors, "合法 default 流不应产生 error")
}
