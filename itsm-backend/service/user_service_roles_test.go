package service

import (
	"context"
	"path/filepath"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestUserService_SyncUserRoles 验证 M2M 角色边写入：整体替换语义 + 跨租户角色拒绝。
// 背景：新建用户此前只写 users.role 枚举、从不写 user_roles 边，导致按 M2M 解析
// 角色的链路（抄送/审批人解析）收不到人（2026-09-15 GitHub issue 复盘）。
func TestUserService_SyncUserRoles(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "user_roles_sync.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()
	ctx := context.Background()
	svc := NewUserService(client, zaptest.NewLogger(t).Sugar())

	tenantA, err := client.Tenant.Create().SetName("A").SetCode("A-URS").SetDomain("a.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().SetName("B").SetCode("B-URS").SetDomain("b.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	r1, err := client.Role.Create().SetCode("manager").SetName("经理").SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)
	r2, err := client.Role.Create().SetCode("it_admin").SetName("IT管理").SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)
	rForeign, err := client.Role.Create().SetCode("agent").SetName("坐席B").SetTenantID(tenantB.ID).Save(ctx)
	require.NoError(t, err)

	u, err := client.User.Create().
		SetUsername("urs-user").
		SetEmail("urs@test.com").
		SetName("URS").
		SetPasswordHash("hash").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)

	roleIDsOf := func(t *testing.T) []int {
		t.Helper()
		roles, err := client.User.QueryRoles(u).All(ctx)
		require.NoError(t, err)
		ids := make([]int, 0, len(roles))
		for _, r := range roles {
			ids = append(ids, r.ID)
		}
		return ids
	}

	t.Run("写入多角色", func(t *testing.T) {
		require.NoError(t, svc.SyncUserRoles(ctx, u.ID, tenantA.ID, []int{r1.ID, r2.ID}))
		assert.ElementsMatch(t, []int{r1.ID, r2.ID}, roleIDsOf(t))
	})

	t.Run("整体替换", func(t *testing.T) {
		require.NoError(t, svc.SyncUserRoles(ctx, u.ID, tenantA.ID, []int{r2.ID}))
		assert.ElementsMatch(t, []int{r2.ID}, roleIDsOf(t), "重复同步应替换而非累积")
	})

	t.Run("清空", func(t *testing.T) {
		require.NoError(t, svc.SyncUserRoles(ctx, u.ID, tenantA.ID, []int{}))
		assert.Empty(t, roleIDsOf(t))
	})

	t.Run("跨租户角色拒绝", func(t *testing.T) {
		err := svc.SyncUserRoles(ctx, u.ID, tenantA.ID, []int{rForeign.ID})
		require.Error(t, err, "挂他租户角色必须被拒绝（防提权）")
		assert.Empty(t, roleIDsOf(t), "拒绝后不得留下半写状态")
	})
}

// TestUserService_CreateUser_SyncsPrimaryRoleToM2M 验证创建用户时主角色同步到 user_roles 边。
func TestUserService_CreateUser_SyncsPrimaryRoleToM2M(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "user_create_roles.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()
	ctx := context.Background()
	svc := NewUserService(client, zaptest.NewLogger(t).Sugar())

	tenantA, err := client.Tenant.Create().SetName("A").SetCode("A-UCR").SetDomain("a.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	_, err = client.Role.Create().SetCode("manager").SetName("经理").SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)

	req := &dto.CreateUserRequest{
		Username: "ucr-user",
		Email:    "ucr@test.com",
		Name:     "UCR",
		Password: "Abcdefgh!23456",
		Role:     "manager",
	}
	_, err = svc.CreateUser(ctx, req, tenantA.ID)
	require.NoError(t, err)

	u, err := client.User.Query().Where(user.UsernameEQ("ucr-user")).Only(ctx)
	require.NoError(t, err)
	roles, err := client.User.QueryRoles(u).All(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 1, "主角色应同步写入 user_roles 边")
	assert.Equal(t, "manager", roles[0].Code)
}
