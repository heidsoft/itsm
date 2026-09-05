package change

import (
	"context"
	"database/sql"
	"time"

	"itsm-backend/database"
)

// changeApprovalChainRepository 负责 change_approval_chains 表的查询与维护。
// 该表目前没有 Ent schema，因此按用户偏好封装成独立 repository：
//   - SQL 参数化、强制 tenant_id 谓词、链层级按 level 升序
//   - 由 EntRepository 聚合持有，对外保持原 Repository 接口不变
type changeApprovalChainRepository struct {
	db *sql.DB
}

func newChangeApprovalChainRepository(db *sql.DB) *changeApprovalChainRepository {
	if db == nil {
		return nil
	}
	return &changeApprovalChainRepository{db: db}
}

// ListByChange 返回审批链中每个 step（含审批人姓名）。
// 返回空切片而非 nil：避免 JSON 序列化为 null 导致前端崩溃。
func (r *changeApprovalChainRepository) ListByChange(ctx context.Context, changeID, tenantID int) ([]*ApprovalChain, error) {
	const query = `
		SELECT c.id, c.level, c.approver_id, u.name as approver_name, c.role, c.status, c.is_required, c.approval_type, c.threshold, c.created_at
		FROM change_approval_chains c
		LEFT JOIN users u ON c.approver_id = u.id
		WHERE c.change_id = $1 AND c.tenant_id = $2
		ORDER BY c.level ASC
	`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) ([]*ApprovalChain, error) {
		rows, err := q.QueryContext(ctx, query, changeID, tenantID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		chain := make([]*ApprovalChain, 0)
		for rows.Next() {
			var item ApprovalChain
			if err := rows.Scan(&item.ID, &item.Level, &item.ApproverID, &item.ApproverName, &item.Role, &item.Status, &item.IsRequired, &item.ApprovalType, &item.Threshold, &item.CreatedAt); err != nil {
				return nil, err
			}
			item.ChangeID, item.TenantID = changeID, tenantID
			chain = append(chain, &item)
		}
		return chain, rows.Err()
	})
}

// DeleteByChange 删除某变更的整条审批链（租户隔离）。
func (r *changeApprovalChainRepository) DeleteByChange(ctx context.Context, changeID, tenantID int) error {
	_, err := database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (struct{}, error) {
		_, err := q.ExecContext(ctx, "DELETE FROM change_approval_chains WHERE change_id = $1 AND tenant_id = $2", changeID, tenantID)
		return struct{}{}, err
	})
	return err
}

// LevelsByApprover 返回审批链中每位审批人的层级列表（按 level 升序），
// 供 ApprovalRecord.Levels 字段派生，避免跨层互相串。
func (r *changeApprovalChainRepository) LevelsByApprover(ctx context.Context, changeID, tenantID int) (map[int][]int, error) {
	const query = `
		SELECT approver_id, level
		FROM change_approval_chains
		WHERE change_id = $1 AND tenant_id = $2
		ORDER BY approver_id, level
	`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (map[int][]int, error) {
		rows, err := q.QueryContext(ctx, query, changeID, tenantID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		levels := make(map[int][]int)
		for rows.Next() {
			var approverID, level int
			if err := rows.Scan(&approverID, &level); err != nil {
				return nil, err
			}
			levels[approverID] = append(levels[approverID], level)
		}
		return levels, rows.Err()
	})
}

// InsertTx 在调用方已开启的 *sql.Tx 中插入一条审批链 step。
// 供 SubmitForApprovalWithWorkflow 这类需要"业务事务 + 通知 outbox + BPMN"原子化的场景使用。
//
// 注意：approvalType 缺省时回填为 "serial"，与 Replace 保持一致，
// 保证下游重解析链时 approval_type/threshold 不丢失。
func (r *changeApprovalChainRepository) InsertTx(ctx context.Context, tx *sql.Tx, changeID, tenantID, level, approverID int, role string, isRequired bool, approvalType string, threshold int, now time.Time) error {
	if approvalType == "" {
		approvalType = "serial"
	}
	const insert = `
		INSERT INTO change_approval_chains
			(change_id, tenant_id, level, approver_id, role, status, is_required, approval_type, threshold, created_at)
		VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7, $8, $9)
	`
	_, err := tx.ExecContext(ctx, insert,
		changeID, tenantID, level, approverID, role, isRequired, approvalType, threshold, now,
	)
	return err
}

// Replace 事务内清空并按 chain 顺序重写整条审批链。
// 必须使用事务保证 replace 语义原子性，否则下游会读到半成品链。
//
// 注意：本函数创建自己的事务并 commit；调用方禁止再开事务。
func (r *changeApprovalChainRepository) Replace(ctx context.Context, changeID, tenantID int, chain []*ApprovalChain) error {
	_, err := database.WithTenantTx(ctx, r.db, tenantID, func(tx *sql.Tx) (struct{}, error) {
		if _, err := tx.ExecContext(ctx,
			"DELETE FROM change_approval_chains WHERE change_id = $1 AND tenant_id = $2",
			changeID, tenantID,
		); err != nil {
			return struct{}{}, err
		}
		for _, item := range chain {
			// 保留 Quorum 元数据，否则重解析审批链会丢失 approval_type/threshold。
			approvalType := item.ApprovalType
			if approvalType == "" {
				approvalType = "serial"
			}
			const insert = `
			INSERT INTO change_approval_chains
				(change_id, tenant_id, level, approver_id, role, status, is_required, approval_type, threshold, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`
			if _, err := tx.ExecContext(ctx, insert,
				changeID, tenantID, item.Level, item.ApproverID, item.Role, item.Status,
				item.IsRequired, approvalType, item.Threshold, time.Now(),
			); err != nil {
				return struct{}{}, err
			}
		}
		return struct{}{}, nil
	})
	return err
}
