package service

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/expr-lang/expr"
)

// ExpressionEngine 表达式引擎
type ExpressionEngine struct {
	// 可选的外部函数映射
	Functions map[string]interface{}
}

// NewExpressionEngine 创建表达式引擎
func NewExpressionEngine() *ExpressionEngine {
	return &ExpressionEngine{
		Functions: make(map[string]interface{}),
	}
}

// RegisterFunction 注册自定义函数
func (e *ExpressionEngine) RegisterFunction(name string, fn interface{}) {
	e.Functions[name] = fn
}

// Evaluate 评估表达式
func (e *ExpressionEngine) Evaluate(expression string, variables map[string]interface{}) (interface{}, error) {
	if expression == "" {
		return true, nil
	}

	// 构建环境
	env := make(map[string]interface{})
	for k, v := range variables {
		env[k] = v
	}

	// 添加内置函数
	e.registerBuiltinFunctions(env)

	// 添加自定义函数
	for name, fn := range e.Functions {
		env[name] = fn
	}

	// 编译并执行表达式
	program, err := expr.Compile(expression, expr.Env(env))
	if err != nil {
		return nil, fmt.Errorf("表达式编译失败: %w", err)
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return nil, fmt.Errorf("表达式执行失败: %w", err)
	}

	return output, nil
}

// EvaluateCondition 评估条件表达式（返回布尔值）
func (e *ExpressionEngine) EvaluateCondition(expression string, variables map[string]interface{}) (bool, error) {
	result, err := e.Evaluate(expression, variables)
	if err != nil {
		return false, err
	}

	// 转换为布尔值
	switch v := result.(type) {
	case bool:
		return v, nil
	case string:
		return strings.ToLower(v) == "true" || v == "1", nil
	case int, int64, float64:
		return reflect.ValueOf(v).Convert(reflect.TypeOf(float64(0))).Float() != 0, nil
	default:
		return false, nil
	}
}

// EvaluateNumeric 评估数值表达式
func (e *ExpressionEngine) EvaluateNumeric(expression string, variables map[string]interface{}) (float64, error) {
	result, err := e.Evaluate(expression, variables)
	if err != nil {
		return 0, err
	}

	switch v := result.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("无法将结果转换为数值: %v", result)
	}
}

// registerBuiltinFunctions 注册内置函数
func (e *ExpressionEngine) registerBuiltinFunctions(env map[string]interface{}) {
	// 字符串函数
	env["contains"] = func(s, substr string) bool {
		return strings.Contains(s, substr)
	}
	env["startsWith"] = func(s, prefix string) bool {
		return strings.HasPrefix(s, prefix)
	}
	env["endsWith"] = func(s, suffix string) bool {
		return strings.HasSuffix(s, suffix)
	}
	env["upper"] = strings.ToUpper
	env["lower"] = strings.ToLower
	env["trim"] = strings.TrimSpace
	env["len"] = func(s string) int {
		return len(s)
	}
	env["replace"] = func(s, old, new string) string {
		return strings.ReplaceAll(s, old, new)
	}
	env["split"] = func(s, sep string) []string {
		return strings.Split(s, sep)
	}
	env["join"] = func(arr []string, sep string) string {
		return strings.Join(arr, sep)
	}

	// 数学函数
	env["max"] = func(a, b float64) float64 {
		if a > b {
			return a
		}
		return b
	}
	env["min"] = func(a, b float64) float64 {
		if a < b {
			return a
		}
		return b
	}
	env["abs"] = func(a float64) float64 {
		if a < 0 {
			return -a
		}
		return a
	}
	env["floor"] = func(a float64) float64 {
		return float64(int(a))
	}
	env["ceil"] = func(a float64) float64 {
		if float64(int(a)) == a {
			return a
		}
		return float64(int(a) + 1)
	}
	env["round"] = func(a float64) float64 {
		return float64(int(a + 0.5))
	}
	env["sum"] = func(arr []float64) float64 {
		total := 0.0
		for _, v := range arr {
			total += v
		}
		return total
	}
	env["avg"] = func(arr []float64) float64 {
		if len(arr) == 0 {
			return 0
		}
		return e.Functions["sum"].(func([]float64) float64)(arr) / float64(len(arr))
	}

	// 比较函数
	env["gt"] = func(a, b float64) bool {
		return a > b
	}
	env["gte"] = func(a, b float64) bool {
		return a >= b
	}
	env["lt"] = func(a, b float64) bool {
		return a < b
	}
	env["lte"] = func(a, b float64) bool {
		return a <= b
	}
	env["eq"] = func(a, b interface{}) bool {
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	}
	env["neq"] = func(a, b interface{}) bool {
		return fmt.Sprintf("%v", a) != fmt.Sprintf("%v", b)
	}

	// 逻辑函数
	env["and"] = func(bools ...bool) bool {
		for _, b := range bools {
			if !b {
				return false
			}
		}
		return true
	}
	env["or"] = func(bools ...bool) bool {
		for _, b := range bools {
			if b {
				return true
			}
		}
		return false
	}
	env["not"] = func(b bool) bool {
		return !b
	}

	// 日期时间函数（简化版）
	env["now"] = func() int64 {
		return Now().Unix()
	}
	env["dateDiff"] = func(d1, d2 int64) float64 {
		diff := d2 - d1
		if diff < 0 {
			diff = -diff
		}
		return float64(diff / 86400) // 天数
	}

	// 数组函数
	env["includes"] = func(arr []interface{}, item interface{}) bool {
		for _, v := range arr {
			if fmt.Sprintf("%v", v) == fmt.Sprintf("%v", item) {
				return true
			}
		}
		return false
	}
	env["filter"] = func(arr []interface{}, predicate func(interface{}) bool) []interface{} {
		result := make([]interface{}, 0)
		for _, v := range arr {
			if predicate(v) {
				result = append(result, v)
			}
		}
		return result
	}
	env["map"] = func(arr []interface{}, transform func(interface{}) interface{}) []interface{} {
		result := make([]interface{}, len(arr))
		for i, v := range arr {
			result[i] = transform(v)
		}
		return result
	}
	env["first"] = func(arr []interface{}) interface{} {
		if len(arr) == 0 {
			return nil
		}
		return arr[0]
	}
	env["last"] = func(arr []interface{}) interface{} {
		if len(arr) == 0 {
			return nil
		}
		return arr[len(arr)-1]
	}
}

// ValidateExpression 验证表达式语法
func (e *ExpressionEngine) ValidateExpression(expression string) error {
	if expression == "" {
		return nil
	}

	// 基本的语法检查
	// 1. 检查括号匹配
	openCount := strings.Count(expression, "(")
	closeCount := strings.Count(expression, ")")
	if openCount != closeCount {
		return fmt.Errorf("括号不匹配: 开启=%d, 关闭=%d", openCount, closeCount)
	}

	// 2. 检查花括号匹配
	braceCount := strings.Count(expression, "{")
	braceCloseCount := strings.Count(expression, "}")
	if braceCount != braceCloseCount {
		return fmt.Errorf("花括号不匹配: 开启=%d, 关闭=%d", braceCount, braceCloseCount)
	}

	// 3. 检查方括号匹配
	squareOpen := strings.Count(expression, "[")
	squareClose := strings.Count(expression, "]")
	if squareOpen != squareClose {
		return fmt.Errorf("方括号不匹配: 开启=%d, 关闭=%d", squareOpen, squareClose)
	}

	// 4. 检查非法字符
	illegalPattern := regexp.MustCompile(`[^\w\s\-\+\*/%<>=!&|,().{}\[\]?:"']`)
	if illegalPattern.MatchString(expression) {
		return fmt.Errorf("表达式包含非法字符")
	}

	// 5. 尝试编译表达式
	env := make(map[string]interface{})
	e.registerBuiltinFunctions(env)
	for name, fn := range e.Functions {
		env[name] = fn
	}

	_, err := expr.Compile(expression, expr.Env(env))
	if err != nil {
		return fmt.Errorf("表达式语法错误: %w", err)
	}

	return nil
}

// Now 获取当前时间（用于测试模拟）
func Now() time.Time {
	return time.Now()
}

// Default time.Now function
var timeNow = time.Now

// SetTimeNowFunction 设置时间函数（用于测试）
var SetTimeNowFunction = func(fn func() time.Time) {
	timeNow = fn
}
