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

func newTimerEventHandlerTestClient(t *testing.T, name string) *ent.Client {
	t.Helper()
	dsn := "file:" + name + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, dialect.SQLite, dsn)
	t.Cleanup(func() { client.Close() })
	return client
}

func createTestProcessDefinition(t *testing.T, client *ent.Client, tenantID int, key string, bpmnXML string) int {
	t.Helper()
	// Create a deployment first (required by schema)
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("deploy-" + key).
		SetDeploymentName("Test Deployment " + key).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey(key).
		SetName("Test Process " + key).
		SetVersion("1.0").
		SetBpmnXML([]byte(bpmnXML)).
		SetIsActive(true).
		SetIsLatest(true).
		SetDeploymentID(deployment.ID).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return def.ID
}

func createTestProcessInstance(t *testing.T, client *ent.Client, tenantID int, processDefID int, activityID string) *ent.ProcessInstance {
	t.Helper()
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("proc-inst-" + activityID).
		SetProcessDefinitionKey("test_proc").
		SetProcessDefinitionID(processDefID).
		SetStatus("running").
		SetCurrentActivityID(activityID).
		SetTenantID(tenantID).
		SetStartTime(time.Now()).
		Save(context.Background())
	require.NoError(t, err)
	return instance
}

// Simple BPMN XML with an intermediate timer catch event
const testBPMNWithIntermediateTimer = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="test_proc" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:intermediateCatchEvent id="timer_wait" name="Wait 30 minutes">
      <bpmn:timerEventDefinition>
        <bpmn:timeDuration>PT30M</bpmn:timeDuration>
      </bpmn:timerEventDefinition>
    </bpmn:intermediateCatchEvent>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="flow1" sourceRef="start" targetRef="timer_wait"/>
    <bpmn:sequenceFlow id="flow2" sourceRef="timer_wait" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

// BPMN XML with a boundary timer event on a user task
const testBPMNWithBoundaryTimer = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="test_proc" isExecutable="true">
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

func TestTimerEventHandler_HandleIntermediateTimer(t *testing.T) {
	client := newTimerEventHandlerTestClient(t, "intermediate-timer")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "int")
	ctx := context.Background()

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithIntermediateTimer)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "timer_wait")

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	handler := NewTimerEventHandler(engine, logger)

	timer := &TimerRecord{
		TimerID:              "timer-001",
		TimerType:            "intermediate",
		ProcessDefinitionKey: "test_proc",
		ProcessInstanceID:    instance.ID,
		ActivityID:           "timer_wait",
		FireAt:               time.Now(),
		TenantID:             tenantID,
	}

	err := handler.HandleTimerFire(ctx, timer)
	require.NoError(t, err)

	// Verify the process instance advanced (should have moved to end event or completed)
	updated, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	// The process should have moved past the timer_wait node
	// (exact status depends on whether end event was reached)
	assert.NotEqual(t, "timer_wait", updated.CurrentActivityID, "Process should have advanced past timer_wait")
}

func TestTimerEventHandler_HandleBoundaryTimer(t *testing.T) {
	client := newTimerEventHandlerTestClient(t, "boundary-timer")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "bound")
	ctx := context.Background()

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithBoundaryTimer)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "review_task")

	// Create a pending task for the user task
	_, err := client.ProcessTask.Create().
		SetTaskID("task-001").
		SetProcessInstanceID(instance.ID).
		SetProcessDefinitionKey("test_proc").
		SetTaskDefinitionKey("review_task").
		SetTaskName("Review Request").
		SetTaskType("userTask").
		SetStatus("created").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	handler := NewTimerEventHandler(engine, logger)

	timer := &TimerRecord{
		TimerID:              "timer-002",
		TimerType:            "boundary",
		ProcessDefinitionKey: "test_proc",
		ProcessInstanceID:    instance.ID,
		ActivityID:           "timeout_boundary",
		FireAt:               time.Now(),
		TenantID:             tenantID,
	}

	err = handler.HandleTimerFire(ctx, timer)
	require.NoError(t, err)

	// Verify the task was cancelled
	task, err := client.ProcessTask.Query().Where().First(ctx)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", task.Status, "Task should be cancelled by boundary timer")

	// Verify the process instance advanced to the timeout path
	updated, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.NotEqual(t, "review_task", updated.CurrentActivityID, "Process should have moved off the interrupted task")
}

func TestTimerEventHandler_TenantIsolation(t *testing.T) {
	client := newTimerEventHandlerTestClient(t, "tenant-isolation")
	logger := zaptest.NewLogger(t).Sugar()
	tenantA := createTimerTenant(t, client, "A")
	tenantB := createTimerTenant(t, client, "B")
	ctx := context.Background()

	processDefID := createTestProcessDefinition(t, client, tenantA, "test_proc", testBPMNWithIntermediateTimer)
	instance := createTestProcessInstance(t, client, tenantA, processDefID, "timer_wait")

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	handler := NewTimerEventHandler(engine, logger)

	// Try to fire timer with wrong tenant ID
	timer := &TimerRecord{
		TimerID:              "timer-003",
		TimerType:            "intermediate",
		ProcessDefinitionKey: "test_proc",
		ProcessInstanceID:    instance.ID,
		ActivityID:           "timer_wait",
		FireAt:               time.Now(),
		TenantID:             tenantB, // Wrong tenant!
	}

	err := handler.HandleTimerFire(ctx, timer)
	require.Error(t, err, "Should fail when tenant ID doesn't match")
	assert.Contains(t, err.Error(), "not found or not running", "Error should indicate instance not found")
}

func TestTimerEventHandler_MissingProcessInstance(t *testing.T) {
	client := newTimerEventHandlerTestClient(t, "missing-instance")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "miss")
	ctx := context.Background()

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	handler := NewTimerEventHandler(engine, logger)

	timer := &TimerRecord{
		TimerID:              "timer-004",
		TimerType:            "intermediate",
		ProcessDefinitionKey: "test_proc",
		ProcessInstanceID:    99999, // Non-existent
		ActivityID:           "timer_wait",
		FireAt:               time.Now(),
		TenantID:             tenantID,
	}

	err := handler.HandleTimerFire(ctx, timer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found or not running")
}

func TestTimerEventHandler_InvalidTimerType(t *testing.T) {
	client := newTimerEventHandlerTestClient(t, "invalid-type")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "inv")
	ctx := context.Background()

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	handler := NewTimerEventHandler(engine, logger)

	timer := &TimerRecord{
		TimerID:              "timer-005",
		TimerType:            "unknown_type",
		ProcessDefinitionKey: "test_proc",
		ProcessInstanceID:    1,
		ActivityID:           "some_activity",
		FireAt:               time.Now(),
		TenantID:             tenantID,
	}

	err := handler.HandleTimerFire(ctx, timer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown timer type")
}

func TestTimerEventHandler_StartTimerMissingProcessDefinition(t *testing.T) {
	client := newTimerEventHandlerTestClient(t, "start-timer")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "start")
	ctx := context.Background()

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	handler := NewTimerEventHandler(engine, logger)

	timer := &TimerRecord{
		TimerID:              "timer-006",
		TimerType:            "start",
		ProcessDefinitionKey: "test_proc",
		ProcessInstanceID:    0,
		ActivityID:           "start_event",
		FireAt:               time.Now(),
		TenantID:             tenantID,
	}

	// Phase 5 起 start timer 已实现：目标是启动流程实例；
	// 流程定义不存在时必须报错（供调度器重试/告警），而不是静默成功。
	err := handler.HandleTimerFire(ctx, timer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start timer")
	assert.Contains(t, err.Error(), "test_proc")
}

func TestTimerEventHandler_BoundaryEventNotFound(t *testing.T) {
	client := newTimerEventHandlerTestClient(t, "boundary-not-found")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "bnf")
	ctx := context.Background()

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithBoundaryTimer)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "review_task")

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	handler := NewTimerEventHandler(engine, logger)

	timer := &TimerRecord{
		TimerID:              "timer-007",
		TimerType:            "boundary",
		ProcessDefinitionKey: "test_proc",
		ProcessInstanceID:    instance.ID,
		ActivityID:           "nonexistent_boundary", // Doesn't exist in BPMN
		FireAt:               time.Now(),
		TenantID:             tenantID,
	}

	err := handler.HandleTimerFire(ctx, timer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in process definition")
}

func TestTimerEventHandler_AuditLogging(t *testing.T) {
	client := newTimerEventHandlerTestClient(t, "audit-logging")
	logger := zaptest.NewLogger(t).Sugar()
	tenantID := createTimerTenant(t, client, "audit")
	ctx := context.Background()

	processDefID := createTestProcessDefinition(t, client, tenantID, "test_proc", testBPMNWithIntermediateTimer)
	instance := createTestProcessInstance(t, client, tenantID, processDefID, "timer_wait")

	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	handler := NewTimerEventHandler(engine, logger)

	timer := &TimerRecord{
		TimerID:              "timer-008",
		TimerType:            "intermediate",
		ProcessDefinitionKey: "test_proc",
		ProcessInstanceID:    instance.ID,
		ActivityID:           "timer_wait",
		FireAt:               time.Now(),
		TenantID:             tenantID,
	}

	err := handler.HandleTimerFire(ctx, timer)
	require.NoError(t, err)

	// Verify audit log was created
	auditCount, err := client.ProcessAuditLog.Query().
		Where().
		Count(ctx)
	require.NoError(t, err)
	assert.Greater(t, auditCount, 0, "Audit log should be created for timer firing")
}
