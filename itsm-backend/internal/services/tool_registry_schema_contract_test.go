package service

import (
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/cirelationship"
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

// create_ci_relationship 的 strength 枚举曾写成 strong/medium/weak，
// 而 cirelationship.strength 是 ent Enum(critical/high/medium/low)。
// Agent 照 schema 传 "strong" 会被 PG check constraint 拒绝，
// handler 兜底成 500「创建CI关系失败」——调用方无法区分「参数错」与「后端故障」。
//
// 本测试把工具声明的枚举钉在 DB 单一源上（经 cirelationship.StrengthValidator 校验），
// 任何一侧漂移都会变红。
func TestCreateCIRelationshipToolStrengthEnumMatchesDBEnum(t *testing.T) {
	reg := NewToolRegistry(nil, nil, nil, nil)
	tools := reg.ListTools()
	var tool ToolDefinition
	found := false
	for _, td := range tools {
		if td.Name == "create_ci_relationship" {
			tool, found = td, true
			break
		}
	}
	if !found {
		t.Fatal("未找到 create_ci_relationship 工具声明")
	}
	props, ok := tool.ArgsSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("create_ci_relationship 缺少 properties")
	}
	raw, ok := props["strength"].(map[string]interface{})
	if !ok {
		t.Fatal("create_ci_relationship 未声明 strength 参数")
	}
	enumAny, ok := raw["enum"].([]any)
	if !ok || len(enumAny) == 0 {
		t.Fatal("create_ci_relationship 的 strength 必须声明 enum（否则 Agent 只能猜）")
	}

	declared := make(map[string]bool, len(enumAny))
	for _, v := range enumAny {
		s, _ := v.(string)
		if s == "" {
			t.Fatalf("strength enum 含非字符串值: %#v", v)
		}
		declared[s] = true
	}

	// dto 侧词汇（符号引用，避免字符串字面量在这份测试里二次硬编码）。
	vocabulary := []dto.RelationshipStrength{
		dto.StrengthCritical, dto.StrengthHigh, dto.StrengthMedium, dto.StrengthLow,
	}
	dbAccepted := make(map[string]bool, len(vocabulary))
	for _, v := range vocabulary {
		// 若 dto 与 ent Enum 漂移，这里先红：dto 常量必须是 DB 认可的值。
		if err := cirelationship.StrengthValidator(cirelationship.Strength(v)); err != nil {
			t.Errorf("dto 值 %q 不在 DB enum 内（dto 与 ent schema 漂移）: %v", v, err)
			continue
		}
		dbAccepted[string(v)] = true
	}

	for s := range declared {
		if !dbAccepted[s] {
			t.Errorf("工具声明的 strength=%q 不是 DB enum 值，Agent 照此调用会撞 check constraint 变成 500", s)
		}
	}
	for s := range dbAccepted {
		if !declared[s] {
			t.Errorf("DB enum 值 %q 未在工具 schema 中声明，Agent 无法写入该档", s)
		}
	}
}

// 写工具的 schema 必须如实声明 Execute 实际读取的字段，
// 否则 agent 只能靠猜——create_ticket 曾经只声明 ciId，连 title 都查不到。
func TestWriteToolSchemasDeclareExecutedArguments(t *testing.T) {
	reg := NewToolRegistry(nil, nil, nil, nil)
	want := map[string][]string{
		"create_ticket": {
			"title", "description", "priority", "type", "category",
			"requesterId", "assigneeId", "ticketTypeId", "ciId",
		},
		"update_ticket": {"ticketId", "status", "priority", "resolution", "assigneeId", "version"},
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
