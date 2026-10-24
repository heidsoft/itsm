// Package adapters 是「云发现适配器注册表」的唯一装配点。
//
// 之所以单独一个包：`cloud/aliyun` / `cloud/kubernetes` 都 import `cloud`，
// 注册代码若放在 cloud 包内就会形成 import 环。注册表本身只认
// `cloud.CloudDiscoveryAdapter` 接口，装配权在这里，cloud 包保持零依赖。
//
// ⚠️ 在补上本文件之前，全仓生产代码没有任何一处调用 `Registry.Register`，
// 而 `DiscoveryJobExecutor` 执行时是按 provider+serviceCode 查注册表的——
// 也就是说**所有云发现任务都会以 "no adapter registered" 失败**，阿里云 ECS
// 适配器虽然写好了却从未被装配。注册是发现的必要前提，不是可选项。
package adapters

import (
	"go.uber.org/zap"

	"itsm-backend/service/cloud"
	"itsm-backend/service/cloud/aliyun"
	"itsm-backend/service/cloud/kubernetes"
)

// RegisterAll 把所有内建适配器注册进 reg。
//
// 重复调用是安全的：注册按 provider+serviceCode 覆盖，不会堆积。
// 新增厂商适配器时，只在这里加一行，并在 SupportedProviders 里登记。
func RegisterAll(reg *cloud.Registry, logger *zap.SugaredLogger) {
	if reg == nil {
		return
	}
	// 阿里云：仅 ECS（实例）已实现。
	reg.Register(aliyun.NewAliyunECSAdapter(logger))
	// Kubernetes：只读发现 Pod，走 APIServer REST，不引 client-go。
	reg.Register(kubernetes.NewAdapter(logger))
}

// ProviderSupport 描述一个厂商当前的发现能力。
//
// 目的是让「未接入」成为**可查询的事实**，而不是等到 job 失败时抛一句
// no adapter registered。API 与前端据此展示「暂不支持」，避免用户为不支持的
// 厂商配半天凭据。
type ProviderSupport struct {
	Provider     string   `json:"provider"`
	DisplayName  string   `json:"displayName"`
	ServiceCodes []string `json:"serviceCodes"`
	// NotImplemented 为 true 表示厂商标识已识别，但适配器尚未接入。
	NotImplemented bool `json:"notImplemented"`
}

// SupportedProviders 返回当前注册表实际具备发现能力的厂商清单。
func SupportedProviders(reg *cloud.Registry) []ProviderSupport {
	if reg == nil {
		return nil
	}
	out := []ProviderSupport{
		{
			Provider:       aliyun.Provider,
			DisplayName:    cloud.ProviderDisplayName(aliyun.Provider),
			ServiceCodes:   []string{aliyun.ServiceCodeECS},
			NotImplemented: false,
		},
		{
			Provider:       kubernetes.Provider,
			DisplayName:    cloud.ProviderDisplayName(kubernetes.Provider),
			ServiceCodes:   []string{kubernetes.ServiceCode},
			NotImplemented: false,
		},
		// 以下厂商标识已在 cloud.NormalizeProvider 归一，凭据解析/ACCOUNT 模型
		// 也已就绪，但发现适配器未接入：接入 AWS/Tencent/火山需要各自的签名
		// （SigV4 / TC3）或官方 SDK，属于独立工作项，不应以「注册了但跑不通」
		// 的方式假装支持。
		{Provider: "aws", DisplayName: cloud.ProviderDisplayName("aws"), NotImplemented: true},
		{Provider: "tencent", DisplayName: cloud.ProviderDisplayName("tencent"), NotImplemented: true},
		{Provider: "volcengine", DisplayName: cloud.ProviderDisplayName("volcengine"), NotImplemented: true},
		{Provider: "huawei", DisplayName: cloud.ProviderDisplayName("huawei"), NotImplemented: true},
		{Provider: "azure", DisplayName: cloud.ProviderDisplayName("azure"), NotImplemented: true},
	}
	// 只保留真的能从注册表取到适配器的条目，防止清单与实现脱节。
	implemented := make([]ProviderSupport, 0, len(out))
	for _, p := range out {
		if p.NotImplemented {
			implemented = append(implemented, p)
			continue
		}
		ok := false
		for _, code := range p.ServiceCodes {
			if reg.HasAdapter(p.Provider, code) {
				ok = true
				break
			}
		}
		if ok {
			implemented = append(implemented, p)
		}
	}
	return implemented
}
