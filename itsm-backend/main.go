package main

import (
	"context"
	"log"

	"itsm-backend/controller"
	"itsm-backend/ent"
	"itsm-backend/router"
	"itsm-backend/service"

	"go.uber.org/zap"
	_ "github.com/lib/pq" // PostgreSQL driver
)

func main() {
	// 初始化日志
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	// 初始化数据库连接
	// TODO: 从配置文件读取数据库连接信息
	client, err := ent.Open("postgres", "host=localhost port=5432 user=dev dbname=itsm sslmode=disable password=123456!@#$%^")
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer client.Close()

	// 运行数据库迁移
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	// 初始化服务层
	ticketService := service.NewTicketService(client, sugar)
	serviceCatalogService := service.NewServiceCatalogService(client)
	serviceRequestService := service.NewServiceRequestService(client, serviceCatalogService)
	
	// 初始化控制器层
	ticketController := controller.NewTicketController(ticketService, sugar)
	serviceController := controller.NewServiceController(serviceCatalogService, serviceRequestService)
	
	// 设置路由
	r := router.SetupRouter(ticketController, serviceController)
	
	// 启动服务器
	log.Println("服务器启动在端口 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("启动服务器失败:", err)
	}
}