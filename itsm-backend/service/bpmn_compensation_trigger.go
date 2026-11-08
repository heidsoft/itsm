package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"itsm-backend/service/bpmn"
)

// ==================== 补偿显式触发入口（管理面） ====================
//
// 此前补偿只有一个触发点：流程走到带 <compensateEventDefinition/> 的结束事件
// （见 bpmn_process_executor.go）。也就是说「业务成功后发现需要回滚」这类
// 场景没有入口，而且内置模板里根本没有补偿节点。本文件把补偿变成可被
// 管理面显式调用的能力：
//   - 补偿范围来自 ServiceTask 完成时登记的 _compensationRegistered；
//   - 支持 BPMN 补偿边界事件声明（boundaryEvent + compensateEventDefinition），
//     按 attachedToRef 匹配实例当前活动，activityRef 非空时只补偿指定活动；
//   - dry-run 只给计划，apply 在单个 ent.Tx 里执行补偿、写触发留痕、提交；
//   - 每个 occurrence 由 _compensationExecuted 幂等，重复触发不会产生二次补偿；
//   - 全程按租户过滤，跨租户实例按「不存在」处理，不泄漏、不猜测。
//
// 刻意不做的部分（与 bpmn_compensation.go 的语义子集一致）：子流程级
// compensation scope 隔离、跨实例补偿、边界事件的自动挂起/中断协议。
// 这里实现的是「按模型声明解析显式触发范围」，不是完整 BPMN 2.0 补偿事务。

// CompensationTriggerRequest 管理面显式补偿请求。
type CompensationTriggerRequest struct {
	TenantID    int
	ActorUserID int
	InstanceID  int
	// ActivityRef 为空表示补偿全部已登记且未补偿的活动；非空时只补偿该活动。
	ActivityRef string
	Reason      string
	DryRun      bool
}

// CompensationTriggerResult 补偿计划或执行结果。
type CompensationTriggerResult struct {
	DryRun       bool     `json:"dryRun"`
	Applied      bool     `json:"applied"`
	TenantID     int      `json:"tenantId"`
	InstanceID   int      `json:"instanceId"`
	InstanceKey  string   `json:"instanceKey"`
	ActivityRef  string   `json:"activityRef,omitempty"`
	BoundaryID   string   `json:"boundaryId,omitempty"`
	Targets      []string `json:"targets"`
	Message      string   `json:"message"`
	TriggerCount int      `json:"triggerCount"`
}

// TriggerCompensation 按已登记补偿范围执行（或仅预演）一次显式补偿。
func (e *CustomProcessEngine) TriggerCompensation(ctx context.Context, req CompensationTriggerRequest) (*CompensationTriggerResult, error) {
	if e == nil || e.client == nil {
		return nil, fmt.Errorf("补偿触发需要有效的流程引擎")
	}
	if req.TenantID <= 0 || req.InstanceID <= 0 {
		return nil, fmt.Errorf("补偿触发需要有效的租户与流程实例")
	}
	if req.ActorUserID <= 0 {
		return nil, fmt.Errorf("补偿触发需要有效的操作人（写入审计）")
	}
	if strings.TrimSpace(req.Reason) == "" {
		return nil, fmt.Errorf("补偿触发必须填写原因（写入审计与留痕）")
	}

	instance, err := e.client.ProcessInstance.Query().Where(
		processinstance.IDEQ(req.InstanceID),
		processinstance.TenantIDEQ(req.TenantID),
	).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("流程实例 %d 不存在或不属于租户 %d，拒绝补偿", req.InstanceID, req.TenantID)
		}
		return nil, fmt.Errorf("加载流程实例失败: %w", err)
	}

	result := &CompensationTriggerResult{
		DryRun:      req.DryRun,
		TenantID:    req.TenantID,
		InstanceID:  instance.ID,
		InstanceKey: instance.ProcessInstanceID,
		Targets:     []string{},
	}

	activityRef := strings.TrimSpace(req.ActivityRef)
	if activityRef == "" {
		boundaryID, boundaryRef, err := e.resolveCompensationBoundary(ctx, req.TenantID, instance)
		if err != nil {
			return nil, err
		}
		result.BoundaryID = boundaryID
		activityRef = boundaryRef
	}
	result.ActivityRef = activityRef

	targets, err := compensationTriggerTargets(instance.Variables, activityRef)
	if err != nil {
		return nil, err
	}
	result.Targets = targets
	if len(targets) == 0 {
		result.Message = "没有待补偿活动（可能从未登记或已全部补偿），不产生写操作"
		return result, nil
	}
	if req.DryRun {
		result.Message = fmt.Sprintf("dry-run：将按逆序补偿 %d 个活动，未写入任何数据", len(targets))
		return result, nil
	}

	// 补偿执行、触发留痕与审计同属一个事务；任一步失败整体回滚，
	// 不留下「补偿了一半」或「补偿完成但没有触发记录」的状态。
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, req.TenantID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, req.ActorUserID)

	tx, err := e.client.Tx(workflowCtx)
	if err != nil {
		return nil, fmt.Errorf("开启补偿事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txc := tx.Client()

	fresh, err := txc.ProcessInstance.Query().Where(
		processinstance.IDEQ(req.InstanceID),
		processinstance.TenantIDEQ(req.TenantID),
	).Only(workflowCtx)
	if err != nil {
		return nil, fmt.Errorf("补偿事务内重新加载流程实例失败: %w", err)
	}
	freshTargets, err := compensationTriggerTargets(fresh.Variables, activityRef)
	if err != nil {
		return nil, err
	}
	if len(freshTargets) == 0 {
		// 并发触发已抢先补偿完成：幂等返回，不重复执行。
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("提交补偿事务失败: %w", err)
		}
		result.Targets = []string{}
		result.Message = "并发触发已完成补偿，本次幂等跳过"
		return result, nil
	}

	if err := e.runCompensation(workflowCtx, txc, fresh, &BPMNCompensateEventDefinition{ActivityRef: activityRef}); err != nil {
		return nil, err
	}

	history := map[string]interface{}{
		"actorUserId": req.ActorUserID,
		"reason":      req.Reason,
		"activityRef": activityRef,
		"boundaryId":  result.BoundaryID,
		"targets":     freshTargets,
		"triggeredAt": time.Now().UTC().Format(time.RFC3339),
	}
	triggers := map[string]interface{}{}
	if raw, ok := fresh.Variables["_compensationTriggerHistory"].(map[string]interface{}); ok {
		for key, value := range raw {
			triggers[key] = value
		}
	}
	triggers[history["triggeredAt"].(string)] = history
	updated, err := updateProcessInstanceVariablesCAS(workflowCtx, txc, fresh.ID, req.TenantID, func(vars map[string]interface{}) error {
		vars["_compensationTriggerHistory"] = triggers
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("写入补偿触发留痕失败: %w", err)
	}
	fresh.Variables = updated.Variables

	actorName := ""
	if actor, actorErr := txc.User.Get(workflowCtx, req.ActorUserID); actorErr == nil {
		actorName = actor.Name
	}
	// AuditContext.ActivityID 在 schema 里有最小长度约束；「补偿全部活动」时
	// activityRef 为空，改用边界事件 ID 或稳定的触发节点名作审计定位，
	// 语义上仍能区分「整流程补偿」与「指定活动补偿」。
	auditActivityID := activityRef
	if auditActivityID == "" {
		if result.BoundaryID != "" {
			auditActivityID = result.BoundaryID
		} else {
			auditActivityID = "compensation-trigger"
		}
	}
	if err := e.auditService.RecordAuditWithClient(workflowCtx, txc, &AuditContext{
		ProcessInstanceID:    fresh.ID,
		ProcessInstanceKey:   fresh.ProcessInstanceID,
		ProcessDefinitionKey: fresh.ProcessDefinitionKey,
		ProcessDefinitionID:  fresh.ProcessDefinitionID,
		ActivityID:           auditActivityID,
		ActivityType:         "CompensationBoundary",
		Action:               AuditActionCompensated,
		UserID:               req.ActorUserID,
		UserName:             actorName,
		VariablesAfter:       fresh.Variables,
		Comment:              fmt.Sprintf("管理面显式补偿（原因：%s；边界事件：%s；目标：%v）", req.Reason, result.BoundaryID, freshTargets),
		TenantID:             req.TenantID,
	}); err != nil {
		return nil, fmt.Errorf("记录补偿触发审计失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交补偿事务失败: %w", err)
	}

	result.Applied = true
	result.Targets = freshTargets
	result.TriggerCount = 1
	result.Message = fmt.Sprintf("已按逆序补偿 %d 个活动，触发留痕与审计同事务落库", len(freshTargets))
	return result, nil
}

// compensationTriggerTargets 计算本次实际会补偿的活动（逆序、剔除已补偿）。
// activityRef 指向未登记活动时 fail-closed，与 runCompensation 的语义一致。
func compensationTriggerTargets(variables map[string]interface{}, activityRef string) ([]string, error) {
	registered := compensationEntries(variables)
	executed := intMapVariable(variables, "_compensationExecuted")
	if activityRef != "" {
		entry, ok := registered[activityRef]
		if !ok {
			return nil, fmt.Errorf("补偿目标 %s 未登记（从未完成或已补偿），拒绝继续", activityRef)
		}
		if executed[activityRef] >= entry.Occurrence {
			return []string{}, nil
		}
		return []string{activityRef}, nil
	}
	targets := []string{}
	for _, elementID := range compensationOrder(variables, registered) {
		entry := registered[elementID]
		if executed[elementID] < entry.Occurrence {
			targets = append(targets, elementID)
		}
	}
	return targets, nil
}

// resolveCompensationBoundary 从流程定义里找挂在实例当前活动上的补偿边界
// 事件声明。命中时返回 (boundaryID, activityRef)；没有声明时返回空值，
// 调用方按「补偿全部已登记活动」处理（等价于 BPMN activityRef 为空）。
func (e *CustomProcessEngine) resolveCompensationBoundary(ctx context.Context, tenantID int, instance *ent.ProcessInstance) (string, string, error) {
	if instance.ProcessDefinitionID <= 0 {
		return "", "", nil
	}
	definition, err := e.client.ProcessDefinition.Query().Where(
		processdefinition.IDEQ(instance.ProcessDefinitionID),
		processdefinition.TenantIDEQ(tenantID),
	).Only(ctx)
	if err != nil {
		return "", "", fmt.Errorf("加载补偿边界事件定义失败: %w", err)
	}
	definitions, err := e.parser.ParseXML(definition.BpmnXML)
	if err != nil {
		return "", "", fmt.Errorf("解析补偿边界事件定义失败: %w", err)
	}
	for _, process := range definitions.Processes {
		for _, boundary := range process.BoundaryEvents {
			if boundary == nil || boundary.CompensateEventDefinition == nil {
				continue
			}
			if strings.TrimSpace(boundary.AttachedToRef) != strings.TrimSpace(instance.CurrentActivityID) {
				continue
			}
			return boundary.ID, strings.TrimSpace(boundary.CompensateEventDefinition.ActivityRef), nil
		}
	}
	return "", "", nil
}
