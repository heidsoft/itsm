package service

import (
	"context"
	"database/sql"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestCINumberExistsRawSeesScrappedCIs 锁定回归：ci_number 是全局唯一索引（不含 tenant_id），
// 而 Ent 拦截器会自动附加生命周期过滤（lifecycle_status != 'scrapped'）。
// 若存在性检查走 Ent 查询，已退役 CI 占用的编号会被误判为"未占用"，
// 进而生成重复编号并撞上全局唯一约束（23505）。
func TestCINumberExistsRawSeesScrappedCIs(t *testing.T) {
	dsn := testDSN()
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()

	// cache=shared 允许另开一条连接直连同一内存库，模拟 bootstrap 注入的 rawDB
	rawDB, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer rawDB.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("CI number tenant").SetCode("ci-number").SetDomain("ci-number.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	const takenNumber = "CI-202609-000042"
	_, err = client.ConfigurationItem.Create().
		SetName("retired-asset").
		SetCiTypeID(ciType.ID).
		SetCiType("Server").
		SetStatus(common.CIStatusRetired).
		SetLifecycleStatus(common.CILifecycleStatusScrapped).
		SetCiNumber(takenNumber).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	svc := NewConfigurationItemService(client, logger, NewCIHistoryService(client, logger), NewCITagService(client, logger))
	svc.SetRawDB(rawDB)

	// 修复：原生 SQL 绕过所有拦截器，必须看到被已退役 CI 占用的编号。
	//
	// 注意：enttest 环境不挂载生产的软删/租户拦截器，因此这里无法直接复现线上故障
	// （线上 Ent 查询会过滤掉 scrapped 记录导致误判为"未占用"）。本用例锁定的契约是：
	// 无论 CI 处于何种生命周期状态，只要 ci_number 已被占用就必须返回 true。
	viaRaw, err := svc.ciNumberExistsRaw(ctx, takenNumber)
	require.NoError(t, err)
	require.True(t, viaRaw, "ciNumberExistsRaw 必须看到已退役 CI 占用的编号，否则会生成重复 ci_number 并撞全局唯一约束")

	// 未被占用的编号仍应返回 false
	free, err := svc.ciNumberExistsRaw(ctx, "CI-202609-999999")
	require.NoError(t, err)
	require.False(t, free)

	// 跨租户可见性：ci_number 是全局唯一索引（不含 tenant_id），
	// 因此其他租户占用的编号也必须判定为"已占用"。
	otherTenant, err := client.Tenant.Create().
		SetName("other tenant").SetCode("ci-number-2").SetDomain("ci-number-2.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	const crossTenantNumber = "CI-202609-000077"
	_, err = client.ConfigurationItem.Create().
		SetName("other-tenant-asset").
		SetCiTypeID(ciType.ID).
		SetCiType("Server").
		SetStatus("active").
		SetCiNumber(crossTenantNumber).
		SetTenantID(otherTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 对照：带租户/生命周期作用域的查询看不到其他租户占用的编号。
	// 这正是线上误判的成因——生产 Ent 拦截器会自动附加这两类过滤，
	// 而 ci_number 的唯一约束是全局的，二者视角不一致必然撞号。
	// （enttest 环境不挂载拦截器，故此处显式写出过滤条件以固定对照语义。）
	viaScoped, err := client.ConfigurationItem.Query().
		Where(
			configurationitem.CiNumberEQ(crossTenantNumber),
			configurationitem.TenantIDEQ(tenant.ID),
			configurationitem.LifecycleStatusNEQ(common.CILifecycleStatusScrapped),
		).
		Exist(ctx)
	require.NoError(t, err)
	require.False(t, viaScoped, "对照：带作用域的查询看不到其他租户/已退役的编号")

	crossTenant, err := svc.ciNumberExistsRaw(ctx, crossTenantNumber)
	require.NoError(t, err)
	require.True(t, crossTenant, "ci_number 为全局唯一索引，其他租户占用的编号同样不可复用")
}

// TestCINumberFallsBackToRawSQLWithoutRawDB 未注入 rawDB 时退化为 Ent 查询，不应 panic。
func TestCINumberFallsBackToRawSQLWithoutRawDB(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewConfigurationItemService(client, logger, NewCIHistoryService(client, logger), NewCITagService(client, logger))

	exists, err := svc.ciNumberExistsRaw(ctx, "CI-202609-000001")
	require.NoError(t, err)
	require.False(t, exists)
}
