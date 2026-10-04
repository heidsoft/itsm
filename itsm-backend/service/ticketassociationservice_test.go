package service

import (
	"context"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"itsm-backend/common"
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

	err := service.AddConfigurationItem(ctx, ticket.ID, ci1.ID, tenant.ID)
	require.NoError(t, err)

	items, err := service.GetConfigurationItems(ctx, ticket.ID, tenant.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, ci1.ID, items[0].ID)
	assert.Equal(t, "server", items[0].CIType)

	err = service.SetConfigurationItems(ctx, ticket.ID, []int{ci2.ID}, tenant.ID)
	require.NoError(t, err)

	items, err = service.GetConfigurationItems(ctx, ticket.ID, tenant.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, ci2.ID, items[0].ID)

	err = service.RemoveConfigurationItem(ctx, ticket.ID, ci2.ID, tenant.ID)
	require.NoError(t, err)

	items, err = service.GetConfigurationItems(ctx, ticket.ID, tenant.ID)
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

	err := service.AddConfigurationItem(ctx, ticket.ID, foreignCI.ID, tenantA.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "配置项不存在")

	items, err := service.GetConfigurationItems(ctx, ticket.ID, tenantA.ID)
	require.NoError(t, err)
	assert.Empty(t, items)
}

// TestTicketAssociationService_RejectsForeignTenantTicketReads 覆盖租户收敛的根因入口：
// 所有关联读写都必须先按调用者租户定位工单，而不是先按 ID 取行再决定是否过滤。
// router/ticket_relations_tenant_scope_route_test.go 在同一不变量上打真实 HTTP 端点。
func TestTicketAssociationService_RejectsForeignTenantTicketReads(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:ticketassoc-tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	tenantA := createTicketAssociationTenant(t, ctx, client, "read-a")
	tenantB := createTicketAssociationTenant(t, ctx, client, "read-b")
	userA := createTicketAssociationUser(t, ctx, client, tenantA.ID, "read-user-a")
	ticketA := createTicketAssociationTicket(t, ctx, client, tenantA.ID, userA.ID, "TKT-READ-A")
	// 乙租户的一行把 parent_ticket_id 指向甲租户的工单：缺少租户谓词时
	// getChildrenTree 会把它当成自己的子工单返回。
	foreignChild := createTicketAssociationTicket(t, ctx, client, tenantB.ID, userA.ID, "TKT-READ-B")
	if _, err := client.Ticket.UpdateOne(foreignChild).SetParentTicketID(ticketA.ID).Save(ctx); err != nil {
		t.Fatalf("注入跨租户口子工单失败: %v", err)
	}

	service := NewTicketAssociationService(client)

	err := service.UpdateTicketAssociations(ctx, ticketA.ID, tenantB.ID, &UpdateAssociationsRequest{})
	requireTicketAssociationNotFound(t, err)

	_, err = service.GetRelatedTickets(ctx, ticketA.ID, tenantB.ID)
	requireTicketAssociationNotFound(t, err)

	_, err = service.GetConfigurationItems(ctx, ticketA.ID, tenantB.ID)
	requireTicketAssociationNotFound(t, err)

	// 同租户读取不受影响，且子工单里不出现对方租户的行。
	deps, err := service.GetTicketDependencies(ctx, ticketA.ID, tenantA.ID)
	require.NoError(t, err)
	for _, item := range deps.ChildrenTree {
		assert.Equal(t, tenantA.ID, item.TenantID, "childrenTree 不得包含其他租户的行")
	}
	assert.Empty(t, deps.ChildrenTree)

	// 缺少租户上下文（<=0）必须 fail-closed，而不是退化成无租户过滤的全库读取。
	_, err = service.GetTicketDependencies(ctx, ticketA.ID, 0)
	requireTicketAssociationNotFound(t, err)
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

	require.NoError(t, service.UpdateTicketAssociations(ctx, ticketA.ID, tenant.ID, &UpdateAssociationsRequest{RelatedIDs: []int{ticketB.ID}}))
	relatedToB, err := service.GetRelatedTickets(ctx, ticketB.ID, tenant.ID)
	require.NoError(t, err)
	require.Len(t, relatedToB, 1)
	assert.Equal(t, ticketA.ID, relatedToB[0].ID)

	require.NoError(t, service.UpdateTicketAssociations(ctx, ticketA.ID, tenant.ID, &UpdateAssociationsRequest{RelatedIDs: []int{ticketC.ID}}))
	relatedToB, err = service.GetRelatedTickets(ctx, ticketB.ID, tenant.ID)
	require.NoError(t, err)
	assert.Empty(t, relatedToB)
	relatedToC, err := service.GetRelatedTickets(ctx, ticketC.ID, tenant.ID)
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

	err := service.UpdateTicketAssociations(ctx, ticketA.ID, tenant.ID, &UpdateAssociationsRequest{ParentID: &ticketA.ID})
	require.ErrorContains(t, err, "自己的父工单")

	err = service.UpdateTicketAssociations(ctx, ticketA.ID, tenant.ID, &UpdateAssociationsRequest{ParentID: &foreignTicket.ID})
	require.ErrorContains(t, err, "父工单不存在")

	require.NoError(t, service.UpdateTicketAssociations(ctx, ticketB.ID, tenant.ID, &UpdateAssociationsRequest{ParentID: &ticketA.ID}))
	err = service.UpdateTicketAssociations(ctx, ticketA.ID, tenant.ID, &UpdateAssociationsRequest{ParentID: &ticketB.ID})
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

// requireTicketAssociationNotFound 断言错误是可分类的 4004 业务拒绝，而不是一条会
// 被 router 映射成 500/5001 并透出底层错误串的裸 error。
func requireTicketAssociationNotFound(t *testing.T, err error) {
	t.Helper()

	var bizErr *common.BusinessError
	require.True(t, errors.As(err, &bizErr), "必须是可分类的业务拒绝: %v", err)
	assert.Equal(t, common.NotFoundCode, bizErr.Code)
}
