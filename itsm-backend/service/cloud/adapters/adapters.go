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
	"itsm-backend/service/cloud/aws"
	"itsm-backend/service/cloud/kubernetes"
	"itsm-backend/service/cloud/tencent"
	"itsm-backend/service/cloud/volcengine"
)

// RegisterAll 把所有内建适配器注册进 reg。
//
// 重复调用是安全的：注册按 provider+serviceCode 覆盖，不会堆积。
// 新增厂商适配器时，只在这里加一行，并在 SupportedProviders 里登记。
func RegisterAll(reg *cloud.Registry, logger *zap.SugaredLogger) {
	if reg == nil {
		return
	}
	// 阿里云：仅 ECS（实例）已实现，走官方 SDK。
	reg.Register(aliyun.NewAliyunECSAdapter(logger))
	// Kubernetes：只读发现 Pod，走 APIServer REST，不引 client-go。
	reg.Register(kubernetes.NewAdapter(logger))
	// AWS：EC2 只读发现，SigV4 手写签名，不引 aws-sdk-go-v2。
	reg.Register(aws.NewAdapter(logger))
	// 腾讯云：CVM 只读发现，TC3-HMAC-SHA256 手写签名。
	reg.Register(tencent.NewAdapter(logger))
	// 火山引擎：ECS 只读发现，HMAC-SHA256 签名（与 SigV4 同骨架、不同细节）。
	reg.Register(volcengine.NewAdapter(logger))
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
		{
			Provider:       aws.Provider,
			DisplayName:    cloud.ProviderDisplayName(aws.Provider),
			ServiceCodes:   []string{aws.ServiceCodeEC2},
			NotImplemented: false,
		},
		{
			Provider:       tencent.Provider,
			DisplayName:    cloud.ProviderDisplayName(tencent.Provider),
			ServiceCodes:   []string{tencent.ServiceCodeCVM},
			NotImplemented: false,
		},
		{
			Provider:       volcengine.Provider,
			DisplayName:    cloud.ProviderDisplayName(volcengine.Provider),
			ServiceCodes:   []string{volcengine.ServiceCodeECS},
			NotImplemented: false,
		},
		// 华为云 / Azure 仍未接入：两家各有自己的签名体系（AK/SK 签名与
		// OAuth2 + HMAC），不是「照抄现成的」就能落地，故继续显式标缺口而不是
		// 注册一个跑不通的适配器。
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
