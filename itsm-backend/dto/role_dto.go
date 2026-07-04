package dto

import "time"

// RoleDTO represents a role data transfer object
type RoleDTO struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Code        string   `json:"code"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status,omitempty"`
	IsSystem    bool     `json:"isSystem,omitempty"`
	UserCount   int      `json:"userCount,omitempty"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
	TenantID    int      `json:"tenantId"`
}

// RoleListResponse represents the response for listing roles
type RoleListResponse struct {
	Roles     []RoleDTO `json:"roles"`
	Total     int       `json:"total"`
	Page      int       `json:"page"`
	PageSize  int       `json:"pageSize"`
	TotalPage int       `json:"totalPages,omitempty"`
}

// CreateRoleRequest represents the request for creating a role
type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Code        string   `json:"code"` // Optional - auto-generated if empty
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
	IsSystem    bool     `json:"isSystem"`
}

// UpdateRoleRequest represents the request for updating a role
type UpdateRoleRequest struct {
	Name        *string  `json:"name"`
	Code        *string  `json:"code"`
	Description *string  `json:"description"`
	Permissions []string `json:"permissions"`
	Status      *string  `json:"status"`
	IsActive    *bool    `json:"isActive"` // 是否启用角色
}

// GetRolesParams represents the query parameters for listing roles
type GetRolesParams struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   string `form:"status"`
	Search   string `form:"search"`
}

// RoleResponse 角色响应（用于新角色服务）
type RoleResponse struct {
	ID          int              `json:"id"`
	Name        string           `json:"name"`
	Code        string           `json:"code"`
	Description string           `json:"description"`
	IsSystem    bool             `json:"isSystem"`
	IsActive    bool             `json:"isActive"` // 角色是否启用
	Permissions []PermissionInfo `json:"permissions"`
	TenantID    int              `json:"tenantId"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

// PermissionInfo 权限信息
type PermissionInfo struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// AssignPermissionsRequest 分配权限请求
type AssignPermissionsRequest struct {
	PermissionIDs []int `json:"permissionIds" binding:"required"`
}

// PermissionDTO represents a permission
type PermissionDTO struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Module      string    `json:"module"`
	Action      string    `json:"action"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// PermissionListResponse represents the response for listing permissions
type PermissionListResponse struct {
	Permissions []PermissionDTO `json:"permissions"`
	Total       int             `json:"total"`
}
