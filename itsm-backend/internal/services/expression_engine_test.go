package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpressionEngine_EnhancedFunctions(t *testing.T) {
	eng := NewExpressionEngine()

	// 正则匹配
	ok, err := eng.EvaluateCondition(`regexMatch("abc123", "^abc")`, nil)
	require.NoError(t, err)
	assert.True(t, ok, "regexMatch 应匹配前缀")

	ok, err = eng.EvaluateCondition(`regexMatch("xyz", "^abc")`, nil)
	require.NoError(t, err)
	assert.False(t, ok, "regexMatch 不应匹配")

	// 解析时间 + 天数差
	res, err := eng.Evaluate(`daysBetween(parseTime("2026-01-11T00:00:00Z"), parseTime("2026-01-01T00:00:00Z"))`, nil)
	require.NoError(t, err)
	assert.Equal(t, 10.0, res, "daysBetween 应返回 10 天")

	// 时间先后比较
	ok, err = eng.EvaluateCondition(`isAfter(parseTime("2026-01-11T00:00:00Z"), parseTime("2026-01-01T00:00:00Z"))`, nil)
	require.NoError(t, err)
	assert.True(t, ok)

	// 类型转换
	res, err = eng.Evaluate(`toInt("42")`, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(42), res)

	res, err = eng.Evaluate(`toFloat("3.14")`, nil)
	require.NoError(t, err)
	assert.Equal(t, 3.14, res)

	// coalesce / isEmpty
	res, err = eng.Evaluate(`coalesce(maybeNil, "fb")`, map[string]interface{}{"maybeNil": nil})
	require.NoError(t, err)
	assert.Equal(t, "fb", res)

	ok, err = eng.EvaluateCondition(`isEmpty("")`, nil)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = eng.EvaluateCondition(`isEmpty("x")`, nil)
	require.NoError(t, err)
	assert.False(t, ok)

	// nowISO 返回非空字符串
	res, err = eng.Evaluate(`nowISO()`, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, res)
}

func TestExpressionEngine_RegexReplace(t *testing.T) {
	eng := NewExpressionEngine()
	res, err := eng.Evaluate(`regexReplace("a-b-c", "-", "_")`, nil)
	require.NoError(t, err)
	assert.Equal(t, "a_b_c", res)
}

func TestExpressionEngine_AvgDoesNotPanic(t *testing.T) {
	eng := NewExpressionEngine()

	res, err := eng.Evaluate(`avg(scores)`, map[string]interface{}{
		"scores": []float64{10, 20, 30},
	})
	require.NoError(t, err)
	assert.Equal(t, 20.0, res)

	res, err = eng.Evaluate(`avg(empty)`, map[string]interface{}{
		"empty": []float64{},
	})
	require.NoError(t, err)
	assert.Equal(t, 0.0, res)

	ok, err := eng.EvaluateCondition(`avg(scores) > 15`, map[string]interface{}{
		"scores": []float64{10, 20, 30},
	})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestExpressionEngine_ValidateExpression_BPMNSyntax(t *testing.T) {
	eng := NewExpressionEngine()

	err := eng.ValidateExpression(`${approved == true}`)
	require.NoError(t, err, "BPMN ${} 表达式应通过校验")

	err = eng.ValidateExpression(`amount > 1000`)
	require.NoError(t, err)

	err = eng.ValidateExpression(`(`)
	require.Error(t, err, "括号不匹配应报错")

	err = eng.ValidateExpression(`{{{`)
	require.Error(t, err, "花括号不匹配应报错")
}
