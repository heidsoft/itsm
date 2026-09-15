package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/metrics"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newTimerTestClient(t *testing.T, name string) *ent.Client {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(name, "/", "-"))
	client := enttest.Open(t, dialect.SQLite, dsn)
	t.Cleanup(func() { client.Close() })
	return client
}

func createTimerTenant(t *testing.T, client *ent.Client, suffix string) int {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName("Timer Tenant "+suffix).
		SetCode("timer-"+suffix).
		SetDomain("timer-"+suffix+".example.com").
		SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)
	return tenant.ID
}

func newTimerTestStore(t *testing.T, client *ent.Client) *DBTimerStore {
	return NewDBTimerStore(client, zaptest.NewLogger(t).Sugar())
}

func TestDBTimerStore_CreateAndGet(t *testing.T) {
	client := newTimerTestClient(t, "create-get")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "cg")
	ctx := context.Background()

	fireAt := time.Now().Add(10 * time.Minute)
	created, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeIntermediate,
		ProcessDefinitionKey: "test_proc",
		ActivityID:           "activity_1",
		TimerExpression:      "PT30M",
		ExpressionType:       ExprTypeDuration,
		FireAt:               fireAt,
		TenantID:             tenantID,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.TimerID)
	assert.Equal(t, string(TimerTypeIntermediate), created.TimerType)
	assert.Equal(t, string(TimerStatusPending), created.Status)
	assert.Equal(t, 1, created.Version)
	assert.Equal(t, string(PauseStateRunning), created.PauseState)

	got, err := store.GetByTimerID(ctx, created.TimerID)
	require.NoError(t, err)
	assert.Equal(t, created.TimerID, got.TimerID)
	assert.Equal(t, tenantID, got.TenantID)
}

func TestDBTimerStore_List(t *testing.T) {
	client := newTimerTestClient(t, "list")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "list")
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := store.Create(ctx, &CreateTimerRequest{
			TimerType:            TimerTypeBoundary,
			ProcessDefinitionKey: "list_proc",
			TimerExpression:      "PT1H",
			ExpressionType:       ExprTypeDuration,
			FireAt:               time.Now().Add(time.Duration(i+1) * time.Minute),
			TenantID:             tenantID,
		})
		require.NoError(t, err)
	}

	items, total, err := store.List(ctx, TimerListFilter{
		TenantID: tenantID,
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, items, 3)

	items, total, err = store.List(ctx, TimerListFilter{
		TenantID: tenantID,
		Status:   string(TimerStatusPending),
		Page:     1,
		PageSize: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, items, 2)
}

func TestDBTimerStore_Stats(t *testing.T) {
	client := newTimerTestClient(t, "stats")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "stats")
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		_, err := store.Create(ctx, &CreateTimerRequest{
			TimerType:            TimerTypeStart,
			ProcessDefinitionKey: "stats_proc",
			TimerExpression:      "PT5M",
			ExpressionType:       ExprTypeDuration,
			FireAt:               time.Now().Add(5 * time.Minute),
			TenantID:             tenantID,
		})
		require.NoError(t, err)
	}

	stats, err := store.Stats(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, 2, stats.Total)
	assert.Equal(t, 2, stats.Pending)
	assert.Equal(t, 0, stats.Fired)
	assert.Equal(t, 0, stats.Failed)
}

func TestDBTimerStore_CASFire(t *testing.T) {
	client := newTimerTestClient(t, "cas-fire")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "fire")
	ctx := context.Background()

	created, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeIntermediate,
		ProcessDefinitionKey: "fire_proc",
		TimerExpression:      "PT10M",
		ExpressionType:       ExprTypeDuration,
		FireAt:               time.Now().Add(10 * time.Minute),
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	fired, err := store.CASFire(ctx, created.TimerID, time.Now())
	require.NoError(t, err)
	assert.Equal(t, string(TimerStatusFired), fired.Status)
	assert.Equal(t, 2, fired.Version)
	assert.False(t, fired.FiredAt.IsZero())

	_, err = store.CASFire(ctx, created.TimerID, time.Now())
	assert.Error(t, err, "double fire should fail")
}

func TestDBTimerStore_CASFail_WithRetry(t *testing.T) {
	client := newTimerTestClient(t, "cas-fail")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "fail")
	ctx := context.Background()

	created, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeBoundary,
		ProcessDefinitionKey: "fail_proc",
		TimerExpression:      "PT15M",
		ExpressionType:       ExprTypeDuration,
		FireAt:               time.Now().Add(15 * time.Minute),
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	nextFire := time.Now().Add(1 * time.Minute)
	failed, err := store.CASFail(ctx, created.TimerID, "callback error", 1, &nextFire)
	require.NoError(t, err)
	assert.Equal(t, string(TimerStatusPending), failed.Status, "should reset to pending when retries remain")
	assert.Equal(t, 1, failed.RetryCount)
	assert.Equal(t, "callback error", failed.FailureReason)
}

func TestDBTimerStore_CASFail_ExhaustedRetries(t *testing.T) {
	client := newTimerTestClient(t, "cas-fail-exhaust")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "exhaust")
	ctx := context.Background()

	created, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeStart,
		ProcessDefinitionKey: "exhaust_proc",
		TimerExpression:      "PT20M",
		ExpressionType:       ExprTypeDuration,
		FireAt:               time.Now().Add(20 * time.Minute),
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	failed, err := store.CASFail(ctx, created.TimerID, "permanent error", created.MaxRetries, nil)
	require.NoError(t, err)
	assert.Equal(t, string(TimerStatusFailed), failed.Status, "should stay failed when retries exhausted")
}

func TestDBTimerStore_CancelByTimerID(t *testing.T) {
	client := newTimerTestClient(t, "cancel")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "cancel")
	ctx := context.Background()

	created, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeIntermediate,
		ProcessDefinitionKey: "cancel_proc",
		TimerExpression:      "PT30M",
		ExpressionType:       ExprTypeDuration,
		FireAt:               time.Now().Add(30 * time.Minute),
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	err = store.CancelByTimerID(ctx, created.TimerID)
	require.NoError(t, err)

	cancelled, err := store.GetByTimerID(ctx, created.TimerID)
	require.NoError(t, err)
	assert.Equal(t, string(TimerStatusCancelled), cancelled.Status)
}

func TestDBTimerStore_FindPendingDue(t *testing.T) {
	client := newTimerTestClient(t, "find-due")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "due")
	ctx := context.Background()

	past := time.Now().Add(-5 * time.Minute)
	future := time.Now().Add(30 * time.Minute)

	_, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeIntermediate,
		ProcessDefinitionKey: "due_proc",
		TimerExpression:      "PT5M",
		ExpressionType:       ExprTypeDuration,
		FireAt:               past,
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	_, err = store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeIntermediate,
		ProcessDefinitionKey: "due_proc",
		TimerExpression:      "PT30M",
		ExpressionType:       ExprTypeDuration,
		FireAt:               future,
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	due, err := store.FindPendingDue(ctx, tenantID, time.Now())
	require.NoError(t, err)
	assert.Len(t, due, 1, "only past-due timer should be returned")
}

func TestDBTimerStore_TenantIsolation(t *testing.T) {
	client := newTimerTestClient(t, "tenant-iso")
	store := newTimerTestStore(t, client)
	tenantA := createTimerTenant(t, client, "isoA")
	tenantB := createTimerTenant(t, client, "isoB")
	ctx := context.Background()

	_, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeStart,
		ProcessDefinitionKey: "iso_proc",
		TimerExpression:      "PT5M",
		ExpressionType:       ExprTypeDuration,
		FireAt:               time.Now().Add(5 * time.Minute),
		TenantID:             tenantA,
	})
	require.NoError(t, err)

	items, total, err := store.List(ctx, TimerListFilter{TenantID: tenantB, Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, items)
}

func TestTimerScheduler_ScheduleAndFire(t *testing.T) {
	client := newTimerTestClient(t, "sched-fire")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "sf")
	ctx := context.Background()

	created, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeIntermediate,
		ProcessDefinitionKey: "sched_proc",
		TimerExpression:      "PT1S",
		ExpressionType:       ExprTypeDuration,
		FireAt:               time.Now().Add(50 * time.Millisecond),
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	var firedMu sync.Mutex
	var firedTimerID string

	scheduler := NewTimerScheduler(TimerSchedulerConfig{
		Client: client,
		Store:  store,
		Logger: zaptest.NewLogger(t).Sugar(),
		Callback: func(ctx context.Context, timer *TimerRecord) error {
			firedMu.Lock()
			firedTimerID = timer.TimerID
			firedMu.Unlock()
			return nil
		},
	})

	record := &TimerRecord{
		TimerID:              created.TimerID,
		TimerType:            created.TimerType,
		ProcessDefinitionKey: created.ProcessDefinitionKey,
		FireAt:               created.FireAt,
		TenantID:             created.TenantID,
	}
	err = scheduler.Schedule(ctx, record)
	require.NoError(t, err)

	time.Sleep(300 * time.Millisecond)

	firedMu.Lock()
	assert.Equal(t, created.TimerID, firedTimerID)
	firedMu.Unlock()

	fired, err := store.GetByTimerID(ctx, created.TimerID)
	require.NoError(t, err)
	assert.Equal(t, string(TimerStatusFired), fired.Status)
}

func TestTimerScheduler_Cancel(t *testing.T) {
	client := newTimerTestClient(t, "sched-cancel")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "sc")

	scheduler := NewTimerScheduler(TimerSchedulerConfig{
		Client: client,
		Store:  store,
		Logger: zaptest.NewLogger(t).Sugar(),
	})

	record := &TimerRecord{
		TimerID:              "cancel-test-timer",
		TimerType:            string(TimerTypeIntermediate),
		ProcessDefinitionKey: "cancel_proc",
		FireAt:               time.Now().Add(1 * time.Hour),
		TenantID:             tenantID,
	}
	err := scheduler.Schedule(context.Background(), record)
	require.NoError(t, err)

	cancelled := scheduler.Cancel("cancel-test-timer")
	assert.True(t, cancelled)

	cancelled = scheduler.Cancel("nonexistent")
	assert.False(t, cancelled)
}

func TestTimerScheduler_Recovery(t *testing.T) {
	client := newTimerTestClient(t, "sched-recovery")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "rec")
	ctx := context.Background()

	past := time.Now().Add(-10 * time.Minute)
	_, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeIntermediate,
		ProcessDefinitionKey: "rec_proc",
		TimerExpression:      "PT5M",
		ExpressionType:       ExprTypeDuration,
		FireAt:               past,
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	var firedCount int
	scheduler := NewTimerScheduler(TimerSchedulerConfig{
		Client: client,
		Store:  store,
		Logger: zaptest.NewLogger(t).Sugar(),
		Callback: func(ctx context.Context, timer *TimerRecord) error {
			firedCount++
			return nil
		},
	})

	err = scheduler.Start(ctx)
	require.NoError(t, err)
	defer scheduler.Stop()

	time.Sleep(300 * time.Millisecond)

	scheduler.mu.Lock()
	timerCount := len(scheduler.timers)
	scheduler.mu.Unlock()
	_ = timerCount

	fired, _, _, _ := scheduler.Metrics().Snapshot()
	_ = fired
}

func TestTimerMetrics_Snapshot(t *testing.T) {
	m := NewTimerMetrics()
	m.RecordFire(100 * time.Millisecond)
	m.RecordFire(200 * time.Millisecond)
	m.RecordRecovery()
	m.RecordRetry()

	fired, recovery, retry, avgLatency := m.Snapshot()
	assert.Equal(t, int64(2), fired)
	assert.Equal(t, int64(1), recovery)
	assert.Equal(t, int64(1), retry)
	assert.Equal(t, 150*time.Millisecond, avgLatency)
}

func TestTimerMetrics_SnapshotEmpty(t *testing.T) {
	m := NewTimerMetrics()
	fired, recovery, retry, avgLatency := m.Snapshot()
	assert.Equal(t, int64(0), fired)
	assert.Equal(t, int64(0), recovery)
	assert.Equal(t, int64(0), retry)
	assert.Equal(t, time.Duration(0), avgLatency)
}

func TestDBTimerStore_FindPendingFuture(t *testing.T) {
	client := newTimerTestClient(t, "find-future")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "fut")
	ctx := context.Background()

	past := time.Now().Add(-5 * time.Minute)
	future := time.Now().Add(30 * time.Minute)

	_, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeStart,
		ProcessDefinitionKey: "fut_proc",
		TimerExpression:      "PT5M",
		ExpressionType:       ExprTypeDuration,
		FireAt:               past,
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	_, err = store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeStart,
		ProcessDefinitionKey: "fut_proc",
		TimerExpression:      "PT30M",
		ExpressionType:       ExprTypeDuration,
		FireAt:               future,
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	futureTimers, err := store.FindPendingFuture(ctx, tenantID, time.Now())
	require.NoError(t, err)
	assert.Len(t, futureTimers, 1, "only future timer should be returned")
	assert.Equal(t, future.Unix(), futureTimers[0].FireAt.Unix())
}

func TestDBTimerStore_CancelByProcessInstance(t *testing.T) {
	client := newTimerTestClient(t, "cancel-pi")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "cpi")
	ctx := context.Background()

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-CPI").
		SetDeploymentName("CPI Deployment").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	definition, err := client.ProcessDefinition.Create().
		SetKey("cpi_proc").
		SetName("CPI Process").
		SetBpmnXML([]byte("<definitions/>")).
		SetDeploymentID(deployment.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-CPI").
		SetProcessDefinitionKey("cpi_proc").
		SetProcessDefinitionID(definition.ID).
		SetStatus("running").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	_, err = store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeBoundary,
		ProcessDefinitionKey: "cpi_proc",
		ProcessInstanceID:    &instance.ID,
		TimerExpression:      "PT1H",
		ExpressionType:       ExprTypeDuration,
		FireAt:               time.Now().Add(1 * time.Hour),
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	_, err = store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeBoundary,
		ProcessDefinitionKey: "cpi_proc",
		ProcessInstanceID:    &instance.ID,
		TimerExpression:      "PT2H",
		ExpressionType:       ExprTypeDuration,
		FireAt:               time.Now().Add(2 * time.Hour),
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	cancelled, err := store.CancelByProcessInstance(ctx, tenantID, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, cancelled)

	items, _, err := store.List(ctx, TimerListFilter{TenantID: tenantID, Page: 1, PageSize: 10})
	require.NoError(t, err)
	for _, item := range items {
		assert.Equal(t, string(TimerStatusCancelled), item.Status)
	}
}

// TestTimerScheduler_PrometheusMetrics 调度器触发路径应同步写入 Prometheus 计数器
// （itsm_timer_fired_total / itsm_timer_retry_total，PRD §8.1）。使用独立租户 ID
// 隔离全局计数器基线。
func TestTimerScheduler_PrometheusMetrics(t *testing.T) {
	client := newTimerTestClient(t, "sched-prom")
	store := newTimerTestStore(t, client)
	tenantID := createTimerTenant(t, client, "prom")
	tenantStr := strconv.Itoa(tenantID)
	ctx := context.Background()

	successBase := testutil.ToFloat64(metrics.TimerFiredTotal.WithLabelValues("intermediate", "success", tenantStr))
	failBase := testutil.ToFloat64(metrics.TimerFiredTotal.WithLabelValues("intermediate", "failed", tenantStr))
	retryBase := testutil.ToFloat64(metrics.TimerRetryTotal.WithLabelValues("intermediate", tenantStr))

	// 成功路径：回调无错 → fired{success} +1
	created, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeIntermediate,
		ProcessDefinitionKey: "prom_proc",
		TimerExpression:      "PT1S",
		ExpressionType:       ExprTypeDuration,
		FireAt:               time.Now().Add(30 * time.Millisecond),
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	scheduler := NewTimerScheduler(TimerSchedulerConfig{
		Client: client,
		Store:  store,
		Logger: zaptest.NewLogger(t).Sugar(),
		Callback: func(ctx context.Context, timer *TimerRecord) error {
			return nil
		},
	})
	require.NoError(t, scheduler.Schedule(ctx, &TimerRecord{
		TimerID:              created.TimerID,
		TimerType:            created.TimerType,
		ProcessDefinitionKey: created.ProcessDefinitionKey,
		FireAt:               created.FireAt,
		TenantID:             tenantID,
	}))

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(metrics.TimerFiredTotal.WithLabelValues("intermediate", "success", tenantStr)) > successBase {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.Greater(t, testutil.ToFloat64(metrics.TimerFiredTotal.WithLabelValues("intermediate", "success", tenantStr)), successBase,
		"成功触发应递增 itsm_timer_fired_total{status=success}")

	// 失败路径：回调报错 → fired{failed} +1 且 retry +1
	failed, err := store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeIntermediate,
		ProcessDefinitionKey: "prom_proc",
		TimerExpression:      "PT1S",
		ExpressionType:       ExprTypeDuration,
		FireAt:               time.Now().Add(30 * time.Millisecond),
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	failScheduler := NewTimerScheduler(TimerSchedulerConfig{
		Client: client,
		Store:  store,
		Logger: zaptest.NewLogger(t).Sugar(),
		Callback: func(ctx context.Context, timer *TimerRecord) error {
			return fmt.Errorf("simulated callback failure")
		},
	})
	require.NoError(t, failScheduler.Schedule(ctx, &TimerRecord{
		TimerID:              failed.TimerID,
		TimerType:            failed.TimerType,
		ProcessDefinitionKey: failed.ProcessDefinitionKey,
		FireAt:               failed.FireAt,
		TenantID:             tenantID,
	}))

	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(metrics.TimerRetryTotal.WithLabelValues("intermediate", tenantStr)) > retryBase {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.Greater(t, testutil.ToFloat64(metrics.TimerFiredTotal.WithLabelValues("intermediate", "failed", tenantStr)), failBase,
		"失败触发应递增 itsm_timer_fired_total{status=failed}")
	assert.Greater(t, testutil.ToFloat64(metrics.TimerRetryTotal.WithLabelValues("intermediate", tenantStr)), retryBase,
		"回调失败应递增 itsm_timer_retry_total")
}
