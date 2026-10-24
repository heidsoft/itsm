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

// SupportedProviders 返回当前注册表的能力矩阵，见 cloud.Registry.SupportedProviders。
//
// 保留本函数只为装配侧调用方便；真源在注册表（按 cloud.KnownProviders 逐个问
// 「装了没有」），本包不再维护第二份条目清单——此前这里硬编码五条 implemented +
// 两条 notImplemented，新增适配器只要忘了同步改这里，能力清单就会说谎。
func SupportedProviders(reg *cloud.Registry) []cloud.ProviderSupport {
	return reg.SupportedProviders()
}
