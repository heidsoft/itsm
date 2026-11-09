package account

// 域内窄契约：本域 handler 只消费共享层用户服务的两个自助方法。
// 生产装配传入具体实现，编译期断言见 internal/layering/domain_contracts_test.go。
// 本文件不得引用共享层包；否则 L.3 的逐域计数会上升。

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

type UserService interface {
	ChangeOwnPassword(ctx context.Context, userID int, oldPassword, newPassword string, tenantID int) error
	UpdateOwnProfile(ctx context.Context, userID, tenantID int, req *dto.UpdateOwnProfileRequest) (*ent.User, error)
}
