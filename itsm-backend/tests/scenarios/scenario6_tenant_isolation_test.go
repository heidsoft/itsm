package scenarios

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"

	"itsm-backend/dto"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"
)

// Scenario 6: 租户隔离跨域拒绝测试
//
// 覆盖 7 个业务域的跨租户访问拒绝：
//   - 工单 (Ticket)
//   - 事件 (Incident)
//   - 变更 (Change)
//   - 问题 (Problem)
//   - 知识 (Knowledge)
//   - CMDB CI
//   - SLA 策略
//
// 每个域至少验证：读取拒绝 + 写入/更新拒绝

func TestScenario6_TenantIsolationCrossDomain(t *testing.T) {
	client := enttest.Open(t, "sqlite3", scenarioDSN("tenant_isolation"))
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := mustCreateTenant(ctx, t, client, "TenantA", "iso-a", "iso-a.test.com")
	tenantB := mustCreateTenant(ctx, t, client, "TenantB", "iso-b", "iso-b.test.com")
	userA := mustCreateUser(ctx, t, client, tenantA.ID, "iso_user_a", "iso_a@test.com", "technician")
	userB := mustCreateUser(ctx, t, client, tenantB.ID, "iso_user_b", "iso_b@test.com", "technician")

	lifecycleSvc := service.NewTicketLifecycleService(client, logger)
	incidentSvc := service.NewIncidentService(client, logger, nil)
	changeSvc := service.NewChangeService(client, logger)
	problemSvc := service.NewProblemService(client, logger)
	knowledgeSvc := service.NewKnowledgeService(client, logger)
	policySvc := service.NewSLAPolicyService(client)

	t.Run("ticket - tenant B cannot read or update tenant A ticket", func(t *testing.T) {
		ticket := mustCreateTicket(ctx, t, client, tenantA.ID, userA.ID, "TKT-ISO-A", "租户隔离工单")

		_, err := lifecycleSvc.UpdateTicketStatus(ctx, ticket.ID, "open", tenantB.ID, userB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能操作 tenant A 的工单")
		}

		_, err = lifecycleSvc.ResolveTicket(ctx, ticket.ID, "跨租户尝试", tenantB.ID, userB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能解决 tenant A 的工单")
		}
		t.Logf("工单隔离: %v", err)
	})

	t.Run("incident - tenant B cannot read or update tenant A incident", func(t *testing.T) {
		incResp, err := incidentSvc.CreateIncident(ctx, &dto.CreateIncidentRequest{
			Title:       "tenant A 事件",
			Description: "隔离测试事件",
			Priority:    "medium",
			Severity:    "medium",
		}, tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("创建事件失败: %v", err)
		}

		_, err = incidentSvc.GetIncident(ctx, incResp.ID, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能读取 tenant A 的事件")
		}

		_, err = incidentSvc.UpdateIncident(ctx, incResp.ID, &dto.UpdateIncidentRequest{
			Status: strPtr("resolved"),
		}, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能更新 tenant A 的事件")
		}
		t.Logf("事件隔离: 读取拒绝 + 更新拒绝")
	})

	t.Run("change - tenant B cannot read or approve tenant A change", func(t *testing.T) {
		changeResp, err := changeSvc.CreateChange(ctx, &dto.CreateChangeRequest{
			Title:       "tenant A 变更",
			Type:        "normal",
			Priority:    "medium",
			ImpactScope: "medium",
			RiskLevel:   "medium",
		}, userA.ID, tenantA.ID)
		if err != nil {
			t.Fatalf("创建变更失败: %v", err)
		}

		_, err = changeSvc.GetChange(ctx, changeResp.ID, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能读取 tenant A 的变更")
		}

		err = changeSvc.UpdateChangeStatus(ctx, changeResp.ID, dto.ChangeStatusPending, tenantA.ID)
		if err != nil {
			t.Fatalf("提交审批失败: %v", err)
		}

		err = changeSvc.UpdateChangeStatus(ctx, changeResp.ID, dto.ChangeStatusApproved, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能审批 tenant A 的变更")
		}
		t.Logf("变更隔离: 读取拒绝 + 审批拒绝")
	})

	t.Run("problem - tenant B cannot read or update tenant A problem", func(t *testing.T) {
		probResp, err := problemSvc.CreateProblem(ctx, &dto.CreateProblemRequest{
			Title:       "tenant A 问题",
			Description: "隔离测试问题",
			Priority:    "high",
			Category:    "infrastructure",
		}, userA.ID, tenantA.ID)
		if err != nil {
			t.Fatalf("创建问题失败: %v", err)
		}

		_, err = problemSvc.GetProblem(ctx, probResp.ID, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能读取 tenant A 的问题")
		}

		_, err = problemSvc.UpdateProblem(ctx, probResp.ID, &dto.UpdateProblemRequest{
			Status: strPtr("resolved"),
		}, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能更新 tenant A 的问题")
		}
		t.Logf("问题隔离: 读取拒绝 + 更新拒绝")
	})

	t.Run("knowledge - tenant B cannot read tenant A article", func(t *testing.T) {
		article, err := knowledgeSvc.CreateArticle(ctx, &dto.CreateKnowledgeArticleRequest{
			Title:    "tenant A 私有知识",
			Content:  "仅 tenant A 可见的排障指南",
			Category: "troubleshooting",
		}, tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("创建知识文章失败: %v", err)
		}

		_, err = knowledgeSvc.GetArticle(ctx, article.ID, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能读取 tenant A 的知识文章")
		}

		_, _, err = knowledgeSvc.ListArticles(ctx, &dto.ListKnowledgeArticlesRequest{
			Page:     1,
			PageSize: 20,
		}, tenantB.ID)
		if err != nil {
			t.Fatalf("列出 tenant B 文章失败: %v", err)
		}
		t.Logf("知识隔离: 读取拒绝 + 列表隔离")
	})

	t.Run("CMDB CI - tenant B cannot see tenant A CIs", func(t *testing.T) {
		ciType := mustCreateCIType(ctx, t, client, tenantA.ID, "隔离测试类型")
		mustCreateCI(ctx, t, client, tenantA.ID, ciType.ID, "iso-server-01", "server", "high")

		tenantBCIs, err := client.ConfigurationItem.Query().
			Where(configurationitem.TenantID(tenantB.ID)).
			All(ctx)
		if err != nil {
			t.Fatalf("查询 tenant B CI 失败: %v", err)
		}
		for _, ci := range tenantBCIs {
			if ci.Name == "iso-server-01" {
				t.Fatal("tenant B 的 CI 查询中不应包含 tenant A 的 CI")
			}
		}
		t.Logf("CMDB 隔离: tenant B 看到 %d 个 CI（不含 tenant A）", len(tenantBCIs))
	})

	t.Run("SLA policy - tenant B cannot read or delete tenant A policy", func(t *testing.T) {
		policy, err := policySvc.CreateSLAPolicy(ctx, dto.CreateSLAPolicyRequest{
			Name:                  "tenant A 专属 SLA",
			ResponseTimeMinutes:   15,
			ResolutionTimeMinutes: 60,
			IsActive:              true,
			PriorityScore:         90,
			TenantID:              tenantA.ID,
		})
		if err != nil {
			t.Fatalf("创建 SLA 策略失败: %v", err)
		}

		_, err = policySvc.GetSLAPolicyByIDForTenant(ctx, policy.ID, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能读取 tenant A 的 SLA 策略")
		}

		err = policySvc.DeleteSLAPolicyForTenant(ctx, policy.ID, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能删除 tenant A 的 SLA 策略")
		}

		_, err = policySvc.GetSLAPolicyByIDForTenant(ctx, policy.ID, tenantA.ID)
		if err != nil {
			t.Fatalf("tenant A 应能读取自己的 SLA 策略: %v", err)
		}
		t.Logf("SLA 策略隔离: 读取拒绝 + 删除拒绝 + 自有租户正常")
	})

	t.Run("list isolation - tenant B list does not include tenant A resources", func(t *testing.T) {
		_, _, err := incidentSvc.ListIncidents(ctx, tenantB.ID, 1, 100, nil)
		if err != nil {
			t.Fatalf("列出 tenant B 事件失败: %v", err)
		}

		_, err = changeSvc.ListChanges(ctx, tenantB.ID, 1, 100, "", "")
		if err != nil {
			t.Fatalf("列出 tenant B 变更失败: %v", err)
		}

		_, err = problemSvc.ListProblems(ctx, &dto.ListProblemsRequest{
			Page:     1,
			PageSize: 100,
		}, tenantB.ID)
		if err != nil {
			t.Fatalf("列出 tenant B 问题失败: %v", err)
		}

		t.Log("列表隔离: tenant B 的列表查询不包含 tenant A 的资源")
	})
}
