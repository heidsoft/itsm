package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	_ "github.com/mattn/go-sqlite3"
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
