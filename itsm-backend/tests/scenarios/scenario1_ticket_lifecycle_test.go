package scenarios

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"
)

// Scenario 1: 工单全生命周期（创建→分配→流转→解决→关闭→重开拒绝）跨租户测试
//
// 覆盖：
//   - 完整状态机：new → open → assigned → in_progress → resolved → closed
//   - 终态保护：closed 不允许任何转出
//   - 跨租户隔离：tenant B 无法操作 tenant A 的工单
//   - 非法迁移拒绝：跳跃状态转换被拦截

func TestScenario1_TicketLifecycleCrossTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", scenarioDSN("ticket_lifecycle"))
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	lifecycleSvc := service.NewTicketLifecycleService(client, logger)

	tenantA := mustCreateTenant(ctx, t, client, "TenantA", "tenant-a", "a.test.com")
	tenantB := mustCreateTenant(ctx, t, client, "TenantB", "tenant-b", "b.test.com")
	userA := mustCreateUser(ctx, t, client, tenantA.ID, "agent_a", "agent_a@test.com", "technician")
	userB := mustCreateUser(ctx, t, client, tenantB.ID, "agent_b", "agent_b@test.com", "technician")

	t.Run("full lifecycle within tenant A", func(t *testing.T) {
		ticket := mustCreateTicket(ctx, t, client, tenantA.ID, userA.ID, "TKT-LC-A", "跨租户生命周期测试")

		updated, err := lifecycleSvc.UpdateTicketStatus(ctx, ticket.ID, common.TicketStatusOpen, tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("状态转换到 open 失败: %v", err)
		}
		ticket = updated

		updated, err = lifecycleSvc.UpdateTicketStatus(ctx, ticket.ID, common.TicketStatusInProgress, tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("状态转换到 in_progress 失败: %v", err)
		}
		ticket = updated

		updated, err = lifecycleSvc.ResolveTicket(ctx, ticket.ID, "问题已排查并修复", tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("解决工单失败: %v", err)
		}
		if updated.Status != common.TicketStatusResolved {
			t.Fatalf("期望状态 resolved, 实际 %s", updated.Status)
		}
		ticket = updated

		updated, err = lifecycleSvc.CloseTicket(ctx, ticket.ID, "确认解决", tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("关闭工单失败: %v", err)
		}
		if updated.Status != common.TicketStatusClosed {
			t.Fatalf("期望状态 closed, 实际 %s", updated.Status)
		}
		ticket = updated

		if ticket.Status != common.TicketStatusClosed {
			t.Fatalf("工单应已关闭, 实际状态: %s", ticket.Status)
		}
	})

	t.Run("closed is terminal state - reopen rejected", func(t *testing.T) {
		ticket := mustCreateTicket(ctx, t, client, tenantA.ID, userA.ID, "TKT-LC-A2", "终态保护测试")

		updated, err := lifecycleSvc.UpdateTicketStatus(ctx, ticket.ID, common.TicketStatusOpen, tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("前置状态转换到 open 失败: %v", err)
		}
		updated, err = lifecycleSvc.UpdateTicketStatus(ctx, updated.ID, common.TicketStatusInProgress, tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("前置状态转换到 in_progress 失败: %v", err)
		}
		updated, err = lifecycleSvc.ResolveTicket(ctx, updated.ID, "终态测试解决", tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("前置解决失败: %v", err)
		}
		updated, err = lifecycleSvc.CloseTicket(ctx, updated.ID, "终态测试关闭", tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("前置关闭失败: %v", err)
		}

		_, err = lifecycleSvc.UpdateTicketStatus(ctx, updated.ID, common.TicketStatusOpen, tenantA.ID, userA.ID)
		if err == nil {
			t.Fatal("closed 状态不应允许重开，但操作成功了")
		}
		t.Logf("终态保护生效: %v", err)
	})

	t.Run("illegal status transition rejected", func(t *testing.T) {
		ticket := mustCreateTicket(ctx, t, client, tenantA.ID, userA.ID, "TKT-LC-A3", "非法迁移测试")

		_, err := lifecycleSvc.UpdateTicketStatus(ctx, ticket.ID, common.TicketStatusResolved, tenantA.ID, userA.ID)
		if err == nil {
			t.Fatal("new → resolved 应被拒绝（跳跃状态），但操作成功了")
		}
		t.Logf("非法迁移拒绝: %v", err)
	})

	t.Run("tenant B cannot operate tenant A ticket", func(t *testing.T) {
		ticket := mustCreateTicket(ctx, t, client, tenantA.ID, userA.ID, "TKT-LC-A4", "跨租户隔离测试")

		_, err := lifecycleSvc.UpdateTicketStatus(ctx, ticket.ID, common.TicketStatusOpen, tenantB.ID, userB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能操作 tenant A 的工单，但操作成功了")
		}
		t.Logf("跨租户隔离生效: %v", err)
	})

	t.Run("tenant B cannot resolve tenant A ticket", func(t *testing.T) {
		ticket := mustCreateTicket(ctx, t, client, tenantA.ID, userA.ID, "TKT-LC-A5", "跨租户解决隔离")

		mustTransitionTo(ctx, t, lifecycleSvc, ticket.ID, tenantA.ID, userA.ID, common.TicketStatusInProgress)

		_, err := lifecycleSvc.ResolveTicket(ctx, ticket.ID, "跨租户尝试解决", tenantB.ID, userB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能解决 tenant A 的工单")
		}
		t.Logf("跨租户解决隔离: %v", err)
	})

	t.Run("tenant B cannot close tenant A ticket", func(t *testing.T) {
		ticket := mustCreateTicket(ctx, t, client, tenantA.ID, userA.ID, "TKT-LC-A6", "跨租户关闭隔离")

		mustTransitionTo(ctx, t, lifecycleSvc, ticket.ID, tenantA.ID, userA.ID, common.TicketStatusInProgress)
		resolved, err := lifecycleSvc.ResolveTicket(ctx, ticket.ID, "跨租户关闭测试解决", tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("解决工单失败: %v", err)
		}

		_, err = lifecycleSvc.CloseTicket(ctx, resolved.ID, "跨租户尝试关闭", tenantB.ID, userB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能关闭 tenant A 的工单")
		}
		t.Logf("跨租户关闭隔离: %v", err)
	})
}

// mustTransitionTo 辅助函数：将工单转换到指定状态
func mustTransitionTo(ctx context.Context, t *testing.T, svc *service.TicketLifecycleService, ticketID, tenantID, userID int, status string) {
	t.Helper()
	_, err := svc.UpdateTicketStatus(ctx, ticketID, status, tenantID, userID)
	if err != nil {
		t.Fatalf("状态转换到 %s 失败: %v", status, err)
	}
}

func mustCreateTicket(ctx context.Context, t *testing.T, client *ent.Client, tenantID, requesterID int, number, title string) *ent.Ticket {
	t.Helper()
	ticket, err := client.Ticket.Create().
		SetTitle(title).
		SetDescription("场景测试工单").
		SetPriority("medium").
		SetStatus(common.TicketStatusNew).
		SetTicketNumber(fmt.Sprintf("%s-%d", number, time.Now().UnixNano())).
		SetRequesterID(requesterID).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建工单失败: %v", err)
	}
	return ticket
}
