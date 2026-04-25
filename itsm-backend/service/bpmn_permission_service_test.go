package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/bpmnpermission"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestBPMNPermissionService_RevokePermission_TenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewBPMNPermissionService(client, logger)
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

	// Tenant 1 creates a permission
	perm, err := client.BPMNPermission.Create().
		SetResourceType("process_definition").
		SetResourceID(100).
		SetPermissionType("read").
		SetPrincipalType("user").
		SetPrincipalID(1).
		SetTenantID(tenant1.ID).
		SetIsGranted(true).
		Save(ctx)
	assert.NoError(t, err)
	assert.NotZero(t, perm.ID)

	// Verify permission exists for tenant 1
	found, err := client.BPMNPermission.Query().
		Where(bpmnpermission.ID(perm.ID)).
		Only(ctx)
	assert.NoError(t, err)
	assert.Equal(t, tenant1.ID, found.TenantID)

	// Tenant 2 tries to revoke Tenant 1's permission using tenant context
	tenant2Ctx := context.WithValue(ctx, "bpmn_tenant_id", tenant2.ID)
	err = svc.RevokePermission(tenant2Ctx, &RevokePermissionRequest{
		ResourceType:   "process_definition",
		ResourceID:    100,
		PermissionType: "read",
		PrincipalType: "user",
		PrincipalID:   1,
	})

	// Should fail with tenant isolation error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found or access denied")

	// Verify permission still exists (was not deleted)
	stillExists, err := client.BPMNPermission.Query().
		Where(bpmnpermission.ID(perm.ID)).
		Only(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stillExists)
}

func TestBPMNPermissionService_RevokePermission_SameTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewBPMNPermissionService(client, logger)
	ctx := context.Background()

	// Create tenant
	tenant1, err := client.Tenant.Create().
		SetName("Tenant 1").
		SetCode("TENANT1").
		SetDomain("tenant1.com").
		SetStatus("active").
		Save(ctx)
	assert.NoError(t, err)

	// Tenant 1 creates a permission
	perm, err := client.BPMNPermission.Create().
		SetResourceType("process_definition").
		SetResourceID(100).
		SetPermissionType("read").
		SetPrincipalType("user").
		SetPrincipalID(1).
		SetTenantID(tenant1.ID).
		SetIsGranted(true).
		Save(ctx)
	assert.NoError(t, err)
	assert.NotZero(t, perm.ID)

	// Tenant 1 revokes their own permission
	tenant1Ctx := context.WithValue(ctx, "bpmn_tenant_id", tenant1.ID)
	err = svc.RevokePermission(tenant1Ctx, &RevokePermissionRequest{
		ResourceType:   "process_definition",
		ResourceID:    100,
		PermissionType: "read",
		PrincipalType: "user",
		PrincipalID:   1,
	})

	// Should succeed
	assert.NoError(t, err)

	// Verify permission is deleted
	notExists, err := client.BPMNPermission.Query().
		Where(bpmnpermission.ID(perm.ID)).
		Only(ctx)
	assert.Error(t, err) // ent.ErrNotFound
	assert.Nil(t, notExists)
}