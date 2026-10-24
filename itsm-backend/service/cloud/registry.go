package cloud

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"itsm-backend/ent"
)

// Registry 云发现适配器注册表
// 按 provider + serviceCode 路由到具体 adapter
type Registry struct {
	adapters map[string]map[string]CloudDiscoveryAdapter // provider → serviceCode → adapter
	mu       sync.RWMutex
}

var globalRegistry *Registry

func init() {
	globalRegistry = NewRegistry()
}

// NewRegistry creates an isolated adapter registry for dependency injection.
func NewRegistry() *Registry {
	return &Registry{adapters: make(map[string]map[string]CloudDiscoveryAdapter)}
}

// GlobalRegistry 获取全局注册表
func GlobalRegistry() *Registry {
	return globalRegistry
}

// Register 注册一个 adapter
// 同一 provider+serviceCode 重复注册会覆盖
func (r *Registry) Register(adapter CloudDiscoveryAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider := adapter.Provider()
	serviceCode := adapter.ServiceCode()

	if r.adapters[provider] == nil {
		r.adapters[provider] = make(map[string]CloudDiscoveryAdapter)
	}
	r.adapters[provider][serviceCode] = adapter
}

// Get 获取 adapter
func (r *Registry) Get(provider, serviceCode string) (CloudDiscoveryAdapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if providers, ok := r.adapters[provider]; ok {
		if adapter, ok := providers[serviceCode]; ok {
			return adapter, true
		}
	}
	return nil, false
}

// HasAdapter reports whether a provider/service adapter is installed.
func (r *Registry) HasAdapter(provider, serviceCode string) bool {
	if r == nil {
		return false
	}
	_, ok := r.Get(NormalizeProvider(provider), strings.ToLower(strings.TrimSpace(serviceCode)))
	return ok
}

// GetByAccount 根据账号获取该厂商所有已注册的 adapter
func (r *Registry) GetByAccount(account *ent.CloudAccount) []CloudDiscoveryAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider := NormalizeProvider(account.Provider)
	adapters, ok := r.adapters[provider]
	if !ok {
		return nil
	}

	result := make([]CloudDiscoveryAdapter, 0, len(adapters))
	for _, adapter := range adapters {
		result = append(result, adapter)
	}
	return result
}

// NormalizeProvider 统一云厂商标识
func NormalizeProvider(provider string) string {
	switch strings.ToLower(provider) {
	case "alibaba", "alicloud", "aliyun":
		return "aliyun"
	case "tencentcloud", "qcloud", "tencent":
		return "tencent"
	case "huaweicloud", "huawei":
		return "huawei"
	case "amazon", "aws":
		return "aws"
	case "azure":
		return "azure"
	case "volcengine", "volc", "volcano", "火山":
		return "volcengine"
	case "kubernetes", "k8s", "kube":
		return "k8s"
	case "onprem", "private", "private_cloud":
		return "onprem"
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

// ProviderDisplayName 云厂商显示名
func ProviderDisplayName(provider string) string {
	switch NormalizeProvider(provider) {
	case "aliyun":
		return "阿里云"
	case "tencent":
		return "腾讯云"
	case "huawei":
		return "华为云"
	case "aws":
		return "AWS"
	case "azure":
		return "Azure"
	case "volcengine":
		return "火山引擎"
	case "k8s":
		return "Kubernetes"
	case "onprem":
		return "私有云"
	default:
		return provider
	}
}

// RequireAdapter 确保 adapter 存在，不存在则返回明确错误
func (r *Registry) RequireAdapter(provider, serviceCode string) (CloudDiscoveryAdapter, error) {
	adapter, ok := r.Get(provider, serviceCode)
	if !ok {
		return nil, fmt.Errorf("no adapter registered for provider=%q service=%q", provider, serviceCode)
	}
	return adapter, nil
}

// KnownProviders 是本产品「识别得到」的云厂商全集（含尚未接入发现适配器的）。
//
// 为什么需要单独一份清单：NormalizeProvider 只做字符串归一（alibaba→aliyun），
// 它回答不了「本产品一共认哪些厂商」。缺这份清单时，「未接入」只能等到 job 跑起来
// 抛一句 no adapter registered 才暴露——用户为一个不支持的厂商配完凭据才发现。
//
// 单一真源：dto 侧的 provider 枚举（oneof）与前端厂商下拉都应以本清单为准，
// 由 dto/cloud_provider_enum_guard_test.go 强制对齐，防止三处各写一份再漂移。
// 顺序即 API 输出顺序，刻意固定以保证响应稳定。
var KnownProviders = []string{
	"aliyun", "tencent", "huawei", "aws", "azure", "volcengine", "k8s", "onprem",
}

// IsKnownProvider 报告 provider 归一后是否属于 KnownProviders。
func IsKnownProvider(provider string) bool {
	normalized := NormalizeProvider(provider)
	for _, p := range KnownProviders {
		if p == normalized {
			return true
		}
	}
	return false
}

// ProviderSupport 描述一个厂商当前的发现能力。
//
// NotImplemented 为 true 表示厂商标识已识别（可建云账号），但发现适配器尚未接入：
// 这不是错误态，而是要让「配了账号也不会自动发现」成为可查询的事实。
type ProviderSupport struct {
	Provider       string   `json:"provider"`
	DisplayName    string   `json:"displayName"`
	ServiceCodes   []string `json:"serviceCodes"`
	NotImplemented bool     `json:"notImplemented"`
}

// SupportedProviders 返回 KnownProviders 的能力矩阵。
//
// 清单按 KnownProviders 全量输出，再逐个问注册表「装了没有」——这样新增适配器时
// 不会出现「注册了但清单里没写」的漂移（此前 adapters 包硬编码一份条目，正是
// 这个漂移的第二种写法）。注册表为 nil 时全部标为未接入，而不是返回空切片：
// 空切片会被读成「本产品不认任何厂商」。
func (r *Registry) SupportedProviders() []ProviderSupport {
	registered := r.registeredServiceCodes()
	out := make([]ProviderSupport, 0, len(KnownProviders))
	for _, p := range KnownProviders {
		codes := registered[p]
		sort.Strings(codes)
		if len(codes) == 0 {
			// 已识别但未接入：ServiceCodes 给空数组而不是 null，前端不必判空。
			codes = []string{}
		}
		out = append(out, ProviderSupport{
			Provider:       p,
			DisplayName:    ProviderDisplayName(p),
			ServiceCodes:   codes,
			NotImplemented: len(codes) == 0,
		})
	}
	return out
}

// registeredServiceCodes 返回 provider → 已注册 serviceCode 列表（升序）。
func (r *Registry) registeredServiceCodes() map[string][]string {
	out := make(map[string][]string)
	if r == nil {
		return out
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for provider, byCode := range r.adapters {
		codes := make([]string, 0, len(byCode))
		for code := range byCode {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		out[provider] = codes
	}
	return out
}
