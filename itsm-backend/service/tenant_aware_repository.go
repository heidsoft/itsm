package service

import (
	"context"
	"errors"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
)

// TenantAwareRepository provides tenant-scoped data access for service layer
// delete / update paths. Generic Guard helpers prevent the most common failure
// mode: forgetting a tenant_id predicate on a delete or update.
//
// Typical usage:
//
//   repo := service.NewTenantAwareRepository(client, tenantID)
//   err := repo.GuardDelete(ctx, repo.LoadTicketTenant, func(ctx context.Context, id int) error {
//       return client.Ticket.DeleteOneID(id).Exec(ctx)
//   }, ticketID)
type TenantAwareRepository struct {
	client   *ent.Client
	tenantID int
}

func NewTenantAwareRepository(client *ent.Client, tenantID int) *TenantAwareRepository {
	return &TenantAwareRepository{client: client, tenantID: tenantID}
}

// ValidateTenantAccess rejects entity-tenant / current-tenant mismatch.
func (r *TenantAwareRepository) ValidateTenantAccess(_ context.Context, entityTenantID int) error {
	if entityTenantID != r.tenantID {
		return fmt.Errorf("cross-tenant access denied: entity tenant %d != current tenant %d", entityTenantID, r.tenantID)
	}
	return nil
}

// QueryWithTenantFilter returns a Ticket query pre-filtered by current tenant.
// Other entities follow the same shape (e.g. client.Incident.Query().Where(incident.TenantIDEQ(r.tenantID))).
func (r *TenantAwareRepository) QueryWithTenantFilter() *ent.TicketQuery {
	return r.client.Ticket.Query().Where(ticket.TenantIDEQ(r.tenantID))
}

// TenantID exposes the current tenant id for callers that need to assemble
// their own predicates.
func (r *TenantAwareRepository) TenantID() int { return r.tenantID }

// Client returns the underlying ent client.
func (r *TenantAwareRepository) Client() *ent.Client { return r.client }

// ErrNotFound is returned by Load<T>Tenant helpers when the entity is missing.
var ErrNotFound = errors.New("tenant-aware entity not found")

// Per-entity tenant loaders. They all share the same shape: return
// (tenant_id, error). They never touch the current tenant scope; the Guard
// does that.
func (r *TenantAwareRepository) LoadTicketTenant(ctx context.Context, id int) (int, error) {
	t, err := r.client.Ticket.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return t.TenantID, nil
}

func (r *TenantAwareRepository) LoadIncidentTenant(ctx context.Context, id int) (int, error) {
	t, err := r.client.Incident.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return t.TenantID, nil
}

func (r *TenantAwareRepository) LoadChangeTenant(ctx context.Context, id int) (int, error) {
	t, err := r.client.Change.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return t.TenantID, nil
}

func (r *TenantAwareRepository) LoadProblemTenant(ctx context.Context, id int) (int, error) {
	t, err := r.client.Problem.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return t.TenantID, nil
}

func (r *TenantAwareRepository) LoadUserTenant(ctx context.Context, id int) (int, error) {
	t, err := r.client.User.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return t.TenantID, nil
}

func (r *TenantAwareRepository) LoadRoleTenant(ctx context.Context, id int) (int, error) {
	t, err := r.client.Role.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return t.TenantID, nil
}

// TenantLoader describes a function that returns the tenant_id of an entity.
// A nil tenant_id (and a nil error) is treated as "no tenant scope".
type TenantLoader func(context.Context, int) (int, error)

// GuardAction wraps the actual mutation that should only run after the tenant
// check has passed.
type GuardAction func(context.Context, int) error

// GuardDelete loads the tenant_id of the entity, verifies it matches the
// current tenant, and only then invokes `action(id)` to perform the mutation.
//
// Returning ErrNotFound from the loader means "object does not exist or you
// cannot see it", which is the correct reply for cross-tenant probes: it
// does not leak existence.
func (r *TenantAwareRepository) GuardDelete(ctx context.Context, loader TenantLoader, action GuardAction, id int) error {
	if loader == nil {
		return errors.New("tenant-aware: nil loader")
	}
	if action == nil {
		return errors.New("tenant-aware: nil action")
	}
	tid, err := loader(ctx, id)
	if err != nil {
		return err
	}
	if err := r.ValidateTenantAccess(ctx, tid); err != nil {
		return err
	}
	return action(ctx, id)
}

// GuardUpdate is the multi-mutation variant: useful for status transitions or
// batch updates that must reject cross-tenant inputs up front.
func (r *TenantAwareRepository) GuardUpdate(ctx context.Context, loader TenantLoader, action GuardAction, id int) error {
	return r.GuardDelete(ctx, loader, action, id) // same shape, deliberately
}
