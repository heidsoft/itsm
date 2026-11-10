package service

import (
	"context"

	"itsm-backend/dto"
)

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
// 定义下沉 dto 是为了让 service 层不反向依赖 HTTP 域。
const (
	ProvisioningCIProjectionCreated     = dto.ProvisioningCIProjectionCreated
	ProvisioningCIProjectionUpdated     = dto.ProvisioningCIProjectionUpdated
	ProvisioningCIProjectionDeferred    = dto.ProvisioningCIProjectionDeferred
	ProvisioningCIProjectionSkipped     = dto.ProvisioningCIProjectionSkipped
	ProvisioningCIProjectionFailed      = dto.ProvisioningCIProjectionFailed
	ProvisioningCIProjectionUnavailable = dto.ProvisioningCIProjectionUnavailable
)

// ProvisioningCIProjectionRequest / Result / Projector 的单一定义在 dto，
// service 侧保留同名别名，交付链路与回归锁的签名不变。
type ProvisioningCIProjectionRequest = dto.ProvisioningCIProjectionRequest
type ProvisioningCIProjectionResult = dto.ProvisioningCIProjectionResult
type ProvisioningCIProjector = dto.ProvisioningCIProjector
