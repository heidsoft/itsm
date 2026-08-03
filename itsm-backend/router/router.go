package router

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
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
	"itsm-backend/handlers/cmdb"
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

func withIncidentIDParam(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			common.Fail(c, common.ParamErrorCode, "invalid request body")
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		var req struct {
			IncidentID int `json:"incidentId"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			common.Fail(c, common.ParamErrorCode, "invalid request body")
			return
		}
		incidentID := req.IncidentID
		if incidentID == 0 {
			common.Fail(c, common.ParamErrorCode, "incidentId is required")
			return
		}

		c.Params = append(c.Params, gin.Param{Key: "id", Value: strconv.Itoa(incidentID)})
		handler(c)
	}
}

func defaultDashboardLayout() gin.H {
	return gin.H{
		"cols":             12,
		"rows":             24,
		"margin":           []int{16, 16},
		"containerPadding": []int{16, 16},
		"rowHeight":        80,
		"isDraggable":      true,
		"isResizable":      true,
	}
}

func defaultDashboardWidgets() []gin.H {
	return []gin.H{
		{
			"id":          "ticket_overview",
			"type":        "metric",
			"title":       "工单总览",
			"description": "工单数量与状态概览",
			"position":    gin.H{"x": 0, "y": 0, "w": 3, "h": 2},
			"config":      gin.H{"showTitle": true, "showBorder": true, "metric": "total", "unit": "个"},
			"dataSource":  "tickets",
			"isVisible":   true,
		},
		{
			"id":          "ticket_trend",
			"type":        "chart",
			"title":       "工单趋势",
			"description": "近期工单创建与解决趋势",
			"position":    gin.H{"x": 3, "y": 0, "w": 6, "h": 4},
			"config":      gin.H{"showTitle": true, "showBorder": true, "chartType": "line", "xAxis": "date", "yAxis": "count"},
			"dataSource":  "ticket_trend",
			"isVisible":   true,
		},
		{
			"id":          "sla_status",
			"type":        "progress",
			"title":       "SLA 达成率",
			"description": "服务级别协议履约情况",
			"position":    gin.H{"x": 9, "y": 0, "w": 3, "h": 2},
			"config":      gin.H{"showTitle": true, "showBorder": true, "metric": "slaCompliance", "unit": "%"},
			"dataSource":  "sla",
			"isVisible":   true,
		},
	}
}

func defaultDashboardConfig() gin.H {
	now := time.Now().Format(time.RFC3339)
	return gin.H{
		"id":          1,
		"name":        "默认仪表盘",
		"description": "系统默认运维视图",
		"isDefault":   true,
		"isPublic":    false,
		"layout":      defaultDashboardLayout(),
		"widgets":     defaultDashboardWidgets(),
		"filters": []gin.H{
			{
				"id":         "time_range",
				"name":       "时间范围",
				"type":       "select",
				"field":      "timeRange",
				"options":    []gin.H{{"label": "最近7天", "value": "7d"}, {"label": "最近30天", "value": "30d"}},
				"isRequired": false,
				"isVisible":  true,
			},
		},
		"permissions": []string{},
		"createdBy":   0,
		"updatedBy":   0,
		"createdAt":   now,
		"updatedAt":   now,
		"shareSettings": gin.H{
			"isShared": false,
		},
	}
}

func defaultDashboardTemplate() gin.H {
	return gin.H{
		"id":            1,
		"name":          "ITSM 运营总览",
		"description":   "适用于服务台、SLA 与工单趋势的默认模板",
		"category":      "operations",
		"tags":          []string{"itsm", "ticket", "sla"},
		"layout":        defaultDashboardLayout(),
		"widgets":       defaultDashboardWidgets(),
		"filters":       []gin.H{},
		"isPublic":      true,
		"downloadCount": 0,
	}
}

func dashboardWidgetByID(widgetID string) gin.H {
	for _, widget := range defaultDashboardWidgets() {
		if widget["id"] == widgetID {
			return widget
		}
	}
	return gin.H{
		"id":         widgetID,
		"type":       "metric",
		"title":      widgetID,
		"position":   gin.H{"x": 0, "y": 0, "w": 3, "h": 2},
		"config":     gin.H{"showTitle": true, "showBorder": true},
		"dataSource": widgetID,
		"isVisible":  true,
	}
}

// RouterConfig 路由配置
type RouterConfig struct {
	JWTSecret string
	Logger    *zap.SugaredLogger
	Client    *ent.Client
	RawDB     *sql.DB

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
	BPMNMonitoringController        *controller.BPMNMonitoringController
	BPMNAIGeneratorController       *controller.BPMNAIGeneratorController

	A2UITicketController *controller.A2UITicketController
	DashboardHandler     *handlers.DashboardHandler

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
	EscalationMatrixController       *controller.EscalationMatrixController
	NotificationPreferenceController *controller.NotificationPreferenceController
	AuditLogController               *controller.AuditLogController
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
	CloudController        *controller.CloudController

	// Domain Handlers
	ServiceCatalogHandler *service_catalog.Handler
	ServiceRequestHandler *service_request.Handler
	CMDBHandler           *cmdb.Handler

	ProblemHandler        *problem.Handler
	ChangeHandler         *change.Handler
	KnowledgeHandler      *knowledge.Handler
	SLAHandler            *sla.Handler
	SLATemplateController *controller.SLATemplateController
	AIHandler             *ai.Handler
	CommonHandler         *domainCommon.Handler
	AuthController        *controller.AuthController
	RoleHandler           *common.RoleHandler

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
	FeishuController      *controller.FeishuController
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
			public.POST("/auth/refresh", config.CommonHandler.RefreshToken)
		}

		// 无需认证的账号自助端点（注册/密码找回/重置）
		if config.AuthController != nil {
			public.POST("/auth/register", config.AuthController.Register)
			public.POST("/auth/forgot-password", config.AuthController.ForgotPassword)
			public.POST("/auth/reset-password", config.AuthController.ResetPassword)
			public.POST("/auth/validate-reset-token", config.AuthController.ValidateResetToken)
		}

		// CSRF token 获取端点（无需认证）
		if config.CSRFEnabled {
			public.GET("/csrf-token", middleware.CSRFTokenEndpoint(middleware.DefaultCSRFConfig()))
		}

		// 系统状态
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "timestamp": time.Now()})
		})
		// /healthz 别名，兼容 K8s / 常见探针路径
		public.GET("/healthz", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "timestamp": time.Now()})
		})
		public.GET("/readyz", func(c *gin.Context) {
			readiness := checkInitializationReadiness(c.Request.Context(), config.RawDB)
			status := 200
			if !readiness.Ready {
				status = 503
			}
			c.JSON(status, readiness)
		})
		public.GET("/version", func(c *gin.Context) {
			c.JSON(200, gin.H{"version": "1.6.4", "build": "release"})
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
	auth := r.Group("/api/v1")
	auth.Use(middleware.AuthMiddleware(config.JWTSecret))
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
				categories.GET("", middleware.RequirePermission("ticket_category", "read"), config.TicketCategoryController.ListCategories)
				categories.POST("", middleware.RequirePermission("ticket_category", "create"), config.TicketCategoryController.CreateCategory)
				categories.GET("/tree", middleware.RequirePermission("ticket_category", "read"), config.TicketCategoryController.GetCategoryTree)
				categories.POST("/import/preview", middleware.RequirePermission("ticket_category", "create"), config.TicketCategoryController.PreviewImport)
				categories.POST("/import", middleware.RequirePermission("ticket_category", "create"), config.TicketCategoryController.ExecuteImport)
				categories.GET("/:id", middleware.RequirePermission("ticket_category", "read"), config.TicketCategoryController.GetCategory)
				categories.PUT("/:id", middleware.RequirePermission("ticket_category", "update"), config.TicketCategoryController.UpdateCategory)
				categories.PUT("/:id/move", middleware.RequirePermission("ticket_category", "update"), config.TicketCategoryController.MoveCategory)
				categories.DELETE("/:id", middleware.RequirePermission("ticket_category", "delete"), config.TicketCategoryController.DeleteCategory)
			}
		}

		if config.TicketTagController != nil {
			tags := tenant.(*gin.RouterGroup).Group("/ticket-tags")
			{
				tags.GET("", middleware.RequirePermission("ticket_tag", "read"), config.TicketTagController.ListTags)
				tags.POST("", middleware.RequirePermission("ticket_tag", "create"), config.TicketTagController.CreateTag)
				tags.GET("/:id", middleware.RequirePermission("ticket_tag", "read"), config.TicketTagController.GetTag)
				tags.PUT("/:id", middleware.RequirePermission("ticket_tag", "update"), config.TicketTagController.UpdateTag)
				tags.DELETE("/:id", middleware.RequirePermission("ticket_tag", "delete"), config.TicketTagController.DeleteTag)
			}
		}

		// ==================== Tickets ====================
		tickets := tenant.(*gin.RouterGroup).Group("/tickets")
		{
			tickets.GET("", middleware.RequirePermission("ticket", "read"), config.TicketController.ListTickets)
			tickets.POST("", middleware.RequirePermission("ticket", "create"), config.TicketController.CreateTicket)

			if config.TicketViewController != nil {
				tickets.GET("/views", middleware.RequirePermission("view", "read"), config.TicketViewController.ListTicketViews)
				tickets.POST("/views", middleware.RequirePermission("view", "create"), config.TicketViewController.CreateTicketView)
				tickets.GET("/views/:id", middleware.RequirePermission("view", "read"), config.TicketViewController.GetTicketView)
				tickets.PUT("/views/:id", middleware.RequirePermission("view", "update"), config.TicketViewController.UpdateTicketView)
				tickets.DELETE("/views/:id", middleware.RequirePermission("view", "delete"), config.TicketViewController.DeleteTicketView)
			}

			tickets.GET("/search", middleware.RequirePermission("ticket", "read"), config.TicketController.SearchTickets)
			tickets.GET("/stats", middleware.RequirePermission("ticket", "read"), config.TicketController.GetTicketStats)
			tickets.GET("/overdue", middleware.RequirePermission("ticket", "read"), config.TicketController.GetOverdueTickets)
			tickets.POST("/export", middleware.RequirePermission("ticket", "export"), config.TicketController.ExportTickets)
			tickets.POST("/batch-delete", middleware.RequirePermission("ticket", "delete"), config.TicketController.BatchDeleteTickets)
			if config.TicketWorkflowController != nil {
				tickets.GET("/cc/my", middleware.RequirePermission("workflow", "read"), config.TicketWorkflowController.ListMyCCRecords)
			}

			// 工单模板
			tickets.GET("/templates", middleware.RequirePermission("template", "read"), config.TicketController.GetTicketTemplates)
			tickets.GET("/templates/categories", middleware.RequirePermission("template", "read"), config.TicketController.GetTicketTemplateCategories)
			tickets.POST("/templates", middleware.RequirePermission("template", "create"), config.TicketController.CreateTicketTemplate)
			tickets.GET("/templates/:id", middleware.RequirePermission("template", "read"), config.TicketController.GetTicketTemplate)
			tickets.PUT("/templates/:id", middleware.RequirePermission("template", "update"), config.TicketController.UpdateTicketTemplate)
			tickets.PATCH("/templates/:id/status", middleware.RequirePermission("template", "update"), config.TicketController.UpdateTicketTemplateStatus)
			tickets.POST("/templates/:id/copy", middleware.RequirePermission("template", "create"), config.TicketController.CopyTicketTemplate)
			tickets.DELETE("/templates/:id", middleware.RequirePermission("template", "delete"), config.TicketController.DeleteTicketTemplate)

			tickets.POST("/:id/escalate", middleware.RequirePermission("ticket", "escalate"), config.TicketController.EscalateTicket)
			tickets.GET("/:id/history", middleware.RequirePermission("ticket", "read"), config.TicketController.GetTicketActivity)
			tickets.GET("/types", middleware.RequirePermission("ticket", "read"), func(c *gin.Context) {
				common.Success(c, gin.H{"types": []gin.H{
					{"id": 1, "name": " Incident", "code": "incident"},
					{"id": 2, "name": "Problem", "code": "problem"},
					{"id": 3, "name": "Change", "code": "change"},
					{"id": 4, "name": "Request", "code": "request"},
				}, "total": 4})
			})
			tickets.GET("/:id", middleware.RequirePermission("ticket", "read"), config.TicketController.GetTicket)
			tickets.PUT("/:id", middleware.RequirePermission("ticket", "update"), config.TicketController.UpdateTicket)
			tickets.PUT("/:id/status", middleware.RequirePermission("ticket", "update"), config.TicketController.UpdateTicketStatus)
			tickets.DELETE("/:id", middleware.RequirePermission("ticket", "delete"), config.TicketController.DeleteTicket)
			tickets.POST("/:id/assign", middleware.RequirePermission("ticket", "assign"), config.TicketController.AssignTicket)
			tickets.POST("/:id/resolve", middleware.RequirePermission("ticket", "update"), config.TicketController.ResolveTicket)
			tickets.POST("/:id/close", middleware.RequirePermission("ticket", "update"), config.TicketController.CloseTicket)

			// 工单SLA信息
			tickets.GET("/:id/sla", middleware.RequirePermission("ticket", "read"), config.TicketController.GetTicketSLAInfo)

			// 子任务管理
			tickets.GET("/:id/subtasks", middleware.RequirePermission("ticket", "read"), config.TicketController.GetSubtasks)
			tickets.POST("/:id/subtasks", middleware.RequirePermission("ticket", "create"), config.TicketController.CreateSubtask)
			tickets.PATCH("/:id/subtasks/:subtask_id", middleware.RequirePermission("ticket", "update"), config.TicketController.UpdateSubtask)
			tickets.DELETE("/:id/subtasks/:subtask_id", middleware.RequirePermission("ticket", "delete"), config.TicketController.DeleteSubtask)

			// 评论
			if config.TicketCommentController != nil {
				tickets.GET("/:id/comments", middleware.RequirePermission("ticket", "read"), config.TicketCommentController.ListTicketComments)
				tickets.POST("/:id/comments", middleware.RequirePermission("ticket", "create"), config.TicketCommentController.CreateTicketComment)
				tickets.PUT("/:id/comments/:comment_id", middleware.RequirePermission("ticket", "update"), config.TicketCommentController.UpdateTicketComment)
				tickets.DELETE("/:id/comments/:comment_id", middleware.RequirePermission("ticket", "delete"), config.TicketCommentController.DeleteTicketComment)
			}

			// 附件
			if config.TicketAttachmentController != nil {
				tickets.GET("/:id/attachments", middleware.RequirePermission("ticket", "read"), config.TicketAttachmentController.ListTicketAttachments)
				tickets.POST("/:id/attachments", middleware.RequirePermission("ticket", "create"), config.TicketAttachmentController.UploadAttachment)
				tickets.DELETE("/:id/attachments/:attachment_id", middleware.RequirePermission("ticket", "delete"), config.TicketAttachmentController.DeleteAttachment)
			}

			// 审批流程管理
			if config.ApprovalController != nil {
				tickets.POST("/approval/submit", middleware.RequirePermission("approval", "create"), config.ApprovalController.SubmitApproval)
				tickets.GET("/approval/records", middleware.RequirePermission("approval", "read"), config.ApprovalController.GetApprovalRecords)

				// 审批工作流CRUD
				approvalWorkflows := tenant.(*gin.RouterGroup).Group("/approval-workflows")
				{
					approvalWorkflows.GET("", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalController.ListWorkflows)
					approvalWorkflows.POST("", middleware.RequirePermission("approval_workflow", "create"), config.ApprovalController.CreateWorkflow)
					approvalWorkflows.GET("/:id", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalController.GetWorkflow)
					approvalWorkflows.PUT("/:id", middleware.RequirePermission("approval_workflow", "update"), config.ApprovalController.UpdateWorkflow)
					approvalWorkflows.POST("/:id/migrate-to-bpmn", middleware.RequirePermission("approval_workflow", "update"), config.ApprovalController.MigrateWorkflowToBPMN)
					approvalWorkflows.PATCH("/:id", middleware.RequirePermission("approval_workflow", "update"), config.ApprovalController.PatchWorkflow)
					approvalWorkflows.DELETE("/:id", middleware.RequirePermission("approval_workflow", "delete"), config.ApprovalController.DeleteWorkflow)
				}
				// 兼容旧路径 /approvals
				approvals := tenant.(*gin.RouterGroup).Group("/approvals")
				{
					approvals.GET("", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalController.ListWorkflows)
					approvals.POST("", middleware.RequirePermission("approval_workflow", "write"), config.ApprovalController.CreateWorkflow)
					approvals.GET("/:id", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalController.GetWorkflow)
					approvals.PUT("/:id", middleware.RequirePermission("approval_workflow", "write"), config.ApprovalController.UpdateWorkflow)
					approvals.PATCH("/:id", middleware.RequirePermission("approval_workflow", "write"), config.ApprovalController.PatchWorkflow)
					approvals.DELETE("/:id", middleware.RequirePermission("approval_workflow", "delete"), config.ApprovalController.DeleteWorkflow)
					approvals.GET("/records", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalController.GetApprovalRecords)
					approvals.POST("/submit", middleware.RequirePermission("approval_workflow", "write"), config.ApprovalController.SubmitApproval)
					// 兼容旧路径：/approval-records 和 /my-approvals
					tenant.GET("/approval-records", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalController.GetApprovalRecords)
					tenant.GET("/my-approvals", middleware.RequirePermission("approval_workflow", "read"), config.ApprovalController.GetApprovalRecords)

				}

			}

			// 工单流转工作流
			if config.TicketWorkflowController != nil {
				tickets.POST("/workflow/accept", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowController.AcceptTicket)
				tickets.POST("/workflow/reject", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowController.RejectTicket)
				tickets.POST("/workflow/withdraw", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowController.WithdrawTicket)
				tickets.POST("/workflow/forward", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowController.ForwardTicket)
				tickets.POST("/workflow/cc", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowController.CCTicket)
				tickets.POST("/workflow/approve", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowController.ApproveTicket)
				tickets.POST("/workflow/resolve", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowController.ResolveTicket)
				tickets.POST("/workflow/close", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowController.CloseTicket)
				tickets.POST("/workflow/reopen", middleware.RequirePermission("workflow", "update"), config.TicketWorkflowController.ReopenTicket)
				tickets.GET("/:id/cc", middleware.RequirePermission("workflow", "read"), config.TicketWorkflowController.ListTicketCCRecords)
				tickets.GET("/:id/workflow/state", middleware.RequirePermission("workflow", "read"), config.TicketWorkflowController.GetTicketWorkflowState)
				tickets.GET("/:id/workflow-history", middleware.RequirePermission("workflow", "read"), config.TicketWorkflowController.GetTicketWorkflowHistory)
				tickets.GET("/:id/workflow_records", middleware.RequirePermission("workflow", "read"), config.TicketWorkflowController.GetTicketWorkflowHistory)
			}

			// 工单自动化规则
			if config.TicketAutomationRuleController != nil {
				tickets.GET("/automation-rules", middleware.RequirePermission("automation_rule", "read"), config.TicketAutomationRuleController.ListAutomationRules)
				tickets.POST("/automation-rules", middleware.RequirePermission("automation_rule", "create"), config.TicketAutomationRuleController.CreateAutomationRule)
				tickets.GET("/automation-rules/:id", middleware.RequirePermission("automation_rule", "read"), config.TicketAutomationRuleController.GetAutomationRule)
				tickets.PUT("/automation-rules/:id", middleware.RequirePermission("automation_rule", "update"), config.TicketAutomationRuleController.UpdateAutomationRule)
				tickets.DELETE("/automation-rules/:id", middleware.RequirePermission("automation_rule", "delete"), config.TicketAutomationRuleController.DeleteAutomationRule)
				tickets.POST("/automation-rules/:id/test", middleware.RequirePermission("automation_rule", "update"), config.TicketAutomationRuleController.TestAutomationRule)
			}

			// 工单分配规则
			if config.TicketAssignmentSmartController != nil {
				tickets.POST("/:id/auto-assign", middleware.RequirePermission("assignment_rule", "update"), config.TicketAssignmentSmartController.AutoAssign)
				tickets.GET("/assign-recommendations/:id", middleware.RequirePermission("assignment_rule", "read"), config.TicketAssignmentSmartController.GetAssignRecommendations)
				tickets.GET("/assignment-rules", middleware.RequirePermission("assignment_rule", "read"), config.TicketAssignmentSmartController.ListAssignmentRules)
				tickets.POST("/assignment-rules", middleware.RequirePermission("assignment_rule", "create"), config.TicketAssignmentSmartController.CreateAssignmentRule)
				tickets.POST("/assignment-rules/test", middleware.RequirePermission("assignment_rule", "update"), config.TicketAssignmentSmartController.TestAssignmentRule)
				tickets.GET("/assignment-rules/:id", middleware.RequirePermission("assignment_rule", "read"), config.TicketAssignmentSmartController.GetAssignmentRule)
				tickets.PUT("/assignment-rules/:id", middleware.RequirePermission("assignment_rule", "update"), config.TicketAssignmentSmartController.UpdateAssignmentRule)
				tickets.DELETE("/assignment-rules/:id", middleware.RequirePermission("assignment_rule", "delete"), config.TicketAssignmentSmartController.DeleteAssignmentRule)
			}

			// 工单评分
			if config.TicketRatingController != nil {
				tickets.POST("/:id/rating", middleware.RequirePermission("ticket", "update"), config.TicketRatingController.SubmitRating)
				tickets.GET("/:id/rating", middleware.RequirePermission("ticket", "read"), config.TicketRatingController.GetRating)
				tickets.GET("/rating-stats", middleware.RequirePermission("ticket", "read"), config.TicketRatingController.GetRatingStats)
			}

			if config.TicketTagController != nil {
				tickets.POST("/:id/tags", middleware.RequirePermission("ticket_tag", "create"), config.TicketTagController.AssignTagsToTicket)
				tickets.DELETE("/:id/tags", middleware.RequirePermission("ticket_tag", "delete"), config.TicketTagController.RemoveTagsFromTicket)
			}

			// 工单分析与预测 (Alias for frontend compatibility)
			if config.AIHandler != nil {
				tickets.POST("/analytics/deep", middleware.RequirePermission("report", "read"), config.AIHandler.GetDeepAnalytics)
				tickets.POST("/prediction/trend", middleware.RequirePermission("report", "read"), config.AIHandler.GetTrendPrediction)
			}

			if config.AnalyticsController != nil {
				tickets.POST("/analytics/export", middleware.RequirePermission("report", "read"), config.AnalyticsController.ExportAnalytics)
				// B8: GET /api/v1/analytics/tickets - 工单分析概览
				// 挂在 tenant 顶层 group，路径 = /api/v1/analytics/tickets
			}

			if config.PredictionController != nil {
				tickets.POST("/prediction/export", middleware.RequirePermission("report", "read"), config.PredictionController.ExportPredictionReport)
			}
		}

		// ==================== 顶层分析别名 ====================
		if config.AnalyticsController != nil {
			tenant.(*gin.RouterGroup).GET("/analytics/tickets", middleware.RequirePermission("report", "read"), config.AnalyticsController.GetTicketAnalytics)
		}

		// B9: AI 工单总结 /ai/tickets/{id}/summary
		aiGroup := tenant.(*gin.RouterGroup).Group("/ai")
		{
			aiGroup.GET("/tickets/:id/summary", middleware.RequirePermission("ticket", "read"), config.AIHandler.SummarizeTicket)
		}

		// ==================== System Configs ====================
		if config.SystemConfigController != nil {
			sysConfigs := tenant.(*gin.RouterGroup).Group("/system-configs")
			{
				// 配置管理
				sysConfigs.GET("", middleware.RequirePermission("config", "read"), config.SystemConfigController.ListConfigs)
				sysConfigs.GET("/init", middleware.RequirePermission("config", "read"), config.SystemConfigController.InitDefaultConfigs)
				sysConfigs.GET("/:id", middleware.RequirePermission("config", "read"), config.SystemConfigController.GetConfig)
				sysConfigs.GET("/key/:key", middleware.RequirePermission("config", "read"), config.SystemConfigController.GetConfigByKey)
				sysConfigs.PUT("/:id", middleware.RequirePermission("config", "update"), config.SystemConfigController.UpdateConfig)
				sysConfigs.PUT("/batch", middleware.RequirePermission("config", "update"), config.SystemConfigController.BatchUpdateConfigs)

				// 系统状态
				sysConfigs.GET("/status", middleware.RequirePermission("config", "read"), func(c *gin.Context) {
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

			// P1-01 别名：/system/config → 系统状态（前端默认 fetch 路径）
			sysRoot := tenant.(*gin.RouterGroup).Group("/system")
			{
				// 兼容旧路径：/configs → /system-configs
				configs := tenant.(*gin.RouterGroup).Group("/configs")
				{
					configs.GET("", middleware.RequirePermission("config", "read"), config.SystemConfigController.ListConfigs)
					configs.GET("/init", middleware.RequirePermission("config", "read"), config.SystemConfigController.InitDefaultConfigs)
					configs.GET("/:id", middleware.RequirePermission("config", "read"), config.SystemConfigController.GetConfig)
					configs.GET("/key/:key", middleware.RequirePermission("config", "read"), config.SystemConfigController.GetConfigByKey)
					configs.PUT("/:id", middleware.RequirePermission("config", "update"), config.SystemConfigController.UpdateConfig)
					configs.PUT("/batch", middleware.RequirePermission("config", "update"), config.SystemConfigController.BatchUpdateConfigs)
					configs.GET("/status", middleware.RequirePermission("config", "read"), func(c *gin.Context) {
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

				sysRoot.GET("/config", middleware.RequirePermission("config", "read"), func(c *gin.Context) {
					c.JSON(200, gin.H{
						"status":    "ok",
						"version":   "1.6.4",
						"timestamp": time.Now(),
					})
				})
			}
		}

		// ==================== Approval Chains ====================
		if config.ApprovalChainController != nil {
			approvalChains := tenant.(*gin.RouterGroup).Group("/approval-chains")
			{
				approvalChains.GET("", middleware.RequirePermission("approval", "read"), config.ApprovalChainController.ListChains)
				approvalChains.GET("/stats", middleware.RequirePermission("approval", "read"), config.ApprovalChainController.GetStats)
				approvalChains.POST("", middleware.RequirePermission("approval", "create"), config.ApprovalChainController.CreateChain)
				approvalChains.GET("/:id", middleware.RequirePermission("approval", "read"), config.ApprovalChainController.GetChain)
				approvalChains.PUT("/:id", middleware.RequirePermission("approval", "update"), config.ApprovalChainController.UpdateChain)
				approvalChains.DELETE("/:id", middleware.RequirePermission("approval", "delete"), config.ApprovalChainController.DeleteChain)
			}
			// ==================== Escalation Matrix ====================
			if config.EscalationMatrixController != nil {
				config.EscalationMatrixController.RegisterRoutes(tenant.(*gin.RouterGroup))
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
				inc.POST("/:id/reopen", middleware.RequirePermission("incident", "write"), config.IncidentController.ReopenIncident)
				inc.POST("/:id/assign", middleware.RequirePermission("incident", "assign"), config.IncidentController.AssignIncident)
				inc.POST("/:id/major-incident", middleware.RequirePermission("incident", "write"), config.IncidentController.EscalateMajorIncident)
				inc.POST("/:id/convert-to-problem", middleware.RequirePermission("incident", "write"), config.IncidentController.ConvertToProblem)
				inc.GET("/:id/impact", middleware.RequirePermission("incident", "read"), config.IncidentController.AnalyzeIncidentImpact)

				// 关联数据
				inc.GET("/:id/events", middleware.RequirePermission("incident", "read"), config.IncidentController.GetIncidentEvents)
				inc.POST("/events", middleware.RequirePermission("incident", "write"), config.IncidentController.CreateIncidentEvent)
				inc.GET("/:id/alerts", middleware.RequirePermission("incident", "read"), config.IncidentController.GetIncidentAlerts)
				inc.GET("/:id/metrics", middleware.RequirePermission("incident", "read"), config.IncidentController.GetIncidentMetrics)

				// 根因分析
				inc.POST("/root-cause", middleware.RequirePermission("incident", "write"), withIncidentIDParam(config.IncidentController.UpdateRootCause))
				inc.GET("/:id/root-cause", middleware.RequirePermission("incident", "read"), config.IncidentController.GetRootCause)
				inc.POST("/:id/root-cause", middleware.RequirePermission("incident", "write"), config.IncidentController.UpdateRootCause)
				inc.PUT("/:id/root-cause", middleware.RequirePermission("incident", "write"), config.IncidentController.UpdateRootCause)

				// 影响评估
				inc.POST("/impact-assessment", middleware.RequirePermission("incident", "write"), withIncidentIDParam(config.IncidentController.UpdateImpactAssessment))
				inc.GET("/:id/impact-assessment", middleware.RequirePermission("incident", "read"), config.IncidentController.GetImpactAssessment)
				inc.PUT("/:id/impact-assessment", middleware.RequirePermission("incident", "write"), config.IncidentController.UpdateImpactAssessment)

				// 事件分类
				inc.POST("/classification", middleware.RequirePermission("incident", "write"), withIncidentIDParam(config.IncidentController.UpdateClassification))
				inc.GET("/:id/classification", middleware.RequirePermission("incident", "read"), config.IncidentController.GetClassification)
				inc.PUT("/:id/classification", middleware.RequirePermission("incident", "write"), config.IncidentController.UpdateClassification)
				inc.PUT("/:id/status", middleware.RequirePermission("incident", "write"), config.IncidentController.UpdateIncident)

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
		if config.IncidentController != nil && config.CMDBController != nil {
			tenant.GET("/incidents/configuration-items", middleware.RequirePermission("cmdb", "read"), config.CMDBController.ListCIs)
		}

		// ==================== Service Catalog & Requests (DDD) ====================
		if config.ServiceCatalogHandler != nil {
			tenant.(*gin.RouterGroup).GET("/service-catalog", middleware.RequirePermission("service_catalog", "read"), config.ServiceCatalogHandler.List)

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
				scServices.GET("", middleware.RequirePermission("service_catalog", "read"), config.ServiceCatalogHandler.List)
				scServices.GET("/:id", middleware.RequirePermission("service_catalog", "read"), config.ServiceCatalogHandler.Get)
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
				sr.PUT("/:id/status", middleware.RequirePermission("service_request", "write"), config.ServiceRequestHandler.UpdateStatus)
				sr.DELETE("/:id", middleware.RequirePermission("service_request", "delete"), config.ServiceRequestHandler.Delete)
				sr.POST("/:id/approval", middleware.RequirePermission("service_request", "write"), config.ServiceRequestHandler.ApplyApproval)
				sr.POST("/:id/approvals", middleware.RequirePermission("service_request", "write"), config.ServiceRequestHandler.ApplyApproval)
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
				problems.POST("/:id/investigate", middleware.RequirePermission("problem", "write"), config.ProblemHandler.InvestigateProblem)
				problems.PUT("/:id/root-cause", middleware.RequirePermission("problem", "write"), config.ProblemHandler.UpdateRootCause)
				problems.PUT("/:id/solution", middleware.RequirePermission("problem", "write"), config.ProblemHandler.UpdateSolution)
				problems.POST("/:id/close", middleware.RequirePermission("problem", "write"), config.ProblemHandler.CloseProblem)
				// 关联管理
				problems.GET("/:id/associations", middleware.RequirePermission("problem", "read"), config.ProblemHandler.GetAssociations)
				problems.POST("/:id/associations", middleware.RequirePermission("problem", "write"), config.ProblemHandler.AddAssociation)
				problems.DELETE("/:id/associations", middleware.RequirePermission("problem", "write"), config.ProblemHandler.RemoveAssociation)
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
				// 状态转换：approve/reject 需要独立审批权限，rollback 需要独立回滚权限（H-15 修复：禁止 write 权限泛化为审批/回滚）
				changes.POST("/:id/approve", middleware.RequirePermission("change", "approve"), config.ChangeHandler.TransitionStatus)
				changes.POST("/:id/reject", middleware.RequirePermission("change", "approve"), config.ChangeHandler.TransitionStatus)
				changes.POST("/:id/start", middleware.RequirePermission("change", "write"), config.ChangeHandler.TransitionStatus)
				changes.POST("/:id/complete", middleware.RequirePermission("change", "write"), config.ChangeHandler.TransitionStatus)
				changes.POST("/:id/rollback", middleware.RequirePermission("change", "rollback"), config.ChangeHandler.TransitionStatus)
				changes.POST("/:id/cancel", middleware.RequirePermission("change", "write"), config.ChangeHandler.TransitionStatus)
				// 审批
				changes.GET("/:id/approvals", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetApprovals)
				changes.POST("/:id/approvals", middleware.RequirePermission("change", "write"), config.ChangeHandler.SubmitApproval)
				changes.GET("/:id/approval-summary", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetApprovalSummary)
				// 风险评估（同时支持 /risk 和 /risk-assessment 两个路径）
				changes.GET("/:id/risk-assessment", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetRiskAssessment)
				changes.GET("/:id/risk", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetRiskAssessment)
				changes.PUT("/:id/risk", middleware.RequirePermission("change", "write"), config.ChangeHandler.UpdateRisk)
				changes.GET("/:id/cmdb-impact", middleware.RequirePermission("change", "read"), config.ChangeHandler.GetCMDBImpactSummary)
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
				releases.POST("/:id/approve", middleware.RequirePermission("release", "approve"), config.ReleaseController.ApproveRelease)
				releases.POST("/:id/reject", middleware.RequirePermission("release", "approve"), config.ReleaseController.RejectRelease)
				releases.POST("/:id/rollback", middleware.RequirePermission("release", "rollback"), config.ReleaseController.RollbackRelease)
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
		// ==================== CMDB ====================
		if config.CMDBController != nil {
			SetupCMDBRoutes(tenant.(*gin.RouterGroup), config.CMDBController, config)
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
					articles.POST("/:id/publish", middleware.RequirePermission("knowledge", "write"), config.KnowledgeHandler.PublishArticle)
					articles.POST("/:id/unpublish", middleware.RequirePermission("knowledge", "write"), config.KnowledgeHandler.UnpublishArticle)

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
			}
		}

		if config.SLAHandler != nil {
			slaGrp := tenant.(*gin.RouterGroup).Group("/sla")
			{
				slaGrp.GET("", middleware.RequirePermission("sla", "read"), config.SLAHandler.ListSLADefinitions)
				// SLA Definitions
				slaGrp.GET("/stats", middleware.RequirePermission("sla", "read"), config.SLAHandler.GetSLAStats)
				slaGrp.GET("/definitions", middleware.RequirePermission("sla", "read"), config.SLAHandler.ListSLADefinitions)
				slaGrp.POST("/definitions", middleware.RequirePermission("sla", "write"), config.SLAHandler.CreateSLADefinition)
				// 兼容旧路径：/sla/policies → /sla/definitions
				slaGrp.GET("/policies", middleware.RequirePermission("sla", "read"), config.SLAHandler.ListSLADefinitions)
				slaGrp.POST("/policies", middleware.RequirePermission("sla", "write"), config.SLAHandler.CreateSLADefinition)
				slaGrp.GET("/policies/:id", middleware.RequirePermission("sla", "read"), config.SLAHandler.GetSLADefinition)
				slaGrp.PUT("/policies/:id", middleware.RequirePermission("sla", "write"), config.SLAHandler.UpdateSLADefinition)
				slaGrp.DELETE("/policies/:id", middleware.RequirePermission("sla", "delete"), config.SLAHandler.DeleteSLADefinition)

				// 兼容旧路径：/sla/monitor → /sla/monitoring
				slaGrp.POST("/monitor", middleware.RequirePermission("sla", "read"), config.SLAHandler.GetSLAMonitoring)

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

			// SLA 模板（开箱即用预置模板）
			if config.SLATemplateController != nil {
				config.SLATemplateController.RegisterRoutes(tenant.(*gin.RouterGroup))
			}
		}

		// ==================== AI & Analytics (DDD) ====================
		if config.AIHandler != nil {
			aiGrp := tenant.(*gin.RouterGroup).Group("/ai")
			{
				aiGrp.POST("/chat", middleware.RequirePermission("ai", "read"), config.AIHandler.Chat)
				aiGrp.POST("/chat/stream", middleware.RequirePermission("ai", "read"), config.AIHandler.ChatStream)
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
				agentGrp.GET("/tools/:id", middleware.RequirePermission("ai", "read"), config.AIHandler.GetToolInvocation)
				agentGrp.POST("/tools/:id/approve", middleware.RequirePermission("ai", "write"), config.AIHandler.ApproveTool)
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
				if config.AuthController != nil {
					authGrp.POST("/switch-tenant", middleware.AuthMiddleware(config.JWTSecret), config.AuthController.SwitchTenant)
				}
			}

			// User Menu (no permission required, will be filtered by role)
			if config.MenuController != nil {
				authGrp.GET("/menus", middleware.AuthMiddleware(config.JWTSecret), config.MenuController.GetUserMenus)
			}

			// ==================== Audit Logs ====================
			SetupAuditLogRoutes(tenant.(*gin.RouterGroup), config.AuditLogController, config.CommonHandler)

			// Users
			if config.UserController != nil {
				users := tenant.(*gin.RouterGroup).Group("/users")
				{
					users.GET("", middleware.RequirePermission("user", "read"), config.UserController.ListUsers)
					users.POST("", middleware.RequirePermission("user", "write"), config.UserController.CreateUser)
					users.GET("/profile", middleware.AuthMiddleware(config.JWTSecret), config.CommonHandler.GetMe) // 获取当前用户信息（需认证）
					users.GET("/me", middleware.AuthMiddleware(config.JWTSecret), config.CommonHandler.GetMe)      // alias of /profile
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
				org.GET("/departments/tree", middleware.RequirePermission("department", "read"), config.CommonHandler.GetDepartmentTree)
				org.GET("/departments/:id", middleware.RequirePermission("department", "read"), config.CommonHandler.GetDepartment)
				org.POST("/departments", middleware.RequirePermission("department", "create"), config.CommonHandler.CreateDepartment)
				org.PUT("/departments/:id", middleware.RequirePermission("department", "update"), config.CommonHandler.UpdateDepartment)
				org.DELETE("/departments/:id", middleware.RequirePermission("department", "delete"), config.CommonHandler.DeleteDepartment)
				org.GET("/teams", middleware.RequirePermission("team", "read"), config.CommonHandler.ListTeams)
				org.GET("/teams/:id", middleware.RequirePermission("team", "read"), config.CommonHandler.GetTeam)
				org.POST("/teams", middleware.RequirePermission("team", "create"), config.CommonHandler.CreateTeam)
				org.PUT("/teams/:id", middleware.RequirePermission("team", "update"), config.CommonHandler.UpdateTeam)
				org.DELETE("/teams/:id", middleware.RequirePermission("team", "delete"), config.CommonHandler.DeleteTeam)
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
					services.GET("/catalogs", middleware.RequirePermission("service_catalog", "read"), config.ServiceController.GetServiceCatalogs)
					services.POST("/catalogs", middleware.RequirePermission("service_catalog", "write"), config.ServiceController.CreateServiceCatalog)
					services.GET("/catalogs/:id", middleware.RequirePermission("service_catalog", "read"), config.ServiceController.GetServiceCatalogByID)
					services.PUT("/catalogs/:id", middleware.RequirePermission("service_catalog", "write"), config.ServiceController.UpdateServiceCatalog)
					services.DELETE("/catalogs/:id", middleware.RequirePermission("service_catalog", "delete"), config.ServiceController.DeleteServiceCatalog)
					services.GET("/requests", middleware.RequirePermission("service_request", "read"), config.ServiceController.GetUserServiceRequests)
					services.POST("/requests", middleware.RequirePermission("service_request", "write"), config.ServiceController.CreateServiceRequest)
					services.GET("/requests/:id", middleware.RequirePermission("service_request", "read"), config.ServiceController.GetServiceRequestByID)
					services.PUT("/requests/:id/status", middleware.RequirePermission("service_request", "write"), config.ServiceController.UpdateServiceRequestStatus)
					services.POST("/requests/approval", middleware.RequirePermission("service_request", "write"), config.ServiceController.ApplyServiceRequestApproval)
					services.GET("/requests/approvals", middleware.RequirePermission("service_request", "read"), config.ServiceController.GetServiceRequestApprovals)
					services.GET("/requests/approvals/pending", middleware.RequirePermission("service_request", "read"), config.ServiceController.GetPendingServiceRequestApprovals)
				}
			}

			// Ticket Dependencies
			if config.TicketDependencyController != nil {
				tickets.GET("/:id/dependencies", middleware.RequirePermission("ticket", "read"), config.TicketDependencyController.AnalyzeDependencyImpact)
			}

			// Ticket Notifications
			if config.TicketNotificationController != nil {
				tickets.GET("/:id/notifications", middleware.RequirePermission("notification", "read"), config.TicketNotificationController.ListTicketNotifications)
				tickets.POST("/:id/notifications", middleware.RequirePermission("notification", "create"), config.TicketNotificationController.SendTicketNotification)
			}

			// System
			sys := tenant.(*gin.RouterGroup).Group("/system")
			{
				sys.GET("/tags", middleware.RequirePermission("tag", "read"), config.CommonHandler.ListTags)
				sys.GET("/audit-logs", middleware.RequirePermission("audit_log", "read"), config.CommonHandler.GetAuditLogs)
			}

			// B7/Bug10: 顶层路由别名（兼容前端 /api/v1/{departments,teams,tags,admin/tenants} 旧路径）
			// 实际业务仍在 /org/* 与 /system/* 与 /tenants，旧路径保持只读 GET，避免破坏现有调用方
			{
				tenant.GET("/departments", middleware.RequirePermission("department", "read"), config.CommonHandler.ListDepartments)
				tenant.GET("/departments/tree", middleware.RequirePermission("department", "read"), config.CommonHandler.GetDepartmentTree)
				tenant.GET("/teams", middleware.RequirePermission("team", "read"), config.CommonHandler.ListTeams)
				tenant.GET("/tags", middleware.RequirePermission("tag", "read"), config.CommonHandler.ListTags)
			}
			if config.TenantController != nil {
				admin := tenant.(*gin.RouterGroup).Group("/admin")
				{
					admin.GET("/tenants", middleware.RequirePermission("tenant", "read"), config.TenantController.ListTenants)
				}
			}

			// Role & Permission Controllers (database-backed with tenant isolation)
			if config.RoleController != nil {
				roles := tenant.(*gin.RouterGroup).Group("/roles")
				{
					roles.GET("", middleware.RequirePermission("role", "read"), config.RoleController.ListRoles)
					roles.POST("", middleware.RequirePermission("role", "create"), config.RoleController.CreateRole)
					roles.GET("/:id", middleware.RequirePermission("role", "read"), config.RoleController.GetRole)
					roles.PUT("/:id", middleware.RequirePermission("role", "update"), config.RoleController.UpdateRole)
					roles.DELETE("/:id", middleware.RequirePermission("role", "delete"), config.RoleController.DeleteRole)
					roles.POST("/:id/permissions", middleware.RequirePermission("role", "update"), config.RoleController.AssignPermissions)
				}
			}

			if config.PermissionController != nil {
				permissions := tenant.(*gin.RouterGroup).Group("/permissions")
				{
					permissions.GET("", middleware.RequirePermission("permission", "read"), config.PermissionController.ListPermissions)
					permissions.POST("", middleware.RequirePermission("permission", "create"), config.PermissionController.CreatePermission)
					permissions.POST("/init", middleware.RequirePermission("permission", "create"), config.PermissionController.InitDefaultPermissions)
				}
			}

			// Menu Controllers (database-backed with tenant isolation)
			if config.MenuController != nil {
				menus := tenant.(*gin.RouterGroup).Group("/menus")
				{
					menus.GET("", middleware.RequirePermission("menu", "read"), config.MenuController.ListMenus)
					menus.POST("", middleware.RequirePermission("menu", "create"), config.MenuController.CreateMenu)
					menus.GET("/:id", middleware.RequirePermission("menu", "read"), config.MenuController.GetMenu)
					menus.PUT("/:id", middleware.RequirePermission("menu", "update"), config.MenuController.UpdateMenu)
					menus.DELETE("/:id", middleware.RequirePermission("menu", "delete"), config.MenuController.DeleteMenu)
					menus.POST("/init", middleware.RequirePermission("menu", "create"), config.MenuController.InitDefaultMenus)
				}
			}

			// Tenant Management (admin only)
			if config.TenantController != nil {
				tenants := tenant.(*gin.RouterGroup).Group("/tenants")
				{
					tenants.GET("", middleware.RequirePermission("tenant", "read"), config.TenantController.ListTenants)
					tenants.POST("", middleware.RequirePermission("tenant", "create"), config.TenantController.CreateTenant)
					tenants.GET("/:id", middleware.RequirePermission("tenant", "read"), config.TenantController.GetTenant)
					tenants.PUT("/:id", middleware.RequirePermission("tenant", "update"), config.TenantController.UpdateTenant)
					tenants.DELETE("/:id", middleware.RequirePermission("tenant", "delete"), config.TenantController.DeleteTenant)
					tenants.PUT("/:id/status", middleware.RequirePermission("tenant", "update"), config.TenantController.UpdateTenantStatus)
				}
			}

			// Notification Preferences
			config.Logger.Info("NotificationPreferenceController check:", zap.Any("controller", config.NotificationPreferenceController))
			if config.NotificationPreferenceController != nil {
				config.Logger.Info("Registering notification-preferences routes")
				notifPrefs := tenant.(*gin.RouterGroup).Group("/notification-preferences")
				{
					notifPrefs.GET("", middleware.RequirePermission("notification", "read"), config.NotificationPreferenceController.ListPreferences)
					notifPrefs.GET("/event-types", middleware.RequirePermission("notification", "read"), config.NotificationPreferenceController.ListEventTypes)
					notifPrefs.GET("/:event_type", middleware.RequirePermission("notification", "read"), config.NotificationPreferenceController.GetPreference)
					notifPrefs.POST("", middleware.RequirePermission("notification", "create"), config.NotificationPreferenceController.CreateOrUpdatePreference)
					notifPrefs.PUT("", middleware.RequirePermission("notification", "update"), config.NotificationPreferenceController.BulkUpdatePreferences)
					notifPrefs.DELETE("/:event_type", middleware.RequirePermission("notification", "delete"), config.NotificationPreferenceController.DeletePreference)
					notifPrefs.POST("/reset", middleware.RequirePermission("notification", "update"), config.NotificationPreferenceController.ResetPreferences)
					notifPrefs.POST("/init", middleware.RequirePermission("notification", "create"), config.NotificationPreferenceController.InitializeDefaultPreferences)
				}
			}

			// Notifications
			if config.NotificationController != nil {
				notifications := tenant.(*gin.RouterGroup).Group("/notifications")
				{
					notifications.GET("", middleware.RequirePermission("notification", "read"), config.NotificationController.GetNotifications)
					notifications.GET("/unread-count", middleware.RequirePermission("notification", "read"), config.NotificationController.GetUnreadCount)
					notifications.PUT("/:id/read", middleware.RequirePermission("notification", "update"), config.NotificationController.MarkNotificationRead)
					notifications.PUT("/read-all", middleware.RequirePermission("notification", "update"), config.NotificationController.MarkAllNotificationsRead)
					notifications.DELETE("/:id", middleware.RequirePermission("notification", "delete"), config.NotificationController.DeleteNotification)
					notifications.POST("", middleware.RequirePermission("notification", "create"), config.NotificationController.CreateNotification)
				}
			}
		}

		// ==================== Problem Investigation ====================
		if config.ProblemInvestigationController != nil {
			problemInvestigation := tenant.(*gin.RouterGroup).Group("/problem-investigation")
			{
				// 问题调查管理
				problemInvestigation.POST("/investigations", middleware.RequirePermission("investigation", "create"), config.ProblemInvestigationController.CreateProblemInvestigation)
				problemInvestigation.GET("/investigations/:id", middleware.RequirePermission("investigation", "read"), config.ProblemInvestigationController.GetProblemInvestigation)
				problemInvestigation.PUT("/investigations/:id", middleware.RequirePermission("investigation", "update"), config.ProblemInvestigationController.UpdateProblemInvestigation)

				// 调查步骤管理
				problemInvestigation.POST("/steps", middleware.RequirePermission("step", "create"), config.ProblemInvestigationController.CreateInvestigationStep)
				problemInvestigation.PUT("/steps/:id", middleware.RequirePermission("step", "update"), config.ProblemInvestigationController.UpdateInvestigationStep)
				problemInvestigation.GET("/investigations/:investigation_id/steps", middleware.RequirePermission("investigation", "read"), config.ProblemInvestigationController.GetInvestigationSteps)

				// 根本原因分析
				problemInvestigation.POST("/root-cause-analysis", middleware.RequirePermission("root_cause", "create"), config.ProblemInvestigationController.CreateRootCauseAnalysis)

				// 解决方案管理
				problemInvestigation.POST("/solutions", middleware.RequirePermission("solution", "create"), config.ProblemInvestigationController.CreateProblemSolution)
				problemInvestigation.GET("/problems/:id/solutions", middleware.RequirePermission("problem", "read"), config.ProblemInvestigationController.GetProblemSolutions)

				// 问题调查摘要
				problemInvestigation.GET("/problems/:id/summary", middleware.RequirePermission("problem", "read"), config.ProblemInvestigationController.GetProblemInvestigationSummary)
			}
		}

		// ==================== BPMN Workflow ====================
		if config.BPMNWorkflowController != nil {
			config.BPMNWorkflowController.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// 简化路由：/workflow/* -> /bpmn/*
		workflow := tenant.(*gin.RouterGroup).Group("/workflow")
		{
			workflow.GET("/instances", middleware.RequirePermission("process_instance", "read"), config.BPMNWorkflowController.ListProcessInstances)
			workflow.GET("/instances/:id", middleware.RequirePermission("process_instance", "read"), config.BPMNWorkflowController.GetProcessInstance)
			workflow.POST("/instances", middleware.RequirePermission("process_instance", "create"), config.BPMNWorkflowController.StartProcess)
			workflow.PUT("/instances/:id/terminate", middleware.RequirePermission("process_instance", "update"), config.BPMNWorkflowController.TerminateProcess)
			workflow.PUT("/instances/:id/suspend", middleware.RequirePermission("process_instance", "update"), config.BPMNWorkflowController.SuspendProcess)
			workflow.PUT("/instances/:id/resume", middleware.RequirePermission("process_instance", "update"), config.BPMNWorkflowController.ResumeProcess)
			// 任务
			workflow.GET("/tasks", middleware.RequirePermission("task", "read"), config.BPMNWorkflowController.ListUserTasks)
			workflow.PUT("/tasks/:id/complete", middleware.RequirePermission("task", "update"), config.BPMNWorkflowController.CompleteTask)
			workflow.POST("/tasks/:id/claim", middleware.RequirePermission("task", "update"), config.BPMNWorkflowController.ClaimTask)
		}

		// BPMN Process Trigger Controller (统一流程触发接口)
		if config.BPMNProcessTriggerController != nil {
			config.BPMNProcessTriggerController.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// BPMN Dashboard Controller (监控仪表盘)
		if config.BPMNDashboardController != nil {
			config.BPMNDashboardController.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// ==================== BPMN Monitoring ====================
		// BPMN AI Generator (AI驱动的流程生成)
		if config.BPMNAIGeneratorController != nil {
			config.BPMNAIGeneratorController.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		if config.BPMNMonitoringController != nil {
			config.BPMNMonitoringController.RegisterRoutes(tenant.(*gin.RouterGroup))
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
				conns.GET("", middleware.RequirePermission("connector", "read"), config.ConnectorController.ListMarket)
				conns.GET("/configs", middleware.RequirePermission("connector", "read"), config.ConnectorController.ListConfigs)
				conns.GET("/lifecycle", middleware.RequirePermission("connector", "read"), config.ConnectorController.Lifecycle)
				conns.POST("/configs", middleware.RequirePermission("connector", "write"), config.ConnectorController.Provision)
				conns.DELETE("/configs/:name", middleware.RequirePermission("connector", "write"), config.ConnectorController.Revoke)
				conns.POST("/:name/send", middleware.RequirePermission("connector", "write"), config.ConnectorController.Send)
				conns.POST("/:name/test", middleware.RequirePermission("connector", "write"), config.ConnectorController.Test)
				conns.GET("/health", middleware.RequirePermission("connector", "read"), config.ConnectorController.Health)
				// 飞书事件回调（独立签名校验）
				conns.POST("/feishu/callback", config.ConnectorController.FeishuCallback)
			}
		}

		if config.DashboardHandler != nil {
			dashboard := tenant.(*gin.RouterGroup).Group("/dashboard")
			{
				// B5: 别名，/api/v1/dashboard 直接返回 overview 数据（前端默认调用）
				dashboard.GET("", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetOverview)
				dashboard.GET("/overview", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetOverview)
				dashboard.GET("/config", middleware.RequirePermission("report", "read"), func(c *gin.Context) {
					common.Success(c, defaultDashboardConfig())
				})
				dashboard.POST("/config", middleware.RequirePermission("report", "update"), func(c *gin.Context) {
					common.Success(c, gin.H{"success": true})
				})
				dashboard.GET("/layout", middleware.RequirePermission("report", "read"), func(c *gin.Context) {
					common.Success(c, defaultDashboardLayout())
				})
				dashboard.POST("/layout", middleware.RequirePermission("report", "update"), func(c *gin.Context) {
					common.Success(c, gin.H{"success": true})
				})
				dashboard.GET("/widgets/available", middleware.RequirePermission("report", "read"), func(c *gin.Context) {
					common.Success(c, defaultDashboardWidgets())
				})
				dashboard.POST("/widgets", middleware.RequirePermission("widget", "create"), func(c *gin.Context) {
					widget := dashboardWidgetByID("custom_widget")
					var payload map[string]interface{}
					if err := c.ShouldBindJSON(&payload); err == nil {
						for key, value := range payload {
							widget[key] = value
						}
					}
					if widget["id"] == nil || widget["id"] == "" {
						widget["id"] = "custom_widget"
					}
					common.Success(c, gin.H{"widget": widget})
				})
				dashboard.GET("/widgets/:widget_id/data", middleware.RequirePermission("report", "read"), func(c *gin.Context) {
					common.Success(c, dashboardWidgetByID(c.Param("widget_id")))
				})
				dashboard.POST("/widgets/:widget_id/refresh", middleware.RequirePermission("report", "update"), func(c *gin.Context) {
					common.Success(c, dashboardWidgetByID(c.Param("widget_id")))
				})
				dashboard.PUT("/widgets/:widget_id", middleware.RequirePermission("widget", "update"), func(c *gin.Context) {
					widget := dashboardWidgetByID(c.Param("widget_id"))
					var payload map[string]interface{}
					if err := c.ShouldBindJSON(&payload); err == nil {
						for key, value := range payload {
							widget[key] = value
						}
					}
					common.Success(c, gin.H{"widget": widget})
				})
				dashboard.DELETE("/widgets/:widget_id", middleware.RequirePermission("widget", "delete"), func(c *gin.Context) {
					common.Success(c, gin.H{"success": true})
				})
				dashboard.GET("/charts/:chart_type", middleware.RequirePermission("report", "read"), func(c *gin.Context) {
					common.Success(c, gin.H{
						"labels":   []string{},
						"datasets": []gin.H{{"label": c.Param("chart_type"), "data": []int{}}},
					})
				})
				dashboard.GET("/realtime/:data_type", middleware.RequirePermission("report", "read"), func(c *gin.Context) {
					common.Success(c, gin.H{
						"type":      c.Param("data_type"),
						"data":      gin.H{},
						"timestamp": time.Now().Format(time.RFC3339),
					})
				})
				dashboard.GET("/stats", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetStats)
				if config.TicketController != nil {
					dashboard.GET("/stats/tickets", middleware.RequirePermission("report", "read"), config.TicketController.GetTicketStats)
				} else {
					dashboard.GET("/stats/tickets", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetStats)
				}
				dashboard.GET("/stats/users", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetUserStats)
				dashboard.GET("/stats/system", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetSystemStats)
				dashboard.GET("/kpi-metrics", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetKPIMetrics)
				dashboard.GET("/metrics/performance", middleware.RequirePermission("report", "read"), func(c *gin.Context) {
					common.Success(c, gin.H{
						"loadTime":      0,
						"renderTime":    0,
						"dataFetchTime": 0,
						"widgetCount":   len(defaultDashboardWidgets()),
						"memoryUsage":   0,
					})
				})
				dashboard.GET("/metrics/usage", middleware.RequirePermission("report", "read"), func(c *gin.Context) {
					common.Success(c, gin.H{
						"totalViews":         0,
						"uniqueUsers":        0,
						"avgSessionDuration": 0,
						"mostUsedWidgets":    []gin.H{},
						"peakUsageHours":     []int{},
					})
				})
				// P1-01 别名：/dashboard/metrics → 通用 stats（前端默认 fetch 路径）
				dashboard.GET("/metrics", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetStats)
				dashboard.GET("/ticket-trend", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetTicketTrend)
				dashboard.GET("/incident-distribution", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetIncidentDistribution)
				dashboard.GET("/sla-data", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetSLAData)
				dashboard.GET("/satisfaction-data", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetSatisfactionData)
				dashboard.GET("/quick-actions", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetQuickActions)
				dashboard.GET("/recent-activities", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetRecentActivities)
				dashboard.GET("/reports", middleware.RequirePermission("report", "read"), func(c *gin.Context) {
					common.Success(c, gin.H{"reports": []gin.H{}, "total": 0, "page": 1, "pageSize": 20})
				})
				dashboard.POST("/reports/:report_type", middleware.RequirePermission("report", "create"), func(c *gin.Context) {
					common.Success(c, gin.H{
						"id":         0,
						"name":       c.Param("report_type"),
						"type":       c.Param("report_type"),
						"template":   gin.H{"title": c.Param("report_type"), "sections": []gin.H{}, "filters": []gin.H{}, "timeRange": "7d", "format": "html"},
						"recipients": []string{},
						"isActive":   false,
						"createdBy":  0,
						"createdAt":  time.Now().Format(time.RFC3339),
						"updatedAt":  time.Now().Format(time.RFC3339),
					})
				})
				dashboard.GET("/reports/:report_id/download", middleware.RequirePermission("report", "read"), func(c *gin.Context) {
					c.Data(200, "text/plain; charset=utf-8", []byte("report is not generated yet"))
				})
				dashboard.POST("/export", middleware.RequirePermission("report", "create"), func(c *gin.Context) {
					common.Success(c, gin.H{"download_url": ""})
				})
				dashboard.GET("/templates", middleware.RequirePermission("report", "read"), func(c *gin.Context) {
					common.Success(c, []gin.H{defaultDashboardTemplate()})
				})
				dashboard.POST("/templates", middleware.RequirePermission("report", "create"), func(c *gin.Context) {
					common.Success(c, gin.H{"template": defaultDashboardTemplate()})
				})
				dashboard.POST("/templates/:template_id/apply", middleware.RequirePermission("report", "update"), func(c *gin.Context) {
					common.Success(c, gin.H{"success": true, "config": defaultDashboardConfig()})
				})
			}
		}

		// ==================== Reports 报表 ====================
		reports := tenant.(*gin.RouterGroup).Group("/reports")
		{
			reports.GET("", middleware.RequirePermission("report", "read"), func(c *gin.Context) {
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
			reports.GET("/tickets", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetStats)
			reports.GET("/incidents", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetIncidentDistribution)
			reports.GET("/problems", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetTicketTrend)
			reports.GET("/changes", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetTicketTrend)
			reports.GET("/sla", middleware.RequirePermission("report", "read"), config.DashboardHandler.GetSLAData)
		}

		// ==================== Surveys 客户满意度调查 ====================
		if config.SurveyController != nil {
			surveys := tenant.(*gin.RouterGroup).Group("/surveys")
			{
				surveys.GET("", middleware.RequirePermission("survey", "read"), config.SurveyController.ListSurveys)
				surveys.POST("", middleware.RequirePermission("survey", "write"), config.SurveyController.CreateSurvey)
				surveys.GET("/:id", middleware.RequirePermission("survey", "read"), config.SurveyController.GetSurvey)
				surveys.PUT("/:id", middleware.RequirePermission("survey", "write"), config.SurveyController.UpdateSurvey)
				surveys.GET("/:id/responses", middleware.RequirePermission("survey", "read"), config.SurveyController.GetSurveyResponses)
				surveys.GET("/:id/analytics", middleware.RequirePermission("survey", "read"), config.SurveyController.GetAnalytics)
				surveys.POST("/responses", middleware.RequirePermission("survey", "write"), config.SurveyController.SubmitResponse)
			}
		}

		// ==================== Cloud (云账号/云资源/云服务) ====================
		if config.CloudController != nil {
			cloud := tenant.(*gin.RouterGroup).Group("/cloud")
			{
				// Cloud Accounts (云账号)
				cloudAccounts := cloud.Group("/accounts")
				{
					cloudAccounts.GET("", middleware.RequirePermission("cloud_account", "read"), config.CloudController.ListCloudAccounts)
					cloudAccounts.POST("", middleware.RequirePermission("cloud_account", "write"), config.CloudController.CreateCloudAccount)
					cloudAccounts.GET("/:id", middleware.RequirePermission("cloud_account", "read"), config.CloudController.GetCloudAccount)
					cloudAccounts.PUT("/:id", middleware.RequirePermission("cloud_account", "write"), config.CloudController.UpdateCloudAccount)
					cloudAccounts.DELETE("/:id", middleware.RequirePermission("cloud_account", "delete"), config.CloudController.DeleteCloudAccount)
				}

				// Cloud Services (云服务)
				cloudServices := cloud.Group("/services")
				{
					cloudServices.GET("", middleware.RequirePermission("cloud_service", "read"), config.CloudController.ListCloudServices)
					cloudServices.POST("", middleware.RequirePermission("cloud_service", "write"), config.CloudController.CreateCloudService)
					cloudServices.GET("/:id", middleware.RequirePermission("cloud_service", "read"), config.CloudController.GetCloudService)
					cloudServices.PUT("/:id", middleware.RequirePermission("cloud_service", "write"), config.CloudController.UpdateCloudService)
					cloudServices.DELETE("/:id", middleware.RequirePermission("cloud_service", "delete"), config.CloudController.DeleteCloudService)
				}

				// Cloud Resources (云资源)
				cloudResources := cloud.Group("/resources")
				{
					cloudResources.GET("", middleware.RequirePermission("cloud_resource", "read"), config.CloudController.ListCloudResources)
					cloudResources.POST("", middleware.RequirePermission("cloud_resource", "write"), config.CloudController.CreateCloudResource)
					cloudResources.GET("/:id", middleware.RequirePermission("cloud_resource", "read"), config.CloudController.GetCloudResource)
					cloudResources.PUT("/:id", middleware.RequirePermission("cloud_resource", "write"), config.CloudController.UpdateCloudResource)
					cloudResources.DELETE("/:id", middleware.RequirePermission("cloud_resource", "delete"), config.CloudController.DeleteCloudResource)
				}
			}
		}

		// ==================== Legacy Compatibility Routes ====================
		// These old paths are kept only to return explicit guidance. They must
		// not pretend that writes succeeded before a real backend is wired.
		tenant.GET("/workflows", middleware.RequirePermission("workflow", "read"), func(c *gin.Context) {
			common.Fail(c, common.BadRequestCode, "兼容接口未接入真实数据，请使用 /api/v1/bpmn/process-definitions")
		})
		tenant.POST("/workflows", middleware.RequirePermission("workflow", "create"), func(c *gin.Context) {
			common.Fail(c, common.BadRequestCode, "兼容接口不支持写入，请使用 /api/v1/bpmn/process-definitions")
		})

		// Legacy /api/v1/bpmn/definitions path; canonical BPMN APIs are
		// registered by BPMNWorkflowController under /api/v1/bpmn/process-*.
		bpmn := tenant.(*gin.RouterGroup).Group("/bpmn")
		{
			bpmn.GET("/definitions", middleware.RequirePermission("workflow", "read"), func(c *gin.Context) {
				common.Fail(c, common.BadRequestCode, "兼容接口未接入真实数据，请使用 /api/v1/bpmn/process-definitions")
			})
			bpmn.POST("/definitions", middleware.RequirePermission("workflow", "create"), func(c *gin.Context) {
				common.Fail(c, common.BadRequestCode, "兼容接口不支持写入，请使用 /api/v1/bpmn/process-definitions")
			})
			bpmn.GET("/definitions/:id", middleware.RequirePermission("workflow", "read"), func(c *gin.Context) {
				common.Fail(c, common.BadRequestCode, "兼容接口未接入真实数据，请使用 /api/v1/bpmn/process-definitions/"+c.Param("id"))
			})
			bpmn.PUT("/definitions/:id", middleware.RequirePermission("workflow", "update"), func(c *gin.Context) {
				common.Fail(c, common.BadRequestCode, "兼容接口不支持写入，请使用 /api/v1/bpmn/process-definitions/"+c.Param("id"))
			})
		}

		// Legacy service catalog path. Canonical APIs are /service-catalogs and
		// /service-catalog-services.
		tenant.GET("/services", middleware.RequirePermission("service_catalog", "read"), func(c *gin.Context) {
			common.Fail(c, common.BadRequestCode, "兼容接口未接入真实数据，请使用 /api/v1/service-catalogs")
		})
		tenant.POST("/services", middleware.RequirePermission("service_catalog", "create"), func(c *gin.Context) {
			common.Fail(c, common.BadRequestCode, "兼容接口不支持写入，请使用 /api/v1/service-catalogs")
		})

		// Legacy SLA path. Canonical SLA APIs are registered under /sla.
		tenant.GET("/slas", middleware.RequirePermission("sla", "read"), func(c *gin.Context) {
			common.Fail(c, common.BadRequestCode, "兼容接口未接入真实数据，请使用 /api/v1/sla/definitions")
		})
		tenant.POST("/slas", middleware.RequirePermission("sla", "create"), func(c *gin.Context) {
			common.Fail(c, common.BadRequestCode, "兼容接口不支持写入，请使用 /api/v1/sla/definitions")
		})

		// Legacy knowledge path. Canonical knowledge APIs are registered under
		// /knowledge/articles and /knowledge-articles.
		tenant.GET("/knowledge", middleware.RequirePermission("knowledge", "read"), func(c *gin.Context) {
			common.Fail(c, common.BadRequestCode, "兼容接口未接入真实数据，请使用 /api/v1/knowledge/articles")
		})
		tenant.POST("/knowledge", middleware.RequirePermission("knowledge", "create"), func(c *gin.Context) {
			common.Fail(c, common.BadRequestCode, "兼容接口不支持写入，请使用 /api/v1/knowledge/articles")
		})

		// Static ticket type lookup kept for old clients.
		tenant.GET("/ticket-types", middleware.RequirePermission("ticket", "read"), func(c *gin.Context) {
			common.Success(c, gin.H{"types": []gin.H{
				{"id": 1, "name": " Incident", "code": "incident"},
				{"id": 2, "name": "Problem", "code": "problem"},
				{"id": 3, "name": "Change", "code": "change"},
				{"id": 4, "name": "Request", "code": "request"},
			}, "total": 4})
		})
		tenant.POST("/ticket-types", middleware.RequirePermission("ticket", "create"), func(c *gin.Context) {
			common.Fail(c, common.BadRequestCode, "兼容接口不支持写入，工单类型由系统配置和分类接口维护")
		})
	}

	// 飞书相关路由
	if config.FeishuController != nil {
		SetupFeishuRoutes(auth, public, config.FeishuController)
	}
}
