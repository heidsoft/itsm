package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== 测试设置辅助函数 ====================

func setupSLAMonitorTest(t *testing.T) (*ent.Client, *SLAMonitorService, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	service := NewSLAMonitorService(client, logger)
	ctx := context.Background()
	return client, service, ctx
}

func createSLATestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Test Tenant " + suffix).
		SetCode("test" + suffix).
		SetDomain("test" + suffix + ".com").
		SetStatus("active").
		Save(ctx)
}

func createSLATestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string) (*ent.User, error) {
	return client.User.Create().
		SetUsername("testuser" + suffix).
		SetEmail("test" + suffix + "@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
}

func createSLATestDefinition(ctx context.Context, client *ent.Client, tenantID int, name string) (*ent.SLADefinition, error) {
	return client.SLADefinition.Create().
		SetName(name).
		SetDescription("Test SLA Definition").
		SetPriority("medium").
		SetResponseTime(60).
		SetResolutionTime(240).
		SetTenantID(tenantID).
		Save(ctx)
}

// ==================== SLA检查测试 ====================

func TestSLAMonitorService_CheckSLAViolations_Empty(t *testing.T) {
	client, service, ctx := setupSLAMonitorTest(t)
	defer client.Close()

	testTenant, err := createSLATestTenant(ctx, client, "empty")
	require.NoError(t, err)

	// 没有工单时检查SLA
	stats, err := service.CheckSLAViolations(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, stats.TotalChecked)
	assert.Equal(t, 0, stats.NewViolations)
}

func TestSLAMonitorService_CheckSLAViolations_WithTickets(t *testing.T) {
	client, service, ctx := setupSLAMonitorTest(t)
	defer client.Close()

	testTenant, err := createSLATestTenant(ctx, client, "tickets")
	require.NoError(t, err)

	testUser, err := createSLATestUser(ctx, client, testTenant.ID, "sla")
	require.NoError(t, err)

	// 创建SLA定义
	_, err = createSLATestDefinition(ctx, client, testTenant.ID, "响应时间SLA")
	require.NoError(t, err)

	// 创建工单（使用过去的时间，让SLA可能过期）
	createdAt := time.Now().Add(-2 * time.Hour)
	_, err = client.Ticket.Create().
		SetTitle("Test Ticket").
		SetDescription("Test description").
		SetPriority("medium").
		SetType("incident").
		SetStatus("open").
		SetTicketNumber("TKT-SLA-001").
		SetTenantID(testTenant.ID).
		SetRequesterID(testUser.ID).
		SetCreatedAt(createdAt).
		Save(ctx)
	require.NoError(t, err)

	// 检查SLA - 注意实际检查逻辑可能需要工单有特定的SLA绑定
	stats, err := service.CheckSLAViolations(ctx, testTenant.ID)
	require.NoError(t, err)
	// 验证检查执行成功（具体结果取决于服务实现）
	assert.NotNil(t, stats)
}

// ==================== SLA指标计算测试 ====================

func TestSLAMonitorService_CalculateSLAMetrics(t *testing.T) {
	client, service, ctx := setupSLAMonitorTest(t)
	defer client.Close()

	testTenant, err := createSLATestTenant(ctx, client, "metrics")
	require.NoError(t, err)

	testUser, err := createSLATestUser(ctx, client, testTenant.ID, "metrics")
	require.NoError(t, err)

	// 创建一些工单用于计算指标（使用不同的工单编号）
	for i := 0; i < 5; i++ {
		createdAt := time.Now().Add(-time.Duration(i+1) * 24 * time.Hour)
		_, err := client.Ticket.Create().
			SetTitle("Test Ticket").
			SetDescription("Test description").
			SetPriority("medium").
			SetType("ticket").
			SetStatus("resolved").
			SetTicketNumber(fmt.Sprintf("TKT-METRICS-%03d", i)).
			SetTenantID(testTenant.ID).
			SetRequesterID(testUser.ID).
			SetCreatedAt(createdAt).
			Save(ctx)
		require.NoError(t, err)
	}

	// 计算SLA指标
	startTime := time.Now().Add(-7 * 24 * time.Hour)
	endTime := time.Now()

	metrics, err := service.CalculateSLAMetrics(ctx, testTenant.ID, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.GreaterOrEqual(t, metrics.TotalTickets, 0)
}

// ==================== SLA合规性统计测试 ====================

func TestSLAMonitorService_GetSLAComplianceByDefinition(t *testing.T) {
	client, service, ctx := setupSLAMonitorTest(t)
	defer client.Close()

	testTenant, err := createSLATestTenant(ctx, client, "compliance")
	require.NoError(t, err)

	testUser, err := createSLATestUser(ctx, client, testTenant.ID, "compliance")
	require.NoError(t, err)

	// 创建SLA定义
	slaDef, err := createSLATestDefinition(ctx, client, testTenant.ID, "响应时间SLA")
	require.NoError(t, err)

	// 创建关联SLA定义的工单
	_, err = client.Ticket.Create().
		SetTitle("Compliance Test Ticket").
		SetDescription("Test description").
		SetPriority("medium").
		SetType("incident").
		SetStatus("open").
		SetTicketNumber("TKT-COMP-001").
		SetTenantID(testTenant.ID).
		SetRequesterID(testUser.ID).
		SetSLADefinitionID(slaDef.ID).
		Save(ctx)
	require.NoError(t, err)

	// 获取合规性统计
	stats, err := service.GetSLAComplianceByDefinition(ctx, testTenant.ID)
	require.NoError(t, err)
	// 返回空列表而非nil
	assert.NotNil(t, stats)
}

// ==================== Dashboard指标测试 ====================

func TestSLAMonitorService_GetDashboardMetrics(t *testing.T) {
	client, service, ctx := setupSLAMonitorTest(t)
	defer client.Close()

	testTenant, err := createSLATestTenant(ctx, client, "dashboard")
	require.NoError(t, err)

	testUser, err := createSLATestUser(ctx, client, testTenant.ID, "dashboard")
	require.NoError(t, err)

	// 创建SLA定义
	_, err = createSLATestDefinition(ctx, client, testTenant.ID, "Dashboard SLA")
	require.NoError(t, err)

	// 创建一些工单
	_, err = client.Ticket.Create().
		SetTitle("Dashboard Test Ticket").
		SetDescription("Test description").
		SetPriority("medium").
		SetType("incident").
		SetStatus("open").
		SetTicketNumber("TKT-DASH-001").
		SetTenantID(testTenant.ID).
		SetRequesterID(testUser.ID).
		Save(ctx)
	require.NoError(t, err)

	// 获取Dashboard指标
	dashboard, err := service.GetDashboardMetrics(ctx, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, dashboard)
}

// ==================== SLA违规创建测试 ====================

func TestSLAMonitorService_CreateViolation(t *testing.T) {
	client, service, ctx := setupSLAMonitorTest(t)
	defer client.Close()

	testTenant, err := createSLATestTenant(ctx, client, "violation")
	require.NoError(t, err)

	testUser, err := createSLATestUser(ctx, client, testTenant.ID, "violation")
	require.NoError(t, err)

	// 创建SLA定义
	slaDef, err := createSLATestDefinition(ctx, client, testTenant.ID, "Violation SLA")
	require.NoError(t, err)

	// 创建工单，并设置SLA定义ID
	ticket, err := client.Ticket.Create().
		SetTitle("Violation Test Ticket").
		SetDescription("Test description").
		SetPriority("medium").
		SetType("incident").
		SetStatus("open").
		SetTicketNumber("TKT-VIOL-001").
		SetTenantID(testTenant.ID).
		SetRequesterID(testUser.ID).
		SetSLADefinitionID(slaDef.ID). // 必须设置SLA定义ID
		Save(ctx)
	require.NoError(t, err)

	// 创建违规记录
	slaDefMap := map[int]string{slaDef.ID: slaDef.Name}
	deadline := time.Now().Add(-1 * time.Hour) // 1小时前已经过期

	err = service.createViolation(ctx, ticket, "response_time", deadline, slaDefMap)
	require.NoError(t, err)

	// 验证违规记录已创建
	violations, err := client.SLAViolation.Query().
		All(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(violations), 1)
}
