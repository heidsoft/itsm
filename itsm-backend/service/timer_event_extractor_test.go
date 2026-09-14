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

func newTimerExtractorTestClient(t *testing.T, name string) *ent.Client {
	t.Helper()
	dsn := "file:" + name + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, dialect.SQLite, dsn)
	t.Cleanup(func() { client.Close() })
	return client
}

// BPMN XML with start timer event
const testBPMNWithStartTimer = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="test_proc" name="Test Process" isExecutable="true">
    <bpmn:startEvent id="start_timer" name="Start after 30 minutes">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT30M</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:startEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start_timer" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

// BPMN XML with intermediate timer event
const testBPMNWithIntermediateTimerEvent = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="test_proc" name="Test Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:intermediateCatchEvent id="timer_wait" name="Wait 1 hour">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT1H</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:intermediateCatchEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="timer_wait"/>
    <bpmn:sequenceFlow id="flow2" sourceRef="timer_wait" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

// BPMN XML with boundary timer event
const testBPMNWithBoundaryTimerEvent = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="test_proc" name="Test Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="review_task" name="Review Request"/>
    <bpmn:boundaryEvent id="timeout_boundary" attachedToRef="review_task" cancelActivity="true">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT2H</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:boundaryEvent>
    <bpmn:endEvent id="normal_end"/>
    <bpmn:endEvent id="timeout_end"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="review_task"/>
    <bpmn:sequenceFlow id="flow2" sourceRef="review_task" targetRef="normal_end"/>
    <bpmn:sequenceFlow id="flow3" sourceRef="timeout_boundary" targetRef="timeout_end"/>
  </bpmn:process>
</bpmn:definitions>`

// BPMN XML with multiple timer events
const testBPMNWithMultipleTimers = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="test_proc" name="Test Process" isExecutable="true">
    <bpmn:startEvent id="start_timer" name="Start after 10 minutes">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT10M</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:startEvent>
    <bpmn:intermediateCatchEvent id="wait_timer" name="Wait 30 minutes">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT30M</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:intermediateCatchEvent>
    <bpmn:userTask id="task" name="Task"/>
    <bpmn:boundaryEvent id="boundary_timer" attachedToRef="task" cancelActivity="true">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT1H</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:boundaryEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start_timer" targetRef="wait_timer"/>
    <bpmn:sequenceFlow id="flow2" sourceRef="wait_timer" targetRef="task"/>
    <bpmn:sequenceFlow id="flow3" sourceRef="task" targetRef="end"/>
    <bpmn:sequenceFlow id="flow4" sourceRef="boundary_timer" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

func TestExtractTimerEvents_StartTimer(t *testing.T) {
	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(testBPMNWithStartTimer))
	require.NoError(t, err)

	timers, err := ExtractTimerEvents(definitions, "test_proc")
	require.NoError(t, err)
	require.Len(t, timers, 1)

	assert.Equal(t, "start", timers[0].TimerType)
	assert.Equal(t, "test_proc", timers[0].ProcessDefinitionKey)
	assert.Equal(t, "start_timer", timers[0].ActivityID)
	assert.Equal(t, "PT30M", timers[0].Expression)
	assert.Equal(t, "duration", timers[0].ExpressionType)
	assert.Equal(t, "Start after 30 minutes", timers[0].EventName)
	assert.NotEmpty(t, timers[0].TimerID)
}

func TestExtractTimerEvents_IntermediateTimer(t *testing.T) {
	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(testBPMNWithIntermediateTimerEvent))
	require.NoError(t, err)

	timers, err := ExtractTimerEvents(definitions, "test_proc")
	require.NoError(t, err)
	require.Len(t, timers, 1)

	assert.Equal(t, "intermediate", timers[0].TimerType)
	assert.Equal(t, "timer_wait", timers[0].ActivityID)
	assert.Equal(t, "PT1H", timers[0].Expression)
	assert.Equal(t, "duration", timers[0].ExpressionType)
}

func TestExtractTimerEvents_BoundaryTimer(t *testing.T) {
	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(testBPMNWithBoundaryTimerEvent))
	require.NoError(t, err)

	timers, err := ExtractTimerEvents(definitions, "test_proc")
	require.NoError(t, err)
	require.Len(t, timers, 1)

	assert.Equal(t, "boundary", timers[0].TimerType)
	assert.Equal(t, "timeout_boundary", timers[0].ActivityID)
	assert.Equal(t, "review_task", timers[0].AttachedToRef)
	assert.True(t, timers[0].CancelActivity)
	assert.Equal(t, "PT2H", timers[0].Expression)
}

func TestExtractTimerEvents_MultipleTimers(t *testing.T) {
	parser := NewBPMNParser()
	definitions, err := parser.ParseXML([]byte(testBPMNWithMultipleTimers))
	require.NoError(t, err)

	timers, err := ExtractTimerEvents(definitions, "test_proc")
	require.NoError(t, err)
	require.Len(t, timers, 3)

	// Verify we have one of each type
	types := make(map[string]bool)
	for _, timer := range timers {
		types[timer.TimerType] = true
	}
	assert.True(t, types["start"])
	assert.True(t, types["intermediate"])
	assert.True(t, types["boundary"])
}

func TestExtractTimerEvents_NoTimers(t *testing.T) {
	parser := NewBPMNParser()
	bpmnXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="test_proc" name="Test Process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	definitions, err := parser.ParseXML([]byte(bpmnXML))
	require.NoError(t, err)

	timers, err := ExtractTimerEvents(definitions, "test_proc")
	require.NoError(t, err)
	assert.Len(t, timers, 0)
}

func TestExtractTimerEvents_NilDefinitions(t *testing.T) {
	timers, err := ExtractTimerEvents(nil, "test_proc")
	require.Error(t, err)
	assert.Nil(t, timers)
	assert.Contains(t, err.Error(), "nil")
}

func TestCalculateFireAt_Duration(t *testing.T) {
	now := time.Now()

	// Test PT30M
	fireAt, err := CalculateFireAt("PT30M", "duration", now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(30*time.Minute).Unix(), fireAt.Unix())

	// Test PT1H
	fireAt, err = CalculateFireAt("PT1H", "duration", now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(1*time.Hour).Unix(), fireAt.Unix())

	// Test PT2H
	fireAt, err = CalculateFireAt("PT2H", "duration", now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(2*time.Hour).Unix(), fireAt.Unix())
}

func TestCalculateFireAt_Date(t *testing.T) {
	futureDate := time.Now().Add(24 * time.Hour)
	dateStr := futureDate.Format(time.RFC3339)

	fireAt, err := CalculateFireAt(dateStr, "date", time.Now())
	require.NoError(t, err)
	assert.Equal(t, futureDate.Unix(), fireAt.Unix())
}

func TestCalculateFireAt_Cycle(t *testing.T) {
	now := time.Now()

	// Test R5/PT10M (repeat 5 times, every 10 minutes)
	fireAt, err := CalculateFireAt("R5/PT10M", "cycle", now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(10*time.Minute).Unix(), fireAt.Unix())
}

func TestCalculateFireAt_InvalidExpression(t *testing.T) {
	now := time.Now()

	// Invalid duration
	_, err := CalculateFireAt("INVALID", "duration", now)
	require.Error(t, err)

	// Invalid date
	_, err = CalculateFireAt("NOT-A-DATE", "date", now)
	require.Error(t, err)

	// Unknown type
	_, err = CalculateFireAt("PT30M", "unknown", now)
	require.Error(t, err)
}

func TestRegisterStartTimers_OnDeployment(t *testing.T) {
	client := newTimerExtractorTestClient(t, "register-start-timers")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "reg")
	ctx := context.Background()

	timerStore := NewDBTimerStore(client, logger)
	deploymentService := NewBPMNDeploymentServiceWithTimerStore(client, timerStore)

	// Deploy process with start timer
	deployment, err := deploymentService.DeployProcessDefinition(ctx, &DeployProcessDefinitionRequest{
		Name:     "Test Process",
		BPMNXML:  testBPMNWithStartTimer,
		TenantID: tenantID,
	})
	require.NoError(t, err)
	require.NotNil(t, deployment)

	// Verify timer was registered
	timers, total, err := timerStore.List(ctx, TimerListFilter{
		TenantID: tenantID,
		Status:   "pending",
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)

	// Verify it's a start timer
	foundStartTimer := false
	for _, timer := range timers {
		if timer.TimerType == string(TimerTypeStart) && timer.ProcessDefinitionKey == "test_proc" {
			foundStartTimer = true
			assert.Equal(t, "start_timer", timer.ActivityID)
			assert.Equal(t, "PT30M", timer.TimerExpression)
			break
		}
	}
	assert.True(t, foundStartTimer, "Start timer should be registered")
}

func TestRegisterStartTimers_NoTimerStore(t *testing.T) {
	client := newTimerExtractorTestClient(t, "no-timer-store")
	tenantID := createTimerTenant(t, client, "nts")
	ctx := context.Background()

	// Create deployment service without timer store
	deploymentService := NewBPMNDeploymentService(client)

	// Deploy process with start timer - should succeed even without timer store
	deployment, err := deploymentService.DeployProcessDefinition(ctx, &DeployProcessDefinitionRequest{
		Name:     "Test Process",
		BPMNXML:  testBPMNWithStartTimer,
		TenantID: tenantID,
	})
	require.NoError(t, err)
	require.NotNil(t, deployment)
}

func TestRegisterStartTimers_OnlyStartTimers(t *testing.T) {
	client := newTimerExtractorTestClient(t, "only-start-timers")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "ost")
	ctx := context.Background()

	timerStore := NewDBTimerStore(client, logger)
	deploymentService := NewBPMNDeploymentServiceWithTimerStore(client, timerStore)

	// Deploy process with multiple timer types
	deployment, err := deploymentService.DeployProcessDefinition(ctx, &DeployProcessDefinitionRequest{
		Name:     "Test Process",
		BPMNXML:  testBPMNWithMultipleTimers,
		TenantID: tenantID,
	})
	require.NoError(t, err)
	require.NotNil(t, deployment)

	// Verify only start timers were registered
	timers, _, err := timerStore.List(ctx, TimerListFilter{
		TenantID: tenantID,
	})
	require.NoError(t, err)

	// Should only have 1 timer (the start timer), not 3
	assert.Equal(t, 1, len(timers), "Only start timers should be registered on deployment")
	assert.Equal(t, string(TimerTypeStart), timers[0].TimerType)
}
