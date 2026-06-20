package router

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/controller"
	marketplaceController "itsm-backend/controller/marketplace"
	"itsm-backend/ent"
	"itsm-backend/handlers"
	"itsm-backend/handlers/ai"
	"itsm-backend/handlers/change"
	domainCommon "itsm-backend/handlers/common"
	"itsm-backend/handlers/knowledge"
	"itsm-backend/handlers/known_error"
	"itsm-backend/handlers/problem"
	"itsm-backend/handlers/service_catalog"
	"itsm-backend/handlers/service_request"
	"itsm-backend/handlers/sla"
	"itsm-backend/handlers/standard_change"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// RateLimiterInterface 限流器接口（支持 Redis 和内存实现）
type RateLimiterInterface interface {
	Allow(ctx context.Context, clientIP string) (bool, error)
}

// RouterConfig 路由配置
type RouterConfig struct {
	JWTSecret string
	Logger    *zap.SugaredLogger
	Client    *ent.Client

	// CSRF configuration
	CSRFEnabled bool

	// Redis rate limiter (optional - uses memory fallback if nil)
	RedisRateLimiter RateLimiterInterface

	// Controllers
	ProblemInvestigationController  *controller.ProblemInvestigationController
	TicketController                *controller.TicketController
	TicketDependencyController      *controller.TicketDependencyController
	TicketCommentController         *controller.TicketCommentController
	TicketAttachmentController      *controller.TicketAttachmentController
	TicketNotificationController    *controller.TicketNotificationController
	TicketRatingController          *controller.TicketRatingController
	TicketAssignmentSmartController *controller.TicketAssignmentSmartController
	TicketViewController            *controller.TicketViewController
	TicketWorkflowController        *controller.TicketWorkflowController
	TicketAutomationRuleController  *controller.TicketAutomationRuleController
	IncidentController              *controller.IncidentController
	ApprovalController              *controller.ApprovalController
	BPMNWorkflowController          *controller.BPMNWorkflowController
	BPMNProcessTriggerController    *controller.BPMNProcessTriggerController
	BPMNDashboardController         *controller.BPMNDashboardController
	A2UITicketController            *controller.A2UITicketController
	DashboardHandler                *handlers.DashboardHandler

	// Organization & Project
	ProjectController     *controller.ProjectController
	ApplicationController *controller.ApplicationController

	// Ticket related controllers
	TicketCategoryController *controller.TicketCategoryController

	// CMDB Controllers
	CMDBController      *controller.CMDBController
	TicketTagController *controller.TicketTagController

	// User Controller
	UserController *controller.UserController

	// Group Controller
	GroupController *controller.GroupController

	// Role & Permission Controllers (new database-backed implementation)
	RoleController                   *controller.RoleController
	PermissionController             *controller.PermissionController
	MenuController                   *controller.MenuController
	TenantController                 *controller.TenantController
	MSPController                    *controller.MSPController
	SystemConfigController           *controller.SystemConfigController
	ApprovalChainController          *controller.ApprovalChainController
	NotificationPreferenceController *controller.NotificationPreferenceController
	NotificationController           *controller.NotificationController

	// Additional domain controllers
	ServiceController      *controller.ServiceController
	ProvisioningController *controller.ProvisioningController
	AnalyticsController    *controller.AnalyticsController
	PredictionController   *controller.PredictionController
	ReleaseController      *controller.ReleaseController
	AssetController        *controller.AssetController
	VendorController       *controller.VendorController
	AssetLicenseController *controller.AssetLicenseController
	SurveyController       *controller.SurveyController

	// Domain Handlers
	ServiceCatalogHandler *service_catalog.Handler
	ServiceRequestHandler *service_request.Handler

	ProblemHandler   *problem.Handler
	ChangeHandler    *change.Handler
	KnowledgeHandler *knowledge.Handler
	SLAHandler       *sla.Handler
	AIHandler        *ai.Handler
	CommonHandler    *domainCommon.Handler
	RoleHandler      *common.RoleHandler

	// WebSocket Service
	WebSocketService *service.WebSocketService

	// Global Search
	GlobalSearchController *controller.GlobalSearchController

	// Standard Change Handler (标准变更模板库)
	StandardChangeHandler *standard_change.Handler

	// Known Error Handler (KEDB)
	KnownErrorHandler *known_error.Handler

	// Connector Controller (连接器/插件/技能市场)
	ConnectorController   *controller.ConnectorController
	MarketplaceController *marketplaceController.Controller
}

// SetupRoutes 设置路由
func SetupRoutes(r *gin.Engine, config *RouterConfig) {
	// 全局中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.PrometheusMetricsMiddleware())

	// 安全中间件
	// 速率限制：优先使用 Redis 限流器（分布式环境），否则使用内存限流器
	if config.RedisRateLimiter != nil {
		// 使用 Redis 限流器（分布式环境）
		config.Logger.Info("Using Redis-based distributed rate limiter")
		r.Use(func(limiter RateLimiterInterface) gin.HandlerFunc {
			return func(c *gin.Context) {
				clientIP := c.ClientIP()
				allowed, err := limiter.Allow(c.Request.Context(), clientIP)
				if err != nil {
					config.Logger.Warnw("Rate limiter error, allowing request", "error", err)
					c.Next()
					return
				}
				if !allowed {
					common.Fail(c, 429, "请求过于频繁，请稍后再试")
					c.Abort()
					return
				}
				c.Next()
			}
		}(config.RedisRateLimiter))
	} else {
		// 使用内存限流器（单机环境）
		config.Logger.Warn("Redis rate limiter not configured, using in-memory rate limiter (not suitable for distributed deployment)")
		rateLimit := 500
		if envLimit := os.Getenv("RATE_LIMIT"); envLimit != "" {
			if parsed, err := strconv.Atoi(envLimit); err == nil && parsed > 0 {
				rateLimit = parsed
			}
		}
		rateLimiter := middleware.NewRateLimiter(rateLimit, time.Minute)
		r.Use(middleware.RateLimitMiddleware(rateLimiter))
	}

	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.SQLInjectionProtectionMiddleware())
	r.Use(middleware.XSSProtectionMiddleware())
	r.Use(middleware.RequestSizeMiddleware(10 * 1024 * 1024)) // 10MB限制

	// 公共路由（无需认证）
	public := r.Group("/api/v1")
	{
		if config.CommonHandler != nil {
			public.POST("/auth/login", config.CommonHandler.Login)
			public.POST("/refresh-token", config.CommonHandler.RefreshToken)
		}

		// CSRF token 获取端点（无需认证）
		if config.CSRFEnabled {
			public.GET("/csrf-token", middleware.CSRFTokenEndpoint(middleware.DefaultCSRFConfig()))
		}

		// 系统状态
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "timestamp": time.Now()})
		})
		public.GET("/version", func(c *gin.Context) {
			c.JSON(200, gin.H{"version": "1.0.0", "build": "dev"})
		})
		public.GET("/readiness/ga", func(c *gin.Context) {
			common.Success(c, buildGAReadiness(c.Request.Context(), config.Client))
		})

		// Prometheus metrics 端点（需要认证，防止信息泄露）
		metricsAuth := r.Group("")
		metricsAuth.Use(middleware.AuthMiddleware(config.JWTSecret))
		metricsAuth.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// 认证路由（需要JWT）
	auth := r.Group("/api/v1").Use(middleware.AuthMiddleware(config.JWTSecret))
	// RBAC 权限控制中间件：保护所有已认证路由
	auth.Use(middleware.RBACMiddleware(config.Client))
	// CSRF 保护中间件（仅对状态变更请求生效）
	if config.CSRFEnabled {
		csrfConfig := middleware.DefaultCSRFConfig()
		// CSRF 不验证登录相关的路径
		csrfConfig.SkipPaths = append(csrfConfig.SkipPaths, "/api/v1/auth/login", "/api/v1/refresh-token")
		auth.Use(middleware.CSRFProtectionMiddleware(csrfConfig))
	}

	// WebSocket 路由（使用短期票据替代JWT query参数，避免token泄露）
	// 票据流程:
	//   1. 客户端 POST /api/v1/ws/ticket (携带 Authorization header) 获取短期票据
	//   2. 客户端使用 ?ticket=<ticket> 建立 WebSocket 连接
	//   3. 票据验证后立即销毁（一次性使用）
	var wsTicketStore *WSTicketStore
	if config.WebSocketService != nil {
		wsTicketStore = NewWSTicketStore(DefaultWSTicketTTL)

		// 票据颁发端点（需要JWT认证）
		auth.POST("/ws/ticket", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			tenantID, _ := c.Get("tenant_id")

			ticketStr, err := wsTicketStore.Generate(userID.(int), tenantID.(int))
			if err != nil {
				common.Fail(c, common.InternalErrorCode, "生成票据失败")
				return
			}

			common.Success(c, gin.H{"ticket": ticketStr})
		})

		// WebSocket 连接端点（使用短期票据认证）
		wsGroup := r.Group("/api/v1")
		wsGroup.GET("/ws/notifications", func(c *gin.Context) {
			ticketStr := c.Query("ticket")
			if ticketStr == "" {
				common.Fail(c, common.AuthFailedCode, "缺少认证票据")
				c.Abort()
				return
			}

			userID, tenantID, ok := wsTicketStore.Redeem(ticketStr)
			if !ok {
				common.Fail(c, common.AuthFailedCode, "票据无效或已过期")
				c.Abort()
				return
			}

			config.WebSocketService.HandleWebSocket(c.Writer, c.Request, userID, tenantID)
		})
	}

	// MSP Routes (跨租户，不需要租户中间件)
	if config.MSPController != nil {
		msp := r.Group("/api/v1/msp")
		msp.Use(middleware.AuthMiddleware(config.JWTSecret))
		msp.Use(middleware.RBACMiddleware(config.Client)) // 设置 client 到 context
		msp.Use(middleware.MSPMiddleware(config.Client))
		{
			// MSP 基础信息 - 允许 MSP 员工和管理员访问
			msp.GET("/status", config.MSPController.GetMSPStatus)
			msp.GET("/context", middleware.RequireMSPPermission("msp", "read"), config.MSPController.GetMSPContext)

			// 分配管理 - 需要 msp_allocation 权限
			msp.GET("/allocations", middleware.RequireMSPPermission("msp_allocation", "read"), config.MSPController.GetAllocations)
			msp.POST("/allocations", middleware.RequireMSPPermission("msp_allocation", "write"), config.MSPController.CreateAllocation)
			msp.POST("/allocations/deallocate", middleware.RequireMSPPermission("msp_allocation", "write"), config.MSPController.Deallocate)

			// 客户管理 - 需要 msp_customer 权限
			msp.GET("/customers", middleware.RequireMSPPermission("msp_customer", "read"), config.MSPController.GetAllCustomers)
			msp.GET("/customers/:customer_tenant_id/tickets", middleware.RequireMSPPermission("msp_ticket", "read"), config.MSPController.GetCustomerTickets)

			// 工单分配 - 需要 msp_ticket:write 权限
			msp.POST("/tickets/:id/assign", middleware.RequireMSPPermission("msp_ticket", "write"), config.MSPController.AssignMSPTechnician)

			// 报表 - 需要 msp_report 权限
			msp.GET("/reports/customers", middleware.RequireMSPPermission("msp_report", "read"), config.MSPController.GetCustomerReports)
			msp.GET("/reports/performance", middleware.RequireMSPPermission("msp_report", "read"), config.MSPController.GetPerformanceReports)
		}
	}

	{
		// 租户中间件
		tenant := auth.Use(middleware.TenantMiddleware(config.Client))

		// ==================== Ticket Categories & Tags ====================
		if config.TicketCategoryController != nil {
			categories := tenant.(*gin.RouterGroup).Group("/ticket-categories")
			{
				categories.GET("", config.TicketCategoryController.ListCategories)
				categories.POST("", middleware.RequirePermission("ticket_category", "create"), config.TicketCategoryController.CreateCategory)
				categories.GET("/:id", config.TicketCategoryController.GetCategory)
				categories.PUT("/:id", middleware.RequirePermission("ticket_category", "update"), config.TicketCategoryController.UpdateCategory)
				categories.PUT("/:id/move", middleware.RequirePermission("ticket_category", "update"), config.TicketCategoryController.MoveCategory)
				categories.DELETE("/:id", middleware.RequirePermission("ticket_category", "delete"), config.TicketCategoryController.DeleteCategory)
				categories.GET("/tree", config.TicketCategoryController.GetCategoryTree)
			}
		}

		if config.TicketTagController != nil {
			tags := tenant.(*gin.RouterGroup).Group("/ticket-tags")
			{
				tags.GET("", config.TicketTagController.ListTags)
				tags.POST("", middleware.RequirePermission("ticket_tag", "create"), config.TicketTagController.CreateTag)
				tags.GET("/:id", config.TicketTagController.GetTag)
				tags.PUT("/:id", middleware.RequirePermission("ticket_tag", "update"), config.TicketTagController.UpdateTag)
				tags.DELETE("/:id", middleware.RequirePermission("ticket_tag", "delete"), config.TicketTagController.DeleteTag)
			}
		}

		// ==================== Tickets ====================
		tickets := tenant.(*gin.RouterGroup).Group("/tickets")
		{
			tickets.GET("", config.TicketController.ListTickets)
			tickets.POST("", config.TicketController.CreateTicket)

			if config.TicketViewController != nil {
				tickets.GET("/views", config.TicketViewController.ListTicketViews)
				tickets.POST("/views", config.TicketViewController.CreateTicketView)
				tickets.GET("/views/:id", config.TicketViewController.GetTicketView)
				tickets.PUT("/views/:id", config.TicketViewController.UpdateTicketView)
				tickets.DELETE("/views/:id", config.TicketViewController.DeleteTicketView)
			}

			tickets.GET("/search", config.TicketController.SearchTickets)
			tickets.GET("/stats", config.TicketController.GetTicketStats)
			tickets.GET("/:id", config.TicketController.GetTicket)
			tickets.PUT("/:id", config.TicketController.UpdateTicket)
			tickets.PUT("/:id/status", config.TicketController.UpdateTicketStatus)
			tickets.DELETE("/:id", config.TicketController.DeleteTicket)
			tickets.POST("/:id/assign", config.TicketController.AssignTicket)
			tickets.POST("/:id/resolve", config.TicketController.ResolveTicket)
			tickets.POST("/:id/close", config.TicketController.CloseTicket)

			// 工单SLA信息
			tickets.GET("/:id/sla", config.TicketController.GetTicketSLAInfo)

			// 子任务管理
			tickets.GET("/:id/subtasks", config.TicketController.GetSubtasks)
			tickets.POST("/:id/subtasks", config.TicketController.CreateSubtask)
			tickets.PATCH("/:id/subtasks/:subtask_id", config.TicketController.UpdateSubtask)
			tickets.DELETE("/:id/subtasks/:subtask_id", config.TicketController.DeleteSubtask)

			// 评论
			if config.TicketCommentController != nil {
				tickets.GET("/:id/comments", config.TicketCommentController.ListTicketComments)
				tickets.POST("/:id/comments", config.TicketCommentController.CreateTicketComment)
				tickets.PUT("/:id/comments/:comment_id", config.TicketCommentController.UpdateTicketComment)
				tickets.DELETE("/:id/comments/:comment_id", config.TicketCommentController.DeleteTicketComment)
			}

			// 附件
			if config.TicketAttachmentController != nil {
				tickets.GET("/:id/attachments", config.TicketAttachmentController.ListTicketAttachments)
				tickets.POST("/:id/attachments", config.TicketAttachmentController.UploadAttachment)
				tickets.DELETE("/:id/attachments/:attachment_id", config.TicketAttachmentController.DeleteAttachment)
			}

			// 审批流程管理
			if config.ApprovalController != nil {
				tickets.POST("/approval/submit", config.ApprovalController.SubmitApproval)
				tickets.GET("/approval/records", config.ApprovalController.GetApprovalRecords)

				// 审批工作流CRUD
				approvalWorkflows := tenant.(*gin.RouterGroup).Group("/approval-workflows")
				{
					approvalWorkflows.GET("", config.ApprovalController.ListWorkflows)
					approvalWorkflows.POST("", config.ApprovalController.CreateWorkflow)
					approvalWorkflows.GET("/:id", config.ApprovalController.GetWorkflow)
					approvalWorkflows.PUT("/:id", config.ApprovalController.UpdateWorkflow)
					approvalWorkflows.PATCH("/:id", config.ApprovalController.PatchWorkflow)
					approvalWorkflows.DELETE("/:id", config.ApprovalController.DeleteWorkflow)
				}
			}

			// 工单流转工作流
			if config.TicketWorkflowController != nil {
				tickets.POST("/workflow/accept", config.TicketWorkflowController.AcceptTicket)
				tickets.POST("/workflow/reject", config.TicketWorkflowController.RejectTicket)
				tickets.POST("/workflow/withdraw", config.TicketWorkflowController.WithdrawTicket)
				tickets.POST("/workflow/forward", config.TicketWorkflowController.ForwardTicket)
				tickets.POST("/workflow/approve", config.TicketWorkflowController.ApproveTicket)
				tickets.POST("/workflow/resolve", config.TicketWorkflowController.ResolveTicket)
				tickets.POST("/workflow/close", config.TicketWorkflowController.CloseTicket)
				tickets.POST("/workflow/reopen", config.TicketWorkflowController.ReopenTicket)
				tickets.GET("/:id/workflow/state", config.TicketWorkflowController.GetTicketWorkflowState)
				tickets.GET("/:id/workflow-history", config.TicketWorkflowController.GetTicketWorkflowHistory)
				tickets.GET("/:id/workflow_records", config.TicketWorkflowController.GetTicketWorkflowHistory)
			}

			// 工单自动化规则
			if config.TicketAutomationRuleController != nil {
				tickets.GET("/automation-rules", config.TicketAutomationRuleController.ListAutomationRules)
				tickets.POST("/automation-rules", config.TicketAutomationRuleController.CreateAutomationRule)
				tickets.GET("/automation-rules/:id", config.TicketAutomationRuleController.GetAutomationRule)
				tickets.PUT("/automation-rules/:id", config.TicketAutomationRuleController.UpdateAutomationRule)
				tickets.DELETE("/automation-rules/:id", config.TicketAutomationRuleController.DeleteAutomationRule)
				tickets.POST("/automation-rules/:id/test", config.TicketAutomationRuleController.TestAutomationRule)
			}

			// 工单分配规则
			if config.TicketAssignmentSmartController != nil {
				tickets.GET("/assignment-rules", config.TicketAssignmentSmartController.ListAssignmentRules)
				tickets.POST("/assignment-rules", config.TicketAssignmentSmartController.CreateAssignmentRule)
				tickets.GET("/assignment-rules/:id", config.TicketAssignmentSmartController.GetAssignmentRule)
				tickets.PUT("/assignment-rules/:id", config.TicketAssignmentSmartController.UpdateAssignmentRule)
				tickets.DELETE("/assignment-rules/:id", config.TicketAssignmentSmartController.DeleteAssignmentRule)
			}

			// 工单评分
			if config.TicketRatingController != nil {
				tickets.POST("/:id/rating", config.TicketRatingController.SubmitRating)
				tickets.GET("/:id/rating", config.TicketRatingController.GetRating)
				tickets.GET("/rating-stats", config.TicketRatingController.GetRatingStats)
			}

			// 工单模板
			tickets.GET("/templates", config.TicketController.GetTicketTemplates)
			tickets.POST("/templates", config.TicketController.CreateTicketTemplate)
			tickets.PUT("/templates/:template_id", config.TicketController.UpdateTicketTemplate)
			tickets.DELETE("/templates/:template_id", config.TicketController.DeleteTicketTemplate)

			// 工单分析与预测 (Alias for frontend compatibility)
			if config.AIHandler != nil {
				tickets.POST("/analytics/deep", config.AIHandler.GetDeepAnalytics)
				tickets.POST("/prediction/trend", config.AIHandler.GetTrendPrediction)
			}

			if config.AnalyticsController != nil {
				tickets.POST("/analytics/export", config.AnalyticsController.ExportAnalytics)
				// B8: GET /api/v1/analytics/tickets - 工单分析概览
				// 挂在 tenant 顶层 group，路径 = /api/v1/analytics/tickets
			}

			if config.PredictionController != nil {
				tickets.POST("/prediction/export", config.PredictionController.ExportPredictionReport)
			}
		}

		// ==================== 顶层分析别名 ====================
		if config.AnalyticsController != nil {
			tenant.(*gin.RouterGroup).GET("/analytics/tickets", config.AnalyticsController.GetTicketAnalytics)
		}

		// B9: AI 工单总结 /ai/tickets/{id}/summary
		aiGroup := tenant.(*gin.RouterGroup).Group("/ai")
		{
			aiGroup.GET("/tickets/:id/summary", config.AIHandler.SummarizeTicket)
		}

		// ==================== System Configs ====================
		if config.SystemConfigController != nil {
			sysConfigs := tenant.(*gin.RouterGroup).Group("/system-configs")
			{
				// 配置管理
				sysConfigs.GET("", config.SystemConfigController.ListConfigs)
				sysConfigs.GET("/init", config.SystemConfigController.InitDefaultConfigs)
				sysConfigs.GET("/:id", config.SystemConfigController.GetConfig)
				sysConfigs.GET("/key/:key", config.SystemConfigController.GetConfigByKey)
				sysConfigs.PUT("/:id", config.SystemConfigController.UpdateConfig)
				sysConfigs.PUT("/batch", config.SystemConfigController.BatchUpdateConfigs)

				// 系统状态
				sysConfigs.GET("/status", func(c *gin.Context) {
					var m runtime.MemStats
					runtime.ReadMemStats(&m)

					c.JSON(200, gin.H{
						"cpu": gin.H{
							"usage": 0,
							"cores": runtime.NumCPU(),
						},
						"memory": gin.H{
							"used":  m.Alloc / 1024 / 1024,
							"total": m.Sys / 1024 / 1024,
							"usage": float64(m.Alloc) / float64(m.Sys) * 100,
						},
						"goroutines": runtime.NumGoroutine(),
						"timestamp":  time.Now(),
					})
				})
			}
		}

		// ==================== Approval Chains ====================
		if config.ApprovalChainController != nil {
			approvalChains := tenant.(*gin.RouterGroup).Group("/approval-chains")
			{
				approvalChains.GET("", config.ApprovalChainController.ListChains)
				approvalChains.GET("/stats", config.ApprovalChainController.GetStats)
				approvalChains.POST("", config.ApprovalChainController.CreateChain)
				approvalChains.GET("/:id", config.ApprovalChainController.GetChain)
				approvalChains.PUT("/:id", config.ApprovalChainController.UpdateChain)
				approvalChains.DELETE("/:id", config.ApprovalChainController.DeleteChain)
			}
		}

		// ==================== Incidents ====================
		if config.IncidentController != nil {
			inc := tenant.(*gin.RouterGroup).Group("/incidents")
			{
				// 核心 CRUD
				inc.GET("", middleware.RequirePermission("incident", "read"), config.IncidentController.ListIncidents)
				inc.POST("", middleware.RequirePermission("incident", "write"), config.IncidentController.CreateIncident)
				inc.GET("/stats", middleware.RequirePermission("incident", "read"), config.IncidentController.GetIncidentStats)
				inc.GET("/:id", middleware.RequirePermission("incident", "read"), config.IncidentController.GetIncident)
				inc.PUT("/:id", middleware.RequirePermission("incident", "write"), config.IncidentController.UpdateIncident)
				inc.DELETE("/:id", middleware.RequirePermission("incident", "delete"), config.IncidentController.DeleteIncident)

				// 事件操作
				inc.POST("/:id/escalate", middleware.RequirePermission("incident", "write"), config.IncidentController.EscalateIncident)
				inc.POST("/:id/acknowledge", middleware.RequirePermission("incident", "write"), config.IncidentController.AcknowledgeIncident)
				inc.POST("/:id/resolve", middleware.RequirePermission("incident", "write"), config.IncidentController.ResolveIncident)
				inc.POST("/:id/close", middleware.RequirePermission("incident", "write"), config.IncidentController.CloseIncident)
				inc.POST("/:id/convert-to-problem", middleware.RequirePermission("incident", "write"), config.IncidentController.ConvertToProblem)
				inc.GET("/:id/impact", middleware.RequirePermission("incident", "read"), config.IncidentController.AnalyzeIncidentImpact)

				// 关联数据
				inc.GET("/:id/events", middleware.RequirePermission("incident", "read"), config.IncidentController.GetIncidentEvents)
				inc.POST("/events", middleware.RequirePermission("incident", "write"), config.IncidentController.CreateIncidentEvent)
				inc.GET("/:id/alerts", middleware.RequirePermission("incident", "read"), config.IncidentController.GetIncidentAlerts)
				inc.GET("/:id/metrics", middleware.RequirePermission("incident", "read"), config.IncidentController.GetIncidentMetrics)

				// 根因分析
				inc.GET("/:id/root-cause", middleware.RequirePermission("incident", "read"), config.IncidentController.GetRootCause)
				inc.PUT("/:id/root-cause", middleware.RequirePermission("incident", "write"), config.IncidentController.UpdateRootCause)

				// 影响评估
				inc.GET("/:id/impact-assessment", middleware.RequirePermission("incident", "read"), config.IncidentController.GetImpactAssessment)
				inc.PUT("/:id/impact-assessment", middleware.RequirePermission("incident", "write"), config.IncidentController.UpdateImpactAssessment)

				// 事件分类
				inc.GET("/:id/classification", middleware.RequirePermission("incident", "read"), config.IncidentController.GetClassification)
				inc.PUT("/:id/classification", middleware.RequirePermission("incident", "write"), config.IncidentController.UpdateClassification)

				// 评论
				inc.GET("/:id/comments", middleware.RequirePermission("incident", "read"), config.IncidentController.GetIncidentComments)
				inc.POST("/:id/comments", middleware.RequirePermission("incident", "write"), config.IncidentController.CreateIncidentComment)

				// 监控
				inc.POST("/monitoring", middleware.RequirePermission("incident", "read"), config.IncidentController.GetIncidentMonitoring)

				// 告警管理
				inc.POST("/alerts", middleware.RequirePermission("incident", "write"), config.IncidentController.CreateIncidentAlert)
				inc.GET("/alerts/active", middleware.RequirePermission("incident", "read"), config.IncidentController.GetActiveAlerts)
				inc.GET("/alerts/statistics", middleware.RequirePermission("incident", "read"), config.IncidentController.GetAlertStatistics)
				inc.POST("/alerts/:id/acknowledge", middleware.RequirePermission("incident", "write"), config.IncidentController.AcknowledgeAlert)
				inc.POST("/alerts/:id/resolve", middleware.RequirePermission("incident", "write"), config.IncidentController.ResolveAlert)
			}
		}

		// ==================== Service Catalog & Requests (DDD) ====================
		if config.ServiceCatalogHandler != nil {
			sc := tenant.(*gin.RouterGroup).Group("/service-catalogs")
			{
				sc.GET("", middleware.RequirePermission("service_catalog", "read"), config.ServiceCatalogHandler.List)
				sc.POST("", middleware.RequirePermission("service_catalog", "write"), config.ServiceCatalogHandler.Create)
				sc.GET("/search", middleware.RequirePermission("service_catalog", "read"), config.ServiceCatalogHandler.Search)
				sc.GET("/stats", middleware.RequirePermission("service_catalog", "read"), config.ServiceCatalogHandler.Stats)
				sc.GET("/:id", middleware.RequirePermission("service_catalog", "read"), config.ServiceCatalogHandler.Get)
				sc.PUT("/:id", middleware.RequirePermission("service_catalog", "write"), config.ServiceCatalogHandler.Update)
				sc.DELETE("/:id", middleware.RequirePermission("service_catalog", "delete"), config.ServiceCatalogHandler.Delete)
			}
			// 简化的服务项路由
			scServices := tenant.(*gin.RouterGroup).Group("/service-catalog-services")
			{
				scServices.GET("", config.ServiceCatalogHandler.List)
				scServices.GET("/:id", config.ServiceCatalogHandler.Get)
			}
		}

		if config.ServiceRequestHandler != nil {
			sr := tenant.(*gin.RouterGroup).Group("/service-requests")
			{
				sr.POST("", middleware.RequirePermission("service_request", "write"), config.ServiceRequestHandler.Create)
				sr.GET("", middleware.RequirePermission("service_request", "read"), config.ServiceRequestHandler.List)
				sr.GET("/me", middleware.RequirePermission("service_request", "read"), config.ServiceRequestHandler.List)
				sr.GET("/approvals/pending", middleware.RequirePermission("service_request", "read"), config.ServiceRequestHandler.ListPending)
				sr.GET("/:id", middleware.RequirePermission("service_request", "read"), config.ServiceRequestHandler.Get)
				sr.PUT("/:id", middleware.RequirePermission("service_request", "write"), config.ServiceRequestHandler.Update)
				sr.DELETE("/:id", middleware.RequirePermission("service_request", "delete"), config.ServiceRequestHandler.Delete)
				sr.POST("/:id/approval", middleware.RequirePermission("service_request", "write"), config.ServiceRequestHandler.ApplyApproval)
			}

			// Provisioning routes
			if config.ProvisioningController != nil {
				sr.POST("/:id/provision", middleware.RequirePermission("service_request", "write"), config.ProvisioningController.StartProvisioning)
				sr.GET("/:id/provisioning-tasks", middleware.RequirePermission("service_request", "read"), config.ProvisioningController.ListProvisioningTasks)
			}

			// Provisioning task routes (separate path)
			provisioning := tenant.(*gin.RouterGroup).Group("/provisioning-tasks")
			{
				provisioning.POST("/:id/execute", middleware.RequirePermission("service_request", "write"), config.ProvisioningController.ExecuteProvisioningTask)
			}
		}

		// ==================== Problems (DDD) ====================
		if config.ProblemHandler != nil {
			problems := tenant.(*gin.RouterGroup).Group("/problems")
			{
				problems.GET("", middleware.RequirePermission("problem", "read"), config.ProblemHandler.List)
				problems.POST("", middleware.RequirePermission("problem", "write"), config.ProblemHandler.Create)
				problems.GET("/stats", middleware.RequirePermission("problem", "read"), config.ProblemHandler.GetStats)
				problems.GET("/:id", middleware.RequirePermission("problem", "read"), config.ProblemHandler.Get)
				problems.PUT("/:id", middleware.RequirePermission("problem", "write"), config.ProblemHandler.Update)
				problems.DELETE("/:id", middleware.RequirePermission("problem", "delete"), config.ProblemHandler.Delete)
			}
		}

		// ==================== Changes (DDD) ====================
		if config.ChangeHandler != nil {
			changes := tenant.(*gin.RouterGroup).Group("/changes")
			{
				changes.GET("", middleware.RequirePermission("change", "read"), config.ChangeHandler.ListChanges)
				changes.POST("", middleware.RequirePermission("change", "write"), config.ChangeHandler.CreateChange)
				changes.GET("/stats", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetStats)
				changes.GET("/:id", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetChange)
				changes.PUT("/:id", middleware.RequirePermission("change", "write"), config.ChangeHandler.UpdateChange)
				changes.DELETE("/:id", middleware.RequirePermission("change", "delete"), config.ChangeHandler.DeleteChange)
				changes.POST("/:id/submit", middleware.RequirePermission("change", "write"), config.ChangeHandler.SubmitChange)
				changes.POST("/:id/assign", middleware.RequirePermission("change", "write"), config.ChangeHandler.AssignChange)
				// 状态转换
				changes.POST("/:id/approve", middleware.RequirePermission("change", "write"), config.ChangeHandler.TransitionStatus)
				changes.POST("/:id/reject", middleware.RequirePermission("change", "write"), config.ChangeHandler.TransitionStatus)
				changes.POST("/:id/start", middleware.RequirePermission("change", "write"), config.ChangeHandler.TransitionStatus)
				changes.POST("/:id/complete", middleware.RequirePermission("change", "write"), config.ChangeHandler.TransitionStatus)
				changes.POST("/:id/rollback", middleware.RequirePermission("change", "write"), config.ChangeHandler.TransitionStatus)
				changes.POST("/:id/cancel", middleware.RequirePermission("change", "write"), config.ChangeHandler.TransitionStatus)
				// 审批
				changes.GET("/:id/approvals", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetApprovals)
				changes.POST("/:id/approvals", middleware.RequirePermission("change", "write"), config.ChangeHandler.SubmitApproval)
				changes.GET("/:id/approval-summary", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetApprovalSummary)
				// 风险评估（同时支持 /risk 和 /risk-assessment 两个路径）
				changes.GET("/:id/risk-assessment", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetRiskAssessment)
				changes.GET("/:id/risk", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetRiskAssessment)
				// 日历视图
				changes.GET("/calendar", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetCalendar)
				// PIR (Post-Implementation Review)
				changes.GET("/pirs", middleware.RequirePermission("change", "read"), config.ChangeHandler.ListPIRs)
				changes.GET("/:id/pir", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetPIR)
				changes.POST("/:id/pir", middleware.RequirePermission("change", "write"), config.ChangeHandler.CreatePIR)
				changes.PUT("/pir/:id", middleware.RequirePermission("change", "write"), config.ChangeHandler.UpdatePIR)
				changes.DELETE("/pir/:id", middleware.RequirePermission("change", "delete"), config.ChangeHandler.DeletePIR)
			}
		}

		// ==================== Releases ====================
		if config.ReleaseController != nil {
			releases := tenant.(*gin.RouterGroup).Group("/releases")
			{
				releases.GET("", middleware.RequirePermission("release", "read"), config.ReleaseController.ListReleases)
				releases.POST("", middleware.RequirePermission("release", "write"), config.ReleaseController.CreateRelease)
				releases.GET("/stats", middleware.RequirePermission("release", "read"), config.ReleaseController.GetReleaseStats)
				releases.GET("/:id", middleware.RequirePermission("release", "read"), config.ReleaseController.GetRelease)
				releases.PUT("/:id", middleware.RequirePermission("release", "write"), config.ReleaseController.UpdateRelease)
				releases.PUT("/:id/status", middleware.RequirePermission("release", "write"), config.ReleaseController.UpdateReleaseStatus)
				releases.DELETE("/:id", middleware.RequirePermission("release", "delete"), config.ReleaseController.DeleteRelease)
			}
		}

		// ==================== Assets ====================
		if config.AssetController != nil {
			assets := tenant.(*gin.RouterGroup).Group("/assets")
			{
				assets.GET("", middleware.RequirePermission("asset", "read"), config.AssetController.ListAssets)
				assets.POST("", middleware.RequirePermission("asset", "write"), config.AssetController.CreateAsset)
				assets.GET("/stats", middleware.RequirePermission("asset", "read"), config.AssetController.GetAssetStats)
				assets.GET("/:id", middleware.RequirePermission("asset", "read"), config.AssetController.GetAsset)
				assets.PUT("/:id", middleware.RequirePermission("asset", "write"), config.AssetController.UpdateAsset)
				assets.PUT("/:id/status", middleware.RequirePermission("asset", "write"), config.AssetController.UpdateAssetStatus)
				assets.PUT("/:id/assign", middleware.RequirePermission("asset", "write"), config.AssetController.AssignAsset)
				assets.PUT("/:id/retire", middleware.RequirePermission("asset", "write"), config.AssetController.RetireAsset)
				assets.DELETE("/:id", middleware.RequirePermission("asset", "delete"), config.AssetController.DeleteAsset)
			}
		}

		// ==================== Asset Licenses ====================
		if config.AssetLicenseController != nil {
			licenses := tenant.(*gin.RouterGroup).Group("/licenses")
			{
				licenses.GET("", middleware.RequirePermission("license", "read"), config.AssetLicenseController.ListLicenses)
				licenses.POST("", middleware.RequirePermission("license", "write"), config.AssetLicenseController.CreateLicense)
				licenses.GET("/stats", middleware.RequirePermission("license", "read"), config.AssetLicenseController.GetLicenseStats)
				licenses.GET("/:id", middleware.RequirePermission("license", "read"), config.AssetLicenseController.GetLicense)
				licenses.PUT("/:id", middleware.RequirePermission("license", "write"), config.AssetLicenseController.UpdateLicense)
				licenses.PUT("/:id/assign", middleware.RequirePermission("license", "write"), config.AssetLicenseController.AssignUsers)
				licenses.DELETE("/:id", middleware.RequirePermission("license", "delete"), config.AssetLicenseController.DeleteLicense)
			}
		}

		// ==================== Vendors ====================
		if config.VendorController != nil {
			vendors := tenant.(*gin.RouterGroup).Group("/vendors")
			{
				vendors.GET("", middleware.RequirePermission("vendor", "read"), config.VendorController.ListVendors)
				vendors.POST("", middleware.RequirePermission("vendor", "write"), config.VendorController.CreateVendor)
				vendors.GET("/:id", middleware.RequirePermission("vendor", "read"), config.VendorController.GetVendor)
				vendors.DELETE("/:id", middleware.RequirePermission("vendor", "delete"), config.VendorController.DeleteVendor)
			}
		}

		// ==================== CMDB ====================
		if config.CMDBController != nil {
			cmdb := tenant.(*gin.RouterGroup).Group("/configuration-items")
			{
				// CI CRUD
				cmdb.GET("", middleware.RequirePermission("cmdb", "read"), config.CMDBController.ListCIs)
				cmdb.POST("", middleware.RequirePermission("cmdb", "write"), config.CMDBController.CreateCI)
				cmdb.GET("/stats", middleware.RequirePermission("cmdb", "read"), config.CMDBController.GetStats)
				cmdb.GET("/relationships", middleware.RequirePermission("cmdb", "read"), config.CMDBController.ListRelationships)
				cmdb.POST("/relationships", middleware.RequirePermission("cmdb", "write"), config.CMDBController.CreateRelationship)
				cmdb.PATCH("/relationships/:id", middleware.RequirePermission("cmdb", "write"), config.CMDBController.UpdateRelationship)
				cmdb.DELETE("/relationships/:id", middleware.RequirePermission("cmdb", "delete"), config.CMDBController.DeleteRelationship)
				cmdb.GET("/relationship-types", middleware.RequirePermission("cmdb", "read"), config.CMDBController.ListRelationshipTypes)
				// Types
				cmdb.GET("/types", middleware.RequirePermission("cmdb", "read"), config.CMDBController.ListTypes)
				cmdb.POST("/types", middleware.RequirePermission("cmdb", "write"), config.CMDBController.CreateType)
				cmdb.PUT("/types/:id", middleware.RequirePermission("cmdb", "write"), config.CMDBController.UpdateType)
				cmdb.DELETE("/types/:id", middleware.RequirePermission("cmdb", "delete"), config.CMDBController.DeleteType)
				// Reconciliation
				cmdb.GET("/reconciliation", middleware.RequirePermission("cmdb", "read"), config.CMDBController.GetReconciliation)
				cmdb.POST("/reconciliation/run", middleware.RequirePermission("cmdb", "write"), config.CMDBController.RunReconciliation)
				// Cloud
				cmdb.GET("/cloud-services", middleware.RequirePermission("cmdb", "read"), config.CMDBController.ListCloudServices)
				cmdb.POST("/cloud-services", middleware.RequirePermission("cmdb", "write"), config.CMDBController.CreateCloudService)
				cmdb.GET("/cloud-accounts", middleware.RequirePermission("cmdb", "read"), config.CMDBController.ListCloudAccounts)
				cmdb.POST("/cloud-accounts", middleware.RequirePermission("cmdb", "write"), config.CMDBController.CreateCloudAccount)
				cmdb.GET("/cloud-resources", middleware.RequirePermission("cmdb", "read"), config.CMDBController.ListCloudResources)
				// Discovery
				cmdb.GET("/discovery-sources", middleware.RequirePermission("cmdb", "read"), config.CMDBController.ListDiscoverySources)
				cmdb.POST("/discovery-sources", middleware.RequirePermission("cmdb", "write"), config.CMDBController.CreateDiscoverySource)
				cmdb.POST("/discovery/jobs", middleware.RequirePermission("cmdb", "write"), config.CMDBController.CreateDiscoveryJob)
				cmdb.GET("/discovery/results", middleware.RequirePermission("cmdb", "read"), config.CMDBController.ListDiscoveryResults)
				cmdb.GET("/discovery/status", middleware.RequirePermission("cmdb", "read"), config.CMDBController.GetDiscoveryStatus)
				cmdb.POST("/discovery/run", middleware.RequirePermission("cmdb", "write"), config.CMDBController.RunDiscovery)
				// Per-CI routes (must come after static routes)
				cmdb.GET("/:id", middleware.RequirePermission("cmdb", "read"), config.CMDBController.GetCI)
				cmdb.PUT("/:id", middleware.RequirePermission("cmdb", "write"), config.CMDBController.UpdateCI)
				cmdb.DELETE("/:id", middleware.RequirePermission("cmdb", "delete"), config.CMDBController.DeleteCI)
				cmdb.GET("/:id/topology", middleware.RequirePermission("cmdb", "read"), config.CMDBController.GetCITopology)
				cmdb.GET("/:id/impact-analysis", middleware.RequirePermission("cmdb", "read"), config.CMDBController.GetCIImpactAnalysis)
				cmdb.GET("/:id/change-history", middleware.RequirePermission("cmdb", "read"), config.CMDBController.GetCIChangeHistory)
			}
		}

		// ==================== Knowledge Base (DDD) ====================
		if config.KnowledgeHandler != nil {
			// New route structure: /api/v1/knowledge/articles
			knowledgeGrp := tenant.(*gin.RouterGroup).Group("/knowledge")
			{
				// Articles
				articles := knowledgeGrp.Group("/articles")
				{
					articles.GET("", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.ListArticles)
					articles.POST("", middleware.RequirePermission("knowledge", "write"), config.KnowledgeHandler.CreateArticle)
					articles.GET("/:id", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.GetArticle)
					articles.PUT("/:id", middleware.RequirePermission("knowledge", "write"), config.KnowledgeHandler.UpdateArticle)
					articles.DELETE("/:id", middleware.RequirePermission("knowledge", "delete"), config.KnowledgeHandler.DeleteArticle)

					// Comments
					articles.GET("/:id/comments", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.GetArticleComments)
					articles.POST("/:id/comments", middleware.RequirePermission("knowledge", "write"), config.KnowledgeHandler.AddArticleComment)
				}

				// Categories
				knowledgeGrp.GET("/categories", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.GetCategories)

				// Search
				knowledgeGrp.POST("/search", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.SearchArticles)

				// Recommendations
				knowledgeGrp.GET("/recommendations", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.GetRecommendations)
				knowledgeGrp.GET("/recent", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.GetRecentArticles)

				// Stats
				knowledgeGrp.GET("/stats", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.GetStats)
			}

			// Legacy route for backward compatibility: /api/v1/knowledge-articles/*
			kbGrp := tenant.(*gin.RouterGroup).Group("/knowledge-articles")
			{
				kbGrp.GET("", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.ListArticles)
				kbGrp.POST("", middleware.RequirePermission("knowledge", "write"), config.KnowledgeHandler.CreateArticle)
				// 静态路由必须在动态路由 /:id 之前注册，否则 Gin 会将 "categories" 当作 :id 参数
				kbGrp.GET("/categories", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.GetCategories)
				kbGrp.GET("/:id", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.GetArticle)
				kbGrp.PUT("/:id", middleware.RequirePermission("knowledge", "write"), config.KnowledgeHandler.UpdateArticle)
				kbGrp.DELETE("/:id", middleware.RequirePermission("knowledge", "delete"), config.KnowledgeHandler.DeleteArticle)
			}
		}

		// ==================== SLA (DDD) ====================
		if config.SLAHandler != nil {
			slaGrp := tenant.(*gin.RouterGroup).Group("/sla")
			{
				// SLA Definitions
				slaGrp.GET("/stats", middleware.RequirePermission("sla", "read"), config.SLAHandler.GetSLAStats)
				slaGrp.GET("/definitions", middleware.RequirePermission("sla", "read"), config.SLAHandler.ListSLADefinitions)
				slaGrp.POST("/definitions", middleware.RequirePermission("sla", "write"), config.SLAHandler.CreateSLADefinition)
				slaGrp.GET("/definitions/:id", middleware.RequirePermission("sla", "read"), config.SLAHandler.GetSLADefinition)
				slaGrp.PUT("/definitions/:id", middleware.RequirePermission("sla", "write"), config.SLAHandler.UpdateSLADefinition)
				slaGrp.DELETE("/definitions/:id", middleware.RequirePermission("sla", "delete"), config.SLAHandler.DeleteSLADefinition)

				// SLA Alert Rules
				slaGrp.POST("/alert-rules", middleware.RequirePermission("sla", "write"), config.SLAHandler.CreateAlertRule)
				slaGrp.GET("/alert-rules", middleware.RequirePermission("sla", "read"), config.SLAHandler.ListAlertRules)
				slaGrp.GET("/alert-rules/:id", middleware.RequirePermission("sla", "read"), config.SLAHandler.GetAlertRule)
				slaGrp.PUT("/alert-rules/:id", middleware.RequirePermission("sla", "write"), config.SLAHandler.UpdateAlertRule)
				slaGrp.DELETE("/alert-rules/:id", middleware.RequirePermission("sla", "delete"), config.SLAHandler.DeleteAlertRule)

				// SLA Metrics
				slaGrp.GET("/metrics", middleware.RequirePermission("sla", "read"), config.SLAHandler.GetSLAMetrics)

				// SLA Violations
				slaGrp.GET("/violations", middleware.RequirePermission("sla", "read"), config.SLAHandler.GetSLAViolations)
				slaGrp.PUT("/violations/:id", middleware.RequirePermission("sla", "write"), config.SLAHandler.UpdateViolationStatus)

				// SLA Monitoring
				slaGrp.POST("/monitoring", middleware.RequirePermission("sla", "read"), config.SLAHandler.GetSLAMonitoring)

				// SLA Compliance Check
				slaGrp.POST("/check-compliance/:ticketId", middleware.RequirePermission("sla", "read"), config.SLAHandler.CheckSLACompliance)

				// SLA Alert History
				slaGrp.GET("/alert-history", middleware.RequirePermission("sla", "read"), config.SLAHandler.GetAlertHistory)
				slaGrp.GET("/compliance-report", middleware.RequirePermission("sla", "read"), config.SLAHandler.GetSLAComplianceReport)
			}
		}

		// ==================== AI & Analytics (DDD) ====================
		if config.AIHandler != nil {
			aiGrp := tenant.(*gin.RouterGroup).Group("/ai")
			{
				aiGrp.POST("/chat", middleware.RequirePermission("ai", "read"), config.AIHandler.Chat)
				aiGrp.POST("/analytics", middleware.RequirePermission("ai", "read"), config.AIHandler.GetDeepAnalytics)
				aiGrp.POST("/predictions", middleware.RequirePermission("ai", "read"), config.AIHandler.GetTrendPrediction)
				aiGrp.POST("/tickets/:id/analyze", middleware.RequirePermission("ai", "read"), config.AIHandler.AnalyzeTicket)
				aiGrp.POST("/feedback", middleware.RequirePermission("ai", "write"), config.AIHandler.SaveFeedback)
				aiGrp.POST("/audit", middleware.RequirePermission("ai", "write"), config.AIHandler.RecordAudit)
				aiGrp.GET("/metrics", middleware.RequirePermission("ai", "read"), config.AIHandler.GetMetrics)
				aiGrp.POST("/triage", middleware.RequirePermission("ai", "read"), config.AIHandler.Triage)
				// RAG endpoints
				aiGrp.GET("/rag/search", middleware.RequirePermission("ai", "read"), config.AIHandler.KnowledgeSearch)
				aiGrp.POST("/rag/ask", middleware.RequirePermission("ai", "read"), config.AIHandler.Chat)
				// AI 工单智能创建与总结
				aiGrp.POST("/ticket/create", middleware.RequirePermission("ticket", "create"), config.AIHandler.CreateTicketByAI)
				aiGrp.POST("/tickets/:id/summarize", middleware.RequirePermission("ticket", "read"), config.AIHandler.SummarizeTicketPost)
			}

			agentGrp := tenant.(*gin.RouterGroup).Group("/agent")
			{
				agentGrp.GET("/tools", middleware.RequirePermission("ai", "read"), config.AIHandler.ListTools)
				agentGrp.POST("/tools/execute", middleware.RequirePermission("ai", "read"), config.AIHandler.ExecuteTool)
			}
		}

		// ==================== Common & System (DDD) ====================
		if config.CommonHandler != nil {
			// Auth scoped
			// 修复 P0：auth/me, auth/tenants, auth/menus 不应在 tenant 分组内
			// 否则 tenant RBAC 中间件会拦截端用户，导致 layout 崩溃
			authGrp := r.Group("/api/v1/auth")
			{
				authGrp.GET("/me", middleware.AuthMiddleware(config.JWTSecret), config.CommonHandler.GetMe)
				authGrp.GET("/tenants", middleware.AuthMiddleware(config.JWTSecret), config.CommonHandler.GetUserTenants)
				authGrp.POST("/logout", middleware.AuthMiddleware(config.JWTSecret), config.CommonHandler.Logout)
			}

			// User Menu (no permission required, will be filtered by role)
			if config.MenuController != nil {
				authGrp.GET("/menus", middleware.AuthMiddleware(config.JWTSecret), config.MenuController.GetUserMenus)
			}

			// Audit Logs (short path for frontend compatibility)
			tenant.GET("/audit-logs", middleware.RequirePermission("audit", "read"), config.CommonHandler.GetAuditLogs)

			// Users
			if config.UserController != nil {
				users := tenant.(*gin.RouterGroup).Group("/users")
				{
					users.GET("", middleware.RequirePermission("user", "read"), config.UserController.ListUsers)
					users.POST("", middleware.RequirePermission("user", "write"), config.UserController.CreateUser)
					users.GET("/:id", middleware.RequirePermission("user", "read"), config.UserController.GetUser)
					users.PUT("/:id", middleware.RequirePermission("user", "write"), config.UserController.UpdateUser)
					users.DELETE("/:id", middleware.RequirePermission("user", "delete"), config.UserController.DeleteUser)
					users.PUT("/:id/status", middleware.RequirePermission("user", "write"), config.UserController.ChangeUserStatus)
					users.PUT("/:id/reset-password", middleware.RequirePermission("user", "write"), config.UserController.ResetPassword)
					users.GET("/stats", middleware.RequirePermission("user", "read"), config.UserController.GetUserStats)
					users.POST("/batch", middleware.RequirePermission("user", "write"), config.UserController.BatchUpdateUsers)
				}
			} else {
				users := tenant.(*gin.RouterGroup).Group("/users")
				{
					users.GET("", middleware.RequirePermission("user", "read"), config.CommonHandler.ListUsers)
				}
			}

			// Groups
			if config.GroupController != nil {
				groups := tenant.(*gin.RouterGroup).Group("/groups")
				{
					groups.GET("", middleware.RequirePermission("groups", "read"), config.GroupController.ListGroups)
					groups.POST("", middleware.RequirePermission("groups", "write"), config.GroupController.CreateGroup)
					groups.GET("/:id", middleware.RequirePermission("groups", "read"), config.GroupController.GetGroup)
					groups.PUT("/:id", middleware.RequirePermission("groups", "write"), config.GroupController.UpdateGroup)
					groups.DELETE("/:id", middleware.RequirePermission("groups", "write"), config.GroupController.DeleteGroup)
					groups.POST("/:id/members", middleware.RequirePermission("groups", "write"), config.GroupController.AddUserToGroup)
					groups.DELETE("/:id/members", middleware.RequirePermission("groups", "write"), config.GroupController.RemoveUserFromGroup)
					groups.GET("/:id/members", middleware.RequirePermission("groups", "read"), config.GroupController.GetGroupMembers)
				}
			}

			// Organization
			org := tenant.(*gin.RouterGroup).Group("/org")
			{
				org.GET("/departments/tree", middleware.RequirePermission("org", "read"), config.CommonHandler.GetDepartmentTree)
				org.POST("/departments", middleware.RequirePermission("org", "write"), config.CommonHandler.CreateDepartment)
				org.PUT("/departments/:id", middleware.RequirePermission("org", "write"), config.CommonHandler.UpdateDepartment)
				org.DELETE("/departments/:id", middleware.RequirePermission("org", "write"), config.CommonHandler.DeleteDepartment)
				org.GET("/teams", middleware.RequirePermission("org", "read"), config.CommonHandler.ListTeams)
				org.POST("/teams", middleware.RequirePermission("org", "write"), config.CommonHandler.CreateTeam)
				org.PUT("/teams/:id", middleware.RequirePermission("org", "write"), config.CommonHandler.UpdateTeam)
				org.DELETE("/teams/:id", middleware.RequirePermission("org", "write"), config.CommonHandler.DeleteTeam)
			}

			// Projects
			if config.ProjectController != nil {
				projects := tenant.(*gin.RouterGroup).Group("/projects")
				{
					projects.GET("", middleware.RequirePermission("project", "read"), config.ProjectController.ListProjects)
					projects.POST("", middleware.RequirePermission("project", "write"), config.ProjectController.CreateProject)
					projects.GET("/:id", middleware.RequirePermission("project", "read"), config.ProjectController.GetProject)
					projects.PUT("/:id", middleware.RequirePermission("project", "write"), config.ProjectController.UpdateProject)
					projects.DELETE("/:id", middleware.RequirePermission("project", "write"), config.ProjectController.DeleteProject)
				}
			}

			// Applications
			if config.ApplicationController != nil {
				applications := tenant.(*gin.RouterGroup).Group("/applications")
				{
					applications.GET("", middleware.RequirePermission("application", "read"), config.ApplicationController.ListApplications)
					applications.POST("", middleware.RequirePermission("application", "write"), config.ApplicationController.CreateApplication)
					applications.GET("/microservices", middleware.RequirePermission("application", "read"), config.ApplicationController.ListMicroservices)
					applications.POST("/microservices", middleware.RequirePermission("application", "write"), config.ApplicationController.CreateMicroservice)
				}
			}

			// Service Catalog & Service Requests
			if config.ServiceController != nil {
				services := tenant.(*gin.RouterGroup).Group("/services")
				{
					services.GET("/catalogs", config.ServiceController.GetServiceCatalogs)
					services.POST("/catalogs", config.ServiceController.CreateServiceCatalog)
					services.GET("/catalogs/:id", config.ServiceController.GetServiceCatalogByID)
					services.PUT("/catalogs/:id", config.ServiceController.UpdateServiceCatalog)
					services.DELETE("/catalogs/:id", config.ServiceController.DeleteServiceCatalog)
					services.GET("/requests", config.ServiceController.GetUserServiceRequests)
					services.POST("/requests", config.ServiceController.CreateServiceRequest)
					services.GET("/requests/:id", config.ServiceController.GetServiceRequestByID)
					services.PUT("/requests/:id/status", config.ServiceController.UpdateServiceRequestStatus)
					services.POST("/requests/approval", config.ServiceController.ApplyServiceRequestApproval)
					services.GET("/requests/approvals", config.ServiceController.GetServiceRequestApprovals)
					services.GET("/requests/approvals/pending", config.ServiceController.GetPendingServiceRequestApprovals)
				}
			}

			// Ticket Dependencies
			if config.TicketDependencyController != nil {
				tickets.GET("/:id/dependencies", config.TicketDependencyController.AnalyzeDependencyImpact)
			}

			// Ticket Notifications
			if config.TicketNotificationController != nil {
				tickets.GET("/:id/notifications", config.TicketNotificationController.ListTicketNotifications)
				tickets.POST("/:id/notifications", config.TicketNotificationController.SendTicketNotification)
			}

			// System
			sys := tenant.(*gin.RouterGroup).Group("/system")
			{
				sys.GET("/tags", config.CommonHandler.ListTags)
				sys.GET("/audit-logs", config.CommonHandler.GetAuditLogs)
			}

			// B7/Bug10: 顶层路由别名（兼容前端 /api/v1/{departments,teams,tags,admin/tenants} 旧路径）
			// 实际业务仍在 /org/* 与 /system/* 与 /tenants，旧路径保持只读 GET，避免破坏现有调用方
			{
				tenant.GET("/departments", config.CommonHandler.ListDepartments)
				tenant.GET("/teams", config.CommonHandler.ListTeams)
				tenant.GET("/tags", config.CommonHandler.ListTags)
			}
			if config.TenantController != nil {
				admin := tenant.(*gin.RouterGroup).Group("/admin")
				{
					admin.GET("/tenants", config.TenantController.ListTenants)
				}
			}

			// Role & Permission Controllers (database-backed with tenant isolation)
			if config.RoleController != nil {
				roles := tenant.(*gin.RouterGroup).Group("/roles")
				{
					roles.GET("", config.RoleController.ListRoles)
					roles.POST("", config.RoleController.CreateRole)
					roles.GET("/:id", config.RoleController.GetRole)
					roles.PUT("/:id", config.RoleController.UpdateRole)
					roles.DELETE("/:id", config.RoleController.DeleteRole)
					roles.POST("/:id/permissions", config.RoleController.AssignPermissions)
				}
			}

			if config.PermissionController != nil {
				permissions := tenant.(*gin.RouterGroup).Group("/permissions")
				{
					permissions.GET("", config.PermissionController.ListPermissions)
					permissions.POST("", config.PermissionController.CreatePermission)
					permissions.POST("/init", config.PermissionController.InitDefaultPermissions)
				}
			}

			// Menu Controllers (database-backed with tenant isolation)
			if config.MenuController != nil {
				menus := tenant.(*gin.RouterGroup).Group("/menus")
				{
					menus.GET("", config.MenuController.ListMenus)
					menus.POST("", config.MenuController.CreateMenu)
					menus.GET("/:id", config.MenuController.GetMenu)
					menus.PUT("/:id", config.MenuController.UpdateMenu)
					menus.DELETE("/:id", config.MenuController.DeleteMenu)
					menus.POST("/init", config.MenuController.InitDefaultMenus)
				}
			}

			// Tenant Management (admin only)
			if config.TenantController != nil {
				tenants := tenant.(*gin.RouterGroup).Group("/tenants")
				{
					tenants.GET("", config.TenantController.ListTenants)
					tenants.POST("", config.TenantController.CreateTenant)
					tenants.GET("/:id", config.TenantController.GetTenant)
					tenants.PUT("/:id", config.TenantController.UpdateTenant)
					tenants.DELETE("/:id", config.TenantController.DeleteTenant)
					tenants.PUT("/:id/status", config.TenantController.UpdateTenantStatus)
				}
			}

			// Notification Preferences
			config.Logger.Info("NotificationPreferenceController check:", zap.Any("controller", config.NotificationPreferenceController))
			if config.NotificationPreferenceController != nil {
				config.Logger.Info("Registering notification-preferences routes")
				notifPrefs := tenant.(*gin.RouterGroup).Group("/notification-preferences")
				{
					notifPrefs.GET("", config.NotificationPreferenceController.ListPreferences)
					notifPrefs.GET("/event-types", config.NotificationPreferenceController.ListEventTypes)
					notifPrefs.GET("/:event_type", config.NotificationPreferenceController.GetPreference)
					notifPrefs.POST("", config.NotificationPreferenceController.CreateOrUpdatePreference)
					notifPrefs.PUT("", config.NotificationPreferenceController.BulkUpdatePreferences)
					notifPrefs.DELETE("/:event_type", config.NotificationPreferenceController.DeletePreference)
					notifPrefs.POST("/reset", config.NotificationPreferenceController.ResetPreferences)
					notifPrefs.POST("/init", config.NotificationPreferenceController.InitializeDefaultPreferences)
				}
			}

			// Notifications
			if config.NotificationController != nil {
				notifications := tenant.(*gin.RouterGroup).Group("/notifications")
				{
					notifications.GET("", config.NotificationController.GetNotifications)
					notifications.GET("/unread-count", config.NotificationController.GetUnreadCount)
					notifications.PUT("/:id/read", config.NotificationController.MarkNotificationRead)
					notifications.PUT("/read-all", config.NotificationController.MarkAllNotificationsRead)
					notifications.DELETE("/:id", config.NotificationController.DeleteNotification)
					notifications.POST("", config.NotificationController.CreateNotification)
				}
			}
		}

		// ==================== Problem Investigation ====================
		if config.ProblemInvestigationController != nil {
			problemInvestigation := tenant.(*gin.RouterGroup).Group("/problem-investigation")
			{
				// 问题调查管理
				problemInvestigation.POST("/investigations", config.ProblemInvestigationController.CreateProblemInvestigation)
				problemInvestigation.GET("/investigations/:id", config.ProblemInvestigationController.GetProblemInvestigation)
				problemInvestigation.PUT("/investigations/:id", config.ProblemInvestigationController.UpdateProblemInvestigation)

				// 调查步骤管理
				problemInvestigation.POST("/steps", config.ProblemInvestigationController.CreateInvestigationStep)
				problemInvestigation.PUT("/steps/:id", config.ProblemInvestigationController.UpdateInvestigationStep)
				problemInvestigation.GET("/investigations/:investigation_id/steps", config.ProblemInvestigationController.GetInvestigationSteps)

				// 根本原因分析
				problemInvestigation.POST("/root-cause-analysis", config.ProblemInvestigationController.CreateRootCauseAnalysis)

				// 解决方案管理
				problemInvestigation.POST("/solutions", config.ProblemInvestigationController.CreateProblemSolution)
				problemInvestigation.GET("/problems/:id/solutions", config.ProblemInvestigationController.GetProblemSolutions)

				// 问题调查摘要
				problemInvestigation.GET("/problems/:id/summary", config.ProblemInvestigationController.GetProblemInvestigationSummary)
			}
		}

		// ==================== BPMN Workflow ====================
		if config.BPMNWorkflowController != nil {
			config.BPMNWorkflowController.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// 简化路由：/workflow/* -> /bpmn/*
		workflow := tenant.(*gin.RouterGroup).Group("/workflow")
		{
			workflow.GET("/instances", config.BPMNWorkflowController.ListProcessInstances)
			workflow.GET("/instances/:id", config.BPMNWorkflowController.GetProcessInstance)
			workflow.POST("/instances", config.BPMNWorkflowController.StartProcess)
			workflow.PUT("/instances/:id/terminate", config.BPMNWorkflowController.TerminateProcess)
			workflow.PUT("/instances/:id/suspend", config.BPMNWorkflowController.SuspendProcess)
			workflow.PUT("/instances/:id/resume", config.BPMNWorkflowController.ResumeProcess)
			// 任务
			workflow.GET("/tasks", config.BPMNWorkflowController.ListUserTasks)
			workflow.POST("/tasks/:id/complete", config.BPMNWorkflowController.CompleteTask)
			workflow.POST("/tasks/:id/claim", config.BPMNWorkflowController.ClaimTask)
		}

		// BPMN Process Trigger Controller (统一流程触发接口)
		if config.BPMNProcessTriggerController != nil {
			config.BPMNProcessTriggerController.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// BPMN Dashboard Controller (监控仪表盘)
		if config.BPMNDashboardController != nil {
			config.BPMNDashboardController.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// A2UI Ticket Controller (AI-driven UI表单)
		if config.A2UITicketController != nil {
			config.A2UITicketController.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// Global Search Controller (全局搜索)
		if config.GlobalSearchController != nil {
			config.GlobalSearchController.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// Standard Change Handler (标准变更模板库)
		if config.StandardChangeHandler != nil {
			config.StandardChangeHandler.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// Known Error Handler (KEDB)
		if config.KnownErrorHandler != nil {
			config.KnownErrorHandler.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		if config.MarketplaceController != nil {
			config.MarketplaceController.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// Connector Controller (连接器/插件/技能市场)
		if config.ConnectorController != nil {
			conns := tenant.(*gin.RouterGroup).Group("/connectors")
			{
				conns.GET("", config.ConnectorController.ListMarket)
				conns.GET("/configs", config.ConnectorController.ListConfigs)
				conns.GET("/lifecycle", config.ConnectorController.Lifecycle)
				conns.POST("/configs", middleware.RequirePermission("connector", "write"), config.ConnectorController.Provision)
				conns.DELETE("/configs/:name", middleware.RequirePermission("connector", "write"), config.ConnectorController.Revoke)
				conns.POST("/:name/send", middleware.RequirePermission("connector", "write"), config.ConnectorController.Send)
				conns.POST("/:name/test", middleware.RequirePermission("connector", "write"), config.ConnectorController.Test)
				conns.GET("/health", config.ConnectorController.Health)
				// 飞书事件回调（独立签名校验）
				conns.POST("/feishu/callback", config.ConnectorController.FeishuCallback)
			}
		}

		if config.DashboardHandler != nil {
			dashboard := tenant.(*gin.RouterGroup).Group("/dashboard")
			{
				// B5: 别名，/api/v1/dashboard 直接返回 overview 数据（前端默认调用）
				dashboard.GET("", config.DashboardHandler.GetOverview)
				dashboard.GET("/overview", config.DashboardHandler.GetOverview)
				dashboard.GET("/stats", config.DashboardHandler.GetStats)
				dashboard.GET("/stats/users", config.DashboardHandler.GetUserStats)
				dashboard.GET("/stats/system", config.DashboardHandler.GetSystemStats)
				dashboard.GET("/kpi-metrics", config.DashboardHandler.GetKPIMetrics)
				dashboard.GET("/ticket-trend", config.DashboardHandler.GetTicketTrend)
				dashboard.GET("/incident-distribution", config.DashboardHandler.GetIncidentDistribution)
				dashboard.GET("/sla-data", config.DashboardHandler.GetSLAData)
				dashboard.GET("/satisfaction-data", config.DashboardHandler.GetSatisfactionData)
				dashboard.GET("/quick-actions", config.DashboardHandler.GetQuickActions)
				dashboard.GET("/recent-activities", config.DashboardHandler.GetRecentActivities)
			}
		}

		// ==================== Reports 报表 ====================
		reports := tenant.(*gin.RouterGroup).Group("/reports")
		{
			reports.GET("", func(c *gin.Context) {
				common.Success(c, gin.H{
					"reports": []gin.H{
						{"id": "tickets", "name": "工单报表", "path": "/reports/tickets"},
						{"id": "incidents", "name": "事件报表", "path": "/reports/incidents"},
						{"id": "problems", "name": "问题报表", "path": "/reports/problems"},
						{"id": "changes", "name": "变更报表", "path": "/reports/changes"},
						{"id": "sla", "name": "SLA报表", "path": "/reports/sla"},
						{"id": "cmdb-quality", "name": "CMDB质量报表", "path": "/reports/cmdb-quality"},
						{"id": "catalog-usage", "name": "服务目录使用报表", "path": "/reports/catalog-usage"},
					},
				})
			})
			reports.GET("/tickets", config.DashboardHandler.GetStats)
			reports.GET("/incidents", config.DashboardHandler.GetIncidentDistribution)
			reports.GET("/problems", config.DashboardHandler.GetTicketTrend)
			reports.GET("/changes", config.DashboardHandler.GetTicketTrend)
			reports.GET("/sla", config.DashboardHandler.GetSLAData)
		}

		// ==================== Surveys 客户满意度调查 ====================
		if config.SurveyController != nil {
			surveys := tenant.(*gin.RouterGroup).Group("/surveys")
			{
				surveys.GET("", config.SurveyController.ListSurveys)
				surveys.POST("", middleware.RequirePermission("survey", "write"), config.SurveyController.CreateSurvey)
				surveys.GET("/:id", config.SurveyController.GetSurvey)
				surveys.PUT("/:id", middleware.RequirePermission("survey", "write"), config.SurveyController.UpdateSurvey)
				surveys.GET("/:id/responses", config.SurveyController.GetSurveyResponses)
				surveys.GET("/:id/analytics", config.SurveyController.GetAnalytics)
				surveys.POST("/responses", config.SurveyController.SubmitResponse)
			}
		}
	}
}
