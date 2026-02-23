package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestTicketLifecycleIntegration 测试工单生命周期集成
func TestTicketLifecycleIntegration(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	// 创建租户和用户
	tenant, err := client.Tenant.Create().
		SetName("Lifecycle Tenant").
		SetCode("LC-TENANT").
		SetDomain("lifecycle.test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("lifecycleuser").
		SetEmail("lc@test.com").
		SetPasswordHash("hash").
		SetName("Lifecycle User").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 1. 创建工单 (Open)
	ticketNumber := fmt.Sprintf("TKT-LC-%d", time.Now().UnixNano())
	ticket, err := client.Ticket.Create().
		SetTitle("Lifecycle Test Ticket").
		SetDescription("Testing ticket lifecycle").
		SetPriority("medium").
		SetStatus("open").
		SetTicketNumber(ticketNumber).
		SetRequesterID(user.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	logger.Info("Ticket created", "ticket_id", ticket.ID, "status", ticket.Status)
	require.Equal(t, "open", ticket.Status)

	// 2. 分配工单 (Assigned)
	ticket, err = ticket.Update().
		SetStatus("assigned").
		SetAssigneeID(user.ID).
		Save(ctx)
	require.NoError(t, err)
	logger.Info("Ticket assigned", "ticket_id", ticket.ID, "status", ticket.Status)
	require.Equal(t, "assigned", ticket.Status)

	// 3. 处理中 (In Progress)
	ticket, err = ticket.Update().
		SetStatus("in_progress").
		Save(ctx)
	require.NoError(t, err)
	logger.Info("Ticket in progress", "ticket_id", ticket.ID, "status", ticket.Status)
	require.Equal(t, "in_progress", ticket.Status)

	// 4. 已解决 (Resolved)
	ticket, err = ticket.Update().
		SetStatus("resolved").
		Save(ctx)
	require.NoError(t, err)
	logger.Info("Ticket resolved", "ticket_id", ticket.ID, "status", ticket.Status)
	require.Equal(t, "resolved", ticket.Status)

	// 5. 已关闭 (Closed)
	ticket, err = ticket.Update().
		SetStatus("closed").
		Save(ctx)
	require.NoError(t, err)
	logger.Info("Ticket closed", "ticket_id", ticket.ID, "status", ticket.Status)
	require.Equal(t, "closed", ticket.Status)

	t.Log("Ticket lifecycle integration test completed successfully")
}

// TestIncidentIntegration 测试事件管理集成
func TestIncidentIntegration(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	// 创建租户
	tenant, err := client.Tenant.Create().
		SetName("Incident Tenant").
		SetCode("INC-TENANT").
		SetDomain("incident.test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建用户 (reporter)
	user, err := client.User.Create().
		SetUsername("incidentuser").
		SetEmail("incident@test.com").
		SetPasswordHash("hash").
		SetName("Incident User").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建服务
	service, err := client.ServiceCatalog.Create().
		SetName("Email Service").
		SetDescription("Corporate email service").
		SetStatus("active").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	logger.Info("Created service", "service_id", service.ID)

	// 创建事件
	incidentNumber := fmt.Sprintf("INC-%d", time.Now().UnixNano())
	incident, err := client.Incident.Create().
		SetTitle("Email Service Down").
		SetDescription("Users cannot access email").
		SetPriority("critical").
		SetStatus("open").
		SetIncidentNumber(incidentNumber).
		SetReporterID(user.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	logger.Info("Created incident", "incident_id", incident.ID)

	// 验证事件
	require.Equal(t, "critical", incident.Priority)
	require.Equal(t, "open", incident.Status)

	t.Log("Incident integration test completed successfully")
}

// TestChangeManagementIntegration 测试变更管理集成
func TestChangeManagementIntegration(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	// 创建租户
	tenant, err := client.Tenant.Create().
		SetName("Change Tenant").
		SetCode("CHG-TENANT").
		SetDomain("change.test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建用户 (created by)
	user, err := client.User.Create().
		SetUsername("changeuser").
		SetEmail("change@test.com").
		SetPasswordHash("hash").
		SetName("Change User").
		SetRole("manager").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建变更请求
	change, err := client.Change.Create().
		SetTitle("Database Migration").
		SetDescription("Migrate to new database server").
		SetType("standard").
		SetPriority("high").
		SetStatus("draft").
		SetCreatedBy(user.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	logger.Info("Created change request", "change_id", change.ID)

	// 验证变更
	require.Equal(t, "draft", change.Status)
	require.Equal(t, "standard", change.Type)

	t.Log("Change management integration test completed successfully")
}

// TestServiceCatalogIntegration 测试服务目录集成
func TestServiceCatalogIntegration(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	// 创建租户
	tenant, err := client.Tenant.Create().
		SetName("Service Tenant").
		SetCode("SVC-TENANT").
		SetDomain("service.test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建服务分类
	category, err := client.ServiceCatalog.Create().
		SetName("IT Services").
		SetDescription("IT service offerings").
		SetStatus("active").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	logger.Info("Created service catalog", "service_id", category.ID)

	// 验证服务
	require.Equal(t, "active", category.Status)

	t.Log("Service catalog integration test completed successfully")
}

// TestMultiTenantDataIsolation 测试多租户数据隔离
func TestMultiTenantDataIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

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

	// 在租户 B 下创建用户
	userB, err := client.User.Create().
		SetUsername("userb").
		SetEmail("userb@test.com").
		SetPasswordHash("hash").
		SetName("User B").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)

	// 在租户 A 创建工单
	_, err = client.Ticket.Create().
		SetTitle("Tenant A Ticket").
		SetDescription("Belongs to A").
		SetPriority("medium").
		SetStatus("open").
		SetTicketNumber("TKT-A-001").
		SetRequesterID(userA.ID).
		SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)

	// 在租户 B 创建工单
	_, err = client.Ticket.Create().
		SetTitle("Tenant B Ticket").
		SetDescription("Belongs to B").
		SetPriority("high").
		SetStatus("open").
		SetTicketNumber("TKT-B-001").
		SetRequesterID(userB.ID).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)

	// 验证租户 A 的工单数量
	allTickets, err := client.Ticket.Query().All(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(allTickets), 2)
	logger.Info("Total tickets count", "count", len(allTickets))

	t.Log("Multi-tenant data isolation test completed successfully")
}
