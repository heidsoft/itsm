package change

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== 实施结果留痕（failed / rolled_back） ====================
//
// 缺陷背景（2026-10-22）：
//  1. 状态机一直允许 in_progress|failed → failed，但**没有任何 API 入口**写这个状态，
//     实施失败的变更只能被报成 completed 或直接 rolled_back——「实施成功/失败」统计
//     与 PIR 结论因此全是失真的；
//  2. 进入 failed / rolled_back 不需要任何原因：审计只能看到"状态从 in_progress
//     变成 rolled_back"，看不到为什么，无法判断影响是否已复原、要不要重新排期。
//
// 这批用例锁的就是这两条：入口存在、原因必填、原因落库。

func TestTransitionStatus_FailRequiresReason(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 301

	c := createTestChange(repo, tenantID, actorID)
	c.Status = "in_progress"

	_, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "failed", "   ", "agent")
	require.Error(t, err, "没有失败原因的 failed 必须被拒绝")
	assert.ErrorIs(t, err, ErrOutcomeNoteRequired)
	assert.Contains(t, err.Error(), "必须填写原因", "错误必须告诉调用方缺什么")

	// 状态不得被推进：拒绝发生在写库之前。
	got, getErr := repo.Get(ctx, c.ID, tenantID)
	require.NoError(t, getErr)
	assert.Equal(t, "in_progress", got.Status, "被拒绝的请求不能留下状态变更")
}

func TestTransitionStatus_RollbackRequiresReason(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 302

	c := createTestChange(repo, tenantID, actorID)
	c.Status = "in_progress"

	_, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "rolled_back", "", "agent")
	require.Error(t, err, "没有回滚原因的 rolled_back 必须被拒绝")
	assert.ErrorIs(t, err, ErrOutcomeNoteRequired)

	got, getErr := repo.Get(ctx, c.ID, tenantID)
	require.NoError(t, getErr)
	assert.Equal(t, "in_progress", got.Status)
}

// TestTransitionStatus_FailRecordsOutcomeNote 失败原因必须与状态推进一并落库：
// 状态变成 failed、outcome_note 是那句原因。
func TestTransitionStatus_FailRecordsOutcomeNote(t *testing.T) {
	entClient := newChangeBridgeEntClient(t, "change_fail_outcome")
	tenantID, actorID := setupChangeBridgeActor(t, entClient, "fo")
	repo := newMockRepository()
	svc := NewService(repo, entClient, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	c := createTestChange(repo, tenantID, actorID)
	c.Status = "in_progress"
	persistChangeInEntClient(t, entClient, c)

	const reason = "第二步脚本报错导致连接池耗尽，已按回滚计划恢复服务"
	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "failed", reason, "agent")
	require.NoError(t, err)
	assert.Equal(t, "failed", updated.Status)
	assert.Equal(t, reason, updated.OutcomeNote, "返回快照要带上结果说明，前端无需再拉一次")

	row, err := entClient.Change.Get(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, "failed", string(row.Status))
	assert.Equal(t, reason, row.OutcomeNote, "结果说明必须与状态同一条 UPDATE 落库")
}

// TestTransitionStatus_RollbackRecordsOutcomeNote 回滚同理。
func TestTransitionStatus_RollbackRecordsOutcomeNote(t *testing.T) {
	entClient := newChangeBridgeEntClient(t, "change_rollback_outcome")
	tenantID, actorID := setupChangeBridgeActor(t, entClient, "ro")
	repo := newMockRepository()
	svc := NewService(repo, entClient, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	c := createTestChange(repo, tenantID, actorID)
	c.Status = "in_progress"
	persistChangeInEntClient(t, entClient, c)

	const reason = "变更窗口内数据库锁等待超时，已回滚到上一个版本"
	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "rolled_back", reason, "agent")
	require.NoError(t, err)
	assert.Equal(t, "rolled_back", updated.Status)

	row, err := entClient.Change.Get(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, "rolled_back", string(row.Status))
	assert.Equal(t, reason, row.OutcomeNote)
}

// TestTransitionStatus_FailedCanRescheduleOrRollback 失败不是死胡同：
// 状态机允许 failed → scheduled（重新排期）与 failed → rolled_back（回滚）。
func TestTransitionStatus_FailedCanRescheduleOrRollback(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, actorID = 1, 303

	// failed → scheduled
	c := createTestChange(repo, tenantID, actorID)
	c.Status = "failed"
	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorID, "scheduled", "", "agent")
	require.NoError(t, err, "实施失败后应能重新排期")
	assert.Equal(t, "scheduled", updated.Status)

	// failed → rolled_back（仍需原因）
	c2 := createTestChange(repo, tenantID, actorID)
	c2.Status = "failed"
	_, err = svc.TransitionStatus(ctx, c2.ID, tenantID, actorID, "rolled_back", "", "agent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOutcomeNoteRequired)
}
