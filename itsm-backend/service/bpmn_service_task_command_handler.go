package service

import (
	"context"
	"fmt"
	"time"

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
	if occurrences[elementID] != occurrence || instance.CurrentActivityID != elementID {
		return fmt.Errorf("stale BPMN ServiceTask command for element %s occurrence %d", elementID, occurrence)
	}
	if actual := serviceTaskReference(task); actual != serviceRef {
		return fmt.Errorf("BPMN ServiceTask handler mismatch")
	}
	handler := e.callbackRegistry.GetHandler(serviceRef)
	if handler == nil {
		handler = e.callbackRegistry.GetHandler(task.GetType())
	}
	if handler == nil {
		return fmt.Errorf("ServiceTask handler '%s' 未注册", serviceRef)
	}
	variables := mergeServiceTaskVariables(instance.Variables, task)
	variables["_commandId"] = cmd.ID
	variables["_idempotencyKey"] = cmd.IdempotencyKey
	variables["_serviceTaskOccurrence"] = occurrence
	result, err := handler.Execute(workflowCtx, nil, variables)
	if err != nil {
		return fmt.Errorf("ServiceTask %s 执行失败: %w", serviceRef, err)
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
	if occurrences[elementID] != occurrence || current.CurrentActivityID != elementID {
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
	if _, err := txc.ProcessInstance.UpdateOneID(current.ID).SetVariables(current.Variables).Save(workflowCtx); err != nil {
		return fmt.Errorf("保存 ServiceTask 输出失败: %w", err)
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
