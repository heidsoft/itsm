package cloud

import (
	"fmt"
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
	globalRegistry = &Registry{
		adapters: make(map[string]map[string]CloudDiscoveryAdapter),
	}
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
	switch provider {
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
