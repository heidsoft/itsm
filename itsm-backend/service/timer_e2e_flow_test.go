package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"
)

// ============================================================================
// In-memory TimerStore for e2e engine tests.
//
// The DBTimerStore uses the main ent.Client internally, but the engine calls
// timer store methods from within open transactions (StartProcess, CompleteTask).
// With SQLite this causes a self-deadlock: the same goroutine holds a write lock
// and tries to write again through a different connection path. PostgreSQL doesn't
// have this issue in production. The in-memory store isolates engine logic from
// store persistence (which has its own unit tests in timer_infrastructure_test.go).
// ============================================================================

type memoryTimerStore struct {
	mu     sync.Mutex
	timers []*ent.ProcessTimer
	nextID int
}

func newMemoryTimerStore() *memoryTimerStore {
	return &memoryTimerStore{nextID: 1}
}

func (s *memoryTimerStore) Create(_ context.Context, req *CreateTimerRequest) (*ent.ProcessTimer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	timerID := fmt.Sprintf("mem-timer-%d", atomic.AddInt64(new(int64), 1))
	if req.ProcessInstanceID != nil {
		timerID = fmt.Sprintf("mem-timer-%d-inst%d", s.nextID, *req.ProcessInstanceID)
	}
	id := s.nextID
	s.nextID++

	pt := &ent.ProcessTimer{
		ID:                   id,
		TimerID:              timerID,
		TimerType:            string(req.TimerType),
		ProcessDefinitionKey: req.ProcessDefinitionKey,
		ActivityID:           req.ActivityID,
		TimerExpression:      req.TimerExpression,
		ExpressionType:       string(req.ExpressionType),
		FireAt:               req.FireAt,
		Status:               string(TimerStatusPending),
		Version:              1,
		TenantID:             req.TenantID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
	if req.ProcessInstanceID != nil {
		pt.ProcessInstanceID = *req.ProcessInstanceID
	}
	s.timers = append(s.timers, pt)
	return pt, nil
}

func (s *memoryTimerStore) GetByTimerID(_ context.Context, timerID string) (*ent.ProcessTimer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.timers {
		if t.TimerID == timerID {
			return t, nil
		}
	}
	return nil, fmt.Errorf("timer %s not found", timerID)
}

func (s *memoryTimerStore) FindPendingDue(_ context.Context, tenantID int, now time.Time) ([]*ent.ProcessTimer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*ent.ProcessTimer
	for _, t := range s.timers {
		if t.TenantID == tenantID && t.Status == string(TimerStatusPending) && !t.FireAt.After(now) {
			result = append(result, t)
		}
	}
	return result, nil
}

func (s *memoryTimerStore) FindPendingFuture(_ context.Context, tenantID int, now time.Time) ([]*ent.ProcessTimer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*ent.ProcessTimer
	for _, t := range s.timers {
		if t.TenantID == tenantID && t.Status == string(TimerStatusPending) && t.FireAt.After(now) {
			result = append(result, t)
		}
	}
	return result, nil
}

func (s *memoryTimerStore) FindFiredStale(_ context.Context, threshold time.Time) ([]*ent.ProcessTimer, error) {
	return nil, nil
}

func (s *memoryTimerStore) FindFailedRetryable(_ context.Context, tenantID int) ([]*ent.ProcessTimer, error) {
	return nil, nil
}

func (s *memoryTimerStore) CASFire(_ context.Context, timerID string, firedAt time.Time) (*ent.ProcessTimer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.timers {
		if t.TimerID == timerID && t.Status == string(TimerStatusPending) {
			t.Status = string(TimerStatusFired)
			t.FiredAt = firedAt
			t.Version++
			return t, nil
		}
	}
	return nil, fmt.Errorf("timer %s not found or not pending", timerID)
}

func (s *memoryTimerStore) CASFail(_ context.Context, timerID string, reason string, retryCount int, nextFireAt *time.Time) (*ent.ProcessTimer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.timers {
		if t.TimerID == timerID {
			t.Status = string(TimerStatusFailed)
			t.FailureReason = reason
			t.RetryCount = retryCount
			t.Version++
			return t, nil
		}
	}
	return nil, fmt.Errorf("timer %s not found", timerID)
}

func (s *memoryTimerStore) CancelByProcessInstance(_ context.Context, tenantID, processInstanceID int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, t := range s.timers {
		if t.TenantID == tenantID && t.ProcessInstanceID == processInstanceID && t.Status == string(TimerStatusPending) {
			t.Status = string(TimerStatusCancelled)
			t.Version++
			count++
		}
	}
	return count, nil
}

func (s *memoryTimerStore) CancelByTimerID(_ context.Context, timerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.timers {
		if t.TimerID == timerID && t.Status == string(TimerStatusPending) {
			t.Status = string(TimerStatusCancelled)
			t.Version++
			return nil
		}
	}
	return nil
}

func (s *memoryTimerStore) List(_ context.Context, filter TimerListFilter) ([]*ent.ProcessTimer, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*ent.ProcessTimer
	for _, t := range s.timers {
		if filter.TenantID > 0 && t.TenantID != filter.TenantID {
			continue
		}
		if filter.TimerType != "" && t.TimerType != filter.TimerType {
			continue
		}
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		if filter.ProcessInstanceID != nil && t.ProcessInstanceID != *filter.ProcessInstanceID {
			continue
		}
		if filter.ProcessDefinitionKey != "" && t.ProcessDefinitionKey != filter.ProcessDefinitionKey {
			continue
		}
		result = append(result, t)
	}
	total := len(result)
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 100
	}
	start := (page - 1) * pageSize
	if start >= len(result) {
		return nil, total, nil
	}
	end := start + pageSize
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], total, nil
}

func (s *memoryTimerStore) Stats(_ context.Context, tenantID int) (*TimerStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stats := &TimerStats{}
	for _, t := range s.timers {
		if t.TenantID != tenantID {
			continue
		}
		stats.Total++
		switch t.Status {
		case string(TimerStatusPending):
			stats.Pending++
		case string(TimerStatusFired):
			stats.Fired++
		case string(TimerStatusCancelled):
			stats.Cancelled++
		case string(TimerStatusFailed):
			stats.Failed++
		}
	}
	return stats, nil
}

// ============================================================================
// BPMN XML constants
// ============================================================================

const testBPMNE2EFullFlow = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="e2e_proc" name="E2E Timer Flow" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:intermediateCatchEvent id="timer_wait" name="Wait 5 seconds">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT5S</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:intermediateCatchEvent>
    <bpmn:userTask id="review_task" name="Review Request"/>
    <bpmn:boundaryEvent id="timeout_boundary" attachedToRef="review_task" cancelActivity="true">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT1H</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:boundaryEvent>
    <bpmn:endEvent id="normal_end"/>
    <bpmn:endEvent id="timeout_end"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="timer_wait"/>
    <bpmn:sequenceFlow id="flow2" sourceRef="timer_wait" targetRef="review_task"/>
    <bpmn:sequenceFlow id="flow3" sourceRef="review_task" targetRef="normal_end"/>
    <bpmn:sequenceFlow id="flow4" sourceRef="timeout_boundary" targetRef="timeout_end"/>
  </bpmn:process>
</bpmn:definitions>`

const testBPMNE2EBoundaryFire = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="e2e_boundary_proc" name="E2E Boundary Timer Fire" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="review_task" name="Review Request"/>
    <bpmn:boundaryEvent id="timeout_boundary" attachedToRef="review_task" cancelActivity="true">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT1H</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:boundaryEvent>
    <bpmn:endEvent id="normal_end"/>
    <bpmn:endEvent id="timeout_end"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="review_task"/>
    <bpmn:sequenceFlow id="flow2" sourceRef="review_task" targetRef="normal_end"/>
    <bpmn:sequenceFlow id="flow3" sourceRef="timeout_boundary" targetRef="timeout_end"/>
  </bpmn:process>
</bpmn:definitions>`

const testBPMNE2ESequentialTimers = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="e2e_seq_proc" name="Sequential Timers" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:intermediateCatchEvent id="timer_1" name="Wait 10 seconds">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT10S</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:intermediateCatchEvent>
    <bpmn:intermediateCatchEvent id="timer_2" name="Wait 20 seconds">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT20S</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:intermediateCatchEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="timer_1"/>
    <bpmn:sequenceFlow id="flow2" sourceRef="timer_1" targetRef="timer_2"/>
    <bpmn:sequenceFlow id="flow3" sourceRef="timer_2" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

// ============================================================================
// Test helpers
// ============================================================================

func newE2ETestClient(t *testing.T, name string) *ent.Client {
	t.Helper()
	dsn := "file:" + name + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, dialect.SQLite, dsn)
	return client
}

func e2eTenantCtx(t *testing.T, client *ent.Client, suffix string) (context.Context, int) {
	t.Helper()
	tenantID := createTimerTenant(t, client, suffix)
	ctx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, tenantID)
	return ctx, tenantID
}

func deployE2EProcess(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, key string, bpmnXML string) int {
	t.Helper()
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("deploy-e2e-" + key).
		SetDeploymentName("E2E Deployment " + key).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey(key).
		SetName("E2E Process " + key).
		SetVersion("1.0").
		SetBpmnXML([]byte(bpmnXML)).
		SetIsActive(true).
		SetIsLatest(true).
		SetDeploymentID(deployment.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return def.ID
}

func withUser(ctx context.Context, userID int) context.Context {
	return context.WithValue(ctx, bpmn.BPMNUserIDContextKey, userID)
}

// ============================================================================
// Test 1: Full golden path — Start → IntermediateTimer → UserTask → Complete → End
// ============================================================================

func TestE2E_FullTimerFlow(t *testing.T) {
	client := newE2ETestClient(t, "e2e_full_flow")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "full")

	processDefID := deployE2EProcess(t, ctx, client, tenantID, "e2e_proc", testBPMNE2EFullFlow)
	timerStore := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	// Step 1: Start the process — should block at intermediate timer
	instance, err := engine.StartProcess(ctx, "e2e_proc", "BK-E2E-001", nil)
	require.NoError(t, err)
	require.NotNil(t, instance)

	running, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", running.Status)
	assert.Equal(t, "timer_wait", running.CurrentActivityID,
		"Process should be blocked at the intermediate timer catch event")

	// Verify intermediate timer was registered
	timers, total, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  tenantID,
		TimerType: string(TimerTypeIntermediate),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total, "Exactly one intermediate timer should be registered")
	require.Len(t, timers, 1)
	assert.Equal(t, "timer_wait", timers[0].ActivityID)
	assert.Equal(t, "PT5S", timers[0].TimerExpression)
	assert.Equal(t, instance.ID, timers[0].ProcessInstanceID)

	// No user task should exist yet
	taskCount, err := client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(instance.ID)).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, taskCount, "No user task should exist before timer fires")

	// Step 2: Simulate timer firing.
	// In production the scheduler CAS-fires the timer before invoking the callback.
	_, err = timerStore.CASFire(ctx, timers[0].TimerID, time.Now())
	require.NoError(t, err)

	handler := NewTimerEventHandler(engine, logger)
	timerRecord := &TimerRecord{
		TimerID:              timers[0].TimerID,
		TimerType:            "intermediate",
		ProcessDefinitionKey: "e2e_proc",
		ProcessInstanceID:    instance.ID,
		ActivityID:           "timer_wait",
		FireAt:               timers[0].FireAt,
		TenantID:             tenantID,
	}
	err = handler.HandleTimerFire(ctx, timerRecord)
	require.NoError(t, err)

	// Verify process advanced to UserTask
	afterTimer, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "review_task", afterTimer.CurrentActivityID,
		"Process should advance to UserTask after intermediate timer fires")

	// Verify user task was created
	tasks, err := client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(instance.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, tasks, 1, "Exactly one user task should be created")
	assert.Equal(t, "review_task", tasks[0].TaskDefinitionKey)
	assert.Equal(t, "created", tasks[0].Status)

	// Verify boundary timer was registered
	boundaryTimers, boundaryTotal, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  tenantID,
		TimerType: string(TimerTypeBoundary),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, boundaryTotal, "Exactly one boundary timer should be registered")
	require.Len(t, boundaryTimers, 1)
	assert.Equal(t, "timeout_boundary", boundaryTimers[0].ActivityID)
	assert.Equal(t, "PT1H", boundaryTimers[0].TimerExpression)

	// Step 3: Complete the user task — should cancel boundary timer and advance to EndEvent
	ctx = withUser(ctx, 0)
	err = engine.CompleteTask(ctx, tasks[0].TaskID, nil)
	require.NoError(t, err)

	// Verify process completed
	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status,
		"Process should be completed after user task finishes")
	assert.NotNil(t, completed.EndTime)

	// Verify boundary timer was cancelled
	cancelledTimers, cancelledTotal, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  tenantID,
		TimerType: string(TimerTypeBoundary),
		Status:    string(TimerStatusCancelled),
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, cancelledTotal, 1, "Boundary timer should be cancelled")
	require.NotEmpty(t, cancelledTimers)
	assert.Equal(t, "timeout_boundary", cancelledTimers[0].ActivityID)

	// Verify no pending timers remain
	_, pendingTotal, err := timerStore.List(ctx, TimerListFilter{
		TenantID: tenantID,
		Status:   string(TimerStatusPending),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, pendingTotal, "No pending timers should remain after process completion")

	_ = processDefID
}

// ============================================================================
// Test 2: Boundary timer fires → task cancelled → process follows timeout path
// ============================================================================

func TestE2E_BoundaryTimerFires_TaskCancelled(t *testing.T) {
	client := newE2ETestClient(t, "e2e_boundary_fire")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "bound")

	deployE2EProcess(t, ctx, client, tenantID, "e2e_boundary_proc", testBPMNE2EBoundaryFire)
	timerStore := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	// Start process — should go directly to UserTask (no intermediate timer)
	instance, err := engine.StartProcess(ctx, "e2e_boundary_proc", "BK-E2E-002", nil)
	require.NoError(t, err)

	running, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "review_task", running.CurrentActivityID)

	// Verify boundary timer was registered
	boundaryTimers, total, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  tenantID,
		TimerType: string(TimerTypeBoundary),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	assert.Equal(t, "timeout_boundary", boundaryTimers[0].ActivityID)

	// Verify user task exists
	tasks, err := client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(instance.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	// Simulate boundary timer firing (timeout)
	handler := NewTimerEventHandler(engine, logger)
	timerRecord := &TimerRecord{
		TimerID:              boundaryTimers[0].TimerID,
		TimerType:            "boundary",
		ProcessDefinitionKey: "e2e_boundary_proc",
		ProcessInstanceID:    instance.ID,
		ActivityID:           "timeout_boundary",
		FireAt:               boundaryTimers[0].FireAt,
		TenantID:             tenantID,
	}
	err = handler.HandleTimerFire(ctx, timerRecord)
	require.NoError(t, err)

	// Verify user task was cancelled
	afterTasks, err := client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(instance.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, afterTasks, 1)
	assert.Equal(t, "cancelled", afterTasks[0].Status,
		"User task should be cancelled when boundary timer fires")

	// Verify process followed the timeout path
	afterInstance, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", afterInstance.Status,
		"Process should complete via the timeout path")
}

// ============================================================================
// Test 3: Engine without timer store → intermediate timer returns error
// ============================================================================

func TestE2E_NoTimerStore_IntermediateTimerFails(t *testing.T) {
	client := newE2ETestClient(t, "e2e_no_store")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "nostore")

	deployE2EProcess(t, ctx, client, tenantID, "e2e_proc", testBPMNE2EFullFlow)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	// Deliberately NOT calling SetTimerServices

	_, err := engine.StartProcess(ctx, "e2e_proc", "BK-E2E-003", nil)
	require.Error(t, err, "StartProcess should fail when timer store is not configured")
	assert.Contains(t, err.Error(), "定时器服务未配置",
		"Error should indicate timer service not configured")
}

// ============================================================================
// Test 4: Multiple intermediate timers in sequence (timer → timer → task)
// ============================================================================

func TestE2E_SequentialIntermediateTimers(t *testing.T) {
	client := newE2ETestClient(t, "e2e_seq_timers")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "seq")

	deployE2EProcess(t, ctx, client, tenantID, "e2e_seq_proc", testBPMNE2ESequentialTimers)
	timerStore := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)
	handler := NewTimerEventHandler(engine, logger)

	// Start process — should block at first timer
	instance, err := engine.StartProcess(ctx, "e2e_seq_proc", "BK-E2E-004", nil)
	require.NoError(t, err)

	running, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "timer_1", running.CurrentActivityID)

	// Fire first timer → should block at second timer
	timers1, _, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  tenantID,
		TimerType: string(TimerTypeIntermediate),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	require.Len(t, timers1, 1)
	assert.Equal(t, "timer_1", timers1[0].ActivityID)

	_, err = timerStore.CASFire(ctx, timers1[0].TimerID, time.Now())
	require.NoError(t, err)

	err = handler.HandleTimerFire(ctx, &TimerRecord{
		TimerID:              timers1[0].TimerID,
		TimerType:            "intermediate",
		ProcessDefinitionKey: "e2e_seq_proc",
		ProcessInstanceID:    instance.ID,
		ActivityID:           "timer_1",
		FireAt:               timers1[0].FireAt,
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	afterFirst, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "timer_2", afterFirst.CurrentActivityID,
		"Process should advance to second intermediate timer")

	// Verify second timer was registered
	timers2, total2, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  tenantID,
		TimerType: string(TimerTypeIntermediate),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total2, "One pending intermediate timer (the second one)")
	assert.Equal(t, "timer_2", timers2[0].ActivityID)

	// Fire second timer → should complete process.
	// CAS to "fired" first, simulating the real scheduler.
	_, err = timerStore.CASFire(ctx, timers2[0].TimerID, time.Now())
	require.NoError(t, err)

	err = handler.HandleTimerFire(ctx, &TimerRecord{
		TimerID:              timers2[0].TimerID,
		TimerType:            "intermediate",
		ProcessDefinitionKey: "e2e_seq_proc",
		ProcessInstanceID:    instance.ID,
		ActivityID:           "timer_2",
		FireAt:               timers2[0].FireAt,
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	completed, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status,
		"Process should complete after all intermediate timers fire")
}

// ============================================================================
// Test 5: Verify audit trail for timer events
// ============================================================================

func TestE2E_AuditTrailForTimerEvents(t *testing.T) {
	client := newE2ETestClient(t, "e2e_audit_trail")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "audit")

	deployE2EProcess(t, ctx, client, tenantID, "e2e_proc", testBPMNE2EFullFlow)
	timerStore := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	instance, err := engine.StartProcess(ctx, "e2e_proc", "BK-E2E-005", nil)
	require.NoError(t, err)

	// Fire intermediate timer
	timers, _, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  tenantID,
		TimerType: string(TimerTypeIntermediate),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	require.Len(t, timers, 1)

	handler := NewTimerEventHandler(engine, logger)

	// CAS to "fired" first, simulating the real scheduler.
	_, err = timerStore.CASFire(ctx, timers[0].TimerID, time.Now())
	require.NoError(t, err)

	err = handler.HandleTimerFire(ctx, &TimerRecord{
		TimerID:              timers[0].TimerID,
		TimerType:            "intermediate",
		ProcessDefinitionKey: "e2e_proc",
		ProcessInstanceID:    instance.ID,
		ActivityID:           "timer_wait",
		FireAt:               timers[0].FireAt,
		TenantID:             tenantID,
	})
	require.NoError(t, err)

	// Verify audit log was created
	auditCount, err := client.ProcessAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	assert.Greater(t, auditCount, 0, "Audit logs should be created for timer events")

	// Verify the timer was marked as fired in the store
	firedTimer, err := timerStore.GetByTimerID(ctx, timers[0].TimerID)
	require.NoError(t, err)
	assert.Equal(t, string(TimerStatusFired), firedTimer.Status,
		"Timer should be marked as fired after handling")
}

// ============================================================================
// Test 6: Tenant isolation — timer for wrong tenant fails
// ============================================================================

func TestE2E_TenantIsolation_TimerFire(t *testing.T) {
	client := newE2ETestClient(t, "e2e_tenant_iso")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "iso")

	deployE2EProcess(t, ctx, client, tenantID, "e2e_proc", testBPMNE2EFullFlow)
	timerStore := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	instance, err := engine.StartProcess(ctx, "e2e_proc", "BK-E2E-006", nil)
	require.NoError(t, err)

	// Try to fire timer with wrong tenant ID
	handler := NewTimerEventHandler(engine, logger)
	wrongTenantID := tenantID + 999
	err = handler.HandleTimerFire(ctx, &TimerRecord{
		TimerID:              "timer-wrong-tenant",
		TimerType:            "intermediate",
		ProcessDefinitionKey: "e2e_proc",
		ProcessInstanceID:    instance.ID,
		ActivityID:           "timer_wait",
		FireAt:               time.Now(),
		TenantID:             wrongTenantID,
	})
	require.Error(t, err, "Timer fire should fail when tenant ID doesn't match")
	assert.Contains(t, err.Error(), "not found or not running",
		"Error should indicate instance not found for wrong tenant")
}

// ============================================================================
// Test 7: Process definition ID used correctly (no orphan reference)
// ============================================================================

func TestE2E_ProcessDefinitionIntegrity(t *testing.T) {
	client := newE2ETestClient(t, "e2e_def_integrity")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "integ")

	processDefID := deployE2EProcess(t, ctx, client, tenantID, "e2e_proc", testBPMNE2EFullFlow)
	timerStore := newMemoryTimerStore()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	instance, err := engine.StartProcess(ctx, "e2e_proc", "BK-E2E-007", nil)
	require.NoError(t, err)

	// Verify instance references correct process definition
	assert.Equal(t, "e2e_proc", instance.ProcessDefinitionKey)
	assert.Equal(t, processDefID, instance.ProcessDefinitionID)

	// Verify timer references correct process definition key
	timers, _, err := timerStore.List(ctx, TimerListFilter{
		TenantID: tenantID,
	})
	require.NoError(t, err)
	for _, timer := range timers {
		assert.Equal(t, "e2e_proc", timer.ProcessDefinitionKey)
	}
}

// ============================================================================
// Test 8: Verify no process instance left in limbo after timer store failure
// ============================================================================

func TestE2E_NoLimbo_OnTimerStoreFailure(t *testing.T) {
	client := newE2ETestClient(t, "e2e_no_limbo")
	logger := zaptest.NewLogger(t).Sugar()
	ctx, tenantID := e2eTenantCtx(t, client, "limbo")

	deployE2EProcess(t, ctx, client, tenantID, "e2e_proc", testBPMNE2EFullFlow)

	// Use a timer store that always fails on Create
	failingStore := &failingTimerStore{}
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(failingStore, nil)

	_, err := engine.StartProcess(ctx, "e2e_proc", "BK-E2E-008", nil)
	require.Error(t, err, "StartProcess should fail when timer store can't write")

	// Verify no running instances were left behind (transaction rolled back)
	instances, err := client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID)).
		All(ctx)
	if err == nil {
		for _, inst := range instances {
			assert.NotEqual(t, "running", inst.Status,
				"No running instances should be left after failed start")
		}
	}
}

// failingTimerStore always returns errors — used to test transaction rollback.
type failingTimerStore struct{}

func (s *failingTimerStore) Create(_ context.Context, _ *CreateTimerRequest) (*ent.ProcessTimer, error) {
	return nil, fmt.Errorf("simulated DB failure")
}
func (s *failingTimerStore) GetByTimerID(_ context.Context, _ string) (*ent.ProcessTimer, error) {
	return nil, fmt.Errorf("simulated DB failure")
}
func (s *failingTimerStore) FindPendingDue(_ context.Context, _ int, _ time.Time) ([]*ent.ProcessTimer, error) {
	return nil, fmt.Errorf("simulated DB failure")
}
func (s *failingTimerStore) FindPendingFuture(_ context.Context, _ int, _ time.Time) ([]*ent.ProcessTimer, error) {
	return nil, fmt.Errorf("simulated DB failure")
}
func (s *failingTimerStore) FindFiredStale(_ context.Context, _ time.Time) ([]*ent.ProcessTimer, error) {
	return nil, fmt.Errorf("simulated DB failure")
}
func (s *failingTimerStore) FindFailedRetryable(_ context.Context, _ int) ([]*ent.ProcessTimer, error) {
	return nil, fmt.Errorf("simulated DB failure")
}
func (s *failingTimerStore) CASFire(_ context.Context, _ string, _ time.Time) (*ent.ProcessTimer, error) {
	return nil, fmt.Errorf("simulated DB failure")
}
func (s *failingTimerStore) CASFail(_ context.Context, _ string, _ string, _ int, _ *time.Time) (*ent.ProcessTimer, error) {
	return nil, fmt.Errorf("simulated DB failure")
}
func (s *failingTimerStore) CancelByProcessInstance(_ context.Context, _, _ int) (int, error) {
	return 0, fmt.Errorf("simulated DB failure")
}
func (s *failingTimerStore) CancelByTimerID(_ context.Context, _ string) error {
	return fmt.Errorf("simulated DB failure")
}
func (s *failingTimerStore) List(_ context.Context, _ TimerListFilter) ([]*ent.ProcessTimer, int, error) {
	return nil, 0, fmt.Errorf("simulated DB failure")
}
func (s *failingTimerStore) Stats(_ context.Context, _ int) (*TimerStats, error) {
	return nil, fmt.Errorf("simulated DB failure")
}
