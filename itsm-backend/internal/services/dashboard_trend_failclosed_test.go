package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newTrendMockClient 用 sqlmock 构造 Ent client，用于在无真实 DB 的前提下
// 注入单维度计数故障。
func newTrendMockClient(t *testing.T) (*ent.Client, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	t.Cleanup(func() { _ = client.Close() })
	return client, mock
}

// expectTrendCounts 按 getTicketTrend 的固定查询顺序登记 5 个 COUNT。
// 顺序为：待处理 / 进行中 / 已解决 / 已关闭 / 新建。
func expectTrendCounts(mock sqlmock.Sqlmock, failAt int) {
	labels := []string{"待处理", "进行中", "已解决", "已关闭", "新建"}
	for i, label := range labels {
		exp := mock.ExpectQuery("SELECT COUNT")
		if i == failAt {
			exp.WillReturnError(errors.New("injected trend count failure for " + label))
		} else {
			// 各维返回可区分的值，成功路径同时验证字段对位。
			exp.WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(i + 1))
		}
	}
}

// TestDashboardService_GetTicketTrend_FailClosedOnAnyCountError 是「撤掉守卫即失败」
// 的回归锁：旧实现 5 个 Count 全部 `count, _ :=` 吞错，DB 故障时返回日期齐全、
// 各维度全 0 的假趋势，服务质量报告据此得出完全错误的结论。
// 这里对 5 个维度逐一注入单点故障，断言整体返回错误且 trend == nil。
func TestDashboardService_GetTicketTrend_FailClosedOnAnyCountError(t *testing.T) {
	labels := []string{"待处理", "进行中", "已解决", "已关闭", "新建"}
	for i, label := range labels {
		t.Run(label, func(t *testing.T) {
			client, mock := newTrendMockClient(t)
			expectTrendCounts(mock, i)

			// getTicketTrend 不依赖 repo/db，只走 client；直接构造可避开无关子系统。
			svc := &DashboardService{client: client, logger: zap.NewNop().Sugar()}
			trend, err := svc.GetTicketTrend(context.Background(), 1, 1)
			require.Error(t, err, "%s 计数失败必须 fail-closed", label)
			assert.Nil(t, trend, "统计失败不得返回半真半假的趋势数据")
			assert.Contains(t, err.Error(), fmt.Sprintf("统计工单趋势[%s]", label),
				"错误必须带维度标签，便于从日志定位")
			assert.Contains(t, err.Error(), time.Now().Format("01-02"),
				"错误必须带日期，便于定位是哪一天的聚合失败")
			assert.Contains(t, err.Error(), "injected trend count failure for "+label,
				"底层 DB 错误必须原样上抛，不能被包装吞掉")
		})
	}
}

// TestDashboardService_GetTicketTrend_AllCountsMapToFields 锁定成功路径的字段对位：
// 5 个 COUNT 依次返回 1..5，派生字段（完成/待处理）也必须由权威计数推导。
func TestDashboardService_GetTicketTrend_AllCountsMapToFields(t *testing.T) {
	client, mock := newTrendMockClient(t)
	expectTrendCounts(mock, -1)

	svc := &DashboardService{client: client, logger: zap.NewNop().Sugar()}
	trend, err := svc.GetTicketTrend(context.Background(), 1, 1)
	require.NoError(t, err)
	require.Len(t, trend, 1)
	assert.Equal(t, 1, trend[0].Open)
	assert.Equal(t, 2, trend[0].InProgress)
	assert.Equal(t, 3, trend[0].Resolved)
	assert.Equal(t, 4, trend[0].Closed)
	assert.Equal(t, 5, trend[0].NewTickets)
	assert.Equal(t, 3, trend[0].CompletedTickets, "已完成 = 已解决")
	assert.Equal(t, 3, trend[0].PendingTickets, "待处理 = 待处理 + 进行中")
	assert.NotEmpty(t, trend[0].Date)
}
