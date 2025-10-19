package router

import (
	"time"

	"itsm-backend/controller"
	"itsm-backend/ent"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RouterConfig 路由配置
type RouterConfig struct {
	JWTSecret string
	Logger    *zap.SugaredLogger
	Client    *ent.Client

	// Controllers
	TicketController   *controller.TicketController
	IncidentController *controller.IncidentController
	SLAController      *controller.SLAController
	AuthController     *controller.AuthController
	UserController     *controller.UserController
	AIController       *controller.AIController
	AuditLogController *controller.AuditLogController
}

// SetupRoutes 设置路由
func SetupRoutes(r *gin.Engine, config *RouterConfig) {
	// 全局中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())

	// 安全中间件
	rateLimiter := middleware.NewRateLimiter(100, time.Minute) // 每分钟100次请求
	r.Use(middleware.RateLimitMiddleware(rateLimiter))
	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.SQLInjectionProtectionMiddleware())
	r.Use(middleware.XSSProtectionMiddleware())
	r.Use(middleware.RequestSizeMiddleware(10 * 1024 * 1024)) // 10MB限制

	// 审计中间件
	r.Use(middleware.AuditMiddleware(config.Client))

	// 公共路由（无需认证）
	public := r.Group("/api/v1")
	{
		public.POST("/login", config.AuthController.Login)
		public.POST("/refresh-token", config.AuthController.RefreshToken)

		// 系统状态
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "timestamp": time.Now()})
		})
		public.GET("/version", func(c *gin.Context) {
			c.JSON(200, gin.H{"version": "1.0.0", "build": "dev"})
		})
	}

	// 认证路由（需要JWT）
	auth := r.Group("/api/v1").Use(middleware.AuthMiddleware(config.JWTSecret))
	// RBAC 权限控制中间件：保护所有已认证路由
	auth.Use(middleware.RBACMiddleware(config.Client))
	{
		// 租户中间件
		tenant := auth.Use(middleware.TenantMiddleware(config.Client))

		// 工单管理
		tickets := tenant.(*gin.RouterGroup).Group("/tickets")
		{
			tickets.GET("", config.TicketController.ListTickets)
			tickets.POST("", config.TicketController.CreateTicket)
			tickets.GET("/:id", config.TicketController.GetTicket)
			tickets.PUT("/:id", config.TicketController.UpdateTicket)
			tickets.DELETE("/:id", config.TicketController.DeleteTicket)

			// 工单操作
			tickets.POST("/:id/assign", config.TicketController.AssignTicket)
			tickets.POST("/:id/escalate", config.TicketController.EscalateTicket)
			tickets.POST("/:id/resolve", config.TicketController.ResolveTicket)
			tickets.POST("/:id/close", config.TicketController.CloseTicket)

			// 工单查询和统计
			tickets.GET("/search", config.TicketController.SearchTickets)
			tickets.GET("/stats", config.TicketController.GetTicketStats)
			tickets.GET("/analytics", config.TicketController.GetTicketAnalytics)

			// 批量操作（当前未实现批量更新，避免注册导致编译错误）
			tickets.POST("/export", config.TicketController.ExportTickets)
			tickets.POST("/import", config.TicketController.ImportTickets)

			// 模板管理
			tickets.GET("/templates", config.TicketController.GetTicketTemplates)
			tickets.POST("/templates", config.TicketController.CreateTicketTemplate)
		}

		// 事件管理
		incidents := tenant.(*gin.RouterGroup).Group("/incidents")
		{
			// 控制器方法与现有实现对齐
			incidents.GET("", config.IncidentController.ListIncidents)
			incidents.POST("", config.IncidentController.CreateIncident)
			incidents.GET("/:id", config.IncidentController.GetIncident)
			incidents.PUT("/:id", config.IncidentController.UpdateIncident)
			// incidents.POST("/:id/close", config.IncidentController.CloseIncident) // TODO: 实现CloseIncident方法
		}

		// SLA管理
		sla := tenant.(*gin.RouterGroup).Group("/sla")
		{
			sla.POST("/definitions", config.SLAController.CreateSLADefinition)
			sla.GET("/definitions/:id", config.SLAController.GetSLADefinition)
			sla.PUT("/definitions/:id", config.SLAController.UpdateSLADefinition)
			sla.DELETE("/definitions/:id", config.SLAController.DeleteSLADefinition)
			sla.GET("/definitions", config.SLAController.ListSLADefinitions)
			sla.POST("/compliance/:ticketId", config.SLAController.CheckSLACompliance)
			sla.GET("/metrics", config.SLAController.GetSLAMetrics)
			sla.GET("/violations", config.SLAController.GetSLAViolations)
			sla.PUT("/violations/:id", config.SLAController.UpdateViolationStatus)
		}

		// 用户管理
		users := tenant.(*gin.RouterGroup).Group("/users")
		{
			users.GET("", config.UserController.ListUsers)
			users.GET("/:id", config.UserController.GetUser)
			users.PUT("/:id", config.UserController.UpdateUser)
			users.DELETE("/:id", config.UserController.DeleteUser)
		}

		// AI功能
		ai := tenant.(*gin.RouterGroup).Group("/ai")
		{
			ai.POST("/search", config.AIController.Search)
			ai.POST("/triage", config.AIController.Triage)
			ai.POST("/summarize", config.AIController.Summarize)
			ai.POST("/similar-incidents", config.AIController.SimilarIncidents)
			ai.POST("/chat", config.AIController.Chat)
			ai.GET("/tools", config.AIController.Tools)
			ai.POST("/tools/:tool_id/execute", config.AIController.ExecuteTool)
			ai.POST("/tools/:tool_id/approve", config.AIController.ApproveTool)
			ai.GET("/tools/invocations/:id", config.AIController.GetToolInvocation)
			ai.POST("/embed", config.AIController.RunEmbed)
			ai.POST("/feedback", config.AIController.SaveFeedback)
			ai.GET("/metrics", config.AIController.GetMetrics)
		}

		// 审计日志（需要具备 audit_logs:read 权限）
		audit := tenant.(*gin.RouterGroup).Group("/audit-logs")
		{
			audit.GET("", middleware.RequirePermission("audit_logs", "read"), config.AuditLogController.ListAuditLogs)
		}
	}
}
