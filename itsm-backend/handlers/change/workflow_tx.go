package change

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"itsm-backend/ent"

	entsql "entgo.io/ent/dialect/sql"
)

// errRawTxUnavailable 表示当前事务句柄不支持原生 SQL（纯 ent 事务，
// 只出现在未装配 *sql.DB 的测试/legacy 装配里）。
var errRawTxUnavailable = errors.New("workflow tx: raw sql unavailable")

// WorkflowTx 是变更审批链路的统一事务句柄：把「BPMN 桥接（ent）」「变更状态 CAS」
// 「change_approval_chains 收口（原生 SQL）」放进同一个数据库事务。
//
// 为什么需要它：change_approval_chains 没有 ent schema，只能用原生 SQL 更新；
// 而桥接与状态推进走 ent。此前终态流转把审批链收口放在 Commit 之后、用全局
// database.GetRawDB() 执行，失败只记日志 —— 于是会出现「变更已是 rejected/
// cancelled，审批链却仍是 pending」的半写状态，审批人继续在待办里看到该单。
// 收敛到同一事务后，任一步失败整体回滚，不存在"状态已终态但链未收口"。
type WorkflowTx struct {
	// sqlTx 为原始事务（支持 ExecContext）；由 BeginWorkflowTx 创建。
	sqlTx *sql.Tx
	// entTx 仅在退化为纯 ent 事务时使用（外层没有 *sql.DB 句柄）。
	entTx  *ent.Tx
	client *ent.Client
}

// Client 返回绑定当前事务的 ent 客户端。事务内的所有 ent 写都必须走它，
// 否则写入会落到事务之外，回滚时不被撤销。
func (t *WorkflowTx) Client() *ent.Client { return t.client }

// Commit 提交事务。
func (t *WorkflowTx) Commit() error {
	if t.sqlTx != nil {
		return t.sqlTx.Commit()
	}
	if t.entTx != nil {
		return t.entTx.Commit()
	}
	return nil
}

// Rollback 回滚事务；已提交后调用返回 sql.ErrTxDone，调用方按 defer 忽略即可。
func (t *WorkflowTx) Rollback() error {
	if t.sqlTx != nil {
		return t.sqlTx.Rollback()
	}
	if t.entTx != nil {
		return t.entTx.Rollback()
	}
	return nil
}

// ClosePendingApprovalChains 在同一事务内把残留 pending 审批链节点置为 obsolete。
//
// 只有拿到原生事务句柄时才能执行；纯 ent 事务（测试/legacy 装配）返回
// errRawTxUnavailable，由调用方退化为提交后收口并记日志。
func (t *WorkflowTx) ClosePendingApprovalChains(ctx context.Context, changeID, tenantID int) error {
	if t.sqlTx == nil {
		return errRawTxUnavailable
	}
	const query = `
		UPDATE change_approval_chains
		SET status = 'obsolete'
		WHERE change_id = $1 AND tenant_id = $2 AND status = 'pending'
	`
	_, err := t.sqlTx.ExecContext(ctx, query, changeID, tenantID)
	return err
}

// BeginWorkflowTx 开启原始事务，并返回绑定该事务的 ent 客户端。
// ent 客户端通过 nopTxDriver 接入（与 SubmitForApprovalWithWorkflow 同一手法）：
// 内层 ent 的 Commit/Rollback 是空操作，事务生命周期由调用方独占管理。
func (r *EntRepository) BeginWorkflowTx(ctx context.Context) (*WorkflowTx, error) {
	if r.db == nil {
		return nil, fmt.Errorf("change workflow transaction database is unavailable")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin change workflow tx: %w", err)
	}
	dialectName, err := sqlDialect(r.db)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	client := ent.NewClient(ent.Driver(&nopTxDriver{
		drv: entsql.NewDriver(dialectName, entsql.Conn{ExecQuerier: tx}),
	}))
	return &WorkflowTx{sqlTx: tx, client: client}, nil
}

// wrapEntTx 把纯 ent 事务包装成 WorkflowTx（无原生 SQL 能力）。
// 仅供未装配 *sql.DB 的装配复用既有事务边界，不提供审批链收口的原子性。
func wrapEntTx(tx *ent.Tx) *WorkflowTx {
	if tx == nil {
		return nil
	}
	return &WorkflowTx{entTx: tx, client: tx.Client()}
}

// workflowTxStarter 由具备原生事务能力的仓储实现（EntRepository）。
// 用可选接口而不是加进 Repository 接口：mock 仓储（单测）没有 *sql.DB，
// 强行加入会让所有既有 mock 编译失败且语义上并不成立。
type workflowTxStarter interface {
	BeginWorkflowTx(ctx context.Context) (*WorkflowTx, error)
}
