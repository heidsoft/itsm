package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/processapprovaldecision"
	"itsm-backend/ent/processtask"
)

// ============================================================================
// 超时自动决策（codex 审计 P2-03）：
//   1. 决策键必须是引擎读取的 approvalAction/approvalResult/approvalComment；
//   2. 自动决策必须以受控系统身份推进（否则被任务归属校验拒绝）；
//   3. 决策必须落正式审计行，且不伪造个人签名；
//   4. 网关必须真的按 approvalResult 分流（这是「键名对不对」的行为证据）；
//   5. 扫描与 timer 共用完整活跃状态表（含 delegated）。
// ============================================================================

// testBPMNTimeoutAutoDecision 超时自动决策流程：审批任务到期后按 timeoutAction
// 自动通过/拒绝，随后由 exclusiveGateway 按 approvalResult 分流到不同后续任务。
// 分支出不去 = 决策变量没写对，这正是本组用例要守的行为。
const testBPMNTimeoutAutoDecision = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="timeout_auto_proc" name="超时自动决策" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="approval" name="审批任务" dueDate="2000-01-01" timeoutAction="%s"/>
    <bpmn:exclusiveGateway id="decision" name="审批结果分流"/>
    <bpmn:userTask id="approved_path" name="通过后续"/>
    <bpmn:userTask id="rejected_path" name="拒绝后续"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="approval"/>
    <bpmn:sequenceFlow id="f2" sourceRef="approval" targetRef="decision"/>
    <bpmn:sequenceFlow id="f3" sourceRef="decision" targetRef="approved_path">
      <bpmn:conditionExpression><![CDATA[${variables['approvalResult'] == 'approved'}]]></bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f4" sourceRef="decision" targetRef="rejected_path">
      <bpmn:conditionExpression><![CDATA[${variables['approvalResult'] != 'approved'}]]></bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f5" sourceRef="approved_path" targetRef="end"/>
    <bpmn:sequenceFlow id="f6" sourceRef="rejected_path" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

// testBPMNTimeoutAutoDecisionFuture 同一结构但截止时间在未来，用于验证 task_due
// timer 回调路径（与恢复扫描路径必须行为一致）。
const testBPMNTimeoutAutoDecisionFuture = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="timeout_auto_future_proc" name="超时自动决策-定时器路径" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="approval" name="审批任务" dueDate="2099-01-01" timeoutAction="auto_approve"/>
    <bpmn:exclusiveGateway id="decision" name="审批结果分流"/>
    <bpmn:userTask id="approved_path" name="通过后续"/>
    <bpmn:userTask id="rejected_path" name="拒绝后续"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="approval"/>
    <bpmn:sequenceFlow id="f2" sourceRef="approval" targetRef="decision"/>
    <bpmn:sequenceFlow id="f3" sourceRef="decision" targetRef="approved_path">
      <bpmn:conditionExpression><![CDATA[${variables['approvalResult'] == 'approved'}]]></bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f4" sourceRef="decision" targetRef="rejected_path">
      <bpmn:conditionExpression><![CDATA[${variables['approvalResult'] != 'approved'}]]></bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="f5" sourceRef="approved_path" targetRef="end"/>
    <bpmn:sequenceFlow id="f6" sourceRef="rejected_path" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

// assertAutoDecisionOutcome 断言一次超时自动决策的完整可观测结果。
func assertAutoDecisionOutcome(t *testing.T, client *ent.Client, ctx context.Context, instance *ent.ProcessInstance, approvalTask *ent.ProcessTask, wantAction, wantResult, wantBranch string) {
	t.Helper()

	after, err := client.ProcessTask.Get(ctx, approvalTask.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", after.Status, "自动决策必须完成审批任务")
	assert.Equal(t, wantAction, after.TaskVariables["approvalAction"], "任务变量必须写引擎读取的 approvalAction")
	assert.Equal(t, wantResult, after.TaskVariables["approvalResult"], "任务变量必须写引擎读取的 approvalResult")

	running, err := client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, wantResult, running.Variables["approvalResult"], "流程实例变量必须带上审批结果，供条件网关求值")

	// 网关按 approvalResult 分流：进了错误分支说明决策变量没写对。
	branch, err := client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(instance.ID), processtask.TaskDefinitionKey(wantBranch)).
		Only(ctx)
	require.NoError(t, err, "自动决策必须进入 %s 分支", wantBranch)
	assert.Equal(t, "created", branch.Status)

	other := "rejected_path"
	if wantBranch == "rejected_path" {
		other = "approved_path"
	}
	_, err = client.ProcessTask.Query().
		Where(processtask.ProcessInstanceID(instance.ID), processtask.TaskDefinitionKey(other)).
		Only(ctx)
	assert.True(t, ent.IsNotFound(err), "自动决策不得进入 %s 分支", other)

	// 正式决策审计：系统身份，绝不伪造个人签名。
	decisions, err := client.ProcessApprovalDecision.Query().
		Where(processapprovaldecision.ProcessTaskID(approvalTask.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, decisions, 1, "自动决策必须写入且只写入一条决策审计")
	assert.Equal(t, "system_decision", decisions[0].Action, "系统推进必须标记为 system_decision")
	assert.Equal(t, wantResult, decisions[0].Decision)
	assert.Equal(t, 0, decisions[0].ActorID, "系统自动决策不得借用真人身份")
	assert.Equal(t, "system", decisions[0].ActorName)
	assert.NotEmpty(t, decisions[0].Comment, "决策审计必须留下自动决策原因")
}

// TestTimeoutScanner_AutoApprove_RoutesApprovedBranch 恢复扫描路径的超时自动通过：
// 决策键正确 + 系统身份放行 + 走通过分支 + 决策审计落库。
func TestTimeoutScanner_AutoApprove_RoutesApprovedBranch(t *testing.T) {
	client := newE2ETestClient(t, "timeout_auto_approve")
	logger := zaptest.NewLogger(t).Sugar()

	instance, task, _, ctx, _ := startTaskDueProcess(t, client,
		formatTimeoutAutoBPMN(testBPMNTimeoutAutoDecision, "auto_approve"),
		"timeout_auto_proc", "BK-TMO-APPROVE")

	scanner := NewTimeoutScanner(client, logger)
	processed, err := scanner.ScanOverdueTasks(ctx, task.TenantID)
	require.NoError(t, err)
	require.Equal(t, 1, processed, "超时任务必须被扫描器处理")

	assertAutoDecisionOutcome(t, client, ctx, instance, task, "approve", "approved", "approved_path")
}

// TestTimeoutScanner_AutoReject_RoutesRejectedBranch 超时自动拒绝：同上，
// 但必须进入拒绝分支并落 rejected 决策。
func TestTimeoutScanner_AutoReject_RoutesRejectedBranch(t *testing.T) {
	client := newE2ETestClient(t, "timeout_auto_reject")
	logger := zaptest.NewLogger(t).Sugar()

	instance, task, _, ctx, _ := startTaskDueProcess(t, client,
		formatTimeoutAutoBPMN(testBPMNTimeoutAutoDecision, "auto_reject"),
		"timeout_auto_proc", "BK-TMO-REJECT")

	scanner := NewTimeoutScanner(client, logger)
	processed, err := scanner.ScanOverdueTasks(ctx, task.TenantID)
	require.NoError(t, err)
	require.Equal(t, 1, processed, "超时任务必须被扫描器处理")

	assertAutoDecisionOutcome(t, client, ctx, instance, task, "reject", "rejected", "rejected_path")
}

// TestTimeoutScanner_ScansDelegatedTasks delegated 是活跃状态之一：
// 已转派的任务同样要被超时扫描选中，否则它既无 timer 兜底也无扫描兜底。
func TestTimeoutScanner_ScansDelegatedTasks(t *testing.T) {
	client := newE2ETestClient(t, "timeout_delegated")
	logger := zaptest.NewLogger(t).Sugar()

	_, task, _, ctx, _ := startTaskDueProcess(t, client, testBPMNTimeoutNotifyPast,
		"timeout_notify_proc", "BK-TMO-DELEGATED")

	_, err := client.ProcessTask.UpdateOneID(task.ID).
		SetStatus("delegated").
		Save(ctx)
	require.NoError(t, err)

	scanner := NewTimeoutScanner(client, logger)
	processed, err := scanner.ScanOverdueTasks(ctx, task.TenantID)
	require.NoError(t, err)
	require.Equal(t, 1, processed, "delegated 任务必须在活跃状态表内")

	after, err := client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "timeout", after.Status)
}

// TestTaskDueTimer_AutoApprove_MatchesScannerPath timer 回调路径必须与恢复扫描
// 路径行为一致：同样带系统身份、同样写对决策键、同样走通过分支。
func TestTaskDueTimer_AutoApprove_MatchesScannerPath(t *testing.T) {
	client := newE2ETestClient(t, "timeout_timer_auto")
	logger := zaptest.NewLogger(t).Sugar()

	instance, task, timerStore, ctx, engine := startTaskDueProcess(t, client,
		testBPMNTimeoutAutoDecisionFuture, "timeout_auto_future_proc", "BK-TMO-TIMER")

	timers, _, err := timerStore.List(ctx, TimerListFilter{
		TenantID:  task.TenantID,
		TimerType: string(TimerTypeTaskDue),
		Status:    string(TimerStatusPending),
	})
	require.NoError(t, err)
	require.Len(t, timers, 1, "未来截止时间必须注册 task_due 定时器")

	_, err = timerStore.CASFire(ctx, timers[0].TimerID, time.Now())
	require.NoError(t, err)

	handler := NewTimerEventHandler(engine, logger)
	require.NoError(t, handler.HandleTimerFire(ctx, &TimerRecord{
		TimerID:              timers[0].TimerID,
		TimerType:            string(TimerTypeTaskDue),
		ProcessDefinitionKey: "timeout_auto_future_proc",
		ProcessInstanceID:    instance.ID,
		ActivityID:           task.TaskID,
		FireAt:               timers[0].FireAt,
		TenantID:             task.TenantID,
	}), "timer 回调路径的超时自动通过必须成功（缺系统身份时这里会 403）")

	assertAutoDecisionOutcome(t, client, ctx, instance, task, "approve", "approved", "approved_path")
}

// formatTimeoutAutoBPMN 用 timeoutAction 填充测试流程模板。
func formatTimeoutAutoBPMN(template, action string) string {
	return strings.Replace(template, "%s", action, 1)
}
