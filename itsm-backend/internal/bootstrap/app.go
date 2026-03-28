package bootstrap

import (
	"context"
	"fmt"
	"log"
	"time"

	"itsm-backend/common"
	"itsm-backend/config"
	"itsm-backend/controller"
	"itsm-backend/database"
	"itsm-backend/ent"
	"itsm-backend/handlers"
	"itsm-backend/handlers/ai"
	"itsm-backend/handlers/change"
	"itsm-backend/handlers/cmdb"
	domainCommon "itsm-backend/handlers/common"
	"itsm-backend/handlers/incident"
	"itsm-backend/handlers/knowledge"
	"itsm-backend/handlers/problem"
	"itsm-backend/handlers/service_catalog"
	"itsm-backend/handlers/service_request"
	"itsm-backend/handlers/sla"
	"itsm-backend/handlers/standard_change"
	"itsm-backend/handlers/known_error"
	"itsm-backend/middleware"
	"itsm-backend/pkg/seeder"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Application struct {
	Cfg         *config.Config
	Logger      *zap.SugaredLogger
	DBClient    *ent.Client
	Router      *gin.Engine
	Embedder    service.Embedder
	VectorStore *service.VectorStore
}

func NewApplication() *Application {
	// 1. 初始化配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. 初始化日志系统
	logger := initLogger(&cfg.Log)
	sugar := logger.Sugar()
	middleware.SetLogger(sugar)

	// 3. 初始化权限配置（数据库优先，不存在时使用硬编码权限）
	middleware.PermissionConfig.Mode = middleware.PermissionConfigModeFallback

	// 3. 初始化数据库连接
	client, err := database.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 4. 创建数据库schema
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("Failed to create schema resources: %v", err)
	}

	// 5. 运行 Seeder
	// 移除了匿名函数，改为使用 Seeder 结构体
	s := seeder.NewSeeder(client, sugar)
	s.SeedAll(context.Background())

	// 6. 初始化服务层 & 控制器
	// 这部分代码量较大，为了简化，我们先在这里进行组装，后续可以进一步拆分为 wires / container

	// 初始化业务服务层
	incidentService := service.NewIncidentService(client, sugar)

	// 初始化 Redis 序列服务（用于工单编号生成）
	// 如果 Redis 不可用，使用数据库回退方案
	var sequenceService *service.SequenceService
	ss := service.NewSequenceService(
		cfg.Redis.Host,
		cfg.Redis.Port,
		cfg.Redis.Password,
		cfg.Redis.DB,
		sugar,
	)
	if ss != nil {
		sequenceService = ss
		sugar.Infow("Redis sequence service initialized successfully")
	} else {
		sugar.Warnw("Redis sequence service not available, will use database fallback for ticket number")
	}
	ticketService := service.NewTicketServiceWithSequence(client, sugar, sequenceService)

	// MSP 服务初始化
	mspAllocationService := service.NewMSPAllocationService(client, sugar)
	mspController := controller.NewMSPController(mspAllocationService, ticketService, sugar)

	serviceCatalogService := service.NewServiceCatalogService(client, sugar)
	// 审批服务
	approvalService := service.NewApprovalService(client, sugar)

	cmdbService := service.NewCMDBService(client)
	// problemService and changeService removed - using Handlers with domain services instead

	// Release & Asset Management Services
	releaseService := service.NewReleaseService(client, sugar)
	assetService := service.NewAssetService(client, sugar)
	assetLicenseService := service.NewAssetLicenseService(client, sugar)

	// LLM/Embedding/VectorStore
	var embedder service.Embedder
	if cfg.LLM.Provider == "openai" || cfg.LLM.Provider == "" {
		embedder = service.NewOpenAIEmbedderWithConfig(cfg.LLM.APIKey, cfg.LLM.Endpoint, cfg.LLM.Model)
	} else {
		embedder = service.NewOpenAIEmbedder()
	}
	vectorStore := service.NewVectorStore(database.GetRawDB())
	ragService := service.NewRAGServiceWithAutoConfig(client, vectorStore, embedder, sugar)
	aiTelemetryService := service.NewAITelemetryService(database.GetRawDB())

	// 控制器依赖
	incidentRuleEngine := service.NewIncidentRuleEngine(client, sugar)
	incidentMonitoringService := service.NewIncidentMonitoringService(client, sugar)
	incidentAlertingService := service.NewIncidentAlertingService(client, sugar)
	ticketDependencyService := service.NewTicketDependencyService(client, sugar)
	analyticsService := service.NewAnalyticsService(client, sugar)
	predictionService := service.NewPredictionService(client, sugar)
	rootCauseService := service.NewRootCauseService(client, sugar)
	// LLM/Embedding/VectorStore

	// AI Tools
	toolRegistry := service.NewToolRegistry(ragService, incidentService, cmdbService, client)
	toolQueue := service.NewToolQueue(client, toolRegistry, 100)

	ticketController := controller.NewTicketController(ticketService, ticketDependencyService, sugar)
	ticketDependencyController := controller.NewTicketDependencyController(ticketDependencyService)

	ticketCommentService := service.NewTicketCommentService(client, sugar)
	ticketCommentController := controller.NewTicketCommentController(ticketCommentService, sugar)
	ticketAttachmentService := service.NewTicketAttachmentService(client, sugar)
	ticketAttachmentController := controller.NewTicketAttachmentController(ticketAttachmentService, sugar)
	ticketNotificationService := service.NewTicketNotificationService(client, sugar)
	ticketNotificationController := controller.NewTicketNotificationController(ticketNotificationService, sugar)

	// General Notification Service & Controller
	notificationService := service.NewNotificationService(client)
	notificationController := controller.NewNotificationController(notificationService)

	// 服务请求服务（依赖通知服务）
	serviceRequestService := service.NewServiceRequestService(client, sugar, approvalService, notificationService)

	ticketRatingService := service.NewTicketRatingService(client, sugar)
	ticketRatingController := controller.NewTicketRatingController(ticketRatingService, sugar)
	ticketViewService := service.NewTicketViewService(client, sugar)
	ticketViewController := controller.NewTicketViewController(ticketViewService, sugar)

	ticketAssignmentService := service.NewTicketAssignmentService(client, sugar)
	ticketAssignmentRuleService := service.NewTicketAssignmentRuleService(client, sugar)
	ticketAssignmentSmartService := service.NewTicketAssignmentSmartService(client, sugar, ticketAssignmentService, ticketAssignmentRuleService)
	ticketAssignmentSmartController := controller.NewTicketAssignmentSmartController(ticketAssignmentSmartService, ticketAssignmentRuleService, sugar)

	// Ticket Workflow Service & Controller
	ticketWorkflowService := service.NewTicketWorkflowService(client, database.GetRawDB(), sugar)
	ticketWorkflowController := controller.NewTicketWorkflowController(ticketWorkflowService, sugar)

	// Ticket Automation Rule Service & Controller
	ticketAutomationRuleService := service.NewTicketAutomationRuleService(client, sugar)
	ticketAutomationRuleController := controller.NewTicketAutomationRuleController(ticketAutomationRuleService, sugar)

	// Set notification service dependencies
	ticketService.SetNotificationService(ticketNotificationService)
	ticketCommentService.SetNotificationService(ticketNotificationService)
	ticketRatingService.SetNotificationService(ticketNotificationService)

	rootCauseAnalysisService := service.NewRootCauseAnalysisService(client)
	incidentController := controller.NewIncidentController(incidentService, incidentRuleEngine, incidentMonitoringService, incidentAlertingService, rootCauseAnalysisService, sugar)
	approvalController := controller.NewApprovalController(approvalService)

	serviceController := controller.NewServiceController(serviceCatalogService, serviceRequestService)
	provisioningService := service.NewProvisioningService(client, sugar)
	provisioningController := controller.NewProvisioningController(provisioningService)

	// ProblemController and ChangeController removed - using Handlers instead

	// Release & Asset Management Controllers
	releaseController := controller.NewReleaseController(sugar, releaseService)
	assetController := controller.NewAssetController(sugar, assetService)
	assetLicenseController := controller.NewAssetLicenseController(sugar, assetLicenseService)

	projectController := controller.NewProjectController(client)
	applicationController := controller.NewApplicationController(client)

	ticketCategoryService := service.NewTicketCategoryService(client)
	ticketCategoryController := controller.NewTicketCategoryController(ticketCategoryService, sugar)
	ticketTagService := service.NewTicketTagService(client)
	ticketTagController := controller.NewTicketTagController(ticketTagService, sugar.Desugar())

	processEngine := service.NewCustomProcessEngine(client, sugar)
	bpmnVersionService := service.NewBPMNVersionService(client, sugar)
	bpmnWorkflowController := controller.NewBPMNWorkflowController(processEngine, bpmnVersionService)

	// BPMN Process Trigger Service & Controller (统一流程触发接口)
	bpmnTemplateService := service.NewBPMNTemplateService(client)
	processBindingService := service.NewProcessBindingService(client)
	processTriggerService := service.NewProcessTriggerService(client, processEngine)
	bpmnProcessTriggerController := controller.NewBPMNProcessTriggerController(processTriggerService, processBindingService)

	// BPMN Dashboard Controller (监控仪表盘)
	bpmnMetricsService := service.NewBPMNMetricsService(client, sugar)
	bpmnAuditService := service.NewBPMNAuditService(client, sugar)
	bpmnTenantService := service.NewBPMNTenantService(client, sugar)
	bpmnSlaService := service.NewBPMNSLAService(client, sugar)
	bpmnDashboardController := controller.NewBPMNDashboardController(bpmnMetricsService, bpmnAuditService, bpmnTenantService, bpmnSlaService)

	// A2UI Ticket Controller (AI-driven UI表单)
	a2uiTicketService := service.NewA2UITicketService(nil)
	a2uiTicketController := controller.NewA2UITicketController(a2uiTicketService)

	// Global Search Controller (全局搜索)
	globalSearchController := controller.NewGlobalSearchController(client)

	// Standard Change Handler (标准变更模板库)
	standardChangeHandler := standard_change.NewHandler(client, sugar)

	// Known Error Handler (KEDB)
	knownErrorHandler := known_error.NewHandler(client, sugar)

	// Set process trigger service for workflow integration (after processTriggerService is declared)
	ticketService.SetProcessTriggerService(processTriggerService)

	// Set approval service for ticket workflow integration
	ticketService.SetApprovalService(approvalService)

	// 初始化模板并部署默认流程
	go func() {
		ctx := context.Background()
		// 为默认租户(1)加载和部署模板
		if _, err := bpmnTemplateService.LoadAndDeployTemplates(ctx, 1); err != nil {
			sugar.Warnw("Failed to deploy BPMN templates", "error", err)
		}
		// 初始化默认流程绑定
		if err := processBindingService.InitDefaultBindings(ctx, 1); err != nil {
			sugar.Warnw("Failed to init default process bindings", "error", err)
		}
	}()

	dashboardService := service.NewDashboardService(client, sugar)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService, ticketService, incidentService, sugar)

	// Domain: Service Catalog (DDD)
	scRepo := service_catalog.NewEntRepository(client)
	scService := service_catalog.NewService(scRepo, sugar)
	scHandler := service_catalog.NewHandler(scService)

	// Domain: CMDB (DDD)
	cmdbRepo := cmdb.NewEntRepository(client)
	cmdbServiceDomain := cmdb.NewService(cmdbRepo, sugar)
	cmdbHandler := cmdb.NewHandler(cmdbServiceDomain)
	_ = cmdbHandler // CMDBHandler routes removed; using CMDBController with /configuration-items instead

	// Domain: Service Request (DDD)
	srRepo := service_request.NewEntRepository(client)
	srService := service_request.NewService(srRepo, scRepo, cmdbRepo, sugar)
	srHandler := service_request.NewHandler(srService)

	// Domain: Incident (DDD)
	// Note: Incident handler has been removed from router config
	_ = incident.NewEntRepository // Prevent unused import warning

	// Domain: Problem (DDD)
	problemRepo := problem.NewEntRepository(client)
	problemServiceDomain := problem.NewService(problemRepo, sugar)
	problemHandler := problem.NewHandler(problemServiceDomain)

	// Domain: Change (DDD)
	changeRepo := change.NewEntRepository(client, database.GetRawDB())
	changeServiceDomain := change.NewService(changeRepo, sugar)
	changeHandler := change.NewHandler(changeServiceDomain)

	// CMDB Controller (新增)
	// Note: cmdbService already declared at line 83
	ciRelationshipService := service.NewCIRelationshipService(client)
	auditLogService := service.NewAuditLogService(client, sugar)
	cloudDiscoveryService := service.NewCloudDiscoveryService(client, sugar)
	cmdbController := controller.NewCMDBController(cmdbService, ciRelationshipService, auditLogService, cmdbServiceDomain)
	cmdbController.SetCloudDiscoveryService(cloudDiscoveryService)

	// Analytics & Prediction Controllers
	analyticsController := controller.NewAnalyticsController(analyticsService)
	predictionController := controller.NewPredictionController(predictionService)

	// Domain: Knowledge (DDD)
	knowledgeRepo := knowledge.NewEntRepository(client)
	knowledgeServiceDomain := knowledge.NewService(knowledgeRepo, sugar)
	knowledgeHandler := knowledge.NewHandler(knowledgeServiceDomain)

	// Domain: SLA (DDD)
	slaRepo := sla.NewEntRepository(client)
	slaServiceDomain := sla.NewService(slaRepo, sugar)
	slaHandler := sla.NewHandler(slaServiceDomain)

	// AI Domain
	aiRepo := ai.NewEntRepository(client)
	aiServiceDomain := ai.NewService(aiRepo, sugar, ragService, toolRegistry, toolQueue, analyticsService, predictionService, rootCauseService, aiTelemetryService)
	aiHandler := ai.NewHandler(aiServiceDomain)

	// Common Domain
	commonRepo := domainCommon.NewEntRepository(client)
	commonServiceDomain := domainCommon.NewService(commonRepo, cfg.JWT.Secret, sugar, client)
	commonHandler := domainCommon.NewHandler(commonServiceDomain)

	// Role Handler (in-memory for now)
	roleHandler := common.NewRoleHandler(client, sugar)

	// User Controller
	userService := service.NewUserService(client, sugar)
	userController := controller.NewUserController(userService, sugar)

	// Group Controller
	groupService := service.NewGroupService(client)
	groupController := controller.NewGroupController(groupService, sugar)

	// Role & Permission Controllers (database-backed with tenant isolation)
	roleService := service.NewRoleService(client, sugar)
	roleController := controller.NewRoleController(roleService, sugar)
	permissionService := service.NewPermissionService(client, sugar)
	permissionController := controller.NewPermissionController(permissionService, sugar)

	// Menu Controller (database-backed with tenant isolation)
	menuService := service.NewMenuService(client, sugar)
	menuController := controller.NewMenuController(menuService)

	// Tenant Controller
	tenantService := service.NewTenantService(client, sugar)
	tenantController := controller.NewTenantController(tenantService, sugar)

	// System Config Controller
	systemConfigService := service.NewSystemConfigService(client, sugar)
	systemConfigController := controller.NewSystemConfigController(systemConfigService, sugar)

	// Approval Chain Controller
	approvalChainService := service.NewApprovalChainService(client, sugar)
	approvalChainController := controller.NewApprovalChainController(approvalChainService, sugar)

	// SLA Monitor & Alert Services (legacy, for background tasks)
	slaMonitorService := service.NewSLAMonitorService(client, sugar)
	slaAlertService := service.NewSLAAlertService(client, sugar)
	escalationService := service.NewEscalationService(client, sugar)
	// Wire up notification service
	slaMonitorService.SetNotificationService(ticketNotificationService)
	slaAlertService.SetNotificationService(ticketNotificationService)
	escalationService.SetNotificationService(ticketNotificationService)

	// 7. 设置路由
	// 根据配置设置 Gin 运行模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.Server.Mode == "test" {
		gin.SetMode(gin.TestMode)
	}
	r := gin.Default()
	if err := r.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		sugar.Warnw("failed to set trusted proxies, falling back to default", "error", err)
	}

	// 初始化 Redis 限流器（分布式环境使用）
	var redisRateLimiter router.RateLimiterInterface
	if cfg.Redis.Host != "" {
		redisClient := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		// 测试 Redis 连接
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			sugar.Warnw("Redis connection failed, rate limiter will use in-memory fallback", "error", err)
			redisRateLimiter = nil
		} else {
			sugar.Info("Redis connection established, using distributed rate limiter")
			// 默认每分钟 500 次请求
			redisRateLimiter = middleware.NewRedisRateLimiter(redisClient, 500, time.Minute)
		}
	} else {
		sugar.Warn("Redis not configured, rate limiter will use in-memory fallback (not suitable for distributed deployment)")
	}

	routerConfig := &router.RouterConfig{
		JWTSecret:                       cfg.JWT.Secret,
		Logger:                          sugar,
		Client:                          client,
		CSRFEnabled:                     cfg.Security.CSRFEnabled,
		RedisRateLimiter:                redisRateLimiter,
		TicketController:                ticketController,
		TicketDependencyController:      ticketDependencyController,
		TicketCommentController:         ticketCommentController,
		TicketAttachmentController:      ticketAttachmentController,
		TicketNotificationController:    ticketNotificationController,
		NotificationController:          notificationController,
		TicketRatingController:          ticketRatingController,
		TicketAssignmentSmartController: ticketAssignmentSmartController,
		TicketViewController:            ticketViewController,
		TicketWorkflowController:        ticketWorkflowController,
		TicketAutomationRuleController:  ticketAutomationRuleController,
		IncidentController:              incidentController,
		ApprovalController:              approvalController,
		BPMNWorkflowController:          bpmnWorkflowController,
		BPMNProcessTriggerController:    bpmnProcessTriggerController,
		BPMNDashboardController:         bpmnDashboardController,
		A2UITicketController:            a2uiTicketController,
		DashboardHandler:                dashboardHandler,
		ProjectController:               projectController,
		ApplicationController:           applicationController,
		TicketCategoryController:        ticketCategoryController,
		TicketTagController:             ticketTagController,
		UserController:                  userController,
		GroupController:                 groupController,

		// Role & Permission Controllers
		RoleController:          roleController,
		PermissionController:    permissionController,
		MenuController:          menuController,
		TenantController:        tenantController,
		MSPController:           mspController,
		SystemConfigController:  systemConfigController,
		ApprovalChainController: approvalChainController,

		// CMDB Controller (新增)
		CMDBController: cmdbController,

		// Additional controllers
		ServiceController:      serviceController,
		ProvisioningController: provisioningController,
		AnalyticsController:    analyticsController,
		PredictionController:     predictionController,
		ReleaseController:        releaseController,
		AssetController:          assetController,
		AssetLicenseController:   assetLicenseController,

		// Domain Handlers
		ServiceCatalogHandler: scHandler,
		ServiceRequestHandler: srHandler,
		ProblemHandler:        problemHandler,
		ChangeHandler:         changeHandler,
		KnowledgeHandler:      knowledgeHandler,
		SLAHandler:            slaHandler,
		AIHandler:             aiHandler, // Added AI domain handler
		CommonHandler:         commonHandler,
		RoleHandler:           roleHandler,

		// Global Search
		GlobalSearchController: globalSearchController,

		// Standard Change Handler
		StandardChangeHandler: standardChangeHandler,

		// Known Error Handler (KEDB)
		KnownErrorHandler: knownErrorHandler,
	}
	router.SetupRoutes(r, routerConfig)

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return &Application{
		Cfg:         cfg,
		Logger:      sugar,
		DBClient:    client,
		Router:      r,
		Embedder:    embedder,
		VectorStore: vectorStore,
	}
}

func (app *Application) Run() {
	defer app.Logger.Sync()
	defer app.DBClient.Close()

	// Start background tasks
	app.startBackgroundTasks()

	app.Logger.Infof("Server starting on port %d", app.Cfg.Server.Port)
	if err := app.Router.Run(fmt.Sprintf(":%d", app.Cfg.Server.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func (app *Application) startBackgroundTasks() {
	go func() {
		pipeline := service.NewEmbeddingPipeline(app.DBClient, app.Embedder, app.Logger, app.VectorStore)
		ctx := context.Background()
		// initial full-ish pass per tenant
		tenants, err := app.DBClient.Tenant.Query().All(ctx)
		if err == nil {
			for _, t := range tenants {
				_ = pipeline.RunOnce(ctx, t.ID, 200)
			}
		} else {
			// fallback default tenant 1
			_ = pipeline.RunOnce(ctx, 1, 200)
		}
		// periodic incremental per tenant
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			tenants, err := app.DBClient.Tenant.Query().All(ctx)
			if err != nil {
				continue
			}
			for _, t := range tenants {
				_ = pipeline.RunOnce(ctx, t.ID, 50)
			}
		}
	}()

	// SLA Monitoring and Escalation background tasks
	go func() {
		slaMonitorService := service.NewSLAMonitorService(app.DBClient, app.Logger)
		escalationService := service.NewEscalationService(app.DBClient, app.Logger)

		ctx := context.Background()
		// Run SLA check every 5 minutes
		slaTicker := time.NewTicker(5 * time.Minute)
		defer slaTicker.Stop()

		// Run escalation check every 15 minutes
		escalationTicker := time.NewTicker(15 * time.Minute)
		defer escalationTicker.Stop()

		for {
			select {
			case <-slaTicker.C:
				tenants, err := app.DBClient.Tenant.Query().All(ctx)
				if err != nil {
					continue
				}
				for _, t := range tenants {
					_, _ = slaMonitorService.CheckSLAViolations(ctx, t.ID)
				}
			case <-escalationTicker.C:
				tenants, err := app.DBClient.Tenant.Query().All(ctx)
				if err != nil {
					continue
				}
				for _, t := range tenants {
					_ = escalationService.ProcessEscalations(ctx, t.ID)
				}
			}
		}
	}()
}
