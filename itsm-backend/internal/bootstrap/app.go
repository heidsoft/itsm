package bootstrap

import (
	"context"
	"fmt"
	"itsm-backend/config"
	"itsm-backend/controller"
	"itsm-backend/database"
	"itsm-backend/ent"
	"itsm-backend/handlers"
	"itsm-backend/internal/domain/ai" // Added AI domain import
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
	"itsm-backend/pkg/seeder"
	"itsm-backend/router"
	"itsm-backend/service"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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
	logger, _ := zap.NewProduction()
	sugar := logger.Sugar()
	middleware.SetLogger(sugar)

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
	ticketService := service.NewTicketService(client, sugar)
	serviceCatalogService := service.NewServiceCatalogService(client, sugar)
	// 审批服务
	approvalService := service.NewApprovalService(client, sugar)
	serviceRequestService := service.NewServiceRequestService(client, sugar, approvalService)

	cmdbService := service.NewCMDBService(client, sugar)
	problemService := service.NewProblemService(client, sugar)
	changeService := service.NewChangeService(client, sugar)
	changeApprovalService := service.NewChangeApprovalService(client, database.GetRawDB(), sugar)

	// LLM/Embedding/VectorStore
	var embedder service.Embedder
	if cfg.LLM.Provider == "openai" || cfg.LLM.Provider == "" {
		embedder = service.NewOpenAIEmbedderWithConfig(cfg.LLM.APIKey, cfg.LLM.Endpoint, cfg.LLM.Model)
	} else {
		embedder = service.NewOpenAIEmbedder()
	}
	vectorStore := service.NewVectorStore(database.GetRawDB())
	ragService := service.NewRAGService(client, vectorStore, embedder)
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
	ticketRatingService := service.NewTicketRatingService(client, sugar)
	ticketRatingController := controller.NewTicketRatingController(ticketRatingService, sugar)
	ticketViewService := service.NewTicketViewService(client, sugar)
	ticketViewController := controller.NewTicketViewController(ticketViewService, sugar)

	ticketAssignmentService := service.NewTicketAssignmentService(client)
	ticketAssignmentRuleService := service.NewTicketAssignmentRuleService(client, sugar)
	ticketAssignmentSmartService := service.NewTicketAssignmentSmartService(client, sugar, ticketAssignmentService, ticketAssignmentRuleService)
	ticketAssignmentSmartController := controller.NewTicketAssignmentSmartController(ticketAssignmentSmartService, ticketAssignmentRuleService, sugar)

	// Set notification service dependencies
	ticketService.SetNotificationService(ticketNotificationService)
	ticketCommentService.SetNotificationService(ticketNotificationService)
	ticketRatingService.SetNotificationService(ticketNotificationService)

	incidentController := controller.NewIncidentController(incidentService, incidentRuleEngine, incidentMonitoringService, incidentAlertingService, sugar)
	approvalController := controller.NewApprovalController(approvalService)

	serviceController := controller.NewServiceController(serviceCatalogService, serviceRequestService)
	provisioningService := service.NewProvisioningService(client, sugar)
	provisioningController := controller.NewProvisioningController(provisioningService)

	problemController := controller.NewProblemController(sugar, problemService)
	changeController := controller.NewChangeController(sugar, changeService)
	changeApprovalController := controller.NewChangeApprovalController(changeApprovalService, sugar)

	projectController := controller.NewProjectController(client)
	applicationController := controller.NewApplicationController(client)

	ticketCategoryService := service.NewTicketCategoryService(client)
	ticketCategoryController := controller.NewTicketCategoryController(ticketCategoryService, sugar)
	ticketTagService := service.NewTicketTagService(client)
	ticketTagController := controller.NewTicketTagController(ticketTagService, sugar.Desugar())

	processEngine := service.NewCustomProcessEngine(client, sugar)
	bpmnWorkflowController := controller.NewBPMNWorkflowController(processEngine)

	dashboardService := service.NewDashboardService(client, sugar)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService, ticketService, incidentService, sugar)

	// Domain: Service Catalog (DDD)
	scRepo := service_catalog.NewEntRepository(client)
	scService := service_catalog.NewService(scRepo, sugar)
	scHandler := service_catalog.NewHandler(scService)

	// Domain: Service Request (DDD)
	srRepo := service_request.NewEntRepository(client)
	srService := service_request.NewService(srRepo, scRepo, sugar)
	srHandler := service_request.NewHandler(srService)

	// Domain: Incident (DDD)
	incRepo := incident.NewEntRepository(client)
	incService := incident.NewService(incRepo, sugar)
	incHandler := incident.NewHandler(incService)

	// Domain: Problem (DDD)
	problemRepo := problem.NewEntRepository(client)
	problemServiceDomain := problem.NewService(problemRepo, sugar)
	problemHandler := problem.NewHandler(problemServiceDomain)

	// Domain: Change (DDD)
	changeRepo := change.NewEntRepository(client, database.GetRawDB())
	changeServiceDomain := change.NewService(changeRepo, sugar)
	changeHandler := change.NewHandler(changeServiceDomain)

	// Domain: CMDB (DDD)
	cmdbRepo := cmdb.NewEntRepository(client)
	cmdbServiceDomain := cmdb.NewService(cmdbRepo, sugar)
	cmdbHandler := cmdb.NewHandler(cmdbServiceDomain)

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

	// 7. 设置路由
	r := gin.Default()
	if err := r.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		sugar.Warnw("failed to set trusted proxies, falling back to default", "error", err)
	}
	routerConfig := &router.RouterConfig{
		JWTSecret:                       cfg.JWT.Secret,
		Logger:                          sugar,
		Client:                          client,
		TicketController:                ticketController,
		TicketDependencyController:      ticketDependencyController,
		TicketCommentController:         ticketCommentController,
		TicketAttachmentController:      ticketAttachmentController,
		TicketNotificationController:    ticketNotificationController,
		TicketRatingController:          ticketRatingController,
		TicketAssignmentSmartController: ticketAssignmentSmartController,
		TicketViewController:            ticketViewController,
		IncidentController:              incidentController,
		ApprovalController:              approvalController,
		BPMNWorkflowController:          bpmnWorkflowController,
		DashboardHandler:                dashboardHandler,
		ProjectController:               projectController,
		ApplicationController:           applicationController,
		TicketCategoryController:        ticketCategoryController,
		TicketTagController:             ticketTagController,

		// Additional controllers
		ServiceController:        serviceController,
		ProvisioningController:   provisioningController,
		ProblemController:        problemController,
		ChangeController:         changeController,
		ChangeApprovalController: changeApprovalController,

		// Domain Handlers
		ServiceCatalogHandler: scHandler,
		ServiceRequestHandler: srHandler,
		IncidentHandler:       incHandler,
		ProblemHandler:        problemHandler,
		ChangeHandler:         changeHandler,
		CMDBHandler:           cmdbHandler,
		KnowledgeHandler:      knowledgeHandler,
		SLAHandler:            slaHandler,
		AIHandler:             aiHandler, // Added AI domain handler
		CommonHandler:         commonHandler,
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
}
