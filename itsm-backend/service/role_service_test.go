package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestRoleService_DeleteRole_TenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
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
