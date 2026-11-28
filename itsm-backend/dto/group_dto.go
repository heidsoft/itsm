package dto

import (
	"time"
)

// CreateGroupRequest 创建组请求
type CreateGroupRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"omitempty,max=500"`
	TenantID    int    `json:"tenantId" binding:"omitempty,min=1"`
}

// UpdateGroupRequest 更新组请求
type UpdateGroupRequest struct {
	Name        string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Description string `json:"description,omitempty" binding:"omitempty,max=500"`
}

// ListGroupsRequest 获取组列表请求
type ListGroupsRequest struct {
	Page     int `form:"page,default=1" binding:"min=1"`
	PageSize int `form:"pageSize,default=10" binding:"min=1,max=100"`
	// TenantID 由 handler 从认证上下文写入，禁止从查询参数绑定，
	// 否则调用方可传 ?tenantId=N 读取他租户的组（跨租户 IDOR）。
	TenantID int    `form:"-"`
	Search   string `form:"search"`
}

// AddUserToGroupRequest 添加用户到组请求
type AddUserToGroupRequest struct {
	UserID int `json:"userId" binding:"required,min=1"`
}

// RemoveUserFromGroupRequest 从组移除用户请求
type RemoveUserFromGroupRequest struct {
	UserID int `json:"userId" binding:"required,min=1"`
}

// GroupResponse 组响应
// MemberCount 让列表页能直接显示成员数，不必为每个组再发一次 /members 请求；
// 历史前端依赖 members[] 而列表接口从未返回该字段，徽标恒为 0。
type GroupResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	TenantID    int       `json:"tenantId"`
	MemberCount int       `json:"memberCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// PagedGroupsResponse 分页组响应（标准列表信封）
type PagedGroupsResponse struct {
	Items      []*GroupResponse `json:"items"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
	TotalPages int              `json:"totalPages"`
}

// GroupMembersResponse 组成员响应（标准列表信封）
type GroupMembersResponse struct {
	Items      []*UserDTO `json:"items"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	TotalPages int        `json:"totalPages"`
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
	TenantID    int        `json:"tenantId"`
	Members     []*UserDTO `json:"members,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// GroupReferenceDTO 一处审批引用来源。
//
// 与 handlers/group.GroupReference 一一对应，是 API 层的稳定契约：
//   - Kind: process_definition（BPMN userTask）/ approval_chain（审批链 step）；
//   - RefID/RefKey: 引用来源实体 ID 与可读标识；
//   - NodeID: 引用位置（BPMN 节点 id / 审批链 level-N）。
//
// 只读诊断用途，前端据此在改名/删除前提示管理员「谁在用这个组」。
type GroupReferenceDTO struct {
	Kind   string `json:"kind"`
	RefID  int    `json:"refId"`
	RefKey string `json:"refKey"`
	NodeID string `json:"nodeId"`
}

// GroupReferencesResponse 组审批引用查询响应（列表信封）。
// Items 永远是非 nil 切片，空引用返回 [] 而非 null，避免前端误判为加载失败。
type GroupReferencesResponse struct {
	Items []*GroupReferenceDTO `json:"items"`
	Total int64                `json:"total"`
}
