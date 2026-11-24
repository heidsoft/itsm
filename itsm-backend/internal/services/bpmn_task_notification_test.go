package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	_ "github.com/mattn/go-sqlite3"
)

// ============================================================================
// 超时通知（codex 审计 P2-03）：
//   1. 状态翻转与提醒入箱必须同事务——入箱失败不得留下「已超时但提醒永久丢失」；
//   2. claim-once 不变：同一任务双路径只生效一次；
//   3. 收件人解析必须同时支持数字ID与用户名（认领路径写的是 username）；
//   4. 收件人不可投递是确定性状态：跳过通知、保留状态翻转，不无限重试。
// ============================================================================

// testBPMNTimeoutNotifyPast 截止时间在过去，供恢复扫描路径选中。
const testBPMNTimeoutNotifyPast = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="timeout_notify_proc" name="超时提醒" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="deadline_task" name="有截止时间的任务" dueDate="2000-01-01" timeoutAction="notify"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="deadline_task"/>
    <bpmn:sequenceFlow id="f2" sourceRef="deadline_task" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

// timeoutNotifyEnv 一次超时通知用例的夹具。
type timeoutNotifyEnv struct {
	client   *ent.Client
	scanner  *TimeoutScanner
	ctx      context.Context
	tenantID int
	instance *ent.ProcessInstance
	task     *ent.ProcessTask
	assignee *ent.User
}

// setupTimeoutNotifyEnv 部署超时提醒流程并把任务指派给 assigneeUsername。
func setupTimeoutNotifyEnv(t *testing.T, name, assigneeUsername string, assigneeActive bool) *timeoutNotifyEnv {
	t.Helper()
	client := newE2ETestClient(t, name)
	logger := zaptest.NewLogger(t).Sugar()

	instance, task, _, ctx, _ := startTaskDueProcess(t, client, testBPMNTimeoutNotifyPast,
		"timeout_notify_proc", "BK-TMO-NOTIFY-"+name)

	tenantID := task.TenantID
	assignee, err := client.User.Create().
		SetUsername(assigneeUsername).
		SetEmail(assigneeUsername + "@test.com").
		SetName("超时提醒接收人").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(assigneeActive).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ProcessTask.UpdateOneID(task.ID).
		SetAssignee(assigneeUsername).
		Save(ctx)
	require.NoError(t, err)

	return &timeoutNotifyEnv{
		client:   client,
		scanner:  NewTimeoutScanner(client, logger),
		ctx:      ctx,
		tenantID: tenantID,
		instance: instance,
		task:     task,
		assignee: assignee,
	}
}

// deliverCommands 查询某任务已入箱的投递命令。
func (env *timeoutNotifyEnv) notifyCommands(t *testing.T) []*ent.OperationalCommand {
	t.Helper()
	cmds, err := env.client.OperationalCommand.Query().
		Where(operationalcommand.CommandTypeEQ(commandbus.CommandDeliverNotification)).
		All(env.ctx)
	require.NoError(t, err)
	return cmds
}

// TestTimeoutScanner_NotifyEnqueuesReminderWithUsernameAssignee
// 用户名形态的 assignee 也必须解析成真实收件人：认领路径写的是 username，
// 只认数字ID会让绝大多数超时提醒静默丢失。
func TestTimeoutScanner_NotifyEnqueuesReminderWithUsernameAssignee(t *testing.T) {
	env := setupTimeoutNotifyEnv(t, "timeout_notify_user", "notify_assignee", true)

	processed, err := env.scanner.ScanOverdueTasks(env.ctx, env.tenantID)
	require.NoError(t, err)
	require.Equal(t, 1, processed)

	after, err := env.client.ProcessTask.Get(env.ctx, env.task.ID)
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusTimeout, after.Status, "notify 动作必须把任务置为 timeout")

	cmds := env.notifyCommands(t)
	require.Len(t, cmds, 1, "必须入箱一条提醒投递命令")
	assert.True(t, strings.Contains(cmds[0].IdempotencyKey, "task_timeout:"),
		"幂等键必须锚定任务超时事件，实际: %s", cmds[0].IdempotencyKey)

	// claim-once：第二次扫描不得重复入箱。
	processed, err = env.scanner.ScanOverdueTasks(env.ctx, env.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 0, processed, "已超时任务不应被再次处理")
	assert.Len(t, env.notifyCommands(t), 1, "重复扫描不得产生第二条提醒命令")
}

// TestTimeoutScanner_ClaimTimeoutStatus_RollsBackWhenEnqueueFails
// 入箱失败必须把状态翻转一起回滚：否则任务已 timeout，后续扫描不再选中，
// 这条提醒永久丢失且无人知晓（旧实现正是「先提交状态、再另开事务入箱并吞错」）。
func TestTimeoutScanner_ClaimTimeoutStatus_RollsBackWhenEnqueueFails(t *testing.T) {
	env := setupTimeoutNotifyEnv(t, "timeout_notify_rollback", "rollback_assignee", true)

	injected := errors.New("injected enqueue failure")
	err := env.scanner.claimTimeoutStatus(env.ctx, env.task, env.tenantID,
		common.ProcessTaskStatusTimeout, nil,
		func(context.Context, *ent.Tx) error { return injected })
	require.ErrorIs(t, err, injected, "入箱失败必须上抛，不得被吞掉")

	after, err := env.client.ProcessTask.Get(env.ctx, env.task.ID)
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusCreated, after.Status,
		"入箱失败时状态翻转必须一起回滚，任务留待下一轮扫描重试")
	assert.Empty(t, env.notifyCommands(t), "回滚后不得留下投递命令")
}

// TestTimeoutScanner_NotifySkipsInactiveRecipient 收件人已停用是确定性数据状态：
// 跳过通知但保留状态翻转，避免该任务每轮扫描都撞同一条错误永远无法收敛。
func TestTimeoutScanner_NotifySkipsInactiveRecipient(t *testing.T) {
	env := setupTimeoutNotifyEnv(t, "timeout_notify_inactive", "inactive_assignee", false)

	processed, err := env.scanner.ScanOverdueTasks(env.ctx, env.tenantID)
	require.NoError(t, err, "收件人不可投递不得让扫描整体失败")
	require.Equal(t, 1, processed)

	after, err := env.client.ProcessTask.Get(env.ctx, env.task.ID)
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusTimeout, after.Status,
		"收件人不可投递时状态仍要翻转，超时事实不能丢")
	assert.Empty(t, env.notifyCommands(t), "不可投递的收件人不得入箱")
}

// TestTimeoutScanner_EscalateFlipsStatusAndEnqueuesReminder escalate 与 notify
// 同样是「状态翻转 + 提醒」的原子动作。
func TestTimeoutScanner_EscalateFlipsStatusAndEnqueuesReminder(t *testing.T) {
	env := setupTimeoutNotifyEnv(t, "timeout_escalate", "escalate_assignee", true)

	// 用 escalate 动作覆盖任务变量。
	_, err := env.client.ProcessTask.UpdateOneID(env.task.ID).
		SetTaskVariables(map[string]interface{}{"timeoutAction": common.TimeoutActionEscalate}).
		Save(env.ctx)
	require.NoError(t, err)

	processed, err := env.scanner.ScanOverdueTasks(env.ctx, env.tenantID)
	require.NoError(t, err)
	require.Equal(t, 1, processed)

	after, err := env.client.ProcessTask.Get(env.ctx, env.task.ID)
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusEscalated, after.Status)
	assert.Equal(t, "任务超时自动升级", after.TaskVariables["escalation_reason"])

	cmds := env.notifyCommands(t)
	require.Len(t, cmds, 1, "升级必须同时入箱提醒")
	assert.True(t, strings.Contains(cmds[0].IdempotencyKey, "task_timeout:"))
}

// TestTimeoutScanner_NotifyWithoutAssigneeNoCrash 未分配任务：只翻转状态，
// 不入箱、不报错。
func TestTimeoutScanner_NotifyWithoutAssigneeNoCrash(t *testing.T) {
	client := newE2ETestClient(t, "timeout_notify_noassignee")
	logger := zaptest.NewLogger(t).Sugar()
	_, task, _, ctx, _ := startTaskDueProcess(t, client, testBPMNTimeoutNotifyPast,
		"timeout_notify_proc", "BK-TMO-NOASSIGN")

	scanner := NewTimeoutScanner(client, logger)
	processed, err := scanner.ScanOverdueTasks(ctx, task.TenantID)
	require.NoError(t, err)
	require.Equal(t, 1, processed)

	after, err := client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusTimeout, after.Status)

	cmds, err := client.OperationalCommand.Query().Where(
		operationalcommand.CommandTypeEQ(commandbus.CommandDeliverNotification),
	).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, cmds)
}

// TestResolveTimeoutRecipient_SupportsIDAndUsername 收件人解析的两种历史写法。
func TestResolveTimeoutRecipient_SupportsIDAndUsername(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:resolve_assignee?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Resolve Tenant").SetCode("resolve").SetDomain("resolve.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	agent, err := client.User.Create().
		SetUsername("resolver").SetEmail("resolver@test.com").SetName("Resolver").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	scanner := NewTimeoutScanner(client, logger)
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	cases := []struct {
		name     string
		assignee string
		want     int
	}{
		{"空指派", "", 0},
		{"占位0", "0", 0},
		{"数字ID", strconv.Itoa(agent.ID), agent.ID},
		{"用户名", "resolver", agent.ID},
		{"未知用户", "ghost", 0},
		{"异租户用户", strconv.Itoa(agent.ID + 9999), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := scanner.resolveTimeoutRecipient(ctx, tx, tenant.ID, tc.assignee)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
