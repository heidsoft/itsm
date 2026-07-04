package dto

// MenuDTO 菜单数据传输对象
type MenuDTO struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Path           string    `json:"path"`
	Icon           string    `json:"icon,omitempty"`
	ParentID       *int      `json:"parentId,omitempty"`
	PermissionCode *string   `json:"permissionCode,omitempty"`
	SortOrder      int       `json:"sortOrder"`
	TenantID       int       `json:"tenantId"`
	IsVisible      bool      `json:"isVisible"`
	IsEnabled      bool      `json:"isEnabled"`
	Description    string    `json:"description,omitempty"`
	Children       []MenuDTO `json:"children,omitempty"`
}

// CreateMenuRequest 创建菜单请求
type CreateMenuRequest struct {
	Name           string  `json:"name" binding:"required"`
	Path           string  `json:"path" binding:"required"`
	Icon           string  `json:"icon,omitempty"`
	ParentID       *int    `json:"parentId,omitempty"`
	PermissionCode *string `json:"permissionCode,omitempty"`
	SortOrder      int     `json:"sortOrder"`
	IsVisible      *bool   `json:"isVisible,omitempty"`
	IsEnabled      *bool   `json:"isEnabled,omitempty"`
	Description    string  `json:"description,omitempty"`
}

// UpdateMenuRequest 更新菜单请求
type UpdateMenuRequest struct {
	Name           *string `json:"name,omitempty"`
	Path           *string `json:"path,omitempty"`
	Icon           *string `json:"icon,omitempty"`
	ParentID       *int    `json:"parentId,omitempty"`
	PermissionCode *string `json:"permissionCode,omitempty"`
	SortOrder      *int    `json:"sortOrder,omitempty"`
	IsVisible      *bool   `json:"isVisible,omitempty"`
	IsEnabled      *bool   `json:"isEnabled,omitempty"`
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
	HasPermission bool `json:"hasPermission"`
}

type MenuListResponse struct {
	Menus []*MenuDTO `json:"menus"`
	Total int        `json:"total"`
}

type MenuInitResponse struct {
	Message string `json:"message"`
	Count   int    `json:"count,omitempty"`
}
