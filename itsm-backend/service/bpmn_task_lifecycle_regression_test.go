package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/processtask"
)

// 回归：审批拒绝未填意见应返回友好校验错误而非 panic（nil 断言）。
func TestRejectWithoutComment_ReturnsFriendlyError(t *testing.T) {
	client := bizTestClient(t, "reject_comment")
	ctx, tenantID := bizTenantCtx(t, client, "rjcm")

	const commentRequiredBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://test">
  <bpmn:process id="reject_comment_req" name="拒绝必填意见" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="approval" name="审批" commentRequiredOnReject="true"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="f1" sourceRef="start" targetRef="approval"/>
    <bpmn:sequenceFlow id="f2" sourceRef="approval" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>`

	bizDeploy(t, ctx, client, tenantID, "reject_comment_req", commentRequiredBPMN)
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)

	instance, err := engine.StartProcess(ctx, "reject_comment_req", "BK-RJCM", nil)
	require.NoError(t, err)
	task := bizFindTask(t, ctx, client, instance.ID, "approval")

	err = engine.CompleteTask(ctx, task.TaskID, map[string]interface{}{
		"approvalAction": "reject",
		// 故意不传 approvalComment：修复前 nil.(string) 断言 panic。
	})
	require.ErrorContains(t, err, "该审批节点要求拒绝时必须填写意见")
}

// 回归：认领必须是条件更新。读到"未认领"快照后、写入前任务被他人认领时，
// 不得覆盖他人认领（last-writer-wins）。
func TestClaimTask_StaleReadCannotStealClaim(t *testing.T) {
	client := bizTestClient(t, "claim_cas")
	ctx, tenantID := bizTenantCtx(t, client, "clm1")

	bizDeploy(t, ctx, client, tenantID, "simple_claim_cas", bizBPMNSimpleApproval)
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	instance, err := engine.StartProcess(ctx, "simple_claim_cas", "BK-CLM", nil)
	require.NoError(t, err)

	second, err := client.User.Create().
		SetUsername("claim_racer").SetEmail("claim_racer@test.com").SetName("Racer").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	// 模拟并发：DB 已被 second 认领，但 claimTask 的 findTask 读到的是旧快照（未认领）。
	task := bizFindTaskRaw(t, ctx, client, instance.ID, "manager_approval")
	_, err = client.ProcessTask.UpdateOne(task).SetAssignee(second.Username).Save(ctx)
	require.NoError(t, err)

	stale := *task
	stale.Assignee = ""
	taskSvc := engine.TaskService().(*bpmnTaskService)
	err = taskSvc.claimTask(ctx, second.ID, func(_ *ent.Client, _ int) (*ent.ProcessTask, error) {
		return &stale, nil
	})
	require.ErrorContains(t, err, "任务已被他人认领")

	after, err := client.ProcessTask.Query().
		Where(processtask.TaskID(task.TaskID), processtask.TenantID(tenantID)).
		First(ctx)
	require.NoError(t, err)
	require.Equal(t, second.Username, after.Assignee, "旧快照认领不得覆盖已在库的认领结果")
}

// 回归：已完成的任务不得被取消。
func TestCancelTask_RejectsCompletedTask(t *testing.T) {
	client := bizTestClient(t, "cancel_done")
	ctx, tenantID := bizTenantCtx(t, client, "cnl1")

	bizDeploy(t, ctx, client, tenantID, "simple_cancel_done", bizBPMNSimpleApproval)
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	instance, err := engine.StartProcess(ctx, "simple_cancel_done", "BK-CNL", nil)
	require.NoError(t, err)

	task := bizFindTask(t, ctx, client, instance.ID, "manager_approval")
	require.NoError(t, engine.CompleteTask(ctx, task.TaskID, map[string]interface{}{"approvalAction": "approve"}))

	taskSvc := engine.TaskService()
	err = taskSvc.CancelTask(ctx, task.TaskID, "迟到的取消")
	require.ErrorContains(t, err, "任务已结束，不能取消")

	after, err := client.ProcessTask.Query().
		Where(processtask.TaskID(task.TaskID), processtask.TenantID(tenantID)).
		First(ctx)
	require.NoError(t, err)
	require.Equal(t, "completed", after.Status, "取消不得覆盖已完成状态")
}
