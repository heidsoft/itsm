package router

import (
	"os"
	"runtime"
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/controller"
	"itsm-backend/ent"
	"itsm-backend/handlers"
	"itsm-backend/internal/domain/ai"
	"itsm-backend/internal/domain/change"
	"itsm-backend/internal/domain/cmdb"
	domainCommon "itsm-backend/internal/domain/common"
	"itsm-backend/internal/domain/incident"
	"itsm-backend/internal/domain/knowledge"
	"itsm-backend/internal/domain/problem"
	"itsm-backend/internal/domain/service_catalog"
	"itsm-backend/internal/domain/service_request"
	"itsm-backend/internal/domain/sla"
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
	ProblemInvestigationController *controller.ProblemInvestigationController
	TicketController                *controller.TicketController
	TicketDependencyController      *controller.TicketDependencyController
	TicketCommentController         *controller.TicketCommentController
	TicketAttachmentController      *controller.TicketAttachmentController
	TicketNotificationController    *controller.TicketNotificationController
	TicketRatingController          *controller.TicketRatingController
	TicketAssignmentSmartController *controller.TicketAssignmentSmartController
	TicketViewController            *controller.TicketViewController
	TicketWorkflowController        *controller.TicketWorkflowController
	TicketAutomationRuleController *controller.TicketAutomationRuleController
	IncidentController              *controller.IncidentController
	ApprovalController              *controller.ApprovalController
	BPMNWorkflowController          *controller.BPMNWorkflowController
	BPMNProcessTriggerController   *controller.BPMNProcessTriggerController
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

	// Role & Permission Controllers (new database-backed implementation)
	RoleController                   *controller.RoleController
	PermissionController             *controller.PermissionController
	NotificationPreferenceController *controller.NotificationPreferenceController
	NotificationController        *controller.NotificationController

	// Additional domain controllers
	ServiceController        *controller.ServiceController
	ProblemController        *controller.ProblemController
	ChangeController         *controller.ChangeController
	ChangeApprovalController *controller.ChangeApprovalController
	ProvisioningController   *controller.ProvisioningController
	AnalyticsController      *controller.AnalyticsController
	PredictionController     *controller.PredictionController

	// Domain Handlers
	ServiceCatalogHandler *service_catalog.Handler
	ServiceRequestHandler *service_request.Handler

	IncidentHandler  *incident.Handler
	ProblemHandler   *problem.Handler
	ChangeHandler    *change.Handler
	CMDBHandler      *cmdb.Handler
	KnowledgeHandler *knowledge.Handler
	SLAHandler       *sla.Handler
	AIHandler        *ai.Handler
	CommonHandler    *domainCommon.Handler
	RoleHandler      *common.RoleHandler
}

// SetupRoutes 设置路由
func SetupRoutes(r *gin.Engine, config *RouterConfig) {
	// 全局中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())

	// 安全中间件
	// 速率限制：从环境变量读取，默认每分钟500次请求（测试环境可配置为更高）
	rateLimit := 500
	if envLimit := os.Getenv("RATE_LIMIT"); envLimit != "" {
		if parsed, err := strconv.Atoi(envLimit); err == nil && parsed > 0 {
			rateLimit = parsed
		}
	}
	rateLimiter := middleware.NewRateLimiter(rateLimit, time.Minute)
	r.Use(middleware.RateLimitMiddleware(rateLimiter))
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

		// ==================== Ticket Categories & Tags ====================
		if config.TicketCategoryController != nil {
			categories := tenant.(*gin.RouterGroup).Group("/ticket-categories")
			{
				categories.GET("", config.TicketCategoryController.ListCategories)
				categories.POST("", middleware.RequirePermission("ticket_category", "create"), config.TicketCategoryController.CreateCategory)
				categories.GET("/:id", config.TicketCategoryController.GetCategory)
				categories.PUT("/:id", middleware.RequirePermission("ticket_category", "update"), config.TicketCategoryController.UpdateCategory)
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
					approvalWorkflows.DELETE("/:id", config.ApprovalController.DeleteWorkflow)
				}
			}

			// 工单流转工作流
			if config.TicketWorkflowController != nil {
				tickets.POST("/workflow/accept", config.TicketWorkflowController.AcceptTicket)
				tickets.POST("/workflow/reject", config.TicketWorkflowController.RejectTicket)
				tickets.POST("/workflow/withdraw", config.TicketWorkflowController.WithdrawTicket)
				tickets.POST("/workflow/forward", config.TicketWorkflowController.ForwardTicket)
				tickets.POST("/workflow/cc", config.TicketWorkflowController.CCTicket)
				tickets.POST("/workflow/approve", config.TicketWorkflowController.ApproveTicket)
				tickets.POST("/workflow/resolve", config.TicketWorkflowController.ResolveTicket)
				tickets.POST("/workflow/close", config.TicketWorkflowController.CloseTicket)
				tickets.POST("/workflow/reopen", config.TicketWorkflowController.ReopenTicket)
				tickets.GET("/:id/workflow/state", config.TicketWorkflowController.GetTicketWorkflowState)
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

			// 工单评分
			if config.TicketRatingController != nil {
				tickets.POST("/:id/rating", config.TicketRatingController.SubmitRating)
				tickets.GET("/:id/rating", config.TicketRatingController.GetRating)
				tickets.GET("/rating-stats", config.TicketRatingController.GetRatingStats)
			}

			// 工单分析与预测 (Alias for frontend compatibility)
			if config.AIHandler != nil {
				tickets.POST("/analytics/deep", config.AIHandler.GetDeepAnalytics)
				tickets.POST("/prediction/trend", config.AIHandler.GetTrendPrediction)
			}

			if config.AnalyticsController != nil {
				tickets.POST("/analytics/export", config.AnalyticsController.ExportAnalytics)
			}

			if config.PredictionController != nil {
				tickets.POST("/prediction/export", config.PredictionController.ExportPredictionReport)
			}
		}

		// ==================== System Configs ====================
		sysConfigs := tenant.(*gin.RouterGroup).Group("/system-configs")
		{
			sysConfigs.GET("/status", func(c *gin.Context) {
				// 返回简单的系统状态信息
				// 实际项目中可以集成 prometheus 或 gopsutil
				var m runtime.MemStats
				runtime.ReadMemStats(&m)

				c.JSON(200, gin.H{
					"cpu": gin.H{
						"usage": 0, // 需要额外库支持
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

		// ==================== Incidents (DDD) ====================
		if config.IncidentHandler != nil {
			inc := tenant.(*gin.RouterGroup).Group("/incidents")
			{
				inc.GET("", middleware.RequirePermission("incident", "read"), config.IncidentHandler.List)
				inc.POST("", middleware.RequirePermission("incident", "write"), config.IncidentHandler.Create)
				inc.GET("/stats", middleware.RequirePermission("incident", "read"), config.IncidentHandler.GetStats)
				inc.GET("/:id", middleware.RequirePermission("incident", "read"), config.IncidentHandler.Get)
				inc.PUT("/:id", middleware.RequirePermission("incident", "write"), config.IncidentHandler.Update)
				inc.POST("/:id/escalate", middleware.RequirePermission("incident", "write"), config.IncidentHandler.Escalate)
				inc.GET("/:id/events", middleware.RequirePermission("incident", "read"), config.IncidentHandler.GetIncidentEvents)
				inc.GET("/:id/alerts", middleware.RequirePermission("incident", "read"), config.IncidentHandler.GetIncidentAlerts)
				inc.GET("/:id/metrics", middleware.RequirePermission("incident", "read"), config.IncidentHandler.GetIncidentMetrics)
				inc.GET("/:id/root-cause", middleware.RequirePermission("incident", "read"), config.IncidentHandler.GetRootCause)
				inc.PUT("/:id/root-cause", middleware.RequirePermission("incident", "write"), config.IncidentHandler.UpdateRootCause)
				inc.GET("/:id/impact-assessment", middleware.RequirePermission("incident", "read"), config.IncidentHandler.GetImpactAssessment)
				inc.PUT("/:id/impact-assessment", middleware.RequirePermission("incident", "write"), config.IncidentHandler.UpdateImpactAssessment)
				inc.GET("/:id/classification", middleware.RequirePermission("incident", "read"), config.IncidentHandler.GetClassification)
				inc.PUT("/:id/classification", middleware.RequirePermission("incident", "write"), config.IncidentHandler.UpdateClassification)
			}
		}

		// ==================== Service Catalog & Requests (DDD) ====================
		if config.ServiceCatalogHandler != nil {
			sc := tenant.(*gin.RouterGroup).Group("/service-catalogs")
			{
				sc.GET("", middleware.RequirePermission("service_catalog", "read"), config.ServiceCatalogHandler.List)
				sc.POST("", middleware.RequirePermission("service_catalog", "write"), config.ServiceCatalogHandler.Create)
				sc.GET("/:id", middleware.RequirePermission("service_catalog", "read"), config.ServiceCatalogHandler.Get)
				sc.PUT("/:id", middleware.RequirePermission("service_catalog", "write"), config.ServiceCatalogHandler.Update)
				sc.DELETE("/:id", middleware.RequirePermission("service_catalog", "delete"), config.ServiceCatalogHandler.Delete)
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
				sr.POST("/:id/approval", middleware.RequirePermission("service_request", "write"), config.ServiceRequestHandler.ApplyApproval)
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
				changes.POST("/:id/approvals", middleware.RequirePermission("change", "write"), config.ChangeHandler.SubmitApproval)
				changes.GET("/:id/approval-summary", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetApprovalSummary)
				changes.GET("/:id/risk-assessment", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetRiskAssessment)
			}
		}

		// ==================== CMDB (DDD) ====================
		if config.CMDBHandler != nil {
			cmdbGrp := tenant.(*gin.RouterGroup).Group("/cmdb")
			{
				cmdbGrp.GET("/cis", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.ListCIs)
				cmdbGrp.POST("/cis", middleware.RequirePermission("cmdb", "write"), config.CMDBHandler.CreateCI)
				cmdbGrp.GET("/cis/:id", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.GetCI)
				cmdbGrp.PUT("/cis/:id", middleware.RequirePermission("cmdb", "write"), config.CMDBHandler.UpdateCI)
				cmdbGrp.DELETE("/cis/:id", middleware.RequirePermission("cmdb", "delete"), config.CMDBHandler.DeleteCI)
				cmdbGrp.GET("/stats", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.GetStats)
				cmdbGrp.GET("/types", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.ListTypes)
				cmdbGrp.GET("/reconciliation", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.GetReconciliation)
				cmdbGrp.GET("/relationship-types", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.ListRelationshipTypes)
				cmdbGrp.GET("/cloud-services", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.ListCloudServices)
				cmdbGrp.POST("/cloud-services", middleware.RequirePermission("cmdb", "write"), config.CMDBHandler.CreateCloudService)
				cmdbGrp.GET("/cloud-accounts", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.ListCloudAccounts)
				cmdbGrp.POST("/cloud-accounts", middleware.RequirePermission("cmdb", "write"), config.CMDBHandler.CreateCloudAccount)
				cmdbGrp.GET("/cloud-resources", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.ListCloudResources)
				cmdbGrp.GET("/discovery-sources", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.ListDiscoverySources)
				cmdbGrp.POST("/discovery-sources", middleware.RequirePermission("cmdb", "write"), config.CMDBHandler.CreateDiscoverySource)
				cmdbGrp.POST("/discovery/jobs", middleware.RequirePermission("cmdb", "write"), config.CMDBHandler.CreateDiscoveryJob)
				cmdbGrp.GET("/discovery/results", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.ListDiscoveryResults)
			}
		}

		// CMDB 新控制器路由
		if config.CMDBController != nil {
			cmdb := tenant.(*gin.RouterGroup).Group("/configuration-items")
			{
				cmdb.GET("", middleware.RequirePermission("cmdb", "read"), config.CMDBController.ListCIs)
				cmdb.POST("", middleware.RequirePermission("cmdb", "write"), config.CMDBController.CreateCI)
				cmdb.GET("/:id", middleware.RequirePermission("cmdb", "read"), config.CMDBController.GetCI)
				cmdb.GET("/:id/topology", middleware.RequirePermission("cmdb", "read"), config.CMDBController.GetCITopology)
				cmdb.GET("/:id/impact-analysis", middleware.RequirePermission("cmdb", "read"), config.CMDBController.GetCIImpactAnalysis)
				cmdb.GET("/:id/change-history", middleware.RequirePermission("cmdb", "read"), config.CMDBController.GetCIChangeHistory)
				cmdb.GET("/relationships", middleware.RequirePermission("cmdb", "read"), config.CMDBController.ListRelationships)
				cmdb.POST("/relationships", middleware.RequirePermission("cmdb", "write"), config.CMDBController.CreateRelationship)
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
			}

			// Legacy route for backward compatibility: /api/v1/knowledge-articles/*
			kbGrp := tenant.(*gin.RouterGroup).Group("/knowledge-articles")
			{
				kbGrp.GET("", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.ListArticles)
				kbGrp.POST("", middleware.RequirePermission("knowledge", "write"), config.KnowledgeHandler.CreateArticle)
				kbGrp.GET("/:id", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.GetArticle)
				kbGrp.PUT("/:id", middleware.RequirePermission("knowledge", "write"), config.KnowledgeHandler.UpdateArticle)
				kbGrp.DELETE("/:id", middleware.RequirePermission("knowledge", "delete"), config.KnowledgeHandler.DeleteArticle)
				kbGrp.GET("/categories", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.GetCategories)
			}
		}

		// ==================== SLA (DDD) ====================
		if config.SLAHandler != nil {
			slaGrp := tenant.(*gin.RouterGroup).Group("/sla")
			{
				// SLA Definitions
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
			}
		}

		// ==================== AI & Analytics (DDD) ====================
		if config.AIHandler != nil {
			aiGrp := tenant.(*gin.RouterGroup).Group("/ai")
			{
				aiGrp.POST("/chat", config.AIHandler.Chat)
				aiGrp.POST("/analytics", config.AIHandler.GetDeepAnalytics)
				aiGrp.POST("/predictions", config.AIHandler.GetTrendPrediction)
				aiGrp.POST("/tickets/:id/analyze", config.AIHandler.AnalyzeTicket)
				aiGrp.POST("/feedback", config.AIHandler.SaveFeedback)
			}
		}

		// ==================== Common & System (DDD) ====================
		if config.CommonHandler != nil {
			// Auth scoped
			authGrp := tenant.(*gin.RouterGroup).Group("/auth")
			{
				authGrp.GET("/me", config.CommonHandler.GetMe)
			}

			// Users
			if config.UserController != nil {
				users := tenant.(*gin.RouterGroup).Group("/users")
				{
					users.GET("", config.UserController.ListUsers)
					users.POST("", config.UserController.CreateUser)
					users.GET("/:id", config.UserController.GetUser)
					users.PUT("/:id", config.UserController.UpdateUser)
					users.DELETE("/:id", config.UserController.DeleteUser)
					users.PUT("/:id/status", config.UserController.ChangeUserStatus)
					users.PUT("/:id/reset-password", config.UserController.ResetPassword)
					users.GET("/stats", config.UserController.GetUserStats)
					users.POST("/batch", config.UserController.BatchUpdateUsers)
				}
			} else {
				users := tenant.(*gin.RouterGroup).Group("/users")
				{
					users.GET("", config.CommonHandler.ListUsers)
				}
			}

			// Organization
			org := tenant.(*gin.RouterGroup).Group("/org")
			{
				org.GET("/departments/tree", config.CommonHandler.GetDepartmentTree)
				org.GET("/teams", config.CommonHandler.ListTeams)
			}

			// System
			sys := tenant.(*gin.RouterGroup).Group("/system")
			{
				sys.GET("/tags", config.CommonHandler.ListTags)
				sys.GET("/audit-logs", config.CommonHandler.GetAuditLogs)
			}

			// Roles (old in-memory handler)
			if config.RoleHandler != nil {
				roles := tenant.(*gin.RouterGroup).Group("/roles")
				{
					roles.GET("", config.RoleHandler.ListRoles)
					roles.POST("", config.RoleHandler.CreateRole)
					roles.GET("/:id", config.RoleHandler.GetRole)
					roles.PUT("/:id", config.RoleHandler.UpdateRole)
					roles.DELETE("/:id", config.RoleHandler.DeleteRole)
				}

				// Permissions
				permissions := tenant.(*gin.RouterGroup).Group("/permissions")
				{
					permissions.GET("", config.RoleHandler.ListPermissions)
				}
			}

			// New Role & Permission Controllers (database-backed with tenant isolation)
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

			// Notification Preferences
			if config.NotificationPreferenceController != nil {
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

		// BPMN Process Trigger Controller (统一流程触发接口)
		if config.BPMNProcessTriggerController != nil {
			config.BPMNProcessTriggerController.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		if config.DashboardHandler != nil {
			dashboard := tenant.(*gin.RouterGroup).Group("/dashboard")
			{
				dashboard.GET("/overview", config.DashboardHandler.GetOverview)
				dashboard.GET("/stats", config.DashboardHandler.GetStats)
				dashboard.GET("/kpi-metrics", config.DashboardHandler.GetKPIMetrics)
				dashboard.GET("/ticket-trend", config.DashboardHandler.GetTicketTrend)
				dashboard.GET("/incident-distribution", config.DashboardHandler.GetIncidentDistribution)
				dashboard.GET("/sla-data", config.DashboardHandler.GetSLAData)
				dashboard.GET("/satisfaction-data", config.DashboardHandler.GetSatisfactionData)
				dashboard.GET("/quick-actions", config.DashboardHandler.GetQuickActions)
				dashboard.GET("/recent-activities", config.DashboardHandler.GetRecentActivities)
			}
		}
	}
}
