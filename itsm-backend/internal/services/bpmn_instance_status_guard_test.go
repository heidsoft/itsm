package service

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// codex 审计 P1-04/P1-05 回归（2026-11-23）：
//   - P1-05 实例状态协议：completeProcess 只允许 running→completed；
//     暂停实例拒绝普通 CompleteTask；ServiceTask consumer 在副作用执行前
//     校验实例状态（终态→命令停止、暂停→可重试）；终止同事务取消未执行命令。
//   - P1-04 的 CAS 收口（ServiceTask 完成事务 version 条件写、工单接受人
//     写入改走 updateProcessInstanceVariablesCAS）是结构修复，其判别力来自
//     version 谓词本身，单线程夹具无法复现并发交错；此处锁行为面。

// TestCompleteProcess_RejectsNonRunningInstance 陈旧快照（内存里还是 running，
// 库里已被终止）调用 completeProcess 必须失败且不改库——无条件写曾把
// terminated 翻回 completed（终态复活）。
func TestCompleteProcess_RejectsNonRunningInstance(t *testing.T) {
	engine, ctx, instanceID, _ := completionGuardFixture(t)

	_, err := engine.client.ProcessInstance.UpdateOneID(instanceID).
		SetStatus("terminated").
		Save(ctx)
	require.NoError(t, err)

	stale, err := engine.client.ProcessInstance.Get(ctx, instanceID)
	require.NoError(t, err)
	require.Equal(t, "terminated", stale.Status)

	// 构造陈旧快照：内存实体仍认为自己 running，模拟 endEvent 推进期间
	// 实例被并发终止。
	staleRunning := *stale
	staleRunning.Status = "running"

	err = engine.completeProcess(ctx, engine.client, &staleRunning)
	require.Error(t, err, "非 running 实例不得被标记完成")
	// completeProcess 用 BusinessError 表达 conflict（与 CompleteTask 守卫同族），
	// classifyError 依赖该类型走到 409 而不是 500。
	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr), "必须是结构化业务错误而非裸 error: %v", err)
	assert.Equal(t, common.ConflictCode, bizErr.Code)

	after, err := engine.client.ProcessInstance.Get(ctx, instanceID)
	require.NoError(t, err)
	assert.Equal(t, "terminated", after.Status, "终态实例不得被 completeProcess 改写")
}

// TestCompleteTask_SuspendedInstance_Rejected 暂停实例不得被普通
// CompleteTask 推进（P1-05：suspend 只挡新入口是不够的）。
func TestCompleteTask_SuspendedInstance_Rejected(t *testing.T) {
	engine, ctx, instanceID, taskID := completionGuardFixture(t)

	require.NoError(t, engine.SuspendProcess(ctx, "PI-GUARD-1", "测试暂停"))

	before := snapshotCompletionGuardState(t, engine, instanceID, taskID)
	require.Equal(t, "suspended", before.instanceStatus)

	err := engine.CompleteTask(ctx, taskID, map[string]interface{}{
		"approvalAction":  "approve",
		"approvalResult":  "approved",
		"approvalComment": "同意",
	})
	require.Error(t, err, "暂停实例必须拒绝任务完成")
	assert.Contains(t, err.Error(), "已暂停")

	after := snapshotCompletionGuardState(t, engine, instanceID, taskID)
	assert.Equal(t, before, after, "被拒绝的完成请求必须零副作用")

	// 恢复后同请求照常完成：守卫针对的是状态而不是请求本身。
	require.NoError(t, engine.ResumeProcess(ctx, "PI-GUARD-1"))
	require.NoError(t, engine.CompleteTask(ctx, taskID, map[string]interface{}{
		"approvalAction": "approve", "approvalResult": "approved",
	}))
	final := snapshotCompletionGuardState(t, engine, instanceID, taskID)
	assert.Equal(t, "completed", final.taskStatus)
}

// statusGuardServiceFixture 启动一条真实可执行的 ServiceTask 流程，
// 返回 (engine, workflowCtx, instanceID, fake handler)。StartProcess 会把
// ServiceTask 入队成 pending 命令（副作用不在启动事务内执行）。
func statusGuardServiceFixture(t *testing.T) (*CustomProcessEngine, context.Context, int, *durableTestHandler) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", testDSN())
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	tenantID := 61
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-STATUS-GUARD").SetDeploymentName("status-guard").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey("statusGuardService").SetName("status-guard").SetBpmnXML([]byte(durableServiceTaskBPMN)).
		SetDeploymentID(deployment.ID).SetTenantID(tenantID).SetIsActive(true).SetIsLatest(true).Save(ctx)
	require.NoError(t, err)

	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	fake := &durableTestHandler{}
	engine.callbackRegistry.RegisterHandler(fake)
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)

	instance, err := engine.StartProcess(workflowCtx, "statusGuardService", "status:1", map[string]interface{}{})
	require.NoError(t, err)
	require.Equal(t, 0, fake.calls)
	return engine, workflowCtx, instance.ID, fake
}

// statusGuardCommand 取出该实例入队的 ServiceTask 命令。
func statusGuardCommand(t *testing.T, engine *CustomProcessEngine, instanceID int) *ent.OperationalCommand {
	t.Helper()
	cmd, err := engine.client.OperationalCommand.Query().Where(
		operationalcommand.CommandTypeEQ(commandbus.CommandExecuteBPMNServiceTask),
		operationalcommand.AggregateIDEQ(instanceID),
	).Only(context.Background())
	require.NoError(t, err)
	return cmd
}

// TestTerminateProcess_CancelsQueuedServiceTaskCommands 终止必须在同一个
// 事务里取消该实例未执行的 ServiceTask 命令——否则已终止流程的命令仍会
// 被 worker 认领执行副作用（P1-05）。
func TestTerminateProcess_CancelsQueuedServiceTaskCommands(t *testing.T) {
	engine, ctx, instanceID, fake := statusGuardServiceFixture(t)
	cmd := statusGuardCommand(t, engine, instanceID)
	require.Equal(t, commandbus.StatusPending, cmd.Status)

	require.NoError(t, engine.TerminateProcess(ctx, strconv.Itoa(instanceID), "测试终止"))

	stored, err := engine.client.OperationalCommand.Get(context.Background(), cmd.ID)
	require.NoError(t, err)
	assert.Equal(t, commandbus.StatusCancelled, stored.Status,
		"终止必须同事务取消排队中的 ServiceTask 命令")

	inst, err := engine.client.ProcessInstance.Get(context.Background(), instanceID)
	require.NoError(t, err)
	assert.Equal(t, "terminated", inst.Status)
	assert.Equal(t, 0, fake.calls, "终止后副作用不得执行")
}

// TestHandleBPMNServiceTaskCommand_TerminalInstanceStopsSideEffect 竞态兜底：
// 命令已被 worker 认领（processing）后实例才被终止，consumer 必须在副作用
// 执行前按终态拦截——返回 NotFound 语义（worker 收敛为 cancelled），副作用
// 零次执行，实例保持 terminated。
func TestHandleBPMNServiceTaskCommand_TerminalInstanceStopsSideEffect(t *testing.T) {
	engine, ctx, instanceID, fake := statusGuardServiceFixture(t)
	cmd := statusGuardCommand(t, engine, instanceID)

	// 模拟竞态：worker 已认领命令，随后实例被终止。
	_, err := engine.client.OperationalCommand.UpdateOneID(cmd.ID).
		SetStatus(commandbus.StatusProcessing).
		SetLeaseOwner("racing-worker").
		SetFencingToken(1).
		SetLeaseExpiresAt(time.Now().Add(time.Minute)).
		Save(ctx)
	require.NoError(t, err)
	_, err = engine.client.ProcessInstance.UpdateOneID(instanceID).
		SetStatus("terminated").
		Save(ctx)
	require.NoError(t, err)

	claimed, err := engine.client.OperationalCommand.Get(ctx, cmd.ID)
	require.NoError(t, err)

	err = engine.HandleBPMNServiceTaskCommand(ctx, claimed)
	require.Error(t, err, "终态实例的命令必须被拦截")
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr), "必须是结构化错误（worker 据此收敛 cancelled）: %v", err)
	assert.Equal(t, common.ErrCodeNotFound, appErr.Code, "终态停止必须走 NotFound 语义 → worker cancelled")

	assert.Equal(t, 0, fake.calls, "终态实例不得执行副作用")

	inst, err := engine.client.ProcessInstance.Get(ctx, instanceID)
	require.NoError(t, err)
	assert.Equal(t, "terminated", inst.Status, "拦截不得改写实例状态")
	_, hasCredential := inst.Variables["_serviceTaskExecutedOccurrences"]
	assert.False(t, hasCredential, "拦截不得写入完成凭据")
}

// TestHandleBPMNServiceTaskCommand_SuspendedInstanceRetriesWithoutSideEffect
// 暂停实例的命令不得执行副作用，但也不得被判终态：返回可重试错误，
// worker 会退避重试，恢复 running 后照常执行（不因暂停丢命令）。
func TestHandleBPMNServiceTaskCommand_SuspendedInstanceRetriesWithoutSideEffect(t *testing.T) {
	engine, ctx, instanceID, fake := statusGuardServiceFixture(t)
	cmd := statusGuardCommand(t, engine, instanceID)

	_, err := engine.client.OperationalCommand.UpdateOneID(cmd.ID).
		SetStatus(commandbus.StatusProcessing).
		SetLeaseOwner("suspended-worker").
		SetFencingToken(1).
		SetLeaseExpiresAt(time.Now().Add(time.Minute)).
		Save(ctx)
	require.NoError(t, err)
	_, err = engine.client.ProcessInstance.UpdateOneID(instanceID).
		SetStatus("suspended").
		Save(ctx)
	require.NoError(t, err)

	claimed, err := engine.client.OperationalCommand.Get(ctx, cmd.ID)
	require.NoError(t, err)

	err = engine.HandleBPMNServiceTaskCommand(ctx, claimed)
	require.Error(t, err, "暂停实例的命令必须暂缓")
	var appErr *common.AppError
	require.False(t, errors.As(err, &appErr) && appErr.Code == common.ErrCodeNotFound,
		"暂停是可重试状态，不得走终态 NotFound 语义（否则命令被取消，恢复后流程卡死）")
	assert.Contains(t, err.Error(), "suspended")
	assert.Equal(t, 0, fake.calls, "暂停实例不得执行副作用")

	// 恢复后同一命令照常执行并推进流程。
	_, err = engine.client.ProcessInstance.UpdateOneID(instanceID).
		SetStatus("running").
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, engine.HandleBPMNServiceTaskCommand(ctx, claimed))
	assert.Equal(t, 1, fake.calls, "恢复 running 后命令必须能继续执行")
}

// TestSuspendTerminate_TransitionsRejectStaleWrites 生命周期条件更新：
// load 校验通过后状态被并发改掉的场景用直接调用覆盖不到（load 在函数内），
// 这里锁定反向语义——非法迁移仍被 loadLifecycleInstance 拒绝，且拒绝后
// 状态与任务零变化（防止无条件写复活）。
func TestLifecycleTransitions_RejectInvalidSourceState(t *testing.T) {
	engine, ctx, instanceID, taskID := completionGuardFixture(t)

	// completed 实例：suspend/resume/terminate 全部 conflict。
	_, err := engine.client.ProcessInstance.UpdateOneID(instanceID).
		SetStatus("completed").
		Save(ctx)
	require.NoError(t, err)
	before := snapshotCompletionGuardState(t, engine, instanceID, taskID)

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"suspend", func() error { return engine.SuspendProcess(ctx, "PI-GUARD-1", "x") }},
		{"resume", func() error { return engine.ResumeProcess(ctx, "PI-GUARD-1") }},
		{"terminate", func() error { return engine.TerminateProcess(ctx, "PI-GUARD-1", "x") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			require.Error(t, err, "%s 已完成实例必须拒绝", tc.name)
			var appErr *common.AppError
			require.True(t, errors.As(err, &appErr), "必须是 conflict 语义: %v", err)
			assert.Equal(t, common.ErrCodeConflict, appErr.Code)
		})
	}

	after := snapshotCompletionGuardState(t, engine, instanceID, taskID)
	assert.Equal(t, before, after, "非法迁移必须零副作用（状态/任务/命令/审计不变）")
}
