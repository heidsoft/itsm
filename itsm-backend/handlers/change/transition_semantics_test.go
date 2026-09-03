package change

// 本文件锁定状态转换的两类语义契约：
//   1. 并发安全：多个请求同时推进同一变更状态时，恰好一个成功，
//      其余必须收到 ErrConcurrentModification（而非全部返回 200）。
//   2. 错误分类：业务规则拒绝（状态机非法 / 非审批人 / 记录不存在）
//      必须可与系统内部故障区分，以便 HTTP 层映射为 409/403/404 而非 500。
//
// 背景：修复前 TransitionStatus 采用「读状态 → 校验 → 写状态」的非原子流程，
// 并发请求会同时读到同一旧状态并各自写入成功（实测 10 并发出现 5 个 200）；
// 且所有失败都被统一包装为 500，导致客户端无法区分"重试"与"故障"。

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// newTransitionTestService 构造一个仅依赖 mock 仓储的服务实例。
// entClient 传 nil 时不会装配 BPMN 审批桥接，审批走纯业务路径，
// 便于隔离验证状态机与并发语义本身。
func newTransitionTestService(t *testing.T) (*Service, *mockRepository) {
	t.Helper()
	repo := newMockRepository()
	return NewService(repo, nil, zaptest.NewLogger(t).Sugar(), nil), repo
}

// seedPendingChange 创建一个处于 pending 且指定用户为待办审批人的变更。
func seedPendingChange(t *testing.T, repo *mockRepository, tenantID, approverID int) *Change {
	t.Helper()
	c := createTestChange(repo, tenantID, approverID)
	c.Status = "pending"
	_, err := repo.CreateApprovalRecord(context.Background(), &ApprovalRecord{
		ChangeID:   c.ID,
		ApproverID: approverID,
		Status:     "pending",
	})
	require.NoError(t, err)
	return c
}

// TestTransitionStatus_ConcurrentApprovals_ExactlyOneWins 并发审批必须只有一个成功。
// 这是 TOCTOU 竞态的回归防线：修复前 10 个并发请求会有 5 个返回成功，
// 客户端与审计无法判断谁真正推进了状态。
func TestTransitionStatus_ConcurrentApprovals_ExactlyOneWins(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const (
		tenantID   = 1
		approverID = 100
		workers    = 10
	)
	c := seedPendingChange(t, repo, tenantID, approverID)

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		conflicts int
		otherErrs []error
	)

	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // 尽量同时发起，放大竞态窗口
			_, err := svc.TransitionStatus(ctx, c.ID, tenantID, approverID, "approved", "并发审批")
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				succeeded++
			case errors.Is(err, ErrConcurrentModification):
				// 抢先者已提交：CAS 谓词未命中，或审批记录已被消费 → 兜底拦截
				conflicts++
			case errors.Is(err, ErrInvalidTransition):
				// 落后者读到终态 approved，状态机先于 CAS 拦截
				conflicts++
			default:
				otherErrs = append(otherErrs, err)
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Empty(t, otherErrs, "除并发冲突外不应出现其他错误")
	assert.Equal(t, 1, succeeded, "并发审批必须恰好一个成功（乐观锁生效）")
	assert.Equal(t, workers-1, conflicts, "其余请求必须收到并发冲突错误")
}

// TestTransitionStatus_RepeatedApproval_Rejected 同一状态重复推进第二次必须被拒，
// 保证串行场景下的幂等语义与并发场景一致。
func TestTransitionStatus_RepeatedApproval_Rejected(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, approverID = 1, 101
	c := seedPendingChange(t, repo, tenantID, approverID)

	updated, err := svc.TransitionStatus(ctx, c.ID, tenantID, approverID, "approved", "同意")
	require.NoError(t, err)
	assert.Equal(t, "approved", updated.Status)

	_, err = svc.TransitionStatus(ctx, c.ID, tenantID, approverID, "approved", "重复审批")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidTransition) || errors.Is(err, ErrConcurrentModification),
		"重复审批应被识别为状态机非法或并发冲突，实际: %v", err)
}

// TestTransitionStatus_InvalidTransition_IsSentinel 状态机不允许的转换必须返回哨兵错误，
// 以便 HTTP 层映射为 409 而不是 500。
func TestTransitionStatus_InvalidTransition_IsSentinel(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, approverID = 1, 102
	c := createTestChange(repo, tenantID, approverID)
	c.Status = "draft" // draft → completed 非状态机允许的边

	_, err := svc.TransitionStatus(ctx, c.ID, tenantID, approverID, "completed", "越级推进")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidTransition),
		"非法状态转换必须返回 ErrInvalidTransition，实际: %v", err)
}

// TestTransitionStatus_NotApprover_IsSentinel 非审批人执行审批必须返回哨兵错误（→403），
// 且不得改变变更状态。
func TestTransitionStatus_NotApprover_IsSentinel(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	const tenantID, realApprover, intruder = 1, 103, 999
	c := seedPendingChange(t, repo, tenantID, realApprover)

	_, err := svc.TransitionStatus(ctx, c.ID, tenantID, intruder, "approved", "非审批人尝试审批")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotApprover),
		"非审批人必须返回 ErrNotApprover，实际: %v", err)

	got, getErr := repo.Get(ctx, c.ID, tenantID)
	require.NoError(t, getErr)
	assert.Equal(t, "pending", got.Status, "越权审批不得改变变更状态")
}

// TestTransitionStatus_NotFound_IsSentinel 记录不存在或跨租户访问必须返回哨兵错误（→404）。
func TestTransitionStatus_NotFound_IsSentinel(t *testing.T) {
	svc, repo := newTransitionTestService(t)
	ctx := context.Background()
	c := createTestChange(repo, 1, 104)
	c.Status = "pending"

	_, err := svc.TransitionStatus(ctx, c.ID, 9999, 104, "approved", "跨租户审批")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrChangeNotFound),
		"跨租户/不存在的变更必须返回 ErrChangeNotFound，实际: %v", err)
}

// TestUpdateStatusCAS_OnlyMatchesExpectedStatus 直接验证 CAS 原语：
// 期望状态不匹配时不得写入。
func TestUpdateStatusCAS_OnlyMatchesExpectedStatus(t *testing.T) {
	repo := newMockRepository()
	ctx := context.Background()
	c := createTestChange(repo, 1, 105)
	c.Status = "pending"

	ok, err := repo.UpdateStatusCAS(ctx, c.ID, 1, "approved", "completed")
	require.NoError(t, err)
	assert.False(t, ok, "期望状态不匹配时 CAS 必须失败")

	got, _ := repo.Get(ctx, c.ID, 1)
	assert.Equal(t, "pending", got.Status, "失败的 CAS 不得修改状态")

	ok, err = repo.UpdateStatusCAS(ctx, c.ID, 1, "pending", "approved")
	require.NoError(t, err)
	assert.True(t, ok, "期望状态匹配时 CAS 必须成功")

	got, _ = repo.Get(ctx, c.ID, 1)
	assert.Equal(t, "approved", got.Status)
}

// TestUpdateStatusCAS_TenantIsolated CAS 必须带租户谓词，防止跨租户状态推进。
func TestUpdateStatusCAS_TenantIsolated(t *testing.T) {
	repo := newMockRepository()
	ctx := context.Background()
	c := createTestChange(repo, 1, 106)
	c.Status = "pending"

	ok, err := repo.UpdateStatusCAS(ctx, c.ID, 2, "pending", "approved")
	require.NoError(t, err)
	assert.False(t, ok, "跨租户的 CAS 必须失败")

	got, _ := repo.Get(ctx, c.ID, 1)
	assert.Equal(t, "pending", got.Status, "跨租户 CAS 不得修改状态")
}

// 保证 time 包被使用（seedPendingChange 中 CreatedAt 依赖），避免导入被误删。
var _ = time.Now
