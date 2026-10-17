package bpmn

import (
	"fmt"
	"sort"
	"strings"

	"itsm-backend/ent"
)

// 本文件收敛「服务任务动作（variables.action）」的校验与报错，
// 供 incident/ticket/change 三个 ServiceTaskHandler 共用。
//
// 背景（2026-10-18 功能点验证缺陷 P1-6）：
//   三个 handler 的 Execute 都从 variables["action"] 取动作，取不到就是空串，
//   落到 default 分支后只输出 `未知的服务任务动作: `——冒号后面是空的。
//   实测日志里就是这样：
//     ServiceTask incident_task 执行失败: 未知的服务任务动作:
//   运维拿不到任何定位线索（哪个流程、哪个实例、哪个节点），
//   而 commandbus 又把它当可重试错误重试 8 次后进 dead_letter，噪声持续放大。
//
//   更根本的是三个 handler 的 Validate() 都是无条件 `return nil`，
//   且**生产代码从不调用 Validate**（全仓只有测试调用），
//   所以「没配 action 的 serviceTask」在部署期完全合法，只在运行时才炸。
//   本次不擅自把 Validate 接进部署链路（涉及 BPMN 部署路径的契约变更），
//   但把 Validate 从空实现改成真校验，并让运行时错误自带定位信息——
//   这样一旦有人接上 Validate 就立即生效，不接也能靠日志定位。

// serviceTaskActionError 构造「action 缺失 / 未知」的可定位错误。
//
// allowed 是该 handler 支持的完整动作集，会排序后写进错误信息，
// 让调用方不必翻代码就能知道该填什么。
func serviceTaskActionError(handlerType string, task *ent.ProcessTask, action string, allowed []string) error {
	sorted := append([]string(nil), allowed...)
	sort.Strings(sorted)

	where := "task=<nil>"
	if task != nil {
		where = fmt.Sprintf(
			"task=%s processInstance=%d processDefinition=%s taskDefinition=%s",
			task.TaskID, task.ProcessInstanceID,
			task.ProcessDefinitionKey, task.TaskDefinitionKey,
		)
	}

	if strings.TrimSpace(action) == "" {
		return fmt.Errorf(
			"服务任务缺少 action 配置（handler=%s, %s）："+
				"请在 BPMN 定义中为该 serviceTask 设置 variables.action，可选值 [%s]",
			handlerType, where, strings.Join(sorted, ", "),
		)
	}
	return fmt.Errorf(
		"未知的服务任务动作 %q（handler=%s, %s）：可选值 [%s]",
		action, handlerType, where, strings.Join(sorted, ", "),
	)
}

// validateServiceTaskAction 校验 config 里的 action 是否在允许集合内。
// 供各 handler 的 Validate 复用；config 为 nil 或 action 缺失时返回明确错误。
func validateServiceTaskAction(handlerType string, config map[string]interface{}, allowed []string) error {
	action, _ := config["action"].(string)
	if strings.TrimSpace(action) == "" {
		sorted := append([]string(nil), allowed...)
		sort.Strings(sorted)
		return fmt.Errorf(
			"服务任务缺少 action 配置（handler=%s）：可选值 [%s]",
			handlerType, strings.Join(sorted, ", "),
		)
	}
	for _, a := range allowed {
		if a == action {
			return nil
		}
	}
	sorted := append([]string(nil), allowed...)
	sort.Strings(sorted)
	return fmt.Errorf(
		"未知的服务任务动作 %q（handler=%s）：可选值 [%s]",
		action, handlerType, strings.Join(sorted, ", "),
	)
}
