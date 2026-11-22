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
		Resource:    def.Resource,
		Action:      def.Action,
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
		out, err := projectOutputSchema(def.ResultSchema)
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

// projectOutputSchema 把工具的结果 schema 归一为 MCP 允许的形态。
//
// 🔴 真实客户端联调抓到的兼容性缺陷（2026-11-18）：
//
// MCP 规范规定 outputSchema 描述的是 `structuredContent` 字段，而
// `structuredContent` **必须是 JSON 对象**；官方 SDK 的 zod schema 更严格——
// `outputSchema.type` 必须是**字面量 "object"**：
//
//	outputSchema: z.ZodOptional<z.ZodObject<{
//	    type: z.ZodLiteral<"object">;   ← 不是 z.ZodString
//	    ...
//
// 而 ITSM 既有工具里有返回数组的（如 list_kb 返回条目数组，其 ResultSchema
// 是 {"type":"array"}）。直接透传会让**整个 tools/list 响应被客户端拒绝**：
// SDK 在 zod 校验阶段抛 ZodError，工具面一个都加载不出来——不是个别工具
// 失效，是整条通道在真实客户端里不可用。
//
// 修法：非 object 型（array 等）的结果 schema 包成对象形态，
// 用 `items` 承载原 schema：
//
//	{"type":"array"} → {"type":"object","properties":{"items":{"type":"array"}},"required":["items"]}
//
// 这样既满足规范，又保留了原结构信息（不是简单丢弃）。
// 实际返回的 structuredContent 也按同一形状给出（见 service.go 的
// wrapStructuredContent），保证 schema 与内容一致。
func projectOutputSchema(raw map[string]interface{}) (map[string]any, error) {
	out, err := projectSchema(raw, "resultSchema")
	if err != nil {
		return nil, err
	}
	// 已是 object 型：原样使用。
	if t, ok := out["type"].(string); ok && t == "object" {
		return out, nil
	}
	// 非 object 型：包一层 items，保留原 schema 作为其值类型。
	wrapped := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": out,
		},
		"required": []string{"items"},
	}
	return wrapped, nil
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
