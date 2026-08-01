package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/common/tenantctx"
	"itsm-backend/config"
	"itsm-backend/connector"
	_ "itsm-backend/connector/builtin/console"
	_ "itsm-backend/connector/builtin/dingtalk"
	_ "itsm-backend/connector/builtin/feishu"
	_ "itsm-backend/connector/builtin/webhook"
	_ "itsm-backend/connector/builtin/wecom"
	"itsm-backend/connector/marketplace"
	"itsm-backend/controller"
	marketplaceController "itsm-backend/controller/marketplace"
	"itsm-backend/pkg/eventbus"
	marketplaceService "itsm-backend/service/marketplace"

	"itsm-backend/database"
	"itsm-backend/docs"
	"itsm-backend/ent"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	"itsm-backend/handlers"
	"itsm-backend/handlers/ai"
	"itsm-backend/handlers/change"
	"itsm-backend/handlers/cmdb"
	domainCommon "itsm-backend/handlers/common"
	"itsm-backend/handlers/incident"
	"itsm-backend/handlers/knowledge"
	"itsm-backend/handlers/known_error"
	"itsm-backend/handlers/problem"
	"itsm-backend/handlers/service_catalog"
	"itsm-backend/handlers/service_request"
	"itsm-backend/handlers/sla"
	"itsm-backend/handlers/standard_change"
	"itsm-backend/internal/initialization"
	"itsm-backend/middleware"
	"itsm-backend/migration"
	"itsm-backend/pkg/seeder"
	repository_ticket "itsm-backend/repository/ticket"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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

// prepareRolePermissionTenantMigration upgrades installations created before
// role_permissions became tenant-scoped. Ent cannot add a required column to a
// populated table directly, so the compatibility step adds it as nullable and
// derives each value from the authoritative roles table first. Ent then applies
// the final NOT NULL contract in Schema.Create.
func prepareRolePermissionTenantMigration(
	ctx context.Context,
	db *sql.DB,
	logger *zap.SugaredLogger,
) error {
	if db == nil {
		return nil
	}

	var tableExists bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = 'role_permissions'
		)
	`).Scan(&tableExists); err != nil {
		return fmt.Errorf("inspect role_permissions table: %w", err)
	}
	if !tableExists {
		return nil
	}

	var columnExists bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'role_permissions'
			  AND column_name = 'tenant_id'
		)
	`).Scan(&columnExists); err != nil {
		return fmt.Errorf("inspect role_permissions.tenant_id: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin role_permissions tenant migration: %w", err)
	}
	defer tx.Rollback()

	if !columnExists {
		if _, err := tx.ExecContext(ctx,
			`ALTER TABLE role_permissions ADD COLUMN tenant_id BIGINT`); err != nil {
			return fmt.Errorf("add role_permissions.tenant_id: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE role_permissions AS rp
		SET tenant_id = r.tenant_id
		FROM roles AS r
		WHERE rp.role_id = r.id
		  AND rp.tenant_id IS NULL
	`); err != nil {
		return fmt.Errorf("backfill role_permissions.tenant_id: %w", err)
	}

	var unresolved int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM role_permissions WHERE tenant_id IS NULL`,
	).Scan(&unresolved); err != nil {
		return fmt.Errorf("verify role_permissions.tenant_id: %w", err)
	}
	if unresolved > 0 {
		return fmt.Errorf(
			"cannot enforce role_permissions.tenant_id: %d rows have no matching tenant-scoped role",
			unresolved,
		)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit role_permissions tenant migration: %w", err)
	}
	logger.Infow("role permission tenant migration prepared", "column_existed", columnExists)
	return nil
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
	LogDefaultCredentialRisks(
		GuardRuntimeCredentials(cfg.Deployment.Mode, cfg.JWT.Secret, cfg.Database.Password),
		sugar,
	)

	// 3. 生产权限必须以数据库为唯一事实来源并在缺失时 fail closed。
	// 只有显式 development/test/local 环境允许使用开发期硬编码回退。
	configurePermissionMode(os.Getenv("ENV"))

	if err := ValidateWebStartupConfig(cfg); err != nil {
		log.Fatalf("Unsafe web startup configuration: %v", err)
	}

	// 3. 初始化数据库连接（带 RLS 装饰器，默认 off 模式=透明）
	client, err := database.InitDatabaseWithRLS(&cfg.Database, &cfg.RLS, sugar)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

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

	// 初始化 EventBus 事件总线
	eventBus, err := eventbus.NewWatermillEventBus(&cfg.Redis, sugar)
	if err != nil {
		sugar.Fatalw("Failed to initialize event bus", "error", err)
	}
	eventbus.SetGlobalEventBus(eventBus)
	sugar.Infow("Event bus initialized successfully")

	// BPMN 子服务（必须在 TicketService 之前创建）
	processBindingService := service.NewProcessBindingService(client)
	processEngine := service.NewCustomProcessEngine(client, sugar)
	processTriggerService := service.NewProcessTriggerService(client, processEngine)
	processResolver := service.NewProcessResolver(client, processBindingService)
	bpmnVersionService := service.NewBPMNVersionService(client, sugar)

	// 工单仓储层（V2 Repository 模式）
	ticketRepoImpl := repository_ticket.NewEntRepository(client, sugar)
	// 注入序列服务（用于 Redis 工单号生成）
	ticketRepoImpl.SetSequenceService(sequenceService)
	// 注入原生数据库连接（用于事务性编号生成）
	ticketRepoImpl.SetRawDB(database.GetRawDB())

	// Connector Manager / Registry / Market —— 连接器/插件/技能市场基础设施
	connectorManager := connector.NewManager(connector.Default(), sugar)
	connectorMarket := marketplace.New()
	connectorController := controller.NewConnectorController(connectorManager, connector.Default(), connectorMarket, sugar)

	// 通知 / 审批 / SLA / 自动化 / 序列服务（V2 子服务）
	ticketNotificationService := service.NewTicketNotificationService(client, sugar)
	ticketSLAService := service.NewTicketSLAService(client, sugar)
	ticketAutomationRuleService := service.NewTicketAutomationRuleService(client, sugar)

	// V2 工单服务（构造函数注入）
	ticketService := service.NewTicketService(&service.TicketServiceConfig{
		Repository:            ticketRepoImpl,
		Client:                client,
		Logger:                sugar,
		NotificationService:   ticketNotificationService,
		ApprovalService:       service.NewApprovalService(client, sugar),
		AutomationRuleService: ticketAutomationRuleService,
		SLAService:            ticketSLAService,
		ProcessTriggerService: processTriggerService,
		ProcessResolver:       processResolver,
		ConnectorManager:      connectorManager,
	})
	_ = sequenceService // V2 内部通过 Repository.GenerateTicketNumber 使用 sequence；保留为运行时上下文依赖

	// 为 IncidentService 注入序列服务
	incidentService.SetSequenceService(sequenceService)

	// MSP 服务初始化
	mspAllocationService := service.NewMSPAllocationService(client, sugar)
	mspController := controller.NewMSPController(mspAllocationService, ticketService, sugar)

	serviceCatalogService := service.NewServiceCatalogService(client, sugar)
	// 审批服务
	approvalService := service.NewApprovalService(client, sugar)

	// problemService and changeService removed - using Handlers with domain services instead

	// Release & Asset Management Services
	releaseService := service.NewReleaseService(client, sugar)
	assetService := service.NewAssetService(client, sugar)
	assetLicenseService := service.NewAssetLicenseService(client, sugar)
	// CMDB Services
	ciTypeService := service.NewCITypeService(client, sugar)
	ciAttributeDefinitionService := service.NewCIAttributeDefinitionService(client, sugar)
	ciHistoryService := service.NewCIHistoryService(client, sugar)
	ciTagService := service.NewCITagService(client, sugar)
	configurationItemService := service.NewConfigurationItemService(client, sugar, ciHistoryService, ciTagService)
	ciRelationshipService := service.NewCIRelationshipService(client, sugar)
	importExportService := service.NewCMDBImportExportService(client, sugar, configurationItemService, ciTagService)
	savedViewService := service.NewCMDBSavedViewService(client, sugar)
	// LLM/Embedding/VectorStore
	var embedder service.Embedder
	if cfg.LLM.Provider == "openai" || cfg.LLM.Provider == "" {
		embedder = service.NewOpenAIEmbedderWithConfig(cfg.LLM.APIKey, cfg.LLM.Endpoint, cfg.LLM.Model)
	} else {
		embedder = service.NewOpenAIEmbedder()
	}

	// Create LLM Gateway for AI services
	llmConfig := service.LoadLLMConfig()
	llmProvider := service.NewProviderFromConfig(llmConfig)
	// Token limiter guards against runaway prompt cost. Default 4000 rune-tokens/request
	// (roughly matches most model context windows). Override via llm.token_cap.
	tokenCap := llmConfig.TokenCap
	if tokenCap <= 0 {
		tokenCap = 4000
	}
	llmLimiter := service.NewFixedWindowLimiter(tokenCap)
	sugar.Infow("LLM token limiter wired", "capacity_runes_per_request", tokenCap)
	llmGateway := service.NewLLMGateway(llmProvider, llmLimiter, nil, llmConfig.Provider)

	vectorStore := service.NewVectorStore(database.GetRawDB())
	ragService := service.NewRAGServiceWithAutoConfig(client, vectorStore, embedder, sugar)
	aiTelemetryService := service.NewAITelemetryService(database.GetRawDB())

	// 非阻塞初始化：向量扩展检测与 Embedding 管道预热
	// 如果 pgvector 扩展未就绪，RAG 功能自动降级为关键字搜索
	go func() {
		ctx := context.Background()
		if err := vectorStore.EnsureExtension(ctx); err != nil {
			sugar.Warnw("pgvector 扩展未就绪，RAG功能降级为关键字搜索", "error", err)
			return
		}
		sugar.Infow("pgvector 扩展初始化成功")
	}()

	// 控制器依赖
	incidentRuleEngine := service.NewIncidentRuleEngine(client, sugar)
	incidentService.SetRuleEngine(incidentRuleEngine)
	incidentMonitoringService := service.NewIncidentMonitoringService(client, sugar)
	incidentAlertingService := service.NewIncidentAlertingService(client, sugar)
	ticketDependencyService := service.NewTicketDependencyService(client, sugar)
	analyticsService := service.NewAnalyticsService(client, sugar)
	predictionService := service.NewPredictionService(client, sugar)
	slaForecastSkill := service.NewSLAForecastSkill(client, llmGateway, sugar)
	// 市场服务
	marketplaceSvc := marketplaceService.NewService(client, sugar)
	marketplaceCtrl := marketplaceController.NewController(marketplaceSvc, connectorManager)

	// Guidance sidecar for constrained JSON generation
	guidanceURL := os.Getenv("GUIDANCE_URL")
	if guidanceURL == "" {
		guidanceURL = "http://localhost:8091"
	}
	guidanceClient := service.NewGuidanceClient(guidanceURL, sugar)
	triageService := service.NewTriageServiceWithGuidanceAndSugaredLogger(llmGateway, guidanceClient, sugar)

	rootCauseService := service.NewRootCauseService(client, sugar)
	// LLM/Embedding/VectorStore

	// AI Tools
	toolRegistry := service.NewToolRegistry(ragService, incidentService, configurationItemService, client)
	toolQueue := service.NewToolQueue(client, toolRegistry, 100, sugar)

	ticketController := controller.NewTicketController(ticketService, ticketDependencyService, database.GetRawDB(), sugar)
	ticketDependencyController := controller.NewTicketDependencyController(ticketDependencyService)

	ticketCommentService := service.NewTicketCommentService(client, sugar)
	ticketCommentController := controller.NewTicketCommentController(ticketCommentService, sugar)
	ticketAttachmentService := service.NewTicketAttachmentService(client, sugar)
	ticketAttachmentController := controller.NewTicketAttachmentController(ticketAttachmentService, sugar)
	ticketNotificationController := controller.NewTicketNotificationController(ticketNotificationService, sugar)
	// ticketNotificationService 已在 128 行创建并注入到 V2

	// General Notification Service & Controller
	notificationService := service.NewNotificationService(client)
	notificationController := controller.NewNotificationController(notificationService)

	// Notification Preference Service & Controller
	notificationPreferenceService := service.NewNotificationPreferenceService(client, sugar)
	notificationPreferenceController := controller.NewNotificationPreferenceController(notificationPreferenceService, sugar)

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
	ticketWorkflowService := service.NewTicketWorkflowService(client, sugar)
	ticketWorkflowService.SetConnectorManager(connectorManager)
	ticketWorkflowController := controller.NewTicketWorkflowController(ticketWorkflowService, database.GetRawDB(), sugar)

	// Ticket Automation Rule Controller (service 已于 131 行预创建并注入 V2)
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
	// CMDB Controller
	cmdbController := controller.NewCMDBController(sugar, ciTypeService, ciAttributeDefinitionService, configurationItemService, ciRelationshipService, ciHistoryService, ciTagService, importExportService, savedViewService)

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

	bpmnWorkflowController := controller.NewBPMNWorkflowController(processEngine, bpmnVersionService)
	bpmnTemplateService := service.NewBPMNTemplateService(client)

	// BPMN Process Trigger Controller (processBindingService/processTriggerService 已于 119-122 行预创建并注入 V2)
	configInheritanceService := service.NewConfigInheritanceService(client, sugar)
	bpmnProcessTriggerController := controller.NewBPMNProcessTriggerController(processTriggerService, processBindingService, configInheritanceService)

	// BPMN Dashboard Controller (监控仪表盘)
	bpmnMetricsService := service.NewBPMNMetricsService(client, sugar)
	bpmnAuditService := service.NewBPMNAuditService(client, sugar)
	bpmnTenantService := service.NewBPMNTenantService(client, sugar)
	bpmnSlaService := service.NewBPMNSLAService(client, sugar)
	bpmnDashboardController := controller.NewBPMNDashboardController(bpmnMetricsService, bpmnAuditService, bpmnTenantService, bpmnSlaService)

	// BPMN Monitoring Service & Controller（监控 + 完整执行轨迹时间线）
	bpmnMonitoringService := service.NewBPMNMonitoringService(client, bpmnAuditService, sugar)
	bpmnMonitoringController := controller.NewBPMNMonitoringController(bpmnMonitoringService)
	// BPMN AI Generator Service & Controller (AI驱动的流程生成)
	bpmnDeploymentService := service.NewBPMNDeploymentService(client)
	bpmnAIGeneratorService := service.NewBPMNAIGeneratorService(llmGateway, bpmnDeploymentService)
	bpmnAIGeneratorController := controller.NewBPMNAIGeneratorController(bpmnAIGeneratorService)

	// A2UI Ticket Controller (AI-driven UI表单)
	a2uiTicketService := service.NewA2UITicketService(nil)
	a2uiTicketController := controller.NewA2UITicketController(a2uiTicketService)

	// Global Search Controller (全局搜索)
	globalSearchController := controller.NewGlobalSearchController(client)

	// Standard Change Handler (标准变更模板库)
	standardChangeHandler := standard_change.NewHandler(client, sugar)

	// Known Error Handler (KEDB)
	knownErrorHandler := known_error.NewHandler(client, sugar)

	// Connector Manager / Registry / Market —— 连接器/插件/技能市场基础设施
	// Feishu 连接器控制器
	feishuSyncService := service.NewFeishuSyncService(client, sugar)
	feishuController := controller.NewFeishuController(connectorManager, feishuSyncService, marketplaceSvc, sugar)

	// Set process trigger service for workflow integration (after processTriggerService is declared)
	ticketService.SetProcessTriggerService(processTriggerService)
	incidentService.SetProcessTriggerService(processTriggerService)

	// Set approval service for ticket workflow integration
	ticketService.SetApprovalService(approvalService)

	// 初始化模板并部署默认流程
	go func() {
		ctx := context.Background()
		const defaultTenantID = 1
		if _, err := bpmnTemplateService.LoadAndDeployTemplates(ctx, defaultTenantID); err != nil {
			sugar.Warnw("Failed to deploy BPMN templates", "error", err)
		}
		if err := processBindingService.InitDefaultBindings(ctx, defaultTenantID); err != nil {
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

	// Domain: Service Request (DDD)
	srRepo := service_request.NewEntRepository(client)
	srService := service_request.NewService(srRepo, scRepo, cmdbRepo, client, sugar)
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
	changeServiceDomain := change.NewService(changeRepo, client, sugar)
	changeHandler := change.NewHandler(changeServiceDomain)

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

	// SLA 模板服务（开箱即用）
	slaTemplateService := service.NewSLATemplateService(client, sugar)
	slaTemplateController := controller.NewSLATemplateController(slaTemplateService)

	// AI Domain
	aiRepo := ai.NewEntRepository(client)
	aiServiceDomain := ai.NewService(aiRepo, sugar, ragService, toolRegistry, toolQueue, analyticsService, predictionService, slaForecastSkill, triageService, rootCauseService, aiTelemetryService)
	aiServiceDomain.SetLLMGateway(llmGateway)
	// P2-6: 注入 ent client 供 AI 工具 RBAC 校验复用 hasResourcePermission
	aiServiceDomain.SetEntClient(client)
	aiHandler := ai.NewHandler(aiServiceDomain)

	// Common Domain
	commonRepo := domainCommon.NewEntRepository(client)
	commonServiceDomain := domainCommon.NewService(commonRepo, cfg.JWT.Secret, sugar, client)
	// 注入 Redis 客户端（如果可用），启用 refresh token 黑名单
	if cfg.Redis.Host != "" {
		commonRedis := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := commonRedis.Ping(pingCtx).Err(); err != nil {
			sugar.Warnw("common domain redis ping failed; refresh token blacklist disabled", "error", err)
		} else {
			commonServiceDomain.SetRedis(commonRedis)
			middleware.ConfigureAccessTokenRevocationRedis(commonRedis)
			sugar.Info("refresh token blacklist enabled via redis")
		}
		pingCancel()
	}
	commonHandler := domainCommon.NewHandler(commonServiceDomain)

	// Auth Controller（装配缺失的 register / forgot-password / reset-password / validate-reset-token / switch-tenant 路由）
	authService := service.NewAuthService(client, cfg.JWT.Secret, sugar, nil)
	authController := controller.NewAuthController(authService)

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

	// Audit Log Controller (支持过滤/分页的审计日志查询)
	auditLogService := service.NewAuditLogService(client, sugar)
	auditLogController := controller.NewAuditLogController(auditLogService, sugar)

	// Tenant Controller
	tenantService := service.NewTenantService(client, sugar)
	tenantController := controller.NewTenantController(tenantService, sugar)

	// System Config Controller
	systemConfigService := service.NewSystemConfigService(client, sugar)
	systemConfigController := controller.NewSystemConfigController(systemConfigService, sugar)

	// Vendor Controller
	vendorService := service.NewVendorService(client, sugar)
	vendorController := controller.NewVendorController(vendorService)

	// Approval Chain Controller
	approvalChainService := service.NewApprovalChainService(client, sugar)
	approvalChainController := controller.NewApprovalChainController(approvalChainService, sugar)

	// SLA Monitor & Alert Services (legacy, for background tasks)
	slaMonitorService := service.NewSLAMonitorService(client, sugar)
	slaAlertService := service.NewSLAAlertService(client, sugar)
	escalationService := service.NewEscalationService(client, sugar)
	escalationMatrixService := service.NewEscalationMatrixService(sugar)
	escalationMatrixController := controller.NewEscalationMatrixController(sugar, escalationMatrixService)

	// Wire up notification service
	slaMonitorService.SetNotificationService(ticketNotificationService)
	slaAlertService.SetNotificationService(ticketNotificationService)
	escalationService.SetNotificationService(ticketNotificationService)

	// Survey Service & Controller
	surveyService := service.NewSurveyService(client, sugar)
	surveyController := controller.NewSurveyController(surveyService)

	// Cloud Service & Controller
	cloudService := service.NewCloudService(client, sugar)
	cloudController := controller.NewCloudController(cloudService, sugar)

	// WebSocket Service
	wsService := service.NewWebSocketService(sugar)

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
		RawDB:                           database.GetRawDB(),
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
		BPMNMonitoringController:        bpmnMonitoringController,
		BPMNAIGeneratorController:       bpmnAIGeneratorController,
		A2UITicketController:            a2uiTicketController,
		CMDBController:                  cmdbController,

		DashboardHandler:         dashboardHandler,
		CMDBHandler:              cmdbHandler,
		ProjectController:        projectController,
		ApplicationController:    applicationController,
		TicketCategoryController: ticketCategoryController,
		TicketTagController:      ticketTagController,
		UserController:           userController,
		GroupController:          groupController,

		// Role & Permission Controllers
		RoleController:             roleController,
		PermissionController:       permissionController,
		MenuController:             menuController,
		TenantController:           tenantController,
		EscalationMatrixController: escalationMatrixController,
		AuditLogController:         auditLogController,

		MSPController:           mspController,
		SystemConfigController:  systemConfigController,
		ApprovalChainController: approvalChainController,

		// Notification Preference Controller
		NotificationPreferenceController: notificationPreferenceController,

		// Vendor Controller
		VendorController: vendorController,

		// Additional controllers
		ServiceController:      serviceController,
		ProvisioningController: provisioningController,
		AnalyticsController:    analyticsController,
		PredictionController:   predictionController,
		ReleaseController:      releaseController,
		AssetController:        assetController,
		AssetLicenseController: assetLicenseController,
		SurveyController:       surveyController,
		CloudController:        cloudController,

		// Domain Handlers
		ServiceCatalogHandler: scHandler,
		ServiceRequestHandler: srHandler,
		ProblemHandler:        problemHandler,
		ChangeHandler:         changeHandler,
		KnowledgeHandler:      knowledgeHandler,
		SLAHandler:            slaHandler,
		SLATemplateController: slaTemplateController,
		AIHandler:             aiHandler, // Added AI domain handler
		CommonHandler:         commonHandler,
		AuthController:        authController,
		RoleHandler:           roleHandler,

		// Global Search
		GlobalSearchController: globalSearchController,

		// Standard Change Handler
		StandardChangeHandler: standardChangeHandler,

		// Known Error Handler (KEDB)
		KnownErrorHandler: knownErrorHandler,

		// Connector Controller
		ConnectorController: connectorController,
		FeishuController:    feishuController,

		MarketplaceController: marketplaceCtrl,

		// WebSocket Service
		WebSocketService: wsService,
	}
	router.SetupRoutes(r, routerConfig)

	// Swagger - configure and register swagger docs
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = ""
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
	docs.SwaggerInfo.Title = "ITSM API"
	docs.SwaggerInfo.Description = "IT Service Management System API Documentation"
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

func configurePermissionMode(environment string) {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "development", "dev", "test", "local":
		middleware.PermissionConfig.Mode = middleware.PermissionConfigModeFallback
	default:
		middleware.PermissionConfig.Mode = middleware.PermissionConfigModeDBOnly
	}
}

// ValidateWebStartupConfig prevents schema or seed mutations from running in
// the long-lived HTTP process. Deployments must execute them through the
// explicit ITSM_BOOTSTRAP_ONLY job before starting application instances.
func ValidateWebStartupConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration is required")
	}
	if cfg.Deployment.AutoMigrate || cfg.Deployment.AutoSeed {
		return fmt.Errorf(
			"ITSM_AUTO_MIGRATE and ITSM_AUTO_SEED are bootstrap-job options; run with ITSM_BOOTSTRAP_ONLY=true",
		)
	}
	return nil
}

func InitializeStorage(cfg *config.Config, client *ent.Client, sugar *zap.SugaredLogger) error {
	// RLS：schema 创建 / seed / DDL 属于跨租户操作，必须显式声明 system bypass
	ctx := tenantctx.SystemContext(context.Background(), "bootstrap:initialize_storage",
		"schema migration and default seed at process boot")

	if cfg.Deployment.AutoMigrate {
		if err := prepareRolePermissionTenantMigration(ctx, database.GetRawDB(), sugar); err != nil {
			return fmt.Errorf("prepare role permission tenant migration: %w", err)
		}
		if err := prepareCMDBModelMigration(ctx, database.GetRawDB(), sugar); err != nil {
			return fmt.Errorf("prepare CMDB model migration: %w", err)
		}
		if err := client.Schema.Create(ctx); err != nil {
			return fmt.Errorf("create schema resources: %w", err)
		}
		migrator := migration.NewMigrator(database.GetRawDB(), sugar)
		if err := runPostSchemaMigrations(ctx, migrator); err != nil {
			return fmt.Errorf("apply versioned post-schema migrations: %w", err)
		}
		sugar.Infow("database schema ensured", "deployment_mode", cfg.Deployment.Mode)
	}

	if cfg.Deployment.AutoSeed {
		needsAdmin, err := needsBootstrapAdmin(ctx, client)
		if err != nil {
			return fmt.Errorf("check bootstrap administrator: %w", err)
		}
		if needsAdmin {
			for _, risk := range GuardBootstrapAdminCredentials(
				cfg.Deployment.Mode,
				os.Getenv("ADMIN_PASSWORD"),
			) {
				if risk.Severity == "fatal" {
					return fmt.Errorf("bootstrap credential rejected [%s]: %s", risk.Code, risk.Message)
				}
				sugar.Warnw("bootstrap credential risk detected", "code", risk.Code, "message", risk.Message)
			}
		}
		s := seeder.NewSeeder(client, sugar, cfg)
		components, err := seeder.ProductionInitializers(s)
		if err != nil {
			return fmt.Errorf("create production initializers: %w", err)
		}
		store, err := initialization.NewSQLStore(database.GetRawDB())
		if err != nil {
			return fmt.Errorf("create initialization store: %w", err)
		}
		engine, err := initialization.NewEngine(
			store,
			components,
			30*time.Second,
		)
		if err != nil {
			return fmt.Errorf("create initialization engine: %w", err)
		}
		executorID, err := os.Hostname()
		if err != nil {
			executorID = "bootstrap-job"
		}
		executorID, err = initialization.NewExecutorID(executorID)
		if err != nil {
			return fmt.Errorf("create initialization executor id: %w", err)
		}
		releaseVersion := strings.TrimSpace(os.Getenv("ITSM_RELEASE_VERSION"))
		if releaseVersion == "" {
			releaseVersion = "unversioned"
		}
		runID, err := engine.Apply(ctx, initialization.Request{
			Scope:          initialization.Scope{Type: "platform", ID: 0},
			TargetVersion:  seeder.CurrentTenantTemplateVersion,
			ReleaseVersion: releaseVersion,
			RequestedBy:    "bootstrap-job",
			ExecutorID:     executorID,
		})
		if err != nil {
			return fmt.Errorf("initialize production defaults (run %d): %w", runID, err)
		}
		sugar.Infow("seed completed", "deployment_mode", cfg.Deployment.Mode, "initialization_run_id", runID)
	}

	return nil
}

type postSchemaMigrator interface {
	EnsureMigrationsTable(context.Context) error
	RunMigrations(context.Context, []migration.Migration) (int, error)
}

func runPostSchemaMigrations(ctx context.Context, migrator postSchemaMigrator) error {
	if migrator == nil {
		return fmt.Errorf("migration runner is required")
	}
	if err := migrator.EnsureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("ensure migration ledger: %w", err)
	}
	if _, err := migrator.RunMigrations(ctx, migration.PostSchemaMigrations()); err != nil {
		return fmt.Errorf("run post-schema migrations: %w", err)
	}
	return nil
}

func RunInitialization() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := initLogger(&cfg.Log)
	defer func() {
		_ = logger.Sync()
	}()

	sugar := logger.Sugar()
	LogDefaultCredentialRisks(
		GuardRuntimeCredentials(cfg.Deployment.Mode, cfg.JWT.Secret, cfg.Database.Password),
		sugar,
	)
	client, err := database.InitDatabaseWithRLS(&cfg.Database, &cfg.RLS, sugar)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer client.Close()

	if err := InitializeStorage(cfg, client, sugar); err != nil {
		log.Fatalf("Initialization failed: %v", err)
	}
}

func needsBootstrapAdmin(ctx context.Context, client *ent.Client) (bool, error) {
	rootTenant, err := client.Tenant.Query().Where(tenant.CodeEQ("default")).Only(ctx)
	if ent.IsNotFound(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	exists, err := client.User.Query().
		Where(user.UsernameEQ("admin"), user.TenantIDEQ(rootTenant.ID)).
		Exist(ctx)
	if err != nil {
		return false, err
	}
	return !exists, nil
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
				if err := pipeline.RunOnce(ctx, t.ID, 200); err != nil {
					app.Logger.Warnw("embedding pipeline failed", "error", err, "tenant_id", t.ID)
				}
			}
		} else {
			// fallback default tenant 1
			if err := pipeline.RunOnce(ctx, 1, 200); err != nil {
				app.Logger.Warnw("embedding pipeline failed", "error", err, "tenant_id", 1)
			}
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
				if err := pipeline.RunOnce(ctx, t.ID, 50); err != nil {
					app.Logger.Warnw("embedding pipeline failed", "error", err, "tenant_id", t.ID)
				}
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
					if _, err := slaMonitorService.CheckSLAViolations(ctx, t.ID); err != nil {
						app.Logger.Warnw("SLA violation check failed", "error", err, "tenant_id", t.ID)
					}
				}
			case <-escalationTicker.C:
				tenants, err := app.DBClient.Tenant.Query().All(ctx)
				if err != nil {
					continue
				}
				for _, t := range tenants {
					if err := escalationService.ProcessEscalations(ctx, t.ID); err != nil {
						app.Logger.Warnw("escalation processing failed", "error", err, "tenant_id", t.ID)
					}
				}
			}
		}
	}()
}
