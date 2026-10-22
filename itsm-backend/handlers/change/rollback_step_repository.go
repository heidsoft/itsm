package change

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"itsm-backend/database"
)

// changeRollbackStepRepository 负责 change_rollback_steps 表的读写。
//
// 该表没有 Ent schema（与 change_approval_chains 同口径：域内子表用原生 SQL，
// 避免为一个纯子表重建整棵 Ent 树），因此本仓储承担三件事：
//   - 参数化 SQL，绝不拼串；
//   - 每条语句强制 (tenant_id, change_id) 双谓词——父表 changes 有 RLS 策略，
//     子表没有，租户边界只能靠仓储自身守住；
//   - 状态推进用条件更新（只有 pending 才能被确认），避免两个人同时勾同一
//     步时后者覆盖前者的备注与时间。
type changeRollbackStepRepository struct {
	db *sql.DB
}

func newChangeRollbackStepRepository(db *sql.DB) *changeRollbackStepRepository {
	if db == nil {
		return nil
	}
	return &changeRollbackStepRepository{db: db}
}

// ListByChange 返回变更的回滚清单（按 step_no 升序）。
// 返回空切片而非 nil：前端对 null 做 map 会崩，且「没有清单」与「读失败」
// 必须在下游可区分（读失败返回 error）。
func (r *changeRollbackStepRepository) ListByChange(ctx context.Context, changeID, tenantID int) ([]*RollbackStep, error) {
	const query = `
		SELECT s.id, s.change_id, s.tenant_id, s.step_no, s.description, s.status,
		       COALESCE(s.note, ''), s.resolved_by, COALESCE(u.name, ''), s.resolved_at,
		       s.created_at, s.updated_at
		FROM change_rollback_steps s
		LEFT JOIN users u ON s.resolved_by = u.id
		WHERE s.change_id = $1 AND s.tenant_id = $2
		ORDER BY s.step_no ASC
	`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) ([]*RollbackStep, error) {
		rows, err := q.QueryContext(ctx, query, changeID, tenantID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		steps := make([]*RollbackStep, 0)
		for rows.Next() {
			var (
				step       RollbackStep
				resolvedBy sql.NullInt64
				resolvedAt sql.NullTime
			)
			if err := rows.Scan(&step.ID, &step.ChangeID, &step.TenantID, &step.StepNo,
				&step.Description, &step.Status, &step.Note, &resolvedBy, &step.ResolvedByName,
				&resolvedAt, &step.CreatedAt, &step.UpdatedAt); err != nil {
				return nil, err
			}
			if resolvedBy.Valid {
				id := int(resolvedBy.Int64)
				step.ResolvedBy = &id
			}
			if resolvedAt.Valid {
				at := resolvedAt.Time
				step.ResolvedAt = &at
			}
			steps = append(steps, &step)
		}
		return steps, rows.Err()
	})
}

// Replace 事务内清空并重写整张清单（replace 语义必须原子，否则中途失败会
// 留下半张清单，回滚门禁就会按残缺清单放行）。
//
// 只允许在「回滚执行前」调用由 service 层把关；本方法只负责持久化。
func (r *changeRollbackStepRepository) Replace(ctx context.Context, changeID, tenantID int, steps []*RollbackStep) error {
	_, err := database.WithTenantTx(ctx, r.db, tenantID, func(tx *sql.Tx) (struct{}, error) {
		if _, err := tx.ExecContext(ctx,
			"DELETE FROM change_rollback_steps WHERE change_id = $1 AND tenant_id = $2",
			changeID, tenantID,
		); err != nil {
			return struct{}{}, err
		}
		const insert = `
			INSERT INTO change_rollback_steps
				(change_id, tenant_id, step_no, description, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'pending', $5, $5)
		`
		now := time.Now()
		for i, step := range steps {
			stepNo := step.StepNo
			if stepNo == 0 {
				stepNo = i + 1
			}
			if _, err := tx.ExecContext(ctx, insert, changeID, tenantID, stepNo, step.Description, now); err != nil {
				return struct{}{}, err
			}
		}
		return struct{}{}, nil
	})
	return err
}

// Resolve 确认某一步的执行结果（含撤销误操作的 status=pending）。
//
// 条件更新的两段语义：
//   - 前进（done/skipped/failed）：只接受当前 status='pending' 的行。已确认的
//     步骤重复提交返回 (false, nil)，由 service 转成「已被某人确认」的 409，
//     而不是静默覆盖前一位操作人的备注与时间戳；
//   - 回退（status='pending'）：允许解开任意已确认行（复核发现勾错了），
//     清空确认人/时间但保留备注线索。若这里也要求 status='pending'，
//     撤销在物理上就不可能发生。
//
// 返回 false 的两种成因（不存在 / 已被确认）由 service 重新读取清单来区分，
// 与审批并发冲突的处置口径一致。
func (r *changeRollbackStepRepository) Resolve(ctx context.Context, stepID, changeID, tenantID, actorID int, status, note string, now time.Time) (bool, error) {
	const update = `
		UPDATE change_rollback_steps
		SET status = $1,
		    note = $2,
		    resolved_by = CASE WHEN $1 = 'pending' THEN NULL ELSE $3 END,
		    resolved_at = CASE WHEN $1 = 'pending' THEN NULL ELSE $4 END,
		    updated_at = $4
		WHERE id = $5 AND change_id = $6 AND tenant_id = $7
		  AND (status = 'pending' OR $1 = 'pending')
	`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (bool, error) {
		res, err := q.ExecContext(ctx, update, status, note, actorID, now, stepID, changeID, tenantID)
		if err != nil {
			return false, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("read rows affected: %w", err)
		}
		return affected > 0, nil
	})
}

// DeleteByChange 事务内删除整张清单（仅用于 Replace 的失败回退与测试装配）。
func (r *changeRollbackStepRepository) DeleteByChange(ctx context.Context, changeID, tenantID int) error {
	_, err := database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (struct{}, error) {
		_, execErr := q.ExecContext(ctx,
			"DELETE FROM change_rollback_steps WHERE change_id = $1 AND tenant_id = $2",
			changeID, tenantID,
		)
		return struct{}{}, execErr
	})
	return err
}
