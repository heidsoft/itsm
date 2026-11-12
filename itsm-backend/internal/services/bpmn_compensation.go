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

// ===========================================================================
// BPMN 补偿事件（Compensation）
// ===========================================================================
//
// 语义子集（与引擎能力对齐的最小闭环）：
//  1. 登记：ServiceTask 完成时把「活动 → handler + 变量快照 + occurrence」
//     写入实例变量 _compensationRegistered，并按完成顺序维护
//     _compensationOrder（JSON 映射无序，顺序必须显式记录）；
//  2. 触发：结束事件携带 <compensateEventDefinition/> 时，到达该节点即对
//     已登记活动执行逆序补偿；activityRef 非空时只补偿指定活动；
//  3. 执行：调用该活动原 handler，变量带 _compensate=true 与完成时快照，
//     每个 occurrence 只补偿一次（_compensationExecuted 幂等）；
//  4. 留痕：每次补偿写 ProcessAuditLog（action=compensated），失败即
//     fail-closed（整单回滚，不留下半补偿状态）。
//
// 刻意不做（避免伪造成支持）：子流程级 scope 隔离、补偿边界事件
// （boundaryEvent+compensateEventDefinition）的挂起/触发协议、跨实例补偿。
// isForCompensation="true" 的活动只作为补偿处理器存在，主流程跳过。

// compensationEntry 单个已登记补偿活动的快照（存于实例变量）。
type compensationEntry struct {
	Occurrence int                    `json:"occurrence"`
	ServiceRef string                 `json:"serviceRef"`
	Variables  map[string]interface{} `json:"variables"`
}

// registerCompensation 在 ServiceTask 完成事务内登记补偿范围。
// 只登记真实执行过的活动（occurrence 级幂等）；重复登记同一活动时
// 刷新快照并提升 occurrence。
//
// 快照必须是副本且剔除补偿簿记键：直接把实例变量 map 存进
// _compensationRegistered 会形成自引用（JSON 序列化报 cycle），
// 且每次登记都嵌入上一次登记的完整簿记，变量无限增长。
func (e *CustomProcessEngine) registerCompensation(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, elementID, serviceRef string, occurrence int, snapshot map[string]interface{}) error {
	updated, err := updateProcessInstanceVariablesCAS(ctx, txc, instance.ID, instance.TenantID, func(vars map[string]interface{}) error {
		registered := compensationEntries(vars)
		if _, exists := registered[elementID]; !exists {
			order := stringSliceVariable(vars, "_compensationOrder")
			order = append(order, elementID)
			vars["_compensationOrder"] = order
		}
		registered[elementID] = compensationEntry{
			Occurrence: occurrence,
			ServiceRef: serviceRef,
			Variables:  compensationSnapshot(snapshot),
		}
		// 必须存可序列化形状（map[string]interface{}）：同一请求内随后读取
		// （补偿触发）走的是内存对象，若存类型化 map，读取函数的类型断言
		// 会静默失败——补偿在落库前被跳过，JSON 往返后反而正常。
		serializable := make(map[string]interface{}, len(registered))
		for id, entry := range registered {
			serializable[id] = map[string]interface{}{
				"occurrence": entry.Occurrence,
				"serviceRef": entry.ServiceRef,
				"variables":  entry.Variables,
			}
		}
		vars["_compensationRegistered"] = serializable
		return nil
	})
	if err != nil {
		return fmt.Errorf("登记补偿范围失败: %w", err)
	}
	instance.Variables = updated.Variables
	return nil
}

// compensationSnapshot 复制业务变量快照（浅拷贝 + 剔除补偿簿记键）。
func compensationSnapshot(variables map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(variables))
	for key, value := range variables {
		switch key {
		case "_compensationRegistered", "_compensationOrder", "_compensationExecuted",
			"_serviceTaskExecutedOccurrences", "_serviceTaskReceipts", "_serviceTaskExecuting",
			"_serviceTaskAttempts", "_serviceTaskCompletedOccurrences", "_serviceTaskOccurrences":
			continue
		}
		out[key] = value
	}
	return out
}

// runCompensation 执行补偿（逆序），全部成功后才允许流程继续。
// def.ActivityRef 非空时只补偿指定活动；活动未登记 → fail-closed。
func (e *CustomProcessEngine) runCompensation(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, def *BPMNCompensateEventDefinition) error {
	if def == nil {
		return nil
	}
	registered := compensationEntries(instance.Variables)
	executed := intMapVariable(instance.Variables, "_compensationExecuted")

	targets := compensationOrder(instance.Variables, registered)
	if def.ActivityRef != "" {
		targets = []string{def.ActivityRef}
		if _, ok := registered[def.ActivityRef]; !ok {
			return fmt.Errorf("补偿目标 %s 未登记（从未完成或已补偿），拒绝继续", def.ActivityRef)
		}
	}

	actorID, _ := ctx.Value(bpmn.BPMNUserIDContextKey).(int)
	for _, elementID := range targets {
		entry := registered[elementID]
		if executed[elementID] >= entry.Occurrence {
			continue // 已补偿过：幂等跳过
		}
		if err := e.compensateOne(ctx, txc, instance, elementID, entry, actorID); err != nil {
			return err
		}
		executed[elementID] = entry.Occurrence
	}
	updated, err := updateProcessInstanceVariablesCAS(ctx, txc, instance.ID, instance.TenantID, func(vars map[string]interface{}) error {
		vars["_compensationExecuted"] = executed
		return nil
	})
	if err != nil {
		return fmt.Errorf("保存补偿执行状态失败: %w", err)
	}
	instance.Variables = updated.Variables
	return nil
}

// compensateOne 补偿单个活动：调用原 handler（_compensate=true），写审计。
func (e *CustomProcessEngine) compensateOne(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, elementID string, entry compensationEntry, actorID int) error {
	handler := e.callbackRegistry.GetHandler(entry.ServiceRef)
	if handler == nil {
		return fmt.Errorf("补偿活动 %s 的 handler '%s' 未注册", elementID, entry.ServiceRef)
	}
	variables := map[string]interface{}{}
	for k, v := range entry.Variables {
		variables[k] = v
	}
	variables["_compensate"] = true
	variables["_compensatingActivity"] = elementID

	result, err := handler.Execute(ctx, nil, variables)
	if err != nil {
		return fmt.Errorf("补偿活动 %s 执行失败: %w", elementID, err)
	}
	if result != nil {
		for k, v := range result.OutputVars {
			instance.Variables[k] = v
		}
	}
	actorName := ""
	if actorID > 0 {
		if actor, aerr := txc.User.Get(ctx, actorID); aerr == nil {
			actorName = actor.Name
		}
	}
	if err := e.auditService.RecordAuditWithClient(ctx, txc, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           elementID,
		ActivityType:         "ServiceTask",
		Action:               AuditActionCompensated,
		UserID:               actorID,
		UserName:             actorName,
		VariablesBefore:      entry.Variables,
		VariablesAfter:       variables,
		Comment:              fmt.Sprintf("补偿活动 %s（occurrence=%d）", elementID, entry.Occurrence),
		TenantID:             instance.TenantID,
	}); err != nil {
		return fmt.Errorf("记录补偿审计失败: %w", err)
	}
	e.logger.Infow("BPMN 补偿已执行",
		"instance_id", instance.ID, "tenant_id", instance.TenantID,
		"element_id", elementID, "service_ref", entry.ServiceRef, "occurrence", entry.Occurrence)
	return nil
}

// compensationEntries 从实例变量读已登记补偿范围。
func compensationEntries(variables map[string]interface{}) map[string]compensationEntry {
	out := map[string]compensationEntry{}
	raw, ok := variables["_compensationRegistered"].(map[string]interface{})
	if !ok {
		return out
	}
	for elementID, value := range raw {
		entryMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		entry := compensationEntry{}
		entry.Occurrence = compensationInt(entryMap["occurrence"])
		if serviceRef, ok := entryMap["serviceRef"].(string); ok {
			entry.ServiceRef = serviceRef
		}
		if vars, ok := entryMap["variables"].(map[string]interface{}); ok {
			entry.Variables = vars
		}
		out[elementID] = entry
	}
	return out
}

// compensationInt 宽松解析整数：内存态是 int，JSON 往返态是 float64。
// 只认一种会把另一种解析成 0——occurrence=0 会被幂等判断误判为
// 「已补偿」而静默跳过，正是补偿不执行的根因。
func compensationInt(value interface{}) int {
	switch number := value.(type) {
	case int:
		return number
	case int64:
		return int(number)
	case float64:
		return int(number)
	default:
		return 0
	}
}

// compensationOrder 补偿执行顺序：后完成的先补偿（栈序）。
// 顺序来源为 _compensationOrder（完成顺序）；无该列表时按元素 ID 排序，
// 保证行为确定（不依赖 map 遍历顺序）。
func compensationOrder(variables map[string]interface{}, registered map[string]compensationEntry) []string {
	order := stringSliceVariable(variables, "_compensationOrder")
	if len(order) == 0 {
		order = make([]string, 0, len(registered))
		for elementID := range registered {
			order = append(order, elementID)
		}
	}
	// 逆序：后登记（后完成）的先补偿。
	reversed := make([]string, 0, len(order))
	for i := len(order) - 1; i >= 0; i-- {
		if _, ok := registered[order[i]]; ok {
			reversed = append(reversed, order[i])
		}
	}
	return reversed
}

// pendingCompensationTargets 返回尚未补偿的已登记活动（盘点/诊断用）。
func pendingCompensationTargets(variables map[string]interface{}) []string {
	registered := compensationEntries(variables)
	executed := intMapVariable(variables, "_compensationExecuted")
	var pending []string
	for elementID, entry := range registered {
		if executed[elementID] < entry.Occurrence {
			pending = append(pending, elementID)
		}
	}
	return pending
}

// stringSliceVariable 从流程变量读字符串切片。
// 同时容忍内存态（[]string）与 JSON 往返态（[]interface{}）。
func stringSliceVariable(variables map[string]interface{}, key string) []string {
	switch raw := variables[key].(type) {
	case []string:
		return append([]string(nil), raw...)
	case []interface{}:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

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
