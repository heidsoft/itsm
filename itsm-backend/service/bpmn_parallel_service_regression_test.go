package service

import (
	"context"
	"fmt"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/ent/processtask"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 并行网关分叉中，服务任务分支按声明顺序先于用户任务分支派发时，
// CurrentActivityID 会被后派发分支覆盖。命令完成不得依赖单位置列判定陈旧，
// 否则命令被误杀、join 永不汇聚、实例永卡 running。

const parallelServiceFirstBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="parallel_service_first" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:parallelGateway id="fork"/>
    <bpmn:serviceTask id="svc_branch" implementation="parallel_service_handler"/>
    <bpmn:userTask id="user_branch" name="分支B"/>
    <bpmn:parallelGateway id="join"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="fork"/>
    <bpmn:sequenceFlow id="f2" sourceRef="fork" targetRef="svc_branch"/>
    <bpmn:sequenceFlow id="f3" sourceRef="fork" targetRef="user_branch"/>
    <bpmn:sequenceFlow id="f4" sourceRef="svc_branch" targetRef="join"/>
    <bpmn:sequenceFlow id="f5" sourceRef="user_branch" targetRef="join"/>
    <bpmn:sequenceFlow id="f6" sourceRef="join" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

type parallelServiceTestHandler struct{ calls int }

func (h *parallelServiceTestHandler) GetTaskType() string                                { return "ServiceTask" }
func (h *parallelServiceTestHandler) GetHandlerID() string                               { return "parallel_service_handler" }
func (h *parallelServiceTestHandler) Validate(context.Context, map[string]interface{}) error { return nil }
func (h *parallelServiceTestHandler) Execute(_ context.Context, _ *ent.ProcessTask, _ map[string]interface{}) (*dto.ServiceTaskResult, error) {
	h.calls++
	return &dto.ServiceTaskResult{Success: true, OutputVars: map[string]interface{}{"svcResult": "ok"}}, nil
}

func TestParallelFork_ServiceTaskBranchBeforeUserTaskBranch(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	tenantID := createTimerTenant(t, client, "parsvc")
	dep, err := client.ProcessDeployment.Create().SetDeploymentID("DEP-PAR-SVC").SetDeploymentName("par-svc").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().SetKey("parallelServiceFirst").SetName("par-svc").SetBpmnXML([]byte(parallelServiceFirstBPMN)).
		SetDeploymentID(dep.ID).SetTenantID(tenantID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)

	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	handler := &parallelServiceTestHandler{}
	engine.callbackRegistry.RegisterHandler(handler)
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)

	actor, err := client.User.Create().SetUsername("par_svc_actor").SetEmail("par_svc_actor@test.com").
		SetName("Par Actor").SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, actor.ID)

	instance, err := engine.StartProcess(workflowCtx, "parallelServiceFirst", "par-svc:1", map[string]interface{}{})
	require.NoError(t, err)

	// 服务任务分支先派发：命令已入队；用户任务分支后派发并覆盖 CurrentActivityID。
	stored, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	require.Equal(t, "user_branch", stored.CurrentActivityID,
		"前置条件：用户任务分支后派发并覆盖了 CurrentActivityID")

	cmd, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID),
		operationalcommand.CommandTypeEQ(commandbus.CommandExecuteBPMNServiceTask),
		operationalcommand.AggregateIDEQ(instance.ID),
	).Only(ctx)
	require.NoError(t, err)

	registry := commandbus.NewRegistry()
	require.NoError(t, registry.Register(commandbus.CommandExecuteBPMNServiceTask, engine.HandleBPMNServiceTaskCommand))
	_, err = commandbus.NewWorker(client, registry, zaptest.NewLogger(t).Sugar(), "par-svc-worker").RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, handler.calls, "ServiceTask 必须被真正执行，不得被位置校验误杀")

	got, err := client.OperationalCommand.Get(ctx, cmd.ID)
	require.NoError(t, err)
	require.Equal(t, commandbus.StatusSucceeded, got.Status)

	after, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	require.Equal(t, "running", after.Status, "分支B任务待办，实例应继续运行")
	doneRaw, _ := after.Variables["_done_"].(map[string]interface{})
	require.Equal(t, true, doneRaw["svc_branch"], "服务任务分支必须已标记完成，否则汇聚死锁")

	// 分支B完成后，join 汇聚 → end，实例完成。
	task, err := client.ProcessTask.Query().
		Where(
			processtask.ProcessInstanceID(instance.ID),
			processtask.TaskDefinitionKey("user_branch"),
			processtask.StatusNotIn("completed", "cancelled"),
		).
		Only(ctx)
	require.NoError(t, err)
	claimed, err := client.ProcessTask.UpdateOne(task).SetAssignee(fmt.Sprintf("%d", actor.ID)).Save(ctx)
	require.NoError(t, err)
	require.NoError(t, engine.CompleteTask(workflowCtx, claimed.TaskID, map[string]interface{}{"approvalAction": "approve"}))

	final, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", final.Status, "服务任务分支先派发时全流程必须能走完，不得死锁")
}
