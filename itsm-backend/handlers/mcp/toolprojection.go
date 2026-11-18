package mcp

import (
	"encoding/json"
	"fmt"

	"itsm-backend/dto"
)

// ProjectTool 把 ITSM 内部的工具定义投影为 MCP 工具描述。
//
// 这是本包最关键的一步：**协议形状转换，不是能力翻译**。
//
// 映射关系：
//
//	ToolDefinition.Name        → Tool.Name
//	ToolDefinition.Description → Tool.Description
//	ToolDefinition.ArgsSchema  → Tool.InputSchema（须是合法 JSON Schema object）
//	ToolDefinition.ResultSchema→ Tool.OutputSchema
//	ToolDefinition.ReadOnly    → Tool.Annotations.ReadOnlyHint
//
// 安全语义：ReadOnly 是 ITSM 侧的执行事实（写工具恒走审批流），
// 投影成 readOnlyHint 后客户端可以据此决定是否提示用户确认；
// 但真正的强制仍在服务端权限链上，标注只是提示，不是边界。
//
// ArgsSchema 缺失或非法时返回错误而不是静默补空 schema ——
// 工具入参契约对外可见，静默降级会让客户端以为「无参可调」，
// 调用时才在业务层炸掉，错误位置比这里晚得多。
func ProjectTool(def dto.ToolDefinition) (Tool, error) {
	if def.Name == "" {
		return Tool{}, fmt.Errorf("工具定义缺少 name，无法投影为 MCP 工具")
	}
	input, err := projectSchema(def.ArgsSchema, "argsSchema")
	if err != nil {
		return Tool{}, fmt.Errorf("工具 %q 的 argsSchema 非法: %w", def.Name, err)
	}
	tool := Tool{
		Name:        def.Name,
		Description: def.Description,
		InputSchema: input,
		Annotations: &ToolAnnotations{
			ReadOnlyHint: def.ReadOnly,
			// 写工具对 ITSM 域数据有副作用，且不可通过简单重放保证幂等：
			// 标注为破坏性 + 非幂等，让客户端在调用前显式确认。
			DestructiveHint: !def.ReadOnly,
			IdempotentHint:  def.ReadOnly,
			// 工具只触及本租户内部域数据，不触达开放互联网。
			OpenWorldHint: false,
		},
	}
	if len(def.ResultSchema) > 0 {
		out, err := projectSchema(def.ResultSchema, "resultSchema")
		if err != nil {
			return Tool{}, fmt.Errorf("工具 %q 的 resultSchema 非法: %w", def.Name, err)
		}
		tool.OutputSchema = out
	}
	return tool, nil
}

// projectSchema 校验并归一一个 JSON Schema。
//
// MCP 规范要求 inputSchema 是 JSON Schema object 类型。
// 传入的 schema 若是 map[string]any 以外的形态（历史上遗留的裸 string 等），
// 这里统一包成 {"type":"object"}，宁可信息少也不能给客户端非法 schema。
func projectSchema(raw map[string]interface{}, field string) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{"type": "object"}, nil
	}
	// 确保 JSON 可往返：map 里的值可能是结构体等不可序列化形态。
	if _, err := json.Marshal(raw); err != nil {
		return nil, fmt.Errorf("%s 无法序列化为 JSON: %w", field, err)
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	if _, ok := out["type"]; !ok {
		out["type"] = "object"
	}
	return out, nil
}

// ProjectTools 批量投影，过滤掉非法项而不是整批失败。
//
// 单个工具的 schema 异常不应该让整个工具面不可用 —— 客户端仍应能看到
// 其余可用工具。被过滤掉的项通过返回值告知调用方（用于日志与告警）。
func ProjectTools(defs []dto.ToolDefinition) ([]Tool, []error) {
	tools := make([]Tool, 0, len(defs))
	var errs []error
	for _, def := range defs {
		t, err := ProjectTool(def)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		tools = append(tools, t)
	}
	return tools, errs
}
