package change

import (
	"context"
	"database/sql"
	"time"
)

// changeStatusTxRepository 提供对 changes 表的"事务内"状态推进操作。
// 当前 SubmitForApprovalWithWorkflow 需要在一个跨表事务内做
//   - changes.status: draft -> pending
//   - change_approvals 写入
//   - change_approval_chains 写入
//   - notification outbox enqueue
// 因此单独抽出 change 状态推进的 Tx 入口，让 SubmitForApprovalWithWorkflow
// 不必再内联 SQL，并保留"条件 UPDATE + 影响行校验"的幂等保证。
type changeStatusTxRepository struct{}

func newChangeStatusTxRepository() *changeStatusTxRepository {
	return &changeStatusTxRepository{}
}

// PromoteDraftToPending 在 *sql.Tx 中将指定变更从 draft 推进到 pending。
// 仅当 change 当前仍处于 draft 且属于该租户时才返回 promoted=true；
// 其它情况返回 false，调用方据此判断是否继续发审批通知。
func (r *changeStatusTxRepository) PromoteDraftToPending(ctx context.Context, tx *sql.Tx, changeID, tenantID int, now time.Time) (bool, error) {
	const query = `
		UPDATE changes
		SET status = 'pending', updated_at = $1
		WHERE id = $2 AND tenant_id = $3 AND status = 'draft'
	`
	result, err := tx.ExecContext(ctx, query, now, changeID, tenantID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}
