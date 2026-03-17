package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/ent/cloudaccount"
	"itsm-backend/ent/cloudservice"
	"itsm-backend/ent/cloudresource"
)

// CloudDiscoveryService 云资源发现服务
type CloudDiscoveryService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewCloudDiscoveryService 创建云资源发现服务
func NewCloudDiscoveryService(client *ent.Client, logger *zap.SugaredLogger) *CloudDiscoveryService {
	return &CloudDiscoveryService{
		client: client,
		logger: logger,
	}
}

// DiscoveryResult 发现结果
type DiscoveryResult struct {
	ServiceType string
	Resources   []DiscoveredResource
}

// DiscoveredResource 发现资源
type DiscoveredResource struct {
	ResourceID   string
	ResourceName string
	Region       string
	Zone         string
	Status       string
	Tags         map[string]string
	Metadata     map[string]interface{}
}

// DiscoverAll 执行全量云资源发现
func (s *CloudDiscoveryService) DiscoverAll(ctx context.Context, tenantID int) error {
	s.logger.Infow("Starting cloud resource discovery", "tenantID", tenantID)

	// 获取所有启用的云账号
	accounts, err := s.client.CloudAccount.Query().
		Where(
			cloudaccount.TenantID(tenantID),
			cloudaccount.IsActive(true),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("查询云账号失败: %w", err)
	}

	s.logger.Infow("Found cloud accounts", "count", len(accounts))

	// 对每个云账号执行发现
	for _, account := range accounts {
		if err := s.DiscoverAccount(ctx, account); err != nil {
			s.logger.Warnw("Discovery failed for account", "account", account.ID, "error", err)
			continue
		}
	}

	return nil
}

// DiscoverAccount 发现单个云账号的资源
func (s *CloudDiscoveryService) DiscoverAccount(ctx context.Context, account *ent.CloudAccount) error {
	s.logger.Infow("Discovering resources for account", "accountID", account.ID, "provider", account.Provider)

	switch account.Provider {
	case "aws":
		return s.discoverAWS(ctx, account)
	case "azure":
		return s.discoverAzure(ctx, account)
	case "aliyun":
		return s.discoverAliyun(ctx, account)
	case "tencent":
		return s.discoverTencent(ctx, account)
	case "huawei":
		return s.discoverHuawei(ctx, account)
	default:
		s.logger.Warnw("Unsupported cloud provider", "provider", account.Provider)
		return fmt.Errorf("不支持的云提供商: %s", account.Provider)
	}
}

// discoverAWS 发现AWS资源
func (s *CloudDiscoveryService) discoverAWS(ctx context.Context, account *ent.CloudAccount) error {
	results := []DiscoveryResult{}

	// 发现EC2实例
	results = append(results, DiscoveryResult{
		ServiceType: "ec2",
		Resources:   s.discoverEC2(ctx, account),
	})

	// 发现S3存储桶
	results = append(results, DiscoveryResult{
		ServiceType: "s3",
		Resources:   s.discoverS3(ctx, account),
	})

	// 发现RDS实例
	results = append(results, DiscoveryResult{
		ServiceType: "rds",
		Resources:   s.discoverRDS(ctx, account),
	})

	// 保存发现结果
	for _, result := range results {
		service, err := s.getOrCreateCloudService(ctx, result.ServiceType, "aws")
		if err != nil {
			continue
		}

		for _, resource := range result.Resources {
			if err := s.upsertCloudResource(ctx, account, service, resource); err != nil {
				s.logger.Warnw("Failed to upsert resource", "resourceID", resource.ResourceID, "error", err)
			}
		}
	}

	return nil
}

// discoverAzure 发现Azure资源
func (s *CloudDiscoveryService) discoverAzure(ctx context.Context, account *ent.CloudAccount) error {
	results := []DiscoveryResult{}
	results = append(results, DiscoveryResult{
		ServiceType: "vm",
		Resources:   s.discoverAzureVM(ctx, account),
	})

	results = append(results, DiscoveryResult{
		ServiceType: "storage",
		Resources:   s.discoverAzureStorage(ctx, account),
	})

	// 保存发现结果
	for _, result := range results {
		service, err := s.getOrCreateCloudService(ctx, result.ServiceType, "azure")
		if err != nil {
			continue
		}

		for _, resource := range result.Resources {
			if err := s.upsertCloudResource(ctx, account, service, resource); err != nil {
				s.logger.Warnw("Failed to upsert resource", "resourceID", resource.ResourceID, "error", err)
			}
		}
	}

	return nil
}

// discoverAliyun 发现阿里云资源
func (s *CloudDiscoveryService) discoverAliyun(ctx context.Context, account *ent.CloudAccount) error {
	s.logger.Info("Aliyun discovery not fully implemented - requires SDK configuration")
	return nil
}

// discoverTencent 发现腾讯云资源
func (s *CloudDiscoveryService) discoverTencent(ctx context.Context, account *ent.CloudAccount) error {
	s.logger.Info("Tencent Cloud discovery not fully implemented - requires SDK configuration")
	return nil
}

// discoverHuawei 发现华为云资源
func (s *CloudDiscoveryService) discoverHuawei(ctx context.Context, account *ent.CloudAccount) error {
	s.logger.Info("Huawei Cloud discovery not fully implemented - requires SDK configuration")
	return nil
}

// discoverEC2 发现EC2实例 (模拟实现)
func (s *CloudDiscoveryService) discoverEC2(ctx context.Context, account *ent.CloudAccount) []DiscoveredResource {
	// TODO: 实现真实的AWS API调用
	// 在实际环境中，需要:
	// 1. 使用AWS SDK配置凭证
	// 2. 调用ec2.DescribeInstances
	// 3. 转换结果

	s.logger.Info("EC2 discovery not implemented - requires AWS credentials")
	return []DiscoveredResource{}
}

// discoverS3 发现S3存储桶 (模拟实现)
func (s *CloudDiscoveryService) discoverS3(ctx context.Context, account *ent.CloudAccount) []DiscoveredResource {
	s.logger.Info("S3 discovery not implemented - requires AWS credentials")
	return []DiscoveredResource{}
}

// discoverRDS 发现RDS实例 (模拟实现)
func (s *CloudDiscoveryService) discoverRDS(ctx context.Context, account *ent.CloudAccount) []DiscoveredResource {
	s.logger.Info("RDS discovery not implemented - requires AWS credentials")
	return []DiscoveredResource{}
}

// discoverAzureVM 发现Azure虚拟机 (模拟实现)
func (s *CloudDiscoveryService) discoverAzureVM(ctx context.Context, account *ent.CloudAccount) []DiscoveredResource {
	s.logger.Info("Azure VM discovery not implemented - requires Azure credentials")
	return []DiscoveredResource{}
}

// discoverAzureStorage 发现Azure存储 (模拟实现)
func (s *CloudDiscoveryService) discoverAzureStorage(ctx context.Context, account *ent.CloudAccount) []DiscoveredResource {
	s.logger.Info("Azure Storage discovery not implemented - requires Azure credentials")
	return []DiscoveredResource{}
}

// getOrCreateCloudService 获取或创建云服务
func (s *CloudDiscoveryService) getOrCreateCloudService(ctx context.Context, serviceCode, provider string) (*ent.CloudService, error) {
	service, err := s.client.CloudService.Query().
		Where(
			cloudservice.ServiceCode(serviceCode),
			cloudservice.Provider(provider),
		).
		Only(ctx)

	if err == nil {
		return service, nil
	}

	if !ent.IsNotFound(err) {
		return nil, err
	}

	// 创建新服务
	return s.client.CloudService.Create().
		SetProvider(provider).
		SetServiceCode(serviceCode).
		SetServiceName(serviceCode).
		SetResourceTypeCode(serviceCode).
		SetResourceTypeName(serviceCode).
		Save(ctx)
}

// upsertCloudResource 更新或创建云资源
func (s *CloudDiscoveryService) upsertCloudResource(
	ctx context.Context,
	account *ent.CloudAccount,
	service *ent.CloudService,
	resource DiscoveredResource,
) error {
	now := time.Now()

	// 检查资源是否已存在
	existing, err := s.client.CloudResource.Query().
		Where(
			cloudresource.CloudAccountID(account.ID),
			cloudresource.ServiceID(service.ID),
			cloudresource.ResourceID(resource.ResourceID),
		).
		Only(ctx)

	if err == nil {
		// 更新现有资源
		_, err = existing.Update().
			SetResourceName(resource.ResourceName).
			SetRegion(resource.Region).
			SetZone(resource.Zone).
			SetStatus(resource.Status).
			SetTags(resource.Tags).
			SetMetadata(resource.Metadata).
			SetLastSeenAt(now).
			Save(ctx)
		return err
	}

	if !ent.IsNotFound(err) {
		return err
	}

	// 创建新资源
	_, err = s.client.CloudResource.Create().
		SetCloudAccountID(account.ID).
		SetServiceID(service.ID).
		SetResourceID(resource.ResourceID).
		SetResourceName(resource.ResourceName).
		SetRegion(resource.Region).
		SetZone(resource.Zone).
		SetStatus(resource.Status).
		SetTags(resource.Tags).
		SetMetadata(resource.Metadata).
		SetFirstSeenAt(now).
		SetLastSeenAt(now).
		SetTenantID(account.TenantID).
		Save(ctx)

	return err
}

// GetDiscoveryStatus 获取发现任务状态
func (s *CloudDiscoveryService) GetDiscoveryStatus(ctx context.Context, tenantID int) (map[string]interface{}, error) {
	accounts, err := s.client.CloudAccount.Query().
		Where(cloudaccount.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	status := map[string]interface{}{
		"total_accounts":   len(accounts),
		"active_accounts":  0,
		"last_discovery":   time.Now().Add(-24 * time.Hour),
		"discovered_count": 0,
	}

	for _, account := range accounts {
		if account.IsActive {
			status["active_accounts"] = status["active_accounts"].(int) + 1
		}
	}

	// 统计已发现资源
	count, err := s.client.CloudResource.Query().
		Where(cloudresource.TenantID(tenantID)).
		Count(ctx)
	if err == nil {
		status["discovered_count"] = count
	}

	return status, nil
}
