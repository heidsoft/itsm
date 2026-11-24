package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/processtask"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ---------------------------------------------------------------------------
// codex 审计 P2-01：会签恢复三个崩溃/串行窗口
//   1. 投票 commit 后、抢占 finalizing 前崩溃 → 父任务仍 assigned，无接续
//   2. 恢复推进时未取消剩余子任务 → 留下兄弟待办
//   3. 串行模式第一人反对后结论仍 pending，但后两人保持 created 无法投票
// ---------------------------------------------------------------------------

type unclaimedFixture struct {
	client   *ent.Client
	scanner  *TimeoutScanner
	tenantID int
	parentID string
}

// setupUnclaimedFixture 构造一个指定状态的会签父任务 + 子任务。
// updatedAt 可人为调老，模拟「崩溃后已经过了宽限期」。
func setupUnclaimedFixture(t *testing.T, parentStatus string, subs []subTaskSpec, threshold int, updatedAt time.Time) *unclaimedFixture {
	t.Helper()
	client := newTimeoutScannerClient(t, t.Name())
	tenantID, _ := setupTimeoutScannerTenant(t, client, "unclaimed")
	ctx := context.Background()

	defKey := "unclaimed_test"
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-UNCLAIMED").SetDeploymentName("Unclaimed Deploy").
		SetIsActive(true).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey(defKey).SetName("未认领会签测试").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).
		SetBpmnXML([]byte(countersignBPMN)).
		SetDeploymentID(dep.ID).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	inst, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-UNCLAIMED-1").
		SetProcessDefinitionKey(defKey).SetProcessDefinitionID(def.ID).
		SetStatus("running").
		SetVariables(map[string]interface{}{"business_type": "ticket", "business_id": "1"}).
		SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)

	_, err = client.ProcessTask.Create().
		SetTaskID("TASK-UNCLAIMED-PARENT").
		SetTaskDefinitionKey("CounterSignTask").
		SetTaskName("会签父任务").
		SetProcessDefinitionKey(defKey).
		SetProcessInstanceID(inst.ID).
		SetStatus(parentStatus).
		SetAssignee("system").
		SetTaskVariables(map[string]interface{}{
			"approval_type": "parallel",
			"threshold":     threshold,
		}).
		SetTenantID(tenantID).
		SetUpdatedAt(updatedAt).
		Save(ctx)
	require.NoError(t, err)

	for i, spec := range subs {
		builder := client.ProcessTask.Create().
			SetTaskID(fmt.Sprintf("TASK-UNCLAIMED-PARENT_countersign_%d", i)).
			SetTaskDefinitionKey("CounterSignTask").
			SetTaskName("会签子任务").
			SetProcessDefinitionKey(defKey).
			SetProcessInstanceID(inst.ID).
			SetStatus(spec.status).
			SetAssignee(spec.assignee).
			SetParentTaskID("TASK-UNCLAIMED-PARENT").
			SetTenantID(tenantID)
		if spec.vars != nil {
			builder = builder.SetTaskVariables(spec.vars)
		}
		_, err := builder.Save(ctx)
		require.NoError(t, err)
	}

	return &unclaimedFixture{
		client:   client,
		scanner:  NewTimeoutScanner(client, zaptest.NewLogger(t).Sugar()),
		tenantID: tenantID,
		parentID: "TASK-UNCLAIMED-PARENT",
	}
}

func (f *unclaimedFixture) parent(t *testing.T) *ent.ProcessTask {
	t.Helper()
	p, err := f.client.ProcessTask.Query().
		Where(processtask.TaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Only(context.Background())
	require.NoError(t, err)
	return p
}

func (f *unclaimedFixture) subs(t *testing.T) []*ent.ProcessTask {
	t.Helper()
	s, err := f.client.ProcessTask.Query().
		Where(processtask.ParentTaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Order(processtask.ByID()).
		All(context.Background())
	require.NoError(t, err)
	return s
}

func createdSub() subTaskSpec {
	return subTaskSpec{status: common.ProcessTaskStatusCreated, assignee: "approver"}
}

// TestScanUnclaimedCountersign_ApprovedWindow_AdvancedAndCancelsSiblings
// 崩溃窗口核心场景：父任务仍 assigned、结论已 approved、还有 created 余票。
// 恢复必须同时完成「推进父任务」与「取消余票」，不能只做一半。
func TestScanUnclaimedCountersign_ApprovedWindow_AdvancedAndCancelsSiblings(t *testing.T) {
	f := setupUnclaimedFixture(t, common.ProcessTaskStatusAssigned,
		[]subTaskSpec{completedApprove(), completedApprove(), createdSub()},
		2, time.Now().Add(-10*time.Minute))

	recovered, err := f.scanner.ScanUnclaimedCountersignParents(context.Background(), f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, recovered, "结论已定的未认领父任务应被接续")

	assert.Equal(t, common.ProcessTaskStatusCompleted, f.parent(t).Status,
		"父任务应被系统推进到 completed")
	subs := f.subs(t)
	require.Len(t, subs, 3)
	assert.Equal(t, common.ProcessTaskStatusCancelled, subs[2].Status,
		"恢复推进时剩余兄弟待办必须被取消（旧实现会留下可投票的孤儿任务）")
}

// TestScanUnclaimedCountersign_PendingParent_LeftUntouched
// 会签正常进行中（结论 pending）绝不能被抢跑结束，也不能替用户投票。
func TestScanUnclaimedCountersign_PendingParent_LeftUntouched(t *testing.T) {
	f := setupUnclaimedFixture(t, common.ProcessTaskStatusAssigned,
		[]subTaskSpec{completedApprove(), createdSub(), createdSub()},
		2, time.Now().Add(-10*time.Minute))

	recovered, err := f.scanner.ScanUnclaimedCountersignParents(context.Background(), f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 0, recovered)
	assert.Equal(t, common.ProcessTaskStatusAssigned, f.parent(t).Status)
	for _, s := range f.subs(t) {
		if s.Status != common.ProcessTaskStatusCreated {
			assert.Equal(t, common.ProcessTaskStatusCompleted, s.Status)
		}
	}
}

// TestScanUnclaimedCountersign_NonCountersignTask_Skipped
// 普通用户任务（无子任务）不应被会签恢复逻辑误处理。
func TestScanUnclaimedCountersign_NonCountersignTask_Skipped(t *testing.T) {
	f := setupUnclaimedFixture(t, common.ProcessTaskStatusAssigned,
		nil, 1, time.Now().Add(-10*time.Minute))

	recovered, err := f.scanner.ScanUnclaimedCountersignParents(context.Background(), f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 0, recovered)
	assert.Equal(t, common.ProcessTaskStatusAssigned, f.parent(t).Status)
}

// TestScanUnclaimedCountersign_FinalizingParent_Skipped
// finalizing 由 ScanStuckFinalizingTasks 负责，本方法不得重复认领。
func TestScanUnclaimedCountersign_FinalizingParent_Skipped(t *testing.T) {
	f := setupUnclaimedFixture(t, common.ProcessTaskStatusFinalizing,
		[]subTaskSpec{completedApprove(), completedApprove()},
		2, time.Now().Add(-10*time.Minute))

	recovered, err := f.scanner.ScanUnclaimedCountersignParents(context.Background(), f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 0, recovered)
	assert.Equal(t, common.ProcessTaskStatusFinalizing, f.parent(t).Status)
}

// TestScanUnclaimedCountersign_GracePeriodNotElapsed
// 刚提交的投票还在抢占 finalizing 的路上，宽限内不得抢跑。
func TestScanUnclaimedCountersign_GracePeriodNotElapsed(t *testing.T) {
	f := setupUnclaimedFixture(t, common.ProcessTaskStatusAssigned,
		[]subTaskSpec{completedApprove(), completedApprove()},
		2, time.Now())

	recovered, err := f.scanner.ScanUnclaimedCountersignParents(context.Background(), f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 0, recovered)
	assert.Equal(t, common.ProcessTaskStatusAssigned, f.parent(t).Status)
}

// TestScanUnclaimedCountersign_InvalidTenant 参数校验。
func TestScanUnclaimedCountersign_InvalidTenant(t *testing.T) {
	f := setupUnclaimedFixture(t, common.ProcessTaskStatusAssigned,
		[]subTaskSpec{completedApprove(), completedApprove()},
		2, time.Now().Add(-10*time.Minute))
	_, err := f.scanner.ScanUnclaimedCountersignParents(context.Background(), 0)
	require.Error(t, err)
}

// TestScanStuckFinalizing_CancelsRemainingSiblings
// finalizing 恢复路径也必须与 Vote 末票一致地取消余票（两条恢复路径共用
// advanceCountersignParent）。旧实现只 CompleteTask，留下兄弟待办。
func TestScanStuckFinalizing_CancelsRemainingSiblings(t *testing.T) {
	f := setupFinalizingFixture(t,
		[]subTaskSpec{completedApprove(), completedApprove(), createdSub()},
		2, time.Now().Add(-10*time.Minute))

	recovered, err := f.scanner.ScanStuckFinalizingTasks(context.Background(), f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, recovered)
	parent, err := f.client.ProcessTask.Query().
		Where(processtask.TaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusCompleted, parent.Status)
	subs, err := f.client.ProcessTask.Query().
		Where(processtask.ParentTaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Order(processtask.ByID()).All(context.Background())
	require.NoError(t, err)
	require.Len(t, subs, 3)
	assert.Equal(t, common.ProcessTaskStatusCancelled, subs[2].Status,
		"finalizing 恢复也必须取消剩余兄弟任务")
}

// TestVote_SerialReject_ActivatesNextApprover
// 三人阈值二，第一人反对后结论仍 pending（0 赞成 + 2 待投 >= 2）。
// 旧实现只在 req.Approved 时激活下一位，后两人永远停在 created 无法投票。
func TestVote_SerialReject_ActivatesNextApprover(t *testing.T) {
	f := setupCounterSignFixture(t, "serial", 3, 2)
	ctx0 := userContext(f.tenantID, f.users["approver0"].ID)

	err := f.engine.TaskService().Vote(ctx0, "TASK-CS-PARENT_countersign_0",
		&VoteRequest{Approved: false, Comment: "反对"})
	require.NoError(t, err)

	parent, err := f.client.ProcessTask.Query().
		Where(processtask.TaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Only(f.ctx)
	require.NoError(t, err)
	assert.Equal(t, "assigned", parent.Status, "结论仍 pending 时父任务不能被提前结束")

	subs, err := f.client.ProcessTask.Query().
		Where(processtask.ParentTaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Order(processtask.ByID()).All(f.ctx)
	require.NoError(t, err)
	require.Len(t, subs, 3)
	assert.Equal(t, common.ProcessTaskStatusCompleted, subs[0].Status)
	assert.Equal(t, common.ProcessTaskStatusAssigned, subs[1].Status,
		"反对票后结论仍 pending，必须激活下一位审批人")
	assert.Equal(t, common.ProcessTaskStatusCreated, subs[2].Status,
		"再下一位仍应等待，串行语义一次只激活一人")
}
