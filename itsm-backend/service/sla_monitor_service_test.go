package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/slastate"
	"itsm-backend/ent/slaviolation"
	"itsm-backend/service/sla"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== 测试设置辅助函数 ====================

func setupSLAMonitorTest(t *testing.T) (*ent.Client, *SLAMonitorService, context.Context) {
	client := enttest.Open(t, "sqlite3", testDSN())
	logger := zaptest.NewLogger(t).Sugar()
	service := NewSLAMonitorService(client, logger)
	// 与 internal/bootstrap/app.go 一致：sla_states 读路径依赖 SetSLAStore 注入的 store。
	service.SetSLAStore(sla.NewStore(client, sla.NewEngine(nil)))
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

// slaStore 由 bootstrap 事后 SetSLAStore 注入；漏注入必须在依赖边界 fail closed，
// 不得在 nil *sla.Store 上解引用 client 触发 panic 并终止 SLA watcher goroutine。
func TestSLAMonitorService_MissingSLAStoreFailsClosed(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()

	monitor := NewSLAMonitorService(client, zaptest.NewLogger(t).Sugar())
	ctx := context.Background()

	testTenant, err := createSLATestTenant(ctx, client, "no_store")
	require.NoError(t, err)

	_, err = monitor.CheckSLAViolations(ctx, testTenant.ID)
	require.Error(t, err, "SLA store 未注入时必须返回错误")
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr)
	assert.Equal(t, common.ServiceUnavailableCode, bizErr.Code, "依赖缺失应为 operational unavailable，不得伪装成空结果")

	_, err = monitor.GetDashboardMetrics(ctx, testTenant.ID)
	require.Error(t, err)
}

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
	testTicket, err := client.Ticket.Create().
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

	// 读路径以 sla_states 为权威源：只写工单内嵌 sla_definition_id 不会被统计到。
	_, err = client.SLAState.Create().
		SetTenantID(testTenant.ID).
		SetAggregateType("ticket").
		SetAggregateID(testTicket.ID).
		SetSLADefinitionID(slaDef.ID).
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 获取合规性统计
	stats, err := service.GetSLAComplianceByDefinition(ctx, testTenant.ID)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	assert.Equal(t, slaDef.ID, stats[0].SLADefinitionID)
	assert.Equal(t, 1, stats[0].TotalTickets)
	assert.Equal(t, 100.0, stats[0].ComplianceRate)
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

	created, err := service.createViolation(ctx, ticket, "response_time", deadline, slaDef.ID, slaDefMap)
	require.NoError(t, err)
	assert.True(t, created, "expected SLA violation to be created")

	// 验证违规记录已创建
	violations, err := client.SLAViolation.Query().
		All(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(violations), 1)

	// 幂等：再次调用应当返回 created=false 而不创建新记录
	created, err = service.createViolation(ctx, ticket, "response_time", deadline, slaDef.ID, slaDefMap)
	require.NoError(t, err)
	assert.False(t, created, "expected duplicate SLA violation to be suppressed")
}

// P1 SLA 口径回归（对应 sla_monitor_service.go 三个口径缺陷）：
//   1. violation_time 曾锚定扫描时刻（SetViolationTime(now)）——违约的
//      权威时刻是 deadline 被突破的瞬间，扫描周期波动会让口径漂移。
//   2. paused 工单违例误判：暂停语义 = 不计时，deadline 在 ResumeSLA
//      才延长；扫描曾直接按原 deadline 判定 paused 工单违规。
//   3. resume 不回收：resume 把暂停时长并入 deadline 后，原违规可能
//      不再成立，必须回收（标 resolved），否则陈旧违规永久留存误导
//      MTTR/超时统计与升级。

func createSlaPauseFixture(t *testing.T, ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, *ent.Ticket, *ent.SLADefinition) {
	t.Helper()
	tn, err := createSLATestTenant(ctx, client, suffix)
	require.NoError(t, err)
	user, err := createSLATestUser(ctx, client, tn.ID, suffix)
	require.NoError(t, err)
	def, err := createSLATestDefinition(ctx, client, tn.ID, "P1 SLA "+suffix)
	require.NoError(t, err)
	tk, err := client.Ticket.Create().
		SetTitle("P1 SLA 口径工单").
		SetTicketNumber("TKT-SLA-P1-" + suffix).
		SetStatus("open").
		SetPriority("medium").
		SetType("incident").
		SetTenantID(tn.ID).
		SetRequesterID(user.ID).
		Save(ctx)
	require.NoError(t, err)
	return tn, tk, def
}

// TestCreateViolation_AnchorsViolationTimeToDeadline violation_time 必须
// 锚定 deadline（权威违约时刻），不得使用扫描时刻。
func TestCreateViolation_AnchorsViolationTimeToDeadline(t *testing.T) {
	client, service, ctx := setupSLAMonitorTest(t)
	defer client.Close()
	_, tk, def := createSlaPauseFixture(t, ctx, client, "anchor")

	deadline := time.Now().Add(-2 * time.Hour)
	created, err := service.createViolation(ctx, tk, "resolution_time", deadline, def.ID, map[int]string{def.ID: "P1 SLA anchor"})
	require.NoError(t, err)
	require.True(t, created)

	v, err := client.SLAViolation.Query().
		Where(slaviolation.TenantIDEQ(tk.TenantID), slaviolation.TicketIDEQ(tk.ID)).
		Only(ctx)
	require.NoError(t, err)

	assert.WithinDuration(t, deadline, v.ViolationTime, time.Minute,
		"violation_time 必须锚定被突破的 deadline，而不是扫描时刻")
	if !v.ViolationOccurredAt.IsZero() {
		assert.WithinDuration(t, deadline, v.ViolationOccurredAt, time.Minute,
			"violation_occurred_at 同样锚定 deadline")
	}
}

// TestSLAViolations_SkipsPausedTickets paused 工单不得按原 deadline 误判违例。
func TestSLAViolations_SkipsPausedTickets(t *testing.T) {
	client, service, ctx := setupSLAMonitorTest(t)
	defer client.Close()
	tn, tk, def := createSlaPauseFixture(t, ctx, client, "paused")

	past := time.Now().Add(-30 * time.Minute)
	_, err := client.SLAState.Create().
		SetTenantID(tn.ID).
		SetAggregateType("ticket").
		SetAggregateID(tk.ID).
		SetSLADefinitionID(def.ID).
		SetStatus("active").
		SetResponseDeadline(past).
		SetResolutionDeadline(past).
		Save(ctx)
	require.NoError(t, err)

	// 基线：active 工单按过期 deadline 会创建违规。
	stats, err := service.CheckSLAViolations(ctx, tn.ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, stats.NewViolations, 1, "active 过期工单必须判定违规")

	// 暂停：同一 deadline，暂停后不得再被误判（也不得对 paused 重复计）。
	require.NoError(t, service.PauseSLA(ctx, tn.ID, "ticket", tk.ID, "等待客户补充信息"))

	stats2, err := service.CheckSLAViolations(ctx, tn.ID)
	require.NoError(t, err)
	// 该唯一工单 paused：违例总数不应因本次扫描而继续增长为新建。
	assert.LessOrEqual(t, stats2.NewViolations, 0, "paused 工单不得创建新违例")
}

// TestResumeSLA_RecyclesNoLongerBreachedViolation resume 延长 deadline 后，
// 不再成立的违规必须自动回收（resolved），不得永久留存。
func TestResumeSLA_RecyclesNoLongerBreachedViolation(t *testing.T) {
	client, service, ctx := setupSLAMonitorTest(t)
	defer client.Close()
	tn, tk, def := createSlaPauseFixture(t, ctx, client, "recycle")

	// 已过期 1 分钟的响应 deadline；解决 deadline 尚未到期。
	responseDeadline := time.Now().Add(-time.Minute)
	resolutionDeadline := time.Now().Add(30 * time.Minute)
	_, err := client.SLAState.Create().
		SetTenantID(tn.ID).
		SetAggregateType("ticket").
		SetAggregateID(tk.ID).
		SetSLADefinitionID(def.ID).
		SetStatus("active").
		SetResponseDeadline(responseDeadline).
		SetResolutionDeadline(resolutionDeadline).
		Save(ctx)
	require.NoError(t, err)

	stats, err := service.CheckSLAViolations(ctx, tn.ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, stats.NewViolations, 1, "响应违规必须先被创建")

	// 暂停（paused_at=now），然后把 paused_at 回拨 2 小时模拟长时间暂停。
	require.NoError(t, service.PauseSLA(ctx, tn.ID, "ticket", tk.ID, "客户要求暂停"))
	_, err = client.SLAState.Update().
		Where(
			slastate.TenantIDEQ(tn.ID),
			slastate.AggregateTypeEQ("ticket"),
			slastate.AggregateIDEQ(tk.ID),
		).
		SetPausedAt(time.Now().Add(-2 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, service.ResumeSLA(ctx, tn.ID, "ticket", tk.ID))

	// 回收断言：响应违规 resolved，解决违规不存在且未被误建。
	violations, err := client.SLAViolation.Query().
		Where(slaviolation.TenantIDEQ(tn.ID), slaviolation.TicketIDEQ(tk.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Equal(t, "response_time", violations[0].ViolationType)
	assert.True(t, violations[0].IsResolved, "deadline 延长 2h 后不再被突破，违例必须回收")
	assert.False(t, violations[0].ResolvedAt.IsZero(), "回收必须记录 resolved_at")
}

// TestResumeSLA_RejectsInvalidState 违规的 SLA 恢复调用必须返回业务冲突
// （未暂停时不得恢复；不存在状态时不得伪造成功）。
func TestResumeSLA_RejectsInvalidState(t *testing.T) {
	client, service, ctx := setupSLAMonitorTest(t)
	defer client.Close()
	_, tk, _ := createSlaPauseFixture(t, ctx, client, "invalidresume")

	err := service.ResumeSLA(ctx, tk.TenantID, "ticket", tk.ID)
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr, "未暂停工单的恢复必须是业务错误")
	assert.Equal(t, common.ConflictCode, bizErr.Code)
}
