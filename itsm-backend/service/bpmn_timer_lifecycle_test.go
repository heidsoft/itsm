package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ============================================================================
// 流程生命周期 × 定时器生命周期（Phase 2 尾巴：Cancel+Recreate）
//
// 挂起 → 取消实例全部 pending 定时器（防止对挂起实例触发→重试风暴）；
// 恢复 → 重建当前活动的定时器（intermediate / boundary）；
// 终止 → 取消实例全部 pending 定时器。
// ============================================================================

func pendingTimerCount(t *testing.T, store *memoryTimerStore, ctx context.Context, tenantID int) int {
	t.Helper()
	_, total, err := store.List(ctx, TimerListFilter{
		TenantID: tenantID,
		Status:   string(TimerStatusPending),
	})
	require.NoError(t, err)
	return total
}

// TestE2E_SuspendCancelsIntermediateTimers 挂起阻塞在中间定时事件上的实例，
// 应取消其 pending intermediate timer。
func TestE2E_SuspendCancelsIntermediateTimers(t *testing.T) {
	client := newE2ETestClient(t, "e2e_suspend_timer")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "suspend")

	deployE2EProcess(t, ctx, client, tenantID, "e2e_proc", testBPMNE2EFullFlow)
	timerStore := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	instance, err := engine.StartProcess(ctx, "e2e_proc", "BK-SUS-001", nil)
	require.NoError(t, err)
	require.Equal(t, 1, pendingTimerCount(t, timerStore, ctx, tenantID),
		"启动后应有一个 pending intermediate timer")

	require.NoError(t, engine.SuspendProcess(ctx, instance.ProcessInstanceID, "等待外部材料"))

	assert.Equal(t, 0, pendingTimerCount(t, timerStore, ctx, tenantID),
		"挂起后不应残留 pending timer")

	suspended, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "suspended", suspended.Status)
}

// TestE2E_ResumeReregistersIntermediateTimer 恢复阻塞在中间定时事件上的实例，
// 应重建该 intermediate timer。
func TestE2E_ResumeReregistersIntermediateTimer(t *testing.T) {
	client := newE2ETestClient(t, "e2e_resume_timer")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "resume")

	deployE2EProcess(t, ctx, client, tenantID, "e2e_proc", testBPMNE2EFullFlow)
	timerStore := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	instance, err := engine.StartProcess(ctx, "e2e_proc", "BK-RES-001", nil)
	require.NoError(t, err)

	require.NoError(t, engine.SuspendProcess(ctx, instance.ProcessInstanceID, "挂起"))
	require.NoError(t, engine.ResumeProcess(ctx, instance.ProcessInstanceID))

	timers, total, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  tenantID,
		TimerType: string(TimerTypeIntermediate),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	require.Equal(t, 1, total, "恢复后应重建 intermediate timer")
	assert.Equal(t, "timer_wait", timers[0].ActivityID)
	assert.Equal(t, "PT5S", timers[0].TimerExpression)
	assert.Equal(t, instance.ID, timers[0].ProcessInstanceID)

	running, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", running.Status)
	assert.Equal(t, "timer_wait", running.CurrentActivityID)
}

// TestE2E_SuspendResumeReregistersBoundaryTimers 挂起停在用户任务上的实例
// 再恢复，应取消并重建绑定到该任务的 boundary timer。
func TestE2E_SuspendResumeReregistersBoundaryTimers(t *testing.T) {
	client := newE2ETestClient(t, "e2e_suspend_boundary")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "suspend_b")

	deployE2EProcess(t, ctx, client, tenantID, "e2e_proc", testBPMNE2EFullFlow)
	timerStore := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	instance, err := engine.StartProcess(ctx, "e2e_proc", "BK-SUSB-001", nil)
	require.NoError(t, err)

	// 推进过中间定时事件到用户任务（生产中由 scheduler 触发）
	intermediate, _, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  tenantID,
		TimerType: string(TimerTypeIntermediate),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	require.Len(t, intermediate, 1)
	_, err = timerStore.CASFire(ctx, intermediate[0].TimerID, time.Now())
	require.NoError(t, err)
	handler := NewTimerEventHandler(engine, logger)
	require.NoError(t, handler.HandleTimerFire(ctx, &TimerRecord{
		TimerID:              intermediate[0].TimerID,
		TimerType:            "intermediate",
		ProcessDefinitionKey: "e2e_proc",
		ProcessInstanceID:    instance.ID,
		ActivityID:           "timer_wait",
		FireAt:               intermediate[0].FireAt,
		TenantID:             tenantID,
	}))

	// 此时停在 review_task，boundary timer 已注册
	require.Equal(t, 1, pendingTimerCount(t, timerStore, ctx, tenantID))

	require.NoError(t, engine.SuspendProcess(ctx, instance.ProcessInstanceID, "挂起"))
	assert.Equal(t, 0, pendingTimerCount(t, timerStore, ctx, tenantID),
		"挂起后 boundary timer 应被取消")

	require.NoError(t, engine.ResumeProcess(ctx, instance.ProcessInstanceID))
	boundaryTimers, total, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  tenantID,
		TimerType: string(TimerTypeBoundary),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	require.Equal(t, 1, total, "恢复后应重建 boundary timer")
	assert.Equal(t, "timeout_boundary", boundaryTimers[0].ActivityID)
	assert.Equal(t, "review_task", func() string {
		inst, err := client.ProcessInstance.Get(ctx, instance.ID)
		require.NoError(t, err)
		return inst.CurrentActivityID
	}())
}

// TestE2E_TerminateCancelsTimers 终止实例应取消全部 pending 定时器。
func TestE2E_TerminateCancelsTimers(t *testing.T) {
	client := newE2ETestClient(t, "e2e_terminate_timer")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "term")

	deployE2EProcess(t, ctx, client, tenantID, "e2e_proc", testBPMNE2EFullFlow)
	timerStore := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	instance, err := engine.StartProcess(ctx, "e2e_proc", "BK-TERM-001", nil)
	require.NoError(t, err)
	require.Equal(t, 1, pendingTimerCount(t, timerStore, ctx, tenantID))

	require.NoError(t, engine.TerminateProcess(ctx, instance.ProcessInstanceID, "客户取消"))

	assert.Equal(t, 0, pendingTimerCount(t, timerStore, ctx, tenantID),
		"终止后不应残留 pending timer")

	terminated, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "terminated", terminated.Status)
}
