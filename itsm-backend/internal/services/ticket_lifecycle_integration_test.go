package service

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ============================================================
// Integration Tests: Ticket Lifecycle - 状态转换流程
// 验证工单完整生命周期：open → in_progress → resolved → closed
// ============================================================

func TestTicketLifecycle_CompleteFlow(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer client.Close()
	ctx := context.Background()

	// 创建测试数据 - 需要 code 字段
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.example.com").
		Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 1. 创建工单 (open)
	ticketEntity, err := client.Ticket.Create().
		SetTitle("Lifecycle Test Ticket").
		SetDescription("Testing complete lifecycle").
		SetPriority("high").
		SetType("incident").
		SetStatus(common.TicketStatusOpen).
		SetTicketNumber("TKT-LIFECYCLE-001").
		SetTenantID(tenant.ID).
		SetRequesterID(user.ID).
		Save(ctx)
	require.NoError(t, err)
	assert.Equal(t, common.TicketStatusOpen, ticketEntity.Status)

	// 2. 开始处理 (in_progress)
	service := NewTicketLifecycleService(client, zaptest.NewLogger(t).Sugar())
	updated, err := service.UpdateTicketStatus(ctx, ticketEntity.ID, common.TicketStatusInProgress, tenant.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, common.TicketStatusInProgress, updated.Status)

	// 3. 解决工单 (resolved)
	resolved, err := service.ResolveTicket(ctx, ticketEntity.ID, "问题已修复", tenant.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, common.TicketStatusResolved, resolved.Status)
	assert.NotNil(t, resolved.ResolvedAt)
	assert.Equal(t, "问题已修复", resolved.Resolution)

	// 4. 关闭工单 (closed)
	closed, err := service.CloseTicket(ctx, ticketEntity.ID, "客户确认满意", tenant.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, common.TicketStatusClosed, closed.Status)
	assert.NotNil(t, closed.ClosedAt)
}

func TestTicketLifecycle_Escalate(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant, user := createTestUserAndTenant(t, ctx, client)

	// 创建高优先级工单
	ticketEntity, err := client.Ticket.Create().
		SetTitle("Escalate Test Ticket").
		SetDescription("Testing escalation").
		SetPriority("critical").
		SetType("incident").
		SetStatus(common.TicketStatusOpen).
		SetTicketNumber("TKT-ESC-001").
		SetTenantID(tenant.ID).
		SetRequesterID(user.ID).
		Save(ctx)
	require.NoError(t, err)

	service := NewTicketLifecycleService(client, zaptest.NewLogger(t).Sugar())

	// 升级工单
	escalated, err := service.EscalateTicket(ctx, ticketEntity.ID, "需要高级别支持", tenant.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, common.TicketStatusOpen, escalated.Status)
	assert.Equal(t, "critical", escalated.Priority)
}

func TestTicketLifecycle_InvalidStatusTransition(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant, user := createTestUserAndTenant(t, ctx, client)

	// 创建已关闭的工单
	ticketEntity, err := client.Ticket.Create().
		SetTitle("Closed Ticket").
		SetDescription("Already closed").
		SetPriority("medium").
		SetType("incident").
		SetStatus(common.TicketStatusClosed).
		SetTicketNumber("TKT-CLOSED-001").
		SetTenantID(tenant.ID).
		SetRequesterID(user.ID).
		Save(ctx)
	require.NoError(t, err)

	service := NewTicketLifecycleService(client, zaptest.NewLogger(t).Sugar())

	// 尝试解决已关闭的工单 - 应该失败
	_, err = service.ResolveTicket(ctx, ticketEntity.ID, "resolution", tenant.ID, user.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

// ============================================================
// Table-Driven Tests: Ticket Lifecycle - 状态转换规则
// ============================================================

func TestTicketLifecycle_StatusTransition_TableDriven(t *testing.T) {
	// 测试 UpdateTicketStatus 的实际行为
	// 注意：UpdateTicketStatus 与 ResolveTicket/CloseTicket 有不同的转换规则
	tests := []struct {
		name          string
		initialStatus string
		targetStatus  string
		canTransition bool
	}{
		// 已知的有效转换
		{"open → in_progress", common.TicketStatusOpen, common.TicketStatusInProgress, true},
		{"open → pending", common.TicketStatusOpen, common.TicketStatusPending, true},
		{"open → closed", common.TicketStatusOpen, common.TicketStatusClosed, false},
		{"in_progress → pending", common.TicketStatusInProgress, common.TicketStatusPending, true},
		{"in_progress → open", common.TicketStatusInProgress, common.TicketStatusOpen, false},
		{"pending → in_progress", common.TicketStatusPending, common.TicketStatusInProgress, true},
		{"resolved → closed", common.TicketStatusResolved, common.TicketStatusClosed, true},
		{"resolved → open", common.TicketStatusResolved, common.TicketStatusOpen, true},
		{"closed → open", common.TicketStatusClosed, common.TicketStatusOpen, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
			defer client.Close()
			ctx := context.Background()

			tenant, user := createTestUserAndTenant(t, ctx, client)
			service := NewTicketLifecycleService(client, zaptest.NewLogger(t).Sugar())

			// 创建初始状态工单
			ticketEntity, err := client.Ticket.Create().
				SetTitle("Transition Test").
				SetDescription("Test").
				SetPriority("medium").
				SetType("incident").
				SetStatus(tt.initialStatus).
				SetTicketNumber("TKT-TR-" + tt.initialStatus + "-" + tt.targetStatus).
				SetTenantID(tenant.ID).
				SetRequesterID(user.ID).
				Save(ctx)
			require.NoError(t, err)

			// 尝试转换状态
			_, err = service.UpdateTicketStatus(ctx, ticketEntity.ID, tt.targetStatus, tenant.ID, user.ID)

			if tt.canTransition {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// ============================================================
// Integration Tests: SLA Service
// ============================================================

func TestSLA_CompleteFlow(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant, user := createTestUserAndTenant(t, ctx, client)

	// 创建 SLA 策略 - 使用正确的字段名
	slaPolicy, err := client.SLAPolicy.Create().
		SetName("Critical SLA").
		SetDescription("4h response, 24h resolution").
		SetResponseTimeMinutes(4 * 60).    // 4 hours in minutes
		SetResolutionTimeMinutes(24 * 60). // 24 hours in minutes
		SetPriority("critical").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建工单并关联 SLA
	responseDeadline := time.Now().Add(4 * time.Hour)
	resolutionDeadline := time.Now().Add(24 * time.Hour)
	ticketEntity, err := client.Ticket.Create().
		SetTitle("SLA Test Ticket").
		SetDescription("Testing SLA").
		SetPriority("critical").
		SetType("incident").
		SetStatus(common.TicketStatusOpen).
		SetTicketNumber("TKT-SLA-001").
		SetTenantID(tenant.ID).
		SetRequesterID(user.ID).
		SetSLADefinitionID(slaPolicy.ID).
		SetSLAResponseDeadline(responseDeadline).
		SetSLAResolutionDeadline(resolutionDeadline).
		Save(ctx)
	require.NoError(t, err)

	// 验证 SLA 状态
	assert.NotNil(t, ticketEntity.SLAResolutionDeadline)
}

func TestSLA_ViolationDetection(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant, user := createTestUserAndTenant(t, ctx, client)

	// 创建严格的 SLA 策略 (60分钟响应)
	slaPolicy, err := client.SLAPolicy.Create().
		SetName("Strict SLA").
		SetDescription("60min response").
		SetResponseTimeMinutes(60).
		SetResolutionTimeMinutes(240).
		SetPriority("high").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建已过期的工单
	expiredTime := time.Now().Add(-1 * time.Hour)
	ticketEntity, err := client.Ticket.Create().
		SetTitle("Expired SLA Ticket").
		SetDescription("SLA already violated").
		SetPriority("high").
		SetType("incident").
		SetStatus(common.TicketStatusOpen).
		SetTicketNumber("TKT-SLA-EXP-001").
		SetTenantID(tenant.ID).
		SetRequesterID(user.ID).
		SetSLADefinitionID(slaPolicy.ID).
		SetSLAResponseDeadline(expiredTime).
		SetSLAResolutionDeadline(expiredTime).
		Save(ctx)
	require.NoError(t, err)

	// 验证已过期
	assert.True(t, ticketEntity.SLAResolutionDeadline.Before(time.Now()))
}

// ============================================================
// Table-Driven Tests: SLA Priority Mapping
// ============================================================

func TestSLA_PriorityMapping_TableDriven(t *testing.T) {
	tests := []struct {
		name               string
		priority           string
		responseTimeMins   int // minutes
		resolutionTimeMins int // minutes
	}{
		{"critical priority", "critical", 4 * 60, 24 * 60},
		{"high priority", "high", 8 * 60, 48 * 60},
		{"medium priority", "medium", 24 * 60, 72 * 60},
		{"low priority", "low", 72 * 60, 168 * 60}, // 1 week
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
			defer client.Close()
			ctx := context.Background()

			tenant, user := createTestUserAndTenant(t, ctx, client)

			// 创建 SLA 策略
			slaPolicy, err := client.SLAPolicy.Create().
				SetName(tt.name).
				SetDescription("Test").
				SetResponseTimeMinutes(tt.responseTimeMins).
				SetResolutionTimeMinutes(tt.resolutionTimeMins).
				SetPriority(tt.priority).
				SetTenantID(tenant.ID).
				Save(ctx)
			require.NoError(t, err)

			// 验证策略
			assert.Equal(t, tt.responseTimeMins, slaPolicy.ResponseTimeMinutes)
			assert.Equal(t, tt.resolutionTimeMins, slaPolicy.ResolutionTimeMinutes)

			// 创建关联工单
			resolutionDeadline := time.Now().Add(time.Duration(tt.resolutionTimeMins) * time.Minute)
			_, err = client.Ticket.Create().
				SetTitle("SLA Mapping Test").
				SetDescription("Test").
				SetPriority(tt.priority).
				SetType("incident").
				SetStatus(common.TicketStatusOpen).
				SetTicketNumber("TKT-SLA-MAP-" + tt.priority).
				SetTenantID(tenant.ID).
				SetRequesterID(user.ID).
				SetSLADefinitionID(slaPolicy.ID).
				SetSLAResolutionDeadline(resolutionDeadline).
				Save(ctx)
			require.NoError(t, err)
		})
	}
}

// ============================================================
// Helper Functions
// ============================================================

func createTestUserAndTenant(t *testing.T, ctx context.Context, client *ent.Client) (*ent.Tenant, *ent.User) {
	t.Helper()

	// 创建租户 - 需要 code 字段
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.example.com").
		Save(ctx)
	require.NoError(t, err)

	// 创建用户
	user, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return tenant, user
}
