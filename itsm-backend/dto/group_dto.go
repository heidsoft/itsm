package dto

import (
	"time"
)

// CreateGroupRequest 创建组请求
type CreateGroupRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"omitempty,max=500"`
	TenantID    int    `json:"tenant_id" binding:"omitempty,min=1"`
}

// UpdateGroupRequest 更新组请求
type UpdateGroupRequest struct {
	Name        string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Description string `json:"description,omitempty" binding:"omitempty,max=500"`
}

// ListGroupsRequest 获取组列表请求
type ListGroupsRequest struct {
	Page     int    `form:"page,default=1" binding:"min=1"`
	PageSize int    `form:"page_size,default=10" binding:"min=1,max=100"`
	TenantID int    `form:"tenant_id"`
	Search   string `form:"search"`
}

// AddUserToGroupRequest 添加用户到组请求
type AddUserToGroupRequest struct {
	UserID int `json:"user_id" binding:"required,min=1"`
}

// RemoveUserFromGroupRequest 从组移除用户请求
type RemoveUserFromGroupRequest struct {
	UserID int `json:"user_id" binding:"required,min=1"`
}

// GroupResponse 组响应
type GroupResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	TenantID    int       `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PagedGroupsResponse 分页组响应
type PagedGroupsResponse struct {
	Groups     []*GroupResponse   `json:"groups"`
	Pagination PaginationResponse `json:"pagination"`
}

// GroupMembersResponse 组成员响应
type GroupMembersResponse struct {
	Members    []*UserDTO         `json:"members"`
	Pagination PaginationResponse `json:"pagination"`
}

// UserDTO 用户简略信息（用于组成员）
type UserDTO struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// GroupDetailResponse 组详细响应（包含成员）
type GroupDetailResponse struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	TenantID    int        `json:"tenant_id"`
	Members     []*UserDTO `json:"members,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
