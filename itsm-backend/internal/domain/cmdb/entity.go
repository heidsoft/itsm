package cmdb

import (
	"time"
)

// ConfigurationItem representing a CI in the CMDB
type ConfigurationItem struct {
	ID                 int                    `json:"id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	Type               string                 `json:"type"` // Legacy type field from schema
	Status             string                 `json:"status"`
	Environment        string                 `json:"environment"`
	Criticality        string                 `json:"criticality"`
	Location           string                 `json:"location"`
	AssetTag           string                 `json:"asset_tag"`
	SerialNumber       string                 `json:"serial_number"`
	Model              string                 `json:"model"`
	Vendor             string                 `json:"vendor"`
	AssignedTo         string                 `json:"assigned_to"`
	OwnedBy            string                 `json:"owned_by"`
	DiscoverySource    string                 `json:"discovery_source"`
	Source             string                 `json:"source"`
	CloudProvider      string                 `json:"cloud_provider"`
	CloudAccountID     string                 `json:"cloud_account_id"`
	CloudRegion        string                 `json:"cloud_region"`
	CloudZone          string                 `json:"cloud_zone"`
	CloudResourceID    string                 `json:"cloud_resource_id"`
	CloudResourceType  string                 `json:"cloud_resource_type"`
	CloudMetadata      map[string]interface{} `json:"cloud_metadata,omitempty"`
	CloudTags          map[string]interface{} `json:"cloud_tags,omitempty"`
	CloudMetrics       map[string]interface{} `json:"cloud_metrics,omitempty"`
	CloudSyncTime      *time.Time             `json:"cloud_sync_time,omitempty"`
	CloudSyncStatus    string                 `json:"cloud_sync_status"`
	CloudResourceRefID int                    `json:"cloud_resource_ref_id"`
	CITypeID           int                    `json:"ci_type_id"`
	TenantID           int                    `json:"tenant_id"`
	Attributes         map[string]interface{} `json:"attributes,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// CloudService represents a cloud service/resource type catalog entry.
type CloudService struct {
	ID               int                    `json:"id"`
	ParentID         int                    `json:"parent_id"`
	Provider         string                 `json:"provider"`
	Category         string                 `json:"category"`
	ServiceCode      string                 `json:"service_code"`
	ServiceName      string                 `json:"service_name"`
	ResourceTypeCode string                 `json:"resource_type_code"`
	ResourceTypeName string                 `json:"resource_type_name"`
	APIVersion       string                 `json:"api_version"`
	AttributeSchema  map[string]interface{} `json:"attribute_schema,omitempty"`
	IsSystem         bool                   `json:"is_system"`
	IsActive         bool                   `json:"is_active"`
	TenantID         int                    `json:"tenant_id"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// CloudAccount represents a cloud account or on-prem connector.
type CloudAccount struct {
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

// CloudResource represents a discovered cloud resource.
type CloudResource struct {
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

// RelationshipType represents CI relationship semantics.
type RelationshipType struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Directional bool      `json:"directional"`
	ReverseName string    `json:"reverse_name,omitempty"`
	Description string    `json:"description,omitempty"`
	TenantID    int       `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DiscoverySource represents a CMDB discovery source.
type DiscoverySource struct {
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

// DiscoveryJob represents a discovery run.
type DiscoveryJob struct {
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

// DiscoveryResult represents a discovery diff item.
type DiscoveryResult struct {
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

// CIType represents a classification of CIs
type CIType struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Icon            string    `json:"icon"`
	Color           string    `json:"color"`
	AttributeSchema string    `json:"attribute_schema"`
	IsActive        bool      `json:"is_active"`
	TenantID        int       `json:"tenant_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CIRelationship represents a link between two CIs
type CIRelationship struct {
	ID                 int       `json:"id"`
	SourceCIID         int       `json:"source_ci_id"`
	TargetCIID         int       `json:"target_ci_id"`
	RelationshipTypeID int       `json:"relationship_type_id"`
	Description        string    `json:"description"`
	TenantID           int       `json:"tenant_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Stats represents CMDB statistics
type Stats struct {
	TotalCount       int            `json:"total_count"`
	ActiveCount      int            `json:"active_count"`
	InactiveCount    int            `json:"inactive_count"`
	MaintenanceCount int            `json:"maintenance_count"`
	TypeDistribution map[string]int `json:"type_distribution"`
}

// ReconciliationSummary represents CMDB reconciliation summary.
type ReconciliationSummary struct {
	ResourceTotal        int `json:"resource_total"`
	BoundResourceCount   int `json:"bound_resource_count"`
	UnboundResourceCount int `json:"unbound_resource_count"`
	OrphanCICount        int `json:"orphan_ci_count"`
	UnlinkedCICount      int `json:"unlinked_ci_count"`
}

// ReconciliationResult represents CMDB reconciliation output.
type ReconciliationResult struct {
	Summary          ReconciliationSummary `json:"summary"`
	UnboundResources []*CloudResource      `json:"unbound_resources"`
	OrphanCIs        []*ConfigurationItem  `json:"orphan_cis"`
	UnlinkedCIs      []*ConfigurationItem  `json:"unlinked_cis"`
}
