package adapters

import (
	"testing"

	"go.uber.org/zap"

	"itsm-backend/service/cloud"
	"itsm-backend/service/cloud/aliyun"
	"itsm-backend/service/cloud/aws"
	"itsm-backend/service/cloud/kubernetes"
	"itsm-backend/service/cloud/tencent"
	"itsm-backend/service/cloud/volcengine"
)

// TestRegisterAll_AdaptersBecomeReachable 回归：注册是云发现的必要前提。
//
// 此前全仓生产代码没有任何一处调用 Registry.Register，而 DiscoveryJobExecutor
// 是按 provider+serviceCode 从注册表取适配器的——结果就是「适配器代码写好了
// 但装配缺失」，所有发现任务都以 no adapter registered 失败。这条断言把装配
// 本身钉住。
func TestRegisterAll_AdaptersBecomeReachable(t *testing.T) {
	reg := cloud.NewRegistry()
	RegisterAll(reg, zap.NewNop().Sugar())

	for _, tc := range []struct{ provider, service string }{
		{aliyun.Provider, aliyun.ServiceCodeECS},
		{kubernetes.Provider, kubernetes.ServiceCode},
		{aws.Provider, aws.ServiceCodeEC2},
		{tencent.Provider, tencent.ServiceCodeCVM},
		{volcengine.Provider, volcengine.ServiceCodeECS},
	} {
		if _, err := reg.RequireAdapter(tc.provider, tc.service); err != nil {
			t.Fatalf("expected %s/%s to be registered: %v", tc.provider, tc.service, err)
		}
	}
}

func TestRegisterAll_NilRegistryIsNoop(t *testing.T) {
	RegisterAll(nil, zap.NewNop().Sugar())
}

// TestSupportedProviders_UnimplementedAreFlagged 覆盖：未接入的厂商必须显式
// 标出，而不是从清单里消失（消失会被读成「还没做清单」，标出才是「已知缺口」）。
func TestSupportedProviders_UnimplementedAreFlagged(t *testing.T) {
	reg := cloud.NewRegistry()
	RegisterAll(reg, zap.NewNop().Sugar())

	list := SupportedProviders(reg)
	byProvider := make(map[string]cloud.ProviderSupport, len(list))
	for _, p := range list {
		byProvider[p.Provider] = p
	}
	for _, p := range []string{"aliyun", "k8s", "aws", "tencent", "volcengine", "huawei", "azure"} {
		if _, ok := byProvider[p]; !ok {
			t.Fatalf("expected %s in capability list, got %v", p, byProvider)
		}
	}
	// 三家新接入的厂商（AWS SigV4 / 腾讯 TC3 / 火山 HMAC-SHA256）与已有的一样，
	// 都已注册真实适配器，不该再被标成未实现。
	for _, p := range []string{"aliyun", "k8s", "aws", "tencent", "volcengine"} {
		if byProvider[p].NotImplemented {
			t.Fatalf("%s is implemented and must not be flagged", p)
		}
		if len(byProvider[p].ServiceCodes) == 0 {
			t.Fatalf("%s must advertise its service codes", p)
		}
	}
	for _, p := range []string{"huawei", "azure"} {
		if !byProvider[p].NotImplemented {
			t.Fatalf("%s has no adapter and must be flagged as not implemented", p)
		}
	}
}

// TestSupportedProviders_FlagsUnregisteredInsteadOfDropping 覆盖：注册表里没有
// 适配器的厂商必须被标成未接入，而不是从清单里消失。
//
// 消失会被读成「还没做清单」，标出才是「已知缺口」；而且清单一旦按「有没有适配器」
// 过滤，新接入的厂商在前端建号下拉里就会凭空消失——正是 2026-10-24 实测到的缺陷。
func TestSupportedProviders_FlagsUnregisteredInsteadOfDropping(t *testing.T) {
	empty := cloud.NewRegistry()
	list := SupportedProviders(empty)
	// 先钉住「不收缩」：空注册表也必须回全集，否则就是上面那条失效方式。
	if len(list) != len(cloud.KnownProviders) {
		t.Fatalf("清单必须是厂商全集: got %d want %d", len(list), len(cloud.KnownProviders))
	}
	for _, p := range list {
		if !p.NotImplemented {
			t.Fatalf("%s 在空注册表里不得标为已接入", p.Provider)
		}
		if p.ServiceCodes == nil {
			t.Fatalf("%s 的 serviceCodes 应为 [] 而非 null（前端渲染会判空）", p.Provider)
		}
	}
}

// TestNormalizeProvider_NewVendors 覆盖新增厂商标识的归一。
func TestNormalizeProvider_NewVendors(t *testing.T) {
	cases := map[string]string{
		"volcengine": "volcengine",
		"volc":       "volcengine",
		"k8s":        "k8s",
		"kubernetes": "k8s",
		"Kubernetes": "k8s",
		"aliyun":     "aliyun",
		"alibaba":    "aliyun",
	}
	for in, want := range cases {
		if got := cloud.NormalizeProvider(in); got != want {
			t.Fatalf("NormalizeProvider(%q)=%q want %q", in, got, want)
		}
	}
}
