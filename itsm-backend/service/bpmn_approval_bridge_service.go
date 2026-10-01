package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"

	"go.uber.org/zap"
)

// BPMNApprovalBridge 将业务域审批入口（工单/变更 approve|reject）桥接到 BPMN 任务完成，
// 保证审批动作以 BPMN 任务为权威来源，流程实例不因业务侧直批而悬挂（P0-1 双轨审批收敛）。
//
// 约定：ProcessTriggerService 以 businessKey = "{business_type}:{business_id}" 启动流程实例，
// 桥接层按同一约定反查运行中的实例及其待办用户任务。
//
// 返回语义（三态，与 AdvanceBusinessWorkflow 一致）：
//   - (true, nil)   已桥接完成流程待办，调用方不得再自行裁决同一审批；
//   - (false, nil)  业务对象从未绑定运行中流程，调用方按旧审批链处理；
//   - (false, err)  已绑定流程但当前无可操作待办，或待办完成失败 —— 必须中止业务直批。
//
// (false, nil) 与 conflict 不能合并：前者是「本就不走 BPMN」的存量/未配置租户，
// 后者是「该由流程裁决却没有待办」的绕过风险面，两者对调用方的处置完全不同。
type BPMNApprovalBridge struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewBPMNApprovalBridge(client *ent.Client, logger *zap.SugaredLogger) *BPMNApprovalBridge {
	return &BPMNApprovalBridge{client: client, logger: logger}
}

// CompleteBusinessApprovalTaskWithClient 使用调用方事务绑定的 Ent client 完成审批任务。
// 它不创建或提交事务，任何失败都由外层事务连同业务状态一起回滚。
func (b *BPMNApprovalBridge) CompleteBusinessApprovalTaskWithClient(ctx context.Context, txc *ent.Client, tenantID, actorUserID int, businessType string, businessID int, action, comment string) (bool, error) {
	if tenantID <= 0 || actorUserID <= 0 {
		return false, common.NewBusinessError(common.UnauthorizedCode, "缺少审批身份或租户上下文", "")
	}
	if businessID <= 0 || strings.TrimSpace(businessType) == "" || (action != "approve" && action != "reject") {
		return false, common.NewBusinessError(common.ParamErrorCode, "审批参数无效", "")
	}

	lookup, err := b.findPendingApprovalTaskWithClient(ctx, txc, tenantID, businessType, businessID)
	if err != nil {
		return false, err
	}
	if !lookup.bound {
		return false, nil
	}
	if lookup.task == nil {
		return false, bpmnApprovalNotActionable(businessType, businessID)
	}
	task := lookup.task

	approvalResult := "approved"
	if action == "reject" {
		approvalResult = "rejected"
	}
	variables := map[string]interface{}{
		"approvalAction":  action,
		"approvalResult":  approvalResult,
		"approvalComment": strings.TrimSpace(comment),
	}

	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, actorUserID)

	engine := NewCustomProcessEngine(txc, b.logger)
	if customEngine, ok := engine.(*CustomProcessEngine); ok {
		if err := customEngine.CompleteTaskWithClient(workflowCtx, txc, task.TaskID, variables); err != nil {
			return false, fmt.Errorf("完成流程审批任务失败: %w", err)
		}
	} else {
		if err := engine.CompleteTask(workflowCtx, task.TaskID, variables); err != nil {
			return false, fmt.Errorf("完成流程审批任务失败: %w", err)
		}
	}

	b.logger.Infow("业务审批已桥接完成BPMN任务(事务内)",
		"businessType", businessType, "businessId", businessID, "taskId", task.TaskID, "action", action, "actorUserId", actorUserID)
	return true, nil
}

// CompleteBusinessApprovalTask 把业务审批桥接到流程待办；三态语义见类型注释。
func (b *BPMNApprovalBridge) CompleteBusinessApprovalTask(ctx context.Context, tenantID, actorUserID int, businessType string, businessID int, action, comment string) (bool, error) {
	if tenantID <= 0 || actorUserID <= 0 {
		return false, common.NewBusinessError(common.UnauthorizedCode, "缺少审批身份或租户上下文", "")
	}
	if businessID <= 0 || strings.TrimSpace(businessType) == "" || (action != "approve" && action != "reject") {
		return false, common.NewBusinessError(common.ParamErrorCode, "审批参数无效", "")
	}

	lookup, err := b.findPendingApprovalTask(ctx, tenantID, businessType, businessID)
	if err != nil {
		return false, err
	}
	if !lookup.bound {
		return false, nil
	}
	if lookup.task == nil {
		return false, bpmnApprovalNotActionable(businessType, businessID)
	}
	task := lookup.task

	approvalResult := "approved"
	if action == "reject" {
		approvalResult = "rejected"
	}
	variables := map[string]interface{}{
		"approvalAction":  action,
		"approvalResult":  approvalResult,
		"approvalComment": strings.TrimSpace(comment),
	}

	// 注入认证操作人与租户，供引擎的 authorizeTaskActor / recordApprovalDecision 使用
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, actorUserID)

	engine := NewCustomProcessEngine(b.client, b.logger)
	if err := engine.CompleteTask(workflowCtx, task.TaskID, variables); err != nil {
		return false, fmt.Errorf("完成流程审批任务失败: %w", err)
	}

	b.logger.Infow("业务审批已桥接完成BPMN任务",
		"businessType", businessType, "businessId", businessID, "taskId", task.TaskID, "action", action, "actorUserId", actorUserID)
	return true, nil
}

func (b *BPMNApprovalBridge) DelegateBusinessApprovalTask(ctx context.Context, tenantID, actorUserID int, businessType string, businessID, newAssigneeUserID int) (bool, error) {
	if tenantID <= 0 || actorUserID <= 0 {
		return false, common.NewBusinessError(common.UnauthorizedCode, "缺少审批身份或租户上下文", "")
	}
	if businessID <= 0 || newAssigneeUserID <= 0 || strings.TrimSpace(businessType) == "" {
		return false, common.NewBusinessError(common.ParamErrorCode, "委派参数无效", "")
	}

	lookup, err := b.findPendingApprovalTask(ctx, tenantID, businessType, businessID)
	if err != nil {
		return false, err
	}
	if !lookup.bound {
		return false, nil
	}
	if lookup.task == nil {
		return false, bpmnApprovalNotActionable(businessType, businessID)
	}
	task := lookup.task

	// 注入认证操作人与租户，委派前校验操作人必须是当前任务的审批人/候选人，防止越权改派
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, actorUserID)

	engine := NewCustomProcessEngine(b.client, b.logger)
	if customEngine, ok := engine.(*CustomProcessEngine); ok {
		if err := customEngine.authorizeTaskActor(workflowCtx, task); err != nil {
			return false, fmt.Errorf("委派流程任务失败: %w", err)
		}
	}
	if err := engine.TaskService().DelegateTask(workflowCtx, task.TaskID, strconv.Itoa(newAssigneeUserID)); err != nil {
		return false, fmt.Errorf("委派流程任务失败: %w", err)
	}

	b.logger.Infow("业务审批委派已同步BPMN任务",
		"businessType", businessType, "businessId", businessID, "taskId", task.TaskID,
		"actorUserId", actorUserID, "newAssigneeUserId", newAssigneeUserID)
	return true, nil
}

// AdvanceBusinessWorkflow 按业务键依次完成流程实例的待办用户任务（最多 maxSteps 个），
// 用于将业务侧生命周期动作（提交/排期/实施/验证/驳回收尾等）同步推进到 BPMN 流程，
// 避免业务状态与流程状态分叉。
//
// 返回语义：
//   - (true, nil)  至少推进了一个流程任务；
//   - (false, nil) 无关联运行中流程实例或无待办用户任务，调用方按旧逻辑处理；
//   - (false, err) 存在待办任务但推进失败，调用方应中止业务动作。
func (b *BPMNApprovalBridge) AdvanceBusinessWorkflow(ctx context.Context, tenantID, actorUserID int, businessType string, businessID int, variables map[string]interface{}, maxSteps int) (bool, error) {
	return b.advanceBusinessWorkflow(ctx, b.client, tenantID, actorUserID, businessType, businessID, variables, maxSteps, false)
}

// AdvanceBusinessWorkflowWithClient 使用调用方事务绑定的 Ent client 推进流程。
// 它不创建或提交事务，任何失败都由外层事务连同业务状态一起回滚。
func (b *BPMNApprovalBridge) AdvanceBusinessWorkflowWithClient(ctx context.Context, txc *ent.Client, tenantID, actorUserID int, businessType string, businessID int, variables map[string]interface{}, maxSteps int) (bool, error) {
	return b.advanceBusinessWorkflow(ctx, txc, tenantID, actorUserID, businessType, businessID, variables, maxSteps, true)
}

func (b *BPMNApprovalBridge) advanceBusinessWorkflow(ctx context.Context, client *ent.Client, tenantID, actorUserID int, businessType string, businessID int, variables map[string]interface{}, maxSteps int, reuseTransaction bool) (bool, error) {
	if tenantID <= 0 || businessID <= 0 || maxSteps <= 0 {
		return false, nil
	}

	handled := false
	for step := 0; step < maxSteps; step++ {
		lookup, err := b.findPendingApprovalTaskWithClient(ctx, client, tenantID, businessType, businessID)
		if err != nil {
			return handled, err
		}
		if !lookup.bound || lookup.task == nil {
			return handled, nil
		}
		task := lookup.task

		workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
		workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, actorUserID)

		engine := NewCustomProcessEngine(client, b.logger)
		var completeErr error
		if customEngine, ok := engine.(*CustomProcessEngine); reuseTransaction && ok {
			completeErr = customEngine.CompleteTaskWithClient(workflowCtx, client, task.TaskID, variables)
		} else {
			completeErr = engine.CompleteTask(workflowCtx, task.TaskID, variables)
		}
		if completeErr != nil {
			return handled, fmt.Errorf("推进流程任务失败: %w", completeErr)
		}
		handled = true
	}

	return handled, nil
}

// HasPendingUserTasksWithClient 检查指定业务键是否仍有待办 BPMN 用户任务（事务内）。
// 用于审批完成后判定业务状态：approve 后无剩余待办 → 全部审批通过。
func (b *BPMNApprovalBridge) HasPendingUserTasksWithClient(ctx context.Context, client *ent.Client, tenantID int, businessType string, businessID int) (bool, error) {
	businessKey := fmt.Sprintf("%s:%d", strings.ToLower(businessType), businessID)
	instance, err := client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
			processinstance.Status("running"),
		).
		Order(ent.Desc(processinstance.FieldStartTime)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("查询业务关联流程实例失败: %w", err)
	}

	exists, err := client.ProcessTask.Query().
		Where(
			processtask.ProcessInstanceID(instance.ID),
			processtask.TenantID(tenantID),
			processtask.TaskType("user_task"),
			processtask.StatusIn(
				common.ProcessTaskStatusCreated,
				common.ProcessTaskStatusAssigned,
				common.ProcessTaskStatusStarted,
				common.ProcessTaskStatusDelegated,
			),
		).
		Exist(ctx)
	if err != nil {
		return false, fmt.Errorf("查询流程待办任务失败: %w", err)
	}
	return exists, nil
}

// bpmnApprovalNotActionable 表示流程已绑定却拿不到可操作待办：待办可能已被并发消费，
// 或流程停在别的节点上。此时业务侧直批会让流程状态与业务状态分叉，必须失败关闭。
func bpmnApprovalNotActionable(businessType string, businessID int) error {
	return common.NewBusinessError(
		common.ConflictCode,
		"流程已绑定但当前没有可处理的审批待办，请勿直接业务审批",
		fmt.Sprintf("businessType=%s businessId=%d", strings.ToLower(businessType), businessID),
	)
}

// approvalTaskLookup 是按业务键反查流程待办的结果，必须区分「从未绑定」与「绑定过但此刻不可操作」：
// 前者调用方回退旧审批链，后者回退就是绕过流程裁决。
// bound 判据是业务键下存在过流程实例（任意状态），不能只看 running —— 实例被挂起或已结束时
// 同样查不到 running 行，若据此回退直批就等于让已交流程裁决的对象绕过流程。
type approvalTaskLookup struct {
	bound bool
	task  *ent.ProcessTask
}

func (b *BPMNApprovalBridge) findPendingApprovalTask(ctx context.Context, tenantID int, businessType string, businessID int) (approvalTaskLookup, error) {
	return b.findApprovalTaskWithClient(ctx, b.client, tenantID, businessType, businessID)
}

func (b *BPMNApprovalBridge) findPendingApprovalTaskWithClient(ctx context.Context, client *ent.Client, tenantID int, businessType string, businessID int) (approvalTaskLookup, error) {
	return b.findApprovalTaskWithClient(ctx, client, tenantID, businessType, businessID)
}

func (b *BPMNApprovalBridge) findApprovalTaskWithClient(ctx context.Context, client *ent.Client, tenantID int, businessType string, businessID int) (approvalTaskLookup, error) {
	businessKey := fmt.Sprintf("%s:%d", strings.ToLower(businessType), businessID)

	// 先确认该业务对象是否交过流程裁决，任意状态的实例都算绑定过。
	boundInstance, err := client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
		).
		Order(ent.Desc(processinstance.FieldStartTime)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return approvalTaskLookup{}, nil
		}
		return approvalTaskLookup{}, fmt.Errorf("查询业务关联流程实例失败: %w", err)
	}
	result := approvalTaskLookup{bound: true}

	if boundInstance.Status != "running" {
		b.logger.Warnw("业务审批桥接：流程实例非运行中，无待办可操作",
			"businessKey", businessKey, "processInstanceID", boundInstance.ID, "status", boundInstance.Status)
		return result, nil
	}

	task, err := client.ProcessTask.Query().
		Where(
			processtask.ProcessInstanceID(boundInstance.ID),
			processtask.TenantID(tenantID),
			processtask.TaskType("user_task"),
			processtask.StatusIn(
				common.ProcessTaskStatusCreated,
				common.ProcessTaskStatusAssigned,
				common.ProcessTaskStatusStarted,
				common.ProcessTaskStatusDelegated,
			),
		).
		Order(ent.Asc(processtask.FieldCreatedTime)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			b.logger.Warnw("业务审批桥接：流程实例无待办用户任务",
				"businessKey", businessKey, "processInstanceID", boundInstance.ID)
			return result, nil
		}
		return approvalTaskLookup{}, fmt.Errorf("查询流程待办任务失败: %w", err)
	}
	result.task = task
	return result, nil
}
