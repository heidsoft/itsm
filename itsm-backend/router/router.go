package router

import (
	"itsm-backend/controller"
	"itsm-backend/ent"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(
	authController *controller.AuthController,
	ticketController *controller.TicketController,
	serviceController *controller.ServiceController,
	dashboardController *controller.DashboardController,
	cmdbController *controller.CMDBController,
	incidentController *controller.IncidentController,
	notificationController *controller.NotificationController,
	userController *controller.UserController,
	knowledgeController *controller.KnowledgeController,
	aiController *controller.AIController,
	workflowController *controller.WorkflowController,
	ticketTagController *controller.TicketTagController,
	ticketAssignmentController *controller.TicketAssignmentController,
	ticketCategoryController *controller.TicketCategoryController,
	problemController *controller.ProblemController,
	changeController *controller.ChangeController,
	client *ent.Client,
	jwtSecret string,
) *gin.Engine {
	r := gin.Default()

	// 中间件
	r.Use(middleware.CORSMiddleware())
	// 请求ID中间件
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.RecoveryMiddleware())

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 公共路由（无需认证）
		public := v1.Group("")
		{
			// 真实登录与刷新Token
			public.POST("/login", authController.Login)
			public.POST("/refresh-token", authController.RefreshToken)
			// 公共健康检查与版本信息
			public.GET("/healthz", func(c *gin.Context) {
				c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{"status": "healthy"}})
			})
			public.GET("/version", func(c *gin.Context) {
				c.JSON(200, gin.H{"code": 0, "message": "ok", "data": gin.H{"version": "1.0.0"}})
			})
		}

		// 认证路由（需要 JWT）
		auth := v1.Group("")
		auth.Use(middleware.AuthMiddleware(jwtSecret))
		// 启用多租户中间件（通过 X-Tenant-Code 或子域解析），与 JWT tenant_id 互补
		auth.Use(middleware.TenantMiddleware(client))
		// 关键写操作审计
		auth.Use(middleware.AuditMiddleware(client))
		{
			// 工单管理路由
			tickets := auth.Group("/tickets")
			{
				tickets.GET("", ticketController.ListTickets)
				tickets.POST("", ticketController.CreateTicket)
				tickets.GET("/stats", ticketController.GetTicketStats)
				tickets.GET("/analytics", ticketController.GetTicketAnalytics)
				tickets.POST("/export", ticketController.ExportTickets)
				tickets.POST("/import", ticketController.ImportTickets)
				tickets.POST("/batch-delete", ticketController.BatchDeleteTickets)
				// 单工单操作
				tickets.POST("/:id/assign", ticketController.AssignTicket)
				tickets.POST("/:id/escalate", ticketController.EscalateTicket)
				tickets.POST("/:id/resolve", ticketController.ResolveTicket)
				tickets.POST("/:id/close", ticketController.CloseTicket)
				tickets.GET("/:id/activity", ticketController.GetTicketActivity)
				// 查询辅助
				tickets.GET("/search", ticketController.SearchTickets)
				tickets.GET("/overdue", ticketController.GetOverdueTickets)
				tickets.GET("/assignee/:assignee_id", ticketController.GetTicketsByAssignee)

				tickets.GET("/:id", ticketController.GetTicket)
				tickets.PUT("/:id", ticketController.UpdateTicket)
				tickets.DELETE("/:id", ticketController.DeleteTicket)
			}

			// 工单标签管理路由
			if ticketTagController != nil {
				ticketTags := auth.Group("/ticket-tags")
				{
					ticketTags.GET("", ticketTagController.ListTags)
					ticketTags.POST("", ticketTagController.CreateTag)
					ticketTags.GET("/:id", ticketTagController.GetTag)
					ticketTags.PUT("/:id", ticketTagController.UpdateTag)
					ticketTags.DELETE("/:id", ticketTagController.DeleteTag)
					// 工单标签分配
					ticketTags.POST("/tickets/:ticketId/assign", ticketTagController.AssignTagsToTicket)
					ticketTags.DELETE("/tickets/:ticketId/remove", ticketTagController.RemoveTagsFromTicket)
				}
			}

			// 工单分类管理路由
			if ticketCategoryController != nil {
				ticketCategories := auth.Group("/ticket-categories")
				{
					ticketCategories.GET("", ticketCategoryController.ListCategories)
					ticketCategories.POST("", ticketCategoryController.CreateCategory)
					ticketCategories.GET("/:id", ticketCategoryController.GetCategory)
					ticketCategories.PUT("/:id", ticketCategoryController.UpdateCategory)
					ticketCategories.DELETE("/:id", ticketCategoryController.DeleteCategory)
				}
			}

			// 工单智能分配路由
			if ticketAssignmentController != nil {
				ticketAssignment := auth.Group("/ticket-assignment")
				{
					ticketAssignment.POST("/assign", ticketAssignmentController.AssignTicket)
					ticketAssignment.GET("/workload/user", ticketAssignmentController.GetUserWorkload)
					ticketAssignment.GET("/workload/team", ticketAssignmentController.GetTeamWorkload)
					ticketAssignment.POST("/reassign", ticketAssignmentController.ReassignTicket)
					ticketAssignment.POST("/load-balance", ticketAssignmentController.LoadBalance)
				}
			}

			// 工单模板路由
			templates := auth.Group("/ticket-templates")
			{
				templates.GET("", ticketController.GetTicketTemplates)
				templates.POST("", ticketController.CreateTicketTemplate)
				templates.PUT("/:id", ticketController.UpdateTicketTemplate)
				templates.DELETE("/:id", ticketController.DeleteTicketTemplate)
			}

			// 工作流管理路由
			if workflowController != nil {
				workflows := auth.Group("/workflows")
				{
					workflows.GET("", workflowController.ListWorkflows)
					workflows.POST("", workflowController.CreateWorkflow)
					workflows.GET("/:id", workflowController.GetWorkflow)
					workflows.PUT("/:id", workflowController.UpdateWorkflow)
					workflows.DELETE("/:id", workflowController.DeleteWorkflow)
					// 工作流实例操作
					workflows.POST("/:id/start", workflowController.StartWorkflow)
					workflows.POST("/:id/execute-step", workflowController.ExecuteWorkflowStep)
				}
			}

			// 服务目录路由
			services := auth.Group("/services")
			{
				services.GET("", serviceController.GetServiceCatalogs)
				services.POST("", serviceController.CreateServiceCatalog)
				services.GET("/:id", serviceController.GetServiceCatalogByID)
				services.PUT("/:id", serviceController.UpdateServiceCatalog)
				services.DELETE("/:id", serviceController.DeleteServiceCatalog)

				// 服务请求
				services.GET("/requests", serviceController.GetUserServiceRequests)
				services.POST("/requests", serviceController.CreateServiceRequest)
				services.GET("/requests/:id", serviceController.GetServiceRequestByID)
				services.PUT("/requests/:id", serviceController.UpdateServiceRequestStatus)
			}

			// 仪表盘路由
			dashboard := auth.Group("/dashboard")
			{
				dashboard.GET("", dashboardController.GetDashboardData)
				dashboard.GET("/kpis", dashboardController.GetKPIMetrics)
				dashboard.GET("/resources/distribution", dashboardController.GetResourceDistribution)
				dashboard.GET("/resources/health", dashboardController.GetResourceHealth)
			}

			// CMDB路由
			cmdb := auth.Group("/cmdb")
			{
				cmdb.GET("/items", cmdbController.ListConfigurationItems)
				cmdb.POST("/items", cmdbController.CreateConfigurationItem)
				cmdb.GET("/items/:id", cmdbController.GetConfigurationItem)
				cmdb.PUT("/items/:id", cmdbController.UpdateConfigurationItem)
				cmdb.DELETE("/items/:id", cmdbController.DeleteConfigurationItem)

				// CMDB属性定义
				cmdb.POST("/attributes", cmdbController.CreateCIAttributeDefinition)
				cmdb.GET("/types/:id", cmdbController.GetCITypeWithAttributes)
				cmdb.POST("/validate", cmdbController.ValidateCIAttributes)
				cmdb.POST("/search", cmdbController.SearchCIsByAttributes)
			}

			// 事件管理路由
			incidents := auth.Group("/incidents")
			{
				incidents.GET("", incidentController.GetIncidents)
				incidents.POST("", incidentController.CreateIncident)
				incidents.GET("/stats", incidentController.GetIncidentStats)

				incidents.GET("/:id", incidentController.GetIncident)
				incidents.PUT("/:id", incidentController.UpdateIncident)
				incidents.PUT("/:id/close", incidentController.CloseIncident)
			}

			// 问题管理路由
			problems := auth.Group("/problems")
			{
				problems.GET("", problemController.ListProblems)
				problems.POST("", problemController.CreateProblem)
				problems.GET("/stats", problemController.GetProblemStats)
				problems.GET("/:id", problemController.GetProblem)
				problems.PUT("/:id", problemController.UpdateProblem)
				problems.DELETE("/:id", problemController.DeleteProblem)
			}

			// 变更管理路由
			changes := auth.Group("/changes")
			{
				changes.GET("", changeController.ListChanges)
				changes.POST("", changeController.CreateChange)
				changes.GET("/:id", changeController.GetChange)
				changes.PUT("/:id", changeController.UpdateChange)
				changes.DELETE("/:id", changeController.DeleteChange)
			}

			// 知识库路由
			if knowledgeController != nil {
				knowledge := auth.Group("/knowledge-articles")
				{
					knowledge.GET("", knowledgeController.ListArticles)
					knowledge.POST("", knowledgeController.CreateArticle)
					knowledge.GET("/:id", knowledgeController.GetArticle)
					knowledge.PUT("/:id", knowledgeController.UpdateArticle)
					knowledge.DELETE("/:id", knowledgeController.DeleteArticle)
					knowledge.GET("/categories", knowledgeController.GetCategories)
				}
			}

			// AI 路由（最小可用：只读 RAG 问答）
			if aiController != nil {
				ai := auth.Group("/ai")
				{
					ai.POST("/chat", aiController.Chat)
					ai.POST("/search", aiController.Search)
					ai.POST("/triage", aiController.Triage)
					ai.POST("/summarize", aiController.Summarize)
					ai.POST("/similar-incidents", aiController.SimilarIncidents)
					ai.GET("/tools", aiController.Tools)
					ai.POST("/tools/execute", aiController.ExecuteTool)
					ai.POST("/tools/:id/approve", aiController.ApproveTool)
					ai.GET("/tools/:id", aiController.GetToolInvocation)
					ai.POST("/embed/run", aiController.RunEmbed)
					ai.POST("/feedback", aiController.SaveFeedback)
					ai.GET("/metrics", aiController.GetMetrics)
				}
			}

			// SLA管理路由
			sla := auth.Group("/sla")
			{
				sla.GET("", func(c *gin.Context) {
					c.JSON(200, gin.H{"code": 0, "message": "SLA管理功能开发中"})
				})
				sla.POST("", func(c *gin.Context) {
					c.JSON(200, gin.H{"code": 0, "message": "SLA管理功能开发中"})
				})
			}

			// 报表路由
			reports := auth.Group("/reports")
			{
				reports.GET("/tickets", func(c *gin.Context) {
					c.JSON(200, gin.H{"code": 0, "message": "工单报表功能开发中"})
				})
				reports.GET("/incidents", func(c *gin.Context) {
					c.JSON(200, gin.H{"code": 0, "message": "事件报表功能开发中"})
				})
				reports.GET("/sla", func(c *gin.Context) {
					c.JSON(200, gin.H{"code": 0, "message": "SLA报表功能开发中"})
				})
			}

			// 通知路由
			notifications := auth.Group("/notifications")
			{
				notifications.GET("", notificationController.GetNotifications)
				notifications.GET("/unread-count", notificationController.GetUnreadCount)
				notifications.PATCH("/:id/read", notificationController.MarkNotificationRead)
				notifications.PATCH("/read-all", notificationController.MarkAllNotificationsRead)
				notifications.DELETE("/:id", notificationController.DeleteNotification)
				notifications.POST("", notificationController.CreateNotification) // 管理员功能
			}

			// 用户管理路由
			users := auth.Group("/users")
			{
				users.GET("", userController.ListUsers)                        // 获取用户列表
				users.POST("", userController.CreateUser)                      // 创建用户
				users.GET("/search", userController.SearchUsers)               // 搜索用户
				users.GET("/stats", userController.GetUserStats)               // 用户统计
				users.PUT("/batch", userController.BatchUpdateUsers)           // 批量更新
				users.GET("/:id", userController.GetUser)                      // 获取用户详情
				users.PUT("/:id", userController.UpdateUser)                   // 更新用户
				users.DELETE("/:id", userController.DeleteUser)                // 删除用户
				users.PUT("/:id/status", userController.ChangeUserStatus)      // 更改状态
				users.PUT("/:id/reset-password", userController.ResetPassword) // 重置密码
			}

			// 角色管理路由
			roles := auth.Group("/roles")
			{
				roles.GET("", func(c *gin.Context) {
					c.JSON(200, gin.H{"code": 0, "message": "角色管理功能开发中"})
				})
				roles.POST("", func(c *gin.Context) {
					c.JSON(200, gin.H{"code": 0, "message": "角色管理功能开发中"})
				})
			}

			// 权限管理路由
			permissions := auth.Group("/permissions")
			{
				permissions.GET("", func(c *gin.Context) {
					c.JSON(200, gin.H{"code": 0, "message": "权限管理功能开发中"})
				})
				permissions.POST("", func(c *gin.Context) {
					c.JSON(200, gin.H{"code": 0, "message": "权限管理功能开发中"})
				})
			}

			// 租户管理路由
			tenants := auth.Group("/tenants")
			{
				tenants.GET("", func(c *gin.Context) {
					c.JSON(200, gin.H{"code": 0, "message": "租户管理功能开发中"})
				})
				tenants.POST("", func(c *gin.Context) {
					c.JSON(200, gin.H{"code": 0, "message": "租户管理功能开发中"})
				})
			}

			// 系统配置路由
			config := auth.Group("/config")
			{
				config.GET("", func(c *gin.Context) {
					c.JSON(200, gin.H{"code": 0, "message": "系统配置功能开发中"})
				})
				config.POST("", func(c *gin.Context) {
					c.JSON(200, gin.H{"code": 0, "message": "系统配置功能开发中"})
				})
			}

			// 健康检查
			auth.GET("/health", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"code":    0,
					"message": "服务正常",
					"data": gin.H{
						"status":    "healthy",
						"timestamp": "2024-01-15T10:30:00Z",
						"version":   "1.0.0",
					},
				})
			})
		}
	}

	return r
}
