# Multi-Tenant Isolation & MSP Access Control Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix P0 security issues in tenant isolation and MSP access control - ensure all delete/update operations validate tenant_id, and MSP employees can only access allocated customer data.

**Architecture:** Two parallel workstreams: (1) Add tenant_id validation to service layer delete/update operations using TenantAwareRepository pattern, (2) Add MSP customer allocation validation to cross-tenant access across all domain services.

**Tech Stack:** Go/Gin backend, Ent ORM, existing tenant middleware

---

## File Structure

### New Files
- `itsm-backend/service/tenant_aware_repository.go` - Tenant-aware repository pattern
- `itsm-backend/service/msp_access_validator.go` - MSP customer access validation
- `itsm-backend/middleware/msp_access.go` - MSP access validation middleware (split from msp_middleware.go)

### Modified Files
- `itsm-backend/service/role_service.go` - Fix DeleteRole tenant isolation
- `itsm-backend/service/bpmn_permission_service.go` - Fix RevokePermission tenant isolation
- `itsm-backend/service/incident_service.go` - Fix cascade delete tenant validation
- `itsm-backend/service/ticket_service.go` - Fix cascade delete tenant validation
- `itsm-backend/service/change_service.go` - Fix UpdateChangeStatus tenant filter
- `itsm-backend/service/user_service.go` - Fix UpdateUser tenant filtering
- `itsm-backend/service/change_service.go` - Add MSP customer access validation
- `itsm-backend/service/incident_service.go` - Add MSP customer access validation
- `itsm-backend/service/problem_service.go` - Add MSP customer access validation
- `itsm-backend/service/cmdb_service.go` - Add MSP customer CI access control
- `itsm-backend/service/knowledge_service.go` - Add MSP customer knowledge access control
- `itsm-backend/service/ticket_service.go` - Fix GetMSPCustomerReports allocation leak
- `itsm-backend/middleware/msp_middleware.go` - Remove admin bypass, fix MSP context building

---

## Phase 1: Tenant Isolation Fixes

### Task 1: Create TenantAwareRepository Pattern

**Files:**
- Create: `itsm-backend/service/tenant_aware_repository.go`
- Test: `itsm-backend/service/tenant_aware_repository_test.go`

- [ ] **Step 1: Create tenant_aware_repository.go with repository struct**

```go
package service

import (
	"context"
	"fmt"

	"github.com/heimimsb/itsm-backend/ent"
)

// TenantAwareRepository provides tenant-scoped data access
type TenantAwareRepository struct {
	client   *ent.Client
	tenantID int
}

// NewTenantAwareRepository creates a tenant-aware repository
func NewTenantAwareRepository(client *ent.Client, tenantID int) *TenantAwareRepository {
	return &TenantAwareRepository{
		client:   client,
		tenantID: tenantID,
	}
}

// ValidateTenantAccess checks if entity belongs to current tenant
func (r *TenantAwareRepository) ValidateTenantAccess(ctx context.Context, entityTenantID int) error {
	if entityTenantID != r.tenantID {
		return fmt.Errorf("cross-tenant access denied: entity tenant %d != current tenant %d", entityTenantID, r.tenantID)
	}
	return nil
}

// QueryWithTenantFilter returns query filtered by tenant_id
func (r *TenantAwareRepository) QueryWithTenantFilter() *ent.TicketQuery {
	return r.client.Ticket.Query().Where(ticket.TenantIDEQ(r.tenantID))
}

// DeleteWithTenantFilter deletes entity only if it belongs to current tenant
func (r *TenantAwareRepository) DeleteWithTenantFilter(ctx context.Context, query *ent.DeleteFromQuery) (int, error) {
	// Note: Ent doesn't support adding Where after Delete, so we validate before delete
	// For services that need this pattern, use ValidateTenantAccess before executing delete
	return query.Exec(ctx)
}
```

- [ ] **Step 2: Create test file tenant_aware_repository_test.go**

```go
package service

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/heimimsb/itsm-backend/ent/testutils"
	"github.com/stretchr/testify/assert"
)

func TestTenantAwareRepository_ValidateTenantAccess(t *testing.T) {
	client := testutils.NewTestClient(t, dialect.SQLite)
	repo := NewTenantAwareRepository(client, 100)

	tests := []struct {
		name          string
		entityTenantID int
		wantErr       bool
	}{
		{"same tenant", 100, false},
		{"different tenant", 200, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.ValidateTenantAccess(context.Background(), tt.entityTenantID)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cross-tenant access denied")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

- [ ] **Step 3: Verify tests pass**

Run: `cd itsm-backend && go test -v ./service/tenant_aware_repository_test.go -run TestTenantAwareRepository`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
cd /Users/heidsoft/Downloads/research/itsm
git add itsm-backend/service/tenant_aware_repository.go itsm-backend/service/tenant_aware_repository_test.go
git commit -m "feat(tenant): add TenantAwareRepository pattern for tenant-scoped access"
```

---

### Task 2: Fix RoleService.DeleteRole() Missing Tenant Filter

**Files:**
- Modify: `itsm-backend/service/role_service.go:165-180`

- [ ] **Step 1: Read current implementation**

Run: `cat -n itsm-backend/service/role_service.go | sed -n '165,180p'`

- [ ] **Step 2: Write test for tenant isolation on DeleteRole**

```go
func TestRoleService_DeleteRole_TenantIsolation(t *testing.T) {
	// Setup: Create two tenants with roles
	client := testutils.NewTestClient(t, dialect.SQLite)
	svc := NewRoleService(client)
	ctx := context.Background()

	// Tenant 1 creates a role
	tenant1Ctx := context.WithValue(ctx, "tenant_id", 1)
	role1, _ := client.Role.Create().SetName("Tenant1Role").SetCode("tenant1_role").SetTenantID(1).Save(tenant1Ctx)

	// Tenant 2 tries to delete Tenant 1's role
	tenant2Ctx := context.WithValue(ctx, "tenant_id", 2)
	err := svc.DeleteRole(tenant2Ctx, role1.ID)

	// Should fail with cross-tenant access error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cross-tenant access denied")
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd itsm-backend && go test -v ./service/role_service_test.go -run TestRoleService_DeleteRole_TenantIsolation`

Expected: FAIL

- [ ] **Step 4: Modify DeleteRole to add tenant filter**

```go
func (s *RoleService) DeleteRole(ctx context.Context, id int) error {
	// Get tenant_id from context
	tenantID, ok := ctx.Value("tenant_id").(int)
	if !ok {
		return fmt.Errorf("tenant_id not found in context")
	}

	// Verify role exists and belongs to current tenant
	role, err := s.client.Role.Query().
		Where(role.IDEQ(id), role.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// Cannot delete system roles
	if role.IsSystem {
		return fmt.Errorf("cannot delete system role")
	}

	// Delete with tenant filter
	_, err = s.client.Role.Delete().
		Where(role.IDEQ(id), role.TenantID(tenantID)).
		Exec(ctx)
	return err
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd itsm-backend && go test -v ./service/role_service_test.go -run TestRoleService_DeleteRole_TenantIsolation`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add itsm-backend/service/role_service.go
git commit -m "fix(tenant): add tenant filter to DeleteRole to prevent cross-tenant deletion"
```

---

### Task 3: Fix BPMNPermissionService.RevokePermission() Missing Tenant Filter

**Files:**
- Modify: `itsm-backend/service/bpmn_permission_service.go:90-110`

- [ ] **Step 1: Read current implementation**

Run: `cat -n itsm-backend/service/bpmn_permission_service.go | sed -n '90,110p'`

- [ ] **Step 2: Add tenant_id to RevokePermission request**

First check if `bpmn_permission.go` has TenantID field:
Run: `grep -n "TenantID\|tenant_id" itsm-backend/ent/schema/bpmn_permission.go`

- [ ] **Step 3: Write test for tenant isolation on RevokePermission**

```go
func TestBPMNPermissionService_RevokePermission_TenantIsolation(t *testing.T) {
	client := testutils.NewTestClient(t, dialect.SQLite)
	svc := NewBPMNPermissionService(client)
	ctx := context.Background()

	// Create permission for tenant 1
	tenant1Ctx := context.WithValue(ctx, "tenant_id", 1)
	perm, _ := client.BPMNPermission.Create().
		SetResourceType("change").
		SetResourceID(100).
		SetTenantID(1).
		Save(tenant1Ctx)

	// Tenant 2 tries to revoke
	tenant2Ctx := context.WithValue(ctx, "tenant_id", 2)
	err := svc.RevokePermission(tenant2Ctx, &RevokePermissionRequest{
		ResourceType: "change",
		ResourceID:   100,
		PrincipalType: "user",
		PrincipalID:   1,
	})

	// Should fail
	assert.Error(t, err)
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `cd itsm-backend && go test -v ./service/bpmn_permission_service_test.go -run TestBPMNPermissionService_RevokePermission_TenantIsolation`

Expected: FAIL

- [ ] **Step 5: Modify RevokePermission to add tenant filter**

```go
func (s *BPMNPermissionService) RevokePermission(ctx context.Context, req *RevokePermissionRequest) error {
	tenantID, ok := ctx.Value("tenant_id").(int)
	if !ok {
		return fmt.Errorf("tenant_id not found in context")
	}

	// Build query with tenant filter
	query := s.client.BPMNPermission.Delete().
		Where(bpmnpermission.ResourceType(req.ResourceType)).
		Where(bpmnpermission.ResourceID(req.ResourceID)).
		Where(bpmnpermission.PrincipalType(req.PrincipalType)).
		Where(bpmnpermission.PrincipalID(req.PrincipalID)).
		Where(bpmnpermission.TenantID(tenantID)) // Add tenant filter

	_, err := query.Exec(ctx)
	return err
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `cd itsm-backend && go test -v ./service/bpmn_permission_service_test.go -run TestBPMNPermissionService_RevokePermission_TenantIsolation`

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add itsm-backend/service/bpmn_permission_service.go
git commit -m "fix(tenant): add tenant filter to RevokePermission to prevent cross-tenant revocation"
```

---

### Task 4: Fix IncidentService.DeleteIncident() Cascade Delete Tenant Validation

**Files:**
- Modify: `itsm-backend/service/incident_service.go:330-360`

- [ ] **Step 1: Read current implementation**

Run: `cat -n itsm-backend/service/incident_service.go | sed -n '320,360p'`

- [ ] **Step 2: Write test for tenant isolation on DeleteIncident cascade**

```go
func TestIncidentService_DeleteIncident_CascadeTenantIsolation(t *testing.T) {
	client := testutils.NewTestClient(t, dialect.SQLite)
	svc := NewIncidentService(client, nil, nil, nil)
	ctx := context.Background()

	// Create incident for tenant 1 with events
	tenant1Ctx := context.WithValue(ctx, "tenant_id", 1)
	incident, _ := client.Incident.Create().SetTitle("Test").SetTenantID(1).Save(tenant1Ctx)
	client.IncidentEvent.Create().SetIncidentID(incident.ID).SetContent("event").SetTenantID(1).Save(tenant1Ctx)

	// Tenant 2 tries to delete - should fail
	tenant2Ctx := context.WithValue(ctx, "tenant_id", 2)
	err := svc.DeleteIncident(tenant2Ctx, incident.ID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cross-tenant access denied")
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd itsm-backend && go test -v ./service/incident_service_test.go -run TestIncidentService_DeleteIncident_CascadeTenantIsolation`

Expected: FAIL

- [ ] **Step 4: Modify DeleteIncident to validate tenant before cascade**

```go
func (s *IncidentService) DeleteIncident(ctx context.Context, id int) error {
	tenantID, ok := ctx.Value("tenant_id").(int)
	if !ok {
		return fmt.Errorf("tenant_id not found in context")
	}

	// Verify incident belongs to current tenant
	incident, err := s.client.Incident.Query().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("incident not found: %w", err)
	}

	// Delete cascade - add tenant filter to prevent cross-tenant cascade
	_, err = s.client.IncidentEvent.Delete().
		Where(incidentevent.IncidentIDEQ(id), incidentevent.TenantIDEQ(tenantID)).
		Exec(ctx)
	// ... similar for other cascade entities

	// Delete incident with tenant filter
	_, err = s.client.Incident.Delete().
		Where(incident.IDEQ(id), incident.TenantIDEQ(tenantID)).
		Exec(ctx)
	return err
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd itsm-backend && go test -v ./service/incident_service_test.go -run TestIncidentService_DeleteIncident_CascadeTenantIsolation`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add itsm-backend/service/incident_service.go
git commit -m "fix(tenant): add tenant validation to DeleteIncident cascade deletes"
```

---

### Task 5: Fix TicketService Cascade Delete Tenant Validation

**Files:**
- Modify: `itsm-backend/service/ticket_service.go:740-800`

- [ ] **Step 1: Read current implementation**

Run: `cat -n itsm-backend/service/ticket_service.go | sed -n '740,800p'`

- [ ] **Step 2: Write test for tenant isolation on DeleteTicket cascade**

```go
func TestTicketService_DeleteTicket_CascadeTenantIsolation(t *testing.T) {
	client := testutils.NewTestClient(t, dialect.SQLite)
	svc := NewTicketService(client, nil, nil, nil)
	ctx := context.Background()

	// Create ticket for tenant 1 with comments
	tenant1Ctx := context.WithValue(ctx, "tenant_id", 1)
	ticket, _ := client.Ticket.Create().SetTitle("Test").SetTenantID(1).Save(tenant1Ctx)
	client.TicketComment.Create().SetTicketID(ticket.ID).SetContent("comment").SetTenantID(1).Save(tenant1Ctx)

	// Tenant 2 tries to delete - should fail
	tenant2Ctx := context.WithValue(ctx, "tenant_id", 2)
	err := svc.DeleteTicket(tenant2Ctx, ticket.ID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cross-tenant access denied")
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd itsm-backend && go test -v ./service/ticket_service_test.go -run TestTicketService_DeleteTicket_CascadeTenantIsolation`

Expected: FAIL

- [ ] **Step 4: Modify DeleteTicket to validate tenant before cascade**

```go
func (s *TicketService) DeleteTicket(ctx context.Context, ticketID int) error {
	tenantID, ok := ctx.Value("tenant_id").(int)
	if !ok {
		return fmt.Errorf("tenant_id not found in context")
	}

	// Verify ticket belongs to current tenant
	ticket, err := s.client.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("ticket not found: %w", err)
	}

	// Cascade delete with tenant filters
	entities := []interface{
		Where(...Query predicates) *ent.DeleteFromQuery
	}{
		s.client.TicketComment.Delete().Where(ticketcomment.TicketIDEQ(ticketID), ticketcomment.TenantIDEQ(tenantID)),
		s.client.TicketAttachment.Delete().Where(ticketattachment.TicketIDEQ(ticketID), ticketattachment.TenantIDEQ(tenantID)),
		// ... other cascade entities with tenant filters
	}

	for _, entity := range entities {
		if _, err := entity.Exec(ctx); err != nil {
			return err
		}
	}

	// Delete ticket with tenant filter
	_, err = s.client.Ticket.Delete().
		Where(ticket.IDEQ(ticketID), ticket.TenantIDEQ(tenantID)).
		Exec(ctx)
	return err
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd itsm-backend && go test -v ./service/ticket_service_test.go -run TestTicketService_DeleteTicket_CascadeTenantIsolation`

Expected: PASS

- [ ] **Step 6: Fix BatchDeleteTickets similarly**

Read: `cat -n itsm-backend/service/ticket_service.go | sed -n '820,840p'`

Apply same tenant validation pattern.

- [ ] **Step 7: Commit**

```bash
git add itsm-backend/service/ticket_service.go
git commit -m "fix(tenant): add tenant validation to DeleteTicket and BatchDeleteTickets cascade deletes"
```

---

### Task 6: Fix ChangeService.UpdateChangeStatus() Tenant Filter

**Files:**
- Modify: `itsm-backend/service/change_service.go:505-520`

- [ ] **Step 1: Read current implementation**

Run: `cat -n itsm-backend/service/change_service.go | sed -n '505,525p'`

- [ ] **Step 2: Write test for tenant isolation on UpdateChangeStatus**

```go
func TestChangeService_UpdateChangeStatus_TenantIsolation(t *testing.T) {
	client := testutils.NewTestClient(t, dialect.SQLite)
	svc := NewChangeService(client, nil, nil)
	ctx := context.Background()

	// Create change for tenant 1
	tenant1Ctx := context.WithValue(ctx, "tenant_id", 1)
	change, _ := client.Change.Create().SetTitle("Test").SetTenantID(1).Save(tenant1Ctx)

	// Tenant 2 tries to update status - should fail
	tenant2Ctx := context.WithValue(ctx, "tenant_id", 2)
	err := svc.UpdateChangeStatus(tenant2Ctx, change.ID, "approved")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cross-tenant access denied")
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd itsm-backend && go test -v ./service/change_service_test.go -run TestChangeService_UpdateChangeStatus_TenantIsolation`

Expected: FAIL

- [ ] **Step 4: Modify UpdateChangeStatus to use tenant-filtered query**

```go
func (s *ChangeService) UpdateChangeStatus(ctx context.Context, id int, status string) error {
	tenantID, ok := ctx.Value("tenant_id").(int)
	if !ok {
		return fmt.Errorf("tenant_id not found in context")
	}

	// Verify change belongs to current tenant before update
	_, err := s.client.Change.Query().
		Where(change.IDEQ(id), change.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("change not found: %w", err)
	}

	// Update with tenant filter using Where clause
	result, err := s.client.Change.Update().
		Where(change.IDEQ(id), change.TenantIDEQ(tenantID)).
		SetStatus(status).
		Save(ctx)

	if result == 0 {
		return fmt.Errorf("change not found or access denied")
	}
	return err
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd itsm-backend && go test -v ./service/change_service_test.go -run TestChangeService_UpdateChangeStatus_TenantIsolation`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add itsm-backend/service/change_service.go
git commit -m "fix(tenant): add tenant filter to UpdateChangeStatus to prevent cross-tenant updates"
```

---

### Task 7: Fix UserService.UpdateUser() Tenant Filtering

**Files:**
- Modify: `itsm-backend/service/user_service.go:180-200`

- [ ] **Step 1: Read current implementation**

Run: `cat -n itsm-backend/service/user_service.go | sed -n '180,205p'`

- [ ] **Step 2: Write test for tenant isolation on UpdateUser**

```go
func TestUserService_UpdateUser_TenantIsolation(t *testing.T) {
	client := testutils.NewTestClient(t, dialect.SQLite)
	svc := NewUserService(client, nil)
	ctx := context.Background()

	// Create user for tenant 1
	tenant1Ctx := context.WithValue(ctx, "tenant_id", 1)
	user, _ := client.User.Create().SetUsername("testuser").SetTenantID(1).Save(tenant1Ctx)

	// Tenant 2 tries to update - should fail
	tenant2Ctx := context.WithValue(ctx, "tenant_id", 2)
	err := svc.UpdateUser(tenant2Ctx, user.ID, &UpdateUserRequest{DisplayName: "Hacked"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cross-tenant access denied")
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd itsm-backend && go test -v ./service/user_service_test.go -run TestUserService_UpdateUser_TenantIsolation`

Expected: FAIL

- [ ] **Step 4: Modify UpdateUser to add tenant filtering**

```go
func (s *UserService) UpdateUser(ctx context.Context, id int, req *UpdateUserRequest) error {
	tenantID, ok := ctx.Value("tenant_id").(int)
	if !ok {
		return fmt.Errorf("tenant_id not found in context")
	}

	// Verify user belongs to current tenant
	_, err := s.client.User.Query().
		Where(user.IDEQ(id), user.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Update with tenant filter
	_, err = s.client.User.UpdateOneID(id).
		Where(user.TenantID(tenantID)).
		SetDisplayName(req.DisplayName).
		// ... other fields
		Save(ctx)

	return err
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd itsm-backend && go test -v ./service/user_service_test.go -run TestUserService_UpdateUser_TenantIsolation`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add itsm-backend/service/user_service.go
git commit -m "fix(tenant): add tenant filter to UpdateUser to prevent cross-tenant updates"
```

---

## Phase 2: MSP Access Control Fixes

### Task 8: Create MSP Access Validator Service

**Files:**
- Create: `itsm-backend/service/msp_access_validator.go`
- Test: `itsm-backend/service/msp_access_validator_test.go`

- [ ] **Step 1: Create msp_access_validator.go**

```go
package service

import (
	"context"
	"fmt"

	"github.com/heimimsb/itsm-backend/ent"
	"github.com/heimimsb/itsm-backend/ent/mspallocation"
)

// MSPAccessValidator validates MSP user access to customer data
type MSPAccessValidator struct {
	client *ent.Client
}

// NewMSPAccessValidator creates a new MSP access validator
func NewMSPAccessValidator(client *ent.Client) *MSPAccessValidator {
	return &MSPAccessValidator{client: client}
}

// ValidateCustomerAccess verifies MSP user can access the specified customer tenant
func (v *MSPAccessValidator) ValidateCustomerAccess(ctx context.Context, mspUserID, customerTenantID int) error {
	// Check if MSP user has active allocation to this customer
	allocation, err := v.client.MSPAllocation.Query().
		Where(mspallocation.MSPUserIDEQ(mspUserID)).
		Where(mspallocation.CustomerTenantIDEQ(customerTenantID)).
		Where(mspallocation.DeassignedAtIsNil()). // Active allocation only
		Only(ctx)

	if err != nil {
		return fmt.Errorf("access denied: no active allocation for customer tenant %d", customerTenantID)
	}

	if allocation == nil {
		return fmt.Errorf("access denied: no active allocation found")
	}

	return nil
}

// GetAllowedCustomerIDs returns list of customer tenant IDs the MSP user can access
func (v *MSPAccessValidator) GetAllowedCustomerIDs(ctx context.Context, mspUserID int) ([]int, error) {
	allocations, err := v.client.MSPAllocation.Query().
		Where(mspallocation.MSPUserIDEQ(mspUserID)).
		Where(mspallocation.DeassignedAtIsNil()).
		All(ctx)

	if err != nil {
		return nil, err
	}

	customerIDs := make([]int, len(allocations))
	for i, a := range allocations {
		customerIDs[i] = a.CustomerTenantID
	}
	return customerIDs, nil
}

// FilterByMSPAllocation filters a list of tenant IDs to only allowed customers
func (v *MSPAccessValidator) FilterByMSPAllocation(ctx context.Context, mspUserID int, tenantIDs []int) ([]int, error) {
	allowed, err := v.GetAllowedCustomerIDs(ctx, mspUserID)
	if err != nil {
		return nil, err
	}

	allowedSet := make(map[int]bool)
	for _, id := range allowed {
		allowedSet[id] = true
	}

	filtered := make([]int, 0)
	for _, id := range tenantIDs {
		if allowedSet[id] {
			filtered = append(filtered, id)
		}
	}
	return filtered, nil
}
```

- [ ] **Step 2: Create test file msp_access_validator_test.go**

```go
package service

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/heimimsb/itsm-backend/ent/testutils"
	"github.com/stretchr/testify/assert"
)

func TestMSPAccessValidator_ValidateCustomerAccess(t *testing.T) {
	client := testutils.NewTestClient(t, dialect.SQLite)
	validator := NewMSPAccessValidator(client)
	ctx := context.Background()

	// Setup: Create MSP tenant, customer tenant, and allocation
	mspTenant, _ := client.Tenant.Create().SetName("MSP").SetCode("msp").SetType("msp").Save(ctx)
	customerTenant, _ := client.Tenant.Create().SetName("Customer").SetCode("cust1").SetType("customer").Save(ctx)
	mspUser, _ := client.User.Create().SetUsername("msp_user").SetTenantID(mspTenant.ID).Save(ctx)

	// Create active allocation
	client.MSPAllocation.Create().
		SetMSPUserID(mspUser.ID).
		SetCustomerTenantID(customerTenant.ID).
		SetRole("provider_agent").
		Save(ctx)

	// Test: Valid access
	err := validator.ValidateCustomerAccess(ctx, mspUser.ID, customerTenant.ID)
	assert.NoError(t, err)

	// Test: Access to non-allocated customer should fail
	unallocatedTenant, _ := client.Tenant.Create().SetName("Other").SetCode("other").SetType("customer").Save(ctx)
	err = validator.ValidateCustomerAccess(ctx, mspUser.ID, unallocatedTenant.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `cd itsm-backend && go test -v ./service/msp_access_validator_test.go -run TestMSPAccessValidator`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add itsm-backend/service/msp_access_validator.go itsm-backend/service/msp_access_validator_test.go
git commit -m "feat(msp): add MSPAccessValidator for customer access validation"
```

---

### Task 9: Fix Admin Bypass in MSP Middleware

**Files:**
- Modify: `itsm-backend/middleware/msp_middleware.go:110-125`

- [ ] **Step 1: Read current implementation**

Run: `cat -n itsm-backend/middleware/msp_middleware.go | sed -n '110,130p'`

- [ ] **Step 2: Write test for admin bypass security issue**

```go
func TestMSPMiddleware_AdminBypassFix(t *testing.T) {
	// Admin user from standard tenant should NOT get MSP access
	// unless they are actually MSP employee with allocation
}
```

- [ ] **Step 3: Modify to remove admin bypass, require actual MSP allocation**

```go
// In GetMSPContext function, replace the admin bypass logic:
// BEFORE (problematic):
// if role == "super_admin" || role == "sysadmin" || role == "admin" {
//     isMSP = true
//     mspRole = "msp_manager"
// }

// AFTER (secure):
// Admin users from non-MSP tenants should NOT get automatic MSP access
// MSP access should require:
// 1. User belongs to MSP tenant (tenant type = msp)
// 2. User has msp_role field set
// 3. User has active allocation to the target customer

// Remove the admin bypass entirely - admins can still have their own privileges
// but MSP customer access requires proper allocation validation
```

- [ ] **Step 4: Verify build passes**

Run: `cd itsm-backend && go build ./...`

Expected: SUCCESS

- [ ] **Step 5: Commit**

```bash
git add itsm-backend/middleware/msp_middleware.go
git commit -m "fix(msp): remove admin bypass in MSP context - MSP access requires allocation"
```

---

### Task 10: Fix GetMSPCustomerReports Allocation Leak

**Files:**
- Modify: `itsm-backend/service/ticket_service.go:1900-1920`

- [ ] **Step 1: Read current implementation**

Run: `cat -n itsm-backend/service/ticket_service.go | sed -n '1900,1925p'`

- [ ] **Step 2: Write test for allocation-aware customer reports**

```go
func TestTicketService_GetMSPCustomerReports_AllocationAware(t *testing.T) {
	// Test that when customerTenantID is nil, only allocated customers are returned
}
```

- [ ] **Step 3: Modify to use MSPAccessValidator**

```go
func (s *TicketService) GetMSPCustomerReports(ctx context.Context, mspUserID int, customerTenantID *int) ([]*CustomerReport, error) {
	// If specific customer requested, validate allocation first
	if customerTenantID != nil {
		err := s.mspValidator.ValidateCustomerAccess(ctx, mspUserID, *customerTenantID)
		if err != nil {
			return nil, err
		}
		return s.getReportForCustomer(ctx, *customerTenantID)
	}

	// If no specific customer, get ALL allocated customers
	allowedIDs, err := s.mspValidator.GetAllowedCustomerIDs(ctx, mspUserID)
	if err != nil {
		return nil, err
	}

	// Only query customers in allowed list, not ALL customer tenants
	reports := make([]*CustomerReport, 0)
	for _, custID := range allowedIDs {
		report, err := s.getReportForCustomer(ctx, custID)
		if err != nil {
			continue // Skip customers we can't access
		}
		reports = append(reports, report)
	}
	return reports, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd itsm-backend && go test -v ./service/ticket_service_test.go -run TestTicketService_GetMSPCustomerReports`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add itsm-backend/service/ticket_service.go
git commit -m "fix(msp): fix GetMSPCustomerReports to only return allocated customer data"
```

---

### Task 11: Add MSP Customer Validation to Domain Services

**Files:**
- Modify: `itsm-backend/service/change_service.go`
- Modify: `itsm-backend/service/incident_service.go`
- Modify: `itsm-backend/service/problem_service.go`
- Modify: `itsm-backend/service/cmdb_service.go`
- Modify: `itsm-backend/service/knowledge_service.go`

- [ ] **Step 1: Create MSP-aware wrapper for tenant context**

In `tenant_isolation.go`, add:

```go
// GetEffectiveTenantID returns the tenant ID to use for queries
// For MSP users accessing customer data, returns customer tenant ID
// For regular users, returns their own tenant ID
func (h *TenantIsolationHelper) GetEffectiveTenantID(ctx context.Context) int {
	mspCtx := GetMSPContext(ctx)
	if mspCtx != nil && mspCtx.IsMSPUser && mspCtx.CustomerTenantID > 0 {
		return mspCtx.CustomerTenantID // MSP accessing customer data
	}
	return h.tenantID // Regular user
}

// RequireMSPCustomerAccess validates MSP user can access the target customer
func (h *TenantIsolationHelper) RequireMSPCustomerAccess(ctx context.Context, mspUserID, customerTenantID int) error {
	if mspCtx := GetMSPContext(ctx); mspCtx != nil && mspCtx.IsMSPUser {
		// MSP user accessing customer data - validate allocation
		return h.mspValidator.ValidateCustomerAccess(ctx, mspUserID, customerTenantID)
	}
	return nil
}
```

- [ ] **Step 2: Add MSP validation to ChangeService methods**

```go
func (s *ChangeService) GetChange(ctx context.Context, id int) (*Change, error) {
	tenantID := s.getEffectiveTenantID(ctx) // Handles MSP context

	change, err := s.client.Change.Query().
		Where(change.IDEQ(id), change.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	// If MSP user accessing customer data, validate allocation
	if mspCtx := GetMSPContext(ctx); mspCtx != nil && mspCtx.IsMSPUser {
		err := s.mspValidator.ValidateCustomerAccess(ctx, mspCtx.MSPUserID, tenantID)
		if err != nil {
			return nil, err
		}
	}

	return change, nil
}
```

Apply same pattern to: `UpdateChange`, `DeleteChange`, `ListChanges`

- [ ] **Step 3: Add MSP validation to IncidentService methods**

Apply same pattern to: `GetIncident`, `UpdateIncident`, `DeleteIncident`, `ListIncidents`

- [ ] **Step 4: Add MSP validation to ProblemService methods**

Apply same pattern to: `GetProblem`, `UpdateProblem`, `DeleteProblem`, `ListProblems`

- [ ] **Step 5: Add MSP validation to CMDBService methods**

Apply same pattern to CI access methods.

- [ ] **Step 6: Add MSP validation to KnowledgeService methods**

Apply same pattern to knowledge base access methods.

- [ ] **Step 7: Run tests to verify all pass**

Run: `cd itsm-backend && go test ./service/... -v -run "MSP|Tenant"`

Expected: PASS

- [ ] **Step 8: Commit each service modification**

```bash
git add itsm-backend/service/change_service.go itsm-backend/service/incident_service.go
git commit -m "fix(msp): add customer allocation validation to change and incident services"

git add itsm-backend/service/problem_service.go itsm-backend/service/cmdb_service.go
git commit -m "fix(msp): add customer allocation validation to problem and cmdb services"

git add itsm-backend/service/knowledge_service.go
git commit -m "fix(msp): add customer allocation validation to knowledge service"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | TenantAwareRepository pattern | new files |
| 2 | Fix RoleService.DeleteRole | role_service.go |
| 3 | Fix BPMNPermissionService.RevokePermission | bpmn_permission_service.go |
| 4 | Fix IncidentService.DeleteIncident cascade | incident_service.go |
| 5 | Fix TicketService.DeleteTicket cascade | ticket_service.go |
| 6 | Fix ChangeService.UpdateChangeStatus | change_service.go |
| 7 | Fix UserService.UpdateUser | user_service.go |
| 8 | MSPAccessValidator service | new files |
| 9 | Fix admin bypass in MSP middleware | msp_middleware.go |
| 10 | Fix GetMSPCustomerReports leak | ticket_service.go |
| 11 | Add MSP validation to domain services | change/incident/problem/cmdb/knowledge services |

---

## Plan complete and saved to `docs/superpowers/plans/2026-04-25-tenant-isolation-msp-access.md`

**Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**