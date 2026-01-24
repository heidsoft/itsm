package dto

import "time"

// RoleDTO represents a role data transfer object
type RoleDTO struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status,omitempty"`
	IsSystem    bool     `json:"is_system,omitempty"`
	UserCount   int      `json:"user_count,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// RoleListResponse represents the response for listing roles
type RoleListResponse struct {
	Roles     []RoleDTO `json:"roles"`
	Total     int       `json:"total"`
	Page      int       `json:"page"`
	PageSize  int       `json:"page_size"`
	TotalPage int       `json:"total_pages,omitempty"`
}

// CreateRoleRequest represents the request for creating a role
type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
}

// UpdateRoleRequest represents the request for updating a role
type UpdateRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
}

// GetRolesParams represents the query parameters for listing roles
type GetRolesParams struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   string `form:"status"`
	Search   string `form:"search"`
}

// PermissionDTO represents a permission
type PermissionDTO struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Module      string    `json:"module"`
	Action      string    `json:"action"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PermissionListResponse represents the response for listing permissions
type PermissionListResponse struct {
	Permissions []PermissionDTO `json:"permissions"`
	Total       int             `json:"total"`
}
