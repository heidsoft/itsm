package change

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	_ "github.com/mattn/go-sqlite3"
)

// persistChangeInEntClient inserts a Change row in the ent client whose ID
// matches the mock repo's Change.ID. Service.TransitionStatus's CAS
// optimistic-locking path uses the ent client to look up the row by ID and
// to issue a conditional UPDATE on (id, tenant_id, status). Without a row
// in the ent client the CAS fails with ErrConcurrentModification, even
// though the mock repo has the correct change.
//
// Strategy: create the change via ent to obtain its auto-generated ID, then
// rewrite the mock Change's ID to match. This keeps the existing test
// fixture (createTestChange) and downstream assertions (repo.changes[c.ID])
// working without renumbering mock IDs.
func persistChangeInEntClient(t *testing.T, client *ent.Client, c *Change) {
	t.Helper()
	ctx := context.Background()
	created, err := client.Change.Create().
		SetChangeNumber(fmt.Sprintf("CHG-TEST-%d", c.ID)).
		SetTitle(c.Title).
		SetDescription(c.Description).
		SetType(c.Type).
		SetStatus(change.Status(c.Status)).
		SetPriority(c.Priority).
		SetImpactScope(c.ImpactScope).
		SetRiskLevel(c.RiskLevel).
		SetCreatedBy(c.CreatedBy).
		SetTenantID(c.TenantID).
		SetCreatedAt(c.CreatedAt).
		SetUpdatedAt(c.UpdatedAt).
		Save(ctx)
	require.NoError(t, err, "persist change into ent client for CAS path")
	c.ID = created.ID
}

// ==================== TransitionStatus ↔ BPMN 桥接集成测试 ====================

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
// nodeOverride 可传 [taskDefKey, taskName] 覆盖默认审批节点，用于构造
// 停在生命周期节点（如排期）上的流程实例。
func createChangeBridgeProcessFixture(t *testing.T, client *ent.Client, tenantID int, keySuffix, businessKey string, assigneeID int, nodeOverride ...string) (taskID int) {
	t.Helper()
	ctx := context.Background()

	taskDefKey, taskName := "Approval_1", "变更审批"
	if len(nodeOverride) >= 2 {
		taskDefKey, taskName = nodeOverride[0], nodeOverride[1]
	}

	defKey := "change_bridge_approval_" + keySuffix
	bpmnXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:itsm="https://github.com/heidsoft/itsm/schema/bpmn" id="Definitions_%s" targetNamespace="https://github.com/heidsoft/itsm"><bpmn:process id="%s" name="Change Bridge Approval %s" isExecutable="true"><bpmn:startEvent id="StartEvent_1"/><bpmn:userTask id="%s" name="%s" itsm:taskPurpose="approval" itsm:approvalMode="single" itsm:assignee="%d"/><bpmn:endEvent id="EndEvent_1"/><bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="%s"/><bpmn:sequenceFlow id="Flow_2" sourceRef="%s" targetRef="EndEvent_1"/></bpmn:process></bpmn:definitions>`, defKey, defKey, keySuffix, taskDefKey, taskName, assigneeID, taskDefKey, taskDefKey)

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
		SetTaskDefinitionKey(taskDefKey).
		SetTaskName(taskName).
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
	persistChangeInEntClient(t, entClient, c)
	_, err := repo.CreateApprovalRecord(ctx, &ApprovalRecord{
		ChangeID:   c.ID,
		ApproverID: actorID,
		Status:     "pending",
	})
	require.NoError(t, err)
	// dc5fbf97 契约：isApprover 校验数据源 = 审批链成员资格，必须 seed 链
	require.NoError(t, repo.ReplaceApprovalChain(ctx, c.ID, tenantID, []*ApprovalChain{{
		ID: 1, ChangeID: c.ID, TenantID: tenantID, Level: 1,
		ApproverID: actorID, Role: "approver", Status: "pending",
		IsRequired: true, ApprovalType: "serial", Threshold: 1,
		CreatedAt: time.Now(),
	}}))
	taskID := createChangeBridgeProcessFixture(t, entClient, tenantID, "e2e1",
		fmt.Sprintf("change:%d", c.ID), actorID)

	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "approved", "同意实施", "agent")
	require.NoError(t, err)
	assert.Equal(t, "approved", updated.Status)

	// BPMN 任务已完成
	task, err := entClient.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", task.Status)

	// 审批决定已由 BPMN bridge 写入 ProcessApprovalDecision（权威审计源，
	// dc5fbf97 起不再回写 legacy ApprovalRecord）

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
	persistChangeInEntClient(t, entClient, c)
	rec, err := repo.CreateApprovalRecord(ctx, &ApprovalRecord{
		ChangeID:   c.ID,
		ApproverID: actorID,
		Status:     "pending",
	})
	require.NoError(t, err)
	// dc5fbf97 契约：isApprover 校验数据源 = 审批链成员资格，必须 seed 链
	require.NoError(t, repo.ReplaceApprovalChain(ctx, c.ID, tenantID, []*ApprovalChain{{
		ID: 1, ChangeID: c.ID, TenantID: tenantID, Level: 1,
		ApproverID: actorID, Role: "approver", Status: "pending",
		IsRequired: true, ApprovalType: "serial", Threshold: 1,
		CreatedAt: time.Now(),
	}}))
	// 流程任务指派给其他人，业务审批人无权完成流程任务
	taskID := createChangeBridgeProcessFixture(t, entClient, tenantID, "fc1",
		fmt.Sprintf("change:%d", c.ID), actorID+1000)

	_, err = svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "rejected", "不同意", "agent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "同步流程审批任务失败")

	// 双轨状态均未被修改
	task, err := entClient.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "assigned", task.Status)
	assert.Equal(t, "pending", repo.changes[c.ID].Status)
	_ = rec // 业务侧记录保留快照；权威决定在 ProcessApprovalDecision（未写入）

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
	persistChangeInEntClient(t, entClient, c)
	_, err := repo.CreateApprovalRecord(ctx, &ApprovalRecord{
		ChangeID:   c.ID,
		ApproverID: actorID,
		Status:     "pending",
	})
	require.NoError(t, err)
	// dc5fbf97 契约：isApprover 校验数据源 = 审批链成员资格，必须 seed 链
	require.NoError(t, repo.ReplaceApprovalChain(ctx, c.ID, tenantID, []*ApprovalChain{{
		ID: 1, ChangeID: c.ID, TenantID: tenantID, Level: 1,
		ApproverID: actorID, Role: "approver", Status: "pending",
		IsRequired: true, ApprovalType: "serial", Threshold: 1,
		CreatedAt: time.Now(),
	}}))

	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "approved", "同意", "agent")
	require.NoError(t, err)
	assert.Equal(t, "approved", updated.Status)
}

// TestTransitionStatus_BridgeRollbackOnCASConflict 事务化回归：
// 业务状态 CAS 落败时，同一事务内已完成的 BPMN 任务必须一起回滚。
// 修复前桥接独立提交，CAS 失败后留下「流程已批、业务未批」的分叉
// （任务 completed、审批决定落库，而变更状态未推进）。
func TestTransitionStatus_BridgeRollbackOnCASConflict(t *testing.T) {
	entClient := newChangeBridgeEntClient(t, "change_bridge_rollback")
	tenantID, actorID := setupChangeBridgeActor(t, entClient, "rb")
	repo := newMockRepository()
	svc := NewService(repo, entClient, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	c := createTestChange(repo, tenantID, actorID)
	c.Status = "pending"
	persistChangeInEntClient(t, entClient, c)
	// 并发推进模拟：ent 行状态被抢先改为 scheduled，而 service 读到的 mock 快照仍是 pending。
	// 终态 CAS 按 StatusEQ("pending") 断言，必然 0 行命中。
	_, err := entClient.Change.UpdateOneID(c.ID).
		SetStatus(change.Status("scheduled")).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.ReplaceApprovalChain(ctx, c.ID, tenantID, []*ApprovalChain{{
		ID: 1, ChangeID: c.ID, TenantID: tenantID, Level: 1,
		ApproverID: actorID, Role: "approver", Status: "pending",
		IsRequired: true, ApprovalType: "serial", Threshold: 1,
		CreatedAt: time.Now(),
	}}))
	taskID := createChangeBridgeProcessFixture(t, entClient, tenantID, "rb1",
		fmt.Sprintf("change:%d", c.ID), actorID)

	_, err = svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "approved", "同意", "agent")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrConcurrentModification), "expected ErrConcurrentModification, got %v", err)

	// 桥接完成的 BPMN 任务被事务回滚，不得留下「流程已批、业务未批」。
	task, err := entClient.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "assigned", task.Status)
	decisionCount, err := entClient.ProcessApprovalDecision.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, decisionCount)
	// ent 行保持被并发推进的 scheduled，未被本次请求覆盖。
	row, err := entClient.Change.Get(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, "scheduled", string(row.Status))
}

// TestTransitionStatus_ApprovalNotMisfireOnLifecycleTask 节点断言回归：
// 流程停在生命周期节点（排期）上时，审批动作必须失败关闭——
// 不得把排期任务当审批完成，也不得在错误节点上写入审批决定。
func TestTransitionStatus_ApprovalNotMisfireOnLifecycleTask(t *testing.T) {
	entClient := newChangeBridgeEntClient(t, "change_bridge_lifecycle")
	tenantID, actorID := setupChangeBridgeActor(t, entClient, "lc")
	repo := newMockRepository()
	svc := NewService(repo, entClient, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	c := createTestChange(repo, tenantID, actorID)
	c.Status = "pending"
	persistChangeInEntClient(t, entClient, c)
	require.NoError(t, repo.ReplaceApprovalChain(ctx, c.ID, tenantID, []*ApprovalChain{{
		ID: 1, ChangeID: c.ID, TenantID: tenantID, Level: 1,
		ApproverID: actorID, Role: "approver", Status: "pending",
		IsRequired: true, ApprovalType: "serial", Threshold: 1,
		CreatedAt: time.Now(),
	}}))
	taskID := createChangeBridgeProcessFixture(t, entClient, tenantID, "lc1",
		fmt.Sprintf("change:%d", c.ID), actorID, "Activity_Schedule", "变更排期")

	_, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "approved", "同意", "agent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "没有可处理的审批待办")

	// 排期任务未被审批动作误完成，审批决定未落库。
	task, err := entClient.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "assigned", task.Status)
	decisionCount, err := entClient.ProcessApprovalDecision.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, decisionCount)
}
