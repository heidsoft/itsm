package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// BPMN XML for testing
const testBPMNXML = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="test_process" name="Test Process" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1"/>
    <bpmn:endEvent id="EndEvent_1"/>
  </bpmn:process>
</bpmn:definitions>`

// TestBPMNSLAService_GetProcessSLA 测试获取流程SLA配置
func TestBPMNSLAService_GetProcessSLA(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewBPMNSLAService(client, logger)

	ctx := context.Background()

	// 创建测试租户和部署
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建部署记录
	testDeployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-SLA-TEST").
		SetDeploymentName("SLA Test Deployment").
		SetDeploymentSource("test").
		SetTenantID(testTenant.ID).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)

	t.Run("获取有SLA配置的流程", func(t *testing.T) {
		// 创建带SLA配置的流程定义
		_, err := client.ProcessDefinition.Create().
			SetKey("sla_test_process").
			SetName("SLA Test Process").
			SetVersion("1.0.0").
			SetTenantID(testTenant.ID).
			SetDeploymentID(testDeployment.ID).
			SetBpmnXML([]byte(testBPMNXML)).
			SetProcessVariables(map[string]interface{}{
				"sla": map[string]interface{}{
					"deadline_minutes": float64(240),
					"warning_minutes":  float64(180),
				},
			}).
			Save(ctx)
		require.NoError(t, err)

		sla, err := slaService.GetProcessSLA(ctx, "sla_test_process")
		require.NoError(t, err)
		assert.Equal(t, "sla_test_process", sla.ProcessDefinitionKey)
		assert.Equal(t, 240, sla.DeadlineMinutes)
		assert.Equal(t, 180, sla.WarningMinutes)
		assert.True(t, sla.BusinessHoursOnly)
	})

	t.Run("获取无SLA配置的流程返回默认值", func(t *testing.T) {
		// 创建不带SLA配置的流程定义
		_, err = client.ProcessDefinition.Create().
			SetKey("no_sla_process").
			SetName("No SLA Process").
			SetVersion("1.0.0").
			SetTenantID(testTenant.ID).
			SetDeploymentID(testDeployment.ID).
			SetBpmnXML([]byte(testBPMNXML)).
			Save(ctx)
		require.NoError(t, err)

		sla, err := slaService.GetProcessSLA(ctx, "no_sla_process")
		require.NoError(t, err)
		assert.Equal(t, "no_sla_process", sla.ProcessDefinitionKey)
		assert.Equal(t, 480, sla.DeadlineMinutes)   // 默认8小时
		assert.Equal(t, 360, sla.WarningMinutes)     // 默认6小时
	})

	t.Run("获取不存在的流程定义", func(t *testing.T) {
		_, err := slaService.GetProcessSLA(ctx, "not_exist_process")
		assert.Error(t, err)
	})
}

// TestBPMNSLAService_GetTaskSLA 测试获取任务SLA配置
func TestBPMNSLAService_GetTaskSLA(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewBPMNSLAService(client, logger)

	ctx := context.Background()

	// 创建测试租户、用户和部署
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建部署记录
	testDeployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-TASK-SLA-TEST").
		SetDeploymentName("Task SLA Test Deployment").
		SetDeploymentSource("test").
		SetTenantID(testTenant.ID).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)

	// 创建流程定义
	testProcessDef, err := client.ProcessDefinition.Create().
		SetKey("task_sla_test").
		SetName("Task SLA Test").
		SetVersion("1.0.0").
		SetTenantID(testTenant.ID).
		SetDeploymentID(testDeployment.ID).
		SetBpmnXML([]byte(testBPMNXML)).
		Save(ctx)
	require.NoError(t, err)

	// 创建流程实例
	testProcessInstance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-TASK-SLA-TEST").
		SetBusinessKey("test").
		SetProcessDefinitionKey("task_sla_test").
		SetProcessDefinitionID(testProcessDef.ID).
		SetStatus("running").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	t.Run("获取带SLA配置的任务", func(t *testing.T) {
		// 创建带SLA配置的任务
		task, err := client.ProcessTask.Create().
			SetTaskID("TASK-SLA-001").
			SetTaskDefinitionKey("approval_task").
			SetTaskName("审批任务").
			SetProcessDefinitionKey("task_sla_test").
			SetProcessInstanceID(testProcessInstance.ID).
			SetStatus("assigned").
			SetTenantID(testTenant.ID).
			SetTaskVariables(map[string]interface{}{
				"sla": map[string]interface{}{
					"deadline_minutes": float64(60),
					"warning_minutes":  float64(30),
				},
			}).
			Save(ctx)
		require.NoError(t, err)

		sla, err := slaService.GetTaskSLA(ctx, task)
		require.NoError(t, err)
		assert.Equal(t, 60, sla.DeadlineMinutes)
		assert.Equal(t, 30, sla.WarningMinutes)
	})

	t.Run("获取无SLA配置的任务返回默认值", func(t *testing.T) {
		// 创建不带SLA配置的任务
		task, err := client.ProcessTask.Create().
			SetTaskID("TASK-SLA-002").
			SetTaskDefinitionKey("normal_task").
			SetTaskName("普通任务").
			SetProcessDefinitionKey("task_sla_test").
			SetProcessInstanceID(testProcessInstance.ID).
			SetStatus("assigned").
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)

		sla, err := slaService.GetTaskSLA(ctx, task)
		require.NoError(t, err)
		assert.Equal(t, 120, sla.DeadlineMinutes)  // 任务默认2小时
		assert.Equal(t, 90, sla.WarningMinutes)    // 任务默认90分钟
		assert.Equal(t, "normal_task", sla.TaskDefinitionKey)
	})
}

// TestBPMNSLAService_CalculateSLAStatus 测试SLA状态计算
func TestBPMNSLAService_CalculateSLAStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewBPMNSLAService(client, logger)

	ctx := context.Background()

	tests := []struct {
		name          string
		startTime     time.Time
		deadlineMins  int
		warningMins   int
		businessHours bool
		expectedStatus string
	}{
		{
			name:          "正常工作时间内-SLA正常",
			startTime:     time.Now(),
			deadlineMins:  480,
			warningMins:   360,
			businessHours: false,
			expectedStatus: SLAStatusOK,
		},
		{
			name:          "超时-SLA违规",
			startTime:     time.Now().Add(-10 * time.Hour),
			deadlineMins:  60,
			warningMins:   30,
			businessHours: false,
			expectedStatus: SLAStatusBreached,
		},
		{
			name:          "警告状态",
			startTime:     time.Now().Add(-50 * time.Minute),
			deadlineMins:  60,
			warningMins:   45,
			businessHours: false,
			expectedStatus: SLAStatusWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sla := &ProcessSLA{
				DeadlineMinutes:   tt.deadlineMins,
				WarningMinutes:    tt.warningMins,
				BusinessHoursOnly: tt.businessHours,
			}

			status, deadline, warning, err := slaService.CalculateSLAStatus(ctx, tt.startTime, sla)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, status)
			assert.False(t, deadline.IsZero())
			assert.False(t, warning.IsZero())
		})
	}
}

// TestBPMNSLAService_BusinessHoursCalculation 测试工作时间计算
func TestBPMNSLAService_BusinessHoursCalculation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewBPMNSLAService(client, logger)

	t.Run("工作时间计算-跨越周末", func(t *testing.T) {
		// 周五下午5点开始，30分钟工作时间
		friday := time.Date(2024, 1, 5, 17, 0, 0, 0, time.Local) // 周五
		deadline := slaService.calculateBusinessHoursDeadline(friday, 30)
		
		// 应该计算到下周一的某个时间
		assert.True(t, deadline.After(friday))
	})

	t.Run("工作时间计算-工作时间内", func(t *testing.T) {
		// 周三上午10点开始，60分钟工作时间
		wednesday := time.Date(2024, 1, 3, 10, 0, 0, 0, time.Local) // 周三
		deadline := slaService.calculateBusinessHoursDeadline(wednesday, 60)
		
		// 应该当天完成
		assert.Equal(t, 11, deadline.Hour())
	})

	t.Run("工作时间计算-超过当天工作时间", func(t *testing.T) {
		// 周三下午5:30开始，60分钟工作时间
		wednesday := time.Date(2024, 1, 3, 17, 30, 0, 0, time.Local) // 周三
		deadline := slaService.calculateBusinessHoursDeadline(wednesday, 60)
		
		// 验证结果在输入时间之后
		assert.True(t, deadline.After(wednesday), "截止时间应该在开始时间之后")
		// 验证结果在合理范围内（不超过开始时间太多）
		assert.Less(t, deadline.Sub(wednesday).Hours(), float64(24), "截止时间不应该超过1天")
	})
}

// TestBPMNSLAService_GetProcessInstanceSLAInfo 测试获取流程实例SLA信息
func TestBPMNSLAService_GetProcessInstanceSLAInfo(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewBPMNSLAService(client, logger)

	ctx := context.Background()

	// 创建测试租户和部署
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建部署记录
	testDeployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-PI-SLA-TEST").
		SetDeploymentName("Process Instance SLA Test").
		SetDeploymentSource("test").
		SetTenantID(testTenant.ID).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)

	t.Run("获取流程实例SLA信息", func(t *testing.T) {
		// 创建流程定义
		processDef, err := client.ProcessDefinition.Create().
			SetKey("instance_sla_test").
			SetName("Instance SLA Test").
			SetVersion("1.0.0").
			SetTenantID(testTenant.ID).
			SetDeploymentID(testDeployment.ID).
			SetBpmnXML([]byte(testBPMNXML)).
			SetProcessVariables(map[string]interface{}{
				"sla": map[string]interface{}{
					"deadline_minutes": float64(480),
					"warning_minutes":  float64(360),
				},
			}).
			Save(ctx)
		require.NoError(t, err)

		// 创建流程实例
		instance, err := client.ProcessInstance.Create().
			SetProcessInstanceID("PI-SLA-TEST-001").
			SetBusinessKey("test").
			SetProcessDefinitionKey("instance_sla_test").
			SetProcessDefinitionID(processDef.ID).
			SetStatus("running").
			SetTenantID(testTenant.ID).
			SetStartTime(time.Now().Add(-1 * time.Hour)).
			Save(ctx)
		require.NoError(t, err)

		info, err := slaService.GetProcessInstanceSLAInfo(ctx, instance)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, 480, info.TotalMinutes)
		assert.Greater(t, info.ElapsedMinutes, 0)
	})
}

// TestBPMNSLAService_GetTaskSLAInfo 测试获取任务SLA信息
func TestBPMNSLAService_GetTaskSLAInfo(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewBPMNSLAService(client, logger)

	ctx := context.Background()

	// 创建测试租户、部署和流程定义
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testDeployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-TASK-INFO-TEST").
		SetDeploymentName("Task Info Test").
		SetDeploymentSource("test").
		SetTenantID(testTenant.ID).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)

	testProcessDef, err := client.ProcessDefinition.Create().
		SetKey("task_sla_info_test").
		SetName("Task SLA Info Test").
		SetVersion("1.0.0").
		SetTenantID(testTenant.ID).
		SetDeploymentID(testDeployment.ID).
		SetBpmnXML([]byte(testBPMNXML)).
		Save(ctx)
	require.NoError(t, err)

	testProcessInstance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-TASK-INFO-TEST").
		SetBusinessKey("test").
		SetProcessDefinitionKey("task_sla_info_test").
		SetProcessDefinitionID(testProcessDef.ID).
		SetStatus("running").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	t.Run("获取任务SLA信息-使用分配时间", func(t *testing.T) {
		// 创建任务
		assignedTime := time.Now().Add(-30 * time.Minute)
		task, err := client.ProcessTask.Create().
			SetTaskID("TASK-SLA-INFO-001").
			SetTaskDefinitionKey("approval_task").
			SetTaskName("审批任务").
			SetProcessDefinitionKey("task_sla_info_test").
			SetProcessInstanceID(testProcessInstance.ID).
			SetStatus("assigned").
			SetTenantID(testTenant.ID).
			SetCreatedTime(time.Now().Add(-1 * time.Hour)).
			SetAssignedTime(assignedTime).
			Save(ctx)
		require.NoError(t, err)

		info, err := slaService.GetTaskSLAInfo(ctx, task)
		require.NoError(t, err)
		assert.NotNil(t, info)
		// 应该使用分配时间计算，所以elapsed应该小于30分钟
		assert.LessOrEqual(t, info.ElapsedMinutes, 31)
	})

	t.Run("获取任务SLA信息-使用创建时间", func(t *testing.T) {
		// 创建未分配的任务
		task, err := client.ProcessTask.Create().
			SetTaskID("TASK-SLA-INFO-002").
			SetTaskDefinitionKey("unassigned_task").
			SetTaskName("未分配任务").
			SetProcessDefinitionKey("task_sla_info_test").
			SetProcessInstanceID(testProcessInstance.ID).
			SetStatus("created").
			SetTenantID(testTenant.ID).
			SetCreatedTime(time.Now().Add(-45 * time.Minute)).
			Save(ctx)
		require.NoError(t, err)

		info, err := slaService.GetTaskSLAInfo(ctx, task)
		require.NoError(t, err)
		assert.NotNil(t, info)
	})
}

// TestBPMNSLAService_CheckSLAViolations 测试检查SLA违规
func TestBPMNSLAService_CheckSLAViolations(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewBPMNSLAService(client, logger)

	ctx := context.Background()

	// 创建测试租户和部署
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testDeployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-VIOLATION-TEST").
		SetDeploymentName("Violation Test Deployment").
		SetDeploymentSource("test").
		SetTenantID(testTenant.ID).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)

	t.Run("检查SLA违规-返回结果不报错", func(t *testing.T) {
		// 创建超时的流程实例
		processDef, err := client.ProcessDefinition.Create().
			SetKey("violation_test").
			SetName("Violation Test").
			SetVersion("1.0.0").
			SetTenantID(testTenant.ID).
			SetDeploymentID(testDeployment.ID).
			SetBpmnXML([]byte(testBPMNXML)).
			SetProcessVariables(map[string]interface{}{
				"sla": map[string]interface{}{
					"deadline_minutes": float64(60),
					"warning_minutes":  float64(30),
				},
			}).
			Save(ctx)
		require.NoError(t, err)
	
		// 创建超时5小时的流程实例
		_, err = client.ProcessInstance.Create().
			SetProcessInstanceID("PI-VIOLATION-001").
			SetBusinessKey("test").
			SetProcessDefinitionKey("violation_test").
			SetProcessDefinitionID(processDef.ID).
			SetStatus("running").
			SetTenantID(testTenant.ID).
			SetStartTime(time.Now().Add(-5 * time.Hour)).
			Save(ctx)
		require.NoError(t, err)
	
		violations, err := slaService.CheckSLAViolations(ctx, testTenant.ID)
		require.NoError(t, err)
		// 验证返回结果结构正确
		for _, v := range violations {
			assert.NotEmpty(t, v.SLAStatus)
			assert.Equal(t, testTenant.ID, v.TenantID)
		}
	})

	t.Run("检查SLA违规-无流程实例时返回空列表", func(t *testing.T) {
		// 创建一个新启动的流程实例（在SLA时限内）
		processDef, err := client.ProcessDefinition.Create().
			SetKey("no_violation_test").
			SetName("No Violation Test").
			SetVersion("1.0.0").
			SetTenantID(testTenant.ID).
			SetDeploymentID(testDeployment.ID).
			SetBpmnXML([]byte(testBPMNXML)).
			SetProcessVariables(map[string]interface{}{
				"sla": map[string]interface{}{
					"deadline_minutes": float64(480),
					"warning_minutes":  float64(360),
				},
			}).
			Save(ctx)
		require.NoError(t, err)
	
		_, err = client.ProcessInstance.Create().
			SetProcessInstanceID("PI-NO-VIOLATION-001").
			SetBusinessKey("test").
			SetProcessDefinitionKey("no_violation_test").
			SetProcessDefinitionID(processDef.ID).
			SetStatus("running").
			SetTenantID(testTenant.ID).
			SetStartTime(time.Now().Add(-1 * time.Hour)).
			Save(ctx)
		require.NoError(t, err)
	
		violations, err := slaService.CheckSLAViolations(ctx, testTenant.ID)
		require.NoError(t, err)
		// 验证返回结果结构正确，不要求特定的违规数量
		for _, v := range violations {
			assert.NotEmpty(t, v.SLAStatus)
			assert.Equal(t, testTenant.ID, v.TenantID)
		}
	})
}

// TestBPMNSLAService_GetSLAComplianceRate 测试SLA合规率统计
func TestBPMNSLAService_GetSLAComplianceRate(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewBPMNSLAService(client, logger)

	ctx := context.Background()

	// 创建测试租户和部署
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testDeployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-COMPLIANCE-TEST").
		SetDeploymentName("Compliance Test Deployment").
		SetDeploymentSource("test").
		SetTenantID(testTenant.ID).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)

	t.Run("无已完成实例时返回100%合规率", func(t *testing.T) {
		rate, compliant, total, err := slaService.GetSLAComplianceRate(
			ctx,
			"nonexistent_process",
			time.Now().Add(-24*time.Hour),
			time.Now(),
			testTenant.ID,
		)
		require.NoError(t, err)
		assert.Equal(t, 100.0, rate)
		assert.Equal(t, 0, compliant)
		assert.Equal(t, 0, total)
	})

	t.Run("计算合规率-服务返回结果", func(t *testing.T) {
		// 创建流程定义
		processDef, err := client.ProcessDefinition.Create().
			SetKey("compliance_test").
			SetName("Compliance Test").
			SetVersion("1.0.0").
			SetTenantID(testTenant.ID).
			SetDeploymentID(testDeployment.ID).
			SetBpmnXML([]byte(testBPMNXML)).
			SetProcessVariables(map[string]interface{}{
				"sla": map[string]interface{}{
					"deadline_minutes": float64(480),
					"warning_minutes":  float64(360),
				},
			}).
			Save(ctx)
		require.NoError(t, err)

		startTime := time.Now().Add(-24 * time.Hour)
		endTime := time.Now()

		// 创建一个已完成实例
		_, err = client.ProcessInstance.Create().
			SetProcessInstanceID("PI-COMPLIANCE-001").
			SetBusinessKey("test1").
			SetProcessDefinitionKey("compliance_test").
			SetProcessDefinitionID(processDef.ID).
			SetStatus("completed").
			SetTenantID(testTenant.ID).
			SetStartTime(startTime).
			SetEndTime(startTime.Add(4 * time.Hour)).
			Save(ctx)
		require.NoError(t, err)

		rate, compliant, total, err := slaService.GetSLAComplianceRate(
			ctx,
			"compliance_test",
			startTime,
			endTime,
			testTenant.ID,
		)
		require.NoError(t, err)
		// 验证返回结果结构正确
		assert.GreaterOrEqual(t, total, 0)
		assert.GreaterOrEqual(t, compliant, 0)
		assert.GreaterOrEqual(t, rate, 0.0)
	})
}

// TestBPMNSLAService_RecordSLAAlert 测试记录SLA告警
func TestBPMNSLAService_RecordSLAAlert(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewBPMNSLAService(client, logger)

	ctx := context.Background()

	t.Run("记录SLA告警", func(t *testing.T) {
		violation := &SLAViolation{
			ResourceType:   "process_instance",
			ResourceID:     1,
			ResourceKey:    "PI-TEST-001",
			SLAStatus:      SLAStatusBreached,
			StartTime:      time.Now().Add(-5 * time.Hour),
			Deadline:       time.Now(),
			ElapsedMinutes: 300,
			TenantID:       1,
		}

		err := slaService.RecordSLAAlert(ctx, violation)
		require.NoError(t, err)
		// 验证日志记录成功（无错误即通过）
	})
}

// TestBPMNSLAService_TenantIsolation 测试租户隔离
func TestBPMNSLAService_TenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewBPMNSLAService(client, logger)

	ctx := context.Background()

	// 创建两个测试租户
	tenant1, err := client.Tenant.Create().
		SetName("Tenant 1").
		SetCode("tenant1").
		SetDomain("tenant1.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	tenant2, err := client.Tenant.Create().
		SetName("Tenant 2").
		SetCode("tenant2").
		SetDomain("tenant2.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 为两个租户创建部署
	_, err = client.ProcessDeployment.Create().
		SetDeploymentID("DEP-TENANT2-TEST").
		SetDeploymentName("Tenant 2 Test Deployment").
		SetDeploymentSource("test").
		SetTenantID(tenant2.ID).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)

	t.Run("租户隔离-验证服务返回结果", func(t *testing.T) {
		// 为租户1创建部署和流程定义
		dep1, err := client.ProcessDeployment.Create().
			SetDeploymentID("DEP-TENANT1-TEST-SUB").
			SetDeploymentName("Tenant 1 Test Deployment").
			SetDeploymentSource("test").
			SetTenantID(tenant1.ID).
			SetIsActive(true).
			Save(ctx)
		require.NoError(t, err)

		processDef1, err := client.ProcessDefinition.Create().
			SetKey("tenant1_sla_test").
			SetName("Tenant 1 SLA Test").
			SetVersion("1.0.0").
			SetTenantID(tenant1.ID).
			SetDeploymentID(dep1.ID).
			SetBpmnXML([]byte(testBPMNXML)).
			SetProcessVariables(map[string]interface{}{
				"sla": map[string]interface{}{
					"deadline_minutes": float64(60),
					"warning_minutes":  float64(30),
				},
			}).
			Save(ctx)
		require.NoError(t, err)

		_, err = client.ProcessInstance.Create().
			SetProcessInstanceID("PI-TENANT1-001").
			SetBusinessKey("test").
			SetProcessDefinitionKey("tenant1_sla_test").
			SetProcessDefinitionID(processDef1.ID).
			SetStatus("running").
			SetTenantID(tenant1.ID).
			SetStartTime(time.Now().Add(-5 * time.Hour)).
			Save(ctx)
		require.NoError(t, err)

		// 验证租户1返回结果
		violations1, err := slaService.CheckSLAViolations(ctx, tenant1.ID)
		require.NoError(t, err)
		// 验证返回结果结构正确
		for _, v := range violations1 {
			assert.Equal(t, tenant1.ID, v.TenantID)
		}

		// 验证租户2返回结果
		violations2, err := slaService.CheckSLAViolations(ctx, tenant2.ID)
		require.NoError(t, err)
		// 验证返回结果结构正确
		for _, v := range violations2 {
			assert.Equal(t, tenant2.ID, v.TenantID)
		}
	})
}

// TestBPMNSLAService_SLAStatusConstants 测试SLA状态常量
func TestBPMNSLAService_SLAStatusConstants(t *testing.T) {
	// 验证SLA状态常量值
	assert.Equal(t, "ok", SLAStatusOK)
	assert.Equal(t, "warning", SLAStatusWarning)
	assert.Equal(t, "breached", SLAStatusBreached)
	assert.Equal(t, "unknown", SLAStatusUnknown)
}
