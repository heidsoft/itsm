package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== 测试设置辅助函数 ====================

func setupWorkflowEngineTest(t *testing.T) (*ent.Client, *WorkflowEngine, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	engine := NewWorkflowEngine(client)
	ctx := context.Background()
	return client, engine, ctx
}

func createWorkflowEngineTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Test Tenant " + suffix).
		SetCode("test" + suffix).
		SetDomain("test" + suffix + ".com").
		SetStatus("active").
		Save(ctx)
}

func createWorkflowEngineTestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string) (*ent.User, error) {
	return client.User.Create().
		SetUsername("testuser" + suffix).
		SetEmail("test" + suffix + "@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
}

// 创建简单的工作流定义
func createSimpleWorkflowDefinition() WorkflowDefinition {
	return WorkflowDefinition{
		ID:         "simple-workflow",
		Name:       "简单工作流",
		Version:    "1.0.0",
		StartEvent: "start",
		EndEvent:   "end",
		Steps: []WorkflowStep{
			{
				ID:   "start",
				Name: "开始",
				Type: "start",
			},
			{
				ID:   "task1",
				Name: "任务1",
				Type: "task",
			},
			{
				ID:   "end",
				Name: "结束",
				Type: "end",
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
				ID:       "t2",
				FromStep: "task1",
				ToStep:   "end",
				Auto:     false,
			},
		},
		Variables: map[string]interface{}{},
	}
}

// 创建带审批的工作流定义
func createApprovalWorkflowDefinition() WorkflowDefinition {
	return WorkflowDefinition{
		ID:         "approval-workflow",
		Name:       "审批工作流",
		Version:    "1.0.0",
		StartEvent: "start",
		EndEvent:   "end",
		Steps: []WorkflowStep{
			{
				ID:   "start",
				Name: "开始",
				Type: "start",
			},
			{
				ID:       "approval1",
				Name:     "审批",
				Type:     "approval",
				Assignee: "manager",
			},
			{
				ID:   "end",
				Name: "结束",
				Type: "end",
			},
		},
		Transitions: []WorkflowTransition{
			{
				ID:       "t1",
				FromStep: "start",
				ToStep:   "approval1",
				Auto:     true,
			},
			{
				ID:       "t2",
				FromStep: "approval1",
				ToStep:   "end",
				Actions:  []string{"approve"},
				Auto:     false,
			},
		},
		Variables: map[string]interface{}{},
	}
}

// ==================== 工作流定义测试 ====================

func TestWorkflowDefinition_Serialization(t *testing.T) {
	def := createSimpleWorkflowDefinition()

	// 序列化
	bytes, err := json.Marshal(def)
	require.NoError(t, err)
	assert.NotEmpty(t, bytes)

	// 反序列化
	var parsed WorkflowDefinition
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)
	assert.Equal(t, def.Name, parsed.Name)
	assert.Equal(t, len(def.Steps), len(parsed.Steps))
	assert.Equal(t, len(def.Transitions), len(parsed.Transitions))
}

// ==================== 工作流执行测试 ====================

func TestWorkflowEngine_ExecuteWorkflow_InstanceNotFound(t *testing.T) {
	client, engine, ctx := setupWorkflowEngineTest(t)
	defer client.Close()

	// 尝试执行不存在的工作流实例
	err := engine.ExecuteWorkflow(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "获取工作流实例失败")
}

func TestWorkflowEngine_ExecuteWorkflow_Success(t *testing.T) {
	client, engine, ctx := setupWorkflowEngineTest(t)
	defer client.Close()

	testTenant, err := createWorkflowEngineTestTenant(ctx, client, "exec")
	require.NoError(t, err)

	// 创建工作流定义
	def := createSimpleWorkflowDefinition()
	defBytes, err := json.Marshal(def)
	require.NoError(t, err)

	// 创建工作流
	workflow, err := client.Workflow.Create().
		SetName("测试工作流").
		SetType("ticket").
		SetDefinition(defBytes).
		SetVersion("1.0.0").
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建工作流实例
	instance, err := client.WorkflowInstance.Create().
		SetStatus("running").
		SetCurrentStep("start").
		SetWorkflowID(workflow.ID).
		SetEntityID(1).
		SetEntityType("ticket").
		SetTenantID(testTenant.ID).
		SetStartedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 执行工作流
	err = engine.ExecuteWorkflow(ctx, instance.ID)
	require.NoError(t, err)

	// 验证实例状态已更新
	updated, err := client.WorkflowInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	// 工作流应该从start自动转换到task1
	assert.Equal(t, "task1", updated.CurrentStep)
}

func TestWorkflowEngine_ExecuteWorkflow_WithApproval(t *testing.T) {
	client, engine, ctx := setupWorkflowEngineTest(t)
	defer client.Close()

	testTenant, err := createWorkflowEngineTestTenant(ctx, client, "approval")
	require.NoError(t, err)

	// 创建审批工作流定义
	def := createApprovalWorkflowDefinition()
	defBytes, err := json.Marshal(def)
	require.NoError(t, err)

	// 创建工作流
	workflow, err := client.Workflow.Create().
		SetName("审批工作流").
		SetType("ticket").
		SetDefinition(defBytes).
		SetVersion("1.0.0").
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建工作流实例
	instance, err := client.WorkflowInstance.Create().
		SetStatus("running").
		SetCurrentStep("start").
		SetWorkflowID(workflow.ID).
		SetEntityID(1).
		SetEntityType("ticket").
		SetTenantID(testTenant.ID).
		SetStartedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 执行工作流 - 应该停在审批步骤
	err = engine.ExecuteWorkflow(ctx, instance.ID)
	require.NoError(t, err)

	// 验证停在审批步骤
	updated, err := client.WorkflowInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "approval1", updated.CurrentStep)
	assert.Equal(t, "running", updated.Status)
}

// ==================== 完成工作流步骤测试 ====================

func TestWorkflowEngine_CompleteWorkflowStep_Success(t *testing.T) {
	client, engine, ctx := setupWorkflowEngineTest(t)
	defer client.Close()

	testTenant, err := createWorkflowEngineTestTenant(ctx, client, "complete")
	require.NoError(t, err)

	testUser, err := createWorkflowEngineTestUser(ctx, client, testTenant.ID, "complete")
	require.NoError(t, err)

	// 创建简单工作流定义
	def := createSimpleWorkflowDefinition()
	defBytes, err := json.Marshal(def)
	require.NoError(t, err)

	// 创建工作流
	workflow, err := client.Workflow.Create().
		SetName("测试工作流").
		SetType("ticket").
		SetDefinition(defBytes).
		SetVersion("1.0.0").
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建工作流实例，停在任务步骤
	instance, err := client.WorkflowInstance.Create().
		SetStatus("running").
		SetCurrentStep("task1").
		SetWorkflowID(workflow.ID).
		SetEntityID(1).
		SetEntityType("ticket").
		SetTenantID(testTenant.ID).
		SetStartedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 完成任务步骤
	req := &CompleteWorkflowStepRequest{
		InstanceID: instance.ID,
		Action:     "complete",
		UserID:     testUser.ID,
		Variables:  map[string]interface{}{},
	}

	err = engine.CompleteWorkflowStep(ctx, req)
	require.NoError(t, err)

	// 验证已移动到结束步骤并完成
	updated, err := client.WorkflowInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)
	assert.NotNil(t, updated.CompletedAt)
}

func TestWorkflowEngine_CompleteWorkflowStep_InstanceNotFound(t *testing.T) {
	client, engine, ctx := setupWorkflowEngineTest(t)
	defer client.Close()

	testTenant, err := createWorkflowEngineTestTenant(ctx, client, "stepnotfound")
	require.NoError(t, err)

	testUser, err := createWorkflowEngineTestUser(ctx, client, testTenant.ID, "notfound")
	require.NoError(t, err)

	req := &CompleteWorkflowStepRequest{
		InstanceID: 99999,
		Action:     "complete",
		UserID:     testUser.ID,
	}

	err = engine.CompleteWorkflowStep(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "获取工作流实例失败")
}

// ==================== 条件评估测试 ====================

func TestWorkflowEngine_EvaluateCondition_FieldEquals(t *testing.T) {
	client, engine, _ := setupWorkflowEngineTest(t)
	defer client.Close()

	variables := map[string]interface{}{
		"status": "approved",
	}

	condition := WorkflowCondition{
		Type:     "field",
		Field:    "status",
		Operator: "equals",
		Value:    "approved",
	}

	result := engine.evaluateCondition(condition, variables)
	assert.True(t, result)

	// 测试不等于
	condition.Value = "rejected"
	result = engine.evaluateCondition(condition, variables)
	assert.False(t, result)
}

func TestWorkflowEngine_EvaluateCondition_FieldNotEquals(t *testing.T) {
	client, engine, _ := setupWorkflowEngineTest(t)
	defer client.Close()

	variables := map[string]interface{}{
		"status": "approved",
	}

	condition := WorkflowCondition{
		Type:     "field",
		Field:    "status",
		Operator: "not_equals",
		Value:    "rejected",
	}

	result := engine.evaluateCondition(condition, variables)
	assert.True(t, result)
}

func TestWorkflowEngine_EvaluateCondition_ApprovalStatus(t *testing.T) {
	client, engine, _ := setupWorkflowEngineTest(t)
	defer client.Close()

	variables := map[string]interface{}{
		"approval_status": "approved",
	}

	condition := WorkflowCondition{
		Type:     "approval",
		Operator: "equals",
		Value:    "approved",
	}

	result := engine.evaluateCondition(condition, variables)
	assert.True(t, result)
}

func TestWorkflowEngine_EvaluateCondition_FieldNotFound(t *testing.T) {
	client, engine, _ := setupWorkflowEngineTest(t)
	defer client.Close()

	variables := map[string]interface{}{
		"other_field": "value",
	}

	condition := WorkflowCondition{
		Type:     "field",
		Field:    "missing_field",
		Operator: "equals",
		Value:    "something",
	}

	result := engine.evaluateCondition(condition, variables)
	assert.False(t, result)
}

// ==================== 辅助函数测试 ====================

func TestWorkflowEngine_FindStep(t *testing.T) {
	client, engine, _ := setupWorkflowEngineTest(t)
	defer client.Close()

	steps := []WorkflowStep{
		{ID: "step1", Name: "步骤1", Type: "task"},
		{ID: "step2", Name: "步骤2", Type: "approval"},
		{ID: "step3", Name: "步骤3", Type: "end"},
	}

	// 测试找到步骤
	step := engine.findStep(steps, "step2")
	require.NotNil(t, step)
	assert.Equal(t, "步骤2", step.Name)

	// 测试找不到步骤
	step = engine.findStep(steps, "nonexistent")
	assert.Nil(t, step)
}

func TestWorkflowEngine_FindTransitions(t *testing.T) {
	client, engine, _ := setupWorkflowEngineTest(t)
	defer client.Close()

	transitions := []WorkflowTransition{
		{ID: "t1", FromStep: "start", ToStep: "step1"},
		{ID: "t2", FromStep: "step1", ToStep: "step2"},
		{ID: "t3", FromStep: "step1", ToStep: "end"},
	}

	// 测试找到转换
	found := engine.findTransitions(transitions, "step1")
	assert.Len(t, found, 2)

	// 测试找不到转换
	found = engine.findTransitions(transitions, "nonexistent")
	assert.Len(t, found, 0)
}

func TestContainsString(t *testing.T) {
	slice := []string{"approve", "reject", "comment"}

	assert.True(t, containsString(slice, "approve"))
	assert.True(t, containsString(slice, "reject"))
	assert.False(t, containsString(slice, "delete"))
	assert.False(t, containsString(slice, ""))
}

// ==================== 审批工作流服务测试 ====================

func TestWorkflowApprovalService_CreateApprovalTask(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	engine := NewWorkflowEngine(client)
	service := NewWorkflowApprovalService(client, engine)
	ctx := context.Background()

	testTenant, err := createWorkflowEngineTestTenant(ctx, client, "was")
	require.NoError(t, err)

	testUser, err := createWorkflowEngineTestUser(ctx, client, testTenant.ID, "was")
	require.NoError(t, err)

	// 创建工作流实例定义
	def := createSimpleWorkflowDefinition()
	defBytes, err := json.Marshal(def)
	require.NoError(t, err)

	// 创建工作流定义
	wf, err := client.Workflow.Create().
		SetName("测试工作流").
		SetType("ticket").
		SetDefinition(defBytes).
		SetVersion("1.0.0").
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建工作流实例
	instance, err := client.WorkflowInstance.Create().
		SetStatus("running").
		SetCurrentStep("task1").
		SetWorkflowID(wf.ID).
		SetEntityID(1).
		SetEntityType("ticket").
		SetTenantID(testTenant.ID).
		SetStartedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 创建审批任务
	req := &CreateApprovalTaskRequest{
		InstanceID:   instance.ID,
		StepID:       "task1",
		StepName:     "任务1",
		AssigneeID:   testUser.ID,
		AssigneeName: testUser.Name,
		Priority:     "medium",
	}

	task, err := service.CreateApprovalTask(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, testUser.ID, task.AssigneeID)
	assert.Equal(t, "pending", task.Status)
}
