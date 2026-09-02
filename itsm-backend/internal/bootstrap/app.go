package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"itsm-backend/common"
	"itsm-backend/common/tenantctx"
	"itsm-backend/config"
	"itsm-backend/connector"
	connectorAlert "itsm-backend/connector/alert"
	_ "itsm-backend/connector/builtin/console"
	_ "itsm-backend/connector/builtin/dingtalk"
	_ "itsm-backend/connector/builtin/email"
	_ "itsm-backend/connector/builtin/feishu"
	_ "itsm-backend/connector/builtin/webhook"
	_ "itsm-backend/connector/builtin/wecom"
	"itsm-backend/connector/marketplace"
	connectorVector "itsm-backend/connector/vector"
	connectorHandler "itsm-backend/handlers/connector"
	marketplaceHandler "itsm-backend/handlers/marketplace"
	"itsm-backend/pkg/eventbus"
	marketplaceService "itsm-backend/service/marketplace"

	"itsm-backend/database"
	"itsm-backend/docs"
	"itsm-backend/ent"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	a2uiHandler "itsm-backend/handlers/a2ui"
	"itsm-backend/handlers/ai"
	analyticsHandler "itsm-backend/handlers/analytics"
	"itsm-backend/handlers/approval"
	approvalChainHandler "itsm-backend/handlers/approval_chain"
	assetHandler "itsm-backend/handlers/asset"
	assignmentSmartHandler "itsm-backend/handlers/assignment_smart"
	auditlogHandler "itsm-backend/handlers/auditlog"
	authHandler "itsm-backend/handlers/auth"
	automationRuleHandler "itsm-backend/handlers/automation_rule"
	bpmnHandler "itsm-backend/handlers/bpmn"
	"itsm-backend/handlers/cab"
	"itsm-backend/handlers/change"
	cloudHandler "itsm-backend/handlers/cloud"
	"itsm-backend/handlers/cmdb"
	domainCommon "itsm-backend/handlers/common"
	"itsm-backend/handlers/common/knowledgeaccess"
	"itsm-backend/handlers/email_intake"
	escalationMatrixHandler "itsm-backend/handlers/escalation_matrix"
	feishuHandler "itsm-backend/handlers/feishu"
	globalSearchHandler "itsm-backend/handlers/global_search"
	groupHandler "itsm-backend/handlers/group"
	"itsm-backend/handlers/incident"
	"itsm-backend/handlers/knowledge"
	"itsm-backend/handlers/known_error"
	mspHandler "itsm-backend/handlers/msp"
	notificationHandler "itsm-backend/handlers/notification"
	predictionHandler "itsm-backend/handlers/prediction"
	"itsm-backend/handlers/problem"
	probleminvestigation "itsm-backend/handlers/problem_investigation"
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
	systemconfig "itsm-backend/handlers/systemconfig"
	tenantHandler "itsm-backend/handlers/tenant"
	"itsm-backend/handlers/ticket"
	ticketAttachmentHandler "itsm-backend/handlers/ticket_attachment"
	ticketCategoryHandler "itsm-backend/handlers/ticket_category"
	ticketCommentHandler "itsm-backend/handlers/ticket_comment"
	ticketDependencyHandler "itsm-backend/handlers/ticket_dependency"
	ticketNotificationHandler "itsm-backend/handlers/ticket_notification"
	ticketRatingHandler "itsm-backend/handlers/ticket_rating"
	ticketTagHandler "itsm-backend/handlers/ticket_tag"
	ticketTypeHandler "itsm-backend/handlers/ticket_type"
	ticketViewHandler "itsm-backend/handlers/ticket_view"
	ticketworkflow "itsm-backend/handlers/ticket_workflow"
	userHandler "itsm-backend/handlers/user"
	vectorStoreHandler "itsm-backend/handlers/vector_store"
	vendorHandler "itsm-backend/handlers/vendor"
	"itsm-backend/internal/commandbus"
	"itsm-backend/internal/initialization"
	"itsm-backend/middleware"
	"itsm-backend/migration"
	"itsm-backend/pkg/seeder"
	repository_ticket "itsm-backend/repository/ticket"
	"itsm-backend/router"
	"itsm-backend/service"
	cloudruntime "itsm-backend/service/cloud"
	cloudaliyun "itsm-backend/service/cloud/aliyun"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

type Application struct {
	Cfg               *config.Config
	Logger            *zap.SugaredLogger
	DBClient          *ent.Client
	Router            *gin.Engine
	Embedder          service.Embedder
	VectorStore       connectorVector.VectorStore
	LegacyVectorStore *service.VectorStore
	CommandWorker     *commandbus.Worker
	SkillRegistry     *service.SkillRegistry

	// backgroundWG 跟踪由 startBackgroundTasks 启动的所有后台 goroutine。
	// 在 Stop() 中等待它们退出，避免应用关闭时强制杀死进行中的任务。
	backgroundWG sync.WaitGroup
}

// makeSequenceDBSyncFn 构造 Redis 序列的 DB 播种函数。
// 支持 key 格式：
//   - sequence:ticket:<tenantID>:<YYYYMM>   -> 当月最大 ticket_number 按租户
//   - sequence:incident:<YYYYMM>            -> 当月最大 incident_number 全局（唯一约束不含租户）
//
// 必须使用原生 SQL：
//  1. Ent 全局软删拦截器 (database/softdelete.go) 会给所有 Query 附加
//     DeletedAtIsNil()，但已软删记录的编号仍占用物理唯一约束，播种若忽略
//     它们会撞号（实测复现）；
//  2. 原生 SQL 与唯一约束视角完全一致（含软删记录）。
//
// 返回 DB 当前最大尾号；SequenceService 用 SETNX 播种该值后首次 INCR 即 max+1。
func makeSequenceDBSyncFn(db *sql.DB, logger *zap.SugaredLogger) func(key string) (int64, error) {
	// 尾号解析：取末段 "-" 之后的数字
	parseSeq := func(number string) int64 {
		if idx := strings.LastIndex(number, "-"); idx >= 0 && idx+1 < len(number) {
			var seq int64
			if _, err := fmt.Sscanf(number[idx+1:], "%d", &seq); err == nil {
				return seq
			}
		}
		return 0
	}

	queryMax := func(query string, args ...interface{}) (int64, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var maxNum string
		err := db.QueryRowContext(ctx, query, args...).Scan(&maxNum)
		if err != nil {
			if sql.ErrNoRows == err {
				return 0, nil
			}
			return 0, err
		}
		if maxNum == "" {
			return 0, nil
		}
		return parseSeq(maxNum), nil
	}

	return func(key string) (int64, error) {
		// 注意：fmt.Sscanf 不支持 %04d 宽度语义（%d 会贪婪吞掉整个 202608），
		// 必须按固定长度手动切分
		if strings.HasPrefix(key, "sequence:incident:") {
			ym := strings.TrimPrefix(key, "sequence:incident:")
			if len(ym) != 6 {
				return 0, fmt.Errorf("invalid incident sequence key: %s", key)
			}
			year, month := ym[:4], ym[4:]
			prefix := fmt.Sprintf("INC-%s%s-", year, month)
			// incident_number 为全局唯一约束，跨租户取最大；原生 SQL 含软删记录
			return queryMax(
				`SELECT incident_number FROM incidents `+
					`WHERE incident_number LIKE $1 AND incident_number IS NOT NULL AND incident_number != '' `+
					`ORDER BY incident_number DESC LIMIT 1`,
				prefix+"%",
			)
		}

		// sequence:ticket:<tenantID>:<YYYYMM>
		if parts := strings.Split(key, ":"); len(parts) == 4 && parts[0] == "sequence" && parts[1] == "ticket" {
			tenantID, ym := parts[2], parts[3]
			if len(ym) != 6 {
				return 0, fmt.Errorf("invalid ticket sequence key: %s", key)
			}
			prefix := fmt.Sprintf("TKT-%s%s-", ym[:4], ym[4:])
			tid, err := strconv.Atoi(tenantID)
			if err != nil {
				return 0, fmt.Errorf("invalid tenant id in sequence key: %s", key)
			}
			return queryMax(
				`SELECT ticket_number FROM tickets `+
					`WHERE tenant_id = $1 AND ticket_number LIKE $2 AND ticket_number IS NOT NULL AND ticket_number != '' `+
					`ORDER BY ticket_number DESC LIMIT 1`,
				tid, prefix+"%",
			)
		}

		logger.Warnw("Unsupported sequence key, skip DB sync", "key", key)
		return 0, fmt.Errorf("unsupported sequence key: %s", key)
	}
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
	ticketSLAService := service.NewTicketSLAService(client, sugar)
	auditLogService := service.NewAuditLogService(client, sugar)
	ticketTypeService := service.NewTicketTypeService(client, sugar)
	ticketTagService := service.NewTicketTagService(client)
	surveyService := service.NewSurveyService(client, sugar)
	cloudService := service.NewCloudService(client, sugar)
	slaTemplateService := service.NewSLATemplateService(client, sugar)
	ticketDependencyService := service.NewTicketDependencyService(client, sugar)
	ticketCommentService := service.NewTicketCommentService(client, sugar)
	ticketAttachmentService := service.NewTicketAttachmentService(client, sugar)
	ticketNotificationService := service.NewTicketNotificationService(client, sugar)
	ticketRatingService := service.NewTicketRatingService(client, sugar)
	ticketViewService := service.NewTicketViewService(client, sugar)
	ticketAssignmentRuleService := service.NewTicketAssignmentRuleService(client, sugar)
	ticketAssignmentService := service.NewTicketAssignmentService(client, sugar)
	ticketAssignmentSmartService := service.NewTicketAssignmentSmartService(client, sugar, ticketAssignmentService, ticketAssignmentRuleService)
	incidentService := service.NewIncidentService(client, sugar, ticketSLAService)
	incidentMonitoringService := service.NewIncidentMonitoringService(client, sugar)
	incidentAlertingService := service.NewIncidentAlertingService(client, sugar)
	rootCauseAnalysisService := service.NewRootCauseAnalysisService(client)
	incidentRepo := incident.NewEntRepository(client)
	incidentHandlerService := incident.NewService(incidentRepo, incidentService, incidentMonitoringService, incidentAlertingService, rootCauseAnalysisService, sugar)
	incidentHandler := incident.NewHandler(incidentHandlerService)

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
		// 注册 DB 同步函数：Redis 序列被清空（重启/淘汰）后从 DB 最大编号播种，
		// 避免从 1 重新计数撞上历史唯一编号（S-4 事件编号复用 P0 的装配缺口）。
		// 用独立 *sql.DB + 原生 SQL：需读取含软删记录的物理最大编号，与唯一
		// 约束视角一致，绕过 Ent 软删拦截器
		if seqDB, derr := database.InitDB(&cfg.Database); derr == nil {
			ss.SetDBQueryFunc(makeSequenceDBSyncFn(seqDB, sugar))
			sugar.Infow("Redis sequence service initialized successfully with DB sync")
		} else {
			sugar.Warnw("Sequence DB sync unavailable, will start from 1 on Redis reset", "error", derr)
		}
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
	commandRegistry := commandbus.NewRegistry()
	if err := commandbus.ValidateStorage(context.Background(), client); err != nil {
		sugar.Fatalw("Operational command storage is not ready; run the bootstrap migration first", "error", err)
	}
	workflowCommandHandler := service.NewWorkflowStartCommandHandler(client, processTriggerService, processResolver, sugar)
	if err := commandRegistry.Register(commandbus.CommandStartBPMN, workflowCommandHandler.Handle); err != nil {
		sugar.Fatalw("Failed to register workflow command handler", "error", err)
	}
	customProcessEngine, ok := processEngine.(*service.CustomProcessEngine)
	if !ok {
		sugar.Fatal("Custom BPMN process engine is required for durable ServiceTask execution")
	}
	if err := commandRegistry.Register(commandbus.CommandExecuteBPMNServiceTask, customProcessEngine.HandleBPMNServiceTaskCommand); err != nil {
		sugar.Fatalw("Failed to register BPMN ServiceTask command handler", "error", err)
	}
	workerOwner, _ := os.Hostname()
	if workerOwner == "" {
		workerOwner = "itsm-api"
	}
	commandWorker := commandbus.NewWorker(client, commandRegistry, sugar, workerOwner)
	incidentService.EnableWorkflowOutbox()
	incidentService.EnableRulesOutbox()
	bpmnVersionService := service.NewBPMNVersionService(client, sugar)

	// 工单仓储层（V2 Repository 模式）
	ticketRepoImpl := repository_ticket.NewEntRepository(client, sugar)
	// 注入序列服务（用于 Redis 工单号生成）
	ticketRepoImpl.SetSequenceService(sequenceService)
	// 注入原生数据库连接（用于事务性编号生成）
	ticketRepoImpl.SetRawDB(database.GetRawDB())

	// Connector Manager / Registry / Market —— 连接器/插件/技能市场基础设施
	connectorManager := connector.NewManager(connector.Default(), sugar)
	alertRegistry := connectorAlert.Default()
	alertConfigPath := os.Getenv("ALERT_SOURCE_CONFIG")
	if alertConfigPath == "" {
		alertConfigPath = "etc/alert-sources/prometheus-alertmanager.yaml"
	}
	if alertConfig, loadErr := connectorAlert.LoadConfigFile(alertConfigPath); loadErr != nil {
		sugar.Warnw("Alert source config was not loaded", "path", alertConfigPath, "error", loadErr)
	} else if alertConfig.Enabled {
		if _, exists := alertRegistry.Get(alertConfig.Source); !exists {
			alertRegistry.Register(func() connectorAlert.AlertSource {
				return connectorAlert.NewWebhookAlertSource(alertConfig)
			})
		}
	}
	connectorMarket := marketplace.New()
	connectorHandler := connectorHandler.NewHandler(connectorManager, connector.Default(), connectorMarket, sugar)
	alertHandler := connectorAlert.NewHandler(alertRegistry, connectorManager, database.GetRawDB(), alertDevelopmentMode())
	connectorEncryptionKey := os.Getenv("CONNECTOR_CONFIG_ENCRYPTION_KEY")
	if connectorEncryptionKey == "" {
		if os.Getenv("ENV") == "production" || os.Getenv("GIN_MODE") == "release" {
			log.Fatal("CONNECTOR_CONFIG_ENCRYPTION_KEY is required in production")
		}
		connectorEncryptionKey = "development-connector-key-" + cfg.JWT.Secret
		sugar.Warn("CONNECTOR_CONFIG_ENCRYPTION_KEY is not set; development fallback uses JWT secret")
	}
	connectorStore, connectorStoreErr := connector.NewPersistentConfigStore(client, connectorEncryptionKey)
	if connectorStoreErr != nil {
		sugar.Fatalw("Failed to initialize connector config store", "error", connectorStoreErr)
	}
	// connector 配置持久化必须注入到实际承接路由的 handler。此前误注入到无路由挂载的
	// controller.ConnectorController，导致 h.store 为 nil，Provision/Revoke 跳过落库，
	// 配置仅存于内存、重启即丢失。
	connectorHandler.SetPersistentStore(connectorStore)
	if persistedConfigs, loadErr := connectorStore.LoadAll(context.Background()); loadErr != nil {
		sugar.Warnw("Failed to reload persisted connector configs", "error", loadErr)
	} else {
		for _, persistedConfig := range persistedConfigs {
			if provisionErr := connectorManager.Provision(context.Background(), persistedConfig); provisionErr != nil {
				sugar.Errorw("Failed to rehydrate connector", "tenant", persistedConfig.TenantID, "name", persistedConfig.Name, "error", provisionErr)
			}
		}
	}
	emailOutboundCommandHandler := email_intake.NewOutboundCommandHandler(client, connectorManager)
	if err := commandRegistry.Register(commandbus.CommandSendIntakeEmail, emailOutboundCommandHandler.Handle); err != nil {
		sugar.Fatalw("Failed to register email intake outbound handler", "error", err)
	}

	// 通知 / 审批 / SLA / 自动化 / 序列服务（V2 子服务）
	notificationCommandHandler := service.NewNotificationDeliveryCommandHandler(client, connectorManager, sugar)
	if err := commandRegistry.Register(commandbus.CommandDeliverNotification, notificationCommandHandler.Handle); err != nil {
		sugar.Fatalw("Failed to register notification command handler", "error", err)
	}
	ticketNotificationService.EnableOutbox()
	// EnableTxOutbox 启用事务入箱：阶段 B（工单创建）/ C（SLA 违规/预警）/ D（变更审批）
	// 三个域下沉时，业务事务内调用 Notify*Tx 才能与主表「同生同死」。未启用时 Tx 方法会
	// fail-closed，避免静默回退到 client 路径产生主表与通知行分离提交的不一致状态。
	ticketNotificationService.EnableTxOutbox()
	ticketAutomationRuleService := service.NewTicketAutomationRuleService(client, sugar)
	ticketAutomationCommandHandler := service.NewTicketAutomationCommandHandler(ticketAutomationRuleService)
	if err := commandRegistry.Register(commandbus.CommandExecuteTicketRules, ticketAutomationCommandHandler.Handle); err != nil {
		sugar.Fatalw("Failed to register ticket automation command handler", "error", err)
	}
	ticketFeishuCommandHandler := service.NewTicketFeishuSyncCommandHandler(client, connectorManager, sugar)
	if err := commandRegistry.Register(commandbus.CommandSyncTicketFeishu, ticketFeishuCommandHandler.Handle); err != nil {
		sugar.Fatalw("Failed to register ticket feishu command handler", "error", err)
	}

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
	ticketService.EnableWorkflowOutbox()
	ticketRepo := ticket.NewEntRepository(ticketRepoImpl)
	ticketHandlerService := ticket.NewService(ticketRepo, ticketService, sugar)
	ticketHandler := ticket.NewHandler(ticketHandlerService)
	ticketService.EnableSideEffectOutbox()
	_ = sequenceService // V2 内部通过 Repository.GenerateTicketNumber 使用 sequence；保留为运行时上下文依赖

	// TicketAssociationService 工单关联服务
	ticketAssociationService := service.NewTicketAssociationService(client)

	// 为 IncidentService 注入序列服务与原生数据库连接（S-4 编号事务锁）
	incidentService.SetSequenceService(sequenceService)
	incidentService.SetRawDB(database.GetRawDB())

	// MSP 服务初始化
	// 审批服务
	approvalService := service.NewApprovalService(client, sugar)
	// 将 ApprovalService 注入 BPMN 引擎的 ApprovalHandler，解决循环依赖
	processEngine.SetApprovalService(approvalService)

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
	if err := commandRegistry.Register(commandbus.CommandProcessCMDBImport, importExportService.HandleImportCommand); err != nil {
		sugar.Fatalw("Failed to register CMDB import command handler", "error", err)
	}
	if err := commandRegistry.Register(commandbus.CommandProcessCMDBExport, importExportService.HandleExportCommand); err != nil {
		sugar.Fatalw("Failed to register CMDB export command handler", "error", err)
	}
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
	// 阻断1 修复：启动期检测 LLM API Key 配置状态。
	// - 占位符/空值：在开发环境 Warn，在生产环境终止启动（生产硬约束见 memory）。
	// - 真实密钥：仅输出 MaskSecret 脱敏值，便于诊断配置是否生效，绝不输出明文。
	if common.IsPlaceholderSecret(llmConfig.APIKey) {
		if os.Getenv("ENV") == "production" || os.Getenv("GIN_MODE") == "release" {
			sugar.Errorw("LLM API Key 未配置或为占位符，生产环境禁止以此状态启动",
				"provider", llmConfig.Provider, "api_key", common.MaskSecret(llmConfig.APIKey))
			// NewApplication 返回 *Application（无 error），生产硬约束用 log.Fatalf 终止。
			log.Fatalf("LLM API Key 未配置：生产环境必须设置真实的 LLM_API_KEY (provider=%s, api_key=%s)",
				llmConfig.Provider, common.MaskSecret(llmConfig.APIKey))
		}
		sugar.Warnw("LLM API Key 未配置或为占位符，AI 功能将降级为禁用",
			"provider", llmConfig.Provider, "api_key", common.MaskSecret(llmConfig.APIKey))
	} else {
		sugar.Infow("LLM API Key 已配置",
			"provider", llmConfig.Provider, "api_key_masked", common.MaskSecret(llmConfig.APIKey))
	}
	llmProvider := service.NewProviderFromConfig(llmConfig)
	// Token limiter guards against runaway prompt cost. Default 4000 rune-tokens/request
	// (roughly matches most model context windows). Override via llm.token_cap.
	tokenCap := llmConfig.TokenCap
	if tokenCap <= 0 {
		tokenCap = 4000
	}
	llmLimiter := service.NewFixedWindowLimiter(tokenCap)
	sugar.Infow("LLM token limiter wired", "capacity_runes_per_request", tokenCap)
	// 网关级可观测性：每次 LLM 调用（成功/限流/失败）都会写入 ai_llm_calls，
	// 供 /api/v1/ai/metrics 输出真实的 avg_response_time_seconds。
	llmObserver := service.NewLLMObserver(database.GetRawDB(), sugar)
	llmGateway := service.NewLLMGateway(llmProvider, llmLimiter, llmObserver, llmConfig.Provider)
	a2uiService := service.NewA2UITicketService(llmGateway)

	vectorStore := service.NewVectorStore(database.GetRawDB())
	ragService := service.NewRAGServiceWithAutoConfig(client, vectorStore, embedder, sugar)
	// 本体增强检索：识别 query 中的业务实体（TKT-/INC-/REL- 编号、CI 名称），
	// 沿 CMDB/工单关系做 1 跳扩展，注入 AI 回答的上下文与引用源。
	ontologyService := service.NewOntologyService(client, sugar)
	ragService.SetOntologyService(ontologyService)
	vectorCtx, vectorCancel := context.WithTimeout(context.Background(), 5*time.Second)
	pluggableVectorStore, vectorErr := connectorVector.NewFromEnvironment(vectorCtx)
	vectorCancel()
	if vectorErr != nil {
		sugar.Warnw("vector store initialization failed; RAG keeps database keyword fallback", "error", vectorErr)
	} else {
		ragService.SetVectorStore(pluggableVectorStore)
		sugar.Infow("pluggable vector store initialized")
	}
	aiTelemetryService := service.NewAITelemetryService(database.GetRawDB())

	// 同步初始化：向量扩展检测与 vectors 表准备。
	// 使用带超时的 context，避免 pgvector 不可用时阻塞整个启动流程。
	// 初始化失败时仅记录告警并继续——RAG 功能会自动降级为关键字搜索。
	// 在 RAG 请求处理路径中，VectorStore 自身也会在首次查询时再次尝试初始化
	// 并缓存状态，因此这里阻塞启动是安全的。
	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := vectorStore.EnsureExtension(initCtx); err != nil {
		sugar.Warnw("pgvector 扩展未就绪，RAG功能降级为关键字搜索", "error", err)
	} else {
		sugar.Infow("pgvector 扩展初始化成功")
	}
	initCancel()

	// 向量存储管理台：只读状态视图 + 连通性测试（配置本身仍由 VECTOR_STORE_CONFIG 部署级管理）

	// 控制器依赖
	incidentRuleEngine := service.NewIncidentRuleEngine(client, sugar)
	incidentService.SetRuleEngine(incidentRuleEngine)
	incidentRulesCommandHandler := service.NewIncidentRulesCommandHandler(client, incidentRuleEngine)
	if err := commandRegistry.Register(commandbus.CommandExecuteIncidentRules, incidentRulesCommandHandler.Handle); err != nil {
		sugar.Fatalw("Failed to register incident rules command handler", "error", err)
	}
	incidentAlertingService.SetConnectorManager(connectorManager)
	analyticsService := service.NewAnalyticsService(client, sugar)
	predictionService := service.NewPredictionService(client, sugar)
	slaForecastSkill := service.NewSLAForecastSkill(client, llmGateway, sugar)
	// 市场服务
	marketplaceSvc := marketplaceService.NewService(client, sugar)
	marketplaceSvc.SetConnectorManager(connectorManager)
	marketplaceHTTPHandler := marketplaceHandler.NewHandler(marketplaceSvc)

	// Guidance sidecar for constrained JSON generation
	guidanceURL := os.Getenv("GUIDANCE_URL")
	if guidanceURL == "" {
		guidanceURL = "http://localhost:8091"
	}
	guidanceClient := service.NewGuidanceClient(guidanceURL, sugar)

	// P2 Handler Services
	triageService := service.NewTriageServiceWithGuidanceAndSugaredLogger(llmGateway, guidanceClient, sugar)

	rootCauseService := service.NewRootCauseService(client, sugar)
	// Bug fix (2026-08-15): inject LLM gateway so AnalyzeTicket / SummarizeTicket
	// actually call the LLM. Previously gateway was constructed but never wired in,
	// so RCA endpoints always fell back to the canned "系统资源不足" template.
	rootCauseService.SetGateway(llmGateway)
	// LLM/Embedding/VectorStore

	// AI Tools
	toolRegistry := service.NewToolRegistry(ragService, incidentService, configurationItemService, client)
	toolQueue := service.NewToolQueue(client, toolRegistry, 100, sugar)
	// 写工具（create_ticket/update_ticket/create_ticket_type）需要领域服务支撑；ticketService 已就绪，此处注入。
	toolRegistry.SetTicketService(ticketService)

	// General Notification Service & Controller
	notificationService := service.NewNotificationService(client)
	// Notification and preference services share the notification domain handler.
	notificationPreferenceService := service.NewNotificationPreferenceService(client, sugar)
	notificationHTTPHandler := notificationHandler.NewHandler(notificationService, notificationPreferenceService, sugar)

	// Ticket Workflow Service & Handler（2026-09-02 迁移至 handlers/ticket_workflow）
	ticketWorkflowService := service.NewTicketWorkflowService(client, sugar)
	ticketWorkflowService.SetConnectorManager(connectorManager)
	ticketWorkflowHandler := ticketworkflow.NewHandler(ticketWorkflowService, database.GetRawDB(), sugar)

	// Ticket Automation Rule Controller (service 已于 131 行预创建并注入 V2)
	// Set notification service dependencies
	ticketService.SetNotificationService(ticketNotificationService)
	ticketCommentService.SetNotificationService(ticketNotificationService)
	ticketRatingService.SetNotificationService(ticketNotificationService)

	rootCauseAnalysisService.SetGateway(llmGateway)
	rootCauseAnalysisService.SetLogger(sugar)
	approvalHandler := approval.NewHandler(approvalService)

	// ProblemController and ChangeController removed - using Handlers instead
	// CMDB ProductionService（原 controller.CMDBController，已迁入 handlers/cmdb）
	cmdbProductionService := cmdb.NewProductionService(sugar, ciTypeService, ciAttributeDefinitionService, configurationItemService, ciRelationshipService, ciHistoryService, ciTagService, importExportService, savedViewService)

	// Release & Asset Management Handlers
	releaseHTTPHandler := releaseHandler.NewHandler(sugar, releaseService)
	assetHTTPHandler := assetHandler.NewHandler(assetService, assetLicenseService, sugar)

	bpmnWorkflowHandler := bpmnHandler.NewWorkflowHandler(processEngine, bpmnVersionService)
	bpmnTemplateService := service.NewBPMNTemplateService(client)

	// BPMN Process Trigger Handler (processBindingService/processTriggerService 已于 119-122 行预创建并注入 V2)
	configInheritanceService := service.NewConfigInheritanceService(client, sugar)
	bpmnProcessTriggerHandler := bpmnHandler.NewProcessTriggerHandler(processTriggerService, processBindingService, configInheritanceService)

	// BPMN Dashboard Handler (监控仪表盘)
	bpmnMetricsService := service.NewBPMNMetricsService(client, sugar)
	bpmnAuditService := service.NewBPMNAuditService(client, sugar)
	bpmnTenantService := service.NewBPMNTenantService(client, sugar)
	bpmnSlaService := service.NewBPMNSLAService(client, sugar)
	bpmnDashboardHandler := bpmnHandler.NewDashboardHandler(bpmnMetricsService, bpmnAuditService, bpmnTenantService, bpmnSlaService)

	// BPMN Monitoring Service & Handler（监控 + 完整执行轨迹时间线）
	bpmnMonitoringService := service.NewBPMNMonitoringService(client, bpmnAuditService, sugar)
	bpmnMonitoringHandler := bpmnHandler.NewMonitoringHandler(bpmnMonitoringService)
	// BPMN AI Generator Service & Handler (AI驱动的流程生成)
	bpmnDeploymentService := service.NewBPMNDeploymentService(client)
	bpmnAIGeneratorService := service.NewBPMNAIGeneratorService(llmGateway, bpmnDeploymentService)
	bpmnAIGeneratorHandler := bpmnHandler.NewAIGeneratorHandler(bpmnAIGeneratorService)

	// BPMN Lint Handler（流程校验真源：设计器校验按钮与 AI 生成后自动 Lint 共用）
	bpmnLintHandler := bpmnHandler.NewLintHandler()
	bpmnHTTPHandler := bpmnHandler.NewHandler(
		bpmnWorkflowHandler,
		bpmnProcessTriggerHandler,
		bpmnDashboardHandler,
		bpmnMonitoringHandler,
		bpmnAIGeneratorHandler,
		bpmnLintHandler,
	)

	// A2UI Ticket Controller (AI-driven UI表单)

	// Global Search Controller (全局搜索)

	// Standard Change Handler (标准变更模板库)
	stdChangeService := standard_change.NewService(client)
	standardChangeHandler := standard_change.NewHandler(stdChangeService, sugar)

	// Known Error Handler (KEDB)
	knownErrorService := known_error.NewService(client)
	knownErrorHandler := known_error.NewHandler(knownErrorService, sugar)

	// Connector Manager / Registry / Market —— 连接器/插件/技能市场基础设施
	// Feishu 连接器控制器
	feishuSyncService := service.NewFeishuSyncService(client, sugar)
	feishuHTTPHandler := feishuHandler.NewHandler(connectorManager, feishuSyncService, marketplaceSvc, sugar)

	// Set process trigger service for workflow integration (after processTriggerService is declared)
	ticketService.SetProcessTriggerService(processTriggerService)
	incidentService.SetProcessTriggerService(processTriggerService)

	// Set approval service for ticket workflow integration
	ticketService.SetApprovalService(approvalService)

	// 初始化模板并部署默认流程
	// 多租户语义:默认流程模板与流程绑定是每租户的基础设施,
	// 部署到所有 active 租户,而不是硬编码 tenant_id=1。
	go func() {
		ctx := context.Background()
		tenants, err := client.Tenant.Query().
			Where(tenant.StatusEQ("active")).
			All(ctx)
		if err != nil {
			sugar.Errorw("Failed to query tenants for BPMN template deployment", "error", err)
			return
		}
		for _, t := range tenants {
			if _, err := bpmnTemplateService.LoadAndDeployTemplates(ctx, t.ID); err != nil {
				sugar.Warnw("Failed to deploy BPMN templates", "tenant_id", t.ID, "error", err)
			}
			if err := processBindingService.InitDefaultBindings(ctx, t.ID); err != nil {
				sugar.Warnw("Failed to init default process bindings", "tenant_id", t.ID, "error", err)
			}
		}
	}()

	// Domain: Service Catalog (DDD)
	scRepo := service_catalog.NewEntRepository(client)
	scService := service_catalog.NewService(scRepo, sugar)
	scHandler := service_catalog.NewHandler(scService)

	// Domain: CMDB (DDD)
	cmdbRepo := cmdb.NewEntRepository(client)
	cloudAdapterRegistry := cloudruntime.NewRegistry()
	cloudAdapterRegistry.Register(cloudaliyun.NewAliyunECSAdapter(sugar))
	cmdbServiceDomain := cmdb.NewServiceWithDiscoveryRuntime(cmdbRepo, cmdbProductionService, sugar, cmdb.DiscoveryRuntime{
		Adapters: cloudAdapterRegistry,
		// secret:// resolution and the durable worker land in later phases.
		CredentialResolverReady: false,
		WorkerReady:             false,
	})
	cmdbHandler := cmdb.NewHandler(cmdbServiceDomain)

	// Approval Chain Service（供服务请求审批链求值引擎消费）
	approvalChainService := service.NewApprovalChainService(client, sugar)
	mspAllocationService := service.NewMSPAllocationService(client, sugar)
	escalationMatrixService := service.NewEscalationMatrixService(sugar)
	vendorService := service.NewVendorService(client, sugar)
	provisioningService := service.NewProvisioningService(client, sugar)
	ticketCategoryService := service.NewTicketCategoryService(client)

	// Domain: Service Request (DDD)
	srRepo := service_request.NewEntRepository(client)
	srService := service_request.NewService(srRepo, scRepo, cmdbRepo, client, sugar, approvalChainService)
	srHandler := service_request.NewHandler(srService)

	// Domain: Incident (DDD)
	// Note: Incident handler has been removed from router config
	_ = incident.NewEntRepository // Prevent unused import warning

	// Domain: Problem (DDD)
	problemRepo := problem.NewEntRepository(client)
	problemServiceDomain := problem.NewService(problemRepo, sugar)
	problemHandler := problem.NewHandler(problemServiceDomain)

	// Problem Investigation Service & Handler（问题调查/RCA/解决方案/知识沉淀）
	// 修复：此前该 controller 从未在 bootstrap 装配，导致 /problem-investigation 路由组整体未注册（404）
	// 2026-09-02 迁移至 handlers/problem_investigation（域切片架构）
	problemInvestigationService := service.NewProblemInvestigationService(database.GetRawDB(), client, sugar)
	problemInvestigationHandler := probleminvestigation.NewHandler(sugar, problemInvestigationService)

	// Domain: Change (DDD)
	changeRepo := change.NewEntRepository(client, database.GetRawDB())
	changeServiceDomain := change.NewService(changeRepo, client, sugar, approvalChainService)
	changeHandler := change.NewHandler(changeServiceDomain)

	// CAB 成员名册 handler（审批流转由审批链引擎 cab: 解析器驱动，handler 仅管名册）
	cabService := service.NewCABService(client, sugar)
	cabHandler := cab.NewHandler(cabService, sugar)

	// Analytics & Prediction Controllers

	// Domain: Knowledge (DDD)
	knowledgeRepo := knowledge.NewEntRepository(client)
	knowledgeServiceDomain := knowledge.NewService(knowledgeRepo, sugar)
	// 向量索引同步：发布→索引，取消发布/软删除→移除向量（RemoveArticle 真实删除）。
	knowledgeServiceDomain.SetRAG(ragService)
	// 知识分类可见性（L0 权限边界）：纳管能力 + AI 检索分类过滤共用同一守卫实例，
	// 保证纳管变更后缓存立即失效，不会出现「改了配置但检索仍放行」的窗口。
	knowledgeGuard := knowledgeaccess.NewGuard(client, sugar)
	knowledgeServiceDomain.SetEntClient(client)
	knowledgeServiceDomain.SetKnowledgeGuard(knowledgeGuard)
	ragService.SetKnowledgeGuard(knowledgeGuard)
	knowledgeHandler := knowledge.NewHandler(knowledgeServiceDomain)

	// Domain: SLA (DDD)
	slaRepo := sla.NewEntRepository(client)
	slaServiceDomain := sla.NewService(slaRepo, sugar)
	slaHandler := sla.NewHandler(slaServiceDomain)

	// SLA 模板服务（开箱即用）

	// AI Domain
	aiRepo := ai.NewEntRepository(client)
	aiServiceDomain := ai.NewService(aiRepo, sugar, ragService, toolRegistry, toolQueue, analyticsService, predictionService, slaForecastSkill, triageService, rootCauseService, aiTelemetryService)
	aiServiceDomain.SetLLMGateway(llmGateway)
	// P2-6: 注入 ent client 供 AI 工具 RBAC 校验复用 hasResourcePermission
	aiServiceDomain.SetEntClient(client)
	aiHandler := ai.NewHandler(aiServiceDomain)

	// Sprint C — Skill Registry v1：在 ai.Service 装配完成后注入内置 Skill。
	// 注册失败按"启动期 fail-fast"原则处理：以 service.SkillRegistry 注入到 ai package 中，
	// 供后续 handlers/skill 管理 API 调用。
	skillRegistry := service.NewSkillRegistry()
	if err := ai.RegisterBuiltinSkills(skillRegistry, aiServiceDomain, sugar); err != nil {
		sugar.Errorw("failed to register builtin AI skills; SkillRegistry will operate in partial mode",
			"error", err)
	}
	sugar.Infow("AI SkillRegistry ready", "total_skills", skillRegistry.Count())

	// Sprint C — Skills Management API：在 SkillRegistry 装配完成后创建 handler。
	// handler 只是 thin wrapper，所有业务逻辑在 SkillRegistry 内。
	skillHandler := skill.NewHandler(skillRegistry, sugar)
	emailIntakeService := email_intake.NewService(client)
	emailIntakeHandler := email_intake.NewHandler(emailIntakeService)
	emailIntakeMode := email_intake.IntakeMode(os.Getenv("EMAIL_INTAKE_MODE"))
	automationReporterID, _ := strconv.Atoi(os.Getenv("EMAIL_INTAKE_AUTOMATION_REPORTER_ID"))
	assignmentGroupID, _ := strconv.Atoi(os.Getenv("EMAIL_INTAKE_DEFAULT_GROUP_ID"))
	var assignmentGroupIDPtr *int
	if assignmentGroupID > 0 {
		assignmentGroupIDPtr = &assignmentGroupID
	}
	emailExtractor := email_intake.NewEmailIntakeExtractor(llmGateway, llmConfig.Model)
	emailIntakeOrchestrator := email_intake.NewEmailIntakeOrchestrator(client, emailExtractor, incidentService, email_intake.OrchestratorConfig{
		Mode: emailIntakeMode, AutomationReporterUserID: automationReporterID, DefaultAssignmentGroupID: assignmentGroupIDPtr,
	})
	if err := commandRegistry.Register(commandbus.CommandProcessIntakeEmail, email_intake.NewIntakeProcessCommandHandler(emailIntakeOrchestrator).Handle); err != nil {
		sugar.Fatalw("Failed to register email intake process handler", "error", err)
	}
	emailIntakeHandler.SetOrchestrator(emailIntakeOrchestrator)
	connectorManager.SetInboundHandler("email", emailIntakeOrchestrator.IngestConnectorMessage)

	// Sprint C — Evaluator bySkill 维度：将 SkillRegistry 注入到 AI 评估服务。
	// 这样 /ai/evaluation 返回的 bySkill 字段会带上 Skill 的 Name/Category 元数据，
	// 同时 byScenario 项也会带 SkillName，便于前端"技能"与"场景"两个视角对齐。
	aiTelemetryService.SetSkillRegistry(skillRegistry)

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
			// Phase 1 P1-4：权限缓存失效跨实例广播（多副本一致性）。
			middleware.ConfigurePermissionCacheBroadcast(commonRedis)
			sugar.Info("refresh token blacklist enabled via redis")
		}
		pingCancel()
	}
	commonHandler := domainCommon.NewHandler(commonServiceDomain)

	// Auth handler owns account self-service and tenant session switching.
	authService := authHandler.NewService(client, cfg.JWT.Secret, sugar, nil)
	authHTTPHandler := authHandler.NewHandler(authService)

	// Role Handler (in-memory for now)
	roleHandler := common.NewRoleHandler(client, sugar)

	// User Handler
	userService := service.NewUserService(client, sugar)
	userHTTPHandler := userHandler.NewHandler(userService, sugar)

	// Group Handler
	groupService := service.NewGroupService(client)
	groupHTTPHandler := groupHandler.NewHandler(groupService, sugar)

	// RBAC handler (database-backed with tenant isolation)
	roleService := service.NewRoleService(client, sugar)
	permissionService := service.NewPermissionService(client, sugar)
	menuService := service.NewMenuService(client, sugar)
	rbacHTTPHandler := rbacHandler.NewHandler(roleService, permissionService, menuService, sugar)

	// Audit Log Controller (支持过滤/分页的审计日志查询)

	// Tenant handler
	tenantService := service.NewTenantService(client, sugar)
	tenantHTTPHandler := tenantHandler.NewHandler(tenantService, sugar)

	// System Config Handler（2026-09-02 迁移至 handlers/systemconfig）
	systemConfigService := service.NewSystemConfigService(client, sugar)
	systemConfigHandler := systemconfig.NewHandler(systemConfigService, sugar)

	// Vendor Controller

	// Approval Chain Controller

	// SLA Monitor & Alert Services (legacy, for background tasks)
	slaMonitorService := service.NewSLAMonitorService(client, sugar)
	slaAlertService := service.NewSLAAlertService(client, sugar)
	escalationService := service.NewEscalationService(client, sugar)

	// Wire up notification service
	slaMonitorService.SetNotificationService(ticketNotificationService)
	slaAlertService.SetNotificationService(ticketNotificationService)
	escalationService.SetNotificationService(ticketNotificationService)

	// Survey Service & Controller

	// Cloud Service & Controller
	// 工单类型服务就绪后注入工具注册表与审批队列，使 create_ticket_type 可经审批流执行。
	toolRegistry.SetTicketTypeService(ticketTypeService)
	toolQueue.SetTicketTypeService(ticketTypeService)

	// WebSocket Service
	wsService := service.NewWebSocketService(sugar)

	// 7. 设置路由
	// 根据配置设置 Gin 运行模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.Server.Mode == "test" {
		gin.SetMode(gin.TestMode)
	}
	// 配置 Trusted Proxies：
	// 1. 默认包含 localhost 与 RFC1918 私有 CIDR，避免 Docker bridge 网段（172.x）
	//    被识别成客户端 IP 而把 nginx 转发链上的容器内网（172.28.0.x）写入审计日志。
	// 2. 通过 TRUSTED_PROXIES 环境变量（逗号分隔 CIDR/IP）可追加额外代理网段，
	//    例如 k8s ingress / 企业 NAT 网关等。空值表示只使用默认值。
	defaultTrustedProxies := []string{
		"127.0.0.1",
		"::1",
		"10.0.0.0/8",
		"172.16.0.0/12", // RFC1918 私有网段，覆盖 docker-compose 默认 bridge 与 k8s pod CIDR
		"192.168.0.0/16",
	}
	if extra := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES")); extra != "" {
		for _, item := range strings.Split(extra, ",") {
			if cidr := strings.TrimSpace(item); cidr != "" {
				defaultTrustedProxies = append(defaultTrustedProxies, cidr)
			}
		}
	}
	r := gin.Default()
	if err := r.SetTrustedProxies(defaultTrustedProxies); err != nil {
		sugar.Warnw("failed to set trusted proxies, falling back to default", "error", err)
	}
	sugar.Infow("gin trusted proxies configured", "cidrs", defaultTrustedProxies)

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
		JWTSecret:                    cfg.JWT.Secret,
		Logger:                       sugar,
		Client:                       client,
		RawDB:                        database.GetRawDB(),
		CSRFEnabled:                  cfg.Security.CSRFEnabled,
		RedisRateLimiter:             redisRateLimiter,
		AppStartTime:                 time.Now(),
		TicketHandler:                ticketHandler,
		TicketDependencyHandler:      ticketDependencyHandler.NewHandler(ticketDependencyService, sugar),
		TicketCommentHandler:         ticketCommentHandler.NewHandler(ticketCommentService, sugar),
		TicketAttachmentHandler:      ticketAttachmentHandler.NewHandler(ticketAttachmentService, sugar),
		TicketNotificationHandler:    ticketNotificationHandler.NewHandler(ticketNotificationService, sugar),
		NotificationHandler:          notificationHTTPHandler,
		TicketRatingHandler:          ticketRatingHandler.NewHandler(ticketRatingService, sugar),
		TicketAssignmentSmartHandler: assignmentSmartHandler.NewHandler(ticketAssignmentSmartService, ticketAssignmentRuleService, sugar),
		TicketViewHandler:            ticketViewHandler.NewHandler(ticketViewService, sugar),
		TicketWorkflowHandler:        ticketWorkflowHandler,
		TicketAutomationRuleHandler:  automationRuleHandler.NewHandler(ticketAutomationRuleService, sugar),
		IncidentHandler:              incidentHandler,
		ApprovalHandler:              approvalHandler,
		BPMNHandler:                  bpmnHTTPHandler,
		A2UIHandler:                  a2uiHandler.NewHandler(a2uiService, sugar),
		CMDBHandler:                  cmdbHandler,
		TicketCategoryHandler:        ticketCategoryHandler.NewHandler(ticketCategoryService, sugar),
		TicketTypeHandler:            ticketTypeHandler.NewHandler(ticketTypeService, sugar),
		TicketTagHandler:             ticketTagHandler.NewHandler(ticketTagService, sugar),
		EscalationMatrixHandler:      escalationMatrixHandler.NewHandler(sugar, escalationMatrixService),
		AuditLogHandler:              auditlogHandler.NewHandler(auditLogService, sugar),
		MSPHandler:                   mspHandler.NewHandler(mspAllocationService, ticketService, sugar),
		SystemConfigHandler:          systemConfigHandler,
		ApprovalChainHandler:         approvalChainHandler.NewHandler(approvalChainService, sugar),

		// Vendor Controller
		VendorHandler: vendorHandler.NewHandler(vendorService, sugar),

		// Additional controllers
		ProvisioningHandler: provisioningHandler.NewHandler(provisioningService, sugar),
		UserHandler:         userHTTPHandler,
		GroupHandler:        groupHTTPHandler,

		// RBAC and tenant handlers
		RBACHandler:       rbacHTTPHandler,
		TenantHandler:     tenantHTTPHandler,
		AnalyticsHandler:  analyticsHandler.NewHandler(analyticsService),
		PredictionHandler: predictionHandler.NewHandler(predictionService, sugar),
		ReleaseHandler:    releaseHTTPHandler,
		AssetHandler:      assetHTTPHandler,
		SurveyHandler:     surveyHandler.NewHandler(surveyService, sugar),
		CloudHandler:      cloudHandler.NewHandler(cloudService, sugar),

		// Domain Handlers
		ServiceCatalogHandler:       scHandler,
		ServiceRequestHandler:       srHandler,
		ProblemHandler:              problemHandler,
		ProblemInvestigationHandler: problemInvestigationHandler,
		ChangeHandler:               changeHandler,
		CABHandler:                  cabHandler,
		KnowledgeHandler:            knowledgeHandler,
		SLAHandler:                  slaHandler,
		SLATemplateHandler:          slaTemplateHandler.NewHandler(slaTemplateService),
		VectorStoreHandler:          vectorStoreHandler.NewHandler(database.GetRawDB(), sugar),
		AIHandler:                   aiHandler, // Added AI domain handler
		EmailIntakeHandler:          emailIntakeHandler,
		CommonHandler:               commonHandler,
		AuthHandler:                 authHTTPHandler,
		RoleHandler:                 roleHandler,

		// Sprint C — Skill Registry v1
		SkillHandler: skillHandler,

		// Global Search
		GlobalSearchHandler: globalSearchHandler.NewHandler(globalSearchHandler.NewService(client)),

		// Standard Change Handler
		StandardChangeHandler: standardChangeHandler,

		// Known Error Handler (KEDB)
		KnownErrorHandler: knownErrorHandler,

		// Connector Handler
		ConnectorHandler: connectorHandler,
		AlertHandler:     alertHandler,
		FeishuHandler:    feishuHTTPHandler,

		MarketplaceHandler: marketplaceHTTPHandler,

		// WebSocket Service
		WebSocketService: wsService,

		// Ticket Association Service
		TicketAssociationService: ticketAssociationService,
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
		Cfg:               cfg,
		Logger:            sugar,
		DBClient:          client,
		Router:            r,
		Embedder:          embedder,
		VectorStore:       pluggableVectorStore,
		LegacyVectorStore: vectorStore,
		CommandWorker:     commandWorker,
		SkillRegistry:     skillRegistry,
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

func alertDevelopmentMode() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ENV"))) {
	case "development", "dev", "test", "testing", "local":
		return true
	default:
		return false
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
		if err := prepareTicketFormFieldsMigration(ctx, database.GetRawDB(), sugar); err != nil {
			return fmt.Errorf("prepare ticket form fields migration: %w", err)
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
		if needsAdmin && os.Getenv("BOOTSTRAP_TOKEN_ENABLED") != "1" {
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
	mode, err := ParseProcessMode(os.Getenv("ITSM_PROCESS_MODE"))
	if err != nil {
		app.Logger.Fatalw("invalid process mode", "error", err)
	}
	environment := os.Getenv("SERVER_ENV")
	if environment == "" {
		environment = os.Getenv("ENV")
	}
	if err := ValidateProcessMode(mode, environment); err != nil {
		app.Logger.Fatalw("unsafe process mode", "error", err)
	}
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if mode == ProcessModeWorker || mode == ProcessModeAll {
		app.startBackgroundTasks(rootCtx)
		app.Logger.Infow("worker schedulers started", "process_mode", mode)
	}
	if mode == ProcessModeWorker {
		<-rootCtx.Done()
		app.Logger.Info("worker shutdown completed")
		return
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.Cfg.Server.Port),
		Handler:           app.Router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		app.Logger.Infow("API server starting", "port", app.Cfg.Server.Port, "process_mode", mode)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-rootCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			app.Logger.Errorw("API graceful shutdown failed", "error", err)
		}
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			app.Logger.Fatalw("API server failed", "error", err)
		}
	}
}

func (app *Application) startBackgroundTasks(ctx context.Context) {
	// safeGo 启动一个 panic-safe 的后台 goroutine：
	// - 任务 panic 会被 recover 并记录完整堆栈，goroutine 不再静默退出
	// - 自动通过 app.backgroundWG 跟踪生命周期，Stop() 中可等待优雅退出
	safeGo := func(name string, fn func()) {
		app.backgroundWG.Add(1)
		go func() {
			defer app.backgroundWG.Done()
			defer func() {
				if r := recover(); r != nil {
					app.Logger.Errorw("background task panicked, recovered",
						"task", name,
						"panic", r,
						"stack", string(debug.Stack()),
					)
				}
			}()
			fn()
		}()
	}

	// Command worker: 依赖外部 Worker 自带清理逻辑，这里仅做 panic 防护
	if app.CommandWorker != nil {
		safeGo("command-worker", func() {
			app.CommandWorker.Run(ctx)
		})
	}

	// Embedding pipeline 后台任务
	safeGo("embedding-pipeline", func() {
		pipeline := service.NewEmbeddingPipeline(app.DBClient, app.Embedder, app.Logger, app.LegacyVectorStore)
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
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
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
	})

	// SLA Monitoring and Escalation background tasks
	safeGo("sla-monitor-escalation", func() {
		slaMonitorService := service.NewSLAMonitorService(app.DBClient, app.Logger)
		escalationService := service.NewEscalationService(app.DBClient, app.Logger)

		// Run SLA check every 5 minutes
		slaTicker := time.NewTicker(5 * time.Minute)
		defer slaTicker.Stop()

		// Run escalation check every 15 minutes
		escalationTicker := time.NewTicker(15 * time.Minute)
		defer escalationTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
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
	})
}

// StopBackgroundTasks 等待所有由 startBackgroundTasks 启动的后台 goroutine
// 退出。调用方需先取消传入 startBackgroundTasks 的 ctx，使得 goroutine 内部的
// select 能感知到 Done 信号；本方法提供在取消之后的同步等待点，
// 超时则强制返回，避免关闭流程被无响应的后台任务永远阻塞。
//
// 默认超时 30 秒。可由 ITSM_BACKGROUND_SHUTDOWN_TIMEOUT_SECONDS 覆盖。
func (app *Application) StopBackgroundTasks() {
	timeout := 30 * time.Second
	if v := os.Getenv("ITSM_BACKGROUND_SHUTDOWN_TIMEOUT_SECONDS"); v != "" {
		if d, err := time.ParseDuration(v + "s"); err == nil && d > 0 {
			timeout = d
		}
	}
	done := make(chan struct{})
	go func() {
		app.backgroundWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		app.Logger.Info("all background tasks stopped cleanly")
	case <-time.After(timeout):
		app.Logger.Warnw("background tasks shutdown timed out; some goroutines may still be running",
			"timeout", timeout)
	}
}
