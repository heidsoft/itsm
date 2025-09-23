package service

import (
	"context"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/user"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestTicketService_CreateTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := &TicketService{
		client: client,
		logger: logger,
	}

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetRole("user").
		SetStatus("active").
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
				Type:        "incident",
				Source:      "web",
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
				Type:        "incident",
				Source:      "web",
				RequesterID: testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: true,
		},
		{
			name: "描述为空",
			request: &dto.CreateTicketRequest{
				Title:       "标题",
				Description: "",
				Priority:    "medium",
				Type:        "incident",
				Source:      "web",
				RequesterID: testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: true,
		},
		{
			name: "无效的优先级",
			request: &dto.CreateTicketRequest{
				Title:       "标题",
				Description: "描述",
				Priority:    "invalid",
				Type:        "incident",
				Source:      "web",
				RequesterID: testUser.ID,
			},
			tenantID:      testTenant.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := ticketService.CreateTicket(ctx, tt.request, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, tt.request.Title, response.Title)
				assert.Equal(t, tt.request.Description, response.Description)
				assert.Equal(t, tt.request.Priority, response.Priority)
				assert.Equal(t, "open", response.Status) // 默认状态
				assert.NotEmpty(t, response.IncidentNumber)
				assert.Equal(t, tt.tenantID, response.TenantID)
			}
		})
	}
}

func TestTicketService_GetTickets(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := &TicketService{
		client: client,
		logger: logger,
	}

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetRole("user").
		SetStatus("active").
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
			SetType("incident").
			SetSource("web").
			SetIncidentNumber(fmt.Sprintf("INC-%d", i+1)).
			SetRequesterID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
		tickets[i] = ticket
	}

	tests := []struct {
		name          string
		request       *dto.GetTicketsRequest
		tenantID      int
		expectedCount int
		expectedError bool
	}{
		{
			name: "获取所有工单",
			request: &dto.GetTicketsRequest{
				Page:     1,
				PageSize: 10,
			},
			tenantID:      testTenant.ID,
			expectedCount: 3,
			expectedError: false,
		},
		{
			name: "分页查询",
			request: &dto.GetTicketsRequest{
				Page:     1,
				PageSize: 2,
			},
			tenantID:      testTenant.ID,
			expectedCount: 2,
			expectedError: false,
		},
		{
			name: "按状态筛选",
			request: &dto.GetTicketsRequest{
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
			request: &dto.GetTicketsRequest{
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
			response, err := ticketService.GetTickets(ctx, tt.request, tt.tenantID)

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
	ticketService := &TicketService{
		client: client,
		logger: logger,
	}

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetRole("user").
		SetStatus("active").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	testTicket, err := client.Ticket.Create().
		SetTitle("测试工单").
		SetDescription("测试工单描述").
		SetPriority("high").
		SetStatus("open").
		SetType("incident").
		SetSource("web").
		SetIncidentNumber("INC-001").
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
			ticket, err := ticketService.GetTicketByID(ctx, tt.ticketID, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, ticket)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ticket)
				assert.Equal(t, testTicket.ID, ticket.ID)
				assert.Equal(t, "测试工单", ticket.Title)
				assert.Equal(t, "high", ticket.Priority)
			}
		})
	}
}

func TestTicketService_UpdateTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := &TicketService{
		client: client,
		logger: logger,
	}

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetRole("user").
		SetStatus("active").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	testTicket, err := client.Ticket.Create().
		SetTitle("原始标题").
		SetDescription("原始描述").
		SetPriority("low").
		SetStatus("open").
		SetType("incident").
		SetSource("web").
		SetIncidentNumber("INC-001").
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
					assert.Equal(t, tt.request.Priority, updatedTicket.Priority)
				}
				if tt.request.Status != "" {
					assert.Equal(t, tt.request.Status, updatedTicket.Status)
				}
			}
		})
	}
}

func TestTicketService_DeleteTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := &TicketService{
		client: client,
		logger: logger,
	}

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetRole("user").
		SetStatus("active").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	testTicket, err := client.Ticket.Create().
		SetTitle("待删除工单").
		SetDescription("待删除工单描述").
		SetPriority("low").
		SetStatus("open").
		SetType("incident").
		SetSource("web").
		SetIncidentNumber("INC-001").
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
			name:          "工单不存在",
			ticketID:      99999,
			tenantID:      testTenant.ID,
			expectedError: true,
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

func TestTicketService_SearchTickets(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	ticketService := &TicketService{
		client: client,
		logger: logger,
	}

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetRole("user").
		SetStatus("active").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建测试工单
	_, err = client.Ticket.Create().
		SetTitle("网络连接问题").
		SetDescription("用户无法连接到网络").
		SetPriority("high").
		SetStatus("open").
		SetType("incident").
		SetSource("web").
		SetIncidentNumber("INC-001").
		SetRequesterID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Ticket.Create().
		SetTitle("打印机故障").
		SetDescription("打印机无法正常工作").
		SetPriority("medium").
		SetStatus("open").
		SetType("incident").
		SetSource("web").
		SetIncidentNumber("INC-002").
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

// 基准测试
func BenchmarkTicketService_CreateTicket(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(b).Sugar()
	ticketService := &TicketService{
		client: client,
		logger: logger,
	}

	ctx := context.Background()

	// 创建测试数据
	testTenant, _ := client.Tenant.Create().
		SetName("Test Tenant").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)

	testUser, _ := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetRole("user").
		SetStatus("active").
		SetTenantID(testTenant.ID).
		Save(ctx)

	request := &dto.CreateTicketRequest{
		Title:       "基准测试工单",
		Description: "这是一个基准测试工单",
		Priority:    "medium",
		Type:        "incident",
		Source:      "web",
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