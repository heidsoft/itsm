package service

import "context"

// 交付完成后的联动契约。
//
// 交付是对真实基础设施产生副作用的动作，任务落库成功只说明「云上有一台机器」，
// 不代表服务请求的生命周期已经收口。这里定义两个回调面，让应用层在事务提交后
// 把结果投影到其他域，同时保持依赖方向：service 只声明接口，实现放在各自域。

// ProvisioningDeliveryListener 接收交付生命周期通知。
// 通知发生在交付事务提交之后；监听方失败只记日志，不得回滚已发生的交付。
type ProvisioningDeliveryListener interface {
	OnProvisioningStarted(ctx context.Context, tenantID, serviceRequestID, actorUserID int)
	OnProvisioningFinished(ctx context.Context, tenantID, serviceRequestID, actorUserID int, delivered bool, detail string)
}

// CI 投影结果状态。与 handlers/cmdb 的 CIProjection* 常量同值，
// 定义在这里是为了让 service 层不反向依赖 HTTP 域。
const (
	ProvisioningCIProjectionCreated     = "created"
	ProvisioningCIProjectionUpdated     = "updated"
	ProvisioningCIProjectionDeferred    = "deferred"
	ProvisioningCIProjectionSkipped     = "skipped"
	ProvisioningCIProjectionFailed      = "failed"
	ProvisioningCIProjectionUnavailable = "unavailable"
)

// ProvisioningCIProjectionRequest 描述一次已成功的云资源交付。
// Resources 是 Provider 返回的原始键值（实例 ID、地域、账号等）。
type ProvisioningCIProjectionRequest struct {
	TenantID           int
	ServiceRequestID   int
	ProvisioningTaskID int
	ResourceType       string
	Provider           string
	Resources          map[string]string
	Payload            map[string]any
}

// ProvisioningCIProjectionResult 是一次 CI 回写的结果；Reason 面向日志，不面向用户文案。
type ProvisioningCIProjectionResult struct {
	Status string
	CIID   int
	Reason string
}

// ProvisioningCIProjector 把已交付的云资源落成配置项。
// 实现位于 handlers/cmdb，复用发现链路的 CI 类型解析与去重规则。
type ProvisioningCIProjector interface {
	ProjectProvisionedResource(ctx context.Context, req ProvisioningCIProjectionRequest) ProvisioningCIProjectionResult
}
