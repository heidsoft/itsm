package rbac

import (
	"context"

	"itsm-backend/dto"
)

// 角色域 / 权限域的 Ent 实现与 ErrPermissionNotInTenant 哨兵已随批次 2 开局
// 迁入本域包（role_service_impl.go），handler 不再依赖共享层；菜单实现暂仍由
// 共享层提供，归后续批次收口。接口签名保持不变，装配点只换构造函数的包限定符。

// RoleService 定义角色域业务接口（依赖倒置）。
// 实现由 internal/bootstrap 装配的 NewRoleService 提供（同包 roleServiceImpl）。
type RoleService interface {
	CreateRole(ctx context.Context, req *dto.CreateRoleRequest, tenantID int) (*dto.RoleResponse, error)
	GetRole(ctx context.Context, id int, tenantID int) (*dto.RoleResponse, error)
	ListRoles(ctx context.Context, tenantID int, params *dto.GetRolesParams) ([]*dto.RoleResponse, int, error)
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
	ExportMenus(ctx context.Context, tenantID int) ([]dto.MenuExportItem, error)
	InitMenusFromBaseline(ctx context.Context, tenantID int) (*dto.MenuInitDiffResponse, error)
}
