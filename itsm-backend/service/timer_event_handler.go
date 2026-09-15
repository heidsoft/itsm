package service

import (
	"context"
	"fmt"
	"slices"
	"time"

	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"
)

// TimerEventHandler bridges the Timer Scheduler to the BPMN process engine.
// It implements the TimerFireCallback and advances process instances when timer events fire.
type TimerEventHandler struct {
	engine *CustomProcessEngine
	store  TimerStore
	logger *zap.SugaredLogger
}

// NewTimerEventHandler creates a new timer event handler.
func NewTimerEventHandler(engine *CustomProcessEngine, logger *zap.SugaredLogger) *TimerEventHandler {
	return &TimerEventHandler{
		engine: engine,
		logger: logger,
	}
}

// SetTimerStore 注入 TimerStore，供 start timer 触发后重排下一跳（Phase 5）。
func (h *TimerEventHandler) SetTimerStore(store TimerStore) {
	h.store = store
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
	case string(TimerTypeTaskDue):
		return h.handleTaskDueTimer(ctx, timer)
	case string(TimerTypeStart):
		return h.handleStartTimer(ctx, timer)
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

// handleTaskDueTimer 处理任务截止定时器（Phase 4）：
// 重新加载任务最新状态，仍活跃才分发超时动作（notify/escalate/auto_reject/auto_approve，
// 由 TimeoutScanner.dispatchTimeoutAction 承载，内部有 claim-once 保护）。
// 任务已完成/取消/已超时 → 静默确认（timer 正常结束，不重试）。
func (h *TimerEventHandler) handleTaskDueTimer(ctx context.Context, timer *TimerRecord) error {
	if timer.ActivityID == "" {
		return fmt.Errorf("task_due timer requires activity_id (task ID)")
	}

	task, err := h.engine.client.ProcessTask.Query().
		Where(
			processtask.TaskID(timer.ActivityID),
			processtask.TenantID(timer.TenantID),
		).
		First(ctx)
	if err != nil {
		return fmt.Errorf("task_due timer: load task %s: %w", timer.ActivityID, err)
	}

	if !slices.Contains(timeoutActiveStatuses, task.Status) {
		h.logger.Infow("task_due timer skipped: task no longer active",
			"task_id", task.TaskID, "status", task.Status)
		return nil
	}

	scanner := NewTimeoutScanner(h.engine.client, h.logger)
	if err := scanner.dispatchTimeoutAction(ctx, task, timer.TenantID); err != nil {
		return fmt.Errorf("task_due timer: dispatch timeout action for %s: %w", task.TaskID, err)
	}
	return nil
}

// handleStartTimer 处理定时启动流程（Phase 5）：
// Timer Start Event 到期后启动一个新的流程实例；cron / cycle 表达式在启动成功后
// 重排下一跳，形成常驻时间表。
//
// 幂等性：businessKey 由 timer_id + 计划触发时刻确定，启动前先查同名实例，
// 命中即视为"本次触发已完成"（重试场景下不会重复启动流程）。
func (h *TimerEventHandler) handleStartTimer(ctx context.Context, timer *TimerRecord) error {
	if timer.ProcessDefinitionKey == "" {
		return fmt.Errorf("start timer requires process_definition_key")
	}

	businessKey := fmt.Sprintf("timer:%s:%d", timer.TimerID, timer.FireAt.Unix())

	// 幂等闸门：同一 timer 的同一次计划触发只允许启动一个实例。
	existing, err := h.engine.client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.ProcessDefinitionKey(timer.ProcessDefinitionKey),
			processinstance.TenantID(timer.TenantID),
		).
		First(ctx)
	if err == nil {
		h.logger.Infow("start timer skipped: process instance already launched (idempotent)",
			"timer_id", timer.TimerID,
			"business_key", businessKey,
			"instance_id", existing.ID,
		)
		// 启动已完成，仍需保证重排（可能是上次重排失败后的重试）。
		return h.rearmStartTimer(ctx, timer)
	}
	if !ent.IsNotFound(err) {
		return fmt.Errorf("start timer: query existing instance: %w", err)
	}

	// StartProcess 强制要求租户上下文（fail-closed，P1-4）
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, timer.TenantID)

	variables := map[string]interface{}{
		"triggered_by":   "timer",
		"timer_id":       timer.TimerID,
		"timer_fired_at": timer.FireAt.UTC().Format(time.RFC3339),
	}
	for k, v := range timer.ContextVariables {
		// 重排元数据（remaining_repetitions 等内部键）不注入业务变量
		if k == "remaining_repetitions" {
			continue
		}
		variables[k] = v
	}

	instance, err := h.engine.StartProcess(workflowCtx, timer.ProcessDefinitionKey, businessKey, variables)
	if err != nil {
		return fmt.Errorf("start timer: start process %s: %w", timer.ProcessDefinitionKey, err)
	}

	h.logger.Infow("start timer launched process instance",
		"timer_id", timer.TimerID,
		"process_key", timer.ProcessDefinitionKey,
		"business_key", businessKey,
		"instance_id", instance.ID,
		"tenant_id", timer.TenantID,
	)

	return h.rearmStartTimer(ctx, timer)
}

// rearmStartTimer 为重复型表达式（cron / cycle）创建下一跳 timer。
// 一次性表达式（duration / date）返回 nil，不做任何事。
func (h *TimerEventHandler) rearmStartTimer(ctx context.Context, timer *TimerRecord) error {
	if timer.TimerExpression == "" {
		return nil // 无表达式元数据（历史记录）→ 按一次性处理
	}

	exprType := ExpressionType(timer.ExpressionType)
	if exprType == "" {
		detected, err := ParseTimerExpression(timer.TimerExpression)
		if err != nil {
			h.logger.Warnw("start timer rearm skipped: unparsable expression",
				"error", err, "timer_id", timer.TimerID, "expression", timer.TimerExpression)
			return nil
		}
		exprType = detected
	}

	if !IsRecurring(timer.TimerExpression, exprType) {
		return nil
	}

	if h.store == nil {
		h.logger.Warnw("start timer is recurring but no TimerStore wired; next occurrence not scheduled",
			"timer_id", timer.TimerID)
		return nil
	}

	// 重排幂等闸门：崩溃恢复/重放会以同一 timer 记录再次进入本函数。
	// 若时间表中已存在同 (tenant, process_key, activity) 的 pending start timer，
	// 说明上一轮重排已成功——再建一条会让 cron 变成双份时间表，之后每次触发启动两个实例。
	if h.hasPendingStartTimer(ctx, timer) {
		h.logger.Infow("start timer rearm skipped: next occurrence already scheduled",
			"timer_id", timer.TimerID,
			"process_key", timer.ProcessDefinitionKey,
			"activity_id", timer.ActivityID,
		)
		return nil
	}

	// cycle 有限次数：递减剩余；本次已是最后一次则不再重排。
	// 注意：仅 cycle 走次数语义。cron 天然无限重复，若也套用 CycleRemaining，
	// 无 '/' 的 cron 字符串会被判成"一次性"（bounded=true, remaining=1），
	// 导致 cron 触发一次后时间表被永久终止——这是必须避免的语义混淆。
	contextVars := map[string]interface{}{}
	if exprType == ExprTypeCycle {
		if remaining, bounded, ok := CycleRemaining(timer.TimerExpression); ok && bounded {
			current := remaining
			if v, exists := timer.ContextVariables["remaining_repetitions"]; exists {
				if n, ok := toInt(v); ok {
					current = n
				}
			}
			if current <= 1 {
				h.logger.Infow("start timer repetitions exhausted",
					"timer_id", timer.TimerID, "expression", timer.TimerExpression)
				return nil
			}
			contextVars["remaining_repetitions"] = current - 1
		}
	}

	loc := LoadTenantLocation(ctx, h.engine.client, timer.TenantID)
	nextFireAt, err := NextFireAt(timer.TimerExpression, exprType, loc, time.Now())
	if err != nil {
		return fmt.Errorf("start timer rearm: compute next fire time: %w", err)
	}

	created, err := h.store.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeStart,
		ProcessDefinitionKey: timer.ProcessDefinitionKey,
		ActivityID:           timer.ActivityID,
		TimerExpression:      timer.TimerExpression,
		ExpressionType:       exprType,
		FireAt:               nextFireAt,
		ContextVariables:     contextVars,
		TenantID:             timer.TenantID,
	})
	if err != nil {
		return fmt.Errorf("start timer rearm: create next timer: %w", err)
	}

	// 立即接驳到内存调度器（否则要等下一次周期性同步才知道它的存在）。
	if h.engine.timerScheduler != nil {
		if err := h.engine.timerScheduler.Schedule(ctx, entTimerToRecord(created)); err != nil {
			// 调度失败不影响正确性：周期性同步/重启恢复会兜底接驳。
			h.logger.Warnw("start timer rearm: failed to schedule next occurrence immediately",
				"error", err, "timer_id", created.TimerID)
		}
	}

	h.logger.Infow("start timer rearmed",
		"previous_timer_id", timer.TimerID,
		"next_timer_id", created.TimerID,
		"next_fire_at", nextFireAt,
		"expression_type", exprType,
	)
	return nil
}

// hasPendingStartTimer 报告时间表中是否已存在同 (tenant, process_key, activity) 的
// pending start timer。用于把重排做成幂等操作。
//
// 查询失败按"不存在"处理并告警：宁可多排一次（下次触发会被实例幂等闸门拦下），
// 也不能因为管理面查询抖动而永久停止 cron 时间表。
func (h *TimerEventHandler) hasPendingStartTimer(ctx context.Context, timer *TimerRecord) bool {
	if h.store == nil {
		return false
	}
	timers, _, err := h.store.List(ctx, TimerListFilter{
		TenantID:             timer.TenantID,
		Status:               string(TimerStatusPending),
		TimerType:            string(TimerTypeStart),
		ProcessDefinitionKey: timer.ProcessDefinitionKey,
	})
	if err != nil {
		h.logger.Warnw("start timer rearm: query existing schedule failed; proceeding as absent",
			"error", err, "timer_id", timer.TimerID)
		return false
	}
	for _, t := range timers {
		if t.ActivityID == timer.ActivityID {
			return true
		}
	}
	return false
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
