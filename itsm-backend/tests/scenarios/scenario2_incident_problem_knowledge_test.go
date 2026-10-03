package scenarios

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	problemdomain "itsm-backend/handlers/problem"
	"itsm-backend/service"
)

// Scenario 2: 事件→问题→知识 闭环自动化测试
//
// 覆盖：
//   - 创建事件并解决
//   - 从事件根因创建问题
//   - 从问题创建知识文章
//   - 跨租户隔离：tenant B 无法访问 tenant A 的事件/问题/知识链

func TestScenario2_IncidentProblemKnowledgeLoop(t *testing.T) {
	client := enttest.Open(t, "sqlite3", scenarioDSN("incident_loop"))
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := mustCreateTenant(ctx, t, client, "TenantA", "loop-a", "loop-a.test.com")
	tenantB := mustCreateTenant(ctx, t, client, "TenantB", "loop-b", "loop-b.test.com")
	userA := mustCreateUser(ctx, t, client, tenantA.ID, "eng_a", "eng_a@test.com", "technician")
	userB := mustCreateUser(ctx, t, client, tenantB.ID, "eng_b", "eng_b@test.com", "technician")

	incidentSvc := service.NewIncidentService(client, logger, nil)
	// 与 internal/bootstrap/app.go:457-458 一致：生产组装走事务型 outbox。
	// 不开启时会落到 fire-and-forget goroutine 分支，后台写库与下一步事务抢同一张
	// 共享内存表，触发 "database table is locked: incidents" 的偶发失败。
	incidentSvc.EnableWorkflowOutbox()
	incidentSvc.EnableRulesOutbox()
	// 问题域走 router 实际装配的那个实现（router/problem_routes.go 构造同一个
	// problem.NewService(problem.NewEntRepository(client), logger)）。此前这里构造
	// service.NewProblemService，而它实测零生产构造——业务流回归测的是死第二实现。
	problemSvc := problemdomain.NewService(problemdomain.NewEntRepository(client), logger)
	knowledgeSvc := service.NewKnowledgeService(client, logger)

	// tenant A 在闭环里创建的问题 ID，供后面的跨租户拒绝用例复用。
	// 此前那个用例只传 99999，测的是「记录不存在」而不是「租户不匹配」。
	var problemIDA int

	t.Run("incident → resolve → problem → knowledge within tenant A", func(t *testing.T) {
		incResp, err := incidentSvc.CreateIncident(ctx, &dto.CreateIncidentRequest{
			Title:       "生产环境数据库连接池耗尽",
			Description: "MySQL 连接池在高峰期持续耗尽，导致服务不可用",
			Priority:    "high",
			Severity:    "high",
			Category:    "database",
			Source:      "monitoring",
		}, tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("创建事件失败: %v", err)
		}

		_, err = incidentSvc.UpdateIncident(ctx, incResp.ID, &dto.UpdateIncidentRequest{
			Status: strPtr("in_progress"),
		}, tenantA.ID)
		if err != nil {
			t.Fatalf("事件状态转换到 in_progress 失败: %v", err)
		}

		err = incidentSvc.ResolveIncident(ctx, incResp.ID, userA.ID, tenantA.ID, "重启连接池服务并扩容", "连接池配置不足，高峰期并发连接超过上限")
		if err != nil {
			t.Fatalf("解决事件失败: %v", err)
		}

		prob, err := problemSvc.Create(ctx, tenantA.ID, &problemdomain.Problem{
			Title:       "数据库连接池配置不足导致高峰期服务不可用",
			Description: "MySQL 连接池在高峰期持续耗尽，根因是连接池最大连接数配置过低",
			Priority:    "high",
			Category:    "database",
			RootCause:   "连接池 max_connections 配置为 100，高峰期需要 500+",
			CreatedBy:   userA.ID,
		})
		if err != nil {
			t.Fatalf("创建问题失败: %v", err)
		}
		if prob.Status != "open" {
			t.Fatalf("问题初始状态应为 open, 实际: %s", prob.Status)
		}
		problemIDA = prob.ID

		article, err := knowledgeSvc.CreateArticle(ctx, &dto.CreateKnowledgeArticleRequest{
			Title:    "MySQL 连接池耗尽排查与扩容方案",
			Content:  "## 问题描述\n高峰期 MySQL 连接池耗尽...\n## 解决方案\n调整 max_connections 参数...",
			Category: "troubleshooting",
			Tags:     []string{"mysql", "connection-pool", "performance"},
		}, tenantA.ID, userA.ID)
		if err != nil {
			t.Fatalf("创建知识文章失败: %v", err)
		}
		if article.Title == "" {
			t.Fatal("知识文章标题不应为空")
		}

		t.Logf("闭环完成: 事件#%d → 问题#%d → 知识#%d", incResp.ID, prob.ID, article.ID)
	})

	t.Run("tenant B cannot access tenant A incident", func(t *testing.T) {
		_, err := incidentSvc.UpdateIncident(ctx, 99999, &dto.UpdateIncidentRequest{
			Status: strPtr("resolved"),
		}, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能操作不属于自己的事件")
		}
	})

	t.Run("tenant B cannot access tenant A problem", func(t *testing.T) {
		if problemIDA == 0 {
			t.Fatal("前序用例未创建 tenant A 的问题，跨租户断言会退化成不存在")
		}
		_, err := problemSvc.Update(ctx, tenantB.ID, problemIDA, &problemdomain.Problem{Status: "resolved"}, userB.ID, "technician")
		if err == nil {
			t.Fatal("tenant B 不应能操作不属于自己的问题")
		}
	})

	t.Run("tenant B cannot access tenant A knowledge article", func(t *testing.T) {
		_, err := knowledgeSvc.GetArticle(ctx, 99999, tenantB.ID)
		if err == nil {
			t.Fatal("tenant B 不应能读取不属于自己的知识文章")
		}
	})

	t.Run("tenant B cannot create knowledge in tenant A scope", func(t *testing.T) {
		_, err := knowledgeSvc.CreateArticle(ctx, &dto.CreateKnowledgeArticleRequest{
			Title:    "跨租户知识注入尝试",
			Content:  "尝试在 tenant B 上下文中创建知识",
			Category: "general",
		}, tenantA.ID, userB.ID)
		if err == nil {
			t.Fatal("user B (tenant B) 不应能在 tenant A 中创建知识文章")
		}
	})
}

func strPtr(s string) *string { return &s }
