package service

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/service/bpmn"
)

// 回归：change / release 老模板没有 itsm:taskPurpose，桥接层过去一律取最早待办
// （变更评估 / 技术评审），导致业务审批 fail-closed 恒 409；显式声明必须优先，
// 推断结果必须能把作业节点和真正的审批节点分开。
func TestInferredTaskPurpose(t *testing.T) {
	cases := []struct {
		name         string
		key          string
		taskName     string
		variables    map[string]interface{}
		want         string
	}{
		{"显式审批优先于名称", "Activity_Assessment", "变更评估", map[string]interface{}{"taskPurpose": "approval"}, "approval"},
		{"显式作业优先于名称", "Activity_Approval", "发布审批", map[string]interface{}{"taskPurpose": "work"}, "work"},
		{"空声明继续推断", "Activity_Assessment", "变更评估", map[string]interface{}{"taskPurpose": "  "}, "work"},
		{"老模板变更评估", "Activity_Assessment", "变更评估", map[string]interface{}{}, "work"},
		{"老模板技术评审", "Activity_TechReview", "技术评审", map[string]interface{}{}, "work"},
		{"老模板CAB审批", "Activity_CABApproval", "CAB审批", map[string]interface{}{}, "approval"},
		{"老模板发布审批", "Activity_Approval", "发布审批", map[string]interface{}{}, "approval"},
		{"无法推断保持兼容", "Activity_Unknown", "自定义节点", map[string]interface{}{}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			task := &ent.ProcessTask{TaskDefinitionKey: c.key, TaskName: c.taskName, TaskVariables: c.variables}
			assert.Equal(t, c.want, inferredTaskPurpose(task))
		})
	}
}

// 回归：审批任务的 assignee 曾回落为 triggered_by（创建人），而业务审批资格
// 来自另一套审批链；审批人和创建人两边都 403，审批永久卡死。受控桥接路径
// 只放行「回落来源」，用户主动认领 / 静态指派的任务仍严格校验。
func TestBusinessApprovalTrustSkipsOnlyTriggeredByFallback(t *testing.T) {
	engine, ctx, client, tenantID, instance := newAssignmentTestEnv(t, "approval-trust")
	creator := newAssignmentUser(t, client, tenantID, "trust-creator")
	approver := newAssignmentUser(t, client, tenantID, "trust-approver")

	newTask := func(t *testing.T, taskID, source string) *ent.ProcessTask {
		t.Helper()
		task, err := client.ProcessTask.Create().
			SetTaskID(taskID).
			SetTaskDefinitionKey("Activity_CABApproval").
			SetTaskName("CAB审批").
			SetTaskType("user_task").
			SetProcessDefinitionKey(instance.ProcessDefinitionKey).
			SetProcessInstanceID(instance.ID).
			SetAssignee(strconv.Itoa(creator.ID)).
			SetStatus("assigned").
			SetTaskVariables(map[string]interface{}{"assignmentSource": source}).
			SetTenantID(tenantID).
			Save(context.Background())
		require.NoError(t, err)
		return task
	}

	actorCtx := context.WithValue(ctx, bpmn.BPMNUserIDContextKey, approver.ID)

	fallbackTask := newTask(t, "TASK-TRUST-FALLBACK", "triggered_by")
	require.Error(t, engine.authorizeTaskActorWithClient(actorCtx, client, fallbackTask),
		"通用任务端点不带信任标记时，回落认领态仍必须挡住非认领人")
	require.NoError(t, engine.authorizeTaskActorWithClient(withBusinessApprovalTrust(actorCtx), client, fallbackTask),
		"业务审批桥接已校验资格，应忽略 triggered_by 回落认领态")

	realClaimTask := newTask(t, "TASK-TRUST-CLAIM", "bpmn_assignee")
	require.Error(t, engine.authorizeTaskActorWithClient(withBusinessApprovalTrust(actorCtx), client, realClaimTask),
		"真实认领/静态指派的任务不得被信任标记放行")
	assert.False(t, businessApprovalTrusted(ctx))
}

// 回归：findApprovalTaskWithClient 在老模板（无 taskPurpose）下必须选中审批节点，
// 并把作业节点留在 workTask 供可行动提示，而不是取最早的评估任务。
func TestFindApprovalTask_PrefersApprovalNodeOnLegacyTemplates(t *testing.T) {
	client := newApprovalBridgeTestClient(t, "bridge_legacy_purpose")
	tenantID, actorID := setupBridgeTenantAndActor(t, client, "legacypurpose")
	ctx := context.Background()
	_, assessmentTaskID := createBridgeProcessFixture(t, client, tenantID, "legacypurpose", "change:77", actorID)

	_, err := client.ProcessTask.UpdateOneID(assessmentTaskID).
		SetTaskDefinitionKey("Activity_Assessment").
		SetTaskName("变更评估").
		SetTaskVariables(map[string]interface{}{"assignmentSource": "triggered_by"}).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Query().Only(ctx)
	require.NoError(t, err)
	cabTask, err := client.ProcessTask.Create().
		SetTaskID("TASK-LEGACY-CAB").
		SetTaskDefinitionKey("Activity_CABApproval").
		SetTaskName("CAB审批").
		SetTaskType("user_task").
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).
		SetProcessInstanceID(instance.ID).
		SetStatus("created").
		SetTaskVariables(map[string]interface{}{"assignmentSource": "triggered_by"}).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	bridge := NewBPMNApprovalBridge(client, zaptest.NewLogger(t).Sugar())
	lookup, err := bridge.findApprovalTaskWithClient(ctx, client, tenantID, "change", 77)
	require.NoError(t, err)
	require.True(t, lookup.bound)
	require.NotNil(t, lookup.task)
	assert.Equal(t, cabTask.ID, lookup.task.ID, "应选真正的审批节点，而不是最早的评估任务")
	require.NotNil(t, lookup.workTask, "作业待办应保留下来供可行动提示")
	assert.Equal(t, assessmentTaskID, lookup.workTask.ID)
}
