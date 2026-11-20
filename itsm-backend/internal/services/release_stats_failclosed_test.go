package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// entsqlDriver 把 sqlmock 的 *sql.DB 包装成 Ent dialect.Driver，
// 用于在不启动真实数据库的前提下注入单计数故障。
func entsqlDriver(db *sql.DB) dialect.Driver { return entsql.OpenDB(dialect.SQLite, db) }

// statsCountSpec 描述一次发布计数查询的可区分指纹：
// total 只带 tenant 一个参数，分状态查询多带一个 status 参数。用参数签名而非
// SQL 文本匹配，避免测试绑死 Ent 生成的 SQL 形状。
type statsCountSpec struct {
	column string // 计数返回列名，用于逐字段对位断言
	status string // 空串代表 total
	value  int
	fail   bool
}

var allStatsSpecs = []statsCountSpec{
	{column: "total", value: 0},
	{column: "draft", status: string(dto.ReleaseStatusDraft), value: 0},
	{column: "scheduled", status: string(dto.ReleaseStatusScheduled), value: 0},
	{column: "in_progress", status: string(dto.ReleaseStatusInProgress), value: 0},
	{column: "completed", status: string(dto.ReleaseStatusCompleted), value: 0},
	{column: "cancelled", status: string(dto.ReleaseStatusCancelled), value: 0},
	{column: "failed", status: string(dto.ReleaseStatusFailed), value: 0},
	{column: "rolled_back", status: string(dto.ReleaseStatusRolledBack), value: 0},
}

// expectStatsCounts 按 spec 登记 8 个 COUNT 期望。MatchExpectationsInOrder(false)
// 让匹配只依据参数签名，计数顺序调整不会造成假红。
func expectStatsCounts(mock sqlmock.Sqlmock, specs []statsCountSpec) {
	mock.MatchExpectationsInOrder(false)
	for _, sp := range specs {
		var exp *sqlmock.ExpectedQuery
		if sp.status == "" {
			exp = mock.ExpectQuery(`SELECT COUNT`).WithArgs(sqlmock.AnyArg())
		} else {
			exp = mock.ExpectQuery(`SELECT COUNT`).WithArgs(sqlmock.AnyArg(), sp.status)
		}
		if sp.fail {
			exp.WillReturnError(errors.New("injected count failure for " + sp.column))
		} else {
			exp.WillReturnRows(sqlmock.NewRows([]string{sp.column}).AddRow(sp.value))
		}
	}
}

// TestGetReleaseStats_FailClosedOnStatusCountError 是「撤掉守卫即失败」的回归锁：
// 旧实现用 `draft, _ := ...Count(ctx)` 吞掉 6 个状态计数错误，DB 故障时返回
// 一个 total 有值、分状态全 0 的假健康统计。这里对 7 个状态逐一注入单点计数
// 失败，断言 GetReleaseStats 必须整体返回错误且 stats == nil。
// 任一处守卫被改回吞错，对应子用例立即变红。
func TestGetReleaseStats_FailClosedOnStatusCountError(t *testing.T) {
	statuses := []struct {
		status string
		name   string
	}{
		{string(dto.ReleaseStatusDraft), "草稿"},
		{string(dto.ReleaseStatusScheduled), "已计划"},
		{string(dto.ReleaseStatusInProgress), "进行中"},
		{string(dto.ReleaseStatusCompleted), "已完成"},
		{string(dto.ReleaseStatusCancelled), "已取消"},
		{string(dto.ReleaseStatusFailed), "失败"},
		{string(dto.ReleaseStatusRolledBack), "已回滚"},
	}
	for _, tc := range statuses {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })

			specs := make([]statsCountSpec, len(allStatsSpecs))
			copy(specs, allStatsSpecs)
			for i := range specs {
				specs[i].fail = specs[i].status == tc.status
			}
			expectStatsCounts(mock, specs)

			client := ent.NewClient(ent.Driver(entsqlDriver(db)))
			defer client.Close()
			svc := NewReleaseService(client, zap.NewNop().Sugar())

			// 找到被注入故障那一维的列名，用于校验底层错误原样上抛。
			failColumn := ""
			for _, sp := range specs {
				if sp.fail {
					failColumn = sp.column
				}
			}
			require.NotEmpty(t, failColumn)
			stats, err := svc.GetReleaseStats(context.Background(), 1)
			require.Error(t, err, "%s 计数失败必须 fail-closed", tc.status)
			assert.Nil(t, stats, "统计失败不得返回半真半假的 stats")
			assert.Contains(t, err.Error(), "统计发布状态 "+tc.status+" 失败",
				"错误必须带失败状态标签，便于从日志定位")
			assert.Contains(t, err.Error(), "injected count failure for "+failColumn,
				"底层 DB 错误必须原样上抛，不能被包装吞掉")
		})
	}
}

// TestGetReleaseStats_FailClosedOnTotalCountError 锁定 total 计数的 fail-closed
// 语义：total 是看板的可信锚点，DB 失败时同样必须整体报错。
func TestGetReleaseStats_FailClosedOnTotalCountError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	specs := make([]statsCountSpec, len(allStatsSpecs))
	copy(specs, allStatsSpecs)
	specs[0].fail = true
	expectStatsCounts(mock, specs)

	client := ent.NewClient(ent.Driver(entsqlDriver(db)))
	defer client.Close()
	svc := NewReleaseService(client, zap.NewNop().Sugar())

	stats, err := svc.GetReleaseStats(context.Background(), 1)
	require.Error(t, err)
	assert.Nil(t, stats)
	assert.Contains(t, err.Error(), "统计发布状态 all 失败")
}

// TestGetReleaseStats_SuccessPath 确认 fail-closed 守卫没有破坏正常统计口径：
// 8 个计数各自返回不同值，响应必须逐字段对位（防止张冠李戴式回归）。
func TestGetReleaseStats_SuccessPath(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	specs := []statsCountSpec{
		{column: "total", value: 100},
		{column: "draft", status: string(dto.ReleaseStatusDraft), value: 10},
		{column: "scheduled", status: string(dto.ReleaseStatusScheduled), value: 20},
		{column: "in_progress", status: string(dto.ReleaseStatusInProgress), value: 30},
		{column: "completed", status: string(dto.ReleaseStatusCompleted), value: 15},
		{column: "cancelled", status: string(dto.ReleaseStatusCancelled), value: 5},
		{column: "failed", status: string(dto.ReleaseStatusFailed), value: 8},
		{column: "rolled_back", status: string(dto.ReleaseStatusRolledBack), value: 12},
	}
	expectStatsCounts(mock, specs)

	client := ent.NewClient(ent.Driver(entsqlDriver(db)))
	defer client.Close()
	svc := NewReleaseService(client, zap.NewNop().Sugar())

	got, err := svc.GetReleaseStats(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, dto.ReleaseStatsResponse{
		Total: 100, Draft: 10, Scheduled: 20, InProgress: 30,
		Completed: 15, Cancelled: 5, Failed: 8, RolledBack: 12,
	}, *got, "8 个状态计数必须逐字段对位，不能张冠李戴")
	assert.NoError(t, mock.ExpectationsWereMet(), "8 个 COUNT 必须全部真实发出")
}
