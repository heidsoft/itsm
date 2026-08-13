package service

import (
	"context"
	"strings"
	"testing"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// createTestTicket wires the minimum required fields for a ticket row.
func createTestTicket(ctx context.Context, t *testing.T, client interface {
	Create() interface {
		SetTicketNumber(string) interface {
			SetTitle(string) interface {
				SetRequesterID(int) interface {
					SetTenantID(int) interface {
						SetStatus(string) interface {
							Save(context.Context) (interface{}, error)
						}
					}
				}
			}
		}
	}
}, number, title string, requesterID, tenantID int, status string,
) {
	t.Helper()
}

// -----------------------------------------------------------------------------
// PR-1.4 — Ticket dependency service (parent/child) impact branches
// -----------------------------------------------------------------------------

func TestTicketDependency_Analyze_NoChildrenReturnsLowRisk(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_dep_no_children?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	tenant, _ := createCMDBTestTenant(ctx, client, "Dep Tenant", "dep-tenant", "dep.example.com")
	user := client.User.Create().SetUsername("dep-user").SetEmail("dep@example.com").
		SetName("Dep User").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	parent := client.Ticket.Create().SetTicketNumber("TCK-DEP-1").SetTitle("parent").
		SetRequesterID(user.ID).SetTenantID(tenant.ID).SetStatus("open").SaveX(ctx)

	svc := NewTicketDependencyService(client, zaptest.NewLogger(t).Sugar())

	impact, err := svc.AnalyzeDependencyImpact(ctx, parent.ID, "close", nil, tenant.ID)
	require.NoError(t, err)
	require.NotNil(t, impact)
	assert.Equal(t, parent.ID, impact.TicketID)
	assert.Equal(t, 0, impact.AffectedCount)
	assert.Equal(t, "low", impact.RiskLevel)
	assert.Empty(t, impact.Warnings)
}

func TestTicketDependency_Analyze_CloseWithOpenChildrenIssuesWarnings(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_dep_close?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	tenant, _ := createCMDBTestTenant(ctx, client, "Close Tenant", "close-tenant", "close.example.com")
	user := client.User.Create().SetUsername("close-user").SetEmail("close@example.com").
		SetName("Close User").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	parent := client.Ticket.Create().SetTicketNumber("TCK-CLOSE-1").SetTitle("parent").
		SetRequesterID(user.ID).SetTenantID(tenant.ID).SetStatus("open").SaveX(ctx)
	childOpen := client.Ticket.Create().SetTicketNumber("TCK-CLOSE-2").SetTitle("child-open").
		SetRequesterID(user.ID).SetTenantID(tenant.ID).SetStatus("open").SetParentTicketID(parent.ID).SaveX(ctx)
	childClosed := client.Ticket.Create().SetTicketNumber("TCK-CLOSE-3").SetTitle("child-closed").
		SetRequesterID(user.ID).SetTenantID(tenant.ID).SetStatus("closed").SetParentTicketID(parent.ID).SaveX(ctx)
	_ = childClosed // keep linter happy; not asserted because close should not warn on it

	svc := NewTicketDependencyService(client, zaptest.NewLogger(t).Sugar())
	impact, err := svc.AnalyzeDependencyImpact(ctx, parent.ID, "close", nil, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, impact.AffectedCount, "must count both children")
	assert.Equal(t, "medium", impact.RiskLevel, "one warning => medium")
	assert.Len(t, impact.Warnings, 1)
	assert.Contains(t, impact.Warnings[0], childOpen.TicketNumber,
		"warning must reference the still-open child, not the closed one")
	for _, w := range impact.AffectedTickets {
		assert.Equal(t, childOpen.ID, w.ID, "affected tickets must contain only the open child")
		assert.Equal(t, "blocked", w.ImpactType)
	}
}

func TestTicketDependency_Analyze_CloseWithMultipleWarningsEscalatesRisk(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_dep_escalate?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	tenant, _ := createCMDBTestTenant(ctx, client, "Esc Tenant", "esc-tenant", "esc.example.com")
	user := client.User.Create().SetUsername("esc-user").SetEmail("esc@example.com").
		SetName("Esc User").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	parent := client.Ticket.Create().SetTicketNumber("TCK-ESC-1").SetTitle("parent").
		SetRequesterID(user.ID).SetTenantID(tenant.ID).SetStatus("open").SaveX(ctx)
	for i := 0; i < 3; i++ {
		client.Ticket.Create().
			SetTicketNumber("TCK-ESC-C" + string(rune('1'+i))).
			SetTitle("open-child").
			SetRequesterID(user.ID).SetTenantID(tenant.ID).
			SetStatus("open").SetParentTicketID(parent.ID).SaveX(ctx)
	}

	svc := NewTicketDependencyService(client, zaptest.NewLogger(t).Sugar())
	impact, err := svc.AnalyzeDependencyImpact(ctx, parent.ID, "close", nil, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "high", impact.RiskLevel, "3 warnings => high risk")
	assert.Len(t, impact.Warnings, 3)
	assert.NotEmpty(t, impact.Recommendations)
}

func TestTicketDependency_Analyze_DeleteAlwaysRecommendsCaution(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_dep_delete?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant, _ := createCMDBTestTenant(ctx, client, "Del Tenant", "del-tenant", "del.example.com")
	user := client.User.Create().SetUsername("del-user").SetEmail("del@example.com").
		SetName("Del User").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	parent := client.Ticket.Create().SetTicketNumber("TCK-DEL-1").SetTitle("parent").
		SetRequesterID(user.ID).SetTenantID(tenant.ID).SetStatus("open").SaveX(ctx)
	// 3 children → 3 warnings → high risk per the calc-tier in service.
	for i := 0; i < 3; i++ {
		client.Ticket.Create().
			SetTicketNumber("TCK-DEL-C" + string(rune('1'+i))).
			SetTitle("open-child").
			SetRequesterID(user.ID).SetTenantID(tenant.ID).
			SetStatus("closed").SetParentTicketID(parent.ID).SaveX(ctx)
	}

	svc := NewTicketDependencyService(client, zaptest.NewLogger(t).Sugar())
	impact, err := svc.AnalyzeDependencyImpact(ctx, parent.ID, "delete", nil, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, impact.AffectedCount)
	assert.Equal(t, "high", impact.RiskLevel, "3 warnings => high risk")
	hasCaution := false
	for _, r := range impact.Recommendations {
		if strings.Contains(r, "不可逆") || strings.Contains(r, "irreversible") {
			hasCaution = true
			break
		}
	}
	assert.True(t, hasCaution, "delete must mention irreversibility")
	for _, w := range impact.AffectedTickets {
		assert.Equal(t, "orphaned", w.ImpactType)
	}
}

func TestTicketDependency_Analyze_ChangeStatusToClosedAffectsOpenChildren(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_dep_chstatus?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant, _ := createCMDBTestTenant(ctx, client, "ChStatus Tenant", "chstatus", "chstatus.example.com")
	user := client.User.Create().SetUsername("chstatus").SetEmail("chstatus@example.com").
		SetName("ChStatus User").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	parent := client.Ticket.Create().SetTicketNumber("TCK-CHST-1").SetTitle("parent").
		SetRequesterID(user.ID).SetTenantID(tenant.ID).SetStatus("open").SaveX(ctx)
	openChild := client.Ticket.Create().SetTicketNumber("TCK-CHST-2").SetTitle("child-open").
		SetRequesterID(user.ID).SetTenantID(tenant.ID).SetStatus("open").SetParentTicketID(parent.ID).SaveX(ctx)
	client.Ticket.Create().SetTicketNumber("TCK-CHST-3").SetTitle("child-closed").
		SetRequesterID(user.ID).SetTenantID(tenant.ID).SetStatus("closed").SetParentTicketID(parent.ID).SaveX(ctx)

	svc := NewTicketDependencyService(client, zaptest.NewLogger(t).Sugar())
	newStatus := "closed"
	impact, err := svc.AnalyzeDependencyImpact(ctx, parent.ID, "change_status", &newStatus, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "medium", impact.RiskLevel, "1 warning => medium")
	assert.Len(t, impact.Warnings, 1)
	assert.Equal(t, openChild.ID, impact.AffectedTickets[0].ID)
	assert.Equal(t, "status_change", impact.AffectedTickets[0].ImpactType)
}

func TestTicketDependency_Analyze_UnknownActionIsNoOp(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_dep_unknown?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant, _ := createCMDBTestTenant(ctx, client, "Unknown Tenant", "unknown-tenant", "unknown.example.com")
	user := client.User.Create().SetUsername("unknown").SetEmail("unknown@example.com").
		SetName("Unknown User").SetPasswordHash("hash").SetTenantID(tenant.ID).SaveX(ctx)
	parent := client.Ticket.Create().SetTicketNumber("TCK-UNK-1").SetTitle("parent").
		SetRequesterID(user.ID).SetTenantID(tenant.ID).SetStatus("open").SaveX(ctx)

	svc := NewTicketDependencyService(client, zaptest.NewLogger(t).Sugar())
	impact, err := svc.AnalyzeDependencyImpact(ctx, parent.ID, "frobnicate", nil, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "low", impact.RiskLevel)
	assert.Empty(t, impact.Warnings)
}

func TestTicketDependency_Analyze_TenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_dep_iso?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenantA, _ := createCMDBTestTenant(ctx, client, "Iso A", "iso-a", "iso-a.example.com")
	tenantB, _ := createCMDBTestTenant(ctx, client, "Iso B", "iso-b", "iso-b.example.com")
	userA := client.User.Create().SetUsername("iso-a-user").SetEmail("a@example.com").
		SetName("A").SetPasswordHash("hash").SetTenantID(tenantA.ID).SaveX(ctx)
	parentA := client.Ticket.Create().SetTicketNumber("TCK-ISO-1").SetTitle("parent").
		SetRequesterID(userA.ID).SetTenantID(tenantA.ID).SetStatus("open").SaveX(ctx)

	svc := NewTicketDependencyService(client, zaptest.NewLogger(t).Sugar())
	if _, err := svc.AnalyzeDependencyImpact(ctx, parentA.ID, "close", nil, tenantB.ID); err == nil {
		t.Fatal("tenant B must not see tenant A's ticket")
	}
}