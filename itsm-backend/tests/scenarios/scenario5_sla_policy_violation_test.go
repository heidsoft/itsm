package scenarios

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"
)

// Scenario 5: SLA 策略绑定 + 违规统计 + 升级矩阵测试
//
// 覆盖：
//   - SLA 策略创建与匹配
//   - SLA 违规检查（超时工单触发违规）
//   - 升级矩阵：按优先级查找下一级升级
//   - 合规率计算
//   - 跨租户 SLA 策略隔离

func TestScenario5_SLAPolicyViolationEscalation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", scenarioDSN("sla_policy"))
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := mustCreateTenant(ctx, t, client, "TenantA", "sla-a", "sla-a.test.com")
	tenantB := mustCreateTenant(ctx, t, client, "TenantB", "sla-b", "sla-b.test.com")

	policySvc := service.NewSLAPolicyService(client)
	monitorSvc := service.NewSLAMonitorService(client, logger)
	escalationSvc := service.NewEscalationMatrixService(logger)

	t.Run("create SLA policy and match by priority", func(t *testing.T) {
		policy, err := policySvc.CreateSLAPolicy(ctx, dto.CreateSLAPolicyRequest{
			Name:                  "黄金客户高优先级 SLA",
			Description:           "黄金客户高优先级事件的响应和解决时限",
			CustomerTier:          strPtr("gold"),
			TicketType:            strPtr("incident"),
			Priority:              strPtr("high"),
			ResponseTimeMinutes:   30,
			ResolutionTimeMinutes: 240,
			IsActive:              true,
			PriorityScore:         80,
			TenantID:              tenantA.ID,
		})
		if err != nil {
			t.Fatalf("创建 SLA 策略失败: %v", err)
		}
		if policy.ID == 0 {
			t.Fatal("SLA 策略 ID 不应为 0")
		}

		matched, err := policySvc.MatchSLAPolicy(ctx, tenantA.ID, "incident", "high", "gold")
		if err != nil {
			t.Fatalf("匹配 SLA 策略失败: %v", err)
		}
		if matched.ID != policy.ID {
			t.Fatalf("匹配到的策略 ID 不匹配: 期望 %d, 实际 %d", policy.ID, matched.ID)
		}
		if matched.ResponseTimeMinutes != 30 {
			t.Fatalf("响应时间应为 30 分钟, 实际: %d", matched.ResponseTimeMinutes)
		}
		t.Logf("SLA 策略匹配成功: %s (响应 %d 分钟, 解决 %d 分钟)",
			matched.Name, matched.ResponseTimeMinutes, matched.ResolutionTimeMinutes)
	})

	t.Run("SLA violation check with overdue tickets", func(t *testing.T) {
		userA := mustCreateUser(ctx, t, client, tenantA.ID, "sla_eng", "sla_eng@test.com", "technician")

		slaDef, err := client.SLADefinition.Create().
			SetName("响应时间SLA").
			SetDescription("测试用 SLA 定义").
			SetPriority("high").
			SetResponseTime(30).
			SetResolutionTime(240).
			SetTenantID(tenantA.ID).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建 SLA 定义失败: %v", err)
		}

		overdueCreatedAt := time.Now().Add(-5 * time.Hour)
		_, err = client.Ticket.Create().
			SetTitle("超时事件 - 数据库宕机").
			SetDescription("数据库服务已宕机 5 小时").
			SetPriority("high").
			SetType("incident").
			SetStatus("open").
			SetTicketNumber("TKT-SLA-OVERDUE-001").
			SetRequesterID(userA.ID).
			SetTenantID(tenantA.ID).
			SetCreatedAt(overdueCreatedAt).
			SetSLADefinitionID(slaDef.ID).
			SetSLAResponseDeadline(time.Now().Add(-2 * time.Hour)).
			SetSLAResolutionDeadline(time.Now().Add(-1 * time.Hour)).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建超时工单失败: %v", err)
		}

		stats, err := monitorSvc.CheckSLAViolations(ctx, tenantA.ID)
		if err != nil {
			t.Fatalf("SLA 违规检查失败: %v", err)
		}
		if stats.TotalChecked < 1 {
			t.Fatalf("应至少检查 1 个工单, 实际: %d", stats.TotalChecked)
		}
		t.Logf("SLA 违规检查: 检查 %d 工单, 新违规 %d, 已有违规 %d",
			stats.TotalChecked, stats.NewViolations, stats.ExistingViolations)
	})

	t.Run("escalation matrix - critical priority 3-level escalation", func(t *testing.T) {
		lvl1 := escalationSvc.FindNextEscalationLevel(tenantA.ID, "critical", 15, 0)
		if lvl1 == nil {
			t.Fatal("critical 15 分钟应触发 L1 升级")
		}
		if lvl1.Level != 1 {
			t.Fatalf("期望 L1, 实际 L%d", lvl1.Level)
		}
		if len(lvl1.NotifyRoles) == 0 || lvl1.NotifyRoles[0] != "team_lead" {
			t.Fatalf("L1 应通知 team_lead, 实际: %v", lvl1.NotifyRoles)
		}

		lvl2 := escalationSvc.FindNextEscalationLevel(tenantA.ID, "critical", 30, 1)
		if lvl2 == nil {
			t.Fatal("critical 30 分钟 currentLevel=1 应触发 L2 升级")
		}
		if lvl2.Level != 2 {
			t.Fatalf("期望 L2, 实际 L%d", lvl2.Level)
		}

		lvl3 := escalationSvc.FindNextEscalationLevel(tenantA.ID, "critical", 60, 2)
		if lvl3 == nil {
			t.Fatal("critical 60 分钟 currentLevel=2 应触发 L3 升级")
		}
		if lvl3.Level != 3 {
			t.Fatalf("期望 L3, 实际 L%d", lvl3.Level)
		}

		lvl4 := escalationSvc.FindNextEscalationLevel(tenantA.ID, "critical", 120, 3)
		if lvl4 != nil {
			t.Fatal("critical 所有升级级别已完成后不应再升级")
		}
		t.Log("升级矩阵验证: critical → L1(15min) → L2(30min) → L3(60min) → 终止")
	})

	t.Run("escalation matrix - high priority 2-level escalation", func(t *testing.T) {
		lvl1 := escalationSvc.FindNextEscalationLevel(tenantA.ID, "high", 60, 0)
		if lvl1 == nil || lvl1.Level != 1 {
			t.Fatal("high 60 分钟应触发 L1")
		}

		lvl2 := escalationSvc.FindNextEscalationLevel(tenantA.ID, "high", 240, 1)
		if lvl2 == nil || lvl2.Level != 2 {
			t.Fatal("high 240 分钟 currentLevel=1 应触发 L2")
		}

		lvl3 := escalationSvc.FindNextEscalationLevel(tenantA.ID, "high", 480, 2)
		if lvl3 != nil {
			t.Fatal("high 所有升级级别已完成后不应再升级")
		}
		t.Log("升级矩阵验证: high → L1(60min) → L2(240min) → 终止")
	})

	t.Run("SLA compliance rate with no tickets returns 100", func(t *testing.T) {
		rate, err := policySvc.GetSLAComplianceRate(ctx, tenantB.ID, time.Now().Add(-24*time.Hour), time.Now())
		if err != nil {
			t.Fatalf("获取合规率失败: %v", err)
		}
		if rate != 100.0 {
			t.Fatalf("无工单时合规率应为 100%%, 实际: %.2f%%", rate)
		}
		t.Logf("tenant B 合规率: %.2f%%", rate)
	})

	t.Run("tenant B cannot access tenant A SLA policy", func(t *testing.T) {
		policies, err := policySvc.QuerySLAPolicies(ctx, tenantA.ID)
		if err != nil {
			t.Fatalf("查询 tenant A 策略失败: %v", err)
		}
		if len(policies) == 0 {
			t.Skip("tenant A 无策略，跳过跨租户测试")
		}

		policyID := policies[0].ID
		_, err = policySvc.GetSLAPolicyByIDForTenant(ctx, policyID, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能读取 tenant A 的 SLA 策略")
		}
		t.Logf("SLA 策略跨租户隔离: %v", err)
	})

	t.Run("tenant B cannot update tenant A SLA policy", func(t *testing.T) {
		policies, err := policySvc.QuerySLAPolicies(ctx, tenantA.ID)
		if err != nil {
			t.Fatalf("查询 tenant A 策略失败: %v", err)
		}
		if len(policies) == 0 {
			t.Skip("tenant A 无策略，跳过跨租户更新测试")
		}

		policyID := policies[0].ID
		newDesc := "被 tenant B 篡改的描述"
		_, err = policySvc.UpdateSLAPolicyForTenant(ctx, policyID, tenantB.ID, dto.UpdateSLAPolicyRequest{
			Description: &newDesc,
		})
		if err == nil {
			t.Fatal("tenant B 不应能更新 tenant A 的 SLA 策略")
		}
		t.Logf("SLA 策略跨租户更新隔离: %v", err)
	})
}
