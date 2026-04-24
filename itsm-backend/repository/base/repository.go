// Package base 提供通用的 Repository 基础设施
// 实现统一的 CRUD 操作接口，减少重复代码
package base

import (
	"context"

	"itsm-backend/ent"
)

// QueryParams 通用查询参数
type QueryParams struct {
	Page     int
	PageSize int
	OrderBy  string
	OrderDir string // "asc" or "desc"
}

// ListResult 列表查询结果
type ListResult[T any] struct {
	Data  []*T
	Total int
}

// BaseRepository 通用 Repository 接口
// 所有实体 Repository 都应该嵌入此接口
type BaseRepository[T any] interface {
	// 基础 CRUD
	Create(ctx context.Context, entity *T) (*T, error)
	GetByID(ctx context.Context, id int, tenantID int) (*T, error)
	Update(ctx context.Context, entity *T) (*T, error)
	Delete(ctx context.Context, id int, tenantID int) error

	// 列表查询
	List(ctx context.Context, tenantID int, params QueryParams) (*ListResult[T], error)

	// 批量操作
	BatchDelete(ctx context.Context, ids []int, tenantID int) error
	Exists(ctx context.Context, id int, tenantID int) (bool, error)
}

// EntRepository Ent 实现的基类
// 提供通用的数据库操作方法
type EntRepository struct {
	client *ent.Client
}

// NewEntRepository 创建 Ent Repository 基类
func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

// Client 获取 Ent Client
func (r *EntRepository) Client() *ent.Client {
	return r.client
}

// WithTx 在事务中执行操作
func (r *EntRepository) WithTx(ctx context.Context, fn func(tx *ent.Tx) error) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if v := recover(); v != nil {
			_ = tx.Rollback()
			panic(v)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit()
}

// CalculateOffset 计算分页偏移量
func (params QueryParams) CalculateOffset() int {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	return (params.Page - 1) * params.PageSize
}

// GetLimit 获取分页大小
func (params QueryParams) GetLimit() int {
	if params.PageSize <= 0 {
		return 20
	}
	if params.PageSize > 100 {
		return 100 // 最大限制
	}
	return params.PageSize
}
