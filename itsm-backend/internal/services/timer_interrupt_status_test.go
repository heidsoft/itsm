package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent/processtask"
)

// interruptibleTaskStatuses 是边界计时器中断必须取消的在途状态集合（审计 P1-08）。
// 覆盖「任务仍可被推进/完成、从而让被中断活动继续存在」的全部状态：
// 正常认领写 assigned（bpmn_task_service.go claim），转派写 delegated；
// started/waiting/in_progress 为存量值（新代码不再写入），一并覆盖老数据。
// finalizing 有意排除：它是会签末票瞬时态，由 TimeoutScanner 恢复，见 interruptActivity 注释。
var interruptibleTaskStatuses = []string{
	"created",
	"assigned",
	"started",
	"delegated",
	"waiting",
	"in_progress",
}

// nonInterruptibleTaskStatuses 是不得被中断改写的终态/恢复态。
var nonInterruptibleTaskStatuses = []string{
	"completed",
	"cancelled",
	"timeout",
	"finalizing",
}

// TestInterruptActivity_CancelsAllInFlightStatuses 断言中断会取消全部在途状态的任务，
// 且不触碰终态/恢复态。旧实现只取消 created/waiting/in_progress，已认领(assigned)与
// 已转派(delegated)的任务会残留，边界超时路径推进后旧任务仍可被完成，形成双分支并行。
func TestInterruptActivity_CancelsAllInFlightStatuses(t *testing.T) {
	client := newTimerEventHandlerTestClient(t, "interrupt-status")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "intr")
	ctx := context.Background()

	const activityID = "review_task"
	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithBoundaryTimer)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, activityID)

	statusByTaskID := map[string]string{}
	for _, status := range append(append([]string{}, interruptibleTaskStatuses...), nonInterruptibleTaskStatuses...) {
		taskID := fmt.Sprintf("task-%s", status)
		_, err := client.ProcessTask.Create().
			SetTaskID(taskID).
			SetProcessInstanceID(instance.ID).
			SetProcessDefinitionKey("test_proc").
			SetTaskDefinitionKey(activityID).
			SetTaskName("Review Request").
			SetTaskType("userTask").
			SetStatus(status).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err, "fixture task %s", status)
		statusByTaskID[taskID] = status
	}

	// 同一实例的另一个活动不应被波及（按 TaskDefinitionKey 过滤）。
	const otherTaskID = "task-other-activity-assigned"
	_, err := client.ProcessTask.Create().
		SetTaskID(otherTaskID).
		SetProcessInstanceID(instance.ID).
		SetProcessDefinitionKey("test_proc").
		SetTaskDefinitionKey("other_task").
		SetTaskName("Other").
		SetTaskType("userTask").
		SetStatus("assigned").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	handler := NewTimerEventHandler(engine, logger)

	require.NoError(t, handler.interruptActivity(ctx, instance, activityID))

	tasks, err := client.ProcessTask.Query().Where(processtask.ProcessInstanceID(instance.ID)).All(ctx)
	require.NoError(t, err)
	statusByActualID := map[string]string{}
	for _, task := range tasks {
		statusByActualID[task.TaskID] = task.Status
	}

	for _, status := range interruptibleTaskStatuses {
		taskID := fmt.Sprintf("task-%s", status)
		assert.Equal(t, "cancelled", statusByActualID[taskID],
			"在途状态 %q 必须被边界中断取消（P1-08）", status)
	}
	for _, status := range nonInterruptibleTaskStatuses {
		taskID := fmt.Sprintf("task-%s", status)
		assert.Equal(t, status, statusByActualID[taskID],
			"终态/恢复态 %q 不得被中断改写", status)
	}
	assert.Equal(t, "assigned", statusByActualID[otherTaskID], "其他活动的任务不得被中断")
}

// TestInterruptActivity_NoInFlightTasksIsNoop 无在途任务时中断必须成功且不报错
// （边界计时器常在任务已被完成的瞬间触发）。
func TestInterruptActivity_NoInFlightTasksIsNoop(t *testing.T) {
	client := newTimerEventHandlerTestClient(t, "interrupt-noop")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "noop")
	ctx := context.Background()

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithBoundaryTimer)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "review_task")

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	handler := NewTimerEventHandler(engine, logger)

	require.NoError(t, handler.interruptActivity(ctx, instance, "review_task"))
}
