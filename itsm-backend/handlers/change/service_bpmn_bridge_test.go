package change

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	_ "github.com/mattn/go-sqlite3"
)

// ==================== TransitionStatus ↔ BPMN 桥接集成测试（P0-1 阶段3） ====================

func newChangeBridgeEntClient(t *testing.T, dbName string) *ent.Client {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", dbName))
	t.Cleanup(func() { client.Close() })
	return client
}

func setupChangeBridgeActor(t *testing.T, client *ent.Client, code string) (tenantID, actorID int) {
	t.Helper()
	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Change Bridge Tenant " + code).
		SetCode("chg-bridge-" + code).
		SetDomain("chg-bridge-" + code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	actor, err := client.User.Create().
		SetUsername("chg-approver-" + code).
		SetEmail("chg-approver-" + code + "@example.com").
		SetName("Change Approver " + code).
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	return tenant.ID, actor.ID
}

// createChangeBridgeProcessFixture 创建 运行中流程实例 + 待办用户任务，
// businessKey 采用与 ProcessTriggerService 相同的 "change:{id}" 约定。
func createChangeBridgeProcessFixture(t *testing.T, client *ent.Client, tenantID int, keySuffix, businessKey string, assigneeID int) (taskID int) {
	t.Helper()
	ctx := context.Background()

	defKey := "change_bridge_approval_" + keySuffix
	bpmnXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:itsm="https://github.com/heidsoft/itsm/schema/bpmn" id="Definitions_%s" targetNamespace="https://github.com/heidsoft/itsm"><bpmn:process id="%s" name="Change Bridge Approval %s" isExecutable="true"><bpmn:startEvent id="StartEvent_1"/><bpmn:userTask id="Approval_1" name="变更审批" itsm:taskPurpose="approval" itsm:approvalMode="single" itsm:assignee="%d"/><bpmn:endEvent id="EndEvent_1"/><bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Approval_1"/><bpmn:sequenceFlow id="Flow_2" sourceRef="Approval_1" targetRef="EndEvent_1"/></bpmn:process></bpmn:definitions>`, defKey, defKey, keySuffix, assigneeID)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("CHG-DEP-" + keySuffix).
		SetDeploymentName("Change Deployment " + keySuffix).
		SetDeploymentTime(time.Now()).
		SetDeployedBy("test").
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey(defKey).
		SetName("Change Bridge Approval " + keySuffix).
		SetVersion("1").
		SetIsLatest(true).
		SetBpmnXML([]byte(bpmnXML)).
		SetDeploymentID(deployment.ID).
		SetDeployedAt(time.Now()).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("CHG-PI-" + keySuffix).
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetBusinessKey(businessKey).
		SetStatus("running").
		SetVariables(map[string]interface{}{
			"business_type": "change",
			"business_key":  businessKey,
		}).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	task, err := client.ProcessTask.Create().
		SetTaskID("CHG-TASK-" + keySuffix).
		SetTaskDefinitionKey("Approval_1").
		SetTaskName("变更审批").
		SetTaskType("user_task").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetAssignee(strconv.Itoa(assigneeID)).
		SetStatus("assigned").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return task.ID
}

// TestTransitionStatus_BridgesBPMNTask 变更审批端到端：
// 审批通过时应同时完成绑定的 BPMN 待办任务，并更新业务审批记录与变更状态。
func TestTransitionStatus_BridgesBPMNTask(t *testing.T) {
	entClient := newChangeBridgeEntClient(t, "change_bridge_e2e")
	tenantID, actorID := setupChangeBridgeActor(t, entClient, "e2e")
	repo := newMockRepository()
	svc := NewService(repo, entClient, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	c := createTestChange(repo, tenantID, actorID)
	c.Status = "pending"
	rec, err := repo.CreateApprovalRecord(ctx, &ApprovalRecord{
		ChangeID:   c.ID,
		ApproverID: actorID,
		Status:     "pending",
	})
	require.NoError(t, err)
	taskID := createChangeBridgeProcessFixture(t, entClient, tenantID, "e2e1",
		fmt.Sprintf("change:%d", c.ID), actorID)

	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "approved", "同意实施")
	require.NoError(t, err)
	assert.Equal(t, "approved", updated.Status)

	// BPMN 任务已完成
	task, err := entClient.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", task.Status)

	// 业务审批记录已更新
	assert.Equal(t, "approved", repo.approvals[rec.ID].Status)

	// 流程审批决策带正确的业务上下文
	decisions, err := entClient.ProcessApprovalDecision.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	assert.Equal(t, "approve", decisions[0].Action)
	assert.Equal(t, actorID, decisions[0].ActorID)
	assert.Equal(t, "change", decisions[0].BusinessType)
	assert.Equal(t, "同意实施", decisions[0].Comment)
}

// TestTransitionStatus_BridgeFailClosed 失败关闭回归：
// 存在待办流程任务但操作人不是流程审批人时，变更审批必须整体中止，双轨状态均不变。
func TestTransitionStatus_BridgeFailClosed(t *testing.T) {
	entClient := newChangeBridgeEntClient(t, "change_bridge_failclosed")
	tenantID, actorID := setupChangeBridgeActor(t, entClient, "fc")
	repo := newMockRepository()
	svc := NewService(repo, entClient, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	c := createTestChange(repo, tenantID, actorID)
	c.Status = "pending"
	rec, err := repo.CreateApprovalRecord(ctx, &ApprovalRecord{
		ChangeID:   c.ID,
		ApproverID: actorID,
		Status:     "pending",
	})
	require.NoError(t, err)
	// 流程任务指派给其他人，业务审批人无权完成流程任务
	taskID := createChangeBridgeProcessFixture(t, entClient, tenantID, "fc1",
		fmt.Sprintf("change:%d", c.ID), actorID+1000)

	_, err = svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "rejected", "不同意")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "同步流程审批任务失败")

	// 双轨状态均未被修改
	task, err := entClient.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "assigned", task.Status)
	assert.Equal(t, "pending", repo.changes[c.ID].Status)
	assert.Equal(t, "pending", repo.approvals[rec.ID].Status)

	decisionCount, err := entClient.ProcessApprovalDecision.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, decisionCount)
}

// TestTransitionStatus_NoBoundInstanceFallsBack 无绑定流程实例时回退纯业务审批。
func TestTransitionStatus_NoBoundInstanceFallsBack(t *testing.T) {
	entClient := newChangeBridgeEntClient(t, "change_bridge_fallback")
	tenantID, actorID := setupChangeBridgeActor(t, entClient, "fb")
	repo := newMockRepository()
	svc := NewService(repo, entClient, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	c := createTestChange(repo, tenantID, actorID)
	c.Status = "pending"
	_, err := repo.CreateApprovalRecord(ctx, &ApprovalRecord{
		ChangeID:   c.ID,
		ApproverID: actorID,
		Status:     "pending",
	})
	require.NoError(t, err)

	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "approved", "同意")
	require.NoError(t, err)
	assert.Equal(t, "approved", updated.Status)
}
