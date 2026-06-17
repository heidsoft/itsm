package service

import (
	"testing"

	"go.uber.org/zap/zaptest"
)

// TestEvaluateCondition_FailureReturnsFalse 测试条件评估失败时返回 false
func TestEvaluateCondition_FailureReturnsFalse(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	engine := &CustomProcessEngine{
		client:     nil,
		logger:     logger,
		parser:     NewBPMNParser(),
		exprEngine: NewExpressionEngine(),
	}

	variables := map[string]interface{}{
		"status": "open",
	}

	// 使用一个无效的表达式，评估应该失败并返回 false
	result := engine.evaluateCondition(&BPMNSequenceFlow{
		ConditionExpression: &BPMNConditionExpression{
			Expression: "invalid {{{{ expression",
		},
	}, variables)

	if result {
		t.Error("无效表达式评估应返回 false，但返回了 true")
	}
}

// TestEvaluateCondition_NoConditionReturnsTrue 测试无条件表达式时返回 true
func TestEvaluateCondition_NoConditionReturnsTrue(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	engine := &CustomProcessEngine{
		client:     nil,
		logger:     logger,
		parser:     NewBPMNParser(),
		exprEngine: NewExpressionEngine(),
	}

	variables := map[string]interface{}{
		"status": "open",
	}

	// 无条件表达式，应该默认通过
	result := engine.evaluateCondition(&BPMNSequenceFlow{
		ConditionExpression: nil,
	}, variables)

	if !result {
		t.Error("无条件表达式应返回 true")
	}
}

// TestEvaluateCondition_EmptyExpressionReturnsTrue 测试空条件表达式时返回 true
func TestEvaluateCondition_EmptyExpressionReturnsTrue(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	engine := &CustomProcessEngine{
		client:     nil,
		logger:     logger,
		parser:     NewBPMNParser(),
		exprEngine: NewExpressionEngine(),
	}

	variables := map[string]interface{}{
		"status": "open",
	}

	// 空条件表达式
	result := engine.evaluateCondition(&BPMNSequenceFlow{
		ConditionExpression: &BPMNConditionExpression{
			Expression: "",
		},
	}, variables)

	if !result {
		t.Error("空条件表达式应返回 true")
	}
}

// TestEvaluateCondition_ValidExpression 测试有效表达式正常评估
func TestEvaluateCondition_ValidExpression(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	engine := &CustomProcessEngine{
		client:     nil,
		logger:     logger,
		parser:     NewBPMNParser(),
		exprEngine: NewExpressionEngine(),
	}

	variables := map[string]interface{}{
		"priority": 1,
	}

	// 有效表达式：priority == 1
	result := engine.evaluateCondition(&BPMNSequenceFlow{
		ConditionExpression: &BPMNConditionExpression{
			Expression: "priority == 1",
		},
	}, variables)

	if !result {
		t.Error("priority == 1 评估应返回 true")
	}
}

// TestGatewayEngine_EvaluateExclusiveGatewayConditions_NoMatchReturnsError
// 测试排他网关无条件匹配时返回错误
func TestGatewayEngine_EvaluateExclusiveGatewayConditions_NoMatchReturnsError(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewGatewayEngine(nil, logger)

	gatewayInfo := map[string]interface{}{
		"outputs":     []string{"activity1", "activity2"},
		"conditions":  []string{"priority > 100", "priority < 10"},
		"defaultFlow": "",
	}

	req := &GatewayExecutionRequest{
		Variables: map[string]interface{}{
			"priority": 50, // 不满足任何条件
		},
	}

	_, err := engine.evaluateExclusiveGatewayConditions(nil, req, gatewayInfo)
	if err == nil {
		t.Error("排他网关无条件匹配且无默认流时，应返回错误")
	}
}

// TestGatewayEngine_EvaluateExclusiveGatewayConditions_DefaultFlow
// 测试排他网关无条件匹配但有默认流时返回默认流
func TestGatewayEngine_EvaluateExclusiveGatewayConditions_DefaultFlow(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewGatewayEngine(nil, logger)

	gatewayInfo := map[string]interface{}{
		"outputs":     []string{"activity1", "activity2"},
		"conditions":  []string{"priority > 100", "priority < 10"},
		"defaultFlow": "activity2",
	}

	req := &GatewayExecutionRequest{
		Variables: map[string]interface{}{
			"priority": 50, // 不满足任何条件
		},
	}

	result, err := engine.evaluateExclusiveGatewayConditions(nil, req, gatewayInfo)
	if err != nil {
		t.Errorf("有默认流时不应返回错误: %v", err)
	}
	if result != "activity2" {
		t.Errorf("期望返回默认流 activity2, 实际 %s", result)
	}
}

// TestGatewayEngine_EvaluateCondition_SimpleComparison
// 测试网关引擎的简单条件评估
func TestGatewayEngine_EvaluateCondition_SimpleComparison(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewGatewayEngine(nil, logger)

	variables := map[string]interface{}{
		"status": "approved",
	}

	// 简单的 == 比较
	result := engine.evaluateCondition("status == approved", variables)
	if !result {
		t.Error("status == approved 应返回 true")
	}

	// 不匹配的条件
	result = engine.evaluateCondition("status == rejected", variables)
	if result {
		t.Error("status == rejected 应返回 false")
	}
}

// TestGatewayEngine_EvaluateCondition_BooleanTrue 测试 true 字面量
func TestGatewayEngine_EvaluateCondition_BooleanTrue(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewGatewayEngine(nil, logger)

	result := engine.evaluateCondition("true", nil)
	if !result {
		t.Error("true 应返回 true")
	}

	result = engine.evaluateCondition("false", nil)
	if result {
		t.Error("false 应返回 false")
	}
}
