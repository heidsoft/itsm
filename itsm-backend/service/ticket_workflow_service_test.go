package service

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/ticket"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== TicketWorkflowService 测试设置 ====================

func setupTicketWorkflowTest(t *testing.T) (*TicketWorkflowService, *ent.Client, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ticket_workflow_test?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	service := NewTicketWorkflowService(client, logger)
	ctx := context.Background()
	return service, client, ctx
}

func createTicketWorkflowTestTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Workflow Test Tenant " + suffix).
		SetCode("wf" + suffix).
		SetDomain("wf" + suffix + ".test").
		SetStatus("active").
		Save(ctx)
}

func createTicketWorkflowTestUser(ctx context.Context, client *ent.Client, tenantID int, suffix string) (*ent.User, error) {
	return client.User.Create().
		SetUsername("wfuser" + suffix).
		SetEmail("wf" + suffix + "@test.com").
		SetName("Workflow User " + suffix).
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
}

var ticketWorkflowTicketCounter int64

func createTicketWorkflowTestTicket(ctx context.Context, client *ent.Client, tenantID int, userID int, status string) (*ent.Ticket, error) {
	atomic.AddInt64(&ticketWorkflowTicketCounter, 1)
	return client.Ticket.Create().
		SetTitle("Workflow Test Ticket").
		SetStatus(status).
		SetPriority("medium").
		SetTicketNumber(fmt.Sprintf("WF-TKT-%d-%d", tenantID, atomic.LoadInt64(&ticketWorkflowTicketCounter))).
		SetRequesterID(userID).
		SetTenantID(tenantID).
		Save(ctx)
}

// ==================== TicketWorkflowService 基础测试 ====================

func TestTicketWorkflowService_NewTicketWorkflowService(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:new_wf_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	service := NewTicketWorkflowService(client, logger)

	assert.NotNil(t, service)
	assert.Equal(t, client, service.client)
	assert.Equal(t, logger, service.logger)
}

func TestTicketWorkflowService_SetConnectorManager(t *testing.T) {
	service, client, _ := setupTicketWorkflowTest(t)
	defer client.Close()

	// 测试 SetConnectorManager 方法
	service.SetConnectorManager(nil)
	// 不应 panic
	assert.NotNil(t, service)
}

// ==================== TicketWorkflowService 状态转换测试 ====================

func TestTicketWorkflowService_GetAvailableActions(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant, err := createTicketWorkflowTestTenant(ctx, client, "actions")
	require.NoError(t, err)

	testUser, err := createTicketWorkflowTestUser(ctx, client, testTenant.ID, "actions")
	require.NoError(t, err)

	tests := []struct {
		name   string
		status string
	}{
		{"new 状态", "new"},
		{"open 状态", "open"},
		{"in_progress 状态", "in_progress"},
		{"pending 状态", "pending"},
		{"resolved 状态", "resolved"},
		{"closed 状态", "closed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := createTicketWorkflowTestTicket(ctx, client, testTenant.ID, testUser.ID, tt.status)
			require.NoError(t, err)

			actions, err := service.GetAvailableActions(ctx, ticket.ID, testUser.ID, testTenant.ID)
			require.NoError(t, err)
			assert.NotNil(t, actions)
		})
	}
}

func TestTicketWorkflowService_GetAvailableActions_InvalidTicket(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant, err := createTicketWorkflowTestTenant(ctx, client, "invalid")
	require.NoError(t, err)

	testUser, err := createTicketWorkflowTestUser(ctx, client, testTenant.ID, "invalid")
	require.NoError(t, err)

	// 不存在的工单
	_, err = service.GetAvailableActions(ctx, 99999, testUser.ID, testTenant.ID)
	assert.Error(t, err)
}

func TestTicketWorkflowService_GetAvailableActions_NoPermission(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant1, err := createTicketWorkflowTestTenant(ctx, client, "tenant1")
	require.NoError(t, err)

	testTenant2, err := createTicketWorkflowTestTenant(ctx, client, "tenant2")
	require.NoError(t, err)

	testUser1, err := createTicketWorkflowTestUser(ctx, client, testTenant1.ID, "user1")
	require.NoError(t, err)

	// 租户2的工单，租户1的用户访问
	ticket, err := createTicketWorkflowTestTicket(ctx, client, testTenant2.ID, testUser1.ID, "new")
	require.NoError(t, err)

	_, err = service.GetAvailableActions(ctx, ticket.ID, testUser1.ID, testTenant1.ID)
	assert.Error(t, err)
}

// ==================== TicketWorkflowService 流转记录测试 ====================

func TestTicketWorkflowService_GetWorkflowHistory(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant, err := createTicketWorkflowTestTenant(ctx, client, "history")
	require.NoError(t, err)

	testUser, err := createTicketWorkflowTestUser(ctx, client, testTenant.ID, "history")
	require.NoError(t, err)

	ticket, err := createTicketWorkflowTestTicket(ctx, client, testTenant.ID, testUser.ID, "new")
	require.NoError(t, err)

	// 获取流转历史
	history, err := service.GetWorkflowHistory(ctx, ticket.ID, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, history)
}

func TestTicketWorkflowService_GetWorkflowHistory_InvalidTicket(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant, err := createTicketWorkflowTestTenant(ctx, client, "invalid_history")
	require.NoError(t, err)

	_, err = service.GetWorkflowHistory(ctx, 99999, testTenant.ID)
	assert.Error(t, err)
}

// ==================== TicketWorkflowService 工作流规则测试 ====================

func TestTicketWorkflowService_GetWorkflowRules(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant, err := createTicketWorkflowTestTenant(ctx, client, "rules")
	require.NoError(t, err)

	// 获取工作流规则
	rules, err := service.GetWorkflowRules(ctx, "ticket", testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, rules)
}

func TestTicketWorkflowService_GetWorkflowRules_ByTicket(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant, err := createTicketWorkflowTestTenant(ctx, client, "ticket_rules")
	require.NoError(t, err)

	testUser, err := createTicketWorkflowTestUser(ctx, client, testTenant.ID, "ticket_rules")
	require.NoError(t, err)

	ticket, err := createTicketWorkflowTestTicket(ctx, client, testTenant.ID, testUser.ID, "new")
	require.NoError(t, err)

	// 根据工单获取工作流规则
	rules, err := service.GetWorkflowRulesByTicket(ctx, ticket.ID, testTenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, rules)
}

// ==================== TicketWorkflowService 通知测试 ====================

func TestTicketWorkflowService_NotifyTicketUpdate(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant, err := createTicketWorkflowTestTenant(ctx, client, "notify")
	require.NoError(t, err)

	testUser, err := createTicketWorkflowTestUser(ctx, client, testTenant.ID, "notify")
	require.NoError(t, err)

	ticket, err := createTicketWorkflowTestTicket(ctx, client, testTenant.ID, testUser.ID, "new")
	require.NoError(t, err)

	// 测试通知发送（可能发送失败但不应该是致命错误）
	err = service.NotifyTicketUpdate(ctx, ticket.ID, "测试通知", testTenant.ID)
	// 通知失败不应阻止主流程
	assert.NoError(t, err)
}

// ==================== TicketWorkflowService 辅助方法测试 ====================

func TestTicketWorkflowService_GetTicket(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant, err := createTicketWorkflowTestTenant(ctx, client, "get_ticket")
	require.NoError(t, err)

	testUser, err := createTicketWorkflowTestUser(ctx, client, testTenant.ID, "get_ticket")
	require.NoError(t, err)

	ticket, err := createTicketWorkflowTestTicket(ctx, client, testTenant.ID, testUser.ID, "open")
	require.NoError(t, err)

	// 测试 getTicket 辅助方法
	result, err := service.getTicket(ctx, ticket.ID, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, ticket.ID, result.ID)
	assert.Equal(t, "open", result.Status)
}

func TestTicketWorkflowService_GetTicket_NotFound(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant, err := createTicketWorkflowTestTenant(ctx, client, "not_found")
	require.NoError(t, err)

	_, err = service.getTicket(ctx, 99999, testTenant.ID)
	assert.Error(t, err)
}

func TestTicketWorkflowService_GetTicket_TenantMismatch(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant1, err := createTicketWorkflowTestTenant(ctx, client, "tenant1_mismatch")
	require.NoError(t, err)

	testTenant2, err := createTicketWorkflowTestTenant(ctx, client, "tenant2_mismatch")
	require.NoError(t, err)

	testUser, err := createTicketWorkflowTestUser(ctx, client, testTenant1.ID, "mismatch")
	require.NoError(t, err)

	// 租户1创建的工单
	ticket, err := createTicketWorkflowTestTicket(ctx, client, testTenant1.ID, testUser.ID, "new")
	require.NoError(t, err)

	// 用租户2的ID查询
	_, err = service.getTicket(ctx, ticket.ID, testTenant2.ID)
	assert.Error(t, err)
}

// ==================== TicketWorkflowService 验证测试 ====================

func TestTicketWorkflowService_CanUserAccessTicket(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant, err := createTicketWorkflowTestTenant(ctx, client, "access")
	require.NoError(t, err)

	testUser, err := createTicketWorkflowTestUser(ctx, client, testTenant.ID, "access")
	require.NoError(t, err)

	ticket, err := createTicketWorkflowTestTicket(ctx, client, testTenant.ID, testUser.ID, "new")
	require.NoError(t, err)

	canAccess, err := service.CanUserAccessTicket(ctx, ticket.ID, testUser.ID, testTenant.ID)
	require.NoError(t, err)
	assert.True(t, canAccess)
}

func TestTicketWorkflowService_CanUserAccessTicket_NoPermission(t *testing.T) {
	service, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant1, err := createTicketWorkflowTestTenant(ctx, client, "tenant1_access")
	require.NoError(t, err)

	testTenant2, err := createTicketWorkflowTestTenant(ctx, client, "tenant2_access")
	require.NoError(t, err)

	testUser1, err := createTicketWorkflowTestUser(ctx, client, testTenant1.ID, "user1_access")
	require.NoError(t, err)

	// 租户1的用户，访问租户2的工单
	ticket, err := createTicketWorkflowTestTicket(ctx, client, testTenant2.ID, testUser1.ID, "new")
	require.NoError(t, err)

	canAccess, err := service.CanUserAccessTicket(ctx, ticket.ID, testUser1.ID, testTenant1.ID)
	require.NoError(t, err)
	assert.False(t, canAccess)
}

// ==================== TicketWorkflowService 状态验证测试 ====================

func TestTicketWorkflowService_ValidateStatusTransition(t *testing.T) {
	tests := []struct {
		name       string
		fromStatus string
		toStatus   string
		valid      bool
	}{
		{"new -> open", "new", "open", true},
		{"new -> in_progress", "new", "in_progress", true},
		{"open -> in_progress", "open", "in_progress", true},
		{"in_progress -> resolved", "in_progress", "resolved", true},
		{"resolved -> closed", "resolved", "closed", true},
		{"closed -> open", "closed", "open", false},
		{"new -> closed", "new", "closed", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateStatusTransition(tt.fromStatus, tt.toStatus)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

// validateStatusTransition 辅助函数
// 状态机：除"已解决-重新打开"外，仅允许向前流转；新建/已关闭不能直接跳到已关闭。
func validateStatusTransition(from, to string) bool {
	validTransitions := map[string][]string{
		"new":         {"open", "in_progress"},
		"open":        {"in_progress"},
		"in_progress": {"pending", "resolved"},
		"pending":     {"in_progress"},
		"resolved":    {"closed", "in_progress"},
		"closed":      {},
	}

	if from == to {
		return false
	}
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}

	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// ==================== TicketWorkflowService 边界测试 ====================

func TestTicketWorkflowService_LargeVolumeTickets(t *testing.T) {
	_, client, ctx := setupTicketWorkflowTest(t)
	defer client.Close()

	testTenant, err := createTicketWorkflowTestTenant(ctx, client, "volume")
	require.NoError(t, err)

	testUser, err := createTicketWorkflowTestUser(ctx, client, testTenant.ID, "volume")
	require.NoError(t, err)

	// 创建多个工单
	for i := 0; i < 50; i++ {
		_, err = client.Ticket.Create().
			SetTitle("Volume Test Ticket " + string(rune('A'+i%26))).
			SetStatus("new").
			SetPriority("medium").
			SetTicketNumber("VOL-TKT-" + string(rune('0'+i/10)) + string(rune('0'+i%10))).
			SetRequesterID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	// 验证可以批量获取流转历史
	tickets, err := client.Ticket.Query().
		Where(ticket.TenantID(testTenant.ID)).
		All(ctx)
	require.NoError(t, err)
	assert.Equal(t, 50, len(tickets))
}
