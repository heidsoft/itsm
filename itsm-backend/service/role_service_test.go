package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

// TestCreateRole_DefaultIsActiveTrue 验证 CreateRole 默认 is_active=true
func TestCreateRole_DefaultIsActiveTrue(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent_role?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewRoleService(client, logger)
	ctx := context.Background()

	// 创建租户
	tenant, err := client.Tenant.Create().
		SetName("Tenant Role").
		SetCode("TR").
		SetDomain("tr.com").
		SetStatus("active").
		Save(ctx)
	assert.NoError(t, err)

	// 创建角色（提供 Code 以绕过 generateCodeFromName 的 regexp 编译 bug）
	req := &dto.CreateRoleRequest{
		Name:        "TestRole",
		Code:        "test_role",
		Description: "Test Description",
		IsSystem:    false,
	}
	resp, err := svc.CreateRole(ctx, req, tenant.ID)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "TestRole", resp.Name)
	assert.True(t, resp.IsActive, "新建角色默认 is_active 应为 true")

	// 从数据库再次验证
	roleEntity, err := client.Role.Get(ctx, resp.ID)
	assert.NoError(t, err)
	assert.True(t, roleEntity.IsActive, "数据库中角色 is_active 字段应为 true")
}

// TestRoleService_DeleteRole_TenantIsolation 跨租户删除角色隔离测试
func TestRoleService_DeleteRole_TenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent_role2?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewRoleService(client, logger)
	ctx := context.Background()

	// Create two tenants
	tenant1, err := client.Tenant.Create().
		SetName("Tenant 1").
		SetCode("TENANT1").
		SetDomain("tenant1.com").
		SetStatus("active").
		Save(ctx)
	assert.NoError(t, err)

	tenant2, err := client.Tenant.Create().
		SetName("Tenant 2").
		SetCode("TENANT2").
		SetDomain("tenant2.com").
		SetStatus("active").
		Save(ctx)
	assert.NoError(t, err)

	// Tenant 1 creates a role
	role1, err := client.Role.Create().
		SetName("Tenant1Role").
		SetCode("tenant1_role").
		SetTenantID(tenant1.ID).
		SetIsSystem(false).
		Save(ctx)
	assert.NoError(t, err)

	// Tenant 2 tries to delete Tenant 1's role
	err = svc.DeleteRole(ctx, role1.ID, tenant2.ID)

	// Should fail with cross-tenant access error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "角色不存在")
}

// TestDeleteRole_IsSystemProtected 验证 is_system=true 的角色不能删除（第一层保护）
func TestDeleteRole_IsSystemProtected(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent_role_sys?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewRoleService(client, logger)
	ctx := context.Background()

	tenant, _ := client.Tenant.Create().
		SetName("T").SetCode("T").SetDomain("t.com").SetStatus("active").
		Save(ctx)

	// 创建 is_system=true 的角色
	role, _ := client.Role.Create().
		SetName("系统角色").SetCode("some_system_role").
		SetTenantID(tenant.ID).
		SetIsSystem(true).
		Save(ctx)

	err := svc.DeleteRole(ctx, role.ID, tenant.ID)
	assert.Error(t, err, "is_system=true 应阻止删除")
	assert.Contains(t, err.Error(), "系统角色不能删除")

	// 角色应仍然存在
	_, err = client.Role.Get(ctx, role.ID)
	assert.NoError(t, err, "被阻止删除的角色应仍然存在")
}

// TestDeleteRole_CodeBasedProtection 验证 code 命中内置词表时即使 is_system=false 也不能删（双重保护）
// 这是 B5.11 P0 修复的核心：防止 seed 时 is_system 漏设导致系统角色可被删
func TestDeleteRole_CodeBasedProtection(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent_role_code?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewRoleService(client, logger)
	ctx := context.Background()

	tenant, _ := client.Tenant.Create().
		SetName("T").SetCode("T").SetDomain("t.com").SetStatus("active").
		Save(ctx)

	// 用内置 code 但 is_system=false（模拟 seeder 漏设场景）
	for _, builtinCode := range []string{"super_admin", "admin", "end_user", "agent", "it_admin"} {
		role, _ := client.Role.Create().
			SetName("内置角色-"+builtinCode).SetCode(builtinCode).
			SetTenantID(tenant.ID).
			SetIsSystem(false). // 关键：is_system=false 但 code 是内置
			Save(ctx)

		err := svc.DeleteRole(ctx, role.ID, tenant.ID)
		assert.Error(t, err, "内置 code=%s 即使 is_system=false 也应被阻止删除", builtinCode)
		assert.Contains(t, err.Error(), "内置系统角色不能删除")

		// 验证角色仍然存在
		_, err = client.Role.Get(ctx, role.ID)
		assert.NoError(t, err, "被阻止删除的角色应仍然存在")
	}
}

// TestDeleteRole_NonBuiltinOK 验证非内置角色 is_system=false 可以正常删除
func TestDeleteRole_NonBuiltinOK(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent_role_ok?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewRoleService(client, logger)
	ctx := context.Background()

	tenant, _ := client.Tenant.Create().
		SetName("T").SetCode("T").SetDomain("t.com").SetStatus("active").
		Save(ctx)

	// 创建自定义角色，code 不在内置词表中
	role, _ := client.Role.Create().
		SetName("自定义外包角色").SetCode("outsourced_contractor").
		SetTenantID(tenant.ID).
		SetIsSystem(false).
		Save(ctx)

	err := svc.DeleteRole(ctx, role.ID, tenant.ID)
	assert.NoError(t, err, "非内置角色应能正常删除")

	// 验证角色已删除
	_, err = client.Role.Get(ctx, role.ID)
	assert.Error(t, err, "删除后的角色不应存在")
}
