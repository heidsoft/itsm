package provisioning

import (
	"context"

	"itsm-backend/ent"
)

// ProvisioningTaskService 是交付任务 handler 真正消费的最小能力面。
// 方法集按整个域包的生产文件对字段的调用并集整理；组合根注入的共享层
// 具体指针自动满足接口，装配代码无需改动。
// 交付动作会对真实基础设施产生副作用，新增方法前先确认该域确实需要。
type ProvisioningTaskService interface {
	CreateTaskFromServiceRequest(ctx context.Context, serviceRequestID, tenantID, actorUserID int) (*ent.ProvisioningTask, error)
	ExecuteTask(ctx context.Context, taskID, tenantID, actorUserID int) (*ent.ProvisioningTask, error)
	ListTasksByServiceRequest(ctx context.Context, serviceRequestID, tenantID int) ([]*ent.ProvisioningTask, error)
	ResolveTaskManually(ctx context.Context, taskID, tenantID, actorUserID int, outcome, note string) (*ent.ProvisioningTask, error)
	RetryTask(ctx context.Context, taskID, tenantID, actorUserID int, reason string) (*ent.ProvisioningTask, error)
}
