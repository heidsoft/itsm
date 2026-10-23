package service

import (
	"strings"
	"testing"
)

// aiTools 的 argsSchema 是 agent 构造调用参数的唯一契约。字段一旦使用
// snake_case，模型就会照着发第二套方言，与 HTTP API 的 camelCase 冲突。
func TestToolSchemasUseCamelCaseArguments(t *testing.T) {
	reg := NewToolRegistry(nil, nil, nil, nil)
	seen := 0
	for _, td := range reg.ListTools() {
		props, ok := td.ArgsSchema["properties"].(map[string]interface{})
		if !ok {
			continue
		}
		for name := range props {
			seen++
			if strings.Contains(name, "_") {
				t.Errorf("tool %s 的参数字段 %q 必须是 camelCase", td.Name, name)
			}
		}
		required, _ := td.ArgsSchema["required"].([]string)
		for _, name := range required {
			if _, declared := props[name]; !declared {
				t.Errorf("tool %s 的 required 字段 %q 未在 properties 中声明", td.Name, name)
			}
			if strings.Contains(name, "_") {
				t.Errorf("tool %s 的 required 字段 %q 必须是 camelCase", td.Name, name)
			}
		}
	}
	if seen == 0 {
		t.Fatal("未扫描到任何参数字段，测试夹具失效")
	}
}

// 写工具的 schema 必须如实声明 Execute 实际读取的字段，
// 否则 agent 只能靠猜——create_ticket 曾经只声明 ciId，连 title 都查不到。
func TestWriteToolSchemasDeclareExecutedArguments(t *testing.T) {
	reg := NewToolRegistry(nil, nil, nil, nil)
	want := map[string][]string{
		"create_ticket": {"title", "description", "priority", "type", "category",
			"requesterId", "assigneeId", "ticketTypeId", "ciId"},
		"update_ticket": {"ticketId", "status", "priority", "resolution", "assigneeId"},
	}
	byName := make(map[string]ToolDefinition)
	for _, td := range reg.ListTools() {
		byName[td.Name] = td
	}
	for tool, fields := range want {
		props, ok := byName[tool].ArgsSchema["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("tool %s 缺少 properties", tool)
		}
		for _, field := range fields {
			if _, ok := props[field]; !ok {
				t.Errorf("tool %s 未声明 Execute 读取的字段 %q", tool, field)
			}
		}
	}
}
