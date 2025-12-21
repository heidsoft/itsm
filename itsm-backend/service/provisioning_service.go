package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/domain/provisioning"
	"itsm-backend/ent"
	"itsm-backend/ent/provisioningtask"
	"itsm-backend/ent/servicerequest"
	"itsm-backend/infrastructure/cloud"
	cloudAlicloud "itsm-backend/infrastructure/cloud/alicloud"

	"go.uber.org/zap"
)

// ProvisioningService（应用层）：服务请求 -> 交付任务 -> 执行 -> 状态回写
// M2：先实现可运行骨架（Stub），后续接入阿里云真实交付。
type ProvisioningService struct {
	client   *ent.Client
	logger   *zap.SugaredLogger
	provider cloud.Provider
}

func NewProvisioningService(client *ent.Client, logger *zap.SugaredLogger) *ProvisioningService {
	return &ProvisioningService{
		client:   client,
		logger:   logger,
		provider: cloudAlicloud.NewStubProvider(),
	}
}

// CreateTaskFromServiceRequest 仅创建交付任务并把 ServiceRequest 置为 provisioning
func (s *ProvisioningService) CreateTaskFromServiceRequest(ctx context.Context, serviceRequestID, tenantID, actorUserID int) (*ent.ProvisioningTask, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	sr, err := tx.ServiceRequest.Query().
		Where(servicerequest.ID(serviceRequestID), servicerequest.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("服务请求不存在")
	}
	if string(sr.Status) != SRStatusSecurityApproved {
		return nil, fmt.Errorf("当前状态不允许启动交付（需要 security_approved）")
	}

	// payload：优先使用 form_data；若为空则降级为空 map
	payload := map[string]any{}
	if sr.FormData != nil {
		payload = sr.FormData
	}

	// 最小推断资源类型（后续基于 catalog/category 或专用字段）
	resourceType := string(provisioning.ResourceECS)
	if v, ok := payload["resource_type"]; ok {
		resourceType = strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
	}

	task, err := tx.ProvisioningTask.Create().
		SetTenantID(tenantID).
		SetServiceRequestID(serviceRequestID).
		SetProvider(string(provisioning.ProviderAlicloud)).
		SetResourceType(resourceType).
		SetPayload(payload).
		SetStatus(string(provisioning.TaskPending)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建交付任务失败: %w", err)
	}

	// 回写 ServiceRequest 状态为 provisioning
	if err := tx.ServiceRequest.Update().
		Where(servicerequest.ID(serviceRequestID), servicerequest.TenantID(tenantID)).
		SetStatus("provisioning").
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("更新服务请求状态失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}
	return task, nil
}

// ExecuteTask 执行交付任务（Stub），并回写 ServiceRequest 状态为 delivered/failed
func (s *ProvisioningService) ExecuteTask(ctx context.Context, taskID, tenantID, actorUserID int) (*ent.ProvisioningTask, error) {
	// 读任务
	task, err := s.client.ProvisioningTask.Query().
		Where(provisioningtask.ID(taskID), provisioningtask.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("交付任务不存在")
	}

	// 设置 running
	task, err = s.client.ProvisioningTask.UpdateOneID(task.ID).
		Where(provisioningtask.TenantID(tenantID)).
		SetStatus(string(provisioning.TaskRunning)).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 执行（stub）
	payload := map[string]any{}
	if task.Payload != nil {
		payload = task.Payload
	}
	res, execErr := s.provider.Execute(ctx, payload)

	// 回写任务状态 + ServiceRequest
	if execErr != nil {
		_ = s.client.ProvisioningTask.UpdateOneID(task.ID).
			Where(provisioningtask.TenantID(tenantID)).
			SetStatus(string(provisioning.TaskFailed)).
			SetErrorMessage(execErr.Error()).
			Exec(ctx)
		_ = s.client.ServiceRequest.Update().
			Where(servicerequest.ID(task.ServiceRequestID), servicerequest.TenantID(tenantID)).
			SetStatus("failed").
			SetLastError(execErr.Error()).
			Exec(ctx)
		return nil, execErr
	}

	result := map[string]any{}
	if res != nil {
		result["resources"] = res.Resources
	}
	if err := s.client.ProvisioningTask.UpdateOneID(task.ID).
		Where(provisioningtask.TenantID(tenantID)).
		SetStatus(string(provisioning.TaskSucceeded)).
		SetResult(result).
		SetErrorMessage("").
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("回写任务结果失败: %w", err)
	}
	_ = s.client.ServiceRequest.Update().
		Where(servicerequest.ID(task.ServiceRequestID), servicerequest.TenantID(tenantID)).
		SetStatus("delivered").
		Exec(ctx)

	// 返回最新任务
	return s.client.ProvisioningTask.Get(ctx, task.ID)
}

func (s *ProvisioningService) ListTasksByServiceRequest(ctx context.Context, serviceRequestID, tenantID int) ([]*ent.ProvisioningTask, error) {
	return s.client.ProvisioningTask.Query().
		Where(provisioningtask.TenantID(tenantID), provisioningtask.ServiceRequestID(serviceRequestID)).
		Order(ent.Desc(provisioningtask.FieldCreatedAt)).
		All(ctx)
}


