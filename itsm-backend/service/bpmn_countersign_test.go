package service

import (
	"context"
	"fmt"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processapprovaldecision"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"

	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const countersignBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  id="Definitions_1"
                  targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Start"/>
    <bpmn:userTask id="CounterSignTask" name="会签审批"/>
    <bpmn:endEvent id="EndEvent_1" name="End"/>
    <bpmn:sequenceFlow id="Flow_0" sourceRef="StartEvent_1" targetRef="CounterSignTask"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="CounterSignTask" targetRef="EndEvent_1"/>
  </bpmn:process>
</bpmn:definitions>`

type countersignFixture struct {
	engine   *CustomProcessEngine
	client   *ent.Client
	ctx      context.Context
	tenantID int
	users    map[string]*ent.User
	parentID string
}

func setupCounterSignFixture(t *testing.T, approvalType string, approverCount int, threshold int) *countersignFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:countersign_"+t.Name()+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("CS Tenant").SetCode("cs").SetDomain("cs.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	users := make(map[string]*ent.User)
	for i := 0; i < approverCount; i++ {
		name := fmt.Sprintf("approver%d", i)
		u, err := client.User.Create().
			SetUsername(name).SetEmail(name + "@test.com").SetName("审批人" + name).
			SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
		users[name] = u
	}

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-CS").SetDeploymentName("CS Deploy").
		SetIsActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey("cs_test").SetName("会签测试").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).SetBpmnXML([]byte(countersignBPMN)).
		SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-CS-1").
		SetProcessDefinitionKey(def.Key).SetProcessDefinitionID(def.ID).
		SetStatus("running").
		SetVariables(map[string]interface{}{"business_type": "ticket", "business_id": "1"}).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	parentTask, err := client.ProcessTask.Create().
		SetTaskID("TASK-CS-PARENT").
		SetTaskDefinitionKey("CounterSignTask").
		SetTaskName("会签父任务").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetStatus("assigned").
		SetAssignee("system").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	_ = parentTask

	approvers := make([]string, approverCount)
	for i := 0; i < approverCount; i++ {
		approvers[i] = fmt.Sprintf("approver%d", i)
	}

	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)

	req := &CounterSignRequest{
		ApprovalType: approvalType,
		Approvers:    approvers,
		Threshold:    threshold,
	}
	_, err = engine.TaskService().CreateCounterSignTasks(workflowCtx, "TASK-CS-PARENT", req)
	require.NoError(t, err)

	return &countersignFixture{
		engine:   engine,
		client:   client,
		ctx:      workflowCtx,
		tenantID: tenant.ID,
		users:    users,
		parentID: "TASK-CS-PARENT",
	}
}

func userContext(tenantID, userID int) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID)
	ctx = context.WithValue(ctx, bpmn.BPMNUserIDContextKey, userID)
	return ctx
}

func TestCreateCounterSignTasks_Parallel_AllAssigned(t *testing.T) {
	f := setupCounterSignFixture(t, "parallel", 3, 2)

	subTasks, err := f.client.ProcessTask.Query().
		Where(processtask.ParentTaskID(f.parentID), processtask.TenantID(f.tenantID)).
		All(f.ctx)
	require.NoError(t, err)
	require.Len(t, subTasks, 3)

	for _, task := range subTasks {
		assert.Equal(t, common.ProcessTaskStatusAssigned, task.Status,
			"parallel mode: all sub-tasks should be assigned")
		assert.Equal(t, f.parentID, task.ParentTaskID)
	}
}

func TestCreateCounterSignTasks_Serial_FirstAssignedRestCreated(t *testing.T) {
	f := setupCounterSignFixture(t, "serial", 3, 1)

	subTasks, err := f.client.ProcessTask.Query().
		Where(processtask.ParentTaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Order(processtask.ByID()).
		All(f.ctx)
	require.NoError(t, err)
	require.Len(t, subTasks, 3)

	assert.Equal(t, common.ProcessTaskStatusAssigned, subTasks[0].Status,
		"serial: first task should be assigned")
	assert.Equal(t, "created", subTasks[1].Status,
		"serial: second task should be created (waiting)")
	assert.Equal(t, "created", subTasks[2].Status,
		"serial: third task should be created (waiting)")
}

func TestGetCounterSignStatus_InitiallyPending(t *testing.T) {
	f := setupCounterSignFixture(t, "parallel", 3, 2)

	status, err := f.engine.TaskService().GetCounterSignStatus(f.ctx, f.parentID)
	require.NoError(t, err)

	assert.Equal(t, 3, status.Total)
	assert.Equal(t, 0, status.Completed)
	assert.Equal(t, 0, status.Approved)
	assert.Equal(t, 0, status.Rejected)
	assert.Equal(t, 3, status.Pending)
	assert.Equal(t, "pending", status.Status)
}

func TestVote_Parallel_ThresholdMet_CompletesParent(t *testing.T) {
	f := setupCounterSignFixture(t, "parallel", 3, 2)

	ctx0 := userContext(f.tenantID, f.users["approver0"].ID)
	err := f.engine.TaskService().Vote(ctx0, "TASK-CS-PARENT_countersign_0", &VoteRequest{Approved: true, Comment: "同意"})
	require.NoError(t, err)

	status, err := f.engine.TaskService().GetCounterSignStatus(f.ctx, f.parentID)
	require.NoError(t, err)
	assert.Equal(t, "pending", status.Status, "1/2 votes: still pending")

	ctx1 := userContext(f.tenantID, f.users["approver1"].ID)
	err = f.engine.TaskService().Vote(ctx1, "TASK-CS-PARENT_countersign_1", &VoteRequest{Approved: true, Comment: "同意"})
	require.NoError(t, err)

	parentTask, err := f.client.ProcessTask.Query().
		Where(processtask.TaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Only(f.ctx)
	require.NoError(t, err)
	assert.Equal(t, "completed", parentTask.Status,
		"threshold met: parent task should be completed")

	remainingSubTasks, err := f.client.ProcessTask.Query().
		Where(processtask.ParentTaskID(f.parentID), processtask.TenantID(f.tenantID),
			processtask.StatusNEQ("completed"), processtask.StatusNEQ("cancelled")).
		All(f.ctx)
	assert.Empty(t, remainingSubTasks,
		"threshold met: remaining sub-tasks should be cancelled or completed")
}

func TestVote_Serial_ActivatesNextOnApprove(t *testing.T) {
	f := setupCounterSignFixture(t, "serial", 3, 1)

	ctx0 := userContext(f.tenantID, f.users["approver0"].ID)
	err := f.engine.TaskService().Vote(ctx0, "TASK-CS-PARENT_countersign_0", &VoteRequest{Approved: true, Comment: "同意"})
	require.NoError(t, err)

	subTasks, err := f.client.ProcessTask.Query().
		Where(processtask.ParentTaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Order(processtask.ByID()).
		All(f.ctx)
	require.NoError(t, err)
	require.Len(t, subTasks, 3)

	assert.Equal(t, "completed", subTasks[0].Status, "first task should be completed after vote")

	parentTask, err := f.client.ProcessTask.Query().
		Where(processtask.TaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Only(f.ctx)
	require.NoError(t, err)
	assert.Equal(t, "completed", parentTask.Status,
		"serial threshold=1: first approve should complete parent")
}

func TestVote_DoubleVote_PreventedByCAS(t *testing.T) {
	f := setupCounterSignFixture(t, "parallel", 2, 1)

	ctx0 := userContext(f.tenantID, f.users["approver0"].ID)
	err := f.engine.TaskService().Vote(ctx0, "TASK-CS-PARENT_countersign_0", &VoteRequest{Approved: true, Comment: "第一次"})
	require.NoError(t, err)

	err = f.engine.TaskService().Vote(ctx0, "TASK-CS-PARENT_countersign_0", &VoteRequest{Approved: false, Comment: "第二次"})
	require.Error(t, err, "double vote should fail")
}

func TestVote_WritesProcessApprovalDecision(t *testing.T) {
	f := setupCounterSignFixture(t, "parallel", 2, 1)

	ctx0 := userContext(f.tenantID, f.users["approver0"].ID)
	err := f.engine.TaskService().Vote(ctx0, "TASK-CS-PARENT_countersign_0", &VoteRequest{Approved: true, Comment: "批准"})
	require.NoError(t, err)

	decisions, err := f.client.ProcessApprovalDecision.Query().
		Where(processapprovaldecision.TenantID(f.tenantID)).
		All(f.ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(decisions), 1, "vote should write at least one audit decision")

	found := false
	for _, d := range decisions {
		if d.TaskID == "TASK-CS-PARENT_countersign_0" {
			assert.Equal(t, "approve", d.Action)
			assert.Equal(t, "approved", d.Decision)
			assert.Equal(t, "批准", d.Comment)
			found = true
		}
	}
	assert.True(t, found, "should find decision record for the voted task")
}

func TestVote_CrossTenant_Rejected(t *testing.T) {
	f := setupCounterSignFixture(t, "parallel", 2, 1)

	otherTenantCtx := userContext(f.tenantID+999, f.users["approver0"].ID)

	err := f.engine.TaskService().Vote(otherTenantCtx, "TASK-CS-PARENT_countersign_0", &VoteRequest{Approved: true})
	require.Error(t, err, "cross-tenant vote should fail: task not found in other tenant")
}

func TestVote_UnauthorizedUser_Rejected(t *testing.T) {
	f := setupCounterSignFixture(t, "parallel", 2, 1)

	outsider, err := f.client.User.Create().
		SetUsername("outsider").SetEmail("outsider@test.com").SetName("外部人").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(f.tenantID).
		Save(f.ctx)
	require.NoError(t, err)

	ctx := userContext(f.tenantID, outsider.ID)
	err = f.engine.TaskService().Vote(ctx, "TASK-CS-PARENT_countersign_0", &VoteRequest{Approved: true})
	require.Error(t, err, "non-assignee should not be able to vote")
	assert.Contains(t, err.Error(), "审批人")
}

func TestCreateCounterSignTasks_EmptyApprovers_Error(t *testing.T) {
	f := setupCounterSignFixture(t, "parallel", 1, 1)

	req := &CounterSignRequest{
		ApprovalType: "parallel",
		Approvers:    []string{},
		Threshold:    1,
	}
	_, err := f.engine.TaskService().CreateCounterSignTasks(f.ctx, "TASK-CS-PARENT", req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "审批人")
}
