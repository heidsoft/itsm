package change

// 常规 Update 的乐观锁回归（P1-2）：
//   change schema 没有 version 列，此前 repo.Update 只带租户谓词，
//   handler 又是「读快照 → 合并请求 → 全列回写」。于是审批刚推进的状态
//   会被一个并发的字段编辑用陈旧快照静默覆盖回去，而同域 incident/problem/
//   service_request 都有各自的并发控制，change 是五域里唯一的裸写。
//
// 修复后：repo.Update 的 WHERE 带上 StatusEQ（状态即版本号），service 把待写
// 状态钉死为读取时的当前值。状态在读写窗口内被抢改 → 0 行命中 → 409 冲突，
// 而不是把推进后的状态覆盖回去。

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"itsm-backend/common"
)

// concurrentAdvanceRepo 在 Get 与 Update 之间插入一次「并发状态推进」：
// Get 返回调用方捕获的陈旧快照，Update 落库前先把真实状态改掉，
// 从而把并发窗口显式化——这正是生产里审批推进与字段编辑相撞的形状。
type concurrentAdvanceRepo struct {
	Repository
	stale  *Change
	onOpen func()
}

func (r *concurrentAdvanceRepo) Get(ctx context.Context, id, tenantID int) (*Change, error) {
	return r.stale, nil
}

func (r *concurrentAdvanceRepo) Update(ctx context.Context, c *Change) (*Change, error) {
	if r.onOpen != nil {
		r.onOpen()
	}
	return r.Repository.Update(ctx, c)
}

// TestUpdateChange_StatusCAS_RejectsConcurrentStatusAdvance 状态在读写窗口内
// 被并发推进时，常规更新必须 409 冲突失败，且不得把状态改回旧值。
func TestUpdateChange_StatusCAS_RejectsConcurrentStatusAdvance(t *testing.T) {
	svc, client, db, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	owner := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, owner)

	// 调用方读到的快照：draft。
	stale, err := svc.repo.Get(ctx, changeID, tenantID)
	require.NoError(t, err)
	require.Equal(t, "draft", stale.Status)

	realRepo := svc.repo.(*EntRepository)
	svc.repo = &concurrentAdvanceRepo{
		Repository: realRepo,
		stale:      stale,
		// 模拟并发：审批把状态推进为 pending。
		onOpen: func() {
			_, err := db.ExecContext(ctx,
				`UPDATE changes SET status = 'pending' WHERE id = $1 AND tenant_id = $2`,
				changeID, tenantID)
			require.NoError(t, err)
		},
	}

	edited := *stale
	edited.Title = "并发编辑的标题"
	_, err = svc.UpdateChange(ctx, &edited, owner, "super_admin")

	// 必须是冲突（409），不是 404——404 会让调用方误以为单子被删。
	require.Error(t, err)
	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr, "并发状态推进必须翻译为可分类的业务错误")
	assert.Equal(t, common.ErrCodeConflict, appErr.Code, "应归类为冲突而非其他语义")

	// 关键：状态不得被陈旧快照覆盖回去，字段编辑也不得半途落库。
	reloaded, err := realRepo.Get(ctx, changeID, tenantID)
	require.NoError(t, err)
	assert.Equal(t, "pending", reloaded.Status, "并发推进后的状态不得被陈旧快照回退")
	assert.Equal(t, "Chain Change", reloaded.Title, "冲突时字段编辑同样不得落库")
}

// TestUpdateChange_PreservesStatus 常规更新不是状态推进入口：
// 即使调用方在快照里塞了别的状态，落库的也必须是读取时的当前状态。
func TestUpdateChange_PreservesStatus(t *testing.T) {
	svc, client, db, tenantID := setupChangeChainTest(t)
	ctx := context.Background()

	owner := mkChangeUser(t, client, tenantID, "end_user")
	changeID := mkChangeDraft(t, client, tenantID, owner)

	_, err := db.ExecContext(ctx,
		`UPDATE changes SET status = 'pending' WHERE id = $1 AND tenant_id = $2`, changeID, tenantID)
	require.NoError(t, err)

	current, err := svc.repo.Get(ctx, changeID, tenantID)
	require.NoError(t, err)
	require.Equal(t, "pending", current.Status)

	// 调用方（或未来的新入口）试图顺手把状态改回 draft。
	current.Status = "draft"
	current.Title = "改标题但不改状态"

	updated, err := svc.UpdateChange(ctx, current, owner, "super_admin")
	require.NoError(t, err)
	assert.Equal(t, "pending", updated.Status, "常规更新不得成为状态推进入口")

	reloaded, err := svc.repo.Get(ctx, changeID, tenantID)
	require.NoError(t, err)
	assert.Equal(t, "pending", reloaded.Status)
	assert.Equal(t, "改标题但不改状态", reloaded.Title, "非状态字段仍应正常落库")
}
