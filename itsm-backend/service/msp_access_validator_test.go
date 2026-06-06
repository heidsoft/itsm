package service

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"itsm-backend/ent/enttest"
)

func TestMSPAccessValidator_ValidateCustomerAccess(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	validator := NewMSPAccessValidator(client)
	ctx := context.Background()

	// Setup: Create MSP tenant, customer tenant, and allocation
	mspTenant, err := client.Tenant.Create().
		SetName("MSP").
		SetCode("msp").
		SetType("msp").
		Save(ctx)
	assert.NoError(t, err)

	customerTenant, err := client.Tenant.Create().
		SetName("Customer").
		SetCode("cust1").
		SetType("customer").
		Save(ctx)
	assert.NoError(t, err)

	mspUser, err := client.User.Create().
		SetUsername("msp_user").
		SetEmail("msp@example.com").
		SetName("MSP User").
		SetPasswordHash("hash").
		SetTenantID(mspTenant.ID).
		Save(ctx)
	assert.NoError(t, err)

	// Create active allocation
	_, err = client.MSPAllocation.Create().
		SetMspUserID(mspUser.ID).
		SetCustomerTenantID(customerTenant.ID).
		SetRole("provider_agent").
		Save(ctx)
	assert.NoError(t, err)

	// Test: Valid access - should succeed
	err = validator.ValidateCustomerAccess(ctx, mspUser.ID, customerTenant.ID)
	assert.NoError(t, err)

	// Test: Access to non-allocated customer should fail
	unallocatedTenant, err := client.Tenant.Create().
		SetName("Other").
		SetCode("other").
		SetType("customer").
		Save(ctx)
	assert.NoError(t, err)

	err = validator.ValidateCustomerAccess(ctx, mspUser.ID, unallocatedTenant.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestMSPAccessValidator_GetAllowedCustomerIDs(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	validator := NewMSPAccessValidator(client)
	ctx := context.Background()

	// Setup: Create MSP tenant and multiple customer tenants
	mspTenant, _ := client.Tenant.Create().
		SetName("MSP").
		SetCode("msp").
		SetType("msp").
		Save(ctx)

	customerTenant1, _ := client.Tenant.Create().
		SetName("Customer1").
		SetCode("cust1").
		SetType("customer").
		Save(ctx)

	customerTenant2, _ := client.Tenant.Create().
		SetName("Customer2").
		SetCode("cust2").
		SetType("customer").
		Save(ctx)

	mspUser, _ := client.User.Create().
		SetUsername("msp_user").
		SetEmail("msp@example.com").
		SetName("MSP User").
		SetPasswordHash("hash").
		SetTenantID(mspTenant.ID).
		Save(ctx)

	// Create allocations to two customers
	client.MSPAllocation.Create().
		SetMspUserID(mspUser.ID).
		SetCustomerTenantID(customerTenant1.ID).
		SetRole("provider_agent").
		Save(ctx)
	client.MSPAllocation.Create().
		SetMspUserID(mspUser.ID).
		SetCustomerTenantID(customerTenant2.ID).
		SetRole("provider_agent").
		Save(ctx)

	// Test: Get allowed customer IDs
	allowed, err := validator.GetAllowedCustomerIDs(ctx, mspUser.ID)
	assert.NoError(t, err)
	assert.Len(t, allowed, 2)
	assert.Contains(t, allowed, customerTenant1.ID)
	assert.Contains(t, allowed, customerTenant2.ID)
}

func TestMSPAccessValidator_FilterByMSPAllocation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	validator := NewMSPAccessValidator(client)
	ctx := context.Background()

	// Setup: Create MSP tenant and customer tenants
	mspTenant, _ := client.Tenant.Create().
		SetName("MSP").
		SetCode("msp").
		SetType("msp").
		Save(ctx)

	allowedTenant1, _ := client.Tenant.Create().
		SetName("Allowed1").
		SetCode("allowed1").
		SetType("customer").
		Save(ctx)

	allowedTenant2, _ := client.Tenant.Create().
		SetName("Allowed2").
		SetCode("allowed2").
		SetType("customer").
		Save(ctx)

	notAllowedTenant, _ := client.Tenant.Create().
		SetName("NotAllowed").
		SetCode("notallowed").
		SetType("customer").
		Save(ctx)

	mspUser, _ := client.User.Create().
		SetUsername("msp_user").
		SetEmail("msp@example.com").
		SetName("MSP User").
		SetPasswordHash("hash").
		SetTenantID(mspTenant.ID).
		Save(ctx)

	// Create allocation only to allowedTenant1 and allowedTenant2
	client.MSPAllocation.Create().
		SetMspUserID(mspUser.ID).
		SetCustomerTenantID(allowedTenant1.ID).
		SetRole("provider_agent").
		Save(ctx)
	client.MSPAllocation.Create().
		SetMspUserID(mspUser.ID).
		SetCustomerTenantID(allowedTenant2.ID).
		SetRole("provider_agent").
		Save(ctx)

	// Test: Filter tenant list
	tenantIDs := []int{allowedTenant1.ID, allowedTenant2.ID, notAllowedTenant.ID}
	filtered, err := validator.FilterByMSPAllocation(ctx, mspUser.ID, tenantIDs)
	assert.NoError(t, err)
	assert.Len(t, filtered, 2)
	assert.Contains(t, filtered, allowedTenant1.ID)
	assert.Contains(t, filtered, allowedTenant2.ID)
	assert.NotContains(t, filtered, notAllowedTenant.ID)
}

func TestMSPAccessValidator_DeassignedAllocationNoLongerValid(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	validator := NewMSPAccessValidator(client)
	ctx := context.Background()

	// Setup
	mspTenant, _ := client.Tenant.Create().
		SetName("MSP").
		SetCode("msp").
		SetType("msp").
		Save(ctx)

	customerTenant, _ := client.Tenant.Create().
		SetName("Customer").
		SetCode("cust1").
		SetType("customer").
		Save(ctx)

	mspUser, _ := client.User.Create().
		SetUsername("msp_user").
		SetEmail("msp@example.com").
		SetName("MSP User").
		SetPasswordHash("hash").
		SetTenantID(mspTenant.ID).
		Save(ctx)

	// Create allocation then deassign it
	allocation, err := client.MSPAllocation.Create().
		SetMspUserID(mspUser.ID).
		SetCustomerTenantID(customerTenant.ID).
		SetRole("provider_agent").
		Save(ctx)
	assert.NoError(t, err)

	// Deassign the allocation
	_, err = allocation.Update().
		SetDeassignedAt(time.Now()).
		Save(ctx)
	assert.NoError(t, err)

	// Test: Access should be denied after deassignment
	err = validator.ValidateCustomerAccess(ctx, mspUser.ID, customerTenant.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}
