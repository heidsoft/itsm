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
