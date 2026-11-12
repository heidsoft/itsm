package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== Incident 生命周期 ↔ BPMN 桥接集成测试 ====================

// createBridgeTestIncident 创建处于 new 状态的事件（可 acknowledge）。
func createBridgeTestIncident(t *testing.T, client *ent.Client, tenantID, reporterID int, suffix string) *ent.Incident {
	t.Helper()
	ctx := context.Background()
	inc, err := client.Incident.Create().
		SetTitle("Bridge Incident " + suffix).
		SetDescription("incident workflow bridge").
		SetStatus("new").
		SetPriority("high").
		SetSeverity("medium").
		SetIncidentNumber("BRG-INC-" + suffix).
		SetReporterID(reporterID).
		SetTenantID(tenantID).
		SetDetectedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	return inc
}

// TestAcknowledgeIncident_BridgesBPMNTask 事件确认端到端：
// 绑定的 BPMN 待办任务应随确认动作在同一事务内完成，事件状态推进，
// 且不产生「任务已完成、事件未确认」的分叉。
func TestAcknowledgeIncident_BridgesBPMNTask(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "incident_bridge_ack")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "inc-ack")
	svc := NewIncidentService(client, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	inc := createBridgeTestIncident(t, client, tenantID, actorID, "ack")
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "inc-ack1",
		fmt.Sprintf("incident:%d", inc.ID), actorID)

	require.NoError(t, svc.AcknowledgeIncident(ctx, inc.ID, actorID, tenantID))

	// 流程任务已完成（桥接生效）
	task, err := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", task.Status)

	// 事件状态已推进
	updated, err := client.Incident.Get(ctx, inc.ID)
	require.NoError(t, err)
	assert.Equal(t, "acknowledged", updated.Status)

	// 生命周期推进不是审批决策：不写决策行（决策表只记 approvalAction）
	decisions, err := client.ProcessApprovalDecision.Query().All(ctx)
	require.NoError(t, err)
	assert.Empty(t, decisions, "生命周期动作不得写成审批决策")

	// 任务完成审计带操作人痕迹（与 CompleteTask 同语义）
	audits, err := client.ProcessAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, audits, 1)
	assert.Equal(t, "completed", audits[0].Action)
	assert.Equal(t, actorID, audits[0].UserID)
}

// TestAcknowledgeIncident_BridgeFailureRollsBackBusinessWrite 原子性反证：
// 桥接推进失败（待办任务不属于操作人）时，事件确认必须随事务整体回滚，
// 不得留下「流程任务被他人完成/事件已确认」的分叉状态。
func TestAcknowledgeIncident_BridgeFailureRollsBackBusinessWrite(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "incident_bridge_rb")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "inc-rb")
	svc := NewIncidentService(client, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	inc := createBridgeTestIncident(t, client, tenantID, actorID, "rb")
	// 待办任务指派给另一人：操作人 acknowledge 时桥接必须 fail-closed
	otherUser, err := client.User.Create().
		SetUsername("incident-other-" + "rb").
		SetEmail("incident-other-rb@example.com").
		SetName("Other").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	_, taskID := createBridgeProcessFixture(t, client, tenantID, "inc-rb1",
		fmt.Sprintf("incident:%d", inc.ID), otherUser.ID)

	err = svc.AcknowledgeIncident(ctx, inc.ID, actorID, tenantID)
	require.Error(t, err, "越权推进流程待办必须 fail-closed")

	// 事件保持 new（业务写入随事务回滚）
	updated, err := client.Incident.Get(ctx, inc.ID)
	require.NoError(t, err)
	assert.Equal(t, "new", updated.Status, "桥接失败必须整单回滚，不得留下已确认的事件")

	// 流程任务保持待办
	task, err := client.ProcessTask.Get(ctx, taskID)
	require.NoError(t, err)
	assert.Contains(t, []string{
		"created", "assigned", "started", "delegated",
	}, task.Status, "任务不得被越权完成")
}

// TestResolveIncident_UnboundFallsBackToLifecycle 未绑定流程的事件：
// 解决动作照常推进（unbound 是存量/未配置租户的本意回退，不是错误）。
func TestResolveIncident_UnboundFallsBackToLifecycle(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "incident_bridge_unbound")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "inc-unb")
	svc := NewIncidentService(client, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	inc := createBridgeTestIncident(t, client, tenantID, actorID, "unb")
	require.NoError(t, svc.AcknowledgeIncident(ctx, inc.ID, actorID, tenantID))
	// 事件状态机要求 acknowledged → in_progress → resolved（生产中由受理/更新推进）
	_, err := client.Incident.UpdateOneID(inc.ID).SetStatus("in_progress").Save(ctx)
	require.NoError(t, err)
	require.NoError(t, svc.ResolveIncident(ctx, inc.ID, actorID, tenantID, "重启服务恢复", "配置错误"))

	updated, err := client.Incident.Get(ctx, inc.ID)
	require.NoError(t, err)
	assert.Equal(t, "resolved", updated.Status)
	require.NotNil(t, updated.ResolvedAt)

	// 无流程实例 ⇒ 零决策留痕
	decisionCount, err := client.ProcessApprovalDecision.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, decisionCount)
}

// TestCloseIncident_BoundWithoutActionableTaskDoesNotWrite 流程实例已结束
// （无 running 实例）时：关闭照常进行，bridge 为 no-op——不存在可推进的
// 待办，也没有可失败的冲突。
func TestCloseIncident_BoundWithoutActionableTaskDoesNotWrite(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "incident_bridge_done")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "inc-done")
	svc := NewIncidentService(client, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	inc := createBridgeTestIncident(t, client, tenantID, actorID, "done")
	// 已结束的实例（completed）仍算「绑定过」，但无待办可推进
	instanceID, taskID := createBridgeProcessFixture(t, client, tenantID, "inc-done1",
		fmt.Sprintf("incident:%d", inc.ID), actorID)
	_, err := client.ProcessTask.UpdateOneID(taskID).SetStatus("completed").SetCompletedTime(time.Now()).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessInstance.UpdateOneID(instanceID).SetStatus("completed").Save(ctx)
	require.NoError(t, err)

	require.NoError(t, svc.AcknowledgeIncident(ctx, inc.ID, actorID, tenantID))
	_, err = client.Incident.UpdateOneID(inc.ID).SetStatus("in_progress").Save(ctx)
	require.NoError(t, err)
	require.NoError(t, svc.ResolveIncident(ctx, inc.ID, actorID, tenantID, "已恢复", "根因"))
	require.NoError(t, svc.CloseIncident(ctx, inc.ID, actorID, tenantID, "关闭"))

	updated, err := client.Incident.Get(ctx, inc.ID)
	require.NoError(t, err)
	assert.Equal(t, "closed", updated.Status)
}
