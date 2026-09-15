package service_request

import (
	"context"
	"path/filepath"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindActiveUsersByRole_DualSource 验证按角色查人覆盖双数据源：
// users.role 枚举 ∪ user_roles M2M 边（roles.code），且租户/活跃/部门过滤正确。
// 背景：此前只查枚举，it_admin 等角色在枚举里缺失时恒查空 → 审批回退 super_admin。
func TestFindActiveUsersByRole_DualSource(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "sr_find_by_role.db") + "?_fk=1"
	client := enttest.Open(t, "sqlite3", dsn)
	defer client.Close()
	ctx := context.Background()

	tenantA, err := client.Tenant.Create().SetName("A").SetCode("A-FBR").SetDomain("a.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().SetName("B").SetCode("B-FBR").SetDomain("b.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	mgrA, err := client.Role.Create().SetCode("manager").SetName("经理").SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)
	itAdminA, err := client.Role.Create().SetCode("it_admin").SetName("IT管理").SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)
	_, err = client.Role.Create().SetCode("it_admin").SetName("IT管理B").SetTenantID(tenantB.ID).Save(ctx)
	require.NoError(t, err)

	newUser := func(tenantID int, username, enumRole, dept string, active bool, roleIDs ...int) int {
		u, err := client.User.Create().
			SetUsername(username).
			SetEmail(username + "@test.com").
			SetName(username).
			SetPasswordHash("hash").
			SetRole(user.Role(enumRole)).
			SetDepartment(dept).
			SetActive(active).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
		if len(roleIDs) > 0 {
			_, err = client.User.UpdateOne(u).AddRoleIDs(roleIDs...).Save(ctx)
			require.NoError(t, err)
		}
		return u.ID
	}

	enumMgr := newUser(tenantA.ID, "enum-manager", "manager", "IT", true)                            // 仅枚举命中
	m2mOnly := newUser(tenantA.ID, "m2m-itadmin", "end_user", "IT", true, itAdminA.ID)               // 仅 M2M 命中
	both := newUser(tenantA.ID, "both-manager", "manager", "IT", true, mgrA.ID, itAdminA.ID)         // 双命中须去重
	inactiveMgr := newUser(tenantA.ID, "inactive-mgr", "manager", "IT", false)                       // 非活跃排除
	hrMgr := newUser(tenantA.ID, "hr-manager", "manager", "HR", true)                                // 部门过滤
	_ = newUser(tenantA.ID, "other-role", "end_user", "IT", true)                                    // 无关角色
	_ = newUser(tenantB.ID, "tenant-b-itadmin", "end_user", "IT", true)                              // 他租户排除

	repo := NewEntRepository(client)

	t.Run("枚举与M2M并集且去重", func(t *testing.T) {
		ids, err := repo.FindActiveUsersByRole(ctx, tenantA.ID, "manager", "")
		require.NoError(t, err)
		assert.ElementsMatch(t, []int{enumMgr, both, hrMgr}, ids)
	})

	t.Run("M2M边命中", func(t *testing.T) {
		ids, err := repo.FindActiveUsersByRole(ctx, tenantA.ID, "it_admin", "")
		require.NoError(t, err)
		assert.ElementsMatch(t, []int{m2mOnly, both}, ids, "it_admin 仅存在于 M2M 边，必须命中")
	})

	t.Run("部门过滤", func(t *testing.T) {
		ids, err := repo.FindActiveUsersByRole(ctx, tenantA.ID, "manager", "HR")
		require.NoError(t, err)
		assert.ElementsMatch(t, []int{hrMgr}, ids)
	})

	t.Run("非活跃排除", func(t *testing.T) {
		ids, err := repo.FindActiveUsersByRole(ctx, tenantA.ID, "manager", "")
		require.NoError(t, err)
		assert.NotContains(t, ids, inactiveMgr)
	})

	t.Run("租户隔离", func(t *testing.T) {
		ids, err := repo.FindActiveUsersByRole(ctx, tenantB.ID, "it_admin", "")
		require.NoError(t, err)
		assert.NotContains(t, ids, m2mOnly)
		assert.NotContains(t, ids, both)
	})
}
