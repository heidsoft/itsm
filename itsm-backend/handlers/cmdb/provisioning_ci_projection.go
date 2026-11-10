package cmdb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service/cloud"
)

// 交付结果回写配置项。
//
// 与发现回写的区别：发现是「先有云上资源、再补账」，交付是「账随资源一起产生」，
// 所以 source=provisioning，且必须按（provider, 实例 ID）去重——重试、人工重放
// 或下一轮发现都可能再次看到同一台机器，重复建账会让 CMDB 出现多份同名配置项。

const CIProvisioningSourceValue = "provisioning"

// ProvisioningCIProjector 把已成功交付的云资源写成 CMDB 配置项。
// CI 写入规则仍由 ConfigurationItemService 承担，这里只决定落点与字段。
type ProvisioningCIProjector struct {
	repo   Repository
	writer DiscoveryCIWriter
	logger *zap.SugaredLogger
}

func NewProvisioningCIProjector(repo Repository, writer DiscoveryCIWriter, logger *zap.SugaredLogger) *ProvisioningCIProjector {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &ProvisioningCIProjector{repo: repo, writer: writer, logger: logger}
}

var _ dto.ProvisioningCIProjector = (*ProvisioningCIProjector)(nil)

func (p *ProvisioningCIProjector) ProjectProvisionedResource(ctx context.Context, req dto.ProvisioningCIProjectionRequest) dto.ProvisioningCIProjectionResult {
	if p == nil || p.repo == nil || p.writer == nil {
		return provisioningProjection(dto.ProvisioningCIProjectionUnavailable, 0, "projector_not_wired")
	}
	if req.TenantID <= 0 || req.ServiceRequestID <= 0 {
		return provisioningProjection(dto.ProvisioningCIProjectionFailed, 0, "request_without_tenant")
	}
	instanceID := strings.TrimSpace(req.Resources["instance_id"])
	if instanceID == "" {
		return provisioningProjection(dto.ProvisioningCIProjectionSkipped, 0, "provider_result_missing_instance_id")
	}
	// req.ResourceType 是交付任务里的服务码（当前只有 ecs），CI 侧资源类型统一是 instance，
	// 与发现链路使用同一张 ecs|cvm|ec2 → cloud_vm 映射表。
	ciTypeName, ok := CITypeForCloudResource(req.ResourceType, "instance")
	if !ok {
		return provisioningProjection(dto.ProvisioningCIProjectionSkipped, 0,
			fmt.Sprintf("unmapped_resource_type=%s|instance", req.ResourceType))
	}
	ciTypeID, err := p.repo.GetCITypeIDByName(ctx, req.TenantID, ciTypeName)
	if err != nil {
		if ent.IsNotFound(err) {
			return provisioningProjection(dto.ProvisioningCIProjectionSkipped, 0,
				fmt.Sprintf("ci_type_missing=%s(tenant=%d)", ciTypeName, req.TenantID))
		}
		p.logger.Warnw("provisioning ci type lookup failed",
			"tenant_id", req.TenantID, "ci_type", ciTypeName, "error", err)
		return provisioningProjection(dto.ProvisioningCIProjectionFailed, 0, err.Error())
	}

	provider := cloud.NormalizeProvider(req.Provider)
	existing, err := p.repo.FindCIByCloudResourceIdentity(ctx, req.TenantID, provider, instanceID)
	if err != nil && !ent.IsNotFound(err) {
		p.logger.Warnw("provisioning ci lookup failed",
			"tenant_id", req.TenantID, "instance_id", instanceID, "error", err)
		return provisioningProjection(dto.ProvisioningCIProjectionFailed, 0, err.Error())
	}
	if existing != nil {
		return p.updateCI(ctx, req, existing, provider, instanceID)
	}
	return p.createCI(ctx, req, ciTypeID, provider, instanceID)
}

func (p *ProvisioningCIProjector) createCI(ctx context.Context, req dto.ProvisioningCIProjectionRequest, ciTypeID int, provider, instanceID string) dto.ProvisioningCIProjectionResult {
	createReq := p.newCreateCIRequest(req, ciTypeID, provider, instanceID)
	created, err := p.writer.CreateCI(ctx, createReq, req.TenantID)
	if err != nil {
		if errors.Is(err, dto.ErrCIAttributeValidation) {
			p.logger.Infow("provisioning ci projection deferred",
				"tenant_id", req.TenantID, "service_request_id", req.ServiceRequestID,
				"instance_id", instanceID, "ci_type_id", ciTypeID, "error", err)
			return provisioningProjection(dto.ProvisioningCIProjectionDeferred, 0, err.Error())
		}
		p.logger.Warnw("provisioning ci projection failed",
			"tenant_id", req.TenantID, "service_request_id", req.ServiceRequestID,
			"instance_id", instanceID, "error", err)
		return provisioningProjection(dto.ProvisioningCIProjectionFailed, 0, err.Error())
	}
	p.logger.Infow("configuration item created by provisioning",
		"tenant_id", req.TenantID, "ci_id", created.ID,
		"service_request_id", req.ServiceRequestID, "instance_id", instanceID)
	return provisioningProjection(dto.ProvisioningCIProjectionCreated, created.ID, "from_provisioning")
}

func (p *ProvisioningCIProjector) updateCI(ctx context.Context, req dto.ProvisioningCIProjectionRequest, existing *ConfigurationItem, provider, instanceID string) dto.ProvisioningCIProjectionResult {
	updated, err := p.writer.UpdateCI(ctx, existing.ID, req.TenantID, p.newUpdateCIRequest(req, provider, instanceID))
	if err != nil {
		p.logger.Warnw("provisioning ci update failed",
			"tenant_id", req.TenantID, "ci_id", existing.ID,
			"service_request_id", req.ServiceRequestID, "instance_id", instanceID, "error", err)
		return provisioningProjection(dto.ProvisioningCIProjectionFailed, existing.ID, err.Error())
	}
	return provisioningProjection(dto.ProvisioningCIProjectionUpdated, updated.ID, "refreshed")
}

func (p *ProvisioningCIProjector) newCreateCIRequest(req dto.ProvisioningCIProjectionRequest, ciTypeID int, provider, instanceID string) *dto.CreateCIRequest {
	return &dto.CreateCIRequest{
		Name:              provisioningCIName(req, instanceID),
		CITypeID:          ciTypeID,
		Status:            common.CIStatusActive,
		Attributes:        provisioningCIAttributes(req),
		Source:            CIProvisioningSourceValue,
		DiscoverySource:   fmt.Sprintf("provisioning_task:%d", req.ProvisioningTaskID),
		CloudProvider:     provider,
		CloudAccountID:    strings.TrimSpace(req.Resources["account_id"]),
		CloudRegion:       strings.TrimSpace(req.Resources["region_id"]),
		CloudResourceID:   instanceID,
		CloudResourceType: "instance",
		CloudMetadata:     provisioningCloudMetadata(req, instanceID),
		CloudTags:         provisioningCloudTags(req),
		CloudSyncStatus:   CISyncStatusSuccess,
	}
}

// newUpdateCIRequest 只刷新云侧事实；名称、属性、环境等可能已被人工修正的字段不动。
func (p *ProvisioningCIProjector) newUpdateCIRequest(req dto.ProvisioningCIProjectionRequest, provider, instanceID string) *dto.UpdateCIRequest {
	return &dto.UpdateCIRequest{
		Status:            common.CIStatusActive,
		Source:            CIProvisioningSourceValue,
		DiscoverySource:   fmt.Sprintf("provisioning_task:%d", req.ProvisioningTaskID),
		CloudProvider:     provider,
		CloudAccountID:    strings.TrimSpace(req.Resources["account_id"]),
		CloudRegion:       strings.TrimSpace(req.Resources["region_id"]),
		CloudResourceID:   instanceID,
		CloudResourceType: "instance",
		CloudMetadata:     provisioningCloudMetadata(req, instanceID),
		CloudTags:         provisioningCloudTags(req),
		CloudSyncStatus:   CISyncStatusSuccess,
	}
}

func provisioningCIName(req dto.ProvisioningCIProjectionRequest, instanceID string) string {
	for _, key := range []string{"instanceName", "name", "hostname"} {
		if value := projectionStringValue(req.Payload, key); value != "" {
			return value
		}
	}
	return instanceID
}

func provisioningCIAttributes(req dto.ProvisioningCIProjectionRequest) map[string]interface{} {
	attrs := map[string]interface{}{
		"cloudProvider": cloud.NormalizeProvider(req.Provider),
	}
	if region := strings.TrimSpace(req.Resources["region_id"]); region != "" {
		attrs["regionId"] = region
	}
	for _, key := range []string{"instanceType", "imageId"} {
		if value := projectionStringValue(req.Payload, key); value != "" {
			attrs[key] = value
		}
	}
	return attrs
}

// provisioningCloudMetadata 保留云侧回执，便于对账；不包含凭据或请求正文。
func provisioningCloudMetadata(req dto.ProvisioningCIProjectionRequest, instanceID string) map[string]interface{} {
	metadata := map[string]interface{}{}
	if ids := strings.TrimSpace(req.Resources["instance_ids"]); ids != "" {
		metadata["instanceIds"] = ids
	}
	if requestID := strings.TrimSpace(req.Resources["request_id"]); requestID != "" {
		metadata["requestId"] = requestID
	}
	if price := strings.TrimSpace(req.Resources["trade_price"]); price != "" {
		metadata["tradePrice"] = price
	}
	if req.ProvisioningTaskID > 0 {
		metadata["provisioningTaskId"] = req.ProvisioningTaskID
	}
	if instanceID != "" {
		metadata["instanceId"] = instanceID
	}
	return metadata
}

// provisioningCloudTags 记录交付时写入的云标签，和 ECS Provider 的实际打标保持一致。
func provisioningCloudTags(req dto.ProvisioningCIProjectionRequest) map[string]interface{} {
	tags := map[string]interface{}{
		"itsm-tenant-id":            fmt.Sprintf("%d", req.TenantID),
		"itsm-service-request-id":   fmt.Sprintf("%d", req.ServiceRequestID),
		"itsm-provisioning-task-id": fmt.Sprintf("%d", req.ProvisioningTaskID),
	}
	return tags
}

// projectionStringValue 读取 form_data 里的字符串字段；数字也按字符串投影，避免属性类型漂移。
func projectionStringValue(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	default:
		return ""
	}
}

func provisioningProjection(status string, ciID int, reason string) dto.ProvisioningCIProjectionResult {
	return dto.ProvisioningCIProjectionResult{Status: status, CIID: ciID, Reason: reason}
}
