package service

import (
	"errors"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件是 F1/O2 的回归锁：prod v1.6.57 浏览器实测「工单已关闭、流程已结束」，
// 但 ticket:8 实例仍 running、审批中心仍有待办；对任务点完成后引擎又新建
// 「升级处理」任务。旧实现（只完成一个作业待办）在下面每个用例里都会红。

// resolveTicketForClose 只把工单推到 resolved，隔离「关闭」动作本身的语义，
// 不让 AcceptTicket / ResolveTicket 的流程推进混进前置条件。
func resolveTicketForClose(t *testing.T, f *ticketWorkflowAdvanceFixture, ticketID int) {
	t.Helper()
	_, err := f.client.Ticket.UpdateOneID(ticketID).SetStatus("resolved").Save(f.ctx)
	require.NoError(t, err)
}

func countActiveProcessTasks(t *testing.T, f *ticketWorkflowAdvanceFixture, instanceID int) int {
	t.Helper()
	n, err := f.client.ProcessTask.Query().Where(
		processtask.ProcessInstanceIDEQ(instanceID),
		processtask.StatusNotIn("completed", "cancelled"),
	).Count(f.ctx)
	require.NoError(t, err)
	return n
}

// TestCloseTicketTerminatesBPMNInstanceAndCancelsPendingTasks：
// 断言必须同时命中「工单状态 / 实例状态 / 活跃待办数」，只查一半旧代码都能通过。
func TestCloseTicketTerminatesBPMNInstanceAndCancelsPendingTasks(t *testing.T) {
	f := newTicketWorkflowAdvanceFixture(t)
	tk := f.createTicket(t, "TKT-CLOSE-TERM-1")
	instance := f.startProcess(t, "ticket_advance_e2e", tk.ID)
	require.Equal(t, 1, countActiveProcessTasks(t, f, instance.ID), "起流后必须有一个活跃作业待办")
	resolveTicketForClose(t, f, tk.ID)

	err := f.workflowSvc.CloseTicket(f.ctx, &dto.CloseTicketRequest{
		TicketID:   tk.ID,
		CloseNotes: "浏览器 QA：关闭已解决工单，验证流程实例收口",
	}, f.actorID, f.tenantID)
	require.NoError(t, err)

	closed, err := f.client.Ticket.Get(f.ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "closed", closed.Status)

	persisted, err := f.client.ProcessInstance.Get(f.ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "terminated", persisted.Status, "关闭工单必须终止关联流程实例")
	assert.Zero(t, countActiveProcessTasks(t, f, instance.ID),
		"关闭后不得残留任何活跃待办（含引擎新生成的下一节点任务）")

	tasks, err := f.client.ProcessTask.Query().
		Where(processtask.ProcessInstanceIDEQ(instance.ID)).All(f.ctx)
	require.NoError(t, err)
	for _, task := range tasks {
		assert.Contains(t, []string{"completed", "cancelled"}, task.Status,
			"遗留待办必须显式置终态，不能只靠列表查询隐藏")
	}
}

// TestCloseTicketRefusesToCancelPendingApproval：终止不得变成「绕过审批」。
// 停在审批节点时关闭必须 409，工单 / 实例 / 审批待办三者原样回滚。
func TestCloseTicketRefusesToCancelPendingApproval(t *testing.T) {
	f := newTicketWorkflowAdvanceFixture(t)
	tk := f.createTicket(t, "TKT-CLOSE-TERM-2")
	instance := f.startProcess(t, "ticket_approval_node_e2e", tk.ID)
	resolveTicketForClose(t, f, tk.ID)

	err := f.workflowSvc.CloseTicket(f.ctx, &dto.CloseTicketRequest{
		TicketID: tk.ID, CloseNotes: "尝试在审批在途时关闭",
	}, f.actorID, f.tenantID)
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr), "必须是可行动的 409，不能被兜底成 500")
	assert.Equal(t, common.ConflictCode, bizErr.Code)

	unchangedTicket, err := f.client.Ticket.Get(f.ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "resolved", unchangedTicket.Status, "业务写入必须随事务回滚")
	persisted, err := f.client.ProcessInstance.Get(f.ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", persisted.Status, "实例不得被关闭动作终止")
	assert.Equal(t, 1, countActiveProcessTasks(t, f, instance.ID), "审批待办必须保留")
}

// TestCloseTicketWithoutBoundWorkflowKeepsLegacyBehaviour：
// 未绑定流程的存量工单仍能照常关闭，终止逻辑不得制造 4xx。
func TestCloseTicketWithoutBoundWorkflowKeepsLegacyBehaviour(t *testing.T) {
	f := newTicketWorkflowAdvanceFixture(t)
	tk := f.createTicket(t, "TKT-CLOSE-TERM-3")
	resolveTicketForClose(t, f, tk.ID)

	require.NoError(t, f.workflowSvc.CloseTicket(f.ctx, &dto.CloseTicketRequest{
		TicketID: tk.ID,
	}, f.actorID, f.tenantID))
	closed, err := f.client.Ticket.Get(f.ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "closed", closed.Status)
}

// TestTerminatedInstanceRefusesLateTaskCompletion：
// 模拟「关闭后残留的 decisions 请求」——即便有竞态把已取消任务改回活跃，
// 引擎也必须在实例终态处 fail closed，不能沿网关新建下一节点任务。
func TestTerminatedInstanceRefusesLateTaskCompletion(t *testing.T) {
	f := newTicketWorkflowAdvanceFixture(t)
	tk := f.createTicket(t, "TKT-CLOSE-TERM-4")
	instance := f.startProcess(t, "ticket_advance_e2e", tk.ID)
	task, err := f.client.ProcessTask.Query().
		Where(processtask.ProcessInstanceIDEQ(instance.ID)).Only(f.ctx)
	require.NoError(t, err)
	resolveTicketForClose(t, f, tk.ID)
	require.NoError(t, f.workflowSvc.CloseTicket(f.ctx, &dto.CloseTicketRequest{
		TicketID: tk.ID,
	}, f.actorID, f.tenantID))

	_, err = f.client.ProcessTask.UpdateOneID(task.ID).SetStatus("created").Save(f.ctx)
	require.NoError(t, err)

	// 系统推进旁路：跳过个人归属校验，确保断言命中的是实例终态守卫本身。
	lateCtx := bpmn.WithBPMNSystemAdvance(f.actorCtxValue())
	err = f.engine.CompleteTask(lateCtx, task.TaskID, map[string]interface{}{"ticketStatus": "closed"})
	require.Error(t, err, "终态实例的迟到任务不得被完成")
	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr), "必须是可行动的 409，不能被兜底成 500")
	assert.Equal(t, common.ConflictCode, bizErr.Code)

	nextNodeTasks, err := f.client.ProcessTask.Query().Where(
		processtask.ProcessInstanceIDEQ(instance.ID),
		processtask.TaskDefinitionKeyEQ("handle"),
	).Count(f.ctx)
	require.NoError(t, err)
	assert.Zero(t, nextNodeTasks, "关闭后不得再生成下一节点待办")
	persisted, err := f.client.ProcessInstance.Get(f.ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "terminated", persisted.Status)
}
