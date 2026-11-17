package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ---------------------------------------------------------------------------
// 会签 finalizing 卡死恢复（P1 崩溃窗口收口）
//
// 场景：Vote 末票「CAS 抢占 finalizing → 取消其余子任务 → CompleteTask 推进」
// 中间进程崩溃，父任务永久停留 finalizing。ScanStuckFinalizingTasks 须在
// 宽限期后恢复：结论已定则系统推进，结论仍 pending 则回退 assigned。
// ---------------------------------------------------------------------------

// finalizingFixture 构造一个停留在 finalizing 的会签父任务 + 指定状态的子任务。
// subStates: 每个子任务的状态与 approved 变量（nil 表示 completed 但无变量，
// 用于验证「变量缺失按反对计」的谨慎口径）。
type finalizingFixture struct {
	client   *ent.Client
	scanner  *TimeoutScanner
	ctx      context.Context
	tenantID int
	parentID string // 业务 task_id（TASK-FIN-PARENT）
}

func setupFinalizingFixture(t *testing.T, subStates []subTaskSpec, threshold int, updatedAt time.Time) *finalizingFixture {
	t.Helper()
	client := newTimeoutScannerClient(t, t.Name())
	tenantID, _ := setupTimeoutScannerTenant(t, client, "finalizing")

	ctx := context.Background()
	defKey := "finalizing_test"
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-FIN").SetDeploymentName("Finalizing Deploy").
		SetIsActive(true).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey(defKey).SetName("finalizing 测试").SetVersion("1").
		SetIsLatest(true).SetIsActive(true).
		SetBpmnXML([]byte(countersignBPMN)).
		SetDeploymentID(deployment.ID).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-FIN-1").
		SetProcessDefinitionKey(defKey).SetProcessDefinitionID(def.ID).
		SetStatus("running").
		SetVariables(map[string]interface{}{"business_type": "ticket", "business_id": "1"}).
		SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)

	parentVars := map[string]interface{}{
		"approval_type": "parallel",
		"threshold":     threshold,
	}
	parent, err := client.ProcessTask.Create().
		SetTaskID("TASK-FIN-PARENT").
		SetTaskDefinitionKey("CounterSignTask").
		SetTaskName("会签父任务").
		SetProcessDefinitionKey(defKey).
		SetProcessInstanceID(instance.ID).
		SetStatus(common.ProcessTaskStatusFinalizing).
		SetAssignee("system").
		SetTaskVariables(parentVars).
		SetTenantID(tenantID).
		SetUpdatedAt(updatedAt).
		Save(ctx)
	require.NoError(t, err)
	_ = parent

	for i, spec := range subStates {
		builder := client.ProcessTask.Create().
			SetTaskID("TASK-FIN-PARENT_countersign_" + string(rune('0'+i))).
			SetTaskDefinitionKey("CounterSignTask").
			SetTaskName("会签子任务").
			SetProcessDefinitionKey(defKey).
			SetProcessInstanceID(instance.ID).
			SetStatus(spec.status).
			SetAssignee(spec.assignee).
			SetParentTaskID("TASK-FIN-PARENT").
			SetTenantID(tenantID)
		if spec.vars != nil {
			builder = builder.SetTaskVariables(spec.vars)
		}
		_, err := builder.Save(ctx)
		require.NoError(t, err)
	}

	return &finalizingFixture{
		client:   client,
		scanner:  NewTimeoutScanner(client, zaptest.NewLogger(t).Sugar()),
		ctx:      context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantID),
		tenantID: tenantID,
		parentID: "TASK-FIN-PARENT",
	}
}

type subTaskSpec struct {
	status   string
	assignee string
	vars     map[string]interface{}
}

func completedApprove() subTaskSpec {
	return subTaskSpec{status: common.ProcessTaskStatusCompleted, vars: map[string]interface{}{"approved": true}}
}

func completedReject() subTaskSpec {
	return subTaskSpec{status: common.ProcessTaskStatusCompleted, vars: map[string]interface{}{"approved": false}}
}

// TestScanStuckFinalizing_ApprovedParent_Advanced：阈值已凑齐（2/2 赞成）→
// 扫描器以系统上下文推进父任务到 completed，实例走到结束。
func TestScanStuckFinalizing_ApprovedParent_Advanced(t *testing.T) {
	f := setupFinalizingFixture(t,
		[]subTaskSpec{completedApprove(), completedApprove()},
		2, time.Now().Add(-10*time.Minute))

	recovered, err := f.scanner.ScanStuckFinalizingTasks(context.Background(), f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, recovered)

	parent, err := f.client.ProcessTask.Query().
		Where(processtask.TaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusCompleted, parent.Status,
		"已凑齐阈值的卡死父任务应被系统推进到 completed")
}

// TestScanStuckFinalizing_PendingParent_RevertedToAssigned：抢占发生在
// 阈值未凑齐时刻（如 1/3 赞成、还有 1 票待投）→ 不能替用户投票，
// 回退 assigned 让投票继续。
func TestScanStuckFinalizing_PendingParent_RevertedToAssigned(t *testing.T) {
	f := setupFinalizingFixture(t,
		[]subTaskSpec{
			completedApprove(),
			{status: common.ProcessTaskStatusAssigned, assignee: "approver1"},
			{status: common.ProcessTaskStatusAssigned, assignee: "approver2"},
		},
		2, time.Now().Add(-10*time.Minute))

	recovered, err := f.scanner.ScanStuckFinalizingTasks(context.Background(), f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, recovered)

	parent, err := f.client.ProcessTask.Query().
		Where(processtask.TaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusAssigned, parent.Status,
		"阈值未凑齐的误抢占应回退 assigned，而不是替用户投票")
}

// TestScanStuckFinalizing_GracePeriodNotElapsed：finalizing 未超过宽限期
// （可能是正在进行的正常推进）→ 不动。
func TestScanStuckFinalizing_GracePeriodNotElapsed(t *testing.T) {
	f := setupFinalizingFixture(t,
		[]subTaskSpec{completedApprove(), completedApprove()},
		2, time.Now()) // 刚更新，在 5 分钟宽限内

	recovered, err := f.scanner.ScanStuckFinalizingTasks(context.Background(), f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 0, recovered, "宽限期内不应恢复（正常推进可能正在进行）")

	parent, err := f.client.ProcessTask.Query().
		Where(processtask.TaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusFinalizing, parent.Status)
}

// TestScanStuckFinalizing_RejectedByMissingVars：completed 但变量缺失的子任务
// 按反对计（谨慎口径）：1 赞成 + 1 无凭据 + threshold=2 → 不可能凑齐 → rejected
// → 系统推进父任务（完成即视作拒绝终局，实例不再卡死）。
func TestScanStuckFinalizing_RejectedByMissingVars(t *testing.T) {
	f := setupFinalizingFixture(t,
		[]subTaskSpec{
			completedApprove(),
			{status: common.ProcessTaskStatusCompleted}, // 无 approved 变量
		},
		2, time.Now().Add(-10*time.Minute))

	recovered, err := f.scanner.ScanStuckFinalizingTasks(context.Background(), f.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, recovered)

	parent, err := f.client.ProcessTask.Query().
		Where(processtask.TaskID(f.parentID), processtask.TenantID(f.tenantID)).
		Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusCompleted, parent.Status,
		"凑不齐阈值的终局应被推进（拒绝路径），不能永久卡死")
}

// TestScanStuckFinalizing_InvalidTenant：无效租户直接报错。
func TestScanStuckFinalizing_InvalidTenant(t *testing.T) {
	f := setupFinalizingFixture(t,
		[]subTaskSpec{completedApprove()},
		1, time.Now().Add(-10*time.Minute))

	_, err := f.scanner.ScanStuckFinalizingTasks(context.Background(), 0)
	require.Error(t, err)
}
