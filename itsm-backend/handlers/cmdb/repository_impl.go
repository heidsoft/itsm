package cmdb

import (
	"context"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/cloudaccount"
	"itsm-backend/ent/cloudresource"
	"itsm-backend/ent/cloudservice"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/discoveryresult"
	"itsm-backend/ent/discoverysource"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

// Map ent CI to domain CI
func toCIDomain(e *ent.ConfigurationItem) *ConfigurationItem {
	if e == nil {
		return nil
	}
	var cloudSyncTime *time.Time
	if !e.CloudSyncTime.IsZero() {
		cloudSyncTime = &e.CloudSyncTime
	}
	return &ConfigurationItem{
		ID:                 e.ID,
		CINumber:           ciNumberString(e.CiNumber),
		Name:               e.Name,
		Description:        e.Description,
		Type:               e.CiType,
		Status:             e.Status,
		Environment:        e.Environment,
		Criticality:        e.Criticality,
		Location:           e.Location,
		AssetTag:           e.AssetTag,
		SerialNumber:       e.SerialNumber,
		Model:              e.Model,
		Vendor:             e.Vendor,
		AssignedTo:         e.AssignedTo,
		OwnedBy:            e.OwnedBy,
		DiscoverySource:    e.DiscoverySource,
		Source:             e.Source,
		CloudProvider:      e.CloudProvider,
		CloudAccountID:     e.CloudAccountID,
		CloudRegion:        e.CloudRegion,
		CloudZone:          e.CloudZone,
		CloudResourceID:    e.CloudResourceID,
		CloudResourceType:  e.CloudResourceType,
		CloudMetadata:      e.CloudMetadata,
		CloudTags:          e.CloudTags,
		CloudMetrics:       e.CloudMetrics,
		CloudSyncTime:      cloudSyncTime,
		CloudSyncStatus:    e.CloudSyncStatus,
		CloudResourceRefID: e.CloudResourceRefID,
		CITypeID:           e.CiTypeID,
		TenantID:           e.TenantID,
		Attributes:         e.Attributes,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
	}
}

// CI CRUD / CIType / 关系相关实现属未注册路由的死代码，已删除；
// toCIDomain 保留给对账查询（ListCIsForReconciliation/GetCIByCloudResourceRefID）使用。

// Cloud services
func (r *EntRepository) CreateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error) {
	create := r.client.CloudService.Create().
		SetProvider(cs.Provider).
		SetServiceCode(cs.ServiceCode).
		SetServiceName(cs.ServiceName).
		SetResourceTypeCode(cs.ResourceTypeCode).
		SetResourceTypeName(cs.ResourceTypeName).
		SetIsSystem(cs.IsSystem).
		SetIsActive(cs.IsActive).
		SetTenantID(cs.TenantID)
	if cs.ParentID > 0 {
		create = create.SetParentID(cs.ParentID)
	}
	if cs.Category != "" {
		create = create.SetCategory(cs.Category)
	}
	if cs.APIVersion != "" {
		create = create.SetAPIVersion(cs.APIVersion)
	}
	if cs.AttributeSchema != nil {
		create = create.SetAttributeSchema(cs.AttributeSchema)
	}
	e, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return &CloudService{
		ID:               e.ID,
		ParentID:         e.ParentID,
		Provider:         e.Provider,
		Category:         e.Category,
		ServiceCode:      e.ServiceCode,
		ServiceName:      e.ServiceName,
		ResourceTypeCode: e.ResourceTypeCode,
		ResourceTypeName: e.ResourceTypeName,
		APIVersion:       e.APIVersion,
		AttributeSchema:  e.AttributeSchema,
		IsSystem:         e.IsSystem,
		IsActive:         e.IsActive,
		TenantID:         e.TenantID,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}, nil
}

func (r *EntRepository) ListCloudServices(ctx context.Context, tenantID int, provider string) ([]*CloudService, error) {
	q := r.client.CloudService.Query().Where(cloudservice.TenantID(tenantID))
	if provider != "" {
		q = q.Where(cloudservice.Provider(provider))
	}
	es, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*CloudService, 0, len(es))
	for _, e := range es {
		results = append(results, &CloudService{
			ID:               e.ID,
			ParentID:         e.ParentID,
			Provider:         e.Provider,
			Category:         e.Category,
			ServiceCode:      e.ServiceCode,
			ServiceName:      e.ServiceName,
			ResourceTypeCode: e.ResourceTypeCode,
			ResourceTypeName: e.ResourceTypeName,
			APIVersion:       e.APIVersion,
			AttributeSchema:  e.AttributeSchema,
			IsSystem:         e.IsSystem,
			IsActive:         e.IsActive,
			TenantID:         e.TenantID,
			CreatedAt:        e.CreatedAt,
			UpdatedAt:        e.UpdatedAt,
		})
	}
	return results, nil
}

func (r *EntRepository) GetCloudService(ctx context.Context, tenantID int, id int) (*CloudService, error) {
	e, err := r.client.CloudService.Query().
		Where(cloudservice.ID(id), cloudservice.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return &CloudService{
		ID:               e.ID,
		ParentID:         e.ParentID,
		Provider:         e.Provider,
		Category:         e.Category,
		ServiceCode:      e.ServiceCode,
		ServiceName:      e.ServiceName,
		ResourceTypeCode: e.ResourceTypeCode,
		ResourceTypeName: e.ResourceTypeName,
		APIVersion:       e.APIVersion,
		AttributeSchema:  e.AttributeSchema,
		IsSystem:         e.IsSystem,
		IsActive:         e.IsActive,
		TenantID:         e.TenantID,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}, nil
}

func (r *EntRepository) UpdateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error) {
	update := r.client.CloudService.UpdateOneID(cs.ID).
		Where(cloudservice.TenantID(cs.TenantID)).
		SetProvider(cs.Provider).
		SetCategory(cs.Category).
		SetServiceCode(cs.ServiceCode).
		SetServiceName(cs.ServiceName).
		SetResourceTypeCode(cs.ResourceTypeCode).
		SetResourceTypeName(cs.ResourceTypeName).
		SetAPIVersion(cs.APIVersion).
		SetAttributeSchema(cs.AttributeSchema).
		SetIsActive(cs.IsActive)
	if cs.ParentID > 0 {
		update = update.SetParentID(cs.ParentID)
	} else {
		update = update.ClearParentID()
	}
	e, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	return &CloudService{
		ID:               e.ID,
		ParentID:         e.ParentID,
		Provider:         e.Provider,
		Category:         e.Category,
		ServiceCode:      e.ServiceCode,
		ServiceName:      e.ServiceName,
		ResourceTypeCode: e.ResourceTypeCode,
		ResourceTypeName: e.ResourceTypeName,
		APIVersion:       e.APIVersion,
		AttributeSchema:  e.AttributeSchema,
		IsSystem:         e.IsSystem,
		IsActive:         e.IsActive,
		TenantID:         e.TenantID,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}, nil
}

func (r *EntRepository) DeleteCloudService(ctx context.Context, id int, tenantID int) (int, error) {
	n, err := r.client.CloudService.Delete().
		Where(cloudservice.ID(id), cloudservice.TenantID(tenantID)).
		Exec(ctx)
	return n, err
}

// Cloud accounts
func (r *EntRepository) CreateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error) {
	create := r.client.CloudAccount.Create().
		SetProvider(ca.Provider).
		SetAccountID(ca.AccountID).
		SetAccountName(ca.AccountName).
		SetCredentialRef(ca.CredentialRef).
		SetIsActive(ca.IsActive).
		SetTenantID(ca.TenantID)
	if ca.RegionWhitelist != nil {
		create = create.SetRegionWhitelist(ca.RegionWhitelist)
	}
	e, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return &CloudAccount{
		ID:              e.ID,
		Provider:        e.Provider,
		AccountID:       e.AccountID,
		AccountName:     e.AccountName,
		CredentialRef:   e.CredentialRef,
		RegionWhitelist: e.RegionWhitelist,
		IsActive:        e.IsActive,
		TenantID:        e.TenantID,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}, nil
}

func (r *EntRepository) ListCloudAccounts(ctx context.Context, tenantID int, provider string) ([]*CloudAccount, error) {
	q := r.client.CloudAccount.Query().Where(cloudaccount.TenantID(tenantID))
	if provider != "" {
		q = q.Where(cloudaccount.Provider(provider))
	}
	es, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*CloudAccount, 0, len(es))
	for _, e := range es {
		results = append(results, &CloudAccount{
			ID:              e.ID,
			Provider:        e.Provider,
			AccountID:       e.AccountID,
			AccountName:     e.AccountName,
			CredentialRef:   e.CredentialRef,
			RegionWhitelist: e.RegionWhitelist,
			IsActive:        e.IsActive,
			TenantID:        e.TenantID,
			CreatedAt:       e.CreatedAt,
			UpdatedAt:       e.UpdatedAt,
		})
	}
	return results, nil
}

func (r *EntRepository) GetCloudAccount(ctx context.Context, tenantID int, id int) (*CloudAccount, error) {
	e, err := r.client.CloudAccount.Query().
		Where(cloudaccount.ID(id), cloudaccount.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return &CloudAccount{
		ID:              e.ID,
		Provider:        e.Provider,
		AccountID:       e.AccountID,
		AccountName:     e.AccountName,
		CredentialRef:   e.CredentialRef,
		RegionWhitelist: e.RegionWhitelist,
		IsActive:        e.IsActive,
		TenantID:        e.TenantID,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}, nil
}

func (r *EntRepository) UpdateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error) {
	e, err := r.client.CloudAccount.UpdateOneID(ca.ID).
		Where(cloudaccount.TenantID(ca.TenantID)).
		SetProvider(ca.Provider).
		SetAccountID(ca.AccountID).
		SetAccountName(ca.AccountName).
		SetCredentialRef(ca.CredentialRef).
		SetRegionWhitelist(ca.RegionWhitelist).
		SetIsActive(ca.IsActive).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return &CloudAccount{
		ID:              e.ID,
		Provider:        e.Provider,
		AccountID:       e.AccountID,
		AccountName:     e.AccountName,
		CredentialRef:   e.CredentialRef,
		RegionWhitelist: e.RegionWhitelist,
		IsActive:        e.IsActive,
		TenantID:        e.TenantID,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}, nil
}

func (r *EntRepository) DeleteCloudAccount(ctx context.Context, id int, tenantID int) (int, error) {
	n, err := r.client.CloudAccount.Delete().
		Where(cloudaccount.ID(id), cloudaccount.TenantID(tenantID)).
		Exec(ctx)
	return n, err
}

// Cloud resources

// applyCloudResourceFilters 把云资源过滤条件翻成谓词。
// 整表读取（对账）与分页读取（列表端点）共用这一份，避免同一个查询参数在两条路径上给出不同结果。
func applyCloudResourceFilters(q *ent.CloudResourceQuery, tenantID int, f CloudResourceFilter) *ent.CloudResourceQuery {
	q = q.Where(cloudresource.TenantID(tenantID))
	if f.Provider != "" {
		q = q.Where(cloudresource.HasAccountWith(cloudaccount.Provider(f.Provider)))
	}
	if f.CloudAccountID > 0 {
		q = q.Where(cloudresource.CloudAccountID(f.CloudAccountID))
	}
	if f.ServiceID > 0 {
		q = q.Where(cloudresource.ServiceID(f.ServiceID))
	}
	if f.Region != "" {
		q = q.Where(cloudresource.Region(f.Region))
	}
	if f.Status != "" {
		q = q.Where(cloudresource.Status(f.Status))
	}
	if f.Search != "" {
		q = q.Where(cloudresource.ResourceIDContains(f.Search))
	}
	return q
}

func toCloudResourceDomain(e *ent.CloudResource) *CloudResource {
	var firstSeenAt *time.Time
	if !e.FirstSeenAt.IsZero() {
		firstSeenAt = &e.FirstSeenAt
	}
	var lastSeenAt *time.Time
	if !e.LastSeenAt.IsZero() {
		lastSeenAt = &e.LastSeenAt
	}
	return &CloudResource{
		ID:              e.ID,
		CloudAccountID:  e.CloudAccountID,
		ServiceID:       e.ServiceID,
		ResourceID:      e.ResourceID,
		IdentityVersion: e.IdentityVersion, Provider: e.Provider, Partition: e.Partition,
		CanonicalAccountID: e.CanonicalAccountID, ResourceScope: e.ResourceScope,
		ServiceCode: e.ServiceCode, ResourceType: e.ResourceType, IdentityHash: e.IdentityHash,
		SourceID: e.SourceID, SourceFingerprint: e.SourceFingerprint, MissingCount: e.MissingCount,
		ResourceName:   e.ResourceName,
		Region:         e.Region,
		Zone:           e.Zone,
		Status:         e.Status,
		Tags:           e.Tags,
		Metadata:       e.Metadata,
		FirstSeenAt:    firstSeenAt,
		LastSeenAt:     lastSeenAt,
		LifecycleState: e.LifecycleState,
		TenantID:       e.TenantID,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func (r *EntRepository) ListCloudResources(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error) {
	es, err := applyCloudResourceFilters(r.client.CloudResource.Query(), tenantID, filter).All(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*CloudResource, 0, len(es))
	for _, e := range es {
		results = append(results, toCloudResourceDomain(e))
	}
	return results, nil
}

// ListCloudResourcesPage 分页读取云资源并返回 (当前页, 总数)。
//
// 排序固定 updated_at DESC + id ASC：只按 updated_at 排时，同一次发现任务批量写入的资源
// 更新时间相同，数据库对并列行的顺序不保证，翻页会重复或漏行。
func (r *EntRepository) ListCloudResourcesPage(
	ctx context.Context,
	tenantID int,
	filter CloudResourceFilter,
	page int,
	pageSize int,
) ([]*CloudResource, int, error) {
	q := applyCloudResourceFilters(r.client.CloudResource.Query(), tenantID, filter)
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	es, err := q.
		Order(ent.Desc(cloudresource.FieldUpdatedAt), ent.Asc(cloudresource.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	results := make([]*CloudResource, 0, len(es))
	for _, e := range es {
		results = append(results, toCloudResourceDomain(e))
	}
	return results, total, nil
}

func (r *EntRepository) GetCloudResource(ctx context.Context, tenantID int, id int) (*CloudResource, error) {
	e, err := r.client.CloudResource.Query().
		Where(cloudresource.ID(id), cloudresource.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	var firstSeenAt *time.Time
	if !e.FirstSeenAt.IsZero() {
		firstSeenAt = &e.FirstSeenAt
	}
	var lastSeenAt *time.Time
	if !e.LastSeenAt.IsZero() {
		lastSeenAt = &e.LastSeenAt
	}
	return &CloudResource{
		ID:              e.ID,
		CloudAccountID:  e.CloudAccountID,
		ServiceID:       e.ServiceID,
		ResourceID:      e.ResourceID,
		IdentityVersion: e.IdentityVersion, Provider: e.Provider, Partition: e.Partition,
		CanonicalAccountID: e.CanonicalAccountID, ResourceScope: e.ResourceScope,
		ServiceCode: e.ServiceCode, ResourceType: e.ResourceType, IdentityHash: e.IdentityHash,
		SourceID: e.SourceID, SourceFingerprint: e.SourceFingerprint, MissingCount: e.MissingCount,
		ResourceName:   e.ResourceName,
		Region:         e.Region,
		Zone:           e.Zone,
		Status:         e.Status,
		Tags:           e.Tags,
		Metadata:       e.Metadata,
		FirstSeenAt:    firstSeenAt,
		LastSeenAt:     lastSeenAt,
		LifecycleState: e.LifecycleState,
		TenantID:       e.TenantID,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}, nil
}

func (r *EntRepository) CreateCloudResource(ctx context.Context, cr *CloudResource) (*CloudResource, error) {
	create := r.client.CloudResource.Create().
		SetCloudAccountID(cr.CloudAccountID).
		SetServiceID(cr.ServiceID).
		SetResourceID(cr.ResourceID).
		SetIdentityVersion(cr.IdentityVersion).
		SetProvider(cr.Provider).
		SetPartition(cr.Partition).
		SetCanonicalAccountID(cr.CanonicalAccountID).
		SetResourceScope(cr.ResourceScope).
		SetServiceCode(cr.ServiceCode).
		SetResourceType(cr.ResourceType).
		SetIdentityHash(cr.IdentityHash).
		SetSourceID(cr.SourceID).
		SetSourceFingerprint(cr.SourceFingerprint).
		SetMissingCount(cr.MissingCount).
		SetResourceName(cr.ResourceName).
		SetRegion(cr.Region).
		SetZone(cr.Zone).
		SetStatus(cr.Status).
		SetTags(cr.Tags).
		SetMetadata(cr.Metadata).
		SetLifecycleState(cr.LifecycleState).
		SetTenantID(cr.TenantID)
	if cr.FirstSeenAt != nil {
		create = create.SetFirstSeenAt(*cr.FirstSeenAt)
	}
	e, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	var firstSeenAt *time.Time
	if !e.FirstSeenAt.IsZero() {
		firstSeenAt = &e.FirstSeenAt
	}
	var lastSeenAt *time.Time
	if !e.LastSeenAt.IsZero() {
		lastSeenAt = &e.LastSeenAt
	}
	return &CloudResource{
		ID:              e.ID,
		CloudAccountID:  e.CloudAccountID,
		ServiceID:       e.ServiceID,
		ResourceID:      e.ResourceID,
		IdentityVersion: e.IdentityVersion, Provider: e.Provider, Partition: e.Partition,
		CanonicalAccountID: e.CanonicalAccountID, ResourceScope: e.ResourceScope,
		ServiceCode: e.ServiceCode, ResourceType: e.ResourceType, IdentityHash: e.IdentityHash,
		SourceID: e.SourceID, SourceFingerprint: e.SourceFingerprint, MissingCount: e.MissingCount,
		ResourceName:   e.ResourceName,
		Region:         e.Region,
		Zone:           e.Zone,
		Status:         e.Status,
		Tags:           e.Tags,
		Metadata:       e.Metadata,
		FirstSeenAt:    firstSeenAt,
		LastSeenAt:     lastSeenAt,
		LifecycleState: e.LifecycleState,
		TenantID:       e.TenantID,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}, nil
}

func (r *EntRepository) UpdateCloudResource(ctx context.Context, cr *CloudResource) (*CloudResource, error) {
	update := r.client.CloudResource.UpdateOneID(cr.ID).
		Where(cloudresource.TenantID(cr.TenantID)).
		SetCloudAccountID(cr.CloudAccountID).
		SetServiceID(cr.ServiceID).
		SetResourceID(cr.ResourceID).
		SetIdentityVersion(cr.IdentityVersion).
		SetProvider(cr.Provider).
		SetPartition(cr.Partition).
		SetCanonicalAccountID(cr.CanonicalAccountID).
		SetResourceScope(cr.ResourceScope).
		SetServiceCode(cr.ServiceCode).
		SetResourceType(cr.ResourceType).
		SetIdentityHash(cr.IdentityHash).
		SetSourceID(cr.SourceID).
		SetSourceFingerprint(cr.SourceFingerprint).
		SetMissingCount(cr.MissingCount).
		SetResourceName(cr.ResourceName).
		SetRegion(cr.Region).
		SetZone(cr.Zone).
		SetStatus(cr.Status).
		SetTags(cr.Tags).
		SetMetadata(cr.Metadata).
		SetLifecycleState(cr.LifecycleState)
	e, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	var firstSeenAt *time.Time
	if !e.FirstSeenAt.IsZero() {
		firstSeenAt = &e.FirstSeenAt
	}
	var lastSeenAt *time.Time
	if !e.LastSeenAt.IsZero() {
		lastSeenAt = &e.LastSeenAt
	}
	return &CloudResource{
		ID:              e.ID,
		CloudAccountID:  e.CloudAccountID,
		ServiceID:       e.ServiceID,
		ResourceID:      e.ResourceID,
		IdentityVersion: e.IdentityVersion, Provider: e.Provider, Partition: e.Partition,
		CanonicalAccountID: e.CanonicalAccountID, ResourceScope: e.ResourceScope,
		ServiceCode: e.ServiceCode, ResourceType: e.ResourceType, IdentityHash: e.IdentityHash,
		SourceID: e.SourceID, SourceFingerprint: e.SourceFingerprint, MissingCount: e.MissingCount,
		ResourceName:   e.ResourceName,
		Region:         e.Region,
		Zone:           e.Zone,
		Status:         e.Status,
		Tags:           e.Tags,
		Metadata:       e.Metadata,
		FirstSeenAt:    firstSeenAt,
		LastSeenAt:     lastSeenAt,
		LifecycleState: e.LifecycleState,
		TenantID:       e.TenantID,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}, nil
}

func (r *EntRepository) DeleteCloudResource(ctx context.Context, id int, tenantID int) (int, error) {
	n, err := r.client.CloudResource.Delete().
		Where(cloudresource.ID(id), cloudresource.TenantID(tenantID)).
		Exec(ctx)
	return n, err
}

func (r *EntRepository) ListCIsForReconciliation(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
	q := r.client.ConfigurationItem.Query().Where(
		configurationitem.TenantID(tenantID),
		configurationitem.Or(
			configurationitem.CloudResourceRefIDNotNil(),
			configurationitem.CloudResourceIDNEQ(""),
		),
	)
	es, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*ConfigurationItem, 0, len(es))
	for _, e := range es {
		results = append(results, toCIDomain(e))
	}
	return results, nil
}

func (r *EntRepository) GetCIByCloudResourceRefID(ctx context.Context, tenantID int, cloudResourceRefID int) (*ConfigurationItem, error) {
	e, err := r.client.ConfigurationItem.Query().
		Where(
			configurationitem.TenantID(tenantID),
			configurationitem.CloudResourceRefIDEQ(cloudResourceRefID),
		).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return toCIDomain(e), nil
}

// Discovery
func (r *EntRepository) CreateDiscoverySource(ctx context.Context, ds *DiscoverySource) (*DiscoverySource, error) {
	create := r.client.DiscoverySource.Create().
		SetID(ds.ID).
		SetName(ds.Name).
		SetSourceType(ds.SourceType).
		SetProvider(ds.Provider).
		SetServiceCodes(ds.ServiceCodes).
		SetRegions(ds.Regions).
		SetSchedule(ds.Schedule).
		SetReconcilePolicy(ds.ReconcilePolicy).
		SetEnabled(ds.IsActive).
		SetDescription(ds.Description).
		SetTenantID(ds.TenantID)
	if ds.CloudAccountID > 0 {
		create = create.SetCloudAccountID(ds.CloudAccountID)
	}
	if ds.StaleThreshold > 0 {
		create = create.SetStaleThreshold(ds.StaleThreshold)
	}
	if ds.LastSuccessAt != nil {
		create = create.SetLastSuccessAt(*ds.LastSuccessAt)
	}
	e, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return &DiscoverySource{
		ID:             e.ID,
		Name:           e.Name,
		SourceType:     e.SourceType,
		Provider:       e.Provider,
		CloudAccountID: e.CloudAccountID, ServiceCodes: e.ServiceCodes, Regions: e.Regions,
		Schedule: e.Schedule, ReconcilePolicy: e.ReconcilePolicy, StaleThreshold: e.StaleThreshold,
		LastSuccessAt: optionalTime(e.LastSuccessAt),
		IsActive:      e.Enabled,
		Description:   e.Description,
		TenantID:      e.TenantID,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}, nil
}

func (r *EntRepository) ListDiscoverySources(ctx context.Context, tenantID int) ([]*DiscoverySource, error) {
	es, err := r.client.DiscoverySource.Query().
		Where(discoverysource.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*DiscoverySource, 0, len(es))
	for _, e := range es {
		results = append(results, &DiscoverySource{
			ID:             e.ID,
			Name:           e.Name,
			SourceType:     e.SourceType,
			Provider:       e.Provider,
			CloudAccountID: e.CloudAccountID, ServiceCodes: e.ServiceCodes, Regions: e.Regions,
			Schedule: e.Schedule, ReconcilePolicy: e.ReconcilePolicy, StaleThreshold: e.StaleThreshold,
			LastSuccessAt: optionalTime(e.LastSuccessAt),
			IsActive:      e.Enabled,
			Description:   e.Description,
			TenantID:      e.TenantID,
			CreatedAt:     e.CreatedAt,
			UpdatedAt:     e.UpdatedAt,
		})
	}
	return results, nil
}

func (r *EntRepository) CreateDiscoveryJob(ctx context.Context, job *DiscoveryJob) (*DiscoveryJob, error) {
	create := r.client.DiscoveryJob.Create().
		SetSourceID(job.SourceID).
		SetIdempotencyKey(job.IdempotencyKey).
		SetRequestFingerprint(job.RequestFingerprint).
		SetSourceSnapshot(job.SourceSnapshot).
		SetScopeSnapshot(job.ScopeSnapshot).
		SetCompletedScopes(job.CompletedScopes).
		SetFailedScopes(job.FailedScopes).
		SetSnapshotGeneration(job.SnapshotGeneration).
		SetLeaseOwner(job.LeaseOwner).
		SetFencingToken(job.FencingToken).
		SetAttempt(job.Attempt).
		SetProgress(job.Progress).
		SetErrorCode(job.ErrorCode).
		SetErrorMessage(job.ErrorMessage).
		SetTenantID(job.TenantID)
	if job.Status != "" {
		create = create.SetStatus(job.Status)
	}
	if job.Operation != "" {
		create = create.SetOperation(job.Operation)
	}
	if job.MaxAttempts > 0 {
		create = create.SetMaxAttempts(job.MaxAttempts)
	}
	if job.RequestedBy > 0 {
		create = create.SetRequestedBy(job.RequestedBy)
	}
	if job.ParentJobID > 0 {
		create = create.SetParentJobID(job.ParentJobID)
	}
	if job.QueuedAt != nil {
		create = create.SetQueuedAt(*job.QueuedAt)
	}
	if job.HeartbeatAt != nil {
		create = create.SetHeartbeatAt(*job.HeartbeatAt)
	}
	if job.LeaseExpiresAt != nil {
		create = create.SetLeaseExpiresAt(*job.LeaseExpiresAt)
	}
	if job.CancelRequestedAt != nil {
		create = create.SetCancelRequestedAt(*job.CancelRequestedAt)
	}
	if job.StartedAt != nil {
		create = create.SetStartedAt(*job.StartedAt)
	}
	if job.FinishedAt != nil {
		create = create.SetFinishedAt(*job.FinishedAt)
	}
	if job.Summary != nil {
		create = create.SetSummary(job.Summary)
	}
	e, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	var startedAt *time.Time
	if !e.StartedAt.IsZero() {
		startedAt = &e.StartedAt
	}
	var finishedAt *time.Time
	if !e.FinishedAt.IsZero() {
		finishedAt = &e.FinishedAt
	}
	return &DiscoveryJob{
		ID:        e.ID,
		SourceID:  e.SourceID,
		Status:    e.Status,
		Operation: e.Operation, IdempotencyKey: e.IdempotencyKey, RequestFingerprint: e.RequestFingerprint,
		SourceSnapshot: e.SourceSnapshot, ScopeSnapshot: e.ScopeSnapshot,
		CompletedScopes: e.CompletedScopes, FailedScopes: e.FailedScopes,
		SnapshotGeneration: e.SnapshotGeneration, RequestedBy: e.RequestedBy,
		QueuedAt: optionalTime(e.QueuedAt), HeartbeatAt: optionalTime(e.HeartbeatAt),
		LeaseOwner: e.LeaseOwner, LeaseExpiresAt: optionalTime(e.LeaseExpiresAt),
		FencingToken: e.FencingToken, Attempt: e.Attempt, ParentJobID: e.ParentJobID,
		MaxAttempts: e.MaxAttempts, Progress: e.Progress, ErrorCode: e.ErrorCode,
		ErrorMessage: e.ErrorMessage, CancelRequestedAt: optionalTime(e.CancelRequestedAt),
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Summary:    e.Summary,
		TenantID:   e.TenantID,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}, nil
}

func (r *EntRepository) ListDiscoveryResults(ctx context.Context, tenantID int, jobID int) ([]*DiscoveryResult, error) {
	q := r.client.DiscoveryResult.Query().Where(discoveryresult.TenantID(tenantID))
	if jobID > 0 {
		q = q.Where(discoveryresult.JobID(jobID))
	}
	es, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*DiscoveryResult, 0, len(es))
	for _, e := range es {
		results = append(results, &DiscoveryResult{
			ID:               e.ID,
			JobID:            e.JobID,
			CIID:             e.CiID,
			Action:           e.Action,
			ResourceType:     e.ResourceType,
			ResourceID:       e.ResourceID,
			ResourceIdentity: e.ResourceIdentity, IdentityVersion: e.IdentityVersion,
			ResourceSnapshot: e.ResourceSnapshot, BeforeHash: e.BeforeHash, AfterHash: e.AfterHash,
			Diff:      e.Diff,
			Status:    e.Status,
			ErrorCode: e.ErrorCode, ErrorMessage: e.ErrorMessage,
			TenantID:  e.TenantID,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
		})
	}
	return results, nil
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

// ciNumberString 安全解引用 ci_number（Optional+Nillable 字段）
func ciNumberString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
