package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
	"itsm-backend/ent/ticketassignmentrule"
	"itsm-backend/ent/user"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service/bpmn"
)

// ---------------------------------------------------------------------------
// bpmn_process_executor.go — 流程执行器（核心状态机）
//
// 职责：流程实例启动、步骤推进（executeStep）、元素分发（handleElement）、
// 网关路由（parallel/inclusive/exclusive）、UserTask 创建与完成、
// ServiceTask 命令入队、条件评估、图导航辅助。
//
// 这是 BPMN 引擎的核心文件。社区贡献者只需理解此文件即可掌握流程如何
// 从 StartEvent 推进到 EndEvent。
// ---------------------------------------------------------------------------

// StartProcess 启动流程实例
func (e *CustomProcessEngine) StartProcess(ctx context.Context, processDefinitionKey string, businessKey string, variables map[string]interface{}) (*ent.ProcessInstance, error) {
	// P1-4：租户上下文必须显式有效（>0），禁止缺上下文时全局查流程定义。
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 获取流程定义（固定使用当前租户过滤，不再存在"无过滤跨租户"分支）
	definition, err := e.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(processDefinitionKey),
			processdefinition.IsActive(true),
			processdefinition.IsLatest(true),
			processdefinition.TenantID(tenantID),
		).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	// 3. 解析BPMN
	bpmnDefinitions, err := e.parser.ParseXML(definition.BpmnXML)
	if err != nil {
		return nil, fmt.Errorf("解析BPMN失败: %w", err)
	}

	if len(bpmnDefinitions.Processes) == 0 {
		return nil, fmt.Errorf("BPMN中未找到流程定义")
	}
	process := bpmnDefinitions.Processes[0]

	// 3. 找到开始事件
	if len(process.StartEvents) == 0 {
		return nil, fmt.Errorf("流程缺少开始事件")
	}
	startEvent := process.StartEvents[0]

	tx, err := e.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启流程启动事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txc := tx.Client()

	// 4. 创建流程实例
	instance, err := txc.ProcessInstance.Create().
		SetProcessInstanceID(fmt.Sprintf("PI-%s-%d", processDefinitionKey, time.Now().UnixNano())).
		SetBusinessKey(businessKey).
		SetProcessDefinitionKey(processDefinitionKey).
		SetProcessDefinitionID(definition.ID).
		SetStatus("running").
		SetVariables(variables).
		SetStartTime(time.Now()).
		SetTenantID(definition.TenantID).
		SetCurrentActivityID(startEvent.ID).
		SetCurrentActivityName(startEvent.Name).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建流程实例失败: %w", err)
	}

	// 5. 执行流程推进（从StartEvent开始）
	if err := e.executeStep(ctx, txc, instance, process, startEvent.ID, variables); err != nil {
		return nil, err
	}

	// 6. 记录审计日志 - 流程启动
	// 从context中获取用户信息
	userID := 0
	userName := ""
	if u, ok := ctx.Value("user").(*ent.User); ok {
		userID = u.ID
		userName = u.Name
	}
	if err := NewBPMNAuditService(txc, e.logger).RecordProcessStarted(ctx, instance, userID, userName, variables); err != nil {
		return nil, fmt.Errorf("记录流程启动审计失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交流程启动事务失败: %w", err)
	}
	return instance.Unwrap(), nil
}

// CompleteTask 完成任务。
// 整个「置完成 → 合并变量 → 推进流程 → 记录审批决策」包进单个 ent.Tx，
// 任一步骤失败整体回滚，彻底消除并发下的半成品状态（P2 事务原子化）。
func (e *CustomProcessEngine) CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error {
	// P1-4：租户上下文必须显式有效（>0），禁止缺上下文时全局查任务
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}

	tx, err := e.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	txc := tx.Client()
	// 异常时回滚，保证不会留下半提交状态
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	task, err := e.completeTaskWithClient(ctx, txc, tenantID, taskID, variables)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	// 7. 提交事务；任一步骤失败已在上方回滚
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 8. 记录审计日志 - 任务完成（事务外，仅审计，失败不阻断）
	e.recordTaskCompletedAudit(ctx, task, variables)
	return nil
}

// CompleteTaskWithClient 在调用方已经建立的数据库事务中完成任务。
// txc 必须绑定到该事务；本方法不提交、回滚，也不写事务外审计，由事务所有者决定最终结果。
func (e *CustomProcessEngine) CompleteTaskWithClient(ctx context.Context, txc *ent.Client, taskID string, variables map[string]interface{}) error {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}
	_, err = e.completeTaskWithClient(ctx, txc, tenantID, taskID, variables)
	return err
}

func (e *CustomProcessEngine) completeTaskWithClient(ctx context.Context, txc *ent.Client, tenantID int, taskID string, variables map[string]interface{}) (*ent.ProcessTask, error) {
	// 1. 获取任务（固定按当前租户过滤，不存在"无过滤跨租户"分支）
	task, err := txc.ProcessTask.Query().
		Where(processtask.TaskID(taskID), processtask.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %w", err)
	}
	if err := e.authorizeTaskActorWithClient(ctx, txc, task); err != nil {
		return nil, err
	}

	// 2. 获取流程实例 - 使用任务中存储的ProcessInstanceID (ent自动生成的ID)
	instance, err := txc.ProcessInstance.Get(ctx, task.ProcessInstanceID)
	if err != nil {
		return nil, fmt.Errorf("获取流程实例失败: %w", err)
	}

	// 3. 获取流程定义并解析
	// A running instance is immutable with respect to its deployed definition.
	// Looking up the latest definition here can silently move an old instance
	// onto a newly published graph halfway through execution.
	definition, err := txc.ProcessDefinition.Query().
		Where(
			processdefinition.ID(instance.ProcessDefinitionID),
			processdefinition.TenantID(instance.TenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	bpmnDefinitions, err := e.parser.ParseXML(definition.BpmnXML)
	if err != nil {
		return nil, fmt.Errorf("解析BPMN失败: %w", err)
	}
	process := bpmnDefinitions.Processes[0]

	// 4. 更新当前任务状态
	if task.Status == "completed" || task.Status == "cancelled" {
		return nil, fmt.Errorf("任务已结束，不能重复完成")
	}

	updated, err := txc.ProcessTask.Update().
		Where(
			processtask.ID(task.ID),
			processtask.TenantID(instance.TenantID),
			processtask.StatusNEQ("completed"),
			processtask.StatusNEQ("cancelled"),
		).
		SetStatus("completed").
		SetCompletedTime(time.Now()).
		SetTaskVariables(variables).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新任务状态失败: %w", err)
	}
	if updated != 1 {
		return nil, fmt.Errorf("任务已被描述，请刷新后重试")
	}

	e.cancelBoundaryTimers(ctx, instance, process, task.TaskDefinitionKey)
	e.cancelTaskDueTimers(ctx, task.TenantID, task.TaskID)

	// 5. 在事务内合并变量（无并发写者，直接合并即可）
	instance, err = e.mergeVariablesInTx(ctx, txc, instance.ID, variables)
	if err != nil {
		return nil, fmt.Errorf("合并实例变量失败: %w", err)
	}

	// 6. 执行流程推进（从当前UserTask继续）。审批拒绝策略属于节点
	// 运行语义，不能只停留在设计器配置中。
	rejectStrategy, _ := task.TaskVariables["rejectStrategy"].(string)
	approvalAction, _ := variables["approvalAction"].(string)
	commentRequired, _ := task.TaskVariables["commentRequiredOnReject"].(bool)

	if approvalAction == "reject" {
		if commentRequired && strings.TrimSpace(variables["approvalComment"].(string)) == "" {
			return nil, fmt.Errorf("该审批节点要求拒绝时必须填写意见")
		}
		switch strings.ToLower(strings.TrimSpace(rejectStrategy)) {
		case "terminate":
			if _, err = txc.ProcessInstance.UpdateOneID(instance.ID).
				SetStatus("terminated").SetEndTime(time.Now()).Save(ctx); err != nil {
				return nil, fmt.Errorf("终止被拒绝流程失败: %w", err)
			}
			if _, err = txc.ProcessTask.Update().Where(
				processtask.ProcessInstanceID(instance.ID), processtask.TenantID(instance.TenantID),
				processtask.StatusNEQ("completed"), processtask.StatusNEQ("cancelled"),
			).SetStatus("cancelled").SetCompletedTime(time.Now()).Save(ctx); err != nil {
				return nil, fmt.Errorf("取消被拒绝流程的剩余任务失败: %w", err)
			}
		case "to_requester":
			if _, err = txc.ProcessTask.Update().Where(
				processtask.ProcessInstanceID(instance.ID), processtask.TenantID(instance.TenantID),
				processtask.StatusNEQ("completed"), processtask.StatusNEQ("cancelled"),
				processtask.IDNEQ(task.ID),
			).SetStatus("cancelled").SetCompletedTime(time.Now()).Save(ctx); err != nil {
				return nil, fmt.Errorf("退回发起人时取消其他活动任务失败: %w", err)
			}
			variables["rework_requested"] = true
			variables["rework_requested_by"] = task.Assignee
			if _, err = e.mergeVariablesInTx(ctx, txc, instance.ID, variables); err != nil {
				return nil, fmt.Errorf("写入退回变量失败: %w", err)
			}
			if err := e.executeStep(ctx, txc, instance, process, task.TaskDefinitionKey, instance.Variables); err != nil {
				return nil, err
			}
		case "gateway":
			if _, err = txc.ProcessTask.Update().Where(
				processtask.ProcessInstanceID(instance.ID), processtask.TenantID(instance.TenantID),
				processtask.StatusNEQ("completed"), processtask.StatusNEQ("cancelled"),
				processtask.IDNEQ(task.ID),
			).SetStatus("cancelled").SetCompletedTime(time.Now()).Save(ctx); err != nil {
				return nil, fmt.Errorf("拒绝分支取消其他活动任务失败: %w", err)
			}
			if err := e.executeStep(ctx, txc, instance, process, task.TaskDefinitionKey, instance.Variables); err != nil {
				return nil, err
			}
		default:
			if err := e.executeStep(ctx, txc, instance, process, task.TaskDefinitionKey, instance.Variables); err != nil {
				return nil, err
			}
		}
	} else if err := e.executeStep(ctx, txc, instance, process, task.TaskDefinitionKey, instance.Variables); err != nil {
		return nil, err
	}
	if err := e.recordApprovalDecision(ctx, txc, instance, task, variables); err != nil {
		return nil, err
	}
	return task, nil
}

func (e *CustomProcessEngine) recordTaskCompletedAudit(ctx context.Context, task *ent.ProcessTask, variables map[string]interface{}) {
	userID := 0
	userName := ""
	if u, ok := ctx.Value("user").(*ent.User); ok {
		userID = u.ID
		userName = u.Name
	}
	variablesBefore := task.TaskVariables
	if err := e.auditService.RecordTaskCompleted(ctx, task, userID, userName, variablesBefore, variables); err != nil {
		e.logger.Warnw("audit record failed", "error", err)
	}
}

// mergeVariablesInTx 在事务内合并流程实例变量并返回更新后的实例。
// 与 mergeVariablesWithOptimisticLock 不同：调用方已持有事务，无需再开事务或重试。
func (e *CustomProcessEngine) mergeVariablesInTx(ctx context.Context, txc *ent.Client, instanceID int, newVars map[string]interface{}) (*ent.ProcessInstance, error) {
	inst, err := txc.ProcessInstance.Get(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("查询流程实例失败: %w", err)
	}
	merged := make(map[string]interface{})
	for k, v := range inst.Variables {
		merged[k] = v
	}
	for k, v := range newVars {
		merged[k] = v
	}
	updated, err := txc.ProcessInstance.UpdateOneID(instanceID).
		SetVariables(merged).
		SetVersion(inst.Version + 1).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新实例变量失败: %w", err)
	}
	return updated, nil
}

func (e *CustomProcessEngine) recordApprovalDecision(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, task *ent.ProcessTask, variables map[string]interface{}) error {
	action, _ := variables["approvalAction"].(string)
	if action == "" {
		return nil
	}
	decision, _ := variables["approvalResult"].(string)
	comment, _ := variables["approvalComment"].(string)
	actorID, _ := ctx.Value(bpmn.BPMNUserIDContextKey).(int)
	if actorID <= 0 {
		return fmt.Errorf("审批决策缺少认证操作人")
	}
	actorName := ""
	if actor, err := txc.User.Get(ctx, actorID); err == nil {
		actorName = actor.Name
	}
	businessType := fmt.Sprint(instance.Variables["business_type"])
	businessID := fmt.Sprint(instance.Variables["business_id"])
	_, err := txc.ProcessApprovalDecision.Create().
		SetProcessInstanceID(instance.ID).SetProcessTaskID(task.ID).
		SetProcessInstanceKey(instance.ProcessInstanceID).SetTaskID(task.TaskID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).SetNodeKey(task.TaskDefinitionKey).
		SetBusinessType(businessType).SetBusinessID(businessID).
		SetActorID(actorID).SetActorName(actorName).SetAction(action).SetDecision(decision).
		SetComment(comment).SetVariablesSnapshot(variables).SetTenantID(instance.TenantID).Save(ctx)
	if err != nil {
		return fmt.Errorf("记录审批决策失败: %w", err)
	}
	return nil
}

// authorizeTaskActor ensures that task actions are performed by the assigned
// user or an explicitly resolved candidate. System/internal calls without an
// authenticated actor keep their existing behavior.
func (e *CustomProcessEngine) authorizeTaskActor(ctx context.Context, task *ent.ProcessTask) error {
	return e.authorizeTaskActorWithClient(ctx, e.client, task)
}

func (e *CustomProcessEngine) authorizeTaskActorWithClient(ctx context.Context, client *ent.Client, task *ent.ProcessTask) error {
	userID, _ := ctx.Value(bpmn.BPMNUserIDContextKey).(int)
	if userID <= 0 {
		return nil
	}
	actor, err := client.User.Query().Where(user.ID(userID)).Only(ctx)
	if err != nil {
		return fmt.Errorf("审批用户不存在: %w", err)
	}
	allowed := func(csv string) bool {
		for _, candidate := range strings.Split(csv, ",") {
			candidate = strings.TrimSpace(candidate)
			if candidate == strconv.Itoa(userID) || candidate == actor.Username {
				return true
			}
		}
		return false
	}
	if allowed(task.Assignee) || allowed(task.CandidateUsers) {
		return nil
	}
	return fmt.Errorf("当前用户不是该任务的审批人或候选人")
}

// ---------------------------------------------------------------------------
// 核心状态机：executeStep → handleElement → 元素处理
// ---------------------------------------------------------------------------

// executeStep 执行流程步骤
func (e *CustomProcessEngine) executeStep(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, process *BPMNProcess, currentElementID string, variables map[string]interface{}) error {
	outgoingFlows := e.findOutgoingFlows(process, currentElementID)

	if len(outgoingFlows) == 0 {
		if e.isEndEvent(process, currentElementID) {
			return e.completeProcess(ctx, txc, instance)
		}
		// 防静默卡死（2026-09-07 内置模板测试 P0-1）：
		// 0 出边且非结束事件时，先尝试用元素声明的 <bpmn:outgoing> 做一次
		// fallback 匹配（声明与 sourceRef 不一致的模板历史遗留较多）；
		// 仍无可走则将实例显式挂起并返回错误，而不是 return nil 静默成功。
		if flowID, ok := firstOutgoingDeclaration(process, currentElementID); ok {
			if flow := e.findSequenceFlow(process, flowID); flow != nil && flow.TargetRef != "" {
				e.logger.Warnw("executeStep: sourceRef 无出边，使用 outgoing 声明 fallback",
					"instance", instance.ID, "element", currentElementID, "declaredFlow", flowID)
				return e.handleElement(ctx, txc, instance, process, flow.TargetRef)
			}
		}
		return e.suspendOnBrokenGraph(ctx, txc, instance, currentElementID,
			fmt.Sprintf("节点 %s 无可用出边且非结束事件（sourceRef 与 outgoing 声明均未命中），实例已挂起", currentElementID))
	}

	var targetRef string
	for _, flow := range outgoingFlows {
		if e.evaluateCondition(flow, variables) {
			targetRef = flow.TargetRef
			break
		}
	}

	if targetRef == "" {
		return fmt.Errorf("没有符合条件的路径")
	}

	// 记录排他网关路由决策，使「走哪条分支」可审计（F-4）
	if e.findExclusiveGateway(process, currentElementID) != nil {
		e.recordGatewayHistory(ctx, txc, instance, currentElementID, "exclusive", "fork", []string{targetRef}, variables)
	}

	return e.handleElement(ctx, txc, instance, process, targetRef)
}

func (e *CustomProcessEngine) handleElement(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, process *BPMNProcess, elementID string) error {
	// Find the element name for logging
	elementName := elementID
	if task := e.findUserTask(process, elementID); task != nil {
		elementName = task.Name
	} else if endEvent := e.findEndEvent(process, elementID); endEvent != nil {
		elementName = endEvent.Name
	}

	_, err := txc.ProcessInstance.UpdateOne(instance).
		SetCurrentActivityID(elementID).
		SetCurrentActivityName(elementName).
		Save(ctx)
	if err != nil {
		return err
	}

	// Debug: log element info
	e.logger.Debugw("handleElement called", "elementID", elementID, "elementName", elementName, "userTasksCount", len(process.UserTasks))

	if task := e.findUserTask(process, elementID); task != nil {
		e.logger.Infow("Found user task, creating task", "taskID", task.ID, "taskName", task.Name)
		return e.createUserTask(ctx, txc, instance, process, task)
	} else if endEvent := e.findEndEvent(process, elementID); endEvent != nil {
		e.markElementDone(ctx, txc, instance, elementID)
		return e.completeProcess(ctx, txc, instance)
	} else if gateway := e.findParallelGateway(process, elementID); gateway != nil {
		// 并行网关：分叉激活所有出边；汇聚等待所有入边分支完成（F-1）
		return e.handleParallelGateway(ctx, txc, instance, process, gateway, 0)
	} else if gateway := e.findInclusiveGateway(process, elementID); gateway != nil {
		// 包容网关：分叉激活所有命中条件的出边；汇聚等待所有入边分支完成（F-1）
		return e.handleInclusiveGateway(ctx, txc, instance, process, gateway, 0)
	} else if gateway := e.findExclusiveGateway(process, elementID); gateway != nil {
		e.markElementDone(ctx, txc, instance, elementID)
		return e.executeStep(ctx, txc, instance, process, elementID, instance.Variables)
	} else if serviceTask := e.findServiceTask(process, elementID); serviceTask != nil {
		return e.enqueueServiceTaskCommand(ctx, txc, instance, serviceTask)
	} else if intermediateEvent := e.findIntermediateEvent(process, elementID); intermediateEvent != nil {
		return e.handleIntermediateCatchEvent(ctx, txc, instance, process, intermediateEvent)
	}

	e.markElementDone(ctx, txc, instance, elementID)
	return e.executeStep(ctx, txc, instance, process, elementID, instance.Variables)
}

// ---------------------------------------------------------------------------
// ServiceTask 命令入队
// ---------------------------------------------------------------------------

func serviceTaskReference(task *BPMNServiceTask) string {
	// 显式声明的内部处理器类型（metaData service_task_type）拥有最高优先级：
	// 「##WebService」「##Java」等 BPMN 标准实现标注是给外部引擎看的，
	// 不应劫持本引擎的 handler 寻址（曾致 incident 流程全部 dead_letter）。
	if task.ServiceTaskType != "" {
		return task.ServiceTaskType
	}
	serviceRef := task.ID
	if task.Name != "" {
		serviceRef = task.Name
	}
	if task.Implementation != "" && !strings.HasPrefix(task.Implementation, "##") {
		serviceRef = task.Implementation
	} else if task.Class != "" {
		serviceRef = task.Class
	} else if task.DelegateExpression != "" {
		serviceRef = task.DelegateExpression
	} else if task.OperationRef != "" {
		serviceRef = task.OperationRef
	}
	return serviceRef
}

func (e *CustomProcessEngine) enqueueServiceTaskCommand(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, task *BPMNServiceTask) error {
	if instance.TenantID <= 0 {
		return fmt.Errorf("ServiceTask 缺少有效租户")
	}
	if instance.Variables == nil {
		instance.Variables = map[string]interface{}{}
	}
	occurrences := intMapVariable(instance.Variables, "_serviceTaskOccurrences")
	occurrence := occurrences[task.ID] + 1
	occurrences[task.ID] = occurrence
	instance.Variables["_serviceTaskOccurrences"] = occurrences
	if _, err := txc.ProcessInstance.UpdateOneID(instance.ID).
		SetVariables(instance.Variables).Save(ctx); err != nil {
		return fmt.Errorf("持久化 ServiceTask occurrence 失败: %w", err)
	}
	idempotencyKey := fmt.Sprintf("%d:workflow.service_task:%d:%s:%d", instance.TenantID, instance.ID, task.ID, occurrence)
	if _, err := commandbus.Enqueue(ctx, txc, commandbus.EnqueueRequest{
		TenantID: instance.TenantID, CommandType: commandbus.CommandExecuteBPMNServiceTask,
		AggregateType: "process_instance", AggregateID: instance.ID,
		IdempotencyKey: idempotencyKey, MaxAttempts: 8,
		Payload: map[string]interface{}{"elementId": task.ID, "serviceRef": serviceTaskReference(task), "occurrence": occurrence},
	}); err != nil {
		return fmt.Errorf("ServiceTask durable command 入队失败: %w", err)
	}
	e.logger.Infow("BPMN ServiceTask durable command enqueued", "tenant_id", instance.TenantID,
		"process_instance_id", instance.ID, "element_id", task.ID, "occurrence", occurrence)
	return nil
}

func intMapVariable(variables map[string]interface{}, key string) map[string]int {
	result := map[string]int{}
	raw, ok := variables[key]
	if !ok {
		return result
	}
	switch values := raw.(type) {
	case map[string]int:
		for name, value := range values {
			result[name] = value
		}
	case map[string]interface{}:
		for name, value := range values {
			switch number := value.(type) {
			case int:
				result[name] = number
			case float64:
				result[name] = int(number)
			}
		}
	}
	return result
}

func mergeServiceTaskVariables(instanceVariables map[string]interface{}, task *BPMNServiceTask) map[string]interface{} {
	variables := make(map[string]interface{}, len(instanceVariables)+12)
	for key, value := range instanceVariables {
		variables[key] = value
	}
	if task == nil {
		return variables
	}
	if task.Type != "" {
		variables["type"] = task.Type
	}
	if task.OperationRef != "" {
		variables["operationRef"] = task.OperationRef
	}
	if task.CCType != "" {
		variables["ccType"] = task.CCType
	}
	if task.CCUserIDs != "" {
		variables["ccUserIds"] = task.CCUserIDs
	}
	if task.CCGroupIDs != "" {
		variables["ccGroupIds"] = task.CCGroupIDs
	}
	if task.CCRoleIDs != "" {
		variables["ccRoleIds"] = task.CCRoleIDs
	}
	if task.CCVariable != "" {
		variables["ccVariable"] = task.CCVariable
	}
	if task.CCNotify != "" {
		variables["ccNotify"] = task.CCNotify
	}
	if task.NotifyChannels != "" {
		variables["notifyChannels"] = task.NotifyChannels
	}
	return variables
}

// ---------------------------------------------------------------------------
// UserTask 创建与分配
// ---------------------------------------------------------------------------

func (e *CustomProcessEngine) createUserTask(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, process *BPMNProcess, task *BPMNUserTask) error {
	// 幂等：同实例同节点已存在未结束任务时直接复用，避免 CompleteTask 流程推进失败重试时重复创建任务（F-2）
	// 必须使用事务客户端：CompleteTask 事务内刚创建的任务对外不可见，用 e.client 会漏读导致重复创建
	if existing, _ := txc.ProcessTask.Query().
		Where(
			processtask.ProcessInstanceID(instance.ID),
			processtask.TaskDefinitionKey(task.ID),
			processtask.StatusNotIn("completed", "cancelled"),
		).
		First(ctx); existing != nil {
		e.logger.Infow("createUserTask: 复用已存在的活跃任务", "existingTaskID", existing.TaskID, "node", task.ID)
		return nil
	}

	// 自动分配逻辑：优先级 BPMN定义 > 流程变量(request/assignee) > 默认分配
	assignee := task.Assignee

	// 辅助函数：从变量中提取用户ID
	getUserID := func(key string) string {
		if v, ok := instance.Variables[key]; ok {
			switch val := v.(type) {
			case float64:
				// JSON numbers are float64
				if val > 0 {
					return strconv.FormatFloat(val, 'f', 0, 64)
				}
			case int:
				if val > 0 {
					return strconv.Itoa(val)
				}
			case string:
				if val != "" && val != "0" {
					return val
				}
			}
		}
		return ""
	}

	// 返工任务（to_requester 拒绝策略）必须分配给原始发起人，不走默认分配
	if assignee == "" && strings.EqualFold(strings.TrimSpace(task.TaskPurpose), "rework") {
		if requester := getUserID("requester_id"); requester != "" {
			assignee = requester
		} else if triggeredBy := getUserID("triggered_by"); triggeredBy != "" {
			assignee = triggeredBy
		}
		if assignee != "" {
			e.logger.Infow("返工任务已分配给发起人", "taskID", task.ID, "assignee", assignee)
		}
	}

	// 如果BPMN没有定义分配人，从流程变量中获取
	if assignee == "" {
		// 优先使用 requester_id（工单申请人）
		assignee = getUserID("requester_id")
		// 其次使用 triggered_by（触发者）
		if assignee == "" {
			assignee = getUserID("triggered_by")
		}
		// 再其次使用 assignee_id
		if assignee == "" {
			assignee = getUserID("assignee_id")
		}
		// 如果还是没有，根据流程变量或数据库规则分配
		if assignee == "" {
			assignee = e.getDefaultAssignee(ctx, instance, task)
		}
	}

	// 展开 candidateGroups 为具体用户，合并到 candidate_users。
	// 这样「我的待办」接口才有可能查到分配给我的任务。
	expandedCandidateUsers := task.CandidateUsers
	if e.groupResolver != nil && strings.TrimSpace(task.CandidateGroups) != "" {
		_, groupUsernames, err := e.groupResolver.ExpandGroupsToUsers(ctx, instance.TenantID, task.CandidateGroups)
		if err != nil {
			// 解析失败：记录警告但不阻塞流程，以免审批组配置漂移导致整个流程中断
			e.logger.Warnw(
				"审批组展开失败，继续仅使用 BPMN candidateUsers",
				"taskID", task.ID,
				"candidateGroups", task.CandidateGroups,
				"error", err,
			)
		} else {
			expandedCandidateUsers = e.groupResolver.MergeCandidateUsers(task.CandidateUsers, groupUsernames)
			e.logger.Infow(
				"审批组已展开",
				"taskID", task.ID,
				"candidateGroups", task.CandidateGroups,
				"expandedUsers", groupUsernames,
			)
		}
	}

	// Use instance.ID (auto-generated integer) for the relationship
	taskConfig := map[string]interface{}{
		"taskPurpose": task.TaskPurpose, "approvalMode": task.ApprovalMode,
		"approvalThreshold": task.ApprovalThreshold, "rejectStrategy": task.RejectStrategy,
		"timeoutAction": task.TimeoutAction, "allowDelegate": task.AllowDelegate,
		"allowAddApprover":        task.AllowAddApprover,
		"commentRequiredOnReject": task.CommentRequiredOnReject,
	}
	createdTask, err := txc.ProcessTask.Create().
		SetTaskID(fmt.Sprintf("TASK-%s-%d", task.ID, time.Now().UnixNano())).
		SetProcessInstanceID(instance.ID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).
		SetTaskDefinitionKey(task.ID).
		SetTaskName(task.Name).
		SetTaskType("user_task").
		SetStatus("created").
		SetAssignee(assignee).
		SetCandidateUsers(expandedCandidateUsers).
		SetCandidateGroups(task.CandidateGroups).
		SetFormKey(task.FormKey).
		SetTaskVariables(taskConfig).
		SetTenantID(instance.TenantID).
		SetCreatedTime(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("创建用户任务失败: %w", err)
	}
	if task.TaskPurpose == "approval" && task.ApprovalMode != "" && task.ApprovalMode != "single" {
		approvers := splitNonEmptyCSV(expandedCandidateUsers)
		if len(approvers) > 1 {
			threshold := task.ApprovalThreshold
			switch task.ApprovalMode {
			case "any":
				threshold = 1
			case "all", "sequential":
				threshold = len(approvers)
			}
			approvalType := "parallel"
			if task.ApprovalMode == "sequential" {
				approvalType = "serial"
			}
			if _, err := createCounterSignTasksWithClient(ctx, txc, createdTask.TaskID, instance.TenantID, &CounterSignRequest{ApprovalType: approvalType, Approvers: approvers, Threshold: threshold}); err != nil {
				return fmt.Errorf("创建会签任务失败: %w", err)
			}
		}
	}
	e.logger.Infow("User task created with auto-assignment", "taskID", task.ID, "taskName", task.Name, "assignee", assignee)

	// BPMN dueDate 属性落地（Phase 4）：此前 XML 解析器校验过 dueDate 但从不写库，
	// 导致 TimeoutScanner 扫描的 due_date 字段永远为空（休眠循环）。
	// 日期语义：截止到当日 23:59:59（本地时区）。
	if dueExpr := strings.TrimSpace(task.DueDate); dueExpr != "" {
		if dueDay, err := time.ParseInLocation("2006-01-02", dueExpr, time.Local); err == nil {
			dueAt := time.Date(dueDay.Year(), dueDay.Month(), dueDay.Day(), 23, 59, 59, 0, time.Local)
			if _, err := txc.ProcessTask.UpdateOne(createdTask).SetDueDate(dueAt).Save(ctx); err != nil {
				e.logger.Warnw("failed to persist BPMN dueDate on task",
					"error", err, "task_def_key", task.ID, "due_date", dueExpr)
			} else {
				e.registerTaskDueTimer(ctx, instance, createdTask, dueAt)
			}
		} else {
			e.logger.Warnw("invalid BPMN dueDate format (expect yyyy-mm-dd)",
				"task_def_key", task.ID, "due_date", dueExpr)
		}
	}

	e.registerBoundaryTimers(ctx, instance, process, task.ID)

	return nil
}

// registerTaskDueTimer 为带截止时间的任务注册 task_due 定时器（Phase 4）。
// 到期由 TimerEventHandler 分发 TimeoutScanner 四动作；已过期的截止时间不注册
// （由恢复兜底扫描处理）。best-effort：注册失败仅告警，扫描器仍在兜底。
func (e *CustomProcessEngine) registerTaskDueTimer(ctx context.Context, instance *ent.ProcessInstance, task *ent.ProcessTask, dueAt time.Time) {
	if e.timerStore == nil || !dueAt.After(time.Now()) {
		return
	}
	instanceID := instance.ID
	if _, err := e.timerStore.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeTaskDue,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessInstanceID:    &instanceID,
		ActivityID:           task.TaskID, // task_due 语义：ActivityID 承载唯一任务 ID
		TimerExpression:      dueAt.Format(time.RFC3339),
		ExpressionType:       ExprTypeDate,
		FireAt:               dueAt,
		ContextVariables: map[string]interface{}{
			"task_id":       task.TaskID,
			"task_def_key":  task.TaskDefinitionKey,
			"timeout_action": task.TaskVariables["timeoutAction"],
		},
		TenantID: instance.TenantID,
	}); err != nil {
		e.logger.Warnw("failed to register task_due timer",
			"error", err, "task_id", task.TaskID, "due_at", dueAt)
		return
	}
	e.logger.Infow("task_due timer registered",
		"task_id", task.TaskID, "due_at", dueAt, "instance_id", instance.ID)
}

func splitNonEmptyCSV(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

// getDefaultAssignee 按优先级解析任务默认分配人：
// 1. 流程变量显式指定 assignee
// 2. 数据库 ticket_assignment_rules 规则匹配
// 均无匹配时返回空字符串，由 candidateUsers/candidateGroups 或手动分配兜底。
func (e *CustomProcessEngine) getDefaultAssignee(ctx context.Context, instance *ent.ProcessInstance, task *BPMNUserTask) string {
	taskName := task.Name

	if instance.Variables != nil {
		if assignee, ok := instance.Variables["assignee"]; ok {
			switch val := assignee.(type) {
			case float64:
				if val > 0 {
					return strconv.FormatFloat(val, 'f', 0, 64)
				}
			case int:
				if val > 0 {
					return strconv.Itoa(val)
				}
			case string:
				if val != "" && val != "0" {
					return val
				}
			}
		}
	}

	if assigneeFromRule := e.getAssigneeFromDBRules(ctx, instance, taskName); assigneeFromRule != "" {
		return assigneeFromRule
	}

	return ""
}

// getAssigneeFromDBRules 从数据库 ticket_assignment_rules 表查询匹配的分配规则
func (e *CustomProcessEngine) getAssigneeFromDBRules(ctx context.Context, instance *ent.ProcessInstance, taskName string) string {
	// 查询当前租户下所有激活的分配规则，按优先级降序排列
	rules, err := e.client.TicketAssignmentRule.Query().
		Where(
			ticketassignmentrule.TenantID(instance.TenantID),
			ticketassignmentrule.IsActive(true),
		).
		Order(ent.Desc(ticketassignmentrule.FieldPriority)).
		All(ctx)
	if err != nil {
		e.logger.Warnw("查询分配规则失败", "error", err)
		return ""
	}

	// 在内存中匹配规则条件
	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}
		// 检查条件是否匹配任务名称
		if matchRuleConditions(rule.Conditions, taskName) {
			// 从 actions 中提取 assignee
			if assigneeVal, ok := rule.Actions["assignee_id"]; ok {
				switch v := assigneeVal.(type) {
				case float64:
					if v > 0 {
						return strconv.FormatFloat(v, 'f', 0, 64)
					}
				case int:
					if v > 0 {
						return strconv.Itoa(v)
					}
				case string:
					if v != "" && v != "0" {
						return v
					}
				}
			}
		}
	}

	return ""
}

// matchRuleConditions 检查规则条件是否与任务名称匹配
func matchRuleConditions(conditions []map[string]interface{}, taskName string) bool {
	if len(conditions) == 0 {
		return false
	}
	for _, cond := range conditions {
		field, _ := cond["field"].(string)
		operator, _ := cond["operator"].(string)
		value, _ := cond["value"].(string)

		if field != "task_name" {
			continue
		}

		switch operator {
		case "equals":
			if taskName == value {
				return true
			}
		case "contains":
			if strings.Contains(taskName, value) {
				return true
			}
		case "prefix":
			if strings.HasPrefix(taskName, value) {
				return true
			}
		case "suffix":
			if strings.HasSuffix(taskName, value) {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// 流程完成与图导航
// ---------------------------------------------------------------------------

func (e *CustomProcessEngine) completeProcess(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance) error {
	_, err := txc.ProcessInstance.UpdateOne(instance).
		SetStatus("completed").
		SetEndTime(time.Now()).
		Save(ctx)
	return err
}

// recordGatewayHistory 将网关路由决策写入流程执行历史，使并行/包容/排他网关的
// 分叉（fork）与汇聚等待（join-wait）可审计（F-4）。
// 写历史属于辅助审计，失败仅告警不阻断主流程。
func (e *CustomProcessEngine) recordGatewayHistory(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, gatewayID, gatewayType, eventType string, nextActivities []string, variables map[string]interface{}) {
	detail := map[string]interface{}{
		"gateway_id":      gatewayID,
		"gateway_type":    gatewayType,
		"event":           eventType,
		"next_activities": nextActivities,
		"tenant_id":       instance.TenantID,
	}
	detailBytes, err := json.Marshal(detail)
	if err != nil {
		e.logger.Warnw("recordGatewayHistory 序列化事件详情失败", "error", err)
		detailBytes = []byte("{}")
	}
	_, err = txc.ProcessExecutionHistory.Create().
		SetHistoryID(fmt.Sprintf("HIST-%s-%d", gatewayID, time.Now().UnixNano())).
		SetProcessInstanceID(instance.ID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).
		SetActivityID(gatewayID).
		SetActivityType("gateway").
		SetEventType(fmt.Sprintf("%s.%s", gatewayType, eventType)).
		SetEventDetail(string(detailBytes)).
		SetVariables(variables).
		SetTenantID(instance.TenantID).
		SetTimestamp(time.Now()).
		Save(ctx)
	if err != nil {
		e.logger.Warnw("recordGatewayHistory 保存失败", "error", err)
	}
}

func (e *CustomProcessEngine) findOutgoingFlows(process *BPMNProcess, sourceRef string) []*BPMNSequenceFlow {
	var flows []*BPMNSequenceFlow
	for _, flow := range process.SequenceFlows {
		if flow.SourceRef == sourceRef {
			flows = append(flows, flow)
		}
	}
	return flows
}

// findSequenceFlow 按 flow id 精确查找顺序流。
func (e *CustomProcessEngine) findSequenceFlow(process *BPMNProcess, flowID string) *BPMNSequenceFlow {
	for _, flow := range process.SequenceFlows {
		if flow.ID == flowID {
			return flow
		}
	}
	return nil
}

// firstOutgoingDeclaration 读取元素声明的第一条 <bpmn:outgoing>（parser 后处理填充的索引）。
func firstOutgoingDeclaration(process *BPMNProcess, elementID string) (string, bool) {
	if process == nil || process.OutgoingDecls == nil {
		return "", false
	}
	flows, ok := process.OutgoingDecls[elementID]
	if !ok {
		return "", false
	}
	for _, fid := range flows {
		if fid != "" {
			return fid, true
		}
	}
	return "", false
}

// suspendOnBrokenGraph 图断裂的显式处置：实例挂起 + 错误返回，
// 替代旧版 return nil 静默卡死（HTTP 200 但实例永不推进）。
func (e *CustomProcessEngine) suspendOnBrokenGraph(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, elementID, reason string) error {
	e.logger.Errorw("executeStep: 流程图断裂，实例挂起",
		"instance", instance.ID, "element", elementID, "reason", reason)
	_, err := txc.ProcessInstance.UpdateOneID(instance.ID).
		SetStatus("suspended").
		SetSuspendedTime(time.Now()).
		SetSuspendedReason(reason).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("标记实例挂起失败（原图断裂）: %w", err)
	}
	return fmt.Errorf("流程图断裂：节点 %s 无可用出边，实例 %d 已挂起", elementID, instance.ID)
}

// evaluateCondition 评估流转条件 (Domain Logic)
// 使用表达式引擎评估条件
func (e *CustomProcessEngine) evaluateCondition(flow *BPMNSequenceFlow, variables map[string]interface{}) bool {
	if flow.ConditionExpression == nil || flow.ConditionExpression.Expression == "" {
		return true // 无条件则默认通过
	}

	// 兼容模板中常见的 ${...} 包裹语法（expr-lang 无法编译 ${} 前缀）
	expr := strings.TrimSpace(flow.ConditionExpression.Expression)
	if strings.HasPrefix(expr, "${") && strings.HasSuffix(expr, "}") {
		expr = strings.TrimSpace(expr[2 : len(expr)-1])
	}
	if expr == "" {
		return true // 剥离包裹后为空视为无条件
	}

	// 合并变量
	evalVars := make(map[string]interface{})
	for k, v := range e.expressionVars {
		evalVars[k] = v
	}
	for k, v := range variables {
		evalVars[k] = v
	}
	// 兼容模板中 variables['xxx'] 的字典访问语法
	evalVars["variables"] = variables

	// 使用表达式引擎评估条件
	result, err := e.exprEngine.EvaluateCondition(expr, evalVars)
	if err != nil {
		// SEC-002 修复：评估失败时默认拒绝（return false），而非放行
		e.logger.Errorw(
			"条件评估失败，默认拒绝流转",
			"expression", expr,
			"error", err,
		)
		return false
	}

	return result
}

func (e *CustomProcessEngine) isEndEvent(process *BPMNProcess, id string) bool {
	for _, event := range process.EndEvents {
		if event.ID == id {
			return true
		}
	}
	return false
}

func (e *CustomProcessEngine) findUserTask(process *BPMNProcess, id string) *BPMNUserTask {
	for _, task := range process.UserTasks {
		if task.ID == id {
			return task
		}
	}
	return nil
}

func (e *CustomProcessEngine) findEndEvent(process *BPMNProcess, id string) *BPMNEndEvent {
	for _, event := range process.EndEvents {
		if event.ID == id {
			return event
		}
	}
	return nil
}

func (e *CustomProcessEngine) findExclusiveGateway(process *BPMNProcess, id string) *BPMNExclusiveGateway {
	for _, gateway := range process.ExclusiveGateways {
		if gateway.ID == id {
			return gateway
		}
	}
	return nil
}

func (e *CustomProcessEngine) findParallelGateway(process *BPMNProcess, id string) *BPMNParallelGateway {
	for _, gateway := range process.ParallelGateways {
		if gateway.ID == id {
			return gateway
		}
	}
	return nil
}

func (e *CustomProcessEngine) findInclusiveGateway(process *BPMNProcess, id string) *BPMNInclusiveGateway {
	for _, gateway := range process.InclusiveGateways {
		if gateway.ID == id {
			return gateway
		}
	}
	return nil
}

// findIncomingFlows 返回以 targetRef 为目标的顺序流（即 targetRef 的入边）
func (e *CustomProcessEngine) findIncomingFlows(process *BPMNProcess, targetRef string) []*BPMNSequenceFlow {
	var flows []*BPMNSequenceFlow
	for _, flow := range process.SequenceFlows {
		if flow.TargetRef == targetRef {
			flows = append(flows, flow)
		}
	}
	return flows
}

func (e *CustomProcessEngine) findServiceTask(process *BPMNProcess, id string) *BPMNServiceTask {
	for _, task := range process.ServiceTasks {
		if task.ID == id {
			return task
		}
	}
	return nil
}

func (e *CustomProcessEngine) findIntermediateEvent(process *BPMNProcess, id string) *BPMNIntermediateEvent {
	for _, event := range process.IntermediateEvents {
		if event.ID == id {
			return event
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// 定时器事件运行时注册（Phase 4）
// ---------------------------------------------------------------------------

// handleIntermediateCatchEvent 处理中间捕获事件。
// 如果包含定时器定义，注册定时器并阻塞流程（不调用 executeStep）；
// 否则直接推进（非定时器类型的中间事件暂按透传处理）。
func (e *CustomProcessEngine) handleIntermediateCatchEvent(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, process *BPMNProcess, event *BPMNIntermediateEvent) error {
	if event.TimerEventDefinition == nil || !e.hasTimerExpression(event.TimerEventDefinition) {
		e.markElementDone(ctx, txc, instance, event.ID)
		return e.executeStep(ctx, txc, instance, process, event.ID, instance.Variables)
	}

	if e.timerStore == nil {
		return fmt.Errorf("定时器服务未配置，无法处理中间定时器事件 %s", event.ID)
	}

	expression, expressionType := e.extractTimerExpression(event.TimerEventDefinition)
	fireAt, err := CalculateFireAt(expression, expressionType, time.Now())
	if err != nil {
		return fmt.Errorf("计算中间定时器触发时间失败: %w", err)
	}

	instanceID := instance.ID
	_, err = e.timerStore.Create(ctx, &CreateTimerRequest{
		TimerType:            TimerTypeIntermediate,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessInstanceID:    &instanceID,
		ActivityID:           event.ID,
		TimerExpression:      expression,
		ExpressionType:       ExpressionType(expressionType),
		FireAt:               fireAt,
		TenantID:             instance.TenantID,
	})
	if err != nil {
		return fmt.Errorf("创建中间定时器记录失败: %w", err)
	}

	e.logger.Infow("Intermediate timer registered at runtime",
		"instance_id", instance.ID,
		"activity_id", event.ID,
		"expression", expression,
		"fire_at", fireAt,
	)

	return nil
}

// registerBoundaryTimers 在 UserTask 创建后，扫描并注册绑定到该任务的 boundary timer。
func (e *CustomProcessEngine) registerBoundaryTimers(ctx context.Context, instance *ent.ProcessInstance, process *BPMNProcess, taskDefKey string) {
	if e.timerStore == nil {
		return
	}

	for _, boundary := range process.BoundaryEvents {
		if boundary.AttachedToRef != taskDefKey {
			continue
		}
		if boundary.TimerEventDefinition == nil || !e.hasTimerExpression(boundary.TimerEventDefinition) {
			continue
		}

		expression, expressionType := e.extractTimerExpression(boundary.TimerEventDefinition)
		fireAt, err := CalculateFireAt(expression, expressionType, time.Now())
		if err != nil {
			e.logger.Warnw("Failed to calculate boundary timer fire time",
				"boundary_event_id", boundary.ID,
				"attached_to", taskDefKey,
				"error", err,
			)
			continue
		}

		instanceID := instance.ID
		_, err = e.timerStore.Create(ctx, &CreateTimerRequest{
			TimerType:            TimerTypeBoundary,
			ProcessDefinitionKey: instance.ProcessDefinitionKey,
			ProcessInstanceID:    &instanceID,
			ActivityID:           boundary.ID,
			TimerExpression:      expression,
			ExpressionType:       ExpressionType(expressionType),
			FireAt:               fireAt,
			TenantID:             instance.TenantID,
		})
		if err != nil {
			e.logger.Warnw("Failed to register boundary timer",
				"boundary_event_id", boundary.ID,
				"attached_to", taskDefKey,
				"error", err,
			)
			continue
		}

		e.logger.Infow("Boundary timer registered at runtime",
			"instance_id", instance.ID,
			"boundary_event_id", boundary.ID,
			"attached_to", taskDefKey,
			"expression", expression,
			"fire_at", fireAt,
		)
	}
}

// cancelTaskDueTimers 在任务完成/终止时取消该任务的 task_due 定时器（Phase 4）。
// 与 cancelBoundaryTimers 同语义：不留到期后才静默跳过的死 timer。
func (e *CustomProcessEngine) cancelTaskDueTimers(ctx context.Context, tenantID int, taskID string) {
	if e.timerStore == nil {
		return
	}
	timers, _, err := e.timerStore.List(ctx, TimerListFilter{
		TenantID:  tenantID,
		TimerType: string(TimerTypeTaskDue),
		Status:    string(TimerStatusPending),
		PageSize:  100,
		Page:      1,
	})
	if err != nil {
		e.logger.Warnw("failed to list task_due timers for cancellation",
			"error", err, "task_id", taskID)
		return
	}
	for _, timer := range timers {
		if timer.ActivityID != taskID {
			continue
		}
		if err := e.timerStore.CancelByTimerID(ctx, timer.TimerID); err != nil {
			e.logger.Warnw("failed to cancel task_due timer",
				"error", err, "timer_id", timer.TimerID, "task_id", taskID)
			continue
		}
		if e.timerScheduler != nil {
			e.timerScheduler.Cancel(timer.TimerID)
		}
		e.logger.Infow("task_due timer cancelled on task completion",
			"timer_id", timer.TimerID, "task_id", taskID)
	}
}

// cancelBoundaryTimers 在 UserTask 正常完成时取消绑定到该任务的所有 pending boundary timer。
func (e *CustomProcessEngine) cancelBoundaryTimers(ctx context.Context, instance *ent.ProcessInstance, process *BPMNProcess, taskDefKey string) {
	if e.timerStore == nil {
		return
	}

	for _, boundary := range process.BoundaryEvents {
		if boundary.AttachedToRef != taskDefKey {
			continue
		}
		if boundary.TimerEventDefinition == nil {
			continue
		}

		timers, _, err := e.timerStore.List(ctx, TimerListFilter{
			TenantID:           instance.TenantID,
			TimerType:          string(TimerTypeBoundary),
			ProcessInstanceID:  &instance.ID,
			Status:             string(TimerStatusPending),
			PageSize:           100,
			Page:               1,
		})
		if err != nil {
			e.logger.Warnw("Failed to list boundary timers for cancellation",
				"instance_id", instance.ID,
				"task_def_key", taskDefKey,
				"error", err,
			)
			continue
		}

		for _, timer := range timers {
			if timer.ActivityID == boundary.ID {
				if err := e.timerStore.CancelByTimerID(ctx, timer.TimerID); err != nil {
					e.logger.Warnw("Failed to cancel boundary timer",
						"timer_id", timer.TimerID,
						"error", err,
					)
				} else if e.timerScheduler != nil {
					e.timerScheduler.Cancel(timer.TimerID)
				}
				e.logger.Infow("Boundary timer cancelled on task completion",
					"timer_id", timer.TimerID,
					"boundary_event_id", boundary.ID,
					"task_def_key", taskDefKey,
				)
			}
		}
	}
}

// hasTimerExpression 检查定时器定义是否包含有效的表达式。
func (e *CustomProcessEngine) hasTimerExpression(def *BPMNTimerEventDefinition) bool {
	return strings.TrimSpace(def.TimeDuration) != "" ||
		strings.TrimSpace(def.TimeDate) != "" ||
		strings.TrimSpace(def.TimeCycle) != ""
}

// extractTimerExpression 从定时器定义中提取表达式和类型。
func (e *CustomProcessEngine) extractTimerExpression(def *BPMNTimerEventDefinition) (expression, expressionType string) {
	if strings.TrimSpace(def.TimeDuration) != "" {
		return strings.TrimSpace(def.TimeDuration), "duration"
	}
	if strings.TrimSpace(def.TimeDate) != "" {
		return strings.TrimSpace(def.TimeDate), "date"
	}
	return strings.TrimSpace(def.TimeCycle), "cycle"
}

// ---------------------------------------------------------------------------
// 网关路由
// ---------------------------------------------------------------------------

const maxGatewayForkDepth = 64

// handleParallelGateway 处理并行网关：多入边时作为汇聚节点等待所有分支完成；否则作为分叉节点激活所有出边（F-1）。
func (e *CustomProcessEngine) handleParallelGateway(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, process *BPMNProcess, gateway *BPMNParallelGateway, depth int) error {
	if depth > maxGatewayForkDepth {
		return fmt.Errorf("并行网关分叉层级超过上限，可能存在环路: %s", gateway.ID)
	}
	incoming := e.findIncomingFlows(process, gateway.ID)
	if len(incoming) > 1 && !e.allIncomingBranchesCompleted(ctx, txc, instance, process, gateway.ID) {
		e.logger.Infow("并行网关汇聚等待其余分支完成", "gateway", gateway.ID, "instance", instance.ID)
		e.recordGatewayHistory(ctx, txc, instance, gateway.ID, "parallel", "join-wait", nil, instance.Variables)
		return nil // 仍有分支未结束，等待，不推进
	}
	var next []string
	for _, flow := range e.findOutgoingFlows(process, gateway.ID) {
		next = append(next, flow.TargetRef)
		if err := e.dispatchGatewayOrElement(ctx, txc, instance, process, flow.TargetRef, depth); err != nil {
			return err
		}
	}
	e.markElementDone(ctx, txc, instance, gateway.ID)
	e.recordGatewayHistory(ctx, txc, instance, gateway.ID, "parallel", "fork", next, instance.Variables)
	return nil
}

// handleInclusiveGateway 处理包容网关：汇聚等待所有入边分支完成；分叉时激活所有命中条件的出边（F-1）。
func (e *CustomProcessEngine) handleInclusiveGateway(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, process *BPMNProcess, gateway *BPMNInclusiveGateway, depth int) error {
	if depth > maxGatewayForkDepth {
		return fmt.Errorf("包容网关分叉层级超过上限，可能存在环路: %s", gateway.ID)
	}
	incoming := e.findIncomingFlows(process, gateway.ID)
	if len(incoming) > 1 && !e.allIncomingBranchesCompleted(ctx, txc, instance, process, gateway.ID) {
		e.logger.Infow("包容网关汇聚等待其余分支完成", "gateway", gateway.ID, "instance", instance.ID)
		e.recordGatewayHistory(ctx, txc, instance, gateway.ID, "inclusive", "join-wait", nil, instance.Variables)
		return nil
	}
	var next []string
	matched := false
	for _, flow := range e.findOutgoingFlows(process, gateway.ID) {
		if e.evaluateCondition(flow, instance.Variables) {
			matched = true
			next = append(next, flow.TargetRef)
			if err := e.dispatchGatewayOrElement(ctx, txc, instance, process, flow.TargetRef, depth); err != nil {
				return err
			}
		}
	}
	e.markElementDone(ctx, txc, instance, gateway.ID)
	e.recordGatewayHistory(ctx, txc, instance, gateway.ID, "inclusive", "fork", next, instance.Variables)
	if !matched {
		e.logger.Warnw("包容网关无满足条件的出边，流程在此等待", "gateway", gateway.ID, "instance", instance.ID)
	}
	return nil
}

// dispatchGatewayOrElement 优先分发到嵌套的并行/包容网关（depth+1 用于环路防护），其余交给 handleElement。
func (e *CustomProcessEngine) dispatchGatewayOrElement(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, process *BPMNProcess, elementID string, depth int) error {
	if pg := e.findParallelGateway(process, elementID); pg != nil {
		return e.handleParallelGateway(ctx, txc, instance, process, pg, depth+1)
	}
	if ig := e.findInclusiveGateway(process, elementID); ig != nil {
		return e.handleInclusiveGateway(ctx, txc, instance, process, ig, depth+1)
	}
	return e.handleElement(ctx, txc, instance, process, elementID)
}

// ---------------------------------------------------------------------------
// 分支完成标记与汇聚判断
// ---------------------------------------------------------------------------

// markElementDone 标记某流程元素已执行完成（写入实例变量 _done_ 并持久化）。
// 用于并行/包容网关汇聚时判断「非用户任务源」分支（服务任务、子网关、排他网关、结束事件）
// 是否已真正结束——这是修复「汇聚只认 user-task 源而漏掉其他分支导致提前汇聚/死锁」的关键。
// 直接就地修改 instance.Variables（同一 map 引用会在同一次执行内对汇聚判断可见），
// 并通过 txc 持久化，使后续 CompleteTask 的汇聚判断也能读到。
func (e *CustomProcessEngine) markElementDone(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, elementID string) {
	if err := e.markElementDoneStrict(ctx, txc, instance, elementID); err != nil {
		e.logger.Warnw("markElementDone 持久化失败", "elementID", elementID, "error", err)
	}
}

func (e *CustomProcessEngine) markElementDoneStrict(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, elementID string) error {
	if instance.Variables == nil {
		instance.Variables = map[string]interface{}{}
	}
	done, ok := instance.Variables["_done_"].(map[string]interface{})
	if !ok {
		done = map[string]interface{}{}
		instance.Variables["_done_"] = done
	}
	done[elementID] = true
	_, err := txc.ProcessInstance.UpdateOneID(instance.ID).SetVariables(instance.Variables).Save(ctx)
	return err
}

// allIncomingBranchesCompleted 判断汇聚网关的所有入边分支是否均已结束。
//   - 以用户任务为源的分支：按 DB 中该任务是否已 completed/cancelled 判断（与历史行为一致）。
//   - 以服务任务/子网关/排他网关/结束事件为源的分支：必须已在 _done_ 中标记为完成。
//     旧实现对这些非用户任务源直接 continue（永远视为完成），会导致提前汇聚或死锁（P1 网关完整性）。
//   - 查询失败时保守返回 false（视为未完成），避免提前汇聚。
//   - 任务状态查询必须使用事务客户端：CompleteTask 事务内刚标记完成的任务对外不可见，
//     用事务外客户端会误判分支未完成导致网关死锁。
func (e *CustomProcessEngine) allIncomingBranchesCompleted(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, process *BPMNProcess, gatewayID string) bool {
	done := map[string]interface{}{}
	if instance.Variables != nil {
		if d, ok := instance.Variables["_done_"].(map[string]interface{}); ok {
			done = d
		}
	}
	for _, flow := range e.findIncomingFlows(process, gatewayID) {
		src := flow.SourceRef
		if e.findUserTask(process, src) != nil {
			// 用户任务源：以 DB 实际完成状态为准
			open, err := txc.ProcessTask.Query().
				Where(
					processtask.ProcessInstanceID(instance.ID),
					processtask.TaskDefinitionKey(src),
					processtask.StatusNotIn("completed", "cancelled"),
				).
				Exist(ctx)
			if err != nil {
				e.logger.Warnw("allIncomingBranchesCompleted 查询失败，保守视为未完成", "error", err)
				return false
			}
			if open {
				return false
			}
			continue
		}
		// 服务任务/子网关/排他网关/结束事件源：必须已记录完成
		if v, ok := done[src]; !ok || v != true {
			e.logger.Infow("并行/包容网关汇聚等待分支完成", "gateway", gatewayID, "waitingSrc", src)
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// 可观测性
// ---------------------------------------------------------------------------

// DetectStuckInstances 返回运行时间超过 olderThan 仍未结束（running）的流程实例，
// 用于可观测性与卡死检测（P3）。结合日志中的 "并行/包容网关汇聚等待分支完成" 可定位汇聚死锁。
func (e *CustomProcessEngine) DetectStuckInstances(ctx context.Context, tenantID int, olderThan time.Time) ([]*ent.ProcessInstance, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("DetectStuckInstances 需要有效的租户上下文")
	}
	return e.client.ProcessInstance.Query().
		Where(
			processinstance.TenantID(tenantID),
			processinstance.Status("running"),
			processinstance.StartTimeLT(olderThan),
		).
		All(ctx)
}
