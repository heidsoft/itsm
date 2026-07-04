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
	AssetTag           string                 `json:"assetTag"`
	SerialNumber       string                 `json:"serialNumber"`
	Model              string                 `json:"model"`
	Vendor             string                 `json:"vendor"`
	AssignedTo         string                 `json:"assignedTo"`
	OwnedBy            string                 `json:"ownedBy"`
	DiscoverySource    string                 `json:"discoverySource"`
	Source             string                 `json:"source"`
	CloudProvider      string                 `json:"cloudProvider"`
	CloudAccountID     string                 `json:"cloudAccountId"`
	CloudRegion        string                 `json:"cloudRegion"`
	CloudZone          string                 `json:"cloudZone"`
	CloudResourceID    string                 `json:"cloudResourceId"`
	CloudResourceType  string                 `json:"cloudResourceType"`
	CloudMetadata      map[string]interface{} `json:"cloudMetadata,omitempty"`
	CloudTags          map[string]interface{} `json:"cloudTags,omitempty"`
	CloudMetrics       map[string]interface{} `json:"cloudMetrics,omitempty"`
	CloudSyncTime      *time.Time             `json:"cloudSyncTime,omitempty"`
	CloudSyncStatus    string                 `json:"cloudSyncStatus"`
	CloudResourceRefID int                    `json:"cloudResourceRefId"`
	CITypeID           int                    `json:"ciTypeId"`
	TenantID           int                    `json:"tenantId"`
	Attributes         map[string]interface{} `json:"attributes,omitempty"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
}

// CloudService represents a cloud service/resource type catalog entry.
type CloudService struct {
	ID               int                    `json:"id"`
	ParentID         int                    `json:"parentId"`
	Provider         string                 `json:"provider"`
	Category         string                 `json:"category"`
	ServiceCode      string                 `json:"serviceCode"`
	ServiceName      string                 `json:"serviceName"`
	ResourceTypeCode string                 `json:"resourceTypeCode"`
	ResourceTypeName string                 `json:"resourceTypeName"`
	APIVersion       string                 `json:"apiVersion"`
	AttributeSchema  map[string]interface{} `json:"attributeSchema,omitempty"`
	IsSystem         bool                   `json:"isSystem"`
	IsActive         bool                   `json:"isActive"`
	TenantID         int                    `json:"tenantId"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
}

// CloudAccount represents a cloud account or on-prem connector.
type CloudAccount struct {
	ID              int       `json:"id"`
	Provider        string    `json:"provider"`
	AccountID       string    `json:"accountId"`
	AccountName     string    `json:"accountName"`
	CredentialRef   string    `json:"credentialRef,omitempty"`
	RegionWhitelist []string  `json:"regionWhitelist,omitempty"`
	IsActive        bool      `json:"isActive"`
	TenantID        int       `json:"tenantId"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// CloudResource represents a discovered cloud resource.
type CloudResource struct {
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
	TenantID       int                    `json:"tenantId"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

// RelationshipType represents CI relationship semantics.
type RelationshipType struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Directional bool      `json:"directional"`
	ReverseName string    `json:"reverseName,omitempty"`
	Description string    `json:"description,omitempty"`
	TenantID    int       `json:"tenantId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// DiscoverySource represents a CMDB discovery source.
type DiscoverySource struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	SourceType  string    `json:"sourceType"`
	Provider    string    `json:"provider,omitempty"`
	IsActive    bool      `json:"isActive"`
	Description string    `json:"description,omitempty"`
	TenantID    int       `json:"tenantId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// DiscoveryJob represents a discovery run.
type DiscoveryJob struct {
	ID         int                    `json:"id"`
	SourceID   string                 `json:"sourceId"`
	Status     string                 `json:"status"`
	StartedAt  *time.Time             `json:"startedAt,omitempty"`
	FinishedAt *time.Time             `json:"finishedAt,omitempty"`
	Summary    map[string]interface{} `json:"summary,omitempty"`
	TenantID   int                    `json:"tenantId"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

// DiscoveryResult represents a discovery diff item.
type DiscoveryResult struct {
	ID           int                    `json:"id"`
	JobID        int                    `json:"jobId"`
	CIID         int                    `json:"ciId,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resourceType,omitempty"`
	ResourceID   string                 `json:"resourceId,omitempty"`
	Diff         map[string]interface{} `json:"diff,omitempty"`
	Status       string                 `json:"status"`
	TenantID     int                    `json:"tenantId"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

// CIType represents a classification of CIs
type CIType struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Icon            string    `json:"icon"`
	Color           string    `json:"color"`
	AttributeSchema string    `json:"attributeSchema"`
	IsActive        bool      `json:"isActive"`
	TenantID        int       `json:"tenantId"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// CIRelationship represents a link between two CIs
type CIRelationship struct {
	ID                 int       `json:"id"`
	SourceCIID         int       `json:"sourceCiId"`
	TargetCIID         int       `json:"targetCiId"`
	RelationshipTypeID int       `json:"relationshipTypeId"`
	Description        string    `json:"description"`
	TenantID           int       `json:"tenantId"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// Stats represents CMDB statistics
type Stats struct {
	TotalCount       int            `json:"totalCount"`
	ActiveCount      int            `json:"activeCount"`
	InactiveCount    int            `json:"inactiveCount"`
	MaintenanceCount int            `json:"maintenanceCount"`
	TypeDistribution map[string]int `json:"typeDistribution"`
}

// ReconciliationSummary represents CMDB reconciliation summary.
type ReconciliationSummary struct {
	ResourceTotal        int `json:"resourceTotal"`
	BoundResourceCount   int `json:"boundResourceCount"`
	UnboundResourceCount int `json:"unboundResourceCount"`
	OrphanCICount        int `json:"orphanCiCount"`
	UnlinkedCICount      int `json:"unlinkedCiCount"`
}

// ReconciliationResult represents CMDB reconciliation output.
type ReconciliationResult struct {
	Summary          ReconciliationSummary `json:"summary"`
	UnboundResources []*CloudResource      `json:"unboundResources"`
	OrphanCIs        []*ConfigurationItem  `json:"orphanCis"`
	UnlinkedCIs      []*ConfigurationItem  `json:"unlinkedCis"`
}
