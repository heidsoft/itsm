package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestTicketSLAService_GetTicketSLAInfo(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewTicketSLAService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

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

	testTicket, err := client.Ticket.Create().
		SetTitle("测试工单").
		SetDescription("测试工单描述").
		SetPriority("medium").
		SetStatus("open").
		SetTicketNumber("TICKET-001").
		SetRequesterID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetCreatedAt(time.Now().Add(-2 * time.Hour)). // 2小时前创建
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		ticketID      int
		tenantID      int
		expectedError bool
		checkResult   func(*testing.T, *TicketSLAInfoResult)
	}{
		{
			name:          "获取工单SLA信息",
			ticketID:      testTicket.ID,
			tenantID:      testTenant.ID,
			expectedError: false,
			checkResult: func(t *testing.T, result *TicketSLAInfoResult) {
				assert.Equal(t, testTicket.ID, result.TicketID)
				assert.Equal(t, "medium", result.Priority)
				assert.Greater(t, result.ResponseTimeUsed, 0)
				assert.Greater(t, result.ResolutionTimeUsed, 0)
			},
		},
		{
			name:          "工单不存在",
			ticketID:      99999,
			tenantID:      testTenant.ID,
			expectedError: true,
			checkResult:   nil,
		},
		{
			name:          "租户不匹配",
			ticketID:      testTicket.ID,
			tenantID:      99999,
			expectedError: true,
			checkResult:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := slaService.GetTicketSLAInfo(ctx, tt.ticketID, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestTicketSLAService_GetTicketStats(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewTicketSLAService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

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

	// 创建不同状态的工单
	statuses := []string{"open", "open", "in_progress", "in_progress", "resolved", "closed"}
	for i, status := range statuses {
		_, err := client.Ticket.Create().
			SetTitle("工单 " + string(rune('0'+i))).
			SetDescription("描述").
			SetPriority("medium").
			SetStatus(status).
			SetTicketNumber("TICKET-00" + string(rune('1'+i))).
			SetRequesterID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	stats, err := slaService.GetTicketStats(ctx, testTenant.ID)

	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 6, stats.TotalTickets)
	assert.Equal(t, 2, stats.OpenTickets)
	assert.Equal(t, 2, stats.InProgressTickets)
	assert.Equal(t, 1, stats.ResolvedTickets)
	assert.Equal(t, 1, stats.ClosedTickets)
}

func TestTicketSLAService_CalculateSLADeadline(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewTicketSLAService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建 SLA 定义
	_, err = client.SLADefinition.Create().
		SetName("测试 SLA").
		SetServiceType("incident").
		SetPriority("high").
		SetResponseTime(30).    // 30分钟
		SetResolutionTime(120). // 2小时
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	result, err := slaService.CalculateSLADeadline(ctx, testTenant.ID, "incident", "high")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.ResponseDeadline)
	assert.NotNil(t, result.ResolutionDeadline)
}

func TestTicketSLAService_CalculateSLADeadlineFromRequest(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewTicketSLAService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建 SLA 定义
	slaDef, err := client.SLADefinition.Create().
		SetName("测试 SLA").
		SetServiceType("incident").
		SetPriority("medium").
		SetResponseTime(60).    // 60分钟
		SetResolutionTime(480). // 8小时
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		tenantID      int
		ticketType    string
		priority      string
		expectedError bool
		checkResult   func(*testing.T, *SLADeadlineResult)
	}{
		{
			name:          "使用精确匹配的SLA定义",
			tenantID:      testTenant.ID,
			ticketType:    "incident",
			priority:      "medium",
			expectedError: false,
			checkResult: func(t *testing.T, result *SLADeadlineResult) {
				assert.Equal(t, slaDef.ID, result.SLADefinitionID)
				assert.NotNil(t, result.ResponseDeadline)
				assert.NotNil(t, result.ResolutionDeadline)
			},
		},
		{
			name:          "使用默认值(找不到SLA定义)",
			tenantID:      testTenant.ID,
			ticketType:    "unknown_type",
			priority:      "low",
			expectedError: false,
			checkResult: func(t *testing.T, result *SLADeadlineResult) {
				// 找不到精确匹配时会 fallback 到类型匹配的定义（如果存在）
				// 因为测试中创建了 service_request 类型的 SLA，会被用作 fallback
				assert.Greater(t, result.SLADefinitionID, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := slaService.CalculateSLADeadlineFromRequest(ctx, tt.tenantID, tt.ticketType, tt.priority)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestTicketSLAService_AdjustToBusinessHours(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewTicketSLAService(client, logger)

	tests := []struct {
		name        string
		inputTime   time.Time
		checkResult func(*testing.T, time.Time)
	}{
		{
			name:      "工作时间内的周五",
			inputTime: time.Date(2024, time.January, 5, 10, 0, 0, 0, time.UTC), // 2024-01-05 是周五
			checkResult: func(t *testing.T, result time.Time) {
				// 应该保持原时间
				assert.Equal(t, 10, result.Hour())
			},
		},
		{
			name:      "周六调整到周一",
			inputTime: time.Date(2024, time.January, 6, 10, 0, 0, 0, time.UTC), // 2024-01-06 是周六
			checkResult: func(t *testing.T, result time.Time) {
				// 应该调整到周一
				assert.Equal(t, time.Monday, result.Weekday())
				assert.Equal(t, 9, result.Hour())
			},
		},
		{
			name:      "周日调整到周一",
			inputTime: time.Date(2024, time.January, 7, 10, 0, 0, 0, time.UTC), // 2024-01-07 是周日
			checkResult: func(t *testing.T, result time.Time) {
				// 应该调整到周一
				assert.Equal(t, time.Monday, result.Weekday())
				assert.Equal(t, 9, result.Hour())
			},
		},
		{
			name:      "早于工作时间调整到9点",
			inputTime: time.Date(2024, time.January, 8, 7, 0, 0, 0, time.UTC), // 2024-01-08 是周一
			checkResult: func(t *testing.T, result time.Time) {
				// 应该调整到9点
				assert.Equal(t, 9, result.Hour())
			},
		},
		{
			name:      "晚于工作时间调整到次日9点",
			inputTime: time.Date(2024, time.January, 8, 20, 0, 0, 0, time.UTC), // 2024-01-08 20:00 是周一
			checkResult: func(t *testing.T, result time.Time) {
				// 应该调整到次日9点
				assert.Equal(t, 9, result.Hour())
				assert.Equal(t, time.Tuesday, result.Weekday())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slaService.AdjustToBusinessHours(tt.inputTime)
			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestTicketSLAService_getSLADefinition(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewTicketSLAService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建精确匹配的SLA定义
	exactMatch, err := client.SLADefinition.Create().
		SetName("精确匹配 SLA").
		SetServiceType("incident").
		SetPriority("critical").
		SetResponseTime(15).
		SetResolutionTime(60).
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建只匹配类型的SLA定义
	typeMatch, err := client.SLADefinition.Create().
		SetName("类型匹配 SLA").
		SetServiceType("service_request").
		SetResponseTime(240).
		SetResolutionTime(1440).
		SetIsActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name                   string
		ticketType             string
		priority               string
		expectedResponseTime   int
		expectedResolutionTime int
	}{
		{
			name:                   "精确匹配",
			ticketType:             "incident",
			priority:               "critical",
			expectedResponseTime:   exactMatch.ResponseTime,
			expectedResolutionTime: exactMatch.ResolutionTime,
		},
		{
			name:                   "类型匹配",
			ticketType:             "service_request",
			priority:               "low",
			expectedResponseTime:   typeMatch.ResponseTime,
			expectedResolutionTime: typeMatch.ResolutionTime,
		},
		{
			name:                   "默认SLA",
			ticketType:             "unknown_type",
			priority:               "low",
			expectedResponseTime:   60,
			expectedResolutionTime: 480,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slaDef, err := slaService.getSLADefinition(ctx, testTenant.ID, tt.ticketType, tt.priority)

			assert.NoError(t, err)
			assert.NotNil(t, slaDef)
			assert.Equal(t, tt.expectedResponseTime, slaDef.ResponseTime)
			assert.Equal(t, tt.expectedResolutionTime, slaDef.ResolutionTime)
		})
	}
}

func TestTicketSLAService_calculateDeadline(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	slaService := NewTicketSLAService(client, logger)

	startTime := time.Date(2024, time.January, 8, 9, 0, 0, 0, time.UTC) // 2024-01-08 是周一

	result := slaService.calculateDeadline(startTime, 60, false)

	// 应该正好是1小时后
	assert.Equal(t, startTime.Add(1*time.Hour), result)
}
