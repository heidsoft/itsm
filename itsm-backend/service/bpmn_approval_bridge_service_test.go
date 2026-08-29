package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processtask"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	_ "github.com/mattn/go-sqlite3"
)

func newApprovalBridgeTestClient(t *testing.T, dbName string) *ent.Client {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", dbName))
	t.Cleanup(func() { client.Close() })
	return client
}

func setupBridgeTenantAndActor(t *testing.T, client *ent.Client, code string) (tenantID, actorID int) {
	t.Helper()
	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Bridge Tenant " + code).
		SetCode("bridge-" + code).
		SetDomain("bridge-" + code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	actor, err := client.User.Create().
		SetUsername("bridge-approver-" + code).
		SetEmail("bridge-approver-" + code + "@example.com").
		SetName("Bridge Approver " + code).
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	return tenant.ID, actor.ID
}

// createBridgeProcessFixture 创建 运行中流程实例 + 待办用户任务，
// businessKey 采用与 ProcessTriggerService 相同的 "{type}:{id}" 约定。
// BPMN 为单审批节点直通结构：Start → Approval_1 → End。
func createBridgeProcessFixture(t *testing.T, client *ent.Client, tenantID int, keySuffix, businessKey string, assigneeID int) (instanceID, taskID int) {
	return createBridgeProcessFixtureWithDelegate(t, client, tenantID, keySuffix, businessKey, assigneeID, true)
}

// createBridgeProcessFixtureWithDelegate 创建与生产一致的流程/待办任务夹具。
// 生产路径 bpmn_process_engine.go 会把 BPMN userTask 的审批配置（含 allowDelegate）
// 写入 ProcessTask.TaskVariables，DelegateTask 依赖该字段做 fail-closed 校验，
// 因此夹具必须同样写入，否则测的是另一种数据形态。
func createBridgeProcessFixtureWithDelegate(t *testing.T, client *ent.Client, tenantID int, keySuffix, businessKey string, assigneeID int, allowDelegate bool) (instanceID, taskID int) {
	t.Helper()
	ctx := context.Background()

	defKey := "bridge_approval_" + keySuffix
	bpmnXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:itsm="https://github.com/heidsoft/itsm/schema/bpmn" id="Definitions_%s" targetNamespace="https://github.com/heidsoft/itsm"><bpmn:process id="%s" name="Bridge Approval %s" isExecutable="true"><bpmn:startEvent id="StartEvent_1"/><bpmn:userTask id="Approval_1" name="审批" itsm:taskPurpose="approval" itsm:approvalMode="single" itsm:allowDelegate="%t" itsm:assignee="%d"/><bpmn:endEvent id="EndEvent_1"/><bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Approval_1"/><bpmn:sequenceFlow id="Flow_2" sourceRef="Approval_1" targetRef="EndEvent_1"/></bpmn:process></bpmn:definitions>`, defKey, defKey, keySuffix, allowDelegate, assigneeID)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-" + keySuffix).
		SetDeploymentName("Deployment " + keySuffix).
		SetDeploymentTime(time.Now()).
		SetDeployedBy("test").
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey(defKey).
		SetName("Bridge Approval " + keySuffix).
		SetVersion("1").
		SetIsLatest(true).
		SetBpmnXML([]byte(bpmnXML)).
		SetDeploymentID(deployment.ID).
		SetDeployedAt(time.Now()).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	// 从 businessKey 推导业务上下文变量，与 ProcessTriggerService 启动流程时的写入格式一致
	bizType, bizID := businessKey, ""
	if parts := strings.SplitN(businessKey, ":", 2); len(parts) == 2 {
		bizType, bizID = parts[0], parts[1]
	}

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-" + keySuffix).
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetBusinessKey(businessKey).
		SetStatus("running").
		SetVariables(map[string]interface{}{
			"business_type": bizType,
			"business_id":   bizID,
			"business_key":  businessKey,
		}).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	task, err := client.ProcessTask.Create().
		SetTaskID("TASK-" + keySuffix).
		SetTaskDefinitionKey("Approval_1").
		SetTaskName("审批").
		SetTaskType("user_task").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetAssignee(strconv.Itoa(assigneeID)).
		SetStatus("assigned").
		SetTaskVariables(map[string]interface{}{
			"taskPurpose":   "approval",
			"approvalMode":  "single",
			"allowDelegate": allowDelegate,
		}).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return instance.ID, task.ID
}

func TestBPMNApprovalBridge_NoInstanceFallsBack(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "bridge_no_instance")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "none")
	bridge := NewBPMNApprovalBridge(client, zaptest.NewLogger(t).Sugar())

	handled, err := bridge.CompleteBusinessApprovalTask(context.Background(), tenantID, actorID, "ticket", 999, "approve", "")
	require.NoError(t, err)
	assert.False(t, handled, "无关联流程实例时应回退旧逻辑")
}

func TestBPMNApprovalBridge_CompletesTaskAndRecordsDecision(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "bridge_complete")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "ok")
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "ok1", "ticket:123", actorID)
	bridge := NewBPMNApprovalBridge(client, zaptest.NewLogger(t).Sugar())

	handled, err := bridge.CompleteBusinessApprovalTask(context.Background(), tenantID, actorID, "ticket", 123, "approve", "同意")
	require.NoError(t, err)
	assert.True(t, handled, "存在待办流程任务时应桥接完成")

	// 任务应已完成
	task, err := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", task.Status)

	// 审批决策应已记录（含操作人与业务上下文）
	decisions, err := client.ProcessApprovalDecision.Query().All(context.Background())
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "approve", decisions[0].Action)
	assert.Equal(t, "approved", decisions[0].Decision)
	assert.Equal(t, "同意", decisions[0].Comment)
	assert.Equal(t, actorID, decisions[0].ActorID)
	assert.Equal(t, "ticket", decisions[0].BusinessType)
	assert.Equal(t, "123", decisions[0].BusinessID)
}

func TestBPMNApprovalBridge_UnauthorizedActorFailsClosed(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "bridge_unauthorized")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "unauth")
	// 任务指派给其他人（actorID+1000 不存在于 assignee/candidate）
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "unauth1", "ticket:123", actorID+1000)
	bridge := NewBPMNApprovalBridge(client, zaptest.NewLogger(t).Sugar())

	handled, err := bridge.CompleteBusinessApprovalTask(context.Background(), tenantID, actorID, "ticket", 123, "reject", "不同意")
	require.Error(t, err, "操作人不是流程任务审批人时必须失败，防止双轨分叉")
	assert.False(t, handled)

	// 任务保持待办状态
	task, err := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, err)
	assert.Equal(t, "assigned", task.Status)
}

func TestBPMNApprovalBridge_TenantIsolation(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "bridge_tenant")
	tenantA, actorA := setupBridgeTenantAndActor(t, client, "ta")
	tenantB, _ := setupBridgeTenantAndActor(t, client, "tb")
	// 流程实例属于租户 B
	createBridgeProcessFixture(t, client, tenantB, "tb1", "ticket:123", actorA)
	bridge := NewBPMNApprovalBridge(client, zaptest.NewLogger(t).Sugar())

	// 租户 A 审批同名业务键：不应命中租户 B 的实例，应回退旧逻辑
	handled, err := bridge.CompleteBusinessApprovalTask(context.Background(), tenantA, actorA, "ticket", 123, "approve", "")
	require.NoError(t, err)
	assert.False(t, handled, "跨租户不得命中其他租户的流程实例")

	// 租户 B 的任务未被动过
	count, err := client.ProcessTask.Query().Where(processtask.Status("assigned")).Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestBPMNApprovalBridge_DelegateNoInstanceFallsBack(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "bridge_delegate_none")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "dnone")
	bridge := NewBPMNApprovalBridge(client, zaptest.NewLogger(t).Sugar())

	handled, err := bridge.DelegateBusinessApprovalTask(context.Background(), tenantID, actorID, "ticket", 999, actorID+1)
	require.NoError(t, err)
	assert.False(t, handled, "无关联流程实例时应回退旧逻辑")
}

func TestBPMNApprovalBridge_DelegateReassignsTaskAndAllowsNewAssigneeToComplete(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "bridge_delegate_ok")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "dok")
	ctx := context.Background()
	delegatee, err := client.User.Create().
		SetUsername("bridge-delegatee-dok").
		SetEmail("bridge-delegatee-dok@example.com").
		SetName("Bridge Delegatee").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	_, taskID := createBridgeProcessFixture(t, client, tenantID, "dok1", "ticket:456", actorID)
	bridge := NewBPMNApprovalBridge(client, zaptest.NewLogger(t).Sugar())

	handled, err := bridge.DelegateBusinessApprovalTask(ctx, tenantID, actorID, "ticket", 456, delegatee.ID)
	require.NoError(t, err)
	assert.True(t, handled, "存在待办流程任务时应同步委派")

	// 任务应已重新指派给受托人，状态为 delegated
	task, err := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "delegated", task.Status)
	assert.Equal(t, strconv.Itoa(delegatee.ID), task.Assignee)

	// 原审批人不再有权完成任务
	_, err = bridge.CompleteBusinessApprovalTask(ctx, tenantID, actorID, "ticket", 456, "approve", "")
	require.Error(t, err, "委派后原审批人不应再能完成任务")

	// 受托人可通过桥接完成委派后的任务（delegated 状态仍被识别为待办）
	handled, err = bridge.CompleteBusinessApprovalTask(ctx, tenantID, delegatee.ID, "ticket", 456, "approve", "代批同意")
	require.NoError(t, err)
	assert.True(t, handled)

	task, err = client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", task.Status)
}

// 回归：BPMN 节点未声明 allowDelegate 时，桥接委派必须 fail closed，
// 并把 TaskVariables 原样保留，不得静默重新指派待办任务。
func TestBPMNApprovalBridge_DelegateRejectedWhenNodeDisallowsDelegation(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "bridge_delegate_denied")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "ddeny")
	ctx := context.Background()
	delegatee, err := client.User.Create().
		SetUsername("bridge-delegatee-ddeny").
		SetEmail("bridge-delegatee-ddeny@example.com").
		SetName("Bridge Delegatee Denied").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	_, taskID := createBridgeProcessFixtureWithDelegate(t, client, tenantID, "ddeny1", "ticket:457", actorID, false)
	bridge := NewBPMNApprovalBridge(client, zaptest.NewLogger(t).Sugar())

	_, err = bridge.DelegateBusinessApprovalTask(ctx, tenantID, actorID, "ticket", 457, delegatee.ID)
	require.Error(t, err, "节点不允许委托时不得委派")

	// 任务归属与状态未被修改
	task, err := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "assigned", task.Status)
	assert.Equal(t, strconv.Itoa(actorID), task.Assignee)
}

func TestBPMNApprovalBridge_DelegateUnauthorizedActorFailsClosed(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "bridge_delegate_unauth")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "dunauth")
	// 任务指派给其他人，actor 不是审批人/候选人
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "dunauth1", "ticket:456", actorID+1000)
	bridge := NewBPMNApprovalBridge(client, zaptest.NewLogger(t).Sugar())

	handled, err := bridge.DelegateBusinessApprovalTask(context.Background(), tenantID, actorID, "ticket", 456, actorID)
	require.Error(t, err, "非任务审批人发起委派必须失败，防止越权改派")
	assert.False(t, handled)

	// 任务保持原状态
	task, err := client.ProcessTask.Get(context.Background(), taskID)
	require.NoError(t, err)
	assert.Equal(t, "assigned", task.Status)
	assert.Equal(t, strconv.Itoa(actorID+1000), task.Assignee)
}
