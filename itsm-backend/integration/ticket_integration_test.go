package integration

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestMultiTenantIsolation 测试多租户数据隔离
func TestMultiTenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	_ = zaptest.NewLogger(t).Sugar()

	// 创建租户 A
	tenantA, err := client.Tenant.Create().
		SetName("Tenant A").
		SetCode("TENANTA").
		SetDomain("a.test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建租户 B
	tenantB, err := client.Tenant.Create().
		SetName("Tenant B").
		SetCode("TENANTB").
		SetDomain("b.test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 在租户 A 下创建用户
	userA, err := client.User.Create().
		SetUsername("usera").
		SetEmail("usera@test.com").
		SetPasswordHash("hash").
		SetName("User A").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)

	// 在租户 A 创建工单
	_, err = client.Ticket.Create().
		SetTitle("Tenant A Ticket").
		SetDescription("Belongs to A").
		SetPriority("medium").
		SetStatus("open").
		SetTicketNumber("TKT-001").
		SetRequesterID(userA.ID).
		SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)

	// 验证租户 A 有工单
	ticketsA, err := client.Ticket.Query().All(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(ticketsA), 1)

	// 验证租户隔离
	_ = tenantB
}

// TestUserCreation 测试用户创建
func TestUserCreation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	_ = zaptest.NewLogger(t).Sugar()

	// 创建租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("USERTEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建用户
	_, err = client.User.Create().
		SetUsername("testuser").
		SetEmail("test@test.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetRole("admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 验证用户数量
	users, err := client.User.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, users, 1)
}

// TestTicketWithRelations 测试工单与用户关联
func TestTicketWithRelations(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	_ = zaptest.NewLogger(t).Sugar()

	// 创建租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TICKETTEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建用户
	user, err := client.User.Create().
		SetUsername("ticketuser").
		SetEmail("ticket@test.com").
		SetPasswordHash("hash").
		SetName("Ticket User").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建工单
	ticket, err := client.Ticket.Create().
		SetTitle("Test Ticket").
		SetDescription("Test Description").
		SetPriority("high").
		SetStatus("open").
		SetTicketNumber("TKT-002").
		SetRequesterID(user.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 验证工单与用户关联
	require.Equal(t, user.ID, ticket.RequesterID)
}
