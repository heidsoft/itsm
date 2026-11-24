package bpmn

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ============================================================================
// ticket_task 通知（codex 审计 P2-08）：
// 旧实现调用的是恒返回 nil 的空通知服务，通知没发生也返回 ServiceTask 成功，
// worker 以成功写完成凭据、永不重试。本组用例守住三条：
//   1. 通知必须真的进入可靠投递队列（operational_commands）；
//   2. 通知发不出去必须显式失败，不得返回成功；
//   3. 分配动作不得复活终态工单。
// ============================================================================

type ticketNotifyEnv struct {
	client   *ent.Client
	handler  *TicketServiceTaskHandler
	ctx      context.Context
	tenantID int
	requester *ent.User
	assignee  *ent.User
}

func setupTicketNotifyTest(t *testing.T, name string) *ticketNotifyEnv {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:ticket_notify_%s_%d?mode=memory&cache=shared&_fk=1", name, time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	handler := NewTicketServiceTaskHandler(client, logger)

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Notify Tenant " + name).SetCode("notify" + name).
		SetDomain("notify" + name + ".test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	requester, err := client.User.Create().
		SetUsername("requester_" + name).SetEmail("requester_" + name + "@test.com").
		SetName("请求人").SetPasswordHash("hash").SetRole("end_user").
		SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	assignee, err := client.User.Create().
		SetUsername("assignee_" + name).SetEmail("assignee_" + name + "@test.com").
		SetName("处理人").SetPasswordHash("hash").SetRole("agent").
		SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	workflowCtx := context.WithValue(ctx, BPMNTenantIDContextKey, tenant.ID)
	return &ticketNotifyEnv{
		client: client, handler: handler, ctx: workflowCtx,
		tenantID: tenant.ID, requester: requester, assignee: assignee,
	}
}

func (env *ticketNotifyEnv) createTicket(t *testing.T, suffix string, requesterID, assigneeID int) *ent.Ticket {
	t.Helper()
	if requesterID <= 0 {
		// requester_id 是 Positive + Required 边，0 连建都建不出来——
		// 「无收件人」这条失败路径在生产只能由未分配的 assignee 触发。
		t.Fatalf("夹具不合法：requester_id 必须为正，实际 %d", requesterID)
	}
	create := env.client.Ticket.Create().
		SetTitle("通知用例工单 " + suffix).
		SetTicketNumber(fmt.Sprintf("TKT-NOTIFY-%s-%d", suffix, time.Now().UnixNano())).
		SetStatus("open").
		SetPriority("medium").
		SetType("incident").
		SetRequesterID(requesterID).
		SetTenantID(env.tenantID)
	if assigneeID > 0 {
		// 显式传 0 会被外键拒绝（不是 NULL），未分配必须不写这个字段。
		create = create.SetAssigneeID(assigneeID)
	}
	tk, err := create.Save(env.ctx)
	require.NoError(t, err)
	return tk
}

// deliverCommands 查询已入箱的通知投递命令。
func (env *ticketNotifyEnv) deliverCommands(t *testing.T) []*ent.OperationalCommand {
	t.Helper()
	cmds, err := env.client.OperationalCommand.Query().
		Where(operationalcommand.CommandTypeEQ(commandbus.CommandDeliverNotification)).
		All(env.ctx)
	require.NoError(t, err)
	return cmds
}

// TestTicketNotifyRequester_EnqueuesDelivery 通知请求人必须进入可靠投递队列，
// 而不是打一行「已通知」的成功日志。
func TestTicketNotifyRequester_EnqueuesDelivery(t *testing.T) {
	env := setupTicketNotifyTest(t, "requester")
	tk := env.createTicket(t, "req", env.requester.ID, 0)

	result, err := env.handler.Execute(env.ctx, nil, map[string]interface{}{
		"business_id": tk.ID,
		"action":      "notify_requester",
		"content":     "您的工单已受理",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	cmds := env.deliverCommands(t)
	require.Len(t, cmds, 1, "通知必须入箱，否则 worker 无从投递")
	assert.Equal(t, commandbus.CommandDeliverNotification, cmds[0].CommandType)
	assert.Equal(t, "ticket", cmds[0].AggregateType)
	assert.Equal(t, tk.ID, cmds[0].AggregateID)
}

// TestTicketNotifyHandler_FailsWhenUnassigned 未分配处理人时必须显式失败：
// 返回成功等于让完成凭据为一件没发生的事背书。
// （requester_id 是 Positive+Required，生产建不出无请求人的工单；
// assignee_id 才是可空的那条，「无收件人」在真实数据里只能长这样。）
func TestTicketNotifyHandler_FailsWhenUnassigned(t *testing.T) {
	env := setupTicketNotifyTest(t, "no_assignee")
	tk := env.createTicket(t, "noasg", env.requester.ID, 0)

	result, err := env.handler.Execute(env.ctx, nil, map[string]interface{}{
		"business_id": tk.ID,
		"action":      "notify_handler",
		"content":     "无处理人的通知",
	})
	require.Error(t, err, "没有收件人时必须失败，不得返回成功")
	if result != nil {
		assert.False(t, result.Success, "失败回执不得标记为成功")
	}
	assert.Empty(t, env.deliverCommands(t))
}

// TestTicketNotifyRequester_FailsWhenRecipientInactive 收件人被停用：
// 入队校验必须失败并上抛，不能 Warn 之后照常返回成功。
func TestTicketNotifyRequester_FailsWhenRecipientInactive(t *testing.T) {
	env := setupTicketNotifyTest(t, "inactive_requester")
	tk := env.createTicket(t, "inactive", env.requester.ID, 0)

	_, err := env.client.User.UpdateOneID(env.requester.ID).SetActive(false).Save(env.ctx)
	require.NoError(t, err)

	result, err := env.handler.Execute(env.ctx, nil, map[string]interface{}{
		"business_id": tk.ID,
		"action":      "notify_requester",
		"content":     "停用收件人的通知",
	})
	require.Error(t, err, "收件人不可投递必须失败，不得假成功")
	if result != nil {
		assert.False(t, result.Success)
	}
	assert.Empty(t, env.deliverCommands(t))
}

// TestTicketNotifyHandler_EnqueuesDelivery 通知处理人同样入箱。
func TestTicketNotifyHandler_EnqueuesDelivery(t *testing.T) {
	env := setupTicketNotifyTest(t, "handler")
	tk := env.createTicket(t, "hnd", env.requester.ID, env.assignee.ID)

	result, err := env.handler.Execute(env.ctx, nil, map[string]interface{}{
		"business_id": tk.ID,
		"action":      "notify_handler",
		"content":     "有新工单指派给你",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	require.Len(t, env.deliverCommands(t), 1)
}

// TestTicketAssign_EnqueuesAssignmentNotification 分配动作必须同时入箱指派提醒。
// 用 new 状态覆盖「合法迁移」分支：new→assigned 在权威状态机里允许。
func TestTicketAssign_EnqueuesAssignmentNotification(t *testing.T) {
	env := setupTicketNotifyTest(t, "assign")
	tk := env.createTicket(t, "asg", env.requester.ID, 0)
	_, err := env.client.Ticket.UpdateOneID(tk.ID).SetStatus(common.TicketStatusNew).Save(env.ctx)
	require.NoError(t, err)

	result, err := env.handler.Execute(env.ctx, nil, map[string]interface{}{
		"business_id": tk.ID,
		"action":      "assign",
		"assignee_id": float64(env.assignee.ID),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	after, err := env.client.Ticket.Get(env.ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, env.assignee.ID, after.AssigneeID)
	assert.Equal(t, common.TicketStatusAssigned, after.Status, "new→assigned 是合法迁移")

	require.Len(t, env.deliverCommands(t), 1, "分配必须入箱指派提醒")
}

// TestTicketAssign_KeepsStatusWhenTransitionIllegal 分配不改状态机不允许的
// 状态：open→assigned 非法，分配只换处理人，不得强行写 assigned。
func TestTicketAssign_KeepsStatusWhenTransitionIllegal(t *testing.T) {
	env := setupTicketNotifyTest(t, "assign_illegal")
	tk := env.createTicket(t, "asgill", env.requester.ID, 0) // 初始 open

	result, err := env.handler.Execute(env.ctx, nil, map[string]interface{}{
		"business_id": tk.ID,
		"action":      "assign",
		"assignee_id": float64(env.assignee.ID),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success, "分配本身仍然成功")

	after, err := env.client.Ticket.Get(env.ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, env.assignee.ID, after.AssigneeID, "处理人必须写入")
	assert.Equal(t, common.TicketStatusOpen, after.Status, "非法迁移不得强写 assigned")
}

// TestTicketAssign_DoesNotResurrectClosedTicket 分配动作不得把已关闭工单改回
// assigned——状态迁移必须过权威状态机。
func TestTicketAssign_DoesNotResurrectClosedTicket(t *testing.T) {
	env := setupTicketNotifyTest(t, "closed_assign")
	tk := env.createTicket(t, "closed", env.requester.ID, 0)
	_, err := env.client.Ticket.UpdateOneID(tk.ID).SetStatus("closed").Save(env.ctx)
	require.NoError(t, err)

	result, err := env.handler.Execute(env.ctx, nil, map[string]interface{}{
		"business_id": tk.ID,
		"action":      "assign",
		"assignee_id": float64(env.assignee.ID),
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	after, err := env.client.Ticket.Get(env.ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "closed", after.Status, "终态工单不得被分配动作复活")
	assert.Equal(t, env.assignee.ID, after.AssigneeID, "分配本身仍然生效")
}

// TestTicketEscalate_KeepsLegalTicketStatus 升级只提优先级：
// tickets.status 词表没有 escalated（那是 incident 的状态），
// 写它会被数据库 CHECK 约束拒绝，导致 escalate 动作在生产必然失败。
func TestTicketEscalate_KeepsLegalTicketStatus(t *testing.T) {
	env := setupTicketNotifyTest(t, "escalate")
	tk := env.createTicket(t, "esc", env.requester.ID, env.assignee.ID)

	result, err := env.handler.Execute(env.ctx, nil, map[string]interface{}{
		"business_id":       tk.ID,
		"action":            "escalate",
		"escalate_to":       "critical",
		"escalation_reason": "影响面扩大",
		"notify_admin_ids":  []int{env.assignee.ID},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	after, err := env.client.Ticket.Get(env.ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "critical", after.Priority, "升级必须提高优先级")
	assert.True(t, common.StatusValueAllowed("tickets", "status", after.Status),
		"升级后的状态必须在 tickets.status 词表内，实际 %q", after.Status)

	require.Len(t, env.deliverCommands(t), 1, "升级必须入箱通知管理员")
}

// TestTicketNotify_SameIdempotencyKeyEnqueuesOnce 生产链路里 ServiceTask 恒以
// task=nil 调用，幂等只能来自命令级 _idempotencyKey。旧 occurrenceKey 在这种
// 输入下落到 unixNano，同一命令重试会重复入箱（codex 审计 P2-08 幂等缺口）。
func TestTicketNotify_SameIdempotencyKeyEnqueuesOnce(t *testing.T) {
	env := setupTicketNotifyTest(t, "idem")
	tk := env.createTicket(t, "idem", env.requester.ID, 0)

	vars := map[string]interface{}{
		"business_id":       tk.ID,
		"action":            "notify_requester",
		"content":           "重试也不该重复入箱",
		"_idempotencyKey":   "cmd-abc-123",
		"_commandId":        float64(4242),
	}

	for i := 0; i < 3; i++ {
		result, err := env.handler.Execute(env.ctx, nil, vars)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "第 %d 次执行", i+1)
	}

	cmds := env.deliverCommands(t)
	require.Len(t, cmds, 1, "同一命令幂等键重复触发只能入箱一次，实际 %d", len(cmds))
}
