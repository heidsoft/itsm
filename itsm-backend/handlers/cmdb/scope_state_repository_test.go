package cmdb

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// 列与 migrations/20261107_discovery_scope_states.sql 一一对应（仅去掉 PG 专属的
// RLS）。两处漂移会让「测试绿、生产红」，改列必须同批改这里。
const scopeStateSQLiteDDL = `
CREATE TABLE IF NOT EXISTS discovery_scope_states (
    tenant_id              BIGINT NOT NULL,
    source_id              VARCHAR(128) NOT NULL,
    service_code           VARCHAR(64) NOT NULL,
    region                 VARCHAR(64) NOT NULL DEFAULT '',
    provider_watermark     TEXT NOT NULL DEFAULT '',
    coverage_start         TEXT NOT NULL DEFAULT '',
    pagination_completed   BOOLEAN NOT NULL DEFAULT FALSE,
    last_full_completed_at TEXT NOT NULL DEFAULT '',
    consecutive_failures   INT NOT NULL DEFAULT 0,
    last_error_code        VARCHAR(128) NOT NULL DEFAULT '',
    created_at             TEXT NOT NULL DEFAULT '',
    updated_at             TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (tenant_id, source_id, service_code, region)
);`

func newScopeStateTestRepo(t *testing.T, name string) (ScopeStateRepository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:scope_state_%s?mode=memory&cache=shared&_fk=1", name))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	_, err = db.ExecContext(context.Background(), scopeStateSQLiteDDL)
	require.NoError(t, err)
	return NewScopeStateRepository(db), db
}

// TestScopeState_LoadMissingReturnsZeroState 从未开窗的 scope 必须返回零值状态
// 而不是错误——执行器据此开新窗。
func TestScopeState_LoadMissingReturnsZeroState(t *testing.T) {
	repo, _ := newScopeStateTestRepo(t, "missing")
	st, err := repo.Load(context.Background(), 1, "ds-1", "ecs", "cn-hangzhou")
	require.NoError(t, err)
	assert.Equal(t, "", st.ProviderWatermark)
	assert.Nil(t, st.CoverageStart)
	assert.False(t, st.PaginationCompleted)
	assert.Nil(t, st.LastFullCompletedAt)
}

// TestScopeState_SaveLoadRoundTrip 水位线全字段必须逐字段保真（含 nil/非 nil
// 时间指针的区分）。
func TestScopeState_SaveLoadRoundTrip(t *testing.T) {
	repo, _ := newScopeStateTestRepo(t, "roundtrip")
	ctx := context.Background()
	start := time.Date(2026, 11, 7, 2, 0, 0, 0, time.UTC)
	done := start.Add(30 * time.Minute)

	st := &DiscoveryScopeState{
		TenantID:            7,
		SourceID:            "ds-rt",
		ServiceCode:         "ecs",
		Region:              "cn-hangzhou",
		ProviderWatermark:   "token-page-2",
		CoverageStart:       &start,
		PaginationCompleted: false,
		LastFullCompletedAt: nil,
		ConsecutiveFailures: 2,
		LastErrorCode:       "discover_region_failed",
	}
	require.NoError(t, repo.Save(ctx, st))

	got, err := repo.Load(ctx, 7, "ds-rt", "ecs", "cn-hangzhou")
	require.NoError(t, err)
	assert.Equal(t, "token-page-2", got.ProviderWatermark)
	require.NotNil(t, got.CoverageStart)
	assert.True(t, got.CoverageStart.Equal(start), "coverage_start 必须保真: %v != %v", got.CoverageStart, start)
	assert.False(t, got.PaginationCompleted)
	assert.Nil(t, got.LastFullCompletedAt)
	assert.Equal(t, 2, got.ConsecutiveFailures)
	assert.Equal(t, "discover_region_failed", got.LastErrorCode)

	// 封窗：清水位线、置完成证明、落水位线时刻。
	got.ProviderWatermark = ""
	got.PaginationCompleted = true
	got.LastFullCompletedAt = &done
	got.ConsecutiveFailures = 0
	got.LastErrorCode = ""
	require.NoError(t, repo.Save(ctx, got))

	got2, err := repo.Load(ctx, 7, "ds-rt", "ecs", "cn-hangzhou")
	require.NoError(t, err)
	assert.Equal(t, "", got2.ProviderWatermark)
	assert.True(t, got2.PaginationCompleted)
	require.NotNil(t, got2.LastFullCompletedAt)
	assert.True(t, got2.LastFullCompletedAt.Equal(done))
	assert.Equal(t, 0, got2.ConsecutiveFailures)
}

// TestScopeState_UpsertAndTenantIsolation 同键 upsert 只留一行；不同租户同键
// 各自独立（水位线是租户内状态）。
func TestScopeState_UpsertAndTenantIsolation(t *testing.T) {
	repo, db := newScopeStateTestRepo(t, "upsert")
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Save(ctx, &DiscoveryScopeState{
			TenantID: 1, SourceID: "ds-1", ServiceCode: "ecs", Region: "r1",
			ProviderWatermark: fmt.Sprintf("token-%d", i),
		}))
	}
	require.NoError(t, repo.Save(ctx, &DiscoveryScopeState{
		TenantID: 2, SourceID: "ds-1", ServiceCode: "ecs", Region: "r1",
		ProviderWatermark: "tenant-2-token",
	}))

	var rows int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM discovery_scope_states`).Scan(&rows))
	assert.Equal(t, 2, rows, "同键 upsert 不得累积行")

	got1, err := repo.Load(ctx, 1, "ds-1", "ecs", "r1")
	require.NoError(t, err)
	assert.Equal(t, "token-2", got1.ProviderWatermark)
	got2, err := repo.Load(ctx, 2, "ds-1", "ecs", "r1")
	require.NoError(t, err)
	assert.Equal(t, "tenant-2-token", got2.ProviderWatermark)
}
