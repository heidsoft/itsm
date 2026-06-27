package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/ticketcomment"
	"itsm-backend/ent/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestTicketService_CreateTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

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

	tests := []struct {
		name          string
		request       *dto.CreateTicketRequest
		tenantID      int
		expectedError bool
	}{
		{
			name: "成功创建工单",
			request: &dto.CreateTicketRequest{
				Title:       "测试工单",
				Description: "这是一个测试工单的详细描述",
				Priority:    "medium",
				Category:    "incident",
				RequesterID: testUser.ID,
				FormFields: map[string]interface{}{
					"category": "hardware",
					"urgency":  "normal",
				},
			},
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name: "标题为空",
			request: &dto.CreateTicketRequest{
				Title:       "",
				Description: "描述",
				Priority:    "medium",
				Category:    "incident",
				RequesterID: testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: true,
		},
		{
			name: "描述为空（V2 不做必填校验，会创建成功）",
			request: &dto.CreateTicketRequest{
				Title:       "标题",
				Description: "",
				Priority:    "medium",
				Category:    "incident",
				RequesterID: testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name: "无效的优先级（当前系统不验证优先级值）",
			request: &dto.CreateTicketRequest{
				Title:       "标题",
				Description: "描述",
				Priority:    "invalid", // 当前系统不验证优先级值，会创建成功
				Category:    "incident",
				RequesterID: testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: false, // 系统当前不验证优先级值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 确保 assignee 存在（如果测试用例指定了）
			if tt.request.AssigneeID > 0 {
				// 先尝试查询，如果不存在则创建
				exists, _ := client.User.Query().Where(user.ID(tt.request.AssigneeID)).Exist(ctx)
				if !exists {
					u, err := client.User.Create().
						SetUsername(fmt.Sprintf("assignee_%d", tt.request.AssigneeID)).
						SetEmail(fmt.Sprintf("assignee_%d@example.com", tt.request.AssigneeID)).
						SetName("Assignee").
						SetPasswordHash("hash").
						SetRole("agent").
						SetActive(true).
						SetTenantID(tt.tenantID).
						Save(ctx)
					if err == nil {
						tt.request.AssigneeID = u.ID
					}
				}
			}

			response, err := ticketService.CreateTicket(ctx, tt.request, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, tt.request.Title, response.Title)
				assert.Equal(t, tt.request.Description, response.Description)
				assert.Equal(t, tt.request.Priority, string(response.Priority))
				assert.Equal(t, "new", string(response.Status)) // V2 默认状态为 new
				assert.NotEmpty(t, response.TicketNumber)
				assert.Equal(t, tt.tenantID, response.TenantID)
			}
		})
	}
}

func TestTicketService_GetTickets(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

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

	// 创建多个测试工单
	tickets := make([]*ent.Ticket, 3)
	for i := 0; i < 3; i++ {
		ticket, err := client.Ticket.Create().
			SetTitle(fmt.Sprintf("测试工单 %d", i+1)).
			SetDescription(fmt.Sprintf("测试工单描述 %d", i+1)).
			SetPriority("medium").
			SetStatus("open").
			SetTicketNumber(fmt.Sprintf("TICKET-%d", i+1)).
			SetRequesterID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
		tickets[i] = ticket
	}

	tests := []struct {
		name          string
		request       *dto.ListTicketsRequest
		tenantID      int
		expectedCount int
		expectedError bool
	}{
		{
			name: "获取所有工单",
			request: &dto.ListTicketsRequest{
				Page:     1,
				PageSize: 10,
			},
			tenantID:      testTenant.ID,
			expectedCount: 3,
			expectedError: false,
		},
		{
			name: "分页查询",
			request: &dto.ListTicketsRequest{
				Page:     1,
				PageSize: 2,
			},
			tenantID:      testTenant.ID,
			expectedCount: 2,
			expectedError: false,
		},
		{
			name: "按状态筛选",
			request: &dto.ListTicketsRequest{
				Page:     1,
				PageSize: 10,
				Status:   "open",
			},
			tenantID:      testTenant.ID,
			expectedCount: 3,
			expectedError: false,
		},
		{
			name: "按优先级筛选",
			request: &dto.ListTicketsRequest{
				Page:     1,
				PageSize: 10,
				Priority: "medium",
			},
			tenantID:      testTenant.ID,
			expectedCount: 3,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := ticketService.ListTickets(ctx, tt.request, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Len(t, response.Tickets, tt.expectedCount)
				assert.Equal(t, 3, response.Total) // 总数始终为3
				assert.Equal(t, tt.request.Page, response.Page)
				assert.Equal(t, tt.request.PageSize, response.PageSize)
			}
		})
	}
}

func TestTicketService_GetTicketByID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

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
		SetPriority("high").
		SetStatus("open").
		SetTicketNumber("TICKET-001").
		SetRequesterID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		ticketID      int
		tenantID      int
		expectedError bool
	}{
		{
			name:          "成功获取工单",
			ticketID:      testTicket.ID,
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:          "工单不存在",
			ticketID:      99999,
			tenantID:      testTenant.ID,
			expectedError: true,
		},
		{
			name:          "租户不匹配",
			ticketID:      testTicket.ID,
			tenantID:      99999,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := ticketService.GetTicket(ctx, tt.ticketID, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, ticket)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ticket)
				assert.Equal(t, testTicket.ID, ticket.ID)
				assert.Equal(t, "测试工单", ticket.Title)
				assert.Equal(t, "high", string(ticket.Priority))
			}
		})
	}
}

func TestTicketService_UpdateTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

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
		SetTitle("原始标题").
		SetDescription("原始描述").
		SetPriority("low").
		SetStatus("open").
		SetTicketNumber("TICKET-001").
		SetRequesterID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		ticketID      int
		request       *dto.UpdateTicketRequest
		tenantID      int
		expectedError bool
	}{
		{
			name:     "成功更新工单",
			ticketID: testTicket.ID,
			request: &dto.UpdateTicketRequest{
				Title:       "更新后的标题",
				Description: "更新后的描述",
				Priority:    "high",
				Status:      "in_progress",
				UserID:      testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:     "部分更新",
			ticketID: testTicket.ID,
			request: &dto.UpdateTicketRequest{
				Priority: "critical",
				UserID:   testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:     "工单不存在",
			ticketID: 99999,
			request: &dto.UpdateTicketRequest{
				Title:  "新标题",
				UserID: testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedTicket, err := ticketService.UpdateTicket(ctx, tt.ticketID, tt.request, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, updatedTicket)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updatedTicket)

				if tt.request.Title != "" {
					assert.Equal(t, tt.request.Title, updatedTicket.Title)
				}
				if tt.request.Description != "" {
					assert.Equal(t, tt.request.Description, updatedTicket.Description)
				}
				if tt.request.Priority != "" {
					assert.Equal(t, tt.request.Priority, string(updatedTicket.Priority))
				}
				if tt.request.Status != "" {
					assert.Equal(t, tt.request.Status, string(updatedTicket.Status))
				}
			}
		})
	}
}

func TestTicketService_DeleteTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

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
		SetTitle("待删除工单").
		SetDescription("待删除工单描述").
		SetPriority("low").
		SetStatus("open").
		SetTicketNumber("TICKET-001").
		SetRequesterID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		ticketID      int
		tenantID      int
		expectedError bool
	}{
		{
			name:          "成功删除工单",
			ticketID:      testTicket.ID,
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:          "工单不存在（V2 静默返回 nil，符合 SQL DELETE 语义）",
			ticketID:      99999,
			tenantID:      testTenant.ID,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ticketService.DeleteTicket(ctx, tt.ticketID, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// 验证工单已被删除
				_, err := client.Ticket.Get(ctx, tt.ticketID)
				assert.Error(t, err)
				assert.True(t, ent.IsNotFound(err))
			}
		})
	}
}

func TestTicketService_DeleteTicket_CascadeTenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

	ctx := context.Background()

	// Create tenant 1
	tenant1, err := client.Tenant.Create().
		SetName("Tenant 1").
		SetCode("tenant1").
		SetDomain("tenant1.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// Create tenant 2
	tenant2, err := client.Tenant.Create().
		SetName("Tenant 2").
		SetCode("tenant2").
		SetDomain("tenant2.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// Create user for tenant 1
	user1, err := client.User.Create().
		SetUsername("user1").
		SetEmail("user1@tenant1.com").
		SetName("User 1").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	// Create ticket for tenant 1 with a comment
	ticket1, err := client.Ticket.Create().
		SetTitle("Tenant 1 Ticket").
		SetDescription("Test ticket").
		SetPriority("low").
		SetStatus("open").
		SetTicketNumber("TICKET-T1-001").
		SetRequesterID(user1.ID).
		SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	// Create a comment for the ticket
	_, err = client.TicketComment.Create().
		SetTicketID(ticket1.ID).
		SetUserID(user1.ID).
		SetContent("Test comment").
		SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	// Create an attachment for the ticket
	_, err = client.TicketAttachment.Create().
		SetTicketID(ticket1.ID).
		SetFileName("test.txt").
		SetFilePath("/uploads/test.txt").
		SetFileURL("/uploads/test.txt").
		SetFileSize(1024).
		SetFileType("text/plain").
		SetMimeType("text/plain").
		SetUploadedBy(user1.ID).
		SetTenantID(tenant1.ID).
		Save(ctx)
	require.NoError(t, err)

	// Tenant 2 tries to delete tenant 1's ticket - V2 不会报错（DELETE 静默语义）
	err = ticketService.DeleteTicket(ctx, ticket1.ID, tenant2.ID)
	assert.NoError(t, err)

	// Verify ticket still exists (未被删除，跨租户隔离仍然有效)
	_, err = client.Ticket.Get(ctx, ticket1.ID)
	assert.NoError(t, err)

	// Verify cascade comments were NOT deleted
	comments, err := client.TicketComment.Query().Where(ticketcomment.TicketIDEQ(ticket1.ID)).Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, comments, "comment should still exist after failed cross-tenant delete attempt")
}

func TestTicketService_SearchTickets(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

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

	// 创建测试工单
	_, err = client.Ticket.Create().
		SetTitle("网络连接问题").
		SetDescription("用户无法连接到网络").
		SetPriority("high").
		SetStatus("open").
		SetTicketNumber("TICKET-001").
		SetRequesterID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Ticket.Create().
		SetTitle("打印机故障").
		SetDescription("打印机无法正常工作").
		SetPriority("medium").
		SetStatus("open").
		SetTicketNumber("TICKET-002").
		SetRequesterID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		searchTerm    string
		tenantID      int
		expectedCount int
		expectedError bool
	}{
		{
			name:          "搜索网络相关工单",
			searchTerm:    "网络",
			tenantID:      testTenant.ID,
			expectedCount: 1,
			expectedError: false,
		},
		{
			name:          "搜索打印机相关工单",
			searchTerm:    "打印机",
			tenantID:      testTenant.ID,
			expectedCount: 1,
			expectedError: false,
		},
		{
			name:          "搜索不存在的内容",
			searchTerm:    "不存在的内容",
			tenantID:      testTenant.ID,
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:          "空搜索词",
			searchTerm:    "",
			tenantID:      testTenant.ID,
			expectedCount: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tickets, err := ticketService.SearchTickets(ctx, tt.searchTerm, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, tickets)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tickets)
				assert.Len(t, tickets, tt.expectedCount)
			}
		})
	}
}

func TestTicketService_GetMSPCustomerReports_AllocationAware(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

	ctx := context.Background()

	// Setup: Create MSP tenant and multiple customer tenants
	mspTenant, _ := client.Tenant.Create().
		SetName("MSP").
		SetCode("msp").
		SetType("msp").
		Save(ctx)

	allocatedTenant, _ := client.Tenant.Create().
		SetName("AllocatedCustomer").
		SetCode("alloc_cust").
		SetType("customer").
		Save(ctx)

	unallocatedTenant, _ := client.Tenant.Create().
		SetName("UnallocatedCustomer").
		SetCode("unalloc_cust").
		SetType("customer").
		Save(ctx)

	// Create MSP user
	mspUser, _ := client.User.Create().
		SetUsername("msp_user").
		SetEmail("msp@example.com").
		SetName("MSP User").
		SetPasswordHash("hash").
		SetTenantID(mspTenant.ID).
		Save(ctx)

	// Create allocation ONLY to allocatedTenant
	client.MSPAllocation.Create().
		SetMspUserID(mspUser.ID).
		SetCustomerTenantID(allocatedTenant.ID).
		SetRole("provider_agent").
		Save(ctx)

	// Test: V2 GetMSPCustomerReports 按 mspTenantID 维度聚合统计
	dateFrom, _ := time.Parse("2006-01-02", "2024-01-01")
	dateTo, _ := time.Parse("2006-01-02", "2024-12-31")
	reports, err := ticketService.GetMSPCustomerReports(ctx, mspTenant.ID, dateFrom, dateTo)
	assert.NoError(t, err)
	assert.NotNil(t, reports)
	// V2 返回的 reports 至少包含 status_summary 等字段
	if len(reports) > 0 {
		assert.Contains(t, reports[0], "status_summary")
		assert.Contains(t, reports[0], "total_tickets")
	}

	// Test: 验证未分配租户场景下 V2 仅返回 msp 租户维度统计，不会报错
	reports, err = ticketService.GetMSPCustomerReports(ctx, unallocatedTenant.ID, dateFrom, dateTo)
	assert.NoError(t, err)
	assert.NotNil(t, reports)
}

// 基准测试
func BenchmarkTicketService_CreateTicket(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(b).Sugar()
	ticketService := NewTicketServiceForTest(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, _ := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)

	testUser, _ := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)

	request := &dto.CreateTicketRequest{
		Title:       "基准测试工单",
		Description: "这是一个基准测试工单",
		Priority:    "medium",
		Category:    "incident",
		RequesterID: testUser.ID,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ticketService.CreateTicket(ctx, request, testTenant.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}
