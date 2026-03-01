package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestTicketLifecycleService_ResolveTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	lifecycleService := NewTicketLifecycleService(client, logger)

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
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		ticketID      int
		tenantID      int
		resolution    string
		resolvedBy    int
		expectedError bool
		expectedStatus string
	}{
		{
			name:           "成功解决工单",
			ticketID:       testTicket.ID,
			tenantID:       testTenant.ID,
			resolution:      "问题已解决",
			resolvedBy:     testUser.ID,
			expectedError:  false,
			expectedStatus: "resolved",
		},
		{
			name:           "工单不存在",
			ticketID:       99999,
			tenantID:       testTenant.ID,
			resolution:      "问题已解决",
			resolvedBy:     testUser.ID,
			expectedError:  true,
			expectedStatus: "",
		},
		{
			name:           "租户不匹配",
			ticketID:       testTicket.ID,
			tenantID:       99999,
			resolution:      "问题已解决",
			resolvedBy:     testUser.ID,
			expectedError:  true,
			expectedStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 为每个测试创建新的工单状态
			if tt.ticketID == testTicket.ID && !tt.expectedError {
				// 更新为 open 状态以便测试
				client.Ticket.UpdateOneID(testTicket.ID).
					SetStatus("open").
					Exec(ctx)
			}

			updatedTicket, err := lifecycleService.ResolveTicket(ctx, tt.ticketID, tt.resolution, tt.tenantID, tt.resolvedBy)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, updatedTicket)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updatedTicket)
				assert.Equal(t, tt.expectedStatus, updatedTicket.Status)
				assert.NotNil(t, updatedTicket.ResolvedAt)
			}
		})
	}
}

func TestTicketLifecycleService_CloseTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	lifecycleService := NewTicketLifecycleService(client, logger)

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

	tests := []struct {
		name          string
		setupStatus   string
		ticketTitle   string
		tenantID      int
		closedBy      int
		feedback      string
		expectedError bool
		expectedStatus string
	}{
		{
			name:           "成功关闭已解决的工单",
			setupStatus:    "resolved",
			ticketTitle:    "测试工单1",
			tenantID:       testTenant.ID,
			closedBy:       testUser.ID,
			feedback:       "满意",
			expectedError:  false,
			expectedStatus: "closed",
		},
		{
			name:           "成功关闭已关闭的工单",
			setupStatus:    "closed",
			ticketTitle:    "测试工单2",
			tenantID:       testTenant.ID,
			closedBy:       testUser.ID,
			feedback:       "满意",
			expectedError:  false,
			expectedStatus: "closed",
		},
		{
			name:           "无法关闭开放状态的工单",
			setupStatus:    "open",
			ticketTitle:    "测试工单3",
			tenantID:       testTenant.ID,
			closedBy:       testUser.ID,
			feedback:       "",
			expectedError:  true,
			expectedStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试工单
			testTicket, err := client.Ticket.Create().
				SetTitle(tt.ticketTitle).
				SetDescription("测试描述").
				SetPriority("medium").
				SetStatus(tt.setupStatus).
				SetTicketNumber("TICKET-" + tt.ticketTitle).
				SetRequesterID(testUser.ID).
				SetTenantID(tt.tenantID).
				Save(ctx)
			require.NoError(t, err)

			updatedTicket, err := lifecycleService.CloseTicket(ctx, testTicket.ID, tt.feedback, tt.tenantID, tt.closedBy)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, updatedTicket)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updatedTicket)
				assert.Equal(t, tt.expectedStatus, updatedTicket.Status)
			}
		})
	}
}

func TestTicketLifecycleService_UpdateTicketStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	lifecycleService := NewTicketLifecycleService(client, logger)

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
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		ticketID      int
		newStatus    string
		tenantID      int
		operatorID   int
		expectedError bool
		expectedStatus string
	}{
		{
			name:           "成功更新状态 open -> in_progress",
			ticketID:       testTicket.ID,
			newStatus:      "in_progress",
			tenantID:       testTenant.ID,
			operatorID:    testUser.ID,
			expectedError:  false,
			expectedStatus: "in_progress",
		},
		{
			name:           "成功更新状态 in_progress -> resolved",
			ticketID:       testTicket.ID,
			newStatus:      "resolved",
			tenantID:       testTenant.ID,
			operatorID:    testUser.ID,
			expectedError:  false,
			expectedStatus: "resolved",
		},
		{
			name:           "无效状态转换 closed -> open",
			ticketID:       testTicket.ID,
			newStatus:      "open",
			tenantID:       testTenant.ID,
			operatorID:    testUser.ID,
			expectedError:  true,
			expectedStatus: "",
		},
		{
			name:           "工单不存在",
			ticketID:       99999,
			newStatus:      "in_progress",
			tenantID:       testTenant.ID,
			operatorID:    testUser.ID,
			expectedError:  true,
			expectedStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 根据测试用例设置初始状态
			initialStatus := "open"
			if tt.newStatus == "resolved" {
				// resolved 需要先到 in_progress
				initialStatus = "in_progress"
			}
			client.Ticket.UpdateOneID(testTicket.ID).
				SetStatus(initialStatus).
				Exec(ctx)

			updatedTicket, err := lifecycleService.UpdateTicketStatus(ctx, tt.ticketID, tt.newStatus, tt.tenantID, tt.operatorID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, updatedTicket)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updatedTicket)
				assert.Equal(t, tt.expectedStatus, updatedTicket.Status)
			}
		})
	}
}

func TestTicketLifecycleService_EscalateTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	lifecycleService := NewTicketLifecycleService(client, logger)

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
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name           string
		ticketID       int
		tenantID       int
		escalatedBy    int
		reason         string
		initialPriority string
		expectedError  bool
		expectedPriority string
	}{
		{
			name:            "成功升级工单 low -> medium",
			ticketID:        testTicket.ID,
			tenantID:        testTenant.ID,
			escalatedBy:     testUser.ID,
			reason:          "需要更快处理",
			initialPriority: "low",
			expectedError:   false,
			expectedPriority: "medium",
		},
		{
			name:            "成功升级工单 medium -> high",
			ticketID:        testTicket.ID,
			tenantID:        testTenant.ID,
			escalatedBy:     testUser.ID,
			reason:          "需要更快处理",
			initialPriority: "medium",
			expectedError:   false,
			expectedPriority: "high",
		},
		{
			name:            "成功升级工单 high -> critical",
			ticketID:        testTicket.ID,
			tenantID:        testTenant.ID,
			escalatedBy:     testUser.ID,
			reason:          "需要更快处理",
			initialPriority: "high",
			expectedError:   false,
			expectedPriority: "critical",
		},
		{
			name:            "critical 优先级不再升级",
			ticketID:        testTicket.ID,
			tenantID:        testTenant.ID,
			escalatedBy:     testUser.ID,
			reason:          "需要更快处理",
			initialPriority: "critical",
			expectedError:   false,
			expectedPriority: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置工单状态和优先级
			client.Ticket.UpdateOneID(testTicket.ID).
				SetStatus("open").
				SetPriority(tt.initialPriority).
				Exec(ctx)

			updatedTicket, err := lifecycleService.EscalateTicket(ctx, tt.ticketID, tt.reason, tt.tenantID, tt.escalatedBy)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, updatedTicket)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updatedTicket)
				assert.Equal(t, tt.expectedPriority, updatedTicket.Priority)
			}
		})
	}
}

func TestTicketLifecycleService_isValidStatusTransition(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	lifecycleService := NewTicketLifecycleService(client, logger)

	tests := []struct {
		currentStatus string
		newStatus    string
		expected     bool
	}{
		{"open", "in_progress", true},
		{"open", "pending", true},
		{"open", "closed", true},
		{"in_progress", "resolved", true},
		{"in_progress", "pending", true},
		{"in_progress", "open", true},
		{"pending", "in_progress", true},
		{"pending", "resolved", true},
		{"pending", "open", true},
		{"resolved", "closed", true},
		{"resolved", "open", true},
		{"closed", "open", false},
		{"closed", "in_progress", false},
		{"open", "resolved", false},
		{"unknown", "open", false},
	}

	for _, tt := range tests {
		t.Run(tt.currentStatus+"_"+tt.newStatus, func(t *testing.T) {
			result := lifecycleService.isValidStatusTransition(tt.currentStatus, tt.newStatus)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTicketLifecycleService_mapProcessStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	lifecycleService := NewTicketLifecycleService(client, logger)

	tests := []struct {
		processStatus string
		expectedTicketStatus string
	}{
		{"completed", "resolved"},
		{"terminated", "closed"},
		{"cancelled", "closed"},
		{"running", "in_progress"},
		{"active", "in_progress"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.processStatus, func(t *testing.T) {
			result := lifecycleService.mapProcessStatus(tt.processStatus)
			assert.Equal(t, tt.expectedTicketStatus, result)
		})
	}
}

func TestTicketLifecycleService_CancelWorkflow(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	lifecycleService := NewTicketLifecycleService(client, logger)

	ctx := context.Background()

	// 创建测试数据
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试工单
	testTicket, err := client.Ticket.Create().
		SetTitle("测试工单").
		SetDescription("测试工单描述").
		SetPriority("medium").
		SetStatus("open").
		SetTicketNumber("TICKET-001").
		SetTenantID(testTenant.ID).
		SetRequesterID(1).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		ticketID      int
		tenantID      int
		reason        string
		setupWorkflow bool
		expectedError bool
	}{
		{
			name:          "取消不存在的工单工作流",
			ticketID:      99999,
			tenantID:      testTenant.ID,
			reason:        "测试",
			setupWorkflow: false,
			expectedError: false,
		},
		{
			name:          "工单存在但无工作流",
			ticketID:      testTicket.ID,
			tenantID:      testTenant.ID,
			reason:        "测试",
			setupWorkflow: false,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := lifecycleService.CancelWorkflow(ctx, tt.ticketID, tt.tenantID, tt.reason)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
