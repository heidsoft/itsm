package service

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sqlRecorder 收集 sqlmock 在每个用例里观察到的 QueryContext SQL。
// 用 QueryMatcherFunc 在匹配阶段把实际 SQL 拷贝进来，
// 避开 sqlmock 不公开"最近一次查询"这一限制。
type sqlRecorder struct {
	mu  sync.Mutex
	sql []string
}

func (r *sqlRecorder) record(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sql = append(r.sql, s)
}

func (r *sqlRecorder) last() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.sql) == 0 {
		return ""
	}
	return r.sql[len(r.sql)-1]
}

// newRepoMockDB 给 dashboardRepository 注入 sqlmock 控制的 *sql.DB。
// 返回 (db, mock, repo, recorder) 四元组：mock 用于声明 expect，repo 执行待测方法，
// recorder 用于断言实际下发的 SQL 文本。
func newRepoMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *dashboardRepository, *sqlRecorder) {
	t.Helper()
	rec := &sqlRecorder{}
	matcher := sqlmock.QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
		rec.record(actualSQL)
		return nil
	})
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock, newDashboardRepository(db), rec
}

// ==================== 仓储构造与 fail-closed 校验 ====================

func TestNewDashboardRepository_NilDBIsAllowed(t *testing.T) {
	repo := newDashboardRepository(nil)
	require.NotNil(t, repo, "构造器允许传入 nil db，由仓储方法内部 fail-closed")
	assert.Nil(t, repo.db)
}

func TestDashboardRepository_RequireDB(t *testing.T) {
	t.Run("nil db 返回明确错误而非 panic", func(t *testing.T) {
		repo := newDashboardRepository(nil)
		err := repo.requireDB()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "数据库连接未初始化")
	})

	t.Run("nil 仓储（未构造）调用方法也安全", func(t *testing.T) {
		var repo *dashboardRepository
		_, _, err := repo.AvgResponseAndResolutionHours(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "数据库连接未初始化")
	})

	t.Run("已注入 db 时不返回错误", func(t *testing.T) {
		db, _, repo, _ := newRepoMockDB(t)
		require.NotNil(t, db)
		assert.NoError(t, repo.requireDB())
	})
}

// ==================== AvgResponseAndResolutionHours ====================

func TestDashboardRepository_AvgResponseAndResolutionHours_ParamBinding(t *testing.T) {
	db, mock, repo, _ := newRepoMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"avg_resp", "avg_res"}).AddRow(2.5, 12.75)
	mock.ExpectQuery(`.+`).
		WithArgs(42).
		WillReturnRows(rows)

	gotResp, gotRes, err := repo.AvgResponseAndResolutionHours(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, 2.5, gotResp)
	assert.Equal(t, 12.75, gotRes)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardRepository_AvgResponseAndResolutionHours_TenantPredicatePresent(t *testing.T) {
	db, mock, repo, rec := newRepoMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"avg_resp", "avg_res"}).AddRow(0, 0)
	mock.ExpectQuery(`.+`).
		WithArgs(7).
		WillReturnRows(rows)

	_, _, err := repo.AvgResponseAndResolutionHours(context.Background(), 7)
	require.NoError(t, err)

	sql := rec.last()
	require.NotEmpty(t, sql, "必须至少触发一次 Query")

	assert.Regexp(t, regexp.MustCompile(`tenant_id\s*=\s*\$1`), sql,
		"SQL 必须按租户隔离：tenant_id = $1 必须出现在 WHERE 子句")
	assert.Contains(t, sql, "deleted_at IS NULL", "SQL 必须过滤软删除")
	assert.Contains(t, sql, "FROM tickets", "SQL 必须查 tickets 表")
	assert.Contains(t, sql, "FILTER (WHERE first_response_at IS NOT NULL)",
		"AVG 必须排除未首次响应的工单，避免污染均值")
	assert.Contains(t, sql, "FILTER (WHERE resolved_at IS NOT NULL)",
		"AVG 必须排除未解决的工单，避免污染均值")
}

func TestDashboardRepository_AvgResponseAndResolutionHours_RejectsInvalidTenant(t *testing.T) {
	db, _, repo, _ := newRepoMockDB(t)
	defer db.Close()

	for _, badID := range []int{0, -1, -100} {
		_, _, err := repo.AvgResponseAndResolutionHours(context.Background(), badID)
		require.Error(t, err, "tenantID=%d 应被拒绝", badID)
		assert.Contains(t, err.Error(), "tenantID 必须为正整数")
	}
}

func TestDashboardRepository_AvgResponseAndResolutionHours_ScanErrorPropagated(t *testing.T) {
	db, mock, repo, _ := newRepoMockDB(t)
	defer db.Close()

	mock.ExpectQuery(`.+`).
		WithArgs(1).
		WillReturnError(errors.New("数据库连接断开"))

	_, _, err := repo.AvgResponseAndResolutionHours(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "数据库连接断开")
}

// ==================== KPIAvgAndSLAScope ====================

func TestDashboardRepository_KPIAvgAndSLAScope_ParamBindingAndScanOrder(t *testing.T) {
	db, mock, repo, _ := newRepoMockDB(t)
	defer db.Close()

	thisMonth := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	lastMonth := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{
		"avg_resp_now", "avg_res_now", "avg_resp_prev", "avg_res_prev",
		"total_sla_tickets", "met_sla_tickets",
	}).AddRow(1.5, 8.0, 2.0, 10.0, 100, 95)
	mock.ExpectQuery(`.+`).
		WithArgs(99, thisMonth, lastMonth).
		WillReturnRows(rows)

	avgRespNow, avgResNow, avgRespPrev, avgResPrev, totalSLA, metSLA, err := repo.KPIAvgAndSLAScope(context.Background(), 99, thisMonth, lastMonth)
	require.NoError(t, err)
	assert.Equal(t, 1.5, avgRespNow)
	assert.Equal(t, 8.0, avgResNow)
	assert.Equal(t, 2.0, avgRespPrev)
	assert.Equal(t, 10.0, avgResPrev)
	assert.Equal(t, 100, totalSLA)
	assert.Equal(t, 95, metSLA)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardRepository_KPIAvgAndSLAScope_CTEStructure(t *testing.T) {
	db, mock, repo, rec := newRepoMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"a", "b", "c", "d", "e", "f"}).AddRow(0, 0, 0, 0, 0, 0)
	mock.ExpectQuery(`.+`).
		WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(rows)

	thisMonth := time.Now()
	lastMonth := thisMonth.AddDate(0, -1, 0)
	avgRespNow, avgResNow, avgRespPrev, avgResPrev, totalSLA, metSLA, err := repo.KPIAvgAndSLAScope(context.Background(), 1, thisMonth, lastMonth)
	require.NoError(t, err)
	_ = avgRespNow
	_ = avgResNow
	_ = avgRespPrev
	_ = avgResPrev
	_ = totalSLA
	_ = metSLA

	sql := rec.last()
	require.NotEmpty(t, sql)

	for _, cte := range []string{"current_month", "prev_month", "sla_scope"} {
		assert.Contains(t, strings.ToLower(sql), cte,
			"必须存在 CTE: %s", cte)
	}

	assert.Regexp(t, regexp.MustCompile(`sla_scope[\s\S]+created_at\s*>=\s*\$3`), sql,
		"sla_scope 应以 $3 (lastMonthStart) 为窗口起点")

	assert.GreaterOrEqual(t, strings.Count(sql, "tenant_id = $1"), 3,
		"三个 CTE 都必须按租户隔离")
}

func TestDashboardRepository_KPIAvgAndSLAScope_RejectsInvalidInputs(t *testing.T) {
	db, _, repo, _ := newRepoMockDB(t)
	defer db.Close()

	thisMonth := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	lastMonth := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	t.Run("非法 tenantID", func(t *testing.T) {
		_, _, _, _, _, _, err := repo.KPIAvgAndSLAScope(context.Background(), 0, thisMonth, lastMonth)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenantID")
	})

	t.Run("时间窗口为零值", func(t *testing.T) {
		_, _, _, _, _, _, err := repo.KPIAvgAndSLAScope(context.Background(), 1, time.Time{}, lastMonth)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "时间窗口")
	})

	t.Run("时间窗口顺序错误", func(t *testing.T) {
		_, _, _, _, _, _, err := repo.KPIAvgAndSLAScope(context.Background(), 1, lastMonth, thisMonth)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "必须早于")
	})
}

// ==================== PreviousSLAScopeCount ====================

func TestDashboardRepository_PreviousSLAScopeCount_ParamBindingAndScanOrder(t *testing.T) {
	db, mock, repo, _ := newRepoMockDB(t)
	defer db.Close()

	lastMonth := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	thisMonth := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"total", "met"}).AddRow(50, 45)
	mock.ExpectQuery(`.+`).
		WithArgs(11, lastMonth, thisMonth).
		WillReturnRows(rows)

	total, met, err := repo.PreviousSLAScopeCount(context.Background(), 11, lastMonth, thisMonth)
	require.NoError(t, err)
	assert.Equal(t, 50, total)
	assert.Equal(t, 45, met)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardRepository_PreviousSLAScopeCount_TenantAndWindowInvariant(t *testing.T) {
	db, mock, repo, rec := newRepoMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"total", "met"}).AddRow(0, 0)
	mock.ExpectQuery(`.+`).
		WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(rows)

	lastMonth := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	thisMonth := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	_, _, err := repo.PreviousSLAScopeCount(context.Background(), 1, lastMonth, thisMonth)
	require.NoError(t, err)

	sql := rec.last()
	require.NotEmpty(t, sql)

	assert.Regexp(t, regexp.MustCompile(`created_at\s*>=\s*\$2`), sql,
		"上月窗口起点：created_at >= $2 (lastMonthStart)")
	assert.Regexp(t, regexp.MustCompile(`created_at\s*<\s*\$3`), sql,
		"上月窗口上界：created_at < $3 (thisMonthStart)")

	assert.Contains(t, sql, "sla_response_deadline IS NULL",
		"未设置响应 SLA 的工单必须按达成处理")
	assert.Contains(t, sql, "sla_resolution_deadline IS NULL",
		"未设置解决 SLA 的工单必须按达成处理")

	assert.Regexp(t, regexp.MustCompile(`tenant_id\s*=\s*\$1`), sql,
		"必须按租户隔离")
}

func TestDashboardRepository_PreviousSLAScopeCount_RejectsInvalidInputs(t *testing.T) {
	db, _, repo, _ := newRepoMockDB(t)
	defer db.Close()

	thisMonth := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	lastMonth := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	t.Run("非法 tenantID", func(t *testing.T) {
		_, _, err := repo.PreviousSLAScopeCount(context.Background(), -5, lastMonth, thisMonth)
		require.Error(t, err)
	})
	t.Run("时间窗口顺序错误", func(t *testing.T) {
		_, _, err := repo.PreviousSLAScopeCount(context.Background(), 1, thisMonth, lastMonth)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "必须早于")
	})
	t.Run("零值时间", func(t *testing.T) {
		_, _, err := repo.PreviousSLAScopeCount(context.Background(), 1, time.Time{}, thisMonth)
		require.Error(t, err)
	})
}
