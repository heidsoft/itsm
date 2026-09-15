package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
	"itsm-backend/metrics"
)

func (e *CustomProcessEngine) SuspendProcess(ctx context.Context, processInstanceID string, reason string) error {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}

	instance, err := e.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(processInstanceID), processinstance.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取流程实例失败: %w", err)
	}

	_, err = e.client.ProcessInstance.UpdateOne(instance).
		SetStatus("suspended").
		SetSuspendedTime(time.Now()).
		SetSuspendedReason(reason).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("暂停流程实例失败: %w", err)
	}

	// 挂起后取消该实例的活跃定时器：timer 回调只认 running 实例
	//（loadRunningInstance 过滤 status=running），不取消会触发→失败→重试风暴。
	cancelled := e.cancelInstanceTimers(ctx, instance)
	if cancelled > 0 {
		metrics.TimerPausedTotal.WithLabelValues(strconv.Itoa(instance.TenantID)).Add(float64(cancelled))
		e.logger.Infow("cancelled active timers on process suspension",
			"instance_id", instance.ID, "cancelled", cancelled)
	}

	userID, userName := extractUserFromContext(ctx)
	if err := e.auditService.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           instance.CurrentActivityID,
		ActivityName:         instance.CurrentActivityName,
		ActivityType:         ActivityTypeUserTask,
		Action:               AuditActionProcessSuspended,
		UserID:               userID,
		UserName:             userName,
		Comment:              reason,
		TenantID:             instance.TenantID,
	}); err != nil {
		e.logger.Warnw("audit record failed", "error", err)
	}

	return nil
}

func (e *CustomProcessEngine) ResumeProcess(ctx context.Context, processInstanceID string) error {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}

	instance, err := e.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(processInstanceID), processinstance.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取流程实例失败: %w", err)
	}

	_, err = e.client.ProcessInstance.UpdateOne(instance).
		SetStatus("running").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("恢复流程实例失败: %w", err)
	}

	// 恢复后重建当前活动的定时器（挂起时已被取消）：
	// 中间定时捕获事件→重建 intermediate timer；用户任务→重建其 boundary timers。
	// best-effort：重建失败仅告警，不阻断恢复（recovery 扫描会在重启后兕底）。
	if err := e.reregisterInstanceTimers(ctx, instance); err != nil {
		e.logger.Warnw("failed to re-register timers on process resume",
			"error", err, "instance_id", instance.ID,
			"current_activity", instance.CurrentActivityID)
	}

	userID, userName := extractUserFromContext(ctx)
	if err := e.auditService.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           instance.CurrentActivityID,
		ActivityName:         instance.CurrentActivityName,
		ActivityType:         ActivityTypeUserTask,
		Action:               AuditActionProcessResumed,
		UserID:               userID,
		UserName:             userName,
		TenantID:             instance.TenantID,
	}); err != nil {
		e.logger.Warnw("audit record failed", "error", err)
	}

	return nil
}

func (e *CustomProcessEngine) TerminateProcess(ctx context.Context, processInstanceID string, reason string) error {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}

	instance, err := e.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(processInstanceID), processinstance.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取流程实例失败: %w", err)
	}

	_, err = e.client.ProcessInstance.UpdateOne(instance).
		SetStatus("terminated").
		SetEndTime(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("终止流程实例失败: %w", err)
	}

	_, err = e.client.ProcessTask.Update().
		Where(processtask.ProcessInstanceID(instance.ID)).
		Where(processtask.StatusNEQ("completed")).
		Where(processtask.StatusNEQ("cancelled")).
		SetStatus("cancelled").
		SetCompletedTime(time.Now()).
		Save(ctx)
	if err != nil {
		e.logger.Warnw("取消流程任务失败", "error", err)
	}

	// 终止后取消该实例的活跃定时器（同挂起：终止实例不应再有 timer 触发）。
	e.cancelInstanceTimers(ctx, instance)

	userID, userName := extractUserFromContext(ctx)
	if err := e.auditService.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           instance.CurrentActivityID,
		ActivityName:         instance.CurrentActivityName,
		ActivityType:         ActivityTypeEndEvent,
		Action:               AuditActionProcessTerminated,
		UserID:               userID,
		UserName:             userName,
		Comment:              reason,
		TenantID:             instance.TenantID,
	}); err != nil {
		e.logger.Warnw("audit record failed", "error", err)
	}

	return nil
}

// cancelInstanceTimers 取消实例的全部 pending 定时器（DB + 内存调度器），
// 返回成功取消的数量。store/scheduler 未配置时静默返回 0（timer 功能未启用的部署）。
func (e *CustomProcessEngine) cancelInstanceTimers(ctx context.Context, instance *ent.ProcessInstance) int {
	if e.timerStore == nil {
		return 0
	}

	timers, _, err := e.timerStore.List(ctx, TimerListFilter{
		TenantID:          instance.TenantID,
		ProcessInstanceID: &instance.ID,
		Status:            string(TimerStatusPending),
		PageSize:          100,
		Page:              1,
	})
	if err != nil {
		e.logger.Warnw("failed to list pending timers for cancellation",
			"error", err, "instance_id", instance.ID)
		return 0
	}

	cancelled := 0
	for _, timer := range timers {
		if err := e.timerStore.CancelByTimerID(ctx, timer.TimerID); err != nil {
			e.logger.Warnw("failed to cancel timer",
				"error", err, "timer_id", timer.TimerID, "instance_id", instance.ID)
			continue
		}
		if e.timerScheduler != nil {
			e.timerScheduler.Cancel(timer.TimerID)
		}
		cancelled++
	}
	return cancelled
}

// reregisterInstanceTimers 在流程恢复后重建当前活动相关的定时器
// （挂起时已被 cancelInstanceTimers 取消）。Cancel+Recreate 语义：
// 重建按完整时长重新计算 fire_at，而非剩余时长（PRD Q4 决策，P0 可接受）。
func (e *CustomProcessEngine) reregisterInstanceTimers(ctx context.Context, instance *ent.ProcessInstance) error {
	if e.timerStore == nil {
		return nil
	}

	processDef, err := e.client.ProcessDefinition.Query().
		Where(processdefinition.ID(instance.ProcessDefinitionID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("加载流程定义 %d 失败: %w", instance.ProcessDefinitionID, err)
	}

	parser := NewBPMNParser()
	definitions, err := parser.ParseXML(processDef.BpmnXML)
	if err != nil {
		return fmt.Errorf("解析 BPMN XML 失败: %w", err)
	}
	if len(definitions.Processes) == 0 {
		return fmt.Errorf("BPMN definitions contain no processes")
	}
	process := definitions.Processes[0]

	// 当前活动是中间定时捕获事件 → 重建 intermediate timer
	for _, event := range process.IntermediateEvents {
		if event.ID != instance.CurrentActivityID {
			continue
		}
		if event.TimerEventDefinition == nil || !e.hasTimerExpression(event.TimerEventDefinition) {
			return nil
		}
		expression, expressionType := e.extractTimerExpression(event.TimerEventDefinition)
		fireAt, err := CalculateFireAt(expression, expressionType, time.Now())
		if err != nil {
			return fmt.Errorf("计算定时器触发时间失败: %w", err)
		}
		instanceID := instance.ID
		if _, err := e.timerStore.Create(ctx, &CreateTimerRequest{
			TimerType:            TimerTypeIntermediate,
			ProcessDefinitionKey: instance.ProcessDefinitionKey,
			ProcessInstanceID:    &instanceID,
			ActivityID:           event.ID,
			TimerExpression:      expression,
			ExpressionType:       ExpressionType(expressionType),
			FireAt:               fireAt,
			TenantID:             instance.TenantID,
		}); err != nil {
			return fmt.Errorf("重建中间定时器失败: %w", err)
		}
		e.logger.Infow("intermediate timer re-registered on process resume",
			"instance_id", instance.ID, "activity_id", event.ID,
			"expression", expression, "fire_at", fireAt)
		return nil
	}

	// 当前活动是用户任务 → 重建绑定到它的 boundary timers
	for _, task := range process.UserTasks {
		if task.ID == instance.CurrentActivityID {
			e.registerBoundaryTimers(ctx, instance, process, task.ID)
			break
		}
	}
	return nil
}

func extractUserFromContext(ctx context.Context) (int, string) {
	if u, ok := ctx.Value("user").(*ent.User); ok {
		return u.ID, u.Name
	}
	return 0, ""
}
