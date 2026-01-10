package cmdb

import (
	"context"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/cloudaccount"
	"itsm-backend/ent/cloudresource"
	"itsm-backend/ent/cloudservice"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/discoveryresult"
	"itsm-backend/ent/discoverysource"
	"itsm-backend/ent/relationshiptype"
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

// Map ent CIType to domain CIType
func toTypeDomain(e *ent.CIType) *CIType {
	if e == nil {
		return nil
	}
	return &CIType{
		ID:              e.ID,
		Name:            e.Name,
		Description:     e.Description,
		Icon:            e.Icon,
		Color:           e.Color,
		AttributeSchema: e.AttributeSchema,
		IsActive:        e.IsActive,
		TenantID:        e.TenantID,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func (r *EntRepository) CreateCI(ctx context.Context, ci *ConfigurationItem) (*ConfigurationItem, error) {
	ciType, err := r.client.CIType.Query().
		Where(citype.ID(ci.CITypeID), citype.TenantID(ci.TenantID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	create := r.client.ConfigurationItem.Create().
		SetName(ci.Name).
		SetCiTypeID(ci.CITypeID).
		SetCiType(ciType.Name).
		SetStatus(ci.Status).
		SetEnvironment(ci.Environment).
		SetCriticality(ci.Criticality).
		SetAssetTag(ci.AssetTag).
		SetLocation(ci.Location).
		SetSerialNumber(ci.SerialNumber).
		SetModel(ci.Model).
		SetVendor(ci.Vendor).
		SetAssignedTo(ci.AssignedTo).
		SetOwnedBy(ci.OwnedBy).
		SetDiscoverySource(ci.DiscoverySource).
		SetSource(ci.Source).
		SetCloudProvider(ci.CloudProvider).
		SetCloudAccountID(ci.CloudAccountID).
		SetCloudRegion(ci.CloudRegion).
		SetCloudZone(ci.CloudZone).
		SetCloudResourceID(ci.CloudResourceID).
		SetCloudResourceType(ci.CloudResourceType).
		SetTenantID(ci.TenantID)

	if ci.Attributes != nil {
		create = create.SetAttributes(ci.Attributes)
	}
	if ci.CloudMetadata != nil {
		create = create.SetCloudMetadata(ci.CloudMetadata)
	}
	if ci.CloudTags != nil {
		create = create.SetCloudTags(ci.CloudTags)
	}
	if ci.CloudMetrics != nil {
		create = create.SetCloudMetrics(ci.CloudMetrics)
	}
	if ci.CloudSyncTime != nil {
		create = create.SetCloudSyncTime(*ci.CloudSyncTime)
	}
	if ci.CloudSyncStatus != "" {
		create = create.SetCloudSyncStatus(ci.CloudSyncStatus)
	}
	if ci.CloudResourceRefID > 0 {
		create = create.SetCloudResourceRefID(ci.CloudResourceRefID)
	}

	e, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toCIDomain(e), nil
}

func (r *EntRepository) GetCI(ctx context.Context, id int, tenantID int) (*ConfigurationItem, error) {
	e, err := r.client.ConfigurationItem.Query().
		Where(configurationitem.ID(id), configurationitem.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return toCIDomain(e), nil
}

func (r *EntRepository) ListCIs(ctx context.Context, tenantID int, page, size int, ciTypeID int, status string) ([]*ConfigurationItem, int, error) {
	q := r.client.ConfigurationItem.Query().Where(configurationitem.TenantID(tenantID))
	if ciTypeID > 0 {
		q = q.Where(configurationitem.CiTypeID(ciTypeID))
	}
	if status != "" {
		q = q.Where(configurationitem.Status(status))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	es, err := q.Limit(size).Offset((page - 1) * size).Order(ent.Desc(configurationitem.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var results []*ConfigurationItem
	for _, e := range es {
		results = append(results, toCIDomain(e))
	}
	return results, total, nil
}

func (r *EntRepository) UpdateCI(ctx context.Context, ci *ConfigurationItem) (*ConfigurationItem, error) {
	ciType, err := r.client.CIType.Query().
		Where(citype.ID(ci.CITypeID), citype.TenantID(ci.TenantID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	update := r.client.ConfigurationItem.UpdateOneID(ci.ID).
		SetName(ci.Name).
		SetCiTypeID(ci.CITypeID).
		SetCiType(ciType.Name).
		SetStatus(ci.Status).
		SetEnvironment(ci.Environment).
		SetCriticality(ci.Criticality).
		SetAssetTag(ci.AssetTag).
		SetLocation(ci.Location).
		SetSerialNumber(ci.SerialNumber).
		SetModel(ci.Model).
		SetVendor(ci.Vendor).
		SetAssignedTo(ci.AssignedTo).
		SetOwnedBy(ci.OwnedBy).
		SetDiscoverySource(ci.DiscoverySource).
		SetSource(ci.Source).
		SetCloudProvider(ci.CloudProvider).
		SetCloudAccountID(ci.CloudAccountID).
		SetCloudRegion(ci.CloudRegion).
		SetCloudZone(ci.CloudZone).
		SetCloudResourceID(ci.CloudResourceID).
		SetCloudResourceType(ci.CloudResourceType)

	if ci.Attributes != nil {
		update = update.SetAttributes(ci.Attributes)
	}
	if ci.CloudMetadata != nil {
		update = update.SetCloudMetadata(ci.CloudMetadata)
	}
	if ci.CloudTags != nil {
		update = update.SetCloudTags(ci.CloudTags)
	}
	if ci.CloudMetrics != nil {
		update = update.SetCloudMetrics(ci.CloudMetrics)
	}
	if ci.CloudSyncTime != nil {
		update = update.SetCloudSyncTime(*ci.CloudSyncTime)
	}
	if ci.CloudSyncStatus != "" {
		update = update.SetCloudSyncStatus(ci.CloudSyncStatus)
	}
	if ci.CloudResourceRefID > 0 {
		update = update.SetCloudResourceRefID(ci.CloudResourceRefID)
	}

	e, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	return toCIDomain(e), nil
}

func (r *EntRepository) DeleteCI(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.ConfigurationItem.Delete().
		Where(configurationitem.ID(id), configurationitem.TenantID(tenantID)).
		Exec(ctx)
	return err
}

func (r *EntRepository) GetStats(ctx context.Context, tenantID int) (*Stats, error) {
	total, _ := r.client.ConfigurationItem.Query().Where(configurationitem.TenantID(tenantID)).Count(ctx)
	active, _ := r.client.ConfigurationItem.Query().Where(configurationitem.TenantID(tenantID), configurationitem.Status("active")).Count(ctx)
	inactive, _ := r.client.ConfigurationItem.Query().Where(configurationitem.TenantID(tenantID), configurationitem.Status("inactive")).Count(ctx)
	maintenance, _ := r.client.ConfigurationItem.Query().Where(configurationitem.TenantID(tenantID), configurationitem.Status("maintenance")).Count(ctx)

	// Simple type distribution
	dist := make(map[string]int)
	// We might need a aggregation query for this, but for now we can do a simplified version or skip if too complex for Ent raw

	return &Stats{
		TotalCount:       total,
		ActiveCount:      active,
		InactiveCount:    inactive,
		MaintenanceCount: maintenance,
		TypeDistribution: dist,
	}, nil
}

// CITypes
func (r *EntRepository) CreateCIType(ctx context.Context, ct *CIType) (*CIType, error) {
	e, err := r.client.CIType.Create().
		SetName(ct.Name).
		SetDescription(ct.Description).
		SetIcon(ct.Icon).
		SetColor(ct.Color).
		SetAttributeSchema(ct.AttributeSchema).
		SetIsActive(ct.IsActive).
		SetTenantID(ct.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toTypeDomain(e), nil
}

func (r *EntRepository) GetCIType(ctx context.Context, id int, tenantID int) (*CIType, error) {
	e, err := r.client.CIType.Query().
		Where(citype.ID(id), citype.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return toTypeDomain(e), nil
}

func (r *EntRepository) ListCITypes(ctx context.Context, tenantID int) ([]*CIType, error) {
	es, err := r.client.CIType.Query().Where(citype.TenantID(tenantID)).All(ctx)
	if err != nil {
		return nil, err
	}
	var results []*CIType
	for _, e := range es {
		results = append(results, toTypeDomain(e))
	}
	return results, nil
}

// Relationships
func (r *EntRepository) CreateRelationship(ctx context.Context, rel *CIRelationship) (*CIRelationship, error) {
	e, err := r.client.CIRelationship.Create().
		SetParentID(rel.SourceCIID).
		SetChildID(rel.TargetCIID).
		SetType(rel.Description). // 使用Description作为Type
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return &CIRelationship{
		ID:                 e.ID,
		SourceCIID:         e.ParentID,
		TargetCIID:         e.ChildID,
		RelationshipTypeID: 0, // 新schema没有这个字段
		Description:        e.Type,
		TenantID:           0, // 新schema没有这个字段
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.CreatedAt, // 新schema只有CreatedAt
	}, nil
}

func (r *EntRepository) GetRelationships(ctx context.Context, ciID int) ([]*CIRelationship, error) {
	es, err := r.client.CIRelationship.Query().
		Where(cirelationship.Or(
			cirelationship.ParentID(ciID),
			cirelationship.ChildID(ciID),
		)).All(ctx)
	if err != nil {
		return nil, err
	}
	var results []*CIRelationship
	for _, e := range es {
		results = append(results, &CIRelationship{
			ID:                 e.ID,
			SourceCIID:         e.ParentID,
			TargetCIID:         e.ChildID,
			RelationshipTypeID: 0,
			Description:        e.Type,
			TenantID:           0,
			CreatedAt:          e.CreatedAt,
			UpdatedAt:          e.CreatedAt,
		})
	}
	return results, nil
}

func (r *EntRepository) DeleteRelationship(ctx context.Context, id int, tenantID int) error {
	_, err := r.client.CIRelationship.Delete().
		Where(cirelationship.ID(id)).
		Exec(ctx)
	return err
}

// Relationship types
func (r *EntRepository) ListRelationshipTypes(ctx context.Context, tenantID int) ([]*RelationshipType, error) {
	es, err := r.client.RelationshipType.Query().Where(relationshiptype.TenantID(tenantID)).All(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*RelationshipType, 0, len(es))
	for _, e := range es {
		results = append(results, &RelationshipType{
			ID:          e.ID,
			Name:        e.Name,
			Directional: e.Directional,
			ReverseName: e.ReverseName,
			Description: e.Description,
			TenantID:    e.TenantID,
			CreatedAt:   e.CreatedAt,
			UpdatedAt:   e.UpdatedAt,
		})
	}
	return results, nil
}

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
		Where(discoverysource.Or(
			discoverysource.TenantID(tenantID),
			discoverysource.TenantIDIsNil(),
		)).
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
