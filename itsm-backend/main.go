package main

import (
	"context"
	"log"

	"itsm-backend/controller"
	"itsm-backend/ent"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
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

	// 初始化控制器层
	ticketController := controller.NewTicketController(ticketService, sugar)

	// 初始化Gin引擎
	r := gin.Default()

	// 设置路由
	router.SetupRoutes(r, ticketController)

	// 启动服务器
	sugar.Info("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}