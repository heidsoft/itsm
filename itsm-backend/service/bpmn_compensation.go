package service

import (
	"context"
	"fmt"

	"itsm-backend/ent"
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
		case "_compensationRegistered", "_compensationOrder", "_compensationExecuted":
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

// hasCompensationBoundaryEvents 检查流程是否声明了补偿边界事件。
// 本引擎不执行边界补偿触发协议——发布期提示，避免模板作者误以为已支持。
func hasCompensationBoundaryEvents(process *BPMNProcess) bool {
	for _, event := range process.BoundaryEvents {
		if event.CompensateEventDefinition != nil {
			return true
		}
	}
	return false
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
