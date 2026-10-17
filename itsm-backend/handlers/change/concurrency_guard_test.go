package change

// P1 并发守卫回归：
//   1. 审批 quorum 汇总曾是「读状态 → 校验 → 无条件全列 repo.Update」——
//      quorum 读到的 status 在并发窗口里失效时，陈旧校验结论（approved）
//      会直接覆盖推进后的真实状态（cancelled），绕过状态机。
//   2. TransitionStatus 非终态分支曾在 CAS 推进状态后，用陈旧快照再调一次
//      repo.Update 做全列覆盖（title/description/affected_cis 全量回写），
//      并发提交的其他字段编辑会被静默卡掉，CAS 等于白做。
// 两个测试都必须在修复前红、修复后绿。

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// staleViewChangeRepo 包装真实仓储：Get 固定返回调用方捕获到的陈旧快照
// （模拟 quorum 求值时读到了并发窗口前的状态），其余方法透传真实仓储，
// 使 CAS 语义作用于真实存储。
type staleViewChangeRepo struct {
	Repository
	view *Change
}

func (r *staleViewChangeRepo) Get(ctx context.Context, id, tenantID int) (*Change, error) {
	return r.view, nil
}

// TestQuorumCASMiss_SkipsClobber quorum 的陈旧校验结论不得覆盖并发后的真实状态。
func TestQuorumCASMiss_SkipsClobber(t *testing.T) {
	svc, client, db, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	approver := mkChangeUser(t, client, tenantID, "manager")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)

	// 审批链：level1 required threshold1（serial）。
	_, err := db.ExecContext(ctx, `
		INSERT INTO change_approval_chains
			(change_id, tenant_id, level, approver_id, role, status, is_required, approval_type, threshold, created_at)
		VALUES (?, ?, 1, ?, 'manager', 'pending', 1, 'serial', 1, CURRENT_TIMESTAMP)
	`, changeID, tenantID, approver)
	require.NoError(t, err)

	// 变更处于 pending，quorum 满足（1 条 approved 决定）。
	_, err = db.ExecContext(ctx, `UPDATE changes SET status = 'pending' WHERE id = ? AND tenant_id = ?`, changeID, tenantID)
	require.NoError(t, err)

	now := time.Now()
	_, err = client.ProcessApprovalDecision.Create().
		SetProcessInstanceID(1).
		SetProcessTaskID(1).
		SetProcessInstanceKey("change:normal:1").
		SetTaskID("task_approve_stale").
		SetProcessDefinitionKey("change_normal_flow").
		SetNodeKey("Activity_Approve").
		SetBusinessType("change").
		SetBusinessID(strconv.Itoa(changeID)).
		SetActorID(approver).
		SetActorName("Manager").
		SetAction("approve").
		SetDecision("approved").
		SetComment("同意").
		SetTenantID(tenantID).
		SetCreatedAt(now).
		Save(ctx)
	require.NoError(t, err)

	// 捕获 quorum 将要看到的快照，然后模拟并发窗口：真实状态已被推进为
	// cancelled（例如变更发起人同步取消）。quorum 的后续校验仍锚定陈旧
	// pending → 按旧实现会 approved 全列覆盖。
	cached, err := svc.repo.Get(ctx, changeID, tenantID)
	require.NoError(t, err)
	require.Equal(t, "pending", cached.Status)

	realRepo := svc.repo.(*EntRepository)
	svc.repo = &staleViewChangeRepo{Repository: realRepo, view: cached}

	_, err = db.ExecContext(ctx, `UPDATE changes SET status = 'cancelled' WHERE id = ? AND tenant_id = ?`, changeID, tenantID)
	require.NoError(t, err)

	// CAS 未命中 = 状态已被并发修改：必须跳过（nil），不得把 quorum 结论落库。
	require.NoError(t, svc.checkAndTransitionChange(ctx, changeID, tenantID))

	reloaded, err := realRepo.Get(ctx, changeID, tenantID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", reloaded.Status,
		"陈旧 quorum 结论（approved）不得覆盖并发推进后的真实状态（cancelled）")
}

// updateCountRepo 统计 repo.Update 调用次数：TransitionStatus 非终态分支
// 修复后会完全不经过全列 Update。
type updateCountRepo struct {
	*mockRepository
	updateCalls int
}

func (r *updateCountRepo) Update(ctx context.Context, c *Change) (*Change, error) {
	r.updateCalls++
	return r.mockRepository.Update(ctx, c)
}

// TestTransitionStatus_NonTerminal_NoStaleFullOverwrite 状态推进成功后，
// 绝不能再用陈旧快照做全列 repo.Update（把并发提交的字段编辑静默卡掉）。
func TestTransitionStatus_NonTerminal_NoStaleFullOverwrite(t *testing.T) {
	repo := &updateCountRepo{mockRepository: newMockRepository()}
	svc := NewService(repo, nil, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()
	const tenantID, ownerID = 1, 42

	c := createTestChange(repo.mockRepository, tenantID, ownerID)
	require.Equal(t, "draft", c.Status)

	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, ownerID, "pending", "提交评审", "agent")
	require.NoError(t, err)
	assert.Equal(t, "pending", updated.Status)

	assert.Zero(t, repo.updateCalls,
		"非终态状态推进只改 status 一个字段，不得触发全列 repo.Update 二次覆盖")

	// 真实仓储侧：状态经 CAS 已推进。
	assert.Equal(t, "pending", repo.changes[c.ID].Status)
}
