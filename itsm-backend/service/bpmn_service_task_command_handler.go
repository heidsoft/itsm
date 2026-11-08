package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service/bpmn"
)

// HandleBPMNServiceTaskCommand executes one persisted BPMN ServiceTask after the
// process transaction has committed, then atomically records completion and
// advances the process. The command worker supplies retry, lease and fencing.
func (e *CustomProcessEngine) HandleBPMNServiceTaskCommand(ctx context.Context, cmd *ent.OperationalCommand) error {
	if cmd == nil || cmd.CommandType != commandbus.CommandExecuteBPMNServiceTask ||
		cmd.AggregateType != "process_instance" || cmd.AggregateID <= 0 || cmd.TenantID <= 0 {
		return fmt.Errorf("invalid BPMN ServiceTask command")
	}
	elementID, _ := cmd.Payload["elementId"].(string)
	serviceRef, _ := cmd.Payload["serviceRef"].(string)
	occurrence := bpmnPayloadInt(cmd.Payload["occurrence"])
	if elementID == "" || serviceRef == "" || occurrence <= 0 {
		return fmt.Errorf("invalid BPMN ServiceTask command payload")
	}

	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, cmd.TenantID)
	instance, definition, process, task, err := e.loadServiceTaskCommandState(workflowCtx, cmd, elementID)
	if err != nil {
		return err
	}
	completed := intMapVariable(instance.Variables, "_serviceTaskCompletedOccurrences")
	if completed[elementID] >= occurrence {
		return nil
	}
	if err := e.verifyServiceTaskCommandLease(workflowCtx, e.client, cmd); err != nil {
		return err
	}
	occurrences := intMapVariable(instance.Variables, "_serviceTaskOccurrences")
	// 只按 occurrence 判陈旧，不校验 CurrentActivityID：并行网关分支会互相覆盖
	// 单位置活动列，位置校验会把合法命令误杀成 stale，实例在汇聚网关永久卡死。
	if occurrences[elementID] != occurrence {
		return fmt.Errorf("stale BPMN ServiceTask command for element %s occurrence %d", elementID, occurrence)
	}
	if actual := serviceTaskReference(task); actual != serviceRef {
		return fmt.Errorf("BPMN ServiceTask handler mismatch")
	}
	handler := e.callbackRegistry.GetHandler(serviceRef)
	if handler == nil {
		// serviceRef（ID/name/implementation 等）未命中时，按显式声明的
		// service_task_type 再试一次。注意 task.GetType() 是 BPMN 元素类型
		// （恒为 "ServiceTask"），registry 里注册的是 handler ID，两者命名
		// 空间不同——保留此 fallback 只会掩盖配置错误，故移除之。
		if task.ServiceTaskType != "" && task.ServiceTaskType != serviceRef {
			handler = e.callbackRegistry.GetHandler(task.ServiceTaskType)
		}
	}
	if handler == nil {
		return fmt.Errorf("ServiceTask handler '%s' 未注册 (element=%s service_task_type=%s)", serviceRef, task.ID, task.ServiceTaskType)
	}
	variables := mergeServiceTaskVariables(instance.Variables, task)
	variables["_commandId"] = cmd.ID
	variables["_idempotencyKey"] = cmd.IdempotencyKey
	variables["_serviceTaskOccurrence"] = occurrence

	// §事务收敛（2026-11-08）：副作用完成凭据与流程完成写进同一个 ent.Tx。
	//
	// 外部副作用（通知/Webhook/邮件）物理上无法随数据库事务回滚，因此
	// 「同事务」的工程含义是：副作用的持久化凭据（_serviceTaskExecutedOccurrences
	// + receipt + 审计）与流程节点完成/推进原子提交。旧实现把凭据放在副作用
	// 之前的独立「执行意图」事务里，一旦 worker 在副作用执行中途被杀，意图已
	// 提交而副作用从未生效，恢复时却被当成「已执行」跳过——at-most-once 静默
	// 漏执行。现在意图只用于诊断，是否重放以完成事务里的凭据为准：凭据不存在
	// 就重放 handler（_idempotencyKey 恒定，由 handler/外部系统按幂等键去重），
	// 凭据存在才跳过。
	//
	// 同时把原先独立的「输出变量持久化」小事务并入完成事务：崩溃恢复重放
	// handler 会重新产出输出，不再需要独立事务兜住 result 丢失。
	executed := intMapVariable(instance.Variables, "_serviceTaskExecutedOccurrences")
	var result *dto.ServiceTaskResult
	if executed[elementID] >= occurrence {
		e.logger.Infow("ServiceTask 完成凭据已存在，跳过重复副作用",
			"command_id", cmd.ID, "tenant_id", cmd.TenantID, "element_id", elementID, "occurrence", occurrence)
	} else {
		if _, err := e.markServiceTaskExecuting(workflowCtx, cmd, instance.ID, elementID, occurrence); err != nil {
			return err
		}
		result, err = handler.Execute(workflowCtx, nil, variables)
		if err != nil {
			// handler 明确报告失败：清除诊断用的执行意图，让重试重新执行副作用。
			if clearErr := e.clearServiceTaskExecuting(workflowCtx, cmd, elementID); clearErr != nil {
				e.logger.Errorw("ServiceTask 执行意图清除失败（诊断用，不影响重放判定）",
					"command_id", cmd.ID, "tenant_id", cmd.TenantID, "element_id", elementID, "error", clearErr)
			}
			// 错误必须自带定位信息：handler.Execute 的签名收的是 *ent.ProcessTask，
			// 而 ServiceTask 来自 BPMN 定义、没有对应的 process_tasks 行，这里只能传
			// nil；在调用点补齐流程实例、定义、节点与 occurrence，dead_letter 里
			// 才能直接定位是哪个流程的哪个节点。
			return fmt.Errorf(
				"ServiceTask %s 执行失败（element=%s occurrence=%d processInstance=%d processDefinition=%s tenant=%d）: %w",
				serviceRef, elementID, occurrence, instance.ID, instance.ProcessDefinitionKey, cmd.TenantID, err,
			)
		}
	}

	tx, err := e.client.Tx(workflowCtx)
	if err != nil {
		return fmt.Errorf("开启 ServiceTask 完成事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txc := tx.Client()
	// 在同一事务中用条件更新锁定并复核当前 fencing 身份。该行锁会阻止
	// lease 接管与流程推进并发提交；旧 worker 失去 lease 后不得推进流程。
	if err := e.lockServiceTaskCommandLease(workflowCtx, txc, cmd); err != nil {
		return err
	}
	current, err := txc.ProcessInstance.Query().Where(
		processinstance.IDEQ(instance.ID), processinstance.TenantIDEQ(cmd.TenantID),
	).Only(workflowCtx)
	if err != nil {
		return fmt.Errorf("重新加载流程实例失败: %w", err)
	}
	completed = intMapVariable(current.Variables, "_serviceTaskCompletedOccurrences")
	if completed[elementID] >= occurrence {
		return tx.Commit()
	}
	occurrences = intMapVariable(current.Variables, "_serviceTaskOccurrences")
	if occurrences[elementID] != occurrence {
		return fmt.Errorf("ServiceTask 完成时流程状态已变化")
	}
	if current.Variables == nil {
		current.Variables = map[string]interface{}{}
	}
	if result != nil {
		for key, value := range result.OutputVars {
			current.Variables[key] = value
		}
	}
	completed[elementID] = occurrence
	current.Variables["_serviceTaskCompletedOccurrences"] = completed

	// 副作用完成凭据：与输出、补偿登记、节点完成、流程推进同属这一个事务。
	// 只有 Commit 成功后它才可见；worker 在 Commit 前被杀 → 整体回滚 →
	// 下一次尝试按凭据缺失重放 handler（幂等键不变），不再静默漏执行。
	// 重新以事务内最新实例变量为准，避免并发分支覆盖刚读到的完成凭据。
	executed = intMapVariable(current.Variables, "_serviceTaskExecutedOccurrences")
	executed[elementID] = occurrence
	current.Variables["_serviceTaskExecutedOccurrences"] = executed
	receipts := map[string]interface{}{}
	if raw, ok := current.Variables["_serviceTaskReceipts"].(map[string]interface{}); ok {
		for key, value := range raw {
			receipts[key] = value
		}
	}
	receipts[elementID] = map[string]interface{}{
		"occurrence":     occurrence,
		"serviceRef":     serviceRef,
		"idempotencyKey": cmd.IdempotencyKey,
	}
	current.Variables["_serviceTaskReceipts"] = receipts

	if _, err := txc.ProcessInstance.UpdateOneID(current.ID).SetVariables(current.Variables).Save(workflowCtx); err != nil {
		return fmt.Errorf("保存 ServiceTask 完成凭据与输出失败: %w", err)
	}
	// 审计与完成凭据同事务：任一条写失败都会让整个完成事务回滚，
	// 不留下「流程已推进但没有副作用凭据」的孤儿分叉。
	if err := e.auditService.RecordAuditWithClient(workflowCtx, txc, &AuditContext{
		ProcessInstanceID:    current.ID,
		ProcessInstanceKey:   current.ProcessInstanceID,
		ProcessDefinitionKey: current.ProcessDefinitionKey,
		ProcessDefinitionID:  current.ProcessDefinitionID,
		ActivityID:           elementID,
		ActivityType:         "ServiceTask",
		Action:               AuditActionActivityCompleted,
		VariablesAfter:       current.Variables,
		Comment: fmt.Sprintf("ServiceTask %s 副作用完成凭据（occurrence=%d）与流程推进同事务提交",
			serviceRef, occurrence),
		TenantID: current.TenantID,
	}); err != nil {
		return fmt.Errorf("记录 ServiceTask 完成凭据审计失败: %w", err)
	}
	// 补偿范围登记与完成同事务：流程推进到结束事件时才能补偿到它。
	if err := e.registerCompensation(workflowCtx, txc, current, elementID, serviceRef, occurrence, current.Variables); err != nil {
		return err
	}
	if err := e.markElementDoneStrict(workflowCtx, txc, current, elementID); err != nil {
		return fmt.Errorf("标记 ServiceTask 完成失败: %w", err)
	}
	if err := e.executeStep(workflowCtx, txc, current, process, elementID, current.Variables); err != nil {
		return fmt.Errorf("推进 ServiceTask 后续流程失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交 ServiceTask 完成事务失败: %w", err)
	}
	e.logger.Infow("BPMN ServiceTask command completed", "command_id", cmd.ID, "tenant_id", cmd.TenantID,
		"process_instance_id", instance.ID, "element_id", elementID, "occurrence", occurrence,
		"process_definition_id", definition.ID)
	return nil
}

// markServiceTaskExecuting 在同一事务里完成两件必须原子的事：
//  1. 锁定命令租约（条件 UPDATE，旧 worker 失约后不得推进）；
//  2. 登记「本 occurrence 已尝试执行副作用」并累计尝试次数
//     （_serviceTaskExecuting / _serviceTaskAttempts，供运维诊断）。
//
// 注意：这个「意图」不再作为 exactly-once 的判定锚点。外部副作用无法随
// 数据库事务回滚，若把「已登记意图」当作「副作用已生效」，worker 在副作用
// 执行中途被杀时会静默漏执行。真正的完成凭据是写进完成事务的
// _serviceTaskExecutedOccurrences + receipt；本函数返回值仅保留兼容语义。
func (e *CustomProcessEngine) markServiceTaskExecuting(ctx context.Context, cmd *ent.OperationalCommand, instanceID int, elementID string, occurrence int) (bool, error) {
	tx, err := e.client.Tx(ctx)
	if err != nil {
		return false, fmt.Errorf("开启 ServiceTask 执行权事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txc := tx.Client()

	if err := e.lockServiceTaskCommandLease(ctx, txc, cmd); err != nil {
		return false, err
	}
	alreadyExecuting := false
	if _, err := updateProcessInstanceVariablesCAS(ctx, txc, instanceID, cmd.TenantID, func(vars map[string]interface{}) error {
		// 每次真正准备执行 handler 都如实累计尝试次数。_serviceTaskExecuting
		// 只是诊断用的意图标记：它在独立事务里提交，完成事务回滚后仍会残留；
		// 若据此早返回，恢复重放的那次真实副作用就会漏计，运维对账会把
		// 「2 次执行」看成 1 次。是否重放的判定权已交给完成事务里的凭据。
		attempts := intMapVariable(vars, "_serviceTaskAttempts")
		attempts[elementID]++
		vars["_serviceTaskAttempts"] = attempts
		executing := intMapVariable(vars, "_serviceTaskExecuting")
		if executing[elementID] == occurrence {
			alreadyExecuting = true
		}
		executing[elementID] = occurrence
		vars["_serviceTaskExecuting"] = executing
		return nil
	}); err != nil {
		return false, fmt.Errorf("登记 ServiceTask 执行权失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("提交 ServiceTask 执行权事务失败: %w", err)
	}
	return alreadyExecuting, nil
}

// saveServiceTaskOutput 把 handler 输出变量在副作用成功后立即持久化（独立小事务）。
//
// 2026-11-08 起主路径不再调用：输出变量、完成凭据、补偿登记与流程推进已在
// 同一个完成事务里原子提交；完成事务回滚后的恢复路径会按凭据缺失重放
// handler，重新产出输出，不再需要独立事务兜住 result 丢失。保留此函数供
// 运维诊断脚本与历史兼容调用，禁止重新接回主命令路径。
func (e *CustomProcessEngine) saveServiceTaskOutput(ctx context.Context, cmd *ent.OperationalCommand, instanceID int, outputs map[string]interface{}) error {
	if len(outputs) == 0 {
		return nil
	}
	tx, err := e.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启 ServiceTask 输出事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txc := tx.Client()
	if err := e.lockServiceTaskCommandLease(ctx, txc, cmd); err != nil {
		return err
	}
	if _, err := updateProcessInstanceVariablesCAS(ctx, txc, instanceID, cmd.TenantID, func(vars map[string]interface{}) error {
		for key, value := range outputs {
			vars[key] = value
		}
		return nil
	}); err != nil {
		return fmt.Errorf("保存 ServiceTask 输出变量失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交 ServiceTask 输出事务失败: %w", err)
	}
	return nil
}

// clearServiceTaskExecuting 清除副作用执行权登记（handler 明确失败时调用），
// 使命令重试时重新执行副作用。仅动实例变量，不重新锁定租约——调用方持有租约。
func (e *CustomProcessEngine) clearServiceTaskExecuting(ctx context.Context, cmd *ent.OperationalCommand, elementID string) error {
	instanceID := cmd.AggregateID
	if _, err := updateProcessInstanceVariablesCAS(ctx, e.client, instanceID, cmd.TenantID, func(vars map[string]interface{}) error {
		executing := intMapVariable(vars, "_serviceTaskExecuting")
		if _, ok := executing[elementID]; !ok {
			return nil
		}
		delete(executing, elementID)
		vars["_serviceTaskExecuting"] = executing
		return nil
	}); err != nil {
		return fmt.Errorf("清除 ServiceTask 执行权失败: %w", err)
	}
	return nil
}

func (e *CustomProcessEngine) lockServiceTaskCommandLease(ctx context.Context, client *ent.Client, cmd *ent.OperationalCommand) error {
	if cmd.LeaseOwner == "" || cmd.FencingToken <= 0 || cmd.LeaseExpiresAt == nil {
		return commandbus.ErrLeaseLost
	}
	_, err := client.OperationalCommand.UpdateOneID(cmd.ID).Where(
		operationalcommand.TenantIDEQ(cmd.TenantID),
		operationalcommand.StatusEQ(commandbus.StatusProcessing),
		operationalcommand.LeaseOwnerEQ(cmd.LeaseOwner),
		operationalcommand.FencingTokenEQ(cmd.FencingToken),
		operationalcommand.LeaseExpiresAtGT(time.Now()),
	).SetUpdatedAt(time.Now()).Save(ctx)
	if ent.IsNotFound(err) {
		return commandbus.ErrLeaseLost
	}
	if err != nil {
		return fmt.Errorf("lock BPMN ServiceTask command lease: %w", err)
	}
	return nil
}

func (e *CustomProcessEngine) verifyServiceTaskCommandLease(ctx context.Context, client *ent.Client, cmd *ent.OperationalCommand) error {
	if cmd.LeaseOwner == "" || cmd.FencingToken <= 0 || cmd.LeaseExpiresAt == nil {
		return commandbus.ErrLeaseLost
	}
	_, err := client.OperationalCommand.Query().Where(
		operationalcommand.IDEQ(cmd.ID),
		operationalcommand.TenantIDEQ(cmd.TenantID),
		operationalcommand.StatusEQ(commandbus.StatusProcessing),
		operationalcommand.LeaseOwnerEQ(cmd.LeaseOwner),
		operationalcommand.FencingTokenEQ(cmd.FencingToken),
		operationalcommand.LeaseExpiresAtGT(time.Now()),
	).Only(ctx)
	if ent.IsNotFound(err) {
		return commandbus.ErrLeaseLost
	}
	if err != nil {
		return fmt.Errorf("verify BPMN ServiceTask command lease: %w", err)
	}
	return nil
}

func (e *CustomProcessEngine) loadServiceTaskCommandState(ctx context.Context, cmd *ent.OperationalCommand, elementID string) (*ent.ProcessInstance, *ent.ProcessDefinition, *BPMNProcess, *BPMNServiceTask, error) {
	instance, err := e.client.ProcessInstance.Query().Where(
		processinstance.IDEQ(cmd.AggregateID), processinstance.TenantIDEQ(cmd.TenantID),
	).Only(ctx)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("load BPMN ServiceTask instance: %w", err)
	}
	definition, err := e.client.ProcessDefinition.Query().Where(
		processdefinition.IDEQ(instance.ProcessDefinitionID), processdefinition.TenantIDEQ(cmd.TenantID),
	).Only(ctx)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("load BPMN ServiceTask definition: %w", err)
	}
	definitions, err := e.parser.ParseXML(definition.BpmnXML)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parse BPMN ServiceTask definition: %w", err)
	}
	if len(definitions.Processes) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("BPMN ServiceTask definition has no process")
	}
	process := definitions.Processes[0]
	task := e.findServiceTask(process, elementID)
	if task == nil {
		return nil, nil, nil, nil, fmt.Errorf("BPMN ServiceTask element not found")
	}
	return instance, definition, process, task, nil
}

// bpmnPayloadInt 从 BPMN 命令 payload 中宽松解析整数（缺失/类型不符时返回 0）。
// 命名带 bpmn 前缀以避免与 notification_delivery_command_handler.go 的 payloadInt 冲突。
func bpmnPayloadInt(value interface{}) int {
	switch number := value.(type) {
	case int:
		return number
	case float64:
		return int(number)
	default:
		return 0
	}
}
