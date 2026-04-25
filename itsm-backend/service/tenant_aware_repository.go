package service

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
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

// Client returns the underlying ent client
func (r *TenantAwareRepository) Client() *ent.Client {
	return r.client
}

// TenantID returns the current tenant ID
func (r *TenantAwareRepository) TenantID() int {
	return r.tenantID
}
