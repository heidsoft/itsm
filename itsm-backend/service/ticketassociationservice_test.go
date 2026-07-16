package service

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTicketAssociationService_ConfigurationItemAssociations(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ticketassoc?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	tenant := createTicketAssociationTenant(t, ctx, client, "assoc")
	user := createTicketAssociationUser(t, ctx, client, tenant.ID, "assoc-user")
	ciType := createTicketAssociationCIType(t, ctx, client, tenant.ID, "server")
	ticket := createTicketAssociationTicket(t, ctx, client, tenant.ID, user.ID, "TKT-ASSOC-001")
	ci1 := createTicketAssociationCI(t, ctx, client, tenant.ID, ciType.ID, "web-01", "SN-001")
	ci2 := createTicketAssociationCI(t, ctx, client, tenant.ID, ciType.ID, "db-01", "SN-002")

	service := NewTicketAssociationService(client)

	err := service.AddConfigurationItem(ctx, ticket.ID, ci1.ID)
	require.NoError(t, err)

	items, err := service.GetConfigurationItems(ctx, ticket.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, ci1.ID, items[0].ID)
	assert.Equal(t, "server", items[0].CIType)

	err = service.SetConfigurationItems(ctx, ticket.ID, []int{ci2.ID})
	require.NoError(t, err)

	items, err = service.GetConfigurationItems(ctx, ticket.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, ci2.ID, items[0].ID)

	err = service.RemoveConfigurationItem(ctx, ticket.ID, ci2.ID)
	require.NoError(t, err)

	items, err = service.GetConfigurationItems(ctx, ticket.ID)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestTicketAssociationService_RejectsCrossTenantConfigurationItemAssociation(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ticketassoc-cross?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	tenantA := createTicketAssociationTenant(t, ctx, client, "tenant-a")
	tenantB := createTicketAssociationTenant(t, ctx, client, "tenant-b")
	user := createTicketAssociationUser(t, ctx, client, tenantA.ID, "user-a")
	ciTypeA := createTicketAssociationCIType(t, ctx, client, tenantA.ID, "server-a")
	ciTypeB := createTicketAssociationCIType(t, ctx, client, tenantB.ID, "server-b")
	ticket := createTicketAssociationTicket(t, ctx, client, tenantA.ID, user.ID, "TKT-ASSOC-002")
	foreignCI := createTicketAssociationCI(t, ctx, client, tenantB.ID, ciTypeB.ID, "foreign-ci", "SN-100")
	_ = ciTypeA

	service := NewTicketAssociationService(client)

	err := service.AddConfigurationItem(ctx, ticket.ID, foreignCI.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "配置项不存在")

	items, err := service.GetConfigurationItems(ctx, ticket.ID)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestTicketAssociationService_MaintainsBidirectionalRelatedTickets(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ticketassoc-related?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	tenant := createTicketAssociationTenant(t, ctx, client, "related")
	user := createTicketAssociationUser(t, ctx, client, tenant.ID, "related-user")
	ticketA := createTicketAssociationTicket(t, ctx, client, tenant.ID, user.ID, "TKT-REL-001")
	ticketB := createTicketAssociationTicket(t, ctx, client, tenant.ID, user.ID, "TKT-REL-002")
	ticketC := createTicketAssociationTicket(t, ctx, client, tenant.ID, user.ID, "TKT-REL-003")
	service := NewTicketAssociationService(client)

	require.NoError(t, service.UpdateTicketAssociations(ctx, ticketA.ID, &UpdateAssociationsRequest{RelatedIDs: []int{ticketB.ID}}))
	relatedToB, err := service.GetRelatedTickets(ctx, ticketB.ID)
	require.NoError(t, err)
	require.Len(t, relatedToB, 1)
	assert.Equal(t, ticketA.ID, relatedToB[0].ID)

	require.NoError(t, service.UpdateTicketAssociations(ctx, ticketA.ID, &UpdateAssociationsRequest{RelatedIDs: []int{ticketC.ID}}))
	relatedToB, err = service.GetRelatedTickets(ctx, ticketB.ID)
	require.NoError(t, err)
	assert.Empty(t, relatedToB)
	relatedToC, err := service.GetRelatedTickets(ctx, ticketC.ID)
	require.NoError(t, err)
	require.Len(t, relatedToC, 1)
	assert.Equal(t, ticketA.ID, relatedToC[0].ID)
}

func TestTicketAssociationService_RejectsInvalidParentRelationships(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ticketassoc-parent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	tenant := createTicketAssociationTenant(t, ctx, client, "parent")
	otherTenant := createTicketAssociationTenant(t, ctx, client, "parent-other")
	user := createTicketAssociationUser(t, ctx, client, tenant.ID, "parent-user")
	otherUser := createTicketAssociationUser(t, ctx, client, otherTenant.ID, "parent-other-user")
	ticketA := createTicketAssociationTicket(t, ctx, client, tenant.ID, user.ID, "TKT-PARENT-001")
	ticketB := createTicketAssociationTicket(t, ctx, client, tenant.ID, user.ID, "TKT-PARENT-002")
	foreignTicket := createTicketAssociationTicket(t, ctx, client, otherTenant.ID, otherUser.ID, "TKT-PARENT-003")
	service := NewTicketAssociationService(client)

	err := service.UpdateTicketAssociations(ctx, ticketA.ID, &UpdateAssociationsRequest{ParentID: &ticketA.ID})
	require.ErrorContains(t, err, "自己的父工单")

	err = service.UpdateTicketAssociations(ctx, ticketA.ID, &UpdateAssociationsRequest{ParentID: &foreignTicket.ID})
	require.ErrorContains(t, err, "父工单不存在")

	require.NoError(t, service.UpdateTicketAssociations(ctx, ticketB.ID, &UpdateAssociationsRequest{ParentID: &ticketA.ID}))
	err = service.UpdateTicketAssociations(ctx, ticketA.ID, &UpdateAssociationsRequest{ParentID: &ticketB.ID})
	require.ErrorContains(t, err, "不能形成循环")
}

func createTicketAssociationTenant(t *testing.T, ctx context.Context, client *ent.Client, code string) *ent.Tenant {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName(code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return tenant
}

func createTicketAssociationUser(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, username string) *ent.User {
	t.Helper()
	user, err := client.User.Create().
		SetUsername(username).
		SetEmail(username + "@example.com").
		SetName(username).
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return user
}

func createTicketAssociationCIType(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, name string) *ent.CIType {
	t.Helper()
	ciType, err := client.CIType.Create().
		SetName(name).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return ciType
}

func createTicketAssociationTicket(t *testing.T, ctx context.Context, client *ent.Client, tenantID, requesterID int, number string) *ent.Ticket {
	t.Helper()
	ticket, err := client.Ticket.Create().
		SetTitle(number).
		SetDescription("ticket for associations").
		SetPriority("medium").
		SetType("incident").
		SetStatus("open").
		SetTicketNumber(number).
		SetRequesterID(requesterID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return ticket
}

func createTicketAssociationCI(t *testing.T, ctx context.Context, client *ent.Client, tenantID, ciTypeID int, name, serial string) *ent.ConfigurationItem {
	t.Helper()
	ci, err := client.ConfigurationItem.Create().
		SetName(name).
		SetCiTypeID(ciTypeID).
		SetCiType("server").
		SetStatus("active").
		SetSerialNumber(serial).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return ci
}
