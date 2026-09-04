package bootstrap

import (
	"context"
	"database/sql"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestMakeSequenceDBSyncFnSupportsCIKey 锁定回归：ci_number 的 Redis 序列键
// sequence:ci:<YYYYMM> 必须能被 DB 同步逻辑识别，否则 Redis 重置后会从 1 重新发号，
// 撞上存量编号（全局唯一约束 23505）。
// 同时验证同步查询因使用原生 SQL 而把已退役（scrapped）CI 的编号计入最大值。
func TestMakeSequenceDBSyncFnSupportsCIKey(t *testing.T) {
	const dsn = "file:bootstrap-ci-seq?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()

	rawDB, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer rawDB.Close()

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("seq tenant").SetCode("bootstrap-seq").SetDomain("bootstrap-seq.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// 当月已占用 000041（在役）与 000042（已退役）
	active := client.ConfigurationItem.Create().
		SetName("live-asset").SetCiTypeID(ciType.ID).SetCiType("Server").
		SetStatus("active").SetCiNumber("CI-202609-000041").SetTenantID(tenant.ID)
	_, err = active.Save(ctx)
	require.NoError(t, err)

	retired := client.ConfigurationItem.Create().
		SetName("retired-asset").SetCiTypeID(ciType.ID).SetCiType("Server").
		SetStatus(common.CIStatusRetired).SetLifecycleStatus(common.CILifecycleStatusScrapped).
		SetCiNumber("CI-202609-000042").SetTenantID(tenant.ID)
	_, err = retired.Save(ctx)
	require.NoError(t, err)

	fn := makeSequenceDBSyncFn(rawDB, zaptest.NewLogger(t).Sugar())

	got, err := fn("sequence:ci:202609")
	require.NoError(t, err)
	require.Equal(t, int64(42), got, "序列起点须对齐当月最大编号，且包含已退役 CI 占用的编号")

	// 无存量记录时返回 0（首次发号从 1 开始）
	empty, err := fn("sequence:ci:202701")
	require.NoError(t, err)
	require.Equal(t, int64(0), empty)

	// 非法 YYYYMM 必须报错，不能静默返回 0（否则序列会从 1 重发）
	_, err = fn("sequence:ci:20")
	require.Error(t, err)
}
