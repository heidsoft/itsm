package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processauditlog"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const ticketTypeWorkflowE2E = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="ticket_type_e2e" name="Ticket Type E2E" isExecutable="true">
    <bpmn:startEvent id="start" name="Start"/>
    <bpmn:userTask id="handle" name="Handle Ticket"/>
    <bpmn:endEvent id="end" name="End"/>
    <bpmn:sequenceFlow id="flow_start" sourceRef="start" targetRef="handle"/>
    <bpmn:sequenceFlow id="flow_end" sourceRef="handle" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

func TestTicketTypeWorkflowBindingCreatesAndCompletesEngineInstanceE2E(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().SetName("Workflow E2E").SetCode("workflow-e2e").SetDomain("workflow.e2e").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	requester, err := client.User.Create().SetUsername("workflow_requester").SetEmail("workflow@example.com").SetName("Workflow Requester").SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	deployment, err := client.ProcessDeployment.Create().SetDeploymentID("dep-ticket-type-e2e").SetDeploymentName("Ticket Type E2E").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().SetKey("ticket_type_e2e").SetName("Ticket Type E2E").SetBpmnXML([]byte(ticketTypeWorkflowE2E)).SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)

	typeService := NewTicketTypeService(client, logger)
	configuredType, err := typeService.CreateTicketType(ctx, &dto.CreateTicketTypeRequest{Code: "workflow_ticket", Name: "Workflow Ticket", WorkflowDefinitionKey: "ticket_type_e2e"}, tenant.ID, requester.ID)
	require.NoError(t, err)

	engine := NewCustomProcessEngine(client, logger)
	ticketService := NewTicketServiceForTest(client, logger)
	ticketService.SetProcessTriggerService(NewProcessTriggerService(client, engine))
	created, err := ticketService.CreateTicket(ctx, &dto.CreateTicketRequest{Title: "Engine E2E", Description: "TicketType binding must start the engine", Priority: "medium", RequesterID: requester.ID, TicketTypeID: &configuredType.ID}, tenant.ID)
	require.NoError(t, err)

	businessKey := fmt.Sprintf("ticket:%d", created.ID)
	var instanceID int
	require.Eventually(t, func() bool {
		instance, queryErr := client.ProcessInstance.Query().Where(processinstance.TenantIDEQ(tenant.ID), processinstance.BusinessKeyEQ(businessKey)).Only(ctx)
		if queryErr != nil {
			return false
		}
		instanceID = instance.ID
		return instance.ProcessDefinitionKey == "ticket_type_e2e" && instance.Status == "running" && instance.CurrentActivityID == "handle"
	}, 3*time.Second, 20*time.Millisecond)

	task, err := client.ProcessTask.Query().Where(processtask.TenantIDEQ(tenant.ID), processtask.ProcessInstanceIDEQ(instanceID), processtask.TaskDefinitionKeyEQ("handle")).Only(ctx)
	require.NoError(t, err)
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)
	require.NoError(t, engine.CompleteTask(workflowCtx, task.TaskID, map[string]interface{}{"resolution": "done"}))

	instance, err := client.ProcessInstance.Get(ctx, instanceID)
	require.NoError(t, err)
	assert.Equal(t, "completed", instance.Status)
	assert.Equal(t, "end", instance.CurrentActivityID)
	completedTask, err := client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completedTask.Status)
	auditTrail, err := client.ProcessAuditLog.Query().Where(processauditlog.TenantIDEQ(tenant.ID), processauditlog.ProcessInstanceIDEQ(instance.ID)).All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, auditTrail)
	actions := make([]string, 0, len(auditTrail))
	for _, entry := range auditTrail {
		actions = append(actions, entry.Action)
		assert.Equal(t, tenant.ID, entry.TenantID)
	}
	assert.Contains(t, actions, "started")
	assert.Contains(t, actions, "completed")
}
