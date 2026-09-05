package router

import (
	slaHandler "itsm-backend/handlers/sla"
	slaTemplateHandler "itsm-backend/handlers/sla_template"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupSLARoutes 注册 SLA 相关路由。
// 从 router.go 的集中注册块抽取而来，路由路径/方法/中间件与抽取前逐行一致。
func SetupSLARoutes(tenant *gin.RouterGroup, h *slaHandler.Handler) {
	slaGrp := tenant.Group("/sla")
	{
		slaGrp.GET("", middleware.RequirePermission("sla", "read"), h.ListSLADefinitions)
		// SLA Definitions
		slaGrp.GET("/stats", middleware.RequirePermission("sla", "read"), h.GetSLAStats)
		slaGrp.GET("/definitions", middleware.RequirePermission("sla", "read"), h.ListSLADefinitions)
		slaGrp.POST("/definitions", middleware.RequirePermission("sla", "write"), h.CreateSLADefinition)
		// 兼容旧路径：/sla/policies → /sla/definitions
		slaGrp.GET("/policies", middleware.RequirePermission("sla", "read"), h.ListSLADefinitions)
		slaGrp.POST("/policies", middleware.RequirePermission("sla", "write"), h.CreateSLADefinition)
		slaGrp.GET("/policies/:id", middleware.RequirePermission("sla", "read"), h.GetSLADefinition)
		slaGrp.PUT("/policies/:id", middleware.RequirePermission("sla", "write"), h.UpdateSLADefinition)
		slaGrp.DELETE("/policies/:id", middleware.RequirePermission("sla", "delete"), h.DeleteSLADefinition)

		// 兼容旧路径：/sla/monitor → /sla/monitoring
		slaGrp.POST("/monitor", middleware.RequirePermission("sla", "read"), h.GetSLAMonitoring)

		slaGrp.GET("/definitions/:id", middleware.RequirePermission("sla", "read"), h.GetSLADefinition)
		slaGrp.PUT("/definitions/:id", middleware.RequirePermission("sla", "write"), h.UpdateSLADefinition)
		slaGrp.DELETE("/definitions/:id", middleware.RequirePermission("sla", "delete"), h.DeleteSLADefinition)

		// SLA Alert Rules
		slaGrp.POST("/alert-rules", middleware.RequirePermission("sla", "write"), h.CreateAlertRule)
		slaGrp.GET("/alert-rules", middleware.RequirePermission("sla", "read"), h.ListAlertRules)
		slaGrp.GET("/alert-rules/:id", middleware.RequirePermission("sla", "read"), h.GetAlertRule)
		slaGrp.PUT("/alert-rules/:id", middleware.RequirePermission("sla", "write"), h.UpdateAlertRule)
		slaGrp.DELETE("/alert-rules/:id", middleware.RequirePermission("sla", "delete"), h.DeleteAlertRule)

		// SLA Metrics
		slaGrp.GET("/metrics", middleware.RequirePermission("sla", "read"), h.GetSLAMetrics)

		// SLA Violations
		slaGrp.GET("/violations", middleware.RequirePermission("sla", "read"), h.GetSLAViolations)
		slaGrp.PUT("/violations/:id", middleware.RequirePermission("sla", "write"), h.UpdateViolationStatus)

		// SLA Monitoring
		slaGrp.POST("/monitoring", middleware.RequirePermission("sla", "read"), h.GetSLAMonitoring)
		// SLA 绩效按维度聚合（serviceType / priority），供监控大屏绩效表格使用
		slaGrp.GET("/performance", middleware.RequirePermission("sla", "read"), h.GetSLAPerformance)

		// SLA Compliance Check
		slaGrp.POST("/check-compliance/:ticketId", middleware.RequirePermission("sla", "read"), h.CheckSLACompliance)

		// SLA Alert History
		slaGrp.GET("/alert-history", middleware.RequirePermission("sla", "read"), h.GetAlertHistory)
		slaGrp.GET("/compliance-report", middleware.RequirePermission("sla", "read"), h.GetSLAComplianceReport)
	}

	// SLA 模板（开箱即用预置模板）- 在 router.go 中单独处理 SLATemplateHandler.RegisterRoutes
}

// SetupSLATemplateRoutes 注册 SLA 模板相关路由。
func SetupSLATemplateRoutes(tenant *gin.RouterGroup, h *slaTemplateHandler.Handler) {
	h.RegisterRoutes(tenant)
}
