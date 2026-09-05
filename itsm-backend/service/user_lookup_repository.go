package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// userLookupRepository 通用用户姓名查询仓储。
//
// 设计目的：
//   - 把 service 层对 users 表的直接 SELECT 收敛到一处。
//   - 强制 tenant_id 过滤，避免跨租户读取用户信息。
//   - 用单独文件避免循环依赖：problem_investigation_service 不直接 import 业务 user repository。
//
// 当前被 problem_investigation_service 使用，其它模块可以按需复用。
type userLookupRepository struct {
	db *sql.DB
}

func newUserLookupRepository(db *sql.DB) *userLookupRepository {
	if db == nil {
		return nil
	}
	return &userLookupRepository{db: db}
}

// GetUserName 按 (id, tenant_id) 查询用户姓名。
//
// 返回 (name, found, error)：
//   - found=true, err=nil：找到
//   - found=false, err=nil：未找到（不视为错误）
//   - found=false, err!=nil：DB 异常
func (r *userLookupRepository) GetUserName(ctx context.Context, userID, tenantID int) (string, bool, error) {
	if r == nil || r.db == nil {
		return "", false, errors.New("user lookup repository not initialised")
	}
	const query = `SELECT name FROM users WHERE id = $1 AND tenant_id = $2`
	var name string
	if err := r.db.QueryRowContext(ctx, query, userID, tenantID).Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("get user name: %w", err)
	}
	return name, true, nil
}
