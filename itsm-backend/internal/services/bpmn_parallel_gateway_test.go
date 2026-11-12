package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processexecutionhistory"
	"itsm-backend/ent/processtask"

	_ "github.com/mattn/go-sqlite3"

	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBPMNParallelGateway_ForkAndJoin 验证 F-1：并行网关分叉生成多个任务，汇聚等待所有分支完成后才推进。
// 使用 enttest（sqlite）在内存库验证，不触碰生产数据。
func TestBPMNParallelGateway_ForkAndJoin(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	ctx := context.Background()
	tenantID := 1

	// 创建最小可用的部署与流程定义（满足 ProcessInstance 的外键约束）
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-PARALLEL-1").
		SetDeploymentName("parallel").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	definition, err := client.ProcessDefinition.Create().
		SetKey("parallelDemo").
		SetName("并行演示").
		SetBpmnXML([]byte("<def/>")).
		SetDeploymentID(deployment.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	// 构造流程实例（直接 seed，跳过 StartProcess 的激活/最新版本约束）
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-PARALLEL-1").
		SetProcessDefinitionKey("parallelDemo").
		SetProcessDefinitionID(definition.ID).
		SetTenantID(tenantID).
		SetStatus("running").
		SetVariables(map[string]interface{}{}).
		Save(ctx)
	require.NoError(t, err)

	// BPMN 图：Start -> G1(并行分叉) -> TaskA / TaskB -> G2(并行汇聚) -> End
	process := &BPMNProcess{
		StartEvents:      []*BPMNStartEvent{{ID: "Start"}},
		ParallelGateways: []*BPMNParallelGateway{{ID: "G1"}, {ID: "G2"}},
		UserTasks:        []*BPMNUserTask{{ID: "TaskA", Name: "分支A"}, {ID: "TaskB", Name: "分支B"}},
		EndEvents:        []*BPMNEndEvent{{ID: "End"}},
		SequenceFlows: []*BPMNSequenceFlow{
			{ID: "f0", SourceRef: "Start", TargetRef: "G1"},
			{ID: "f1", SourceRef: "G1", TargetRef: "TaskA"},
			{ID: "f2", SourceRef: "G1", TargetRef: "TaskB"},
			{ID: "f3", SourceRef: "TaskA", TargetRef: "G2"},
			{ID: "f4", SourceRef: "TaskB", TargetRef: "G2"},
			{ID: "f5", SourceRef: "G2", TargetRef: "End"},
		},
	}

	g1 := process.ParallelGateways[0]
	g2 := process.ParallelGateways[1]

	// 1) 分叉：从 G1 出发应同时创建 TaskA 与 TaskB（修复「静默串行化」）
	err = engine.handleParallelGateway(ctx, client, instance, process, g1, 0)
	require.NoError(t, err)

	tasks, err := client.ProcessTask.Query().Where(processtask.ProcessInstanceID(instance.ID)).All(ctx)
	require.NoError(t, err)
	assert.Len(t, tasks, 2, "并行分叉应生成 2 个任务，而非串行化的 1 个")

	// F-4 验证：并行网关分叉应写入 ProcessExecutionHistory（activity_type=gateway），使路由可审计
	gwHist, err := client.ProcessExecutionHistory.Query().
		Where(
			processexecutionhistory.ProcessInstanceID(instance.ID),
			processexecutionhistory.ActivityType("gateway"),
		).
		All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, gwHist, "并行网关分叉应写入流程执行历史")
	assert.Equal(t, "parallel.fork", gwHist[0].EventType, "网关历史事件类型应为 parallel.fork")

	// 2) 汇聚等待：仅完成 TaskA 时，G2 不应推进
	_, err = client.ProcessTask.Update().
		Where(processtask.ProcessInstanceID(instance.ID), processtask.TaskDefinitionKey("TaskA")).
		SetStatus("completed").
		Save(ctx)
	require.NoError(t, err)

	err = engine.handleParallelGateway(ctx, client, instance, process, g2, 0)
	require.NoError(t, err)

	// 此时不应再产生新任务（End 不是任务），实例不应结束
	tasksAfter, err := client.ProcessTask.Query().Where(processtask.ProcessInstanceID(instance.ID)).All(ctx)
	require.NoError(t, err)
	assert.Len(t, tasksAfter, 2, "G2 汇聚等待期间不应新增任务")
	instStillRunning, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", instStillRunning.Status, "TaskB 未结束时实例不应到达 End")

	// 3) 完成 TaskB 后，G2 汇聚放行，流程到达 End
	_, err = client.ProcessTask.Update().
		Where(processtask.ProcessInstanceID(instance.ID), processtask.TaskDefinitionKey("TaskB")).
		SetStatus("completed").
		Save(ctx)
	require.NoError(t, err)

	err = engine.handleParallelGateway(ctx, client, instance, process, g2, 0)
	require.NoError(t, err)

	instDone, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", instDone.Status, "所有分支完成后，并行汇聚应推进到 End 并结束流程")
}
