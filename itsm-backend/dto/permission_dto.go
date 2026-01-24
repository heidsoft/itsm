package dto

import "time"

// CreatePermissionRequest 创建权限请求
type CreatePermissionRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Resource    string `json:"resource" binding:"required"`
	Action      string `json:"action" binding:"required"`
}

// PermissionResponse 权限响应
type PermissionResponse struct {
	ID          int       `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	TenantID    int       `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
