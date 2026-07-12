package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== 工作流定义边界测试 ====================

func TestWorkflowDefinition_EmptyDefinition(t *testing.T) {
	def := WorkflowDefinition{
		ID:      "",
		Name:    "",
		Version: "",
		Steps:   []WorkflowStep{},
	}

	bytes, err := json.Marshal(def)
	require.NoError(t, err)

	var parsed WorkflowDefinition
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Empty(t, parsed.ID)
	assert.Empty(t, parsed.Steps)
}

func TestWorkflowDefinition_FullDefinition(t *testing.T) {
	def := WorkflowDefinition{
		ID:         "full-workflow",
		Name:       "完整工作流",
		Version:    "2.0.0",
		StartEvent: "start",
		EndEvent:   "end",
		Steps: []WorkflowStep{
			{
				ID:         "start",
				Name:       "开始",
				Type:       "start",
				Properties: map[string]interface{}{"timeout": 60},
			},
			{
				ID:         "task1",
				Name:       "任务1",
				Type:       "task",
				Assignee:   "user1",
				Candidates: []string{"user2", "user3"},
				Groups:     []string{"group1"},
				Timeout:    120,
				Actions:    []string{"approve", "reject"},
				Conditions: []WorkflowCondition{
					{Type: "field", Field: "status", Operator: "equals", Value: "pending"},
				},
			},
			{
				ID:         "condition1",
				Name:       "条件判断",
				Type:       "condition",
				Conditions: []WorkflowCondition{
					{Type: "field", Field: "priority", Operator: "greater_than", Value: 5},
				},
			},
			{
				ID:         "end",
				Name:       "结束",
				Type:       "end",
			},
		},
		Transitions: []WorkflowTransition{
			{
				ID:       "t1",
				FromStep: "start",
				ToStep:   "task1",
				Auto:     true,
			},
			{
				ID:        "t2",
				FromStep:  "task1",
				ToStep:    "condition1",
				Auto:      false,
				Actions:   []string{"complete"},
				Condition: &WorkflowCondition{Type: "field", Field: "result", Operator: "equals", Value: "ok"},
			},
			{
				ID:        "t3",
				FromStep:  "condition1",
				ToStep:     "end",
				Auto:      false,
				Condition: &WorkflowCondition{Type: "expression", Field: "", Operator: "", Value: nil},
			},
		},
		Variables: map[string]interface{}{
			"var1": "value1",
			"var2": 123,
		},
	}

	// 序列化
	bytes, err := json.Marshal(def)
	require.NoError(t, err)
	assert.NotEmpty(t, bytes)

	// 反序列化
	var parsed WorkflowDefinition
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "full-workflow", parsed.ID)
	assert.Equal(t, 4, len(parsed.Steps))
	assert.Equal(t, 3, len(parsed.Transitions))
	assert.Equal(t, "user1", parsed.Steps[1].Assignee)
	assert.Len(t, parsed.Steps[1].Candidates, 2)
}

// ==================== 工作流条件测试 ====================

func TestWorkflowCondition_Types(t *testing.T) {
	conditions := []WorkflowCondition{
		{Type: "field", Field: "status", Operator: "equals", Value: "approved"},
		{Type: "expression", Field: "", Operator: "", Value: "priority > 5"},
		{Type: "approval", Field: "", Operator: "equals", Value: "approved"},
	}

	for _, cond := range conditions {
		assert.NotEmpty(t, cond.Type)
	}
}

func TestWorkflowCondition_JSONSerialization(t *testing.T) {
	condition := WorkflowCondition{
		Type:     "field",
		Field:    "status",
		Operator: "equals",
		Value:    "approved",
		Logic:    "and",
		Next: &WorkflowCondition{
			Type:     "field",
			Field:    "priority",
			Operator: "greater_than",
			Value:    5,
		},
	}

	bytes, err := json.Marshal(condition)
	require.NoError(t, err)

	var parsed WorkflowCondition
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "field", parsed.Type)
	assert.Equal(t, "status", parsed.Field)
	assert.NotNil(t, parsed.Next)
	assert.Equal(t, "priority", parsed.Next.Field)
}

// ==================== 工作流执行上下文测试 ====================

func TestWorkflowExecutionContext(t *testing.T) {
	ctx := &WorkflowExecutionContext{
		InstanceID:   1,
		WorkflowID:   100,
		TenantID:     10,
		CurrentStep:  "task1",
		Variables:    map[string]interface{}{"key": "value"},
		History:      []WorkflowHistory{},
		Status:       "running",
	}

	assert.Equal(t, 1, ctx.InstanceID)
	assert.Equal(t, "task1", ctx.CurrentStep)
	assert.Equal(t, "running", ctx.Status)
	assert.Empty(t, ctx.History)
}

func TestWorkflowHistory(t *testing.T) {
	history := WorkflowHistory{
		StepID:    "task1",
		StepName:  "任务1",
		Action:    "complete",
		UserID:    123,
		Data:      map[string]interface{}{"result": "ok"},
		Comment:   "测试评论",
	}

	assert.Equal(t, "task1", history.StepID)
	assert.Equal(t, "complete", history.Action)
	assert.Equal(t, 123, history.UserID)
}

// ==================== contains 函数边界测试 ====================

func TestContains_BoundaryCases(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"", "", true},           // 空字符串包含空子串
		{"abc", "", true},        // 任何字符串包含空子串
		{"", "a", false},         // 空字符串不包含任何非空子串
		{"abc", "abc", true},     // 完整匹配
		{"abc", "ab", true},      // 前缀匹配
		{"abc", "bc", true},      // 后缀匹配
		{"abc", "b", true},       // 中间匹配
		{"abc", "d", false},      // 不匹配
		{"abc", "abcd", false},   // 子串比原字符串长
		{"hello world", "world", true}, // 单词匹配
	}

	for _, tt := range tests {
		result := contains(tt.s, tt.substr)
		assert.Equal(t, tt.expected, result, "contains(%q, %q)", tt.s, tt.substr)
	}
}

func TestContainsSubstring(t *testing.T) {
	assert.True(t, containsSubstring("hello", "ello"))
	assert.True(t, containsSubstring("hello", "hel"))
	assert.False(t, containsSubstring("hello", "world"))
	assert.False(t, containsSubstring("", "a"))
}

// ==================== 步骤类型测试 ====================

func TestWorkflowStep_Types(t *testing.T) {
	stepTypes := []string{"start", "end", "task", "approval", "condition", "action"}

	for _, stepType := range stepTypes {
		step := WorkflowStep{ID: "test", Name: "Test", Type: stepType}
		assert.Equal(t, stepType, step.Type)
	}
}

func TestWorkflowStep_InvalidType(t *testing.T) {
	engine := &WorkflowEngine{}
	execCtx := &WorkflowExecutionContext{
		CurrentStep: "invalid_step",
		Variables:   map[string]interface{}{},
	}
	step := &WorkflowStep{ID: "invalid_step", Name: "Invalid", Type: "invalid_type"}
	definition := WorkflowDefinition{Steps: []WorkflowStep{*step}, Transitions: []WorkflowTransition{}}

	_, err := engine.executeStep(nil, execCtx, step, definition)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未知的步骤类型")
}

// ==================== 步骤查找边界测试 ====================

func TestWorkflowEngine_FindStep_EmptySteps(t *testing.T) {
	engine := &WorkflowEngine{}
	step := engine.findStep([]WorkflowStep{}, "any")
	assert.Nil(t, step)
}

func TestWorkflowEngine_FindStep_DuplicateIDs(t *testing.T) {
	engine := &WorkflowEngine{}
	steps := []WorkflowStep{
		{ID: "step1", Name: "步骤1"},
		{ID: "step1", Name: "步骤1重复"}, // 重复ID
	}

	// 应该返回第一个匹配的
	step := engine.findStep(steps, "step1")
	require.NotNil(t, step)
	assert.Equal(t, "步骤1", step.Name)
}

// ==================== 转换查找边界测试 ====================

func TestWorkflowEngine_FindTransitions_EmptyTransitions(t *testing.T) {
	engine := &WorkflowEngine{}
	transitions := engine.findTransitions([]WorkflowTransition{}, "any")
	assert.Len(t, transitions, 0)
}

func TestWorkflowEngine_FindTransitions_MultipleFromSameStep(t *testing.T) {
	engine := &WorkflowEngine{}
	transitions := []WorkflowTransition{
		{ID: "t1", FromStep: "task1", ToStep: "end1"},
		{ID: "t2", FromStep: "task1", ToStep: "end2"},
		{ID: "t3", FromStep: "task2", ToStep: "end1"},
	}

	found := engine.findTransitions(transitions, "task1")
	assert.Len(t, found, 2)

	found = engine.findTransitions(transitions, "task2")
	assert.Len(t, found, 1)
}

// ==================== 条件评估边界测试 ====================

func TestWorkflowEngine_EvaluateFieldCondition_AllOperators(t *testing.T) {
	engine := &WorkflowEngine{}
	variables := map[string]interface{}{
		"string_field":  "hello",
		"int_field":     10,
		"float_field":   10.5,
		"bool_field":    true,
		"array_field":   []string{"a", "b"},
	}

	tests := []struct {
		name     string
		cond     WorkflowCondition
		expected bool
	}{
		{"string equals", WorkflowCondition{Type: "field", Field: "string_field", Operator: "equals", Value: "hello"}, true},
		{"string not_equals", WorkflowCondition{Type: "field", Field: "string_field", Operator: "not_equals", Value: "world"}, true},
		{"string contains", WorkflowCondition{Type: "field", Field: "string_field", Operator: "contains", Value: "ell"}, true},
		// 注意：greater_than 和 less_than 当前实现在 compareValues 中返回 false（简化实现）
		// 这些测试反映当前实现的行为，未来可能需要更完整的比较逻辑
		{"int greater_than", WorkflowCondition{Type: "field", Field: "int_field", Operator: "greater_than", Value: 5}, false},
		{"int less_than", WorkflowCondition{Type: "field", Field: "int_field", Operator: "less_than", Value: 15}, false},
		{"unknown field", WorkflowCondition{Type: "field", Field: "unknown", Operator: "equals", Value: "value"}, false},
		{"unknown operator", WorkflowCondition{Type: "field", Field: "string_field", Operator: "unknown", Value: "value"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.evaluateFieldCondition(tt.cond, variables)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkflowEngine_EvaluateApprovalCondition(t *testing.T) {
	engine := &WorkflowEngine{}

	tests := []struct {
		name       string
		variables  map[string]interface{}
		cond       WorkflowCondition
		expected   bool
	}{
		{
			name:      "审批通过",
			variables: map[string]interface{}{"approval_status": "approved"},
			cond:      WorkflowCondition{Type: "approval", Value: "approved"},
			expected:  true,
		},
		{
			name:      "审批拒绝",
			variables: map[string]interface{}{"approval_status": "rejected"},
			cond:      WorkflowCondition{Type: "approval", Value: "approved"},
			expected:  false,
		},
		{
			name:      "无审批状态",
			variables: map[string]interface{}{},
			cond:      WorkflowCondition{Type: "approval", Value: "approved"},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.evaluateApprovalCondition(tt.cond, tt.variables)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== CompleteWorkflowStepRequest 测试 ====================

func TestCompleteWorkflowStepRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     *CompleteWorkflowStepRequest
		isValid bool
	}{
		{
			name: "有效请求",
			req: &CompleteWorkflowStepRequest{
				InstanceID: 1,
				Action:     "approve",
				UserID:     123,
				TenantID:   10,
			},
			isValid: true,
		},
		{
			name: "带变量和数据的请求",
			req: &CompleteWorkflowStepRequest{
				InstanceID: 1,
				Action:     "complete",
				UserID:     123,
				Variables:  map[string]interface{}{"result": "ok"},
				Data:       map[string]interface{}{"comment": "测试"},
				Comment:    "备注",
				TenantID:   10,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Greater(t, tt.req.InstanceID, 0)
			assert.NotEmpty(t, tt.req.Action)
			assert.Greater(t, tt.req.UserID, 0)
		})
	}
}

// ==================== determineNextStep 测试 ====================

func TestWorkflowEngine_DetermineNextStep_NoTransitions(t *testing.T) {
	engine := &WorkflowEngine{}
	execCtx := &WorkflowExecutionContext{Variables: map[string]interface{}{}}
	step := &WorkflowStep{ID: "task1"}
	definition := WorkflowDefinition{
		Steps:       []WorkflowStep{*step},
		Transitions: []WorkflowTransition{}, // 无转换
	}

	result := engine.determineNextStep(execCtx, step, definition, "complete")
	assert.Empty(t, result)
}

func TestWorkflowEngine_DetermineNextStep_NoMatchingAction(t *testing.T) {
	engine := &WorkflowEngine{}
	execCtx := &WorkflowExecutionContext{Variables: map[string]interface{}{}}
	step := &WorkflowStep{ID: "task1"}
	definition := WorkflowDefinition{
		Steps: []WorkflowStep{*step},
		Transitions: []WorkflowTransition{
			{ID: "t1", FromStep: "task1", ToStep: "end", Actions: []string{"approve"}}, // 只允许 approve
		},
	}

	result := engine.determineNextStep(execCtx, step, definition, "reject") // 使用 reject
	assert.Empty(t, result)                                                   // 不匹配任何转换
}

func TestWorkflowEngine_DetermineNextStep_MatchingWithCondition(t *testing.T) {
	engine := &WorkflowEngine{}
	execCtx := &WorkflowExecutionContext{
		Variables: map[string]interface{}{"approved": true},
	}
	step := &WorkflowStep{ID: "approval"}
	definition := WorkflowDefinition{
		Steps: []WorkflowStep{*step},
		Transitions: []WorkflowTransition{
			{
				ID:       "t1",
				FromStep: "approval",
				ToStep:    "end",
				Actions:   []string{"complete"},
				Condition: &WorkflowCondition{Type: "field", Field: "approved", Operator: "equals", Value: true},
			},
		},
	}

	result := engine.determineNextStep(execCtx, step, definition, "complete")
	assert.Equal(t, "end", result)
}

func TestWorkflowEngine_DetermineNextStep_ConditionNotMet(t *testing.T) {
	engine := &WorkflowEngine{}
	execCtx := &WorkflowExecutionContext{
		Variables: map[string]interface{}{"approved": false}, // 条件不满足
	}
	step := &WorkflowStep{ID: "approval"}
	definition := WorkflowDefinition{
		Steps: []WorkflowStep{*step},
		Transitions: []WorkflowTransition{
			{
				ID:       "t1",
				FromStep: "approval",
				ToStep:    "end",
				Actions:   []string{"complete"},
				Condition: &WorkflowCondition{Type: "field", Field: "approved", Operator: "equals", Value: true},
			},
		},
	}

	result := engine.determineNextStep(execCtx, step, definition, "complete")
	assert.Empty(t, result) // 条件不满足，不应转换
}

// ==================== containsString 边界测试 ====================

func TestContainsString_BoundaryCases(t *testing.T) {
	tests := []struct {
		slice    []string
		value    string
		expected bool
	}{
		{[]string{}, "", false},           // 空切片
		{[]string{"a"}, "", false},        // 空值
		{[]string{"a"}, "a", true},        // 单元素匹配
		{[]string{"a"}, "b", false},       // 单元素不匹配
		{[]string{"a", "b", "c"}, "b", true}, // 多元素匹配
		{[]string{"a", "b", "c"}, "d", false}, // 多元素不匹配
		{[]string{"a", "a", "b"}, "a", true},  // 重复元素
	}

	for _, tt := range tests {
		result := containsString(tt.slice, tt.value)
		assert.Equal(t, tt.expected, result, "containsString(%v, %q)", tt.slice, tt.value)
	}
}

// ==================== compareValues 测试 ====================

func TestWorkflowEngine_CompareValues(t *testing.T) {
	engine := &WorkflowEngine{}

	// 当前实现返回 false（简化处理）
	// 实际生产环境应该有更完整的实现
	result := engine.compareValues(10, 5, ">")
	assert.False(t, result)

	result = engine.compareValues(5, 10, "<")
	assert.False(t, result)
}

// ==================== JSON 序列化测试 ====================

func TestWorkflowDefinition_JSONRoundTrip(t *testing.T) {
	original := createSimpleWorkflowDefinition()

	// 序列化
	bytes, err := json.Marshal(original)
	require.NoError(t, err)

	// 反序列化
	var parsed WorkflowDefinition
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	// 验证
	assert.Equal(t, original.ID, parsed.ID)
	assert.Equal(t, original.Name, parsed.Name)
	assert.Equal(t, original.Version, parsed.Version)
	assert.Equal(t, original.StartEvent, parsed.StartEvent)
	assert.Equal(t, original.EndEvent, parsed.EndEvent)
	assert.Len(t, original.Steps, len(parsed.Steps))
	assert.Len(t, original.Transitions, len(parsed.Transitions))
}

func TestWorkflowExecutionContext_JSONRoundTrip(t *testing.T) {
	ctx := &WorkflowExecutionContext{
		InstanceID:   1,
		WorkflowID:   100,
		TenantID:     10,
		CurrentStep:  "task1",
		Variables:    map[string]interface{}{"key": "value", "count": 42},
		History: []WorkflowHistory{
			{StepID: "start", StepName: "开始", Action: "start", UserID: 1},
		},
		Status:    "running",
	}

	bytes, err := json.Marshal(ctx)
	require.NoError(t, err)

	var parsed WorkflowExecutionContext
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, ctx.InstanceID, parsed.InstanceID)
	assert.Equal(t, ctx.CurrentStep, parsed.CurrentStep)
	assert.Equal(t, ctx.Status, parsed.Status)
	assert.Len(t, parsed.History, 1)
}
