package service

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== ExpressionEngine 测试 ====================

func TestExpressionEngine_NewExpressionEngine(t *testing.T) {
	engine := NewExpressionEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.Functions)
	assert.Equal(t, 0, len(engine.Functions))
}

func TestExpressionEngine_RegisterFunction(t *testing.T) {
	engine := NewExpressionEngine()

	// 注册一个测试函数
	testFn := func(ctx interface{}, name string) string {
		return "Hello, " + name
	}
	engine.RegisterFunction("greet", testFn)

	assert.Equal(t, 1, len(engine.Functions))
	assert.NotNil(t, engine.Functions["greet"])
}

func TestExpressionEngine_RegisterMultipleFunctions(t *testing.T) {
	engine := NewExpressionEngine()

	engine.RegisterFunction("add", func(a, b int) int { return a + b })
	engine.RegisterFunction("subtract", func(a, b int) int { return a - b })
	engine.RegisterFunction("multiply", func(a, b int) int { return a * b })

	assert.Equal(t, 3, len(engine.Functions))
}

func TestExpressionEngine_Evaluate_EmptyExpression(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{}

	result, err := engine.Evaluate("", variables)
	require.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestExpressionEngine_Evaluate_SimpleArithmetic(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"a": float64(10),
		"b": float64(5),
	}

	tests := []struct {
		name       string
		expression string
		expected   interface{}
	}{
		{"加法", "a + b", float64(15)},
		{"减法", "a - b", float64(5)},
		{"乘法", "a * b", float64(50)},
		{"除法", "a / b", float64(2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(tt.expression, variables)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpressionEngine_Evaluate_StringOperations(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"name":    "World",
		"greeting": "Hello",
	}

	result, err := engine.Evaluate("greeting + \", \" + name", variables)
	require.NoError(t, err)
	assert.Equal(t, "Hello, World", result)
}

func TestExpressionEngine_Evaluate_Comparison(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"age":      25,
		"adultAge": 18,
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"大于", "age > adultAge", true},
		{"小于", "age < adultAge", false},
		{"大于等于", "age >= 25", true},
		{"小于等于", "age <= 25", true},
		{"等于", "age == 25", true},
		{"不等于", "age != 25", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(tt.expression, variables)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpressionEngine_Evaluate_Conditional(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"score":     85,
		"passScore": 60,
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"及格判断", "score >= passScore", true},
		{"优秀判断", "score >= 90", false},
		{"不及格判断", "score < passScore", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.EvaluateCondition(tt.expression, variables)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpressionEngine_Evaluate_LogicalOperators(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"isAdmin":  true,
		"isActive": true,
		"isGuest":  false,
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"AND", "isAdmin && isActive", true},
		{"OR", "isAdmin || isGuest", true},
		{"NOT", "!isGuest", true},
		{"复合条件", "isAdmin && isActive && !isGuest", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(tt.expression, variables)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpressionEngine_Evaluate_InvalidExpression(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"a": 10,
	}

	_, err := engine.Evaluate("a +++ b", variables)
	assert.Error(t, err)
}

func TestExpressionEngine_EvaluateCondition_InvalidCondition(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{}

	_, err := engine.EvaluateCondition("invalid !!! expression", variables)
	assert.Error(t, err)
}

func TestExpressionEngine_EvaluateNumeric(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"a": float64(10),
		"b": float64(3),
	}

	tests := []struct {
		name       string
		expression string
		expected   float64
	}{
		{"整数除法", "a / b", 3.3333333333333335},
		{"乘法", "a * b", 30},
		{"加法", "a + b", 13},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.EvaluateNumeric(tt.expression, variables)
			require.NoError(t, err)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

func TestExpressionEngine_EvaluateNumeric_Invalid(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"text": "hello",
	}

	_, err := engine.EvaluateNumeric("text", variables)
	assert.Error(t, err)
}

func TestExpressionEngine_Evaluate_TernaryOperator(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"age":      20,
		"adultAge": 18,
	}

	result, err := engine.Evaluate("age >= adultAge ? \"adult\" : \"minor\"", variables)
	require.NoError(t, err)
	assert.Equal(t, "adult", result)
}

func TestExpressionEngine_Evaluate_MapAccess(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"user": map[string]interface{}{
			"name":  "John",
			"age":   30,
			"admin": true,
		},
	}

	tests := []struct {
		name       string
		expression string
		expected   interface{}
	}{
		{"访问name", "user.name", "John"},
		{"访问age", "user.age", 30},
		{"访问admin", "user.admin", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Evaluate(tt.expression, variables)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== BPMNParser 基本测试 ====================

func TestBPMNParser_NewBPMNParser(t *testing.T) {
	parser := NewBPMNParser()
	assert.NotNil(t, parser)
}

func TestBPMNParser_ParseXML_InvalidXML(t *testing.T) {
	parser := NewBPMNParser()

	invalidXML := []byte(`<invalid>not bpmn xml</invalid>`)

	_, err := parser.ParseXML(invalidXML)
	assert.Error(t, err)
}

// ==================== BPMN 流程执行测试 ====================

func setupBPMNEngineTest(t *testing.T) (*CustomProcessEngine, *ent.Client) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_engine_test?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	engine := &CustomProcessEngine{
		client:           client,
		logger:           logger,
		parser:           NewBPMNParser(),
		exprEngine:       NewExpressionEngine(),
		expressionVars:   make(map[string]interface{}),
	}
	engine.registerProcessFunctions()
	return engine, client
}

func TestCustomProcessEngine_ExpressionVars(t *testing.T) {
	engine, client := setupBPMNEngineTest(t)
	defer client.Close()

	assert.NotNil(t, engine.expressionVars)
	assert.Equal(t, 0, len(engine.expressionVars))
}

func TestCustomProcessEngine_RegisterBuiltinFunctions(t *testing.T) {
	engine, client := setupBPMNEngineTest(t)
	defer client.Close()

	// 验证表达式引擎有函数注册
	assert.NotNil(t, engine.exprEngine)
	assert.NotNil(t, engine.exprEngine.Functions)
}

func TestCustomProcessEngine_EvaluateCondition(t *testing.T) {
	engine, client := setupBPMNEngineTest(t)
	defer client.Close()

	variables := map[string]interface{}{
		"approved": true,
		"amount":   1000,
	}

	// 测试条件评估
	flow := &BPMNSequenceFlow{
		ConditionExpression: &BPMNConditionExpression{
			Expression: "approved == true",
		},
	}

	result := engine.evaluateCondition(flow, variables)
	assert.True(t, result)
}

func TestCustomProcessEngine_EvaluateCondition_NilExpression(t *testing.T) {
	engine, client := setupBPMNEngineTest(t)
	defer client.Close()

	variables := map[string]interface{}{}

	// 无条件表达式应该返回 true
	flow := &BPMNSequenceFlow{
		ConditionExpression: nil,
	}

	result := engine.evaluateCondition(flow, variables)
	assert.True(t, result)
}

func TestCustomProcessEngine_EvaluateCondition_InvalidExpression(t *testing.T) {
	engine, client := setupBPMNEngineTest(t)
	defer client.Close()

	variables := map[string]interface{}{
		"value": 10,
	}

	flow := &BPMNSequenceFlow{
		ConditionExpression: &BPMNConditionExpression{
			Expression: "invalid +++ expression",
		},
	}

	result := engine.evaluateCondition(flow, variables)
	assert.False(t, result)
}

func TestCustomProcessEngine_EvaluateCondition_EmptyExpression(t *testing.T) {
	engine, client := setupBPMNEngineTest(t)
	defer client.Close()

	variables := map[string]interface{}{}

	flow := &BPMNSequenceFlow{
		ConditionExpression: &BPMNConditionExpression{
			Expression: "",
		},
	}

	result := engine.evaluateCondition(flow, variables)
	assert.True(t, result)
}

// ==================== 流程定义请求/响应结构测试 ====================

func TestCreateProcessDefinitionRequest_Structure(t *testing.T) {
	req := &CreateProcessDefinitionRequest{
		Key:        "test-process",
		Name:       "测试流程",
		Category:   "approval",
		BPMNXML:    "<definitions>...</definitions>",
		TenantID:   1,
	}

	assert.Equal(t, "test-process", req.Key)
	assert.Equal(t, "测试流程", req.Name)
	assert.Equal(t, "approval", req.Category)
	assert.Equal(t, 1, req.TenantID)
}

func TestUpdateProcessDefinitionRequest_Structure(t *testing.T) {
	isActive := false
	req := &UpdateProcessDefinitionRequest{
		Name:     "更新后的流程",
		BPMNXML:  "<definitions>updated...</definitions>",
		IsActive: &isActive,
	}

	assert.Equal(t, "更新后的流程", req.Name)
	assert.NotNil(t, req.IsActive)
	assert.False(t, *req.IsActive)
}

func TestListProcessDefinitionsRequest_Structure(t *testing.T) {
	req := &ListProcessDefinitionsRequest{
		Category: "approval",
		TenantID: 1,
		Page:     1,
		PageSize: 20,
	}

	assert.Equal(t, "approval", req.Category)
	assert.Equal(t, 1, req.Page)
	assert.Equal(t, 20, req.PageSize)
}

func TestInstanceStatistics_Structure(t *testing.T) {
	stats := &InstanceStatistics{
		Total:      100,
		Running:    30,
		Completed:  50,
		Suspended:  10,
		Terminated: 10,
	}

	assert.Equal(t, 100, stats.Total)
	assert.Equal(t, 30, stats.Running)
	assert.Equal(t, 70, stats.Completed+stats.Suspended+stats.Terminated)
}

func TestCounterSignRequest_Structure(t *testing.T) {
	req := &CounterSignRequest{
		ApprovalType: "parallel",
		Approvers:   []string{"user1", "user2", "user3"},
		Threshold:   2,
	}

	assert.Len(t, req.Approvers, 3)
	assert.Equal(t, "parallel", req.ApprovalType)
	assert.Equal(t, 2, req.Threshold)
}

func TestCounterSignStatus_Structure(t *testing.T) {
	status := &CounterSignStatus{
		Total:     5,
		Completed: 3,
		Approved:  2,
		Rejected:  1,
		Pending:   2,
		Status:    "pending",
	}

	assert.Equal(t, 5, status.Total)
	assert.Equal(t, 3, status.Completed)
	assert.Equal(t, "pending", status.Status)
}

func TestVoteRequest_Structure(t *testing.T) {
	req := &VoteRequest{
		Approved: true,
		Comment:  "同意",
	}

	assert.True(t, req.Approved)
	assert.Equal(t, "同意", req.Comment)
}

// ==================== 表达式引擎边界情况测试 ====================

func TestExpressionEngine_Evaluate_IntegerVariables(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"count": 10,
	}

	result, err := engine.Evaluate("count * 2", variables)
	require.NoError(t, err)
	assert.Equal(t, 20, result)
}

func TestExpressionEngine_Evaluate_NilValue(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"value": nil,
	}

	// nil 值的处理
	result, err := engine.Evaluate("value == nil", variables)
	require.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestExpressionEngine_Evaluate_ComplexCondition(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"age":          25,
		"hasLicense":   true,
		"hasInsurance": true,
	}

	expression := "(age >= 18 && hasLicense) && hasInsurance"
	result, err := engine.Evaluate(expression, variables)
	require.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestExpressionEngine_Evaluate_StringContains(t *testing.T) {
	engine := NewExpressionEngine()
	variables := map[string]interface{}{
		"email": "test@example.com",
	}

	result, err := engine.Evaluate("email contains(\"@\")", variables)
	require.NoError(t, err)
	assert.Equal(t, true, result)
}
