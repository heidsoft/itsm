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
	ValueType   string    `json:"value_type"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"createdBy"`
	TenantID    int       `json:"tenant_id"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// SystemConfigListResponse 列表响应
type SystemConfigListResponse struct {
	Configs []SystemConfigResponse `json:"configs"`
	Total   int                    `json:"total"`
	Page    int                    `json:"page"`
	Size    int                    `json:"size"`
}

// UpdateSystemConfigRequest 更新请求
type UpdateSystemConfigRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"valueType"`
	Description string `json:"description"`
}
