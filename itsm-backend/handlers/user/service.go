package user

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

// Service 定义 user 域的业务接口（依赖倒置）。
// 实现由 internal/bootstrap 装配的 service.UserService 提供；
// 测试可注入 mock 实现，使 handler 逻辑可独立验证。
// 这是 58 个裸奔域向五件套迁移的示范样板：域内自持接口，
// 业务实现沉入 handlers/user/service.go（或复用遗留 service.UserService）。
type Service interface {
	CreateUser(ctx context.Context, req *dto.CreateUserRequest, tenantID int) (*ent.User, error)
	ListUsers(ctx context.Context, req *dto.ListUsersRequest, tenantID int) (*dto.PagedUsersResponse, error)
	GetUserByID(ctx context.Context, id int, tenantID int) (*ent.User, error)
	UpdateUser(ctx context.Context, id int, req *dto.UpdateUserRequest, tenantID int) (*ent.User, error)
	DeleteUser(ctx context.Context, id int, tenantID int) error
	ChangeUserStatus(ctx context.Context, id int, active bool, currentUserID int, tenantID int) error
	ResetPassword(ctx context.Context, id int, newPassword string, tenantID int) error
	GetUserStats(ctx context.Context, tenantID int) (*dto.UserStatsResponse, error)
	BatchUpdateUsers(ctx context.Context, req *dto.BatchUpdateUsersRequest, tenantID int) error
	SearchUsers(ctx context.Context, req *dto.SearchUsersRequest, tenantID int) ([]*dto.UserDetailResponse, error)
}
