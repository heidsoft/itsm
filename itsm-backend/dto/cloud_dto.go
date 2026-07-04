package dto

// ===================================
// CloudAccount Request DTOs
// ===================================

// CreateCloudAccountRequest 创建云账号请求
type CreateCloudAccountRequest struct {
	Provider         string   `json:"provider" binding:"required,oneof=aliyun tencent huawei aws azure onprem" comment:"云厂商标识"`
	AccountID        string   `json:"accountId" binding:"required,max=100" comment:"云账号ID"`
	AccountName      string   `json:"accountName" binding:"required,max=200" comment:"云账号名称"`
	CredentialRef    string   `json:"credentialRef,omitempty" binding:"omitempty,max=200" comment:"凭据引用"`
	RegionWhitelist []string `json:"regionWhitelist,omitempty" comment:"可用Region白名单"`
	IsActive        *bool   `json:"isActive,omitempty" comment:"是否启用"`
}

// UpdateCloudAccountRequest 更新云账号请求
type UpdateCloudAccountRequest struct {
	AccountName      *string  `json:"accountName,omitempty" binding:"omitempty,max=200" comment:"云账号名称"`
	CredentialRef    *string  `json:"credentialRef,omitempty" binding:"omitempty,max=200" comment:"凭据引用"`
	RegionWhitelist []string `json:"regionWhitelist,omitempty" comment:"可用Region白名单"`
	IsActive        *bool    `json:"isActive,omitempty" comment:"是否启用"`
}

// ListCloudAccountsRequest 云账号列表请求
type ListCloudAccountsRequest struct {
	Page       int    `form:"page,default=1" binding:"min=1" comment:"页码"`
	PageSize   int    `form:"pageSize,default=10" binding:"min=1,max=100" comment:"每页数量"`
	Provider   string `form:"provider" binding:"omitempty,oneof=aliyun tencent huawei aws azure onprem" comment:"云厂商过滤"`
	IsActive   *bool  `form:"isActive" comment:"是否启用过滤"`
	Search     string `form:"search" comment:"搜索关键词（账号ID/名称）"`
}

// CloudAccountListResponse 云账号列表响应
type CloudAccountListResponse struct {
	CloudAccounts []CloudAccountResponse `json:"cloudAccounts"`
	Total         int                    `json:"total"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"pageSize"`
}

// ===================================
// CloudService Request DTOs
// ===================================

// CreateCloudServiceRequest 创建云服务请求
type CreateCloudServiceRequest struct {
	Provider            string                 `json:"provider" binding:"required,oneof=aliyun tencent huawei aws azure onprem" comment:"云厂商标识"`
	Category           string                 `json:"category,omitempty" binding:"omitempty,max=100" comment:"服务分类"`
	ServiceCode        string                 `json:"serviceCode" binding:"required,max=50" comment:"云服务代码，例如 ecs/rds/oss"`
	ServiceName        string                 `json:"serviceName" binding:"required,max=200" comment:"云服务名称"`
	ResourceTypeCode   string                 `json:"resourceTypeCode" binding:"required,max=50" comment:"资源类型代码，例如 instance/volume/vpc"`
	ResourceTypeName   string                 `json:"resourceTypeName" binding:"required,max=200" comment:"资源类型名称"`
	APIVersion         string                 `json:"apiVersion,omitempty" binding:"omitempty,max=50" comment:"API版本"`
	AttributeSchema    map[string]interface{} `json:"attributeSchema,omitempty" comment:"动态属性Schema"`
	IsSystem          *bool                  `json:"isSystem,omitempty" comment:"是否系统预置"`
	IsActive          *bool                  `json:"isActive,omitempty" comment:"是否启用"`
	ParentID          *int                   `json:"parentId,omitempty" comment:"父级服务ID"`
}

// UpdateCloudServiceRequest 更新云服务请求
type UpdateCloudServiceRequest struct {
	Category         *string                 `json:"category,omitempty" binding:"omitempty,max=100" comment:"服务分类"`
	ServiceName      *string                 `json:"serviceName,omitempty" binding:"omitempty,max=200" comment:"云服务名称"`
	ResourceTypeName *string                 `json:"resourceTypeName,omitempty" binding:"omitempty,max=200" comment:"资源类型名称"`
	APIVersion       *string                 `json:"apiVersion,omitempty" binding:"omitempty,max=50" comment:"API版本"`
	AttributeSchema  map[string]interface{} `json:"attributeSchema,omitempty" comment:"动态属性Schema"`
	IsActive        *bool                   `json:"isActive,omitempty" comment:"是否启用"`
}

// ListCloudServicesRequest 云服务列表请求
type ListCloudServicesRequest struct {
	Page       int    `form:"page,default=1" binding:"min=1" comment:"页码"`
	PageSize   int    `form:"pageSize,default=10" binding:"min=1,max=100" comment:"每页数量"`
	Provider   string `form:"provider" binding:"omitempty,oneof=aliyun tencent huawei aws azure onprem" comment:"云厂商过滤"`
	Category   string `form:"category" comment:"服务分类过滤"`
	IsSystem   *bool  `form:"isSystem" comment:"是否系统预置过滤"`
	IsActive   *bool  `form:"isActive" comment:"是否启用过滤"`
	Search     string `form:"search" comment:"搜索关键词"`
	ParentID   *int   `form:"parentId" comment:"父级服务ID过滤"`
}

// CloudServiceListResponse 云服务列表响应
type CloudServiceListResponse struct {
	CloudServices []CloudServiceResponse `json:"cloudServices"`
	Total         int                    `json:"total"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"pageSize"`
}

// ===================================
// CloudResource Request DTOs
// ===================================

// CreateCloudResourceRequest 创建云资源请求
type CreateCloudResourceRequest struct {
	CloudAccountID int                     `json:"cloudAccountId" binding:"required" comment:"云账号ID"`
	ServiceID      int                     `json:"serviceId" binding:"required" comment:"云服务类型ID"`
	ResourceID     string                  `json:"resourceId" binding:"required,max=200" comment:"云资源唯一ID"`
	ResourceName   string                  `json:"resourceName,omitempty" binding:"omitempty,max=200" comment:"云资源名称"`
	Region         string                  `json:"region,omitempty" binding:"omitempty,max=100" comment:"Region"`
	Zone           string                  `json:"zone,omitempty" binding:"omitempty,max=100" comment:"Zone"`
	Status         string                  `json:"status,omitempty" binding:"omitempty,max=50" comment:"资源状态"`
	Tags           map[string]string       `json:"tags,omitempty" comment:"资源标签"`
	Metadata       map[string]interface{}  `json:"metadata,omitempty" comment:"资源元数据"`
	LifecycleState string                  `json:"lifecycleState,omitempty" binding:"omitempty,max=50" comment:"生命周期状态"`
}

// UpdateCloudResourceRequest 更新云资源请求
type UpdateCloudResourceRequest struct {
	ResourceName   *string                `json:"resourceName,omitempty" binding:"omitempty,max=200" comment:"云资源名称"`
	Region         *string                `json:"region,omitempty" binding:"omitempty,max=100" comment:"Region"`
	Zone           *string                `json:"zone,omitempty" binding:"omitempty,max=100" comment:"Zone"`
	Status         *string                `json:"status,omitempty" binding:"omitempty,max=50" comment:"资源状态"`
	Tags           map[string]string      `json:"tags,omitempty" comment:"资源标签"`
	Metadata       map[string]interface{} `json:"metadata,omitempty" comment:"资源元数据"`
	LifecycleState *string                `json:"lifecycleState,omitempty" binding:"omitempty,max=50" comment:"生命周期状态"`
}

// ListCloudResourcesRequest 云资源列表请求
type ListCloudResourcesRequest struct {
	Page           int    `form:"page,default=1" binding:"min=1" comment:"页码"`
	PageSize       int    `form:"pageSize,default=10" binding:"min=1,max=100" comment:"每页数量"`
	CloudAccountID int    `form:"cloudAccountId" comment:"云账号ID过滤"`
	ServiceID      int    `form:"serviceId" comment:"云服务类型ID过滤"`
	Region         string `form:"region" comment:"Region过滤"`
	Status         string `form:"status" comment:"资源状态过滤"`
	Search         string `form:"search" comment:"搜索关键词（资源ID/名称）"`
}

// CloudResourceListResponse 云资源列表响应
type CloudResourceListResponse struct {
	CloudResources []CloudResourceResponse `json:"cloudResources"`
	Total          int                      `json:"total"`
	Page           int                      `json:"page"`
	PageSize       int                      `json:"pageSize"`
}
