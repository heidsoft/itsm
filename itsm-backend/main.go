//go:build !migrate && !create_user
// +build !migrate,!create_user

// 构建标签：指定在什么条件下编译这个文件
// !migrate && !create_user 表示当不执行数据库迁移和创建用户时编译此文件
// 这样可以避免在运行迁移脚本时启动完整的Web服务器

package main

// Go语言的包声明
// main包是Go程序的入口点，包含main函数

import (
	"context"                 // Go标准库，用于处理上下文（超时、取消等）
	"fmt"                     // Go标准库，用于格式化输出
	"itsm-backend/config"     // 自定义配置包
	"itsm-backend/controller" // 自定义控制器包
	"itsm-backend/database"   // 自定义数据库包
	"itsm-backend/middleware" // 注入全局日志器
	"itsm-backend/router"     // 自定义路由包
	"itsm-backend/service"    // 自定义服务包
	"log"                     // Go标准库，用于日志记录
	"time"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap" // 第三方日志库，高性能的结构化日志
)

// main函数：Go程序的入口点
// 当程序启动时，首先执行这个函数
func main() {
	// 1. 初始化配置
	// LoadConfig函数从配置文件（如config.yaml）加载应用配置
	// 包括数据库连接、服务器端口、JWT密钥等
	cfg, err := config.LoadConfig()
	if err != nil {
		// log.Fatalf 会打印错误信息并退出程序
		// %v 是Go的格式化动词，用于显示错误详情
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. 初始化日志系统
	// zap.NewProduction() 创建生产环境的日志配置
	// 包括时间戳、日志级别、调用者信息等
	logger, _ := zap.NewProduction()
	defer logger.Sync()     // defer确保程序退出前同步日志到磁盘
	sugar := logger.Sugar() // Sugar()提供更简单的日志API
	// 注入到中间件，避免每请求新建 logger
	middleware.SetLogger(sugar)

	// 3. 初始化数据库连接
	// InitDatabase函数建立与PostgreSQL数据库的连接
	// 使用Ent ORM框架进行数据库操作
	client, err := database.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer client.Close() // defer确保程序退出前关闭数据库连接

	// 4. 创建数据库schema
	// Ent ORM会自动创建数据库表结构
	// 根据Go结构体定义生成对应的数据库表
	// context.Background() 创建根上下文
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("Failed to create schema resources: %v", err)
	}

	// 5. 初始化业务服务层
	// 服务层包含业务逻辑，是控制器和数据层之间的桥梁
	// NewNotificationService 创建通知服务实例
	notificationService := service.NewNotificationService(client)
	// NewIncidentService 创建事件服务实例，传入数据库客户端和日志器
	incidentService := service.NewIncidentService(client, sugar)
	// NewUserService 创建用户服务实例，传入数据库客户端和日志器
	userService := service.NewUserService(client, sugar)
	// 其他服务
	ticketService := service.NewTicketService(client, sugar)
	serviceCatalogService := service.NewServiceCatalogService(client, sugar)
	serviceRequestService := service.NewServiceRequestService(client, sugar)
	cmdbService := service.NewCMDBService(client, sugar)
	dashboardService := service.NewDashboardService(client, sugar)
	knowledgeService := service.NewKnowledgeService(client, sugar)
	knowledgeIntegrationService := service.NewKnowledgeIntegrationService(client)
	// 新增服务
	workflowService := service.NewWorkflowService(client)
	ticketTagService := service.NewTicketTagService(client)
	ticketAssignmentService := service.NewTicketAssignmentService(client)
	ticketCategoryService := service.NewTicketCategoryService(client)
	problemService := service.NewProblemService(client, sugar)
	changeService := service.NewChangeService(client, sugar)
	// 5.1 初始化 LLM/Embedding/VectorStore
	// LLM/Embedding Provider: use cfg.LLM
	var embedder service.Embedder
	if cfg.LLM.Provider == "openai" || cfg.LLM.Provider == "" {
		embedder = service.NewOpenAIEmbedderWithConfig(cfg.LLM.APIKey, cfg.LLM.Endpoint, cfg.LLM.Model)
	} else {
		embedder = service.NewOpenAIEmbedder() // fallback
	}
	vectorStore := service.NewVectorStore(database.GetRawDB())
	ragService := service.NewRAGService(client, vectorStore, embedder)
	aiTelemetryService := service.NewAITelemetryService(database.GetRawDB())

	// 6. 初始化控制器层
	// 控制器层处理HTTP请求，调用服务层执行业务逻辑
	// NewNotificationController 创建通知控制器
	notificationController := controller.NewNotificationController(notificationService)
	// NewIncidentController 创建事件控制器
	incidentController := controller.NewIncidentController(incidentService, sugar)
	// NewUserController 创建用户控制器
	userController := controller.NewUserController(userService, sugar)
	// 其他控制器
	ticketController := controller.NewTicketController(ticketService, sugar)
	serviceController := controller.NewServiceController(serviceCatalogService, serviceRequestService)
	cmdbController := controller.NewCMDBController(cmdbService)
	dashboardController := controller.NewDashboardController(dashboardService, sugar)
	knowledgeController := controller.NewKnowledgeController(knowledgeService, sugar)
	knowledgeIntegrationController := controller.NewKnowledgeIntegrationController(knowledgeIntegrationService, sugar)
	aiController := controller.NewAIController(ragService, client, aiTelemetryService)
	// 初始化工作流引擎和相关服务
	workflowEngine := service.NewWorkflowEngine(client)
	workflowApprovalService := service.NewWorkflowApprovalService(client, workflowEngine)
	workflowTaskService := service.NewWorkflowTaskService(client, workflowEngine)
	workflowMonitorService := service.NewWorkflowMonitorService(client, workflowEngine)
	
	// 初始化BPMN工作流引擎
	bpmnProcessEngine := service.NewBPMNProcessEngine(client)

	// 新增控制器
	workflowController := controller.NewWorkflowController(
		workflowService,
		workflowEngine,
		workflowApprovalService,
		workflowTaskService,
		workflowMonitorService,
		logger,
	)
	
	// 初始化BPMN工作流控制器
	bpmnWorkflowController := controller.NewBPMNWorkflowController(bpmnProcessEngine)
	
	// 初始化BPMN监控控制器
	bpmnMonitoringService := service.NewBPMNMonitoringService(client)
	bpmnMonitoringController := controller.NewBPMNMonitoringController(bpmnMonitoringService)
	ticketTagController := controller.NewTicketTagController(ticketTagService, logger)
	ticketAssignmentController := controller.NewTicketAssignmentController(ticketAssignmentService, logger)
	ticketCategoryController := controller.NewTicketCategoryController(ticketCategoryService, sugar)
	problemController := controller.NewProblemController(sugar, problemService)
	changeController := controller.NewChangeController(sugar, changeService)
	// 初始化 ToolRegistry（含只读与危险工具）
	toolRegistry := service.NewToolRegistry(ragService, incidentService, cmdbService, client)
	aiController.SetToolRegistry(toolRegistry)
	// 初始化内存队列用于后台执行危险工具
	toolQueue := service.NewToolQueue(client, toolRegistry, 100)
	aiController.SetToolQueue(toolQueue)
	aiController.SetEmbeddingResources(embedder, vectorStore)
	// 认证控制器
	authController := controller.NewAuthController(cfg.JWT.Secret, client)

	// 7. 设置路由
	// SetupRouter函数配置Gin路由，定义API端点
	// 参数说明：
	// - 装配所有控制器，启用真实登录与受保护路由
	r := router.SetupRouter(
		authController,
		ticketController,
		serviceController,
		dashboardController,
		cmdbController,
		incidentController,
		notificationController,
		userController,
		knowledgeController,
		aiController,
		workflowController,
		bpmnWorkflowController,
		bpmnMonitoringController,
		ticketTagController,
		ticketAssignmentController,
		ticketCategoryController,
		problemController,
		changeController,
		knowledgeIntegrationController,
		client,
		cfg.JWT.Secret,
	)

	// Swagger 文档（仅在开发/测试环境开启）
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 7.1 启动嵌入任务：一次性 + 定时增量（示例每15分钟）
	go func() {
		pipeline := service.NewEmbeddingPipeline(client, embedder, sugar, vectorStore)
		ctx := context.Background()
		// initial full-ish pass per tenant
		tenants, err := client.Tenant.Query().All(ctx)
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
			tenants, err := client.Tenant.Query().All(ctx)
			if err != nil {
				continue
			}
			for _, t := range tenants {
				_ = pipeline.RunOnce(ctx, t.ID, 50)
			}
		}
	}()

	// 8. 启动Web服务器
	// sugar.Infof 记录信息级别的日志
	// fmt.Sprintf 格式化字符串，生成端口地址（如":8080"）
	sugar.Infof("Server starting on port %d", cfg.Server.Port)
	// r.Run 启动Gin服务器，监听指定端口
	// 这是阻塞调用，服务器会一直运行直到程序退出
	if err := r.Run(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// 程序执行流程说明：
// 1. 加载配置文件 → 2. 初始化日志 → 3. 连接数据库 → 4. 创建表结构
// 5. 初始化服务 → 6. 初始化控制器 → 7. 设置路由 → 8. 启动服务器
//
// 架构说明：
// - 配置层：管理应用配置
// - 数据层：Ent ORM处理数据库操作
// - 服务层：包含业务逻辑
// - 控制器层：处理HTTP请求
// - 路由层：定义API端点
// - 中间件：处理认证、日志、CORS等
