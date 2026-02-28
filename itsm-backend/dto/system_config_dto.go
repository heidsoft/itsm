package dto

import "time"

// SystemConfigRequest 请求
type SystemConfigRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"value_type"`
	Category    string `json:"category"`
	Description string `json:"description"`
	TenantID    int    `json:"tenant_id"`
}

// SystemConfigResponse 响应
type SystemConfigResponse struct {
	ID          int       `json:"id"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	ValueType   string    `json:"value_type"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	TenantID    int       `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
	ValueType   string `json:"value_type"`
	Description string `json:"description"`
}
