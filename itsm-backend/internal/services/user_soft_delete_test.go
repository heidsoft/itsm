package service

import (
	"context"
	"testing"

	"itsm-backend/database"
	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 删除/停用语义分离回归（2026-10-29，DB 审计 P3）。
//
// 修复前：DeleteUser 与 ChangeUserStatus(false) 同态（都只写 active=false）——
// 列表仍能看到「已删除」的用户，且 toggle 启用可以复活已删除账号。
// 修复后：删除写 deleted_at，读路径经 RegisterSoftDeleteInterceptors（生产
// 由 internal/bootstrap 注册）统一屏蔽；停用语义不变（仍可见可启用）。
//
// 本测试显式注册拦截器对齐生产读路径，断言的结果在修复前的实现上必然失败
// （列表过滤与「复活拒绝」都不会发生），即修复前红/修复后绿。
func TestUserService_DeleteUser_SeparatesFromDisable(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	database.RegisterSoftDeleteInterceptors(client)

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewUserService(client, logger)
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("DelSep Tenant").
		SetCode("DELSEP").
		SetDomain("delsep.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	mkUser := func(t *testing.T, username string) int {
		t.Helper()
		u, err := client.User.Create().
			SetUsername(username).
			SetEmail(username + "@delsep.test").
			SetPasswordHash("hashed").
			SetName(username).
			SetActive(true).
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
		return u.ID
	}

	deletedID := mkUser(t, "deleted_user")
	disabledID := mkUser(t, "disabled_user")
	// 另一位活跃用户保证列表非空
	mkUser(t, "survivor")

	t.Run("删除后从列表消失、不可复活，停用仍可见可启用", func(t *testing.T) {
		// 1. 删除语义：deleted_at 隔离
		require.NoError(t, svc.DeleteUser(ctx, deletedID, tenant.ID))

		users, err := svc.ListUsers(ctx, &dto.ListUsersRequest{Page: 1, PageSize: 20}, tenant.ID)
		require.NoError(t, err)
		for _, u := range users.Items {
			assert.NotEqual(t, deletedID, u.ID, "已删除用户不应出现在列表中")
		}

		_, err = svc.GetUserByID(ctx, deletedID, tenant.ID)
		require.Error(t, err, "已删除用户应查询不到")
		assert.Contains(t, err.Error(), "用户不存在")

		// toggle 启用对已删除用户必须拒绝——删除语义不可被启用复活
		err = svc.ChangeUserStatus(ctx, deletedID, true, 0, tenant.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "用户不存在")

		// 2. 停用语义：active=false 但仍可见、可再启用
		require.NoError(t, svc.ChangeUserStatus(ctx, disabledID, false, 0, tenant.ID))

		users, err = svc.ListUsers(ctx, &dto.ListUsersRequest{Page: 1, PageSize: 20}, tenant.ID)
		require.NoError(t, err)
		var disabled *dto.UserDetailResponse
		for _, u := range users.Items {
			if u.ID == disabledID {
				disabled = u
			}
		}
		require.NotNil(t, disabled, "已停用用户应仍在列表中可见")
		assert.False(t, disabled.Active)

		require.NoError(t, svc.ChangeUserStatus(ctx, disabledID, true, 0, tenant.ID))
		users, err = svc.ListUsers(ctx, &dto.ListUsersRequest{Page: 1, PageSize: 20}, tenant.ID)
		require.NoError(t, err)
		for _, u := range users.Items {
			if u.ID == disabledID {
				assert.True(t, u.Active, "停用用户应可重新启用")
			}
		}

		// 3. 身份隔离：两个语义各自独占的终态——活跃租户里始终只剩
		// survivor(active) 与 disabled_user（bool 复位后 active）
		users, err = svc.ListUsers(ctx, &dto.ListUsersRequest{Page: 1, PageSize: 20}, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(2), users.Total)
	})
}
