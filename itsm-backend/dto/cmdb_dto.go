package dto

import "time"

// CreateConfigurationItemRequest 创建配置项请求
type CreateConfigurationItemRequest struct {
	Name            string                 `json:"name" binding:"required,max=255"`
	Type            string                 `json:"type" binding:"required,oneof=server database application network storage"`
	BusinessService *string                `json:"businessService,omitempty" binding:"omitempty,max=255"`
	Owner           *string                `json:"owner,omitempty" binding:"omitempty,max=100"`
	Environment     *string                `json:"environment,omitempty" binding:"omitempty,oneof=production staging development"`
	Location        *string                `json:"location,omitempty" binding:"omitempty,max=255"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	MonitoringData  map[string]interface{} `json:"monitoringData,omitempty"`
	RelatedItemIDs  []int                  `json:"relatedItemIds,omitempty"`
}

// UpdateConfigurationItemRequest 更新配置项请求
type UpdateConfigurationItemRequest struct {
	Name            *string                `json:"name,omitempty" binding:"omitempty,max=255"`
	Type            *string                `json:"type,omitempty" binding:"omitempty,oneof=server database application network storage"`
	Status          *string                `json:"status,omitempty" binding:"omitempty,oneof=active inactive maintenance retired"`
	BusinessService *string                `json:"businessService,omitempty" binding:"omitempty,max=255"`
	Owner           *string                `json:"owner,omitempty" binding:"omitempty,max=100"`
	Environment     *string                `json:"environment,omitempty" binding:"omitempty,oneof=production staging development"`
	Location        *string                `json:"location,omitempty" binding:"omitempty,max=255"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	MonitoringData  map[string]interface{} `json:"monitoringData,omitempty"`
	RelatedItemIDs  []int                  `json:"relatedItemIds,omitempty"`
}

// ListConfigurationItemsRequest 获取配置项列表请求
type ListConfigurationItemsRequest struct {
	Page            int    `form:"page,default=1" binding:"min=1"`
	Size            int    `form:"size,default=10" binding:"min=1,max=100"`
	Type            string `form:"type" binding:"omitempty,oneof=server database application network storage"`
	Status          string `form:"status" binding:"omitempty,oneof=active inactive maintenance retired"`
	BusinessService string `form:"businessService"`
	Owner           string `form:"owner"`
	Environment     string `form:"environment" binding:"omitempty,oneof=production staging development"`
	Search          string `form:"search"`
}

// ConfigurationItemResponse 配置项响应
type ConfigurationItemResponse struct {
	ID              int                    `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Status          string                 `json:"status"`
	BusinessService *string                `json:"businessService,omitempty"`
	Owner           *string                `json:"owner,omitempty"`
	Environment     *string                `json:"environment,omitempty"`
	Location        *string                `json:"location,omitempty"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	MonitoringData  map[string]interface{} `json:"monitoringData,omitempty"`
	TenantID        int                    `json:"tenant_id,omitempty"`
	// 生命周期管理
	LifecycleStatus string                      `json:"lifecycle_status,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time                  `json:"effective_at,omitempty"`
	ExpireAt        *time.Time                  `json:"expire_at,omitempty"`
	CreatedAt       time.Time                   `json:"createdAt"`
	UpdatedAt       time.Time                   `json:"updatedAt"`
	RelatedItems    []ConfigurationItemResponse `json:"relatedItems,omitempty"`
	ParentItems     []ConfigurationItemResponse `json:"parentItems,omitempty"`
}

// ConfigurationItemListResponse 配置项列表响应
type ConfigurationItemListResponse struct {
	Items []ConfigurationItemResponse `json:"items"`
	Total int                         `json:"total"`
	Page  int                         `json:"page"`
	Size  int                         `json:"size"`
}

// ConfigurationItemStatsResponse 配置项统计响应
type ConfigurationItemStatsResponse struct {
	TotalCount              int            `json:"totalCount"`
	ActiveCount             int            `json:"activeCount"`
	InactiveCount           int            `json:"inactiveCount"`
	MaintenanceCount        int            `json:"maintenanceCount"`
	TypeDistribution        map[string]int `json:"typeDistribution"`
	EnvironmentDistribution map[string]int `json:"environmentDistribution"`
}

// CreateCIRequest 创建配置项请求
type CreateCIRequest struct {
	Name                    string                 `json:"name" binding:"required,max=255"`
	CITypeID                int                    `json:"ciTypeId"`
	CITypeIDSnake           int                    `json:"ci_type_id,omitempty"`
	Description             string                 `json:"description"`
	Status                  string                 `json:"status" binding:"required"`
	Environment             string                 `json:"environment,omitempty"`
	Criticality             string                 `json:"criticality,omitempty"`
	AssetTag                string                 `json:"assetTag,omitempty"`
	AssetTagSnake           string                 `json:"asset_tag,omitempty"`
	Attributes              map[string]interface{} `json:"attributes,omitempty"`
	SerialNumber            string                 `json:"serialNumber,omitempty"`
	SerialNumberSnake       string                 `json:"serial_number,omitempty"`
	Model                   string                 `json:"model,omitempty"`
	Vendor                  string                 `json:"vendor,omitempty"`
	Location                string                 `json:"location,omitempty"`
	AssignedTo              string                 `json:"assignedTo,omitempty"`
	AssignedToSnake         string                 `json:"assigned_to,omitempty"`
	OwnedBy                 string                 `json:"ownedBy,omitempty"`
	OwnedBySnake            string                 `json:"owned_by,omitempty"`
	DiscoverySource         string                 `json:"discoverySource,omitempty"`
	DiscoverySourceSnake    string                 `json:"discovery_source,omitempty"`
	Source                  string                 `json:"source,omitempty"`
	CloudProvider           string                 `json:"cloudProvider,omitempty"`
	CloudProviderSnake      string                 `json:"cloud_provider,omitempty"`
	CloudAccountID          string                 `json:"cloudAccountId,omitempty"`
	CloudAccountIDSnake     string                 `json:"cloud_account_id,omitempty"`
	CloudRegion             string                 `json:"cloudRegion,omitempty"`
	CloudRegionSnake        string                 `json:"cloud_region,omitempty"`
	CloudZone               string                 `json:"cloudZone,omitempty"`
	CloudZoneSnake          string                 `json:"cloud_zone,omitempty"`
	CloudResourceID         string                 `json:"cloudResourceId,omitempty"`
	CloudResourceIDSnake    string                 `json:"cloud_resource_id,omitempty"`
	CloudResourceType       string                 `json:"cloudResourceType,omitempty"`
	CloudResourceTypeSnake  string                 `json:"cloud_resource_type,omitempty"`
	CloudMetadata           map[string]interface{} `json:"cloudMetadata,omitempty"`
	CloudMetadataSnake      map[string]interface{} `json:"cloud_metadata,omitempty"`
	CloudTags               map[string]interface{} `json:"cloudTags,omitempty"`
	CloudTagsSnake          map[string]interface{} `json:"cloud_tags,omitempty"`
	CloudMetrics            map[string]interface{} `json:"cloudMetrics,omitempty"`
	CloudMetricsSnake       map[string]interface{} `json:"cloud_metrics,omitempty"`
	CloudSyncTime           *time.Time             `json:"cloudSyncTime,omitempty"`
	CloudSyncTimeSnake      *time.Time             `json:"cloud_sync_time,omitempty"`
	CloudSyncStatus         string                 `json:"cloudSyncStatus,omitempty"`
	CloudSyncStatusSnake    string                 `json:"cloud_sync_status,omitempty"`
	CloudResourceRefID      int                    `json:"cloudResourceRefId,omitempty"`
	CloudResourceRefIDSnake int                    `json:"cloud_resource_ref_id,omitempty"`
	TenantID                int                    `json:"tenant_id,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycle_status,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
	ExpireAt        *time.Time `json:"expire_at,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type string `json:"type,omitempty"`
}

// CIResponse 配置项响应
type CIResponse struct {
	ID                 int                    `json:"id"`
	Name               string                 `json:"name"`
	Type               string                 `json:"type"`
	CITypeID           int                    `json:"ciTypeId"`
	Description        string                 `json:"description"`
	Status             string                 `json:"status"`
	Environment        string                 `json:"environment,omitempty"`
	Criticality        string                 `json:"criticality,omitempty"`
	AssetTag           string                 `json:"assetTag,omitempty"`
	Attributes         map[string]interface{} `json:"attributes,omitempty"`
	SerialNumber       string                 `json:"serialNumber,omitempty"`
	Model              string                 `json:"model,omitempty"`
	Vendor             string                 `json:"vendor,omitempty"`
	Location           string                 `json:"location,omitempty"`
	AssignedTo         string                 `json:"assignedTo,omitempty"`
	OwnedBy            string                 `json:"ownedBy,omitempty"`
	DiscoverySource    string                 `json:"discoverySource,omitempty"`
	LastDiscovered     time.Time              `json:"lastDiscovered,omitempty"`
	Source             string                 `json:"source,omitempty"`
	CloudProvider      string                 `json:"cloudProvider,omitempty"`
	CloudAccountID     string                 `json:"cloudAccountId,omitempty"`
	CloudRegion        string                 `json:"cloudRegion,omitempty"`
	CloudZone          string                 `json:"cloudZone,omitempty"`
	CloudResourceID    string                 `json:"cloudResourceId,omitempty"`
	CloudResourceType  string                 `json:"cloudResourceType,omitempty"`
	CloudMetadata      map[string]interface{} `json:"cloudMetadata,omitempty"`
	CloudTags          map[string]interface{} `json:"cloudTags,omitempty"`
	CloudMetrics       map[string]interface{} `json:"cloudMetrics,omitempty"`
	CloudSyncTime      *time.Time             `json:"cloudSyncTime,omitempty"`
	CloudSyncStatus    string                 `json:"cloudSyncStatus,omitempty"`
	CloudResourceRefID int                    `json:"cloudResourceRefId,omitempty"`
	TenantID           int                    `json:"tenantId"`
	// 生命周期管理
	LifecycleStatus   string                    `json:"lifecycleStatus,omitempty"`
	EffectiveAt       *time.Time                `json:"effectiveAt,omitempty"`
	ExpireAt          *time.Time                `json:"expireAt,omitempty"`
	CreatedAt         time.Time                 `json:"createdAt"`
	UpdatedAt         time.Time                 `json:"updatedAt"`
	OutgoingRelations []*CIRelationshipResponse `json:"outgoingRelations,omitempty"`
	IncomingRelations []*CIRelationshipResponse `json:"incomingRelations,omitempty"`
	Tags              []string                  `json:"tags,omitempty"`
	TagDetails        []*CITagResponse          `json:"tagDetails,omitempty"`
}

// ListCIsRequest 获取配置项列表请求
type ListCIsRequest struct {
	TenantID int `json:"tenant_id,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycle_status,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
	ExpireAt        *time.Time `json:"expire_at,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type     string `json:"type,omitempty"`
	CITypeID int    `json:"ciTypeId,omitempty"`
	Status   string `json:"status,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

// ListCIsResponse 配置项列表响应
type ListCIsResponse struct {
	CIs   []*CIResponse `json:"cis"`
	Total int           `json:"total"`
}

// UpdateCIRequest 更新配置项请求
type UpdateCIRequest struct {
	CITypeID                int                    `json:"ciTypeId,omitempty"`
	CITypeIDSnake           int                    `json:"ci_type_id,omitempty"`
	Name                    string                 `json:"name,omitempty"`
	Description             string                 `json:"description,omitempty"`
	Status                  string                 `json:"status,omitempty"`
	Environment             string                 `json:"environment,omitempty"`
	Criticality             string                 `json:"criticality,omitempty"`
	AssetTag                string                 `json:"assetTag,omitempty"`
	AssetTagSnake           string                 `json:"asset_tag,omitempty"`
	Attributes              map[string]interface{} `json:"attributes,omitempty"`
	SerialNumber            string                 `json:"serialNumber,omitempty"`
	SerialNumberSnake       string                 `json:"serial_number,omitempty"`
	Model                   string                 `json:"model,omitempty"`
	Vendor                  string                 `json:"vendor,omitempty"`
	Location                string                 `json:"location,omitempty"`
	AssignedTo              string                 `json:"assignedTo,omitempty"`
	AssignedToSnake         string                 `json:"assigned_to,omitempty"`
	OwnedBy                 string                 `json:"ownedBy,omitempty"`
	OwnedBySnake            string                 `json:"owned_by,omitempty"`
	DiscoverySource         string                 `json:"discoverySource,omitempty"`
	DiscoverySourceSnake    string                 `json:"discovery_source,omitempty"`
	Source                  string                 `json:"source,omitempty"`
	CloudProvider           string                 `json:"cloudProvider,omitempty"`
	CloudProviderSnake      string                 `json:"cloud_provider,omitempty"`
	CloudAccountID          string                 `json:"cloudAccountId,omitempty"`
	CloudAccountIDSnake     string                 `json:"cloud_account_id,omitempty"`
	CloudRegion             string                 `json:"cloudRegion,omitempty"`
	CloudRegionSnake        string                 `json:"cloud_region,omitempty"`
	CloudZone               string                 `json:"cloudZone,omitempty"`
	CloudZoneSnake          string                 `json:"cloud_zone,omitempty"`
	CloudResourceID         string                 `json:"cloudResourceId,omitempty"`
	CloudResourceIDSnake    string                 `json:"cloud_resource_id,omitempty"`
	CloudResourceType       string                 `json:"cloudResourceType,omitempty"`
	CloudResourceTypeSnake  string                 `json:"cloud_resource_type,omitempty"`
	CloudMetadata           map[string]interface{} `json:"cloudMetadata,omitempty"`
	CloudMetadataSnake      map[string]interface{} `json:"cloud_metadata,omitempty"`
	CloudTags               map[string]interface{} `json:"cloudTags,omitempty"`
	CloudTagsSnake          map[string]interface{} `json:"cloud_tags,omitempty"`
	CloudMetrics            map[string]interface{} `json:"cloudMetrics,omitempty"`
	CloudMetricsSnake       map[string]interface{} `json:"cloud_metrics,omitempty"`
	CloudSyncTime           *time.Time             `json:"cloudSyncTime,omitempty"`
	CloudSyncTimeSnake      *time.Time             `json:"cloud_sync_time,omitempty"`
	CloudSyncStatus         string                 `json:"cloudSyncStatus,omitempty"`
	CloudSyncStatusSnake    string                 `json:"cloud_sync_status,omitempty"`
	CloudResourceRefID      int                    `json:"cloudResourceRefId,omitempty"`
	CloudResourceRefIDSnake int                    `json:"cloud_resource_ref_id,omitempty"`
	// 生命周期管理
	LifecycleStatus      string     `json:"lifecycleStatus,omitempty"`
	LifecycleStatusSnake string     `json:"lifecycle_status,omitempty"`
	EffectiveAt          *time.Time `json:"effectiveAt,omitempty"`
	EffectiveAtSnake     *time.Time `json:"effective_at,omitempty"`
	ExpireAt             *time.Time `json:"expireAt,omitempty"`
	ExpireAtSnake        *time.Time `json:"expire_at,omitempty"`
}

// CloudService DTOs
type CloudServiceRequest struct {
	ParentID         int                    `json:"parentId,omitempty"`
	Provider         string                 `json:"provider" binding:"required"`
	Category         string                 `json:"category,omitempty"`
	ServiceCode      string                 `json:"serviceCode" binding:"required"`
	ServiceName      string                 `json:"serviceName" binding:"required"`
	ResourceTypeCode string                 `json:"resourceTypeCode" binding:"required"`
	ResourceTypeName string                 `json:"resourceTypeName" binding:"required"`
	APIVersion       string                 `json:"apiVersion,omitempty"`
	AttributeSchema  map[string]interface{} `json:"attributeSchema,omitempty"`
	IsSystem         *bool                  `json:"isSystem,omitempty"`
	IsActive         *bool                  `json:"isActive,omitempty"`
}

type CloudServiceResponse struct {
	ID               int                    `json:"id"`
	ParentID         int                    `json:"parentId"`
	Provider         string                 `json:"provider"`
	Category         string                 `json:"category,omitempty"`
	ServiceCode      string                 `json:"serviceCode"`
	ServiceName      string                 `json:"serviceName"`
	ResourceTypeCode string                 `json:"resourceTypeCode"`
	ResourceTypeName string                 `json:"resourceTypeName"`
	APIVersion       string                 `json:"apiVersion,omitempty"`
	AttributeSchema  map[string]interface{} `json:"attributeSchema,omitempty"`
	IsSystem         bool                   `json:"isSystem"`
	IsActive         bool                   `json:"isActive"`
	TenantID         int                    `json:"tenant_id,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycle_status,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
	ExpireAt        *time.Time `json:"expire_at,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CloudAccount DTOs
type CloudAccountRequest struct {
	Provider        string   `json:"provider" binding:"required"`
	AccountID       string   `json:"accountId" binding:"required"`
	AccountName     string   `json:"accountName" binding:"required"`
	CredentialRef   string   `json:"credentialRef,omitempty"`
	RegionWhitelist []string `json:"regionWhitelist,omitempty"`
	IsActive        *bool    `json:"isActive,omitempty"`
}

type CloudAccountResponse struct {
	ID              int      `json:"id"`
	Provider        string   `json:"provider"`
	AccountID       string   `json:"accountId"`
	AccountName     string   `json:"accountName"`
	CredentialRef   string   `json:"credentialRef,omitempty"`
	RegionWhitelist []string `json:"regionWhitelist,omitempty"`
	IsActive        bool     `json:"isActive"`
	TenantID        int      `json:"tenant_id,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycle_status,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
	ExpireAt        *time.Time `json:"expire_at,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CloudResource DTOs
type CloudResourceResponse struct {
	ID             int                    `json:"id"`
	CloudAccountID int                    `json:"cloudAccountId"`
	ServiceID      int                    `json:"serviceId"`
	ResourceID     string                 `json:"resourceId"`
	ResourceName   string                 `json:"resourceName,omitempty"`
	Region         string                 `json:"region,omitempty"`
	Zone           string                 `json:"zone,omitempty"`
	Status         string                 `json:"status,omitempty"`
	Tags           map[string]string      `json:"tags,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	FirstSeenAt    *time.Time             `json:"firstSeenAt,omitempty"`
	LastSeenAt     *time.Time             `json:"lastSeenAt,omitempty"`
	LifecycleState string                 `json:"lifecycleState,omitempty"`
	TenantID       int                    `json:"tenant_id,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycle_status,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
	ExpireAt        *time.Time `json:"expire_at,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// RelationshipType DTOs
type RelationshipTypeResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Directional bool   `json:"directional"`
	ReverseName string `json:"reverseName,omitempty"`
	Description string `json:"description,omitempty"`
	TenantID    int    `json:"tenant_id,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycle_status,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
	ExpireAt        *time.Time `json:"expire_at,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Discovery DTOs
type DiscoverySourceRequest struct {
	Name        string `json:"name" binding:"required"`
	SourceType  string `json:"sourceType" binding:"required"`
	Provider    string `json:"provider,omitempty"`
	IsActive    *bool  `json:"isActive,omitempty"`
	Description string `json:"description,omitempty"`
}

type DiscoverySourceResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SourceType  string `json:"sourceType"`
	Provider    string `json:"provider,omitempty"`
	IsActive    bool   `json:"isActive"`
	Description string `json:"description,omitempty"`
	TenantID    int    `json:"tenant_id,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycle_status,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
	ExpireAt        *time.Time `json:"expire_at,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DiscoveryJobRequest struct {
	SourceID string `json:"sourceId" binding:"required"`
}

type DiscoveryJobResponse struct {
	ID         int                    `json:"id"`
	SourceID   string                 `json:"sourceId"`
	Status     string                 `json:"status"`
	StartedAt  *time.Time             `json:"startedAt,omitempty"`
	FinishedAt *time.Time             `json:"finishedAt,omitempty"`
	Summary    map[string]interface{} `json:"summary,omitempty"`
	TenantID   int                    `json:"tenant_id,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycle_status,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
	ExpireAt        *time.Time `json:"expire_at,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DiscoveryResultResponse struct {
	ID           int                    `json:"id"`
	JobID        int                    `json:"jobId"`
	CIID         int                    `json:"ciId,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resourceType,omitempty"`
	ResourceID   string                 `json:"resourceId,omitempty"`
	Diff         map[string]interface{} `json:"diff,omitempty"`
	Status       string                 `json:"status"`
	TenantID     int                    `json:"tenant_id,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycle_status,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
	ExpireAt        *time.Time `json:"expire_at,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Reconciliation DTOs
type ReconciliationSummary struct {
	ResourceTotal        int `json:"resourceTotal"`
	BoundResourceCount   int `json:"boundResourceCount"`
	UnboundResourceCount int `json:"unboundResourceCount"`
	OrphanCICount        int `json:"orphanCICount"`
	UnlinkedCICount      int `json:"unlinkedCICount"`
}

type ReconciliationResponse struct {
	Summary          ReconciliationSummary    `json:"summary"`
	UnboundResources []*CloudResourceResponse `json:"unboundResources"`
	OrphanCIs        []*CIResponse            `json:"orphanCIs"`
	UnlinkedCIs      []*CIResponse            `json:"unlinkedCIs"`
}

// ---- 以下为从 service/cmdb_service.go 迁移过来的内部服务请求/响应体 ----
// 保留 snake_case json 标签风格，供 CMDBService 内部使用

// CMDBCreateCIRequest service 层创建配置项请求（内部使用）
type CMDBCreateCIRequest struct {
	Name        string `json:"name"`
	CiType      string `json:"ci_type"`
	CiTypeID    int    `json:"ci_type_id"`
	Status      string `json:"status"`
	Environment string `json:"environment"`
	Criticality string `json:"criticality"`
	TenantID    int    `json:"tenant_id,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycle_status,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
	ExpireAt        *time.Time `json:"expire_at,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type            string                  `json:"type,omitempty"`
	AssetTag        *string                 `json:"asset_tag,omitempty"`
	SerialNumber    *string                 `json:"serial_number,omitempty"`
	Location        *string                 `json:"location,omitempty"`
	AssignedTo      *string                 `json:"assigned_to,omitempty"`
	OwnedBy         *string                 `json:"owned_by,omitempty"`
	DiscoverySource *string                 `json:"discovery_source,omitempty"`
	Attributes      *map[string]interface{} `json:"attributes,omitempty"`
}

// CMDBListCIsRequest service 层查询配置项列表请求（内部使用）
type CMDBListCIsRequest struct {
	TenantID int `json:"tenant_id,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycle_status,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
	ExpireAt        *time.Time `json:"expire_at,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type        string `json:"type,omitempty"`
	CiType      string `json:"ci_type,omitempty"`
	Status      string `json:"status,omitempty"`
	Environment string `json:"environment,omitempty"`
	Offset      int    `json:"offset"`
	Limit       int    `json:"limit"`
}

// CMDBCreateRelationshipRequest service 层创建 CI 关系请求（内部使用）
type CMDBCreateRelationshipRequest struct {
	SourceCIID  int     `json:"source_ci_id"`
	TargetCIID  int     `json:"target_ci_id"`
	Type        string  `json:"type"`
	Description *string `json:"description,omitempty"`
	TenantID    int     `json:"tenant_id,omitempty"`
}

// CMDBUpdateRelationshipRequest service 层更新 CI 关系请求（内部使用）
type CMDBUpdateRelationshipRequest struct {
	Type        string  `json:"type,omitempty"`
	Description *string `json:"description,omitempty"`
}

// CIHistoryResponse CI历史记录响应
type CIHistoryResponse struct {
	ID            int                    `json:"id"`
	CIID          int                    `json:"ciId"`
	Version       int                    `json:"version"`
	Operation     string                 `json:"operation"`
	Before        map[string]interface{} `json:"before,omitempty"`
	After         map[string]interface{} `json:"after,omitempty"`
	ChangedFields []string               `json:"changedFields,omitempty"`
	OperatorID    int                    `json:"operatorId"`
	OperatorName  string                 `json:"operatorName,omitempty"`
	Remark        string                 `json:"remark,omitempty"`
	TenantID      int                    `json:"tenant_id,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycle_status,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effective_at,omitempty"`
	ExpireAt        *time.Time `json:"expire_at,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// CIHistoryListResponse CI历史列表响应
type CIHistoryListResponse struct {
	Items []*CIHistoryResponse `json:"items"`
	Total int                  `json:"total"`
	Page  int                  `json:"page"`
	Size  int                  `json:"size"`
}

// RevertCIVersionRequest 回滚CI版本请求
type RevertCIVersionRequest struct {
	Version int    `json:"version" binding:"required,min=1"`
	Remark  string `json:"remark,omitempty"`
}
