package seeder

import (
	"strings"
	"testing"

	"itsm-backend/handlers/cmdb"
	"itsm-backend/service/cloud"
	"itsm-backend/service/cloud/adapters"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 云发现基线与适配器装配的同源守卫。
//
// 两个方向都必须查，因为它们各自的失效方式不同：
//   - 基线多出未接入的厂商 → 前端能选、账号能建，任务跑起来才「no adapter」，
//     等于把假能力重新装回产品（本仓库反复踩过的枚举漂移）；
//   - 基线漏掉已接入的厂商 → 该厂商的新租户第一次点「开始发现」就在
//     runScope 的「load cloud service」处整批失败，而这次失败的报错会把人指向
//     凭据而不是缺基线。

func registeredAdapterPairs(t *testing.T) map[string]struct{} {
	t.Helper()
	reg := cloud.NewRegistry()
	adapters.RegisterAll(reg, zap.NewNop().Sugar())
	out := map[string]struct{}{}
	for _, support := range reg.SupportedProviders() {
		for _, code := range support.ServiceCodes {
			out[support.Provider+"/"+code] = struct{}{}
		}
	}
	return out
}

func TestCloudServiceBaselineMatchesRegisteredAdapters(t *testing.T) {
	registered := registeredAdapterPairs(t)

	baselined := map[string]struct{}{}
	for _, def := range cloudServiceTypeDefinitions() {
		key := def.Provider + "/" + def.ServiceCode
		// 集合去重前先看条目数：同一 provider+serviceCode 写两遍不会被集合差异
		// 暴露，但会在真实安装时撞复合唯一索引。
		require.NotContains(t, baselined, key, "基线存在重复条目 "+key)
		baselined[key] = struct{}{}
	}

	assert.Equal(t, registered, baselined,
		"云服务类型基线必须与 service/cloud/adapters.RegisterAll 的装配结果同源")
}

func TestCloudServiceBaselineFieldsArePopulated(t *testing.T) {
	// provider/service_code/resource_type_code/service_name/resource_type_name
	// 在 schema 里都是 NotEmpty，基线留空只会在第一次 seed 时报错，
	// 而那时报错信息看不出是清单写漏了。
	for _, def := range cloudServiceTypeDefinitions() {
		require.NotEmpty(t, def.Provider, def.ServiceCode)
		require.NotEmpty(t, def.ServiceCode, def.Provider)
		require.NotEmpty(t, def.ServiceName, def.Provider)
		require.NotEmpty(t, def.ResourceTypeCode, def.Provider)
		require.NotEmpty(t, def.ResourceTypeName, def.Provider)
	}
}

// TestDiscoveryCIProjectionMatchesBaseline 锁住「发现回写映射表 ↔ 基线」这条链：
// 基线改了 resource_type_code、或租户默认不再预置某个 CI 类型时，回写会静默变成
// 「所有资源都 skipped」，任务照样成功、配置项一条不建。
func TestDiscoveryCIProjectionMatchesBaseline(t *testing.T) {
	resourceTypes := map[string]struct{}{}
	for _, def := range cloudServiceTypeDefinitions() {
		resourceTypes[def.ServiceCode+"|"+def.ResourceTypeCode] = struct{}{}
	}
	ciTypes := map[string]struct{}{}
	for _, def := range ciTypeDefinitions() {
		ciTypes[def.Name] = struct{}{}
	}

	keys := cmdb.DiscoveryCITypeMappingKeys()
	require.NotEmpty(t, keys, "发现回写映射表不能是空的")
	for _, key := range keys {
		require.Contains(t, resourceTypes, key, "回写映射的云服务类型不在 cloud_services 基线里")
		parts := strings.SplitN(key, "|", 2)
		require.Len(t, parts, 2, key)
		ciType, ok := cmdb.CITypeForCloudResource(parts[0], parts[1])
		require.True(t, ok, key)
		require.Contains(t, ciTypes, ciType, "回写目标 CI 类型不在租户基线清单里")
	}
}
