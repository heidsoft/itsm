package service

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/mspallocation"
	"itsm-backend/ent/tenant"
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
	// An active allocation means: msp_user_id matches AND deassigned_at is nil AND customer_tenant matches
	allocations, err := v.client.MSPAllocation.Query().
		Where(mspallocation.MspUserIDEQ(mspUserID)).
		Where(mspallocation.DeassignedAtIsNil()).
		Where(mspallocation.HasCustomerTenantWith(tenant.IDEQ(customerTenantID))).
		All(ctx)
	if err != nil {
		return fmt.Errorf("access denied: failed to query allocations: %w", err)
	}

	if len(allocations) == 0 {
		return fmt.Errorf("access denied: no active allocation for customer tenant %d", customerTenantID)
	}

	return nil
}

// GetAllowedCustomerIDs returns list of customer tenant IDs the MSP user can access
func (v *MSPAccessValidator) GetAllowedCustomerIDs(ctx context.Context, mspUserID int) ([]int, error) {
	allocations, err := v.client.MSPAllocation.Query().
		Where(mspallocation.MspUserIDEQ(mspUserID)).
		Where(mspallocation.DeassignedAtIsNil()).
		WithCustomerTenant().
		All(ctx)
	if err != nil {
		return nil, err
	}

	customerIDs := make([]int, 0)
	seen := make(map[int]bool)
	for _, a := range allocations {
		if a.Edges.CustomerTenant != nil && !seen[a.Edges.CustomerTenant.ID] {
			customerIDs = append(customerIDs, a.Edges.CustomerTenant.ID)
			seen[a.Edges.CustomerTenant.ID] = true
		}
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
