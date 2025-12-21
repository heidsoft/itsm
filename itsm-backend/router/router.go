package router

import (
	"time"

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
	TicketController                *controller.TicketController
	TicketDependencyController      *controller.TicketDependencyController
	TicketCommentController         *controller.TicketCommentController
	TicketAttachmentController      *controller.TicketAttachmentController
	TicketNotificationController    *controller.TicketNotificationController
	TicketRatingController          *controller.TicketRatingController
	TicketAssignmentSmartController *controller.TicketAssignmentSmartController
	TicketViewController            *controller.TicketViewController
	IncidentController              *controller.IncidentController
	ApprovalController              *controller.ApprovalController
	BPMNWorkflowController          *controller.BPMNWorkflowController
	DashboardHandler                *handlers.DashboardHandler

	// Organization & Project
	ProjectController     *controller.ProjectController
	ApplicationController *controller.ApplicationController

	// Ticket related controllers
	TicketCategoryController *controller.TicketCategoryController
	TicketTagController      *controller.TicketTagController

	// Additional domain controllers
	ServiceController        *controller.ServiceController
	ProblemController        *controller.ProblemController
	ChangeController         *controller.ChangeController
	ChangeApprovalController *controller.ChangeApprovalController
	ProvisioningController   *controller.ProvisioningController

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

	// 公共路由（无需认证）
	public := r.Group("/api/v1")
	{
		if config.CommonHandler != nil {
			public.POST("/auth/login", config.CommonHandler.Login)
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
			tickets.DELETE("/:id", config.TicketController.DeleteTicket)

			// 子任务管理
			tickets.GET("/:id/subtasks", config.TicketController.GetSubtasks)
			tickets.POST("/:id/subtasks", config.TicketController.CreateSubtask)
			tickets.PATCH("/:id/subtasks/:subtask_id", config.TicketController.UpdateSubtask)
			tickets.DELETE("/:id/subtasks/:subtask_id", config.TicketController.DeleteSubtask)

			// 审批流程管理
			if config.ApprovalController != nil {
				tickets.POST("/approval/submit", config.ApprovalController.SubmitApproval)
				tickets.GET("/approval/records", config.ApprovalController.GetApprovalRecords)
			}
		}

		// ==================== Incidents (DDD) ====================
		if config.IncidentHandler != nil {
			inc := tenant.(*gin.RouterGroup).Group("/incidents")
			{
				inc.GET("", middleware.RequirePermission("incident", "read"), config.IncidentHandler.List)
				inc.POST("", middleware.RequirePermission("incident", "write"), config.IncidentHandler.Create)
				inc.GET("/:id", middleware.RequirePermission("incident", "read"), config.IncidentHandler.Get)
				inc.PUT("/:id", middleware.RequirePermission("incident", "write"), config.IncidentHandler.Update)
				inc.POST("/:id/escalate", middleware.RequirePermission("incident", "write"), config.IncidentHandler.Escalate)
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
			}
		}

		// ==================== Knowledge Base (DDD) ====================
		if config.KnowledgeHandler != nil {
			kb := tenant.(*gin.RouterGroup).Group("/knowledge-articles")
			{
				kb.GET("", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.ListArticles)
				kb.POST("", middleware.RequirePermission("knowledge", "write"), config.KnowledgeHandler.CreateArticle)
				kb.GET("/categories", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.GetCategories)
				kb.GET("/:id", middleware.RequirePermission("knowledge", "read"), config.KnowledgeHandler.GetArticle)
				kb.PUT("/:id", middleware.RequirePermission("knowledge", "write"), config.KnowledgeHandler.UpdateArticle)
				kb.DELETE("/:id", middleware.RequirePermission("knowledge", "delete"), config.KnowledgeHandler.DeleteArticle)
			}
		}

		// ==================== SLA (DDD) ====================
		if config.SLAHandler != nil {
			slaGrp := tenant.(*gin.RouterGroup).Group("/sla")
			{
				slaGrp.GET("/definitions", middleware.RequirePermission("sla", "read"), config.SLAHandler.ListSLADefinitions)
				slaGrp.POST("/definitions", middleware.RequirePermission("sla", "write"), config.SLAHandler.CreateSLADefinition)
				slaGrp.GET("/definitions/:id", middleware.RequirePermission("sla", "read"), config.SLAHandler.GetSLADefinition)
				slaGrp.PUT("/definitions/:id", middleware.RequirePermission("sla", "write"), config.SLAHandler.UpdateSLADefinition)
				slaGrp.DELETE("/definitions/:id", middleware.RequirePermission("sla", "delete"), config.SLAHandler.DeleteSLADefinition)
				slaGrp.POST("/alert-rules", middleware.RequirePermission("sla", "write"), config.SLAHandler.CreateAlertRule)
				slaGrp.GET("/alert-rules", middleware.RequirePermission("sla", "read"), config.SLAHandler.ListAlertRules)
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
			users := tenant.(*gin.RouterGroup).Group("/users")
			{
				users.GET("", config.CommonHandler.ListUsers)
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
		}

		// ==================== Miscellaneous ====================
		if config.BPMNWorkflowController != nil {
			config.BPMNWorkflowController.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		if config.DashboardHandler != nil {
			dashboard := tenant.(*gin.RouterGroup).Group("/dashboard")
			{
				dashboard.GET("/overview", config.DashboardHandler.GetOverview)
			}
		}
	}
}
