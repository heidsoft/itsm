package asset

import (
	"context"
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

// newLicenseStatsMockClient 用 sqlmock 构造 Ent client，用于注入单状态计数故障。
func newLicenseStatsMockClient(t *testing.T) (*ent.Client, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	t.Cleanup(func() { _ = client.Close() })
	return client, mock
}

// licenseStatusCases 按 GetLicenseStats 的实际查询顺序列出 4 个状态。
var licenseStatusCases = []struct {
	status dto.LicenseStatus
	label  string
}{
	{dto.LicenseStatusActive, "有效"},
	{dto.LicenseStatusExpired, "已过期"},
	{dto.LicenseStatusExpiringSoon, "即将过期"},
	{dto.LicenseStatusDepleted, "已耗尽"},
}

// expectLicenseCounts 登记 total + 4 个状态计数。failTotal 与 failStatusIdx
// 分开表达，避免用负数哨兵造成「本想打状态却打中 total」的测试自身错误。
func expectLicenseCounts(mock sqlmock.Sqlmock, failTotal bool, failStatusIdx int) {
	exp := mock.ExpectQuery("SELECT COUNT")
	if failTotal {
		exp.WillReturnError(errors.New("injected license total failure"))
	} else {
		exp.WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	}
	for i, tc := range licenseStatusCases {
		exp := mock.ExpectQuery("SELECT COUNT").WithArgs(sqlmock.AnyArg(), string(tc.status))
		if i == failStatusIdx {
			exp.WillReturnError(errors.New("injected license count failure for " + string(tc.status)))
		} else {
			exp.WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(i + 1))
		}
	}
}

// TestGetLicenseStats_FailClosedOnStatusCountError 是「撤掉守卫即失败」的回归锁：
// 旧实现 4 个状态 Count 全部吞错为 0，DB 故障时合规率算成 0%，把故障伪装成
// 「许可证全部失效」的业务结论。逐个状态注入故障，断言整体 fail-closed。
func TestGetLicenseStats_FailClosedOnStatusCountError(t *testing.T) {
	for i, tc := range licenseStatusCases {
		t.Run(tc.label, func(t *testing.T) {
			client, mock := newLicenseStatsMockClient(t)
			expectLicenseCounts(mock, false, i)

			svc := NewAssetLicenseService(client, zap.NewNop().Sugar())
			stats, err := svc.GetLicenseStats(context.Background(), 1)
			require.Error(t, err, "%s 计数失败必须 fail-closed", tc.status)
			assert.Nil(t, stats, "统计失败不得返回半真半假的 stats")
			assert.Contains(t, err.Error(), "统计许可证状态 "+string(tc.status)+" 失败",
				"错误必须带状态标签，便于从日志定位")
			assert.Contains(t, err.Error(), "injected license count failure for "+string(tc.status),
				"底层 DB 错误必须原样上抛，不能被包装吞掉")
		})
	}
}

// TestGetLicenseStats_FailClosedOnTotalCountError 锁定 total 的 fail-closed 语义：
// total 是合规率的分母，DB 失败时同样必须整体报错。
func TestGetLicenseStats_FailClosedOnTotalCountError(t *testing.T) {
	client, mock := newLicenseStatsMockClient(t)
	expectLicenseCounts(mock, true, -1) // total 失败，状态计数正常登记

	svc := NewAssetLicenseService(client, zap.NewNop().Sugar())
	stats, err := svc.GetLicenseStats(context.Background(), 1)
	require.Error(t, err)
	assert.Nil(t, stats)
	assert.Contains(t, err.Error(), "injected license total failure")
}

// TestGetLicenseStats_AllCountsMapToFields 锁定成功路径的字段对位与合规率计算。
func TestGetLicenseStats_AllCountsMapToFields(t *testing.T) {
	client, mock := newLicenseStatsMockClient(t)
	expectLicenseCounts(mock, false, -1)

	svc := NewAssetLicenseService(client, zap.NewNop().Sugar())
	stats, err := svc.GetLicenseStats(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 10, stats.Total)
	assert.Equal(t, 1, stats.Active)
	assert.Equal(t, 2, stats.Expired)
	assert.Equal(t, 3, stats.ExpiringSoon)
	assert.Equal(t, 4, stats.Depleted)
	assert.InDelta(t, 10.0, stats.ComplianceRate, 0.001, "合规率 = active/total*100 = 1/10*100")
}
