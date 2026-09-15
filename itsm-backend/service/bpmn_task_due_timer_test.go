package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/processtask"
)

// ============================================================================
// 任务截止 Timer 化（Phase 4）：
// BPMN userTask dueDate 属性 → ProcessTask.due_date 落库 + task_due 定时器注册；
// 到期回调分发 TimeoutScanner 四动作；扫描器与 timer 双路径 claim-once 安全。
// ============================================================================

const testBPMNTaskDue = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="task_due_proc" name="Task Due Flow" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="deadline_task" name="有截止时间的任务" dueDate="2099-01-01" timeoutAction="notify"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="deadline_task"/>
    <bpmn:sequenceFlow id="f2" sourceRef="deadline_task" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

const testBPMNNoDue = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="no_due_proc" name="No Due Flow" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="plain_task" name="无截止时间"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="plain_task"/>
    <bpmn:sequenceFlow id="f2" sourceRef="plain_task" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

// startTaskDueProcess 部署流程并启动，返回实例与创建出的任务。
func startTaskDueProcess(t *testing.T, client *ent.Client, bpmnXML, key, bk string) (*ent.ProcessInstance, *ent.ProcessTask, *memoryTimerStore, context.Context, *CustomProcessEngine) {
	t.Helper()
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, key)

	deployE2EProcess(t, ctx, client, tenantID, key, bpmnXML)
	timerStore := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	instance, err := engine.StartProcess(ctx, key, bk, nil)
	require.NoError(t, err)

	task, err := client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(instance.ID)).
		First(ctx)
	require.NoError(t, err)
	return instance, task, timerStore, ctx, engine
}

// TestE2E_BPMDueDatePersistedAndTimerRegistered BPMN dueDate 属性应落库并注册 task_due 定时器。
func TestE2E_BPMDueDatePersistedAndTimerRegistered(t *testing.T) {
	client := newE2ETestClient(t, "e2e_due_reg")
	instance, task, timerStore, ctx, _ := startTaskDueProcess(t, client, testBPMNTaskDue, "task_due_proc", "BK-DUE-001")

	require.False(t, task.DueDate.IsZero(), "BPMN dueDate 应落库到 ProcessTask.due_date")
	assert.Equal(t, 23, task.DueDate.Hour(), "截止语义为当日 23:59:59")
	assert.Equal(t, 59, task.DueDate.Minute())

	timers, total, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  task.TenantID,
		TimerType: string(TimerTypeTaskDue),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	require.Equal(t, 1, total, "应注册一个 task_due timer")
	assert.Equal(t, task.TaskID, timers[0].ActivityID, "ActivityID 承载唯一任务 ID")
	assert.Equal(t, instance.ID, timers[0].ProcessInstanceID)
}

// TestE2E_BPMDueDateAbsentSkipsTimer 无 dueDate 属性的流程不应注册 task_due 定时器。
func TestE2E_BPMDueDateAbsentSkipsTimer(t *testing.T) {
	client := newE2ETestClient(t, "e2e_due_absent")
	_, task, timerStore, ctx, _ := startTaskDueProcess(t, client, testBPMNNoDue, "no_due_proc", "BK-DUE-002")

	require.True(t, task.DueDate.IsZero(), "无 dueDate 属性时不应写 due_date")

	_, total, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  task.TenantID,
		TimerType: string(TimerTypeTaskDue),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

// TestE2E_TaskDueTimerFiresNotify task_due timer 到期应把任务置为 timeout（默认 notify 动作）。
func TestE2E_TaskDueTimerFiresNotify(t *testing.T) {
	client := newE2ETestClient(t, "e2e_due_fire")
	_, task, timerStore, ctx, engine := startTaskDueProcess(t, client, testBPMNTaskDue, "task_due_proc", "BK-DUE-003")

	timers, _, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  task.TenantID,
		TimerType: string(TimerTypeTaskDue),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	require.Len(t, timers, 1)

	// 生产中 scheduler CAS-fire 后回调 handler
	_, err = timerStore.CASFire(ctx, timers[0].TimerID, time.Now())
	require.NoError(t, err)

	handler := NewTimerEventHandler(engine, zaptest.NewLogger(t).Sugar())
	require.NoError(t, handler.HandleTimerFire(ctx, &TimerRecord{
		TimerID:              timers[0].TimerID,
		TimerType:            string(TimerTypeTaskDue),
		ProcessDefinitionKey: "task_due_proc",
		ProcessInstanceID:    instanceIDOf(t, client, task),
		ActivityID:           task.TaskID,
		FireAt:               timers[0].FireAt,
		TenantID:             task.TenantID,
	}))

	timedOut, err := client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "timeout", timedOut.Status, "notify 动作应把任务置为 timeout")
}

// TestE2E_TaskDueTimerCancelledOnCompletion 任务完成时应取消 task_due 定时器；
// handler 的活跃检查作为兜底（定时器已 fire 但任务已完成的场景静默确认）。
func TestE2E_TaskDueTimerCancelledOnCompletion(t *testing.T) {
	client := newE2ETestClient(t, "e2e_due_cancel")
	_, task, timerStore, ctx, engine := startTaskDueProcess(t, client, testBPMNTaskDue, "task_due_proc", "BK-DUE-004")

	// 人工完成任务 → task_due timer 应被同步取消
	ctxWithUser := withUser(ctx, 0)
	require.NoError(t, engine.CompleteTask(ctxWithUser, task.TaskID, map[string]interface{}{"approvalDecision": "approve"}))

	_, total, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  task.TenantID,
		TimerType: string(TimerTypeTaskDue),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total, "任务完成后 task_due timer 应被取消")

	// 兜底路径：假设 timer 已 fire 而任务已完成 → handler 静默确认不报错
	handler := NewTimerEventHandler(engine, zaptest.NewLogger(t).Sugar())
	require.NoError(t, handler.HandleTimerFire(ctx, &TimerRecord{
		TimerID:              "task-due-simulated",
		TimerType:            string(TimerTypeTaskDue),
		ProcessDefinitionKey: "task_due_proc",
		ProcessInstanceID:    instanceIDOf(t, client, task),
		ActivityID:           task.TaskID,
		FireAt:               time.Now().Add(-time.Hour),
		TenantID:             task.TenantID,
	}), "任务已完成的 timer 回调应静默成功，避免无意义重试")

	done, err := client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.NotEqual(t, "timeout", done.Status, "已完成任务不应被改写为 timeout")
}

// TestTimeoutScanner_ClaimOnce 已被处理（非活跃）的任务不应被重复分发。
func TestTimeoutScanner_ClaimOnce(t *testing.T) {
	client := newE2ETestClient(t, "scanner_claim")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "claim")

	deployE2EProcess(t, ctx, client, tenantID, "task_due_proc", testBPMNTaskDue)
	timerStore := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)
	instance, err := engine.StartProcess(ctx, "task_due_proc", "BK-DUE-005", nil)
	require.NoError(t, err)
	task, err := client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(instance.ID)).
		First(ctx)
	require.NoError(t, err)

	scanner := NewTimeoutScanner(client, logger)

	// 模拟另一路径（timer 回调）已处理：状态置为 timeout
	require.NoError(t, scanner.actionNotify(ctx, task, tenantID))

	// 第二路径（恢复扫描）拿着过期快照再分发 → claim 失败静默返回
	staleTask, err := client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	require.NoError(t, scanner.actionNotify(ctx, staleTask, tenantID), "claim-once 下重复分发不应报错")

	after, err := client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "timeout", after.Status)
}

func instanceIDOf(t *testing.T, client *ent.Client, task *ent.ProcessTask) int {
	t.Helper()
	inst, err := client.ProcessInstance.Get(context.Background(), task.ProcessInstanceID)
	require.NoError(t, err)
	return inst.ID
}
