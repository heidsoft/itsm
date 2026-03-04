package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestTicketSearchService_SearchTickets(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	searchService := NewTicketSearchService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建测试工单
	testCases := []struct {
		title       string
		description string
	}{
		{"PC Hardware Issue", "My computer is not turning on"},
		{"Network Problem", "Unable to connect to internet"},
		{"Software Bug", "Application crashes on startup"},
		{"Printer Not Working", "Printer is offline"},
	}

	for i, tc := range testCases {
		_, err := client.Ticket.Create().
			SetTitle(tc.title).
			SetDescription(tc.description).
			SetPriority("medium").
			SetType("ticket").
			SetStatus("open").
			SetTicketNumber(fmt.Sprintf("TKT-SEARCH-%03d", i+1)).
			SetTenantID(testTenant.ID).
			SetRequesterID(testUser.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	t.Run("搜索存在的关键词", func(t *testing.T) {
		tickets, err := searchService.SearchTickets(ctx, "PC Hardware", testTenant.ID)
		require.NoError(t, err)
		assert.Len(t, tickets, 1)
		assert.Contains(t, tickets[0].Title, "PC Hardware")
	})

	t.Run("搜索不存在的关键词", func(t *testing.T) {
		tickets, err := searchService.SearchTickets(ctx, "nonexistent", testTenant.ID)
		require.NoError(t, err)
		assert.Len(t, tickets, 0)
	})

	t.Run("空搜索词", func(t *testing.T) {
		_, err := searchService.SearchTickets(ctx, "", testTenant.ID)
		assert.Error(t, err)
	})
}

func TestTicketSearchService_GetOverdueTickets(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	searchService := NewTicketSearchService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	now := time.Now()
	past := now.Add(-48 * time.Hour) // 48小时前

	// 创建过期工单
	overdueTicket, err := client.Ticket.Create().
		SetTitle("Overdue Ticket").
		SetDescription("This ticket is overdue").
		SetPriority("high").
		SetType("ticket").
		SetStatus("open").
		SetTicketNumber("TKT-OVERDUE-001").
		SetTenantID(testTenant.ID).
		SetRequesterID(testUser.ID).
		SetSLAResponseDeadline(past).
		Save(ctx)
	require.NoError(t, err)

	t.Run("获取过期工单", func(t *testing.T) {
		overdueTickets, err := searchService.GetOverdueTickets(ctx, testTenant.ID)
		require.NoError(t, err)
		assert.Len(t, overdueTickets, 1)
		assert.Equal(t, overdueTicket.ID, overdueTickets[0].ID)
	})
}

func TestTicketSearchService_GetTicketStats(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	searchService := NewTicketSearchService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建不同状态和优先级的工单
	testCases := []struct {
		status   string
		priority string
	}{
		{"open", "high"},
		{"open", "medium"},
		{"in_progress", "low"},
		{"resolved", "medium"},
		{"closed", "critical"},
	}

	for i, tc := range testCases {
		_, err := client.Ticket.Create().
			SetTitle("Test Ticket " + string(rune('A'+i))).
			SetDescription("Test description").
			SetPriority(tc.priority).
			SetType("ticket").
			SetStatus(tc.status).
			SetTicketNumber(fmt.Sprintf("TKT-STATS-%03d", i+1)).
			SetTenantID(testTenant.ID).
			SetRequesterID(testUser.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	t.Run("获取统计信息", func(t *testing.T) {
		stats, err := searchService.GetTicketStats(ctx, testTenant.ID)
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats.Total, 5)
		assert.GreaterOrEqual(t, stats.Open, 0)
		assert.GreaterOrEqual(t, stats.HighPriority, 0)
	})
}
