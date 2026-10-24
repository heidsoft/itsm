package problem

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Problem 生命周期 ↔ BPMN 桥接集成测试 ====================

// problemBridgeBPMNXML 单审批节点直通结构：Start → Approval_1 → End。
// 与 service 包桥接夹具同构（业务写入与流程推进同事务）。
const problemBridgeBPMNXML = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omid.com/schema/1.0/bpmn" id="Defs">
  <bpmn:process id="proc_problem_bridge" name="Problem Bridge" isExecutable="true">
    <bpmn:startEvent id="Start_1"/>
    <bpmn:userTask id="Approval_1" name="审批"/>
    <bpmn:endEvent id="End_1"/>
    <bpmn:sequenceFlow id="f1" sourceRef="Start_1" targetRef="Approval_1"/>
    <bpmn:sequenceFlow id="f2" sourceRef="Approval_1" targetRef="End_1"/>
  </bpmn:process>
</bpmn:definitions>`

// createProblemBridgeFixture 创建运行中流程实例 + 待办用户任务，
// businessKey 采用与 ProcessTriggerService 相同的 "{type}:{id}" 约定。
func createProblemBridgeFixture(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, suffix, businessKey string, assigneeID int) (instanceID, taskID int) {
	t.Helper()
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-" + suffix).
		SetDeploymentName("Deployment " + suffix).
		SetDeploymentTime(time.Now()).
		SetDeployedBy("test").
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey("problem_bridge_" + suffix).
		SetName("Problem Bridge " + suffix).
		SetVersion("1").
		SetIsLatest(true).
		SetBpmnXML([]byte(problemBridgeBPMNXML)).
		SetDeploymentID(deployment.ID).
		SetDeployedAt(time.Now()).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	bizType, bizID := businessKey, ""
	if parts := splitBusinessKey(businessKey); len(parts) == 2 {
		bizType, bizID = parts[0], parts[1]
	}
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-" + suffix).
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
		SetTaskID("TASK-" + suffix).
		SetTaskDefinitionKey("Approval_1").
		SetTaskName("审批").
		SetTaskType("user_task").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetAssignee(fmt.Sprintf("%d", assigneeID)).
		SetStatus("assigned").
		SetTaskVariables(map[string]interface{}{"taskPurpose": "approval", "approvalMode": "single"}).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return instance.ID, task.ID
}

func splitBusinessKey(key string) []string {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key}
}

// TestCloseProblem_BridgesBPMNTask 问题关闭端到端：
// 绑定的 BPMN 待办任务应随关闭动作在同一事务内完成，问题状态推进。
func TestCloseProblem_BridgesBPMNTask(t *testing.T) {
	client, svc, ctx := setupProblemHandlerTest(t)
	tenant := createProblemHandlerTenant(t, ctx, client, "bridge-close")
	user := createProblemHandlerUser(t, ctx, client, tenant.ID, "bridge-close")
	p := createProblemHandlerProblem(t, ctx, svc, tenant.ID, user.ID)

	// 关闭前置：investigating → resolved 的证据（根因+解决方案）
	_, err := svc.Update(ctx, tenant.ID, p.ID, &Problem{Status: "investigating", RootCause: "交换机端口故障", Resolution: "更换端口"}, user.ID, "agent")
	require.NoError(t, err)
	_, err = svc.Update(ctx, tenant.ID, p.ID, &Problem{Status: "resolved"}, user.ID, "agent")
	require.NoError(t, err)
	_, taskID := createProblemBridgeFixture(t, ctx, client, tenant.ID, "pb-close",
		fmt.Sprintf("problem:%d", p.ID), user.ID)

	_, err = svc.CloseProblem(ctx, tenant.ID, p.ID, user.ID, "admin", "agent")
	require.NoError(t, err)

	// 流程任务已完成（桥接生效）
	task, err := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", task.Status)

	// 问题已关闭
	updated, err := client.Problem.Get(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "closed", updated.Status)
	require.NotNil(t, updated.ClosedAt)

	// 任务完成审计带操作人痕迹
	audits, err := client.ProcessAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, audits, 1)
	assert.Equal(t, "completed", audits[0].Action)
	assert.Equal(t, user.ID, audits[0].UserID)
}

// TestCloseProblem_BridgeFailureRollsBackBusinessWrite 原子性反证：
// 桥接推进失败（待办任务不属于操作人）时，问题关闭必须随事务整体回滚，
// 不得留下「任务被他人完成/问题已关闭」的分叉状态。
func TestCloseProblem_BridgeFailureRollsBackBusinessWrite(t *testing.T) {
	client, svc, ctx := setupProblemHandlerTest(t)
	tenant := createProblemHandlerTenant(t, ctx, client, "bridge-rb")
	user := createProblemHandlerUser(t, ctx, client, tenant.ID, "bridge-rb")
	p := createProblemHandlerProblem(t, ctx, svc, tenant.ID, user.ID)

	_, err := svc.Update(ctx, tenant.ID, p.ID, &Problem{Status: "investigating", RootCause: "根因", Resolution: "方案"}, user.ID, "agent")
	require.NoError(t, err)
	_, err = svc.Update(ctx, tenant.ID, p.ID, &Problem{Status: "resolved"}, user.ID, "agent")
	require.NoError(t, err)

	// 待办任务指派给另一人：操作人关闭时桥接必须 fail-closed
	otherUser := createProblemHandlerUser(t, ctx, client, tenant.ID, "bridge-rb-other")
	_, taskID := createProblemBridgeFixture(t, ctx, client, tenant.ID, "pb-rb",
		fmt.Sprintf("problem:%d", p.ID), otherUser.ID)

	_, err = svc.CloseProblem(ctx, tenant.ID, p.ID, user.ID, "admin", "agent")
	require.Error(t, err, "越权推进流程待办必须 fail-closed")

	// 问题保持 resolved（关闭的业务写入随事务回滚）
	updated, err := client.Problem.Get(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "resolved", updated.Status, "桥接失败必须整单回滚，不得留下已关闭的问题")

	// 流程任务保持待办
	task, err := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Contains(t, []string{"created", "assigned", "started", "delegated"},
		task.Status, "任务不得被越权完成")
}

// TestInvestigateProblem_UnboundDoesNotBridge 未绑定流程的问题：
// 调查动作照常推进（unbound 是存量/未配置租户的本意回退，不是错误）。
func TestInvestigateProblem_UnboundDoesNotBridge(t *testing.T) {
	client, svc, ctx := setupProblemHandlerTest(t)
	tenant := createProblemHandlerTenant(t, ctx, client, "bridge-unbound")
	user := createProblemHandlerUser(t, ctx, client, tenant.ID, "bridge-unbound")
	p := createProblemHandlerProblem(t, ctx, svc, tenant.ID, user.ID)

	_, err := svc.InvestigateProblem(ctx, tenant.ID, p.ID, user.ID, "agent")
	require.NoError(t, err)

	updated, err := client.Problem.Get(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "investigating", updated.Status)

	instances, err := client.ProcessInstance.Query().All(ctx)
	require.NoError(t, err)
	assert.Empty(t, instances)
}

// TestUpdateProblem_FieldOnlyDoesNotBridge 字段级更新（标题/描述）不触发桥接：
// 只有前向生命周期状态跃迁才推进流程待办。
func TestUpdateProblem_FieldOnlyDoesNotBridge(t *testing.T) {
	client, svc, ctx := setupProblemHandlerTest(t)
	tenant := createProblemHandlerTenant(t, ctx, client, "bridge-field")
	user := createProblemHandlerUser(t, ctx, client, tenant.ID, "bridge-field")
	p := createProblemHandlerProblem(t, ctx, svc, tenant.ID, user.ID)

	_, taskID := createProblemBridgeFixture(t, ctx, client, tenant.ID, "pb-field",
		fmt.Sprintf("problem:%d", p.ID), user.ID)

	_, err := svc.Update(ctx, tenant.ID, p.ID, &Problem{Title: "重命名后的问题"}, user.ID, "agent")
	require.NoError(t, err)

	// 任务仍待办（字段更新不推进流程）
	task, err := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Contains(t, []string{"created", "assigned", "started", "delegated"}, task.Status)
}
