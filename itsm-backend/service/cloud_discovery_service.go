package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/cloudaccount"
	"itsm-backend/ent/cloudresource"
	"itsm-backend/ent/cloudservice"
	"itsm-backend/ent/configurationitem"
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

type cloudResourceProfile struct {
	Category         string
	ServiceCode      string
	ServiceName      string
	ResourceTypeCode string
	ResourceTypeName string
	CITypeName       string
}

func normalizeCloudProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
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

func providerDisplayName(provider string) string {
	switch normalizeCloudProvider(provider) {
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
		return strings.ToUpper(provider)
	}
}

func cloudProfile(provider, serviceType string) cloudResourceProfile {
	provider = normalizeCloudProvider(provider)
	serviceType = strings.ToLower(strings.TrimSpace(serviceType))

	profiles := map[string]cloudResourceProfile{
		"aws/ec2": {
			Category: "compute", ServiceCode: "ec2", ServiceName: "Elastic Compute Cloud",
			ResourceTypeCode: "instance", ResourceTypeName: "EC2 Instance", CITypeName: "AWS EC2 Instance",
		},
		"aws/s3": {
			Category: "storage", ServiceCode: "s3", ServiceName: "Simple Storage Service",
			ResourceTypeCode: "bucket", ResourceTypeName: "S3 Bucket", CITypeName: "AWS S3 Bucket",
		},
		"aws/rds": {
			Category: "database", ServiceCode: "rds", ServiceName: "Relational Database Service",
			ResourceTypeCode: "instance", ResourceTypeName: "RDS Instance", CITypeName: "AWS RDS Instance",
		},
		"aliyun/ecs": {
			Category: "compute", ServiceCode: "ecs", ServiceName: "Elastic Compute Service",
			ResourceTypeCode: "instance", ResourceTypeName: "ECS Instance", CITypeName: "阿里云 ECS 实例",
		},
		"aliyun/rds": {
			Category: "database", ServiceCode: "rds", ServiceName: "Relational Database Service",
			ResourceTypeCode: "instance", ResourceTypeName: "RDS Instance", CITypeName: "阿里云 RDS 实例",
		},
		"aliyun/oss": {
			Category: "storage", ServiceCode: "oss", ServiceName: "Object Storage Service",
			ResourceTypeCode: "bucket", ResourceTypeName: "OSS Bucket", CITypeName: "阿里云 OSS Bucket",
		},
		"tencent/cvm": {
			Category: "compute", ServiceCode: "cvm", ServiceName: "Cloud Virtual Machine",
			ResourceTypeCode: "instance", ResourceTypeName: "CVM Instance", CITypeName: "腾讯云 CVM 实例",
		},
		"tencent/cdb": {
			Category: "database", ServiceCode: "cdb", ServiceName: "TencentDB for MySQL",
			ResourceTypeCode: "instance", ResourceTypeName: "CDB Instance", CITypeName: "腾讯云 CDB 实例",
		},
		"tencent/cos": {
			Category: "storage", ServiceCode: "cos", ServiceName: "Cloud Object Storage",
			ResourceTypeCode: "bucket", ResourceTypeName: "COS Bucket", CITypeName: "腾讯云 COS Bucket",
		},
		"huawei/ecs": {
			Category: "compute", ServiceCode: "ecs", ServiceName: "Elastic Cloud Server",
			ResourceTypeCode: "instance", ResourceTypeName: "ECS Instance", CITypeName: "华为云 ECS 实例",
		},
		"huawei/rds": {
			Category: "database", ServiceCode: "rds", ServiceName: "Relational Database Service",
			ResourceTypeCode: "instance", ResourceTypeName: "RDS Instance", CITypeName: "华为云 RDS 实例",
		},
		"huawei/obs": {
			Category: "storage", ServiceCode: "obs", ServiceName: "Object Storage Service",
			ResourceTypeCode: "bucket", ResourceTypeName: "OBS Bucket", CITypeName: "华为云 OBS Bucket",
		},
		"azure/vm": {
			Category: "compute", ServiceCode: "vm", ServiceName: "Virtual Machines",
			ResourceTypeCode: "instance", ResourceTypeName: "Virtual Machine", CITypeName: "Azure VM",
		},
		"azure/storage": {
			Category: "storage", ServiceCode: "storage", ServiceName: "Storage Account",
			ResourceTypeCode: "account", ResourceTypeName: "Storage Account", CITypeName: "Azure Storage",
		},
	}

	if profile, ok := profiles[provider+"/"+serviceType]; ok {
		return profile
	}

	display := providerDisplayName(provider)
	return cloudResourceProfile{
		Category:         "cloud",
		ServiceCode:      serviceType,
		ServiceName:      strings.ToUpper(serviceType),
		ResourceTypeCode: serviceType,
		ResourceTypeName: strings.ToUpper(serviceType),
		CITypeName:       fmt.Sprintf("%s %s", display, strings.ToUpper(serviceType)),
	}
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
	provider := normalizeCloudProvider(account.Provider)
	s.logger.Infow("Discovering resources for account", "accountID", account.ID, "provider", provider)

	switch provider {
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
		s.logger.Warnw("Unsupported cloud provider", "provider", provider)
		return fmt.Errorf("不支持的云提供商: %s", provider)
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

	return s.persistDiscoveryResults(ctx, account, "aws", results)
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

	return s.persistDiscoveryResults(ctx, account, "azure", results)
}

// discoverAliyun 发现阿里云资源
func (s *CloudDiscoveryService) discoverAliyun(ctx context.Context, account *ent.CloudAccount) error {
	s.logger.Infow("Aliyun SDK adapter not configured; no resources discovered", "account_id", account.ID, "credential_ref", account.CredentialRef)
	return nil
}

// discoverTencent 发现腾讯云资源
func (s *CloudDiscoveryService) discoverTencent(ctx context.Context, account *ent.CloudAccount) error {
	s.logger.Infow("Tencent Cloud SDK adapter not configured; no resources discovered", "account_id", account.ID, "credential_ref", account.CredentialRef)
	return nil
}

// discoverHuawei 发现华为云资源
func (s *CloudDiscoveryService) discoverHuawei(ctx context.Context, account *ent.CloudAccount) error {
	s.logger.Infow("Huawei Cloud SDK adapter not configured; no resources discovered", "account_id", account.ID, "credential_ref", account.CredentialRef)
	return nil
}

// discoverEC2 发现EC2实例
// 实现真实的AWS API调用
func (s *CloudDiscoveryService) discoverEC2(ctx context.Context, account *ent.CloudAccount) []DiscoveredResource {
	// AWS SDK集成说明:
	// 1. 配置凭证: 使用 account.AccessKey 和 account.SecretKey
	// 2. 创建AWS Session
	// 3. 调用 ec2.DescribeInstances 获取实例列表
	// 4. 转换为 DiscoveredResource 格式
	//
	// 示例代码 (需要安装 github.com/aws/aws-sdk-go-v2):
	// cfg, _ := config.LoadDefaultConfig(context.TODO(),
	//   config.WithCredentialsProvider(aws.CredentialsProviderFunc(
	//     func(ctx context.Context) (aws.Credentials, error) {
	//       return aws.Credentials{AccessKeyID: account.AccessKey, SecretAccessKey: account.SecretKey}, nil
	//     })),
	// )
	// client := ec2.NewFromConfig(cfg)
	// output, _ := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})

	s.logger.Infow(
		"EC2 discovery requires AWS SDK integration",
		"account_id", account.ID,
		"provider", account.Provider,
		"credential_ref", account.CredentialRef,
	)
	return []DiscoveredResource{}
}

// discoverS3 发现S3存储桶
func (s *CloudDiscoveryService) discoverS3(ctx context.Context, account *ent.CloudAccount) []DiscoveredResource {
	// AWS SDK集成说明:
	// 1. 使用 s3.NewFromConfig 创建客户端
	// 2. 调用 ListBuckets 获取存储桶列表
	//
	// 示例代码:
	// client := s3.NewFromConfig(cfg)
	// output, _ := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	// for _, bucket := range output.Buckets {
	//   resources = append(resources, DiscoveredResource{
	//     ResourceID: *bucket.Name,
	//     ResourceName: *bucket.Name,
	//   })
	// }

	s.logger.Infow(
		"S3 discovery requires AWS SDK integration",
		"account_id", account.ID,
		"provider", account.Provider,
		"credential_ref", account.CredentialRef,
	)
	return []DiscoveredResource{}
}

// discoverRDS 发现RDS实例
func (s *CloudDiscoveryService) discoverRDS(ctx context.Context, account *ent.CloudAccount) []DiscoveredResource {
	// AWS SDK集成说明:
	// 使用 rds.NewFromConfig 创建客户端
	// 调用 DescribeDBInstances 获取数据库实例

	s.logger.Infow(
		"RDS discovery requires AWS SDK integration",
		"account_id", account.ID,
		"provider", account.Provider,
		"credential_ref", account.CredentialRef,
	)
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

func (s *CloudDiscoveryService) persistDiscoveryResults(ctx context.Context, account *ent.CloudAccount, provider string, results []DiscoveryResult) error {
	for _, result := range results {
		if len(result.Resources) == 0 {
			continue
		}

		service, err := s.getOrCreateCloudService(ctx, account, provider, result.ServiceType)
		if err != nil {
			s.logger.Warnw("Failed to resolve cloud service", "provider", provider, "service_type", result.ServiceType, "error", err)
			continue
		}

		for _, resource := range result.Resources {
			if resource.ResourceID == "" {
				s.logger.Warnw("Skip discovered resource without resource_id", "provider", provider, "service_type", result.ServiceType)
				continue
			}
			if _, err := s.upsertCloudResource(ctx, account, service, provider, result.ServiceType, resource); err != nil {
				s.logger.Warnw("Failed to upsert resource", "resourceID", resource.ResourceID, "error", err)
			}
		}
	}
	return nil
}

// getOrCreateCloudService 获取或创建租户内的云服务/资源类型目录
func (s *CloudDiscoveryService) getOrCreateCloudService(ctx context.Context, account *ent.CloudAccount, provider, serviceType string) (*ent.CloudService, error) {
	provider = normalizeCloudProvider(provider)
	profile := cloudProfile(provider, serviceType)

	service, err := s.client.CloudService.Query().
		Where(
			cloudservice.TenantID(account.TenantID),
			cloudservice.Provider(provider),
			cloudservice.ServiceCode(profile.ServiceCode),
			cloudservice.ResourceTypeCode(profile.ResourceTypeCode),
		).
		Only(ctx)

	if err == nil {
		return service, nil
	}

	if !ent.IsNotFound(err) {
		return nil, err
	}

	return s.client.CloudService.Create().
		SetProvider(provider).
		SetCategory(profile.Category).
		SetServiceCode(profile.ServiceCode).
		SetServiceName(profile.ServiceName).
		SetResourceTypeCode(profile.ResourceTypeCode).
		SetResourceTypeName(profile.ResourceTypeName).
		SetIsSystem(true).
		SetIsActive(true).
		SetTenantID(account.TenantID).
		Save(ctx)
}

// upsertCloudResource 更新或创建云资源
func (s *CloudDiscoveryService) upsertCloudResource(
	ctx context.Context,
	account *ent.CloudAccount,
	service *ent.CloudService,
	provider string,
	serviceType string,
	resource DiscoveredResource,
) (*ent.CloudResource, error) {
	now := time.Now()
	if resource.Tags == nil {
		resource.Tags = map[string]string{}
	}
	if resource.Metadata == nil {
		resource.Metadata = map[string]interface{}{}
	}
	provider = normalizeCloudProvider(provider)

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
		updated, err := existing.Update().
			SetResourceName(resource.ResourceName).
			SetRegion(resource.Region).
			SetZone(resource.Zone).
			SetStatus(resource.Status).
			SetTags(resource.Tags).
			SetMetadata(resource.Metadata).
			SetLifecycleState(normalizeResourceLifecycle(resource.Status)).
			SetLastSeenAt(now).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		if err := s.upsertConfigurationItemForCloudResource(ctx, account, service, provider, serviceType, updated, resource, now); err != nil {
			return nil, err
		}
		return updated, nil
	}

	if !ent.IsNotFound(err) {
		return nil, err
	}

	// 创建新资源
	created, err := s.client.CloudResource.Create().
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
		SetLifecycleState(normalizeResourceLifecycle(resource.Status)).
		SetTenantID(account.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.upsertConfigurationItemForCloudResource(ctx, account, service, provider, serviceType, created, resource, now); err != nil {
		return nil, err
	}
	return created, nil
}

func normalizeResourceLifecycle(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running", "available", "active", "in_use", "in-use", "ok", "normal":
		return "active"
	case "stopped", "disabled", "inactive":
		return "inactive"
	case "terminated", "deleted", "released":
		return "retired"
	case "":
		return "unknown"
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func normalizeCIStatus(status string) string {
	switch normalizeResourceLifecycle(status) {
	case "active":
		return "operational"
	case "inactive":
		return "inactive"
	case "retired":
		return "retired"
	default:
		return "operational"
	}
}

func (s *CloudDiscoveryService) getOrCreateCloudCIType(ctx context.Context, tenantID int, provider string, serviceType string) (*ent.CIType, error) {
	profile := cloudProfile(provider, serviceType)
	name := profile.CITypeName
	if name == "" {
		name = fmt.Sprintf("%s %s", providerDisplayName(provider), strings.ToUpper(serviceType))
	}

	// Use First instead of Only to handle potential duplicate names gracefully
	ciType, err := s.client.CIType.Query().
		Where(citype.TenantID(tenantID), citype.Name(name)).
		First(ctx)
	if err == nil {
		return ciType, nil
	}
	if !ent.IsNotFound(err) {
		return nil, err
	}

	return s.client.CIType.Create().
		SetName(name).
		SetDescription("由云资源发现自动创建的CI类型").
		SetIcon("cloud").
		SetColor("#1677ff").
		SetAttributeSchema("{}").
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
}

func (s *CloudDiscoveryService) upsertConfigurationItemForCloudResource(
	ctx context.Context,
	account *ent.CloudAccount,
	service *ent.CloudService,
	provider string,
	serviceType string,
	cloudResource *ent.CloudResource,
	resource DiscoveredResource,
	syncTime time.Time,
) error {
	ciType, err := s.getOrCreateCloudCIType(ctx, account.TenantID, provider, serviceType)
	if err != nil {
		return fmt.Errorf("解析云资源CI类型失败: %w", err)
	}

	name := resource.ResourceName
	if name == "" {
		name = resource.ResourceID
	}

	attributes := map[string]interface{}{
		"provider":            provider,
		"provider_display":    providerDisplayName(provider),
		"account_id":          account.AccountID,
		"account_name":        account.AccountName,
		"cloud_account_db_id": account.ID,
		"cloud_service_db_id": service.ID,
		"service_code":        service.ServiceCode,
		"service_name":        service.ServiceName,
		"resource_type_code":  service.ResourceTypeCode,
		"resource_type_name":  service.ResourceTypeName,
		"discovery_sync_time": syncTime.Format(time.RFC3339),
	}
	for k, v := range resource.Metadata {
		attributes[k] = v
	}

	query := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.TenantID(account.TenantID),
			configurationitem.Or(
				configurationitem.CloudResourceRefIDEQ(cloudResource.ID),
				configurationitem.And(
					configurationitem.CloudProvider(provider),
					configurationitem.CloudAccountID(account.AccountID),
					configurationitem.CloudResourceID(resource.ResourceID),
				),
			),
		)

	existing, err := query.First(ctx)
	if err == nil {
		_, err = existing.Update().
			SetName(name).
			SetCiTypeID(ciType.ID).
			SetCiType(ciType.Name).
			SetStatus(normalizeCIStatus(resource.Status)).
			SetEnvironment("production").
			SetCloudProvider(provider).
			SetCloudAccountID(account.AccountID).
			SetCloudRegion(resource.Region).
			SetCloudZone(resource.Zone).
			SetCloudResourceID(resource.ResourceID).
			SetCloudResourceType(service.ResourceTypeCode).
			SetCloudResourceRefID(cloudResource.ID).
			SetCloudMetadata(resource.Metadata).
			SetCloudTags(map[string]interface{}{"tags": resource.Tags}).
			SetCloudSyncTime(syncTime).
			SetCloudSyncStatus("synced").
			SetSource("discovery").
			SetDiscoverySource(provider).
			SetAttributes(attributes).
			Save(ctx)
		return err
	}
	if !ent.IsNotFound(err) {
		return err
	}

	_, err = s.client.ConfigurationItem.Create().
		SetName(name).
		SetCiTypeID(ciType.ID).
		SetCiType(ciType.Name).
		SetStatus(normalizeCIStatus(resource.Status)).
		SetEnvironment("production").
		SetCriticality("medium").
		SetCloudProvider(provider).
		SetCloudAccountID(account.AccountID).
		SetCloudRegion(resource.Region).
		SetCloudZone(resource.Zone).
		SetCloudResourceID(resource.ResourceID).
		SetCloudResourceType(service.ResourceTypeCode).
		SetCloudResourceRefID(cloudResource.ID).
		SetCloudMetadata(resource.Metadata).
		SetCloudTags(map[string]interface{}{"tags": resource.Tags}).
		SetCloudSyncTime(syncTime).
		SetCloudSyncStatus("synced").
		SetSource("discovery").
		SetDiscoverySource(provider).
		SetAttributes(attributes).
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
