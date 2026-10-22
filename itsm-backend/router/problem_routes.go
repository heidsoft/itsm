package router

import (
	problemHandler "itsm-backend/handlers/problem"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupProblemRoutes 注册问题管理相关路由。
func SetupProblemRoutes(tenant *gin.RouterGroup, h *problemHandler.Handler) {
	problems := tenant.Group("/problems")
	{
		problems.GET("", middleware.RequirePermission("problem", "read"), h.List)
		problems.POST("", middleware.RequirePermission("problem", "write"), h.Create)
		problems.GET("/stats", middleware.RequirePermission("problem", "read"), h.GetStats)
		problems.GET("/trend", middleware.RequirePermission("problem", "read"), h.GetTrends)
		problems.GET("/hotspots", middleware.RequirePermission("problem", "read"), h.GetHotspots)
		problems.GET("/:id", middleware.RequirePermission("problem", "read"), h.Get)
		problems.PUT("/:id", middleware.RequirePermission("problem", "write"), h.Update)
		problems.DELETE("/:id", middleware.RequirePermission("problem", "delete"), h.Delete)
		problems.POST("/:id/investigate", middleware.RequirePermission("problem", "write"), h.InvestigateProblem)
		problems.PUT("/:id/root-cause", middleware.RequirePermission("problem", "write"), h.UpdateRootCause)
		problems.PUT("/:id/solution", middleware.RequirePermission("problem", "write"), h.UpdateSolution)
		problems.POST("/:id/close", middleware.RequirePermission("problem", "write"), h.CloseProblem)
		problems.GET("/:id/sla", middleware.RequirePermission("problem", "read"), h.GetProblemSLA)
		problems.GET("/:id/comments", middleware.RequirePermission("problem", "read"), h.GetProblemComments)
		problems.POST("/:id/comments", middleware.RequirePermission("problem", "write"), h.AddProblemComment)
		// 知识沉淀闭环：动作语义是「创建知识」，写权限用 knowledge:write，
		// 不因入口挂在问题域而放宽；反查走 knowledge:read 同理。
		problems.POST("/:id/knowledge", middleware.RequirePermission("knowledge", "write"), h.PromoteProblemKnowledge)
		problems.GET("/:id/knowledge", middleware.RequirePermission("knowledge", "read"), h.ListProblemKnowledge)
		// 关联管理
		problems.GET("/:id/associations", middleware.RequirePermission("problem", "read"), h.GetAssociations)
		problems.POST("/:id/associations", middleware.RequirePermission("problem", "write"), h.AddAssociation)
		problems.DELETE("/:id/associations", middleware.RequirePermission("problem", "write"), h.RemoveAssociation)
		// 问题调查关联列表（前端契约：GET /api/v1/problems/:id/relationships）
		// 注意：ProblemInvestigationHandler 在调用处通过 config 访问，需在 router.go 中保持该逻辑
	}
}
