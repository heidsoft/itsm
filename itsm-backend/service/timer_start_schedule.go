package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
)

// ---------------------------------------------------------------------------
// timer_start_schedule.go — Timer Start Event 时间表同步（Phase 5）
//
// 职责：把一份 BPMN 定义中的 Timer Start Event 翻译成 process_timer 记录，
// 并在流程重发布/重部署时整体替换旧时间表。
//
// 调用点：
//   - BPMNDeploymentService.DeployProcessDefinition（部署即注册）
//   - bpmnProcessDefinitionService.PublishProcessDefinition（设计器发布，生产主路径）
//   - bpmnProcessDefinitionService.SetProcessDefinitionActive(false)（停用即取消）
//
// 语义：**整体替换**。同一 (tenant, process_key) 下先取消全部 pending start timer，
// 再按新定义重建。cron/cycle 依租户时区解析（PRD §12 决策 Q1），fire_at 以 UTC 落库（决策 Q4）。
// ---------------------------------------------------------------------------

// SyncStartTimers 用 BPMN XML 同步 start timer 时间表，返回注册的 timer 数量。
func SyncStartTimers(ctx context.Context, client *ent.Client, store TimerStore, bpmnXML []byte, processKey string, tenantID int) (int, error) {
	if store == nil {
		return 0, nil
	}
	parser := NewBPMNParser()
	definitions, err := parser.ParseXML(bpmnXML)
	if err != nil {
		return 0, fmt.Errorf("解析 BPMN XML 失败: %w", err)
	}
	return SyncStartTimersFromDefinitions(ctx, client, store, definitions, processKey, tenantID)
}

// SyncStartTimersFromDefinitions 同步 start timer 时间表（已有解析结果时使用）。
func SyncStartTimersFromDefinitions(ctx context.Context, client *ent.Client, store TimerStore, definitions *BPMNDefinitions, processKey string, tenantID int) (int, error) {
	if store == nil || definitions == nil {
		return 0, nil
	}

	timerEvents, err := ExtractTimerEvents(definitions, processKey)
	if err != nil {
		return 0, fmt.Errorf("提取定时器事件失败: %w", err)
	}

	// 幂等：重发布前清空旧时间表（否则每次发布都会追加一份，导致重复启动流程实例）。
	if _, err := CancelPendingStartTimers(ctx, client, tenantID, processKey); err != nil {
		return 0, fmt.Errorf("取消旧开始定时器失败: %w", err)
	}

	loc := LoadTenantLocation(ctx, client, tenantID)
	now := time.Now()
	registered := 0

	for _, timerEvent := range timerEvents {
		if timerEvent.TimerType != "start" {
			continue
		}

		exprType := ExpressionType(timerEvent.ExpressionType)

		fireAt, err := NextFireAt(timerEvent.Expression, exprType, loc, now)
		if err != nil {
			return registered, fmt.Errorf("计算定时器触发时间失败（%s）: %w", timerEvent.ActivityID, err)
		}

		// 一次性（date）且已过期 → 不注册：启动一个"本应在过去启动"的流程没有业务语义。
		// 静默跳过优于立即触发；误配由 lint 规则提示。
		if !IsRecurring(timerEvent.Expression, exprType) && !fireAt.After(now) {
			continue
		}

		// 重排元数据：cycle 的剩余次数用于触发后递减；R/ 无限重复不写入。
		ctxVars := map[string]interface{}{}
		if remaining, bounded, ok := CycleRemaining(timerEvent.Expression); ok && bounded {
			ctxVars["remaining_repetitions"] = remaining
		}

		if _, err := store.Create(ctx, &CreateTimerRequest{
			TimerType:            TimerType(timerEvent.TimerType),
			ProcessDefinitionKey: timerEvent.ProcessDefinitionKey,
			ActivityID:           timerEvent.ActivityID,
			TimerExpression:      timerEvent.Expression,
			ExpressionType:       exprType,
			FireAt:               fireAt,
			ContextVariables:     ctxVars,
			TenantID:             tenantID,
		}); err != nil {
			return registered, fmt.Errorf("创建定时器记录失败（%s）: %w", timerEvent.ActivityID, err)
		}
		registered++
	}

	return registered, nil
}

// CancelStartTimers 取消某流程定义 Key 的 start timer 时间表（流程停用/删除时调用）。
func CancelStartTimers(ctx context.Context, client *ent.Client, processKey string, tenantID int) (int, error) {
	return CancelPendingStartTimers(ctx, client, tenantID, processKey)
}
