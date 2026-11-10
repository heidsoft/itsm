package service

import (
	"context"
	"fmt"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/internal/commandbus"
	"itsm-backend/repository/ticket"
	"itsm-backend/service/bpmn"

	"go.uber.org/zap"
)

// ticketWorkflowApprovalNode 判定当前待办是否是「审批签名」节点。
//
// 工单生命周期动作（分配/接单/解决/关闭）绝不能代替审批人签名：一旦完成审批待办，
// 引擎会把审批决策写进 ProcessApprovalDecision，等于业务侧绕过职责分离伪造了审批。
// 其它域的桥接用 isLifecycleTask 保护审批入口，这里是反向保护，故单独收口。
func ticketWorkflowApprovalNode(task *ent.ProcessTask) bool {
	if task == nil {
		return false
	}
	if purpose, ok := task.TaskVariables["taskPurpose"].(string); ok &&
		strings.EqualFold(strings.TrimSpace(purpose), "approval") {
		return true
	}
	if inferredTaskPurpose(task) == "approval" {
		return true
	}
	// 引擎对未声明用途的老模板会默认按 purpose=work 建待办，不能只信变量：
	// 节点名/ID 本身带审批语义时同样必须 fail-closed。
	haystack := strings.ToLower(task.TaskDefinitionKey + " " + task.TaskName)
	return strings.Contains(haystack, "approv") || strings.Contains(haystack, "审批") || strings.Contains(haystack, "cab")
}

// advanceTicketWorkflowTx 在调用方事务内把工单生命周期动作同步推进到 BPMN 流程。
//
// 与 ApproveTicket 的 CompleteBusinessApprovalTaskWithClient 是同一原语，但语义不同：
// 审批桥接只认审批待办，这里推进「当前这一个作业待办」，把工单状态与流程节点
// 放进同一个 ent.Tx —— 任一侧失败整体回滚，不再出现「工单已解决、实例仍 running」。
//
// 三态（与 AdvanceBusinessWorkflow 对齐）：
//   - 未绑定流程 / 流程已结束 / 当前无待办 → 返回 nil，存量工单与未配置租户保持旧行为；
//   - 当前停在审批节点 → 409 fail-closed，绝不代签；
//   - 已绑定作业待办 → 完成该待办；失败原样上抛（AppError 经 %w 仍可被 errors.As 取回），
//     调用方事务整体回滚，不产生分叉。
func advanceTicketWorkflowTx(
	ctx context.Context,
	txc *ent.Client,
	logger *zap.SugaredLogger,
	tenantID, actorUserID, ticketID int,
	action, comment, status string,
) error {
	if txc == nil || tenantID <= 0 || ticketID <= 0 {
		return nil
	}
	bridge := &BPMNApprovalBridge{client: txc, logger: logger}
	lookup, err := bridge.findFirstPendingTaskWithClient(ctx, txc, tenantID, string(dto.BusinessTypeTicket), ticketID)
	if err != nil {
		return fmt.Errorf("Failed to query ticket workflow tasks: %w", err)
	}
	if !lookup.bound || lookup.task == nil {
		return nil
	}
	task := lookup.task
	if ticketWorkflowApprovalNode(task) {
		return common.NewBusinessError(common.ConflictCode,
			"The workflow is waiting at an approval node. Complete workflow approval before acting on the ticket",
			fmt.Sprintf("ticketId=%d taskId=%s", ticketID, task.TaskID))
	}

	// 刻意不写 approvalAction / approvalResult：作业推进不是审批签名，
	// 带这些变量会让引擎的 recordApprovalDecision 往不可变审计表写假账。
	variables := map[string]interface{}{
		"ticketAction":  action,
		"ticketStatus":  status,
		"actionComment": strings.TrimSpace(comment),
		"actorUserId":   actorUserID,
	}
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	if actorUserID > 0 {
		workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, actorUserID)
	}
	// 与审批桥接同一受控信任：只放开「triggered_by 回落认领态」，
	// 用户主动认领 / 静态指派的任务仍严格按 assignee 校验。
	workflowCtx = withBusinessApprovalTrust(workflowCtx)
	// 未指派任务是模板没有候选人、也没有 triggered_by 回落时的形态（生产日志
	// assignmentSource=none）。业务侧已过 ticket:assign/resolve/close 路由闸门，
	// 此处若继续按个人归属校验会恒 403，把已经可用的工单动作整体堵死。
	// 仅对「无人认领」放开，且保留 actorUserID 供审计；已指派给他人的任务不受影响。
	if strings.TrimSpace(task.Assignee) == "" && strings.TrimSpace(task.CandidateUsers) == "" {
		workflowCtx = bpmn.WithBPMNSystemAdvance(workflowCtx)
	}

	engine := NewCustomProcessEngine(txc, logger)
	if customEngine, ok := engine.(*CustomProcessEngine); ok {
		if err := customEngine.CompleteTaskWithClient(workflowCtx, txc, task.TaskID, variables); err != nil {
			return fmt.Errorf("Failed to advance ticket workflow task: %w", err)
		}
	} else if err := engine.CompleteTask(workflowCtx, task.TaskID, variables); err != nil {
		return fmt.Errorf("Failed to advance ticket workflow task: %w", err)
	}
	logger.Infow("工单生命周期动作已同事务推进BPMN待办",
		"ticket_id", ticketID, "tenant_id", tenantID, "action", action,
		"task_id", task.TaskID, "actor_user_id", actorUserID)
	return nil
}

// AssignTicketWithWorkflowAdvance 在单个事务内完成「工单分配 + BPMN 当前作业待办推进」。
//
// `/tickets/:id/assign` 是详情页实际调用的入口，原来只走 productionSvc.AssignTicket
// （工单表更新在 updateTicketWithFeishuCommand 的事务里，流程推进零调用），因此
// 「分配成功但实例停在任务分配节点」。这里复用 repository 的 UpdateWithTxHook，
// 把流程推进挂进同一个事务，避免复制第二套状态机或引入两段提交分叉。
//
// 无事务钩子的仓储（单测 mock / 旧装配）保持旧行为，由盘点工具暴露分叉，不静默伪报成功。
func (s *TicketService) AssignTicketWithWorkflowAdvance(
	ctx context.Context, ticketID, assigneeID, tenantID, actorUserID int,
) (*ticket.Ticket, error) {
	current, err := s.repo.GetByID(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}
	if err := current.Assign(assigneeID); err != nil {
		return nil, err
	}
	if err := s.ValidateAssignee(ctx, assigneeID, tenantID); err != nil {
		return nil, err
	}
	status := current.Status
	params := &ticket.UpdateParams{
		AssigneeID: &assigneeID,
		Status:     &status,
		Version:    current.Version,
	}

	updater, ok := s.repo.(ticket.TransactionalUpdater)
	if !ok {
		return s.AssignTicket(ctx, ticketID, assigneeID, tenantID)
	}

	updated, err := updater.UpdateWithTxHook(ctx, ticketID, params, tenantID,
		func(tx *ent.Tx, updatedTicket *ticket.Ticket) error {
			// 飞书同步是 best-effort 副作用，入箱失败不回滚主流程（与原实现一致）。
			if s.sideEffectOutboxEnabled {
				if _, err := commandbus.EnqueueTx(ctx, tx, commandbus.EnqueueRequest{
					TenantID: tenantID, CommandType: commandbus.CommandSyncTicketFeishu,
					AggregateType: "ticket", AggregateID: updatedTicket.ID,
					IdempotencyKey: fmt.Sprintf("ticket:%d:feishu:sync:v%d", updatedTicket.ID, updatedTicket.Version),
					Payload:        map[string]interface{}{"event": "assigned", "version": updatedTicket.Version},
				}); err != nil {
					s.logger.Warnw("Enqueue ticket feishu sync failed (best-effort, ignoring)",
						"error", err, "ticket_id", updatedTicket.ID)
				}
			}
			// 流程推进是业务一致性的一部分，失败必须 fail-closed 让整个事务回滚。
			return advanceTicketWorkflowTx(ctx, tx.Client(), s.logger, tenantID, actorUserID,
				ticketID, "assign", "", string(updatedTicket.Status))
		})
	if err != nil {
		return nil, err
	}

	if s.notificationSvc != nil {
		if err := s.notificationSvc.NotifyTicketAssigned(ctx, ticketID, assigneeID, tenantID); err != nil {
			s.logger.Warnw("Assignment notification enqueue failed", "error", err, "ticket_id", ticketID)
		}
	}
	if s.connectorManager != nil && !s.sideEffectOutboxEnabled {
		s.syncTicketToFeishuLegacyAsync(updated, tenantID)
	}
	return updated, nil
}
