package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
)

// TimerEventHandler bridges the Timer Scheduler to the BPMN process engine.
// It implements the TimerFireCallback and advances process instances when timer events fire.
type TimerEventHandler struct {
	engine *CustomProcessEngine
	logger *zap.SugaredLogger
}

// NewTimerEventHandler creates a new timer event handler.
func NewTimerEventHandler(engine *CustomProcessEngine, logger *zap.SugaredLogger) *TimerEventHandler {
	return &TimerEventHandler{
		engine: engine,
		logger: logger,
	}
}

// Callback returns the TimerFireCallback function for registration with the scheduler.
func (h *TimerEventHandler) Callback() TimerFireCallback {
	return h.HandleTimerFire
}

// HandleTimerFire is invoked by the Timer Scheduler when a timer fires.
func (h *TimerEventHandler) HandleTimerFire(ctx context.Context, timer *TimerRecord) error {
	if timer == nil {
		return fmt.Errorf("timer record is nil")
	}

	h.logger.Infow("Timer event fired",
		"timer_id", timer.TimerID,
		"timer_type", timer.TimerType,
		"process_instance_id", timer.ProcessInstanceID,
		"activity_id", timer.ActivityID,
		"tenant_id", timer.TenantID,
		"scheduled_fire_at", timer.FireAt,
		"actual_fire_at", time.Now(),
	)

	switch timer.TimerType {
	case "intermediate":
		return h.handleIntermediateTimer(ctx, timer)
	case "boundary":
		return h.handleBoundaryTimer(ctx, timer)
	case "start":
		return fmt.Errorf("start timer not yet implemented (deferred to Phase 5)")
	default:
		return fmt.Errorf("unknown timer type: %s", timer.TimerType)
	}
}

// handleIntermediateTimer advances the process from an intermediate timer catch event.
func (h *TimerEventHandler) handleIntermediateTimer(ctx context.Context, timer *TimerRecord) error {
	if timer.ProcessInstanceID <= 0 {
		return fmt.Errorf("intermediate timer requires process_instance_id")
	}
	if timer.ActivityID == "" {
		return fmt.Errorf("intermediate timer requires activity_id")
	}

	instance, process, err := h.loadRunningInstance(ctx, timer)
	if err != nil {
		return err
	}

	if instance.CurrentActivityID != timer.ActivityID {
		h.logger.Warnw("Process current activity does not match timer activity_id",
			"instance_id", instance.ID,
			"current_activity", instance.CurrentActivityID,
			"timer_activity", timer.ActivityID,
		)
	}

	h.logger.Infow("Advancing process from intermediate timer event",
		"instance_id", instance.ID,
		"activity_id", timer.ActivityID,
	)

	h.recordAudit(ctx, instance, timer.ActivityID, "intermediateCatchEvent", "timer_fired",
		fmt.Sprintf("Intermediate timer %s fired, advancing process", timer.TimerID),
		timer)

	return h.engine.executeStep(ctx, h.engine.client, instance, process, timer.ActivityID, instance.Variables)
}

// handleBoundaryTimer interrupts the attached activity and follows the exception path.
func (h *TimerEventHandler) handleBoundaryTimer(ctx context.Context, timer *TimerRecord) error {
	if timer.ProcessInstanceID <= 0 {
		return fmt.Errorf("boundary timer requires process_instance_id")
	}
	if timer.ActivityID == "" {
		return fmt.Errorf("boundary timer requires activity_id (boundary event ID)")
	}

	instance, process, err := h.loadRunningInstance(ctx, timer)
	if err != nil {
		return err
	}

	boundaryEvent := h.findBoundaryEvent(process, timer.ActivityID)
	if boundaryEvent == nil {
		return fmt.Errorf("boundary event %s not found in process definition", timer.ActivityID)
	}

	h.logger.Infow("Boundary timer fired, interrupting activity",
		"instance_id", instance.ID,
		"boundary_event_id", boundaryEvent.ID,
		"attached_to", boundaryEvent.AttachedToRef,
		"cancel_activity", boundaryEvent.CancelActivity,
	)

	if boundaryEvent.CancelActivity {
		if err := h.interruptActivity(ctx, instance, boundaryEvent.AttachedToRef); err != nil {
			return fmt.Errorf("failed to interrupt activity %s: %w", boundaryEvent.AttachedToRef, err)
		}
	}

	h.recordAudit(ctx, instance, boundaryEvent.ID, "boundaryEvent", "timer_fired",
		fmt.Sprintf("Boundary timer %s fired on activity %s", timer.TimerID, boundaryEvent.AttachedToRef),
		timer)

	return h.engine.executeStep(ctx, h.engine.client, instance, process, boundaryEvent.ID, instance.Variables)
}

// loadRunningInstance loads the process instance and parses its BPMN definition.
func (h *TimerEventHandler) loadRunningInstance(ctx context.Context, timer *TimerRecord) (*ent.ProcessInstance, *BPMNProcess, error) {
	instance, err := h.engine.client.ProcessInstance.Query().
		Where(
			processinstance.ID(timer.ProcessInstanceID),
			processinstance.TenantID(timer.TenantID),
			processinstance.Status("running"),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil, fmt.Errorf("process instance %d not found or not running", timer.ProcessInstanceID)
		}
		return nil, nil, fmt.Errorf("failed to load process instance: %w", err)
	}

	process, err := h.loadAndParseProcess(ctx, instance.ProcessDefinitionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load process definition: %w", err)
	}

	return instance, process, nil
}

// loadAndParseProcess loads a process definition and parses its BPMN XML.
func (h *TimerEventHandler) loadAndParseProcess(ctx context.Context, processDefID int) (*BPMNProcess, error) {
	processDef, err := h.engine.client.ProcessDefinition.Get(ctx, processDefID)
	if err != nil {
		return nil, fmt.Errorf("failed to load process definition %d: %w", processDefID, err)
	}

	parser := NewBPMNParser()
	definitions, err := parser.ParseXML(processDef.BpmnXML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse BPMN XML: %w", err)
	}

	if len(definitions.Processes) == 0 {
		return nil, fmt.Errorf("BPMN definitions contain no processes")
	}

	return definitions.Processes[0], nil
}

// findBoundaryEvent finds a boundary event by ID in the process.
func (h *TimerEventHandler) findBoundaryEvent(process *BPMNProcess, eventID string) *BPMNBoundaryEvent {
	for _, event := range process.BoundaryEvents {
		if event.ID == eventID {
			return event
		}
	}
	return nil
}

// interruptActivity cancels pending tasks for the attached activity and records the interruption.
func (h *TimerEventHandler) interruptActivity(ctx context.Context, instance *ent.ProcessInstance, activityID string) error {
	h.logger.Infow("Interrupting activity",
		"instance_id", instance.ID,
		"activity_id", activityID,
	)

	_, err := h.engine.client.ProcessTask.Update().
		Where(
			processtask.ProcessInstanceID(instance.ID),
			processtask.TaskDefinitionKey(activityID),
			processtask.StatusIn("created", "waiting", "in_progress"),
		).
		SetStatus("cancelled").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to cancel tasks for activity %s: %w", activityID, err)
	}

	_, err = h.engine.client.ProcessExecutionHistory.Create().
		SetHistoryID(fmt.Sprintf("hist-timer-interrupt-%d-%s-%d", instance.ID, activityID, time.Now().UnixNano())).
		SetProcessInstanceID(instance.ID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).
		SetActivityID(activityID).
		SetActivityType("userTask").
		SetEventType("cancel").
		SetEventDetail(fmt.Sprintf("Activity %s interrupted by boundary timer", activityID)).
		SetTenantID(instance.TenantID).
		Save(ctx)
	if err != nil {
		h.logger.Warnw("Failed to log activity interruption to execution history",
			"error", err,
			"instance_id", instance.ID,
			"activity_id", activityID,
		)
	}

	return nil
}

// recordAudit writes a process audit log for the timer firing.
func (h *TimerEventHandler) recordAudit(ctx context.Context, instance *ent.ProcessInstance, activityID, activityType, action, comment string, timer *TimerRecord) {
	_, err := h.engine.client.ProcessAuditLog.Create().
		SetProcessInstanceID(instance.ID).
		SetProcessInstanceKey(instance.ProcessInstanceID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).
		SetProcessDefinitionID(instance.ProcessDefinitionID).
		SetActivityID(activityID).
		SetActivityType(activityType).
		SetAction(action).
		SetComment(comment).
		SetTenantID(instance.TenantID).
		SetMetadata(map[string]interface{}{
			"timer_id":          timer.TimerID,
			"timer_type":        timer.TimerType,
			"scheduled_fire_at": timer.FireAt.Format(time.RFC3339),
			"actual_fire_at":    time.Now().Format(time.RFC3339),
		}).
		Save(ctx)
	if err != nil {
		h.logger.Warnw("Failed to record timer fire audit log",
			"error", err,
			"timer_id", timer.TimerID,
		)
	}
}
