package router

import (
	"time"

	"itsm-backend/controller"
	"itsm-backend/ent"
	"itsm-backend/handlers"
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
		TicketController          *controller.TicketController
		TicketDependencyController *controller.TicketDependencyController
		TicketCommentController   *controller.TicketCommentController
		TicketAttachmentController *controller.TicketAttachmentController
		TicketNotificationController *controller.TicketNotificationController
		TicketRatingController    *controller.TicketRatingController
		TicketAssignmentSmartController *controller.TicketAssignmentSmartController
		TicketViewController      *controller.TicketViewController
		IncidentController        *controller.IncidentController
		SLAController          *controller.SLAController
		ApprovalController       *controller.ApprovalController
		AnalyticsController    *controller.AnalyticsController
		PredictionController   *controller.PredictionController
		RootCauseController    *controller.RootCauseController
		AuthController         *controller.AuthController
	UserController         *controller.UserController
	AIController           *controller.AIController
	AuditLogController     *controller.AuditLogController
	BPMNWorkflowController *controller.BPMNWorkflowController
	DashboardHandler       *handlers.DashboardHandler

	// New Controllers (uncomment after ent generate)
	DepartmentController  *controller.DepartmentController
	ProjectController     *controller.ProjectController
	ApplicationController *controller.ApplicationController
	TeamController        *controller.TeamController
	TagController         *controller.TagController
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
			
			// 工单视图路由 - 必须在 /:id 之前，避免路由冲突
			if config.TicketViewController != nil {
				tickets.GET("/views", middleware.RequirePermission("ticket", "read"), config.TicketViewController.ListTicketViews)
				tickets.POST("/views", middleware.RequirePermission("ticket", "read"), config.TicketViewController.CreateTicketView)
				tickets.GET("/views/:id", middleware.RequirePermission("ticket", "read"), config.TicketViewController.GetTicketView)
				tickets.PUT("/views/:id", middleware.RequirePermission("ticket", "read"), config.TicketViewController.UpdateTicketView)
				tickets.DELETE("/views/:id", middleware.RequirePermission("ticket", "read"), config.TicketViewController.DeleteTicketView)
				tickets.POST("/views/:id/share", middleware.RequirePermission("ticket", "read"), config.TicketViewController.ShareTicketView)
			}
			
			// 工单查询和统计 - 必须在 /:id 之前
			tickets.GET("/search", config.TicketController.SearchTickets)
			tickets.GET("/stats", config.TicketController.GetTicketStats)
			tickets.GET("/analytics", config.TicketController.GetTicketAnalytics)
			tickets.GET("/rating-stats", config.TicketRatingController.GetRatingStats)
			tickets.GET("/templates", config.TicketController.GetTicketTemplates)
			tickets.POST("/templates", config.TicketController.CreateTicketTemplate)
			tickets.POST("/export", config.TicketController.ExportTickets)
			tickets.POST("/import", config.TicketController.ImportTickets)
			
			// 工单基础CRUD操作 - 放在最后，避免与其他路由冲突
			tickets.GET("/:id", config.TicketController.GetTicket)
			tickets.PUT("/:id", config.TicketController.UpdateTicket)
			tickets.DELETE("/:id", config.TicketController.DeleteTicket)

			// 工单操作
			tickets.POST("/:id/assign", config.TicketController.AssignTicket)
			tickets.POST("/:id/escalate", config.TicketController.EscalateTicket)
			tickets.POST("/:id/resolve", config.TicketController.ResolveTicket)
			tickets.POST("/:id/close", config.TicketController.CloseTicket)

			// 工单评论
			if config.TicketCommentController != nil {
				tickets.GET("/:id/comments", config.TicketCommentController.ListTicketComments)
				tickets.POST("/:id/comments", config.TicketCommentController.CreateTicketComment)
				tickets.PUT("/:id/comments/:comment_id", config.TicketCommentController.UpdateTicketComment)
				tickets.DELETE("/:id/comments/:comment_id", config.TicketCommentController.DeleteTicketComment)
			}

			// 工单附件
			if config.TicketAttachmentController != nil {
				tickets.GET("/:id/attachments", config.TicketAttachmentController.ListTicketAttachments)
				tickets.POST("/:id/attachments", config.TicketAttachmentController.UploadAttachment)
				tickets.GET("/:id/attachments/:attachment_id", config.TicketAttachmentController.DownloadAttachment)
				tickets.GET("/:id/attachments/:attachment_id/preview", config.TicketAttachmentController.PreviewAttachment)
				tickets.DELETE("/:id/attachments/:attachment_id", config.TicketAttachmentController.DeleteAttachment)
			}

			// 工单通知
			if config.TicketNotificationController != nil {
				tickets.GET("/:id/notifications", config.TicketNotificationController.ListTicketNotifications)
				tickets.POST("/:id/notifications", config.TicketNotificationController.SendTicketNotification)
			}

			// 工单评分
			if config.TicketRatingController != nil {
				tickets.POST("/:id/rating", config.TicketRatingController.SubmitRating)
				tickets.GET("/:id/rating", config.TicketRatingController.GetRating)
				// rating-stats 已在上面定义
			}

			// 智能分配
			if config.TicketAssignmentSmartController != nil {
				tickets.POST("/:id/auto-assign", middleware.RequirePermission("ticket", "assign"), config.TicketAssignmentSmartController.AutoAssign)
				tickets.GET("/assign-recommendations/:id", middleware.RequirePermission("ticket", "read"), config.TicketAssignmentSmartController.GetAssignRecommendations)
				tickets.GET("/assignment-rules", middleware.RequirePermission("ticket", "read"), config.TicketAssignmentSmartController.ListAssignmentRules)
				tickets.POST("/assignment-rules", middleware.RequirePermission("ticket", "admin"), config.TicketAssignmentSmartController.CreateAssignmentRule)
				tickets.GET("/assignment-rules/:id", middleware.RequirePermission("ticket", "read"), config.TicketAssignmentSmartController.GetAssignmentRule)
				tickets.PUT("/assignment-rules/:id", middleware.RequirePermission("ticket", "admin"), config.TicketAssignmentSmartController.UpdateAssignmentRule)
				tickets.DELETE("/assignment-rules/:id", middleware.RequirePermission("ticket", "admin"), config.TicketAssignmentSmartController.DeleteAssignmentRule)
				tickets.POST("/assignment-rules/test", middleware.RequirePermission("ticket", "admin"), config.TicketAssignmentSmartController.TestAssignmentRule)
			}

			// 批量操作（当前未实现批量更新，避免注册导致编译错误）
			// 注意：export 和 import 已在上面定义

			// 子任务管理
			tickets.GET("/:id/subtasks", config.TicketController.GetSubtasks)
			tickets.POST("/:id/subtasks", config.TicketController.CreateSubtask)
			tickets.PATCH("/:id/subtasks/:subtask_id", config.TicketController.UpdateSubtask)
			tickets.DELETE("/:id/subtasks/:subtask_id", config.TicketController.DeleteSubtask)

			// 审批流程管理
			if config.ApprovalController != nil {
				tickets.POST("/approval/workflows", config.ApprovalController.CreateWorkflow)
				tickets.GET("/approval/workflows", config.ApprovalController.ListWorkflows)
				tickets.GET("/approval/workflows/:id", config.ApprovalController.GetWorkflow)
				tickets.PUT("/approval/workflows/:id", config.ApprovalController.UpdateWorkflow)
				tickets.DELETE("/approval/workflows/:id", config.ApprovalController.DeleteWorkflow)
				tickets.GET("/approval/records", config.ApprovalController.GetApprovalRecords)
				tickets.POST("/approval/submit", config.ApprovalController.SubmitApproval)
			}

			// 深度数据分析
			if config.AnalyticsController != nil {
				tickets.POST("/analytics/deep", config.AnalyticsController.GetDeepAnalytics)
				tickets.POST("/analytics/export", config.AnalyticsController.ExportAnalytics)
			}

			// 趋势预测
			if config.PredictionController != nil {
				tickets.POST("/prediction/trend", config.PredictionController.GetTrendPrediction)
				tickets.POST("/prediction/export", config.PredictionController.ExportPredictionReport)
			}

			// 根因分析
			if config.RootCauseController != nil {
				tickets.POST("/:id/root-cause/analyze", config.RootCauseController.AnalyzeTicket)
				tickets.GET("/:id/root-cause/report", config.RootCauseController.GetAnalysisReport)
				tickets.POST("/:id/root-cause/:rootCauseId/confirm", config.RootCauseController.ConfirmRootCause)
				tickets.POST("/:id/root-cause/:rootCauseId/resolve", config.RootCauseController.ResolveRootCause)
			}

			// 依赖关系影响分析
			if config.TicketDependencyController != nil {
				tickets.POST("/:id/dependency-impact", config.TicketDependencyController.AnalyzeDependencyImpact)
			}
		}

		// 事件管理
		incidents := tenant.(*gin.RouterGroup).Group("/incidents")
		{
			// 控制器方法与现有实现对齐
			incidents.GET("", config.IncidentController.ListIncidents)
			incidents.POST("", config.IncidentController.CreateIncident)
			incidents.GET("/stats", config.IncidentController.GetIncidentStats)
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
			sla.POST("/monitoring", config.SLAController.GetSLAMonitoring)

			// SLA预警规则
			sla.POST("/alert-rules", config.SLAController.CreateAlertRule)
			sla.GET("/alert-rules", config.SLAController.ListAlertRules)
			sla.GET("/alert-rules/:id", config.SLAController.GetAlertRule)
			sla.PUT("/alert-rules/:id", config.SLAController.UpdateAlertRule)
			sla.DELETE("/alert-rules/:id", config.SLAController.DeleteAlertRule)
			sla.POST("/alert-history", config.SLAController.GetAlertHistory)
		}

		// 用户管理
		users := tenant.(*gin.RouterGroup).Group("/users")
		{
			users.GET("", config.UserController.ListUsers)
			users.GET("/:id", config.UserController.GetUser)
			users.PUT("/:id", config.UserController.UpdateUser)
			users.DELETE("/:id", config.UserController.DeleteUser)
			// 用户通知偏好
			if config.TicketNotificationController != nil {
				users.GET("/:id/notification-preferences", config.TicketNotificationController.GetNotificationPreferences)
				users.PUT("/:id/notification-preferences", config.TicketNotificationController.UpdateNotificationPreferences)
			}
		}

		// 通知管理
		if config.TicketNotificationController != nil {
			notifications := tenant.(*gin.RouterGroup).Group("/notifications")
			{
				notifications.GET("", config.TicketNotificationController.ListUserNotifications)
				notifications.PUT("/:id/read", config.TicketNotificationController.MarkNotificationRead)
				notifications.PUT("/read-all", config.TicketNotificationController.MarkAllNotificationsRead)
			}
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

		// Dashboard仪表盘
		dashboard := tenant.(*gin.RouterGroup).Group("/dashboard")
		{
			// 概览数据
			dashboard.GET("/overview", config.DashboardHandler.GetOverview)

			// 分项数据
			dashboard.GET("/kpi-metrics", config.DashboardHandler.GetKPIMetrics)
			dashboard.GET("/ticket-trend", config.DashboardHandler.GetTicketTrend)
			dashboard.GET("/incident-distribution", config.DashboardHandler.GetIncidentDistribution)
			dashboard.GET("/sla-data", config.DashboardHandler.GetSLAData)
			dashboard.GET("/satisfaction-data", config.DashboardHandler.GetSatisfactionData)
			dashboard.GET("/quick-actions", config.DashboardHandler.GetQuickActions)
			dashboard.GET("/recent-activities", config.DashboardHandler.GetRecentActivities)
		}

		// 部门管理
		departments := tenant.(*gin.RouterGroup).Group("/departments")
		{
			departments.POST("", config.DepartmentController.CreateDepartment)
			departments.GET("/tree", config.DepartmentController.GetDepartmentTree)
			departments.PUT("/:id", config.DepartmentController.UpdateDepartment)
			departments.DELETE("/:id", config.DepartmentController.DeleteDepartment)
		}

		// 团队管理
		teams := tenant.(*gin.RouterGroup).Group("/teams")
		{
			teams.GET("", config.TeamController.ListTeams)
			teams.POST("", config.TeamController.CreateTeam)
			teams.PUT("/:id", config.TeamController.UpdateTeam)
			teams.DELETE("/:id", config.TeamController.DeleteTeam)
			teams.POST("/members", config.TeamController.AddMember)
		}

		// 项目管理
		projects := tenant.(*gin.RouterGroup).Group("/projects")
		{
			projects.GET("", config.ProjectController.ListProjects)
			projects.POST("", config.ProjectController.CreateProject)
			projects.PUT("/:id", config.ProjectController.UpdateProject)
			projects.DELETE("/:id", config.ProjectController.DeleteProject)
		}

		// 应用管理
		apps := tenant.(*gin.RouterGroup).Group("/applications")
		{
			apps.GET("", config.ApplicationController.ListApplications)
			apps.POST("", config.ApplicationController.CreateApplication)
			apps.PUT("/:id", config.ApplicationController.UpdateApplication)
			apps.DELETE("/:id", config.ApplicationController.DeleteApplication)
			apps.GET("/microservices", config.ApplicationController.ListMicroservices)
			apps.POST("/microservices", config.ApplicationController.CreateMicroservice)
			apps.PUT("/microservices/:id", config.ApplicationController.UpdateMicroservice)
			apps.DELETE("/microservices/:id", config.ApplicationController.DeleteMicroservice)
		}

		// 通用标签管理
		tags := tenant.(*gin.RouterGroup).Group("/tags")
		{
			tags.GET("", config.TagController.ListTags)
			tags.POST("", config.TagController.CreateTag)
			tags.PUT("/:id", config.TagController.UpdateTag)
			tags.DELETE("/:id", config.TagController.DeleteTag)
			tags.POST("/bind", config.TagController.BindTag)
		}

		// 审计日志（需要具备 audit_logs:read 权限）
		audit := tenant.(*gin.RouterGroup).Group("/audit-logs")
		{
			audit.GET("", middleware.RequirePermission("audit_logs", "read"), config.AuditLogController.ListAuditLogs)
		}

		// BPMN 工作流
		if config.BPMNWorkflowController != nil {
			config.BPMNWorkflowController.RegisterRoutes(tenant.(*gin.RouterGroup))
		}
	}
}
