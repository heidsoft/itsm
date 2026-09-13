package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"

	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rejectionTestBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  targetNamespace="http://test">
  <bpmn:process id="rejection_test" name="拒绝策略测试" isExecutable="true">
    <bpmn:startEvent id="Start_1" name="开始"/>
    <bpmn:userTask id="ApprovalTask" name="审批" taskPurpose="approval" rejectStrategy="%s" commentRequiredOnReject="%s"/>
    <bpmn:userTask id="ReworkTask" name="返工" taskPurpose="rework"/>
    <bpmn:endEvent id="End_1" name="结束"/>
    <bpmn:endEvent id="End_2" name="终止结束"/>
    <bpmn:sequenceFlow id="F_Start" sourceRef="Start_1" targetRef="ApprovalTask"/>
    <bpmn:sequenceFlow id="F_Reject" sourceRef="ApprovalTask" targetRef="ReworkTask">
      <bpmn:conditionExpression>approvalAction == 'reject'</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="F_Approve" sourceRef="ApprovalTask" targetRef="End_1">
      <bpmn:conditionExpression>approvalAction == 'approve'</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="F_Rework_End" sourceRef="ReworkTask" targetRef="End_1"/>
  </bpmn:process>
</bpmn:definitions>`

// setupRejectionFixture returns (engine, workflowCtx, tenantID, instanceID, taskBusinessID).
func setupRejectionFixture(t *testing.T, rejectStrategy string, commentRequired bool) (*CustomProcessEngine, context.Context, int, int, string) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:rejection_test_"+t.Name()+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Rejection Tenant").SetCode("reject").SetDomain("reject.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	approver, err := client.User.Create().
		SetUsername("approver").SetEmail("approver@test.com").SetName("审批人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	requester, err := client.User.Create().
		SetUsername("requester").SetEmail("requester@test.com").SetName("发起人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	commentRequiredStr := "false"
	if commentRequired {
		commentRequiredStr = "true"
	}
	xml := []byte(fmt.Sprintf(rejectionTestBPMN, rejectStrategy, commentRequiredStr))

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-REJECT").SetDeploymentName("Reject Deploy").SetDeploymentTime(time.Now()).
		SetDeployedBy("test").SetIsActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey("rejection_test").SetName("拒绝测试").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).SetBpmnXML(xml).
		SetDeploymentID(deployment.ID).SetDeployedAt(time.Now()).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-REJECT-1").
		SetProcessDefinitionKey(def.Key).SetProcessDefinitionID(def.ID).
		SetStatus("running").
		SetVariables(map[string]interface{}{
			"requester_id":  requester.ID,
			"triggered_by":  requester.Username,
			"business_type": "ticket", "business_id": "1",
		}).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	taskVars := map[string]interface{}{
		"taskPurpose":             "approval",
		"rejectStrategy":          rejectStrategy,
		"commentRequiredOnReject": commentRequired,
		"allowDelegate":           false,
		"allowAddApprover":        false,
	}
	task, err := client.ProcessTask.Create().
		SetTaskID("TASK-REJECT-1").
		SetTaskDefinitionKey("ApprovalTask").
		SetTaskName("审批").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetStatus("created").
		SetAssignee(approver.Username).
		SetTenantID(tenant.ID).
		SetTaskVariables(taskVars).
		Save(ctx)
	require.NoError(t, err)

	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, approver.ID)

	return engine, workflowCtx, tenant.ID, instance.ID, task.TaskID
}

func TestCompleteTask_CommentRequiredOnReject(t *testing.T) {
	engine, ctx, _, _, taskID := setupRejectionFixture(t, "terminate", true)

	variables := map[string]interface{}{
		"approvalAction":  "reject",
		"approvalResult":  "rejected",
		"approvalComment": "",
	}
	err := engine.CompleteTask(ctx, taskID, variables)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "拒绝时必须填写意见")
}

func TestCompleteTask_CommentProvided_Passes(t *testing.T) {
	engine, ctx, _, _, taskID := setupRejectionFixture(t, "terminate", true)

	variables := map[string]interface{}{
		"approvalAction":  "reject",
		"approvalResult":  "rejected",
		"approvalComment": "预算不足",
	}
	err := engine.CompleteTask(ctx, taskID, variables)
	require.NoError(t, err)
}

func TestCompleteTask_ToRequester_CancelsOtherTasksAndRoutesToRework(t *testing.T) {
	engine, ctx, tenantID, instanceID, taskID := setupRejectionFixture(t, "to_requester", false)

	otherTask, err := engine.client.ProcessTask.Create().
		SetTaskID("TASK-OTHER-1").
		SetTaskDefinitionKey("SomeOtherNode").
		SetTaskName("其他任务").
		SetProcessDefinitionKey("rejection_test").
		SetProcessInstanceID(instanceID).
		SetStatus("created").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	variables := map[string]interface{}{
		"approvalAction":  "reject",
		"approvalResult":  "rejected",
		"approvalComment": "需要修改",
	}
	err = engine.CompleteTask(ctx, taskID, variables)
	require.NoError(t, err)

	other, err := engine.client.ProcessTask.Get(ctx, otherTask.ID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", other.Status, "to_requester 应取消其他活动任务")

	instance, err := engine.client.ProcessInstance.Get(ctx, instanceID)
	require.NoError(t, err)
	assert.Equal(t, "running", instance.Status, "to_requester 不应终止流程实例")

	reworkRequested, ok := instance.Variables["rework_requested"]
	assert.True(t, ok, "流程实例应包含 rework_requested 变量")
	assert.Equal(t, true, reworkRequested)

	reworkTasks, err := engine.client.ProcessTask.Query().
		Where(
			processtask.ProcessInstanceID(instanceID),
			processtask.TaskDefinitionKey("ReworkTask"),
			processtask.StatusNEQ("cancelled"),
		).
		All(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(reworkTasks), 1, "应创建返工任务")
}

func TestCompleteTask_Terminate_EndsProcess(t *testing.T) {
	engine, ctx, _, instanceID, taskID := setupRejectionFixture(t, "terminate", false)

	variables := map[string]interface{}{
		"approvalAction":  "reject",
		"approvalResult":  "rejected",
		"approvalComment": "不同意",
	}
	err := engine.CompleteTask(ctx, taskID, variables)
	require.NoError(t, err)

	instance, err := engine.client.ProcessInstance.Get(ctx, instanceID)
	require.NoError(t, err)
	assert.Equal(t, "terminated", instance.Status, "terminate 策略应终止流程实例")
}

func TestCompleteTask_Approve_RoutesNormally(t *testing.T) {
	engine, ctx, _, instanceID, taskID := setupRejectionFixture(t, "terminate", false)

	variables := map[string]interface{}{
		"approvalAction":  "approve",
		"approvalResult":  "approved",
		"approvalComment": "同意",
	}
	err := engine.CompleteTask(ctx, taskID, variables)
	require.NoError(t, err)

	instance, err := engine.client.ProcessInstance.Get(ctx, instanceID)
	require.NoError(t, err)
	assert.NotEqual(t, "terminated", instance.Status, "审批通过不应终止流程")
}
