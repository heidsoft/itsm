package service

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent/auditlog"
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

// TestTicketTypeFullChain_InstallToAuditE2E 覆盖完整业务链路断言：
// TicketType 预设安装 → 绑定 Workflow/SLA → 工单创建 → SLA 期限计算 →
// Workflow 引擎实例 → 任务自动分配 → 流程审计 + 业务审计。
func TestTicketTypeFullChain_InstallToAuditE2E(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	// --- 基础数据：租户 / 申请人 / BPMN 部署 ---
	tenant, err := client.Tenant.Create().SetName("Full Chain E2E").SetCode("full-chain-e2e").SetDomain("fullchain.e2e").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	requester, err := client.User.Create().SetUsername("chain_requester").SetEmail("chain@example.com").SetName("Chain Requester").SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	radiologyDept, err := client.Department.Create().SetName("放射科").SetCode("radiology").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	deployment, err := client.ProcessDeployment.Create().SetDeploymentID("dep-full-chain-e2e").SetDeploymentName("Full Chain E2E").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().SetKey("ticket_type_e2e").SetName("Ticket Type E2E").SetBpmnXML([]byte(ticketTypeWorkflowE2E)).SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)

	// --- 环节 1: TicketType 预设安装（租户隔离 + preset.install 审计） ---
	typeService := NewTicketTypeService(client, logger)
	installed, err := typeService.InstallPreset(ctx, "pacs-incident", &dto.InstallTicketTypePresetRequest{}, tenant.ID, 9)
	require.NoError(t, err)
	assert.Equal(t, tenant.ID, installed.TenantID)
	assert.Equal(t, "pacs_incident", installed.Code)
	require.NotEmpty(t, installed.CustomFields)
	installLogs, err := client.AuditLog.Query().Where(auditlog.TenantIDEQ(tenant.ID), auditlog.ActionEQ("preset.install")).All(ctx)
	require.NoError(t, err)
	require.Len(t, installLogs, 1)
	assert.Contains(t, *installLogs[0].RequestBody, `"presetId":"pacs-incident"`)

	// --- SLA 定义（租户内，供工单类型绑定） ---
	slaDef, err := client.SLADefinition.Create().
		SetName("PACS 高优先级 SLA").
		SetServiceType("incident").
		SetPriority("high").
		SetResponseTime(30).
		SetResolutionTime(240).
		SetIsActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// --- 环节 1b: 绑定 Workflow + SLA（binding.update 审计） ---
	slaEnabled := true
	workflowKey := "ticket_type_e2e"
	updated, err := typeService.UpdateTicketType(ctx, installed.ID, &dto.UpdateTicketTypeRequest{
		WorkflowDefinitionKey: &workflowKey,
		SLAEnabled:            &slaEnabled,
		DefaultSLAID:          &slaDef.ID,
	}, tenant.ID, 10)
	require.NoError(t, err)
	assert.Equal(t, "ticket_type_e2e", updated.WorkflowDefinitionKey)
	assert.True(t, updated.SLAEnabled)

	// --- 环节 2: 工单创建（绑定 TicketType，触发 Workflow） ---
	engine := NewCustomProcessEngine(client, logger)
	ticketService := NewTicketServiceForTest(client, logger)
	ticketService.SetProcessTriggerService(NewProcessTriggerService(client, engine))
	ticketService.SetSLAService(NewTicketSLAService(client, logger))
	created, err := ticketService.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:        "Full Chain E2E",
		Description:  "install->create->workflow->sla->assignment->audit",
		Priority:     "medium",
		RequesterID:  requester.ID,
		TicketTypeID: &installed.ID,
		FormFields: map[string]interface{}{
			"pacsNode":           "node-01",
			"affectedDepartment": radiologyDept.ID,
		},
	}, tenant.ID)
	require.NoError(t, err)

	// --- 环节 3: SLA 断言（preset 默认优先级 high 覆盖请求的 medium；期限来自绑定的 SLA 定义） ---
	assert.Equal(t, "high", string(created.Priority))
	ticketRow, err := client.Ticket.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, slaDef.ID, ticketRow.SLADefinitionID)
	assert.False(t, ticketRow.SLAResponseDeadline.IsZero())
	assert.False(t, ticketRow.SLAResolutionDeadline.IsZero())
	require.WithinDuration(t, time.Now().Add(30*time.Minute), ticketRow.SLAResponseDeadline, 2*time.Minute)
	require.WithinDuration(t, time.Now().Add(240*time.Minute), ticketRow.SLAResolutionDeadline, 2*time.Minute)

	// --- 环节 4: Workflow 引擎实例（businessKey 绑定工单） ---
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

	// --- 环节 5: Assignment 断言（requester_id 流程变量自动分配给申请人） ---
	task, err := client.ProcessTask.Query().Where(processtask.TenantIDEQ(tenant.ID), processtask.ProcessInstanceIDEQ(instanceID), processtask.TaskDefinitionKeyEQ("handle")).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, strconv.Itoa(requester.ID), task.Assignee)

	// --- 完成任务 → 实例结束断言 ---
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)
	require.NoError(t, engine.CompleteTask(workflowCtx, task.TaskID, map[string]interface{}{"resolution": "done"}))
	instance, err := client.ProcessInstance.Get(ctx, instanceID)
	require.NoError(t, err)
	assert.Equal(t, "completed", instance.Status)
	assert.Equal(t, "end", instance.CurrentActivityID)
	completedTask, err := client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completedTask.Status)

	// --- 环节 6a: 流程审计（ProcessAuditLog） ---
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

	// --- 环节 6b: 业务审计（AuditLog：preset.install + binding.update） ---
	businessLogs, err := client.AuditLog.Query().Where(auditlog.TenantIDEQ(tenant.ID)).All(ctx)
	require.NoError(t, err)
	businessActions := make([]string, 0, len(businessLogs))
	for _, entry := range businessLogs {
		businessActions = append(businessActions, entry.Action)
	}
	assert.Contains(t, businessActions, "preset.install")
	assert.Contains(t, businessActions, "binding.update")
}
