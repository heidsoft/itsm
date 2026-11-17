package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/common"
	"itsm-backend/common/tenantctx"
	"itsm-backend/ent"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"

	"go.uber.org/zap"
)

type TimeoutScanner struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewTimeoutScanner(client *ent.Client, logger *zap.SugaredLogger) *TimeoutScanner {
	return &TimeoutScanner{client: client, logger: logger}
}

// ScanOverdueTasks finds active tasks past their due date within a tenant
// and dispatches the configured timeout action for each.
// timeoutActiveStatuses 视为"未处理"的任务状态。扫描器查询与
// claim-once 条件更新共用，保证 timer 回调与恢复兜底扫描双路径下
// 同一任务的超时动作只生效一次。
var timeoutActiveStatuses = []string{
	common.ProcessTaskStatusCreated,
	common.ProcessTaskStatusAssigned,
	common.ProcessTaskStatusStarted,
}

func (s *TimeoutScanner) ScanOverdueTasks(ctx context.Context, tenantID int) (int, error) {
	if tenantID <= 0 {
		return 0, fmt.Errorf("timeout scanner: invalid tenant ID")
	}

	now := time.Now()
	tasks, err := s.client.ProcessTask.Query().
		Where(
			processtask.TenantID(tenantID),
			processtask.StatusIn(timeoutActiveStatuses...),
			processtask.DueDateNotNil(),
			processtask.DueDateLT(now),
		).
		All(ctx)
	if err != nil {
		return 0, fmt.Errorf("timeout scanner: query overdue tasks: %w", err)
	}

	processed := 0
	for _, task := range tasks {
		if err := s.dispatchTimeoutAction(ctx, task, tenantID); err != nil {
			s.logger.Warnw("timeout scanner: dispatch failed",
				"task_id", task.TaskID,
				"tenant_id", tenantID,
				"error", err,
			)
			continue
		}
		processed++
	}

	return processed, nil
}

func (s *TimeoutScanner) dispatchTimeoutAction(ctx context.Context, task *ent.ProcessTask, tenantID int) error {
	action := extractTimeoutAction(task.TaskVariables)
	if action == "" {
		action = common.TimeoutActionNotify
	}

	taskCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)

	switch action {
	case common.TimeoutActionNotify:
		return s.actionNotify(taskCtx, task, tenantID)
	case common.TimeoutActionEscalate:
		return s.actionEscalate(taskCtx, task, tenantID)
	case common.TimeoutActionAutoReject:
		return s.actionAutoReject(taskCtx, task, tenantID)
	case common.TimeoutActionAutoApprove:
		return s.actionAutoApprove(taskCtx, task, tenantID)
	default:
		s.logger.Warnw("timeout scanner: unknown action, falling back to notify",
			"task_id", task.TaskID, "action", action)
		return s.actionNotify(taskCtx, task, tenantID)
	}
}

func extractTimeoutAction(vars map[string]interface{}) string {
	if vars == nil {
		return ""
	}
	v, ok := vars["timeoutAction"]
	if !ok {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		return ""
	}
	return str
}

// actionNotify marks the task as timed out and enqueues a reminder notification.
func (s *TimeoutScanner) actionNotify(ctx context.Context, task *ent.ProcessTask, tenantID int) error {
	// claim-once：仅当任务仍处于活跃状态时才置为 timeout。
	// 影响行数为 0 说明 timer 回调或恢复扫描已处理过（双路径竞态保护）。
	claimed, err := s.client.ProcessTask.Update().
		Where(
			processtask.ID(task.ID),
			processtask.StatusIn(timeoutActiveStatuses...),
		).
		SetStatus(common.ProcessTaskStatusTimeout).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("set timeout status: %w", err)
	}
	if claimed == 0 {
		s.logger.Infow("timeout action: notify skipped (already claimed)",
			"task_id", task.TaskID, "tenant_id", tenantID)
		return nil
	}

	assigneeID, _ := parseAssigneeID(task.Assignee)
	if assigneeID > 0 {
		s.enqueueTimeoutNotification(ctx, task, tenantID, assigneeID, "task_timeout_reminder",
			fmt.Sprintf("任务「%s」已超时，请尽快处理", task.TaskName))
	}

	s.logger.Infow("timeout action: notify",
		"task_id", task.TaskID, "tenant_id", tenantID, "assignee", task.Assignee)
	return nil
}

// actionEscalate marks the task as escalated and notifies the assignee.
func (s *TimeoutScanner) actionEscalate(ctx context.Context, task *ent.ProcessTask, tenantID int) error {
	vars := task.TaskVariables
	if vars == nil {
		vars = make(map[string]interface{})
	}
	vars["escalation_reason"] = "任务超时自动升级"
	vars["escalated_time"] = time.Now().Format(time.RFC3339)

	// claim-once：同 actionNotify，防止 timer 回调与恢复扫描双路径重复升级。
	claimed, err := s.client.ProcessTask.Update().
		Where(
			processtask.ID(task.ID),
			processtask.StatusIn(timeoutActiveStatuses...),
		).
		SetStatus(common.ProcessTaskStatusEscalated).
		SetTaskVariables(vars).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("set escalated status: %w", err)
	}
	if claimed == 0 {
		s.logger.Infow("timeout action: escalate skipped (already claimed)",
			"task_id", task.TaskID, "tenant_id", tenantID)
		return nil
	}

	assigneeID, _ := parseAssigneeID(task.Assignee)
	if assigneeID > 0 {
		s.enqueueTimeoutNotification(ctx, task, tenantID, assigneeID, "task_escalated",
			fmt.Sprintf("任务「%s」已超时并升级，请关注", task.TaskName))
	}

	s.logger.Infow("timeout action: escalate",
		"task_id", task.TaskID, "tenant_id", tenantID, "assignee", task.Assignee)
	return nil
}

// actionAutoReject completes the task with rejection decision, triggering the BPMN reject flow.
func (s *TimeoutScanner) actionAutoReject(ctx context.Context, task *ent.ProcessTask, tenantID int) error {
	engine := NewCustomProcessEngine(s.client, s.logger)
	vars := map[string]interface{}{
		"approvalDecision": "reject",
		"comment":          "任务超时自动拒绝",
		"autoAction":       "auto_reject",
	}
	if err := engine.CompleteTask(ctx, task.TaskID, vars); err != nil {
		return fmt.Errorf("auto-reject complete task: %w", err)
	}

	assigneeID, _ := parseAssigneeID(task.Assignee)
	if assigneeID > 0 {
		s.enqueueTimeoutNotification(ctx, task, tenantID, assigneeID, "task_auto_rejected",
			fmt.Sprintf("任务「%s」已超时并自动拒绝", task.TaskName))
	}

	s.logger.Infow("timeout action: auto_reject",
		"task_id", task.TaskID, "tenant_id", tenantID)
	return nil
}

// actionAutoApprove completes the task with approval decision, triggering the BPMN approve flow.
func (s *TimeoutScanner) actionAutoApprove(ctx context.Context, task *ent.ProcessTask, tenantID int) error {
	engine := NewCustomProcessEngine(s.client, s.logger)
	vars := map[string]interface{}{
		"approvalDecision": "approve",
		"comment":          "任务超时自动通过",
		"autoAction":       "auto_approve",
	}
	if err := engine.CompleteTask(ctx, task.TaskID, vars); err != nil {
		return fmt.Errorf("auto-approve complete task: %w", err)
	}

	assigneeID, _ := parseAssigneeID(task.Assignee)
	if assigneeID > 0 {
		s.enqueueTimeoutNotification(ctx, task, tenantID, assigneeID, "task_auto_approved",
			fmt.Sprintf("任务「%s」已超时并自动通过", task.TaskName))
	}

	s.logger.Infow("timeout action: auto_approve",
		"task_id", task.TaskID, "tenant_id", tenantID)
	return nil
}

func (s *TimeoutScanner) enqueueTimeoutNotification(ctx context.Context, task *ent.ProcessTask, tenantID, recipientID int, notificationType, content string) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		s.logger.Warnw("timeout scanner: begin tx for notification failed", "error", err)
		return
	}
	err = EnqueueResourceNotificationTx(ctx, tx, ResourceNotificationCommand{
		TenantID:         tenantID,
		ResourceType:     "process_task",
		ResourceID:       task.ID,
		RecipientID:      recipientID,
		NotificationType: notificationType,
		Channel:          "in_app",
		Content:          content,
		OccurrenceKey:    fmt.Sprintf("task_timeout:%s:%d", task.TaskID, task.DueDate.Unix()),
	})
	if err != nil {
		_ = tx.Rollback()
		s.logger.Warnw("timeout scanner: enqueue notification failed",
			"task_id", task.TaskID, "error", err)
		return
	}
	if err := tx.Commit(); err != nil {
		s.logger.Warnw("timeout scanner: commit notification failed",
			"task_id", task.TaskID, "error", err)
	}
}

// ---------------------------------------------------------------------------
// 会签 finalizing 卡死恢复
// ---------------------------------------------------------------------------

// countersignFinalizingGrace 会签父任务停留在 finalizing 状态超过该时长即视为
// 「抢占后推进失败/进程崩溃」的残留，触发系统恢复推进。正常路径下末票投票
// 抢占 finalizing 到 CompleteTask 推进在毫秒级完成，5 分钟宽限足以避开
// 慢查询下的误恢复，又能把崩溃窗口的卡死时长压到分钟级。
const countersignFinalizingGrace = 5 * time.Minute

// ScanStuckFinalizingTasks 恢复卡在 finalizing 状态的会签父任务。
//
// 背景（P1 崩溃窗口）：Vote 的末票流程是「CAS 抢占 finalizing（事务外）→
// 取消其余子任务 → engine.CompleteTask 系统推进（新事务）」。抢占后、推进前
// 进程崩溃，父任务永久停留 finalizing：无定时器、无扫描器认领，只能人工修库。
//
// 恢复语义与 Vote 末票保持一致：
//  1. 重新按子任务实际状态推导会签结论（approved/rejected）；
//  2. 结论仍为 pending（阈值未凑齐，说明抢占是误写）→ 回退 assigned，让投票继续；
//  3. 结论已定 → 走与 Vote 相同的系统上下文 CompleteTask 推进父任务。
//
// claim-once：恢复前用条件更新把 updated_at 推进作为认领标记（乐观锁），
// 同一任务不会被两个扫描周期并发恢复；CompleteTask 自带
// StatusNEQ(completed/cancelled) 条件更新，重复推进天然幂等失败。
func (s *TimeoutScanner) ScanStuckFinalizingTasks(ctx context.Context, tenantID int) (int, error) {
	if tenantID <= 0 {
		return 0, fmt.Errorf("finalizing scanner: invalid tenant ID")
	}

	cutoff := time.Now().Add(-countersignFinalizingGrace)
	// parent_task_id 是 Optional 字段：会签父任务未设置该列（NULL），
	// 不是空串——ParentTaskIDEQ("") 永远匹配不到，必须用 IsNil。
	stuck, err := s.client.ProcessTask.Query().
		Where(
			processtask.TenantID(tenantID),
			processtask.StatusEQ(common.ProcessTaskStatusFinalizing),
			processtask.ParentTaskIDIsNil(),
			processtask.UpdatedAtLT(cutoff),
		).
		All(ctx)
	if err != nil {
		return 0, fmt.Errorf("finalizing scanner: query stuck parents: %w", err)
	}

	recovered := 0
	for _, parent := range stuck {
		if err := s.recoverFinalizingParent(ctx, parent, tenantID); err != nil {
			s.logger.Warnw("finalizing scanner: recover failed",
				"task_id", parent.TaskID, "tenant_id", tenantID, "error", err)
			continue
		}
		recovered++
	}
	return recovered, nil
}

// recoverFinalizingParent 恢复单个卡死的会签父任务。
func (s *TimeoutScanner) recoverFinalizingParent(ctx context.Context, parent *ent.ProcessTask, tenantID int) error {
	status, err := s.countersignStatusFromDB(ctx, parent.TaskID, tenantID)
	if err != nil {
		return fmt.Errorf("重算会签状态失败: %w", err)
	}

	switch status.Status {
	case "pending":
		// 阈值未凑齐却停在 finalizing：抢占写发生在错误时刻（如取消子任务
		// 与末票投票并发）。回退到 assigned 让会签继续，不能替用户投票。
		reverted, err := s.client.ProcessTask.Update().
			Where(
				processtask.ID(parent.ID),
				processtask.StatusEQ(common.ProcessTaskStatusFinalizing),
			).
			SetStatus(common.ProcessTaskStatusAssigned).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("回退误抢占的 finalizing 状态失败: %w", err)
		}
		if reverted == 0 {
			return nil // 并发者已处理
		}
		s.logger.Infow("finalizing scanner: reverted premature claim back to assigned",
			"task_id", parent.TaskID, "tenant_id", tenantID)
		return nil
	case "approved", "rejected":
		// 与 Vote 末票一致的系统上下文推进：显式声明特权路径，可审计。
		systemCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
		systemCtx = context.WithValue(systemCtx, bpmn.BPMNUserIDContextKey, nil)
		systemCtx = tenantctx.SystemContext(
			systemCtx,
			"bpmn:countersign-finalizing-recovery",
			"会签父任务抢占 finalizing 后未推进（进程崩溃或推进失败），由扫描器恢复推进",
		)
		engine := NewCustomProcessEngine(s.client, s.logger)
		if err := engine.CompleteTask(systemCtx, parent.TaskID, map[string]interface{}{
			"approvalResult": status.Status,
			"approved":       status.Status == "approved",
		}); err != nil {
			return fmt.Errorf("恢复推进会签父任务失败: %w", err)
		}
		s.logger.Infow("finalizing scanner: recovered stuck countersign parent",
			"task_id", parent.TaskID, "tenant_id", tenantID, "final_status", status.Status)
		return nil
	default:
		return fmt.Errorf("未知会签状态 %q", status.Status)
	}
}

// countersignStatusFromDB 按子任务实际状态重算会签结论（与
// bpmnTaskService.GetCounterSignStatus 同一推导口径，但不依赖请求上下文，
// 供扫描器在系统上下文使用）。
func (s *TimeoutScanner) countersignStatusFromDB(ctx context.Context, parentTaskID string, tenantID int) (*CounterSignStatus, error) {
	subTasks, err := s.client.ProcessTask.Query().
		Where(processtask.ParentTaskID(parentTaskID), processtask.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询会签子任务失败: %w", err)
	}

	status := &CounterSignStatus{
		ParentTaskID: parentTaskID,
		Total:        len(subTasks),
		Pending:      len(subTasks),
		Status:       "pending",
	}
	for _, task := range subTasks {
		switch task.Status {
		case common.ProcessTaskStatusCompleted:
			status.Completed++
			status.Pending--
			if vars := task.TaskVariables; vars != nil {
				if approved, ok := vars["approved"].(bool); ok && approved {
					status.Approved++
				} else {
					status.Rejected++
				}
			} else {
				// 变量缺失按反对计，与 GetCounterSignStatus 现行口径一致；
				// 谨慎侧：宁可重走人工确认也不要把无凭据的票算赞成。
				status.Rejected++
			}
		case common.ProcessTaskStatusAssigned, common.ProcessTaskStatusCreated:
			// 仍待投
		}
	}

	threshold := status.Total
	if parent, err := s.client.ProcessTask.Query().
		Where(processtask.TaskID(parentTaskID), processtask.TenantID(tenantID)).Only(ctx); err == nil {
		if value, ok := numericInt(parent.TaskVariables["threshold"]); ok && value > 0 {
			threshold = value
		}
	}
	if status.Approved >= threshold {
		status.Status = "approved"
	} else if status.Approved+status.Pending < threshold {
		status.Status = "rejected"
	} else {
		status.Status = "pending"
	}
	return status, nil
}
