package cmdb

import (
	"time"
)

// ConfigurationItem representing a CI in the CMDB
type ConfigurationItem struct {
	ID                 int                    `json:"id"`
	CINumber           string                 `json:"ciNumber,omitempty"`
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
	ID                 int                    `json:"id"`
	CloudAccountID     int                    `json:"cloudAccountId"`
	ServiceID          int                    `json:"serviceId"`
	ResourceID         string                 `json:"resourceId"`
	IdentityVersion    int                    `json:"identityVersion"`
	Provider           string                 `json:"provider,omitempty"`
	Partition          string                 `json:"partition,omitempty"`
	CanonicalAccountID string                 `json:"canonicalAccountId,omitempty"`
	ResourceScope      string                 `json:"resourceScope,omitempty"`
	ServiceCode        string                 `json:"serviceCode,omitempty"`
	ResourceType       string                 `json:"resourceType,omitempty"`
	IdentityHash       string                 `json:"identityHash,omitempty"`
	SourceID           string                 `json:"sourceId,omitempty"`
	SourceFingerprint  string                 `json:"sourceFingerprint,omitempty"`
	MissingCount       int                    `json:"missingCount"`
	ResourceName       string                 `json:"resourceName,omitempty"`
	Region             string                 `json:"region,omitempty"`
	Zone               string                 `json:"zone,omitempty"`
	Status             string                 `json:"status,omitempty"`
	Tags               map[string]string      `json:"tags,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	FirstSeenAt        *time.Time             `json:"firstSeenAt,omitempty"`
	LastSeenAt         *time.Time             `json:"lastSeenAt,omitempty"`
	LifecycleState     string                 `json:"lifecycleState,omitempty"`
	TenantID           int                    `json:"tenantId"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
}

// DiscoverySource represents a CMDB discovery source.
type DiscoverySource struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	SourceType      string     `json:"sourceType"`
	Provider        string     `json:"provider,omitempty"`
	CloudAccountID  int        `json:"cloudAccountId,omitempty"`
	ServiceCodes    []string   `json:"serviceCodes,omitempty"`
	Regions         []string   `json:"regions,omitempty"`
	Schedule        string     `json:"schedule,omitempty"`
	ReconcilePolicy string     `json:"reconcilePolicy"`
	StaleThreshold  int        `json:"staleThreshold"`
	LastSuccessAt   *time.Time `json:"lastSuccessAt,omitempty"`
	IsActive        bool       `json:"isActive"`
	Description     string     `json:"description,omitempty"`
	TenantID        int        `json:"tenantId"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// DiscoveryJob represents a discovery run.
type DiscoveryJob struct {
	ID                 int                    `json:"id"`
	SourceID           string                 `json:"sourceId"`
	Status             string                 `json:"status"`
	Operation          string                 `json:"operation"`
	IdempotencyKey     string                 `json:"idempotencyKey,omitempty"`
	RequestFingerprint string                 `json:"requestFingerprint,omitempty"`
	SourceSnapshot     map[string]interface{} `json:"sourceSnapshot,omitempty"`
	ScopeSnapshot      map[string]interface{} `json:"scopeSnapshot,omitempty"`
	CompletedScopes    []string               `json:"completedScopes,omitempty"`
	FailedScopes       []string               `json:"failedScopes,omitempty"`
	SnapshotGeneration string                 `json:"snapshotGeneration,omitempty"`
	RequestedBy        int                    `json:"requestedBy,omitempty"`
	QueuedAt           *time.Time             `json:"queuedAt,omitempty"`
	HeartbeatAt        *time.Time             `json:"heartbeatAt,omitempty"`
	LeaseOwner         string                 `json:"leaseOwner,omitempty"`
	LeaseExpiresAt     *time.Time             `json:"leaseExpiresAt,omitempty"`
	FencingToken       int64                  `json:"fencingToken"`
	Attempt            int                    `json:"attempt"`
	ParentJobID        int                    `json:"parentJobId,omitempty"`
	MaxAttempts        int                    `json:"maxAttempts"`
	Progress           int                    `json:"progress"`
	ErrorCode          string                 `json:"errorCode,omitempty"`
	ErrorMessage       string                 `json:"errorMessage,omitempty"`
	CancelRequestedAt  *time.Time             `json:"cancelRequestedAt,omitempty"`
	StartedAt          *time.Time             `json:"startedAt,omitempty"`
	FinishedAt         *time.Time             `json:"finishedAt,omitempty"`
	Summary            map[string]interface{} `json:"summary,omitempty"`
	TenantID           int                    `json:"tenantId"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
}

// DiscoveryResult represents a discovery diff item.
type DiscoveryResult struct {
	ID               int                    `json:"id"`
	JobID            int                    `json:"jobId"`
	CIID             int                    `json:"ciId,omitempty"`
	Action           string                 `json:"action"`
	ResourceType     string                 `json:"resourceType,omitempty"`
	ResourceID       string                 `json:"resourceId,omitempty"`
	ResourceIdentity string                 `json:"resourceIdentity,omitempty"`
	IdentityVersion  int                    `json:"identityVersion"`
	ResourceSnapshot map[string]interface{} `json:"resourceSnapshot,omitempty"`
	BeforeHash       string                 `json:"beforeHash,omitempty"`
	AfterHash        string                 `json:"afterHash,omitempty"`
	Diff             map[string]interface{} `json:"diff,omitempty"`
	Status           string                 `json:"status"`
	ErrorCode        string                 `json:"errorCode,omitempty"`
	ErrorMessage     string                 `json:"errorMessage,omitempty"`
	TenantID         int                    `json:"tenantId"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
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
