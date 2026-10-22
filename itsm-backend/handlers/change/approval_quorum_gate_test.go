package change

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"itsm-backend/ent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	_ "github.com/mattn/go-sqlite3"
)

// ==================== 生产路径（/transition approved）会签门禁 ====================
//
// 这批用例锁定一个已发生的真实缺陷：会签（parallel/all）与阈值（N of M）语义
// 此前只写在无路由的 legacy 入口里，生产走 TransitionStatus 时"一票即批准"。
// 回归锁必须走 TransitionStatus（生产入口），不能只测 evaluateApprovalQuorum 纯函数。

// seedBridgeDecision 直接注入一条审批决定的审计行（模拟另一位审批人已经批过）。
// business_id 必须写：会签求值按 (business_type, business_id) 过滤决策表。
func seedBridgeDecision(t *testing.T, client *ent.Client, tenantID, changeID, approverID int, decision string) {
	t.Helper()
	action := "approve"
	if decision == "rejected" {
		action = "reject"
	}
	_, err := client.ProcessApprovalDecision.Create().
		SetProcessInstanceID(changeID).
		SetProcessTaskID(changeID*100 + approverID).
		SetProcessInstanceKey(fmt.Sprintf("change-evt-%d", changeID)).
		SetTaskID(fmt.Sprintf("TASK-%d-%d", changeID, approverID)).
		SetProcessDefinitionKey("change").
		SetNodeKey("Approval_1").
		SetBusinessType("change").
		SetBusinessID(strconv.Itoa(changeID)).
		SetActorID(approverID).SetActorName(fmt.Sprintf("approver-%d", approverID)).
		SetAction(action).SetDecision(decision).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
}

func countDecisions(t *testing.T, client *ent.Client) int {
	t.Helper()
	n, err := client.ProcessApprovalDecision.Query().Count(context.Background())
	require.NoError(t, err)
	return n
}

// TestTransitionStatus_ParallelQuorumNotMet_StaysPending 两人会签，一人批准：
// 决策必须落库，但变更不得进入 approved（此前这里直接 200 + approved）。
func TestTransitionStatus_ParallelQuorumNotMet_StaysPending(t *testing.T) {
	entClient := newChangeBridgeEntClient(t, "change_quorum_not_met")
	tenantID, actorA := setupChangeBridgeActor(t, entClient, "qm1")
	actorB := mkChangeUser(t, entClient, tenantID, "security_admin")
	repo := newMockRepository()
	svc := NewService(repo, entClient, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	c := createTestChange(repo, tenantID, actorA)
	c.Status = "pending"
	persistChangeInEntClient(t, entClient, c)
	require.NoError(t, repo.ReplaceApprovalChain(ctx, c.ID, tenantID, []*ApprovalChain{
		{ID: 1, ChangeID: c.ID, TenantID: tenantID, Level: 1, ApproverID: actorA, Role: "approver",
			Status: "pending", IsRequired: true, ApprovalType: "parallel", Threshold: 2, CreatedAt: time.Now()},
		{ID: 2, ChangeID: c.ID, TenantID: tenantID, Level: 1, ApproverID: actorB, Role: "approver",
			Status: "pending", IsRequired: true, ApprovalType: "parallel", Threshold: 2, CreatedAt: time.Now()},
	}))
	createChangeBridgeProcessFixture(t, entClient, tenantID, "qm1",
		fmt.Sprintf("change:%d", c.ID), actorA)

	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorA, "approved", "同意", "agent")
	require.NoError(t, err, "记录审批决定本身不应报错")
	assert.Equal(t, "pending", updated.Status, "会签未集齐不得推进 approved")

	// 决策已写入审计表：审批动作不能"看起来成功、实际没留痕"。
	assert.Equal(t, 1, countDecisions(t, entClient), "本次审批的决策必须落库")
}

// TestTransitionStatus_ParallelQuorumMet_Approves 另一位审批人已批过，最后一票到位即批准。
func TestTransitionStatus_ParallelQuorumMet_Approves(t *testing.T) {
	entClient := newChangeBridgeEntClient(t, "change_quorum_met")
	tenantID, actorA := setupChangeBridgeActor(t, entClient, "qm2")
	actorB := mkChangeUser(t, entClient, tenantID, "security_admin")
	repo := newMockRepository()
	svc := NewService(repo, entClient, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	c := createTestChange(repo, tenantID, actorA)
	c.Status = "pending"
	persistChangeInEntClient(t, entClient, c)
	require.NoError(t, repo.ReplaceApprovalChain(ctx, c.ID, tenantID, []*ApprovalChain{
		{ID: 1, ChangeID: c.ID, TenantID: tenantID, Level: 1, ApproverID: actorA, Role: "approver",
			Status: "pending", IsRequired: true, ApprovalType: "parallel", Threshold: 2, CreatedAt: time.Now()},
		{ID: 2, ChangeID: c.ID, TenantID: tenantID, Level: 1, ApproverID: actorB, Role: "approver",
			Status: "pending", IsRequired: true, ApprovalType: "parallel", Threshold: 2, CreatedAt: time.Now()},
	}))
	// b 已经批过（第一票）
	seedBridgeDecision(t, entClient, tenantID, c.ID, actorB, "approved")
	createChangeBridgeProcessFixture(t, entClient, tenantID, "qm2",
		fmt.Sprintf("change:%d", c.ID), actorA)

	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorA, "approved", "同意", "agent")
	require.NoError(t, err)
	assert.Equal(t, "approved", updated.Status, "会签集齐应批准")
	assert.Equal(t, 2, countDecisions(t, entClient))
}

// TestTransitionStatus_ThresholdNotMet_StaysPending 3 候选取 2 票：一票不得放行。
func TestTransitionStatus_ThresholdNotMet_StaysPending(t *testing.T) {
	entClient := newChangeBridgeEntClient(t, "change_quorum_threshold")
	tenantID, actorA := setupChangeBridgeActor(t, entClient, "qm3")
	actorB := mkChangeUser(t, entClient, tenantID, "security_admin")
	actorC := mkChangeUser(t, entClient, tenantID, "technician")
	repo := newMockRepository()
	svc := NewService(repo, entClient, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	c := createTestChange(repo, tenantID, actorA)
	c.Status = "pending"
	persistChangeInEntClient(t, entClient, c)
	chain := make([]*ApprovalChain, 0, 3)
	for i, id := range []int{actorA, actorB, actorC} {
		chain = append(chain, &ApprovalChain{ID: i + 1, ChangeID: c.ID, TenantID: tenantID, Level: 1,
			ApproverID: id, Role: "approver", Status: "pending", IsRequired: true,
			ApprovalType: "parallel", Threshold: 2, CreatedAt: time.Now()})
	}
	require.NoError(t, repo.ReplaceApprovalChain(ctx, c.ID, tenantID, chain))
	createChangeBridgeProcessFixture(t, entClient, tenantID, "qm3",
		fmt.Sprintf("change:%d", c.ID), actorA)

	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorA, "approved", "同意", "agent")
	require.NoError(t, err)
	assert.Equal(t, "pending", updated.Status, "2/3 阈值未达，单票不得放行")
}

// TestTransitionStatus_RejectedDecisionBlocksApprove 链上已有驳回票时，
// 任何"通过"动作都必须被拒（409），不能把已否决的单子翻回来。
func TestTransitionStatus_RejectedDecisionBlocksApprove(t *testing.T) {
	entClient := newChangeBridgeEntClient(t, "change_quorum_rejected")
	tenantID, actorA := setupChangeBridgeActor(t, entClient, "qm4")
	actorB := mkChangeUser(t, entClient, tenantID, "security_admin")
	repo := newMockRepository()
	svc := NewService(repo, entClient, zaptest.NewLogger(t).Sugar(), nil)
	ctx := context.Background()

	c := createTestChange(repo, tenantID, actorA)
	c.Status = "pending"
	persistChangeInEntClient(t, entClient, c)
	require.NoError(t, repo.ReplaceApprovalChain(ctx, c.ID, tenantID, []*ApprovalChain{
		{ID: 1, ChangeID: c.ID, TenantID: tenantID, Level: 1, ApproverID: actorA, Role: "approver",
			Status: "pending", IsRequired: true, ApprovalType: "any", Threshold: 1, CreatedAt: time.Now()},
		{ID: 2, ChangeID: c.ID, TenantID: tenantID, Level: 1, ApproverID: actorB, Role: "approver",
			Status: "pending", IsRequired: true, ApprovalType: "any", Threshold: 1, CreatedAt: time.Now()},
	}))
	seedBridgeDecision(t, entClient, tenantID, c.ID, actorB, "rejected")
	createChangeBridgeProcessFixture(t, entClient, tenantID, "qm4",
		fmt.Sprintf("change:%d", c.ID), actorA)

	_, err := svc.TransitionStatus(ctx, c.ID, tenantID, actorA, "approved", "同意", "agent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

// ==================== 终态收口的事务原子性 ====================

// TestTransitionStatus_Reject_ClosesChainsSameTx 终态流转必须把 pending 审批链
// 置为 obsolete，否则审批人的待办里会永远挂着已终结的变更。
func TestTransitionStatus_Reject_ClosesChainsSameTx(t *testing.T) {
	svc, client, db, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	approver := mkChangeUser(t, client, tenantID, "manager")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)
	plan := []ApprovalLevelPlan{{Level: 1, ApprovalType: "serial", Threshold: 1, Required: true, ApproverIDs: []int{approver}}}
	require.NoError(t, svc.repo.SubmitForApproval(ctx, changeID, tenantID, plan, "submit"))
	require.Equal(t, "pending", queryChangeStatus(t, client, changeID))

	_, err := svc.TransitionStatus(ctx, changeID, tenantID, approver, "rejected", "不同意", "manager")
	require.NoError(t, err)
	assert.Equal(t, "rejected", queryChangeStatus(t, client, changeID))

	var pending int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM change_approval_chains WHERE change_id = $1 AND tenant_id = $2 AND status = 'pending'`,
		changeID, tenantID).Scan(&pending))
	assert.Zero(t, pending, "终态后不得残留 pending 审批链")
}

// TestTransitionStatus_ChainCloseFailure_RollsBackStatus 收口审批链失败必须整体回滚：
// 状态不能已经落成终态、链却没收口（旧实现 commit 后再收口，失败只记日志）。
func TestTransitionStatus_ChainCloseFailure_RollsBackStatus(t *testing.T) {
	svc, client, db, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	approver := mkChangeUser(t, client, tenantID, "manager")
	creator := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, creator)
	plan := []ApprovalLevelPlan{{Level: 1, ApprovalType: "serial", Threshold: 1, Required: true, ApproverIDs: []int{approver}}}
	require.NoError(t, svc.repo.SubmitForApproval(ctx, changeID, tenantID, plan, "submit"))

	// 注入故障：任何对 change_approval_chains 的 UPDATE 都 ABORT。
	// 这样读路径（isApprover 用的 SELECT）仍正常，只有收口语句失败 ——
	// 正是要验证的那条边。
	_, err := db.ExecContext(ctx, `
		CREATE TRIGGER fail_chain_close BEFORE UPDATE ON change_approval_chains
		BEGIN SELECT RAISE(ABORT, 'injected chain close failure'); END;
	`)
	require.NoError(t, err)

	_, err = svc.TransitionStatus(ctx, changeID, tenantID, approver, "rejected", "不同意", "manager")
	require.Error(t, err, "收口失败必须冒泡，不能静默降级")

	// 关键断言：状态仍是 pending —— 状态更新与收口在同一事务，收口失败即回滚。
	assert.Equal(t, "pending", queryChangeStatus(t, client, changeID), "收口失败时状态推进必须一并回滚")
}
