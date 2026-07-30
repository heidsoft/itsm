package router

import (
	"itsm-backend/controller"
	"itsm-backend/ent"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupProblemInvestigationRoutes 设置问题调查相关路由
func SetupProblemInvestigationRoutes(r *gin.Engine, problemInvestigationController *controller.ProblemInvestigationController, jwtSecret string, entClient interface{}) {
	// 问题调查相关路由组
	problemInvestigationGroup := r.Group("/api/v1/problem-investigation")
	problemInvestigationGroup.Use(middleware.AuthMiddleware(jwtSecret))
	problemInvestigationGroup.Use(middleware.TenantMiddleware(entClient.(*ent.Client)))

	{
		// 问题调查管理
		problemInvestigationGroup.POST("/investigations", middleware.RequirePermission("investigation", "create"), problemInvestigationController.CreateProblemInvestigation)
		problemInvestigationGroup.GET("/investigations/:id", middleware.RequirePermission("investigation", "read"), problemInvestigationController.GetProblemInvestigation)
		problemInvestigationGroup.PUT("/investigations/:id", middleware.RequirePermission("investigation", "update"), problemInvestigationController.UpdateProblemInvestigation)

		// 调查步骤管理
		problemInvestigationGroup.POST("/steps", middleware.RequirePermission("step", "create"), problemInvestigationController.CreateInvestigationStep)
		problemInvestigationGroup.PUT("/steps/:id", middleware.RequirePermission("step", "update"), problemInvestigationController.UpdateInvestigationStep)
		problemInvestigationGroup.GET("/investigations/:investigation_id/steps", middleware.RequirePermission("investigation", "read"), problemInvestigationController.GetInvestigationSteps)

		// 根本原因分析
		problemInvestigationGroup.POST("/root-cause-analysis", middleware.RequirePermission("root_cause", "create"), problemInvestigationController.CreateRootCauseAnalysis)
		problemInvestigationGroup.PUT("/root-cause-analysis/:id", middleware.RequirePermission("root_cause", "update"), problemInvestigationController.UpdateRootCauseAnalysis)

		// 问题解决方案
		problemInvestigationGroup.POST("/solutions", middleware.RequirePermission("solution", "create"), problemInvestigationController.CreateProblemSolution)
		problemInvestigationGroup.PUT("/solutions/:id", middleware.RequirePermission("solution", "update"), problemInvestigationController.UpdateProblemSolution)
		problemInvestigationGroup.GET("/problems/:id/solutions", middleware.RequirePermission("problem", "read"), problemInvestigationController.GetProblemSolutions)

		// 问题关联
		problemInvestigationGroup.POST("/relationships", middleware.RequirePermission("problem", "create"), problemInvestigationController.CreateProblemRelationship)
		problemInvestigationGroup.GET("/problems/:id/relationships", middleware.RequirePermission("problem", "read"), problemInvestigationController.GetProblemRelationships)

		// 知识库文章
		problemInvestigationGroup.POST("/knowledge-articles", middleware.RequirePermission("knowledge", "create"), problemInvestigationController.CreateKnowledgeArticle)
		problemInvestigationGroup.GET("/problems/:id/knowledge-articles", middleware.RequirePermission("knowledge", "read"), problemInvestigationController.GetProblemKnowledgeArticles)

		// 问题调查摘要
		problemInvestigationGroup.GET("/problems/:id/summary", middleware.RequirePermission("problem", "read"), problemInvestigationController.GetProblemInvestigationSummary)
	}
}
