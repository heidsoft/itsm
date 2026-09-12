package dto

import (
	"strings"
	"time"
)

// 角色启用状态的线上契约取值。持久层是 role.is_active(bool)，
// 对外统一表达为 status，避免出现 isActive/status 两套同义字段。
const (
	RoleStatusActive   = "active"
	RoleStatusInactive = "inactive"
)

// RoleStatusFromActive 把持久层 is_active 映射为线上契约 status。
func RoleStatusFromActive(isActive bool) string {
	if isActive {
		return RoleStatusActive
	}
	return RoleStatusInactive
}

// RoleActiveFromStatus 解析线上契约 status。ok=false 表示取值非法，
// 调用方必须拒绝请求（1001），不得静默回退到默认值落库。
func RoleActiveFromStatus(status string) (active bool, ok bool) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case RoleStatusActive:
		return true, true
	case RoleStatusInactive:
		return false, true
	default:
		return false, false
	}
}

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
	DataScope   string   `json:"dataScope"` // all/department/owner
}

// RoleListResponse represents the response for listing roles
type RoleListResponse struct {
	Roles      []RoleDTO `json:"roles"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"pageSize"`
	TotalPages int       `json:"totalPages,omitempty"`
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
	// Status 是启用状态的唯一线上契约（active/inactive），落库到 role.is_active。
	// 曾并存同义字段 isActive，但 service 只读 isActive 导致前端发送的 status 被静默丢弃，
	// 角色启用/禁用开关表现为「更新成功却毫无变化」，故删除双轨字段。
	Status *string `json:"status"`
}

// GetRolesParams represents the query parameters for listing roles
type GetRolesParams struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
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
	Status      string           `json:"status"`    // active/inactive，启用状态的唯一线上契约
	IsActive    bool             `json:"-"`         // 内部字段：Status 的来源，不单独出 JSON，避免同义双字段
	DataScope   string           `json:"dataScope"` // all/department/owner
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
