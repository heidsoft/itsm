package cmdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DiscoveryScopeState 是 discovery_scope_states 的一行：云发现每个
// (source, serviceCode, region) scope 的覆盖水位线状态。语义见迁移
// migrations/20261107_discovery_scope_states.sql（蓝图 §3.2-17：每个 scope
// 独立保存 coverage start/end、分页完成证明与 provider watermark）。
type DiscoveryScopeState struct {
	TenantID            int
	SourceID            string
	ServiceCode         string
	Region              string
	ProviderWatermark   string     // 未完成翻页的 nextToken；窗口封口即清空
	CoverageStart       *time.Time // 当前覆盖窗口起点（跨尝试续窗，不得伪装新快照）
	PaginationCompleted bool       // 当前窗口的完整覆盖证明
	LastFullCompletedAt *time.Time // 水位线：上次完整覆盖完成时刻
	ConsecutiveFailures int
	LastErrorCode       string
	UpdatedAt           time.Time
}

// ScopeStateRepository 管理覆盖水位线状态。不走 Ent：本机 Ent codegen 不可用
// （与 cloudsignal 的 syncStateRepository 同因），表由版本化迁移承载。
type ScopeStateRepository interface {
	// Load 返回该 scope 的水位线状态；从未开窗时返回零值状态（不是错误）。
	Load(ctx context.Context, tenantID int, sourceID, serviceCode, region string) (*DiscoveryScopeState, error)
	// Save 幂等 upsert 全字段（执行器改完整体提交，不做局部更新）。
	Save(ctx context.Context, st *DiscoveryScopeState) error
}

type sqlScopeStateRepository struct {
	db *sql.DB
}

// NewScopeStateRepository 创建水位线仓储。db 允许 nil（未装配时执行器降级为
// 无水位线的每轮新窗，语义与旧实现一致）。
func NewScopeStateRepository(db *sql.DB) ScopeStateRepository {
	if db == nil {
		return nil
	}
	return &sqlScopeStateRepository{db: db}
}

const scopeStateColumns = `tenant_id, source_id, service_code, region,
	provider_watermark, coverage_start, pagination_completed, last_full_completed_at,
	consecutive_failures, last_error_code, updated_at`

func (r *sqlScopeStateRepository) Load(ctx context.Context, tenantID int, sourceID, serviceCode, region string) (*DiscoveryScopeState, error) {
	st := &DiscoveryScopeState{
		TenantID:    tenantID,
		SourceID:    sourceID,
		ServiceCode: serviceCode,
		Region:      region,
	}
	var coverageStart, lastFull, updatedAt string
	err := r.db.QueryRowContext(ctx, `SELECT `+scopeStateColumns+`
		FROM discovery_scope_states
		WHERE tenant_id = $1 AND source_id = $2 AND service_code = $3 AND region = $4`,
		tenantID, sourceID, serviceCode, region,
	).Scan(&st.TenantID, &st.SourceID, &st.ServiceCode, &st.Region,
		&st.ProviderWatermark, &coverageStart, &st.PaginationCompleted, &lastFull,
		&st.ConsecutiveFailures, &st.LastErrorCode, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return st, nil
		}
		return nil, fmt.Errorf("load discovery scope state: %w", err)
	}
	st.CoverageStart = parseScopeTime(coverageStart)
	st.LastFullCompletedAt = parseScopeTime(lastFull)
	st.UpdatedAt = time.Time{}
	if t := parseScopeTime(updatedAt); t != nil {
		st.UpdatedAt = *t
	}
	return st, nil
}

func (r *sqlScopeStateRepository) Save(ctx context.Context, st *DiscoveryScopeState) error {
	if st == nil {
		return fmt.Errorf("save discovery scope state: nil state")
	}
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `INSERT INTO discovery_scope_states
		(tenant_id, source_id, service_code, region, provider_watermark, coverage_start,
		 pagination_completed, last_full_completed_at, consecutive_failures, last_error_code,
		 created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
		ON CONFLICT (tenant_id, source_id, service_code, region) DO UPDATE SET
			provider_watermark = EXCLUDED.provider_watermark,
			coverage_start = EXCLUDED.coverage_start,
			pagination_completed = EXCLUDED.pagination_completed,
			last_full_completed_at = EXCLUDED.last_full_completed_at,
			consecutive_failures = EXCLUDED.consecutive_failures,
			last_error_code = EXCLUDED.last_error_code,
			updated_at = EXCLUDED.updated_at`,
		st.TenantID, st.SourceID, st.ServiceCode, st.Region, st.ProviderWatermark,
		formatScopeTime(st.CoverageStart), st.PaginationCompleted, formatScopeTime(st.LastFullCompletedAt),
		st.ConsecutiveFailures, st.LastErrorCode, now.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("save discovery scope state: %w", err)
	}
	st.UpdatedAt = now
	return nil
}

// formatScopeTime 把可空时间写成 RFC3339 文本；nil 写空串（列 NOT NULL DEFAULT ”）。
// 文本存储跨 PG/SQLite 逐字节一致，水位线比较在 Go 侧完成。
func formatScopeTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func parseScopeTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
