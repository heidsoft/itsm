package dto

import "time"

// CreateConfigurationItemRequest 创建配置项请求
type CreateConfigurationItemRequest struct {
	Name            string                 `json:"name" binding:"required,max=255"`
	Type            string                 `json:"type" binding:"required,oneof=server database application network storage"`
	BusinessService *string                `json:"business_service,omitempty" binding:"omitempty,max=255"`
	Owner           *string                `json:"owner,omitempty" binding:"omitempty,max=100"`
	Environment     *string                `json:"environment,omitempty" binding:"omitempty,oneof=production staging development"`
	Location        *string                `json:"location,omitempty" binding:"omitempty,max=255"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	MonitoringData  map[string]interface{} `json:"monitoring_data,omitempty"`
	RelatedItemIDs  []int                  `json:"related_item_ids,omitempty"`
}

// UpdateConfigurationItemRequest 更新配置项请求
type UpdateConfigurationItemRequest struct {
	Name            *string                `json:"name,omitempty" binding:"omitempty,max=255"`
	Type            *string                `json:"type,omitempty" binding:"omitempty,oneof=server database application network storage"`
	Status          *string                `json:"status,omitempty" binding:"omitempty,oneof=active inactive maintenance"`
	BusinessService *string                `json:"business_service,omitempty" binding:"omitempty,max=255"`
	Owner           *string                `json:"owner,omitempty" binding:"omitempty,max=100"`
	Environment     *string                `json:"environment,omitempty" binding:"omitempty,oneof=production staging development"`
	Location        *string                `json:"location,omitempty" binding:"omitempty,max=255"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	MonitoringData  map[string]interface{} `json:"monitoring_data,omitempty"`
	RelatedItemIDs  []int                  `json:"related_item_ids,omitempty"`
}

// ListConfigurationItemsRequest 获取配置项列表请求
type ListConfigurationItemsRequest struct {
	Page            int    `form:"page,default=1" binding:"min=1"`
	Size            int    `form:"size,default=10" binding:"min=1,max=100"`
	Type            string `form:"type" binding:"omitempty,oneof=server database application network storage"`
	Status          string `form:"status" binding:"omitempty,oneof=active inactive maintenance"`
	BusinessService string `form:"business_service"`
	Owner           string `form:"owner"`
	Environment     string `form:"environment" binding:"omitempty,oneof=production staging development"`
	Search          string `form:"search"`
}

// ConfigurationItemResponse 配置项响应
type ConfigurationItemResponse struct {
	ID              int                         `json:"id"`
	Name            string                      `json:"name"`
	Type            string                      `json:"type"`
	Status          string                      `json:"status"`
	BusinessService *string                     `json:"business_service,omitempty"`
	Owner           *string                     `json:"owner,omitempty"`
	Environment     *string                     `json:"environment,omitempty"`
	Location        *string                     `json:"location,omitempty"`
	Attributes      map[string]interface{}      `json:"attributes,omitempty"`
	MonitoringData  map[string]interface{}      `json:"monitoring_data,omitempty"`
	TenantID        int                         `json:"tenant_id"`
	CreatedAt       time.Time                   `json:"created_at"`
	UpdatedAt       time.Time                   `json:"updated_at"`
	RelatedItems    []ConfigurationItemResponse `json:"related_items,omitempty"`
	ParentItems     []ConfigurationItemResponse `json:"parent_items,omitempty"`
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
	TotalCount              int            `json:"total_count"`
	ActiveCount             int            `json:"active_count"`
	InactiveCount           int            `json:"inactive_count"`
	MaintenanceCount        int            `json:"maintenance_count"`
	TypeDistribution        map[string]int `json:"type_distribution"`
	EnvironmentDistribution map[string]int `json:"environment_distribution"`
}

// CreateCIRequest 创建配置项请求
type CreateCIRequest struct {
	Name               string                 `json:"name" binding:"required,max=255"`
	CITypeID           int                    `json:"ci_type_id" binding:"required"`
	Description        string                 `json:"description"`
	Status             string                 `json:"status" binding:"required,oneof=active inactive maintenance"`
	Environment        string                 `json:"environment,omitempty"`
	Criticality        string                 `json:"criticality,omitempty"`
	AssetTag           string                 `json:"asset_tag,omitempty"`
	Attributes         map[string]interface{} `json:"attributes,omitempty"`
	SerialNumber       string                 `json:"serial_number,omitempty"`
	Model              string                 `json:"model,omitempty"`
	Vendor             string                 `json:"vendor,omitempty"`
	Location           string                 `json:"location,omitempty"`
	AssignedTo         string                 `json:"assigned_to,omitempty"`
	OwnedBy            string                 `json:"owned_by,omitempty"`
	DiscoverySource    string                 `json:"discovery_source,omitempty"`
	Source             string                 `json:"source,omitempty"`
	CloudProvider      string                 `json:"cloud_provider,omitempty"`
	CloudAccountID     string                 `json:"cloud_account_id,omitempty"`
	CloudRegion        string                 `json:"cloud_region,omitempty"`
	CloudZone          string                 `json:"cloud_zone,omitempty"`
	CloudResourceID    string                 `json:"cloud_resource_id,omitempty"`
	CloudResourceType  string                 `json:"cloud_resource_type,omitempty"`
	CloudMetadata      map[string]interface{} `json:"cloud_metadata,omitempty"`
	CloudTags          map[string]interface{} `json:"cloud_tags,omitempty"`
	CloudMetrics       map[string]interface{} `json:"cloud_metrics,omitempty"`
	CloudSyncTime      *time.Time             `json:"cloud_sync_time,omitempty"`
	CloudSyncStatus    string                 `json:"cloud_sync_status,omitempty"`
	CloudResourceRefID int                    `json:"cloud_resource_ref_id,omitempty"`
	TenantID           int                    `json:"tenant_id,omitempty"`
}

// CIResponse 配置项响应
type CIResponse struct {
	ID                 int                    `json:"id"`
	Name               string                 `json:"name"`
	Type               string                 `json:"type"`
	CITypeID           int                    `json:"ci_type_id"`
	Description        string                 `json:"description"`
	Status             string                 `json:"status"`
	Environment        string                 `json:"environment,omitempty"`
	Criticality        string                 `json:"criticality,omitempty"`
	AssetTag           string                 `json:"asset_tag,omitempty"`
	Attributes         map[string]interface{} `json:"attributes,omitempty"`
	SerialNumber       string                 `json:"serial_number,omitempty"`
	Model              string                 `json:"model,omitempty"`
	Vendor             string                 `json:"vendor,omitempty"`
	Location           string                 `json:"location,omitempty"`
	AssignedTo         string                 `json:"assigned_to,omitempty"`
	OwnedBy            string                 `json:"owned_by,omitempty"`
	DiscoverySource    string                 `json:"discovery_source,omitempty"`
	Source             string                 `json:"source,omitempty"`
	CloudProvider      string                 `json:"cloud_provider,omitempty"`
	CloudAccountID     string                 `json:"cloud_account_id,omitempty"`
	CloudRegion        string                 `json:"cloud_region,omitempty"`
	CloudZone          string                 `json:"cloud_zone,omitempty"`
	CloudResourceID    string                 `json:"cloud_resource_id,omitempty"`
	CloudResourceType  string                 `json:"cloud_resource_type,omitempty"`
	CloudMetadata      map[string]interface{} `json:"cloud_metadata,omitempty"`
	CloudTags          map[string]interface{} `json:"cloud_tags,omitempty"`
	CloudMetrics       map[string]interface{} `json:"cloud_metrics,omitempty"`
	CloudSyncTime      *time.Time             `json:"cloud_sync_time,omitempty"`
	CloudSyncStatus    string                 `json:"cloud_sync_status,omitempty"`
	CloudResourceRefID int                    `json:"cloud_resource_ref_id,omitempty"`
	TenantID           int                    `json:"tenant_id"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// ListCIsRequest 获取配置项列表请求
type ListCIsRequest struct {
	TenantID int    `json:"tenant_id" binding:"required"`
	CITypeID int    `json:"ci_type_id,omitempty"`
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
	CITypeID           int                    `json:"ci_type_id,omitempty"`
	Name               string                 `json:"name,omitempty"`
	Description        string                 `json:"description,omitempty"`
	Status             string                 `json:"status,omitempty"`
	Environment        string                 `json:"environment,omitempty"`
	Criticality        string                 `json:"criticality,omitempty"`
	AssetTag           string                 `json:"asset_tag,omitempty"`
	Attributes         map[string]interface{} `json:"attributes,omitempty"`
	SerialNumber       string                 `json:"serial_number,omitempty"`
	Model              string                 `json:"model,omitempty"`
	Vendor             string                 `json:"vendor,omitempty"`
	Location           string                 `json:"location,omitempty"`
	AssignedTo         string                 `json:"assigned_to,omitempty"`
	OwnedBy            string                 `json:"owned_by,omitempty"`
	DiscoverySource    string                 `json:"discovery_source,omitempty"`
	Source             string                 `json:"source,omitempty"`
	CloudProvider      string                 `json:"cloud_provider,omitempty"`
	CloudAccountID     string                 `json:"cloud_account_id,omitempty"`
	CloudRegion        string                 `json:"cloud_region,omitempty"`
	CloudZone          string                 `json:"cloud_zone,omitempty"`
	CloudResourceID    string                 `json:"cloud_resource_id,omitempty"`
	CloudResourceType  string                 `json:"cloud_resource_type,omitempty"`
	CloudMetadata      map[string]interface{} `json:"cloud_metadata,omitempty"`
	CloudTags          map[string]interface{} `json:"cloud_tags,omitempty"`
	CloudMetrics       map[string]interface{} `json:"cloud_metrics,omitempty"`
	CloudSyncTime      *time.Time             `json:"cloud_sync_time,omitempty"`
	CloudSyncStatus    string                 `json:"cloud_sync_status,omitempty"`
	CloudResourceRefID int                    `json:"cloud_resource_ref_id,omitempty"`
}

// CloudService DTOs
type CloudServiceRequest struct {
	ParentID         int                    `json:"parent_id,omitempty"`
	Provider         string                 `json:"provider" binding:"required"`
	Category         string                 `json:"category,omitempty"`
	ServiceCode      string                 `json:"service_code" binding:"required"`
	ServiceName      string                 `json:"service_name" binding:"required"`
	ResourceTypeCode string                 `json:"resource_type_code" binding:"required"`
	ResourceTypeName string                 `json:"resource_type_name" binding:"required"`
	APIVersion       string                 `json:"api_version,omitempty"`
	AttributeSchema  map[string]interface{} `json:"attribute_schema,omitempty"`
	IsSystem         *bool                  `json:"is_system,omitempty"`
	IsActive         *bool                  `json:"is_active,omitempty"`
}

type CloudServiceResponse struct {
	ID               int                    `json:"id"`
	ParentID         int                    `json:"parent_id"`
	Provider         string                 `json:"provider"`
	Category         string                 `json:"category,omitempty"`
	ServiceCode      string                 `json:"service_code"`
	ServiceName      string                 `json:"service_name"`
	ResourceTypeCode string                 `json:"resource_type_code"`
	ResourceTypeName string                 `json:"resource_type_name"`
	APIVersion       string                 `json:"api_version,omitempty"`
	AttributeSchema  map[string]interface{} `json:"attribute_schema,omitempty"`
	IsSystem         bool                   `json:"is_system"`
	IsActive         bool                   `json:"is_active"`
	TenantID         int                    `json:"tenant_id"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// CloudAccount DTOs
type CloudAccountRequest struct {
	Provider        string   `json:"provider" binding:"required"`
	AccountID       string   `json:"account_id" binding:"required"`
	AccountName     string   `json:"account_name" binding:"required"`
	CredentialRef   string   `json:"credential_ref,omitempty"`
	RegionWhitelist []string `json:"region_whitelist,omitempty"`
	IsActive        *bool    `json:"is_active,omitempty"`
}

type CloudAccountResponse struct {
	ID              int       `json:"id"`
	Provider        string    `json:"provider"`
	AccountID       string    `json:"account_id"`
	AccountName     string    `json:"account_name"`
	CredentialRef   string    `json:"credential_ref,omitempty"`
	RegionWhitelist []string  `json:"region_whitelist,omitempty"`
	IsActive        bool      `json:"is_active"`
	TenantID        int       `json:"tenant_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CloudResource DTOs
type CloudResourceResponse struct {
	ID             int                    `json:"id"`
	CloudAccountID int                    `json:"cloud_account_id"`
	ServiceID      int                    `json:"service_id"`
	ResourceID     string                 `json:"resource_id"`
	ResourceName   string                 `json:"resource_name,omitempty"`
	Region         string                 `json:"region,omitempty"`
	Zone           string                 `json:"zone,omitempty"`
	Status         string                 `json:"status,omitempty"`
	Tags           map[string]string      `json:"tags,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	FirstSeenAt    *time.Time             `json:"first_seen_at,omitempty"`
	LastSeenAt     *time.Time             `json:"last_seen_at,omitempty"`
	LifecycleState string                 `json:"lifecycle_state,omitempty"`
	TenantID       int                    `json:"tenant_id"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// RelationshipType DTOs
type RelationshipTypeResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Directional bool      `json:"directional"`
	ReverseName string    `json:"reverse_name,omitempty"`
	Description string    `json:"description,omitempty"`
	TenantID    int       `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Discovery DTOs
type DiscoverySourceRequest struct {
	Name        string `json:"name" binding:"required"`
	SourceType  string `json:"source_type" binding:"required"`
	Provider    string `json:"provider,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
	Description string `json:"description,omitempty"`
}

type DiscoverySourceResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	SourceType  string    `json:"source_type"`
	Provider    string    `json:"provider,omitempty"`
	IsActive    bool      `json:"is_active"`
	Description string    `json:"description,omitempty"`
	TenantID    int       `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DiscoveryJobRequest struct {
	SourceID string `json:"source_id" binding:"required"`
}

type DiscoveryJobResponse struct {
	ID         int                    `json:"id"`
	SourceID   string                 `json:"source_id"`
	Status     string                 `json:"status"`
	StartedAt  *time.Time             `json:"started_at,omitempty"`
	FinishedAt *time.Time             `json:"finished_at,omitempty"`
	Summary    map[string]interface{} `json:"summary,omitempty"`
	TenantID   int                    `json:"tenant_id"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

type DiscoveryResultResponse struct {
	ID           int                    `json:"id"`
	JobID        int                    `json:"job_id"`
	CIID         int                    `json:"ci_id,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type,omitempty"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	Diff         map[string]interface{} `json:"diff,omitempty"`
	Status       string                 `json:"status"`
	TenantID     int                    `json:"tenant_id"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// Reconciliation DTOs
type ReconciliationSummary struct {
	ResourceTotal        int `json:"resource_total"`
	BoundResourceCount   int `json:"bound_resource_count"`
	UnboundResourceCount int `json:"unbound_resource_count"`
	OrphanCICount        int `json:"orphan_ci_count"`
	UnlinkedCICount      int `json:"unlinked_ci_count"`
}

type ReconciliationResponse struct {
	Summary          ReconciliationSummary    `json:"summary"`
	UnboundResources []*CloudResourceResponse `json:"unbound_resources"`
	OrphanCIs        []*CIResponse            `json:"orphan_cis"`
	UnlinkedCIs      []*CIResponse            `json:"unlinked_cis"`
}
