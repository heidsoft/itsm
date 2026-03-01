package bpmn

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestTicketServiceTaskHandler_UpdateTicketStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	handler := NewTicketServiceTaskHandler(client, logger)

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
		variables     map[string]interface{}
		expectedStatus string
		expectedError  bool
	}{
		{
			name:      "更新状态为 in_progress",
			ticketID:  testTicket.ID,
			variables: map[string]interface{}{
				"business_id": testTicket.ID,
				"action":      "update_status",
				"new_status":  "in_progress",
			},
			expectedStatus: "in_progress",
			expectedError:  false,
		},
		{
			name:      "更新状态为 resolved",
			ticketID:  testTicket.ID,
			variables: map[string]interface{}{
				"business_id": testTicket.ID,
				"action":      "update_status",
				"new_status":  "resolved",
			},
			expectedStatus: "resolved",
			expectedError:  false,
		},
		{
			name:      "使用默认状态",
			ticketID:  testTicket.ID,
			variables: map[string]interface{}{
				"business_id": testTicket.ID,
				"action":      "update_status",
			},
			expectedStatus: "in_progress",
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.updateTicketStatus(ctx, tt.ticketID, tt.variables)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.Success)

				// 验证工单状态已更新
				updatedTicket, err := client.Ticket.Get(ctx, tt.ticketID)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, updatedTicket.Status)
			}
		})
	}
}

func TestTicketServiceTaskHandler_EscalateTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	handler := NewTicketServiceTaskHandler(client, logger)

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
		variables     map[string]interface{}
		expectedPriority string
		expectedError  bool
	}{
		{
			name:      "升级工单到 high",
			ticketID:  testTicket.ID,
			variables: map[string]interface{}{
				"business_id":       testTicket.ID,
				"action":            "escalate",
				"escalate_to":       "high",
				"escalation_reason": "需要更快处理",
			},
			expectedPriority: "high",
			expectedError:    false,
		},
		{
			name:      "升级工单到 critical",
			ticketID:  testTicket.ID,
			variables: map[string]interface{}{
				"business_id":       testTicket.ID,
				"action":            "escalate",
				"escalate_to":       "critical",
				"escalation_reason": "紧急问题",
			},
			expectedPriority: "critical",
			expectedError:    false,
		},
		{
			name:      "使用默认升级优先级",
			ticketID:  testTicket.ID,
			variables: map[string]interface{}{
				"business_id": testTicket.ID,
				"action":      "escalate",
			},
			expectedPriority: "high",
			expectedError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置工单优先级
			client.Ticket.UpdateOneID(testTicket.ID).
				SetPriority("medium").
				Exec(ctx)

			result, err := handler.escalateTicket(ctx, tt.ticketID, tt.variables)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.Success)

				// 验证工单优先级已更新
				updatedTicket, err := client.Ticket.Get(ctx, tt.ticketID)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPriority, updatedTicket.Priority)
			}
		})
	}
}

func TestTicketServiceTaskHandler_AssignTicket(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	handler := NewTicketServiceTaskHandler(client, logger)

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

	assignee, err := client.User.Create().
		SetUsername("assignee").
		SetEmail("assignee@example.com").
		SetName("Assignee User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
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
		variables     map[string]interface{}
		expectedAssigneeID int
		expectedError  bool
	}{
		{
			name:      "分配工单给处理人",
			ticketID:  testTicket.ID,
			variables: map[string]interface{}{
				"business_id": testTicket.ID,
				"action":      "assign",
				"assignee_id": float64(assignee.ID),
			},
			expectedAssigneeID: assignee.ID,
			expectedError:      false,
		},
		{
			name:      "使用 int 类型分配",
			ticketID:  testTicket.ID,
			variables: map[string]interface{}{
				"business_id": testTicket.ID,
				"action":      "assign",
				"assignee_id": assignee.ID,
			},
			expectedAssigneeID: assignee.ID,
			expectedError:      false,
		},
		{
			name:      "未指定处理人",
			ticketID:  testTicket.ID,
			variables: map[string]interface{}{
				"business_id": testTicket.ID,
				"action":      "assign",
			},
			expectedAssigneeID: 0,
			expectedError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置工单分配
			client.Ticket.UpdateOneID(testTicket.ID).
				SetAssigneeID(0).
				SetStatus("open").
				Exec(ctx)

			result, err := handler.assignTicket(ctx, tt.ticketID, tt.variables)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.Success)

				// 验证工单已被分配
				updatedTicket, err := client.Ticket.Get(ctx, tt.ticketID)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAssigneeID, updatedTicket.AssigneeID)
			}
		})
	}
}

func TestTicketServiceTaskHandler_Execute(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	handler := NewTicketServiceTaskHandler(client, logger)

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
		variables     map[string]interface{}
		expectedError bool
		checkResult   func(*testing.T, *dto.ServiceTaskResult)
	}{
		{
			name: "执行 update_status 动作",
			variables: map[string]interface{}{
				"business_id": testTicket.ID,
				"action":      "update_status",
				"new_status":  "in_progress",
			},
			expectedError: false,
			checkResult: func(t *testing.T, result *dto.ServiceTaskResult) {
				assert.True(t, result.Success)
			},
		},
		{
			name: "执行 escalate 动作",
			variables: map[string]interface{}{
				"business_id":       testTicket.ID,
				"action":            "escalate",
				"escalate_to":       "high",
				"escalation_reason": "测试升级",
			},
			expectedError: false,
			checkResult: func(t *testing.T, result *dto.ServiceTaskResult) {
				assert.True(t, result.Success)
			},
		},
		{
			name: "执行 assign 动作",
			variables: map[string]interface{}{
				"business_id": testTicket.ID,
				"action":      "assign",
				"assignee_id": float64(testUser.ID),
			},
			expectedError: false,
			checkResult: func(t *testing.T, result *dto.ServiceTaskResult) {
				assert.True(t, result.Success)
			},
		},
		{
			name: "无效的 business_id",
			variables: map[string]interface{}{
				"business_id": 99999,
				"action":      "update_status",
			},
			expectedError: true,
			checkResult:   nil,
		},
		{
			name: "默认动作",
			variables: map[string]interface{}{
				"business_id": testTicket.ID,
				"action":      "unknown_action",
			},
			expectedError: false,
			checkResult: func(t *testing.T, result *dto.ServiceTaskResult) {
				assert.True(t, result.Success)
				assert.Equal(t, "无操作执行", result.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置工单状态
			client.Ticket.UpdateOneID(testTicket.ID).
				SetStatus("open").
				SetPriority("medium").
				SetAssigneeID(0).
				Exec(ctx)

			result, err := handler.Execute(ctx, nil, tt.variables)

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

func TestTicketServiceTaskHandler_GetTaskType(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	handler := NewTicketServiceTaskHandler(client, logger)

	taskType := handler.GetTaskType()
	assert.Equal(t, "ticket_task", taskType)
}

func TestTicketServiceTaskHandler_GetHandlerID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	handler := NewTicketServiceTaskHandler(client, logger)

	handlerID := handler.GetHandlerID()
	assert.Equal(t, "ticket_service_handler", handlerID)
}

func TestTicketServiceTaskHandler_Validate(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	handler := NewTicketServiceTaskHandler(client, logger)

	ctx := context.Background()

	// 测试配置验证
	config := map[string]interface{}{
		"action":     "update_status",
		"new_status": "in_progress",
	}

	err := handler.Validate(ctx, config)
	assert.NoError(t, err)
}
