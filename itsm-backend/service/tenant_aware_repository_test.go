package service

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
	"itsm-backend/ent/enttest"
)

func TestTenantAwareRepository_ValidateTenantAccess(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewTenantAwareRepository(client, 100)

	tests := []struct {
		name           string
		entityTenantID int
		wantErr        bool
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

func TestTenantAwareRepository_GuardDelete_RejectsCrossTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent_guard?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant1, _ := client.Tenant.Create().SetName("T1").SetCode("t1").Save(ctx)
	tenant2, _ := client.Tenant.Create().SetName("T2").SetCode("t2").Save(ctx)
	other, _ := client.User.Create().
		SetUsername("u").SetEmail("u@x").SetName("u").
		SetPasswordHash("x").SetTenantID(tenant1.ID).Save(ctx)
	ticket, _ := client.Ticket.Create().
		SetTitle("t").SetTenantID(tenant1.ID).SetRequesterID(other.ID).
		SetTicketNumber("T-1").SetStatus("open").Save(ctx)

	repo := NewTenantAwareRepository(client, tenant2.ID)
	deleted := false
	err := repo.GuardDelete(ctx,
		repo.LoadTicketTenant,
		func(_ context.Context, _ int) error { deleted = true; return nil },
		ticket.ID)
	if err == nil {
		t.Fatalf("expected cross-tenant error, got nil (action ran=%v)", deleted)
	}
	if deleted {
		t.Fatalf("action must not run when guard rejects")
	}

	repoSame := NewTenantAwareRepository(client, tenant1.ID)
	err = repoSame.GuardDelete(ctx,
		repoSame.LoadTicketTenant,
		func(_ context.Context, _ int) error { deleted = true; return nil },
		ticket.ID)
	if err != nil {
		t.Fatalf("same-tenant delete must succeed: %v", err)
	}
	if !deleted {
		t.Fatalf("action must run when guard accepts")
	}
}

func TestTenantAwareRepository_GuardDelete_NotFoundReturnsErrNotFound(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent_nf?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant1, _ := client.Tenant.Create().SetName("T1").SetCode("t1").Save(ctx)
	repo := NewTenantAwareRepository(client, tenant1.ID)
	err := repo.GuardDelete(ctx, repo.LoadTicketTenant,
		func(_ context.Context, _ int) error { return nil }, 999999)
	if err == nil {
		t.Fatalf("expected error for non-existent ticket")
	}
}
