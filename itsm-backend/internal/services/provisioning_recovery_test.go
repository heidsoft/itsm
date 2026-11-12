package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"itsm-backend/domain/provisioning"
	domainSR "itsm-backend/domain/servicerequest"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	entsr "itsm-backend/ent/servicerequest"
	"itsm-backend/internal/commandbus"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 交付失败恢复（2026-10-21）回归锁。
//
// 这一组测试锁的是「交付是副作用动作」这件事的边界：
//   - 已完成的任务不能再执行（重复交付 = 多开一台机器）；
//   - 执行权必须独占（并发执行同样会重复交付）；
//   - worker 崩溃留下的 running 必须能自愈（否则交付永久卡死）；
//   - 重试必须换幂等键（否则命令被入箱层判重跳过，接口 200 但什么都没发生）；
//   - 自动重试走不通时，人工判定必须留痕。

func newRecoveryFixture(t *testing.T) (*ent.Client, context.Context, int, int) {
	t.Helper()
	dsn := "file:provisioning_recovery_" + t.Name() + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, dialect.SQLite, dsn)
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	tenantID, requesterID := seedProvisioningPrerequisites(t, ctx, client)
	return client, ctx, tenantID, requesterID
}

// seedProvisioningTask 直接落一条指定状态的交付任务（绕过创建流程，聚焦执行层语义）。
func seedProvisioningTask(t *testing.T, client *ent.Client, ctx context.Context, tenantID, requesterID int, status provisioning.TaskStatus, attemptCount int, lastAttemptAt *time.Time, payload map[string]any) (*ent.ServiceRequest, *ent.ProvisioningTask) {
	t.Helper()
	catalog, err := client.ServiceCatalog.Create().
		SetTenantID(tenantID).
		SetName("provisioning-" + string(status)).
		SetStatus("enabled").
		Save(ctx)
	require.NoError(t, err)

	srStatus := domainSR.StatusProvisioning
	if status == provisioning.TaskFailed {
		srStatus = domainSR.StatusFailed
	}
	req, err := client.ServiceRequest.Create().
		SetTenantID(tenantID).
		SetCatalogID(catalog.ID).
		SetRequesterID(requesterID).
		SetTitle("Need ECS").
		SetStatus(srStatus).
		Save(ctx)
	require.NoError(t, err)

	create := client.ProvisioningTask.Create().
		SetTenantID(tenantID).
		SetServiceRequestID(req.ID).
		SetProvider(string(provisioning.ProviderAlicloud)).
		SetResourceType(string(provisioning.ResourceECS)).
		SetStatus(string(status)).
		SetAttemptCount(attemptCount)
	if lastAttemptAt != nil {
		create = create.SetLastAttemptAt(*lastAttemptAt)
	}
	if payload != nil {
		create = create.SetPayload(payload)
	}
	task, err := create.Save(ctx)
	require.NoError(t, err)
	return req, task
}

// 已交付的任务再执行必须是幂等空操作：返回 ErrProvisioningTaskSucceeded，
// 且不产生第二次 provider 调用（attempt_count 不增加）。
func TestExecuteTaskRefusesReExecutionAfterSuccess(t *testing.T) {
	client, ctx, tenantID, requesterID := newRecoveryFixture(t)
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())

	_, task := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskPending, 0, nil, nil)

	first, err := svc.ExecuteTask(ctx, task.ID, tenantID, requesterID)
	require.NoError(t, err)
	require.Equal(t, string(provisioning.TaskSucceeded), first.Status)
	require.Equal(t, 1, first.AttemptCount)

	second, err := svc.ExecuteTask(ctx, task.ID, tenantID, requesterID)
	require.ErrorIs(t, err, ErrProvisioningTaskSucceeded,
		"已完成任务重复执行必须被拒绝，否则会重复创建云资源")
	require.Equal(t, string(provisioning.TaskSucceeded), second.Status)
	require.Equal(t, 1, second.AttemptCount, "被拒绝的重复执行不得消耗尝试次数")
}

// 执行中的任务不能被第二个调用方抢走。
func TestExecuteTaskRefusesConcurrentExecution(t *testing.T) {
	client, ctx, tenantID, requesterID := newRecoveryFixture(t)
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())

	fresh := time.Now().Add(-time.Minute)
	_, task := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskRunning, 1, &fresh, nil)

	_, err := svc.ExecuteTask(ctx, task.ID, tenantID, requesterID)
	require.ErrorIs(t, err, ErrProvisioningTaskInProgress)

	reloaded, err := client.ProvisioningTask.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, string(provisioning.TaskRunning), reloaded.Status)
	require.Equal(t, 1, reloaded.AttemptCount)
}

// worker 崩溃留下的 running（租约过期）必须能被重新领取，否则交付永久卡死。
func TestExecuteTaskReclaimsStaleRunningTask(t *testing.T) {
	client, ctx, tenantID, requesterID := newRecoveryFixture(t)
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())

	stale := time.Now().Add(-2 * staleTaskLease)
	_, task := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskRunning, 1, &stale, nil)

	executed, err := svc.ExecuteTask(ctx, task.ID, tenantID, requesterID)
	require.NoError(t, err)
	require.Equal(t, string(provisioning.TaskSucceeded), executed.Status)
	require.Equal(t, 2, executed.AttemptCount)
}

// 重试必须换幂等键：复用首次的键会被入箱层判重跳过，
// 结果是接口成功、任务永远停在 failed——最难排查的一类「假成功」。
func TestRetryTaskRequeuesWithFreshIdempotencyKey(t *testing.T) {
	client, ctx, tenantID, requesterID := newRecoveryFixture(t)
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())

	// 用 compliance_ack=false 让 stub provider 真实失败一次，得到真实的 failed 状态。
	_, task := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskPending, 0, nil,
		map[string]any{"compliance_ack": false})
	_, execErr := svc.ExecuteTask(ctx, task.ID, tenantID, requesterID)
	require.Error(t, execErr)

	failed, err := client.ProvisioningTask.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, string(provisioning.TaskFailed), failed.Status)
	require.Equal(t, 1, failed.AttemptCount)
	require.NotEmpty(t, failed.ErrorMessage, "失败原因必须落库，供人工判定时判断是否可重试")

	requeued, err := svc.RetryTask(ctx, task.ID, tenantID, requesterID, "已补合规确认")
	require.NoError(t, err)
	require.Equal(t, string(provisioning.TaskPending), requeued.Status)
	require.Equal(t, 1, requeued.AttemptCount, "重试本身不执行，尝试次数在领取时才 +1")
	require.NotEmpty(t, requeued.ErrorMessage, "重新排队期间必须保留上一次失败原因作为证据")

	// 服务请求必须同步回到 provisioning，否则详情页显示「已失败」而任务正在跑。
	sr, err := client.ServiceRequest.Query().Where(entsr.IDEQ(failed.ServiceRequestID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, domainSR.StatusProvisioning, sr.Status)

	cmds, err := client.OperationalCommand.Query().
		Where(operationalcommand.AggregateTypeEQ("provisioning_task"), operationalcommand.AggregateIDEQ(task.ID)).
		Order(ent.Asc(operationalcommand.FieldID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, cmds, 1, "直接落库的任务没有首次入箱命令，重试只应产生一条")
	require.Equal(t, "provisioning_task:"+itoa(task.ID)+":execute:2", cmds[0].IdempotencyKey,
		"重试必须使用第 N 次尝试的幂等键，不能复用首次的键")

	// 重试后可以真正执行成功（运维补上了合规确认，payload 已修正）。
	_, err = client.ProvisioningTask.UpdateOneID(task.ID).
		SetPayload(map[string]any{"compliance_ack": true}).
		Save(ctx)
	require.NoError(t, err)
	done, err := svc.ExecuteTask(ctx, task.ID, tenantID, requesterID)
	require.NoError(t, err)
	require.Equal(t, string(provisioning.TaskSucceeded), done.Status)
	require.Equal(t, 2, done.AttemptCount)
}

func TestRetryTaskRejectsNonFailedStates(t *testing.T) {
	client, ctx, tenantID, requesterID := newRecoveryFixture(t)
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())

	_, pending := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskPending, 0, nil, nil)
	_, err := svc.RetryTask(ctx, pending.ID, tenantID, requesterID, "")
	require.ErrorIs(t, err, ErrProvisioningTaskNotRetryable, "pending 已在队列里，重试没有语义")

	_, succeeded := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskSucceeded, 1, nil, nil)
	_, err = svc.RetryTask(ctx, succeeded.ID, tenantID, requesterID, "")
	require.ErrorIs(t, err, ErrProvisioningTaskSucceeded, "已交付的任务不能重试")
}

func TestRetryTaskStopsAtMaxAttempts(t *testing.T) {
	client, ctx, tenantID, requesterID := newRecoveryFixture(t)
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())

	_, task := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskFailed, provisioning.MaxTaskAttempts, nil, nil)
	_, err := svc.RetryTask(ctx, task.ID, tenantID, requesterID, "")
	require.ErrorIs(t, err, ErrProvisioningTaskAttemptsExhausted)

	cmdCount, err := client.OperationalCommand.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cmdCount, "被拒绝的重试不得留下命令")
}

// 人工接管：自动重试走不通时的唯一出口，必须留痕（谁判定、依据是什么）。
func TestResolveTaskManuallyMarksDeliveredAndSyncsRequest(t *testing.T) {
	client, ctx, tenantID, requesterID := newRecoveryFixture(t)
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())

	_, task := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskFailed, 2, nil, nil)

	resolved, err := svc.ResolveTaskManually(ctx, task.ID, tenantID, requesterID, "delivered",
		"资源已在阿里云控制台手工创建 i-abc123，凭据吊销无法走自动交付")
	require.NoError(t, err)
	require.Equal(t, string(provisioning.TaskSucceeded), resolved.Status)
	require.Equal(t, requesterID, resolved.ResolvedByUserID)
	require.Contains(t, resolved.ResolutionNote, "i-abc123")

	sr, err := client.ServiceRequest.Query().Where(entsr.IDEQ(task.ServiceRequestID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, domainSR.StatusDelivered, sr.Status, "人工判定为已交付时服务请求必须同步")
}

func TestResolveTaskManuallyRequiresNote(t *testing.T) {
	client, ctx, tenantID, requesterID := newRecoveryFixture(t)
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())

	_, task := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskFailed, 1, nil, nil)

	_, err := svc.ResolveTaskManually(ctx, task.ID, tenantID, requesterID, "delivered", "   ")
	require.Error(t, err, "没有依据说明的人工判定无法审计，必须拒绝")

	reloaded, err := client.ProvisioningTask.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, string(provisioning.TaskFailed), reloaded.Status, "被拒绝的判定不得改变任务状态")
}

func TestResolveTaskManuallyRefusesRunning(t *testing.T) {
	client, ctx, tenantID, requesterID := newRecoveryFixture(t)
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())

	fresh := time.Now()
	_, task := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskRunning, 1, &fresh, nil)

	_, err := svc.ResolveTaskManually(ctx, task.ID, tenantID, requesterID, "failed", "判定失败")
	require.ErrorIs(t, err, ErrProvisioningTaskNotResolvable,
		"执行中的任务即将回写自己的结果，人工判定会被覆盖")
}

// 跨租户不可见：所有入口都必须按 tenant_id 收敛，且错误里不得暴露任务是否存在。
func TestProvisioningRecoveryIsTenantScoped(t *testing.T) {
	client, ctx, tenantID, requesterID := newRecoveryFixture(t)
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())

	_, task := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskFailed, 1, nil, nil)

	otherTenant, err := client.Tenant.Create().SetName("other").SetCode("other").Save(ctx)
	require.NoError(t, err)

	_, err = svc.RetryTask(ctx, task.ID, otherTenant.ID, requesterID, "")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrProvisioningTaskNotRetryable, "跨租户不应命中状态判定分支")

	_, err = svc.ResolveTaskManually(ctx, task.ID, otherTenant.ID, requesterID, "delivered", "跨租户判定")
	require.Error(t, err)

	reloaded, err := client.ProvisioningTask.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, string(provisioning.TaskFailed), reloaded.Status, "跨租户操作不得改变任务状态")
	require.Zero(t, reloaded.ResolvedByUserID)
}

func TestProvisioningTaskStatusPredicates(t *testing.T) {
	require.True(t, provisioning.CanExecute(provisioning.TaskPending))
	require.True(t, provisioning.CanExecute(provisioning.TaskFailed))
	require.False(t, provisioning.CanExecute(provisioning.TaskSucceeded), "succeeded 重入 = 重复交付")
	require.False(t, provisioning.CanExecute(provisioning.TaskRunning), "running 的执行权已被独占")

	require.True(t, provisioning.CanRetry(provisioning.TaskFailed, 0))
	require.False(t, provisioning.CanRetry(provisioning.TaskFailed, provisioning.MaxTaskAttempts))
	require.False(t, provisioning.CanRetry(provisioning.TaskPending, 0))

	require.True(t, provisioning.CanResolveManually(provisioning.TaskFailed))
	require.True(t, provisioning.CanResolveManually(provisioning.TaskPending))
	require.False(t, provisioning.CanResolveManually(provisioning.TaskRunning))
	require.False(t, provisioning.CanResolveManually(provisioning.TaskSucceeded))

	require.Equal(t, 5, provisioning.MaxTaskAttempts)
}

func TestProvisioningExecuteKeyIsAttemptScoped(t *testing.T) {
	require.Equal(t, "provisioning_task:7:execute:1", provisioningExecuteKey(7, 1))
	require.NotEqual(t, provisioningExecuteKey(7, 1), provisioningExecuteKey(7, 2),
		"不同尝试必须使用不同幂等键，否则重试命令会被判重跳过")
}

func itoa(v int) string {
	return strconv.Itoa(v)
}

// 队列侧语义：重投一条「已完成/正在执行」的命令不算失败。
//
// 若返回 error，命令总线会把它重投到 MaxAttempts 再进死信，产出一批
// 「重投 5 次仍失败」的假告警——而系统状态其实是对的。
func TestProvisioningCommandHandlerTreatsTerminalStatesAsSuccess(t *testing.T) {
	client, ctx, tenantID, requesterID := newRecoveryFixture(t)
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())
	handler := NewProvisioningTaskCommandHandler(svc)
	require.NotNil(t, handler)

	_, succeeded := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskSucceeded, 1, nil, nil)
	err := handler.Handle(ctx, &ent.OperationalCommand{
		CommandType:   commandbus.CommandExecuteProvisioningTask,
		AggregateType: "provisioning_task",
		AggregateID:   succeeded.ID,
		TenantID:      tenantID,
		Payload:       map[string]interface{}{"actorUserId": float64(requesterID)},
	})
	require.NoError(t, err, "已完成任务的命令重投必须视为成功，不得重投到死信")

	fresh := time.Now()
	_, running := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskRunning, 1, &fresh, nil)
	err = handler.Handle(ctx, &ent.OperationalCommand{
		CommandType:   commandbus.CommandExecuteProvisioningTask,
		AggregateType: "provisioning_task",
		AggregateID:   running.ID,
		TenantID:      tenantID,
		Payload:       map[string]interface{}{"actorUserId": float64(requesterID)},
	})
	require.NoError(t, err, "别人正在执行不是失败，本命令的目标正在被达成")

	// provider 真实失败必须仍然返回错误，让命令总线按策略重投。
	_, failing := seedProvisioningTask(t, client, ctx, tenantID, requesterID, provisioning.TaskPending, 0, nil,
		map[string]any{"compliance_ack": false})
	err = handler.Handle(ctx, &ent.OperationalCommand{
		CommandType:   commandbus.CommandExecuteProvisioningTask,
		AggregateType: "provisioning_task",
		AggregateID:   failing.ID,
		TenantID:      tenantID,
		Payload:       map[string]interface{}{"actorUserId": float64(requesterID)},
	})
	require.Error(t, err, "provider 执行失败必须冒泡，否则重投机制形同虚设")
}
