package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"itsm-backend/ent"
	"itsm-backend/metrics"

	"go.uber.org/zap"
)

type TimerFireCallback func(ctx context.Context, timer *TimerRecord) error

// timerSyncInterval 周期性同步间隔；timerSyncLookahead 为未来窗口接驳的前瞻跨度，
// 取 2 倍 tick 以保证新 timer 在 fire_at 之前至少被扫描到一次。
const (
	timerSyncInterval  = 1 * time.Minute
	timerSyncLookahead = 2 * time.Minute
)

type TimerRecord struct {
	TimerID              string
	TimerType            string
	ProcessDefinitionKey string
	ProcessInstanceID    int
	ActivityID           string
	FireAt               time.Time
	TenantID             int
	// Phase 5：start timer 触发后需要按原表达式重排下一跳，故回调必须携带表达式上下文。
	TimerExpression  string
	ExpressionType   string
	ContextVariables map[string]interface{}
}

type TimerScheduler struct {
	client   *ent.Client
	store    TimerStore
	logger   *zap.SugaredLogger
	callback TimerFireCallback
	now      func() time.Time

	mu      sync.Mutex
	timers  map[string]*time.Timer
	running bool
	stopCh  chan struct{}

	metrics *TimerMetrics
}

type TimerMetrics struct {
	mu               sync.Mutex
	FiredTotal       int64
	RecoveryTotal    int64
	RetryTotal       int64
	FireLatencySum   time.Duration
	FireLatencyCount int64
}

func NewTimerMetrics() *TimerMetrics {
	return &TimerMetrics{}
}

func (m *TimerMetrics) RecordFire(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FiredTotal++
	m.FireLatencySum += latency
	m.FireLatencyCount++
}

func (m *TimerMetrics) RecordRecovery() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RecoveryTotal++
}

func (m *TimerMetrics) RecordRetry() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RetryTotal++
}

func (m *TimerMetrics) Snapshot() (fired int64, recovery int64, retry int64, avgLatency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FireLatencyCount > 0 {
		avgLatency = m.FireLatencySum / time.Duration(m.FireLatencyCount)
	}
	return m.FiredTotal, m.RecoveryTotal, m.RetryTotal, avgLatency
}

type TimerSchedulerConfig struct {
	Client   *ent.Client
	Store    TimerStore
	Logger   *zap.SugaredLogger
	Callback TimerFireCallback
	Now      func() time.Time
}

func NewTimerScheduler(cfg TimerSchedulerConfig) *TimerScheduler {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &TimerScheduler{
		client:   cfg.Client,
		store:    cfg.Store,
		logger:   cfg.Logger,
		callback: cfg.Callback,
		now:      now,
		timers:   make(map[string]*time.Timer),
		stopCh:   make(chan struct{}),
		metrics:  NewTimerMetrics(),
	}
}

func (s *TimerScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("timer scheduler already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("timer scheduler starting recovery scan")
	if err := s.recover(ctx); err != nil {
		s.logger.Errorf("recovery scan failed: %v", err)
	}

	go s.periodicSync(ctx)

	return nil
}

func (s *TimerScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)

	for timerID, timer := range s.timers {
		timer.Stop()
		s.logger.Debugf("stopped timer %s on scheduler shutdown", timerID)
	}
	s.timers = make(map[string]*time.Timer)
	s.logger.Info("timer scheduler stopped")
}

func (s *TimerScheduler) Schedule(ctx context.Context, record *TimerRecord) error {
	delay := record.FireAt.Sub(s.now())
	if delay <= 0 {
		delay = time.Millisecond
	}

	s.mu.Lock()
	if existing, ok := s.timers[record.TimerID]; ok {
		existing.Stop()
	}

	timer := time.AfterFunc(delay, func() {
		s.fireCallback(ctx, record)
	})
	s.timers[record.TimerID] = timer
	s.mu.Unlock()

	s.logger.Debugf("scheduled timer %s to fire at %s (delay=%s)", record.TimerID, record.FireAt.Format(time.RFC3339), delay)
	return nil
}

func (s *TimerScheduler) Cancel(timerID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	timer, ok := s.timers[timerID]
	if !ok {
		return false
	}
	stopped := timer.Stop()
	delete(s.timers, timerID)
	return stopped
}

func (s *TimerScheduler) fireCallback(ctx context.Context, record *TimerRecord) {
	s.mu.Lock()
	delete(s.timers, record.TimerID)
	s.mu.Unlock()

	now := s.now()
	latency := now.Sub(record.FireAt)
	if latency < 0 {
		latency = 0
	}

	s.logger.Infof("timer %s fired (type=%s, tenant=%d, latency=%s)", record.TimerID, record.TimerType, record.TenantID, latency)

	dbTimer, err := s.store.CASFire(ctx, record.TimerID, now)
	if err != nil {
		s.logger.Errorf("CAS fire failed for timer %s: %v", record.TimerID, err)
		metrics.TimerFiredTotal.WithLabelValues(record.TimerType, "cas_failed", strconv.Itoa(record.TenantID)).Inc()
		return
	}

	if s.callback != nil {
		callbackRecord := &TimerRecord{
			TimerID:              dbTimer.TimerID,
			TimerType:            dbTimer.TimerType,
			ProcessDefinitionKey: dbTimer.ProcessDefinitionKey,
			ProcessInstanceID:    dbTimer.ProcessInstanceID,
			ActivityID:           dbTimer.ActivityID,
			FireAt:               dbTimer.FireAt,
			TenantID:             dbTimer.TenantID,
			TimerExpression:      dbTimer.TimerExpression,
			ExpressionType:       dbTimer.ExpressionType,
			ContextVariables:     dbTimer.ContextVariables,
		}

		if err := s.callback(ctx, callbackRecord); err != nil {
			s.logger.Errorf("timer callback failed for %s: %v", record.TimerID, err)
			metrics.TimerFiredTotal.WithLabelValues(record.TimerType, "failed", strconv.Itoa(record.TenantID)).Inc()
			s.handleFireFailure(ctx, record, err)
			return
		}
	}

	metrics.TimerFiredTotal.WithLabelValues(record.TimerType, "success", strconv.Itoa(record.TenantID)).Inc()
	metrics.TimerFireLatency.WithLabelValues(record.TimerType, strconv.Itoa(record.TenantID)).Observe(latency.Seconds())
	s.metrics.RecordFire(latency)
}

func (s *TimerScheduler) handleFireFailure(ctx context.Context, record *TimerRecord, fireErr error) {
	s.metrics.RecordRetry()
	metrics.TimerRetryTotal.WithLabelValues(record.TimerType, strconv.Itoa(record.TenantID)).Inc()

	dbTimer, err := s.store.GetByTimerID(ctx, record.TimerID)
	if err != nil {
		s.logger.Errorf("failed to load timer %s for failure handling: %v", record.TimerID, err)
		return
	}

	newRetryCount := dbTimer.RetryCount + 1
	var nextFireAt *time.Time
	if newRetryCount < dbTimer.MaxRetries {
		backoff := time.Duration(1<<uint(dbTimer.RetryCount)) * time.Minute
		if backoff > 30*time.Minute {
			backoff = 30 * time.Minute
		}
		t := s.now().Add(backoff)
		nextFireAt = &t
	}

	reason := fireErr.Error()
	if len(reason) > 2000 {
		reason = reason[:2000]
	}

	if _, err := s.store.CASFail(ctx, record.TimerID, reason, newRetryCount, nextFireAt); err != nil {
		s.logger.Errorf("CAS fail failed for timer %s: %v", record.TimerID, err)
		return
	}

	if nextFireAt != nil {
		retryRecord := &TimerRecord{
			TimerID:              record.TimerID,
			TimerType:            record.TimerType,
			ProcessDefinitionKey: record.ProcessDefinitionKey,
			ProcessInstanceID:    record.ProcessInstanceID,
			ActivityID:           record.ActivityID,
			FireAt:               *nextFireAt,
			TenantID:             record.TenantID,
		}
		if err := s.Schedule(ctx, retryRecord); err != nil {
			s.logger.Errorf("failed to reschedule timer %s: %v", record.TimerID, err)
		}
	}
}

func (s *TimerScheduler) recover(ctx context.Context) error {
	tenants, err := s.listTenants(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tenants for recovery: %w", err)
	}

	now := s.now()
	totalRecovered := 0

	for _, tenantID := range tenants {
		overdue, err := s.store.FindPendingDue(ctx, tenantID, now)
		if err != nil {
			s.logger.Errorf("recovery: failed to find overdue timers for tenant %d: %v", tenantID, err)
			continue
		}
		for _, timer := range overdue {
			record := entTimerToRecord(timer)
			if err := s.Schedule(ctx, record); err != nil {
				s.logger.Errorf("recovery: failed to schedule overdue timer %s: %v", timer.TimerID, err)
				continue
			}
			totalRecovered++
			s.metrics.RecordRecovery()
			metrics.TimerRecoveryTotal.WithLabelValues(strconv.Itoa(tenantID)).Inc()
		}

		future, err := s.store.FindPendingFuture(ctx, tenantID, now)
		if err != nil {
			s.logger.Errorf("recovery: failed to find future timers for tenant %d: %v", tenantID, err)
			continue
		}
		for _, timer := range future {
			record := entTimerToRecord(timer)
			if err := s.Schedule(ctx, record); err != nil {
				s.logger.Errorf("recovery: failed to schedule future timer %s: %v", timer.TimerID, err)
				continue
			}
			totalRecovered++
		}

		staleFired, err := s.store.FindFiredStale(ctx, now.Add(-5*time.Minute))
		if err != nil {
			s.logger.Errorf("recovery: failed to find stale fired timers: %v", err)
			continue
		}
		for _, timer := range staleFired {
			s.logger.Warnf("recovery: stale fired timer %s (fire_at=%s) detected, may need manual review", timer.TimerID, timer.FireAt.Format(time.RFC3339))
		}
	}

	s.logger.Infof("recovery scan complete: %d timers rescheduled across %d tenants", totalRecovered, len(tenants))
	return nil
}

func (s *TimerScheduler) periodicSync(ctx context.Context) {
	ticker := time.NewTicker(timerSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.syncWithDB(ctx)
		}
	}
}

func (s *TimerScheduler) syncWithDB(ctx context.Context) {
	now := s.now()
	tenants, err := s.listTenants(ctx)
	if err != nil {
		s.logger.Errorf("periodic sync: failed to list tenants: %v", err)
		return
	}

	for _, tenantID := range tenants {
		overdue, err := s.store.FindPendingDue(ctx, tenantID, now)
		if err != nil {
			s.logger.Errorf("periodic sync: failed to find overdue timers for tenant %d: %v", tenantID, err)
			continue
		}
		for _, timer := range overdue {
			s.mu.Lock()
			_, exists := s.timers[timer.TimerID]
			s.mu.Unlock()
			if !exists {
				record := entTimerToRecord(timer)
				if err := s.Schedule(ctx, record); err != nil {
					s.logger.Errorf("periodic sync: failed to schedule timer %s: %v", timer.TimerID, err)
				}
			}
		}

		// 未来窗口接驳：运行期新建的 timer（流程重新部署注册的 start timer、
		// 触发后重排的下一跳等）必须在其 fire_at 之前进入内存调度器，
		// 否则只能等它逾期后才被上面的 overdue 分支拾起（触发时间漂移到下一个 tick）。
		// 窗口取 2 倍 tick 以保证"在 fire_at 之前至少被扫描到一次"。
		future, err := s.store.FindPendingFuture(ctx, tenantID, now)
		if err != nil {
			s.logger.Errorf("periodic sync: failed to find future timers for tenant %d: %v", tenantID, err)
			continue
		}
		horizon := now.Add(timerSyncLookahead)
		for _, timer := range future {
			if timer.FireAt.After(horizon) {
				break // 已按 fire_at 升序，后续更远，无需继续
			}
			s.mu.Lock()
			_, exists := s.timers[timer.TimerID]
			s.mu.Unlock()
			if exists {
				continue
			}
			if err := s.Schedule(ctx, entTimerToRecord(timer)); err != nil {
				s.logger.Errorf("periodic sync: failed to schedule future timer %s: %v", timer.TimerID, err)
			}
		}
	}
}

func entTimerToRecord(timer *ent.ProcessTimer) *TimerRecord {
	return &TimerRecord{
		TimerID:              timer.TimerID,
		TimerType:            timer.TimerType,
		ProcessDefinitionKey: timer.ProcessDefinitionKey,
		ProcessInstanceID:    timer.ProcessInstanceID,
		ActivityID:           timer.ActivityID,
		FireAt:               timer.FireAt,
		TenantID:             timer.TenantID,
		TimerExpression:      timer.TimerExpression,
		ExpressionType:       timer.ExpressionType,
		ContextVariables:     timer.ContextVariables,
	}
}

func (s *TimerScheduler) listTenants(ctx context.Context) ([]int, error) {
	tenants, err := s.client.Tenant.Query().IDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("query tenants failed: %w", err)
	}
	return tenants, nil
}

func (s *TimerScheduler) Metrics() *TimerMetrics {
	return s.metrics
}
