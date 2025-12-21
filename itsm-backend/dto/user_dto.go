package dto

import (
	"time"
)

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
    Username   string `json:"username" binding:"required,min=3,max=50"`
    Email      string `json:"email" binding:"required,email"`
    Name       string `json:"name" binding:"required,min=1,max=100"`
    Department string `json:"department"`
    Phone      string `json:"phone"`
    Password   string `json:"password" binding:"required,min=6"`
    TenantID   int    `json:"tenant_id" binding:"required,min=1"`
    // 角色，可选；不提供时使用后端默认值（end_user）
    Role       string `json:"role,omitempty" binding:"omitempty,oneof=super_admin admin manager agent technician security end_user"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
    Username   string `json:"username,omitempty" binding:"omitempty,min=3,max=50"`
    Email      string `json:"email,omitempty" binding:"omitempty,email"`
    Name       string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
    Department string `json:"department,omitempty"`
    Phone      string `json:"phone,omitempty"`
    // 角色更新，仅管理员有权限更新
    Role       string `json:"role,omitempty" binding:"omitempty,oneof=super_admin admin manager agent technician security end_user"`
}

// ListUsersRequest 获取用户列表请求
type ListUsersRequest struct {
	Page       int    `form:"page,default=1" binding:"min=1"`
	PageSize   int    `form:"page_size,default=10" binding:"min=1,max=100"`
	TenantID   int    `form:"tenant_id"`
	Status     string `form:"status"` // active, inactive
	Department string `form:"department"`
	Search     string `form:"search"`
}

// UserDetailResponse 用户详细响应
type UserDetailResponse struct {
    ID         int       `json:"id"`
    Username   string    `json:"username"`
    Email      string    `json:"email"`
    Name       string    `json:"name"`
    Department string    `json:"department"`
    Phone      string    `json:"phone"`
    Active     bool      `json:"active"`
    TenantID   int       `json:"tenant_id"`
    Role       string    `json:"role"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}

// PagedUsersResponse 分页用户响应
type PagedUsersResponse struct {
	Users      []*UserDetailResponse `json:"users"`
	Pagination PaginationResponse    `json:"pagination"`
}

// PaginationResponse 分页响应
type PaginationResponse struct {
	Page      int `json:"page"`
	PageSize  int `json:"page_size"`
	Total     int `json:"total"`
	TotalPage int `json:"total_page"`
}

// ChangeUserStatusRequest 更改用户状态请求
type ChangeUserStatusRequest struct {
	Active bool `json:"active"`
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// UserStatsResponse 用户统计响应
type UserStatsResponse struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Inactive int `json:"inactive"`
}

// BatchUpdateUsersRequest 批量更新用户请求
type BatchUpdateUsersRequest struct {
	UserIDs    []int  `json:"user_ids" binding:"required,min=1"`
	Action     string `json:"action" binding:"required,oneof=activate deactivate department"`
	Department string `json:"department,omitempty"`
}

// SearchUsersRequest 搜索用户请求
type SearchUsersRequest struct {
	Keyword  string `json:"keyword" form:"keyword" binding:"omitempty,min=1"`
	TenantID int    `json:"tenant_id" form:"tenant_id"`
	Limit    int    `json:"limit" form:"limit,default=10" binding:"min=1,max=50"`
}

// ImportUsersRequest 批量导入用户请求
type ImportUsersRequest struct {
    Users    []CreateUserRequest `json:"users" binding:"required,min=1,max=100"`
    TenantID int                 `json:"tenant_id" binding:"required,min=1"`
}

// ImportUsersResponse 批量导入用户响应
type ImportUsersResponse struct {
	Success   []UserDetailResponse `json:"success"`
	Failed    []ImportError        `json:"failed"`
	Total     int                  `json:"total"`
	Processed int                  `json:"processed"`
}

// ImportError 导入错误
type ImportError struct {
	Index int               `json:"index"`
	User  CreateUserRequest `json:"user"`
	Error string            `json:"error"`
}
