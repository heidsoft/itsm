package router

import (
	incidentHandler "itsm-backend/handlers/incident"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupIncidentRoutes 注册事件管理相关路由。
// 从 router.go 的集中注册块抽取而来，路由路径/方法/中间件与抽取前逐行一致。
func SetupIncidentRoutes(tenant *gin.RouterGroup, h *incidentHandler.IncidentHandler) {
	inc := tenant.Group("/incidents")
	{
		// 核心 CRUD
		inc.GET("", middleware.RequirePermission("incident", "read"), h.Lists)
		inc.POST("", middleware.RequirePermission("incident", "write"), h.Create)
		inc.GET("/stats", middleware.RequirePermission("incident", "read"), h.GetStats)
		inc.GET("/:id", middleware.RequirePermission("incident", "read"), h.Get)
		inc.PUT("/:id", middleware.RequirePermission("incident", "write"), h.Update)
		inc.DELETE("/:id", middleware.RequirePermission("incident", "delete"), h.Delete)

		// 事件操作
		inc.POST("/:id/escalate", middleware.RequirePermission("incident", "write"), h.Escalate)
		inc.POST("/:id/acknowledge", middleware.RequirePermission("incident", "write"), h.Acknowledge)
		inc.POST("/:id/resolve", middleware.RequirePermission("incident", "write"), h.Resolve)
		inc.POST("/:id/close", middleware.RequirePermission("incident", "write"), h.Close)
		inc.POST("/:id/reopen", middleware.RequirePermission("incident", "write"), h.Reopen)
		inc.POST("/:id/assign", middleware.RequirePermission("incident", "assign"), h.Assign)
		inc.PUT("/:id/sla/pause", middleware.RequirePermission("incident", "write"), h.PauseSLA)
		inc.PUT("/:id/sla/resume", middleware.RequirePermission("incident", "write"), h.ResumeSLA)
		inc.POST("/:id/major-incident", middleware.RequirePermission("incident", "write"), h.EscalateMajor)
		inc.POST("/:id/convert-to-problem", middleware.RequirePermission("incident", "write"), h.ConvertToProblem)
		inc.GET("/:id/impact", middleware.RequirePermission("incident", "read"), h.AnalyzeImpact)

		// 关联数据
		inc.GET("/:id/events", middleware.RequirePermission("incident", "read"), h.GetEvents)
		inc.POST("/events", middleware.RequirePermission("incident", "write"), h.CreateEvent)
		inc.GET("/:id/alerts", middleware.RequirePermission("incident", "read"), h.GetAlerts)
		inc.GET("/:id/metrics", middleware.RequirePermission("incident", "read"), h.GetMetrics)

		// 根因分析
		inc.POST("/root-cause", middleware.RequirePermission("incident", "write"), withIncidentIDParam(h.UpdateRootCause))
		inc.GET("/:id/root-cause", middleware.RequirePermission("incident", "read"), h.GetRootCause)
		inc.POST("/:id/root-cause", middleware.RequirePermission("incident", "write"), h.UpdateRootCause)
		inc.PUT("/:id/root-cause", middleware.RequirePermission("incident", "write"), h.UpdateRootCause)

		// 影响评估
		inc.POST("/impact-assessment", middleware.RequirePermission("incident", "write"), withIncidentIDParam(h.UpdateImpactAssessment))
		inc.GET("/:id/impact-assessment", middleware.RequirePermission("incident", "read"), h.GetImpactAssessment)
		inc.PUT("/:id/impact-assessment", middleware.RequirePermission("incident", "write"), h.UpdateImpactAssessment)

		// 事件分类
		inc.POST("/classification", middleware.RequirePermission("incident", "write"), withIncidentIDParam(h.UpdateClassification))
		inc.GET("/:id/classification", middleware.RequirePermission("incident", "read"), h.GetClassification)
		inc.PUT("/:id/classification", middleware.RequirePermission("incident", "write"), h.UpdateClassification)
		inc.PUT("/:id/status", middleware.RequirePermission("incident", "write"), h.Update)

		// 评论
		inc.GET("/:id/comments", middleware.RequirePermission("incident", "read"), h.GetComments)
		inc.POST("/:id/comments", middleware.RequirePermission("incident", "write"), h.CreateComment)
		inc.DELETE("/:id/comments/:commentId", middleware.RequirePermission("incident", "write"), h.DeleteComment)

		// 监控
		inc.POST("/monitoring", middleware.RequirePermission("incident", "read"), h.GetMonitoring)

		// 告警管理
		inc.POST("/alerts", middleware.RequirePermission("incident", "write"), h.CreateAlert)
		inc.GET("/alerts/active", middleware.RequirePermission("incident", "read"), h.GetActiveAlerts)
		inc.GET("/alerts/statistics", middleware.RequirePermission("incident", "read"), h.GetAlertStatistics)
		inc.POST("/alerts/:id/acknowledge", middleware.RequirePermission("incident", "write"), h.AcknowledgeAlert)
		inc.POST("/alerts/:id/resolve", middleware.RequirePermission("incident", "write"), h.ResolveAlert)
	}
}
