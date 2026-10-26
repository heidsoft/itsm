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
	TenantID        int                    `json:"tenantId,omitempty"`
	// 生命周期管理
	LifecycleStatus string                      `json:"lifecycleStatus,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time                  `json:"effectiveAt,omitempty"`
	ExpireAt        *time.Time                  `json:"expireAt,omitempty"`
	CreatedAt       time.Time                   `json:"createdAt"`
	UpdatedAt       time.Time                   `json:"updatedAt"`
	RelatedItems    []ConfigurationItemResponse `json:"relatedItems,omitempty"`
	ParentItems     []ConfigurationItemResponse `json:"parentItems,omitempty"`
}

// ConfigurationItemStatsResponse 配置项统计响应
type ConfigurationItemStatsResponse struct {
	Total                   int            `json:"total"`
	ActiveCount             int            `json:"activeCount"`
	InactiveCount           int            `json:"inactiveCount"`
	MaintenanceCount        int            `json:"maintenanceCount"`
	TypeDistribution        map[string]int `json:"typeDistribution"`
	EnvironmentDistribution map[string]int `json:"environmentDistribution"`
}

// CreateCIRequest 创建配置项请求
type CreateCIRequest struct {
	Name               string                 `json:"name" binding:"required,max=255"`
	CITypeID           int                    `json:"ciTypeId"`
	Description        string                 `json:"description"`
	Status             string                 `json:"status" binding:"required"`
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
	Source             string                 `json:"source,omitempty"`
	LastDiscovered     *time.Time             `json:"lastDiscovered,omitempty"`
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
	TenantID           int                    `json:"tenantId,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycleStatus,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effectiveAt,omitempty"`
	ExpireAt        *time.Time `json:"expireAt,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type string `json:"type,omitempty"`
}

// CIResponse 配置项响应
type CIResponse struct {
	ID              int                    `json:"id"`
	CINumber        string                 `json:"ciNumber,omitempty"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	CITypeID        int                    `json:"ciTypeId"`
	Description     string                 `json:"description"`
	Status          string                 `json:"status"`
	Environment     string                 `json:"environment,omitempty"`
	Criticality     string                 `json:"criticality,omitempty"`
	AssetTag        string                 `json:"assetTag,omitempty"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	SerialNumber    string                 `json:"serialNumber,omitempty"`
	Model           string                 `json:"model,omitempty"`
	Vendor          string                 `json:"vendor,omitempty"`
	Location        string                 `json:"location,omitempty"`
	AssignedTo      string                 `json:"assignedTo,omitempty"`
	OwnedBy         string                 `json:"ownedBy,omitempty"`
	DiscoverySource string                 `json:"discoverySource,omitempty"`
	// LastDiscovered 用指针表达「从未被发现」：time.Time 的 omitempty 不生效，
	// 零值会被序列化成 0001-01-01T00:00:00Z，调用方无法区分「没发现过」和
	// 「真的在这一天发现过」。
	LastDiscovered     *time.Time             `json:"lastDiscovered,omitempty"`
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
	Version            int                    `json:"version"`
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
	TenantID int `json:"tenantId,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycleStatus,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effectiveAt,omitempty"`
	ExpireAt        *time.Time `json:"expireAt,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type     string `json:"type,omitempty"`
	CITypeID int    `json:"ciTypeId,omitempty"`
	Status   string `json:"status,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

// UpdateCIRequest 更新配置项请求
type UpdateCIRequest struct {
	CITypeID           int                    `json:"ciTypeId,omitempty"`
	Name               string                 `json:"name,omitempty"`
	Description        string                 `json:"description,omitempty"`
	Status             string                 `json:"status,omitempty"`
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
	Source             string                 `json:"source,omitempty"`
	LastDiscovered     *time.Time             `json:"lastDiscovered,omitempty"`
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
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycleStatus,omitempty"`
	EffectiveAt     *time.Time `json:"effectiveAt,omitempty"`
	ExpireAt        *time.Time `json:"expireAt,omitempty"`
	// 版本号，用于乐观锁
	Version int `json:"version,omitempty"`
}

// UpdateCILifecycleRequest CI 生命周期状态变更请求（PUT /cmdb/cis/:id/lifecycle）。
// version 为乐观锁必填；retirementReason 仅在目标为 scrapped 时必填，
// 取值限受控退役理由词表（common/ci_lifecycle.go）。
type UpdateCILifecycleRequest struct {
	Status           string `json:"status" binding:"required,oneof=draft online maintenance offline scrapped"`
	Version          int    `json:"version" binding:"required"`
	Remark           string `json:"remark,omitempty"`
	RetirementReason string `json:"retirementReason,omitempty" binding:"omitempty,oneof=manual decommissioned replaced discovery_missing end_of_life"`
}

// CloudService DTOs
//
// Provider 的取值集合在 ent/schema/cloud_service.go 的字段注释、前端云厂商选择器和
// service/cloud.NormalizeProvider 三处都只认这六个规范值；写入侧过去没有任何校验，
// 一个拼错的厂商名会一路落到数据库并在发现阶段表现为「没有适配器」。
// 别名（alibaba/alicloud/huaweicloud/qcloud/amazon/private 等）只在适配器边界由
// NormalizeProvider 归一化，不作为可写入的取值。
type CloudServiceRequest struct {
	ParentID         int                    `json:"parentId,omitempty"`
	Provider         string                 `json:"provider" binding:"required,oneof=aliyun tencent huawei aws azure volcengine k8s onprem"`
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
	TenantID         int                    `json:"tenantId,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycleStatus,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effectiveAt,omitempty"`
	ExpireAt        *time.Time `json:"expireAt,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CloudAccount DTOs
type CloudAccountRequest struct {
	Provider        string   `json:"provider" binding:"required,oneof=aliyun tencent huawei aws azure volcengine k8s onprem"`
	AccountID       string   `json:"accountId" binding:"required"`
	AccountName     string   `json:"accountName" binding:"required"`
	CredentialRef   string   `json:"credentialRef,omitempty"`
	RegionWhitelist []string `json:"regionWhitelist,omitempty"`
	IsActive        *bool    `json:"isActive,omitempty"`
}

type CMDBCloudAccountUpdateRequest struct {
	AccountName     *string   `json:"accountName,omitempty" binding:"omitempty,max=200"`
	CredentialRef   *string   `json:"credentialRef,omitempty" binding:"omitempty,max=200"`
	RegionWhitelist *[]string `json:"regionWhitelist,omitempty"`
	IsActive        *bool     `json:"isActive,omitempty"`
}

type CloudAccountResponse struct {
	ID              int      `json:"id"`
	Provider        string   `json:"provider"`
	AccountID       string   `json:"accountId"`
	AccountName     string   `json:"accountName"`
	HasCredential   bool     `json:"hasCredential"`
	RegionWhitelist []string `json:"regionWhitelist,omitempty"`
	IsActive        bool     `json:"isActive"`
	TenantID        int      `json:"tenantId,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycleStatus,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effectiveAt,omitempty"`
	ExpireAt        *time.Time `json:"expireAt,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CMDBCapabilityResponse struct {
	Key                 string   `json:"key"`
	State               string   `json:"state"`
	BuildCapability     bool     `json:"buildCapability"`
	DeploymentReadiness bool     `json:"deploymentReadiness"`
	TenantReadiness     bool     `json:"tenantReadiness"`
	ActorPermission     bool     `json:"actorPermission"`
	MissingRequirements []string `json:"missingRequirements"`
}

type CMDBCapabilitiesResponse struct {
	Items []CMDBCapabilityResponse `json:"items"`
}

// CloudResource DTOs
type CloudResourceResponse struct {
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
	TenantID           int                    `json:"tenantId,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycleStatus,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effectiveAt,omitempty"`
	ExpireAt        *time.Time `json:"expireAt,omitempty"`
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
	TenantID    int    `json:"tenantId,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycleStatus,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effectiveAt,omitempty"`
	ExpireAt        *time.Time `json:"expireAt,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Discovery DTOs
type DiscoverySourceRequest struct {
	Name            string   `json:"name" binding:"required"`
	SourceType      string   `json:"sourceType" binding:"required"`
	Provider        string   `json:"provider,omitempty"`
	IsActive        *bool    `json:"isActive,omitempty"`
	Description     string   `json:"description,omitempty"`
	CloudAccountID  int      `json:"cloudAccountId,omitempty"`
	ServiceCodes    []string `json:"serviceCodes,omitempty"`
	Regions         []string `json:"regions,omitempty"`
	Schedule        string   `json:"schedule,omitempty"`
	ReconcilePolicy string   `json:"reconcilePolicy,omitempty" binding:"omitempty,oneof=manual discovered_wins cmdb_wins"`
	StaleThreshold  int      `json:"staleThreshold,omitempty" binding:"omitempty,min=1"`
}

type DiscoverySourceResponse struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	SourceType     string   `json:"sourceType"`
	Provider       string   `json:"provider,omitempty"`
	IsActive       bool     `json:"isActive"`
	Description    string   `json:"description,omitempty"`
	CloudAccountID int      `json:"cloudAccountId,omitempty"`
	ServiceCodes   []string `json:"serviceCodes,omitempty"`
	Regions        []string `json:"regions,omitempty"`
	Schedule       string   `json:"schedule,omitempty"`
	// NextRunAt 是调度器下一次会为该来源排任务的时刻（按租户时区计算）。
	// 未配置 schedule 时省略。
	NextRunAt *time.Time `json:"nextRunAt,omitempty"`
	// ScheduleError 只在调度表达式无法解析时出现：配了表达式却永远不会触发，
	// 必须在保存那一刻就可见，而不是等到「到了点没跑」再排查。
	ScheduleError   string     `json:"scheduleError,omitempty"`
	ReconcilePolicy string     `json:"reconcilePolicy"`
	StaleThreshold  int        `json:"staleThreshold"`
	LastSuccessAt   *time.Time `json:"lastSuccessAt,omitempty"`
	TenantID        int        `json:"tenantId,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycleStatus,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effectiveAt,omitempty"`
	ExpireAt        *time.Time `json:"expireAt,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DiscoveryJobRequest struct {
	SourceID string `json:"sourceId" binding:"required"`
	// IdempotencyKey 可选：携带时同键重复创建返回 409（附既有任务 ID）；
	// 不携带则每次生成唯一键（手动重跑不受指纹去重限制）。
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
}

// DiscoveryJobDetailResponse 发现任务详情（操作者读面）。
// 租约字段（lease_owner/lease_expires_at/fencing_token/heartbeat_at）是执行器
// 内部协调状态，不对外暴露。
type DiscoveryJobDetailResponse struct {
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
	ParentJobID        int                    `json:"parentJobId,omitempty"`
	MaxAttempts        int                    `json:"maxAttempts"`
	Attempt            int                    `json:"attempt"`
	Progress           int                    `json:"progress"`
	ErrorCode          string                 `json:"errorCode,omitempty"`
	ErrorMessage       string                 `json:"errorMessage,omitempty"`
	CancelRequestedAt  *time.Time             `json:"cancelRequestedAt,omitempty"`
	QueuedAt           *time.Time             `json:"queuedAt,omitempty"`
	StartedAt          *time.Time             `json:"startedAt,omitempty"`
	FinishedAt         *time.Time             `json:"finishedAt,omitempty"`
	Summary            map[string]interface{} `json:"summary,omitempty"`
	TenantID           int                    `json:"tenantId"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
}

type DiscoveryJobResponse struct {
	ID         int                    `json:"id"`
	SourceID   string                 `json:"sourceId"`
	Status     string                 `json:"status"`
	StartedAt  *time.Time             `json:"startedAt,omitempty"`
	FinishedAt *time.Time             `json:"finishedAt,omitempty"`
	Summary    map[string]interface{} `json:"summary,omitempty"`
	TenantID   int                    `json:"tenantId,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycleStatus,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effectiveAt,omitempty"`
	ExpireAt        *time.Time `json:"expireAt,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DiscoveryResultResponse struct {
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
	TenantID         int                    `json:"tenantId,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycleStatus,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effectiveAt,omitempty"`
	ExpireAt        *time.Time `json:"expireAt,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// DiscoveryResultRejectRequest 驳回一条待确认差异。理由可选——强制填写会让
// 运维在噪声差异上随手填「-」，反而污染记录。
type DiscoveryResultRejectRequest struct {
	Reason string `json:"reason,omitempty"`
}

// PendingDiscoveryAttributeResponse 是待补全资源上一条必填属性的现状。
// satisfied=false 即「需要人工补录」；currentValue 来自云标签，用于前端预填。
type PendingDiscoveryAttributeResponse struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"displayName"`
	DataType     string   `json:"dataType"`
	EnumValues   []string `json:"enumValues"`
	DefaultValue string   `json:"defaultValue"`
	Placeholder  string   `json:"placeholder"`
	HelpText     string   `json:"helpText"`
	Satisfied    bool     `json:"satisfied"`
	CurrentValue string   `json:"currentValue"`
}

// PendingDiscoveryResourceResponse 一条「发现 deferred、等待人工补全建账」的云资源。
type PendingDiscoveryResourceResponse struct {
	ID                    int                                 `json:"id"`
	ResourceID            string                              `json:"resourceId"`
	ResourceName          string                              `json:"resourceName"`
	Provider              string                              `json:"provider"`
	Region                string                              `json:"region"`
	Zone                  string                              `json:"zone"`
	Status                string                              `json:"status"`
	ServiceCode           string                              `json:"serviceCode"`
	ResourceType          string                              `json:"resourceType"`
	SourceID              string                              `json:"sourceId"`
	Tags                  map[string]string                   `json:"tags,omitempty"`
	LastSeenAt            *time.Time                          `json:"lastSeenAt,omitempty"`
	TargetCiType          string                              `json:"targetCiType"`
	TargetCiTypeID        int                                 `json:"targetCiTypeId"`
	RequiredAttributes    []PendingDiscoveryAttributeResponse `json:"requiredAttributes"`
	MissingAttributeNames []string                            `json:"missingAttributeNames"`
}

// RegisterPendingDiscoveryResourceRequest 人工补全建账的提交内容。
// 可选字段留空表示沿用云标签推断结果；attributes 与标签同名键按提交值覆盖。
type RegisterPendingDiscoveryResourceRequest struct {
	Name        string                 `json:"name,omitempty"`
	Environment string                 `json:"environment,omitempty" binding:"omitempty,oneof=production staging development"`
	AssignedTo  string                 `json:"assignedTo,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
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
	TenantID      int                    `json:"tenantId,omitempty"`
	// 生命周期管理
	LifecycleStatus string     `json:"lifecycleStatus,omitempty" binding:"omitempty,oneof=draft online maintenance offline scrapped"`
	EffectiveAt     *time.Time `json:"effectiveAt,omitempty"`
	ExpireAt        *time.Time `json:"expireAt,omitempty"`
	// Type is optional; will be set from CIType if not provided
	Type      string    `json:"type,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// CIHistoryListResponse CI历史列表响应
type CIHistoryListResponse struct {
	Items      []*CIHistoryResponse `json:"items"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"pageSize"`
	TotalPages int                  `json:"totalPages"`
}

// RevertCIVersionRequest 回滚CI版本请求
type RevertCIVersionRequest struct {
	Version int    `json:"version" binding:"required,min=1"`
	Remark  string `json:"remark,omitempty"`
}
