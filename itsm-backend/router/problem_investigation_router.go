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
		problemInvestigationGroup.POST("/investigations", problemInvestigationController.CreateProblemInvestigation)
		problemInvestigationGroup.GET("/investigations/:id", problemInvestigationController.GetProblemInvestigation)
		problemInvestigationGroup.PUT("/investigations/:id", problemInvestigationController.UpdateProblemInvestigation)

		// 调查步骤管理
		problemInvestigationGroup.POST("/steps", problemInvestigationController.CreateInvestigationStep)
		problemInvestigationGroup.PUT("/steps/:id", problemInvestigationController.UpdateInvestigationStep)
		problemInvestigationGroup.GET("/investigations/:investigation_id/steps", problemInvestigationController.GetInvestigationSteps)

		// 根本原因分析
		problemInvestigationGroup.POST("/root-cause-analysis", problemInvestigationController.CreateRootCauseAnalysis)

		// 问题解决方案
		problemInvestigationGroup.POST("/solutions", problemInvestigationController.CreateProblemSolution)
		problemInvestigationGroup.GET("/problems/:id/solutions", problemInvestigationController.GetProblemSolutions)

		// 问题调查摘要
		problemInvestigationGroup.GET("/problems/:id/summary", problemInvestigationController.GetProblemInvestigationSummary)
	}
}
