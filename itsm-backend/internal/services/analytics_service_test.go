package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== 测试设置辅助函数 ====================

func setupAnalyticsTest(t *testing.T) (*ent.Client, *AnalyticsService, context.Context) {
	client := enttest.Open(t, "sqlite3", testDSN())
	logger := zaptest.NewLogger(t).Sugar()
	service := NewAnalyticsService(client, logger)
	ctx := context.Background()
	return client, service, ctx
}

func createAnalyticsTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Test Tenant " + suffix).
		SetCode("test" + suffix).
		SetDomain("test" + suffix + ".com").
		SetStatus("active").
		Save(ctx)
}

func createAnalyticsTestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string) (*ent.User, error) {
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

// ==================== 深度分析测试 ====================

func TestAnalyticsService_GetDeepAnalytics_Empty(t *testing.T) {
	client, service, ctx := setupAnalyticsTest(t)
	defer client.Close()

	testTenant, err := createAnalyticsTestTenant(ctx, client, "empty")
	require.NoError(t, err)

	req := &dto.DeepAnalyticsRequest{
		TimeRange:  []string{"2024-01-01", "2024-01-31"},
		Dimensions: []string{"status"},
		Metrics:    []string{"count"},
	}

	response, err := service.GetDeepAnalytics(ctx, req, testTenant.ID, DashboardViewer{ScopeAll: true})
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 0, response.Summary.Total)
}

func TestAnalyticsService_GetDeepAnalytics_WithTickets(t *testing.T) {
	client, service, ctx := setupAnalyticsTest(t)
	defer client.Close()

	testTenant, err := createAnalyticsTestTenant(ctx, client, "tickets")
	require.NoError(t, err)

	testUser, err := createAnalyticsTestUser(ctx, client, testTenant.ID, "analytics")
	require.NoError(t, err)

	// 创建不同状态的工单
	statuses := []string{"open", "in_progress", "resolved", "closed"}
	for i, status := range statuses {
		createdAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		_, err := client.Ticket.Create().
			SetTitle(fmt.Sprintf("Test Ticket %d", i)).
			SetDescription("Test description").
			SetPriority("medium").
			SetType("incident").
			SetStatus(status).
			SetTicketNumber(fmt.Sprintf("TKT-ANAL-%03d", i)).
			SetTenantID(testTenant.ID).
			SetRequesterID(testUser.ID).
			SetCreatedAt(createdAt).
			Save(ctx)
		require.NoError(t, err)
	}

	req := &dto.DeepAnalyticsRequest{
		TimeRange:  []string{"2024-01-01", "2024-01-31"},
		Dimensions: []string{"status"},
		Metrics:    []string{"count"},
	}

	response, err := service.GetDeepAnalytics(ctx, req, testTenant.ID, DashboardViewer{ScopeAll: true})
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 4, response.Summary.Total)
	assert.GreaterOrEqual(t, len(response.Data), 1)
}

func TestAnalyticsService_GetDeepAnalytics_WithFilters(t *testing.T) {
	client, service, ctx := setupAnalyticsTest(t)
	defer client.Close()

	testTenant, err := createAnalyticsTestTenant(ctx, client, "filter")
	require.NoError(t, err)

	testUser, err := createAnalyticsTestUser(ctx, client, testTenant.ID, "filter")
	require.NoError(t, err)

	// 创建不同优先级的工单
	priorities := []string{"low", "medium", "high", "critical"}
	for i, priority := range priorities {
		createdAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		_, err := client.Ticket.Create().
			SetTitle(fmt.Sprintf("Test Ticket %d", i)).
			SetDescription("Test description").
			SetPriority(priority).
			SetType("incident").
			SetStatus("open").
			SetTicketNumber(fmt.Sprintf("TKT-FLT-%03d", i)).
			SetTenantID(testTenant.ID).
			SetRequesterID(testUser.ID).
			SetCreatedAt(createdAt).
			Save(ctx)
		require.NoError(t, err)
	}

	req := &dto.DeepAnalyticsRequest{
		TimeRange:  []string{"2024-01-01", "2024-01-31"},
		Dimensions: []string{"priority"},
		Metrics:    []string{"count"},
		Filters: map[string]interface{}{
			"priority": "high",
		},
	}

	response, err := service.GetDeepAnalytics(ctx, req, testTenant.ID, DashboardViewer{ScopeAll: true})
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 1, response.Summary.Total) // 只有high优先级的工单
}

func TestAnalyticsService_GetDeepAnalytics_InvalidTimeFormat(t *testing.T) {
	client, service, ctx := setupAnalyticsTest(t)
	defer client.Close()

	testTenant, err := createAnalyticsTestTenant(ctx, client, "invalid")
	require.NoError(t, err)

	req := &dto.DeepAnalyticsRequest{
		TimeRange:  []string{"invalid-date", "2024-01-31"},
		Dimensions: []string{"status"},
		Metrics:    []string{"count"},
	}

	_, err = service.GetDeepAnalytics(ctx, req, testTenant.ID, DashboardViewer{ScopeAll: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "起始日期格式非法")
	// 关键回归（2026-10-18 缺陷 P1-5）：参数错误必须归类为 400，不能兜底成 500。
	// 断言 AppError 的 HTTPStatus，而不是只断言文案——文案会变，分类不能变。
	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 400, appErr.HTTPStatus)
}

func TestAnalyticsService_GetDeepAnalytics_RejectsIncompleteTimeRange(t *testing.T) {
	client, service, ctx := setupAnalyticsTest(t)
	defer client.Close()

	testTenant, err := createAnalyticsTestTenant(ctx, client, "incomplete-range")
	require.NoError(t, err)
	req := &dto.DeepAnalyticsRequest{
		TimeRange:  []string{"2024-01-01"},
		Dimensions: []string{"status"},
		Metrics:    []string{"count"},
	}

	_, err = service.GetDeepAnalytics(ctx, req, testTenant.ID, DashboardViewer{ScopeAll: true})
	require.ErrorContains(t, err, "必须恰好包含起止两个日期")
	// 同上：缺 timeRange 属调用方参数错误 → 400，不是 500。
	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 400, appErr.HTTPStatus)
}

func TestAnalyticsService_GetDeepAnalytics_GroupBy(t *testing.T) {
	client, service, ctx := setupAnalyticsTest(t)
	defer client.Close()

	testTenant, err := createAnalyticsTestTenant(ctx, client, "group")
	require.NoError(t, err)

	testUser, err := createAnalyticsTestUser(ctx, client, testTenant.ID, "group")
	require.NoError(t, err)

	// 创建不同状态的工单
	statuses := []string{"open", "open", "resolved", "resolved", "closed"}
	for i, status := range statuses {
		createdAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		_, err := client.Ticket.Create().
			SetTitle(fmt.Sprintf("Test Ticket %d", i)).
			SetDescription("Test description").
			SetPriority("medium").
			SetType("incident").
			SetStatus(status).
			SetTicketNumber(fmt.Sprintf("TKT-GRP-%03d", i)).
			SetTenantID(testTenant.ID).
			SetRequesterID(testUser.ID).
			SetCreatedAt(createdAt).
			Save(ctx)
		require.NoError(t, err)
	}

	groupBy := "status"
	req := &dto.DeepAnalyticsRequest{
		TimeRange:  []string{"2024-01-01", "2024-01-31"},
		Dimensions: []string{"status"},
		Metrics:    []string{"count"},
		GroupBy:    &groupBy,
	}

	response, err := service.GetDeepAnalytics(ctx, req, testTenant.ID, DashboardViewer{ScopeAll: true})
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 5, response.Summary.Total)
	// 应该有3个分组：open, resolved, closed
	assert.GreaterOrEqual(t, len(response.Data), 1)
}

// ==================== 导出测试 ====================

func TestAnalyticsService_ExportAnalytics_CSV(t *testing.T) {
	client, service, ctx := setupAnalyticsTest(t)
	defer client.Close()

	testTenant, err := createAnalyticsTestTenant(ctx, client, "csv")
	require.NoError(t, err)

	testUser, err := createAnalyticsTestUser(ctx, client, testTenant.ID, "csv")
	require.NoError(t, err)

	// 创建工单
	createdAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	_, err = client.Ticket.Create().
		SetTitle("Test Ticket").
		SetDescription("Test description").
		SetPriority("medium").
		SetType("incident").
		SetStatus("open").
		SetTicketNumber("TKT-CSV-001").
		SetTenantID(testTenant.ID).
		SetRequesterID(testUser.ID).
		SetCreatedAt(createdAt).
		Save(ctx)
	require.NoError(t, err)

	req := &dto.DeepAnalyticsRequest{
		TimeRange:  []string{"2024-01-01", "2024-01-31"},
		Dimensions: []string{"status"},
		Metrics:    []string{"count"},
	}

	data, filename, err := service.ExportAnalytics(ctx, req, "csv", testTenant.ID, DashboardViewer{ScopeAll: true})
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, filename, "analytics_")
	assert.Contains(t, filename, ".csv")
}

func TestAnalyticsService_ExportAnalytics_UnsupportedFormat(t *testing.T) {
	client, service, ctx := setupAnalyticsTest(t)
	defer client.Close()

	testTenant, err := createAnalyticsTestTenant(ctx, client, "format")
	require.NoError(t, err)

	req := &dto.DeepAnalyticsRequest{
		TimeRange:  []string{"2024-01-01", "2024-01-31"},
		Dimensions: []string{"status"},
		Metrics:    []string{"count"},
	}

	_, _, err = service.ExportAnalytics(ctx, req, "unsupported", testTenant.ID, DashboardViewer{ScopeAll: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

// ==================== 维度值测试 ====================

func TestAnalyticsService_GetDimensionValue(t *testing.T) {
	client, service, ctx := setupAnalyticsTest(t)
	defer client.Close()

	testTenant, err := createAnalyticsTestTenant(ctx, client, "dim")
	require.NoError(t, err)

	testUser, err := createAnalyticsTestUser(ctx, client, testTenant.ID, "dim")
	require.NoError(t, err)

	testAssignee, err := createAnalyticsTestUser(ctx, client, testTenant.ID, "dimassignee")
	require.NoError(t, err)

	// 创建工单
	ticket, err := client.Ticket.Create().
		SetTitle("Test Ticket").
		SetDescription("Test description").
		SetPriority("high").
		SetType("incident").
		SetStatus("open").
		SetTicketNumber("TKT-DIM-001").
		SetTenantID(testTenant.ID).
		SetRequesterID(testUser.ID).
		SetAssigneeID(testAssignee.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		dimension string
		expected  string
	}{
		{"status", "open"},
		{"priority", "high"},
		{"assignee", fmt.Sprintf("用户%d", testAssignee.ID)},
		{"unknown", "未知"},
	}

	for _, tt := range tests {
		t.Run(tt.dimension, func(t *testing.T) {
			result := service.getDimensionValue(ticket, tt.dimension)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== 汇总数据测试 ====================

func TestAnalyticsService_CalculateSummary(t *testing.T) {
	client, service, ctx := setupAnalyticsTest(t)
	defer client.Close()

	testTenant, err := createAnalyticsTestTenant(ctx, client, "summary")
	require.NoError(t, err)

	testUser, err := createAnalyticsTestUser(ctx, client, testTenant.ID, "summary")
	require.NoError(t, err)

	// 创建不同状态的工单
	tickets := make([]*ent.Ticket, 0)
	now := time.Now()

	for i := 0; i < 5; i++ {
		status := "open"
		if i < 3 {
			status = "resolved"
		}

		createdAt := now.Add(-time.Duration(i+1) * time.Hour)
		resolvedAt := time.Time{}
		if status == "resolved" {
			resolvedAt = now.Add(-time.Duration(i) * time.Minute)
		}

		ticket, err := client.Ticket.Create().
			SetTitle(fmt.Sprintf("Test Ticket %d", i)).
			SetDescription("Test description").
			SetPriority("medium").
			SetType("incident").
			SetStatus(status).
			SetTicketNumber(fmt.Sprintf("TKT-SUM-%03d", i)).
			SetTenantID(testTenant.ID).
			SetRequesterID(testUser.ID).
			SetCreatedAt(createdAt).
			SetNillableResolvedAt(&resolvedAt).
			Save(ctx)
		require.NoError(t, err)
		tickets = append(tickets, ticket)
	}

	summary, err := service.calculateSummary(ctx, testTenant.ID, tickets)
	require.NoError(t, err)
	assert.Equal(t, 5, summary.Total)
	assert.Equal(t, 3, summary.Resolved)
}

// ==================== 指标计算测试 ====================

func TestAnalyticsService_CalculateMetrics_Count(t *testing.T) {
	client, service, ctx := setupAnalyticsTest(t)
	defer client.Close()

	testTenant, err := createAnalyticsTestTenant(ctx, client, "metrics")
	require.NoError(t, err)

	testUser, err := createAnalyticsTestUser(ctx, client, testTenant.ID, "metrics")
	require.NoError(t, err)

	// 创建工单
	tickets := make([]*ent.Ticket, 0)
	for i := 0; i < 3; i++ {
		ticket, err := client.Ticket.Create().
			SetTitle(fmt.Sprintf("Test Ticket %d", i)).
			SetDescription("Test description").
			SetPriority("medium").
			SetType("incident").
			SetStatus("open").
			SetTicketNumber(fmt.Sprintf("TKT-MET-%03d", i)).
			SetTenantID(testTenant.ID).
			SetRequesterID(testUser.ID).
			Save(ctx)
		require.NoError(t, err)
		tickets = append(tickets, ticket)
	}

	point := dto.AnalyticsDataPoint{Name: "test"}
	point = service.calculateMetrics(point, tickets, []string{"count"})

	assert.Equal(t, 3, point.Count)
	assert.Equal(t, float64(3), point.Value)
}

// TestAnalyticsService_GetTicketStats_EmptyTenantArraysNotNil 锁定「已装配但无数据」的
// 契约：分组字段必须下发空数组而非 null。
//
// 为什么必须锁 JSON：Go 里 nil slice 和空 slice 用 assert.NotNil 能区分，但真正让
// 前端崩的是序列化后的 null。统一服务质量视图的 ServiceQualityOverview 直接消费
// 这个 DTO（绕过 analytics-service.ts 的 ?? [] 适配层），statusGroups 为 null 时
// .map() 会抛 TypeError，导致整个报表中心白屏。
//
// 「块未装配」的表达是块本身缺省（ServiceQualityReport.TicketStats == nil），
// 与本测试锁的「块已装配、内部数组为空」是两个不同语义，不得互相替代。
func TestAnalyticsService_GetTicketStats_EmptyTenantArraysNotNil(t *testing.T) {
	client, service, ctx := setupAnalyticsTest(t)
	defer client.Close()

	tenant, err := createAnalyticsTestTenant(ctx, client, "empty-ticket-stats")
	require.NoError(t, err)

	stats, err := service.GetTicketStats(ctx, tenant.ID)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 0, stats.Total, "空租户工单总数必须为 0")
	assert.NotNil(t, stats.StatusGroups, "状态分布必须是空数组，不得为 nil")
	assert.NotNil(t, stats.PriorityGroups, "优先级分布必须是空数组，不得为 nil")
	assert.NotNil(t, stats.Trend30d, "30 天趋势必须是空数组，不得为 nil")

	encoded, err := json.Marshal(stats)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "null",
		"序列化结果不得出现 null：前端直接 .map() 会整页崩溃；got %s", string(encoded))
	assert.Contains(t, string(encoded), `"statusGroups":[]`)
	assert.Contains(t, string(encoded), `"priorityGroups":[]`)
}
