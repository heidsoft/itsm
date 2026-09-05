package router

import (
	"github.com/gin-gonic/gin"

	"itsm-backend/handlers/problem_investigation"
	"itsm-backend/middleware"
)

// SetupProblemInvestigationRoutes 注册问题调查域路由。
// 从 router.go 的集中注册块抽取而来，路由路径/方法/中间件与抽取前逐行一致。
// 权限说明：investigation/step/root_cause/solution 等资源未在 seeder 中定义，
// 统一复用已 seeded 且已赋权运营角色的 problem 权限，避免注册后全员 403。
func SetupProblemInvestigationRoutes(tenant *gin.RouterGroup, h *problem_investigation.Handler) {
	problemInvestigation := tenant.Group("/problem-investigation")
	{
		// 问题调查管理
		problemInvestigation.POST("/investigations", middleware.RequirePermission("problem", "write"), h.CreateProblemInvestigation)
		problemInvestigation.GET("/investigations/:id", middleware.RequirePermission("problem", "read"), h.GetProblemInvestigation)
		problemInvestigation.PUT("/investigations/:id", middleware.RequirePermission("problem", "write"), h.UpdateProblemInvestigation)

		// 调查步骤管理
		problemInvestigation.POST("/steps", middleware.RequirePermission("problem", "write"), h.CreateInvestigationStep)
		problemInvestigation.PUT("/steps/:id", middleware.RequirePermission("problem", "write"), h.UpdateInvestigationStep)
		// Gin 路由树不允许同一位置出现不同参数名（:id vs :investigation_id 会 panic），
		// 统一使用 :id，与上方 /investigations/:id 保持一致。
		problemInvestigation.GET("/investigations/:id/steps", middleware.RequirePermission("problem", "read"), h.GetInvestigationSteps)

		// 根本原因分析
		problemInvestigation.POST("/root-cause-analysis", middleware.RequirePermission("problem", "write"), h.CreateRootCauseAnalysis)
		problemInvestigation.PUT("/root-cause-analysis/:id", middleware.RequirePermission("problem", "write"), h.UpdateRootCauseAnalysis)

		// 解决方案管理
		problemInvestigation.POST("/solutions", middleware.RequirePermission("problem", "write"), h.CreateProblemSolution)
		problemInvestigation.PUT("/solutions/:id", middleware.RequirePermission("problem", "write"), h.UpdateProblemSolution)
		problemInvestigation.GET("/problems/:id/solutions", middleware.RequirePermission("problem", "read"), h.GetProblemSolutions)

		// 问题调查摘要
		problemInvestigation.GET("/problems/:id/summary", middleware.RequirePermission("problem", "read"), h.GetProblemInvestigationSummary)
	}

	// 关联管理与知识沉淀（前端契约：/api/v1/problem-relationships、/api/v1/problem-knowledge-articles）
	tenant.POST("/problem-relationships", middleware.RequirePermission("problem", "write"), h.CreateProblemRelationship)
	tenant.POST("/problem-knowledge-articles", middleware.RequirePermission("problem", "write"), h.CreateKnowledgeArticle)
	tenant.GET("/problem-knowledge-articles/problems/:id", middleware.RequirePermission("problem", "read"), h.GetProblemKnowledgeArticles)
}
