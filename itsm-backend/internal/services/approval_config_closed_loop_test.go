package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"

	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApprovalConfigClosedLoop_ProcessBindingToTaskVariables 验证审批配置闭环：
// TicketType.WorkflowDefinitionKey → BPMN 流程启动 → ProcessTask 创建 → TaskVariables 正确传播 → 运行时能力可用
func TestApprovalConfigClosedLoop_ProcessBindingToTaskVariables(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:approval_closed_loop?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	// 1. 创建租户和用户
	tenant, err := client.Tenant.Create().
		SetName("Closed Loop Tenant").SetCode("closed-loop").SetDomain("closed-loop.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	requester, err := client.User.Create().
		SetUsername("requester").SetEmail("requester@test.com").SetName("申请人").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	approver1, err := client.User.Create().
		SetUsername("approver1").SetEmail("approver1@test.com").SetName("审批人1").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	approver2, err := client.User.Create().
		SetUsername("approver2").SetEmail("approver2@test.com").SetName("审批人2").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 2. 创建流程部署和定义（BPMN XML 中包含 allowDelegate 和 allowAddApprover 属性）
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-CLOSED-LOOP").SetDeploymentName("Closed Loop Deploy").
		SetIsActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	bpmnXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:itsm="http://itsm.example.com/bpmn"
                  targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="ticket_approval_flow" name="工单审批流程" isExecutable="true">
    <bpmn:startEvent id="Start"/>
    <bpmn:userTask id="ApprovalTask" name="审批" itsm:taskPurpose="approval" itsm:assignee="` + approver1.Username + `" itsm:allowDelegate="true" itsm:allowAddApprover="true">
      <bpmn:incoming>Flow1</bpmn:incoming>
      <bpmn:outgoing>Flow2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="End"/>
    <bpmn:sequenceFlow id="Flow1" sourceRef="Start" targetRef="ApprovalTask"/>
    <bpmn:sequenceFlow id="Flow2" sourceRef="ApprovalTask" targetRef="End"/>
  </bpmn:process>
</bpmn:definitions>`

	_, err = client.ProcessDefinition.Create().
		SetKey("ticket_approval_flow").SetName("工单审批流程").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).SetBpmnXML([]byte(bpmnXML)).
		SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 3. 创建 TicketType 并绑定 WorkflowDefinitionKey
	// 工单创建时通过 TicketType.WorkflowDefinitionKey 解析到正确的流程定义
	typeService := NewTicketTypeService(client, logger)
	configuredType, err := typeService.CreateTicketType(ctx, &dto.CreateTicketTypeRequest{
		Code:                  "approval_test",
		Name:                  "Approval Test Type",
		WorkflowDefinitionKey: "ticket_approval_flow",
	}, tenant.ID, requester.ID)
	require.NoError(t, err)

	// 4. 创建工单服务并触发流程
	ticketService := NewTicketServiceForTest(client, logger)
	engine := NewCustomProcessEngine(client, logger)
	ticketService.SetProcessTriggerService(NewProcessTriggerService(client, engine))

	created, err := ticketService.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:        "审批配置闭环测试",
		Description:  "验证 TicketType.WorkflowDefinitionKey → BPMN → TaskVariables 完整链路",
		Priority:     "medium",
		RequesterID:  requester.ID,
		TicketTypeID: &configuredType.ID,
	}, tenant.ID)
	require.NoError(t, err)

	// 5. 验证 BPMN 流程实例已创建
	ticketBusinessKey := fmt.Sprintf("ticket:%d", created.ID)
	var instances []*ent.ProcessInstance
	require.Eventually(t, func() bool {
		instances, err = client.ProcessInstance.Query().
			Where(
				processinstance.TenantIDEQ(tenant.ID),
				processinstance.BusinessKeyEQ(ticketBusinessKey),
			).
			All(ctx)
		return err == nil && len(instances) == 1
	}, 3*time.Second, 20*time.Millisecond, "应恰好创建一个 ProcessInstance")

	instance := instances[0]
	assert.Equal(t, "running", instance.Status)
	assert.Equal(t, "ticket_approval_flow", instance.ProcessDefinitionKey)

	// 6. 验证 ProcessTask 已创建且 TaskVariables 包含正确的审批能力配置
	var tasks []*ent.ProcessTask
	require.Eventually(t, func() bool {
		tasks, err = client.ProcessTask.Query().
			Where(
				processtask.TenantIDEQ(tenant.ID),
				processtask.ProcessInstanceIDEQ(instance.ID),
			).
			All(ctx)
		return err == nil && len(tasks) >= 1
	}, 3*time.Second, 20*time.Millisecond, "应创建至少一个 ProcessTask")

	// 找到审批任务
	var approvalTask *ent.ProcessTask
	for _, task := range tasks {
		if task.TaskDefinitionKey == "ApprovalTask" {
			approvalTask = task
			break
		}
	}
	require.NotNil(t, approvalTask, "应找到 ApprovalTask")

	// 7. 验证 TaskVariables 正确传播
	assert.Equal(t, "approval", approvalTask.TaskVariables["taskPurpose"])
	assert.Equal(t, true, approvalTask.TaskVariables["allowDelegate"])
	assert.Equal(t, true, approvalTask.TaskVariables["allowAddApprover"])
	assert.Equal(t, approver1.Username, approvalTask.Assignee)
	// BPMN 引擎创建任务时初始状态为 "created"（非 "assigned"），委托后才转为 "assigned"
	assert.Equal(t, "created", approvalTask.Status)

	// 8. 验证运行时能力：委托（allowDelegate=true 应成功）
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, approver1.ID)

	err = engine.TaskService().DelegateTask(workflowCtx, approvalTask.TaskID, approver2.Username)
	require.NoError(t, err, "allowDelegate=true 时委托应成功")

	// 验证委托后任务状态
	updatedTask, err := engine.TaskService().GetTask(workflowCtx, approvalTask.TaskID)
	require.NoError(t, err)
	assert.Equal(t, approver2.Username, updatedTask.Assignee)
	assert.Equal(t, "assigned", updatedTask.Status)
	assert.Equal(t, approver1.Username, updatedTask.TaskVariables["delegated_from"])

	// 9. 验证审计记录已写入
	decisions, err := client.ProcessApprovalDecision.Query().All(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(decisions), 1)
	found := false
	for _, d := range decisions {
		if d.Action == "delegate" && d.Decision == "delegated" {
			found = true
			break
		}
	}
	assert.True(t, found, "应写入 delegate 审计记录")
}

// TestApprovalConfigClosedLoop_AllowDelegateFalse 验证 allowDelegate=false 时委托应失败
func TestApprovalConfigClosedLoop_AllowDelegateFalse(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:approval_closed_loop_no_delegate?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("No Delegate Tenant").SetCode("no-delegate").SetDomain("no-delegate.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	requester, err := client.User.Create().
		SetUsername("requester").SetEmail("requester@test.com").SetName("申请人").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	approver, err := client.User.Create().
		SetUsername("approver").SetEmail("approver@test.com").SetName("审批人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-NO-DELEGATE").SetDeploymentName("No Delegate Deploy").
		SetIsActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// BPMN XML 中 allowDelegate=false
	bpmnXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:itsm="http://itsm.example.com/bpmn"
                  targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="no_delegate_flow" name="禁止委托流程" isExecutable="true">
    <bpmn:startEvent id="Start"/>
    <bpmn:userTask id="ApprovalTask" name="审批" itsm:taskPurpose="approval" itsm:assignee="` + approver.Username + `" itsm:allowDelegate="false">
      <bpmn:incoming>Flow1</bpmn:incoming>
      <bpmn:outgoing>Flow2</bpmn:outgoing>
    </bpmn:userTask>
    <bpmn:endEvent id="End"/>
    <bpmn:sequenceFlow id="Flow1" sourceRef="Start" targetRef="ApprovalTask"/>
    <bpmn:sequenceFlow id="Flow2" sourceRef="ApprovalTask" targetRef="End"/>
  </bpmn:process>
</bpmn:definitions>`

	_, err = client.ProcessDefinition.Create().
		SetKey("no_delegate_flow").SetName("禁止委托流程").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).SetBpmnXML([]byte(bpmnXML)).
		SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建 TicketType 并绑定 WorkflowDefinitionKey
	typeService := NewTicketTypeService(client, logger)
	configuredType, err := typeService.CreateTicketType(ctx, &dto.CreateTicketTypeRequest{
		Code:                  "no_delegate_type",
		Name:                  "No Delegate Type",
		WorkflowDefinitionKey: "no_delegate_flow",
	}, tenant.ID, requester.ID)
	require.NoError(t, err)

	ticketService := NewTicketServiceForTest(client, logger)
	engine := NewCustomProcessEngine(client, logger)
	ticketService.SetProcessTriggerService(NewProcessTriggerService(client, engine))

	created, err := ticketService.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title:        "禁止委托测试",
		Description:  "验证 allowDelegate=false 时委托失败",
		Priority:     "medium",
		RequesterID:  requester.ID,
		TicketTypeID: &configuredType.ID,
	}, tenant.ID)
	require.NoError(t, err)

	ticketBusinessKey := fmt.Sprintf("ticket:%d", created.ID)
	var instances []*ent.ProcessInstance
	require.Eventually(t, func() bool {
		instances, _ = client.ProcessInstance.Query().
			Where(processinstance.BusinessKeyEQ(ticketBusinessKey)).
			All(ctx)
		return len(instances) == 1
	}, 3*time.Second, 20*time.Millisecond)

	var tasks []*ent.ProcessTask
	require.Eventually(t, func() bool {
		tasks, _ = client.ProcessTask.Query().
			Where(processtask.ProcessInstanceIDEQ(instances[0].ID)).
			All(ctx)
		return len(tasks) >= 1
	}, 3*time.Second, 20*time.Millisecond)

	var approvalTask *ent.ProcessTask
	for _, task := range tasks {
		if task.TaskDefinitionKey == "ApprovalTask" {
			approvalTask = task
			break
		}
	}
	require.NotNil(t, approvalTask)

	// allowDelegate=false 时委托应失败
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, approver.ID)

	err = engine.TaskService().DelegateTask(workflowCtx, approvalTask.TaskID, "other_user")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不允许委托")
}
