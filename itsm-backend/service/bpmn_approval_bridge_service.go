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
// 保证审批动作以 BPMN 任务为权威来源，流程实例不因业务侧直批而悬挂。
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

// businessApprovalTrustKey 只在本包内使用：标记「业务域已完成审批资格校验」的桥接调用。
//
// 背景：change / release 等老 BPMN 模板没有声明 taskPurpose 与候选人，引擎把
// triggered_by（=创建人）回落成独占 assignee；而业务审批资格来自另一套审批链。
// 两套身份互不相交时，审批人 403「已被他人认领」、创建人 403「不是待办审批人」，
// 审批永久卡死。业务域（change 审批链 / release 非创建人 / service_request
// canApprove）已经各自做了资格校验，因此只在这条受控的桥接路径上忽略「回落认领态」；
// 通用任务端点 PUT /workflow/tasks/{id}/complete 不带此标记，归属校验保持原样。
type businessApprovalTrustKey struct{}

func withBusinessApprovalTrust(ctx context.Context) context.Context {
	return context.WithValue(ctx, businessApprovalTrustKey{}, true)
}

func businessApprovalTrusted(ctx context.Context) bool {
	trusted, _ := ctx.Value(businessApprovalTrustKey{}).(bool)
	return trusted
}

// preApprovalEvaluationMarkers 是「审批之前的前置评估/评审」节点词根。
//
// 与 lifecycleTaskMarkers 的区别：后者覆盖整个生命周期（排期/实施/验证/关闭），
// 用于「审批绝不能落到这些节点」的判定；这里只圈定「提交动作已经隐含完成」的
// 那一小段——变更评估、技术评审、初审、合规检查。业务侧提交时这些内容已经被
// 门禁校验过（影响分析、实施方案、回滚计划、窗口冲突），流程里再留一个待办
// 只会让审批入口永远拿不到审批待办。
//
// 实施/验证/关闭不在此列：那些节点代表真实作业，替业务方完成等于伪造事实。
var preApprovalEvaluationMarkers = []string{
	"assessment", "evaluation", "techreview", "tech_review", "review", "intake",
	"评估", "评审", "初审", "复核", "检查",
}

// isPreApprovalEvaluationTask 判断待办是否是审批前置的评估/评审节点。
func isPreApprovalEvaluationTask(task *ent.ProcessTask) bool {
	if task == nil {
		return false
	}
	haystack := strings.ToLower(task.TaskDefinitionKey + " " + task.TaskName)
	for _, marker := range preApprovalEvaluationMarkers {
		if strings.Contains(haystack, marker) {
			return true
		}
	}
	return false
}

// inferredTaskPurpose 为没有声明 itsm:taskPurpose 的老模板推断节点用途。
// 显式声明始终优先；只有明确的作业/审批语义才推断，推断不出时返回空串，
// 由调用方维持「取最早待办」的既有行为，避免误伤其它流程模板。
func inferredTaskPurpose(task *ent.ProcessTask) string {
	if declared, ok := task.TaskVariables["taskPurpose"].(string); ok {
		if purpose := strings.ToLower(strings.TrimSpace(declared)); purpose != "" {
			return purpose
		}
	}
	haystack := strings.ToLower(task.TaskDefinitionKey + " " + task.TaskName)
	workMarkers := []string{
		"assessment", "review", "schedule", "implement", "verify", "close", "fulfill", "submit", "execute",
		"评估", "评审", "排期", "实施", "验证", "关闭", "履行", "提交", "执行",
	}
	for _, marker := range workMarkers {
		if strings.Contains(haystack, marker) {
			return "work"
		}
	}
	if strings.Contains(haystack, "approv") || strings.Contains(haystack, "审批") || strings.Contains(haystack, "cab") {
		return "approval"
	}
	return ""
}

// CompleteBusinessApprovalTaskWithClient 使用调用方事务绑定的 Ent client 完成审批任务。
// 它不创建或提交事务，任何失败都由外层事务连同业务状态一起回滚。
func (b *BPMNApprovalBridge) CompleteBusinessApprovalTaskWithClient(ctx context.Context, txc *ent.Client, tenantID, actorUserID int, businessType string, businessID int, action, comment string) (bool, error) {
	if tenantID <= 0 || actorUserID <= 0 {
		return false, common.NewBusinessError(common.UnauthorizedCode, "Missing approval identity or tenant context", "")
	}
	if businessID <= 0 || strings.TrimSpace(businessType) == "" || (action != "approve" && action != "reject") {
		return false, common.NewBusinessError(common.ParamErrorCode, "Invalid approval parameters", "")
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
	if isLifecycleTask(task) {
		return false, bpmnApprovalNotActionable(businessType, businessID)
	}

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
	workflowCtx = withBusinessApprovalTrust(workflowCtx)

	engine := NewCustomProcessEngine(txc, b.logger)
	if customEngine, ok := engine.(*CustomProcessEngine); ok {
		if err := customEngine.CompleteTaskWithClient(workflowCtx, txc, task.TaskID, variables); err != nil {
			return false, fmt.Errorf("Failed to complete process approval task: %w", err)
		}
	} else {
		if err := engine.CompleteTask(workflowCtx, task.TaskID, variables); err != nil {
			return false, fmt.Errorf("Failed to complete process approval task: %w", err)
		}
	}

	b.logger.Infow("业务审批已桥接完成BPMN任务(事务内)",
		"businessType", businessType, "businessId", businessID, "taskId", task.TaskID, "action", action, "actorUserId", actorUserID)
	return true, nil
}

// CheckActionableApprovalTask 只读确认业务对象此刻是否停在可操作的审批待办上，
// 不消费待办；会签需要先确认待办存在，但只有凑齐 quorum 的那一票才完成它。
// 返回 bound=false 表示该业务对象从未绑定流程（或实例不属于本租户），由调用方处置；
// 已绑定却拿不到可操作待办时返回 conflict 错误。
func (b *BPMNApprovalBridge) CheckActionableApprovalTask(ctx context.Context, tenantID int, businessType string, businessID int) (bool, error) {
	lookup, err := b.findPendingApprovalTaskWithClient(ctx, b.client, tenantID, businessType, businessID)
	if err != nil {
		return false, err
	}
	if !lookup.bound {
		return false, nil
	}
	if lookup.task == nil || isLifecycleTask(lookup.task) {
		return true, bpmnApprovalNotActionable(businessType, businessID)
	}
	return true, nil
}

// CompleteBusinessApprovalTask 把业务审批桥接到流程待办；三态语义见类型注释。
func (b *BPMNApprovalBridge) CompleteBusinessApprovalTask(ctx context.Context, tenantID, actorUserID int, businessType string, businessID int, action, comment string) (bool, error) {
	if tenantID <= 0 || actorUserID <= 0 {
		return false, common.NewBusinessError(common.UnauthorizedCode, "Missing approval identity or tenant context", "")
	}
	if businessID <= 0 || strings.TrimSpace(businessType) == "" || (action != "approve" && action != "reject") {
		return false, common.NewBusinessError(common.ParamErrorCode, "Invalid approval parameters", "")
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
	if isLifecycleTask(task) {
		return false, bpmnApprovalNotActionable(businessType, businessID)
	}

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
	workflowCtx = withBusinessApprovalTrust(workflowCtx)

	engine := NewCustomProcessEngine(b.client, b.logger)
	if err := engine.CompleteTask(workflowCtx, task.TaskID, variables); err != nil {
		return false, fmt.Errorf("Failed to complete process approval task: %w", err)
	}

	b.logger.Infow("业务审批已桥接完成BPMN任务",
		"businessType", businessType, "businessId", businessID, "taskId", task.TaskID, "action", action, "actorUserId", actorUserID)
	return true, nil
}

func (b *BPMNApprovalBridge) DelegateBusinessApprovalTask(ctx context.Context, tenantID, actorUserID int, businessType string, businessID, newAssigneeUserID int) (bool, error) {
	if tenantID <= 0 || actorUserID <= 0 {
		return false, common.NewBusinessError(common.UnauthorizedCode, "Missing approval identity or tenant context", "")
	}
	if businessID <= 0 || newAssigneeUserID <= 0 || strings.TrimSpace(businessType) == "" {
		return false, common.NewBusinessError(common.ParamErrorCode, "Invalid delegation parameter", "")
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
	if isLifecycleTask(task) {
		return false, bpmnApprovalNotActionable(businessType, businessID)
	}

	// 注入认证操作人与租户，委派前校验操作人必须是当前任务的审批人/候选人，防止越权改派
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, actorUserID)

	engine := NewCustomProcessEngine(b.client, b.logger)
	if customEngine, ok := engine.(*CustomProcessEngine); ok {
		if err := customEngine.authorizeTaskActor(workflowCtx, task); err != nil {
			return false, fmt.Errorf("Failed to delegate process task: %w", err)
		}
	}
	if err := engine.TaskService().DelegateTask(workflowCtx, task.TaskID, strconv.Itoa(newAssigneeUserID)); err != nil {
		return false, fmt.Errorf("Failed to delegate process task: %w", err)
	}

	b.logger.Infow("业务审批委派已同步BPMN任务",
		"businessType", businessType, "businessId", businessID, "taskId", task.TaskID,
		"actorUserId", actorUserID, "newAssigneeUserId", newAssigneeUserID)
	return true, nil
}

// TerminateBusinessWorkflow 终止业务对象关联的运行中/暂停流程实例。
// 业务侧取消已提交后调用：没有实例、或实例已结束都视为幂等成功。
func (b *BPMNApprovalBridge) TerminateBusinessWorkflow(ctx context.Context, tenantID, actorUserID int, businessType string, businessID int, reason string) error {
	if tenantID <= 0 || businessID <= 0 || strings.TrimSpace(businessType) == "" {
		return nil
	}
	businessKey := fmt.Sprintf("%s:%d", strings.ToLower(businessType), businessID)
	instances, err := b.client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
			processinstance.StatusIn("running", "suspended"),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("Failed to query business association process instance: %w", err)
	}
	if len(instances) == 0 {
		return nil
	}

	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, actorUserID)
	engine := NewCustomProcessEngine(b.client, b.logger)
	for _, instance := range instances {
		if err := engine.TerminateProcess(workflowCtx, instance.ProcessInstanceID, reason); err != nil {
			return fmt.Errorf("Termination of process instance %s failed: %w", instance.ProcessInstanceID, err)
		}
		b.logger.Infow("业务流程已终止",
			"businessKey", businessKey, "processInstanceID", instance.ProcessInstanceID, "reason", reason)
	}
	return nil
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
		lookup, err := b.findFirstPendingTaskWithClient(ctx, client, tenantID, businessType, businessID)
		if err != nil {
			return handled, err
		}
		if !lookup.bound || lookup.task == nil {
			return handled, nil
		}
		task := lookup.task

		workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
		workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, actorUserID)
		// 作业节点同样可能因「triggered_by 回落」被认领给创建人；推进作业不是审批签名，
		// 带受控信任标记让非创建人的提交/审批动作也能越过前置作业节点。
		workflowCtx = withBusinessApprovalTrust(workflowCtx)

		engine := NewCustomProcessEngine(client, b.logger)
		var completeErr error
		if customEngine, ok := engine.(*CustomProcessEngine); reuseTransaction && ok {
			completeErr = customEngine.CompleteTaskWithClient(workflowCtx, client, task.TaskID, variables)
		} else {
			completeErr = engine.CompleteTask(workflowCtx, task.TaskID, variables)
		}
		if completeErr != nil {
			return handled, fmt.Errorf("Advance process task failed: %w", completeErr)
		}
		handled = true
	}

	return handled, nil
}

// CompleteBusinessWorkTasks 依次完成业务对象当前待办中的作业任务（taskPurpose=work），
// 用于交付完成后关闭模板里的「执行服务」「用户确认」等履约节点。
//
// 与审批桥接的边界：这里只认显式声明 purpose=work 的待办；当前停在审批节点时
// 直接返回冲突，绝不代替审批人签名，否则交付结果会被写成审批决定。
// 推进使用系统上下文：交付 worker 没有对应业务操作人，但仍保留 actorUserID 供审计。
//
// 返回值为实际完成的作业待办数量；0 表示没有可推进的作业节点（未绑定流程或已结束）。
func (b *BPMNApprovalBridge) CompleteBusinessWorkTasks(ctx context.Context, tenantID, actorUserID int, businessType string, businessID int, variables map[string]interface{}, maxSteps int) (int, error) {
	if tenantID <= 0 || businessID <= 0 || strings.TrimSpace(businessType) == "" || maxSteps <= 0 {
		return 0, nil
	}
	businessKey := fmt.Sprintf("%s:%d", strings.ToLower(businessType), businessID)

	completed := 0
	for step := 0; step < maxSteps; step++ {
		instance, err := b.client.ProcessInstance.Query().
			Where(
				processinstance.BusinessKey(businessKey),
				processinstance.TenantID(tenantID),
				processinstance.Status("running"),
			).
			Order(ent.Desc(processinstance.FieldStartTime)).
			First(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return completed, nil
			}
			return completed, fmt.Errorf("Failed to query business association process instance: %w", err)
		}

		tasks, err := b.client.ProcessTask.Query().
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
			Order(ent.Asc(processtask.FieldCreatedTime)).
			All(ctx)
		if err != nil {
			return completed, fmt.Errorf("Failed to query process to-do tasks: %w", err)
		}
		if len(tasks) == 0 {
			return completed, nil
		}

		var workTask *ent.ProcessTask
		for _, candidate := range tasks {
			// 老模板（change/release）没有声明 taskPurpose，用节点语义兜底，
			// 否则交付收尾时永远找不到可推进的作业待办。
			if inferredTaskPurpose(candidate) == "work" {
				workTask = candidate
				break
			}
		}
		if workTask == nil {
			return completed, bpmnWorkTaskNotActionable(businessType, businessID)
		}

		// 只放开归属校验，租户上下文原样透传，RLS 仍按调用方租户过滤。
		workflowCtx := bpmn.WithBPMNSystemAdvance(ctx)
		workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNTenantIDContextKey, tenantID)
		if actorUserID > 0 {
			workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, actorUserID)
		}

		engine := NewCustomProcessEngine(b.client, b.logger)
		if err := engine.CompleteTask(workflowCtx, workTask.TaskID, variables); err != nil {
			return completed, fmt.Errorf("Failed to complete process job task: %w", err)
		}
		completed++
		b.logger.Infow("业务作业已推进BPMN任务",
			"businessKey", businessKey, "taskId", workTask.TaskID, "actorUserId", actorUserID)
	}
	return completed, nil
}

// AdvanceBusinessWorkTaskWithClient 在调用方事务内推进业务对象当前待办中的作业节点。
//
// 用途：变更/发布的 BPMN 模板把「变更评估 / 技术评审」放在审批节点之前，
// 而产品没有独立的「完成评估」入口，业务提交动作就该把评估一并带过去。
// 旧流程停在评估节点时，审批入口永远拿不到审批待办（恒 409）。
//
// 与审批桥接的硬边界：只推进 purpose=work 的节点，抵达审批节点立刻停下。
// 绝不代签审批——否则提交动作就等于自动通过审批。
//
// 返回推进的节点数；0 表示当前没有可推进的作业节点（未绑定流程 / 已到审批节点 / 已结束）。
func (b *BPMNApprovalBridge) AdvanceBusinessWorkTaskWithClient(ctx context.Context, txc *ent.Client, tenantID, actorUserID int, businessType string, businessID int, variables map[string]interface{}) (int, error) {
	if tenantID <= 0 || businessID <= 0 || strings.TrimSpace(businessType) == "" {
		return 0, nil
	}
	businessKey := fmt.Sprintf("%s:%d", strings.ToLower(businessType), businessID)

	instance, err := txc.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
			processinstance.Status("running"),
		).
		Order(ent.Desc(processinstance.FieldStartTime)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("Failed to query business association process instance: %w", err)
	}

	tasks, err := txc.ProcessTask.Query().
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
		Order(ent.Asc(processtask.FieldCreatedTime)).
		All(ctx)
	if err != nil {
		return 0, fmt.Errorf("Failed to query process to-do tasks: %w", err)
	}

	var workTask *ent.ProcessTask
	for _, candidate := range tasks {
		// 只推进「审批前置的评估/评审」节点：这类节点的内容已被业务门禁校验过，
		// 提交动作就等于把它带过去。实施/验证/关闭等真实作业节点绝不在此列。
		if isPreApprovalEvaluationTask(candidate) {
			workTask = candidate
			break
		}
	}
	if workTask == nil {
		return 0, nil
	}

	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, actorUserID)
	workflowCtx = withBusinessApprovalTrust(workflowCtx)

	engine := NewCustomProcessEngine(txc, b.logger)
	if customEngine, ok := engine.(*CustomProcessEngine); ok {
		if completeErr := customEngine.CompleteTaskWithClient(workflowCtx, txc, workTask.TaskID, variables); completeErr != nil {
			return 0, fmt.Errorf("Failed to advance process job backlog: %w", completeErr)
		}
	} else if completeErr := engine.CompleteTask(workflowCtx, workTask.TaskID, variables); completeErr != nil {
		return 0, fmt.Errorf("Failed to advance process job backlog: %w", completeErr)
	}

	b.logger.Infow("业务前置作业待办已推进",
		"businessKey", businessKey, "taskId", workTask.TaskID, "actorUserId", actorUserID)
	return 1, nil
}

// bpmnWorkTaskNotActionable 表示业务已到履约收尾，但流程停在了审批或其他非作业节点。
// 此时代替完成会把交付结果写进审批审计，必须失败关闭并交人工核对流程实例。
func bpmnWorkTaskNotActionable(businessType string, businessID int) error {
	return common.NewBusinessError(
		common.ConflictCode,
		"The current to-do process is not a job node, and advancement is refused.",
		fmt.Sprintf("businessType=%s businessId=%d", strings.ToLower(businessType), businessID),
	)
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
		return false, fmt.Errorf("Failed to query business association process instance: %w", err)
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
		return false, fmt.Errorf("Failed to query process to-do tasks: %w", err)
	}
	return exists, nil
}

// bpmnApprovalNotActionable 表示流程已绑定却拿不到可操作待办：待办可能已被并发消费，
// 或流程停在别的节点上。此时业务侧直批会让流程状态与业务状态分叉，必须失败关闭。
func bpmnApprovalNotActionable(businessType string, businessID int) error {
	return common.NewBusinessError(
		common.ConflictCode,
		"The process has been bound but there is currently no approval to-do that can be processed. Please do not approve the business directly. If the process stops at an operation node such as \"Change Assessment/Technical Review\", please go to the approval center to complete the operation to-do first, and then approve the process after the process has advanced to the approval node.",
		fmt.Sprintf("businessType=%s businessId=%d", strings.ToLower(businessType), businessID),
	)
}

// lifecycleTaskMarkers 是内置模板中「确定不是审批」的生命周期节点词根（小写）。
// 审批桥接遇到这些节点时 fail-closed：审批动作不得代替排期/实施/验证等生命周期
// 步骤，也不得在错误节点上写入审批决定（recordApprovalDecision 只要 variables 带
// approvalAction 就落库，误完成会污染审计）。清单刻意收窄到确定项——流程待办
// 未命中时保持原行为，避免误伤「变更经理初审」「请求审核」这类审核类任务。
var lifecycleTaskMarkers = []string{
	"assessment", "schedule", "implement", "verify", "close", "fulfill", "submit",
	"评估", "排期", "实施", "验证", "关闭", "履行", "提交",
}

func isLifecycleTask(task *ent.ProcessTask) bool {
	haystack := strings.ToLower(task.TaskDefinitionKey + " " + task.TaskName)
	for _, marker := range lifecycleTaskMarkers {
		if strings.Contains(haystack, marker) {
			return true
		}
	}
	return false
}

// approvalTaskLookup 是按业务键反查流程待办的结果，必须区分「从未绑定」与「绑定过但此刻不可操作」：
// 前者调用方回退旧审批链，后者回退就是绕过流程裁决。
// bound 判据是业务键下存在过流程实例（任意状态），不能只看 running —— 实例被挂起或已结束时
// 同样查不到 running 行，若据此回退直批就等于让已交流程裁决的对象绕过流程。
type approvalTaskLookup struct {
	bound bool
	task  *ent.ProcessTask
	// workTask 记录流程当前停在作业节点（如变更评估）的待办，
	// 供调用方生成「请先完成评估」这类可行动指引，而不是只报一句 409。
	workTask *ent.ProcessTask
}

func (b *BPMNApprovalBridge) findPendingApprovalTask(ctx context.Context, tenantID int, businessType string, businessID int) (approvalTaskLookup, error) {
	return b.findApprovalTaskWithClient(ctx, b.client, tenantID, businessType, businessID)
}

func (b *BPMNApprovalBridge) findPendingApprovalTaskWithClient(ctx context.Context, client *ent.Client, tenantID int, businessType string, businessID int) (approvalTaskLookup, error) {
	return b.findApprovalTaskWithClient(ctx, client, tenantID, businessType, businessID)
}

// findFirstPendingTaskWithClient 返回业务对象当前最早的待办用户任务（不分用途）。
//
// 与 findApprovalTaskWithClient 的区别：后者只认审批节点（审批动作绝不能落到
// 评估/排期/实施这类作业节点上）；本函数用于「推进当前节点」语义——
// 事件确认、问题关闭、变更提交都需要把流程推进一步，推的正是当前那个待办，
// 哪怕它的 taskPurpose 是 approval（老模板里事件/问题的首个待办就是审批节点）。
func (b *BPMNApprovalBridge) findFirstPendingTaskWithClient(ctx context.Context, client *ent.Client, tenantID int, businessType string, businessID int) (approvalTaskLookup, error) {
	businessKey := fmt.Sprintf("%s:%d", strings.ToLower(businessType), businessID)

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
		return approvalTaskLookup{}, fmt.Errorf("Failed to query business association process instance: %w", err)
	}
	result := approvalTaskLookup{bound: true}
	if boundInstance.Status != "running" {
		return result, nil
	}

	tasks, err := client.ProcessTask.Query().
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
		All(ctx)
	if err != nil {
		return approvalTaskLookup{}, fmt.Errorf("Failed to query process to-do tasks: %w", err)
	}
	if len(tasks) == 0 {
		return result, nil
	}
	result.task = tasks[0]
	return result, nil
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
		return approvalTaskLookup{}, fmt.Errorf("Failed to query business association process instance: %w", err)
	}
	result := approvalTaskLookup{bound: true}

	if boundInstance.Status != "running" {
		b.logger.Warnw("业务审批桥接：流程实例非运行中，无待办可操作",
			"businessKey", businessKey, "processInstanceID", boundInstance.ID, "status", boundInstance.Status)
		return result, nil
	}

	tasks, err := client.ProcessTask.Query().
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
		All(ctx)
	if err != nil {
		return approvalTaskLookup{}, fmt.Errorf("Failed to query process to-do tasks: %w", err)
	}
	if len(tasks) == 0 {
		b.logger.Warnw("业务审批桥接：流程实例无待办用户任务",
			"businessKey", businessKey, "processInstanceID", boundInstance.ID)
		return result, nil
	}
	// 优先选审批节点；老模板没有声明 taskPurpose 时按节点语义推断，
	// 避免把「变更评估 / 技术评审」这类生命周期任务当成审批待办，
	// 从而让业务审批恒 409 或把审批意见写进错误的节点。
	// 完全推断不出用途时才退回最早待办，保持其它模板的既有行为不变。
	var task *ent.ProcessTask
	purposeDeclared := false
	for _, candidate := range tasks {
		purpose := inferredTaskPurpose(candidate)
		if purpose == "" {
			continue
		}
		purposeDeclared = true
		switch purpose {
		case "approval":
			task = candidate
		case "work":
			if result.workTask == nil {
				result.workTask = candidate
			}
		}
		if task != nil {
			break
		}
	}
	if task == nil {
		if purposeDeclared {
			b.logger.Warnw("业务审批桥接：当前待办均非审批节点",
				"businessKey", businessKey, "processInstanceID", boundInstance.ID,
				"hasPendingWorkTask", result.workTask != nil)
			return result, nil
		}
		task = tasks[0]
	}
	result.task = task
	return result, nil
}
