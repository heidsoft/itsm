package service

import (
	"strconv"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/processtask"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// P0-2 回归锁（2026-11-21 第三轮）：
// 接受后新生成的活跃待办必须归属接受人（服务台），而不是发起人。
// 旧行为：resolveTaskAssignment 回退顺序 requester_id/triggered_by/assignee_id
// 不含流程变量 assignee_id，活跃待办挂到员工名下——与 ticket.assignee_id 不一致。
func TestAcceptTicketOwnsNextTodoByAcceptee(t *testing.T) {
	f := newTicketWorkflowAdvanceFixture(t)

	// 请求人（员工）与接受人（服务台）刻意不同，才能区分待办归属。
	requester, err := f.client.User.Create().
		SetUsername("ticket_accept_owner_requester").
		SetEmail("accept_owner_req@example.com").SetName("发起人").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).
		SetTenantID(f.tenantID).Save(f.ctx)
	require.NoError(t, err)

	tk, err := f.client.Ticket.Create().
		SetTitle("接受人归属回归").SetDescription("接受后活跃待办必须归属服务台").
		SetStatus("new").SetType("incident").SetPriority("medium").
		SetTicketNumber("TKT-OWNER-ACCEPT-1").
		SetRequesterID(requester.ID).SetTenantID(f.tenantID).SetVersion(1).
		Save(f.ctx)
	require.NoError(t, err)

	instance := f.startProcess(t, "ticket_advance_e2e", tk.ID)
	require.NoError(t, f.workflowSvc.AcceptTicket(f.ctx, &dto.AcceptTicketRequest{TicketID: tk.ID}, f.actorID, f.tenantID))

	tasks, err := f.client.ProcessTask.Query().
		Where(
			processtask.ProcessInstanceIDEQ(instance.ID),
			processtask.StatusNotIn("completed", "cancelled"),
		).
		All(f.ctx)
	require.NoError(t, err)
	require.Len(t, tasks, 1, "接受后必须恰好有一个未完成的活跃待办")
	assert.Equal(t, strconv.Itoa(f.actorID), tasks[0].Assignee,
		"活跃待办必须归属接受人（服务台），而不是发起人")
}
