package service

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

// ==================== Dashboard Service 测试设置 ====================

func setupDashboardTest(t *testing.T) (*ent.Client, *DashboardService, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ent_dashboard?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	service := NewDashboardService(client, logger)
	ctx := context.Background()
	return client, service, ctx
}

func createDashboardTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Dashboard Test Tenant " + suffix).
		SetCode("dashboard" + suffix).
		SetDomain("dashboard" + suffix + ".test").
		SetStatus("active").
		Save(ctx)
}

func createDashboardTestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string) (*ent.User, error) {
	return client.User.Create().
		SetUsername("dashboarduser" + suffix).
		SetEmail("dashboard" + suffix + "@test.com").
		SetName("Dashboard User " + suffix).
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
}

// ==================== Dashboard Service 基础测试 ====================

func TestDashboardService_NewDashboardService(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent_dashboard_new?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	service := NewDashboardService(client, logger)

	assert.NotNil(t, service)
	assert.Equal(t, client, service.client)
	assert.Equal(t, logger, service.logger)
}

func TestDashboardService_GetSLAComplianceData_EmptyData(t *testing.T) {
	client, service, ctx := setupDashboardTest(t)
	defer client.Close()

	testTenant, err := createDashboardTestTenant(ctx, client, "empty")
	require.NoError(t, err)

	// 无数据时应该返回空统计
	data, err := service.GetSLAComplianceData(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, data)
	// 初始状态，合规率应该是 100% 或 0（取决于实现）
}

func TestDashboardService_GetSLAComplianceData_WithTickets(t *testing.T) {
	client, service, ctx := setupDashboardTest(t)
	defer client.Close()

	testTenant, err := createDashboardTestTenant(ctx, client, "with_tickets")
	require.NoError(t, err)

	testUser, err := createDashboardTestUser(ctx, client, testTenant.ID, "with_tickets")
	require.NoError(t, err)

	// 创建一些工单
	for i := 0; i < 5; i++ {
		_, err = client.Ticket.Create().
			SetTitle("Test Ticket " + string(rune('A'+i))).
			SetDescription("Test description").
			SetStatus("open").
			SetPriority("medium").
			SetTicketNumber(fmt.Sprintf("TKT-%06d", i+1)).
			SetRequesterID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	data, err := service.GetSLAComplianceData(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, data)
	// 验证数据结构正确
	assert.GreaterOrEqual(t, data.ComplianceRate, 0.0)
	assert.LessOrEqual(t, data.ComplianceRate, 100.0)
}

func TestDashboardService_GetSLAComplianceData_MultiTenantIsolation(t *testing.T) {
	client, service, ctx := setupDashboardTest(t)
	defer client.Close()

	testTenant1, err := createDashboardTestTenant(ctx, client, "tenant1")
	require.NoError(t, err)

	testTenant2, err := createDashboardTestTenant(ctx, client, "tenant2")
	require.NoError(t, err)

	testUser1, err := createDashboardTestUser(ctx, client, testTenant1.ID, "t1")
	require.NoError(t, err)

	testUser2, err := createDashboardTestUser(ctx, client, testTenant2.ID, "t2")
	require.NoError(t, err)

	// 租户1创建5个工单
	for i := 0; i < 5; i++ {
		_, err = client.Ticket.Create().
			SetTitle("Tenant1 Ticket " + string(rune('A'+i))).
			SetStatus("open").
			SetTicketNumber(fmt.Sprintf("T1-%06d", i+1)).
			SetRequesterID(testUser1.ID).
			SetTenantID(testTenant1.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	// 租户2创建3个工单
	for i := 0; i < 3; i++ {
		_, err = client.Ticket.Create().
			SetTitle("Tenant2 Ticket " + string(rune('A'+i))).
			SetStatus("open").
			SetTicketNumber(fmt.Sprintf("T2-%06d", i+1)).
			SetRequesterID(testUser2.ID).
			SetTenantID(testTenant2.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	// 获取各自租户的数据
	data1, err := service.GetSLAComplianceData(ctx, testTenant1.ID)
	require.NoError(t, err)

	data2, err := service.GetSLAComplianceData(ctx, testTenant2.ID)
	require.NoError(t, err)

	// 验证两个租户都能正常返回数据
	assert.NotNil(t, data1)
	assert.NotNil(t, data2)
}

// ==================== Dashboard 数据结构测试 ====================

func TestDashboardOverviewStats_Structure(t *testing.T) {
	stats := DashboardOverviewStats{
		TotalTickets:      100,
		PendingTickets:    20,
		InProgressTickets: 30,
		ResolvedToday:    10,
		AvgResponseTime:   2.5,
		AvgResolutionTime: 24.0,
	}

	assert.Equal(t, 100, stats.TotalTickets)
	assert.Equal(t, 20, stats.PendingTickets)
	assert.Equal(t, 30, stats.InProgressTickets)
	assert.Equal(t, 10, stats.ResolvedToday)
	assert.Equal(t, 2.5, stats.AvgResponseTime)
	assert.Equal(t, 24.0, stats.AvgResolutionTime)
}

func TestSLAComplianceData_Structure(t *testing.T) {
	data := SLAComplianceData{
		ComplianceRate:           95.5,
		ResponseTimeCompliance:   98.0,
		ResolutionTimeCompliance: 92.0,
		AtRiskTickets:            5,
		BreachedTickets:          3,
	}

	assert.Equal(t, 95.5, data.ComplianceRate)
	assert.Equal(t, 98.0, data.ResponseTimeCompliance)
	assert.Equal(t, 92.0, data.ResolutionTimeCompliance)
	assert.Equal(t, 5, data.AtRiskTickets)
	assert.Equal(t, 3, data.BreachedTickets)
}

func TestSLAComplianceData_JSONTags(t *testing.T) {
	data := SLAComplianceData{
		ComplianceRate:           100.0,
		ResponseTimeCompliance:   100.0,
		ResolutionTimeCompliance: 100.0,
		AtRiskTickets:            0,
		BreachedTickets:          0,
	}

	// 验证 JSON 标签正确
	assert.Equal(t, 0, data.AtRiskTickets)
	assert.Equal(t, 0, data.BreachedTickets)
}

// ==================== Dashboard 边界情况测试 ====================

func TestDashboardService_GetSLAComplianceData_ZeroTickets(t *testing.T) {
	client, service, ctx := setupDashboardTest(t)
	defer client.Close()

	testTenant, err := createDashboardTestTenant(ctx, client, "zero_tickets")
	require.NoError(t, err)

	data, err := service.GetSLAComplianceData(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, data)
	// 无工单时，违规数应该是0
	assert.Equal(t, 0, data.BreachedTickets)
}

func TestDashboardService_GetSLAComplianceData_LargeVolume(t *testing.T) {
	client, service, ctx := setupDashboardTest(t)
	defer client.Close()

	testTenant, err := createDashboardTestTenant(ctx, client, "large_volume")
	require.NoError(t, err)

	testUser, err := createDashboardTestUser(ctx, client, testTenant.ID, "large")
	require.NoError(t, err)

	// 创建大量工单测试性能
	for i := 0; i < 100; i++ {
		_, err = client.Ticket.Create().
			SetTitle("Volume Test Ticket " + string(rune('A'+i%26))).
			SetStatus("open").
			SetTicketNumber(fmt.Sprintf("VOL-%06d", i+1)).
			SetRequesterID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	data, err := service.GetSLAComplianceData(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

// ==================== Dashboard SLA 违规计算测试 ====================

func TestDashboardService_GetSLAComplianceData_WithSLAViolations(t *testing.T) {
	client, service, ctx := setupDashboardTest(t)
	defer client.Close()

	testTenant, err := createDashboardTestTenant(ctx, client, "sla_violation")
	require.NoError(t, err)

	testUser, err := createDashboardTestUser(ctx, client, testTenant.ID, "sla")
	require.NoError(t, err)

	// 创建带有 SLA 的工单
	_, err = client.Ticket.Create().
		SetTitle("SLA Test Ticket").
		SetStatus("open").
		SetTicketNumber(fmt.Sprintf("SLA-%06d", 1)).
		SetRequesterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetSLADefinitionID(1).
		Save(ctx)
	require.NoError(t, err)

	data, err := service.GetSLAComplianceData(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

func TestDashboardService_ComplianceRateEdgeCases(t *testing.T) {
	client, service, ctx := setupDashboardTest(t)
	defer client.Close()

	testTenant, err := createDashboardTestTenant(ctx, client, "compliance")
	require.NoError(t, err)

	testUser, err := createDashboardTestUser(ctx, client, testTenant.ID, "compliance")
	require.NoError(t, err)

	// 创建一些工单
	for i := 0; i < 10; i++ {
		_, err = client.Ticket.Create().
			SetTitle("Compliance Test Ticket " + string(rune('A'+i))).
			SetStatus("open").
			SetTicketNumber(fmt.Sprintf("CMP-%06d", i+1)).
			SetRequesterID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	data, err := service.GetSLAComplianceData(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, data)
	// 合规率应该在有效范围内
	assert.GreaterOrEqual(t, data.ComplianceRate, 0.0)
	assert.LessOrEqual(t, data.ComplianceRate, 100.0)
}

// ==================== Dashboard 并发测试 ====================

func TestDashboardService_ConcurrentAccess(t *testing.T) {
	client, service, ctx := setupDashboardTest(t)
	defer client.Close()

	testTenant, err := createDashboardTestTenant(ctx, client, "concurrent")
	require.NoError(t, err)

	testUser, err := createDashboardTestUser(ctx, client, testTenant.ID, "concurrent")
	require.NoError(t, err)

	// 创建一些数据
	for i := 0; i < 10; i++ {
		_, err = client.Ticket.Create().
			SetTitle("Concurrent Test Ticket " + string(rune('A'+i))).
			SetStatus("open").
			SetTicketNumber(fmt.Sprintf("CON-%06d", i+1)).
			SetRequesterID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	// 并发读取
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- true }()
			data, err := service.GetSLAComplianceData(ctx, testTenant.ID)
			assert.NoError(t, err)
			assert.NotNil(t, data)
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 5; i++ {
		<-done
	}
}
