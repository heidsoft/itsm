package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const durableServiceTaskBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="Process_durable_service" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1"/>
    <bpmn:serviceTask id="ServiceTask_1" implementation="test_durable_handler"/>
    <bpmn:endEvent id="EndEvent_1"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="ServiceTask_1"/>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="ServiceTask_1" targetRef="EndEvent_1"/>
  </bpmn:process>
</bpmn:definitions>`

type durableTestHandler struct {
	calls          int
	idempotencyKey string
}

func (h *durableTestHandler) GetTaskType() string                                    { return "ServiceTask" }
func (h *durableTestHandler) GetHandlerID() string                                   { return "test_durable_handler" }
func (h *durableTestHandler) Validate(context.Context, map[string]interface{}) error { return nil }
func (h *durableTestHandler) Execute(_ context.Context, _ *ent.ProcessTask, variables map[string]interface{}) (*dto.ServiceTaskResult, error) {
	h.calls++
	h.idempotencyKey, _ = variables["_idempotencyKey"].(string)
	return &dto.ServiceTaskResult{Success: true, OutputVars: map[string]interface{}{"serviceResult": "ok"}}, nil
}

func TestStartProcessEnqueuesServiceTaskAndWorkerAdvancesAfterCommit(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	tenantID := 31
	deployment, err := client.ProcessDeployment.Create().SetDeploymentID("DEP-DURABLE-SERVICE").SetDeploymentName("durable").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().SetKey("durableService").SetName("durable").SetBpmnXML([]byte(durableServiceTaskBPMN)).
		SetDeploymentID(deployment.ID).SetTenantID(tenantID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	fake := &durableTestHandler{}
	engine.callbackRegistry.RegisterHandler(fake)
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)

	instance, err := engine.StartProcess(workflowCtx, "durableService", "change:42", map[string]interface{}{})
	require.NoError(t, err)
	require.Equal(t, 0, fake.calls, "ServiceTask must not execute inside StartProcess transaction")
	cmd, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenantID),
		operationalcommand.CommandTypeEQ(commandbus.CommandExecuteBPMNServiceTask),
		operationalcommand.AggregateIDEQ(instance.ID),
	).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, commandbus.StatusPending, cmd.Status)

	registry := commandbus.NewRegistry()
	require.NoError(t, registry.Register(commandbus.CommandExecuteBPMNServiceTask, engine.HandleBPMNServiceTaskCommand))
	processed, err := commandbus.NewWorker(client, registry, zaptest.NewLogger(t).Sugar(), "bpmn-test-worker").RunOnce(ctx)
	require.NoError(t, err)
	require.True(t, processed)
	require.Equal(t, 1, fake.calls)
	require.Equal(t, cmd.IdempotencyKey, fake.idempotencyKey)
	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", completed.Status)
	require.Equal(t, "ok", completed.Variables["serviceResult"])

	storedCommand, err := client.OperationalCommand.Get(ctx, cmd.ID)
	require.NoError(t, err)
	require.Equal(t, commandbus.StatusSucceeded, storedCommand.Status)
	require.NoError(t, engine.HandleBPMNServiceTaskCommand(workflowCtx, storedCommand))
	require.Equal(t, 1, fake.calls, "already applied command must not repeat external callback")
}

func TestBPMNServiceTaskCommandRejectsCrossTenantInstance(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	err := engine.HandleBPMNServiceTaskCommand(context.Background(), &ent.OperationalCommand{
		TenantID: 99, CommandType: commandbus.CommandExecuteBPMNServiceTask,
		AggregateType: "process_instance", AggregateID: 123,
		Payload: map[string]interface{}{"elementId": "ServiceTask_1", "serviceRef": "test_durable_handler", "occurrence": 1},
	})
	require.Error(t, err)
}
