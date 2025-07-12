package dto

import "time"

// CreateTenantRequest 创建租户请求
type CreateTenantRequest struct {
	Name      string                 `json:"name" binding:"required,max=100" comment:"租户名称"`
	Code      string                 `json:"code" binding:"required,max=50,alphanum" comment:"租户代码"`
	Domain    *string                `json:"domain,omitempty" binding:"omitempty,max=100" comment:"自定义域名"`
	Type      string                 `json:"type" binding:"required,oneof=trial standard professional enterprise" comment:"租户类型"`
	Settings  map[string]interface{} `json:"settings,omitempty" comment:"租户配置"`
	Quota     map[string]interface{} `json:"quota,omitempty" comment:"资源配额"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty" comment:"过期时间"`
}

// UpdateTenantRequest 更新租户请求
type UpdateTenantRequest struct {
	Name      *string                `json:"name,omitempty" binding:"omitempty,max=100" comment:"租户名称"`
	Domain    *string                `json:"domain,omitempty" binding:"omitempty,max=100" comment:"自定义域名"`
	Type      *string                `json:"type,omitempty" binding:"omitempty,oneof=trial standard professional enterprise" comment:"租户类型"`
	Status    *string                `json:"status,omitempty" binding:"omitempty,oneof=active suspended expired deleted" comment:"租户状态"`
	Settings  map[string]interface{} `json:"settings,omitempty" comment:"租户配置"`
	Quota     map[string]interface{} `json:"quota,omitempty" comment:"资源配额"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty" comment:"过期时间"`
}

// ListTenantsRequest 租户列表请求
type ListTenantsRequest struct {
	Page     int    `form:"page,default=1" binding:"min=1" comment:"页码"`
	PageSize int    `form:"page_size,default=10" binding:"min=1,max=100" comment:"每页数量"`
	Status   string `form:"status" binding:"omitempty,oneof=active suspended expired deleted" comment:"状态过滤"`
	Type     string `form:"type" binding:"omitempty,oneof=trial standard professional enterprise" comment:"类型过滤"`
	Search   string `form:"search" comment:"搜索关键词"`
}

// TenantResponse 租户响应
type TenantResponse struct {
	ID        int                    `json:"id"`
	Name      string                 `json:"name"`
	Code      string                 `json:"code"`
	Domain    *string                `json:"domain,omitempty"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
	Quota     map[string]interface{} `json:"quota,omitempty"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// TenantListResponse 租户列表响应
type TenantListResponse struct {
	Tenants []TenantResponse `json:"tenants"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	Size    int              `json:"size"`
}
