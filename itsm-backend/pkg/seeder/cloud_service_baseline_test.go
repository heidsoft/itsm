package seeder

import (
	"context"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/cloudservice"
	"itsm-backend/ent/predicate"
	"itsm-backend/ent/tenant"
	"itsm-backend/internal/initialization"
	"itsm-backend/pkg/tenantmode"

	"github.com/stretchr/testify/require"
)

// 云服务类型基线的安装、幂等与租户边界回归锁。
//
// 缺陷本体：`cloud_services` 从未被任何生产代码写入（seeder 零引用、迁移零 INSERT），
// 而发现执行器要求它存在。这里既锁住 helper 层的行为，也锁住真实生产入口
// （cmdb-core 组件）确实会落地这些行。

func newCloudServiceTenant(t *testing.T, seeder *Seeder, ctx context.Context, code string) *ent.Tenant {
	t.Helper()
	target, err := seeder.client.Tenant.Create().
		SetCode(code).SetName("CS " + code).
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)
	return target
}

func countCloudServices(t *testing.T, seeder *Seeder, ctx context.Context, tenantID int, predicates ...predicate.CloudService) int {
	t.Helper()
	all := append([]predicate.CloudService{cloudservice.TenantIDEQ(tenantID)}, predicates...)
	count, err := seeder.client.CloudService.Query().Where(all...).Count(ctx)
	require.NoError(t, err)
	return count
}

func TestCloudServiceBaselineIsTenantScopedAndIdempotent(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	root, err := seeder.client.Tenant.Query().Where(tenant.CodeEQ(platformTenantCode)).Only(ctx)
	require.NoError(t, err)
	target := newCloudServiceTenant(t, seeder, ctx, "cs-idem")

	defs := cloudServiceTypeDefinitions()
	scoped := seeder.withBaselineTenant(target.ID)
	// 重复执行是初始化重放与并发安装的常态：第二遍不得增行，也不得报错。
	require.NoError(t, scoped.seedCloudServiceTypes(ctx))
	require.NoError(t, scoped.seedCloudServiceTypes(ctx))

	require.Equal(t, len(defs), countCloudServices(t, seeder, ctx, target.ID))
	// 预置行必须标 is_system：否则租户会把自己环境的基础定义当成手工数据清理掉。
	require.Equal(t, len(defs), countCloudServices(t, seeder, ctx, target.ID, cloudservice.IsSystem(true)))
	// 按目标租户写入，不得顺手污染平台基线租户。
	require.Equal(t, 0, countCloudServices(t, seeder, ctx, root.ID))
}

func TestCloudServiceBaselinePreservesTenantCustomization(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	target := newCloudServiceTenant(t, seeder, ctx, "cs-custom")

	defs := cloudServiceTypeDefinitions()
	first := defs[0]
	_, err := seeder.client.CloudService.Create().
		SetTenantID(target.ID).SetProvider(first.Provider).
		SetCategory("custom-category").SetServiceCode(first.ServiceCode).
		SetServiceName("租户自定义名称").SetResourceTypeCode(first.ResourceTypeCode).
		SetResourceTypeName("自定义类型").SetIsActive(true).Save(ctx)
	require.NoError(t, err)

	require.NoError(t, seeder.withBaselineTenant(target.ID).seedCloudServiceTypes(ctx))

	// 已存在的行归租户所有：既不能重复建（撞唯一索引），也不能把改名/分类改回去。
	require.Equal(t, len(defs), countCloudServices(t, seeder, ctx, target.ID))
	kept, err := seeder.client.CloudService.Query().Where(
		cloudservice.TenantIDEQ(target.ID),
		cloudservice.ServiceNameEQ("租户自定义名称"),
	).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, kept, "预置逻辑不得覆盖租户已改过的定义")
}

func TestCloudServiceBaselineVerifyFailsWhenMissing(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	target := newCloudServiceTenant(t, seeder, ctx, "cs-verify")

	// 只落一条就校验：verify 必须逐条查，否则空表也会被判成基线完整。
	def := cloudServiceTypeDefinitions()[0]
	_, err := seeder.client.CloudService.Create().
		SetTenantID(target.ID).SetProvider(def.Provider).SetCategory(def.Category).
		SetServiceCode(def.ServiceCode).SetServiceName(def.ServiceName).
		SetResourceTypeCode(def.ResourceTypeCode).SetResourceTypeName(def.ResourceTypeName).
		SetIsSystem(true).Save(ctx)
	require.NoError(t, err)

	err = seeder.withBaselineTenant(target.ID).verifyCloudServiceTypes(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing")

	require.NoError(t, seeder.withBaselineTenant(target.ID).seedCloudServiceTypes(ctx))
	require.NoError(t, seeder.withBaselineTenant(target.ID).verifyCloudServiceTypes(ctx))
}

// TestCMDBCoreComponentInstallsCloudServiceBaseline 锁真实生产入口：
// 云服务类型必须随 cmdb-core 组件一起装进目标租户。只测 seedCloudServiceTypes
// 而组件里没接线，表现依然是新租户的发现任务整批失败。
func TestCMDBCoreComponentInstallsCloudServiceBaseline(t *testing.T) {
	seeder, ctx := newTestSeeder(t, tenantmode.DeploymentModePrivate)
	seeder.seedDefaultTenant(ctx)
	target, err := seeder.client.Tenant.Create().
		SetCode("cs-component").SetName("CS Component").
		SetType(tenant.TypeSaasCustomer).Save(ctx)
	require.NoError(t, err)

	installTenantBaseline(t, seeder, ctx, initialization.Scope{Type: "tenant", ID: int64(target.ID)})

	require.Equal(t, len(cloudServiceTypeDefinitions()),
		countCloudServices(t, seeder, ctx, target.ID, cloudservice.IsSystem(true)))
}
