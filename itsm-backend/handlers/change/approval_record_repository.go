package change

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"itsm-backend/database"
)

// changeApprovalRecordRepository 负责 change_approvals 表的 CRUD。
// 该表目前没有 Ent schema，因此按用户的偏好封装成独立 repository：
//   - SQL 参数化、强制 tenant_id 谓词、空集返回 make([]*ApprovalRecord, 0)
//   - 由 EntRepository 聚合持有，对外保持原 Repository 接口不变
//   - SQL 字符串集中在文件顶部，后续如新增 schema 可平滑迁移到 Ent
type changeApprovalRecordRepository struct {
	db *sql.DB
}

func newChangeApprovalRecordRepository(db *sql.DB) *changeApprovalRecordRepository {
	if db == nil {
		return nil
	}
	return &changeApprovalRecordRepository{db: db}
}

// Create 插入一条审批记录并回填 ID/createdAt 到入参 rec。
// 租户隔离：所有参数显式绑定 tenantID，不依赖外部调用方补齐。
func (r *changeApprovalRecordRepository) Create(ctx context.Context, rec *ApprovalRecord) (*ApprovalRecord, error) {
	const query = `
		INSERT INTO change_approvals (change_id, tenant_id, approver_id, status, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	return database.WithTenantTx(ctx, r.db, rec.TenantID, func(tx *sql.Tx) (*ApprovalRecord, error) {
		now := time.Now()
		if err := tx.QueryRowContext(ctx, query,
			rec.ChangeID, rec.TenantID, rec.ApproverID, rec.Status, rec.Comment, now, now,
		).Scan(&rec.ID, &rec.CreatedAt); err != nil {
			return nil, err
		}
		return rec, nil
	})
}

// CreateTx 是 Create 的事务版：在调用方已开启的 *sql.Tx 中插入，
// 供 SubmitForApprovalWithWorkflow 这类需要"业务事务 + 通知 outbox + BPMN"原子化的场景使用。
func (r *changeApprovalRecordRepository) CreateTx(ctx context.Context, tx *sql.Tx, changeID, tenantID, approverID int, comment string, now time.Time) error {
	const query = `
		INSERT INTO change_approvals
			(change_id, tenant_id, approver_id, status, comment, created_at, updated_at)
		VALUES ($1, $2, $3, 'pending', $4, $5, $5)
	`
	_, err := tx.ExecContext(ctx, query, changeID, tenantID, approverID, comment, now)
	return err
}

// Update 条件更新审批记录（C-5 修复：必须加 AND status='pending' 守卫）。
// 校验 RowsAffected == 1，否则返回冲突错误，避免幂等问题。
func (r *changeApprovalRecordRepository) Update(ctx context.Context, rec *ApprovalRecord) (*ApprovalRecord, error) {
	const query = `
		UPDATE change_approvals
		SET status = $1, comment = $2, approved_at = $3, updated_at = $4
		WHERE id = $5 AND tenant_id = $6 AND status = 'pending'
		RETURNING id, change_id, tenant_id, approver_id, status, comment, approved_at, created_at
	`
	return database.WithTenantTx(ctx, r.db, rec.TenantID, func(tx *sql.Tx) (*ApprovalRecord, error) {
		var approvedAt sql.NullTime
		now := time.Now()
		err := tx.QueryRowContext(ctx, query,
			rec.Status, rec.Comment, now, now, rec.ID, rec.TenantID,
		).Scan(&rec.ID, &rec.ChangeID, &rec.TenantID, &rec.ApproverID, &rec.Status, &rec.Comment, &approvedAt, &rec.CreatedAt)
		if err == nil {
			if approvedAt.Valid {
				rec.ApprovedAt = &approvedAt.Time
			}
			return rec, nil
		}
		if err != sql.ErrNoRows {
			return nil, err
		}
		// 区分"不存在/跨租户"与"已处理"：读取必须使用同一 tenant-scoped Tx。
		var curStatus string
		_ = tx.QueryRowContext(ctx,
			`SELECT status FROM change_approvals WHERE id = $1 AND tenant_id = $2`,
			rec.ID, rec.TenantID,
		).Scan(&curStatus)
		if curStatus != "" {
			return nil, fmt.Errorf("审批记录已处理（当前状态=%s），不可重复审批", curStatus)
		}
		return nil, fmt.Errorf("审批记录不存在或跨租户")
	})
}

// ListByChange 返回某变更的全部审批记录（按 created_at 升序），
// 并由调用方 getApprovalChainLevels 补全 Levels 字段。
//
// 返回空切片而非 nil：避免 JSON 序列化为 null 导致前端崩溃。
func (r *changeApprovalRecordRepository) ListByChange(ctx context.Context, changeID, tenantID int) ([]*ApprovalRecord, error) {
	const query = `
		SELECT a.id, a.approver_id, u.name as approver_name, a.status, a.comment, a.approved_at, a.created_at
		FROM change_approvals a
		LEFT JOIN users u ON a.approver_id = u.id
		LEFT JOIN changes c ON a.change_id = c.id
		WHERE a.change_id = $1 AND a.tenant_id = $2 AND c.tenant_id = $2
		ORDER BY a.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, changeID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*ApprovalRecord, 0)
	for rows.Next() {
		var rec ApprovalRecord
		var approvedAt sql.NullTime
		if err := rows.Scan(&rec.ID, &rec.ApproverID, &rec.ApproverName, &rec.Status, &rec.Comment, &approvedAt, &rec.CreatedAt); err != nil {
			return nil, err
		}
		if approvedAt.Valid {
			rec.ApprovedAt = &approvedAt.Time
		}
		rec.ChangeID = changeID
		rec.TenantID = tenantID
		records = append(records, &rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}
