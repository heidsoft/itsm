package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/common/tenantctx"
	"itsm-backend/ent"
	"itsm-backend/ent/processtask"
	"itsm-backend/ent/user"
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
//
// 这张表是扫描器与 task_due timer 回调共用的「完整活跃状态表」：
// 漏掉 delegated 会让已转派的任务既不被 timer 分发（timer 侧用
// slices.Contains 同一变量），也不被恢复扫描选中，超时动作静默失效
// （codex 审计 P2-03）。
var timeoutActiveStatuses = []string{
	common.ProcessTaskStatusCreated,
	common.ProcessTaskStatusAssigned,
	common.ProcessTaskStatusStarted,
	common.ProcessTaskStatusDelegated,
}

// errTimeoutAlreadyClaimed 表示该任务已被另一路径（timer 回调 / 恢复扫描 /
// 并发扫描周期）处理过，调用方按「静默跳过」收敛，不计入错误。
var errTimeoutAlreadyClaimed = errors.New("timeout action already claimed")

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

	taskCtx := s.systemAdvanceContext(ctx, tenantID)

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

// systemAdvanceContext 构造超时兜底动作的系统推进上下文（唯一构造点）。
//
// 没有它，auto_approve/auto_reject 走 CompleteTask 会被任务归属校验判成
// 「缺少执行人身份」而失败——timer 回调与恢复扫描两条路径都拿不到决策
// （codex 审计 P2-03）。三点必须显式：
//   - 租户键照旧；
//   - 执行人键显式置空，避免从调用方继承某个真人身份，把系统自动决策
//     伪造成该用户的签名；
//   - BPMN 系统推进标记 + tenantctx 系统旁路，让特权路径在审计里可区分。
//
// timer 回调与扫描器都从这里取上下文，两条路径行为一致。
func (s *TimeoutScanner) systemAdvanceContext(ctx context.Context, tenantID int) context.Context {
	sys := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	sys = context.WithValue(sys, bpmn.BPMNUserIDContextKey, nil)
	sys = bpmn.WithBPMNSystemAdvance(sys)
	return tenantctx.SystemContext(sys,
		"bpmn:timeout-scanner",
		"任务超时兜底动作（notify/escalate/auto_approve/auto_reject）由扫描器以系统身份执行")
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
	// 状态翻转与提醒入箱必须同事务（codex 审计 P2-03）：
	// 旧实现先提交 timeout，再另开事务入箱且吞错——入箱失败后任务已不再是
	// 活跃状态，后续扫描不再选中，这条提醒永久丢失且无人知晓。
	err := s.claimTimeoutStatus(ctx, task, tenantID, common.ProcessTaskStatusTimeout, nil,
		s.timeoutNotifyFunc(task, tenantID, "task_timeout_reminder",
			fmt.Sprintf("任务「%s」已超时，请尽快处理", task.TaskName)))
	if errors.Is(err, errTimeoutAlreadyClaimed) {
		s.logger.Infow("timeout action: notify skipped (already claimed)",
			"task_id", task.TaskID, "tenant_id", tenantID)
		return nil
	}
	if err != nil {
		return err
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

	// 同 actionNotify：升级状态与提醒入箱同事务，任一失败整体回滚。
	err := s.claimTimeoutStatus(ctx, task, tenantID, common.ProcessTaskStatusEscalated, vars,
		s.timeoutNotifyFunc(task, tenantID, "task_escalated",
			fmt.Sprintf("任务「%s」已超时并升级，请关注", task.TaskName)))
	if errors.Is(err, errTimeoutAlreadyClaimed) {
		s.logger.Infow("timeout action: escalate skipped (already claimed)",
			"task_id", task.TaskID, "tenant_id", tenantID)
		return nil
	}
	if err != nil {
		return err
	}

	s.logger.Infow("timeout action: escalate",
		"task_id", task.TaskID, "tenant_id", tenantID, "assignee", task.Assignee)
	return nil
}

// claimTimeoutStatus 在单个事务内完成「claim-once 条件状态翻转 + 提醒入箱」。
//
// claim-once 语义不变：条件更新影响 0 行说明 timer 回调或恢复扫描已处理，
// 返回 errTimeoutAlreadyClaimed 由调用方静默跳过。区别在于入箱失败不再留下
// 「状态已翻转、提醒永久丢失」的半截事实——整个事务回滚，下一轮扫描重试。
// 收件人已失效（被停用/删除）不属于可重试故障：跳过通知、保留状态翻转，
// 否则该任务会每轮扫描都撞同一条确定性错误，永远无法收敛。
func (s *TimeoutScanner) claimTimeoutStatus(
	ctx context.Context,
	task *ent.ProcessTask,
	tenantID int,
	targetStatus string,
	taskVars map[string]interface{},
	notify func(context.Context, *ent.Tx) error,
) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin timeout claim tx: %w", err)
	}
	rollback := func(cause error) error {
		_ = tx.Rollback()
		return cause
	}

	update := tx.ProcessTask.Update().
		Where(
			processtask.ID(task.ID),
			processtask.StatusIn(timeoutActiveStatuses...),
		).
		SetStatus(targetStatus)
	if taskVars != nil {
		update = update.SetTaskVariables(taskVars)
	}
	claimed, err := update.Save(ctx)
	if err != nil {
		return rollback(fmt.Errorf("set timeout status %s: %w", targetStatus, err))
	}
	if claimed == 0 {
		_ = tx.Rollback()
		return errTimeoutAlreadyClaimed
	}

	if notify != nil {
		if err := notify(ctx, tx); err != nil {
			return rollback(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit timeout claim tx: %w", err)
	}
	return nil
}

// timeoutNotifyFunc 返回可在事务内执行的提醒入箱闭包；无 assignee 时为空操作。
func (s *TimeoutScanner) timeoutNotifyFunc(task *ent.ProcessTask, tenantID int, notificationType, content string) func(context.Context, *ent.Tx) error {
	if strings.TrimSpace(task.Assignee) == "" {
		return nil
	}
	return func(ctx context.Context, tx *ent.Tx) error {
		return s.enqueueTimeoutNotificationTx(ctx, tx, task, tenantID, notificationType, content)
	}
}

// enqueueTimeoutNotificationTx 在调用方事务内解析收件人并写入提醒投递命令。
//
// 收件人解析必须在事务内完成：assignee 有两种历史写法——数字用户ID 与
// 用户名（认领/候选回落写的是 username）。旧实现只认数字前缀，于是以用户名
// 认领的任务超时提醒全部静默丢失。解析不到可投递用户（未分配、已停用、
// 跨租户）属于确定性数据状态，跳过通知但保留状态翻转，否则该任务会每轮
// 扫描都撞同一条错误，永远无法收敛。
func (s *TimeoutScanner) enqueueTimeoutNotificationTx(ctx context.Context, tx *ent.Tx, task *ent.ProcessTask, tenantID int, notificationType, content string) error {
	recipientID, err := s.resolveTimeoutRecipient(ctx, tx, tenantID, task.Assignee)
	if err != nil {
		return fmt.Errorf("解析超时提醒收件人失败: %w", err)
	}
	if recipientID <= 0 {
		s.logger.Warnw("timeout scanner: assignee 不是本租户有效用户，跳过超时提醒（状态照常翻转）",
			"task_id", task.TaskID, "tenant_id", tenantID, "assignee", task.Assignee)
		return nil
	}
	if err := EnqueueResourceNotificationTx(ctx, tx, ResourceNotificationCommand{
		TenantID:         tenantID,
		ResourceType:     "process_task",
		ResourceID:       task.ID,
		RecipientID:      recipientID,
		NotificationType: notificationType,
		Channel:          "in_app",
		Content:          content,
		OccurrenceKey:    fmt.Sprintf("task_timeout:%s:%d", task.TaskID, task.DueDate.Unix()),
	}); err != nil {
		return fmt.Errorf("入箱超时提醒失败: %w", err)
	}
	return nil
}

// resolveTimeoutRecipient 把任务 assignee 解析为租户内活跃用户ID；
// 返回 0 表示没有可投递的收件人（不视为错误）。
func (s *TimeoutScanner) resolveTimeoutRecipient(ctx context.Context, tx *ent.Tx, tenantID int, assignee string) (int, error) {
	assignee = strings.TrimSpace(assignee)
	if assignee == "" || assignee == "0" {
		return 0, nil
	}
	base := tx.User.Query().Where(user.TenantIDEQ(tenantID), user.ActiveEQ(true))
	if id, err := strconv.Atoi(assignee); err == nil && id > 0 {
		u, err := base.Where(user.IDEQ(id)).First(ctx)
		if ent.IsNotFound(err) {
			return 0, nil
		}
		if err != nil {
			return 0, err
		}
		return u.ID, nil
	}
	u, err := base.Where(user.UsernameEQ(assignee)).First(ctx)
	if ent.IsNotFound(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return u.ID, nil
}

// actionAutoReject completes the task with rejection decision, triggering the BPMN reject flow.
//
// 决策键必须与引擎读取口径一致：completeTaskWithClient 读的是
// approvalAction / approvalResult / approvalComment（拒绝策略、条件网关、
// 决策审计都只认这三个）。旧实现写 approvalDecision + comment，引擎一个都
// 不读——自动拒绝既不触发配置的拒绝策略，也不写决策审计，依赖
// approvalResult 的网关拿不到决定（codex 审计 P2-03）。
func (s *TimeoutScanner) actionAutoReject(ctx context.Context, task *ent.ProcessTask, tenantID int) error {
	engine := NewCustomProcessEngine(s.client, s.logger)
	vars := map[string]interface{}{
		"approvalAction":  "reject",
		"approvalResult":  "rejected",
		"approvalComment": "任务超时自动拒绝",
		"autoAction":      "auto_reject",
	}
	// ctx 已由 dispatchTimeoutAction 标注为系统推进：归属校验放行，
	// 决策审计按系统身份落 actor_id=0，不伪造个人签名。
	if err := engine.CompleteTask(ctx, task.TaskID, vars); err != nil {
		return fmt.Errorf("auto-reject complete task: %w", err)
	}

	// 决策已在 CompleteTask 的事务里提交；提醒只能随后入箱（CompleteTask
	// 自持事务，无法并入）。失败必须可见：任务已结束，扫描不会再选中它。
	s.notifyAutoDecision(ctx, task, tenantID, "task_auto_rejected",
		fmt.Sprintf("任务「%s」已超时并自动拒绝", task.TaskName))

	s.logger.Infow("timeout action: auto_reject",
		"task_id", task.TaskID, "tenant_id", tenantID)
	return nil
}

// actionAutoApprove completes the task with approval decision, triggering the BPMN approve flow.
// 决策键与系统身份的说明见 actionAutoReject。
func (s *TimeoutScanner) actionAutoApprove(ctx context.Context, task *ent.ProcessTask, tenantID int) error {
	engine := NewCustomProcessEngine(s.client, s.logger)
	vars := map[string]interface{}{
		"approvalAction":  "approve",
		"approvalResult":  "approved",
		"approvalComment": "任务超时自动通过",
		"autoAction":      "auto_approve",
	}
	if err := engine.CompleteTask(ctx, task.TaskID, vars); err != nil {
		return fmt.Errorf("auto-approve complete task: %w", err)
	}

	s.notifyAutoDecision(ctx, task, tenantID, "task_auto_approved",
		fmt.Sprintf("任务「%s」已超时并自动通过", task.TaskName))

	s.logger.Infow("timeout action: auto_approve",
		"task_id", task.TaskID, "tenant_id", tenantID)
	return nil
}

// notifyAutoDecision 为已提交的自动决策补一条提醒；失败只记录（决策已生效，
// 不能因为提醒失败把已完成的任务回滚），但按 Error 级别暴露丢失事实。
func (s *TimeoutScanner) notifyAutoDecision(ctx context.Context, task *ent.ProcessTask, tenantID int, notificationType, content string) {
	if strings.TrimSpace(task.Assignee) == "" {
		return
	}
	if err := s.enqueueTimeoutNotification(ctx, task, tenantID, notificationType, content); err != nil {
		s.logger.Errorw("timeout scanner: 自动决策已生效但提醒未入箱（任务已结束，扫描不再重试）",
			"task_id", task.TaskID, "tenant_id", tenantID,
			"assignee", task.Assignee, "notification_type", notificationType, "error", err)
	}
}

// enqueueTimeoutNotification 独立入箱一条提醒（自动决策路径用：决策已随
// CompleteTask 提交，无法与提醒同事务）。失败返回 error 由调用方记录。
func (s *TimeoutScanner) enqueueTimeoutNotification(ctx context.Context, task *ent.ProcessTask, tenantID int, notificationType, content string) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin notification tx: %w", err)
	}
	if err := s.enqueueTimeoutNotificationTx(ctx, tx, task, tenantID, notificationType, content); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit notification tx: %w", err)
	}
	return nil
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
