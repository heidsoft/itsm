package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/common"
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
