package ent

import (
	"context"
	"database/sql"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
)

// ExecContext 在事务连接上执行原生 SQL。
//
// 手写而非生成：项目未启用 ent 的 sql/execquery 特性，且本地代码生成会重写
// 整个 ent/ 目录（见 AGENTS 约定），无法只为这一个方法重新生成。
// 实现依赖生成代码 txDriver.tx 的 *entsql.Tx 形态；无法识别时返回错误，
// 不做静默降级，避免调用方误以为语句已在事务内执行。
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if tx == nil {
		return nil, fmt.Errorf("ent: ExecContext requires a non-nil transaction")
	}
	driver, ok := tx.config.driver.(*txDriver)
	if !ok {
		return nil, fmt.Errorf("ent: ExecContext requires a transactional client")
	}
	sqlTx, ok := driver.tx.(*entsql.Tx)
	if !ok {
		return nil, fmt.Errorf("ent: ExecContext unsupported for transaction driver %T", driver.tx)
	}
	return sqlTx.ExecContext(ctx, query, args...)
}
