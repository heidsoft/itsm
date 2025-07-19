package main

import (
	"context"
	"fmt"
	"itsm-backend/config"
	"itsm-backend/controller"
	"itsm-backend/database"
	"itsm-backend/router"
	"itsm-backend/service"
	"log"

	"go.uber.org/zap"
)

func main() {
	// 初始化配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	// 初始化数据库
	client, err := database.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer client.Close()

	// 创建数据库schema
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("Failed to create schema resources: %v", err)
	}

	// 初始化服务
	ticketService := service.NewTicketService(client, sugar)
	serviceCatalogService := service.NewServiceCatalogService(client)
	serviceRequestService := service.NewServiceRequestService(client, serviceCatalogService)
	dashboardService := service.NewDashboardService(client, sugar)
	cmdbService := service.NewCMDBService(client, sugar)
	incidentService := service.NewIncidentService(client, sugar)

	// 初始化控制器
	ticketController := controller.NewTicketController(ticketService, sugar)
	serviceController := controller.NewServiceController(serviceCatalogService, serviceRequestService)
	dashboardController := controller.NewDashboardController(dashboardService, sugar)
	cmdbController := controller.NewCMDBController(cmdbService)
	incidentController := controller.NewIncidentController(incidentService, sugar)

	// 设置路由
	r := router.SetupRouter(ticketController, serviceController, dashboardController, cmdbController, incidentController, client, cfg.JWT.Secret)

	// 启动服务器
	sugar.Infof("Server starting on port %d", cfg.Server.Port)
	if err := r.Run(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
