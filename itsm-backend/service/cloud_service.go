package service

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cloudaccount"
	"itsm-backend/ent/cloudresource"
	"itsm-backend/ent/cloudservice"

	"go.uber.org/zap"
)

// CloudService 云服务
type CloudService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewCloudService 创建云服务
func NewCloudService(client *ent.Client, logger *zap.SugaredLogger) *CloudService {
	return &CloudService{
		client: client,
		logger: logger,
	}
}

// ===================================
// CloudAccount Methods
// ===================================

// CreateCloudAccount 创建云账号
func (s *CloudService) CreateCloudAccount(ctx context.Context, tenantID int, req *dto.CreateCloudAccountRequest) (*ent.CloudAccount, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	account, err := s.client.CloudAccount.Create().
		SetProvider(req.Provider).
		SetAccountID(req.AccountID).
		SetAccountName(req.AccountName).
		SetCredentialRef(req.CredentialRef).
		SetRegionWhitelist(req.RegionWhitelist).
		SetIsActive(isActive).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorf("创建云账号失败: %v", err)
		return nil, err
	}

	return account, nil
}

// GetCloudAccount 获取云账号详情
func (s *CloudService) GetCloudAccount(ctx context.Context, tenantID, id int) (*ent.CloudAccount, error) {
	account, err := s.client.CloudAccount.Query().
		Where(cloudaccount.ID(id), cloudaccount.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		s.logger.Errorf("获取云账号失败: %v", err)
		return nil, err
	}
	return account, nil
}

// UpdateCloudAccount 更新云账号
func (s *CloudService) UpdateCloudAccount(ctx context.Context, tenantID, id int, req *dto.UpdateCloudAccountRequest) (*ent.CloudAccount, error) {
	account, err := s.client.CloudAccount.Query().
		Where(cloudaccount.ID(id), cloudaccount.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		s.logger.Errorf("查询云账号失败: %v", err)
		return nil, err
	}

	update := account.Update()
	if req.AccountName != nil {
		update.SetAccountName(*req.AccountName)
	}
	if req.CredentialRef != nil {
		update.SetCredentialRef(*req.CredentialRef)
	}
	if req.RegionWhitelist != nil {
		update.SetRegionWhitelist(req.RegionWhitelist)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorf("更新云账号失败: %v", err)
		return nil, err
	}

	return updated, nil
}

// DeleteCloudAccount 删除云账号
func (s *CloudService) DeleteCloudAccount(ctx context.Context, tenantID, id int) error {
	_, err := s.client.CloudAccount.Delete().
		Where(cloudaccount.ID(id), cloudaccount.TenantID(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorf("删除云账号失败: %v", err)
		return err
	}
	return nil
}

// ListCloudAccounts 获取云账号列表
func (s *CloudService) ListCloudAccounts(ctx context.Context, tenantID int, req *dto.ListCloudAccountsRequest) ([]*ent.CloudAccount, int, error) {
	query := s.client.CloudAccount.Query().
		Where(cloudaccount.TenantID(tenantID))

	// 过滤条件
	if req.Provider != "" {
		query = query.Where(cloudaccount.Provider(req.Provider))
	}
	if req.IsActive != nil {
		query = query.Where(cloudaccount.IsActive(*req.IsActive))
	}
	if req.Search != "" {
		query = query.Where(
			cloudaccount.AccountIDContains(req.Search),
		)
	}

	// 统计总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorf("统计云账号总数失败: %v", err)
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	accounts, err := query.
		Order(ent.Desc(cloudaccount.FieldUpdatedAt)).
		Offset(offset).
		Limit(req.PageSize).
		All(ctx)
	if err != nil {
		s.logger.Errorf("查询云账号列表失败: %v", err)
		return nil, 0, err
	}

	return accounts, total, nil
}

// ===================================
// CloudService CRUD Methods
// ===================================

// CreateCloudService 创建云服务
func (s *CloudService) CreateCloudService(ctx context.Context, tenantID int, req *dto.CreateCloudServiceRequest) (*ent.CloudService, error) {
	isSystem := false
	isActive := true
	if req.IsSystem != nil {
		isSystem = *req.IsSystem
	}
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	service, err := s.client.CloudService.Create().
		SetProvider(req.Provider).
		SetCategory(req.Category).
		SetServiceCode(req.ServiceCode).
		SetServiceName(req.ServiceName).
		SetResourceTypeCode(req.ResourceTypeCode).
		SetResourceTypeName(req.ResourceTypeName).
		SetAPIVersion(req.APIVersion).
		SetAttributeSchema(req.AttributeSchema).
		SetIsSystem(isSystem).
		SetIsActive(isActive).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorf("创建云服务失败: %v", err)
		return nil, err
	}

	// 如果有父级服务，建立关联
	if req.ParentID != nil {
		_, err = s.client.CloudService.UpdateOne(service).
			SetParentID(*req.ParentID).
			Save(ctx)
		if err != nil {
			s.logger.Warnf("设置云服务父级失败: %v", err)
		}
	}

	return service, nil
}

// GetCloudService 获取云服务详情
func (s *CloudService) GetCloudService(ctx context.Context, tenantID, id int) (*ent.CloudService, error) {
	service, err := s.client.CloudService.Query().
		Where(cloudservice.ID(id), cloudservice.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		s.logger.Errorf("获取云服务失败: %v", err)
		return nil, err
	}
	return service, nil
}

// UpdateCloudService 更新云服务
func (s *CloudService) UpdateCloudService(ctx context.Context, tenantID, id int, req *dto.UpdateCloudServiceRequest) (*ent.CloudService, error) {
	service, err := s.client.CloudService.Query().
		Where(cloudservice.ID(id), cloudservice.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		s.logger.Errorf("查询云服务失败: %v", err)
		return nil, err
	}

	update := service.Update()
	if req.Category != nil {
		update.SetCategory(*req.Category)
	}
	if req.ServiceName != nil {
		update.SetServiceName(*req.ServiceName)
	}
	if req.ResourceTypeName != nil {
		update.SetResourceTypeName(*req.ResourceTypeName)
	}
	if req.APIVersion != nil {
		update.SetAPIVersion(*req.APIVersion)
	}
	if req.AttributeSchema != nil {
		update.SetAttributeSchema(req.AttributeSchema)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorf("更新云服务失败: %v", err)
		return nil, err
	}

	return updated, nil
}

// DeleteCloudService 删除云服务
func (s *CloudService) DeleteCloudService(ctx context.Context, tenantID, id int) error {
	_, err := s.client.CloudService.Delete().
		Where(cloudservice.ID(id), cloudservice.TenantID(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorf("删除云服务失败: %v", err)
		return err
	}
	return nil
}

// ListCloudServices 获取云服务列表
func (s *CloudService) ListCloudServices(ctx context.Context, tenantID int, req *dto.ListCloudServicesRequest) ([]*ent.CloudService, int, error) {
	query := s.client.CloudService.Query().
		Where(cloudservice.TenantID(tenantID))

	// 过滤条件
	if req.Provider != "" {
		query = query.Where(cloudservice.Provider(req.Provider))
	}
	if req.Category != "" {
		query = query.Where(cloudservice.Category(req.Category))
	}
	if req.IsSystem != nil {
		query = query.Where(cloudservice.IsSystem(*req.IsSystem))
	}
	if req.IsActive != nil {
		query = query.Where(cloudservice.IsActive(*req.IsActive))
	}
	if req.ParentID != nil {
		query = query.Where(cloudservice.ParentID(*req.ParentID))
	}
	if req.Search != "" {
		query = query.Where(
			cloudservice.ServiceNameContains(req.Search),
		)
	}

	// 统计总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorf("统计云服务总数失败: %v", err)
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	services, err := query.
		Order(ent.Desc(cloudservice.FieldUpdatedAt)).
		Offset(offset).
		Limit(req.PageSize).
		All(ctx)
	if err != nil {
		s.logger.Errorf("查询云服务列表失败: %v", err)
		return nil, 0, err
	}

	return services, total, nil
}

// ===================================
// CloudResource CRUD Methods
// ===================================

// CreateCloudResource 创建云资源
func (s *CloudService) CreateCloudResource(ctx context.Context, tenantID int, req *dto.CreateCloudResourceRequest) (*ent.CloudResource, error) {
	resource, err := s.client.CloudResource.Create().
		SetCloudAccountID(req.CloudAccountID).
		SetServiceID(req.ServiceID).
		SetResourceID(req.ResourceID).
		SetResourceName(req.ResourceName).
		SetRegion(req.Region).
		SetZone(req.Zone).
		SetStatus(req.Status).
		SetTags(req.Tags).
		SetMetadata(req.Metadata).
		SetLifecycleState(req.LifecycleState).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorf("创建云资源失败: %v", err)
		return nil, err
	}

	return resource, nil
}

// GetCloudResource 获取云资源详情
func (s *CloudService) GetCloudResource(ctx context.Context, tenantID, id int) (*ent.CloudResource, error) {
	resource, err := s.client.CloudResource.Query().
		Where(cloudresource.ID(id), cloudresource.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		s.logger.Errorf("获取云资源失败: %v", err)
		return nil, err
	}
	return resource, nil
}

// UpdateCloudResource 更新云资源
func (s *CloudService) UpdateCloudResource(ctx context.Context, tenantID, id int, req *dto.UpdateCloudResourceRequest) (*ent.CloudResource, error) {
	resource, err := s.client.CloudResource.Query().
		Where(cloudresource.ID(id), cloudresource.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		s.logger.Errorf("查询云资源失败: %v", err)
		return nil, err
	}

	update := resource.Update()
	if req.ResourceName != nil {
		update.SetResourceName(*req.ResourceName)
	}
	if req.Region != nil {
		update.SetRegion(*req.Region)
	}
	if req.Zone != nil {
		update.SetZone(*req.Zone)
	}
	if req.Status != nil {
		update.SetStatus(*req.Status)
	}
	if req.Tags != nil {
		update.SetTags(req.Tags)
	}
	if req.Metadata != nil {
		update.SetMetadata(req.Metadata)
	}
	if req.LifecycleState != nil {
		update.SetLifecycleState(*req.LifecycleState)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorf("更新云资源失败: %v", err)
		return nil, err
	}

	return updated, nil
}

// DeleteCloudResource 删除云资源
func (s *CloudService) DeleteCloudResource(ctx context.Context, tenantID, id int) error {
	_, err := s.client.CloudResource.Delete().
		Where(cloudresource.ID(id), cloudresource.TenantID(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorf("删除云资源失败: %v", err)
		return err
	}
	return nil
}

// ListCloudResources 获取云资源列表
func (s *CloudService) ListCloudResources(ctx context.Context, tenantID int, req *dto.ListCloudResourcesRequest) ([]*ent.CloudResource, int, error) {
	query := s.client.CloudResource.Query().
		Where(cloudresource.TenantID(tenantID))

	// 过滤条件
	if req.CloudAccountID > 0 {
		query = query.Where(cloudresource.CloudAccountID(req.CloudAccountID))
	}
	if req.ServiceID > 0 {
		query = query.Where(cloudresource.ServiceID(req.ServiceID))
	}
	if req.Region != "" {
		query = query.Where(cloudresource.Region(req.Region))
	}
	if req.Status != "" {
		query = query.Where(cloudresource.Status(req.Status))
	}
	if req.Search != "" {
		query = query.Where(
			cloudresource.ResourceIDContains(req.Search),
		)
	}

	// 统计总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorf("统计云资源总数失败: %v", err)
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	resources, err := query.
		Order(ent.Desc(cloudresource.FieldUpdatedAt)).
		Offset(offset).
		Limit(req.PageSize).
		All(ctx)
	if err != nil {
		s.logger.Errorf("查询云资源列表失败: %v", err)
		return nil, 0, err
	}

	return resources, total, nil
}
