package integration

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestTenantServiceIntegration 测试租户服务
func TestTenantServiceIntegration(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	_ = logger

	t.Run("创建租户", func(t *testing.T) {
		tenant, err := client.Tenant.Create().
			SetName("Test Company").
			SetCode("TESTCO").
			SetDomain("test.com").
			SetStatus("active").
			Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, tenant)
		require.Equal(t, "Test Company", tenant.Name)
	})
}

// TestDepartmentServiceIntegration 测试部门服务
func TestDepartmentServiceIntegration(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	_ = logger

	// 先创建租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("DEPT").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	t.Run("创建部门", func(t *testing.T) {
		dept, err := client.Department.Create().
			SetName("IT部门").
			SetCode("IT").
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, dept)
		require.Equal(t, "IT部门", dept.Name)
	})
}

// TestRoleServiceIntegration 测试角色服务
func TestRoleServiceIntegration(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	_ = logger

	// 先创建租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("ROLE").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	t.Run("创建角色", func(t *testing.T) {
		role, err := client.Role.Create().
			SetName("管理员").
			SetCode("admin").
			SetDescription("系统管理员").
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, role)
		require.Equal(t, "管理员", role.Name)
	})
}

// TestTeamServiceIntegration 测试团队服务
func TestTeamServiceIntegration(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	_ = logger

	// 先创建租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEAM").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	t.Run("创建团队", func(t *testing.T) {
		team, err := client.Team.Create().
			SetName("运维团队").
			SetCode("OPS").
			SetDescription("负责系统运维").
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
		require.NotNil(t, team)
		require.Equal(t, "运维团队", team.Name)
	})
}
