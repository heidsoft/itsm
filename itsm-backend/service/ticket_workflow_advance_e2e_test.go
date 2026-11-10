package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// 两个节点的最小流程：起流停在「任务分配」，完成一次作业待办后到「工单处理」。
// 与内置 ticket_general_flow 的前两段同构，避免把模板细节编进测试断言。
const ticketWorkflowAdvanceXML = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/bpmn/20100524/MODEL" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="ticket_advance_e2e" name="Ticket Advance E2E" isExecutable="true">
    <bpmn:startEvent id="start" name="工单创建"/>
    <bpmn:userTask id="assign" name="任务分配"/>
    <bpmn:userTask id="handle" name="工单处理"/>
    <bpmn:endEvent id="end" name="工单关闭"/>
    <bpmn:sequenceFlow id="flow_start" sourceRef="start" targetRef="assign"/>
    <bpmn:sequenceFlow id="flow_handle" sourceRef="assign" targetRef="handle"/>
    <bpmn:sequenceFlow id="flow_end" sourceRef="handle" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

// 审批节点版本：用于证明生命周期动作绝不代签审批。
const ticketWorkflowApprovalNodeXML = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/bpmn/20100524/MODEL" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="ticket_approval_node_e2e" name="Ticket Approval Node E2E" isExecutable="true">
    <bpmn:startEvent id="start" name="工单创建"/>
    <bpmn:userTask id="approve" name="工单审批"/>
    <bpmn:endEvent id="end" name="工单关闭"/>
    <bpmn:sequenceFlow id="flow_start" sourceRef="start" targetRef="approve"/>
    <bpmn:sequenceFlow id="flow_end" sourceRef="approve" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

type ticketWorkflowAdvanceFixture struct {
	client      *ent.Client
	ctx         context.Context
	actorCtx    context.Context
	logger      *zap.SugaredLogger
	tenantID    int
	actorID     int
	engine      *CustomProcessEngine
	workflowSvc *TicketWorkflowService
}

func newTicketWorkflowAdvanceFixture(t *testing.T) *ticketWorkflowAdvanceFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3", testDSN())
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().SetName("工单流程推进 E2E").SetCode("ticket-advance-e2e").
		SetDomain("advance.e2e").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	actor, err := client.User.Create().SetUsername("ticket_advance_actor").
		SetEmail("advance@example.com").SetName("处理人").SetPasswordHash("hash").
		SetRole("technician").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	deployment, err := client.ProcessDeployment.Create().SetDeploymentID("dep-ticket-advance-e2e").
		SetDeploymentName("Ticket Advance E2E").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	for key, xml := range map[string][]byte{
		"ticket_advance_e2e":       []byte(ticketWorkflowAdvanceXML),
		"ticket_approval_node_e2e": []byte(ticketWorkflowApprovalNodeXML),
	} {
		_, err := client.ProcessDefinition.Create().SetKey(key).SetName(key).
			SetBpmnXML(xml).SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).
			SetIsActive(true).SetIsLatest(true).Save(ctx)
		require.NoError(t, err)
	}

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, actor.ID)

	return &ticketWorkflowAdvanceFixture{
		client: client, ctx: ctx, logger: logger, tenantID: tenant.ID, actorID: actor.ID,
		engine: engine, workflowSvc: NewTicketWorkflowService(client, logger),
		actorCtx: workflowCtx,
	}
}

// actorCtx 是带 BPMN 租户/操作人身份的上下文，起流时用它，保证任务归属与
// 后续业务动作的操作人一致（否则引擎会按 assignee 拒绝，测试变成假绿）。
func (f *ticketWorkflowAdvanceFixture) actorCtxValue() context.Context { return f.actorCtx }

func (f *ticketWorkflowAdvanceFixture) createTicket(t *testing.T, number string) *ent.Ticket {
	t.Helper()
	tk, err := f.client.Ticket.Create().SetTitle("流程推进回归锁").SetDescription("工单状态与流程节点必须原子一致").
		SetStatus("new").SetType("incident").SetPriority("medium").SetTicketNumber(number).
		SetRequesterID(f.actorID).SetTenantID(f.tenantID).SetVersion(1).Save(f.ctx)
	require.NoError(t, err)
	return tk
}

func (f *ticketWorkflowAdvanceFixture) startProcess(t *testing.T, key string, ticketID int) *ent.ProcessInstance {
	t.Helper()
	instance, err := f.engine.StartProcess(f.actorCtxValue(), key, fmt.Sprintf("ticket:%d", ticketID),
		map[string]interface{}{"business_id": ticketID, "ticket_id": ticketID})
	require.NoError(t, err)
	return instance
}

func (f *ticketWorkflowAdvanceFixture) instanceFor(t *testing.T, ticketID int) *ent.ProcessInstance {
	t.Helper()
	instance, err := f.client.ProcessInstance.Query().
		Where(processinstance.TenantIDEQ(f.tenantID), processinstance.BusinessKeyEQ(fmt.Sprintf("ticket:%d", ticketID))).
		Only(f.ctx)
	require.NoError(t, err)
	return instance
}

// TestAcceptTicketAdvancesBPMNInSameTransaction 是 P0 的回归锁：
// 旧实现只改 tickets 表，流程实例永远停在「任务分配」（QA 实测 #8 悬挂）。
// 断言必须同时命中「工单状态」与「流程节点」，缺一半都会让老代码通过。
func TestAcceptTicketAdvancesBPMNInSameTransaction(t *testing.T) {
	f := newTicketWorkflowAdvanceFixture(t)
	tk := f.createTicket(t, "TKT-ADVANCE-1")
	instance := f.startProcess(t, "ticket_advance_e2e", tk.ID)
	// StartProcess 返回的实例对象仍是创建时的 startEvent 节点，流程推进只落库；
	// 前置基线必须从数据库读，否则断言的是引擎实现细节而非流程状态。
	persisted := f.instanceFor(t, tk.ID)
	require.Equal(t, "assign", persisted.CurrentActivityID, "起流后必须停在任务分配节点")

	err := f.workflowSvc.AcceptTicket(f.ctx, &dto.AcceptTicketRequest{TicketID: tk.ID}, f.actorID, f.tenantID)
	require.NoError(t, err)

	updatedTicket, err := f.client.Ticket.Get(f.ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "in_progress", updatedTicket.Status, "业务状态必须推进")

	updatedInstance, err := f.client.ProcessInstance.Get(f.ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "handle", updatedInstance.CurrentActivityID, "流程节点必须同时推进")

	finished, err := f.client.ProcessTask.Query().
		Where(processtask.ProcessInstanceIDEQ(instance.ID), processtask.TaskDefinitionKeyEQ("assign")).Only(f.ctx)
	require.NoError(t, err)
	assert.Equal(t, "completed", finished.Status, "当前作业待办必须被真实完成")
}

// TestAcceptTicketRefusesToSignApprovalNode 证明桥接绝不代签审批：
// 停在「工单审批」时接单必须 409，且工单与流程都原样回滚。
func TestAcceptTicketRefusesToSignApprovalNode(t *testing.T) {
	f := newTicketWorkflowAdvanceFixture(t)
	tk := f.createTicket(t, "TKT-ADVANCE-2")
	instance := f.startProcess(t, "ticket_approval_node_e2e", tk.ID)
	// StartProcess 返回的是创建时的内存对象（CurrentActivityID 仍是 startEvent.ID），
	// 流程推进只落库、不回写该对象；回归锁以数据库里的审批待办为事实源，
	// 不把「引擎是否刷新返回对象」这种实现细节编进断言。
	approvalTask, err := f.client.ProcessTask.Query().
		Where(processtask.ProcessInstanceIDEQ(instance.ID), processtask.TaskDefinitionKeyEQ("approve")).Only(f.ctx)
	require.NoError(t, err, "起流必须停在审批节点并生成审批待办")
	require.NotEqual(t, "completed", approvalTask.Status)

	err = f.workflowSvc.AcceptTicket(f.ctx, &dto.AcceptTicketRequest{TicketID: tk.ID}, f.actorID, f.tenantID)
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr), "必须是可行动的 409，不能被兜底成 500")
	assert.Equal(t, common.ConflictCode, bizErr.Code)

	unchangedTicket, err := f.client.Ticket.Get(f.ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "new", unchangedTicket.Status, "业务写入必须随事务回滚")
	pending, err := f.client.ProcessTask.Query().
		Where(processtask.ProcessInstanceIDEQ(instance.ID), processtask.TaskDefinitionKeyEQ("approve")).Only(f.ctx)
	require.NoError(t, err)
	assert.NotEqual(t, "completed", pending.Status, "审批待办不允许被生命周期动作消费")
}

// TestAcceptTicketWithoutBoundWorkflowKeepsLegacyBehaviour 保护存量工单：
// 从未绑定流程（旧数据 / 未配置租户）时仍可照常接单，不能因为桥接而 4xx。
func TestAcceptTicketWithoutBoundWorkflowKeepsLegacyBehaviour(t *testing.T) {
	f := newTicketWorkflowAdvanceFixture(t)
	tk := f.createTicket(t, "TKT-ADVANCE-3")

	require.NoError(t, f.workflowSvc.AcceptTicket(f.ctx, &dto.AcceptTicketRequest{TicketID: tk.ID}, f.actorID, f.tenantID))
	updated, err := f.client.Ticket.Get(f.ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "in_progress", updated.Status)
}

// assignTaskTo 把起流后待办改成指定的归属形态，用于验证桥接的安全边界。
func (f *ticketWorkflowAdvanceFixture) assignTaskTo(t *testing.T, instanceID int, assignee string, source string) {
	t.Helper()
	task, err := f.client.ProcessTask.Query().
		Where(processtask.ProcessInstanceIDEQ(instanceID), processtask.TaskDefinitionKeyEQ("assign")).Only(f.ctx)
	require.NoError(t, err)
	vars := map[string]interface{}{}
	for k, v := range task.TaskVariables {
		vars[k] = v
	}
	vars["assignmentSource"] = source
	_, err = task.Update().SetAssignee(assignee).SetTaskVariables(vars).Save(f.ctx)
	require.NoError(t, err)
}

// TestAcceptTicketHonoursTriggeredByFallback 模拟生产起流形态：引擎把 triggered_by
// （创建人）回落成 assignee，接单人是另一名技术员。业务域已完成资格校验，
// 受控信任标记应放行，工单与流程一同推进（否则生产里接单恒 403）。
func TestAcceptTicketHonoursTriggeredByFallback(t *testing.T) {
	f := newTicketWorkflowAdvanceFixture(t)
	tk := f.createTicket(t, "TKT-ADVANCE-4")
	instance := f.startProcess(t, "ticket_advance_e2e", tk.ID)
	creator, err := f.client.User.Create().SetUsername("ticket_advance_creator").
		SetEmail("creator@example.com").SetName("创建人").SetPasswordHash("hash").
		SetRole("end_user").SetActive(true).SetTenantID(f.tenantID).Save(f.ctx)
	require.NoError(t, err)
	f.assignTaskTo(t, instance.ID, strconv.Itoa(creator.ID), "triggered_by")

	require.NoError(t, f.workflowSvc.AcceptTicket(f.ctx, &dto.AcceptTicketRequest{TicketID: tk.ID}, f.actorID, f.tenantID))
	updated, err := f.client.ProcessInstance.Get(f.ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "handle", updated.CurrentActivityID)
}

// TestAcceptTicketDoesNotOverrideClaimedTask 被他人主动认领的任务仍严格校验归属：
// 新增的「未指派」兜底不得扩大成「任何任务都能推进」。
func TestAcceptTicketDoesNotOverrideClaimedTask(t *testing.T) {
	f := newTicketWorkflowAdvanceFixture(t)
	tk := f.createTicket(t, "TKT-ADVANCE-5")
	instance := f.startProcess(t, "ticket_advance_e2e", tk.ID)
	other, err := f.client.User.Create().SetUsername("ticket_advance_other").
		SetEmail("other@example.com").SetName("其他处理人").SetPasswordHash("hash").
		SetRole("technician").SetActive(true).SetTenantID(f.tenantID).Save(f.ctx)
	require.NoError(t, err)
	f.assignTaskTo(t, instance.ID, strconv.Itoa(other.ID), "claimed")

	err = f.workflowSvc.AcceptTicket(f.ctx, &dto.AcceptTicketRequest{TicketID: tk.ID}, f.actorID, f.tenantID)
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr), "必须是可行动的 403")
	assert.Equal(t, common.ForbiddenErrorCode, bizErr.Code)

	unchanged, err := f.client.Ticket.Get(f.ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "new", unchanged.Status, "桥接失败时业务写入必须整体回滚")
}
