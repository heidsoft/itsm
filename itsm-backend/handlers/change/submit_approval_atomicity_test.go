package change

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestSubmitApproval_AtomicSuccess_RecordAndStateCommittedTogether
//
// P0-1 修复的 happy path：
// 1) 仓储在 *sql.Tx 中写入审批记录（RETURNING 出 ID/CreatedAt）；
// 2) 仓储在同一 *sql.Tx 中 CAS 推进 draft→pending；
// 3) service 不再调用非 CAS 的 repo.Update(c)；
// 4) service 直接复用事务内返回的 rec 作为响应，**不允许**二次
//    CreateApprovalRecord 造成重复插入。
func TestSubmitApproval_AtomicSuccess_RecordAndStateCommittedTogether(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	repo := newMockRepository()

	const (
		tenantID   = 1
		changeID   = 10
		approverID = 2
	)
	// 准备 draft 变更，并把 approverID 加入审批链（service 会校验）
	repo.changes[changeID] = &Change{
		ID:        changeID,
		TenantID:  tenantID,
		CreatedBy: 1,
		Status:    "draft",
	}
	repo.chains[changeID] = []*ApprovalChain{
		{ID: 1, ChangeID: changeID, TenantID: tenantID, Level: 1, ApproverID: approverID},
	}

	svc := NewService(repo, nil, logger, nil)
	comment := "ok"
	rec := &ApprovalRecord{
		ChangeID:   changeID,
		ApproverID: approverID,
		Comment:    &comment,
	}

	returned, err := svc.SubmitApproval(context.Background(), rec, tenantID)
	require.NoError(t, err)

	// rec 已用 RETURNING 回填 ID / CreatedAt
	assert.NotZero(t, returned.ID, "SubmitApprovalRecordTx 应通过 RETURNING 回填 ID")
	assert.False(t, returned.CreatedAt.IsZero(), "SubmitApprovalRecordTx 应通过 RETURNING 回填 CreatedAt")
	assert.Equal(t, "pending", returned.Status)

	// draft 已 CAS 推进到 pending
	repo.mu.Lock()
	defer repo.mu.Unlock()
	assert.Equal(t, "pending", repo.changes[changeID].Status)

	// 必须只有一条审批记录（无重复插入）
	count := 0
	for _, a := range repo.approvals {
		if a.ChangeID == changeID && a.ApproverID == approverID {
			count++
		}
	}
	assert.Equal(t, 1, count, "必须恰好写入一条审批记录，禁止重复插入")
}

// TestSubmitApproval_TransactionFailure_LeavesNoSideEffects
//
// P0-1 修复的关键不变量：若仓储内 *sql.Tx 中任一写入失败，必须整体回滚——
// 不能出现"审批记录已写入但状态未推进"或"状态推进了但没有审批记录"的
// 部分失败窗口。该测试在仓储层模拟事务失败，断言 service 透传错误并且
// mock 上的副作用状态保持失败前的快照。
func TestSubmitApproval_TransactionFailure_LeavesNoSideEffects(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	repo := newMockRepository()
	repo.submitApprovalErr = errors.New("injected atomic tx failure")

	const (
		tenantID   = 1
		changeID   = 11
		approverID = 3
	)
	repo.changes[changeID] = &Change{
		ID:        changeID,
		TenantID:  tenantID,
		CreatedBy: 1,
		Status:    "draft",
	}
	repo.chains[changeID] = []*ApprovalChain{
		{ID: 1, ChangeID: changeID, TenantID: tenantID, Level: 1, ApproverID: approverID},
	}

	svc := NewService(repo, nil, logger, nil)
	_, err := svc.SubmitApproval(context.Background(), &ApprovalRecord{
		ChangeID:   changeID,
		ApproverID: approverID,
	}, tenantID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "提交审批失败")

	repo.mu.Lock()
	defer repo.mu.Unlock()
	// 不变量：状态没被推进；没有审批记录。
	assert.Equal(t, "draft", repo.changes[changeID].Status)
	for _, a := range repo.approvals {
		assert.NotEqual(t, changeID, a.ChangeID, "事务失败时审批记录必须回滚")
	}
}

// TestSubmitApproval_CASGuard_NonDraftDoesNotRegressState
//
// P0-1 修复：CAS 守卫保证即使变更已处于 pending/approved/in_progress
// 等非 draft 状态，重复调用 SubmitApproval 也只会追加审批记录，
// 不会把状态回退或覆盖。多个审批人分轮次提交同一变更审批时
// 行为必须一致。
func TestSubmitApproval_CASGuard_NonDraftDoesNotRegressState(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	repo := newMockRepository()

	const (
		tenantID    = 1
		changeID    = 12
		approverID1 = 4
		approverID2 = 5
	)
	repo.changes[changeID] = &Change{
		ID:        changeID,
		TenantID:  tenantID,
		CreatedBy: 1,
		Status:    "pending", // 已经处于 pending
	}
	repo.chains[changeID] = []*ApprovalChain{
		{ID: 1, ChangeID: changeID, TenantID: tenantID, Level: 1, ApproverID: approverID1},
		{ID: 2, ChangeID: changeID, TenantID: tenantID, Level: 2, ApproverID: approverID2},
	}

	svc := NewService(repo, nil, logger, nil)
	returned, err := svc.SubmitApproval(context.Background(), &ApprovalRecord{
		ChangeID:   changeID,
		ApproverID: approverID1,
	}, tenantID)
	require.NoError(t, err)
	assert.NotZero(t, returned.ID)

	repo.mu.Lock()
	assert.Equal(t, "pending", repo.changes[changeID].Status, "非 draft 提交必须保持状态不被回退")
	recordCount := 0
	for _, a := range repo.approvals {
		if a.ChangeID == changeID {
			recordCount++
		}
	}
	repo.mu.Unlock()
	assert.Equal(t, 1, recordCount, "追加一条审批记录")
}

// TestSubmitApproval_RejectsApproverNotInChain
//
// P0-1 修复：service 必须拒绝把请求体里的 approverId 注入到审批流中，
// 仅允许审批链中指定的审批人提交。
func TestSubmitApproval_RejectsApproverNotInChain(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	repo := newMockRepository()

	const (
		tenantID      = 1
		changeID      = 13
		designated    = 6
		stranger      = 999
		approverValid = true
	)
	repo.changes[changeID] = &Change{
		ID:        changeID,
		TenantID:  tenantID,
		CreatedBy: 1,
		Status:    "draft",
	}
	repo.chains[changeID] = []*ApprovalChain{
		{ID: 1, ChangeID: changeID, TenantID: tenantID, Level: 1, ApproverID: designated},
	}
	repo.approverValid = approverValid

	svc := NewService(repo, nil, logger, nil)
	_, err := svc.SubmitApproval(context.Background(), &ApprovalRecord{
		ChangeID:   changeID,
		ApproverID: stranger,
	}, tenantID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不是变更 13 的指定审批人")

	// 拒绝路径不应触达仓储原子写入
	repo.mu.Lock()
	defer repo.mu.Unlock()
	assert.Equal(t, "draft", repo.changes[changeID].Status)
	for _, a := range repo.approvals {
		assert.NotEqual(t, changeID, a.ChangeID)
	}
}

// TestSubmitApproval_TimestampFromCallerNotElapsedClock
//
// P0-1 修复：SubmitApprovalRecordTx 必须使用 service 传入的 now，
// 由仓储负责事务边界。这避免了"INSERT 的 created_at 来自
// 数据库默认时间、CAS 的 updated_at 来自 service.now"导致
// 同一事务内两条写入时间戳不一致的审计缺陷。
func TestSubmitApproval_TimestampFromCallerNotElapsedClock(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	repo := newMockRepository()

	const (
		tenantID   = 1
		changeID   = 14
		approverID = 7
	)
	repo.changes[changeID] = &Change{
		ID:        changeID,
		TenantID:  tenantID,
		CreatedBy: 1,
		Status:    "draft",
	}
	repo.chains[changeID] = []*ApprovalChain{
		{ID: 1, ChangeID: changeID, TenantID: tenantID, Level: 1, ApproverID: approverID},
	}

	svc := NewService(repo, nil, logger, nil)
	returned, err := svc.SubmitApproval(context.Background(), &ApprovalRecord{
		ChangeID:   changeID,
		ApproverID: approverID,
	}, tenantID)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), returned.CreatedAt, 2*time.Second,
		"审批记录的 CreatedAt 必须取自 service 传入的 now（事务时间戳）")
}
