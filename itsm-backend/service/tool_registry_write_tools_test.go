package service

import (
	"testing"
)

// TestToolRegistryWriteToolDefinitions 验证写工具（create_ticket/update_ticket/create_ticket_type）
// 已正确注册为需审批的写操作（ReadOnly=false，且拥有独立的 resource/action）。
// 这是 AI 写路径打通的基础：聊天 LLM 与 agent 路径都依赖这些定义判断RBAC与审批流。
func TestToolRegistryWriteToolDefinitions(t *testing.T) {
	reg := NewToolRegistry(nil, nil, nil, nil)
	tools := reg.ListTools()

	byName := make(map[string]ToolDefinition, len(tools))
	for _, td := range tools {
		byName[td.Name] = td
	}

	cases := []struct {
		name     string
		resource string
		action   string
	}{
		{"create_ticket", "ticket", "write"},
		{"update_ticket", "ticket", "write"},
		{"create_ticket_type", "ticket_type", "write"},
	}

	for _, c := range cases {
		td, ok := byName[c.name]
		if !ok {
			t.Fatalf("tool %s 未注册", c.name)
		}
		if td.ReadOnly {
			t.Errorf("tool %s 应为写操作(ReadOnly=false)，实际 ReadOnly=true", c.name)
		}
		if td.Resource != c.resource {
			t.Errorf("tool %s resource 期望 %s，实际 %s", c.name, c.resource, td.Resource)
		}
		if td.Action != c.action {
			t.Errorf("tool %s action 期望 %s，实际 %s", c.name, c.action, td.Action)
		}
	}

	// 只读工具不应被误标为写操作
	for _, ro := range []string{"get_incident_stats", "list_kb", "list_tickets", "list_cis"} {
		td, ok := byName[ro]
		if !ok {
			t.Fatalf("只读工具 %s 未注册", ro)
		}
		if !td.ReadOnly {
			t.Errorf("只读工具 %s 被错误标记为写操作", ro)
		}
	}
}
