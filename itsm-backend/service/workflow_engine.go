package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"itsm-backend/ent"
)

// WorkflowEngine 工作流执行引擎
type WorkflowEngine struct {
	client *ent.Client
}

// NewWorkflowEngine 创建工作流引擎实例
func NewWorkflowEngine(client *ent.Client) *WorkflowEngine {
	return &WorkflowEngine{client: client}
}

// WorkflowDefinition 工作流定义结构
type WorkflowDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	StartEvent  string                 `json:"start_event"`
	EndEvent    string                 `json:"end_event"`
	Steps       []WorkflowStep         `json:"steps"`
	Transitions []WorkflowTransition   `json:"transitions"`
	Variables   map[string]interface{} `json:"variables"`
}

// WorkflowStep 工作流步骤
type WorkflowStep struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // start, end, task, approval, condition, action
	Assignee   string                 `json:"assignee,omitempty"`
	Candidates []string               `json:"candidates,omitempty"`
	Groups     []string               `json:"groups,omitempty"`
	Timeout    int                    `json:"timeout,omitempty"` // 超时时间（分钟）
	Actions    []string               `json:"actions,omitempty"` // 允许的操作
	Properties map[string]interface{} `json:"properties,omitempty"`
	Conditions []WorkflowCondition    `json:"conditions,omitempty"`
}

// WorkflowTransition 工作流转换
type WorkflowTransition struct {
	ID        string             `json:"id"`
	FromStep  string             `json:"from_step"`
	ToStep    string             `json:"to_step"`
	Condition *WorkflowCondition `json:"condition,omitempty"`
	Actions   []string           `json:"actions,omitempty"`
	Auto      bool               `json:"auto"` // 是否自动转换
}

// WorkflowCondition 工作流条件
type WorkflowCondition struct {
	Type     string             `json:"type"` // field, expression, approval
	Field    string             `json:"field,omitempty"`
	Operator string             `json:"operator"` // equals, not_equals, contains, greater_than, less_than
	Value    interface{}        `json:"value"`
	Logic    string             `json:"logic"` // and, or
	Next     *WorkflowCondition `json:"next,omitempty"`
}

// WorkflowExecutionContext 工作流执行上下文
type WorkflowExecutionContext struct {
	InstanceID  int                    `json:"instance_id"`
	WorkflowID  int                    `json:"workflow_id"`
	CurrentStep string                 `json:"current_step"`
	Variables   map[string]interface{} `json:"variables"`
	History     []WorkflowHistory      `json:"history"`
	Status      string                 `json:"status"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// WorkflowHistory 工作流执行历史
type WorkflowHistory struct {
	StepID    string                 `json:"step_id"`
	StepName  string                 `json:"step_name"`
	Action    string                 `json:"action"`
	UserID    int                    `json:"user_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Comment   string                 `json:"comment,omitempty"`
}

// ExecuteWorkflow 执行工作流
func (e *WorkflowEngine) ExecuteWorkflow(ctx context.Context, instanceID int) error {
	// 获取工作流实例
	instance, err := e.client.WorkflowInstance.Get(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("获取工作流实例失败: %w", err)
	}

	// 获取工作流定义
	workflow, err := e.client.Workflow.Get(ctx, instance.WorkflowID)
	if err != nil {
		return fmt.Errorf("获取工作流定义失败: %w", err)
	}

	// 解析工作流定义
	var definition WorkflowDefinition
	if err := json.Unmarshal(workflow.Definition, &definition); err != nil {
		return fmt.Errorf("解析工作流定义失败: %w", err)
	}

	// 创建执行上下文
	execCtx := &WorkflowExecutionContext{
		InstanceID:  instance.ID,
		WorkflowID:  instance.WorkflowID,
		CurrentStep: instance.CurrentStep,
		Variables:   make(map[string]interface{}),
		History:     []WorkflowHistory{},
		Status:      instance.Status,
		StartedAt:   instance.StartedAt,
	}

	// 解析实例上下文
	if len(instance.Context) > 0 {
		if err := json.Unmarshal(instance.Context, &execCtx.Variables); err != nil {
			return fmt.Errorf("解析实例上下文失败: %w", err)
		}
	}

	// 执行工作流
	return e.executeWorkflowSteps(ctx, execCtx, definition)
}

// executeWorkflowSteps 执行工作流步骤
func (e *WorkflowEngine) executeWorkflowSteps(ctx context.Context, execCtx *WorkflowExecutionContext, definition WorkflowDefinition) error {
	for {
		// 获取当前步骤
		currentStep := e.findStep(definition.Steps, execCtx.CurrentStep)
		if currentStep == nil {
			return fmt.Errorf("步骤 %s 不存在", execCtx.CurrentStep)
		}

		// 执行步骤
		nextStep, err := e.executeStep(ctx, execCtx, currentStep, definition)
		if err != nil {
			return fmt.Errorf("执行步骤 %s 失败: %w", currentStep.ID, err)
		}

		// 检查是否完成
		if currentStep.Type == "end" {
			execCtx.Status = "completed"
			now := time.Now()
			execCtx.CompletedAt = &now
			break
		}

		// 移动到下一步
		if nextStep != "" {
			execCtx.CurrentStep = nextStep
		} else {
			// 没有下一步，等待用户操作
			break
		}
	}

	// 更新实例状态
	return e.updateInstanceStatus(ctx, execCtx)
}

// executeStep 执行单个步骤
func (e *WorkflowEngine) executeStep(ctx context.Context, execCtx *WorkflowExecutionContext, step *WorkflowStep, definition WorkflowDefinition) (string, error) {
	// 记录步骤开始
	e.recordStepHistory(execCtx, step, "start", 0, nil, "")

	switch step.Type {
	case "start":
		return e.executeStartStep(execCtx, step, definition)
	case "task":
		return e.executeTaskStep(execCtx, step, definition)
	case "approval":
		return e.executeApprovalStep(execCtx, step, definition)
	case "condition":
		return e.executeConditionStep(execCtx, step, definition)
	case "action":
		return e.executeActionStep(execCtx, step, definition)
	case "end":
		return "", nil
	default:
		return "", fmt.Errorf("未知的步骤类型: %s", step.Type)
	}
}

// executeStartStep 执行开始步骤
func (e *WorkflowEngine) executeStartStep(execCtx *WorkflowExecutionContext, step *WorkflowStep, definition WorkflowDefinition) (string, error) {
	// 开始步骤通常自动转换到下一步
	transitions := e.findTransitions(definition.Transitions, step.ID)
	if len(transitions) > 0 {
		return transitions[0].ToStep, nil
	}
	return "", nil
}

// executeTaskStep 执行任务步骤
func (e *WorkflowEngine) executeTaskStep(execCtx *WorkflowExecutionContext, step *WorkflowStep, definition WorkflowDefinition) (string, error) {
	// 任务步骤需要用户操作，等待完成
	// 这里可以创建任务记录，等待用户处理
	return "", nil
}

// executeApprovalStep 执行审批步骤
func (e *WorkflowEngine) executeApprovalStep(execCtx *WorkflowExecutionContext, step *WorkflowStep, definition WorkflowDefinition) (string, error) {
	// 审批步骤需要用户审批，等待完成
	// 这里可以创建审批任务，等待用户处理
	return "", nil
}

// executeConditionStep 执行条件步骤
func (e *WorkflowEngine) executeConditionStep(execCtx *WorkflowExecutionContext, step *WorkflowStep, definition WorkflowDefinition) (string, error) {
	// 评估条件，决定下一步
	for _, condition := range step.Conditions {
		if e.evaluateCondition(condition, execCtx.Variables) {
			// 条件满足，找到对应的转换
			transitions := e.findTransitions(definition.Transitions, step.ID)
			for _, transition := range transitions {
				if transition.Condition != nil && e.evaluateCondition(*transition.Condition, execCtx.Variables) {
					return transition.ToStep, nil
				}
			}
		}
	}
	return "", nil
}

// executeActionStep 执行动作步骤
func (e *WorkflowEngine) executeActionStep(execCtx *WorkflowExecutionContext, step *WorkflowStep, definition WorkflowDefinition) (string, error) {
	// 执行预定义的动作
	// 这里可以执行各种自动化动作，如发送通知、更新状态等
	return "", nil
}

// evaluateCondition 评估条件
func (e *WorkflowEngine) evaluateCondition(condition WorkflowCondition, variables map[string]interface{}) bool {
	switch condition.Type {
	case "field":
		return e.evaluateFieldCondition(condition, variables)
	case "expression":
		return e.evaluateExpressionCondition(condition, variables)
	case "approval":
		return e.evaluateApprovalCondition(condition, variables)
	default:
		return false
	}
}

// evaluateFieldCondition 评估字段条件
func (e *WorkflowEngine) evaluateFieldCondition(condition WorkflowCondition, variables map[string]interface{}) bool {
	fieldValue, exists := variables[condition.Field]
	if !exists {
		return false
	}

	switch condition.Operator {
	case "equals":
		return fieldValue == condition.Value
	case "not_equals":
		return fieldValue != condition.Value
	case "contains":
		if str, ok := fieldValue.(string); ok {
			if target, ok := condition.Value.(string); ok {
				return contains(str, target)
			}
		}
	case "greater_than":
		return e.compareValues(fieldValue, condition.Value, ">")
	case "less_than":
		return e.compareValues(fieldValue, condition.Value, "<")
	}
	return false
}

// evaluateExpressionCondition 评估表达式条件
func (e *WorkflowEngine) evaluateExpressionCondition(condition WorkflowCondition, variables map[string]interface{}) bool {
	// 这里可以实现更复杂的表达式评估逻辑
	// 暂时返回false
	return false
}

// evaluateApprovalCondition 评估审批条件
func (e *WorkflowEngine) evaluateApprovalCondition(condition WorkflowCondition, variables map[string]interface{}) bool {
	// 检查审批状态
	approvalStatus, exists := variables["approval_status"]
	if !exists {
		return false
	}
	return approvalStatus == condition.Value
}

// compareValues 比较值
func (e *WorkflowEngine) compareValues(a, b interface{}, operator string) bool {
	// 实现类型安全的比较逻辑
	// 这里简化处理，实际应该根据类型进行适当的比较
	return false
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr))))
}

// containsSubstring 检查字符串是否包含子串（简化实现）
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// findStep 查找步骤
func (e *WorkflowEngine) findStep(steps []WorkflowStep, stepID string) *WorkflowStep {
	for i := range steps {
		if steps[i].ID == stepID {
			return &steps[i]
		}
	}
	return nil
}

// findTransitions 查找转换
func (e *WorkflowEngine) findTransitions(transitions []WorkflowTransition, fromStep string) []WorkflowTransition {
	var result []WorkflowTransition
	for i := range transitions {
		if transitions[i].FromStep == fromStep {
			result = append(result, transitions[i])
		}
	}
	return result
}

// recordStepHistory 记录步骤历史
func (e *WorkflowEngine) recordStepHistory(execCtx *WorkflowExecutionContext, step *WorkflowStep, action string, userID int, data map[string]interface{}, comment string) {
	history := WorkflowHistory{
		StepID:    step.ID,
		StepName:  step.Name,
		Action:    action,
		UserID:    userID,
		Timestamp: time.Now(),
		Data:      data,
		Comment:   comment,
	}
	execCtx.History = append(execCtx.History, history)
}

// updateInstanceStatus 更新实例状态
func (e *WorkflowEngine) updateInstanceStatus(ctx context.Context, execCtx *WorkflowExecutionContext) error {
	// 序列化上下文
	contextBytes, err := json.Marshal(execCtx.Variables)
	if err != nil {
		return fmt.Errorf("序列化上下文失败: %w", err)
	}

	// 更新实例
	update := e.client.WorkflowInstance.UpdateOneID(execCtx.InstanceID).
		SetStatus(execCtx.Status).
		SetCurrentStep(execCtx.CurrentStep).
		SetContext(contextBytes).
		SetUpdatedAt(time.Now())

	if execCtx.Status == "completed" && execCtx.CompletedAt != nil {
		update.SetCompletedAt(*execCtx.CompletedAt)
	}

	_, err = update.Save(ctx)
	return err
}

// CompleteWorkflowStep 完成工作流步骤
func (e *WorkflowEngine) CompleteWorkflowStep(ctx context.Context, req *CompleteWorkflowStepRequest) error {
	// 获取工作流实例
	instance, err := e.client.WorkflowInstance.Get(ctx, req.InstanceID)
	if err != nil {
		return fmt.Errorf("获取工作流实例失败: %w", err)
	}

	// 获取工作流定义
	workflow, err := e.client.Workflow.Get(ctx, instance.WorkflowID)
	if err != nil {
		return fmt.Errorf("获取工作流定义失败: %w", err)
	}

	// 解析工作流定义
	var definition WorkflowDefinition
	if err := json.Unmarshal(workflow.Definition, &definition); err != nil {
		return fmt.Errorf("解析工作流定义失败: %w", err)
	}

	// 创建执行上下文
	execCtx := &WorkflowExecutionContext{
		InstanceID:  instance.ID,
		WorkflowID:  instance.WorkflowID,
		CurrentStep: instance.CurrentStep,
		Variables:   make(map[string]interface{}),
		History:     []WorkflowHistory{},
		Status:      instance.Status,
		StartedAt:   instance.StartedAt,
	}

	// 解析实例上下文
	if len(instance.Context) > 0 {
		if err := json.Unmarshal(instance.Context, &execCtx.Variables); err != nil {
			return fmt.Errorf("解析实例上下文失败: %w", err)
		}
	}

	// 更新变量
	for k, v := range req.Variables {
		execCtx.Variables[k] = v
	}

	// 记录步骤完成
	currentStep := e.findStep(definition.Steps, execCtx.CurrentStep)
	if currentStep != nil {
		e.recordStepHistory(execCtx, currentStep, req.Action, req.UserID, req.Data, req.Comment)
	}

	// 确定下一步
	nextStep := e.determineNextStep(execCtx, currentStep, definition, req.Action)
	if nextStep != "" {
		execCtx.CurrentStep = nextStep
		// 继续执行工作流
		return e.executeWorkflowSteps(ctx, execCtx, definition)
	}

	// 更新实例状态
	return e.updateInstanceStatus(ctx, execCtx)
}

// determineNextStep 确定下一步
func (e *WorkflowEngine) determineNextStep(execCtx *WorkflowExecutionContext, currentStep *WorkflowStep, definition WorkflowDefinition, action string) string {
	if currentStep == nil {
		return ""
	}

	// 查找转换
	transitions := e.findTransitions(definition.Transitions, currentStep.ID)
	for _, transition := range transitions {
		// 检查动作是否匹配
		if len(transition.Actions) == 0 || containsString(transition.Actions, action) {
			// 检查条件
			if transition.Condition == nil || e.evaluateCondition(*transition.Condition, execCtx.Variables) {
				return transition.ToStep
			}
		}
	}

	return ""
}

// containsString 检查字符串数组是否包含指定值
func containsString(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// CompleteWorkflowStepRequest 完成工作流步骤请求
type CompleteWorkflowStepRequest struct {
	InstanceID int                    `json:"instance_id" binding:"required"`
	Action     string                 `json:"action" binding:"required"`
	Variables  map[string]interface{} `json:"variables"`
	Data       map[string]interface{} `json:"data"`
	Comment    string                 `json:"comment"`
	UserID     int                    `json:"user_id" binding:"required"`
}
