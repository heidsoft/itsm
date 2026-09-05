package ticket

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"itsm-backend/database"
)

// ticketNumberRepository 负责工单编号生成相关的"必须 raw SQL"操作。
//
// 编号生成的并发竞态需要：
//  1. 在事务内对"本月已有最大编号"行加 FOR UPDATE NOWAIT 锁
//  2. 计算下一个候选编号后提交事务释放锁
//  3. 再做一次唯一性 COUNT(*) 兜底（防止极端的锁失效/手工干预）
//
// 这些都是 PostgreSQL / SQL 语法层操作，没有 Ent 等价表达，集中在此便于未来
// 迁移序列或换方言时只改一处。
type ticketNumberRepository struct {
	db *sql.DB
}

func newTicketNumberRepository(db *sql.DB) *ticketNumberRepository {
	if db == nil {
		return nil
	}
	return &ticketNumberRepository{db: db}
}

// beginLockedLookup 开启一个事务并加 FOR UPDATE NOWAIT 锁，返回 tx 与回退函数。
// 回退函数负责：未提交时回滚；ErrNoRows 时回滚后返回 nil；其它错误回滚后包装返回。
//
// 使用方调用顺序：
//
//	tx, rollback := r.beginLockedLookup(ctx, tenantID, prefix+"%")
//	defer rollback(&err)
//	if err := tx.QueryRowContext(...).Scan(&maxNum); err != nil { ... }
//	if err := tx.Commit(); err != nil { ... }
func (r *ticketNumberRepository) beginLockedLookup(ctx context.Context, tenantID int, likePattern string) (*sql.Tx, func(*error), error) {
	if r == nil || r.db == nil {
		return nil, func(*error) {}, errors.New("ticket number repository not initialised")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, func(*error) {}, fmt.Errorf("begin tx: %w", err)
	}
	rollback := func(errp *error) {
		if errp != nil && *errp != nil {
			_ = tx.Rollback()
			return
		}
		// err==nil 表示调用方已 Commit，无需 Rollback；调用方负责 Commit
	}
	return tx, rollback, nil
}

// queryMaxLocked 在已开启的事务内查询"本月已有最大工单号"，同时加 FOR UPDATE NOWAIT。
// 查询不到（ErrNoRows）返回空串且不视为错误；其它错误向上抛出。
func (r *ticketNumberRepository) queryMaxLocked(ctx context.Context, tx *sql.Tx, tenantID int, likePattern string) (string, error) {
	const query = `
		SELECT ticket_number
		FROM tickets
		WHERE tenant_id = $1
				AND ticket_number LIKE $2
				AND ticket_number IS NOT NULL
				AND ticket_number != ''
		ORDER BY ticket_number DESC
		LIMIT 1
		FOR UPDATE NOWAIT
	`
	var maxTicketNum string
	if err := tx.QueryRowContext(ctx, query, tenantID, likePattern).Scan(&maxTicketNum); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("locked max query: %w", err)
	}
	return maxTicketNum, nil
}

// existsNumber 检查指定编号在该租户下是否已被使用。用于事务提交后的"双重保险"。
func (r *ticketNumberRepository) existsNumber(ctx context.Context, tenantID int, candidate string) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("ticket number repository not initialised")
	}
	const query = `SELECT COUNT(*) FROM tickets WHERE ticket_number = $1 AND tenant_id = $2`
	count, err := database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (int, error) {
		var count int
		if err := q.QueryRowContext(ctx, query, candidate, tenantID).Scan(&count); err != nil {
			return 0, err
		}
		return count, nil
	})
	if err != nil {
		return false, fmt.Errorf("exists check: %w", err)
	}
	return count > 0, nil
}

// parseSequenceSuffix 从完整工单号中抽取末尾数字后缀。
// 形如 TKT-202509-000042 → 42；解析失败返回 0。
func parseSequenceSuffix(ticketNumber string) int {
	if ticketNumber == "" {
		return 0
	}
	idx := strings.LastIndex(ticketNumber, "-")
	if idx < 0 {
		return 0
	}
	var seq int
	fmt.Sscanf(ticketNumber[idx+1:], "%d", &seq)
	return seq
}
