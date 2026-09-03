package dto

import "time"

// SystemConfigRequest 请求
type SystemConfigRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"valueType"`
	Category    string `json:"category"`
	Description string `json:"description"`
	TenantID    int    `json:"tenantId"`
}

// SystemConfigResponse 响应
type SystemConfigResponse struct {
	ID          int       `json:"id"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	ValueType   string    `json:"valueType"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"createdBy"`
	TenantID    int       `json:"tenantId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// SystemConfigListResponse 列表响应
type SystemConfigListResponse struct {
	Items      []SystemConfigResponse `json:"items"`
	Total      int                    `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"pageSize"`
	TotalPages int                    `json:"totalPages"`
}

type ListSystemConfigsRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"pageSize" binding:"min=1,max=1000"`
	Category string `form:"category"`
}

// UpdateSystemConfigRequest 更新请求
type UpdateSystemConfigRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"valueType"`
	Description string `json:"description"`
}
