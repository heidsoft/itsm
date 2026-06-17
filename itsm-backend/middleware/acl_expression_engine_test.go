package middleware

import (
	"testing"
)

// TestACLEvaluate_EmptyStringReturnsTrue 空字符串返回 true
func TestACLEvaluate_EmptyStringReturnsTrue(t *testing.T) {
	engine := NewACLExpressionEngine()
	vars := map[string]interface{}{}

	result := engine.Evaluate("", vars)
	if !result {
		t.Error("空字符串应返回 true")
	}
}

// TestACLEvaluate_SimpleIntComparison 简单整数比较
func TestACLEvaluate_SimpleIntComparison(t *testing.T) {
	engine := NewACLExpressionEngine()
	vars := map[string]interface{}{
		"ctx.user_id": 1,
	}

	// ctx.user_id == 1
	result := engine.Evaluate("ctx.user_id == 1", vars)
	if !result {
		t.Error("ctx.user_id == 1 应返回 true")
	}

	// ctx.user_id != 2
	result = engine.Evaluate("ctx.user_id != 2", vars)
	if !result {
		t.Error("ctx.user_id != 2 应返回 true")
	}

	// ctx.user_id == 2
	result = engine.Evaluate("ctx.user_id == 2", vars)
	if result {
		t.Error("ctx.user_id == 2 应返回 false")
	}
}

// TestACLEvaluate_StringComparison 字符串比较
func TestACLEvaluate_StringComparison(t *testing.T) {
	engine := NewACLExpressionEngine()
	vars := map[string]interface{}{
		"ctx.role": "admin",
	}

	// ctx.role == "admin"
	result := engine.Evaluate(`ctx.role == "admin"`, vars)
	if !result {
		t.Error(`ctx.role == "admin" 应返回 true`)
	}

	// ctx.role != "user"
	result = engine.Evaluate(`ctx.role != "user"`, vars)
	if !result {
		t.Error(`ctx.role != "user" 应返回 true`)
	}

	// ctx.role == "user"
	result = engine.Evaluate(`ctx.role == "user"`, vars)
	if result {
		t.Error(`ctx.role == "user" 应返回 false`)
	}
}

// TestACLEvaluate_LogicAnd 逻辑与
func TestACLEvaluate_LogicAnd(t *testing.T) {
	engine := NewACLExpressionEngine()
	vars := map[string]interface{}{
		"ctx.role":      "admin",
		"ctx.tenant_id": 1,
	}

	// ctx.role == "admin" && ctx.tenant_id == 1
	result := engine.Evaluate(`ctx.role == "admin" && ctx.tenant_id == 1`, vars)
	if !result {
		t.Error(`ctx.role == "admin" && ctx.tenant_id == 1 应返回 true`)
	}

	// ctx.role == "admin" && ctx.tenant_id == 2
	result = engine.Evaluate(`ctx.role == "admin" && ctx.tenant_id == 2`, vars)
	if result {
		t.Error(`ctx.role == "admin" && ctx.tenant_id == 2 应返回 false`)
	}
}

// TestACLEvaluate_LogicOr 逻辑或
func TestACLEvaluate_LogicOr(t *testing.T) {
	engine := NewACLExpressionEngine()
	vars := map[string]interface{}{
		"ctx.role":      "admin",
		"ctx.tenant_id": 1,
	}

	// ctx.role == "user" || ctx.tenant_id == 1
	result := engine.Evaluate(`ctx.role == "user" || ctx.tenant_id == 1`, vars)
	if !result {
		t.Error(`ctx.role == "user" || ctx.tenant_id == 1 应返回 true`)
	}
}

// TestACLEvaluate_LogicNot 逻辑非
func TestACLEvaluate_LogicNot(t *testing.T) {
	engine := NewACLExpressionEngine()
	vars := map[string]interface{}{
		"ctx.role": "user",
	}

	// !(ctx.role == "admin")
	result := engine.Evaluate(`!(ctx.role == "admin")`, vars)
	if !result {
		t.Error(`!(ctx.role == "admin") 当 role="user" 时应返回 true`)
	}
}

// TestACLEvaluate_InvalidExpression 无效表达式返回 false
func TestACLEvaluate_InvalidExpression(t *testing.T) {
	engine := NewACLExpressionEngine()
	vars := map[string]interface{}{}

	result := engine.Evaluate("{{{{invalid", vars)
	if result {
		t.Error("无效表达式应返回 false")
	}
}

// TestACLEvaluate_MissingVariable 缺失变量返回 false
func TestACLEvaluate_MissingVariable(t *testing.T) {
	engine := NewACLExpressionEngine()
	vars := map[string]interface{}{
		"ctx.user_id": 1,
	}

	// ctx.nonexistent == 1 - 变量不存在
	result := engine.Evaluate("ctx.nonexistent == 1", vars)
	if result {
		t.Error("缺失变量的表达式应返回 false")
	}
}

// TestACLEvaluate_ComparisonOperators 各种比较运算符
func TestACLEvaluate_ComparisonOperators(t *testing.T) {
	engine := NewACLExpressionEngine()
	vars := map[string]interface{}{
		"ctx.tenant_id": 5,
	}

	tests := []struct {
		expr     string
		expected bool
	}{
		{"ctx.tenant_id > 3", true},
		{"ctx.tenant_id < 3", false},
		{"ctx.tenant_id >= 5", true},
		{"ctx.tenant_id <= 5", true},
		{"ctx.tenant_id > 10", false},
		{"ctx.tenant_id >= 6", false},
	}

	for _, tt := range tests {
		result := engine.Evaluate(tt.expr, vars)
		if result != tt.expected {
			t.Errorf("表达式 %s: 期望 %v, 实际 %v", tt.expr, tt.expected, result)
		}
	}
}

// TestACLEvaluate_ComplexExpression 复杂组合表达式
func TestACLEvaluate_ComplexExpression(t *testing.T) {
	engine := NewACLExpressionEngine()
	vars := map[string]interface{}{
		"ctx.user_id":       1,
		"ctx.tenant_id":     1,
		"ctx.role":          "admin",
		"ctx.resource_type": "ticket",
	}

	// (ctx.role == "admin" || ctx.role == "security") && ctx.tenant_id == 1
	result := engine.Evaluate(`(ctx.role == "admin" || ctx.role == "security") && ctx.tenant_id == 1`, vars)
	if !result {
		t.Error("复杂组合表达式应返回 true")
	}
}

// TestEvaluateACLScript_Integration 集成测试：通过 EvaluateACLScript 函数测试
func TestEvaluateACLScript_Integration(t *testing.T) {
	// 空字符串
	if !EvaluateACLScript("", ACLScriptContext{}) {
		t.Error("空 ACL 脚本应放行")
	}

	// 用户ID匹配
	if !EvaluateACLScript("ctx.user_id == 1", ACLScriptContext{UserID: 1}) {
		t.Error("用户ID=1 时 ctx.user_id == 1 应放行")
	}

	// 角色匹配
	if !EvaluateACLScript(`ctx.role == "admin"`, ACLScriptContext{Role: "admin"}) {
		t.Error("角色=admin 时应放行")
	}

	// 角色不匹配
	if EvaluateACLScript(`ctx.role == "admin"`, ACLScriptContext{Role: "user"}) {
		t.Error("角色=user 时 ctx.role == admin 应拒绝")
	}

	// 无效脚本
	if EvaluateACLScript("{{{invalid", ACLScriptContext{}) {
		t.Error("无效脚本应拒绝")
	}
}

// TestACLScriptContext_ResourceID 测试 resource_id 变量
func TestACLScriptContext_ResourceID(t *testing.T) {
	// resource_id 为整数
	if !EvaluateACLScript("ctx.resource_id == 42", ACLScriptContext{ResourceID: 42}) {
		t.Error("resource_id == 42 应放行")
	}
}
