package service

import (
	"context"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
)

func newPhase4TestClient(t *testing.T, name string) *ent.Client {
	t.Helper()
	dsn := "file:" + name + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, dialect.SQLite, dsn)
	t.Cleanup(func() { client.Close() })
	return client
}

// ==================== Helper method tests ====================

func TestFindIntermediateEvent(t *testing.T) {
	engine := &CustomProcessEngine{}
	process := &BPMNProcess{
		IntermediateEvents: []*BPMNIntermediateEvent{
			{ID: "timer_wait", Name: "Wait 1 hour"},
			{ID: "signal_wait", Name: "Wait for signal"},
		},
	}

	event := engine.findIntermediateEvent(process, "timer_wait")
	require.NotNil(t, event)
	assert.Equal(t, "Wait 1 hour", event.Name)

	event = engine.findIntermediateEvent(process, "signal_wait")
	require.NotNil(t, event)
	assert.Equal(t, "Wait for signal", event.Name)

	event = engine.findIntermediateEvent(process, "nonexistent")
	assert.Nil(t, event)
}

func TestHasTimerExpression(t *testing.T) {
	engine := &CustomProcessEngine{}

	tests := []struct {
		name     string
		def      *BPMNTimerEventDefinition
		expected bool
	}{
		{"duration only", &BPMNTimerEventDefinition{TimeDuration: "PT1H"}, true},
		{"date only", &BPMNTimerEventDefinition{TimeDate: "2026-01-01T00:00:00Z"}, true},
		{"cycle only", &BPMNTimerEventDefinition{TimeCycle: "R3/PT10M"}, true},
		{"all empty", &BPMNTimerEventDefinition{}, false},
		{"whitespace only", &BPMNTimerEventDefinition{TimeDuration: "  "}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, engine.hasTimerExpression(tt.def))
		})
	}
}

func TestExtractTimerExpression(t *testing.T) {
	engine := &CustomProcessEngine{}

	tests := []struct {
		name         string
		def          *BPMNTimerEventDefinition
		expectedExpr string
		expectedType string
	}{
		{"duration priority", &BPMNTimerEventDefinition{TimeDuration: "PT30M", TimeDate: "2026-01-01T00:00:00Z"}, "PT30M", "duration"},
		{"date when no duration", &BPMNTimerEventDefinition{TimeDate: "2026-01-01T00:00:00Z", TimeCycle: "R3/PT10M"}, "2026-01-01T00:00:00Z", "date"},
		{"cycle fallback", &BPMNTimerEventDefinition{TimeCycle: "R5/PT1H"}, "R5/PT1H", "cycle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, exprType := engine.extractTimerExpression(tt.def)
			assert.Equal(t, tt.expectedExpr, expr)
			assert.Equal(t, tt.expectedType, exprType)
		})
	}
}

// ==================== Intermediate timer registration tests ====================

func TestHandleIntermediateCatchEvent_TimerRegistration(t *testing.T) {
	client := newPhase4TestClient(t, "p4-intermediate-reg")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "p4int")
	ctx := context.Background()

	timerStore := NewDBTimerStore(client, logger)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithIntermediateTimerEvent)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "timer_wait")

	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(testBPMNWithIntermediateTimerEvent))
	require.NoError(t, err)
	process := definitions.Processes[0]

	event := engine.findIntermediateEvent(process, "timer_wait")
	require.NotNil(t, event)

	err = engine.handleIntermediateCatchEvent(ctx, client, instance, process, event)
	require.NoError(t, err)

	timers, total, err := timerStore.List(ctx, TimerListFilter{
		TenantID:          tenantID,
		TimerType:         string(TimerTypeIntermediate),
		ProcessInstanceID: &instance.ID,
		Status:            string(TimerStatusPending),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, timers, 1)
	assert.Equal(t, "timer_wait", timers[0].ActivityID)
	assert.Equal(t, "PT1H", timers[0].TimerExpression)
	assert.Equal(t, string(TimerTypeIntermediate), timers[0].TimerType)
}

func TestHandleIntermediateCatchEvent_NoTimerStore(t *testing.T) {
	client := newPhase4TestClient(t, "p4-intermediate-no-store")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "p4ints")
	ctx := context.Background()

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithIntermediateTimerEvent)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "timer_wait")

	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(testBPMNWithIntermediateTimerEvent))
	require.NoError(t, err)
	process := definitions.Processes[0]

	event := engine.findIntermediateEvent(process, "timer_wait")
	require.NotNil(t, event)

	err = engine.handleIntermediateCatchEvent(ctx, client, instance, process, event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "定时器服务未配置")
}

func TestHandleIntermediateCatchEvent_NonTimerPassThrough(t *testing.T) {
	client := newPhase4TestClient(t, "p4-intermediate-passthrough")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "p4intp")
	ctx := context.Background()

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	nonTimerBPMN := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="test_proc" name="Test Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:intermediateCatchEvent id="signal_wait" name="Wait for signal"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="signal_wait"/>
    <bpmn:sequenceFlow id="flow2" sourceRef="signal_wait" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", nonTimerBPMN)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "signal_wait")

	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(nonTimerBPMN))
	require.NoError(t, err)
	process := definitions.Processes[0]

	event := engine.findIntermediateEvent(process, "signal_wait")
	require.NotNil(t, event)
	assert.Nil(t, event.TimerEventDefinition)

	err = engine.handleIntermediateCatchEvent(ctx, client, instance, process, event)
	require.NoError(t, err)
}

// ==================== Boundary timer registration tests ====================

func TestRegisterBoundaryTimers_OnUserTaskCreation(t *testing.T) {
	client := newPhase4TestClient(t, "p4-boundary-reg")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "p4bnd")
	ctx := context.Background()

	timerStore := NewDBTimerStore(client, logger)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithBoundaryTimerEvent)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "review_task")

	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(testBPMNWithBoundaryTimerEvent))
	require.NoError(t, err)
	process := definitions.Processes[0]

	engine.registerBoundaryTimers(ctx, instance, process, "review_task")

	timers, total, err := timerStore.List(ctx, TimerListFilter{
		TenantID:          tenantID,
		TimerType:         string(TimerTypeBoundary),
		ProcessInstanceID: &instance.ID,
		Status:            string(TimerStatusPending),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, timers, 1)
	assert.Equal(t, "timeout_boundary", timers[0].ActivityID)
	assert.Equal(t, "PT2H", timers[0].TimerExpression)
	assert.Equal(t, string(TimerTypeBoundary), timers[0].TimerType)
}

func TestRegisterBoundaryTimers_NoMatchingBoundary(t *testing.T) {
	client := newPhase4TestClient(t, "p4-boundary-nomatch")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "p4bndn")
	ctx := context.Background()

	timerStore := NewDBTimerStore(client, logger)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithBoundaryTimerEvent)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "review_task")

	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(testBPMNWithBoundaryTimerEvent))
	require.NoError(t, err)
	process := definitions.Processes[0]

	engine.registerBoundaryTimers(ctx, instance, process, "some_other_task")

	_, total, err := timerStore.List(ctx, TimerListFilter{
		TenantID:          tenantID,
		TimerType:         string(TimerTypeBoundary),
		ProcessInstanceID: &instance.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestRegisterBoundaryTimers_NilTimerStore(t *testing.T) {
	client := newPhase4TestClient(t, "p4-boundary-nil-store")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "p4bndi")
	ctx := context.Background()

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithBoundaryTimerEvent)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "review_task")

	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(testBPMNWithBoundaryTimerEvent))
	require.NoError(t, err)
	process := definitions.Processes[0]

	engine.registerBoundaryTimers(ctx, instance, process, "review_task")
}

// ==================== Boundary timer cancellation tests ====================

func TestCancelBoundaryTimers_OnTaskCompletion(t *testing.T) {
	client := newPhase4TestClient(t, "p4-boundary-cancel")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "p4bndc")
	ctx := context.Background()

	timerStore := NewDBTimerStore(client, logger)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithBoundaryTimerEvent)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "review_task")

	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(testBPMNWithBoundaryTimerEvent))
	require.NoError(t, err)
	process := definitions.Processes[0]

	engine.registerBoundaryTimers(ctx, instance, process, "review_task")

	timersBefore, totalBefore, err := timerStore.List(ctx, TimerListFilter{
		TenantID:          tenantID,
		TimerType:         string(TimerTypeBoundary),
		ProcessInstanceID: &instance.ID,
		Status:            string(TimerStatusPending),
	})
	require.NoError(t, err)
	require.Equal(t, 1, totalBefore)
	require.Len(t, timersBefore, 1)

	engine.cancelBoundaryTimers(ctx, instance, process, "review_task")

	timersAfter, totalAfter, err := timerStore.List(ctx, TimerListFilter{
		TenantID:          tenantID,
		TimerType:         string(TimerTypeBoundary),
		ProcessInstanceID: &instance.ID,
		Status:            string(TimerStatusPending),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, totalAfter)
	assert.Len(t, timersAfter, 0)

	cancelledTimers, totalCancelled, err := timerStore.List(ctx, TimerListFilter{
		TenantID:          tenantID,
		TimerType:         string(TimerTypeBoundary),
		ProcessInstanceID: &instance.ID,
		Status:            string(TimerStatusCancelled),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, totalCancelled)
	assert.Len(t, cancelledTimers, 1)
	assert.Equal(t, "timeout_boundary", cancelledTimers[0].ActivityID)
}

func TestCancelBoundaryTimers_NilTimerStore(t *testing.T) {
	client := newPhase4TestClient(t, "p4-boundary-cancel-nil")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "p4bndcn")
	ctx := context.Background()

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithBoundaryTimerEvent)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "review_task")

	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(testBPMNWithBoundaryTimerEvent))
	require.NoError(t, err)
	process := definitions.Processes[0]

	engine.cancelBoundaryTimers(ctx, instance, process, "review_task")
}

// ==================== SetTimerServices wiring test ====================

func TestSetTimerServices(t *testing.T) {
	client := newPhase4TestClient(t, "p4-set-timer-svc")
	logger := zaptest.NewLogger(t).Sugar()

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	assert.Nil(t, engine.timerStore)
	assert.Nil(t, engine.timerScheduler)

	timerStore := NewDBTimerStore(client, logger)
	engine.SetTimerServices(timerStore, nil)

	assert.NotNil(t, engine.timerStore)
	assert.Nil(t, engine.timerScheduler)
}

// ==================== Boundary timer fire-at calculation test ====================

func TestRegisterBoundaryTimers_FireAtCalculation(t *testing.T) {
	client := newPhase4TestClient(t, "p4-boundary-fireat")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "p4bndf")
	ctx := context.Background()

	timerStore := NewDBTimerStore(client, logger)
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	engine.SetTimerServices(timerStore, nil)

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithBoundaryTimerEvent)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "review_task")

	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(testBPMNWithBoundaryTimerEvent))
	require.NoError(t, err)
	process := definitions.Processes[0]

	before := time.Now()
	engine.registerBoundaryTimers(ctx, instance, process, "review_task")
	after := time.Now()

	timers, _, err := timerStore.List(ctx, TimerListFilter{
		TenantID:          tenantID,
		TimerType:         string(TimerTypeBoundary),
		ProcessInstanceID: &instance.ID,
	})
	require.NoError(t, err)
	require.Len(t, timers, 1)

	expectedMin := before.Add(2 * time.Hour)
	expectedMax := after.Add(2 * time.Hour)
	assert.True(t, timers[0].FireAt.After(expectedMin.Add(-time.Second)) || timers[0].FireAt.Equal(expectedMin),
		"FireAt should be approximately 2 hours from now")
	assert.True(t, timers[0].FireAt.Before(expectedMax.Add(time.Second)),
		"FireAt should be approximately 2 hours from now")
}
