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
