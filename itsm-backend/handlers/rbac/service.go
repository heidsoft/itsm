package rbac

import (
	"context"

	"itsm-backend/dto"
)

// 注意：handler 仍需 import "itsm-backend/service" 仅用于哨兵
// service.ErrPermissionNotInTenant 的 errors.Is 判断（跨租户权限分配 400 语义）。
// 哨兵属 service 包定义，接口化后 handler 对该包的依赖收敛为常量引用，不再依赖具体类型。

// RoleService 定义角色域业务接口（依赖倒置）。
// 实现由 internal/bootstrap 装配的 service.RoleService 提供。
type RoleService interface {
	CreateRole(ctx context.Context, req *dto.CreateRoleRequest, tenantID int) (*dto.RoleResponse, error)
	GetRole(ctx context.Context, id int, tenantID int) (*dto.RoleResponse, error)
	ListRoles(ctx context.Context, tenantID int, page, pageSize int, search string) ([]*dto.RoleResponse, int, error)
	UpdateRole(ctx context.Context, id int, req *dto.UpdateRoleRequest, tenantID int) (*dto.RoleResponse, error)
	DeleteRole(ctx context.Context, id int, tenantID int) error
	AssignPermissions(ctx context.Context, roleID int, permissionIDs []int, tenantID int) error
}

// PermissionService 定义权限域业务接口（依赖倒置）。
type PermissionService interface {
	CreatePermission(ctx context.Context, req *dto.CreatePermissionRequest, tenantID int) (*dto.PermissionResponse, error)
	ListPermissions(ctx context.Context, tenantID int, resource string) ([]*dto.PermissionResponse, error)
	InitDefaultPermissions(ctx context.Context, tenantID int) error
}

// MenuService 定义菜单域业务接口（依赖倒置）。
type MenuService interface {
	CreateMenu(ctx context.Context, req *dto.CreateMenuRequest, tenantID int) (*dto.MenuDTO, error)
	GetMenu(ctx context.Context, id int, tenantID int) (*dto.MenuDTO, error)
	ListMenus(ctx context.Context, tenantID int) ([]*dto.MenuDTO, error)
	UpdateMenu(ctx context.Context, id int, req *dto.UpdateMenuRequest, tenantID int) (*dto.MenuDTO, error)
	DeleteMenu(ctx context.Context, id int, tenantID int) error
	GetUserMenus(ctx context.Context, userID int, tenantID int) (*dto.MenuTreeResponse, error)
}
