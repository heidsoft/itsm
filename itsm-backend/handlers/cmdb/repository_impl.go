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
		Name:               e.Name,
		Description:        "",
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
	e, err := r.client.CloudService.UpdateOneID(cs.ID).
		Where(cloudservice.TenantID(cs.TenantID)).
		SetProvider(cs.Provider).
		SetCategory(cs.Category).
		SetServiceCode(cs.ServiceCode).
		SetServiceName(cs.ServiceName).
		SetResourceTypeCode(cs.ResourceTypeCode).
		SetResourceTypeName(cs.ResourceTypeName).
		SetAPIVersion(cs.APIVersion).
		SetAttributeSchema(cs.AttributeSchema).
		SetIsActive(cs.IsActive).
		Save(ctx)
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

func (r *EntRepository) DeleteCloudService(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.CloudService.Delete().
		Where(cloudservice.ID(id), cloudservice.TenantID(tenantID)).
		Exec(ctx)
	return err
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

func (r *EntRepository) DeleteCloudAccount(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.CloudAccount.Delete().
		Where(cloudaccount.ID(id), cloudaccount.TenantID(tenantID)).
		Exec(ctx)
	return err
}

// Cloud resources
func (r *EntRepository) ListCloudResources(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error) {
	q := r.client.CloudResource.Query().Where(cloudresource.TenantID(tenantID))
	if provider != "" {
		q = q.Where(cloudresource.HasAccountWith(cloudaccount.Provider(provider)))
	}
	if serviceID > 0 {
		q = q.Where(cloudresource.ServiceID(serviceID))
	}
	if region != "" {
		q = q.Where(cloudresource.Region(region))
	}
	es, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*CloudResource, 0, len(es))
	for _, e := range es {
		var firstSeenAt *time.Time
		if !e.FirstSeenAt.IsZero() {
			firstSeenAt = &e.FirstSeenAt
		}
		var lastSeenAt *time.Time
		if !e.LastSeenAt.IsZero() {
			lastSeenAt = &e.LastSeenAt
		}
		results = append(results, &CloudResource{
			ID:             e.ID,
			CloudAccountID: e.CloudAccountID,
			ServiceID:      e.ServiceID,
			ResourceID:     e.ResourceID,
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
		})
	}
	return results, nil
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
		ID:             e.ID,
		CloudAccountID: e.CloudAccountID,
		ServiceID:      e.ServiceID,
		ResourceID:     e.ResourceID,
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
		ID:             e.ID,
		CloudAccountID: e.CloudAccountID,
		ServiceID:      e.ServiceID,
		ResourceID:     e.ResourceID,
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
		ID:             e.ID,
		CloudAccountID: e.CloudAccountID,
		ServiceID:      e.ServiceID,
		ResourceID:     e.ResourceID,
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

func (r *EntRepository) DeleteCloudResource(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.CloudResource.Delete().
		Where(cloudresource.ID(id), cloudresource.TenantID(tenantID)).
		Exec(ctx)
	return err
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
		SetEnabled(ds.IsActive).
		SetDescription(ds.Description).
		SetTenantID(ds.TenantID)
	e, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return &DiscoverySource{
		ID:          e.ID,
		Name:        e.Name,
		SourceType:  e.SourceType,
		Provider:    e.Provider,
		IsActive:    e.Enabled,
		Description: e.Description,
		TenantID:    e.TenantID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
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
			ID:          e.ID,
			Name:        e.Name,
			SourceType:  e.SourceType,
			Provider:    e.Provider,
			IsActive:    e.Enabled,
			Description: e.Description,
			TenantID:    e.TenantID,
			CreatedAt:   e.CreatedAt,
			UpdatedAt:   e.UpdatedAt,
		})
	}
	return results, nil
}

func (r *EntRepository) CreateDiscoveryJob(ctx context.Context, job *DiscoveryJob) (*DiscoveryJob, error) {
	create := r.client.DiscoveryJob.Create().
		SetSourceID(job.SourceID).
		SetStatus(job.Status).
		SetTenantID(job.TenantID)
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
		ID:         e.ID,
		SourceID:   e.SourceID,
		Status:     e.Status,
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
			ID:           e.ID,
			JobID:        e.JobID,
			CIID:         e.CiID,
			Action:       e.Action,
			ResourceType: e.ResourceType,
			ResourceID:   e.ResourceID,
			Diff:         e.Diff,
			Status:       e.Status,
			TenantID:     e.TenantID,
			CreatedAt:    e.CreatedAt,
			UpdatedAt:    e.UpdatedAt,
		})
	}
	return results, nil
}
