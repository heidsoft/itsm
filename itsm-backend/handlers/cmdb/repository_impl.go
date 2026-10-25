package cmdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/cloudaccount"
	"itsm-backend/ent/cloudresource"
	"itsm-backend/ent/cloudservice"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/discoveryjob"
	"itsm-backend/ent/discoveryresult"
	"itsm-backend/ent/discoverysource"
	"itsm-backend/internal/commandbus"
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
		LifecycleStatus:    e.LifecycleStatus,
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
	if cr.LastSeenAt != nil {
		create = create.SetLastSeenAt(*cr.LastSeenAt)
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

func (r *EntRepository) FindCIByCloudResourceIdentity(ctx context.Context, tenantID int, provider, cloudResourceID string) (*ConfigurationItem, error) {
	if strings.TrimSpace(cloudResourceID) == "" {
		return nil, &ent.NotFoundError{}
	}
	e, err := r.client.ConfigurationItem.Query().
		Where(
			configurationitem.TenantID(tenantID),
			configurationitem.CloudProviderEQ(provider),
			configurationitem.CloudResourceIDEQ(cloudResourceID),
		).
		Order(ent.Asc(configurationitem.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(e) == 1 {
		return toCIDomain(e[0]), nil
	}
	// 多条同名命中说明资源身份不够区分（例如不同集群里的同名 Pod）。链接错的
	// CI 会静默改掉一台机器的台账，重复建一条至少还能被对账看见并人工合并。
	return nil, &ent.NotFoundError{}
}

func (r *EntRepository) GetCITypeIDByName(ctx context.Context, tenantID int, name string) (int, error) {
	e, err := r.client.CIType.Query().
		Where(citype.TenantIDEQ(tenantID), citype.NameEQ(name), citype.IsActive(true)).
		First(ctx)
	if err != nil {
		return 0, err
	}
	return e.ID, nil
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
		results = append(results, discoveryResultFromEnt(e))
	}
	return results, nil
}

func discoveryResultFromEnt(e *ent.DiscoveryResult) *DiscoveryResult {
	if e == nil {
		return nil
	}
	return &DiscoveryResult{
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
	}
}

// GetDiscoveryResult 租户内按 ID 取单条差异（人工决策的读取面）。
func (r *EntRepository) GetDiscoveryResult(ctx context.Context, tenantID, id int) (*DiscoveryResult, error) {
	e, err := r.client.DiscoveryResult.Query().
		Where(discoveryresult.TenantID(tenantID), discoveryresult.ID(id)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return discoveryResultFromEnt(e), nil
}

// UpdateDiscoveryResult 写回差异的处理状态与错误摘要（人工应用/驳回的持久化通道）。
func (r *EntRepository) UpdateDiscoveryResult(ctx context.Context, res *DiscoveryResult) error {
	if res == nil || res.ID <= 0 {
		return fmt.Errorf("discovery result id is required")
	}
	update := r.client.DiscoveryResult.UpdateOneID(res.ID).
		Where(discoveryresult.TenantID(res.TenantID)).
		SetStatus(res.Status)
	if res.CIID > 0 {
		update = update.SetCiID(res.CIID)
	}
	if res.ErrorCode != "" {
		update = update.SetErrorCode(res.ErrorCode)
	} else {
		update = update.ClearErrorCode()
	}
	if res.ErrorMessage != "" {
		update = update.SetErrorMessage(res.ErrorMessage)
	} else {
		update = update.ClearErrorMessage()
	}
	return update.Exec(ctx)
}

// TransitionDiscoveryResult 以状态 CAS 更新差异决策状态。
func (r *EntRepository) TransitionDiscoveryResult(ctx context.Context, tenantID, id int, fromStatus, toStatus, errorCode, errorMessage string) (int, error) {
	if tenantID <= 0 || id <= 0 || fromStatus == "" || toStatus == "" {
		return 0, fmt.Errorf("discovery result transition requires tenant, id, and statuses")
	}
	update := r.client.DiscoveryResult.Update().
		Where(
			discoveryresult.TenantID(tenantID),
			discoveryresult.ID(id),
			discoveryresult.StatusEQ(fromStatus),
		).
		SetStatus(toStatus)
	if errorCode != "" {
		update = update.SetErrorCode(errorCode)
	} else {
		update = update.ClearErrorCode()
	}
	if errorMessage != "" {
		update = update.SetErrorMessage(errorMessage)
	} else {
		update = update.ClearErrorMessage()
	}
	return update.Save(ctx)
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

// discoveryJobFromEnt 是 ent.DiscoveryJob → 域结构体的唯一映射点。
func discoveryJobFromEnt(e *ent.DiscoveryJob) *DiscoveryJob {
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
		StartedAt:  optionalTime(e.StartedAt),
		FinishedAt: optionalTime(e.FinishedAt),
		Summary:    e.Summary,
		TenantID:   e.TenantID,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
}

// discoveryJobApplyCreate 把域结构体的可选字段铺到 ent create builder。
func discoveryJobApplyCreate(create *ent.DiscoveryJobCreate, job *DiscoveryJob) *ent.DiscoveryJobCreate {
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
	return create
}

// GetDiscoverySource 租户内按 ID 取发现源；跨租户/不存在统一 NotFound。
func (r *EntRepository) GetDiscoverySource(ctx context.Context, tenantID int, id string) (*DiscoverySource, error) {
	e, err := r.client.DiscoverySource.Query().
		Where(discoverysource.TenantID(tenantID), discoverysource.ID(id)).
		Only(ctx)
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

// TouchDiscoverySourceSuccess 更新发现源最近成功时间（执行器终态写回）。
func (r *EntRepository) TouchDiscoverySourceSuccess(ctx context.Context, tenantID int, id string, at time.Time) error {
	n, err := r.client.DiscoverySource.Update().
		Where(discoverysource.TenantID(tenantID), discoverysource.ID(id)).
		SetLastSuccessAt(at).
		SetUpdatedAt(at).
		Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return &ent.NotFoundError{}
	}
	return nil
}

// CreateDiscoveryJobWithCommand 同一事务内：discovery_jobs 落库 + 出箱命令入队。
// 「有 job 必有命令」是可靠执行的前提——两步拆开时，job 落库而入队失败会产生
// 永远停在 queued 的僵尸任务（这正是旧 fail-closed 挡住的东西）。
func (r *EntRepository) CreateDiscoveryJobWithCommand(ctx context.Context, job *DiscoveryJob, commandType string, payload map[string]interface{}) (*DiscoveryJob, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin discovery job tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	create := tx.DiscoveryJob.Create().
		SetSourceID(job.SourceID).
		SetIdempotencyKey(job.IdempotencyKey).
		SetRequestFingerprint(job.RequestFingerprint).
		SetSourceSnapshot(job.SourceSnapshot).
		SetScopeSnapshot(job.ScopeSnapshot).
		SetCompletedScopes(job.CompletedScopes).
		SetFailedScopes(job.FailedScopes).
		SetSnapshotGeneration(job.SnapshotGeneration).
		SetFencingToken(job.FencingToken).
		SetAttempt(job.Attempt).
		SetProgress(job.Progress).
		SetTenantID(job.TenantID)
	create = discoveryJobApplyCreate(create, job)
	e, err := create.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create discovery job: %w", err)
	}

	if payload == nil {
		payload = map[string]interface{}{}
	}
	payload["discoveryJobId"] = e.ID
	payload["tenantId"] = e.TenantID
	if _, err := commandbus.EnqueueTx(ctx, tx, commandbus.EnqueueRequest{
		TenantID:       e.TenantID,
		CommandType:    commandType,
		AggregateType:  "discovery_job",
		AggregateID:    e.ID,
		IdempotencyKey: fmt.Sprintf("discovery-job-%d", e.ID),
		Payload:        payload,
	}); err != nil {
		return nil, fmt.Errorf("enqueue discovery command: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit discovery job tx: %w", err)
	}
	committed = true
	return discoveryJobFromEnt(e), nil
}

// GetDiscoveryJob 租户内按 ID 取任务。
func (r *EntRepository) GetDiscoveryJob(ctx context.Context, tenantID, id int) (*DiscoveryJob, error) {
	e, err := r.client.DiscoveryJob.Query().
		Where(discoveryjob.TenantID(tenantID), discoveryjob.ID(id)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return discoveryJobFromEnt(e), nil
}

// FindDiscoveryJobByIdempotencyKey 幂等查询：(tenant, source, key) 唯一索引的读面。
func (r *EntRepository) FindDiscoveryJobByIdempotencyKey(ctx context.Context, tenantID int, sourceID, key string) (*DiscoveryJob, error) {
	e, err := r.client.DiscoveryJob.Query().
		Where(discoveryjob.TenantID(tenantID), discoveryjob.SourceID(sourceID), discoveryjob.IdempotencyKey(key)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return discoveryJobFromEnt(e), nil
}

// ListScheduledDiscoverySources 返回启用且配置了调度表达式的来源。
//
// 空串与纯空白都视为「未配置」：cron 解析器会把空白表达式判为非法，若放进
// 扫描集，调度器每分钟都会为一个没配调度的来源告警一次。
func (r *EntRepository) ListScheduledDiscoverySources(ctx context.Context, tenantID int) ([]*DiscoverySource, error) {
	es, err := r.client.DiscoverySource.Query().
		Where(
			discoverysource.TenantID(tenantID),
			discoverysource.Enabled(true),
			discoverysource.ScheduleNotNil(),
			discoverysource.ScheduleNEQ(""),
		).
		Order(ent.Asc(discoverysource.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*DiscoverySource, 0, len(es))
	for _, e := range es {
		if strings.TrimSpace(e.Schedule) == "" {
			continue
		}
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

// LatestDiscoveryJobForSource 取来源最近一次任务（入队时间倒序，同刻按 ID 倒序）。
// 没有任何任务时返回 ent.NotFoundError，由调用方按「从未跑过」处理。
func (r *EntRepository) LatestDiscoveryJobForSource(ctx context.Context, tenantID int, sourceID string) (*DiscoveryJob, error) {
	e, err := r.client.DiscoveryJob.Query().
		Where(discoveryjob.TenantID(tenantID), discoveryjob.SourceID(sourceID)).
		Order(ent.Desc(discoveryjob.FieldQueuedAt), ent.Desc(discoveryjob.FieldID)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return discoveryJobFromEnt(e), nil
}

// LatestScheduledDiscoveryJobForSource 取最近一次由 cron 调度器创建的任务。
func (r *EntRepository) LatestScheduledDiscoveryJobForSource(ctx context.Context, tenantID int, sourceID string) (*DiscoveryJob, error) {
	prefix := "sched-" + sourceID + "-"
	e, err := r.client.DiscoveryJob.Query().
		Where(
			discoveryjob.TenantID(tenantID),
			discoveryjob.SourceID(sourceID),
			discoveryjob.IdempotencyKeyHasPrefix(prefix),
		).
		Order(ent.Desc(discoveryjob.FieldQueuedAt), ent.Desc(discoveryjob.FieldID)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return discoveryJobFromEnt(e), nil
}

// ListDiscoveryJobs 分页列出任务；status 为空不过滤。
func (r *EntRepository) ListDiscoveryJobs(ctx context.Context, tenantID int, status string, page, pageSize int) ([]*DiscoveryJob, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	q := r.client.DiscoveryJob.Query().Where(discoveryjob.TenantID(tenantID))
	if status != "" {
		q = q.Where(discoveryjob.Status(status))
	}
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	es, err := q.Order(ent.Desc(discoveryjob.FieldCreatedAt), ent.Desc(discoveryjob.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	results := make([]*DiscoveryJob, 0, len(es))
	for _, e := range es {
		results = append(results, discoveryJobFromEnt(e))
	}
	return results, total, nil
}

// UpdateDiscoveryJob 全字段写回（执行器通道）。只按 tenant+id 命中行更新。
func (r *EntRepository) UpdateDiscoveryJob(ctx context.Context, job *DiscoveryJob) error {
	if job == nil || job.ID <= 0 {
		return fmt.Errorf("update discovery job: id required")
	}
	update := r.client.DiscoveryJob.Update().
		Where(discoveryjob.TenantID(job.TenantID), discoveryjob.ID(job.ID)).
		SetStatus(job.Status).
		SetCompletedScopes(job.CompletedScopes).
		SetFailedScopes(job.FailedScopes).
		SetLeaseOwner(job.LeaseOwner).
		SetFencingToken(job.FencingToken).
		SetAttempt(job.Attempt).
		SetProgress(job.Progress).
		SetErrorCode(job.ErrorCode).
		SetErrorMessage(job.ErrorMessage).
		SetHeartbeatAt(time.Now())
	if job.LeaseExpiresAt != nil {
		update = update.SetLeaseExpiresAt(*job.LeaseExpiresAt)
	} else {
		update = update.ClearLeaseExpiresAt()
	}
	if job.CancelRequestedAt != nil {
		update = update.SetCancelRequestedAt(*job.CancelRequestedAt)
	}
	if job.StartedAt != nil {
		update = update.SetStartedAt(*job.StartedAt)
	}
	if job.FinishedAt != nil {
		update = update.SetFinishedAt(*job.FinishedAt)
	}
	if job.Summary != nil {
		update = update.SetSummary(job.Summary)
	}
	n, err := update.Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return &ent.NotFoundError{}
	}
	return nil
}

// CreateDiscoveryResult 落一条发现 diff 结果。
func (r *EntRepository) CreateDiscoveryResult(ctx context.Context, res *DiscoveryResult) (*DiscoveryResult, error) {
	create := r.client.DiscoveryResult.Create().
		SetJobID(res.JobID).
		SetAction(res.Action).
		SetResourceType(res.ResourceType).
		SetResourceID(res.ResourceID).
		SetResourceIdentity(res.ResourceIdentity).
		SetIdentityVersion(res.IdentityVersion).
		SetResourceSnapshot(res.ResourceSnapshot).
		SetBeforeHash(res.BeforeHash).
		SetAfterHash(res.AfterHash).
		SetDiff(res.Diff).
		SetStatus(res.Status).
		SetErrorCode(res.ErrorCode).
		SetErrorMessage(res.ErrorMessage).
		SetTenantID(res.TenantID)
	if res.CIID > 0 {
		create = create.SetCiID(res.CIID)
	}
	if res.Status == "" {
		create = create.SetStatus("pending")
	}
	e, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	res.ID = e.ID
	res.CreatedAt = e.CreatedAt
	res.UpdatedAt = e.UpdatedAt
	return res, nil
}

// GetCloudResourceByIdentity 按规范身份哈希取资源（diff 现状读取面）。
func (r *EntRepository) GetCloudResourceByIdentity(ctx context.Context, tenantID int, identityHash string) (*CloudResource, error) {
	e, err := r.client.CloudResource.Query().
		Where(cloudresource.TenantID(tenantID), cloudresource.IdentityHash(identityHash)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return cloudResourceFromEnt(e), nil
}

// ListCloudResourcesByAccountService 按账号+服务维度列资源（缺失扫描）。
func (r *EntRepository) ListCloudResourcesByAccountService(ctx context.Context, tenantID, cloudAccountID int, serviceCode string) ([]*CloudResource, error) {
	es, err := r.client.CloudResource.Query().
		Where(
			cloudresource.TenantID(tenantID),
			cloudresource.CloudAccountID(cloudAccountID),
			cloudresource.ServiceCode(serviceCode),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*CloudResource, 0, len(es))
	for _, e := range es {
		results = append(results, cloudResourceFromEnt(e))
	}
	return results, nil
}

// UpdateDiscoveredCloudResource 发现执行器专用写回：全字段 + seen 时间戳。
func (r *EntRepository) UpdateDiscoveredCloudResource(ctx context.Context, cr *CloudResource) error {
	if cr == nil || cr.ID <= 0 {
		return fmt.Errorf("update discovered cloud resource: id required")
	}
	now := time.Now()
	update := r.client.CloudResource.Update().
		Where(cloudresource.TenantID(cr.TenantID), cloudresource.ID(cr.ID)).
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
		SetLastSeenAt(now).
		SetUpdatedAt(now)
	e, err := update.Save(ctx)
	if err != nil {
		return err
	}
	if e == 0 {
		return &ent.NotFoundError{}
	}
	cr.LastSeenAt = &now
	return nil
}

// cloudResourceFromEnt ent.CloudResource → 域结构体唯一映射点。
func cloudResourceFromEnt(e *ent.CloudResource) *CloudResource {
	return &CloudResource{
		ID:                 e.ID,
		CloudAccountID:     e.CloudAccountID,
		ServiceID:          e.ServiceID,
		ResourceID:         e.ResourceID,
		IdentityVersion:    e.IdentityVersion,
		Provider:           e.Provider,
		Partition:          e.Partition,
		CanonicalAccountID: e.CanonicalAccountID,
		ResourceScope:      e.ResourceScope,
		ServiceCode:        e.ServiceCode,
		ResourceType:       e.ResourceType,
		IdentityHash:       e.IdentityHash,
		SourceID:           e.SourceID,
		SourceFingerprint:  e.SourceFingerprint,
		MissingCount:       e.MissingCount,
		ResourceName:       e.ResourceName,
		Region:             e.Region,
		Zone:               e.Zone,
		Status:             e.Status,
		Tags:               e.Tags,
		Metadata:           e.Metadata,
		FirstSeenAt:        optionalTime(e.FirstSeenAt),
		LastSeenAt:         optionalTime(e.LastSeenAt),
		LifecycleState:     e.LifecycleState,
		TenantID:           e.TenantID,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
	}
}

// ciNumberString 安全解引用 ci_number（Optional+Nillable 字段）
func ciNumberString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
