package router

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"time"

	"itsm-backend/common"
	connectorAlert "itsm-backend/connector/alert"
	"itsm-backend/ent"
	"itsm-backend/handlers"
	a2uiHandler "itsm-backend/handlers/a2ui"
	"itsm-backend/handlers/ai"
	analyticsHandler "itsm-backend/handlers/analytics"
	applicationHandler "itsm-backend/handlers/application"
	approvalHandler "itsm-backend/handlers/approval"
	approvalChainHandler "itsm-backend/handlers/approval_chain"
	assetHandler "itsm-backend/handlers/asset"
	assignmentSmartHandler "itsm-backend/handlers/assignment_smart"
	auditlogHandler "itsm-backend/handlers/auditlog"
	authHandler "itsm-backend/handlers/auth"
	automationRuleHandler "itsm-backend/handlers/automation_rule"
	bpmnHandler "itsm-backend/handlers/bpmn"
	"itsm-backend/handlers/cab"
	"itsm-backend/handlers/capability"
	"itsm-backend/handlers/change"
	"itsm-backend/handlers/cloud"
	"itsm-backend/handlers/cmdb"
	domainCommon "itsm-backend/handlers/common"
	connectorHandler "itsm-backend/handlers/connector"
	"itsm-backend/handlers/email_intake"
	escalationMatrixHandler "itsm-backend/handlers/escalation_matrix"
	feishuHandler "itsm-backend/handlers/feishu"
	globalSearchHandler "itsm-backend/handlers/global_search"
	groupHandler "itsm-backend/handlers/group"
	incidentHandler "itsm-backend/handlers/incident"
	"itsm-backend/handlers/knowledge"
	"itsm-backend/handlers/known_error"
	marketplaceHandler "itsm-backend/handlers/marketplace"
	mspHandler "itsm-backend/handlers/msp"
	notificationHandler "itsm-backend/handlers/notification"
	"itsm-backend/handlers/operations"
	predictionHandler "itsm-backend/handlers/prediction"
	"itsm-backend/handlers/problem"
	"itsm-backend/handlers/problem_investigation"
	projectHandler "itsm-backend/handlers/project"
	provisioningHandler "itsm-backend/handlers/provisioning"
	rbacHandler "itsm-backend/handlers/rbac"
	releaseHandler "itsm-backend/handlers/release"
	"itsm-backend/handlers/service_catalog"
	"itsm-backend/handlers/service_request"
	"itsm-backend/handlers/skill"
	"itsm-backend/handlers/sla"
	slaTemplateHandler "itsm-backend/handlers/sla_template"
	"itsm-backend/handlers/standard_change"
	surveyHandler "itsm-backend/handlers/survey"
	systemConfigHandler "itsm-backend/handlers/systemconfig"
	tenantHandler "itsm-backend/handlers/tenant"
	ticketHandler "itsm-backend/handlers/ticket"
	ticketAttachmentHandler "itsm-backend/handlers/ticket_attachment"
	ticketCategoryHandler "itsm-backend/handlers/ticket_category"
	ticketCommentHandler "itsm-backend/handlers/ticket_comment"
	ticketDependencyHandler "itsm-backend/handlers/ticket_dependency"
	ticketNotificationHandler "itsm-backend/handlers/ticket_notification"
	ticketRatingHandler "itsm-backend/handlers/ticket_rating"
	ticketTagHandler "itsm-backend/handlers/ticket_tag"
	ticketTypeHandler "itsm-backend/handlers/ticket_type"
	ticketViewHandler "itsm-backend/handlers/ticket_view"
	ticketWorkflowHandler "itsm-backend/handlers/ticket_workflow"
	usersHandler "itsm-backend/handlers/user"
	vectorStoreHandler "itsm-backend/handlers/vector_store"
	vendorHandler "itsm-backend/handlers/vendor"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

	// 进程启动时间（用于系统状态接口的 uptime 计算）
	AppStartTime time.Time

	// Controllers
	ProblemInvestigationHandler  *problem_investigation.Handler
	TicketHandler                *ticketHandler.Handler
	TicketDependencyHandler      *ticketDependencyHandler.Handler
	TicketCommentHandler         *ticketCommentHandler.Handler
	TicketAttachmentHandler      *ticketAttachmentHandler.Handler
	TicketNotificationHandler    *ticketNotificationHandler.Handler
	TicketRatingHandler          *ticketRatingHandler.Handler
	TicketAssignmentSmartHandler *assignmentSmartHandler.Handler
	TicketViewHandler            *ticketViewHandler.Handler
	TicketWorkflowHandler        *ticketWorkflowHandler.Handler
	TicketAutomationRuleHandler  *automationRuleHandler.Handler
	IncidentHandler              *incidentHandler.IncidentHandler
	ApprovalHandler              *approvalHandler.Handler
	BPMNHandler                  *bpmnHandler.Handler

	A2UIHandler      *a2uiHandler.Handler
	DashboardHandler *handlers.DashboardHandler

	// Organization & Project
	ProjectHandler     *projectHandler.Handler
	ApplicationHandler *applicationHandler.Handler

	// Ticket related controllers
	TicketCategoryHandler *ticketCategoryHandler.Handler
	TicketTypeHandler     *ticketTypeHandler.Handler

	// CMDB Controllers
	TicketTagHandler *ticketTagHandler.Handler

	// User Handler
	UserHandler *usersHandler.UserHandler

	// Group Handler
	GroupHandler *groupHandler.Handler

	// RBAC and tenant domain handlers
	RBACHandler             *rbacHandler.Handler
	TenantHandler           *tenantHandler.Handler
	MSPHandler              *mspHandler.Handler
	SystemConfigHandler     *systemConfigHandler.Handler
	ApprovalChainHandler    *approvalChainHandler.Handler
	EscalationMatrixHandler *escalationMatrixHandler.Handler
	AuditLogHandler         *auditlogHandler.Handler
	NotificationHandler     *notificationHandler.Handler

	// Additional domain controllers
	ProvisioningHandler *provisioningHandler.Handler
	AnalyticsHandler    *analyticsHandler.Handler
	PredictionHandler   *predictionHandler.Handler
	ReleaseHandler      *releaseHandler.ReleaseHandler
	AssetHandler        *assetHandler.Handler
	VendorHandler       *vendorHandler.Handler
	SurveyHandler       *surveyHandler.Handler
	CloudHandler        *cloud.Handler

	// Domain Handlers
	ServiceCatalogHandler *service_catalog.Handler
	ServiceRequestHandler *service_request.Handler
	CMDBHandler           *cmdb.Handler

	ProblemHandler     *problem.Handler
	ChangeHandler      *change.Handler
	CABHandler         *cab.Handler
	KnowledgeHandler   *knowledge.Handler
	SLAHandler         *sla.Handler
	SLATemplateHandler *slaTemplateHandler.Handler
	AIHandler          *ai.Handler
	EmailIntakeHandler *email_intake.Handler
	// VectorStoreController 提供向量存储（RAG 检索底座）状态查看与连通性测试，
	// 注册 /api/v1/system/vector-store*；为 nil 时路由不注册。
	VectorStoreHandler *vectorStoreHandler.Handler
	CommonHandler      *domainCommon.Handler
	AuthHandler        *authHandler.Handler
	RoleHandler        *common.RoleHandler

	// Sprint C — Skill Registry v1
	SkillHandler *skill.Handler

	// WebSocket Service
	WebSocketService *service.WebSocketService

	// Global Search
	GlobalSearchHandler *globalSearchHandler.Handler

	// Standard Change Handler (标准变更模板库)
	StandardChangeHandler *standard_change.Handler

	// Known Error Handler (KEDB)
	KnownErrorHandler *known_error.Handler

	// Connector Controller (连接器/插件/技能市场)
	ConnectorHandler   *connectorHandler.Handler
	AlertHandler       *connectorAlert.Handler
	FeishuHandler      *feishuHandler.Handler
	MarketplaceHandler *marketplaceHandler.Handler

	// Ticket Association Service (工单关联服务)
	TicketAssociationService *service.TicketAssociationService
}

// SetupRoutes 设置路由
func SetupRoutes(r *gin.Engine, config *RouterConfig) {
	// Swagger UI：默认关闭，开发环境设 ENABLE_SWAGGER=true 开启（见 .env.dev.example）。
	// 生产环境不应暴露 API 文档，故不在此无条件注册。
	if os.Getenv("ENABLE_SWAGGER") == "true" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
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
		// Redis 故障时使用更严格的本地限制（10/min），禁止故障放行。
		fallbackLimiter := middleware.NewRateLimiter(10, time.Minute)
		r.Use(func(limiter RateLimiterInterface) gin.HandlerFunc {
			return func(c *gin.Context) {
				clientIP := c.ClientIP()
				allowed, err := limiter.Allow(c.Request.Context(), clientIP)
				if err != nil {
					config.Logger.Warnw("Redis rate limiter unavailable, using strict in-memory fallback", "error", err)
					allowed = fallbackLimiter.Allow(clientIP)
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
		registerBootstrapRoutes(public, config)
		if config.CommonHandler != nil {
			public.POST("/auth/login", middleware.LoginRateLimiter(), config.CommonHandler.Login)
			public.POST("/refresh-token", config.CommonHandler.RefreshToken)
			public.POST("/auth/refresh", config.CommonHandler.RefreshToken)
		}

		// 无需认证的账号自助端点（注册/密码找回/重置）
		if config.AuthHandler != nil {
			public.POST("/auth/register", middleware.LoginRateLimiter(), config.AuthHandler.Register)
			public.POST("/auth/forgot-password", middleware.LoginRateLimiter(), config.AuthHandler.ForgotPassword)
			public.POST("/auth/reset-password", middleware.LoginRateLimiter(), config.AuthHandler.ResetPassword)
			public.POST("/auth/validate-reset-token", middleware.LoginRateLimiter(), config.AuthHandler.ValidateResetToken)
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
			// Keep APP_VERSION in sync with itsm-frontend/package.json and the
			// release tag; the fallback is only used when it is not injected.
			version := os.Getenv("APP_VERSION")
			if version == "" {
				version = "1.6.9"
			}
			c.JSON(200, gin.H{"version": version, "build": "release"})
		})
		public.GET("/readiness/ga", func(c *gin.Context) {
			common.Success(c, buildGAReadiness(c.Request.Context(), config.Client))
		})

		// Prometheus metrics endpoint.
		//
		// The backend publishes no host port in docker-compose.prod.yml (nginx is
		// the only entrypoint), so /metrics is not reachable from outside the
		// container network. It stays authenticated by default so that adding a
		// port mapping later cannot silently expose metrics.
		//
		// To let Prometheus scrape it, set METRICS_REQUIRE_AUTH=false AND keep the
		// backend unpublished to untrusted networks.
		metricsAuth := r.Group("")
		if os.Getenv("METRICS_REQUIRE_AUTH") == "false" {
			zap.S().Warn("METRICS_REQUIRE_AUTH=false: /metrics is unauthenticated; " +
				"only safe while the backend port is not published to untrusted networks")
		} else {
			metricsAuth.Use(middleware.AuthMiddleware(config.JWTSecret))
		}
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
	auth.GET("/capabilities", capability.Handler)
	if config.Client != nil {
		operationHandler := operations.NewHandler(operations.NewService(config.Client))
		operationRoutes := auth.Group("/admin/operations/commands")
		operationRoutes.Use(middleware.RequirePermission("system", "write"))
		operationRoutes.GET("", operationHandler.List)
		operationRoutes.GET("/:id", operationHandler.Get)
		operationRoutes.POST("/:id/replay", operationHandler.Replay)
		operationRoutes.POST("/:id/cancel", operationHandler.Cancel)
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
	if config.MSPHandler != nil {
		msp := r.Group("/api/v1/msp")
		msp.Use(middleware.AuthMiddleware(config.JWTSecret))
		msp.Use(middleware.RBACMiddleware(config.Client)) // 设置 client 到 context
		msp.Use(middleware.MSPMiddleware(config.Client))
		{
			// MSP 基础信息 - 允许 MSP 员工和管理员访问
			msp.GET("/status", config.MSPHandler.GetMSPStatus)
			msp.GET("/context", middleware.RequireMSPPermission("msp", "read"), config.MSPHandler.GetMSPContext)

			// 分配管理 - 需要 msp_allocation 权限
			msp.GET("/allocations", middleware.RequireMSPPermission("msp_allocation", "read"), config.MSPHandler.GetAllocations)
			msp.POST("/allocations", middleware.RequireMSPPermission("msp_allocation", "write"), config.MSPHandler.CreateAllocation)
			msp.POST("/allocations/deallocate", middleware.RequireMSPPermission("msp_allocation", "write"), config.MSPHandler.Deallocate)

			// 客户管理 - 需要 msp_customer 权限
			msp.GET("/customers", middleware.RequireMSPPermission("msp_customer", "read"), config.MSPHandler.GetAllCustomers)
			msp.GET("/customers/:customer_tenant_id/tickets", middleware.RequireMSPPermission("msp_ticket", "read"), config.MSPHandler.GetCustomerTickets)

			// 工单分配 - 需要 msp_ticket:write 权限
			msp.POST("/tickets/:id/assign", middleware.RequireMSPPermission("msp_ticket", "write"), config.MSPHandler.AssignMSPTechnician)

			// 报表 - 需要 msp_report 权限
			msp.GET("/reports/customers", middleware.RequireMSPPermission("msp_report", "read"), config.MSPHandler.GetCustomerReports)
			msp.GET("/reports/performance", middleware.RequireMSPPermission("msp_report", "read"), config.MSPHandler.GetPerformanceReports)
		}
	}

	{
		// 租户中间件
		tenant := auth.Use(middleware.TenantMiddleware(config.Client))
		// 审计必须位于租户解析之后，确保所有租户内写操作都有可靠的租户和操作者上下文。
		tenant.(*gin.RouterGroup).Use(middleware.AuditMiddleware(config.Client))

		// ==================== Ticket Categories & Tags ====================
		if config.TicketCategoryHandler != nil {
			SetupTicketCategoryRoutes(tenant.(*gin.RouterGroup), config.TicketCategoryHandler)
		}

		if config.TicketTagHandler != nil {
			SetupTicketTagRoutes(tenant.(*gin.RouterGroup), config.TicketTagHandler)
		}

		// ==================== Tickets ====================
		SetupTicketRoutes(tenant.(*gin.RouterGroup), config)

		// ==================== System Configs ====================
		if config.SystemConfigHandler != nil {
			SetupSystemConfigRoutes(tenant.(*gin.RouterGroup), config.SystemConfigHandler, config.TenantHandler, config.VectorStoreHandler, config.AppStartTime)
		}

		// ==================== Approval Chains ====================
		if config.ApprovalChainHandler != nil {
			SetupApprovalChainRoutes(tenant.(*gin.RouterGroup), config.ApprovalChainHandler, config.EscalationMatrixHandler)
		}

		// ==================== Incidents ====================
		if config.IncidentHandler != nil {
			SetupIncidentRoutes(tenant.(*gin.RouterGroup), config.IncidentHandler)
		}
		if config.CMDBHandler != nil {
			tenant.GET("/incidents/configuration-items", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.ListCIs)
		}

		// ==================== Service Catalog & Requests (DDD) ====================
		if config.ServiceCatalogHandler != nil {
			SetupServiceCatalogRoutes(tenant.(*gin.RouterGroup), config.ServiceCatalogHandler)
		}

		if config.ServiceRequestHandler != nil {
			SetupServiceRequestRoutes(tenant.(*gin.RouterGroup), config.ServiceRequestHandler, config.ProvisioningHandler)
		}

		// ==================== Problems (DDD) ====================
		if config.ProblemHandler != nil {
			SetupProblemRoutes(tenant.(*gin.RouterGroup), config.ProblemHandler)
			// 问题调查关联列表（前端契约：GET /api/v1/problems/:id/relationships）
			if config.ProblemInvestigationHandler != nil {
				tenant.(*gin.RouterGroup).GET("/problems/:id/relationships", middleware.RequirePermission("problem", "read"), config.ProblemInvestigationHandler.GetProblemRelationships)
			}
		}

		// ==================== Changes (DDD) ====================
		if config.ChangeHandler != nil {
			SetupChangeRoutes(tenant.(*gin.RouterGroup), config.ChangeHandler)
		}

		// ==================== CAB (Change Advisory Board) ====================
		// 名册管理走 change 权限（CAB 属变更管理范畴，变更管理员/管理员已持该权限）。
		// CAB 审批流转由审批链引擎（cab:CAB / cab:ECAB 解析器）统一驱动，不在此暴露。
		if config.CABHandler != nil {
			SetupCABRoutes(tenant.(*gin.RouterGroup), config.CABHandler)
		}

		// ==================== Releases ====================
		if config.ReleaseHandler != nil {
			SetupReleaseRoutes(tenant.(*gin.RouterGroup), config.ReleaseHandler)
		}

		// ==================== Assets ====================
		if config.AssetHandler != nil {
			assetHandler.SetupRoutes(tenant.(*gin.RouterGroup), config.AssetHandler)
		}
		// ==================== CMDB ====================
		if config.CMDBHandler != nil {
			SetupCMDBRoutes(tenant.(*gin.RouterGroup), config)
		}

		// ==================== Asset Licenses ====================
		// Licenses are served by the same AssetHandler
		// (no separate handler — routes registered above via SetupRoutes)

		// ==================== Vendors ====================
		if config.VendorHandler != nil {
			SetupVendorRoutes(tenant.(*gin.RouterGroup), config.VendorHandler)
		}

		// ==================== Knowledge Base (DDD) ====================
		if config.KnowledgeHandler != nil {
			SetupKnowledgeRoutes(tenant.(*gin.RouterGroup), config.KnowledgeHandler)
		}

		if config.SLAHandler != nil {
			SetupSLARoutes(tenant.(*gin.RouterGroup), config.SLAHandler)
			// SLA 模板（开箱即用预置模板）
			if config.SLATemplateHandler != nil {
				SetupSLATemplateRoutes(tenant.(*gin.RouterGroup), config.SLATemplateHandler)
			}
		}

		// ==================== AI & Analytics (DDD) ====================
		if config.AIHandler != nil {
			SetupAIRoutes(tenant.(*gin.RouterGroup), config.AIHandler)
		}

		// ==================== Skill Registry v1 ====================
		// Sprint C：技能管理与调用入口。
		// 路径：
		//   GET    /api/v1/skills                  列表
		//   GET    /api/v1/skills/:code            详情
		//   POST   /api/v1/admin/skills            注册（custom）
		//   PUT    /api/v1/admin/skills/:code      更新
		//   POST   /api/v1/admin/skills/:code/promote  pilot → ga
		//   DELETE /api/v1/admin/skills/:code      禁用
		//   POST   /api/v1/admin/skills/:code/invoke   统一调用入口
		if config.SkillHandler != nil {
			config.SkillHandler.RegisterRoutes(tenant.(*gin.RouterGroup))
		}
		if config.EmailIntakeHandler != nil {
			config.EmailIntakeHandler.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// ==================== Common & System (DDD) ====================
		if config.CommonHandler != nil {
			SetupCommonSystemRoutes(r, tenant.(*gin.RouterGroup), config)
		}

		// ==================== Problem Investigation ====================
		// 权限说明：investigation/step/root_cause/solution 等资源未在 seeder 中定义，
		// 统一复用已 seeded 且已赋权运营角色的 problem 权限，避免注册后全员 403。
		if config.ProblemInvestigationHandler != nil {
			SetupProblemInvestigationRoutes(tenant.(*gin.RouterGroup), config.ProblemInvestigationHandler)
		}

		// ==================== BPMN ====================
		if config.BPMNHandler != nil {
			config.BPMNHandler.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// A2UI Ticket Controller (AI-driven UI表单)
		if config.A2UIHandler != nil {
			config.A2UIHandler.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// Global Search Controller (全局搜索)
		if config.GlobalSearchHandler != nil {
			config.GlobalSearchHandler.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// Standard Change Handler (标准变更模板库)
		if config.StandardChangeHandler != nil {
			config.StandardChangeHandler.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// Known Error Handler (KEDB)
		if config.KnownErrorHandler != nil {
			config.KnownErrorHandler.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		if config.MarketplaceHandler != nil {
			config.MarketplaceHandler.RegisterRoutes(tenant.(*gin.RouterGroup))
		}

		// Connector Controller (连接器/插件/技能市场)
		if config.ConnectorHandler != nil {
			SetupConnectorRoutes(tenant.(*gin.RouterGroup), config.ConnectorHandler)
		}

		if config.AlertHandler != nil {
			SetupAlertRoutes(tenant.(*gin.RouterGroup), config.AlertHandler)
		}

		if config.DashboardHandler != nil {
			SetupDashboardRoutes(tenant.(*gin.RouterGroup), config.DashboardHandler, config.TicketHandler)
		}

		// ==================== Reports 报表 ====================
		if config.DashboardHandler != nil {
			SetupReportsRoutes(tenant.(*gin.RouterGroup), config.DashboardHandler)
		}

		// ==================== Surveys 客户满意度调查 ====================
		if config.SurveyHandler != nil {
			SetupSurveyRoutes(tenant.(*gin.RouterGroup), config.SurveyHandler)
		}

		// ==================== Cloud (云账号/云资源/云服务) ====================
		if config.CloudHandler != nil {
			SetupCloudRoutes(tenant.(*gin.RouterGroup), config.CloudHandler)
		}

		// ==================== Legacy Compatibility Routes ====================
		// Legacy /workflows → /bpmn/process-definitions (BPMN handler handles /bpmn/*)
		tenant.GET("/workflows", middleware.RequirePermission("workflow", "read"), func(c *gin.Context) {
			common.Fail(c, common.BadRequestCode, "请使用 /api/v1/bpmn/process-definitions")
		})
		tenant.POST("/workflows", middleware.RequirePermission("workflow", "create"), func(c *gin.Context) {
			common.Fail(c, common.BadRequestCode, "请使用 /api/v1/bpmn/process-definitions")
		})

		if config.TicketTypeHandler != nil {
			SetupTicketTypeRoutes(tenant.(*gin.RouterGroup), config.TicketTypeHandler)
		}
	}

	// 飞书相关路由
	if config.FeishuHandler != nil {
		SetupFeishuRoutes(auth, public, config.FeishuHandler)
	}
}
