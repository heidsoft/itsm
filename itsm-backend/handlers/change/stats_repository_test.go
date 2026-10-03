package change

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createStatsChange 写入一条最小可变更。status 必须是 Ent 枚举内的值，
// 枚举外的存量值请用 seedLegacyStatus 还原。
func createStatsChange(t *testing.T, client *ent.Client, tenantID int, status, changeType, title string) int {
	t.Helper()
	created, err := client.Change.Create().
		SetTitle(title).
		SetDescription("统计回归数据").
		SetType(changeType).
		SetStatus(change.Status(status)).
		SetPriority("medium").
		SetImpactScope("medium").
		SetRiskLevel("low").
		SetCreatedBy(1).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return created.ID
}

// seedLegacyStatus 用 raw SQL 把状态改成枚举里没有的历史值。
// Ent 的 StatusValidator 会拒绝这类写入，而它们确实存在于状态枚举化之前
// 写入的存量库中，所以统计口径必须继续把它们算进来。
func seedLegacyStatus(t *testing.T, db *sql.DB, changeID int, status string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `UPDATE changes SET status = ? WHERE id = ?`, status, changeID)
	require.NoError(t, err)
}

func openStatsTestDB(t *testing.T) (*ent.Client, *sql.DB) {
	t.Helper()
	// 每个用例独立库名，避免共享缓存在同一次 go test 进程里串数据。
	dsn := fmt.Sprintf("file:change-stats-%s?mode=memory&cache=shared&_fk=1", t.Name())
	client := enttest.Open(t, "sqlite3", dsn)
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
		client.Close()
	})
	return client, db
}

// TestStatsRepository_StatusBuckets 锁定统计契约的三个不变量：
//  1. 每个枚举内状态都有自己的桶（draft/closed 曾因 DTO 缺字段被静默丢数）；
//  2. 枚举外的存量值（pending_review/submitted）折叠进 pending，
//     与"按待审批过滤"的读规则一致；
//  3. 各桶之和 == total，否则报表卡片与总数永远对不上账。
func TestStatsRepository_StatusBuckets(t *testing.T) {
	client, db := openStatsTestDB(t)
	ctx := context.Background()

	enumerated := []string{
		"draft", "pending", "approved", "scheduled",
		"in_progress", "completed", "failed", "rolled_back",
		"rejected", "cancelled", "closed",
	}
	for _, status := range enumerated {
		createStatsChange(t, client, 1, status, "normal", "统计 "+status)
	}
	legacyPendingReview := createStatsChange(t, client, 1, "pending", "standard", "统计 pending_review 存量")
	legacySubmitted := createStatsChange(t, client, 1, "approved", "emergency", "统计 submitted 存量")
	seedLegacyStatus(t, db, legacyPendingReview, "pending_review")
	seedLegacyStatus(t, db, legacySubmitted, "submitted")

	// 租户 2 的数据不得进入租户 1 的统计。
	createStatsChange(t, client, 2, "completed", "emergency", "别的租户")

	stats, err := NewEntRepository(client, nil).GetStats(ctx, 1)
	require.NoError(t, err)

	assert.Equal(t, 13, stats.Total)
	assert.Equal(t, 1, stats.Draft, "draft 曾因 toStatsDTO 未映射而恒为 0")
	assert.Equal(t, 3, stats.Pending, "pending 必须含 pending_review 与存量 submitted")
	assert.Equal(t, 1, stats.Approved)
	assert.Equal(t, 1, stats.Scheduled)
	assert.Equal(t, 1, stats.InProgress)
	assert.Equal(t, 1, stats.Completed)
	assert.Equal(t, 1, stats.Failed)
	assert.Equal(t, 1, stats.RolledBack)
	assert.Equal(t, 1, stats.Rejected)
	assert.Equal(t, 1, stats.Cancelled)
	assert.Equal(t, 1, stats.Closed, "closed 曾因没有桶而只出现在 total 里")

	sum := stats.Draft + stats.Pending + stats.Approved + stats.Scheduled + stats.InProgress +
		stats.Completed + stats.Failed + stats.RolledBack + stats.Rejected + stats.Cancelled + stats.Closed
	assert.Equal(t, stats.Total, sum, "状态桶必须与 total 对账，不得有状态被静默丢弃")

	tenantSum := func() int {
		s, err := NewEntRepository(client, nil).GetStats(ctx, 2)
		require.NoError(t, err)
		return s.Total
	}
	assert.Equal(t, 1, tenantSum(), "租户 2 只应看到自己的 1 条")
}

// TestStatsRepository_ByTypeIsRealAggregation 类型分布必须来自真实 GROUP BY，
// 顺序固定可重放；/reports/change-success 此前用 total 的 30/50/20 伪造过这张图。
func TestStatsRepository_ByTypeIsRealAggregation(t *testing.T) {
	client, _ := openStatsTestDB(t)

	createStatsChange(t, client, 1, "draft", "standard", "标准 1")
	createStatsChange(t, client, 1, "pending", "standard", "标准 2")
	createStatsChange(t, client, 1, "completed", "normal", "普通 1")
	createStatsChange(t, client, 1, "cancelled", "legacy_widget", "枚举外的历史类型")

	stats, err := NewEntRepository(client, nil).GetStats(context.Background(), 1)
	require.NoError(t, err)

	assert.Equal(t, []TypeCount{
		{Type: "standard", Count: 2},
		{Type: "normal", Count: 1},
		{Type: "legacy_widget", Count: 1},
	}, stats.ByType, "已知类型按 canonical 顺序，枚举外值追加在尾部而不是丢弃")

	typeSum := 0
	for _, c := range stats.ByType {
		typeSum += c.Count
	}
	assert.Equal(t, stats.Total, typeSum, "类型分布必须与 total 对账")
}

// TestStatsRepository_RequiresTenantContext 缺租户上下文必须 fail closed，
// 不得退化成全表统计。
func TestStatsRepository_RequiresTenantContext(t *testing.T) {
	client, _ := openStatsTestDB(t)
	createStatsChange(t, client, 1, "draft", "normal", "租户 1 数据")

	repo := NewEntRepository(client, nil)
	for _, tenantID := range []int{0, -1} {
		stats, err := repo.GetStats(context.Background(), tenantID)
		require.Error(t, err, "tenantID=%d", tenantID)
		assert.Nil(t, stats)
	}

	// 空租户（存在数据但查询另一个租户）返回零值而不是报错。
	stats, err := repo.GetStats(context.Background(), 99)
	require.NoError(t, err)
	assert.Equal(t, 0, stats.Total)
	assert.NotNil(t, stats.ByType)
	assert.Empty(t, stats.ByType)
}
