package service

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"itsm-backend/ent/processtimer"
	"itsm-backend/metrics"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestTimerScheduler_StaleFiredReclaimedAndRedriven 崩溃恢复：CASFire 在回调之前
// 落库，进程在回调前崩溃会把 timer 永久卡在 fired。滞留回收必须把它收回 pending
// 并重新驱动到 completed，业务效果（回调）不得丢失。
func TestTimerScheduler_StaleFiredReclaimedAndRedriven(t *testing.T) {
	client := newTimerTestClient(t, "stale-fired")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "sfr")
	ctx := context.Background()

	created, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeIntermediate,
		ProcessDefinitionKey: "stale_proc",
		TimerExpression:      "PT1S",
		ExpressionType:       ExprTypeDuration,
		FireAt:               time.Now().Add(-time.Minute),
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	// 模拟崩溃窗口：触发已落库（fired），回调从未执行。
	_, err = store.CASFire(ctx, created.TimerID, time.Now())
	require.NoError(t, err)
	stuck, err := store.GetByTimerID(ctx, created.TimerID)
	require.NoError(t, err)
	require.Equal(t, string(TimerStatusFired), stuck.Status)

	// 回收阈值按 UpdatedAt 判定，显式回拨越过阈值。
	_, err = client.ProcessTimer.Update().
		Where(processtimer.ID(stuck.ID)).
		SetUpdatedAt(time.Now().Add(-timerStaleFiredThreshold - time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	var mu sync.Mutex
	var callbackRuns int
	scheduler := NewTimerScheduler(TimerSchedulerConfig{
		Client: client,
		Store:  store,
		Logger: zaptest.NewLogger(t).Sugar(),
		Callback: func(_ context.Context, _ *TimerRecord) error {
			mu.Lock()
			callbackRuns++
			mu.Unlock()
			return nil
		},
	})

	require.NoError(t, scheduler.reclaimStaleFired(ctx, time.Now()))
	reclaimed, err := store.GetByTimerID(ctx, created.TimerID)
	require.NoError(t, err)
	require.Equal(t, string(TimerStatusPending), reclaimed.Status, "滞留 fired 必须被收回 pending")

	// 下一个同步周期拾起并重新驱动到 completed。
	scheduler.syncWithDB(ctx)
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	runs := callbackRuns
	mu.Unlock()
	assert.Equal(t, 1, runs, "重驱后回调必须恰好执行一次")

	final, err := store.GetByTimerID(ctx, created.TimerID)
	require.NoError(t, err)
	assert.Equal(t, string(TimerStatusCompleted), final.Status, "重驱成功后必须收敛为 completed")
}

// TestTimerScheduler_DeadLetterOnRetryExhaustion 重试预算耗尽必须进入终态 failed
// （死信），并递增死信指标；死信不会有自动重试，等待运维显式 replay。
func TestTimerScheduler_DeadLetterOnRetryExhaustion(t *testing.T) {
	client := newTimerTestClient(t, "dead-letter")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "dl")
	ctx := context.Background()

	created, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeIntermediate,
		ProcessDefinitionKey: "dl_proc",
		TimerExpression:      "PT1S",
		ExpressionType:       ExprTypeDuration,
		FireAt:               time.Now().Add(-time.Second),
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	scheduler := NewTimerScheduler(TimerSchedulerConfig{
		Client:   client,
		Store:    store,
		Logger:   zaptest.NewLogger(t).Sugar(),
		Callback: func(_ context.Context, _ *TimerRecord) error { return errors.New("business step failed") },
	})

	metric := metrics.TimerDeadLetterTotal.WithLabelValues("intermediate", strconv.Itoa(tenantID))
	before := testutil.ToFloat64(metric)

	_, err = store.CASFire(ctx, created.TimerID, time.Now())
	require.NoError(t, err)
	record := mustTimerRecord(t, store, ctx, created.TimerID)
	// MaxRetries 默认 3：三次失败处理分别累计 retry 1/2/3，第三次落入终态 failed。
	for i := 0; i < 3; i++ {
		scheduler.handleFireFailure(ctx, record, errors.New("business step failed"))
		record = mustTimerRecord(t, store, ctx, created.TimerID)
	}

	final, err := store.GetByTimerID(ctx, created.TimerID)
	require.NoError(t, err)
	assert.Equal(t, string(TimerStatusFailed), final.Status, "重试耗尽必须进入终态 failed")
	assert.Equal(t, 3, final.RetryCount)
	assert.NotEmpty(t, final.FailureReason)

	after := testutil.ToFloat64(metric)
	assert.GreaterOrEqual(t, after, before+1, "死信必须递增 itsm_timer_dead_letter_total")
}

func mustTimerRecord(t *testing.T, store *DBTimerStore, ctx context.Context, timerID string) *TimerRecord {
	t.Helper()
	timer, err := store.GetByTimerID(ctx, timerID)
	require.NoError(t, err)
	return entTimerToRecord(timer)
}

// TestTimerStore_CASCompleteAndReplayGuards timer 状态机的 CAS 守卫：
// completed 只能从 fired 收敛，reclaim 只能回收 fired，replay 只接受 failed/fired。
func TestTimerStore_CASCompleteAndReplayGuards(t *testing.T) {
	client := newTimerTestClient(t, "cas-complete")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "csg")
	ctx := context.Background()

	mkTimer := func(suffix string) string {
		created, err := store.Create(ctx, &CreateTimerRequest{
			TimerType:            TimerTypeIntermediate,
			ProcessDefinitionKey: "guard_proc_" + suffix,
			TimerExpression:      "PT1S",
			ExpressionType:       ExprTypeDuration,
			FireAt:               time.Now().Add(-time.Second),
			TenantID:             tenantID,
		})
		require.NoError(t, err)
		return created.TimerID
	}

	// complete：pending 不能直接收敛。
	t1 := mkTimer("c1")
	_, err := store.CASComplete(ctx, t1)
	require.Error(t, err, "pending 不能直接 complete")

	_, err = store.CASFire(ctx, t1, time.Now())
	require.NoError(t, err)
	completed, err := store.CASComplete(ctx, t1)
	require.NoError(t, err)
	assert.Equal(t, string(TimerStatusCompleted), completed.Status)

	// reclaim：completed 不是孤儿，不得回收。
	_, err = store.CASReclaim(ctx, t1)
	require.Error(t, err, "completed 不得被回收")

	// replay：completed 不可重放。
	_, err = store.Replay(ctx, tenantID, t1)
	require.Error(t, err, "completed 不可重放")

	// reclaim：fired 孤儿可以收回 pending。
	t2 := mkTimer("c2")
	_, err = store.CASFire(ctx, t2, time.Now())
	require.NoError(t, err)
	reclaimed, err := store.CASReclaim(ctx, t2)
	require.NoError(t, err)
	assert.Equal(t, string(TimerStatusPending), reclaimed.Status)

	// replay：failed 死信重置为 pending，重试预算与失败原因清零。
	t3 := mkTimer("c3")
	_, err = store.CASFire(ctx, t3, time.Now())
	require.NoError(t, err)
	_, err = store.CASFail(ctx, t3, "boom", 3, nil)
	require.NoError(t, err)
	replayed, err := store.Replay(ctx, tenantID, t3)
	require.NoError(t, err)
	assert.Equal(t, string(TimerStatusPending), replayed.Status)
	assert.Equal(t, 0, replayed.RetryCount)
	assert.Empty(t, replayed.FailureReason)

	// replay：跨租户同形掩码。
	_, err = store.Replay(ctx, tenantID+1000, t3)
	require.Error(t, err)
}
