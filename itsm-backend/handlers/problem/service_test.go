package problem

import (
	"context"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupProblemHandlerTest(t *testing.T) (*ent.Client, *Service, context.Context) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:problem-handler-%s?mode=memory&cache=shared&_fk=1", t.Name()))
	repo := NewEntRepository(client)
	return client, NewService(repo, zaptest.NewLogger(t).Sugar()), context.Background()
}

func createProblemHandlerTenant(t *testing.T, ctx context.Context, client *ent.Client, suffix string) *ent.Tenant {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName("Problem Tenant " + suffix).
		SetCode("problem-" + suffix).
		SetDomain("problem-" + suffix + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return tenant
}

func createProblemHandlerUser(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, suffix string) *ent.User {
	t.Helper()
	user, err := client.User.Create().
		SetUsername("problem-" + suffix).
		SetEmail("problem-" + suffix + "@example.com").
		SetName("Problem User").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return user
}

func createProblemHandlerProblem(t *testing.T, ctx context.Context, service *Service, tenantID, userID int) *Problem {
	t.Helper()
	p, err := service.Create(ctx, tenantID, &Problem{
		Title: "Repeated outage", Description: "Repeated production outage", Priority: "high", CreatedBy: userID,
	})
	require.NoError(t, err)
	return p
}

func TestProblemServiceLifecycleAndTimestamps(t *testing.T) {
	client, service, ctx := setupProblemHandlerTest(t)
	defer client.Close()
	tenant := createProblemHandlerTenant(t, ctx, client, "lifecycle")
	user := createProblemHandlerUser(t, ctx, client, tenant.ID, "lifecycle")
	p := createProblemHandlerProblem(t, ctx, service, tenant.ID, user.ID)

	assert.Equal(t, "open", p.Status)
	p, err := service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "investigating"})
	require.NoError(t, err)
	assert.Nil(t, p.ResolvedAt)
	p, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "resolved"})
	require.NoError(t, err)
	require.NotNil(t, p.ResolvedAt)
	p, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "investigating"})
	require.NoError(t, err)
	assert.Nil(t, p.ResolvedAt)

	_, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "unknown"})
	require.ErrorContains(t, err, "invalid problem status transition")
}

func TestProblemRepositorySoftDeleteExcludedEverywhere(t *testing.T) {
	client, service, ctx := setupProblemHandlerTest(t)
	defer client.Close()
	tenant := createProblemHandlerTenant(t, ctx, client, "delete")
	user := createProblemHandlerUser(t, ctx, client, tenant.ID, "delete")
	p := createProblemHandlerProblem(t, ctx, service, tenant.ID, user.ID)

	require.NoError(t, service.Delete(ctx, p.ID, tenant.ID))
	_, err := service.Get(ctx, p.ID, tenant.ID)
	require.True(t, ent.IsNotFound(err))
	list, total, err := service.List(ctx, tenant.ID, 1, 10, nil)
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, list)
	stats, err := service.GetStats(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Zero(t, stats.Total)

	stored, err := client.Problem.Get(ctx, p.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.DeletedAt)
}

func TestProblemAssociationsEnforceTenantBoundary(t *testing.T) {
	client, service, ctx := setupProblemHandlerTest(t)
	defer client.Close()
	tenantA := createProblemHandlerTenant(t, ctx, client, "association-a")
	tenantB := createProblemHandlerTenant(t, ctx, client, "association-b")
	userA := createProblemHandlerUser(t, ctx, client, tenantA.ID, "association-a")
	userB := createProblemHandlerUser(t, ctx, client, tenantB.ID, "association-b")
	p := createProblemHandlerProblem(t, ctx, service, tenantA.ID, userA.ID)

	localTicket, err := client.Ticket.Create().
		SetTitle("Local ticket").SetTicketNumber("PRB-LOCAL").SetRequesterID(userA.ID).SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)
	foreignTicket, err := client.Ticket.Create().
		SetTitle("Foreign ticket").SetTicketNumber("PRB-FOREIGN").SetRequesterID(userB.ID).SetTenantID(tenantB.ID).Save(ctx)
	require.NoError(t, err)

	require.NoError(t, service.AddAssociations(ctx, tenantA.ID, p.ID, "ticket", []int{localTicket.ID, localTicket.ID}))
	err = service.AddAssociations(ctx, tenantA.ID, p.ID, "ticket", []int{foreignTicket.ID})
	require.ErrorContains(t, err, "current tenant")

	withAssociations, err := service.GetWithAssociations(ctx, p.ID, tenantA.ID)
	require.NoError(t, err)
	require.Len(t, withAssociations.Tickets, 1)
	assert.Equal(t, localTicket.ID, withAssociations.Tickets[0].ID)
}
