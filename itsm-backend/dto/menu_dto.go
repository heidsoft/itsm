package dto

// MenuDTO 菜单数据传输对象
type MenuDTO struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Path           string    `json:"path"`
	Icon           string    `json:"icon,omitempty"`
	ParentID       *int      `json:"parent_id,omitempty"`
	PermissionCode *string   `json:"permission_code,omitempty"`
	SortOrder      int       `json:"sort_order"`
	TenantID       int       `json:"tenant_id"`
	IsVisible      bool      `json:"is_visible"`
	IsEnabled      bool      `json:"is_enabled"`
	Description    string    `json:"description,omitempty"`
	Children       []MenuDTO `json:"children,omitempty"`
}

// CreateMenuRequest 创建菜单请求
type CreateMenuRequest struct {
	Name           string  `json:"name" binding:"required"`
	Path           string  `json:"path" binding:"required"`
	Icon           string  `json:"icon,omitempty"`
	ParentID       *int    `json:"parent_id,omitempty"`
	PermissionCode *string `json:"permission_code,omitempty"`
	SortOrder      int     `json:"sort_order"`
	IsVisible      *bool   `json:"is_visible,omitempty"`
	IsEnabled      *bool   `json:"is_enabled,omitempty"`
	Description    string  `json:"description,omitempty"`
}

// UpdateMenuRequest 更新菜单请求
type UpdateMenuRequest struct {
	Name           *string `json:"name,omitempty"`
	Path           *string `json:"path,omitempty"`
	Icon           *string `json:"icon,omitempty"`
	ParentID       *int    `json:"parent_id,omitempty"`
	PermissionCode *string `json:"permission_code,omitempty"`
	SortOrder      *int    `json:"sort_order,omitempty"`
	IsVisible      *bool   `json:"is_visible,omitempty"`
	IsEnabled      *bool   `json:"is_enabled,omitempty"`
	Description    *string `json:"description,omitempty"`
}

// MenuTreeResponse 菜单树响应
type MenuTreeResponse struct {
	Main  []MenuDTO `json:"main"`
	Admin []MenuDTO `json:"admin"`
}

// MenuWithPermission 菜单权限信息
type MenuWithPermission struct {
	MenuDTO
	HasPermission bool `json:"has_permission"`
}
